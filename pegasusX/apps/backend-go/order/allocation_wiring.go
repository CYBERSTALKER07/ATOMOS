package order

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/allocation"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/segment"
	"google.golang.org/api/iterator"
)

// ErrOrderHoldRequired indicates a MustHave line could not be filled or substituted and
// the order must be held rather than shipped incomplete.
var ErrOrderHoldRequired = errors.New("order_hold_required")

// StockReservationOpts controls whether inventory is reserved during order persistence.
type StockReservationOpts struct {
	Skip bool
}

func deferStockReservationAtCreate(allocationRequired bool, status Status, source OrderSource, _ string, lineCount int) bool {
	if !allocationRequired || lineCount == 0 {
		return false
	}
	if source == OrderSourceBackorder {
		return false
	}
	switch status {
	case StatusScheduled, StatusAutoAccepted, StatusDelayed, StatusPending:
		return true
	default:
		return false
	}
}

func lineIDForIndex(sku string, index int) string {
	return fmt.Sprintf("%s:%d", strings.TrimSpace(sku), index)
}

// ValidateFulfillmentForManifestTxn ensures order warehouse and line allocations match the manifest warehouse.
// When no allocation rows exist, only Orders.WarehouseId is checked (legacy path).
func ValidateFulfillmentForManifestTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID, manifestWarehouseID string) error {
	orderID = strings.TrimSpace(orderID)
	manifestWarehouseID = strings.TrimSpace(manifestWarehouseID)
	if orderID == "" || manifestWarehouseID == "" {
		return fmt.Errorf("order_id and manifest warehouse required")
	}

	row, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{"WarehouseId", "Status", "ConfirmationStatus"})
	if err != nil {
		return fmt.Errorf("read order %s: %w", orderID, err)
	}
	var orderWarehouse spanner.NullString
	var orderStatus, confirmationStatus string
	if err := row.Columns(&orderWarehouse, &orderStatus, &confirmationStatus); err != nil {
		return fmt.Errorf("decode order manifest state %s: %w", orderID, err)
	}
	if Status(orderStatus) != StatusPending {
		return fmt.Errorf("order %s status %s cannot be manifested (must be %s)", orderID, orderStatus, StatusPending)
	}
	conf := ConfirmationStatus(confirmationStatus)
	if conf != ConfirmationStatusConfirmed && conf != ConfirmationStatusAutoConfirmed {
		return fmt.Errorf("order %s confirmation_status %s is not confirmed for dispatch", orderID, confirmationStatus)
	}
	if orderWarehouse.Valid && strings.TrimSpace(orderWarehouse.StringVal) != "" &&
		strings.TrimSpace(orderWarehouse.StringVal) != manifestWarehouseID {
		return fmt.Errorf("order %s warehouse %s does not match manifest warehouse %s",
			orderID, orderWarehouse.StringVal, manifestWarehouseID)
	}

	stmt := spanner.Statement{
		SQL:    `SELECT WarehouseId FROM OrderLineAllocations WHERE OrderId = @orderId`,
		Params: map[string]any{"orderId": orderID},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	allocationCount := 0
	for {
		r, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("read allocations for order %s: %w", orderID, err)
		}
		var wid string
		if err := r.Column(0, &wid); err != nil {
			return fmt.Errorf("decode allocation warehouse %s: %w", orderID, err)
		}
		allocationCount++
		if strings.TrimSpace(wid) != manifestWarehouseID {
			return fmt.Errorf("order %s allocation warehouse %s does not match manifest warehouse %s",
				orderID, wid, manifestWarehouseID)
		}
	}

	if allocationCount > 0 {
		_, err := txn.ReadRow(ctx, "OrderStockReservationMarkers", spanner.Key{orderID}, []string{"OrderId"})
		if err != nil {
			return fmt.Errorf("order %s has allocations but no stock reservation marker", orderID)
		}
	}

	return nil
}

type preparedAllocation struct {
	result          *allocation.AllocationResult
	decisionBySku   map[string]allocation.LineDecision
	targetWarehouse string
	reservedLines   []LineItem
	allocRows       []map[string]interface{}
	decisionRows    []map[string]interface{}
}

