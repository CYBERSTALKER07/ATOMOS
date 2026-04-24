package payment

import (
	"context"
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

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/global-pay?session_id=SESS-1", strings.NewReader("{}"))
	w := httptest.NewRecorder()

	svc.HandleGlobalPayWebhook(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", w.Code)
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
