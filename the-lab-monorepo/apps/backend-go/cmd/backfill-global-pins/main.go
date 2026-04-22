// Command backfill-global-pins migrates legacy driver/staff PINs into the
// GlobalPins table. Because bcrypt hashes are one-way, the original plaintext
// cannot be recovered. This tool generates a NEW 8-digit PIN for every entity
// that has a PinHash but no GlobalPins row, writes the GlobalPins entry, and
// updates the entity's PinHash.
//
// Output: a CSV on stdout with (entity_type, entity_id, new_pin) so operators
// can distribute new credentials.
//
// Usage:
//
//	SPANNER_EMULATOR_HOST=localhost:9010 go run ./cmd/backfill-global-pins
//	SPANNER_DB=projects/X/instances/Y/databases/Z go run ./cmd/backfill-global-pins
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"backend-go/pkg/pin"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func main() {
	db := os.Getenv("SPANNER_DB")
	if db == "" {
		db = "projects/the-lab-project/instances/lab-instance/databases/the-lab-db"
	}

	ctx := context.Background()
	client, err := spanner.NewClient(ctx, db)
	if err != nil {
		log.Fatalf("spanner.NewClient: %v", err)
	}
	defer client.Close()

	fmt.Println("entity_type,entity_id,new_pin")

	backfillTable(ctx, client, "Drivers", "DriverId", "PinHash", pin.EntityDriver)
	backfillTable(ctx, client, "WarehouseStaff", "WorkerId", "PinHash", pin.EntityWarehouseStaff)
	backfillTable(ctx, client, "FactoryStaff", "StaffId", "PinHash", pin.EntityFactoryStaff)

	log.Println("backfill complete")
}

// backfillTable scans every row in `table` that has a non-empty PinHash and
// registers it in GlobalPins. Entities already in GlobalPins are skipped.
func backfillTable(ctx context.Context, client *spanner.Client, table, idCol, hashCol string, entityType pin.EntityType) {
	// Collect entity IDs that have a PinHash.
	stmt := spanner.Statement{
		SQL: fmt.Sprintf("SELECT %s, %s FROM %s WHERE %s IS NOT NULL AND %s != ''", idCol, hashCol, table, hashCol, hashCol),
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	type entity struct {
		id      string
		pinHash string
	}
	var entities []entity

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("scan %s: %v", table, err)
		}
		var e entity
		if err := row.Columns(&e.id, &e.pinHash); err != nil {
			log.Fatalf("parse %s row: %v", table, err)
		}
		entities = append(entities, e)
	}

	if len(entities) == 0 {
		log.Printf("[%s] no entities with PinHash — skipping", table)
		return
	}
	log.Printf("[%s] found %d entities with PinHash", table, len(entities))

	// For each entity, check if a GlobalPins row already exists.
	// If not, rotate to a new PIN.
	for _, e := range entities {
		if hasGlobalPin(ctx, client, entityType, e.id) {
			log.Printf("[%s] %s already in GlobalPins — skipping", table, e.id)
			continue
		}

		// Generate new PIN inside a RW txn.
		var result *pin.Result
		_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			var genErr error
			result, genErr = pin.GenerateUnique(ctx, txn, entityType, e.id)
			if genErr != nil {
				return genErr
			}
			return txn.BufferWrite([]*spanner.Mutation{
				spanner.Update(table,
					[]string{idCol, hashCol},
					[]interface{}{e.id, result.BcryptHash}),
			})
		})
		if err != nil {
			log.Printf("[%s] FAILED %s: %v", table, e.id, err)
			continue
		}

		// CSV line: entity_type,entity_id,new_pin
		fmt.Printf("%s,%s,%s\n", entityType, e.id, result.Plaintext)
		log.Printf("[%s] rotated %s → new PIN issued", table, e.id)
	}
}

// hasGlobalPin checks whether a GlobalPins row exists for the given entity.
func hasGlobalPin(ctx context.Context, client *spanner.Client, entityType pin.EntityType, entityID string) bool {
	stmt := spanner.Statement{
		SQL:    "SELECT 1 FROM GlobalPins WHERE EntityType = @et AND EntityId = @eid LIMIT 1",
		Params: map[string]interface{}{"et": string(entityType), "eid": entityID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	_, err := iter.Next()
	return err != iterator.Done
}

// sha256Hex computes SHA-256 of a string. Unused currently but retained
// in case a future mode needs to register known-plaintext legacy PINs.
func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