func (s *Service) prepareAllocationInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, order *Order) (*preparedAllocation, error) {
	if s == nil || order == nil {
		return nil, fmt.Errorf("allocate: invalid service or order")
	}
	if !s.allocationRequired {
		return nil, nil
	}
	if s.allocator == nil {
		return nil, fmt.Errorf("allocation required but allocator not configured")
	}

	req := &allocation.AllocationRequest{
		SupplierId: order.SupplierID,
		RetailerId: order.RetailerID,
		OrderId:    order.OrderID,
	}
	for _, line := range order.LineItems {
		sku := strings.TrimSpace(line.SKU)
		if sku == "" || line.Quantity <= 0 {
			continue
		}
		req.Items = append(req.Items, allocation.AllocationItem{
			ProductId:        sku,
			QuantityRequired: line.Quantity,
		})
	}
	if len(req.Items) == 0 {
		return nil, nil
	}

	result, err := s.allocator.AllocateOrderTxn(ctx, txn, req)
	if err != nil {
		return nil, fmt.Errorf("allocation failed (out of stock): %w", err)
	}

	decisionBySku := make(map[string]allocation.LineDecision, len(result.Decisions))
	for _, d := range result.Decisions {
		decisionBySku[d.Sku] = d
	}

	warehouseLines := make(map[string][]LineItem)
	allowPartial := allocation.PartialAllocationEnabled()
	nowTS := s.now()
	var allocRows []map[string]interface{}
	var decisionRows []map[string]interface{}
	for idx, line := range order.LineItems {
		sku := strings.TrimSpace(line.SKU)
		if sku == "" || line.Quantity <= 0 {
			continue
		}
		wid, ok := result.Fulfillments[sku]
		if !ok {
			if line.MustHave {
				return nil, fmt.Errorf("%w: must-have sku %s not fulfilled", ErrOrderHoldRequired, sku)
			}
			if allowPartial {
				continue
			}
			return nil, fmt.Errorf("partial allocation: sku %s not fulfilled", sku)
		}
		allocQty := line.Quantity
		if dec, ok := decisionBySku[sku]; ok && dec.AllocatedQty > 0 && dec.AllocatedQty < line.Quantity {
			if line.MustHave {
				return nil, fmt.Errorf("%w: must-have sku %s short (%d < %d)", ErrOrderHoldRequired, sku, dec.AllocatedQty, line.Quantity)
			}
			if !allowPartial {
				return nil, fmt.Errorf("partial allocation: sku %s shortfall", sku)
			}
			allocQty = dec.AllocatedQty
		}
		adjusted := line
		adjusted.Quantity = allocQty
		warehouseLines[wid] = append(warehouseLines[wid], adjusted)

		lineID := lineIDForIndex(sku, idx)
		row := map[string]interface{}{
			"OrderId":     order.OrderID,
			"OrderLineId": lineID,
			"WarehouseId": wid,
			"Sku":         sku,
			"Qty":         allocQty,
			"CreatedAt":   nowTS,
		}
		if dec, ok := decisionBySku[sku]; ok {
			row["AllocationMode"] = dec.AllocationMode
			row["PriorityScore"] = dec.PriorityScore
			row["FairShareBps"] = dec.FairShareBps
			if dec.PolicyId != "" {
				row["PolicyId"] = dec.PolicyId
			}
		} else if result.Mode != "" {
			row["AllocationMode"] = result.Mode
		}
		allocRows = append(allocRows, row)

		if dec, ok := decisionBySku[sku]; ok {
			decisionRow := map[string]interface{}{
				"DecisionId":      uuid.NewString(),
				"OrderId":         order.OrderID,
				"OrderLineId":     lineID,
				"SupplierId":      order.SupplierID,
				"RetailerId":      order.RetailerID,
				"Sku":             sku,
				"WarehouseId":     wid,
				"Qty":             allocQty,
				"RequestedQty":    line.Quantity,
				"AllocatedQty":    allocQty,
				"AllocationMode":  dec.AllocationMode,
				"PriorityScore":   dec.PriorityScore,
				"FairShareBps":    dec.FairShareBps,
				"RetailerSegment": dec.RetailerSegment,
				"SkuClass":        dec.SkuClass,
				"RiskTier":        dec.RiskTier,
				"CreatedAt":       nowTS,
			}
			if dec.ConstraintReason != "" {
				decisionRow["ConstraintReason"] = dec.ConstraintReason
			}
			if dec.PolicyId != "" {
				decisionRow["PolicyId"] = dec.PolicyId
			}
			decisionRows = append(decisionRows, decisionRow)
		}
	}

	if len(warehouseLines) > 1 {
		return nil, fmt.Errorf("multi-warehouse splits not supported: order lines must align to a single manifest")
	}

	var targetWarehouse string
	for wid := range warehouseLines {
		targetWarehouse = wid
		break
	}
	if targetWarehouse == "" {
		return nil, fmt.Errorf("partial allocation: no fulfillable line items")
	}

	return &preparedAllocation{
		result:          result,
		decisionBySku:   decisionBySku,
		targetWarehouse: targetWarehouse,
		reservedLines:   append([]LineItem(nil), warehouseLines[targetWarehouse]...),
		allocRows:       allocRows,
		decisionRows:    decisionRows,
	}, nil
}

