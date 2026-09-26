package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
	"google.golang.org/api/iterator"
)

// ReserveLineItemsInTxn increments QuantityReserved for each line when stock is available.
// When WMS_LOTS_ENABLED, allocates FEFO/FIFO lots via stocklots (orderID required).
func ReserveLineItemsInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, warehouseID string, lineItems []LineItem) error {
	return ReserveLineItemsForOrderInTxn(ctx, txn, supplierID, warehouseID, "", "", time.Time{}, lineItems)
}

// ReserveLineItemsForOrderInTxn reserves bag SKU or lot-level stock for an order.
func ReserveLineItemsForOrderInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	supplierID, warehouseID, orderID, retailerID string,
	expectedDelivery time.Time,
	lineItems []LineItem,
) error {
	if strings.TrimSpace(warehouseID) == "" || len(lineItems) == 0 {
		return nil
	}
	supplierID = strings.TrimSpace(supplierID)

	if stocklots.LotsEnabled() {
		lines := make([]stocklots.LineQty, 0, len(lineItems))
		for _, item := range lineItems {
			sku := strings.TrimSpace(item.SKU)
			if sku == "" || item.Quantity <= 0 {
				continue
			}
			lines = append(lines, stocklots.LineQty{SKU: sku, Quantity: item.Quantity})
		}
		err := stocklots.ReserveFEFOInTxn(ctx, txn, supplierID, warehouseID, orderID, retailerID, expectedDelivery, lines)
		if err != nil {
			if errors.Is(err, stocklots.ErrInventoryExhausted) {
				return fmt.Errorf("%w: %v", ErrInventoryExhausted, err)
			}
			return err
		}
		return nil
	}

	// Aggregate quantities by SKU to avoid double-read/overwrite issues for duplicate SKUs
	skuQuantities := make(map[string]int64)
	for _, item := range lineItems {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" || item.Quantity <= 0 {
			continue
		}
		skuQuantities[sku] += item.Quantity
	}

	for sku, quantity := range skuQuantities {
		row, err := txn.ReadRow(ctx, "SupplierInventoryV2",
			spanner.Key{supplierID, warehouseID, sku},
			[]string{"QuantityOnHand", "QuantityReserved"})
		if err != nil {
			if spanner.ErrCode(err) == 5 {
				return fmt.Errorf("%w: sku %s not found in warehouse %s", ErrInventoryExhausted, sku, warehouseID)
			}
			return fmt.Errorf("read inventory %s: %w", sku, err)
		}
		var qoh, qr int64
		if err := row.Columns(&qoh, &qr); err != nil {
			return fmt.Errorf("decode inventory columns: %w", err)
		}
		if qoh-qr < quantity {
			return fmt.Errorf("%w: sku %s has %d available, requested %d", ErrInventoryExhausted, sku, qoh-qr, quantity)
		}
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       supplierID,
			"WarehouseId":      warehouseID,
			"ProductId":        sku,
			"QuantityReserved": qr + quantity,
			"UpdatedAt":        spanner.CommitTimestamp,
		})}); err != nil {
			return fmt.Errorf("buffer inventory update: %w", err)
		}
	}
	return nil
}

func insertStockReservationMarkerInTxn(txn *spanner.ReadWriteTransaction, orderID string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil
	}
	return txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("OrderStockReservationMarkers", map[string]any{
		"OrderId":    orderID,
		"ReservedAt": spanner.CommitTimestamp,
	})})
}

// BackfillScheduledReservations reserves inventory for legacy scheduled pre-orders created before at-create reservation.
func (s *Service) BackfillScheduledReservations(ctx context.Context, limit int) (int, error) {
	if s == nil || s.spannerClient == nil {
		return 0, nil
	}
	if limit <= 0 {
		limit = 200
	}
	stmt := spanner.Statement{
		SQL: `SELECT o.OrderId, o.SupplierId, o.WarehouseId, o.LineItemsJson, o.OrderSource, o.Status
		      FROM Orders o
		      LEFT JOIN OrderStockReservationMarkers m ON m.OrderId = o.OrderId
		      WHERE m.OrderId IS NULL
		        AND o.OrderSource = @src
		        AND o.Status IN ('SCHEDULED', 'AUTO_ACCEPTED')
		        AND o.WarehouseId IS NOT NULL
		      LIMIT @lim`,
		Params: map[string]any{
			"src": string(OrderSourceManualPreorder),
			"lim": int64(limit),
		},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	type candidate struct {
		orderID     string
		supplierID  string
		warehouseID string
		lineItems   []LineItem
	}
	var pending []candidate
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("backfill list orders: %w", err)
		}
		var orderID, supplierID, warehouseID, source, status string
		var lineItemsRaw []byte
		if err := row.Columns(&orderID, &supplierID, &warehouseID, &lineItemsRaw, &source, &status); err != nil {
			return 0, fmt.Errorf("backfill scan order: %w", err)
		}
		if source == string(OrderSourceBackorder) {
			continue
		}
		var items []LineItem
		if len(lineItemsRaw) > 0 {
			_ = json.Unmarshal(lineItemsRaw, &items)
		}
		if len(items) == 0 {
			continue
		}
		pending = append(pending, candidate{
			orderID:     orderID,
			supplierID:  supplierID,
			warehouseID: warehouseID,
			lineItems:   items,
		})
	}

	backfilled := 0
	for _, c := range pending {
		err := spannerutils.RunReadWriteTransaction(ctx, s.spannerClient, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			if err := ReserveLineItemsInTxn(ctx, txn, c.supplierID, c.warehouseID, c.lineItems); err != nil {
				return err
			}
			if err := insertStockReservationMarkerInTxn(txn, c.orderID); err != nil {
				return err
			}
			buf := outbox.NewSpannerTxnBuffer(txn)
			if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, c.orderID, events.TopicOrders, map[string]any{
				"type":         "ORDER_INVENTORY_RESERVED",
				"order_id":     c.orderID,
				"supplier_id":  c.supplierID,
				"warehouse_id": c.warehouseID,
			}); err != nil {
				return err
			}
			return buf.Flush(ctx)
		})
		if err != nil {
			if s.log != nil {
				s.log.Warn("scheduled reservation backfill skipped order", "order_id", c.orderID, "err", err)
			}
			continue
		}
		backfilled++
	}
	if backfilled > 0 && s.cache != nil {
		s.cache.Invalidate(ctx, "catalog:products:"+s.resolveSupplierScope(ctx))
	}
	return backfilled, nil
}
