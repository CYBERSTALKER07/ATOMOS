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

func TestLoadConfig_M05BigQueryAndVertexFailClosed(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	t.Setenv("INTERNAL_API_KEY", "prod-strong-secret-12345")

	t.Setenv("AI_WORKER_BIGQUERY_ENABLED", "true")
	t.Setenv("BIGQUERY_PROJECT_ID", "")
	t.Setenv("AI_WORKER_VERTEX_ENABLED", "false")
	if _, err := loadConfig(); err == nil {
		t.Fatal("expected loadConfig to fail in production when AI_WORKER_BIGQUERY_ENABLED=true without BIGQUERY_PROJECT_ID")
	}

	t.Setenv("BIGQUERY_PROJECT_ID", "pegasusx-prod")
	t.Setenv("AI_WORKER_VERTEX_ENABLED", "true")
	t.Setenv("VERTEX_AI_STAGING_BUCKET", "")
	if _, err := loadConfig(); err == nil {
		t.Fatal("expected loadConfig to fail in production when AI_WORKER_VERTEX_ENABLED=true without VERTEX_AI_STAGING_BUCKET")
	}

	t.Setenv("VERTEX_AI_STAGING_BUCKET", "pegasusx-uz-vertex-staging")
	t.Setenv("BIGQUERY_DATASET_PREFIX", "uz-cell")
	t.Setenv("VERTEX_AI_LOCATION", "europe-west3")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("expected loadConfig to succeed with valid M0.5 BigQuery and Vertex AI settings, got: %v", err)
	}
	if cfg.BigQueryFeaturesDataset != "uz_cell_features" {
		t.Fatalf("BigQueryFeaturesDataset = %q, want %q", cfg.BigQueryFeaturesDataset, "uz_cell_features")
	}
	if cfg.VertexAIProjectID != "pegasusx-prod" {
		t.Fatalf("VertexAIProjectID = %q, want %q", cfg.VertexAIProjectID, "pegasusx-prod")
	}
	if cfg.VertexAILocation != "europe-west3" {
		t.Fatalf("VertexAILocation = %q, want %q", cfg.VertexAILocation, "europe-west3")
	}
}

func TestResolveFeaturesDataset(t *testing.T) {
	cases := []struct {
		prefix   string
		override string
		want     string
	}{
		{"", "", "features"},
		{"uz", "", "uz_features"},
		{"eu-west", "", "eu_west_features"},
		{"uz", "custom_features_ds", "custom_features_ds"},
	}
	for _, tc := range cases {
		if got := resolveFeaturesDataset(tc.prefix, tc.override); got != tc.want {
			t.Fatalf("resolveFeaturesDataset(%q, %q) = %q, want %q", tc.prefix, tc.override, got, tc.want)
		}
	}
}
