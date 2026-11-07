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

func TestClickExecutor_Unkeyed(t *testing.T) {
	t.Parallel()
	exec := newClickProviderExecutor("", "", "", "")
	_, err := exec.Execute(context.Background(), ExecutionRequest{
		Gateway: "CLICK",
		Action:  ExecutionActionCheckoutInit,
		OrderID: "ord-1",
	})
	var policy *GatewayPolicyError
	if !errors.As(err, &policy) || policy.Code != "no_live_keys" {
		t.Fatalf("err=%v", err)
	}
}

func TestClickExecutor_CheckoutInitHostedURL(t *testing.T) {
	t.Parallel()
	exec := newClickProviderExecutor("111", "222", "", "")
	got, err := exec.Execute(context.Background(), ExecutionRequest{
		Action:      ExecutionActionCheckoutInit,
		OrderID:     "ord-9",
		AmountMinor: 100000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.RedirectURL, "my.click.uz/services/pay") {
		t.Fatalf("url=%q", got.RedirectURL)
	}
	if !strings.Contains(got.RedirectURL, "amount=1000.00") {
		t.Fatalf("amount missing in %q", got.RedirectURL)
	}
}

func TestClickExecutor_MerchantEndpoints(t *testing.T) {
	t.Parallel()
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		if !strings.HasPrefix(r.Header.Get("Auth"), "user-1:") {
			t.Errorf("Auth=%q", r.Header.Get("Auth"))
		}
		body, _ := io.ReadAll(r.Body)
		_ = body
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/invoice/create"):
			_, _ = w.Write([]byte(`{"error_code":0,"invoice_id":77}`))
		case strings.Contains(r.URL.Path, "/invoice/status"):
			_, _ = w.Write([]byte(`{"error_code":0,"status":2}`))
		case strings.Contains(r.URL.Path, "/payment/status_by_mti"):
			_, _ = w.Write([]byte(`{"error_code":0,"payment_status":2}`))
		case strings.Contains(r.URL.Path, "/payment/status"):
			_, _ = w.Write([]byte(`{"error_code":0,"payment_status":2}`))
		case strings.Contains(r.URL.Path, "/payment/reversal"):
			_, _ = w.Write([]byte(`{"error_code":0}`))
		case strings.Contains(r.URL.Path, "/payment/partial_reversal"):
			_, _ = w.Write([]byte(`{"error_code":0}`))
		case strings.Contains(r.URL.Path, "/payment/ofd_data/submit_items"):
			_, _ = w.Write([]byte(`{"error_code":0}`))
		case strings.Contains(r.URL.Path, "/payment/ofd_data/submit_qrcode"):
			_, _ = w.Write([]byte(`{"error_code":0}`))
		case strings.Contains(r.URL.Path, "/payment/ofd_data"):
			_, _ = w.Write([]byte(`{"error_code":0,"qr_code_url":"https://ofd.example/q"}`))
		case strings.Contains(r.URL.Path, "/click_pass/payment"):
			_, _ = w.Write([]byte(`{"error_code":0,"payment_id":88}`))
		case strings.Contains(r.URL.Path, "/click_pass/confirm"):
			_, _ = w.Write([]byte(`{"error_code":0}`))
		case strings.Contains(r.URL.Path, "/click_pass/confirmation"):
			_, _ = w.Write([]byte(`{"error_code":0}`))
		case strings.Contains(r.URL.Path, "/card_token"):
			_, _ = w.Write([]byte(`{"error_code":0,"card_token":"tok-1"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	exec := newClickProviderExecutorWithOptions("111", "222", "user-1", "secret", srv.URL, nil)
	exec.nowUnix = func() int64 { return 1700000000 }

	inv, err := exec.executeCreateInvoice(context.Background(), ExecutionRequest{OrderID: "ord-9", AmountMinor: 100000}, "998901234567")
	if err != nil {
		t.Fatal(err)
	}
	if inv.ProviderRef != "77" {
		t.Fatalf("invoice ref=%q", inv.ProviderRef)
	}
	if _, err := exec.executeInvoiceStatus(context.Background(), "77"); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.Execute(context.Background(), ExecutionRequest{Action: ExecutionActionStatusCheck, SessionID: "pay-1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.executePaymentStatusByMTI(context.Background(), "ord-9", "2026-08-18"); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.Execute(context.Background(), ExecutionRequest{Action: ExecutionActionRefund, SessionID: "pay-1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.executeCardTokenRequest(context.Background(), "8600000000000000", "0327"); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.executeCardTokenVerify(context.Background(), "tok-1", "12345"); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.executeCardTokenPayment(context.Background(), "tok-1", 100000, "ord-9"); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.executeCardTokenDelete(context.Background(), "tok-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.executePartialReversal(context.Background(), "pay-1", 50000); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.executeOFDSubmitItems(context.Background(), "77", []clickOFDItem{
		{Name: "water", PriceMinor: 100000, Quantity: 1, VATPercent: 12},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.executeOFDSubmitQR(context.Background(), "77", "https://ofd.example/q"); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.executeOFDGet(context.Background(), "77"); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.executeClickPassPayment(context.Background(), ExecutionRequest{OrderID: "ord-9", AmountMinor: 100000}, "998901234567"); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.executeClickPassConfirm(context.Background(), "88"); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.executeClickPassEnableConfirmation(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.Execute(context.Background(), ExecutionRequest{
		Action:      ExecutionActionCheckoutCapture,
		OrderID:     "ord-9",
		AmountMinor: 100000,
		CardToken:   "tok-1",
	}); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(paths)
	if len(paths) < 16 {
		t.Fatalf("paths=%s", raw)
	}
}

func TestClickExecutor_OFDRefusesInventedQR(t *testing.T) {
	t.Parallel()
	exec := newClickProviderExecutor("111", "222", "user-1", "secret")
	if _, err := exec.executeOFDSubmitQR(context.Background(), "77", ""); err == nil {
		t.Fatal("must not invent qr_code_url")
	}
	if _, err := exec.executeOFDSubmitItems(context.Background(), "77", nil); err == nil {
		t.Fatal("must not invent ofd items")
	}
}
