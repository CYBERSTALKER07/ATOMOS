package auth

import (
	"context"
	"testing"
)

func TestPackCurrency_ShippedUZ(t *testing.T) {
	p, ok := ResolveShippedMarketPack("UZ")
	if !ok {
		t.Fatal("uz pack")
	}
	c, err := PackCurrency(p)
	if err != nil || c != "UZS" {
		t.Fatalf("cur=%s err=%v", c, err)
	}
}

func TestCurrencyFromContext_NoClaimsUsesEnv(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	c, err := CurrencyFromContext(context.Background(), "")
	if err != nil || c != "UZS" {
		t.Fatalf("cur=%s err=%v", c, err)
	}
}

func TestCurrencyFromContext_PlannedFailsClosed(t *testing.T) {
	ctx := WithClaims(context.Background(), Claims{MarketCode: "EU"})
	if _, err := CurrencyFromContext(ctx, "sup-1"); err != ErrMarketPackNotShipped {
		t.Fatalf("err=%v", err)
	}
}

func TestCoalesceCurrency_StoredWins(t *testing.T) {
	c, err := CoalesceCurrency(context.Background(), "", "usd")
	if err != nil || c != "USD" {
		t.Fatalf("cur=%s err=%v", c, err)
	}
}

func TestCoalesceCurrency_EmptyUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	c, err := CoalesceCurrency(context.Background(), "", "")
	if err != nil || c != "UZS" {
		t.Fatalf("cur=%s err=%v", c, err)
	}
}
