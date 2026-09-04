package auth

import (
	"context"
	"testing"
)

func TestPackCountryCode_ShippedUZ(t *testing.T) {
	p, ok := ResolveShippedMarketPack("UZ")
	if !ok {
		t.Fatal("uz pack")
	}
	c, err := PackCountryCode(p)
	if err != nil || c != "UZ" {
		t.Fatalf("country=%s err=%v", c, err)
	}
	if p.CheckoutReadsThis {
		t.Fatal("flag stays false")
	}
}

func TestPackCountryCode_PlannedFailsClosed(t *testing.T) {
	p, ok := ResolveMarketPack("KZ")
	if !ok {
		t.Fatal("kz catalog")
	}
	if _, err := PackCountryCode(p); err != ErrMarketPackNotShipped {
		t.Fatalf("err=%v", err)
	}
}

func TestCountryFromContext_EnvUZ(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	c, err := CountryFromContext(context.Background(), "")
	if err != nil || c != "UZ" {
		t.Fatalf("country=%s err=%v", c, err)
	}
}

func TestCountryFromContext_PlannedFailsClosed(t *testing.T) {
	ctx := WithClaims(context.Background(), Claims{MarketCode: "KZ"})
	if _, err := CountryFromContext(ctx, "sup-1"); err != ErrMarketPackNotShipped {
		t.Fatalf("err=%v", err)
	}
}

func TestAssertSameMarket(t *testing.T) {
	t.Parallel()
	if err := AssertSameMarket("UZ", "", "uz"); err != nil {
		t.Fatal(err)
	}
	if err := AssertSameMarket("", "UZ"); err != ErrGeographyIncomplete {
		t.Fatalf("err=%v", err)
	}
	if err := AssertSameMarket("UZ", "PK"); err != ErrCrossMarketDeferred {
		t.Fatalf("err=%v", err)
	}
}

func TestCountryFromContext_KZTDoesNotInventKZ(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	c, err := CountryFromContext(context.Background(), "")
	if err != nil || c != "UZ" {
		t.Fatalf("pack country must not come from currency; got %s err=%v", c, err)
	}
}
