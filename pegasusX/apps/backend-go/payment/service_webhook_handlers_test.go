package payment

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleGlobalPayWebhook_ReplayNoDuplicatePersist(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)

	body := `{"session_id":"sess-1","transaction_id":"tx-1","order_id":"order-1","status":"PAID","amount_minor":1200,"currency":"UZS"}`
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte("Paycom:gp-secret"))

	req1 := httptest.NewRequest(http.MethodPost, "/v1/webhooks/global-pay", strings.NewReader(body))
	req1.Header.Set("Authorization", authHeader)
	res1 := httptest.NewRecorder()
	svc.HandleGlobalPayWebhook(res1, req1)

	if res1.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d", res1.Code, http.StatusOK)
	}
	if repo.webhookCalls != 1 {
		t.Fatalf("webhookCalls after first request = %d, want 1", repo.webhookCalls)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/v1/webhooks/global-pay", strings.NewReader(body))
	req2.Header.Set("Authorization", authHeader)
	res2 := httptest.NewRecorder()
	svc.HandleGlobalPayWebhook(res2, req2)

	if res2.Code != http.StatusOK {
		t.Fatalf("replay status = %d, want %d", res2.Code, http.StatusOK)
	}
	if repo.webhookCalls != 1 {
		t.Fatalf("webhookCalls after replay = %d, want 1", repo.webhookCalls)
	}

	var payload map[string]any
	if err := json.Unmarshal(res2.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode replay response: %v", err)
	}
	if payload["status"] != "accepted" {
		t.Fatalf("status = %v, want accepted", payload["status"])
	}
}
