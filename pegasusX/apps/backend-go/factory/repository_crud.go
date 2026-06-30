package factory

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// Factory represents the factory entity.
type Factory struct {
	FactoryID  string    `json:"factory_id" spanner:"FactoryId"`
	SupplierID string    `json:"supplier_id" spanner:"SupplierId"`
	Name       string    `json:"name" spanner:"Name"`
	Lat        *float64  `json:"lat,omitempty" spanner:"Lat"`
	Lng        *float64  `json:"lng,omitempty" spanner:"Lng"`
	H3Cell     *string   `json:"h3_cell,omitempty" spanner:"H3Cell"`
	Address    *string   `json:"address,omitempty" spanner:"Address"`
	PlaceID    *string   `json:"place_id,omitempty" spanner:"PlaceId"`
	IsActive   bool      `json:"is_active" spanner:"IsActive"`
	CreatedAt  time.Time `json:"created_at" spanner:"CreatedAt"`
	UpdatedAt  time.Time `json:"updated_at" spanner:"UpdatedAt"`
}

// CreateFactory inserts a new factory record.
func (r *SpannerRepository) CreateFactory(ctx context.Context, f Factory) error {
	f.CreatedAt = spanner.CommitTimestamp
	f.UpdatedAt = spanner.CommitTimestamp
	m, err := spanner.InsertStruct("Factories", f)
	if err != nil {
		return err
	}
	_, err = r.client.Apply(ctx, []*spanner.Mutation{m})
	return err
}

// GetFactory retrieves a factory by its ID.
func (r *SpannerRepository) GetFactory(ctx context.Context, factoryID string) (Factory, error) {
	row, err := r.client.Single().ReadRow(ctx, "Factories", spanner.Key{factoryID}, []string{
		"FactoryId", "SupplierId", "Name", "Lat", "Lng", "H3Cell", "Address", "PlaceId",
		"IsActive", "CreatedAt", "UpdatedAt",
	})
	if err != nil {
		return Factory{}, err
	}
	var f Factory
	if err := row.ToStruct(&f); err != nil {
		return Factory{}, err
	}
	return f, nil
}

// UpdateFactory updates an existing factory record.
func (r *SpannerRepository) UpdateFactory(ctx context.Context, f Factory) error {
	f.UpdatedAt = spanner.CommitTimestamp
	m, err := spanner.UpdateStruct("Factories", f)
	if err != nil {
		return err
	}
	_, err = r.client.Apply(ctx, []*spanner.Mutation{m})
	return err
}

// ListFactories returns a list of factories filtered by supplier ID.
func (r *SpannerRepository) ListFactories(ctx context.Context, supplierID string, limit, offset int) ([]Factory, error) {
	stmt := spanner.Statement{
		SQL: `SELECT * FROM Factories WHERE SupplierId = @supplierId LIMIT @limit OFFSET @offset`,
		Params: map[string]interface{}{
			"supplierId": supplierID,
			"limit":      limit,
			"offset":     offset,
		},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var factories []Factory
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var f Factory
		if err := row.ToStruct(&f); err != nil {
			return nil, err
		}
		factories = append(factories, f)
	}
	return factories, nil
}
