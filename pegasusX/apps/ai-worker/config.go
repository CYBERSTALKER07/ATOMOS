package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Config holds ai-worker runtime settings without importing backend-go/bootstrap.
type Config struct {
	SpannerEmulatorHost string
	SpannerProject      string
	SpannerInstance     string
	SpannerDatabase     string
	KafkaBrokers        string
	InternalAPIKey      string
	SeedSupplierCurrency string
}

func loadConfig() (*Config, error) {
	cfg := &Config{
		SpannerEmulatorHost:  envOr("SPANNER_EMULATOR_HOST", "localhost:9010"),
		SpannerProject:       envOr("SPANNER_PROJECT", "pegasusx-local"),
		SpannerInstance:      envOr("SPANNER_INSTANCE", "pegasusx-instance"),
		SpannerDatabase:      envOr("SPANNER_DATABASE", "pegasusx-db"),
		KafkaBrokers:         envOr("KAFKA_BROKERS", "localhost:9092"),
		InternalAPIKey:       envOr("INTERNAL_API_KEY", "dev-internal-key"),
		SeedSupplierCurrency: envOr("SEED_SUPPLIER_CURRENCY", seedCurrencyFromPack()),
	}
	if strings.TrimSpace(cfg.InternalAPIKey) == "" {
		return nil, fmt.Errorf("INTERNAL_API_KEY required")
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

// seedCurrencyFromPack is the same empty-seed law as backend LoadConfig:
// shipped pack currency, planned/unknown stays empty — never invent UZS.
func seedCurrencyFromPack() string {
	c, err := auth.CurrencyFromContext(context.Background(), "")
	if err != nil {
		return ""
	}
	return c
}
