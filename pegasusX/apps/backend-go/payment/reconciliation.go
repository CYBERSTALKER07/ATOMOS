package payment

import (
	"context"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// WebhookReconciler periodically polls the database for stuck payment sessions
// and queries the gateway for their true status.
type WebhookReconciler struct {
	paymentRepo Repository
	orderRepo   order.Repository
	execution   *ProviderExecutionRouter
	now         func() time.Time
}

// NewWebhookReconciler creates a new reconciler.
func NewWebhookReconciler(paymentRepo Repository, orderRepo order.Repository, execution *ProviderExecutionRouter, now func() time.Time) *WebhookReconciler {
	return &WebhookReconciler{
		paymentRepo: paymentRepo,
		orderRepo:   orderRepo,
		execution:   execution,
		now:         now,
	}
}

// ReconcileStuckSessions polls for sessions that have been awaiting payment for over 15 minutes.
func (r *WebhookReconciler) ReconcileStuckSessions(ctx context.Context) error {
	now := r.now()
	cutoff := now.Add(-15 * time.Minute)

	sessions, err := r.paymentRepo.FindStuckSessions(ctx, cutoff, 100)
	if err != nil {
		return err
	}

	for _, s := range sessions {
		req := ExecutionRequest{
			Action:      ExecutionActionStatusCheck,
			Gateway:     s.Gateway,
			SessionID:   s.SessionID,
			OrderID:     s.OrderID,
			AmountMinor: s.AmountMinor,
			Currency:    s.Currency,
		}
		result, err := r.execution.Execute(ctx, req)
		if err != nil {
			// Log error, continue with next session
			continue
		}

		status := "AWAITING_PAYMENT"
		if result.ProviderRef == "PAID" || result.ProviderRef == "gp_status_stub_paid" || result.ProviderRef == "AUTHORIZED" || result.ProviderRef == "CAPTURED" || result.ProviderRef == "SUCCESS" {
			status = "PAID"
		} else if result.ProviderRef == "FAILED" || result.ProviderRef == "DECLINED" || result.ProviderRef == "CANCELLED" {
			status = "FAILED"
		}

		if status != "AWAITING_PAYMENT" && status != s.Status {
			// Simulate a webhook received to advance the state securely through
			// the outbox and ledger.
			w := WebhookRecord{
				WebhookID:      "rec_" + s.SessionID + "_" + now.Format("20060102150405"),
				Gateway:        s.Gateway,
				TransactionID:  "rec_" + result.ProviderRef,
				SessionID:      s.SessionID,
				OrderID:        s.OrderID,
				SupplierID:     s.SupplierID,
				RetailerID:     s.RetailerID,
				Status:         status,
				AmountMinor:    s.AmountMinor,
				Currency:       s.Currency,
				SignatureValid: true,
				ReceivedAt:     now,
			}

			// We write this exactly like a webhook would be written, emitting the relevant event.
			_ = r.paymentRepo.SaveWebhook(ctx, w, func(txn outbox.TxnBuffer) error {
				eventType := events.EventPaymentRequired
				if status == "PAID" {
					eventType = events.EventPaymentCleared
				} else if status == "FAILED" {
					eventType = events.EventPaymentFailed
				}

				payload := events.FinanceEvent{
					BaseEvent: events.BaseEvent{
						Type:      eventType,
						Timestamp: now.Format(time.RFC3339Nano),
					},
					SessionID:     s.SessionID,
					OrderID:       s.OrderID,
					SupplierID:    s.SupplierID,
					RetailerID:    s.RetailerID,
					Gateway:       s.Gateway,
					Status:        status,
					AmountMinor:   s.AmountMinor,
					Currency:      s.Currency,
					TransactionID: w.TransactionID,
					Source:        "payment.reconciliation",
				}

				return outbox.EmitJSON(ctx, txn, events.AggregateSession, w.WebhookID, events.TopicMain, payload)
			})
		}
	}
	return nil
}
