package payment

import (
	"context"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/order"
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
	// 1. Fetch sessions stuck in AWAITING_PAYMENT (or similar) from the database
	// This would require a repository method like `paymentRepo.FindStuckSessions(cutoffTime)`

	// 2. For each session, invoke the execution router to check status:
	// result, err := r.execution.Execute(ctx, ExecutionRequest{
	//     Action: ExecutionActionStatusCheck, // (requires addition to execution actions)
	//     ...
	// })

	// 3. Update session and order state if the payment was actually captured

	// Since we are mocking/stubbing for now based on the GlobalPay docs constraint:
	return nil
}
