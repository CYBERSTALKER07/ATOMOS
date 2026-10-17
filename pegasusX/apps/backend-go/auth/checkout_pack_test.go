package auth

import (
	"context"
	"net/http"
	"testing"
)

func TestRequireCheckoutPack_ShippedUZ(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	p, err := RequireCheckoutPack(Claims{MarketCode: "UZ"})
	if err != nil || p.Code != "UZ" || p.CurrencyCode != "UZS" {
		t.Fatalf("pack=%+v err=%v", p, err)
	}
	if p.CheckoutReadsThis {
		t.Fatal("catalog flag stays false until M2")
	}
}

func TestRequireCheckoutPack_PlannedEU(t *testing.T) {
	_, err := RequireCheckoutPack(Claims{MarketCode: "EU"})
	if err != ErrMarketPackNotShipped {
		t.Fatalf("err=%v", err)
	}
}

func TestRequireCheckoutPack_PlannedClonePacks(t *testing.T) {
	for _, code := range []string{"CA", "AU", "GB", "PK"} {
		_, err := RequireCheckoutPack(Claims{MarketCode: code})
		if err != ErrMarketPackNotShipped {
			t.Fatalf("%s err=%v", code, err)
		}
	}
}

func TestRequireCheckoutPack_Unknown(t *testing.T) {
	_, err := RequireCheckoutPack(Claims{MarketCode: "XX"})
	if err != ErrMarketPackUnknown {
		t.Fatalf("err=%v", err)
	}
}

func TestCheckoutPackFromContext_NoClaimsUsesEnv(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	p, err := CheckoutPackFromContext(context.Background())
	if err != nil || p.Code != "UZ" {
		t.Fatalf("pack=%+v err=%v", p, err)
	}
}

func TestPackAllowsPSP(t *testing.T) {
	p, _ := ResolveShippedMarketPack("UZ")
	if !PackAllowsPSP(p, "GLOBAL_PAY") || !PackAllowsPSP(p, "CASH") || !PackAllowsPSP(p, "pegasus") {
		t.Fatal("UZ pack must allow GLOBAL_PAY, CASH, and PEGASUS alias")
	}
	if !PackAllowsPSP(p, "PAYME") || !PackAllowsPSP(p, "CLICK") {
		t.Fatal("UZ pack lists PAYME and CLICK as unkeyed local rails")
	}
	if PackAllowsPSP(p, "STRIPE") {
		t.Fatal("STRIPE is not on the UZ pack")
	}
	if err := AssertPackPSP(p, "STRIPE"); err != ErrPackGatewayForbidden {
		t.Fatalf("err=%v", err)
	}
}

func TestResolveCheckoutCurrency(t *testing.T) {
	p, _ := ResolveShippedMarketPack("UZ")
	got, err := ResolveCheckoutCurrency(p, "")
	if err != nil || got != "UZS" {
		t.Fatalf("empty → %s %v", got, err)
	}
	if _, err := ResolveCheckoutCurrency(p, "EUR"); err != ErrPackCurrencyMismatch {
		t.Fatalf("err=%v", err)
	}
}

func TestIsShippedPackCurrency(t *testing.T) {
	if !IsShippedPackCurrency("uzs") || IsShippedPackCurrency("EUR") || IsShippedPackCurrency("") {
		t.Fatal("only shipped-pack currencies (UZS today)")
	}
}

func TestCheckoutPackHTTPStatus(t *testing.T) {
	st, code := CheckoutPackHTTPStatus(ErrMarketPackNotShipped)
	if st != http.StatusNotFound || code != ErrMarketPackNotShipped.Error() {
		t.Fatalf("%d %s", st, code)
	}
	st, code = CheckoutPackHTTPStatus(ErrPackGatewayForbidden)
	if st != http.StatusUnprocessableEntity || code != ErrPackGatewayForbidden.Error() {
		t.Fatalf("%d %s", st, code)
	}
	st, code = CheckoutPackHTTPStatus(ErrGeographyIncomplete)
	if st != http.StatusUnprocessableEntity || code != ErrGeographyIncomplete.Error() {
		t.Fatalf("%d %s", st, code)
	}
	st, code = CheckoutPackHTTPStatus(ErrCrossMarketDeferred)
	if st != http.StatusUnprocessableEntity || code != ErrCrossMarketDeferred.Error() {
		t.Fatalf("%d %s", st, code)
	}
}
