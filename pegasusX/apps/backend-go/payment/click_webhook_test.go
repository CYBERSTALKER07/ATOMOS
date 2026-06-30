package payment

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClickWebhook_AcceptsSignedPrepare(t *testing.T) {
	t.Parallel()
	repo := &paymentRepoStub{
		created: SessionRecord{
			OrderID:     "ord-1",
			AmountMinor: 1000,
		},
	}
	svc := newPaymentServiceForExecutionTest(repo)
	secret := "click-secret"
	svc.clickWebhookSecret = secret

	reqBody := clickWebhookRequest{
		ClickTransID:    "clk-1",
		MerchantTransID: "ord-1",
		ServiceID:       "svc-1",
		Amount:          "1000",
		Action:          "1",
		SignTime:        "2026-06-28 12:00:00",
	}
	reqBody.SignString = clickSignString(reqBody, secret)
	raw, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/click", strings.NewReader(string(raw)))
	res := httptest.NewRecorder()
	svc.HandleClickWebhook(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d body %s", res.Code, res.Body.String())
	}
	if repo.webhookCalls != 1 {
		t.Fatalf("webhookCalls = %d want 1", repo.webhookCalls)
	}
}

func clickSignString(req clickWebhookRequest, secret string) string {
	payload := strings.TrimSpace(req.ClickTransID) +
		strings.TrimSpace(req.ServiceID) +
		secret +
		strings.TrimSpace(req.MerchantTransID) +
		strings.TrimSpace(req.Amount) +
		strings.TrimSpace(req.Action) +
		strings.TrimSpace(req.SignTime)
	sum := md5.Sum([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func TestClickWebhook_RejectsBadSignature(t *testing.T) {
	t.Parallel()
	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)
	svc.clickWebhookSecret = "click-secret"

	raw := `{"click_trans_id":"clk-2","merchant_trans_id":"ord-2","amount":"100","sign_string":"bad"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/click", strings.NewReader(raw))
	res := httptest.NewRecorder()
	svc.HandleClickWebhook(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d want 401", res.Code)
	}
}

func TestClickWebhook_RejectsMissingSession(t *testing.T) {
	t.Parallel()
	repo := &paymentRepoStub{} // no created session
	svc := newPaymentServiceForExecutionTest(repo)
	secret := "click-secret"
	svc.clickWebhookSecret = secret

	reqBody := clickWebhookRequest{
		ClickTransID:    "clk-missing",
		MerchantTransID: "ord-missing",
		Amount:          "1000",
		Action:          "1",
		SignTime:        "2026-06-28 12:00:00",
	}
	reqBody.SignString = clickSignString(reqBody, secret)
	raw, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/click", strings.NewReader(string(raw)))
	res := httptest.NewRecorder()
	svc.HandleClickWebhook(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d want 200", res.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp["error"] != float64(-5) {
		t.Fatalf("error = %v want -5", resp["error"])
	}
}

func TestClickWebhook_RejectsAmountMismatch(t *testing.T) {
	t.Parallel()
	repo := &paymentRepoStub{
		created: SessionRecord{
			OrderID:     "ord-mismatch",
			AmountMinor: 1000,
		},
	}
	svc := newPaymentServiceForExecutionTest(repo)
	secret := "click-secret"
	svc.clickWebhookSecret = secret

	reqBody := clickWebhookRequest{
		ClickTransID:    "clk-mismatch",
		MerchantTransID: "ord-mismatch",
		Amount:          "500", // mismatched amount
		Action:          "1",
		SignTime:        "2026-06-28 12:00:00",
	}
	reqBody.SignString = clickSignString(reqBody, secret)
	raw, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/click", strings.NewReader(string(raw)))
	res := httptest.NewRecorder()
	svc.HandleClickWebhook(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d want 200", res.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp["error"] != float64(-2) {
		t.Fatalf("error = %v want -2", resp["error"])
	}
}
