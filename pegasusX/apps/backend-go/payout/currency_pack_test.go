package payout

import (
	"context"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestCoalescePayoutCurrency_EmptyUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	c, err := coalescePayoutCurrency(context.Background(), "sup-1", "")
	if err != nil || c != "UZS" {
		t.Fatalf("cur=%q err=%v want UZS from pack", c, err)
	}
}

func TestCoalescePayoutCurrency_StoredWins(t *testing.T) {
	c, err := coalescePayoutCurrency(context.Background(), "sup-1", "usd")
	if err != nil || c != "USD" {
		t.Fatalf("cur=%q err=%v want USD", c, err)
	}
}

func TestCoalescePayoutCurrency_PlannedFailsClosed(t *testing.T) {
	ctx := auth.WithClaims(context.Background(), auth.Claims{MarketCode: "EU"})
	if _, err := coalescePayoutCurrency(ctx, "sup-1", ""); err != auth.ErrMarketPackNotShipped {
		t.Fatalf("err=%v want %v", err, auth.ErrMarketPackNotShipped)
	}
}
