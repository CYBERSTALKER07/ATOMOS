package order

import (
	"testing"
)

func TestParseCurrencyAllowlist_AlwaysIncludesOperating(t *testing.T) {
	got := ParseCurrencyAllowlist("USD,eur", "UZS")
	if len(got) < 1 || got[0] != "UZS" {
		t.Fatalf("operating first: got %#v", got)
	}
	want := map[string]bool{"UZS": true, "USD": true, "EUR": true}
	for _, c := range got {
		if !want[c] {
			t.Fatalf("unexpected %q in %#v", c, got)
		}
		delete(want, c)
	}
	if len(want) != 0 {
		t.Fatalf("missing %#v", want)
	}
}

func TestResolveOrderCurrency_DisabledIgnoresRequest(t *testing.T) {
	svc := NewService(ServiceConfig{
		Currency:              "UZS",
		CurrencyPickerEnabled: false,
		CurrencyAllowlist:     ParseCurrencyAllowlist("USD", "UZS"),
	})
	got, err := svc.resolveOrderCurrency("USD")
	if err != nil {
		t.Fatal(err)
	}
	if got != "UZS" {
		t.Fatalf("got %q want UZS", got)
	}
}

func TestResolveOrderCurrency_EnabledAllowlist(t *testing.T) {
	svc := NewService(ServiceConfig{
		Currency:              "UZS",
		CurrencyPickerEnabled: true,
		CurrencyAllowlist:     ParseCurrencyAllowlist("USD", "UZS"),
	})
	got, err := svc.resolveOrderCurrency("")
	if err != nil || got != "UZS" {
		t.Fatalf("empty → operating: got %q err %v", got, err)
	}
	got, err = svc.resolveOrderCurrency("usd")
	if err != nil || got != "USD" {
		t.Fatalf("allowlisted: got %q err %v", got, err)
	}
	_, err = svc.resolveOrderCurrency("EUR")
	if err != ErrCurrencyNotAllowed {
		t.Fatalf("want ErrCurrencyNotAllowed, got %v", err)
	}
}

func TestCurrencyOptions(t *testing.T) {
	svc := NewService(ServiceConfig{
		Currency:              "UZS",
		CurrencyPickerEnabled: true,
		CurrencyAllowlist:     ParseCurrencyAllowlist("USD", "UZS"),
	})
	opts := svc.CurrencyOptions()
	if !opts.Enabled || opts.OperatingCurrency != "UZS" || len(opts.Allowlist) < 2 {
		t.Fatalf("unexpected options %#v", opts)
	}
}
