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
		s.log.InfoContext(ctx, "skipping external payment settlement, order not awaiting payment", "order_id", orderID, "status", orderRecord.Status)
		return nil
	}

	previousStatus := orderRecord.Status
	orderRecord.Status = StatusCompleted
	orderRecord.Version++
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
