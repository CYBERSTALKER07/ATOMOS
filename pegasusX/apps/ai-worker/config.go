package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

// Config holds ai-worker runtime settings without importing backend-go/bootstrap.
type Config struct {
	SpannerEmulatorHost  string
	SpannerProject       string
	SpannerInstance      string
	SpannerDatabase      string
	KafkaBrokers         string
	KafkaAuthMode        string
	KafkaSASLUsername    string
	InternalAPIKey       string
	SeedSupplierCurrency string

	// M0.5 GCP Data Platform, BigQuery Feature Store, Vertex AI & Kafka topic alignment.
	CellID                   string
	BigQueryProjectID        string
	BigQueryDatasetPrefix    string
	BigQueryFeaturesDataset  string
	VertexAIProjectID        string
	VertexAILocation         string
	VertexStagingBucket      string
	AIWorkerBigQueryEnabled  bool
	AIWorkerVertexEnabled    bool
	KafkaConsumeDomainTopics bool
	KafkaConsumeIntel        bool
	KafkaTopicOrders         string
	KafkaTopicIntel          string
}

func loadConfig() (*Config, error) {
	bqProject := envOr("BIGQUERY_PROJECT_ID", "")
	bqPrefix := envOr("BIGQUERY_DATASET_PREFIX", "")
	vertexProject := envOr("VERTEX_AI_PROJECT_ID", bqProject)
	vertexLocation := envOr("VERTEX_AI_LOCATION", envOr("GCP_REGION", "europe-west3"))

	cfg := &Config{
		SpannerEmulatorHost:      resolveSpannerEmulatorHost(),
		SpannerProject:           envOr("SPANNER_PROJECT", "pegasusx-local"),
		SpannerInstance:          envOr("SPANNER_INSTANCE", "pegasusx-instance"),
		SpannerDatabase:          envOr("SPANNER_DATABASE", "pegasusx-db"),
		KafkaBrokers:             envOr("KAFKA_BROKERS", "localhost:9092"),
		KafkaAuthMode:            envOr("KAFKA_AUTH_MODE", ""),
		KafkaSASLUsername:        envOr("KAFKA_SASL_USERNAME", ""),
		InternalAPIKey:           envOr("INTERNAL_API_KEY", "dev-internal-key"),
		SeedSupplierCurrency:     envOr("SEED_SUPPLIER_CURRENCY", seedCurrencyFromPack()),
		CellID:                   resolveCellID(),
		BigQueryProjectID:        bqProject,
		BigQueryDatasetPrefix:    bqPrefix,
		BigQueryFeaturesDataset:  resolveFeaturesDataset(bqPrefix, envOr("BIGQUERY_FEATURES_DATASET", "")),
		VertexAIProjectID:        vertexProject,
		VertexAILocation:         vertexLocation,
		VertexStagingBucket:      envOr("VERTEX_AI_STAGING_BUCKET", ""),
		AIWorkerBigQueryEnabled:  envTruthy("AI_WORKER_BIGQUERY_ENABLED"),
		AIWorkerVertexEnabled:    envTruthy("AI_WORKER_VERTEX_ENABLED"),
		KafkaConsumeDomainTopics: events.ConsumeDomainTopics(),
		KafkaConsumeIntel:        envTruthy("AI_WORKER_CONSUME_INTEL") || events.ConsumeDomainTopics(),
		KafkaTopicOrders:         envOr("KAFKA_TOPIC_ORDERS", events.TopicOrders),
		KafkaTopicIntel:          envOr("KAFKA_TOPIC_INTEL", events.TopicIntel),
	}
	if auth.IsProduction() || auth.IsStaging() {
		if strings.TrimSpace(cfg.InternalAPIKey) == "" || cfg.InternalAPIKey == "dev-internal-key" {
			return nil, fmt.Errorf("INTERNAL_API_KEY must be set to a non-default secret in production and staging environments")
		}
		if cfg.AIWorkerBigQueryEnabled && strings.TrimSpace(cfg.BigQueryProjectID) == "" {
			return nil, fmt.Errorf("BIGQUERY_PROJECT_ID must be set when AI_WORKER_BIGQUERY_ENABLED=true in production and staging")
		}
		if cfg.AIWorkerVertexEnabled {
			if strings.TrimSpace(cfg.VertexAIProjectID) == "" || strings.TrimSpace(cfg.VertexStagingBucket) == "" {
				return nil, fmt.Errorf("VERTEX_AI_PROJECT_ID (or BIGQUERY_PROJECT_ID) and VERTEX_AI_STAGING_BUCKET must be set when AI_WORKER_VERTEX_ENABLED=true")
			}
		}
	}
	if strings.TrimSpace(cfg.InternalAPIKey) == "" {
		return nil, fmt.Errorf("INTERNAL_API_KEY required")
	}
	return cfg, nil
}

func resolveCellID() string {
	if c := strings.TrimSpace(os.Getenv("CELL_ID")); c != "" {
		return c
	}
	if h := strings.TrimSpace(auth.DefaultHomeCellFromEnv()); h != "" {
		return h
	}
	return "uz"
}

func resolveFeaturesDataset(prefix, override string) string {
	if strings.TrimSpace(override) != "" {
		return strings.TrimSpace(override)
	}
	p := strings.Trim(strings.ReplaceAll(strings.TrimSpace(prefix), "-", "_"), "_")
	if p == "" {
		return "features"
	}
	return p + "_features"
}

func envTruthy(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func resolveSpannerEmulatorHost() string {
	if v, ok := os.LookupEnv("SPANNER_EMULATOR_HOST"); ok {
		return strings.TrimSpace(v)
	}
	project := strings.TrimSpace(os.Getenv("SPANNER_PROJECT"))
	if project != "" && project != "pegasusx-local" {
		return ""
	}
	return "localhost:9010"
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
