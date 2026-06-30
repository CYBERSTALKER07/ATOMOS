package payment

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

func TestHandleCheckout_PreDeliveryDisabled(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)

	tests := []struct {
		name string
		path string
		call func(http.ResponseWriter, *http.Request)
	}{
		{"b2b", "/v1/checkout/b2b", svc.HandleB2BCheckout},
		{"unified_order_id", "/v1/checkout/unified", svc.HandleUnifiedCheckout},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(`{"order_id":"o-1","retailer_id":"r-1","gateway":"ADYEN","amount_minor":1000}`))
			res := httptest.NewRecorder()
			tc.call(res, req)
			if res.Code != http.StatusGone {
				t.Fatalf("status = %d, want %d body=%s", res.Code, http.StatusGone, res.Body.String())
			}
			if repo.createCalls != 0 {
				t.Fatalf("createCalls = %d, want 0", repo.createCalls)
			}
		})
	}
}

func TestHandleCheckout_AirwallexPolicyViolation(t *testing.T) {
	t.Skip("pre-delivery B2B checkout removed; pay at delivery only")
}

func TestHandleChargeback_UnknownGatewayPolicyViolation(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)

	req := httptest.NewRequest(http.MethodPost, "/v1/payment/chargeback", strings.NewReader(`{"order_id":"o-3","retailer_id":"r-3","gateway":"UNKNOWN","amount":700}`))
	res := httptest.NewRecorder()

	svc.HandleChargeback(res, req)
	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusUnprocessableEntity)
	}
	if repo.chargebackCalls != 0 {
		t.Fatalf("chargebackCalls = %d, want 0", repo.chargebackCalls)
	}

	var payload map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["code"] != "payment_gateway_policy_violation" {
		t.Fatalf("code = %v, want payment_gateway_policy_violation", payload["code"])
	}
}

func TestHandleChargeback_EmitsSupplierScopedEvent(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)

	req := httptest.NewRequest(http.MethodPost, "/v1/payment/chargeback", strings.NewReader(`{"order_id":"o-3","retailer_id":"r-3","gateway":"ADYEN","amount":700}`))
	res := httptest.NewRecorder()

	svc.HandleChargeback(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if len(repo.lastOutboxEvents) != 2 {
		t.Fatalf("outbox events = %d, want 2", len(repo.lastOutboxEvents))
	}

	var payload paymentEvent
	if err := json.Unmarshal(repo.lastOutboxEvents[0].Payload, &payload); err != nil {
		t.Fatalf("decode outbox payload: %v", err)
	}
	if payload.SupplierID != "supplier-1" {
		t.Fatalf("supplier_id = %q, want supplier-1", payload.SupplierID)
	}
	if payload.Status != "CHARGEBACK_RECORDED" {
		t.Fatalf("status = %q, want CHARGEBACK_RECORDED", payload.Status)
	}

	var disputed struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(repo.lastOutboxEvents[1].Payload, &disputed); err != nil {
		t.Fatalf("decode dispute payload: %v", err)
	}
	if disputed.Type != events.EventDeliveryDisputed {
		t.Fatalf("event type = %q, want %s", disputed.Type, events.EventDeliveryDisputed)
	}
}

func TestHandleChargebackReversal_EmitsSupplierScopedEvent(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{hasChargeback: true}
	svc := newPaymentServiceForExecutionTest(repo)

	req := httptest.NewRequest(http.MethodPost, "/v1/payment/chargeback/reversal", strings.NewReader(`{"session_id":"sess-1"}`))
	res := httptest.NewRecorder()

	svc.HandleChargebackReversal(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if len(repo.lastOutboxEvents) != 1 {
		t.Fatalf("outbox events = %d, want 1", len(repo.lastOutboxEvents))
	}

	var payload paymentEvent
	if err := json.Unmarshal(repo.lastOutboxEvents[0].Payload, &payload); err != nil {
		t.Fatalf("decode outbox payload: %v", err)
	}
	if payload.SupplierID != "supplier-1" {
		t.Fatalf("supplier_id = %q, want supplier-1", payload.SupplierID)
	}
	if payload.Status != "CHARGEBACK_REVERSAL_RECORDED" {
		t.Fatalf("status = %q, want CHARGEBACK_REVERSAL_RECORDED", payload.Status)
	}
}

func TestHandleChargebackReversal_FailsWhenNoChargeback(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{hasChargeback: false}
	svc := newPaymentServiceForExecutionTest(repo)

	req := httptest.NewRequest(http.MethodPost, "/v1/payment/chargeback/reversal", strings.NewReader(`{"session_id":"sess-1"}`))
	res := httptest.NewRecorder()

	svc.HandleChargebackReversal(res, req)
	if res.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusConflict)
	}
}

func newPaymentServiceForExecutionTest(repo *paymentRepoStub) *Service {
	cacheClient := cache.New(cache.NewInMemoryBackend(), slog.Default())
	return NewService(ServiceConfig{
		Repo:                   repo,
		Cache:                  cacheClient,
		Idem:                   idempotency.NewInMemoryStore(),
		SupplierID:             "supplier-1",
		Currency:               "UZS",
		GlobalPayWebhookSecret: "gp-secret",
		AdyenWebhookSecret:     "adyen-secret",
		StripeWebhookSecret:    "stripe-secret",
		Now:                    func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
		Policy:                 testPolicyResolver{},
	})
}

type testPolicyResolver struct{}

