package synthesis

import (
	"context"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestPreorderOperatingCurrency_StoredWins(t *testing.T) {
	got := preorderOperatingCurrency(context.Background(), "sup-1", "usd")
	if got != "USD" {
		t.Fatalf("got=%q", got)
	}
}

func TestPreorderOperatingCurrency_EmptyUsesShippedPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	got := preorderOperatingCurrency(context.Background(), "", "")
	if got != "UZS" {
		t.Fatalf("got=%q want UZS from shipped pack", got)
	}
}

func TestPreorderOperatingCurrency_PlannedDoesNotInvent(t *testing.T) {
	ctx := auth.WithClaims(context.Background(), auth.Claims{MarketCode: "EU"})
	got := preorderOperatingCurrency(ctx, "sup-1", "")
	if got != "" {
		t.Fatalf("planned pack must stay empty, got=%q", got)
	}
}
