package main

import (
	"context"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestSmokeOperatingCurrency_EmptyUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	if got := smokeOperatingCurrency(context.Background(), ""); got != "UZS" {
		t.Fatalf("got %q want UZS from pack", got)
	}
}

func TestSmokeOperatingCurrency_StoredWins(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	if got := smokeOperatingCurrency(context.Background(), "eur"); got != "EUR" {
		t.Fatalf("got %q want EUR", got)
	}
}

func TestSmokeOperatingCurrency_PlannedDoesNotInvent(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "EU")
	if got := smokeOperatingCurrency(context.Background(), ""); got != "" {
		t.Fatalf("planned pack must not invent UZS, got %q", got)
	}
}

func TestSmokeOperatingCurrency_ClaimsPlannedDoesNotInvent(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	ctx := auth.WithClaims(context.Background(), auth.Claims{MarketCode: "EU"})
	if got := smokeOperatingCurrency(ctx, ""); got != "" {
		t.Fatalf("EU claims must not invent UZS, got %q", got)
	}
}
