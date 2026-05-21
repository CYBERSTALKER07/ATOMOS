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
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

func TestHandleCheckout_AirwallexPolicyViolation(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)

	req := httptest.NewRequest(http.MethodPost, "/v1/checkout/b2b", strings.NewReader(`{"order_id":"o-1","retailer_id":"r-1","gateway":"AIRWALLEX","amount_minor":1000}`))
	res := httptest.NewRecorder()

	svc.HandleB2BCheckout(res, req)
	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusUnprocessableEntity)
	}
	if repo.createCalls != 0 {
		t.Fatalf("createCalls = %d, want 0", repo.createCalls)
	}

	var payload map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["code"] != "card_tokenization_gateway_unsupported" {
		t.Fatalf("code = %v, want card_tokenization_gateway_unsupported", payload["code"])
	}
}

func TestHandleCheckout_UsesExecutionRouterResult(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)

	req := httptest.NewRequest(http.MethodPost, "/v1/checkout/unified", strings.NewReader(`{"order_id":"o-2","retailer_id":"r-2","gateway":"ADYEN","amount_minor":2300}`))
	res := httptest.NewRecorder()

	svc.HandleUnifiedCheckout(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if repo.createCalls != 1 {
		t.Fatalf("createCalls = %d, want 1", repo.createCalls)
	}
	if repo.createWithAttemptCalls != 1 {
		t.Fatalf("createWithAttemptCalls = %d, want 1", repo.createWithAttemptCalls)
	}
	if repo.attemptCalls != 1 {
		t.Fatalf("attemptCalls = %d, want 1", repo.attemptCalls)
	}
	if repo.created.Gateway != "ADYEN" {
		t.Fatalf("persisted gateway = %s, want ADYEN", repo.created.Gateway)
	}
	if repo.created.Mode != "UNIFIED" {
		t.Fatalf("persisted mode = %s, want UNIFIED", repo.created.Mode)
	}
	if repo.lastAttempt.ExecutionAction != string(ExecutionActionCheckoutInit) {
		t.Fatalf("attempt execution_action = %s, want %s", repo.lastAttempt.ExecutionAction, ExecutionActionCheckoutInit)
	}
	if repo.lastAttempt.ExecutionMode != string(ExecutionModeHostedRedirect) {
		t.Fatalf("attempt execution_mode = %s, want %s", repo.lastAttempt.ExecutionMode, ExecutionModeHostedRedirect)
	}

	var payload CheckoutResponse
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.ResolvedGateway != "ADYEN" {
		t.Fatalf("resolved gateway = %s, want ADYEN", payload.ResolvedGateway)
	}
	if !strings.Contains(payload.PaymentURL, "/v1/payment/redirect/adyen/") {
		t.Fatalf("payment_url = %s, expected adyen redirect", payload.PaymentURL)
	}
	if payload.PolicySource != "SUPPLIER_DEFAULT" {
		t.Fatalf("policy_source = %s, want SUPPLIER_DEFAULT", payload.PolicySource)
	}
	if payload.AttemptID == "" {
		t.Fatal("attempt_id should be present")
	}
	if payload.ExecutionAction != string(ExecutionActionCheckoutInit) {
		t.Fatalf("execution_action = %s, want %s", payload.ExecutionAction, ExecutionActionCheckoutInit)
	}
	if payload.ExecutionMode != string(ExecutionModeHostedRedirect) {
		t.Fatalf("execution_mode = %s, want %s", payload.ExecutionMode, ExecutionModeHostedRedirect)
	}
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
	})
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
}

func (r *paymentRepoStub) CreateSession(ctx context.Context, s SessionRecord, emit func(outbox.TxnBuffer) error) error {
	r.createCalls++
	r.created = s
	if emit != nil {
		txn := &paymentTxnBufferStub{}
		if err := emit(txn); err != nil {
			return err
		}
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

type paymentTxnBufferStub struct {
	events []outbox.Event
}

func (b *paymentTxnBufferStub) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}
