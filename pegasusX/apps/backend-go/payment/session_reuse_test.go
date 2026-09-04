package payment

import (
	"context"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type mockSessionReuseRepo struct {
	session SessionRecord
}

func (m *mockSessionReuseRepo) CreateSession(ctx context.Context, s SessionRecord, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *mockSessionReuseRepo) CreateSessionWithAttempt(ctx context.Context, s SessionRecord, a PaymentAttemptRecord, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *mockSessionReuseRepo) SaveAttempt(ctx context.Context, a PaymentAttemptRecord, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *mockSessionReuseRepo) SaveChargeback(ctx context.Context, c ChargebackRecord, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *mockSessionReuseRepo) SaveReversal(ctx context.Context, rev ReversalRecord, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *mockSessionReuseRepo) SaveWebhook(ctx context.Context, w WebhookRecord, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *mockSessionReuseRepo) FindStuckSessions(ctx context.Context, cutoff time.Time, limit int) ([]SessionRecord, error) {
	return nil, nil
}
func (m *mockSessionReuseRepo) GetSession(ctx context.Context, sessionID string) (SessionRecord, bool, error) {
	if m.session.SessionID == sessionID {
		return m.session, true, nil
	}
	return SessionRecord{}, false, nil
}
func (m *mockSessionReuseRepo) GetSessionByOrderID(ctx context.Context, orderID string) (SessionRecord, bool, error) {
	if m.session.OrderID == orderID {
		return m.session, true, nil
	}
	return SessionRecord{}, false, nil
}
func (m *mockSessionReuseRepo) HasChargebackForOrder(ctx context.Context, orderID string) (bool, error) {
	return false, nil
}

func TestInitCheckoutSession_ReusesActiveSession(t *testing.T) {
	repo := &mockSessionReuseRepo{
		session: SessionRecord{
			SessionID:   "psess_existing_123",
			OrderID:     "ord_reuse_1",
			SupplierID:  "sup-1",
			RetailerID:  "ret-1",
			Gateway:     "GLOBAL_PAY",
			Currency:    "UZS",
			AmountMinor: 50000,
			Mode:        "CARD",
			Status:      "PAYMENT_REQUIRED",
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		},
	}

	svc := NewService(ServiceConfig{
		SupplierID: "sup-1",
		Repo:       repo,
	})

	req := CheckoutRequest{
		OrderID:     "ord_reuse_1",
		RetailerID:  "ret-1",
		Gateway:     "GLOBAL_PAY",
		Currency:    "UZS",
		AmountMinor: 50000,
	}

	sess, attempt, exec, err := svc.initCheckoutSession(context.Background(), "CARD", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sess.SessionID != "psess_existing_123" {
		t.Fatalf("expected reused session ID psess_existing_123, got %q", sess.SessionID)
	}
	if attempt.SessionID != "psess_existing_123" {
		t.Fatalf("expected attempt session ID psess_existing_123, got %q", attempt.SessionID)
	}
	if exec.ResolvedGateway != "GLOBAL_PAY" {
		t.Fatalf("expected gateway GLOBAL_PAY, got %q", exec.ResolvedGateway)
	}
}

func (m *mockSessionReuseRepo) ListLedgerEntries(ctx context.Context, q LedgerQuery) ([]LedgerEntryRecord, error) {
	return nil, nil
}
func (m *mockSessionReuseRepo) SummarizeLedgerEntries(ctx context.Context, q SettlementAuthorityQuery) ([]SettlementAuthorityRow, error) {
	return nil, nil
}
func (m *mockSessionReuseRepo) CreatePayer(ctx context.Context, p Payer) error {
	return nil
}
func (m *mockSessionReuseRepo) GetPayer(ctx context.Context, payerID string) (Payer, error) {
	return Payer{}, nil
}
func (m *mockSessionReuseRepo) UpdatePayer(ctx context.Context, p Payer) error {
	return nil
}
func (m *mockSessionReuseRepo) ListPayers(ctx context.Context, limit, offset int) ([]Payer, error) {
	return nil, nil
}
