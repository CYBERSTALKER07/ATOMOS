package payment

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPaymeWebhook_AcceptsValidPerform(t *testing.T) {
	t.Parallel()
	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)
	svc.paymeWebhookSecret = "payme-secret"

	body := `{"jsonrpc":"2.0","method":"PerformTransaction","id":1,"params":{"id":"tx-payme-1","order_id":"ord-1","amount":50000,"state":2}}`
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte("Paycom:payme-secret"))

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/payme", strings.NewReader(body))
	req.Header.Set("Authorization", auth)
	res := httptest.NewRecorder()
	svc.HandlePaymeWebhook(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d body %s", res.Code, res.Body.String())
	}
	if repo.webhookCalls != 1 {
		t.Fatalf("webhookCalls = %d want 1", repo.webhookCalls)
	}
	var payload map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := payload["result"]; !ok {
		t.Fatalf("missing result: %v", payload)
	}
}

func TestPaymeWebhook_RejectsInvalidAuth(t *testing.T) {
	t.Parallel()
	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)
	svc.paymeWebhookSecret = "payme-secret"

	body := `{"jsonrpc":"2.0","method":"PerformTransaction","id":1,"params":{"id":"tx-payme-2","order_id":"ord-2","amount":100,"state":2}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/payme", strings.NewReader(body))
	req.Header.Set("Authorization", "Basic invalid")
	res := httptest.NewRecorder()
	svc.HandlePaymeWebhook(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d want 401", res.Code)
	}
	if repo.webhookCalls != 0 {
		t.Fatalf("webhookCalls = %d want 0", repo.webhookCalls)
	}
}
