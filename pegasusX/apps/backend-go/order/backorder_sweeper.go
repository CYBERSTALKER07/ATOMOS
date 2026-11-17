package order

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
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

		// Step 1: Transactionally claim the backorder & reserve stock BEFORE charging the card
		// so a concurrent checkout cannot steal stock after payment capture.
		claimTag := fmt.Sprintf("CAPTURING:%d", s.now().Unix())
		reservedOrder, claimed, claimErr := s.claimAndReserveBackorderInTxn(ctx, o, claimTag)
		if claimErr != nil {
			s.log.Warn("backorder pre-capture claim/reserve skipped", "order_id", o.OrderID, "err", claimErr)
			continue
		}
		if !claimed {
			continue
		}

		// Step 2: Capture the deferred payment now that inventory is safely reserved.
		providerRef, capErr := s.paymentCapturer.CaptureCardPayment(ctx, o.OrderID, reservedOrder.TotalMinor, reservedOrder.Currency)
		if capErr != nil {
			s.log.Warn("failed to capture payment for backordered order; releasing reserved stock", "order_id", o.OrderID, "err", capErr)
			if rerr := s.releaseBackorderClaimInTxn(ctx, reservedOrder); rerr != nil {
				s.log.ErrorContext(ctx, "failed to release backorder reservation after capture failure", "order_id", o.OrderID, "err", rerr)
			}
			continue
		}

		// Step 3: Commit payment capture leg + convert status BACKORDERED -> PENDING atomically.
		now := s.now()
		leg := PaymentLeg{
			OrderID:        reservedOrder.OrderID,
			LegID:          s.newID(),
			Method:         MethodCard,
			AmountMinor:    reservedOrder.TotalMinor,
			Status:         PaymentStatusCaptured,
			IdempotencyKey: "card-backorder-" + reservedOrder.OrderID,
			ProviderRef:    spanner.NullString{StringVal: providerRef, Valid: providerRef != ""},
			CreatedAt:      now,
			CapturedAt:     spanner.NullTime{Time: now, Valid: true},
		}
		if ferr := s.finalizeBackorderCaptureInTxn(ctx, reservedOrder, leg); ferr != nil {
			s.log.ErrorContext(ctx, "failed to finalize captured backorder", "order_id", reservedOrder.OrderID, "err", ferr)
			continue
		}
		s.log.Info("backordered order reserved, payment captured, and activated", "order_id", reservedOrder.OrderID)
	}
	return nil
}

func (s *Service) claimAndReserveBackorderInTxn(ctx context.Context, o Order, claimTag string) (Order, bool, error) {
	if s.spannerClient == nil {
		return o, true, nil
	}
	var out Order
	var claimed bool
	err := spannerutils.RunReadWriteTransaction(ctx, s.spannerClient, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, rerr := txn.ReadRow(txCtx, "Orders", spanner.Key{o.OrderID}, []string{"Status", "HoldReason", "WarehouseId", "UpdatedAt"})
		if rerr != nil {
			return rerr
		}
		var curStatus, curWH string
		var holdReason spanner.NullString
		var updatedAt spanner.NullTime
		if err := row.Columns(&curStatus, &holdReason, &curWH, &updatedAt); err != nil {
			return err
		}
		if curStatus != string(StatusBackordered) {
			return nil
		}
		if holdReason.Valid && strings.HasPrefix(holdReason.StringVal, "CAPTURING:") && updatedAt.Valid && s.now().Sub(updatedAt.Time) < 60*time.Second {
			return nil
		}
		out = o
		if strings.TrimSpace(curWH) != "" {
			out.WarehouseID = curWH
		}
		alreadyReserved := false
		if _, merr := txn.ReadRow(txCtx, "OrderStockReservationMarkers", spanner.Key{o.OrderID}, []string{"OrderId"}); merr == nil {
			alreadyReserved = true
		}
		if !alreadyReserved {
			if s.allocationRequired {
				if err := s.allocateAndReserveInTxn(txCtx, txn, &out); err != nil {
					return err
				}
			} else {
				expected := time.Time{}
				if out.RequestedDeliveryDate != nil {
					expected = out.RequestedDeliveryDate.UTC()
				} else if out.DeliverBefore != nil {
					expected = out.DeliverBefore.UTC()
				}
				if err := ReserveLineItemsForOrderInTxn(txCtx, txn, out.SupplierID, out.WarehouseID, out.OrderID, out.RetailerID, expected, out.LineItems); err != nil {
					return err
				}
			}
			if err := insertStockReservationMarkerInTxn(txn, out.OrderID); err != nil {
				return err
			}
		}
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("Orders", map[string]any{
				"OrderId":     out.OrderID,
				"WarehouseId": out.WarehouseID,
				"HoldReason":  spanner.NullString{StringVal: claimTag, Valid: true},
				"UpdatedAt":   spanner.CommitTimestamp,
			}),
		}); err != nil {
			return err
		}
		claimed = true
		return nil
	})
	return out, claimed, err
}

func (s *Service) releaseBackorderClaimInTxn(ctx context.Context, o Order) error {
	if s.spannerClient == nil {
		return nil
	}
	return spannerutils.RunReadWriteTransaction(ctx, s.spannerClient, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := releaseOrderReservationsInTxn(txCtx, txn, &o); err != nil {
			return err
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Delete("OrderStockReservationMarkers", spanner.Key{o.OrderID}),
			spanner.UpdateMap("Orders", map[string]any{
				"OrderId":    o.OrderID,
				"HoldReason": spanner.NullString{StringVal: "INSUFFICIENT_STOCK", Valid: true},
				"UpdatedAt":  spanner.CommitTimestamp,
			}),
		})
	})
}

func (s *Service) finalizeBackorderCaptureInTxn(ctx context.Context, o Order, leg PaymentLeg) error {
	return spannerutils.RunReadWriteTransaction(ctx, s.spannerClient, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, rerr := txn.ReadRow(txCtx, "Orders", spanner.Key{o.OrderID}, []string{"Version", "Status"})
		if rerr != nil {
			return rerr
		}
		var ver int64
		var st string
		if err := row.Columns(&ver, &st); err != nil {
			return err
		}
		if st != string(StatusBackordered) {
			return nil
		}
		if legErr := s.RecordPaymentLeg(txCtx, txn, leg); legErr != nil && status.Code(legErr) != codes.AlreadyExists {
			return legErr
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		if err := outbox.EmitJSON(txCtx, buf, events.AggregateOrder, o.OrderID, events.TopicMain, events.OrderEvent{
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
		}); err != nil {
			return err
		}
		if err := buf.Flush(txCtx); err != nil {
			return err
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("Orders", map[string]any{
				"OrderId":    o.OrderID,
				"Status":     string(StatusPending),
				"HoldReason": spanner.NullString{},
				"Version":    ver + 1,
				"UpdatedAt":  spanner.CommitTimestamp,
			}),
			spanner.InsertMap("OrderStatusTransitions", map[string]any{
				"OrderId":        o.OrderID,
				"TransitionId":   uuid.NewString(),
				"PreviousStatus": string(StatusBackordered),
				"NewStatus":      string(StatusPending),
				"Reason":         "backorder_fulfilled",
				"ActorRole":      "SYSTEM",
				"ActorId":        "system:backorder_sweeper",
				"EventKind":      DefaultTransitionEventKind(StatusPending),
				"MetadataJson":   []byte("{}"),
				"CreatedAt":      spanner.CommitTimestamp,
			}),
		})
	})
}
