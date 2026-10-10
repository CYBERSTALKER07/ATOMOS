package fxrates

import (
	"context"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestSeedOperatingCurrency_EmptyUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	if got := seedOperatingCurrency(context.Background(), ""); got != "UZS" {
		t.Fatalf("got %q want UZS from pack", got)
	}
}

func TestSeedOperatingCurrency_StoredWins(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	if got := seedOperatingCurrency(context.Background(), "eur"); got != "EUR" {
		t.Fatalf("got %q want EUR", got)
	}
}

func TestSeedOperatingCurrency_PlannedDoesNotInvent(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "EU")
	if got := seedOperatingCurrency(context.Background(), ""); got != "" {
		t.Fatalf("planned pack must not invent UZS, got %q", got)
	}
}

func TestSeedBootstrapRates_EmptyUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	repo := NewMemoryRepository()
	if err := SeedBootstrapRates(context.Background(), repo, SeedOptions{}); err != nil {
		t.Fatal(err)
	}
	rate, ok, err := repo.GetAsOf(context.Background(), "UZS", "UZS", time.Now())
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if rate.RateScaled != DefaultScale {
		t.Fatalf("scaled=%d", rate.RateScaled)
	}
}

func TestSeedBootstrapRates_EmptyPlannedDoesNotInvent(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "EU")
	repo := NewMemoryRepository()
	if err := SeedBootstrapRates(context.Background(), repo, SeedOptions{}); err != nil {
		t.Fatal(err)
	}
	_, ok, err := repo.GetAsOf(context.Background(), "UZS", "UZS", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("planned pack must not seed UZS identity")
	}
	rates, err := repo.ListLatest(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rates) != 0 {
		t.Fatalf("want no identity seed, got %+v", rates)
	}
}

func TestSeedBootstrapRates_ClaimsPlannedDoesNotInvent(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	repo := NewMemoryRepository()
	ctx := auth.WithClaims(context.Background(), auth.Claims{MarketCode: "EU"})
	if err := SeedBootstrapRates(ctx, repo, SeedOptions{}); err != nil {
		t.Fatal(err)
	}
	_, ok, err := repo.GetAsOf(ctx, "UZS", "UZS", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("EU claims must not seed UZS identity")
	}
}
