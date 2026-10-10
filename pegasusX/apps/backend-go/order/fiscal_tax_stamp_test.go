package order

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/tax"
)

func TestStampTaxRegimeTxn_PackCountryNotCurrency(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := NewService(ServiceConfig{})
	svc.SetTaxService(tax.NewService(tax.NewMemoryRepository(), nil, slog.Default()))
	err := svc.stampTaxRegimeTxn(context.Background(), nil, &Order{
		OrderID:    "o1",
		SupplierID: "s1",
		Currency:   "KZT",
		LineItems:  []LineItem{{SKU: "a", Quantity: 1, UnitPrice: 100}},
	})
	if err == nil || !strings.Contains(err.Error(), "country UZ") {
		t.Fatalf("err=%v want missing regime for pack UZ (not KZ from KZT)", err)
	}
}

func TestStampTaxRegimeTxn_PlannedFailsClosed(t *testing.T) {
	svc := NewService(ServiceConfig{})
	svc.SetTaxService(tax.NewService(tax.NewMemoryRepository(), nil, slog.Default()))
	ctx := auth.WithClaims(context.Background(), auth.Claims{MarketCode: "KZ", SupplierID: "s1"})
	err := svc.stampTaxRegimeTxn(ctx, nil, &Order{
		OrderID:    "o1",
		SupplierID: "s1",
		Currency:   "KZT",
		LineItems:  []LineItem{{SKU: "a", Quantity: 1, UnitPrice: 100}},
	})
	if !errors.Is(err, auth.ErrMarketPackNotShipped) {
		t.Fatalf("err=%v", err)
	}
}

func TestStampTaxRegimeTxn_NilTaxServiceSkips(t *testing.T) {
	svc := NewService(ServiceConfig{})
	if err := svc.stampTaxRegimeTxn(context.Background(), nil, &Order{Currency: "KZT"}); err != nil {
		t.Fatalf("nil tax svc must skip: %v", err)
	}
}
