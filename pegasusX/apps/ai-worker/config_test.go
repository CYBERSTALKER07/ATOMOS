package main

import (
	"testing"
)

func TestSeedCurrencyFromPack_ShippedUZ(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	if got := seedCurrencyFromPack(); got != "UZS" {
		t.Fatalf("got=%q want UZS", got)
	}
}

func TestSeedCurrencyFromPack_PlannedEmpty(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "EU")
	if got := seedCurrencyFromPack(); got != "" {
		t.Fatalf("planned pack must not invent, got=%q", got)
	}
}

func TestLoadConfig_SeedCurrencyFromPackWhenUnset(t *testing.T) {
	t.Setenv("INTERNAL_API_KEY", "dev-internal-key")
	t.Setenv("SEED_SUPPLIER_CURRENCY", "")
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SeedSupplierCurrency != "UZS" {
		t.Fatalf("got=%q", cfg.SeedSupplierCurrency)
	}
}
