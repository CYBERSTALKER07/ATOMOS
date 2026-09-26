package payment

import (
	"testing"
)

func TestParseDecimalToMinor_NoFloat(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in      string
		dec     int
		want    int64
		wantErr bool
	}{
		{in: "1000", dec: 2, want: 100000},
		{in: "1000.00", dec: 2, want: 100000},
		{in: "10.50", dec: 2, want: 1050},
		{in: "0.01", dec: 2, want: 1},
		{in: "1e3", dec: 2, wantErr: true},
		{in: "-1", dec: 2, wantErr: true},
		{in: "1.234", dec: 2, wantErr: true},
	}
	for _, tc := range cases {
		got, err := parseDecimalToMinor(tc.in, tc.dec)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("%q: want error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("%q: got %d want %d", tc.in, got, tc.want)
		}
	}
}

func TestFormatMinorAsDecimal(t *testing.T) {
	t.Parallel()
	if got := formatMinorAsDecimal(100000, 2); got != "1000.00" {
		t.Fatalf("got %q", got)
	}
	if got := clickMinorToSom(1050); got != "10.50" {
		t.Fatalf("got %q", got)
	}
}

func TestClickShopSign_CompleteIncludesPrepareID(t *testing.T) {
	t.Parallel()
	secret := "click-secret"
	prepare := clickWebhookRequest{
		ClickTransID:    "1",
		ServiceID:       "2",
		MerchantTransID: "ord-1",
		Amount:          "1000.00",
		Action:          clickActionPrepare,
		SignTime:        "2026-06-28 12:00:00",
	}
	complete := prepare
	complete.Action = clickActionComplete
	complete.MerchantPrepareID = "sess-1"
	if clickShopSign(prepare, secret) == clickShopSign(complete, secret) {
		t.Fatal("prepare and complete signatures must differ")
	}
	if !verifyClickSignature(clickWebhookRequest{
		ClickTransID:      complete.ClickTransID,
		ServiceID:         complete.ServiceID,
		MerchantTransID:   complete.MerchantTransID,
		MerchantPrepareID: complete.MerchantPrepareID,
		Amount:            complete.Amount,
		Action:            complete.Action,
		SignTime:          complete.SignTime,
		SignString:        clickShopSign(complete, secret),
	}, secret) {
		t.Fatal("complete signature should verify")
	}
}

func TestClickMerchantAuthHeader_SHA1(t *testing.T) {
	t.Parallel()
	got := clickMerchantAuthHeader("user-1", "secret", 1700000000)
	if got == "" || got[:7] != "user-1:" {
		t.Fatalf("got %q", got)
	}
}
