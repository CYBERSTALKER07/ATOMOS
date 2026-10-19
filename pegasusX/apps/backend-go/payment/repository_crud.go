package payment

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// Payer represents a billing profile entity.
type Payer struct {
	PayerID        string    `json:"payer_id" spanner:"PayerId"`
	Name           string    `json:"name" spanner:"Name"`
	Email          string    `json:"email" spanner:"Email"`
	Phone          *string   `json:"phone,omitempty" spanner:"Phone"`
	BillingAddress *string   `json:"billing_address,omitempty" spanner:"BillingAddress"`
	TaxID          *string   `json:"tax_id,omitempty" spanner:"TaxId"`
	IsActive       bool      `json:"is_active" spanner:"IsActive"`
	CreatedAt      time.Time `json:"created_at" spanner:"CreatedAt"`
	UpdatedAt      time.Time `json:"updated_at" spanner:"UpdatedAt"`
}

// CreatePayer inserts a new payer billing profile.
func (r *SpannerRepository) CreatePayer(ctx context.Context, p Payer) error {
	p.CreatedAt = spanner.CommitTimestamp
	p.UpdatedAt = spanner.CommitTimestamp
	m, err := spanner.InsertStruct("Payers", p)
	if err != nil {
		return err
	}
	_, err = r.client.Apply(ctx, []*spanner.Mutation{m})
	return err
}

// GetPayer retrieves a payer billing profile by ID.
func (r *SpannerRepository) GetPayer(ctx context.Context, payerID string) (Payer, error) {
	row, err := r.client.Single().ReadRow(ctx, "Payers", spanner.Key{payerID}, []string{
		"PayerId", "Name", "Email", "Phone", "BillingAddress", "TaxId",
		"IsActive", "CreatedAt", "UpdatedAt",
	})
	if err != nil {
		return Payer{}, err
	}
	var p Payer
	if err := row.ToStruct(&p); err != nil {
		return Payer{}, err
	}
	return p, nil
}

// UpdatePayer updates an existing payer billing profile.
func (r *SpannerRepository) UpdatePayer(ctx context.Context, p Payer) error {
	p.UpdatedAt = spanner.CommitTimestamp
	m, err := spanner.UpdateStruct("Payers", p)
	if err != nil {
		return err
	}
	_, err = r.client.Apply(ctx, []*spanner.Mutation{m})
	return err
}

// ListPayers lists all active payers (limited).
func (r *SpannerRepository) ListPayers(ctx context.Context, limit, offset int) ([]Payer, error) {
	stmt := spanner.Statement{
		SQL: `SELECT * FROM Payers WHERE IsActive = TRUE ORDER BY CreatedAt DESC LIMIT @limit OFFSET @offset`,
		Params: map[string]interface{}{
			"limit":  limit,
			"offset": offset,
		},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var payers []Payer
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var p Payer
		if err := row.ToStruct(&p); err != nil {
			return nil, err
		}
		payers = append(payers, p)
	}
	return payers, nil
}
