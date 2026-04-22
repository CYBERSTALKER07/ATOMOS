package pin

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
)

// EntityType identifies which table a PIN belongs to.
type EntityType string

const (
	EntityDriver         EntityType = "DRIVER"
	EntityWarehouseStaff EntityType = "WAREHOUSE_STAFF"
	EntityFactoryStaff   EntityType = "FACTORY_STAFF"
)

// maxCollisionRetries is the number of Generate→check cycles before giving up.
// With 10^8 (100 million) possible 8-digit PINs and typical fleet sizes under
// 100k, the probability of even one collision is negligible. Three retries is
// a generous safety margin.
const maxCollisionRetries = 3

// Registry manages GlobalPins rows inside Spanner transactions.
type Registry struct {
	Spanner *spanner.Client
}

// Result holds the outputs of a successful GenerateUnique call.
type Result struct {
	Plaintext  string // 8-digit PIN — return to the caller once, never persist
	BcryptHash string // bcrypt hash — store in the entity table's PinHash column
	SHA256Hex  string // SHA-256 hex — stored in GlobalPins by the txn
}

// GenerateUnique generates a globally unique PIN, registers it in GlobalPins,
// and returns the plaintext + bcrypt hash for the caller to write into the
// entity row — all inside the provided ReadWriteTransaction so the entity
// INSERT and the GlobalPins INSERT are atomic.
//
// entityType and entityID identify the owner so the row can be cleaned up on
// PIN rotation or entity deletion.
func GenerateUnique(ctx context.Context, txn *spanner.ReadWriteTransaction, entityType EntityType, entityID string) (*Result, error) {
	for attempt := 0; attempt < maxCollisionRetries; attempt++ {
		plain, err := Generate()
		if err != nil {
			return nil, fmt.Errorf("pin.GenerateUnique: %w", err)
		}

		shaHex := SHA256Hex(plain)

		// Check collision inside the txn (strong read — serializable).
		row, err := txn.ReadRow(ctx, "GlobalPins", spanner.Key{shaHex}, []string{"PinSha256"})
		if err == nil && row != nil {
			// Collision — retry with a new PIN.
			continue
		}

		// Row not found → no collision. Register the PIN.
		hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("pin.GenerateUnique: bcrypt: %w", err)
		}

		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.Insert("GlobalPins",
				[]string{"PinSha256", "EntityType", "EntityId", "CreatedAt"},
				[]interface{}{shaHex, string(entityType), entityID, spanner.CommitTimestamp},
			),
		}); err != nil {
			return nil, fmt.Errorf("pin.GenerateUnique: buffer GlobalPins insert: %w", err)
		}

		return &Result{
			Plaintext:  plain,
			BcryptHash: string(hash),
			SHA256Hex:  shaHex,
		}, nil
	}

	return nil, fmt.Errorf("pin.GenerateUnique: exhausted %d retries — all generated PINs collided", maxCollisionRetries)
}

// RegisterExisting registers an admin-provided PIN in GlobalPins. Used by
// warehouse staff creation where the admin chooses the PIN. Returns the
// bcrypt hash for storage.
//
// Returns an error if the PIN is already taken.
func RegisterExisting(ctx context.Context, txn *spanner.ReadWriteTransaction, plaintext string, entityType EntityType, entityID string) (bcryptHash string, err error) {
	shaHex := SHA256Hex(plaintext)

	row, readErr := txn.ReadRow(ctx, "GlobalPins", spanner.Key{shaHex}, []string{"PinSha256"})
	if readErr == nil && row != nil {
		return "", fmt.Errorf("pin.RegisterExisting: PIN already in use")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("pin.RegisterExisting: bcrypt: %w", err)
	}

	if err := txn.BufferWrite([]*spanner.Mutation{
		spanner.Insert("GlobalPins",
			[]string{"PinSha256", "EntityType", "EntityId", "CreatedAt"},
			[]interface{}{shaHex, string(entityType), entityID, spanner.CommitTimestamp},
		),
	}); err != nil {
		return "", fmt.Errorf("pin.RegisterExisting: buffer GlobalPins insert: %w", err)
	}

	return string(hash), nil
}

// Delete removes the GlobalPins row for a given entity. Call inside a
// ReadWriteTransaction when rotating a PIN or deactivating an entity.
func Delete(ctx context.Context, txn *spanner.ReadWriteTransaction, entityType EntityType, entityID string) error {
	// Look up the SHA-256 key via the reverse index.
	stmt := spanner.Statement{
		SQL:    `SELECT PinSha256 FROM GlobalPins WHERE EntityType = @et AND EntityId = @eid`,
		Params: map[string]interface{}{"et": string(entityType), "eid": entityID},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil // No existing PIN — nothing to delete.
	}
	if err != nil {
		return fmt.Errorf("pin.Delete: query existing: %w", err)
	}

	var shaHex string
	if err := row.Columns(&shaHex); err != nil {
		return fmt.Errorf("pin.Delete: scan: %w", err)
	}

	return txn.BufferWrite([]*spanner.Mutation{
		spanner.Delete("GlobalPins", spanner.Key{shaHex}),
	})
}

// Rotate deletes the existing GlobalPins row for the entity (if any) and
// generates a new unique PIN. Must be called inside a ReadWriteTransaction.
// The caller is responsible for updating the entity table's PinHash column
// with Result.BcryptHash in the same txn.
func Rotate(ctx context.Context, txn *spanner.ReadWriteTransaction, entityType EntityType, entityID string) (*Result, error) {
	if err := Delete(ctx, txn, entityType, entityID); err != nil {
		return nil, fmt.Errorf("pin.Rotate: delete old: %w", err)
	}
	result, err := GenerateUnique(ctx, txn, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("pin.Rotate: generate new: %w", err)
	}
	return result, nil
}