func (s *Service) commitAllocationInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, order *Order, prep *preparedAllocation, updateOrderWarehouse bool) error {
	if prep == nil {
		return nil
	}
	for _, row := range prep.allocRows {
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("OrderLineAllocations", row)}); err != nil {
			return fmt.Errorf("buffer order line allocation: %w", err)
		}
	}
	for _, decisionRow := range prep.decisionRows {
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("AllocationDecisions", decisionRow)}); err != nil {
			return fmt.Errorf("buffer allocation decision: %w", err)
		}
	}

	targetWarehouse := prep.targetWarehouse
	if targetWarehouse != order.WarehouseID {
		order.WarehouseID = targetWarehouse
		if updateOrderWarehouse {
			if err := txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("Orders",
					[]string{"OrderId", "WarehouseId", "UpdatedAt"},
					[]any{order.OrderID, targetWarehouse, spanner.CommitTimestamp},
				),
			}); err != nil {
				return fmt.Errorf("update order warehouse: %w", err)
			}
		}
	}

	reserveLines := prep.reservedLines
	if len(reserveLines) == 0 {
		reserveLines = order.LineItems
	}
	expected := time.Time{}
	if order.RequestedDeliveryDate != nil {
		expected = order.RequestedDeliveryDate.UTC()
	} else if order.DeliverBefore != nil {
		expected = order.DeliverBefore.UTC()
	}
	if err := ReserveLineItemsForOrderInTxn(ctx, txn, order.SupplierID, targetWarehouse, order.OrderID, order.RetailerID, expected, reserveLines); err != nil {
		return fmt.Errorf("reserve stock race: %w", err)
	}
	if err := insertStockReservationMarkerInTxn(txn, order.OrderID); err != nil {
		return err
	}

	now := s.now().UTC()
	buf := &spannerTxnBuffer{}
	if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, order.OrderID, events.TopicMain, events.OrderEvent{
		BaseEvent:   events.BaseEvent{Type: events.EventOrderAllocated, Timestamp: now.Format(time.RFC3339Nano)},
		OrderID:     order.OrderID,
		SupplierID:  order.SupplierID,
		RetailerID:  order.RetailerID,
		WarehouseID: targetWarehouse,
		Status:      string(order.Status),
		LineItems:   reserveLines,
	}); err != nil {
		return err
	}
	if prep.result.Mode == segment.AllocationModePolicy {
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, order.OrderID, events.TopicMain, map[string]interface{}{
			"type":            events.EventAllocationPolicyApplied,
			"timestamp":       now.Format(time.RFC3339Nano),
			"order_id":        order.OrderID,
			"supplier_id":     order.SupplierID,
			"retailer_id":     order.RetailerID,
			"warehouse_id":    targetWarehouse,
			"allocation_mode": prep.result.Mode,
		}); err != nil {
			return err
		}
	}
	var outboxMuts []*spanner.Mutation
	for _, e := range buf.events {
		outboxMuts = append(outboxMuts, spanner.InsertOrUpdateMap("OutboxEvents", outbox.EventRowMap(e)))
	}
	return txn.BufferWrite(outboxMuts)
}

func (s *Service) allocateAndReserveInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, order *Order) error {
	prep, err := s.prepareAllocationInTxn(ctx, txn, order)
	if err != nil {
		return err
	}
	return s.commitAllocationInTxn(ctx, txn, order, prep, true)
}

