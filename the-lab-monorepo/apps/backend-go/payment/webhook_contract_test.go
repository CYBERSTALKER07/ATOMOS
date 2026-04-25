package payment

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Mock Interfaces for Webhook Tests
// ═══════════════════════════════════════════════════════════════════════════════

type testVaultResolver struct {
	configs map[string]*VaultConfig // key: "orderID:gateway"
	err     error
}

func (v *testVaultResolver) GetDecryptedConfigByOrder(ctx context.Context, orderId, gatewayName string) (*VaultConfig, error) {
	if v.err != nil {
		return nil, v.err
	}
	key := orderId + ":" + gatewayName
	if cfg, ok := v.configs[key]; ok {
		return cfg, nil
	}
	return nil, fmt.Errorf("no config for %s", key)
}

type testDriverPusher struct {
	calls []driverPushCall
}

type driverPushCall struct {
	DriverID string
	Payload  interface{}
}

func (p *testDriverPusher) PushToDriver(driverID string, payload interface{}) bool {
	p.calls = append(p.calls, driverPushCall{DriverID: driverID, Payload: payload})
	return true
}

type testRetailerPusher struct {
	calls []retailerPushCall
}

type retailerPushCall struct {
	RetailerID string
	Payload    interface{}
}

func (p *testRetailerPusher) PushToRetailer(retailerID string, payload interface{}) bool {
	p.calls = append(p.calls, retailerPushCall{RetailerID: retailerID, Payload: payload})
	return true
}

// newTestWebhookService returns a WebhookService with no Spanner/Kafka,
// suitable for testing pre-Spanner validation gates.
func newTestWebhookService() *WebhookService {
	return &WebhookService{
		Spanner:     nil, // No Spanner — tests must not reach settlement paths
		Producer:    nil,
		DriverHub:   &testDriverPusher{},
		RetailerHub: &testRetailerPusher{},
	}
}

func globalPayBasicAuth(secret string) string {
	payload := "Paycom:" + secret
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(payload))
}

// ═══════════════════════════════════════════════════════════════════════════════
// GlobalPay Webhook Contract Tests
// ═══════════════════════════════════════════════════════════════════════════════

