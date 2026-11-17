package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func uzumSignHex(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func uzumSignBase64(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestUzumWebhook_AcceptsValidHMACAndPersistsIdempotently(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{
		created: SessionRecord{
			SessionID:   "sess-uzum-1",
			OrderID:     "ord-uzum-1",
			SupplierID:  "sup-uzum-1",
			RetailerID:  "ret-uzum-1",
			AmountMinor: 1500000,
			Currency:    "UZS",
		},
	}
	svc := newPaymentServiceForExecutionTest(repo)
	secret := "uzum-prod-secret"
	svc.SetUzumWebhookSecret(secret)

	body := []byte(`{"transId":"uz-tx-100","orderId":"ord-uzum-1","amount":1500000,"currency":"UZS","status":"CONFIRMED"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/uzum", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Uzum-Signature", "sha256="+uzumSignHex(body, secret))
	res := httptest.NewRecorder()
	svc.HandleUzumWebhook(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d want 200 body=%s", res.Code, res.Body.String())
	}
	if repo.webhookCalls != 1 {
		t.Fatalf("webhookCalls = %d want 1", repo.webhookCalls)
	}
	var resp map[string]string
	if err := json.Unmarshal(res.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["gateway"] != "UZUM" || resp["webhook_status"] != "PAID" || resp["transaction_id"] != "uz-tx-100" {
		t.Fatalf("unexpected response: %+v", resp)
	}

	// Idempotent replay with identical payload must replay without duplicate SaveWebhook.
	reqReplay := httptest.NewRequest(http.MethodPost, "/v1/webhooks/uzum", strings.NewReader(string(body)))
	reqReplay.Header.Set("X-Signature", uzumSignBase64(body, secret))
	resReplay := httptest.NewRecorder()
	svc.HandleUzumWebhook(resReplay, reqReplay)
	if resReplay.Code != http.StatusOK {
		t.Fatalf("replay status = %d want 200 body=%s", resReplay.Code, resReplay.Body.String())
	}
	if repo.webhookCalls != 1 {
		t.Fatalf("replay incremented webhookCalls to %d want 1", repo.webhookCalls)
	}

	// Conflicting payload under same transId + status must return 409 Conflict.
	conflictBody := []byte(`{"transId":"uz-tx-100","orderId":"ord-uzum-1","amount":1500000,"currency":"UZS","status":"PAID"}`)
	reqConflict := httptest.NewRequest(http.MethodPost, "/v1/webhooks/uzum", strings.NewReader(string(conflictBody)))
	reqConflict.Header.Set("X-Uzum-Signature", uzumSignHex(conflictBody, secret))
	resConflict := httptest.NewRecorder()
	svc.HandleUzumWebhook(resConflict, reqConflict)
	if resConflict.Code != http.StatusConflict {
		t.Fatalf("conflict status = %d want 409 body=%s", resConflict.Code, resConflict.Body.String())
	}
}

func TestUzumWebhook_RejectsInvalidAndMissingSignatureFailClosed(t *testing.T) {
	t.Setenv("UZUM_WEBHOOK_SECRET", "")

	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)

	body := []byte(`{"transId":"uz-tx-1","orderId":"ord-1","amount":100000,"status":"CONFIRMED"}`)
	reqNoSecret := httptest.NewRequest(http.MethodPost, "/v1/webhooks/uzum", strings.NewReader(string(body)))
	reqNoSecret.Header.Set("X-Uzum-Signature", uzumSignHex(body, "guessed"))
	resNoSecret := httptest.NewRecorder()
	svc.HandleUzumWebhook(resNoSecret, reqNoSecret)
	if resNoSecret.Code != http.StatusUnauthorized {
		t.Fatalf("unconfigured secret status = %d want 401", resNoSecret.Code)
	}

	svc.SetUzumWebhookSecret("real-secret")
	reqBadSig := httptest.NewRequest(http.MethodPost, "/v1/webhooks/uzum", strings.NewReader(string(body)))
	reqBadSig.Header.Set("X-Uzum-Signature", uzumSignHex(body, "wrong-secret"))
	resBadSig := httptest.NewRecorder()
	svc.HandleUzumWebhook(resBadSig, reqBadSig)
	if resBadSig.Code != http.StatusUnauthorized {
		t.Fatalf("bad signature status = %d want 401", resBadSig.Code)
	}
	if repo.webhookCalls != 0 {
		t.Fatalf("webhookCalls = %d want 0", repo.webhookCalls)
	}
}

func TestUzumWebhook_ProductionRejectsDevSecretAndReadsEnvFallback(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	t.Setenv("UZUM_WEBHOOK_SECRET", "dev-local-uzum")

	svc := NewService(ServiceConfig{Repo: &paymentRepoStub{}})
	body := []byte(`{"transId":"uz-tx-2","orderId":"ord-2","amount":100000,"status":"CONFIRMED"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/uzum", strings.NewReader(string(body)))
	req.Header.Set("X-Uzum-Signature", uzumSignHex(body, "dev-local-uzum"))
	res := httptest.NewRecorder()
	svc.HandleUzumWebhook(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("production with dev-* secret must fail closed 401, got %d", res.Code)
	}

	t.Setenv("UZUM_WEBHOOK_SECRET", "prod-live-uzum-secret-xyz")
	reqOK := httptest.NewRequest(http.MethodPost, "/v1/webhooks/uzum", strings.NewReader(string(body)))
	reqOK.Header.Set("X-Uzum-Signature", uzumSignHex(body, "prod-live-uzum-secret-xyz"))
	resOK := httptest.NewRecorder()
	svc.HandleUzumWebhook(resOK, reqOK)
	if resOK.Code != http.StatusOK {
		t.Fatalf("production with valid env secret status = %d want 200 body=%s", resOK.Code, resOK.Body.String())
	}
}

func TestUzumWebhook_ValidationAmountAndCurrencyMismatch(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{
		created: SessionRecord{
			SessionID:   "sess-uzum-2",
			OrderID:     "ord-uzum-2",
			AmountMinor: 500000,
			Currency:    "UZS",
		},
	}
	svc := newPaymentServiceForExecutionTest(repo)
	secret := "uzum-val-secret"
	svc.SetUzumWebhookSecret(secret)

	send := func(method string, payload string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/v1/webhooks/uzum", strings.NewReader(payload))
		req.Header.Set("X-Uzum-Signature", uzumSignHex([]byte(payload), secret))
		res := httptest.NewRecorder()
		svc.HandleUzumWebhook(res, req)
		return res
	}

	if r := send(http.MethodGet, `{}`); r.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status = %d want 405", r.Code)
	}
	if r := send(http.MethodPost, `{bad json`); r.Code != http.StatusBadRequest {
		t.Fatalf("bad JSON status = %d want 400", r.Code)
	}
	if r := send(http.MethodPost, `{"transId":"uz-1","orderId":"ord-uzum-2","amount":0,"status":"CONFIRMED"}`); r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("zero amount status = %d want 422", r.Code)
	}
	if r := send(http.MethodPost, `{"transId":"uz-1","orderId":"ord-uzum-2","amount":500000,"status":"WEIRD"}`); r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("unknown status code = %d want 422", r.Code)
	}
	if r := send(http.MethodPost, `{"transId":"uz-1","orderId":"ord-uzum-2","amount":499900,"status":"CONFIRMED"}`); r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("amount mismatch status = %d want 422", r.Code)
	}
	if r := send(http.MethodPost, `{"transId":"uz-1","orderId":"ord-uzum-2","amount":500000,"currency":"USD","status":"CONFIRMED"}`); r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("currency mismatch status = %d want 422", r.Code)
	}
}
