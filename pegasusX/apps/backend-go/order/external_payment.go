package order

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// SettleExternalPayment transitions AWAITING_PAYMENT → FISCALIZING after card/webhook clear (ADR-009).
// COMPLETED is deferred until fiscal SUCCESS (worker) or audited force-complete.
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
			return s.flagPaymentAfterCancel(ctx, orderRecord, gateway)
		}
		// P0 T7: late webhooks against terminal / post-capture states are no-ops.
		if isTerminalMoneyStatus(orderRecord.Status) ||
			orderRecord.Status == StatusFiscalizing ||
			orderRecord.Status == StatusFiscalFailed {
			s.log.InfoContext(ctx, "external payment settlement ignored (terminal or fiscal path)",
				"order_id", orderID, "status", orderRecord.Status, "gateway", gateway)
			return nil
		}
		s.log.InfoContext(ctx, "skipping external payment settlement, order not awaiting payment", "order_id", orderID, "status", orderRecord.Status)
		return nil
	}

	method := strings.TrimSpace(gateway)
	if method == "" {
		method = "CARD"
	}
	return s.beginFiscalFromAwaitingPayment(ctx, orderRecord, method, "external_payment_cleared", nil)
}

// ConfirmPaymentBypass validates driver ownership then opens the fiscal gate (ADR-009).
// Must never write COMPLETED directly — COMPLETED only after fiscal SUCCESS.
func (s *Service) ConfirmPaymentBypass(ctx context.Context, orderID, driverID, expectedSupplierID string) error {
	orderRecord, found, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if !found {
		return ErrOrderNotFound
	}
	if orderRecord.Status != StatusAwaitingPayment {
		return fmt.Errorf("%w: order must be AWAITING_PAYMENT, got %s", ErrInvalidStatusTransition, orderRecord.Status)
	}
	if strings.TrimSpace(orderRecord.DriverID) == "" || strings.TrimSpace(orderRecord.DriverID) != strings.TrimSpace(driverID) {
		return ErrOrderForbidden
	}
	if expectedSupplierID != "" && strings.TrimSpace(orderRecord.SupplierID) != strings.TrimSpace(expectedSupplierID) {
		return ErrOrderForbidden
	}
	ts := time.Now().UTC()
	if s.now != nil {
		ts = s.now()
	}
	extra := func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateOrder, orderID, events.TopicMain, map[string]any{
			"type":      "PAYMENT_BYPASS_CONFIRMED",
			"order_id":  orderID,
			"driver_id": driverID,
			"timestamp": ts.Format(time.RFC3339Nano),
		})
	}
	return s.beginFiscalFromAwaitingPayment(ctx, orderRecord, "PAYMENT_BYPASS", "payment_bypass_confirmed", extra)
}

// beginFiscalFromAwaitingPayment is the shared AWAITING_PAYMENT → FISCALIZING path.
func (s *Service) beginFiscalFromAwaitingPayment(
	ctx context.Context,
	orderRecord Order,
	method, reason string,
	extraEmit func(outbox.TxnBuffer) error,
) error {
	if err := s.requireFiscalPack(ctx, orderRecord.SupplierID); err != nil {
		return err
	}
	previousStatus := orderRecord.Status
	row := s.newFiscalPendingRow(ctx, orderRecord, method, "", orderRecord.TotalMinor)
	orderRecord.Status = StatusFiscalizing
	// Version must stay at the value read from Spanner: UpdateOrder compares it
	// against the stored row for optimistic concurrency and increments it itself.
	if s.now != nil {
		orderRecord.UpdatedAt = s.now()
	} else {
		orderRecord.UpdatedAt = time.Now().UTC()
	}
	orderRecord.FiscalStatus = FiscalStatusPending
	orderRecord.LatestFiscalAttemptID = row.AttemptID
	orderRecord.PendingFiscalReceipts = []FiscalReceiptRow{row}

	err := s.repo.UpdateOrder(ctx, orderRecord, nil, func(txn outbox.TxnBuffer) error {
		if err := emitOrderStatusChanged(ctx, txn, orderStatusEmitParams{
			Order:          orderRecord,
			PreviousStatus: previousStatus,
			Reason:         reason,
		}); err != nil {
			return err
		}
		if err := emitPaymentCaptureFiscal(ctx, txn, orderRecord, row, method); err != nil {
			return err
		}
		if extraEmit != nil {
			return extraEmit(txn)
		}
		return nil
	})
	if err != nil {
		return err
	}

	s.afterOrderMutation(ctx, orderRecord)
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
