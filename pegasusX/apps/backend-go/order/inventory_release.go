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
	"google.golang.org/api/iterator"
)

// ReleaseReservationsInTxn decrements QuantityReserved for each line item within an existing RW transaction.
// Deprecated: zero production callers remain; callers releasing order stock must call ReleaseReservationsForOrderInTxn with a non-empty orderID so OrderLotReservations and inventory.lot.released outbox events stay consistent.
func ReleaseReservationsInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, warehouseID string, source OrderSource, lineItems []LineItem) error {
	return ReleaseReservationsForOrderInTxn(ctx, txn, supplierID, warehouseID, "", source, lineItems)
}

// ReleaseReservationsForOrderInTxn releases bag and lot reservations for an order.
// Backorders and unreleased/historical MIGRATION orders never held stock reservations,
// so release is a no-op for those sources (MIGRATION_RELEASED does reserve on release).
func ReleaseReservationsForOrderInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, warehouseID, orderID string, source OrderSource, lineItems []LineItem) error {
	if source == OrderSourceBackorder || source == OrderSourceMigration || strings.TrimSpace(warehouseID) == "" {
		return nil
	}
	supplierID = strings.TrimSpace(supplierID)
	warehouseID = strings.TrimSpace(warehouseID)
	orderID = strings.TrimSpace(orderID)

	skuQuantities := aggregateSKUQuantities(lineItems)
	var muts []*spanner.Mutation
	if len(skuQuantities) > 0 {
		bagMuts, err := buildBagReleaseMutationsInTxn(ctx, txn, supplierID, warehouseID, skuQuantities)
		if err != nil {
			return err
		}
		muts = append(muts, bagMuts...)
	}

	var lotQty map[string]int64
	if orderID != "" && txn != nil {
		var lotMuts []*spanner.Mutation
		var err error
		lotQty, lotMuts, err = buildOrderLotReleaseMutationsInTxn(ctx, txn, orderID)
		if err != nil {
			return err
		}
		muts = append(muts, lotMuts...)
	}

	if len(muts) > 0 {
		if err := txn.BufferWrite(muts); err != nil {
			return err
		}
	}

	if len(lotQty) > 0 && txn != nil {
		ev := buildInventoryLotReleasedEvent(orderID, supplierID, warehouseID, lotQty, time.Now().UTC())
		if len(ev.ReleasedLots) > 0 {
			buf := outbox.NewSpannerTxnBuffer(txn)
			if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, orderID, events.TopicMain, ev); err != nil {
				return fmt.Errorf("emit inventory lot released event: %w", err)
			}
			if err := buf.Flush(ctx); err != nil {
				return fmt.Errorf("flush inventory lot released outbox: %w", err)
			}
		}
	}
	return nil
}

func aggregateSKUQuantities(lineItems []LineItem) map[string]int64 {
	skuQuantities := make(map[string]int64, len(lineItems))
	for _, item := range lineItems {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" || item.Quantity <= 0 {
			continue
		}
		skuQuantities[sku] += item.Quantity
	}
	return skuQuantities
}

func buildBagReleaseMutationsInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	supplierID, warehouseID string,
	skuQuantities map[string]int64,
) ([]*spanner.Mutation, error) {
	if txn == nil || len(skuQuantities) == 0 {
		return nil, nil
	}
	skus := sortedMapKeys(skuQuantities)
	keySets := make([]spanner.KeySet, 0, len(skus))
	for _, sku := range skus {
		keySets = append(keySets, spanner.Key{supplierID, warehouseID, sku})
	}

	reservedBySKU := make(map[string]int64, len(skus))
	invIter := txn.Read(ctx, "SupplierInventoryV2", spanner.KeySets(keySets...), []string{"ProductId", "QuantityReserved"})
	defer invIter.Stop()
	for {
		row, err := invIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read reserved qty batch: %w", err)
		}
		var sku string
		var qr int64
		if err := row.Columns(&sku, &qr); err != nil {
			return nil, err
		}
		reservedBySKU[sku] = qr
	}

	muts := make([]*spanner.Mutation, 0, len(skus))
	for _, sku := range skus {
		qr, exists := reservedBySKU[sku]
		if !exists {
			continue
		}
		next := qr - skuQuantities[sku]
		if next < 0 {
			next = 0
		}
		muts = append(muts, spanner.UpdateMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       supplierID,
			"WarehouseId":      warehouseID,
			"ProductId":        sku,
			"QuantityReserved": next,
			"UpdatedAt":        spanner.CommitTimestamp,
		}))
	}
	return muts, nil
}

func loadOrderLotReservationsMapInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string) (map[string]int64, error) {
	lotQty := make(map[string]int64)
	if txn == nil || strings.TrimSpace(orderID) == "" {
		return lotQty, nil
	}
	iter := txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT LotId, Quantity FROM OrderLotReservations WHERE OrderId = @oid`,
		Params: map[string]any{"oid": strings.TrimSpace(orderID)},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read order lot reservations: %w", err)
		}
		var lotID string
		var qty int64
		if err := row.Columns(&lotID, &qty); err != nil {
			return nil, err
		}
		lotQty[lotID] += qty
	}
	return lotQty, nil
}

func readStockLotsReservedInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, lotIDs []string) (map[string]int64, error) {
	out := make(map[string]int64, len(lotIDs))
	if txn == nil || len(lotIDs) == 0 {
		return out, nil
	}
	keySets := make([]spanner.KeySet, 0, len(lotIDs))
	for _, lotID := range lotIDs {
		keySets = append(keySets, spanner.Key{lotID})
	}
	iter := txn.Read(ctx, "StockLots", spanner.KeySets(keySets...), []string{"LotId", "QuantityReserved"})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read stock lots reserved batch: %w", err)
		}
		var lotID string
		var qr int64
		if err := row.Columns(&lotID, &qr); err != nil {
			return nil, err
		}
		out[lotID] = qr
	}
	return out, nil
}

func buildOrderLotReleaseMutationsInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string) (map[string]int64, []*spanner.Mutation, error) {
	if txn == nil || strings.TrimSpace(orderID) == "" {
		return nil, nil, nil
	}
	lotQty, err := loadOrderLotReservationsMapInTxn(ctx, txn, orderID)
	if err != nil {
		return nil, nil, err
	}
	if len(lotQty) == 0 {
		return nil, nil, nil
	}
	lotIDs := sortedMapKeys(lotQty)
	stockReserved, err := readStockLotsReservedInTxn(ctx, txn, lotIDs)
	if err != nil {
		return nil, nil, err
	}
	muts := make([]*spanner.Mutation, 0, len(lotIDs)*2)
	for _, lotID := range lotIDs {
		if qr, exists := stockReserved[lotID]; exists {
			next := qr - lotQty[lotID]
			if next < 0 {
				next = 0
			}
			muts = append(muts, spanner.UpdateMap("StockLots", map[string]any{
				"LotId":            lotID,
				"QuantityReserved": next,
				"UpdatedAt":        spanner.CommitTimestamp,
			}))
		}
		muts = append(muts, spanner.Delete("OrderLotReservations", spanner.Key{orderID, lotID}))
	}
	return lotQty, muts, nil
}

func buildInventoryLotReleasedEvent(orderID, supplierID, warehouseID string, lotQty map[string]int64, now time.Time) events.InventoryLotReleasedEvent {
	lotIDs := sortedMapKeys(lotQty)
	releasedLots := make([]events.ReleasedLotQuantity, 0, len(lotIDs))
	for _, lotID := range lotIDs {
		qty := lotQty[lotID]
		if qty <= 0 {
			continue
		}
		releasedLots = append(releasedLots, events.ReleasedLotQuantity{
			LotID:    lotID,
			Quantity: qty,
		})
	}
	return events.InventoryLotReleasedEvent{
		BaseEvent: events.BaseEvent{
			Type:      events.EventInventoryLotReleased,
			Version:   1,
			Timestamp: now.UTC().Format(time.RFC3339Nano),
		},
		OrderID:      orderID,
		SupplierID:   supplierID,
		WarehouseID:  warehouseID,
		ReleasedLots: releasedLots,
	}
}

func sortedMapKeys(m map[string]int64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ReleaseReservationsFromOrderFields unmarshals LineItemsJson and releases reserved stock in-txn.
// Used by shop-closed cancel paths that write CANCELLED via raw UpdateMap (not UpdateOrder).
// Deprecated: zero production callers remain; callers must use ReleaseReservationsFromOrderFieldsWithID with a non-empty orderID so OrderLotReservations and inventory.lot.released outbox events stay consistent.
func ReleaseReservationsFromOrderFields(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	supplierID, warehouseID, orderSource string,
	lineItemsRaw []byte,
) error {
	return ReleaseReservationsFromOrderFieldsWithID(ctx, txn, supplierID, warehouseID, "", orderSource, lineItemsRaw)
}

// ReleaseReservationsFromOrderFieldsWithID releases with an explicit order ID for lot paths.
func ReleaseReservationsFromOrderFieldsWithID(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	supplierID, warehouseID, orderID, orderSource string,
	lineItemsRaw []byte,
) error {
	var lineItems []LineItem
	if len(lineItemsRaw) > 0 {
		if err := json.Unmarshal(lineItemsRaw, &lineItems); err != nil {
			return fmt.Errorf("parse line items for release: %w", err)
		}
	}
	return ReleaseReservationsForOrderInTxn(ctx, txn, supplierID, warehouseID, orderID, OrderSource(orderSource), lineItems)
}

// releaseOrderReservationsInTxn releases stock for a loaded Order aggregate.
func releaseOrderReservationsInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, o *Order) error {
	if o == nil {
		return nil
	}
	return ReleaseReservationsForOrderInTxn(ctx, txn, o.SupplierID, o.WarehouseID, o.OrderID, o.Source, o.LineItems)
}
