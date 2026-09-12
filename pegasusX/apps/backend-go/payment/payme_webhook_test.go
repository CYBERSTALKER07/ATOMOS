package payment

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func paymeAuthHeader(secret string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte("Paycom:"+secret))
}

func postPayme(t *testing.T, svc *Service, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/payme", strings.NewReader(body))
	req.Header.Set("Authorization", paymeAuthHeader(svc.paymeWebhookSecret))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	svc.HandlePaymeWebhook(res, req)
	return res
}

func TestPaymeWebhook_RejectsInvalidAuth(t *testing.T) {
	t.Parallel()
	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)
	svc.paymeWebhookSecret = "payme-secret"

	body := `{"jsonrpc":"2.0","method":"CheckPerformTransaction","id":1,"params":{"amount":100,"account":{"order_id":"ord-2"}}}`
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

func TestPaymeMerchant_CheckCreatePerformCancelCheckStatement(t *testing.T) {
	t.Parallel()
	repo := &paymentRepoStub{
		created: SessionRecord{
			SessionID:   "sess-payme",
			OrderID:     "ord-1",
			AmountMinor: 50000,
			Status:      "PENDING",
		},
	}
	svc := newPaymentServiceForExecutionTest(repo)
	svc.paymeWebhookSecret = "payme-secret"

	res := postPayme(t, svc, `{"jsonrpc":"2.0","method":"CheckPerformTransaction","id":1,"params":{"amount":50000,"account":{"order_id":"ord-1"}}}`)
	if res.Code != http.StatusOK {
		t.Fatalf("checkperform status=%d body=%s", res.Code, res.Body.String())
	}
	var check map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &check); err != nil {
		t.Fatal(err)
	}
	result, _ := check["result"].(map[string]any)
	if result["allow"] != true {
		t.Fatalf("allow=%v", check)
	}

	res = postPayme(t, svc, `{"jsonrpc":"2.0","method":"CheckPerformTransaction","id":2,"params":{"amount":1,"account":{"order_id":"ord-1"}}}`)
	var amountErr map[string]any
	_ = json.Unmarshal(res.Body.Bytes(), &amountErr)
	rpcErr, _ := amountErr["error"].(map[string]any)
	if rpcErr["code"] != float64(paymeErrInvalidAmount) {
		t.Fatalf("amount error=%v", amountErr)
	}

	res = postPayme(t, svc, `{"jsonrpc":"2.0","method":"CreateTransaction","id":3,"params":{"id":"tx-payme-1","time":1399114284039,"amount":50000,"account":{"order_id":"ord-1"}}}`)
	if res.Code != http.StatusOK {
		t.Fatalf("create status=%d body=%s", res.Code, res.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(res.Body.Bytes(), &created)
	cr, _ := created["result"].(map[string]any)
	if cr["state"] != float64(paymeStateCreated) {
		t.Fatalf("create result=%v", created)
	}
	if repo.webhookCalls != 1 {
		t.Fatalf("webhookCalls after create=%d", repo.webhookCalls)
	}

	res = postPayme(t, svc, `{"jsonrpc":"2.0","method":"CreateTransaction","id":4,"params":{"id":"tx-payme-1","time":1399114284039,"amount":50000,"account":{"order_id":"ord-1"}}}`)
	_ = json.Unmarshal(res.Body.Bytes(), &created)
	if repo.webhookCalls != 1 {
		t.Fatalf("idempotent create persisted again: %d", repo.webhookCalls)
	}

	res = postPayme(t, svc, `{"jsonrpc":"2.0","method":"PerformTransaction","id":5,"params":{"id":"tx-payme-1"}}`)
	var performed map[string]any
	_ = json.Unmarshal(res.Body.Bytes(), &performed)
	pr, _ := performed["result"].(map[string]any)
	if pr["state"] != float64(paymeStatePerformed) {
		t.Fatalf("perform=%v body=%s", performed, res.Body.String())
	}
	if repo.webhookCalls != 2 {
		t.Fatalf("webhookCalls after perform=%d", repo.webhookCalls)
	}

	res = postPayme(t, svc, `{"jsonrpc":"2.0","method":"CheckTransaction","id":6,"params":{"id":"tx-payme-1"}}`)
	var checked map[string]any
	_ = json.Unmarshal(res.Body.Bytes(), &checked)
	ch, _ := checked["result"].(map[string]any)
	if ch["state"] != float64(paymeStatePerformed) {
		t.Fatalf("check=%v", checked)
	}

	res = postPayme(t, svc, `{"jsonrpc":"2.0","method":"SetFiscalData","id":61,"params":{"id":"tx-payme-1","type":"PERFORM","fiscal_data":{"receipt_id":121,"status_code":0,"fiscal_sign":"800031554082","qr_code_url":"https://ofd.example/q"}}}`)
	var fiscal map[string]any
	_ = json.Unmarshal(res.Body.Bytes(), &fiscal)
	fr, _ := fiscal["result"].(map[string]any)
	if fr["success"] != true {
		t.Fatalf("setfiscal=%v body=%s", fiscal, res.Body.String())
	}

	res = postPayme(t, svc, `{"jsonrpc":"2.0","method":"CancelTransaction","id":7,"params":{"id":"tx-payme-1","reason":5}}`)
	var cancelled map[string]any
	_ = json.Unmarshal(res.Body.Bytes(), &cancelled)
	cn, _ := cancelled["result"].(map[string]any)
	if cn["state"] != float64(paymeStateReverted) {
		t.Fatalf("cancel=%v body=%s", cancelled, res.Body.String())
	}

	res = postPayme(t, svc, `{"jsonrpc":"2.0","method":"GetStatement","id":8,"params":{"from":1,"to":9999999999999}}`)
	var stmt map[string]any
	_ = json.Unmarshal(res.Body.Bytes(), &stmt)
	st, _ := stmt["result"].(map[string]any)
	txs, _ := st["transactions"].([]any)
	if len(txs) != 1 {
		t.Fatalf("statement=%v", stmt)
	}

	res = postPayme(t, svc, `{"jsonrpc":"2.0","method":"UnknownMethod","id":9,"params":{}}`)
	var unknown map[string]any
	_ = json.Unmarshal(res.Body.Bytes(), &unknown)
	ue, _ := unknown["error"].(map[string]any)
	if ue["code"] != float64(paymeErrMethod) {
		t.Fatalf("unknown=%v", unknown)
	}
}

func TestPaymeMerchant_CheckPerformMissingOrder(t *testing.T) {
	t.Parallel()
	svc := newPaymentServiceForExecutionTest(&paymentRepoStub{})
	svc.paymeWebhookSecret = "payme-secret"
	res := postPayme(t, svc, `{"jsonrpc":"2.0","method":"CheckPerformTransaction","id":1,"params":{"amount":100,"account":{"order_id":"missing"}}}`)
	var payload map[string]any
	_ = json.Unmarshal(res.Body.Bytes(), &payload)
	rpcErr, _ := payload["error"].(map[string]any)
	if rpcErr["code"] != float64(paymeErrAccount) {
		t.Fatalf("payload=%v", payload)
	}
}

func TestPaymeMerchant_SetFiscalMissingTx(t *testing.T) {
	t.Parallel()
	svc := newPaymentServiceForExecutionTest(&paymentRepoStub{})
	svc.paymeWebhookSecret = "payme-secret"
	res := postPayme(t, svc, `{"jsonrpc":"2.0","method":"SetFiscalData","id":1,"params":{"id":"missing","type":"PERFORM","fiscal_data":{"receipt_id":1}}}`)
	var payload map[string]any
	_ = json.Unmarshal(res.Body.Bytes(), &payload)
	rpcErr, _ := payload["error"].(map[string]any)
	if rpcErr["code"] != float64(paymeErrFiscalNotFound) {
		t.Fatalf("payload=%v", payload)
	}
}

func TestPaymeWebhook_AcceptsValidPerform(t *testing.T) {
	t.Parallel()
	repo := &paymentRepoStub{
		created: SessionRecord{
			SessionID:   "sess-1",
			OrderID:     "ord-1",
			AmountMinor: 50000,
		},
	}
	svc := newPaymentServiceForExecutionTest(repo)
	svc.paymeWebhookSecret = "payme-secret"

	create := postPayme(t, svc, `{"jsonrpc":"2.0","method":"CreateTransaction","id":1,"params":{"id":"tx-payme-1","amount":50000,"account":{"order_id":"ord-1"}}}`)
	if create.Code != http.StatusOK {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	res := postPayme(t, svc, `{"jsonrpc":"2.0","method":"PerformTransaction","id":2,"params":{"id":"tx-payme-1"}}`)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d body %s", res.Code, res.Body.String())
	}
	if repo.webhookCalls != 2 {
		t.Fatalf("webhookCalls = %d want 2", repo.webhookCalls)
	}
	var payload map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := payload["result"]; !ok {
		t.Fatalf("missing result: %v", payload)
	}
}
