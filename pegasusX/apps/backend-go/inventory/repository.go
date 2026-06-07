package inventory

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// Level mirrors the InventoryLevels Spanner row.
type Level struct {
	InventoryID      string    `json:"inventory_id" spanner:"InventoryId"`
	ProductID        string    `json:"product_id" spanner:"ProductId"`
	WarehouseID      string    `json:"warehouse_id" spanner:"WarehouseId"`
	SupplierID       string    `json:"supplier_id" spanner:"SupplierId"`
	QuantityOnHand   int64     `json:"quantity_on_hand" spanner:"QuantityOnHand"`
	QuantityReserved int64     `json:"quantity_reserved" spanner:"QuantityReserved"`
	ReorderThreshold int64     `json:"reorder_threshold" spanner:"ReorderThreshold"`
	Version          int64     `json:"version" spanner:"Version"`
	UpdatedAt        time.Time `json:"updated_at" spanner:"UpdatedAt"`
}

// Available returns the effective stock available for new orders.
func (l Level) Available() int64 {
	avail := l.QuantityOnHand - l.QuantityReserved
	if avail < 0 {
		return 0
	}
	return avail
}

// IsBelowThreshold returns true when on-hand stock is at or below the reorder point.
func (l Level) IsBelowThreshold() bool {
	return l.QuantityOnHand <= l.ReorderThreshold
}

// Repository defines the data access contract for inventory operations.
type Repository interface {
	ListByWarehouse(ctx context.Context, warehouseID string) ([]Level, error)
	ListBySupplier(ctx context.Context, supplierID string) ([]Level, error)
	Get(ctx context.Context, inventoryID string) (*Level, error)
	GetByWarehouseProduct(ctx context.Context, warehouseID, productID string) (*Level, error)
	Upsert(ctx context.Context, l Level) error
	AdjustStock(ctx context.Context, inventoryID string, delta int64, expectedVersion int64) error
	ReserveForOrder(ctx context.Context, inventoryID string, quantity int64, expectedVersion int64) error
	ReleaseReservation(ctx context.Context, inventoryID string, quantity int64, expectedVersion int64) error
}

// SpannerRepository implements Repository backed by Cloud Spanner.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository creates a Spanner-backed inventory repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

var levelColumns = []string{
	"InventoryId", "ProductId", "WarehouseId", "SupplierId",
	"QuantityOnHand", "QuantityReserved", "ReorderThreshold", "Version", "UpdatedAt",
}

func scanLevel(row *spanner.Row) (Level, error) {
	var l Level
	if err := row.Columns(&l.InventoryID, &l.ProductID, &l.WarehouseID, &l.SupplierID,
		&l.QuantityOnHand, &l.QuantityReserved, &l.ReorderThreshold, &l.Version, &l.UpdatedAt); err != nil {
		return Level{}, fmt.Errorf("scan inventory level: %w", err)
	}
	return l, nil
}

// ListByWarehouse returns all inventory rows for a warehouse.
func (r *SpannerRepository) ListByWarehouse(ctx context.Context, warehouseID string) ([]Level, error) {
	stmt := spanner.Statement{
		SQL:    "SELECT " + colList() + " FROM InventoryLevels WHERE WarehouseId = @wid ORDER BY ProductId",
		Params: map[string]any{"wid": warehouseID},
	}
	return r.queryLevels(ctx, stmt, "warehouse "+warehouseID)
}

// ListBySupplier returns all inventory rows for a supplier.
func (r *SpannerRepository) ListBySupplier(ctx context.Context, supplierID string) ([]Level, error) {
	stmt := spanner.Statement{
		SQL:    "SELECT " + colList() + " FROM InventoryLevels WHERE SupplierId = @sid ORDER BY WarehouseId, ProductId",
		Params: map[string]any{"sid": supplierID},
	}
	return r.queryLevels(ctx, stmt, "supplier "+supplierID)
}

