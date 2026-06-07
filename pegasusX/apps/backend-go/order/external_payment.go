package order

import (
	"context"
	"fmt"

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
	s.broadcastOrderStatusChanged(ctx, orderRecord, previousStatus, "external_payment_cleared", orderRecord.Version)
	s.broadcastPaymentCleared(ctx, orderRecord)
	s.broadcastOrderFinalized(ctx, orderRecord)

	// Manifest completion logic
	if orderRecord.ManifestID != "" {
		if err := s.tryCompleteManifest(ctx, orderRecord.ManifestID); err != nil {
			s.log.ErrorContext(ctx, "failed to complete manifest during settlement", "manifest_id", orderRecord.ManifestID, "err", err)
		}
	}

	return nil
}

func (s *Service) tryCompleteManifest(ctx context.Context, manifestID string) error {
	// 1. Fetch all orders for this manifest
	orders, err := s.repo.ListManifestOrders(ctx, manifestID)
	if err != nil {
		return err
	}

	allDone := true
	for _, o := range orders {
		if !isTerminalStatus(o.Status) {
			allDone = false
			break
		}
	}

	if allDone {
		// Just emit an event for manifest completion for now
		s.log.InfoContext(ctx, "manifest is fully completed via external payment", "manifest_id", manifestID)
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
