package payment

import (
	"context"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
)

func TestRollupOperatingCurrencyMinor_Identity(t *testing.T) {
	t.Parallel()
	rows := []SettlementAuthorityRow{
		{Currency: "UZS", AmountMinorTotal: 1000, LastOccurredAt: time.Now().UTC()},
		{Currency: "uzs", AmountMinorTotal: 500, LastOccurredAt: time.Now().UTC()},
	}
	total, partial := rollupOperatingCurrencyMinor(context.Background(), nil, "UZS", rows, time.Now)
	if partial {
		t.Fatal("expected no partial for same-currency rows")
	}
	if total != 1500 {
		t.Fatalf("total=%d want 1500", total)
	}
}

func TestRollupOperatingCurrencyMinor_ConvertsWithRate(t *testing.T) {
	t.Parallel()
	repo := fxrates.NewMemoryRepository()
	at := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	_ = repo.Upsert(context.Background(), fxrates.ScaledRate("USD", "UZS", 12_750_000_000, "TEST", at))
	fx := fxrates.NewService(repo)
	rows := []SettlementAuthorityRow{
		{Currency: "USD", AmountMinorTotal: 200, LastOccurredAt: at.Add(time.Hour)},
		{Currency: "UZS", AmountMinorTotal: 100, LastOccurredAt: at.Add(time.Hour)},
	}
	total, partial := rollupOperatingCurrencyMinor(context.Background(), fx, "UZS", rows, func() time.Time { return at.Add(2 * time.Hour) })
	if partial {
		t.Fatal("expected full conversion")
	}
	// 200 USD minor * 127.5 = 25500 UZS minor + 100
	if total != 25600 {
		t.Fatalf("total=%d want 25600", total)
	}
}

func TestRollupOperatingCurrencyMinor_PartialOnMissingRate(t *testing.T) {
	t.Parallel()
	fx := fxrates.NewService(fxrates.NewMemoryRepository())
	rows := []SettlementAuthorityRow{
		{Currency: "EUR", AmountMinorTotal: 100, LastOccurredAt: time.Now().UTC()},
		{Currency: "UZS", AmountMinorTotal: 50, LastOccurredAt: time.Now().UTC()},
	}
	total, partial := rollupOperatingCurrencyMinor(context.Background(), fx, "UZS", rows, time.Now)
	if !partial {
		t.Fatal("expected partial")
	}
	if total != 50 {
		t.Fatalf("total=%d want 50 (UZS only)", total)
	}
}
