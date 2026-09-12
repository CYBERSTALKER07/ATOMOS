package bootstrap

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailer"
)

// Config carries the runtime parameters. Loaded from environment by LoadConfig.
type Config struct {
	HTTPPort       string
	WorkerHTTPPort string
	RunMode        string

	SpannerEmulatorHost string
	SpannerProject      string
	SpannerInstance     string
	SpannerDatabase     string

	RedisAddr        string
	RedisPassword    string
	RedisPoolSize    int
	RedisMaxRetries  int
	RedisTLSEnabled  bool
	RedisCACertPEM   string // Memorystore server CA PEM (optional)
	RedisTLSInsecure bool   // skip TLS verify (staging only; prefer RedisCACertPEM)

	KafkaBrokers            string
	KafkaTopicMain          string
	KafkaTopicMainDLQ       string
	KafkaAuthMode           string // empty | GCP_MANAGED_OAUTH
	KafkaSASLUsername       string // service account email for GCP Managed Kafka
	WebSocketAllowedOrigins []string

	JWTSecret string
	JWTIssuer string
	// PlatformAdminMFARequired forces TOTP enroll+verify for PLATFORM_ADMIN governance routes.
	PlatformAdminMFARequired bool

	FirebaseAuthEnabled             bool
	FirebaseProjectID               string
	FirebaseCertsURL                string
	FirebaseCredentialsPath         string
	GlobalPayEnv                    string
	GlobalPayServiceID              string
	GlobalPayUsername               string
	GlobalPayPassword               string
	GlobalPayWebhookSecret          string
	AdyenWebhookSecret              string
	StripeWebhookSecret             string
	PaymeWebhookSecret              string
	ClickWebhookSecret              string
	AirwallexDirectExecutionEnabled bool

	TestingMode bool // bypasses strict-mode checks for infra that cannot be mocked

	SeedSupplierName       string
	SeedSupplierCountry    string
	SeedSupplierCurrency   string
	FxSeedUSDToUZSScaled   int64 // 0 = skip USD/UZS seed; set FX_SEED_USD_UZS_SCALED for tests
	DeliveryZoneCenterLat  float64
	DeliveryZoneCenterLng  float64
	DeliveryZoneRadiusKm   float64
	DeliveryZoneResolution int

	LogLevel  string
	LogFormat string

	RequireInfraAdapters bool
	// AllowMemoryFallback permits in-memory repos/outbox/cache when Spanner/Redis
	// are down. Forced false in production and whenever RequireInfraAdapters is true.
	// Set ALLOW_MEMORY_FALLBACK=true only for SSMR/local drills without full infra.
	AllowMemoryFallback bool
	ReliabilityEnabled  bool
	AllowAuthBypass     bool
	MaxSuppliers        int
	// AllowMultiSupplierRegister is ignored on POST /v1/auth/supplier/register
	// (GS-T2). New tenants mint via POST /v1/platform/tenants/register.
	AllowMultiSupplierRegister bool
	// TenantContextEnforced fail-closes authenticated routes missing TenantContext.
	TenantContextEnforced bool

	OptimizerBaseURL string
	RoutingOSRMURL   string
	// RoutingProvider selects geometry backends: auto|google|osrm (default auto).
	RoutingProvider  string
	InternalAPIKey   string
	GCSBucketName    string
	GoogleMapsAPIKey string

	// UpdatesBaseURL is the public CDN/origin used by OTA manifests (no trailing slash).
	// Env: UPDATES_BASE_URL. Required when PEGASUSX_ENV=production.
	UpdatesBaseURL string
	// UpdatesDefaultVersion is the fallback app version advertised by updater routes.
	UpdatesDefaultVersion string

	WeatherWorkerEnabled bool
	WeatherBaseURL       string
}

