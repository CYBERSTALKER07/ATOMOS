package payment

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPaymeExecutor_Unkeyed(t *testing.T) {
	t.Parallel()
	exec := newPaymeProviderExecutor("", "", "dev")
	_, err := exec.Execute(context.Background(), ExecutionRequest{
		Gateway:     "PAYME",
		Action:      ExecutionActionCheckoutInit,
		OrderID:     "ord-1",
		AmountMinor: 1000,
	})
	var policy *GatewayPolicyError
	if !errors.As(err, &policy) || policy.Code != "no_live_keys" {
		t.Fatalf("err=%v", err)
	}
	if !errors.Is(err, errPaymeCredentialsMissing) {
		t.Fatalf("unwrap: %v", err)
	}
}

func TestPaymeExecutor_CheckoutInitHostedURL(t *testing.T) {
	t.Parallel()
	exec := newPaymeProviderExecutor("merchant-1", "", "dev")
	got, err := exec.Execute(context.Background(), ExecutionRequest{
		Gateway:     "PAYME",
		Action:      ExecutionActionCheckoutInit,
		OrderID:     "ord-9",
		AmountMinor: 12500,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.RedirectURL == "" || !strings.Contains(got.RedirectURL, "test.paycom.uz") {
		t.Fatalf("redirect=%q", got.RedirectURL)
	}
	if got.Mode != ExecutionModeHostedRedirect {
		t.Fatalf("mode=%s", got.Mode)
	}
}

func TestPaymeExecutor_ReceiptsCreateAndCheck(t *testing.T) {
	t.Parallel()
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth") != "merchant-1:key-1" {
			t.Errorf("X-Auth=%q", r.Header.Get("X-Auth"))
		}
		body, _ := io.ReadAll(r.Body)
		var rpc map[string]any
		_ = json.Unmarshal(body, &rpc)
		method, _ := rpc["method"].(string)
		seen = append(seen, method)
		switch method {
		case "receipts.create":
			_, _ = w.Write([]byte(`{"result":{"receipt":{"_id":"rcp-1"}}}`))
		case "receipts.check":
			_, _ = w.Write([]byte(`{"result":{"state":4}}`))
		case "receipts.cancel":
			_, _ = w.Write([]byte(`{"result":{"receipt":{"_id":"rcp-1"}}}`))
		case "receipts.send", "receipts.get", "receipts.get_all":
			_, _ = w.Write([]byte(`{"result":{}}`))
		case "receipts.pay", "receipts.set_fiscal_data":
			_, _ = w.Write([]byte(`{"result":{"receipt":{"_id":"rcp-1","state":4}}}`))
		case "cards.create", "cards.get_verify_code", "cards.verify", "cards.check", "cards.remove":
			_, _ = w.Write([]byte(`{"result":{"card":{"token":"tok-1","verify":true}}}`))
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)

	exec := newPaymeProviderExecutorWithOptions("merchant-1", "key-1", "dev", srv.URL, nil)
	got, err := exec.Execute(context.Background(), ExecutionRequest{
		Action:      ExecutionActionCheckoutInit,
		OrderID:     "ord-9",
		AmountMinor: 50000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ProviderRef != "rcp-1" {
		t.Fatalf("ref=%q", got.ProviderRef)
	}
	_, err = exec.Execute(context.Background(), ExecutionRequest{Action: ExecutionActionStatusCheck, SessionID: "rcp-1"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = exec.Execute(context.Background(), ExecutionRequest{Action: ExecutionActionRefund, SessionID: "rcp-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := exec.receiptsSend(context.Background(), "rcp-1", "998901234567"); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.receiptsGet(context.Background(), "rcp-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.receiptsGetAll(context.Background(), 1, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.cardsCreate(context.Background(), "8600060450896291", "0399", false); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.cardsGetVerifyCode(context.Background(), "tok-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.cardsVerify(context.Background(), "tok-1", "666666"); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.cardsCheck(context.Background(), "tok-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.Execute(context.Background(), ExecutionRequest{
		Action:    ExecutionActionCheckoutCapture,
		SessionID: "rcp-1",
		CardToken: "tok-1",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.receiptsSetFiscalData(context.Background(), "rcp-1", map[string]any{
		"receipt_id": 121, "status_code": 0, "fiscal_sign": "800031554082",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.cardsRemove(context.Background(), "tok-1"); err != nil {
		t.Fatal(err)
	}
	if len(seen) < 12 {
		t.Fatalf("seen=%v", seen)
	}
}

func TestPaymeExecutor_CaptureWithoutToken(t *testing.T) {
	t.Parallel()
	exec := newPaymeProviderExecutor("merchant-1", "key-1", "dev")
	_, err := exec.Execute(context.Background(), ExecutionRequest{
		Action:    ExecutionActionCheckoutCapture,
		SessionID: "rcp-1",
	})
	var policy *GatewayPolicyError
	if !errors.As(err, &policy) || policy.Code != "payment_gateway_policy_violation" {
		t.Fatalf("err=%v", err)
	}
}

func TestPaymeExecutor_SetFiscalRefusesEmpty(t *testing.T) {
	t.Parallel()
	exec := newPaymeProviderExecutor("merchant-1", "key-1", "dev")
	if _, err := exec.receiptsSetFiscalData(context.Background(), "rcp-1", nil); err == nil {
		t.Fatal("must not invent fiscal_data")
	}
}
