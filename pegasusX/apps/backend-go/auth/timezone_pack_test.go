package auth

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestPackTimezone_ShippedUZ(t *testing.T) {
	p, ok := ResolveShippedMarketPack("UZ")
	if !ok {
		t.Fatal("uz pack")
	}
	name, err := PackTimezoneName(p)
	if err != nil || name != "Asia/Tashkent" {
		t.Fatalf("tz=%s err=%v", name, err)
	}
	loc, err := PackLocation(p)
	if err != nil || loc == nil {
		t.Fatal(err)
	}
	grace, err := PackShopClosedGrace(p)
	if err != nil || grace != 10*time.Minute {
		t.Fatalf("grace=%s err=%v", grace, err)
	}
	scope, err := PackWeatherScope(p)
	if err != nil || scope != "city:Tashkent" {
		t.Fatalf("scope=%s err=%v", scope, err)
	}
	hours, err := PackFactorySLADefaultHours(p)
	if err != nil || hours != 48 {
		t.Fatalf("sla=%v err=%v", hours, err)
	}
	maxH, err := PackLaborMaxShiftHours(p)
	if err != nil || maxH != 12 {
		t.Fatalf("max=%d err=%v", maxH, err)
	}
}

func TestPackTimezone_PlannedFailsClosed(t *testing.T) {
	p, ok := ResolveMarketPack("EU")
	if !ok {
		t.Fatal("eu pack")
	}
	if _, err := PackTimezoneName(p); err != ErrMarketPackNotShipped {
		t.Fatalf("tz err=%v", err)
	}
	if _, err := PackShopClosedGrace(p); err != ErrMarketPackNotShipped {
		t.Fatalf("grace err=%v", err)
	}
	if _, err := PackWeatherScope(p); err != ErrMarketPackNotShipped {
		t.Fatalf("weather err=%v", err)
	}
}

func TestTimezoneFromContext_PlannedClaim(t *testing.T) {
	ctx := WithClaims(context.Background(), Claims{MarketCode: "EU"})
	if _, err := TimezoneFromContext(ctx, "sup-1"); err != ErrMarketPackNotShipped {
		t.Fatalf("err=%v", err)
	}
	if _, err := ShopClosedGraceFromContext(ctx, "sup-1"); err != ErrMarketPackNotShipped {
		t.Fatalf("grace err=%v", err)
	}
	if _, err := WeatherScopeFromContext(ctx, "sup-1"); err != ErrMarketPackNotShipped {
		t.Fatalf("weather err=%v", err)
	}
}

func TestTimezoneFromContext_NoClaimsUsesEnv(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	loc, err := TimezoneFromContext(context.Background(), "")
	if err != nil || loc.String() != "Asia/Tashkent" {
		t.Fatalf("loc=%v err=%v", loc, err)
	}
}

func TestTimezonePackHTTPStatus(t *testing.T) {
	st, code := TimezonePackHTTPStatus(ErrTimezoneInvalid)
	if st != http.StatusUnprocessableEntity || code != ErrTimezoneInvalid.Error() {
		t.Fatalf("%d %s", st, code)
	}
	st, _ = TimezonePackHTTPStatus(ErrMarketPackNotShipped)
	if st != http.StatusNotFound {
		t.Fatalf("%d", st)
	}
}
