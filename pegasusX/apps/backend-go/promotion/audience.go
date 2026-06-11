package promotion

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const (
	maxEngagedRetailers = 5000
	maxCartSuppliers    = 25
)

// AudienceResolver finds retailer audiences for ALL-scope promotion fanout.
type AudienceResolver struct {
	spanner *spanner.Client
}

// NewAudienceResolver constructs a promotion audience resolver.
func NewAudienceResolver(client *spanner.Client) *AudienceResolver {
	return &AudienceResolver{spanner: client}
}

// EngagedRetailerIDs returns distinct retailers with order history for the supplier.
func (r *AudienceResolver) EngagedRetailerIDs(ctx context.Context, supplierID string) ([]string, error) {
	if r == nil || r.spanner == nil || supplierID == "" {
		return nil, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT DISTINCT RetailerId
			FROM Orders@{FORCE_INDEX=Idx_Orders_BySupplierCreated}
			WHERE SupplierId = @sid AND RetailerId IS NOT NULL
			LIMIT @lim`,
		Params: map[string]any{
			"sid": supplierID,
			"lim": maxEngagedRetailers,
		},
	}
	iter := r.spanner.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
	defer iter.Stop()

	ids := make([]string, 0, 64)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("engaged retailers for supplier %s: %w", supplierID, err)
		}
		var retailerID spanner.NullString
		if err := row.Columns(&retailerID); err != nil {
			return nil, fmt.Errorf("scan engaged retailer: %w", err)
		}
		if retailerID.Valid && retailerID.StringVal != "" {
			ids = append(ids, retailerID.StringVal)
		}
	}
	return ids, nil
}

// CartSupplierIDs returns suppliers present in the retailer's active cart.
func (r *AudienceResolver) CartSupplierIDs(ctx context.Context, retailerID string) ([]string, error) {
	if r == nil || r.spanner == nil || retailerID == "" {
		return nil, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT DISTINCT SupplierId
			FROM CartItems@{FORCE_INDEX=Idx_CartItems_ByRetailerSupplier}
			WHERE RetailerId = @rid
			LIMIT @lim`,
		Params: map[string]any{
			"rid": retailerID,
			"lim": maxCartSuppliers,
		},
	}
	iter := r.spanner.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
	defer iter.Stop()

	ids := make([]string, 0, 8)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("cart suppliers for retailer %s: %w", retailerID, err)
		}
		var supplierID spanner.NullString
		if err := row.Columns(&supplierID); err != nil {
			return nil, fmt.Errorf("scan cart supplier: %w", err)
		}
		if supplierID.Valid && supplierID.StringVal != "" {
			ids = append(ids, supplierID.StringVal)
		}
	}
	return ids, nil
}