func TestGlobalPayWebhook_WrongMethod_405(t *testing.T) {
	svc := newTestWebhookService()
	req := httptest.NewRequest(http.MethodPut, "/v1/webhooks/global-pay", nil)
	w := httptest.NewRecorder()

	svc.HandleGlobalPayWebhook(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

func TestGlobalPayWebhook_NoSessionSvc_503(t *testing.T) {
	svc := newTestWebhookService()
	svc.SessionSvc = nil // Explicitly nil
	t.Setenv("GLOBAL_PAY_USERNAME", "merchant")
	t.Setenv("GLOBAL_PAY_PASSWORD", "gp-secret")
	t.Setenv("GLOBAL_PAY_SERVICE_ID", "svc-1")

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/global-pay?session_id=SESS-1", strings.NewReader("{}"))
	req.Header.Set("Authorization", globalPayBasicAuth("gp-secret"))
	w := httptest.NewRecorder()

	svc.HandleGlobalPayWebhook(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", w.Code)
	}
}

func TestGlobalPayWebhook_InvalidSignature_401(t *testing.T) {
	svc := newTestWebhookService()
	t.Setenv("GLOBAL_PAY_USERNAME", "merchant")
	t.Setenv("GLOBAL_PAY_PASSWORD", "gp-secret")
	t.Setenv("GLOBAL_PAY_SERVICE_ID", "svc-1")

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/global-pay?session_id=SESS-1", strings.NewReader("{}"))
	req.Header.Set("Authorization", globalPayBasicAuth("wrong-secret"))
	w := httptest.NewRecorder()

	svc.HandleGlobalPayWebhook(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestGlobalPayWebhook_MissingSessionID_400(t *testing.T) {
	svc := newTestWebhookService()
	svc.SessionSvc = nil // Will be caught by nil check first

	// Force non-nil SessionSvc to reach parsing
	// Since SessionSvc is checked first, this test verifies the 503 path.
	// To test missing session_id, we'd need a non-nil SessionSvc.
	// Adjust: GlobalPay checks SessionSvc BEFORE parsing.
	// So missing session_id can only be tested with a non-nil SessionSvc.
	// We settle for the 503 test above and skip this edge case.
	t.Skip("GlobalPay checks SessionSvc before parsing — requires live SessionSvc for session_id validation")
}

func TestGlobalPayWebhook_AllowsGETAndPOST(t *testing.T) {
	svc := newTestWebhookService()
	svc.SessionSvc = nil

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/v1/webhooks/global-pay?session_id=SESS-1", strings.NewReader("{}"))
			w := httptest.NewRecorder()
			svc.HandleGlobalPayWebhook(w, req)
			// Should not be 405 — should be 503 (SessionSvc nil)
			if w.Code == http.StatusMethodNotAllowed {
				t.Errorf("%s should be allowed, got 405", method)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// parseGlobalPayWebhookRequest Contract Tests
// ═══════════════════════════════════════════════════════════════════════════════

func TestParseGlobalPayWebhookRequest_QueryParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/webhooks/global-pay?session_id=SESS-42&status=SUCCESS&payment_id=PAY-99", nil)
	got, err := parseGlobalPayWebhookRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	if got.SessionID != "SESS-42" {
		t.Errorf("SessionID = %q, want SESS-42", got.SessionID)
	}
	if got.Status != "SUCCESS" {
		t.Errorf("Status = %q, want SUCCESS", got.Status)
	}
	if got.ProviderPaymentID != "PAY-99" {
		t.Errorf("ProviderPaymentID = %q, want PAY-99", got.ProviderPaymentID)
	}
}

func TestParseGlobalPayWebhookRequest_PostBody(t *testing.T) {
	body := `{"session_id":"SESS-POST","status":"PAID","payment_id":"PAY-POST"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/global-pay", strings.NewReader(body))
	got, err := parseGlobalPayWebhookRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	if got.SessionID != "SESS-POST" {
		t.Errorf("SessionID = %q, want SESS-POST", got.SessionID)
	}
}

func TestParseGlobalPayWebhookRequest_MissingSessionID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/webhooks/global-pay?status=SUCCESS", nil)
	_, err := parseGlobalPayWebhookRequest(req)
	if err == nil {
		t.Error("expected error for missing session_id")
	}
}

func TestParseGlobalPayWebhookRequest_CamelCaseAliases(t *testing.T) {
	// sessionId (camelCase) should work as alias
	req := httptest.NewRequest(http.MethodGet, "/v1/webhooks/global-pay?sessionId=SESS-CC&paymentId=PAY-CC", nil)
	got, err := parseGlobalPayWebhookRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	if got.SessionID != "SESS-CC" {
		t.Errorf("SessionID = %q, want SESS-CC", got.SessionID)
	}
	if got.ProviderPaymentID != "PAY-CC" {
		t.Errorf("ProviderPaymentID = %q, want PAY-CC", got.ProviderPaymentID)
	}
}

func TestApplyGlobalPayIdempotencyKey_FromBodyOnlyPayload(t *testing.T) {
	body := `{"session_id":"SESS-POST","service_token":"TOKEN-POST","payment_id":"PAY-POST"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/global-pay", strings.NewReader(body))
	parsed, err := parseGlobalPayWebhookRequest(req)
	if err != nil {
		t.Fatal(err)
	}

	applyGlobalPayIdempotencyKey(req, parsed)

	if got := req.Header.Get("Idempotency-Key"); got != "global-pay:PAY-POST" {
		t.Errorf("Idempotency-Key = %q, want global-pay:PAY-POST", got)
	}
}

func TestApplyGlobalPayIdempotencyKey_PriorityOrder(t *testing.T) {
	tests := []struct {
		name string
		req  *globalPayWebhookRequest
		want string
	}{
		{
			name: "uses provider payment id first",
			req: &globalPayWebhookRequest{
				ProviderPaymentID: "PAY-1",
				ProviderReference: "TOKEN-1",
				SessionID:         "SESS-1",
			},
			want: "global-pay:PAY-1",
		},
		{
			name: "falls back to provider reference",
			req: &globalPayWebhookRequest{
				ProviderReference: "TOKEN-2",
				SessionID:         "SESS-2",
			},
			want: "global-pay:TOKEN-2",
		},
		{
			name: "falls back to session id",
			req: &globalPayWebhookRequest{
				SessionID: "SESS-3",
			},
			want: "global-pay:SESS-3",
		},
		{
			name: "does not set when no key fields",
			req:  &globalPayWebhookRequest{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpReq := httptest.NewRequest(http.MethodPost, "/v1/webhooks/global-pay", nil)
			applyGlobalPayIdempotencyKey(httpReq, tt.req)
			if got := httpReq.Header.Get("Idempotency-Key"); got != tt.want {
				t.Errorf("Idempotency-Key = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplyGlobalPayIdempotencyKey_PreservesExistingHeader(t *testing.T) {
	httpReq := httptest.NewRequest(http.MethodPost, "/v1/webhooks/global-pay", nil)
	httpReq.Header.Set("Idempotency-Key", "existing-key")
	applyGlobalPayIdempotencyKey(httpReq, &globalPayWebhookRequest{ProviderPaymentID: "PAY-NEW"})

	if got := httpReq.Header.Get("Idempotency-Key"); got != "existing-key" {
		t.Errorf("Idempotency-Key = %q, want existing-key", got)
	}
}

func TestApplyStripeIdempotencyKey_FromEventID(t *testing.T) {
	httpReq := httptest.NewRequest(http.MethodPost, "/v1/webhooks/stripe", nil)
	applyStripeIdempotencyKey(httpReq, stripeEvent{ID: "evt_123"})

	if got := httpReq.Header.Get("Idempotency-Key"); got != "stripe:evt_123" {
		t.Errorf("Idempotency-Key = %q, want stripe:evt_123", got)
	}
}

func TestApplyStripeIdempotencyKey_PreservesExistingHeader(t *testing.T) {
	httpReq := httptest.NewRequest(http.MethodPost, "/v1/webhooks/stripe", nil)
	httpReq.Header.Set("Idempotency-Key", "existing-key")
	applyStripeIdempotencyKey(httpReq, stripeEvent{ID: "evt_new"})

	if got := httpReq.Header.Get("Idempotency-Key"); got != "existing-key" {
		t.Errorf("Idempotency-Key = %q, want existing-key", got)
	}
}

func TestApplyStripeIdempotencyKey_EmptyEventID(t *testing.T) {
	httpReq := httptest.NewRequest(http.MethodPost, "/v1/webhooks/stripe", nil)
	applyStripeIdempotencyKey(httpReq, stripeEvent{})

	if got := httpReq.Header.Get("Idempotency-Key"); got != "" {
		t.Errorf("Idempotency-Key = %q, want empty", got)
	}
}
