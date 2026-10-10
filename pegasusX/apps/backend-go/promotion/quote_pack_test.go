package promotion

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestQuoteCheckout_StampsShippedPackCurrency(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := NewService(nil, nil, nil, slog.Default())
	q, err := svc.QuoteCheckout(context.Background(), "sup-1", "ret-1", []LineInput{
		{ProductID: "a", Quantity: 1, UnitPrice: 100},
	})
	if err != nil {
		t.Fatal(err)
	}
	if q.Currency != "UZS" || q.MarketCode != "UZ" {
		t.Fatalf("quote=%+v", q)
	}
	if len(q.Lines) != 1 || q.Lines[0].Currency != "UZS" {
		t.Fatalf("lines=%+v", q.Lines)
	}
}

func TestQuoteCheckout_LineCurrencyMismatch(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := NewService(nil, nil, nil, slog.Default())
	_, err := svc.QuoteCheckout(context.Background(), "sup-1", "ret-1", []LineInput{
		{ProductID: "a", Quantity: 1, UnitPrice: 100, Currency: "EUR"},
	})
	if !errors.Is(err, auth.ErrPackCurrencyMismatch) {
		t.Fatalf("err=%v", err)
	}
}

func TestQuoteCheckout_PlannedPackFailsClosed(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := NewService(nil, nil, nil, slog.Default())
	ctx := auth.WithClaims(context.Background(), auth.Claims{MarketCode: "EU"})
	_, err := svc.QuoteCheckout(ctx, "sup-1", "ret-1", []LineInput{
		{ProductID: "a", Quantity: 1, UnitPrice: 100},
	})
	if !errors.Is(err, auth.ErrMarketPackNotShipped) {
		t.Fatalf("err=%v", err)
	}
}
