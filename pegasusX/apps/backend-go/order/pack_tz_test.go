package order

import (
	"context"
	"errors"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestResolveCalendarLocation_StoredTZWins(t *testing.T) {
	loc, err := resolveCalendarLocation(context.Background(), "sup-1", "Europe/Berlin")
	if err != nil || loc.String() != "Europe/Berlin" {
		t.Fatalf("loc=%v err=%v", loc, err)
	}
}

func TestResolveCalendarLocation_PackUZ(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	loc, err := resolveCalendarLocation(context.Background(), "", "")
	if err != nil || loc.String() != "Asia/Tashkent" {
		t.Fatalf("loc=%v err=%v", loc, err)
	}
}

func TestResolveCalendarLocation_PlannedFailsClosed(t *testing.T) {
	ctx := auth.WithClaims(context.Background(), auth.Claims{MarketCode: "EU"})
	if _, err := resolveCalendarLocation(ctx, "sup-1", ""); !errors.Is(err, auth.ErrMarketPackNotShipped) {
		t.Fatalf("err=%v", err)
	}
}
