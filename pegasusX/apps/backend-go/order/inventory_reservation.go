package order

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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
// When WMS_LOTS_ENABLED, allocates /FIFO lots via stocklots (orderID required).
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
	warehouseID = strings.TrimSpace(warehouseID)
	orderID = strings.TrimSpace(orderID)

	// Aggregate quantities by SKU to avoid double-read/overwrite issues for duplicate SKUs
	skuQuantities := make(map[string]int64)
	for _, item := range lineItems {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" || item.Quantity <= 0 {
			continue
		}
		skuQuantities[sku] += item.Quantity
	}
	if len(skuQuantities) == 0 {
		return nil
	}

	skus := make([]string, 0, len(skuQuantities))
	keySets := make([]spanner.KeySet, 0, len(skuQuantities))
	for sku := range skuQuantities {
		skus = append(skus, sku)
	}
	sort.Strings(skus)
	for _, sku := range skus {
		keySets = append(keySets, spanner.Key{supplierID, warehouseID, sku})
	}

	type invSnapshot struct {
		qoh int64
		qr  int64
	}
	invBySKU := make(map[string]invSnapshot, len(skus))
	invIter := txn.Read(ctx, "SupplierInventoryV2", spanner.KeySets(keySets...), []string{"ProductId", "QuantityOnHand", "QuantityReserved"})
	defer invIter.Stop()
	for {
		row, err := invIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("read inventory batch: %w", err)
		}
		var sku string
		var qoh, qr int64
		if err := row.Columns(&sku, &qoh, &qr); err != nil {
			return fmt.Errorf("decode inventory columns: %w", err)
		}
		invBySKU[sku] = invSnapshot{qoh: qoh, qr: qr}
	}

	invMuts := make([]*spanner.Mutation, 0, len(skus))
	for _, sku := range skus {
		quantity := skuQuantities[sku]
		snap, ok := invBySKU[sku]
		if !ok {
			return fmt.Errorf("%w: sku %s not found in warehouse %s", ErrInventoryExhausted, sku, warehouseID)
		}
		if snap.qoh-snap.qr < quantity {
			return fmt.Errorf("%w: sku %s has %d available, requested %d", ErrInventoryExhausted, sku, snap.qoh-snap.qr, quantity)
		}
		invMuts = append(invMuts, spanner.UpdateMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       supplierID,
			"WarehouseId":      warehouseID,
			"ProductId":        sku,
			"QuantityReserved": snap.qr + quantity,
			"UpdatedAt":        spanner.CommitTimestamp,
		}))
	}
	if err := txn.BufferWrite(invMuts); err != nil {
		return fmt.Errorf("buffer inventory update: %w", err)
	}

	if stocklots.EffectiveLots(ctx, warehouseID, supplierID) && orderID != "" {
		if err := reserveLotsForSKUsInTxn(ctx, txn, warehouseID, orderID, skus, skuQuantities); err != nil {
			return err
		}
	}
	return nil
}

func reserveLotsForSKUsInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	warehouseID, orderID string,
	skus []string,
	skuQuantities map[string]int64,
) error {
	stmt := spanner.Statement{
		SQL: `SELECT ProductId, LotId, QuantityOnHand, QuantityReserved
		      FROM StockLots
		      WHERE WarehouseId = @w AND ProductId IN UNNEST(@skus) AND Status = 'AVAILABLE'
		        AND QuantityOnHand - QuantityReserved > 0
		      ORDER BY ProductId ASC, ExpiryDate ASC, ReceivedAt ASC`,
		Params: map[string]any{"w": warehouseID, "skus": skus},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	remaining := make(map[string]int64, len(skuQuantities))
	for sku, qty := range skuQuantities {
		remaining[sku] = qty
	}

	var muts []*spanner.Mutation
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("read lots for FEFO: %w", err)
		}
		var sku, lotID string
		var lqoh, lqr int64
		if err := row.Columns(&sku, &lotID, &lqoh, &lqr); err != nil {
			return fmt.Errorf("decode lot for FEFO: %w", err)
		}
		rem := remaining[sku]
		if rem <= 0 {
			continue
		}
		avail := lqoh - lqr
		if avail <= 0 {
			continue
		}
		take := avail
		if rem < take {
			take = rem
		}
		remaining[sku] = rem - take
		muts = append(muts,
			spanner.UpdateMap("StockLots", map[string]any{
				"LotId":            lotID,
				"QuantityReserved": lqr + take,
				"UpdatedAt":        spanner.CommitTimestamp,
			}),
			spanner.InsertOrUpdateMap("OrderLotReservations", map[string]any{
				"OrderId":  orderID,
				"LotId":    lotID,
				"Quantity": take,
			}),
		)
	}

	for _, sku := range skus {
		if rem := remaining[sku]; rem > 0 {
			return fmt.Errorf("%w: sku %s short by %d on lots", ErrInventoryExhausted, sku, rem)
		}
	}
	if len(muts) > 0 {
		if err := txn.BufferWrite(muts); err != nil {
			return fmt.Errorf("buffer lot reserve: %w", err)
		}
	}
	return nil
}

func stockReservationMarkerMutation(orderID string) *spanner.Mutation {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil
	}
	return spanner.InsertOrUpdateMap("OrderStockReservationMarkers", map[string]any{
		"OrderId":    orderID,
		"ReservedAt": spanner.CommitTimestamp,
	})
}

func insertStockReservationMarkerInTxn(txn *spanner.ReadWriteTransaction, orderID string) error {
	m := stockReservationMarkerMutation(orderID)
	if m == nil {
		return nil
	}
	return txn.BufferWrite([]*spanner.Mutation{m})
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
