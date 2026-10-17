package storage

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/api/iterator"
)

var (
	ErrDossierSealed   = errors.New("dossier is sealed and cannot be modified")
	ErrDossierNotFound = errors.New("dossier not found")
)

const (
	DossierStatusOpen   = "OPEN"
	DossierStatusSealed = "SEALED"
)

type EvidenceDossier struct {
	DossierID  string             `spanner:"DossierId"`
	TargetID   string             `spanner:"TargetId"`
	TargetType string             `spanner:"TargetType"`
	Status     string             `spanner:"Status"`
	SealedAt   spanner.NullTime   `spanner:"SealedAt"`
	SealedHash spanner.NullString `spanner:"SealedHash"`
	CreatedAt  time.Time          `spanner:"CreatedAt"`
}

type EvidenceItem struct {
	DossierID      string              `spanner:"DossierId"`
	ItemID         string              `spanner:"ItemId"`
	ItemType       string              `spanner:"ItemType"`
	StorageURI     string              `spanner:"StorageUri"`
	FileHash       string              `spanner:"FileHash"`
	MimeType       string              `spanner:"MimeType"`
	SizeBytes      int64               `spanner:"SizeBytes"`
	UploaderUserID string              `spanner:"UploaderUserId"`
	CapturedAt     spanner.NullTime    `spanner:"CapturedAt"`
	Latitude       spanner.NullFloat64 `spanner:"Latitude"`
	Longitude      spanner.NullFloat64 `spanner:"Longitude"`
	CreatedAt      time.Time           `spanner:"CreatedAt"`
}

// Vault manages EvidenceDossiers in Spanner.
type Vault struct {
	client *spanner.Client
}

func NewVault(client *spanner.Client) *Vault {
	return &Vault{client: client}
}

// CreateDossier creates a new OPEN dossier for a target entity.
func (v *Vault) CreateDossier(ctx context.Context, targetID, targetType string) (*EvidenceDossier, error) {
	dossier := &EvidenceDossier{
		DossierID:  uuid.NewString(),
		TargetID:   targetID,
		TargetType: targetType,
		Status:     DossierStatusOpen,
		CreatedAt:  time.Now(),
	}

	m := spanner.Insert("EvidenceDossiers",
		[]string{"DossierId", "TargetId", "TargetType", "Status", "CreatedAt"},
		[]interface{}{dossier.DossierID, dossier.TargetID, dossier.TargetType, dossier.Status, spanner.CommitTimestamp},
	)

	_, err := v.client.Apply(ctx, []*spanner.Mutation{m})
	if err != nil {
		return nil, fmt.Errorf("apply dossier insert: %w", err)
	}

	return dossier, nil
}

// AddItem adds an EvidenceItem to a dossier, failing if the dossier is SEALED.
func (v *Vault) AddItem(ctx context.Context, dossierID string, req *EvidenceItem) (*EvidenceItem, error) {
	req.DossierID = dossierID
	if req.ItemID == "" {
		req.ItemID = uuid.NewString()
	}

	_, err := v.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "EvidenceDossiers", spanner.Key{dossierID}, []string{"Status"})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return ErrDossierNotFound
			}
			return err
		}
		var status string
		if err := row.Columns(&status); err != nil {
			return err
		}
		if status == DossierStatusSealed {
			return ErrDossierSealed
		}

		m := spanner.Insert("EvidenceItems",
			[]string{
				"DossierId", "ItemId", "ItemType", "StorageUri", "FileHash",
				"MimeType", "SizeBytes", "UploaderUserId", "CapturedAt",
				"Latitude", "Longitude", "CreatedAt",
			},
			[]interface{}{
				req.DossierID, req.ItemID, req.ItemType, req.StorageURI, req.FileHash,
				req.MimeType, req.SizeBytes, req.UploaderUserID, req.CapturedAt,
				req.Latitude, req.Longitude, spanner.CommitTimestamp,
			},
		)
		return txn.BufferWrite([]*spanner.Mutation{m})
	})

	if err != nil {
		return nil, fmt.Errorf("add item transaction: %w", err)
	}

	return req, nil
}

// SealDossier locks the dossier and computes its SealedHash from all its items.
func (v *Vault) SealDossier(ctx context.Context, dossierID string) error {
	_, err := v.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "EvidenceDossiers", spanner.Key{dossierID}, []string{"Status"})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return ErrDossierNotFound
			}
			return err
		}
		var status string
		if err := row.Columns(&status); err != nil {
			return err
		}
		if status == DossierStatusSealed {
			return ErrDossierSealed
		}

		stmt := spanner.Statement{
			SQL:    `SELECT FileHash FROM EvidenceItems WHERE DossierId = @dossierId`,
			Params: map[string]interface{}{"dossierId": dossierID},
		}
		iter := txn.Query(ctx, stmt)
		var hashes []string
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}
			var h string
			if err := row.Columns(&h); err != nil {
				return err
			}
			hashes = append(hashes, h)
		}

		sort.Strings(hashes)

		hasher := sha256.New()
		for _, h := range hashes {
			hasher.Write([]byte(h))
		}
		sealedHash := fmt.Sprintf("%x", hasher.Sum(nil))

		m := spanner.Update("EvidenceDossiers",
			[]string{"DossierId", "Status", "SealedAt", "SealedHash"},
			[]interface{}{dossierID, DossierStatusSealed, spanner.CommitTimestamp, sealedHash},
		)

		return txn.BufferWrite([]*spanner.Mutation{m})
	})

	return err
}

// GetDossier fetches the dossier and its items.
func (v *Vault) GetDossier(ctx context.Context, dossierID string) (*EvidenceDossier, []*EvidenceItem, error) {
	ro := v.client.ReadOnlyTransaction()
	defer ro.Close()

	row, err := ro.ReadRow(ctx, "EvidenceDossiers", spanner.Key{dossierID},
		[]string{"DossierId", "TargetId", "TargetType", "Status", "SealedAt", "SealedHash", "CreatedAt"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return nil, nil, ErrDossierNotFound
		}
		return nil, nil, err
	}
	var dossier EvidenceDossier
	if err := row.ToStruct(&dossier); err != nil {
		return nil, nil, err
	}

	stmt := spanner.Statement{
		SQL:    `SELECT * FROM EvidenceItems WHERE DossierId = @dossierId ORDER BY CreatedAt ASC`,
		Params: map[string]interface{}{"dossierId": dossierID},
	}
	iter := ro.Query(ctx, stmt)
	var items []*EvidenceItem
	for {
		itemRow, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		var item EvidenceItem
		if err := itemRow.ToStruct(&item); err != nil {
			return nil, nil, err
		}
		items = append(items, &item)
	}

	return &dossier, items, nil
}
