package main

import (
	"context"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

func TestSimOperatingCurrency_EmptyUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	if got := simOperatingCurrency(context.Background(), ""); got != "UZS" {
		t.Fatalf("got %q want UZS from pack", got)
	}
}

func TestSimOperatingCurrency_StoredWins(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	if got := simOperatingCurrency(context.Background(), "eur"); got != "EUR" {
		t.Fatalf("got %q want EUR", got)
	}
}

func TestSimOperatingCurrency_PlannedDoesNotInvent(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "EU")
	if got := simOperatingCurrency(context.Background(), ""); got != "" {
		t.Fatalf("planned pack must not invent UZS, got %q", got)
	}
}

func TestSimOperatingCurrency_ClaimsPlannedDoesNotInvent(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	ctx := auth.WithClaims(context.Background(), auth.Claims{MarketCode: "EU"})
	if got := simOperatingCurrency(ctx, ""); got != "" {
		t.Fatalf("EU claims must not invent UZS, got %q", got)
	}
}

func TestSimulatorOperatingCurrency_UsesSeed(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	s := NewSimulator(&bootstrap.Config{SeedSupplierCurrency: "kzt"}, "http://localhost")
	if got := s.operatingCurrency(context.Background()); got != "KZT" {
		t.Fatalf("got %q want KZT from seed", got)
	}
}