func (testPolicyResolver) Resolve(_ context.Context, _, _ string) (GatewayPolicy, error) {
	return NormalizeGatewayPolicy(PaymentAcceptorSupplier, []string{
		"GLOBAL_PAY", "ADYEN", "AIRWALLEX", "STRIPE", "CASH",
	}, "SUPPLIER_DEFAULT"), nil
}

func (testPolicyResolver) AllowedGateways(ctx context.Context, supplierID, warehouseID string) ([]string, string, error) {
	policy, err := (testPolicyResolver{}).Resolve(ctx, supplierID, warehouseID)
	if err != nil {
		return nil, "", err
	}
	return policy.AllowedGateways, policy.Acceptor, nil
}

type paymentRepoStub struct {
	createCalls            int
	createWithAttemptCalls int
	created                SessionRecord
	attemptCalls           int
	lastAttempt            PaymentAttemptRecord
	chargebackCalls        int
	reversalCalls          int
	webhookCalls           int
	ledgerItems            []LedgerEntryRecord
	lastLedgerQuery        LedgerQuery
	settlementRows         []SettlementAuthorityRow
	lastSettlementQuery    SettlementAuthorityQuery
	lastOutboxEvents       []outbox.Event
	hasChargeback          bool
}

func (r *paymentRepoStub) CreateSession(ctx context.Context, s SessionRecord, emit func(outbox.TxnBuffer) error) error {
	r.createCalls++
	r.created = s
	if emit != nil {
		txn := &paymentTxnBufferStub{}
		if err := emit(txn); err != nil {
			return err
		}
		r.lastOutboxEvents = append([]outbox.Event(nil), txn.events...)
	}
	_ = ctx
	return nil
}

func (r *paymentRepoStub) CreateSessionWithAttempt(ctx context.Context, s SessionRecord, a PaymentAttemptRecord, emit func(outbox.TxnBuffer) error) error {
	r.createWithAttemptCalls++
	r.createCalls++
	r.attemptCalls++
	r.created = s
	r.lastAttempt = a
	if emit != nil {
		txn := &paymentTxnBufferStub{}
		if err := emit(txn); err != nil {
			return err
		}
		r.lastOutboxEvents = append([]outbox.Event(nil), txn.events...)
	}
	_ = ctx
	return nil
}

func (r *paymentRepoStub) SaveAttempt(ctx context.Context, a PaymentAttemptRecord, emit func(outbox.TxnBuffer) error) error {
	r.attemptCalls++
	r.lastAttempt = a
	if emit != nil {
		txn := &paymentTxnBufferStub{}
		if err := emit(txn); err != nil {
			return err
		}
		r.lastOutboxEvents = append([]outbox.Event(nil), txn.events...)
	}
	_ = ctx
	return nil
}

func (r *paymentRepoStub) SaveChargeback(ctx context.Context, c ChargebackRecord, emit func(outbox.TxnBuffer) error) error {
	r.chargebackCalls++
	if emit != nil {
		txn := &paymentTxnBufferStub{}
		if err := emit(txn); err != nil {
			return err
		}
		r.lastOutboxEvents = append([]outbox.Event(nil), txn.events...)
	}
	_ = c
	_ = ctx
	return nil
}

func (r *paymentRepoStub) SaveReversal(ctx context.Context, rev ReversalRecord, emit func(outbox.TxnBuffer) error) error {
	r.reversalCalls++
	if emit != nil {
		txn := &paymentTxnBufferStub{}
		if err := emit(txn); err != nil {
			return err
		}
		r.lastOutboxEvents = append([]outbox.Event(nil), txn.events...)
	}
	_ = rev
	_ = ctx
	return nil
}

func (r *paymentRepoStub) SaveWebhook(ctx context.Context, w WebhookRecord, emit func(outbox.TxnBuffer) error) error {
	r.webhookCalls++
	if emit != nil {
		txn := &paymentTxnBufferStub{}
		if err := emit(txn); err != nil {
			return err
		}
	}
	_ = w
	_ = ctx
	return nil
}

func (r *paymentRepoStub) GetSession(ctx context.Context, sessionID string) (SessionRecord, bool, error) {
	if sessionID == "sess-1" {
		return SessionRecord{
			SessionID: "sess-1",
			OrderID:   "o-3",
		}, true, nil
	}
	return SessionRecord{}, false, nil
}

func (r *paymentRepoStub) HasChargebackForOrder(ctx context.Context, orderID string) (bool, error) {
	return r.hasChargeback, nil
}


func (r *paymentRepoStub) FindStuckSessions(ctx context.Context, cutoff time.Time, limit int) ([]SessionRecord, error) {
	return nil, nil
}

func (r *paymentRepoStub) GetSessionByOrderID(ctx context.Context, orderID string) (SessionRecord, bool, error) {
	if r.created.OrderID == orderID {
		return r.created, true, nil
	}
	return SessionRecord{}, false, nil
}

func (r *paymentRepoStub) ListLedgerEntries(_ context.Context, q LedgerQuery) ([]LedgerEntryRecord, error) {
	r.lastLedgerQuery = q
	items := make([]LedgerEntryRecord, len(r.ledgerItems))
	copy(items, r.ledgerItems)
	return items, nil
}

func (r *paymentRepoStub) SummarizeLedgerEntries(_ context.Context, q SettlementAuthorityQuery) ([]SettlementAuthorityRow, error) {
	r.lastSettlementQuery = q
	rows := make([]SettlementAuthorityRow, len(r.settlementRows))
	copy(rows, r.settlementRows)
	return rows, nil
}

type paymentTxnBufferStub struct {
	events []outbox.Event
}

func (b *paymentTxnBufferStub) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}
