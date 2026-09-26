package auth

import (
	"context"
	"net/http"
	"testing"
)

func TestPackFiscalAdapter_ShippedUZ(t *testing.T) {
	p, ok := ResolveShippedMarketPack("UZ")
	if !ok {
		t.Fatal("uz pack")
	}
	ad, err := PackFiscalAdapter(p)
	if err != nil || ad != "MY_SOLIQ" {
		t.Fatalf("adapter=%s err=%v", ad, err)
	}
}

func TestPackFiscalAdapter_PEPPOLUnimplemented(t *testing.T) {
	p, ok := ResolveMarketPack("EU")
	if !ok {
		t.Fatal("eu pack")
	}
	if _, err := PackFiscalAdapter(p); err != ErrMarketPackNotShipped {
		t.Fatalf("planned pack err=%v", err)
	}
	p.Status = MarketPackShipped
	if _, err := PackFiscalAdapter(p); err != ErrFiscalAdapterUnimplemented {
		t.Fatalf("peppol err=%v", err)
	}
}

func TestFiscalPackFromContext_NoClaimsUsesEnv(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	p, err := FiscalPackFromContext(context.Background(), "")
	if err != nil || p.Code != "UZ" {
		t.Fatalf("pack=%+v err=%v", p, err)
	}
}

func TestFiscalPackFromContext_PlannedClaim(t *testing.T) {
	ctx := WithClaims(context.Background(), Claims{MarketCode: "EU"})
	_, err := FiscalPackFromContext(ctx, "sup-1")
	if err != ErrMarketPackNotShipped {
		t.Fatalf("err=%v", err)
	}
}

func TestFiscalPackForSupplier_ProfilePlanned(t *testing.T) {
	SetMarketProfileLookup(func(id string) (MarketProfile, bool) {
		if id == "sup-eu" {
			return MarketProfile{MarketCode: "EU", HomeCell: "cell-eu"}, true
		}
		return MarketProfile{}, false
	})
	t.Cleanup(func() { SetMarketProfileLookup(nil) })
	_, err := FiscalPackForSupplier("sup-eu")
	if err != ErrMarketPackNotShipped {
		t.Fatalf("err=%v", err)
	}
	p, err := FiscalPackForSupplier("sup-other")
	if err != nil || p.Code != "UZ" {
		t.Fatalf("fallback pack=%+v err=%v", p, err)
	}
}

func TestAssertFiscalRuntime_FakeForbiddenInProd(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	p, _ := ResolveShippedMarketPack("UZ")
	if err := AssertFiscalRuntime(p, "FAKE"); err != ErrFakeFiscalForbidden {
		t.Fatalf("err=%v", err)
	}
}

func TestAssertFiscalRuntime_PegasusAllowedOnUZOutsideProd(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "ssmr")
	p, _ := ResolveShippedMarketPack("UZ")
	if err := AssertFiscalRuntime(p, "PEGASUS"); err != nil {
		t.Fatal(err)
	}
	if err := AssertFiscalRuntime(p, "FAKE"); err != nil {
		t.Fatal(err)
	}
}

func TestBuyerAcceptancePollerAllowed_UZOnly(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	if !BuyerAcceptancePollerAllowed() {
		t.Fatal("UZ MY_SOLIQ pack must allow poller")
	}
	t.Setenv("DEFAULT_MARKET_CODE", "EU")
	if BuyerAcceptancePollerAllowed() {
		t.Fatal("planned EU must not start Soliq poller")
	}
}

func TestFiscalPackHTTPStatus(t *testing.T) {
	st, code := FiscalPackHTTPStatus(ErrFiscalAdapterUnimplemented)
	if st != http.StatusUnprocessableEntity || code != ErrFiscalAdapterUnimplemented.Error() {
		t.Fatalf("%d %s", st, code)
	}
	st, code = FiscalPackHTTPStatus(ErrMarketPackNotShipped)
	if st != http.StatusNotFound {
		t.Fatalf("%d %s", st, code)
	}
}
