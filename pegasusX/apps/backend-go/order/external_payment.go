package order

import (
	"context"
	"fmt"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// SettleExternalPayment transitions an order from AWAITING_PAYMENT to COMPLETED, driven by webhook notifications.
func (s *Service) SettleExternalPayment(ctx context.Context, orderID string, gateway string) error {
	orderRecord, found, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("order not found: %s", orderID)
	}
	if orderRecord.Status != StatusAwaitingPayment {
		if orderRecord.Status == StatusCancelled {
			// Money captured against a cancelled order must never vanish into a
			// log line: route it to the supplier reconciliation queue.
			return s.flagPaymentAfterCancel(ctx, orderRecord, gateway)
		}
		s.log.InfoContext(ctx, "skipping external payment settlement, order not awaiting payment", "order_id", orderID, "status", orderRecord.Status)
		return nil
	}

	previousStatus := orderRecord.Status
	orderRecord.Status = StatusCompleted
	// Version must stay at the value read from Spanner: UpdateOrder compares it
	// against the stored row for optimistic concurrency and increments it itself.
	orderRecord.UpdatedAt = s.now()

	err = s.repo.UpdateOrder(ctx, orderRecord, nil, func(txn outbox.TxnBuffer) error {
		if err := emitOrderStatusChanged(ctx, txn, orderStatusEmitParams{
			Order:          orderRecord,
			PreviousStatus: previousStatus,
			Reason:         "external_payment_cleared",
		}); err != nil {
			return err
		}
		if err := emitPaymentCleared(ctx, txn, orderRecord, gateway); err != nil {
			return err
		}
		return emitOrderFinalized(ctx, txn, orderRecord)
	})
	if err != nil {
		return err
	}

	s.afterOrderMutation(ctx, orderRecord)

	// Manifest completion logic
	if orderRecord.ManifestID != "" {
		if err := s.tryCompleteManifest(ctx, orderRecord.ManifestID); err != nil {
			s.log.ErrorContext(ctx, "failed to complete manifest during settlement", "manifest_id", orderRecord.ManifestID, "err", err)
		}
	}

	return nil
}

// flagPaymentAfterCancel moves a cancelled order that still received a gateway
// settlement into RECONCILIATION_REQUIRED so the supplier reconciliation queue
// (GET /v1/supplier/reconciliation) surfaces the trapped funds. Idempotent.
func (s *Service) flagPaymentAfterCancel(ctx context.Context, orderRecord Order, gateway string) error {
	if orderRecord.Status != StatusCancelled {
		return nil
	}
	previousStatus := orderRecord.Status
	orderRecord.Status = StatusReconciliationRequired
	orderRecord.UpdatedAt = s.now()

	err := s.repo.UpdateOrder(ctx, orderRecord, nil, func(txn outbox.TxnBuffer) error {
		return emitOrderStatusChanged(ctx, txn, orderStatusEmitParams{
			Order:          orderRecord,
			PreviousStatus: previousStatus,
			Reason:         "payment_received_after_cancel:" + strings.TrimSpace(gateway),
		})
	})
	if err != nil {
		return fmt.Errorf("flag payment after cancel %s: %w", orderRecord.OrderID, err)
	}
	s.invalidatePaymentCaches(ctx, orderRecord)
	s.afterOrderMutation(ctx, orderRecord)
	s.log.WarnContext(ctx, "payment settled for cancelled order, moved to reconciliation",
		"order_id", orderRecord.OrderID, "gateway", gateway)
	return nil
}

func (s *Service) tryCompleteManifest(ctx context.Context, manifestID string) error {
	orders, err := s.repo.ListManifestOrders(ctx, manifestID)
	if err != nil {
		return err
	}

	allDone := true
	driverID := ""
	for _, o := range orders {
		if !isTerminalStatus(o.Status) {
			allDone = false
			break
		}
		if driverID == "" && strings.TrimSpace(o.DriverID) != "" {
			driverID = strings.TrimSpace(o.DriverID)
		}
	}

	if !allDone {
		return nil
	}
	if s.manifestStore == nil {
		s.log.InfoContext(ctx, "manifest fully completed but manifest store unavailable", "manifest_id", manifestID)
		return nil
	}
	if driverID == "" {
		return fmt.Errorf("manifest %s completed but driver_id missing", manifestID)
	}

	returned, ok, err := s.manifestStore.ReturnDriver(ctx, driverID, s.now())
	if err != nil {
		return err
	}
	if ok {
		s.log.InfoContext(ctx, "manifest completed and driver returned", "manifest_id", manifestID, "driver_id", driverID, "returned_manifest_id", returned.ManifestID)
	}
	return nil
}

func isTerminalStatus(st Status) bool {
	switch st {
	case StatusCompleted, StatusCancelled:
		return true
	default:
		return false
	}
}
