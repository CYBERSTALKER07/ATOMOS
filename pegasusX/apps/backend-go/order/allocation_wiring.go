package order

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/allocation"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/segment"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// StockReservationOpts controls whether inventory is reserved during order persistence.
type StockReservationOpts struct {
	Skip bool
}

func deferStockReservationAtCreate(allocationRequired bool, status Status, source OrderSource, warehouseID string, lineCount int) bool {
	if !allocationRequired || lineCount == 0 || strings.TrimSpace(warehouseID) == "" {
		return false
	}
	if source == OrderSourceBackorder {
		return false
	}
	switch status {
	case StatusScheduled, StatusAutoAccepted, StatusPending:
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

	row, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{"WarehouseId"})
	if err != nil {
		return fmt.Errorf("read order %s: %w", orderID, err)
	}
	var orderWarehouse spanner.NullString
	if err := row.Column(0, &orderWarehouse); err != nil {
		return fmt.Errorf("decode order warehouse %s: %w", orderID, err)
	}
	if orderWarehouse.Valid && strings.TrimSpace(orderWarehouse.StringVal) != "" &&
		strings.TrimSpace(orderWarehouse.StringVal) != manifestWarehouseID {
		return fmt.Errorf("order %s warehouse %s does not match manifest warehouse %s",
			orderID, orderWarehouse.StringVal, manifestWarehouseID)
	}

	stmt := spanner.Statement{
		SQL: `SELECT WarehouseId FROM OrderLineAllocations WHERE OrderId = @orderId`,
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

func (s *Service) allocateAndReserveInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, order *Order) error {
	if s == nil || order == nil {
		return fmt.Errorf("allocate: invalid service or order")
	}
	if !s.allocationRequired {
		return nil
	}
	if s.allocator == nil {
		return fmt.Errorf("allocation required but allocator not configured")
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
		return nil
	}

	result, err := s.allocator.AllocateOrderTxn(ctx, txn, req)
	if err != nil {
		return fmt.Errorf("allocation failed (out of stock): %w", err)
	}

	decisionBySku := make(map[string]allocation.LineDecision, len(result.Decisions))
	for _, d := range result.Decisions {
		decisionBySku[d.Sku] = d
	}

	warehouseLines := make(map[string][]LineItem)
	for idx, line := range order.LineItems {
		sku := strings.TrimSpace(line.SKU)
		if sku == "" || line.Quantity <= 0 {
			continue
		}
		wid, ok := result.Fulfillments[sku]
		if !ok {
			return fmt.Errorf("partial allocation: sku %s not fulfilled", sku)
		}
		warehouseLines[wid] = append(warehouseLines[wid], line)

		lineID := lineIDForIndex(sku, idx)
		row := map[string]interface{}{
			"OrderId":     order.OrderID,
			"OrderLineId": lineID,
			"WarehouseId": wid,
			"Sku":         sku,
			"Qty":         line.Quantity,
			"CreatedAt":   spanner.CommitTimestamp,
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
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("OrderLineAllocations", row)}); err != nil {
			return fmt.Errorf("buffer order line allocation: %w", err)
		}

		if dec, ok := decisionBySku[sku]; ok {
			decisionRow := map[string]interface{}{
				"DecisionId":      uuid.NewString(),
				"OrderId":         order.OrderID,
				"OrderLineId":     lineID,
				"SupplierId":      order.SupplierID,
				"RetailerId":      order.RetailerID,
				"Sku":             sku,
				"WarehouseId":     wid,
				"Qty":             line.Quantity,
				"RequestedQty":    line.Quantity,
				"AllocatedQty":    line.Quantity,
				"AllocationMode":  dec.AllocationMode,
				"PriorityScore":   dec.PriorityScore,
				"FairShareBps":    dec.FairShareBps,
				"RetailerSegment": dec.RetailerSegment,
				"SkuClass":        dec.SkuClass,
				"RiskTier":        dec.RiskTier,
				"CreatedAt":       spanner.CommitTimestamp,
			}
			if dec.ConstraintReason != "" {
				decisionRow["ConstraintReason"] = dec.ConstraintReason
			}
			if dec.PolicyId != "" {
				decisionRow["PolicyId"] = dec.PolicyId
			}
			if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("AllocationDecisions", decisionRow)}); err != nil {
				return fmt.Errorf("buffer allocation decision: %w", err)
			}
		}
	}

	if len(warehouseLines) > 1 {
		return fmt.Errorf("multi-warehouse splits not supported: order lines must align to a single manifest")
	}

	var targetWarehouse string
	for wid := range warehouseLines {
		targetWarehouse = wid
		break
	}
	if targetWarehouse == "" {
		return fmt.Errorf("partial allocation: no fulfillable line items")
	}

	if targetWarehouse != order.WarehouseID {
		order.WarehouseID = targetWarehouse
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("Orders",
				[]string{"OrderId", "WarehouseId", "UpdatedAt"},
				[]any{order.OrderID, targetWarehouse, spanner.CommitTimestamp},
			),
		}); err != nil {
			return fmt.Errorf("update order warehouse: %w", err)
		}
	}

	expected := time.Time{}
	if order.RequestedDeliveryDate != nil {
		expected = order.RequestedDeliveryDate.UTC()
	} else if order.DeliverBefore != nil {
		expected = order.DeliverBefore.UTC()
	}
	if err := ReserveLineItemsForOrderInTxn(ctx, txn, order.SupplierID, targetWarehouse, order.OrderID, order.RetailerID, expected, order.LineItems); err != nil {
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
		LineItems:   order.LineItems,
	}); err != nil {
		return err
	}
	if result.Mode == segment.AllocationModePolicy {
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, order.OrderID, events.TopicMain, map[string]interface{}{
			"type":            events.EventAllocationPolicyApplied,
			"timestamp":       now.Format(time.RFC3339Nano),
			"order_id":        order.OrderID,
			"supplier_id":     order.SupplierID,
			"retailer_id":     order.RetailerID,
			"warehouse_id":    targetWarehouse,
			"allocation_mode": result.Mode,
		}); err != nil {
			return err
		}
	}
	var outboxMuts []*spanner.Mutation
	for _, e := range buf.events {
		outboxMuts = append(outboxMuts, spanner.InsertOrUpdateMap("OutboxEvents", map[string]any{
			"EventId":       e.EventID,
			"AggregateType": e.AggregateType,
			"AggregateId":   e.AggregateID,
			"TopicName":     e.TopicName,
			"Payload":       e.Payload,
			"CreatedAt":     e.CreatedAt,
		}))
	}
	return txn.BufferWrite(outboxMuts)
}
