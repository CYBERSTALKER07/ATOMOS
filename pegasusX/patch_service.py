import re

with open("apps/backend-go/order/service.go", "r") as f:
    content = f.read()

# 1. We need to construct the backorder *before* CreateOrderWithBackorder
replacement_1 = """
	stockOpts := StockReservationOpts{
		Skip: deferStockReservationAtCreate(s.allocationRequired, status, source, warehouseID, len(lineItems)),
	}

	var bo *Order
	if len(invPlan.Backorder) > 0 {
		var boTotal int64
		for _, li := range invPlan.Backorder {
			boTotal += li.Quantity * li.UnitPrice
		}
		boCurrency := strings.TrimSpace(o.Currency)
		if boCurrency == "" {
			boCurrency = s.currency
		}
		bo = &Order{
			OrderID:            s.newID(),
			SupplierID:         o.SupplierID,
			RetailerID:         o.RetailerID,
			WarehouseID:        o.WarehouseID,
			Status:             StatusBackordered,
			Source:             OrderSourceBackorder,
			ConfirmationStatus: ConfirmationStatusConfirmed,
			LineItems:          invPlan.Backorder,
			TotalMinor:         boTotal,
			Currency:           boCurrency,
			H3Cell:             o.H3Cell,
			Lat:                o.Lat,
			Lng:                o.Lng,
			DerivedFromOrderID: o.OrderID,
			Version:            1,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
	}

	err = s.repo.CreateOrderWithBackorder(ctx, &o, bo, func(txn outbox.TxnBuffer) error {
		if err := outbox.EmitJSON(ctx, txn, events.AggregateOrder, o.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:             events.BaseEvent{Type: events.EventOrderCreated, Timestamp: o.CreatedAt.Format(time.RFC3339Nano)},
			OrderID:               o.OrderID,
			SupplierID:            o.SupplierID,
			RetailerID:            o.RetailerID,
			WarehouseID:           o.WarehouseID,
			Status:                string(o.Status),
			OrderSource:           string(o.Source),
			ConfirmationStatus:    string(o.ConfirmationStatus),
			TotalMinor:            o.TotalMinor,
			Currency:              o.Currency,
			H3Cell:                o.H3Cell,
			Lat:                   o.Lat,
			Lng:                   o.Lng,
			RequestedDeliveryDate: formatOptionalRFC3339(o.RequestedDeliveryDate),
			ReceivingWindowOpen:   o.ReceivingWindowOpen,
			ReceivingWindowClose:  o.ReceivingWindowClose,
			LineItems:             o.LineItems,
		}); err != nil {
			return err
		}
		if bo != nil {
			if err := outbox.EmitJSON(ctx, txn, events.AggregateOrder, bo.OrderID, events.TopicMain, events.OrderEvent{
				BaseEvent:          events.BaseEvent{Type: events.EventOrderCreated, Timestamp: bo.CreatedAt.Format(time.RFC3339Nano)},
				OrderID:            bo.OrderID,
				SupplierID:         bo.SupplierID,
				RetailerID:         bo.RetailerID,
				WarehouseID:        bo.WarehouseID,
				Status:             string(bo.Status),
				OrderSource:        string(bo.Source),
				ConfirmationStatus: string(bo.ConfirmationStatus),
				TotalMinor:         bo.TotalMinor,
				Currency:           bo.Currency,
				LineItems:          bo.LineItems,
			}); err != nil {
				return err
			}
		}
		if o.Status == StatusScheduled {
			if err := emitPreorderEvent(ctx, txn, events.EventPreOrderNotified, o, string(auth.RoleRetailer), retailerID); err != nil {
				return err
			}
		}
		if ab, ok := txn.(outbox.AuditBuffer); ok {
			return outbox.WriteAudit(ctx, ab, o.SupplierID, retailerID, "RETAILER", "ORDER_CREATED", "Order", o.OrderID, map[string]any{
				"receiving_window_open":  o.ReceivingWindowOpen,
				"receiving_window_close": o.ReceivingWindowClose,
			})
		}
		return nil
	}, stockOpts)
"""

pattern_1 = re.compile(r'\n\tstockOpts := StockReservationOpts\{.*?\n\t}, stockOpts\)\n', re.DOTALL)
content = pattern_1.sub(replacement_1, content)


# 2. We need to cancel the backorder if allocation fails.
replacement_2 = """
	if stockOpts.Skip && o.Status == StatusPending {
		if err := s.ConfirmAndAllocate(ctx, o.OrderID); err != nil {
			if _, cancelErr := s.cancelOrderWithReason(ctx, &o, "system", "SYSTEM", "allocation_failed", ""); cancelErr != nil {
				s.log.Warn("failed to cancel order after allocation failure", "order_id", o.OrderID, "err", cancelErr)
			}
			if bo != nil {
				if _, cancelErr := s.cancelOrderWithReason(ctx, bo, "system", "SYSTEM", "parent_allocation_failed", ""); cancelErr != nil {
					s.log.Warn("failed to cancel backorder after allocation failure", "order_id", bo.OrderID, "err", cancelErr)
				}
			}
			return CreateResponse{}, fmt.Errorf("allocation failed: %w", err)
		}
		o, _, _ = s.repo.GetOrder(ctx, o.OrderID)
	}

	var backorderOrderID string
	if bo != nil {
		backorderOrderID = bo.OrderID
	}
"""

pattern_2 = re.compile(r'\n\tif stockOpts\.Skip && o\.Status == StatusPending \{.*?\n\t\}\n\n\tvar backorderOrderID string\n\tif len\(invPlan\.Backorder\) > 0 \{\n\t\tbackorderOrderID, err = s\.createBackorderOrder.*?\}\n\t\}\n', re.DOTALL)
content = pattern_2.sub(replacement_2, content)

with open("apps/backend-go/order/service.go", "w") as f:
    f.write(content)

