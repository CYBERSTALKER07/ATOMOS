package promotion

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// AudienceResolver resolves retailer-side promotion subscription audiences.
// Live ALL-scope fanout is O(1) via ws.SupplierPromoRoom — not per-retailer loops.
type AudienceResolver struct {
	spanner *spanner.Client
}

// NewAudienceResolver constructs a promotion audience resolver.
func NewAudienceResolver(client *spanner.Client) *AudienceResolver {
	return &AudienceResolver{spanner: client}
}

// CartSupplierIDs returns distinct suppliers present in the retailer's active cart.
// Index-scoped on RetailerId; cardinality is bounded by cart contents (no SQL cap).
func (r *AudienceResolver) CartSupplierIDs(ctx context.Context, retailerID string) ([]string, error) {
	if r == nil || r.spanner == nil || retailerID == "" {
		return nil, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT DISTINCT SupplierId
			FROM CartItems@{FORCE_INDEX=Idx_CartItems_ByRetailerSupplier}
			WHERE RetailerId = @rid AND SupplierId IS NOT NULL`,
		Params: map[string]any{
			"rid": retailerID,
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
