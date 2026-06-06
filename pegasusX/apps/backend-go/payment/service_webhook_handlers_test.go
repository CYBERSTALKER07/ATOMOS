package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func TestHandleStripeWebhook_IdempotencyConflict(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)
	timestamp := int64(1_700_000_000)

	body1 := `{"id":"evt_1","type":"payment_intent.succeeded","data":{"object":{"id":"pi_1","status":"succeeded","amount":500,"currency":"usd","metadata":{"session_id":"sess-2","order_id":"order-2"}}}}`
	req1 := httptest.NewRequest(http.MethodPost, "/v1/webhooks/stripe", strings.NewReader(body1))
	req1.Header.Set("Stripe-Signature", stripeSignatureHeaderForTest([]byte(body1), "stripe-secret", timestamp))
	res1 := httptest.NewRecorder()
	svc.HandleStripeWebhook(res1, req1)

	if res1.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d", res1.Code, http.StatusOK)
	}
	if repo.webhookCalls != 1 {
		t.Fatalf("webhookCalls after first request = %d, want 1", repo.webhookCalls)
	}

	body2 := `{"id":"evt_1","type":"payment_intent.succeeded","data":{"object":{"id":"pi_1","status":"succeeded","amount":700,"currency":"usd","metadata":{"session_id":"sess-2","order_id":"order-2"}}}}`
	req2 := httptest.NewRequest(http.MethodPost, "/v1/webhooks/stripe", strings.NewReader(body2))
	req2.Header.Set("Stripe-Signature", stripeSignatureHeaderForTest([]byte(body2), "stripe-secret", timestamp))
	res2 := httptest.NewRecorder()
	svc.HandleStripeWebhook(res2, req2)

	if res2.Code != http.StatusConflict {
		t.Fatalf("conflict status = %d, want %d", res2.Code, http.StatusConflict)
	}
	if repo.webhookCalls != 1 {
		t.Fatalf("webhookCalls after conflict = %d, want 1", repo.webhookCalls)
	}
}

func TestHandleAdyenWebhook_ReplayReturnsSingleValidJSON(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)

	body := signedAdyenWebhookBodyForTest("adyen-secret", "psp_abc")

	req1 := httptest.NewRequest(http.MethodPost, "/v1/webhooks/adyen", strings.NewReader(body))
	res1 := httptest.NewRecorder()
	svc.HandleAdyenWebhook(res1, req1)

	if res1.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d", res1.Code, http.StatusOK)
	}
	if repo.webhookCalls != 1 {
		t.Fatalf("webhookCalls after first request = %d, want 1", repo.webhookCalls)
	}

	var firstPayload map[string]any
	if err := json.Unmarshal(res1.Body.Bytes(), &firstPayload); err != nil {
		t.Fatalf("decode first response: %v", err)
	}
	if firstPayload["processed_items"] != float64(1) {
		t.Fatalf("processed_items first = %v, want 1", firstPayload["processed_items"])
	}

	req2 := httptest.NewRequest(http.MethodPost, "/v1/webhooks/adyen", strings.NewReader(body))
	res2 := httptest.NewRecorder()
	svc.HandleAdyenWebhook(res2, req2)

	if res2.Code != http.StatusOK {
		t.Fatalf("replay status = %d, want %d", res2.Code, http.StatusOK)
	}
	if repo.webhookCalls != 1 {
		t.Fatalf("webhookCalls after replay = %d, want 1", repo.webhookCalls)
	}

	var replayPayload map[string]any
	if err := json.Unmarshal(res2.Body.Bytes(), &replayPayload); err != nil {
		t.Fatalf("decode replay response: %v", err)
	}
	if replayPayload["processed_items"] != float64(0) {
		t.Fatalf("processed_items replay = %v, want 0", replayPayload["processed_items"])
	}
}

func TestHandleAdyenWebhook_InvalidSignatureRejected(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)

	body := signedAdyenWebhookBodyForTest("wrong-secret", "psp_bad")
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/adyen", strings.NewReader(body))
	res := httptest.NewRecorder()
	svc.HandleAdyenWebhook(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusUnauthorized)
	}
	if repo.webhookCalls != 0 {
		t.Fatalf("webhookCalls = %d, want 0", repo.webhookCalls)
	}
}

func stripeSignatureHeaderForTest(body []byte, secret string, timestamp int64) string {
	ts := strconv.FormatInt(timestamp, 10)
	signedPayload := ts + "." + string(body)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signedPayload))
	sig := hex.EncodeToString(mac.Sum(nil))
	return "t=" + ts + ",v1=" + sig
}

func signedAdyenWebhookBodyForTest(secret, pspReference string) string {
	item := adyenNotificationItem{
		PspReference:        pspReference,
		OriginalReference:   "orig_1",
		MerchantReference:   "order_1",
		MerchantAccountCode: "merchant_1",
		EventCode:           "AUTHORISATION",
		Success:             "true",
		Amount: adyenAmount{
			Currency: "UZS",
			Value:    2300,
		},
		AdditionalData: map[string]string{
			"session_id": "sess_adyen_1",
		},
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(adyenSigningData(item)))
	item.AdditionalData["hmacSignature"] = base64.StdEncoding.EncodeToString(mac.Sum(nil))

	payload := map[string]any{
		"notificationItems": []map[string]any{{
			"NotificationRequestItem": item,
		}},
	}
	encoded, _ := json.Marshal(payload)
	return string(encoded)
}
