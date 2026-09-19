package payment

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func clickSigned(req clickWebhookRequest, secret string) clickWebhookRequest {
	req.SignString = clickShopSign(req, secret)
	return req
}

func postClickJSON(t *testing.T, svc *Service, reqBody clickWebhookRequest) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(reqBody)
	httpReq := httptest.NewRequest(http.MethodPost, "/v1/webhooks/click", strings.NewReader(string(raw)))
	httpReq.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	svc.HandleClickWebhook(res, httpReq)
	return res
}

func TestClickWebhook_PrepareDoesNotMarkPaid(t *testing.T) {
	t.Parallel()
	repo := &paymentRepoStub{
		created: SessionRecord{
			SessionID:   "sess-1",
			OrderID:     "ord-1",
			AmountMinor: 100000,
		},
	}
	svc := newPaymentServiceForExecutionTest(repo)
	secret := "click-secret"
	svc.clickWebhookSecret = secret

	reqBody := clickSigned(clickWebhookRequest{
		ClickTransID:    "clk-1",
		MerchantTransID: "ord-1",
		ServiceID:       "svc-1",
		Amount:          "1000.00",
		Action:          clickActionPrepare,
		SignTime:        "2026-06-28 12:00:00",
	}, secret)
	res := postClickJSON(t, svc, reqBody)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d body %s", res.Code, res.Body.String())
	}
	if repo.webhookCalls != 1 {
		t.Fatalf("webhookCalls = %d want 1", repo.webhookCalls)
	}
	var resp map[string]any
	_ = json.Unmarshal(res.Body.Bytes(), &resp)
	if resp["error"] != float64(0) {
		t.Fatalf("error=%v body=%s", resp["error"], res.Body.String())
	}
	if resp["merchant_prepare_id"] != "sess-1" {
		t.Fatalf("prepare_id=%v", resp["merchant_prepare_id"])
	}
}

func TestClickWebhook_AcceptsSignedComplete(t *testing.T) {
	t.Parallel()
	repo := &paymentRepoStub{
		created: SessionRecord{
			SessionID:   "sess-1",
			OrderID:     "ord-1",
			AmountMinor: 100000,
		},
	}
	svc := newPaymentServiceForExecutionTest(repo)
	secret := "click-secret"
	svc.clickWebhookSecret = secret

	reqBody := clickSigned(clickWebhookRequest{
		ClickTransID:      "clk-1",
		MerchantTransID:   "ord-1",
		MerchantPrepareID: "sess-1",
		ServiceID:         "svc-1",
		Amount:            "1000.00",
		Action:            clickActionComplete,
		SignTime:          "2026-06-28 12:00:00",
	}, secret)
	res := postClickJSON(t, svc, reqBody)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d body %s", res.Code, res.Body.String())
	}
	if repo.webhookCalls != 1 {
		t.Fatalf("webhookCalls = %d want 1", repo.webhookCalls)
	}
}

func TestClickWebhook_AcceptsSignedPrepare(t *testing.T) {
	TestClickWebhook_AcceptsSignedComplete(t)
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
	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)
	secret := "click-secret"
	svc.clickWebhookSecret = secret

	reqBody := clickSigned(clickWebhookRequest{
		ClickTransID:      "clk-missing",
		MerchantTransID:   "ord-missing",
		MerchantPrepareID: "sess-x",
		Amount:            "1000.00",
		Action:            clickActionComplete,
		SignTime:          "2026-06-28 12:00:00",
	}, secret)
	res := postClickJSON(t, svc, reqBody)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d want 200", res.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp["error"] != float64(clickErrUserMissing) {
		t.Fatalf("error = %v want %d", resp["error"], clickErrUserMissing)
	}
}

func TestClickWebhook_RejectsAmountMismatch(t *testing.T) {
	t.Parallel()
	repo := &paymentRepoStub{
		created: SessionRecord{
			SessionID:   "sess-m",
			OrderID:     "ord-mismatch",
			AmountMinor: 100000,
		},
	}
	svc := newPaymentServiceForExecutionTest(repo)
	secret := "click-secret"
	svc.clickWebhookSecret = secret

	reqBody := clickSigned(clickWebhookRequest{
		ClickTransID:      "clk-mismatch",
		MerchantTransID:   "ord-mismatch",
		MerchantPrepareID: "sess-m",
		Amount:            "500.00",
		Action:            clickActionComplete,
		SignTime:          "2026-06-28 12:00:00",
	}, secret)
	res := postClickJSON(t, svc, reqBody)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d want 200", res.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp["error"] != float64(clickErrAmount) {
		t.Fatalf("error = %v want %d", resp["error"], clickErrAmount)
	}
}

func TestClickWebhook_FormURLEncodedPrepare(t *testing.T) {
	t.Parallel()
	repo := &paymentRepoStub{
		created: SessionRecord{
			SessionID:   "sess-form",
			OrderID:     "ord-form",
			AmountMinor: 2500,
		},
	}
	svc := newPaymentServiceForExecutionTest(repo)
	secret := "click-secret"
	svc.clickWebhookSecret = secret
	signed := clickSigned(clickWebhookRequest{
		ClickTransID:    "9",
		MerchantTransID: "ord-form",
		ServiceID:       "3",
		Amount:          "25.00",
		Action:          clickActionPrepare,
		SignTime:        "2026-06-28 12:00:00",
	}, secret)
	form := url.Values{}
	form.Set("click_trans_id", signed.ClickTransID)
	form.Set("merchant_trans_id", signed.MerchantTransID)
	form.Set("service_id", signed.ServiceID)
	form.Set("amount", signed.Amount)
	form.Set("action", signed.Action)
	form.Set("sign_time", signed.SignTime)
	form.Set("sign_string", signed.SignString)
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/click", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	svc.HandleClickWebhook(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(res.Body.Bytes(), &resp)
	if resp["error"] != float64(0) {
		t.Fatalf("resp=%v", resp)
	}
}
