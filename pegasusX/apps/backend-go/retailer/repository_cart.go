package retailer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
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
	ListByRetailerAll(ctx context.Context, retailerID string) ([]CartItem, error)
	UpsertItems(ctx context.Context, items []CartItem) error
	ClearCart(ctx context.Context, retailerID, supplierID string) error
	ClearCartAll(ctx context.Context, retailerID string) error
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
	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(5*time.Second)).Query(ctx, stmt)
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

// ListByRetailerAll returns cart items across all suppliers for a retailer.
func (r *SpannerCartRepository) ListByRetailerAll(ctx context.Context, retailerID string) ([]CartItem, error) {
	stmt := spanner.Statement{
		SQL: `SELECT CartItemId, RetailerId, SupplierId, ProductId, Quantity, PriceSnapshot, Currency, UpdatedAt
		      FROM CartItems WHERE RetailerId = @rid
		      ORDER BY UpdatedAt DESC`,
		Params: map[string]any{"rid": retailerID},
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
			return nil, fmt.Errorf("list all cart items for retailer %s: %w", retailerID, err)
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

// UpsertItems writes cart items + CART_SYNC_UPDATED outbox in one ReadWriteTransaction (B3 M-P1-3).
func (r *SpannerCartRepository) UpsertItems(ctx context.Context, items []CartItem) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := make([]*spanner.Mutation, 0, len(items))
		retailerID := ""
		supplierID := ""
		for _, ci := range items {
			if retailerID == "" {
				retailerID = strings.TrimSpace(ci.RetailerID)
			}
			if supplierID == "" {
				supplierID = strings.TrimSpace(ci.SupplierID)
			}
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
		if len(mutations) > 0 {
			if err := txn.BufferWrite(mutations); err != nil {
				return err
			}
		}
		return emitCartSyncOutbox(ctx, txn, retailerID, supplierID, len(items))
	})
	if err != nil {
		return fmt.Errorf("upsert cart items: %w", err)
	}
	return nil
}

// ClearCart deletes all cart items for a retailer + supplier pair (+ cart sync outbox).
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
			if err := txn.BufferWrite(mutations); err != nil {
				return err
			}
		}
		return emitCartSyncOutbox(ctx, txn, retailerID, supplierID, 0)
	})
	if err != nil {
		return fmt.Errorf("clear cart for retailer %s: %w", retailerID, err)
	}
	return nil
}

// ClearCartAll deletes all cart items for a retailer across suppliers (+ cart sync outbox).
func (r *SpannerCartRepository) ClearCartAll(ctx context.Context, retailerID string) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL:    "SELECT CartItemId FROM CartItems WHERE RetailerId = @rid",
			Params: map[string]any{"rid": retailerID},
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
				return fmt.Errorf("list cart items for delete-all: %w", err)
			}
			var cartItemID string
			if err := row.Columns(&cartItemID); err != nil {
				return fmt.Errorf("scan cart item id: %w", err)
			}
			mutations = append(mutations, spanner.Delete("CartItems", spanner.Key{cartItemID}))
		}
		if len(mutations) > 0 {
			if err := txn.BufferWrite(mutations); err != nil {
				return err
			}
		}
		return emitCartSyncOutbox(ctx, txn, retailerID, "", 0)
	})
	if err != nil {
		return fmt.Errorf("clear all cart items for retailer %s: %w", retailerID, err)
	}
	return nil
}

// emitCartSyncOutbox buffers CART_SYNC_UPDATED for multi-device RetailerHub fanout.
func emitCartSyncOutbox(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID, supplierID string, itemCount int) error {
	retailerID = strings.TrimSpace(retailerID)
	if txn == nil || retailerID == "" {
		return nil
	}
	payload := map[string]any{
		"type":        events.EventCartSyncUpdated,
		"timestamp":   time.Now().UTC().Format(time.RFC3339Nano),
		"retailer_id": retailerID,
		"supplier_id": strings.TrimSpace(supplierID),
		"item_count":  itemCount,
	}
	buf := outbox.NewSpannerTxnBuffer(txn)
	if err := outbox.EmitJSON(ctx, buf, events.AggregateRetailer, retailerID, events.TopicMain, payload); err != nil {
		return err
	}
	return buf.Flush(ctx)
}
