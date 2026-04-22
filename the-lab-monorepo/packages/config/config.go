package config

import (
	"fmt"
	"os"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

// EnvConfig represents the strictly loaded environment variables needed to boot the services in this monorepo.
type EnvConfig struct {
	// Spanner Details
	SpannerProject       string `env:"SPANNER_PROJECT,required"`
	SpannerInstance      string `env:"SPANNER_INSTANCE,required"`
	SpannerDatabase      string `env:"SPANNER_DATABASE,required"`
	SpannerHotShardCount int    `env:"SPANNER_HOT_SHARD_COUNT" envDefault:"16"`

	// Kafka Details
	KafkaBrokerAddress string `env:"KAFKA_BROKER_ADDRESS,required"`

	// Web Backend Details
	BackendPort string `env:"BACKEND_PORT" envDefault:"8080"`

	// Authentication
	JWTSecret      string `env:"JWT_SECRET,required"`
	InternalAPIKey string `env:"INTERNAL_API_KEY,required"`

	// Runtime Environment — must be set explicitly ("development", "staging", "production").
	// No default: unset ENVIRONMENT is a fail-closed boot error.
	Environment string `env:"ENVIRONMENT,required"`

	// Optional Flag for Development / Redis
	RedisAddress string `env:"REDIS_ADDRESS"`

	// Field General Routing Settings
	GoogleMapsAPIKey string `env:"GOOGLE_MAPS_API_KEY"`
	DepotLocation    string `env:"DEPOT_LOCATION" envDefault:"41.2995,69.2401"` // Default to Tashkent coordinates

	// Supplier Cockpit Storage
	GCSBucketName string `env:"GCS_BUCKET_NAME"`

	// CORS — comma-separated origin allowlist. Unset yields a localhost-only
	// dev default evaluated by bootstrap (never trust this in production).
	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS" envSeparator:","`

	// Phase 2 dispatch optimiser (apps/ai-worker). When unset, bootstrap
	// leaves App.OptimizerClient nil and the dispatch orchestrator falls
	// straight through to the Phase 1 KMeans + binpack pipeline. Set to
	// e.g. "http://ai-worker:8081" in staging/production.
	OptimizerBaseURL string `env:"OPTIMIZER_BASE_URL"`
}

// LoadConfig leverages caarlos0/env to parse the system environment variables and populate the struct.
// It will instantly error and crash if any `required` tags are missing.
func LoadConfig() (*EnvConfig, error) {
	// Attempt to load .env from the current working directory, but don't fail if it's missing (e.g. production)
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		// Ignore missing file, keep going
	}

	cfg := &EnvConfig{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("FATAL: Environment configuration parsing failed: %w", err)
	}
	return cfg, nil
}

// IsDevelopment reports whether the process is running under ENVIRONMENT=development.
func (c *EnvConfig) IsDevelopment() bool { return c.Environment == "development" }

// IsProduction reports whether the process is running under ENVIRONMENT=production.
func (c *EnvConfig) IsProduction() bool { return c.Environment == "production" }
