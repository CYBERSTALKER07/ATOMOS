package retailer

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// CartItem mirrors the CartItems Spanner row.
type CartItem struct {
	CartItemID    string    `json:"cart_item_id"`
	RetailerID    string    `json:"retailer_id"`
	SupplierID    string    `json:"supplier_id"`
	ProductID     string    `json:"product_id"`
	Quantity      int64     `json:"quantity"`
	PriceSnapshot int64     `json:"price_snapshot"`
	Currency      string    `json:"currency"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CartRepository defines cart data access.
type CartRepository interface {
	ListByRetailer(ctx context.Context, retailerID, supplierID string) ([]CartItem, error)
	UpsertItems(ctx context.Context, items []CartItem) error
	ClearCart(ctx context.Context, retailerID, supplierID string) error
}

// SpannerCartRepository implements CartRepository backed by Cloud Spanner.
type SpannerCartRepository struct {
	client *spanner.Client
}

// NewSpannerCartRepository creates a Spanner-backed cart repository.
func NewSpannerCartRepository(client *spanner.Client) *SpannerCartRepository {
	return &SpannerCartRepository{client: client}
}

// ListByRetailer returns all cart items for a retailer + supplier pair.
func (r *SpannerCartRepository) ListByRetailer(ctx context.Context, retailerID, supplierID string) ([]CartItem, error) {
	stmt := spanner.Statement{
		SQL:    "SELECT CartItemId, RetailerId, SupplierId, ProductId, Quantity, PriceSnapshot, Currency, UpdatedAt FROM CartItems@{FORCE_INDEX=Idx_CartItems_ByRetailerSupplier} WHERE RetailerId = @rid AND SupplierId = @sid",
		Params: map[string]any{"rid": retailerID, "sid": supplierID},
	}
	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(5 * time.Second)).Query(ctx, stmt)
	defer iter.Stop()

	var items []CartItem
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list cart items for retailer %s: %w", retailerID, err)
		}
		var ci CartItem
		if err := row.Columns(&ci.CartItemID, &ci.RetailerID, &ci.SupplierID, &ci.ProductID,
			&ci.Quantity, &ci.PriceSnapshot, &ci.Currency, &ci.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan cart item: %w", err)
		}
		items = append(items, ci)
	}
	return items, nil
}

// UpsertItems writes cart items inside a ReadWriteTransaction.
func (r *SpannerCartRepository) UpsertItems(ctx context.Context, items []CartItem) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := make([]*spanner.Mutation, 0, len(items))
		for _, ci := range items {
			m := spanner.InsertOrUpdateMap("CartItems", map[string]any{
				"CartItemId":    ci.CartItemID,
				"RetailerId":    ci.RetailerID,
				"SupplierId":    ci.SupplierID,
				"ProductId":     ci.ProductID,
				"Quantity":      ci.Quantity,
				"PriceSnapshot": ci.PriceSnapshot,
				"Currency":      ci.Currency,
				"UpdatedAt":     spanner.CommitTimestamp,
			})
			mutations = append(mutations, m)
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("upsert cart items: %w", err)
	}
	return nil
}

// ClearCart deletes all cart items for a retailer + supplier pair.
func (r *SpannerCartRepository) ClearCart(ctx context.Context, retailerID, supplierID string) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL:    "SELECT CartItemId FROM CartItems@{FORCE_INDEX=Idx_CartItems_ByRetailerSupplier} WHERE RetailerId = @rid AND SupplierId = @sid",
			Params: map[string]any{"rid": retailerID, "sid": supplierID},
		}
		iter := txn.Query(ctx, stmt)
		defer iter.Stop()

		var mutations []*spanner.Mutation
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return fmt.Errorf("list cart items for delete: %w", err)
			}
			var cartItemID string
			if err := row.Columns(&cartItemID); err != nil {
				return fmt.Errorf("scan cart item id: %w", err)
			}
			mutations = append(mutations, spanner.Delete("CartItems", spanner.Key{cartItemID}))
		}
		if len(mutations) > 0 {
			return txn.BufferWrite(mutations)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("clear cart for retailer %s: %w", retailerID, err)
	}
	return nil
}
