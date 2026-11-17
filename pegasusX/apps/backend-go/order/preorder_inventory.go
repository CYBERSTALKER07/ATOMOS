package order

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
	"google.golang.org/api/iterator"
)

// reservationCreditForLines sums reserved qty per SKU for an existing order.
func reservationCreditForLines(lines []LineItem) map[string]int64 {
	credit := make(map[string]int64)
	for _, item := range lines {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" || item.Quantity <= 0 {
			continue
		}
		credit[sku] += item.Quantity
	}
	return credit
}

// applyPreorderInventoryGuard plans checkout for new lines and returns fulfillable lines only.
func (s *Service) applyPreorderInventoryGuard(
	ctx context.Context,
	supplierID, warehouseID string,
	prevLines, requestedLines []LineItem,
) ([]LineItem, InventoryPlan, error) {
	if s == nil || s.spannerClient == nil || len(requestedLines) == 0 {
		return requestedLines, InventoryPlan{Fulfillable: append([]LineItem(nil), requestedLines...)}, nil
	}
	credit := reservationCreditForLines(prevLines)
	plan, err := PlanInventoryCheckoutWithCredit(ctx, s.spannerClient, supplierID, warehouseID, requestedLines, "", credit)
	if err != nil {
		return nil, plan, err
	}
	if len(plan.Fulfillable) == 0 && len(plan.Backorder) == 0 {
		return nil, plan, ErrInventoryExhausted
	}
	return plan.Fulfillable, plan, nil
}

// reconcilePreorderReservationsInTxn reconciles bag and lot reservations in a single Spanner read-write pass.
func reconcilePreorderReservationsInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	supplierID, warehouseID, orderID string,
	source OrderSource,
	prevLines, fulfillableLines []LineItem,
) error {
	if txn == nil || strings.TrimSpace(warehouseID) == "" {
		return nil
	}
	supplierID = strings.TrimSpace(supplierID)
	warehouseID = strings.TrimSpace(warehouseID)
	orderID = strings.TrimSpace(orderID)

	oldQty := make(map[string]int64)
	if source != OrderSourceBackorder {
		oldQty = aggregateSKUQuantities(prevLines)
	}
	newQty := aggregateSKUQuantities(fulfillableLines)

	var muts []*spanner.Mutation
	bagMuts, err := buildBagReconcileMutationsInTxn(ctx, txn, supplierID, warehouseID, oldQty, newQty)
	if err != nil {
		return err
	}
	muts = append(muts, bagMuts...)

	if orderID != "" {
		lotMuts, err := buildLotReconcileMutationsInTxn(ctx, txn, supplierID, warehouseID, orderID, newQty)
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
	return nil
}

func buildBagReconcileMutationsInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	supplierID, warehouseID string,
	oldQty, newQty map[string]int64,
) ([]*spanner.Mutation, error) {
	union := make(map[string]int64, len(oldQty)+len(newQty))
	for sku := range oldQty {
		union[sku] = 1
	}
	for sku := range newQty {
		union[sku] = 1
	}
	if len(union) == 0 {
		return nil, nil
	}
	allSKUs := sortedMapKeys(union)
	keySets := make([]spanner.KeySet, 0, len(allSKUs))
	for _, sku := range allSKUs {
		keySets = append(keySets, spanner.Key{supplierID, warehouseID, sku})
	}

	type invSnap struct {
		qoh int64
		qr  int64
	}
	snaps := make(map[string]invSnap, len(allSKUs))
	iter := txn.Read(ctx, "SupplierInventoryV2", spanner.KeySets(keySets...), []string{"ProductId", "QuantityOnHand", "QuantityReserved"})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read inventory batch: %w", err)
		}
		var sku string
		var qoh, qr int64
		if err := row.Columns(&sku, &qoh, &qr); err != nil {
			return nil, fmt.Errorf("decode inventory columns: %w", err)
		}
		snaps[sku] = invSnap{qoh: qoh, qr: qr}
	}

	var muts []*spanner.Mutation
	for _, sku := range allSKUs {
		delta := newQty[sku] - oldQty[sku]
		if delta == 0 {
			continue
		}
		snap, ok := snaps[sku]
		if delta > 0 {
			if !ok {
				return nil, fmt.Errorf("%w: sku %s not found in warehouse %s", ErrInventoryExhausted, sku, warehouseID)
			}
			if snap.qoh-snap.qr < delta {
				return nil, fmt.Errorf("%w: sku %s has %d available, requested extra %d", ErrInventoryExhausted, sku, snap.qoh-snap.qr, delta)
			}
		} else if !ok {
			continue
		}
		next := snap.qr + delta
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

func buildLotReconcileMutationsInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	supplierID, warehouseID, orderID string,
	newQty map[string]int64,
) ([]*spanner.Mutation, error) {
	oldLots, err := loadOrderLotReservationsMapInTxn(ctx, txn, orderID)
	if err != nil {
		return nil, err
	}
	oldLotIDs := sortedMapKeys(oldLots)
	stockReserved, err := readStockLotsReservedInTxn(ctx, txn, oldLotIDs)
	if err != nil {
		return nil, err
	}

	// Fast exit: no prior lot reservations and lot tracking disabled or empty fulfillable lines.
	if !stocklots.EffectiveLots(ctx, warehouseID, supplierID) || len(newQty) == 0 {
		var muts []*spanner.Mutation
		for _, lotID := range oldLotIDs {
			if qr, exists := stockReserved[lotID]; exists {
				next := qr - oldLots[lotID]
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
		return muts, nil
	}

	newSKUs := sortedMapKeys(newQty)
	sql := `SELECT ProductId, LotId, QuantityOnHand, QuantityReserved
	        FROM StockLots
	        WHERE WarehouseId = @w AND ProductId IN UNNEST(@skus) AND Status = 'AVAILABLE'
	          AND QuantityOnHand - QuantityReserved > 0`
	params := map[string]any{"w": warehouseID, "skus": newSKUs}
	if len(oldLotIDs) > 0 {
		sql = `SELECT ProductId, LotId, QuantityOnHand, QuantityReserved
		       FROM StockLots
		       WHERE WarehouseId = @w AND ProductId IN UNNEST(@skus) AND Status = 'AVAILABLE'
		         AND (QuantityOnHand - QuantityReserved > 0 OR LotId IN UNNEST(@oldLots))`
		params["oldLots"] = oldLotIDs
	}
	sql += ` ORDER BY ProductId ASC, ExpiryDate ASC, ReceivedAt ASC`

	iter := txn.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	remaining := make(map[string]int64, len(newQty))
	for sku, qty := range newQty {
		remaining[sku] = qty
	}

	baseQR := make(map[string]int64, len(oldLotIDs))
	for _, lotID := range oldLotIDs {
		if qr, exists := stockReserved[lotID]; exists {
			b := qr - oldLots[lotID]
			if b < 0 {
				b = 0
			}
			baseQR[lotID] = b
		}
	}

	newLots := make(map[string]int64)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read lots for FEFO reconcile: %w", err)
		}
		var sku, lotID string
		var lqoh, lqr int64
		if err := row.Columns(&sku, &lotID, &lqoh, &lqr); err != nil {
			return nil, fmt.Errorf("decode lot for FEFO reconcile: %w", err)
		}
		effectiveQR := lqr - oldLots[lotID]
		if effectiveQR < 0 {
			effectiveQR = 0
		}
		baseQR[lotID] = effectiveQR
		if _, seen := stockReserved[lotID]; !seen {
			stockReserved[lotID] = lqr
		}
		rem := remaining[sku]
		if rem <= 0 {
			continue
		}
		avail := lqoh - effectiveQR
		if avail <= 0 {
			continue
		}
		take := avail
		if rem < take {
			take = rem
		}
		remaining[sku] = rem - take
		newLots[lotID] += take
	}

	for _, sku := range newSKUs {
		if rem := remaining[sku]; rem > 0 {
			return nil, fmt.Errorf("%w: sku %s short by %d on lots", ErrInventoryExhausted, sku, rem)
		}
	}

	touchedLots := make(map[string]int64, len(oldLots)+len(newLots))
	for lotID := range oldLots {
		touchedLots[lotID] = 1
	}
	for lotID := range newLots {
		touchedLots[lotID] = 1
	}
	sortedTouched := sortedMapKeys(touchedLots)
	muts := make([]*spanner.Mutation, 0, len(sortedTouched)*2)
	for _, lotID := range sortedTouched {
		nextQR := baseQR[lotID] + newLots[lotID]
		if prevQR, exists := stockReserved[lotID]; exists && nextQR != prevQR {
			muts = append(muts, spanner.UpdateMap("StockLots", map[string]any{
				"LotId":            lotID,
				"QuantityReserved": nextQR,
				"UpdatedAt":        spanner.CommitTimestamp,
			}))
		}
		if qty := newLots[lotID]; qty > 0 {
			muts = append(muts, spanner.InsertOrUpdateMap("OrderLotReservations", map[string]any{
				"OrderId":   orderID,
				"LotId":     lotID,
				"Quantity":  qty,
				"CreatedAt": spanner.CommitTimestamp,
			}))
		} else if oldLots[lotID] > 0 {
			muts = append(muts, spanner.Delete("OrderLotReservations", spanner.Key{orderID, lotID}))
		}
	}
	return muts, nil
}

