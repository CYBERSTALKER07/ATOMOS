package payment

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type reconcilerPaymentRepoStub struct {
	paymentRepoStub
	stuckSessions []SessionRecord
	savedWebhooks []WebhookRecord
	emittedEvents []string
}

func (r *reconcilerPaymentRepoStub) FindStuckSessions(ctx context.Context, cutoff time.Time, limit int) ([]SessionRecord, error) {
	return r.stuckSessions, nil
}

func (r *reconcilerPaymentRepoStub) SaveWebhook(ctx context.Context, w WebhookRecord, emit func(outbox.TxnBuffer) error) error {
	r.savedWebhooks = append(r.savedWebhooks, w)
	buf := &spannerTxnBuffer{}
	if emit != nil {
		if err := emit(buf); err != nil {
			return err
		}
		for _, e := range buf.events {
			r.emittedEvents = append(r.emittedEvents, string(e.Payload))
		}
	}
	return nil
}

func TestWebhookReconciler_AllGatewaysAndStatuses(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	cutoff := now.Add(-20 * time.Minute)

	sessions := []SessionRecord{
		// 1. GlobalPay successful settlement
		{SessionID: "sess-gp-1", OrderID: "ord-gp-1", Gateway: "GLOBAL_PAY", Status: "AWAITING_PAYMENT", AmountMinor: 50000, Currency: "UZS", SupplierID: "sup-1", RetailerID: "ret-1", UpdatedAt: cutoff},
		// 2. Click status:2 (confirmed/paid)
		{SessionID: "sess-click-1", OrderID: "ord-click-1", Gateway: "CLICK", Status: "AWAITING_PAYMENT", AmountMinor: 75000, Currency: "UZS", SupplierID: "sup-1", RetailerID: "ret-1", UpdatedAt: cutoff},
		// 3. Payme state:2 (paid)
		{SessionID: "sess-payme-1", OrderID: "ord-payme-1", Gateway: "PAYME", Status: "AWAITING_PAYMENT", AmountMinor: 120000, Currency: "UZS", SupplierID: "sup-1", RetailerID: "ret-1", UpdatedAt: cutoff},
		// 4. Click status:-1 (cancelled -> FAILED)
		{SessionID: "sess-click-2", OrderID: "ord-click-2", Gateway: "CLICK", Status: "AWAITING_PAYMENT", AmountMinor: 30000, Currency: "UZS", SupplierID: "sup-1", RetailerID: "ret-1", UpdatedAt: cutoff},
		// 5. Payme state:-1 (cancelled -> FAILED)
		{SessionID: "sess-payme-2", OrderID: "ord-payme-2", Gateway: "PAYME", Status: "AWAITING_PAYMENT", AmountMinor: 40000, Currency: "UZS", SupplierID: "sup-1", RetailerID: "ret-1", UpdatedAt: cutoff},
		// 6. GlobalPay stub (must be skipped)
		{SessionID: "sess-gp-stub", OrderID: "ord-gp-stub", Gateway: "GLOBAL_PAY", Status: "AWAITING_PAYMENT", AmountMinor: 10000, Currency: "UZS", SupplierID: "sup-1", RetailerID: "ret-1", UpdatedAt: cutoff},
	}

	repo := &reconcilerPaymentRepoStub{
		stuckSessions: sessions,
	}

	// Mock execution router
	router := NewProviderExecutionRouter(ProviderExecutionRouterConfig{})
	mockExec := &mockStatusExecutor{
		statusResponses: map[string]ExecutionResult{
			"sess-gp-1":    {ResolvedGateway: "GLOBAL_PAY", ProviderRef: "SETTLED"},
			"sess-click-1": {ResolvedGateway: "CLICK", ProviderRef: "status:2"},
			"sess-payme-1": {ResolvedGateway: "PAYME", ProviderRef: "state:2"},
			"sess-click-2": {ResolvedGateway: "CLICK", ProviderRef: "status:-1"},
			"sess-payme-2": {ResolvedGateway: "PAYME", ProviderRef: "state:-1"},
			"sess-gp-stub": {ResolvedGateway: "GLOBAL_PAY", PolicySource: "GLOBAL_PAY_STUB", ProviderRef: "gp_status_stub_123"},
		},
	}
	router.executors["GLOBAL_PAY"] = mockExec
	router.executors["CLICK"] = mockExec
	router.executors["PAYME"] = mockExec

	reconciler := NewWebhookReconciler(repo, nil, router, func() time.Time { return now })

	err := reconciler.ReconcileStuckSessions(context.Background())
	if err != nil {
		t.Fatalf("ReconcileStuckSessions failed: %v", err)
	}

	// We expect 5 saved webhooks (gp-stub skipped)
	if len(repo.savedWebhooks) != 5 {
		t.Fatalf("expected 5 saved webhooks, got %d", len(repo.savedWebhooks))
	}

	// Verify statuses and unique transaction IDs
	seenTxIDs := make(map[string]bool)
	for _, w := range repo.savedWebhooks {
		if seenTxIDs[w.TransactionID] {
			t.Errorf("duplicate TransactionID across sessions: %s", w.TransactionID)
		}
		seenTxIDs[w.TransactionID] = true

		switch w.SessionID {
		case "sess-gp-1":
			if w.Status != "PAID" {
				t.Errorf("sess-gp-1 status = %s, want PAID", w.Status)
			}
			if !strings.Contains(w.TransactionID, "sess-gp-1") {
				t.Errorf("TransactionID missing session ID: %s", w.TransactionID)
			}
		case "sess-click-1":
			if w.Status != "PAID" {
				t.Errorf("sess-click-1 status = %s, want PAID", w.Status)
			}
		case "sess-payme-1":
			if w.Status != "PAID" {
				t.Errorf("sess-payme-1 status = %s, want PAID", w.Status)
			}
		case "sess-click-2":
			if w.Status != "FAILED" {
				t.Errorf("sess-click-2 status = %s, want FAILED", w.Status)
			}
		case "sess-payme-2":
			if w.Status != "FAILED" {
				t.Errorf("sess-payme-2 status = %s, want FAILED", w.Status)
			}
		default:
			t.Errorf("unexpected session saved: %s", w.SessionID)
		}
	}
}

type mockStatusExecutor struct {
	statusResponses map[string]ExecutionResult
}

func (m *mockStatusExecutor) Execute(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {
	if res, ok := m.statusResponses[req.SessionID]; ok {
		return res, nil
	}
	return ExecutionResult{ProviderRef: "UNKNOWN"}, nil
}
