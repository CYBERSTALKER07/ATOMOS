package auth

import (
	"context"
	"net/http"
	"testing"
)

func TestPackBreachRadiusMeters_ShippedUZ(t *testing.T) {
	p, ok := ResolveShippedMarketPack("UZ")
	if !ok {
		t.Fatal("uz pack")
	}
	r, err := PackBreachRadiusMeters(p)
	if err != nil || r != 150 {
		t.Fatalf("radius=%v err=%v", r, err)
	}
}

func TestPackBreachRadiusMeters_PlannedFailsClosed(t *testing.T) {
	p, ok := ResolveMarketPack("EU")
	if !ok {
		t.Fatal("eu pack")
	}
	if _, err := PackBreachRadiusMeters(p); err != ErrMarketPackNotShipped {
		t.Fatalf("err=%v", err)
	}
}

func TestPackBreachRadiusMeters_ZeroRadiusInvalid(t *testing.T) {
	p, ok := ResolveShippedMarketPack("UZ")
	if !ok {
		t.Fatal("uz pack")
	}
	p.BreachRadiusMeters = 0
	if _, err := PackBreachRadiusMeters(p); err != ErrBreachRadiusInvalid {
		t.Fatalf("err=%v", err)
	}
}

func TestBreachRadiusFromContext_NoClaimsUsesEnv(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	r, err := BreachRadiusFromContext(context.Background(), "")
	if err != nil || r != 150 {
		t.Fatalf("radius=%v err=%v", r, err)
	}
}

func TestBreachRadiusFromContext_PlannedClaim(t *testing.T) {
	ctx := WithClaims(context.Background(), Claims{MarketCode: "EU"})
	if _, err := BreachRadiusFromContext(ctx, "sup-1"); err != ErrMarketPackNotShipped {
		t.Fatalf("err=%v", err)
	}
}

func TestBreachRadiusFromContext_ProfilePlanned(t *testing.T) {
	SetMarketProfileLookup(func(id string) (MarketProfile, bool) {
		if id == "sup-eu" {
			return MarketProfile{MarketCode: "EU", HomeCell: "cell-eu"}, true
		}
		return MarketProfile{}, false
	})
	t.Cleanup(func() { SetMarketProfileLookup(nil) })
	if _, err := BreachRadiusFromContext(context.Background(), "sup-eu"); err != ErrMarketPackNotShipped {
		t.Fatalf("err=%v", err)
	}
	r, err := BreachRadiusFromContext(context.Background(), "sup-other")
	if err != nil || r != 150 {
		t.Fatalf("fallback radius=%v err=%v", r, err)
	}
}

func TestBreachPackHTTPStatus(t *testing.T) {
	st, code := BreachPackHTTPStatus(ErrBreachRadiusInvalid)
	if st != http.StatusUnprocessableEntity || code != ErrBreachRadiusInvalid.Error() {
		t.Fatalf("%d %s", st, code)
	}
	st, code = BreachPackHTTPStatus(ErrMarketPackNotShipped)
	if st != http.StatusNotFound {
		t.Fatalf("%d %s", st, code)
	}
}
