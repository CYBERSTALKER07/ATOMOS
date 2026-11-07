package inventory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
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
	ReserveTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, warehouseID, productID string, quantity int64) error
	ReleaseTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, warehouseID, productID string, quantity int64) error
	ReleaseReservation(ctx context.Context, inventoryID string, quantity int64, expectedVersion int64) error
	CreateHold(ctx context.Context, hold Hold) error
	ConfirmHold(ctx context.Context, holdID string) error
	ReleaseHold(ctx context.Context, holdID string) error
	ListExpiredHolds(ctx context.Context, now time.Time, limit int) ([]Hold, error)
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
	if err == iterator.Done {
		return nil, nil
	}
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
		if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
			return err
		}
		aggID := l.WarehouseID
		if aggID == "" {
			aggID = l.InventoryID
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		if err := outbox.EmitJSON(ctx, buf, events.AggregateWarehouse, aggID, events.TopicMain, map[string]any{
			"type":         events.EventInventoryQuantityUpdated,
			"inventory_id": l.InventoryID,
			"warehouse_id": l.WarehouseID,
			"supplier_id":  l.SupplierID,
			"product_id":   l.ProductID,
			"quantity":     l.QuantityOnHand,
			"timestamp":    time.Now().UTC().Format(time.RFC3339Nano),
		}); err != nil {
			return err
		}
		return buf.Flush(ctx)
	})
	if err != nil {
		return fmt.Errorf("upsert inventory %s: %w", l.InventoryID, err)
	}
	return nil
}

// AdjustStock adds delta to QuantityOnHand with optimistic concurrency.
// B4: emits INVENTORY_QUANTITY_UPDATED in the same RW txn (supplier inventory plane).
func (r *SpannerRepository) AdjustStock(ctx context.Context, inventoryID string, delta int64, expectedVersion int64) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "InventoryLevels", spanner.Key{inventoryID},
			[]string{"QuantityOnHand", "Version", "ProductId", "WarehouseId", "SupplierId"})
		if readErr != nil {
			return fmt.Errorf("read inventory %s: %w", inventoryID, readErr)
		}
		var onHand, version int64
		var productID, warehouseID, supplierID string
		if err := row.Columns(&onHand, &version, &productID, &warehouseID, &supplierID); err != nil {
			return fmt.Errorf("scan inventory %s: %w", inventoryID, err)
		}
		if version != expectedVersion {
			return fmt.Errorf("inventory %s version conflict: expected %d got %d", inventoryID, expectedVersion, version)
		}
		newOnHand := onHand + delta
		if newOnHand < 0 {
			// Fail loudly: silently clamping negative stock to zero hides
			// oversells and corrupts availability math downstream.
			return fmt.Errorf("insufficient stock for %s: on_hand %d + delta %d would go negative", inventoryID, onHand, delta)
		}
		m := spanner.UpdateMap("InventoryLevels", map[string]any{
			"InventoryId":    inventoryID,
			"QuantityOnHand": newOnHand,
			"Version":        version + 1,
			"UpdatedAt":      spanner.CommitTimestamp,
		})
		if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
			return err
		}
		// Fanout key: warehouse when present (WMS room); else inventory id.
		aggID := warehouseID
		if aggID == "" {
			aggID = inventoryID
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		if err := outbox.EmitJSON(ctx, buf, events.AggregateWarehouse, aggID, events.TopicMain, map[string]any{
			"type":         events.EventInventoryQuantityUpdated,
			"inventory_id": inventoryID,
			"warehouse_id": warehouseID,
			"supplier_id":  supplierID,
			"product_id":   productID,
			"delta":        delta,
			"quantity":     newOnHand,
			"timestamp":    time.Now().UTC().Format(time.RFC3339Nano),
		}); err != nil {
			return err
		}
		return buf.Flush(ctx)
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
			[]string{"QuantityOnHand", "QuantityReserved", "Version", "WarehouseId", "SupplierId", "ProductId"})
		if readErr != nil {
			return fmt.Errorf("read inventory %s: %w", inventoryID, readErr)
		}
		var onHand, reserved, version int64
		var warehouseID, supplierID, productID string
		if err := row.Columns(&onHand, &reserved, &version, &warehouseID, &supplierID, &productID); err != nil {
			return fmt.Errorf("scan inventory %s: %w", inventoryID, err)
		}
		if version != expectedVersion {
			return fmt.Errorf("inventory %s version conflict: expected %d got %d", inventoryID, expectedVersion, version)
		}
		available := onHand - reserved
		if available < quantity {
			return fmt.Errorf("inventory %s insufficient stock: %d available, %d requested", inventoryID, available, quantity)
		}
		newReserved := reserved + quantity
		m := spanner.UpdateMap("InventoryLevels", map[string]any{
			"InventoryId":      inventoryID,
			"QuantityOnHand":   onHand,
			"QuantityReserved": newReserved,
			"Version":          version + 1,
			"UpdatedAt":        spanner.CommitTimestamp,
		})
		if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
			return err
		}
		aggID := warehouseID
		if aggID == "" {
			aggID = inventoryID
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		if err := outbox.EmitJSON(ctx, buf, events.AggregateWarehouse, aggID, events.TopicMain, map[string]any{
			"type":              events.EventInventoryQuantityUpdated,
			"inventory_id":      inventoryID,
			"warehouse_id":      warehouseID,
			"supplier_id":       supplierID,
			"product_id":        productID,
			"quantity":          onHand,
			"quantity_reserved": newReserved,
			"timestamp":         time.Now().UTC().Format(time.RFC3339Nano),
		}); err != nil {
			return err
		}
		return buf.Flush(ctx)
	})
	if err != nil {
		return fmt.Errorf("reserve stock %s: %w", inventoryID, err)
	}
	return nil
}

