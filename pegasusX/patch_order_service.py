import re

with open("apps/backend-go/order/service.go", "r") as f:
    content = f.read()

# Replace the CreateOrderWithBackorder call
pattern = re.compile(r'err = s\.repo\.CreateOrderWithBackorder\(ctx, &o, bo, func\(txn outbox\.TxnBuffer\) error \{.*?\n\t\t\}\n\t\treturn nil\n\t\}, stockOpts\)\n\tif err != nil \{\n\t\treturn CreateResponse\{\}, fmt\.Errorf\("persist order: %w", err\)\n\t\}\n\n\tif s\.credit != nil && total > 0 && creditReserveAtCreateEnabled\(\) \{\n\t\tif check, cerr := s\.credit\.CheckCreditPath\(ctx, retailerID, supplierID, total\); cerr == nil && check\.Allowed \{\n\t\t\tif rerr := s\.credit\.Reserve\(ctx, retailerID, supplierID, o\.OrderID, total\); rerr != nil \{\n\t\t\t\ts\.log\.Warn\("credit reserve at create failed", "order_id", o\.OrderID, "err", rerr\)\n\t\t\t\}\n\t\t\}\n\t\}', re.DOTALL)

replacement = r"""	inTxn := func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if s.credit != nil && total > 0 && creditReserveAtCreateEnabled() {
			check, cerr := s.credit.CheckCreditPath(ctx, retailerID, supplierID, total)
			if cerr != nil {
				return cerr
			}
			if !check.Allowed {
				return ErrCreditLimitBreached
			}
			if rerr := s.credit.ReserveOrderInTxn(ctx, txn, retailerID, supplierID, o.OrderID, total); rerr != nil {
				return fmt.Errorf("credit reserve failed: %w", rerr)
			}
		}
		return nil
	}

	err = s.repo.CreateOrderWithBackorder(ctx, &o, bo, inTxn, func(txn outbox.TxnBuffer) error {
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
	if err != nil {
		return CreateResponse{}, fmt.Errorf("persist order: %w", err)
	}"""
content = pattern.sub(replacement, content)

with open("apps/backend-go/order/service.go", "w") as f:
    f.write(content)

