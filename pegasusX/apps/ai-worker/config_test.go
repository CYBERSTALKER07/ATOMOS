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
	t.Setenv("PEGASUSX_ENV", "local")
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

func TestLoadConfig_FailClosedProduction(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	t.Setenv("INTERNAL_API_KEY", "dev-internal-key")
	if _, err := loadConfig(); err == nil {
		t.Fatal("expected loadConfig to fail in production with default internal api key")
	}

	t.Setenv("INTERNAL_API_KEY", "")
	if _, err := loadConfig(); err == nil {
		t.Fatal("expected loadConfig to fail in production with empty internal api key")
	}

	t.Setenv("INTERNAL_API_KEY", "prod-strong-secret-12345")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("expected loadConfig to succeed with strong key in production, got: %v", err)
	}
	if cfg.InternalAPIKey != "prod-strong-secret-12345" {
		t.Fatalf("expected key to match, got %q", cfg.InternalAPIKey)
	}
}

func TestLoadConfig_FailClosedStaging(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "staging")
	t.Setenv("INTERNAL_API_KEY", "dev-internal-key")
	if _, err := loadConfig(); err == nil {
		t.Fatal("expected loadConfig to fail in staging with default internal api key")
	}

	t.Setenv("INTERNAL_API_KEY", "staging-strong-secret-12345")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("expected loadConfig to succeed with strong key in staging, got: %v", err)
	}
	if cfg.InternalAPIKey != "staging-strong-secret-12345" {
		t.Fatalf("expected key to match, got %q", cfg.InternalAPIKey)
	}
}