// ReserveTxn reserves stock within an existing transaction with version bump and outbox buffering.
func (r *SpannerRepository) ReserveTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, warehouseID, productID string, quantity int64) error {
	stmt := spanner.Statement{
		SQL:    "SELECT InventoryId, QuantityOnHand, QuantityReserved, Version, SupplierId FROM InventoryLevels@{FORCE_INDEX=Idx_InventoryLevels_ByWarehouseProduct} WHERE WarehouseId = @wid AND ProductId = @pid LIMIT 1",
		Params: map[string]any{"wid": warehouseID, "pid": productID},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return fmt.Errorf("inventory not found for warehouse %s product %s", warehouseID, productID)
	}
	if err != nil {
		return err
	}
	var inventoryID, supplierID string
	var onHand, reserved, version int64
	if err := row.Columns(&inventoryID, &onHand, &reserved, &version, &supplierID); err != nil {
		return err
	}
	if (onHand - reserved) < quantity {
		return fmt.Errorf("insufficient stock for product %s in warehouse %s", productID, warehouseID)
	}
	newReserved := reserved + quantity
	m := spanner.UpdateMap("InventoryLevels", map[string]any{
		"InventoryId":      inventoryID,
		"QuantityReserved": newReserved,
		"Version":          version + 1,
		"UpdatedAt":        spanner.CommitTimestamp,
	})
	if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
		return err
	}
	aggID := warehouseID
	if aggID == "" {
		aggID = inventoryID
	}
	buf := outbox.NewSpannerTxnBuffer(txn)
	if err := outbox.EmitJSON(ctx, buf, events.AggregateWarehouse, aggID, events.TopicMain, map[string]any{
		"type":              events.EventInventoryQuantityUpdated,
		"inventory_id":      inventoryID,
		"warehouse_id":      warehouseID,
		"supplier_id":       supplierID,
		"product_id":        productID,
		"quantity":          onHand,
		"quantity_reserved": newReserved,
		"timestamp":         time.Now().UTC().Format(time.RFC3339Nano),
	}); err != nil {
		return err
	}
	return buf.Flush(ctx)
}