// LoadConfig reads environment-backed configuration with safe defaults for
// local development.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		HTTPPort:       envOr("HTTP_PORT", "8080"),
		WorkerHTTPPort: envOr("WORKER_HTTP_PORT", "8081"),
		RunMode:        envOr("PEGASUSX_RUN_MODE", RunModeAll),
		// Empty emulator host = real GCP. Local SSMR defaults to emulator only when
		// SPANNER_PROJECT is the local sandbox (or SPANNER_EMULATOR_HOST is set).
		SpannerEmulatorHost: resolveSpannerEmulatorHost(),
		SpannerProject:      envOr("SPANNER_PROJECT", "pegasusx-local"),
		SpannerInstance:     envOr("SPANNER_INSTANCE", "pegasusx-instance"),
		SpannerDatabase:     envOr("SPANNER_DATABASE", "pegasusx-db"),
		RedisAddr:           envOr("REDIS_ADDR", "localhost:6379"),
		RedisPassword:       envOr("REDIS_PASSWORD", ""),
		RedisPoolSize:       envInt("REDIS_POOL_SIZE", 50),
		RedisMaxRetries:     envInt("REDIS_MAX_RETRIES", 3),
		RedisTLSEnabled:     envBool("REDIS_TLS_ENABLED", false),
		RedisCACertPEM:      envOr("REDIS_CA_CERT", ""),
		RedisTLSInsecure:    envBool("REDIS_TLS_INSECURE", false),
		KafkaBrokers:        envOr("KAFKA_BROKERS", "localhost:9092"),
		KafkaTopicMain:      envOr("KAFKA_TOPIC_MAIN", "pegasusx-main"),
		KafkaTopicMainDLQ:   envOr("KAFKA_TOPIC_MAIN_DLQ", ""),
		// GCP_MANAGED_OAUTH = Managed Service for Apache Kafka (SASL_SSL + access token).
		KafkaAuthMode:                   envOr("KAFKA_AUTH_MODE", ""),
		KafkaSASLUsername:               envOr("KAFKA_SASL_USERNAME", ""),
		WebSocketAllowedOrigins:         splitAndTrimCSV(envOr("WS_ALLOWED_ORIGINS", "")),
		JWTSecret:                       envOr("JWT_SECRET", "dev-only-change-me"),
		JWTIssuer:                       envOr("JWT_ISSUER", "pegasusx-dev"),
		PlatformAdminMFARequired:        envBool("PLATFORM_ADMIN_MFA_REQUIRED", isProductionEnv()),
		FirebaseAuthEnabled:             envBool("FIREBASE_AUTH_ENABLED", false),
		FirebaseProjectID:               envOr("FIREBASE_PROJECT_ID", ""),
		FirebaseCertsURL:                envOr("FIREBASE_CERTS_URL", "https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com"),
		FirebaseCredentialsPath:         envOr("FIREBASE_CREDENTIALS_PATH", ""),
		GlobalPayEnv:                    envOr("GLOBAL_PAY_ENV", "dev"),
		GlobalPayServiceID:              envOr("GLOBAL_PAY_SERVICE_ID", "doc-supplier-service"),
		GlobalPayUsername:               envOr("GLOBAL_PAY_USERNAME", "doc-username"),
		GlobalPayPassword:               envOr("GLOBAL_PAY_PASSWORD", "doc-password"),
		GlobalPayWebhookSecret:          envOr("GLOBAL_PAY_WEBHOOK_SECRET", "dev-global-pay-secret"), // local/SSMR only; production rejected by ValidateProductionProfile; NewService never invents (P2-9)
		AdyenWebhookSecret:              envOr("ADYEN_WEBHOOK_SECRET", "dev-adyen-secret"),
		StripeWebhookSecret:             envOr("STRIPE_WEBHOOK_SECRET", "dev-stripe-secret"),
		PaymeWebhookSecret:              envOr("PAYME_WEBHOOK_SECRET", "dev-payme-secret"),
		ClickWebhookSecret:              envOr("CLICK_WEBHOOK_SECRET", "dev-click-secret"),
		AirwallexDirectExecutionEnabled: envBool("AIRWALLEX_DIRECT_EXECUTION_ENABLED", false),
		SeedSupplierName:                envOr("SEED_SUPPLIER_NAME", "pegasusX Supplier"),
		SeedSupplierCountry:             envOr("SEED_SUPPLIER_COUNTRY", "UZ"),
		SeedSupplierCurrency:            envOr("SEED_SUPPLIER_CURRENCY", seedCurrencyFromPack()),
		FxSeedUSDToUZSScaled:            envInt64("FX_SEED_USD_UZS_SCALED", 0),
		DeliveryZoneCenterLat:           envFloat("DELIVERY_ZONE_CENTER_LAT", defaultDeliveryZoneCenterLat),
		DeliveryZoneCenterLng:           envFloat("DELIVERY_ZONE_CENTER_LNG", defaultDeliveryZoneCenterLng),
		DeliveryZoneRadiusKm:            envFloat("DELIVERY_ZONE_RADIUS_KM", defaultDeliveryZoneRadiusKm),
		DeliveryZoneResolution:          envInt("DELIVERY_ZONE_RESOLUTION", retailer.DefaultPerimeterResolution),
		LogLevel:                        envOr("LOG_LEVEL", "info"),
		LogFormat:                       envOr("LOG_FORMAT", "json"),
		RequireInfraAdapters:            envBool("REQUIRE_INFRA_ADAPTERS", true),
		AllowMemoryFallback:             envBool("ALLOW_MEMORY_FALLBACK", false),
		ReliabilityEnabled:              envBool("RELIABILITY_MIDDLEWARE_ENABLED", true),
		AllowAuthBypass:                 envBool("ALLOW_AUTH_BYPASS", false),
		MaxSuppliers:                    envInt("MAX_SUPPLIERS", 1),
		AllowMultiSupplierRegister:      envBool("ALLOW_MULTI_SUPPLIER_REGISTER", false),
		// Same rule as seed fail-closed: explicit TENANT_CONTEXT_ENFORCED, else sandbox|production.
		TenantContextEnforced: auth.TenantContextEnforced(),
		OptimizerBaseURL:      envOr("OPTIMIZER_BASE_URL", "http://localhost:8081"),
		RoutingOSRMURL:        envOr("ROUTING_OSRM_URL", ""),
		RoutingProvider:       envOr("ROUTING_PROVIDER", "auto"),
		InternalAPIKey:        envOr("INTERNAL_API_KEY", "dev-internal-key"),
		GCSBucketName:         envOr("GCS_BUCKET_NAME", ""),
		GoogleMapsAPIKey:      envOr("GOOGLE_MAPS_API_KEY", envOr("GOOGLE_PLACES_API_KEY", "")),
		UpdatesBaseURL:        strings.TrimRight(strings.TrimSpace(envOr("UPDATES_BASE_URL", "")), "/"),
		UpdatesDefaultVersion: envOr("UPDATES_DEFAULT_VERSION", "1.0.0"),
		WeatherWorkerEnabled:  envBool("WEATHER_WORKER_ENABLED", true),
		WeatherBaseURL:        envOr("WEATHER_BASE_URL", "https://api.open-meteo.com/v1/forecast"),
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET required")
	}
	if strings.TrimSpace(cfg.KafkaTopicMainDLQ) == "" && strings.TrimSpace(cfg.KafkaTopicMain) != "" {
		cfg.KafkaTopicMainDLQ = strings.TrimSpace(cfg.KafkaTopicMain) + "-dlq"
	}
	// Enterprise fail-closed: production and strict infra never allow memory repos.
	if isProductionEnv() || cfg.RequireInfraAdapters {
		cfg.AllowMemoryFallback = false
	}
	if err := cfg.ValidateProductionProfile(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func seedCurrencyFromPack() string {
	c, err := auth.CurrencyFromContext(context.Background(), "")
	if err != nil {
		return ""
	}
	return c
}

// resolveSpannerEmulatorHost returns the emulator endpoint for local SSMR, or
// empty string for real Cloud Spanner (ADC + public API).
//
// Priority:
//  1. SPANNER_EMULATOR_HOST if set (including empty → force cloud)
//  2. Non-local SPANNER_PROJECT → cloud (no emulator)
//  3. Default localhost:9010 for pegasusx-local / unset project
func resolveSpannerEmulatorHost() string {
	if v, ok := os.LookupEnv("SPANNER_EMULATOR_HOST"); ok {
		return v
	}
	project := strings.TrimSpace(os.Getenv("SPANNER_PROJECT"))
	if project != "" && project != "pegasusx-local" {
		return ""
	}
	return "localhost:9010"
}

func envBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func envFloat(key string, fallback float64) float64 {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return fallback
	}
	return f
}

func envInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	i, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return fallback
	}
	return i
}

func envInt64(key string, fallback int64) int64 {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	i, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	if err != nil {
		return fallback
	}
	return i
}

func shopClosedGraceDuration() time.Duration {
	if _, ok := os.LookupEnv("SHOP_CLOSED_GRACE_MINUTES"); ok {
		minutes := envInt("SHOP_CLOSED_GRACE_MINUTES", 5)
		if minutes < 1 {
			minutes = 5
		}
		return time.Duration(minutes) * time.Minute
	}
	d, err := auth.ShopClosedGraceFromContext(context.Background(), "")
	if err != nil {
		return 0
	}
	return d
}

func splitAndTrimCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