func (s *Service) loadOrderShortageContext(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	supplierID, warehouseID string,
	items []LineItem,
) ([]LineItem, ShortageContext, error) {
	working := append([]LineItem(nil), items...)
	if s != nil && s.shortageStockReader != nil {
		sc, err := s.shortageStockReader(ctx, txn, supplierID, warehouseID, working)
		return working, sc, err
	}
	if txn == nil {
		return working, ShortageContext{}, nil
	}
	return loadOrderShortageContextInTxn(ctx, txn, supplierID, warehouseID, working)
}

func loadOrderShortageContextInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	supplierID, warehouseID string,
	items []LineItem,
) ([]LineItem, ShortageContext, error) {
	sc := ShortageContext{
		AvailableBySKU:  make(map[string]int64, len(items)),
		CandidatesBySKU: make(map[string][]SubstituteCandidate),
	}
	if txn == nil || len(items) == 0 {
		return items, sc, nil
	}

	skuSet := make(map[string]struct{}, len(items))
	skus := make([]string, 0, len(items))
	missingBrandSKUs := make([]string, 0)
	for _, it := range items {
		sku := strings.TrimSpace(it.SKU)
		if sku == "" {
			continue
		}
		if _, seen := skuSet[sku]; !seen {
			skuSet[sku] = struct{}{}
			skus = append(skus, sku)
		}
		if strings.TrimSpace(it.BrandID) == "" {
			missingBrandSKUs = append(missingBrandSKUs, sku)
		}
	}
	if len(skus) == 0 {
		return items, sc, nil
	}

	// 1. Live available stock in SupplierInventoryV2.
	stockStmt := spanner.Statement{
		SQL: `SELECT ProductId, COALESCE(MAX(CASE WHEN QuantityOnHand - QuantityReserved > 0 THEN QuantityOnHand - QuantityReserved ELSE 0 END), 0)
			  FROM SupplierInventoryV2
			  WHERE SupplierId = @sid AND (@wid = '' OR WarehouseId = @wid) AND ProductId IN UNNEST(@skus)
			  GROUP BY ProductId`,
		Params: map[string]any{
			"sid":  strings.TrimSpace(supplierID),
			"wid":  strings.TrimSpace(warehouseID),
			"skus": skus,
		},
	}
	stockIter := txn.Query(ctx, stockStmt)
	defer stockIter.Stop()
	for {
		row, err := stockIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, sc, fmt.Errorf("read shortage inventory: %w", err)
		}
		var productID string
		var avail int64
		if err := row.Columns(&productID, &avail); err != nil {
			return nil, sc, fmt.Errorf("decode shortage inventory: %w", err)
		}
		sc.AvailableBySKU[strings.TrimSpace(productID)] = avail
	}

	// 2. Resolve missing BrandID via SupplierProductOffers + GlobalProducts.
	if len(missingBrandSKUs) > 0 && strings.TrimSpace(supplierID) != "" {
		brandStmt := spanner.Statement{
			SQL: `SELECT spo.ProductId, gp.BrandId
				  FROM SupplierProductOffers spo
				  JOIN GlobalProducts gp ON gp.GlobalProductId = spo.GlobalProductId
				  WHERE spo.SupplierId = @sid AND spo.ProductId IN UNNEST(@skus) AND spo.Status = 'ACTIVE'`,
			Params: map[string]any{
				"sid":  strings.TrimSpace(supplierID),
				"skus": missingBrandSKUs,
			},
		}
		brandIter := txn.Query(ctx, brandStmt)
		defer brandIter.Stop()
		resolvedBrands := make(map[string]string, len(missingBrandSKUs))
		for {
			row, err := brandIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, sc, fmt.Errorf("read sku brand ids: %w", err)
			}
			var sku, brandID string
			if err := row.Columns(&sku, &brandID); err != nil {
				return nil, sc, fmt.Errorf("decode sku brand id: %w", err)
			}
			if strings.TrimSpace(brandID) != "" {
				resolvedBrands[strings.TrimSpace(sku)] = strings.TrimSpace(brandID)
			}
		}
		for i := range items {
			if strings.TrimSpace(items[i].BrandID) == "" {
				if bid := resolvedBrands[strings.TrimSpace(items[i].SKU)]; bid != "" {
					items[i].BrandID = bid
				}
			}
		}
	}

	// 3. Load same-brand, same-supplier active offers for lines allowing substitutes.
	brandSet := make(map[string]struct{})
	brandIDs := make([]string, 0)
	for _, it := range items {
		if !it.AllowSubstitutes {
			continue
		}
		bid := strings.TrimSpace(it.BrandID)
		if bid == "" {
			continue
		}
		if _, seen := brandSet[bid]; !seen {
			brandSet[bid] = struct{}{}
			brandIDs = append(brandIDs, bid)
		}
	}
	if len(brandIDs) > 0 && strings.TrimSpace(supplierID) != "" {
		candStmt := spanner.Statement{
			SQL: `SELECT spo.ProductId, spo.SupplierId, gp.BrandId, p.Name, p.PriceMinor, spo.PriceMinor,
			             COALESCE(MAX(CASE WHEN inv.QuantityOnHand - inv.QuantityReserved > 0 THEN inv.QuantityOnHand - inv.QuantityReserved ELSE 0 END), 0)
			      FROM SupplierProductOffers spo
			      JOIN GlobalProducts gp ON gp.GlobalProductId = spo.GlobalProductId
			      JOIN Products p ON p.ProductId = spo.ProductId
			      LEFT JOIN SupplierInventoryV2 inv ON inv.SupplierId = spo.SupplierId AND inv.ProductId = spo.ProductId
			           AND (@wid = '' OR inv.WarehouseId = @wid)
			      WHERE spo.SupplierId = @sid AND gp.BrandId IN UNNEST(@brandIDs) AND spo.Status = 'ACTIVE'
			      GROUP BY spo.ProductId, spo.SupplierId, gp.BrandId, p.Name, p.PriceMinor, spo.PriceMinor`,
			Params: map[string]any{
				"sid":      strings.TrimSpace(supplierID),
				"wid":      strings.TrimSpace(warehouseID),
				"brandIDs": brandIDs,
			},
		}
		candIter := txn.Query(ctx, candStmt)
		defer candIter.Stop()
		byBrand := make(map[string][]SubstituteCandidate, len(brandIDs))
		for {
			row, err := candIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, sc, fmt.Errorf("read brand substitute candidates: %w", err)
			}
			var sku, sup, bid string
			var name spanner.NullString
			var pPrice, spoPrice spanner.NullInt64
			var avail int64
			if err := row.Columns(&sku, &sup, &bid, &name, &pPrice, &spoPrice, &avail); err != nil {
				return nil, sc, fmt.Errorf("decode brand substitute candidate: %w", err)
			}
			price := int64(0)
			if pPrice.Valid {
				price = pPrice.Int64
			} else if spoPrice.Valid {
				price = spoPrice.Int64
			}
			normBrand := strings.ToUpper(strings.TrimSpace(bid))
			byBrand[normBrand] = append(byBrand[normBrand], SubstituteCandidate{
				SKU:          strings.TrimSpace(sku),
				SupplierID:   strings.TrimSpace(sup),
				BrandID:      strings.TrimSpace(bid),
				Name:         strings.TrimSpace(name.StringVal),
				AvailableQty: avail,
				PriceMinor:   price,
			})
		}
		for _, it := range items {
			if !it.AllowSubstitutes {
				continue
			}
			sku := strings.TrimSpace(it.SKU)
			normBrand := strings.ToUpper(strings.TrimSpace(it.BrandID))
			if normBrand != "" && len(byBrand[normBrand]) > 0 {
				sc.CandidatesBySKU[sku] = byBrand[normBrand]
			}
		}
	}

	return items, sc, nil
}

// ReleaseAllocation releases warehouse allocation rows and reservation marker for a cancelled order.
func (s *Service) ReleaseAllocation(ctx context.Context, orderID string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" || s == nil || s.spannerClient == nil {
		return nil
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return prepareReleaseAllocationInTxn(ctx, txn, orderID)
	})
	return err
}

func prepareReleaseAllocationInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string) error {
	if txn == nil || strings.TrimSpace(orderID) == "" {
		return nil
	}
	muts := []*spanner.Mutation{
		spanner.Delete("OrderLineAllocations", spanner.KeyRange{
			Start: spanner.Key{orderID},
			End:   spanner.Key{orderID},
			Kind:  spanner.ClosedClosed,
		}),
		spanner.Delete("OrderStockReservationMarkers", spanner.Key{orderID}),
	}
	return txn.BufferWrite(muts)
}