// ReleaseTxn releases reserved stock within an existing transaction with version bump and outbox buffering.
func (r *SpannerRepository) ReleaseTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, warehouseID, productID string, quantity int64) error {
	stmt := spanner.Statement{
		SQL:    "SELECT InventoryId, QuantityOnHand, QuantityReserved, Version, SupplierId FROM InventoryLevels@{FORCE_INDEX=Idx_InventoryLevels_ByWarehouseProduct} WHERE WarehouseId = @wid AND ProductId = @pid LIMIT 1",
		Params: map[string]any{"wid": warehouseID, "pid": productID},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return fmt.Errorf("inventory not found for warehouse %s product %s", warehouseID, productID)
	}
	if err != nil {
		return err
	}
	var inventoryID, supplierID string
	var onHand, reserved, version int64
	if err := row.Columns(&inventoryID, &onHand, &reserved, &version, &supplierID); err != nil {
		return err
	}
	releaseQty := quantity
	if releaseQty > reserved {
		releaseQty = reserved
	}
	newReserved := reserved - releaseQty
	m := spanner.UpdateMap("InventoryLevels", map[string]any{
		"InventoryId":      inventoryID,
		"QuantityReserved": newReserved,
		"Version":          version + 1,
		"UpdatedAt":        spanner.CommitTimestamp,
	})
	if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
		return err
	}
	aggID := warehouseID
	if aggID == "" {
		aggID = inventoryID
	}
	buf := outbox.NewSpannerTxnBuffer(txn)
	if err := outbox.EmitJSON(ctx, buf, events.AggregateWarehouse, aggID, events.TopicMain, map[string]any{
		"type":              events.EventInventoryQuantityUpdated,
		"inventory_id":      inventoryID,
		"warehouse_id":      warehouseID,
		"supplier_id":       supplierID,
		"product_id":        productID,
		"quantity":          onHand,
		"quantity_reserved": newReserved,
		"timestamp":         time.Now().UTC().Format(time.RFC3339Nano),
	}); err != nil {
		return err
	}
	return buf.Flush(ctx)
}

// ReleaseReservation moves quantity from reserved back to on-hand.
func (r *SpannerRepository) ReleaseReservation(ctx context.Context, inventoryID string, quantity int64, expectedVersion int64) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "InventoryLevels", spanner.Key{inventoryID},
			[]string{"QuantityOnHand", "QuantityReserved", "Version", "WarehouseId", "SupplierId", "ProductId"})
		if readErr != nil {
			return fmt.Errorf("read inventory %s: %w", inventoryID, readErr)
		}
		var onHand, reserved, version int64
		var warehouseID, supplierID, productID string
		if err := row.Columns(&onHand, &reserved, &version, &warehouseID, &supplierID, &productID); err != nil {
			return fmt.Errorf("scan inventory %s: %w", inventoryID, err)
		}
		if version != expectedVersion {
			return fmt.Errorf("inventory %s version conflict: expected %d got %d", inventoryID, expectedVersion, version)
		}
		releaseQty := quantity
		if releaseQty > reserved {
			releaseQty = reserved
		}
		newReserved := reserved - releaseQty
		m := spanner.UpdateMap("InventoryLevels", map[string]any{
			"InventoryId":      inventoryID,
			"QuantityOnHand":   onHand + releaseQty,
			"QuantityReserved": newReserved,
			"Version":          version + 1,
			"UpdatedAt":        spanner.CommitTimestamp,
		})
		if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
			return err
		}
		aggID := warehouseID
		if aggID == "" {
			aggID = inventoryID
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		if err := outbox.EmitJSON(ctx, buf, events.AggregateWarehouse, aggID, events.TopicMain, map[string]any{
			"type":              events.EventInventoryQuantityUpdated,
			"inventory_id":      inventoryID,
			"warehouse_id":      warehouseID,
			"supplier_id":       supplierID,
			"product_id":        productID,
			"quantity":          onHand + releaseQty,
			"quantity_reserved": newReserved,
			"timestamp":         time.Now().UTC().Format(time.RFC3339Nano),
		}); err != nil {
			return err
		}
		return buf.Flush(ctx)
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

// CreateHold reserves stock and writes a new InventoryHolds record within a single RW transaction.
func (r *SpannerRepository) CreateHold(ctx context.Context, hold Hold) error {
	linesJSON, err := json.Marshal(hold.Lines)
	if err != nil {
		return fmt.Errorf("marshal hold lines: %w", err)
	}
	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		for _, l := range hold.Lines {
			if l.Quantity <= 0 {
				continue
			}
			if err := r.ReserveTxn(ctx, txn, hold.WarehouseID, l.ProductID, l.Quantity); err != nil {
				return fmt.Errorf("hold reserve line %s: %w", l.ProductID, err)
			}
		}

		m := spanner.InsertMap("InventoryHolds", map[string]any{
			"HoldId":        hold.HoldID,
			"SupplierId":    hold.SupplierID,
			"WarehouseId":   hold.WarehouseID,
			"ReferenceType": hold.ReferenceType,
			"ReferenceId":   hold.ReferenceID,
			"Status":        HoldStatusHeld,
			"HoldLinesJson": linesJSON,
			"ExpiresAt":     hold.ExpiresAt,
			"CreatedAt":     spanner.CommitTimestamp,
			"UpdatedAt":     spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	return err
}

// ConfirmHold marks a hold as CONFIRMED.
func (r *SpannerRepository) ConfirmHold(ctx context.Context, holdID string) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "InventoryHolds", spanner.Key{holdID}, []string{"Status"})
		if err != nil {
			return err
		}
		var status string
		if err := row.Column(0, &status); err != nil {
			return err
		}
		if status == HoldStatusConfirmed {
			return nil // idempotent
		}
		if status != HoldStatusHeld {
			return fmt.Errorf("cannot confirm hold %s: status is %s", holdID, status)
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("InventoryHolds", map[string]any{
				"HoldId":    holdID,
				"Status":    HoldStatusConfirmed,
				"UpdatedAt": spanner.CommitTimestamp,
			}),
		})
	})
	return err
}

