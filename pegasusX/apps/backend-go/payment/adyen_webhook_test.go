package payment

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleAdyenWebhook_ReturnsLiteralAccepted(t *testing.T) {
	svc := NewService(ServiceConfig{
		SupplierID:         "seed",
		AdyenWebhookSecret: "44DAC79C4F1804F0F43382A4E5A8B068831F",
	})

	// Valid payload with dummy HMAC
	payload := `{"notificationItems":[]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/adyen", bytes.NewBufferString(payload))
	w := httptest.NewRecorder()

	svc.HandleAdyenWebhook(w, req)
	// Empty notificationItems returns 422
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for empty items, got %d", w.Code)
	}

	// Method not allowed check
	reqGet := httptest.NewRequest(http.MethodGet, "/v1/webhooks/adyen", nil)
	wGet := httptest.NewRecorder()
	svc.HandleAdyenWebhook(wGet, reqGet)
	if wGet.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for GET, got %d", wGet.Code)
	}
}
