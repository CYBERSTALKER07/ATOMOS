cat pegasusX/apps/backend-go/order/service.go | sed '/func (s \*Service) Create/i\
// Cancel cancels an order and emits an outbox event.\\
func (s *Service) Cancel(ctx context.Context, orderID string, retailerID string) error {\\
	o, found, err := s.repo.GetOrder(ctx, orderID)\\
	if err \!= nil {\\
		return fmt.Errorf("get order %s: %w", orderID, err)\\
	}\\
	if \!found {\\
		return fmt.Errorf("order %s not found", orderID)\\
	}\\
	if o.RetailerID \!= retailerID {\\
		return fmt.Errorf("retailer %s not authorized to cancel order %s", retailerID, orderID)\\
	}\\
	if o.Status \!= StatusPending {\\
		return fmt.Errorf("cannot cancel order in state %s", o.Status)\\
	}\\
\\
	o.Status = StatusCancelled\\
\\
	err = s.repo.UpdateOrder(ctx, o, func(txn outbox.TxnBuffer) error {\\
		return outbox.EmitJSON(txn, events.AggregateOrder, o.OrderID, events.TopicMain, events.EventManifestCancelled, o)\\
	})\\
	if err \!= nil {\\
		return fmt.Errorf("update order status: %w", err)\\
	}\\
\\
	keys := []string{fmt.Sprintf("order:%s", orderID), fmt.Sprintf("retailer:%s:orders", retailerID)}\\
	if err := s.cache.Invalidate(ctx, keys...); err \!= nil {\\
		// Best effort, log and continue\\
		fmt.Printf("cache invalidate failed: %v\\n", err)\\
	}\\
	return nil\\
}\\
' > new_service.go
mv new_service.go pegasusX/apps/backend-go/order/service.go
