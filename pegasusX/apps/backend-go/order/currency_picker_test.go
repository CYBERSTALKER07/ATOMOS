package order

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
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
	got, err := svc.resolveOrderCurrency(context.Background(), "", "USD")
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
	got, err := svc.resolveOrderCurrency(context.Background(), "", "")
	if err != nil || got != "UZS" {
		t.Fatalf("empty → operating: got %q err %v", got, err)
	}
	got, err = svc.resolveOrderCurrency(context.Background(), "", "usd")
	if err != nil || got != "USD" {
		t.Fatalf("allowlisted: got %q err %v", got, err)
	}
	_, err = svc.resolveOrderCurrency(context.Background(), "", "EUR")
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

func TestNewService_EmptyCurrencyUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := NewService(ServiceConfig{})
	if svc.currency != "UZS" {
		t.Fatalf("got %q want UZS from pack", svc.currency)
	}
}

func TestNewService_EmptyCurrencyPlannedStaysEmpty(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "EU")
	svc := NewService(ServiceConfig{})
	if svc.currency != "" {
		t.Fatalf("got %q want empty (no UZS invent)", svc.currency)
	}
	_, err := svc.resolveOrderCurrency(context.Background(), "", "")
	if err != auth.ErrMarketPackNotShipped {
		t.Fatalf("err=%v", err)
	}
}

func TestParseCurrencyAllowlist_EmptyOperatingUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	got := ParseCurrencyAllowlist("", "")
	if len(got) != 1 || got[0] != "UZS" {
		t.Fatalf("got %#v", got)
	}
}

func TestNewFiscalPendingRow_EmptyCurrencyUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := NewService(ServiceConfig{})
	row := svc.newFiscalPendingRow(context.Background(), Order{OrderID: "o1", SupplierID: "s1"}, "CASH", "a1", 100)
	if row.Currency != "UZS" {
		t.Fatalf("got %q want UZS from pack", row.Currency)
	}
}

func TestHandleOrderCurrencies_PlannedPack404(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "EU")
	svc := NewService(ServiceConfig{})
	req := httptest.NewRequest(http.MethodGet, "/v1/order/currencies", nil).WithContext(
		auth.WithClaims(context.Background(), auth.Claims{MarketCode: "EU"}),
	)
	rec := httptest.NewRecorder()
	svc.HandleOrderCurrencies(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestFiscalCurrency_EmptyUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	c, err := fiscalCurrency(context.Background(), "", "")
	if err != nil || c != "UZS" {
		t.Fatalf("cur=%q err=%v want UZS from pack", c, err)
	}
}

func TestFiscalCurrency_StoredWins(t *testing.T) {
	c, err := fiscalCurrency(context.Background(), "", "usd")
	if err != nil || c != "USD" {
		t.Fatalf("cur=%q err=%v want USD", c, err)
	}
}

func TestFiscalCurrency_PlannedFailsClosed(t *testing.T) {
	ctx := auth.WithClaims(context.Background(), auth.Claims{MarketCode: "EU"})
	if _, err := fiscalCurrency(ctx, "sup-1", ""); err != auth.ErrMarketPackNotShipped {
		t.Fatalf("err=%v want %v", err, auth.ErrMarketPackNotShipped)
	}
}

func TestMySoliqEmptyCurrency_PlannedFailsClosedBeforeSigner(t *testing.T) {
	p := &MySoliqProvider{}
	ctx := auth.WithClaims(context.Background(), auth.Claims{MarketCode: "EU"})
	_, err := p.CreateReceipt(ctx, FiscalCreateRequest{AttemptID: "a", OrderID: "o", AmountMinor: 100})
	if err != auth.ErrMarketPackNotShipped {
		t.Fatalf("err=%v want planned pack fail-closed", err)
	}
}
