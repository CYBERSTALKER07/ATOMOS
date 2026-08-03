package order

import (
	"context"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// StartDeferredPaymentSweeper runs periodically to find backordered orders that can now be fulfilled,
// captures payment, and transitions them to PENDING.
func (s *Service) StartDeferredPaymentSweeper(ctx context.Context, interval time.Duration) {
	if s.spannerClient == nil || s.paymentCapturer == nil {
		s.log.Warn("DeferredPaymentSweeper disabled: spanner or paymentCapturer not configured")
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.sweepDeferredPayments(ctx); err != nil {
				s.log.ErrorContext(ctx, "sweepDeferredPayments failed", "err", err)
			}
		}
	}
}

func (s *Service) sweepDeferredPayments(ctx context.Context) error {
	orders, err := s.repo.ListBackorderedOrders(ctx, 100)
	if err != nil {
		return err
	}
	for _, o := range orders {
		// Verify if stock is now available by planning inventory (read-only check).
		// We use nil checkout policy override here as this is a background check.
		invPlan, err := PlanInventoryCheckout(ctx, s.spannerClient, o.SupplierID, o.WarehouseID, o.LineItems, "")
		if err != nil {
			s.log.ErrorContext(ctx, "failed to plan inventory for backordered order", "order_id", o.OrderID, "err", err)
			continue
		}
		// If there are still backordered items, it's not ready.
		if len(invPlan.Backorder) > 0 {
			continue
		}

		// Capture the deferred payment.
		err = s.paymentCapturer.CaptureCardPayment(ctx, o.OrderID, o.TotalMinor, o.Currency)
		if err != nil {
			s.log.Warn("failed to capture payment for backordered order", "order_id", o.OrderID, "err", err)
			// Payment failed, we leave it in backorder for now.
			continue
		}

		// Payment succeeded, convert backorder to active order.
		stockOpts := StockReservationOpts{}
		if s.allocationRequired {
			stockOpts.Skip = true
		}
		if err := s.repo.ClearBackorder(ctx, o.OrderID, func(txn outbox.TxnBuffer) error {
			return outbox.EmitJSON(ctx, txn, events.AggregateOrder, o.OrderID, events.TopicMain, events.OrderEvent{
				BaseEvent:      events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: s.now().UTC().Format(time.RFC3339Nano)},
				OrderID:        o.OrderID,
				SupplierID:     o.SupplierID,
				RetailerID:     o.RetailerID,
				WarehouseID:    o.WarehouseID,
				PreviousStatus: string(StatusBackordered),
				Status:         string(StatusPending),
				Reason:         "backorder_fulfilled",
				ActorRole:      "system",
				OrderSource:    string(o.Source),
			})
		}, stockOpts); err != nil {
			s.log.ErrorContext(ctx, "failed to clear backorder", "order_id", o.OrderID, "err", err)
			continue
		}
		if s.allocationRequired {
			if err := s.ConfirmAndAllocate(ctx, o.OrderID); err != nil {
				s.log.ErrorContext(ctx, "failed to allocate cleared backorder", "order_id", o.OrderID, "err", err)
				continue
			}
		}
		s.log.Info("backordered order payment captured and activated", "order_id", o.OrderID)
	}
	return nil
}
