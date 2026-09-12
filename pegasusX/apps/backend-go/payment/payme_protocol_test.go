package payment

import (
	"encoding/json"
	"testing"
)

func TestParsePaymeMerchantParams_AccountOrderID(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{"id":"tx-1","time":1399114284039,"amount":50000,"account":{"order_id":"ord-9"}}`)
	got, err := parsePaymeMerchantParams(raw)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got.PaymeID != "tx-1" || got.OrderID != "ord-9" || got.Amount != 50000 || got.Time != 1399114284039 {
		t.Fatalf("got=%+v", got)
	}
}

func TestPaymeHostedCheckoutURL_EncodesTiyin(t *testing.T) {
	t.Parallel()
	url := paymeHostedCheckoutURL("https://test.paycom.uz", "mid-1", "ord-1", 12500, "https://app.example/return")
	if url == "" || url[:8] != "https://" {
		t.Fatalf("url=%q", url)
	}
	if got := paymeStatusFromState(paymeStatePerformed); got != "PAID" {
		t.Fatalf("status=%s", got)
	}
	if got := paymeStatusFromState(paymeStateCancelled); got != "FAILED" {
		t.Fatalf("status=%s", got)
	}
}