// Get reads a single inventory row by PK.
func (r *SpannerRepository) Get(ctx context.Context, inventoryID string) (*Level, error) {
	row, err := r.client.Single().ReadRow(ctx, "InventoryLevels", spanner.Key{inventoryID}, levelColumns)
	if err != nil {
		return nil, fmt.Errorf("get inventory %s: %w", inventoryID, err)
	}
	l, err := scanLevel(row)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// GetByWarehouseProduct finds the inventory row for a (warehouse, product) pair.
func (r *SpannerRepository) GetByWarehouseProduct(ctx context.Context, warehouseID, productID string) (*Level, error) {
	stmt := spanner.Statement{
		SQL:    "SELECT " + colList() + " FROM InventoryLevels@{FORCE_INDEX=Idx_InventoryLevels_ByWarehouseProduct} WHERE WarehouseId = @wid AND ProductId = @pid LIMIT 1",
		Params: map[string]any{"wid": warehouseID, "pid": productID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return nil, fmt.Errorf("get inventory for warehouse %s product %s: %w", warehouseID, productID, err)
	}
	l, err := scanLevel(row)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// Upsert creates or replaces an inventory level row.
func (r *SpannerRepository) Upsert(ctx context.Context, l Level) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		m := spanner.InsertOrUpdateMap("InventoryLevels", map[string]any{
			"InventoryId":      l.InventoryID,
			"ProductId":        l.ProductID,
			"WarehouseId":      l.WarehouseID,
			"SupplierId":       l.SupplierID,
			"QuantityOnHand":   l.QuantityOnHand,
			"QuantityReserved": l.QuantityReserved,
			"ReorderThreshold": l.ReorderThreshold,
			"Version":          l.Version,
			"UpdatedAt":        spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	if err != nil {
		return fmt.Errorf("upsert inventory %s: %w", l.InventoryID, err)
	}
	return nil
}

// AdjustStock adds delta to QuantityOnHand with optimistic concurrency.
func (r *SpannerRepository) AdjustStock(ctx context.Context, inventoryID string, delta int64, expectedVersion int64) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "InventoryLevels", spanner.Key{inventoryID},
			[]string{"QuantityOnHand", "Version"})
		if readErr != nil {
			return fmt.Errorf("read inventory %s: %w", inventoryID, readErr)
		}
		var onHand, version int64
		if err := row.Columns(&onHand, &version); err != nil {
			return fmt.Errorf("scan inventory %s: %w", inventoryID, err)
		}
		if version != expectedVersion {
			return fmt.Errorf("inventory %s version conflict: expected %d got %d", inventoryID, expectedVersion, version)
		}
		newOnHand := onHand + delta
		if newOnHand < 0 {
			newOnHand = 0
		}
		m := spanner.UpdateMap("InventoryLevels", map[string]any{
			"InventoryId":    inventoryID,
			"QuantityOnHand": newOnHand,
			"Version":        version + 1,
			"UpdatedAt":      spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	if err != nil {
		return fmt.Errorf("adjust stock %s: %w", inventoryID, err)
	}
	return nil
}

// ReserveForOrder moves quantity from on-hand to reserved.
func (r *SpannerRepository) ReserveForOrder(ctx context.Context, inventoryID string, quantity int64, expectedVersion int64) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "InventoryLevels", spanner.Key{inventoryID},
			[]string{"QuantityOnHand", "QuantityReserved", "Version"})
		if readErr != nil {
			return fmt.Errorf("read inventory %s: %w", inventoryID, readErr)
		}
		var onHand, reserved, version int64
		if err := row.Columns(&onHand, &reserved, &version); err != nil {
			return fmt.Errorf("scan inventory %s: %w", inventoryID, err)
		}
		if version != expectedVersion {
			return fmt.Errorf("inventory %s version conflict: expected %d got %d", inventoryID, expectedVersion, version)
		}
		if onHand < quantity {
			return fmt.Errorf("inventory %s insufficient stock: %d available, %d requested", inventoryID, onHand, quantity)
		}
		m := spanner.UpdateMap("InventoryLevels", map[string]any{
			"InventoryId":      inventoryID,
			"QuantityOnHand":   onHand - quantity,
			"QuantityReserved": reserved + quantity,
			"Version":          version + 1,
			"UpdatedAt":        spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	if err != nil {
		return fmt.Errorf("reserve stock %s: %w", inventoryID, err)
	}
	return nil
}

// ReleaseReservation moves quantity from reserved back to on-hand.
func (r *SpannerRepository) ReleaseReservation(ctx context.Context, inventoryID string, quantity int64, expectedVersion int64) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "InventoryLevels", spanner.Key{inventoryID},
			[]string{"QuantityOnHand", "QuantityReserved", "Version"})
		if readErr != nil {
			return fmt.Errorf("read inventory %s: %w", inventoryID, readErr)
		}
		var onHand, reserved, version int64
		if err := row.Columns(&onHand, &reserved, &version); err != nil {
			return fmt.Errorf("scan inventory %s: %w", inventoryID, err)
		}
		if version != expectedVersion {
			return fmt.Errorf("inventory %s version conflict: expected %d got %d", inventoryID, expectedVersion, version)
		}
		releaseQty := quantity
		if releaseQty > reserved {
			releaseQty = reserved
		}
		m := spanner.UpdateMap("InventoryLevels", map[string]any{
			"InventoryId":      inventoryID,
			"QuantityOnHand":   onHand + releaseQty,
			"QuantityReserved": reserved - releaseQty,
			"Version":          version + 1,
			"UpdatedAt":        spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	if err != nil {
		return fmt.Errorf("release reservation %s: %w", inventoryID, err)
	}
	return nil
}

func (r *SpannerRepository) queryLevels(ctx context.Context, stmt spanner.Statement, scope string) ([]Level, error) {
	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
	defer iter.Stop()
	var levels []Level
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list inventory for %s: %w", scope, err)
		}
		l, scanErr := scanLevel(row)
		if scanErr != nil {
			return nil, scanErr
		}
		levels = append(levels, l)
	}
	return levels, nil
}

func colList() string {
	return "InventoryId, ProductId, WarehouseId, SupplierId, QuantityOnHand, QuantityReserved, ReorderThreshold, Version, UpdatedAt"
}
