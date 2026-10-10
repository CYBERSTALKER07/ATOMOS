package fxrates

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestConvertSameCurrency(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	got, err := svc.ConvertMinor(context.Background(), "uzs", "UZS", 1500, time.Now())
	if err != nil || got != 1500 {
		t.Fatalf("got %d err=%v", got, err)
	}
}

func TestConvertUSDToUZS(t *testing.T) {
	repo := NewMemoryRepository()
	at := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	// 1 USD = 12500.00 UZS → scaled = 12500 * 1e8 = 1_250_000_000_000
	_ = repo.Upsert(context.Background(), ScaledRate("USD", "UZS", 12_500*DefaultScale, "SEED", at))
	svc := NewService(repo)
	// 2.00 USD minor = 200 → 2_500_000 UZS minor
	got, err := svc.ConvertMinor(context.Background(), "USD", "UZS", 200, at.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	want := int64(2_500_000)
	if got != want {
		t.Fatalf("got %d want %d", got, want)
	}
}

func TestConvertMissingRate(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	_, err := svc.ConvertMinor(context.Background(), "EUR", "UZS", 100, time.Now())
	if !errors.Is(err, ErrRateMissing) {
		t.Fatalf("err=%v", err)
	}
}

func TestConvertInverse(t *testing.T) {
	repo := NewMemoryRepository()
	at := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	_ = repo.Upsert(context.Background(), ScaledRate("USD", "UZS", 10_000*DefaultScale, "SEED", at))
	svc := NewService(repo)
	// 10000.00 UZS (1_000_000 minor) → 1.00 USD (100 minor)
	got, err := svc.ConvertMinor(context.Background(), "UZS", "USD", 1_000_000, at.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if got != 100 {
		t.Fatalf("got %d want 100", got)
	}
}

func TestAssertSameCurrency(t *testing.T) {
	if err := AssertSameCurrency("UZS", "uzs"); err != nil {
		t.Fatal(err)
	}
	if err := AssertSameCurrency("UZS", "USD"); !errors.Is(err, ErrCurrencyMismatch) {
		t.Fatalf("err=%v", err)
	}
	if err := AssertSameCurrency("UZS", ""); err != nil {
		t.Fatal(err)
	}
}

func TestNeverSilentOneToOne(t *testing.T) {
	repo := NewMemoryRepository()
	_ = repo.Upsert(context.Background(), IdentityRate("UZS", "SEED", time.Now()))
	svc := NewService(repo)
	_, err := svc.ConvertMinor(context.Background(), "USD", "EUR", 100, time.Now())
	if !errors.Is(err, ErrRateMissing) {
		t.Fatalf("expected missing, got %v", err)
	}
}
