package factory

import (
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestNewService_EmptyCurrencyUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := NewService(ServiceConfig{})
	pack, _ := auth.ResolveShippedMarketPack("UZ")
	want, _ := auth.PackCurrency(pack)
	if svc.currency != want {
		t.Fatalf("currency=%q want %q", svc.currency, want)
	}
}

func TestNewService_EmptyCurrencyDoesNotInventOnPlannedPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "EU")
	svc := NewService(ServiceConfig{})
	if svc.currency != "" {
		t.Fatalf("planned pack must not invent UZS, got %q", svc.currency)
	}
}
