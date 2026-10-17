package payment

import (
	"context"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestAssertSameMarket_L4Defense(t *testing.T) {
	t.Parallel()
	if err := auth.AssertSameMarket("UZ", "PK"); err != auth.ErrCrossMarketDeferred {
		t.Fatalf("credit/payout/fiscal must refuse mixed markets: %v", err)
	}
	if err := auth.AssertSameMarket("UZ", "UZ"); err != nil {
		t.Fatal(err)
	}
}

func TestNewService_EmptyCurrencyUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := NewService(ServiceConfig{})
	if svc.currency != "UZS" {
		t.Fatalf("got %q want UZS from pack", svc.currency)
	}
}

func TestNewService_EmptyCurrencyPlannedStaysEmpty(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "EU")
	svc := NewService(ServiceConfig{})
	if svc.currency != "" {
		t.Fatalf("got %q want empty (no UZS invent)", svc.currency)
	}
}

func TestRefundCardPayment_EmptyCurrencyPlannedFailsClosed(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "EU")
	svc := NewService(ServiceConfig{})
	ctx := auth.WithClaims(context.Background(), auth.Claims{MarketCode: "EU"})
	_, err := svc.RefundCardPayment(ctx, "ord-1", 100, "")
	if err != auth.ErrMarketPackNotShipped {
		t.Fatalf("err=%v", err)
	}
}

func TestRollupOperatingCurrencyMinor_EmptyUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	rows := []SettlementAuthorityRow{
		{Currency: "UZS", AmountMinorTotal: 250},
	}
	total, partial := rollupOperatingCurrencyMinor(context.Background(), nil, "", rows, nil)
	if partial || total != 250 {
		t.Fatalf("total=%d partial=%v", total, partial)
	}
}

func TestRollupOperatingCurrencyMinor_EmptyPlannedPartial(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "EU")
	rows := []SettlementAuthorityRow{
		{Currency: "EUR", AmountMinorTotal: 100},
	}
	total, partial := rollupOperatingCurrencyMinor(context.Background(), nil, "", rows, nil)
	if !partial || total != 0 {
		t.Fatalf("total=%d partial=%v want fail-closed partial", total, partial)
	}
}