func totalMinorForLines(lines []LineItem) int64 {
	var total int64
	for _, li := range lines {
		total += li.Quantity * li.UnitPrice
	}
	return total
}

// updatePreorderLines applies inventory guard and persists line changes with reservation reconcile.
func (s *Service) updatePreorderLines(
	ctx context.Context,
	current Order,
	requestedLines []LineItem,
	emit func(outbox.TxnBuffer, Order) error,
) (Order, error) {
	prevLines := append([]LineItem(nil), current.LineItems...)
	fulfillable, _, err := s.applyPreorderInventoryGuard(ctx, current.SupplierID, current.WarehouseID, prevLines, requestedLines)
	if err != nil {
		return current, err
	}
	prevTotal := current.TotalMinor
	current.LineItems = fulfillable
	tot := totalMinorForLines(fulfillable)
	if tot > 0 && current.DeliveryFeeMinor > 0 {
		tot += current.DeliveryFeeMinor
	}
	current.TotalMinor = tot
	err = s.repo.UpdateOrderWithTxn(ctx, current, nil, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := reconcilePreorderReservationsInTxn(ctx, txn, current.SupplierID, current.WarehouseID, current.OrderID, current.Source, prevLines, fulfillable); err != nil {
			return err
		}
		if s.credit != nil && txn != nil && current.TotalMinor != prevTotal {
			if err := s.credit.AdjustReserveInTxn(ctx, txn, current.OrderID, current.TotalMinor); err != nil {
				return fmt.Errorf("adjust credit reserve on preorder edit: %w", err)
			}
		}
		return nil
	}, func(txn outbox.TxnBuffer) error {
		if emit != nil {
			return emit(txn, current)
		}
		return nil
	})
	return current, err
}