// ReleaseHold returns reserved stock and marks the hold as RELEASED.
func (r *SpannerRepository) ReleaseHold(ctx context.Context, holdID string) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "InventoryHolds", spanner.Key{holdID},
			[]string{"WarehouseId", "Status", "HoldLinesJson"})
		if err != nil {
			return err
		}
		var wid, status string
		var linesJSON []byte
		if err := row.Columns(&wid, &status, &linesJSON); err != nil {
			return err
		}
		if status == HoldStatusReleased || status == HoldStatusExpired {
			return nil // idempotent
		}
		if status != HoldStatusHeld {
			return fmt.Errorf("cannot release hold %s: status is %s", holdID, status)
		}

		var lines []HoldLine
		if len(linesJSON) > 0 {
			if err := json.Unmarshal(linesJSON, &lines); err != nil {
				return err
			}
		}
		for _, l := range lines {
			if l.Quantity <= 0 {
				continue
			}
			if err := r.ReleaseTxn(ctx, txn, wid, l.ProductID, l.Quantity); err != nil {
				return fmt.Errorf("release line %s: %w", l.ProductID, err)
			}
		}

		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("InventoryHolds", map[string]any{
				"HoldId":    holdID,
				"Status":    HoldStatusReleased,
				"UpdatedAt": spanner.CommitTimestamp,
			}),
		})
	})
	return err
}

// ListExpiredHolds lists holds currently in HELD state with ExpiresAt in the past.
func (r *SpannerRepository) ListExpiredHolds(ctx context.Context, now time.Time, limit int) ([]Hold, error) {
	if limit <= 0 {
		limit = 100
	}
	stmt := spanner.Statement{
		SQL: `SELECT HoldId, SupplierId, WarehouseId, ReferenceType, ReferenceId, Status, HoldLinesJson, ExpiresAt, CreatedAt, UpdatedAt
		      FROM InventoryHolds@{FORCE_INDEX=Idx_InventoryHolds_ByExpires}
		      WHERE Status = @st AND ExpiresAt < @now
		      LIMIT @limit`,
		Params: map[string]any{
			"st":    HoldStatusHeld,
			"now":   now,
			"limit": int64(limit),
		},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var holds []Hold
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var h Hold
		var linesJSON []byte
		if err := row.Columns(&h.HoldID, &h.SupplierID, &h.WarehouseID, &h.ReferenceType, &h.ReferenceID, &h.Status, &linesJSON, &h.ExpiresAt, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, err
		}
		if len(linesJSON) > 0 {
			_ = json.Unmarshal(linesJSON, &h.Lines)
		}
		holds = append(holds, h)
	}
	return holds, nil
}

