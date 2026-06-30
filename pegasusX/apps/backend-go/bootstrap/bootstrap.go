// Package bootstrap is the composition root for the pegasusX backend.
//
// It owns construction of every long-lived singleton: Spanner client, Redis
// cache, Kafka writers, WebSocket hubs, services, and middleware. Domain
// packages receive their dependencies through narrow Deps structs; they never
// reach into bootstrap.
package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap/memory"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/catalog"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/optimizerclient"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/plan"
	"github.com/pegasusx/pegasusx/apps/backend-go/driver"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/factory"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/infraroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/inventory"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafka"
	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/payload"
	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
	"github.com/pegasusx/pegasusx/apps/backend-go/platform"
	"github.com/pegasusx/pegasusx/apps/backend-go/pulse"
	"github.com/pegasusx/pegasusx/apps/backend-go/promotion"
	"github.com/pegasusx/pegasusx/apps/backend-go/replenishment"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailer"
	"github.com/pegasusx/pegasusx/apps/backend-go/returns"
	"github.com/pegasusx/pegasusx/apps/backend-go/routing"
	"github.com/pegasusx/pegasusx/apps/backend-go/seed"
	"github.com/pegasusx/pegasusx/apps/backend-go/storage"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouse"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
	"github.com/pegasusx/pegasusx/packages/handoff"
	"google.golang.org/api/iterator"
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

	RedisAddr       string
	RedisPassword   string
	RedisPoolSize   int
	RedisMaxRetries int
	RedisTLSEnabled bool

	KafkaBrokers            string
	KafkaTopicMain          string
	KafkaTopicMainDLQ       string
	WebSocketAllowedOrigins []string

	JWTSecret string
	JWTIssuer string

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
	DeliveryZoneCenterLat  float64
	DeliveryZoneCenterLng  float64
	DeliveryZoneRadiusKm   float64
	DeliveryZoneResolution int

	LogLevel  string
	LogFormat string

	RequireInfraAdapters bool
	ReliabilityEnabled   bool
	AllowAuthBypass      bool
	MaxSuppliers         int

	OptimizerBaseURL string
	RoutingOSRMURL   string
	InternalAPIKey   string
	GCSBucketName    string
	GoogleMapsAPIKey string
}

// App holds every long-lived singleton. Wire new app-wide dependencies here,
// never as package-level globals.
type App struct {
	Config                 *Config
	Cache                  *cache.Cache
	Idempotency            idempotency.Store
	Supplier               seed.Supplier
	CatalogService         *catalog.Service
	PromotionService       *promotion.Service
	PromotionAudience      *promotion.AudienceResolver
	InventoryService       *inventory.Service
	NotificationService    *notifications.Service
	NotificationInbox      *notifications.InboxHandlers
	SupplierService        *supplier.Service
	RetailerService        *retailer.Service
	RetailerProximity      *retailer.RetailerProximityService
	DriverService          *driver.Service
	FactoryService         *factory.Service
	PayloadService         *payload.Service
	PaymentService         *payment.Service
	WebhookInbox           *payment.WebhookInboxStore
	WarehouseService       *warehouse.Service
	ReturnsService         *returns.Service
	OrderService           *order.Service
	HandoffEngine          *handoff.Engine
	DriverLocations        telemetry.LastLocationStore
	RetailerHub            *ws.Hub
	SupplierHub            *ws.Hub
	DriverHub              *ws.Hub
	PayloadHub             *ws.Hub
	WarehouseHub           *ws.Hub
	FactoryHub             *ws.Hub
	TelemetryHub           *ws.Hub
	OutboxRelay            *outbox.Relay
	NotificationConsumer   *kafka.Consumer
	OrderEventConsumer     *kafka.Consumer
	WarehouseEventConsumer *kafka.Consumer
	Reliability            *ReliabilityMiddleware
	InfraHealth            infraroutes.Deps
	OutboundCircuits       *OutboundCircuits
	PlatformService        *platform.Service
	PlatformHandler        *platform.Handler
	PulseHandlers          *pulse.Handlers
	PushBridge             *notifications.PushBridge
	Spanner                *spanner.Client
	OptimizerClient        *optimizerclient.Client
	DispatchPlanCounters   *plan.SourceCounters
	ReplenishmentEngine    *replenishment.Engine
	cleanup                []func()
	// Spanner *spanner.Client (added when the Spanner client lands)
	// Kafka   *kafka.SyncWriter
	// Outbox  *outbox.Relay
}

// LoadConfig reads environment-backed configuration with safe defaults for
// local development.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		HTTPPort:                        envOr("HTTP_PORT", "8080"),
		WorkerHTTPPort:                  envOr("WORKER_HTTP_PORT", "8081"),
		RunMode:                         envOr("PEGASUSX_RUN_MODE", RunModeAll),
		SpannerEmulatorHost:             envOr("SPANNER_EMULATOR_HOST", "localhost:9010"),
		SpannerProject:                  envOr("SPANNER_PROJECT", "pegasusx-local"),
		SpannerInstance:                 envOr("SPANNER_INSTANCE", "pegasusx-instance"),
		SpannerDatabase:                 envOr("SPANNER_DATABASE", "pegasusx-db"),
		RedisAddr:                       envOr("REDIS_ADDR", "localhost:6379"),
		RedisPassword:                   envOr("REDIS_PASSWORD", ""),
		RedisPoolSize:                   envInt("REDIS_POOL_SIZE", 50),
		RedisMaxRetries:                 envInt("REDIS_MAX_RETRIES", 3),
		RedisTLSEnabled:                 envBool("REDIS_TLS_ENABLED", false),
		KafkaBrokers:                    envOr("KAFKA_BROKERS", "localhost:9092"),
		KafkaTopicMain:                  envOr("KAFKA_TOPIC_MAIN", "pegasusx-main"),
		KafkaTopicMainDLQ:               envOr("KAFKA_TOPIC_MAIN_DLQ", ""),
		WebSocketAllowedOrigins:         splitAndTrimCSV(envOr("WS_ALLOWED_ORIGINS", "")),
		JWTSecret:                       envOr("JWT_SECRET", "dev-only-change-me"),
		JWTIssuer:                       envOr("JWT_ISSUER", "pegasusx-dev"),
		FirebaseAuthEnabled:             envBool("FIREBASE_AUTH_ENABLED", false),
		FirebaseProjectID:               envOr("FIREBASE_PROJECT_ID", ""),
		FirebaseCertsURL:                envOr("FIREBASE_CERTS_URL", "https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com"),
		FirebaseCredentialsPath:         envOr("FIREBASE_CREDENTIALS_PATH", ""),
		GlobalPayEnv:                    envOr("GLOBAL_PAY_ENV", "dev"),
		GlobalPayServiceID:              envOr("GLOBAL_PAY_SERVICE_ID", "doc-supplier-service"),
		GlobalPayUsername:               envOr("GLOBAL_PAY_USERNAME", "doc-username"),
		GlobalPayPassword:               envOr("GLOBAL_PAY_PASSWORD", "doc-password"),
		GlobalPayWebhookSecret:          envOr("GLOBAL_PAY_WEBHOOK_SECRET", "dev-global-pay-secret"),
		AdyenWebhookSecret:              envOr("ADYEN_WEBHOOK_SECRET", "dev-adyen-secret"),
		StripeWebhookSecret:             envOr("STRIPE_WEBHOOK_SECRET", "dev-stripe-secret"),
		PaymeWebhookSecret:              envOr("PAYME_WEBHOOK_SECRET", "dev-payme-secret"),
		ClickWebhookSecret:              envOr("CLICK_WEBHOOK_SECRET", "dev-click-secret"),
		AirwallexDirectExecutionEnabled: envBool("AIRWALLEX_DIRECT_EXECUTION_ENABLED", false),
		SeedSupplierName:                envOr("SEED_SUPPLIER_NAME", "pegasusX Supplier"),
		SeedSupplierCountry:             envOr("SEED_SUPPLIER_COUNTRY", "UZ"),
		SeedSupplierCurrency:            envOr("SEED_SUPPLIER_CURRENCY", "UZS"),
		DeliveryZoneCenterLat:           envFloat("DELIVERY_ZONE_CENTER_LAT", defaultDeliveryZoneCenterLat),
		DeliveryZoneCenterLng:           envFloat("DELIVERY_ZONE_CENTER_LNG", defaultDeliveryZoneCenterLng),
		DeliveryZoneRadiusKm:            envFloat("DELIVERY_ZONE_RADIUS_KM", defaultDeliveryZoneRadiusKm),
		DeliveryZoneResolution:          envInt("DELIVERY_ZONE_RESOLUTION", retailer.DefaultPerimeterResolution),
		LogLevel:                        envOr("LOG_LEVEL", "info"),
		LogFormat:                       envOr("LOG_FORMAT", "json"),
		RequireInfraAdapters:            envBool("REQUIRE_INFRA_ADAPTERS", true),
		ReliabilityEnabled:              envBool("RELIABILITY_MIDDLEWARE_ENABLED", true),
		AllowAuthBypass:                 envBool("ALLOW_AUTH_BYPASS", false),
		MaxSuppliers:                    envInt("MAX_SUPPLIERS", 10),
		OptimizerBaseURL:                envOr("OPTIMIZER_BASE_URL", "http://localhost:8081"),
		RoutingOSRMURL:                  envOr("ROUTING_OSRM_URL", ""),
		InternalAPIKey:                  envOr("INTERNAL_API_KEY", "dev-internal-key"),
		GCSBucketName:                   envOr("GCS_BUCKET_NAME", ""),
		GoogleMapsAPIKey:                envOr("GOOGLE_MAPS_API_KEY", envOr("GOOGLE_PLACES_API_KEY", "")),
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET required")
	}
	if strings.TrimSpace(cfg.KafkaTopicMainDLQ) == "" && strings.TrimSpace(cfg.KafkaTopicMain) != "" {
		cfg.KafkaTopicMainDLQ = strings.TrimSpace(cfg.KafkaTopicMain) + "-dlq"
	}
	if err := cfg.ValidateProductionProfile(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// NewApp constructs the composition root. New singletons attach here.
//
// Scaffold note: the cache + idempotency + retailer-repo bindings are
// in-memory while the Spanner / Redis / Kafka clients are still being wired.
// Swap each binding inside this function when the production client lands —
// downstream packages depend only on the interfaces, not the implementations.
func NewApp(ctx context.Context, cfg *Config) (*App, error) {
	log := slog.Default()
	outboundCircuits := NewOutboundCircuits()
	ws.SetAllowedOrigins(cfg.WebSocketAllowedOrigins)
	if cfg.RequireInfraAdapters {
		log.Info("strict infra adapter mode enabled")
	}

	if err := storage.InitGCS(ctx, cfg.GCSBucketName); err != nil {
		log.Warn("gcs init failed; catalog image uploads use placeholders", "err", err)
	}

	cacheBackend := cache.Backend(cache.NewInMemoryBackend())
	redisEnabled := false
	var redisAdapter redisRuntimeAdapter
	redisCfg := cache.RedisConfig{
		Addr:            cfg.RedisAddr,
		Password:        cfg.RedisPassword,
		PoolSize:        cfg.RedisPoolSize,
		MaxRetries:      cfg.RedisMaxRetries,
		TLSEnabled:      cfg.RedisTLSEnabled,
		MinIdleConns:    10,
		MaxIdleTime:     5 * time.Minute,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
	}
	if redisBackend, err := newRedisRuntimeAdapter(redisCfg); err != nil {
		log.Warn("redis backend init failed; using in-memory cache",
			"addr", cfg.RedisAddr,
			"err", err,
		)
	} else if err := redisBackend.Ping(ctx); err != nil {
		log.Warn("redis ping failed; using in-memory cache",
			"addr", cfg.RedisAddr,
			"err", err,
		)
		_ = redisBackend.Close()
	} else {
		redisAdapter = redisBackend
		cbBackend := cache.NewCircuitBreakerBackend(redisBackend, cacheBackend)
		cacheBackend = cbBackend
		redisEnabled = true
		log.Info("redis cache backend enabled", "addr", cfg.RedisAddr, "pool_size", cfg.RedisPoolSize, "tls", cfg.RedisTLSEnabled)
	}
	if cfg.RequireInfraAdapters && !redisEnabled {
		return nil, fmt.Errorf("require infra adapters: redis unavailable")
	}
	cacheClient := cache.New(cacheBackend, log)
	driverLocations := telemetry.NewCacheLastLocationStore(cacheClient, telemetry.DefaultLastLocationTTL)
	var idemStore idempotency.Store = idempotency.NewInMemoryStore()
	if redisAdapter != nil {
		if client := redisAdapter.Client(); client != nil {
			idemStore = idempotency.NewRedisStore(client)
			log.Info("idempotency redis store enabled", "addr", cfg.RedisAddr)
		}
	}
	if cfg.RequireInfraAdapters && redisEnabled {
		if _, ok := idemStore.(*idempotency.InMemoryStore); ok {
			return nil, fmt.Errorf("require infra adapters: idempotency redis store unavailable")
		}
	}
	memoryOutboxStore := memory.NewOutboxStore()
	relayStore := outbox.Store(memoryOutboxStore)
	outboxAppender := outboxEventAppender(memoryOutboxStore)
	spannerOutboxEnabled := false
	var spannerClient *spanner.Client
	var manifestStore *manifest.Store
	var routeGeometryBuilder *routing.GeometryBuilder
	outboxPublisher := outbox.Publisher(&loggingOutboxPublisher{log: log})
	cleanup := make([]func(), 0, 3)
	if client, store, err := tryNewSpannerOutboxStore(ctx, cfg); err != nil {
		log.Warn("spanner outbox store unavailable; using in-memory outbox store",
			"database", spannerDatabasePath(cfg),
			"err", err,
		)
	} else {
		spannerClient = client
		relayStore = store
		outboxAppender = store
		spannerOutboxEnabled = true
		cleanup = append(cleanup, func() {
			client.Close()
		})
		log.Info("spanner outbox store enabled", "database", spannerDatabasePath(cfg))
		manifestStore = manifest.NewStore(spannerClient)
		osrmClient := routing.NewOSRMClient(cfg.RoutingOSRMURL, outboundCircuits.OSRM)
		routeGeometryBuilder = routing.NewGeometryBuilder(osrmClient)
		manifestStore.SetGeometryBuilder(routeGeometryBuilder)
		if osrmClient != nil {
			log.Info("OSRM routing enabled", "base_url", cfg.RoutingOSRMURL)
		}
	}
	kafkaEnabled := false
	if kafkaPublisher, err := newKafkaRuntimePublisher(cfg.KafkaBrokers, outbox.KafkaPublisherConfig{}); err != nil {
		log.Warn("kafka publisher init failed; using logging publisher",
			"brokers", cfg.KafkaBrokers,
			"err", err,
		)
	} else {
		outboxPublisher = kafkaPublisher
		kafkaEnabled = true
		cleanup = append(cleanup, func() {
			if err := kafkaPublisher.Close(); err != nil {
				log.Warn("kafka publisher close failed", "err", err)
			}
		})
		log.Info("kafka outbox publisher enabled", "brokers", cfg.KafkaBrokers)
	}
	if cfg.RequireInfraAdapters && !kafkaEnabled {
		return nil, fmt.Errorf("require infra adapters: kafka unavailable")
	}
	if cfg.RequireInfraAdapters && spannerClient == nil && !cfg.TestingMode {
		return nil, fmt.Errorf("require infra adapters: spanner unavailable")
	}
	outboxRelay := outbox.NewRelay(relayStore, outboxPublisher, outbox.RelayConfig{}, log)

	var seedRepo seed.Repository
	if spannerClient != nil {
		seedRepo = &runtimeSeedRepository{client: spannerClient}
	}
	supplierSeed, err := seed.EnsureSupplier(ctx, seedRepo,
		cfg.SeedSupplierName, cfg.SeedSupplierCountry, cfg.SeedSupplierCurrency, log)
	if err != nil {
		return nil, fmt.Errorf("seed supplier: %w", err)
	}

	var retailerRepo retailer.Repository
	var supplierRepo supplier.Repository

	if spannerClient != nil {
		retailerRepo = retailer.NewSpannerRepository(spannerClient, supplierSeed.SupplierID)
		supplierRepo = supplier.NewSpannerRepository(spannerClient)
		log.Info("retailer and supplier repositories enabled", "backend", "spanner")
	} else {
		retailerRepo = memory.NewRetailerRepo(outboxAppender)
		supplierRepo = memory.NewSupplierRepo(outboxAppender)
		log.Warn("retailer and supplier repository fallback enabled", "backend", "in-memory")
	}

	var perimeterStore retailer.PerimeterSetStore
	if store, ok := cacheBackend.(retailer.PerimeterSetStore); ok {
		perimeterStore = store
	}

	retailerProximity := retailer.NewRetailerProximityService(perimeterStore, retailer.RetailerProximityConfig{
		Resolution: cfg.DeliveryZoneResolution,
		Log:        log,
	})

	zoneSeed := resolveDeliveryZoneSeed(cfg)
	if profile, found, profileErr := supplierRepo.GetProfile(ctx, supplierSeed.SupplierID); profileErr != nil {
		log.Warn("load supplier profile for delivery-zone precompute failed", "err", profileErr)
	} else if found {
		zoneSeed = zoneSeed.withSupplierProfile(profile)
	}
	if snapshot, zoneErr := retailerProximity.PrecomputeDeliveryZoneForCenter(ctx, zoneSeed.CenterLat, zoneSeed.CenterLng, zoneSeed.RadiusKm); zoneErr != nil {
		log.Warn("delivery perimeter precompute skipped",
			"err", zoneErr,
			"center_lat", zoneSeed.CenterLat,
			"center_lng", zoneSeed.CenterLng,
			"radius_km", zoneSeed.RadiusKm,
		)
	} else {
		log.Info("delivery perimeter ready",
			"cells", snapshot.Cells,
			"compacted_cells", snapshot.CompactedCells,
			"resolution", snapshot.Resolution,
		)
	}

	var promotionSvc *promotion.Service
	var promotionAudience *promotion.AudienceResolver
	if spannerClient != nil {
		promoRepo := promotion.NewSpannerRepository(spannerClient)
		promotionSvc = promotion.NewService(promoRepo, cacheClient, idemStore, log)
		promotionAudience = promotion.NewAudienceResolver(spannerClient)
		log.Info("promotion service enabled", "backend", "spanner")
	}

	var catalogSvc *catalog.Service
	if spannerClient != nil {
		catalogRepo := catalog.NewSpannerRepository(spannerClient)
		catalogSvc = catalog.NewService(catalogRepo, cacheClient, log, promotionSvc, catalog.NewStockEnricher(spannerClient))
		log.Info("catalog service enabled", "backend", "spanner")
	}

	var inventorySvc *inventory.Service
	if spannerClient != nil {
		inventoryRepo := inventory.NewSpannerRepository(spannerClient)
		inventorySvc = inventory.NewService(inventoryRepo, cacheClient, log)
		log.Info("inventory service enabled", "backend", "spanner")
	}

	var cartRepo retailer.CartRepository
	if spannerClient != nil {
		cartRepo = retailer.NewSpannerCartRepository(spannerClient)
	}

	var notifSvc *notifications.Service
	var notifInbox *notifications.InboxHandlers
	var notifAdapter *notificationReaderAdapter
	if spannerClient != nil {
		notifRepo := notifications.NewSpannerRepository(spannerClient)
		notifSvc = notifications.NewService(notifRepo, cacheClient, log)
		notifInbox = &notifications.InboxHandlers{Service: notifSvc, Log: log}
		notifAdapter = &notificationReaderAdapter{svc: notifSvc}
		log.Info("notification service enabled", "backend", "spanner")
	}

	retailerSvc := retailer.NewService(retailer.ServiceConfig{
		Repo:        retailerRepo,
		CartRepo:    cartRepo,
		NotifSvc:    notifAdapter,
		Cache:       cacheClient,
		Idem:        idemStore,
		Locations:   driverLocations,
		Proximity:   retailerProximity,
		SupplierID:  supplierSeed.SupplierID,
		CountryCode: cfg.SeedSupplierCountry,
		JWTSecret:   cfg.JWTSecret,
		JWTIssuer:   cfg.JWTIssuer,
		Log:         log,
	})

	var supplierInventory supplier.InventoryServicer
	if inventorySvc != nil {
		supplierInventory = &inventoryAdapter{svc: inventorySvc}
	}
	var dashboardQuery supplier.DashboardCountQuery
	if spannerClient != nil {
		dashboardQuery = supplierDashboardCountQuery(spannerClient)
	}
	supplierSvc := supplier.NewService(supplier.ServiceConfig{
		Repo:             supplierRepo,
		Cache:            cacheClient,
		Idem:             idemStore,
		Locations:        driverLocations,
		InventoryService: supplierInventory,
		DashboardQuery:   dashboardQuery,
		SupplierID:       supplierSeed.SupplierID,
		SeedSupplierID:   supplierSeed.SupplierID,
		MaxSuppliers:     cfg.MaxSuppliers,
		Country:          cfg.SeedSupplierCountry,
		Currency:         cfg.SeedSupplierCurrency,
		JWTSecret:        cfg.JWTSecret,
		JWTIssuer:        cfg.JWTIssuer,
		JWTTTL:           24 * time.Hour,
		CookieSecure:     false,
		Log:              log,
	})

	// Role hubs share the cache backend for Pub/Sub fan-out. Production swaps
	// the backend for a real Redis client; the seam is the same.
	retailerHub := ws.NewHub("retailer", cacheBackend, log)
	supplierHub := ws.NewHub("supplier", cacheBackend, log)
	driverHub := ws.NewHub("driver", cacheBackend, log)
	payloadHub := ws.NewHub("payload", cacheBackend, log)
	warehouseHub := ws.NewHub("warehouse", cacheBackend, log)
	factoryHub := ws.NewHub("factory", cacheBackend, log)
	telemetryHub := ws.NewHub("telemetry", cacheBackend, log)
	if promotionSvc != nil {
		promotionSvc.BindRetailerHub(retailerHub)
	}

	var orderRepo order.Repository
	var orderWarehouseResolver order.WarehouseResolver
	if spannerClient != nil {
		orderRepo = order.NewSpannerRepository(spannerClient)
		orderWarehouseResolver = order.NewSpannerWarehouseResolver(spannerClient)
		log.Info("order repository enabled", "backend", "spanner")
	} else {
		orderRepo = memory.NewOrderRepo(outboxAppender, &memory.RetailerReceivingWindowAdapter{Repo: retailerRepo})
		log.Warn("order repository fallback enabled", "backend", "in-memory")
	}
	var paymentRepo payment.Repository
	if spannerClient != nil {
		paymentRepo = payment.NewSpannerRepository(spannerClient)
		log.Info("payment repository enabled", "backend", "spanner")
	} else {
		paymentRepo = memory.NewPaymentRepo(outboxAppender)
		log.Warn("payment repository fallback enabled", "backend", "in-memory")
	}
	supplierSvc.SetEarningsLookup(func(ctx context.Context, supplierID, currency string, now time.Time) (supplier.SupplierEarningsResponse, error) {
		return loadSupplierEarningsAuthority(ctx, paymentRepo, supplierID, currency, now)
	})
	handoffEngine := handoff.FromEnv()
	var gatewayPolicyReader *payment.SpannerPolicyResolver
	if spannerClient != nil {
		gatewayPolicyReader = payment.NewSpannerPolicyResolver(spannerClient)
	}
	orderSvc := order.NewService(order.ServiceConfig{
		Repo:            orderRepo,
		Cache:           cacheClient,
		Warehouse:       orderWarehouseResolver,
		Promotions:      promotionSvc,
		SupplierID:      supplierSeed.SupplierID,
		SupplierName:    cfg.SeedSupplierName,
		Currency:        cfg.SeedSupplierCurrency,
		RetailerHub:     retailerHub,
		SupplierHub:     supplierHub,
		DriverHub:       driverHub,
		SpannerClient:   spannerClient,
		ShopClosedGrace: shopClosedGraceDuration(),
		Log:             log,
		JWTSecret:       cfg.JWTSecret,
		Handoff:         handoffEngine,
		Idem:            idemStore,
	})
	orderSvc.SetManifestStore(manifestStore)
	if gatewayPolicyReader != nil {
		orderSvc.SetGatewayPolicyReader(gatewayPolicyReader)
	}
	order.StartPreorderSweeper(orderSvc)
	go orderSvc.StartNegotiationSweeper(context.Background())
	go orderSvc.StartDeferredPaymentSweeper(context.Background(), 5*time.Minute)
	if spannerClient != nil {
		if n, err := orderSvc.BackfillScheduledReservations(context.Background(), 500); err != nil {
			log.Warn("scheduled reservation backfill failed", "err", err)
		} else if n > 0 {
			log.Info("scheduled reservation backfill complete", "orders", n)
		}
	}
	var optimizerCli *optimizerclient.Client
	if strings.TrimSpace(cfg.OptimizerBaseURL) != "" && strings.TrimSpace(cfg.InternalAPIKey) != "" {
		optimizerCli = optimizerclient.New(cfg.OptimizerBaseURL, cfg.InternalAPIKey)
		log.Info("dispatch optimiser client enabled", "base_url", cfg.OptimizerBaseURL)
	}
	dispatchCounters := &plan.SourceCounters{}

	var replenishmentEngine *replenishment.Engine
	if spannerClient != nil {
		replenishmentEngine = replenishment.NewEngine(spannerClient, log)
	}

	supplierSvc.SetPortalOps(supplier.PortalOpsConfig{
		Spanner:              spannerClient,
		ManifestStore:        manifestStore,
		RouteGeometryBuilder: routeGeometryBuilder,
		SupplierHub:          supplierHub,
		WarehouseHub:         warehouseHub,
		DriverHub:            driverHub,
		RetailerHub:          retailerHub,
		PayloadHub:           payloadHub,
		FactoryHub:           factoryHub,
		OptimizerClient:      optimizerCli,
		PlanCounters:         dispatchCounters,
		FallbackDepotLat:     cfg.DeliveryZoneCenterLat,
		FallbackDepotLng:     cfg.DeliveryZoneCenterLng,
		ReplenishmentEngine:  replenishmentEngine,
	})
	retailerSvc.SetOrderLifecycle(orderSvc)

	factoryNodeID := strings.TrimSpace(os.Getenv("FACTORY_DEMO_ID"))
	if factoryNodeID == "" {
		factoryNodeID = "factory-demo-1"
	}
	warehouseNodeID := strings.TrimSpace(os.Getenv("WAREHOUSE_DEMO_ID"))
	if warehouseNodeID == "" {
		warehouseNodeID = strings.TrimSpace(os.Getenv("SSMR_SMOKE_WAREHOUSE_ID"))
	}
	if warehouseNodeID == "" {
		warehouseNodeID = "wh-demo-1"
	}

	var driverRepo driver.Repository
	var factoryRepo factory.Repository
	var payloadRepo payload.Repository
	if spannerClient != nil {
		driverRepo = driver.NewSpannerRepository(spannerClient)
		factoryRepo = factory.NewSpannerRepository(spannerClient, supplierSeed.SupplierID, factoryNodeID)
		payloadRepo = payload.NewSpannerRepository(spannerClient, supplierSeed.SupplierID, warehouseNodeID)
		log.Info("factory and payload repositories enabled", "backend", "spanner")
	} else {
		driverRepo = driver.NewInMemoryRepository()
		factoryRepo = factory.NewInMemoryRepository()
		payloadRepo = payload.NewInMemoryRepository()
		log.Warn("driver/factory/payload repository fallback enabled", "backend", "in-memory")
	}
	var driverAvailReader driver.AvailabilityReader
	if spannerClient != nil {
		driverAvailReader = driverAvailabilityReader(spannerClient)
	}

	factorySvc := factory.NewService(factory.ServiceConfig{
		Repo:          factoryRepo,
		Cache:         cacheClient,
		SupplierHub:   supplierHub,
		FactoryHub:    factoryHub,
		Log:           log,
		Spanner:       spannerClient,
		SupplierID:    supplierSeed.SupplierID,
		FactoryNodeID: factoryNodeID,
		Currency:      cfg.SeedSupplierCurrency,
		JWTSecret:     cfg.JWTSecret,
		JWTIssuer:     cfg.JWTIssuer,
		Idem:          idemStore,
	})
	payloadSvc := payload.NewService(payload.ServiceConfig{
		Repo:          payloadRepo,
		Cache:         cacheClient,
		SupplierHub:   supplierHub,
		PayloadHub:    payloadHub,
		DriverHub:     driverHub,
		NotifSvc:      notifSvc,
		Log:           log,
		SupplierID:    supplierSeed.SupplierID,
		Currency:      cfg.SeedSupplierCurrency,
		JWTSecret:     cfg.JWTSecret,
		JWTIssuer:     cfg.JWTIssuer,
		ManifestStore: manifest.NewStore(spannerClient),
		Idem:          idemStore,
	})
	payloadSvc.SetPortalManifestLister(&supplier.ManifestLister{Service: supplierSvc})
	payloadSvc.SetOrderExpectationReader(orderRepo)
	payloadSvc.WarmManifestCache(ctx)
	factorySvc.WarmManifestCache(ctx)
	returnsSvc := returns.NewService(returns.ServiceConfig{
		Spanner:      spannerClient,
		Cache:        cacheClient,
		PayloadHub:   payloadHub,
		WarehouseHub: warehouseHub,
		SupplierHub:  supplierHub,
		Log:          log,
		Idem:         idemStore,
	})
	var driverOrderList driver.DriverOrderQuery
	var driverOrderGet driver.DriverOrderGetQuery
	var driverProfileLookup driver.DriverProfileLookup
	var driverRouteGeometry driver.RouteGeometryLookup
	var driverDepart driver.DepartFn
	var driverReturnComplete driver.ReturnCompleteFn
	if spannerClient != nil {
		driverOrderList = driverOrderListQuery(spannerClient)
		driverOrderGet = driverOrderGetQuery(spannerClient)
		driverProfileLookup = driverProfileLookupQuery(spannerClient)
		driverRouteGeometry = driverRouteGeometryQuery(spannerClient, routeGeometryBuilder)
		driverDepart = func(ctx context.Context, driverID string) (driver.DepartResult, bool, error) {
			departed, ok, err := manifestStore.DepartDriver(ctx, driverID, time.Now().UTC())
			if err != nil || !ok {
				return driver.DepartResult{}, ok, err
			}
			return driver.DepartResult{
				ManifestID: departed.ManifestID,
				OrderIDs:   departed.OrderIDs,
				Count:      len(departed.OrderIDs),
			}, true, nil
		}
		driverReturnComplete = func(ctx context.Context, driverID string) (driver.ReturnCompleteResult, bool, error) {
			returned, ok, err := manifestStore.ReturnDriver(ctx, driverID, time.Now().UTC())
			if err != nil || !ok {
				return driver.ReturnCompleteResult{}, ok, err
			}
			if returnsSvc != nil {
				if hookErr := returnsSvc.OnDriverReturnComplete(ctx, returned); hookErr != nil {
					log.WarnContext(ctx, "return-complete physical hook failed", "err", hookErr, "driver_id", driverID)
				}
			}
			return driver.ReturnCompleteResult{
				ManifestID: returned.ManifestID,
				OrderIDs:   returned.OrderIDs,
				Count:      len(returned.OrderIDs),
			}, true, nil
		}
	}
	driverSvc := driver.NewService(driver.ServiceConfig{
		Repo:           driverRepo,
		Cache:          cacheClient,
		NotifSvc:       notifAdapter,
		OrderList:      driverOrderList,
		OrderGet:       driverOrderGet,
		ProfileLookup:  driverProfileLookup,
		AvailabilityReader: driverAvailReader,
		RouteGeometry:  driverRouteGeometry,
		Depart:         driverDepart,
		ReturnComplete: driverReturnComplete,
		ManifestTokens: func(ctx context.Context, orderIDs []string) map[string]string {
			tokens := make(map[string]string, len(orderIDs))
			for _, orderID := range orderIDs {
				token, err := orderSvc.ResolveDeliveryToken(ctx, orderID)
				if err == nil && strings.TrimSpace(token) != "" {
					tokens[orderID] = token
				}
			}
			return tokens
		},
		SupplierHub: supplierHub,
		DriverHub:   driverHub,
		Log:         log,
		ManifestGate: func(manifestID string) (driver.ManifestGateResult, bool) {
			if state, stopCount, totalVolumeVU, ok := payloadSvc.ManifestGateSnapshot(manifestID); ok {
				return driver.ManifestGateResult{
					ManifestID: manifestID,
					State:      state,
					StopCount:  stopCount,
					VolumeVU:   totalVolumeVU,
				}, true
			}
			state, stopCount, totalVolumeVU, ok := factorySvc.ManifestGateSnapshot(manifestID)
			if !ok {
				return driver.ManifestGateResult{}, false
			}
			return driver.ManifestGateResult{
				ManifestID: manifestID,
				State:      state,
				StopCount:  stopCount,
				VolumeVU:   totalVolumeVU,
			}, true
		},
		Manifest: func(driverID, manifestID, date string) (factory.ManifestDetailSnapshot, bool) {
			if snap, ok := payloadSvc.ManifestDetailSnapshotForDriver(driverID, manifestID, date); ok {
				return snap, true
			}
			return factorySvc.ManifestDetailSnapshotForDriver(driverID, manifestID, date)
		},
		SupplierID: supplierSeed.SupplierID,
		Currency:   cfg.SeedSupplierCurrency,
		JWTSecret:  cfg.JWTSecret,
		JWTIssuer:  cfg.JWTIssuer,
		Idem:       idemStore,
	})
	paymentExec := payment.NewProviderExecutionRouter(payment.ProviderExecutionRouterConfig{
		AirwallexDirectExecutionEnabled: cfg.AirwallexDirectExecutionEnabled,
		GlobalPayEnv:                    cfg.GlobalPayEnv,
		GlobalPayServiceID:              cfg.GlobalPayServiceID,
		GlobalPayUsername:               cfg.GlobalPayUsername,
		GlobalPayPassword:               cfg.GlobalPayPassword,
		PaymentBreaker:                  outboundCircuits.Payment,
	})
	var webhookInbox *payment.WebhookInboxStore
	if spannerClient != nil {
		webhookInbox = payment.NewWebhookInboxStore(spannerClient)
	}
	paymentSvc := payment.NewService(payment.ServiceConfig{
		Repo:                            paymentRepo,
		Cache:                           cacheClient,
		Idem:                            idemStore,
		SupplierID:                      supplierSeed.SupplierID,
		Currency:                        cfg.SeedSupplierCurrency,
		Execution:                       paymentExec,
		GlobalPayEnv:                    cfg.GlobalPayEnv,
		GlobalPayServiceID:              cfg.GlobalPayServiceID,
		GlobalPayUsername:               cfg.GlobalPayUsername,
		GlobalPayPassword:               cfg.GlobalPayPassword,
		GlobalPayWebhookSecret:          cfg.GlobalPayWebhookSecret,
		AdyenWebhookSecret:              cfg.AdyenWebhookSecret,
		StripeWebhookSecret:             cfg.StripeWebhookSecret,
		PaymeWebhookSecret:              cfg.PaymeWebhookSecret,
		ClickWebhookSecret:              cfg.ClickWebhookSecret,
		WebhookInbox:                    webhookInbox,
		AirwallexDirectExecutionEnabled: cfg.AirwallexDirectExecutionEnabled,
		Log:                             log,
		Policy:                          gatewayPolicyReader,
	})
	paymentSvc.BindCartCheckout(orderSvc)
	paymentSvc.BindCheckoutPreview(orderSvc)
	paymentSvc.BindOrderCheckoutReader(orderSvc)
	orderSvc.SetPaymentCapturer(paymentSvc)
	var warehouseRepo warehouse.Repository
	if spannerClient != nil {
		warehouseRepo = warehouse.NewSpannerRepository(spannerClient)
	} else {
		warehouseRepo = warehouse.NewInMemoryRepository()
	}

	var warehouseAnalytics warehouse.WarehouseAnalyticsQuery
	var whOpsOrders warehouse.WarehouseOpsOrdersQuery
	var whOpsDrivers warehouse.WarehouseOpsDriversQuery
	var whOpsVehicles warehouse.WarehouseOpsVehiclesQuery
	if spannerClient != nil {
		warehouseAnalytics = warehouseAnalyticsCountQuery(spannerClient)
		whOpsOrders = warehouseOpsOrdersQuery(spannerClient)
		whOpsDrivers = warehouseOpsDriversQuery(spannerClient)
		whOpsVehicles = warehouseOpsVehiclesQuery(spannerClient)
	}
	warehouseSvc := warehouse.NewService(warehouse.ServiceConfig{
		Repo:                 warehouseRepo,
		Planner:              orderSvc,
		AnalyticsQuery:       warehouseAnalytics,
		OpsOrders:            whOpsOrders,
		OpsDrivers:           whOpsDrivers,
		OpsVehicles:          whOpsVehicles,
		Cache:                cacheClient,
		Idem:                 idemStore,
		Spanner:              spannerClient,
		ManifestStore:        manifestStore,
		RouteGeometryBuilder: routeGeometryBuilder,
		Locations:            driverLocations,
		SupplierHub:          supplierHub,
		WarehouseHub:         warehouseHub,
		DriverHub:            driverHub,
		RetailerHub:          retailerHub,
		Log:                  log,
		SupplierID:           supplierSeed.SupplierID,
		Currency:             cfg.SeedSupplierCurrency,
		JWTSecret:            cfg.JWTSecret,
		JWTIssuer:            cfg.JWTIssuer,
		OptimizerClient:      optimizerCli,
		PlanCounters:         dispatchCounters,
		FallbackDepotLat:     cfg.DeliveryZoneCenterLat,
		FallbackDepotLng:     cfg.DeliveryZoneCenterLng,
	})
	warehouseSvc.SetOrderStockReader(orderSvc)
	driverSvc.SetDispatchPlanInvalidate(func(ctx context.Context, warehouseID string) {
		warehouseSvc.InvalidateDispatchPlanCache(ctx, warehouseID)
	})
	orderSvc.SetDispatchPlanWarm(func(ctx context.Context, warehouseID string) {
		warehouseSvc.InvalidateDispatchPlanCache(ctx, warehouseID)
	})
	driverSvc.SetFleetAvailabilityBroadcaster(func(ctx context.Context, warehouseID string, payload map[string]any) {
		warehouseSvc.BroadcastFleetEvent(ctx, warehouseID, payload)
	})

	var reliabilityMiddleware *ReliabilityMiddleware
	if cfg.ReliabilityEnabled {
		reliabilityMiddleware = NewReliabilityMiddleware(ReliabilityConfigFromEnv())
		if redisEnabled {
			if rb, ok := cacheBackend.(*cache.RedisBackend); ok {
				if client := rb.Client(); client != nil {
					reliabilityMiddleware.SetRedisRateLimiter(client)
					log.Info("reliability redis rate limiter enabled")
				}
			}
		}
		log.Info("reliability middleware enabled")
	}
	infraHealth := buildInfraHealthChecks(redisEnabled, cacheBackend, spannerClient)

	policyRepo := platform.PolicyRepository(platform.NewMemoryPolicyRepository())
	tokenRepo := platform.DeviceTokenRepository(platform.NewMemoryDeviceTokenRepository())
	sessionChecker := platform.SessionChecker(platform.NoopSessionChecker{})
	if spannerClient != nil {
		policyRepo = platform.NewSpannerPolicyRepository(spannerClient)
		tokenRepo = platform.NewSpannerDeviceTokenRepository(spannerClient)
		sessionChecker = platform.NewSpannerSessionChecker(spannerClient)
	}
	platformSvc := platform.NewService(policyRepo, sessionChecker, log)
	platformHandler := platform.NewHandler(platform.HandlerConfig{
		Service:      platformSvc,
		DeviceTokens: tokenRepo,
		Log:          log,
	})

	var fcmClient *notifications.FCMClient
	if strings.TrimSpace(cfg.FirebaseCredentialsPath) != "" {
		fcmClient, err = notifications.InitFCM(cfg.FirebaseCredentialsPath, spannerClient, log)
		if err != nil {
			log.Warn("FCM init failed; using no-op", "err", err)
			fcmClient = notifications.NewNoOpFCMClient(log)
		}
	} else {
		fcmClient = notifications.NewNoOpFCMClient(log)
	}
	pushBridge := notifications.NewPushBridge(fcmClient, tokenRepo, log)

	var notificationConsumer *kafka.Consumer
	var orderEventConsumer *kafka.Consumer
	var warehouseEventConsumer *kafka.Consumer
	if kafkaEnabled && cfg.KafkaTopicMain != "" {
		dlqWriter, err := newKafkaRuntimeDLQWriter(cfg.KafkaBrokers, cfg.KafkaTopicMainDLQ)
		if err != nil {
			if cfg.RequireInfraAdapters {
				return nil, fmt.Errorf("init notification dlq writer: %w", err)
			}
			log.Warn("notification consumer disabled; dlq writer init failed",
				"topic", cfg.KafkaTopicMain,
				"dlq_topic", cfg.KafkaTopicMainDLQ,
				"err", err,
			)
		} else {
			var kafkaEventDedup kafka.EventDedupStore = kafka.NewInMemoryEventDedup(7 * 24 * time.Hour)
			if redisAdapter != nil {
				if rc := redisAdapter.Client(); rc != nil {
					kafkaEventDedup = kafka.NewRedisEventDedup(rc, 7*24*time.Hour)
				}
			}
			const notificationConsumerGroup = "void-notification-dispatcher"
			const orderConsumerGroup = "void-order-mutator"
			const warehouseConsumerGroup = "void-warehouse-mutator"
			dispatcher := kafka.NewNotificationDispatcher(kafka.DispatcherDeps{
				RetailerHub:     retailerHub,
				SupplierHub:     supplierHub,
				DriverHub:       driverHub,
				WarehouseHub:    warehouseHub,
				FactoryHub:      factoryHub,
				PayloadHub:      payloadHub,
				Push:            pushBridge,
				Inbox:           notifSvc,
				EventDedup:      kafkaEventDedup,
				ConsumerGroupID: notificationConsumerGroup,
			})
			dispatcherTopics := events.DispatcherConsumerTopics()
			notificationConsumer = kafka.NewMultiTopicConsumer(kafka.ConsumerDeps{
				Brokers:   strings.Split(cfg.KafkaBrokers, ","),
				GroupID:   notificationConsumerGroup,
				Topics:    dispatcherTopics,
				Handler:   dispatcher.HandleEvent,
				DLQWriter: dlqWriter,
			})
			orderHandler := kafka.WithEventDedup(kafkaEventDedup, orderConsumerGroup, order.NewEventConsumer(orderSvc, log).HandleEvent)
			warehouseHandler := kafka.WithEventDedup(kafkaEventDedup, warehouseConsumerGroup, warehouse.NewEventConsumer(warehouseSvc, log).HandleEvent)
			orderEventConsumer = kafka.NewConsumer(kafka.ConsumerDeps{
				Brokers:   strings.Split(cfg.KafkaBrokers, ","),
				GroupID:   orderConsumerGroup,
				Topic:     events.OrderConsumerTopic(),
				Handler:   orderHandler,
				DLQWriter: dlqWriter,
			})
			warehouseEventConsumer = kafka.NewConsumer(kafka.ConsumerDeps{
				Brokers:   strings.Split(cfg.KafkaBrokers, ","),
				GroupID:   warehouseConsumerGroup,
				Topic:     events.DispatchConsumerTopic(),
				Handler:   warehouseHandler,
				DLQWriter: dlqWriter,
			})
			cleanup = append(cleanup, func() {
				if err := notificationConsumer.Close(); err != nil {
					log.Warn("notification consumer close failed", "err", err)
				}
				if err := orderEventConsumer.Close(); err != nil {
					log.Warn("order event consumer close failed", "err", err)
				}
				if err := warehouseEventConsumer.Close(); err != nil {
					log.Warn("warehouse event consumer close failed", "err", err)
				}
				if err := dlqWriter.Close(); err != nil {
					log.Warn("notification dlq writer close failed", "err", err)
				}
			})
			log.Info("notification consumer enabled",
				"topic", cfg.KafkaTopicMain,
				"order_topic", events.OrderConsumerTopic(),
				"dispatch_topic", events.DispatchConsumerTopic(),
				"dlq_topic", cfg.KafkaTopicMainDLQ,
				"consume_domain", events.ConsumeDomainTopics(),
				"dual_write", events.DualWriteDomainTopics(),
			)
		}
	}

	if spannerClient != nil {
		if err := auth.EnsureDemoScopeLinks(ctx, spannerClient, supplierSeed.SupplierID); err != nil {
			log.Warn("demo scope link seed failed", "err", err)
		}
	}

	_, idempotencyRedis := idemStore.(*idempotency.RedisStore)
	logInfraBackendBanner(log, cfg, infraBackendStatus{
		Spanner:          spannerClient != nil,
		RedisCache:       redisEnabled,
		IdempotencyRedis: idempotencyRedis,
		Kafka:            kafkaEnabled,
		SpannerOutbox:    spannerOutboxEnabled,
	})

	pulseSvc := pulse.NewService(pulse.Config{
		Notifications: notifSvc,
		Spanner:       spannerClient,
		SupplierAct:   &supplierPulseLoader{svc: supplierSvc},
		Log:           log,
	})
	pulseHandlers := &pulse.Handlers{Service: pulseSvc}

	return &App{
		Config:                 cfg,
		Cache:                  cacheClient,
		Idempotency:            idemStore,
		Supplier:               supplierSeed,
		CatalogService:         catalogSvc,
		PromotionService:       promotionSvc,
		PromotionAudience:      promotionAudience,
		InventoryService:       inventorySvc,
		NotificationService:    notifSvc,
		NotificationInbox:      notifInbox,
		SupplierService:        supplierSvc,
		RetailerService:        retailerSvc,
		DriverService:          driverSvc,
		FactoryService:         factorySvc,
		PayloadService:         payloadSvc,
		PaymentService:         paymentSvc,
		WebhookInbox:           webhookInbox,
		WarehouseService:       warehouseSvc,
		ReturnsService:         returnsSvc,
		OrderService:           orderSvc,
		HandoffEngine:          handoffEngine,
		DriverLocations:        driverLocations,
		RetailerHub:            retailerHub,
		SupplierHub:            supplierHub,
		DriverHub:              driverHub,
		PayloadHub:             payloadHub,
		WarehouseHub:           warehouseHub,
		FactoryHub:             factoryHub,
		TelemetryHub:           telemetryHub,
		NotificationConsumer:   notificationConsumer,
		OrderEventConsumer:     orderEventConsumer,
		WarehouseEventConsumer: warehouseEventConsumer,
		OutboxRelay:            outboxRelay,
		Reliability:            reliabilityMiddleware,
		InfraHealth:            infraHealth,
		OutboundCircuits:       outboundCircuits,
		PlatformService:        platformSvc,
		PlatformHandler:        platformHandler,
		PulseHandlers:          pulseHandlers,
		PushBridge:             pushBridge,
		Spanner:                spannerClient,
		OptimizerClient:        optimizerCli,
		DispatchPlanCounters:   dispatchCounters,
		ReplenishmentEngine:    replenishmentEngine,
		cleanup:                cleanup,
	}, nil
}

// inventoryAdapter bridges inventory.Service → supplier.InventoryServicer.
type inventoryAdapter struct {
	svc *inventory.Service
}

func (a *inventoryAdapter) ListBySupplier(ctx context.Context, supplierID string) ([]supplier.InventoryLevelView, error) {
	levels, err := a.svc.ListBySupplier(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	views := make([]supplier.InventoryLevelView, len(levels))
	for i, l := range levels {
		views[i] = supplier.InventoryLevelView{
			InventoryID:      l.InventoryID,
			ProductID:        l.ProductID,
			WarehouseID:      l.WarehouseID,
			SupplierID:       l.SupplierID,
			QuantityOnHand:   l.QuantityOnHand,
			QuantityReserved: l.QuantityReserved,
			ReorderThreshold: l.ReorderThreshold,
			Version:          l.Version,
		}
	}
	return views, nil
}

func (a *inventoryAdapter) AdjustStock(ctx context.Context, inventoryID string, delta int64, expectedVersion int64) error {
	return a.svc.AdjustStock(ctx, inventoryID, delta, expectedVersion)
}

func (a *inventoryAdapter) FindByWarehouseProduct(ctx context.Context, warehouseID, productID string) (string, bool, error) {
	level, err := a.svc.FindByWarehouseProduct(ctx, warehouseID, productID)
	if err != nil {
		return "", false, err
	}
	if level == nil {
		return "", false, nil
	}
	return level.InventoryID, true, nil
}

func (a *inventoryAdapter) UpsertLevel(ctx context.Context, level supplier.InventoryLevelUpsert) error {
	return a.svc.Upsert(ctx, inventory.Level{
		InventoryID:      level.InventoryID,
		ProductID:        level.ProductID,
		WarehouseID:      level.WarehouseID,
		SupplierID:       level.SupplierID,
		QuantityOnHand:   level.QuantityOnHand,
		QuantityReserved: level.QuantityReserved,
		ReorderThreshold: level.ReorderThreshold,
		Version:          level.Version,
	})
}

// driverOrderListQuery returns a DriverOrderQuery backed by stale Spanner reads.
func driverOrderListQuery(client *spanner.Client) driver.DriverOrderQuery {
	return func(ctx context.Context, driverID string) ([]driver.DriverOrderView, error) {
		stmt := spanner.Statement{
			SQL: `SELECT o.OrderId, o.RetailerId, COALESCE(r.Name, o.RetailerId), o.Status,
			             o.TotalMinor, o.DeliveryFeeMinor, o.Lat, o.Lng, COALESCE(o.RouteId, ''),
			             o.LineItemsJson, o.CreatedAt, o.UpdatedAt,
			             COALESCE(mo.SequenceIndex, 0)
			      FROM Orders o
			      LEFT JOIN Retailers r ON r.RetailerId = o.RetailerId
			      LEFT JOIN ManifestOrders mo ON mo.ManifestId = o.ManifestId AND mo.OrderId = o.OrderId
			      WHERE o.DriverId = @did AND o.Status NOT IN ('COMPLETED', 'CANCELLED')
			      ORDER BY COALESCE(mo.SequenceIndex, 999999) ASC, o.CreatedAt ASC`,
			Params: map[string]interface{}{"did": driverID},
		}
		iter := client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
		defer iter.Stop()
		var orders []driver.DriverOrderView
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("driver order list: %w", err)
			}
			var o driver.DriverOrderView
			var lat, lng spanner.NullFloat64
			var lineItems []byte
			var createdAt, updatedAt time.Time
			if err := row.Columns(
				&o.OrderID, &o.RetailerID, &o.RetailerName, &o.Status, &o.TotalMinor, &o.DeliveryFeeMinor,
				&lat, &lng, &o.RouteID, &lineItems, &createdAt, &updatedAt, &o.SequenceIndex,
			); err != nil {
				return nil, fmt.Errorf("driver order scan: %w", err)
			}
			if lat.Valid {
				o.Lat = lat.Float64
			}
			if lng.Valid {
				o.Lng = lng.Float64
			}
			o.Items = decodeDriverOrderLineItems(lineItems)
			o.CreatedAt = createdAt.Format(time.RFC3339Nano)
			o.UpdatedAt = updatedAt.Format(time.RFC3339Nano)
			orders = append(orders, o)
		}
		if orders == nil {
			orders = []driver.DriverOrderView{}
		}
		return orders, nil
	}
}

func driverProfileLookupQuery(client *spanner.Client) driver.DriverProfileLookup {
	return func(ctx context.Context, driverID string) (driver.DriverProfileSnapshot, bool, error) {
		row, err := client.Single().ReadRow(ctx, "Drivers", spanner.Key{driverID},
			[]string{"VehicleId"})
		if err != nil {
			return driver.DriverProfileSnapshot{}, false, nil
		}
		var vehicleID spanner.NullString
		if err := row.Columns(&vehicleID); err != nil {
			return driver.DriverProfileSnapshot{}, false, err
		}
		return driver.DriverProfileSnapshot{
			VehicleID: strings.TrimSpace(vehicleID.StringVal),
		}, true, nil
	}
}

func decodeDriverOrderLineItems(raw []byte) []driver.DriverOrderLineView {
	if len(raw) == 0 {
		return nil
	}
	var source []struct {
		SKU       string `json:"sku_id"`
		Name      string `json:"name"`
		Quantity  int64  `json:"quantity"`
		UnitPrice int64  `json:"unit_price"`
	}
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil
	}
	items := make([]driver.DriverOrderLineView, 0, len(source))
	for _, item := range source {
		items = append(items, driver.DriverOrderLineView{
			ProductID:   item.SKU,
			ProductName: item.Name,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
		})
	}
	return items
}

// driverOrderGetQuery returns a DriverOrderGetQuery backed by Spanner.
func driverOrderGetQuery(client *spanner.Client) driver.DriverOrderGetQuery {
	return func(ctx context.Context, orderID string) (driver.DriverOrderView, bool, error) {
		stmt := spanner.Statement{
			SQL: `SELECT o.OrderId, o.RetailerId, COALESCE(r.Name, o.RetailerId), o.Status,
			             o.TotalMinor, o.DeliveryFeeMinor, o.Lat, o.Lng, COALESCE(o.RouteId, ''),
			             o.LineItemsJson, o.CreatedAt, o.UpdatedAt,
			             COALESCE(mo.SequenceIndex, 0)
			      FROM Orders o
			      LEFT JOIN Retailers r ON r.RetailerId = o.RetailerId
			      LEFT JOIN ManifestOrders mo ON mo.ManifestId = o.ManifestId AND mo.OrderId = o.OrderId
			      WHERE o.OrderId = @oid`,
			Params: map[string]interface{}{"oid": orderID},
		}
		iter := client.Single().Query(ctx, stmt)
		defer iter.Stop()
		row, err := iter.Next()
		if err == iterator.Done {
			return driver.DriverOrderView{}, false, nil
		}
		if err != nil {
			return driver.DriverOrderView{}, false, fmt.Errorf("driver order get: %w", err)
		}
		var o driver.DriverOrderView
		var lat, lng spanner.NullFloat64
		var lineItems []byte
		var createdAt, updatedAt time.Time
		if err := row.Columns(
			&o.OrderID, &o.RetailerID, &o.RetailerName, &o.Status, &o.TotalMinor, &o.DeliveryFeeMinor,
			&lat, &lng, &o.RouteID, &lineItems, &createdAt, &updatedAt, &o.SequenceIndex,
		); err != nil {
			return driver.DriverOrderView{}, false, fmt.Errorf("driver order get scan: %w", err)
		}
		if lat.Valid {
			o.Lat = lat.Float64
		}
		if lng.Valid {
			o.Lng = lng.Float64
		}
		o.Items = decodeDriverOrderLineItems(lineItems)
		o.CreatedAt = createdAt.Format(time.RFC3339Nano)
		o.UpdatedAt = updatedAt.Format(time.RFC3339Nano)
		return o, true, nil
	}
}

func driverRouteGeometryQuery(client *spanner.Client, builder *routing.GeometryBuilder) driver.RouteGeometryLookup {
	return func(ctx context.Context, driverID, routeID string, opts driver.RouteGeometryOptions) (routing.RouteGeometry, bool, error) {
		driverID = strings.TrimSpace(driverID)
		routeID = strings.TrimSpace(routeID)
		if driverID == "" || routeID == "" {
			return routing.RouteGeometry{}, false, nil
		}

		ownStmt := spanner.Statement{
			SQL: `SELECT COUNT(*) FROM Orders
			      WHERE DriverId = @did AND RouteId = @rid
			        AND Status NOT IN ('COMPLETED', 'CANCELLED')`,
			Params: map[string]interface{}{"did": driverID, "rid": routeID},
		}
		ownIter := client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, ownStmt)
		defer ownIter.Stop()
		ownRow, err := ownIter.Next()
		if err != nil {
			return routing.RouteGeometry{}, false, fmt.Errorf("route ownership check: %w", err)
		}
		var owned int64
		if err := ownRow.Columns(&owned); err != nil {
			return routing.RouteGeometry{}, false, fmt.Errorf("route ownership scan: %w", err)
		}
		if owned == 0 {
			return routing.RouteGeometry{}, false, nil
		}

		waypoints, waypointErr := routing.WaypointsForDriverRoute(ctx, client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)), driverID, routeID)
		if waypointErr != nil {
			return routing.RouteGeometry{}, false, waypointErr
		}

		if opts.RerouteFrom != nil {
			waypoints = routing.WaypointsAhead(*opts.RerouteFrom, waypoints, 0)
			var geometry routing.RouteGeometry
			if builder != nil {
				geometry = builder.BuildDetail(ctx, routeID, waypoints, opts.IncludeSteps)
			} else {
				geometry = routing.BuildDenseRouteGeometry(routeID, waypoints)
			}
			geometry.Source = "reroute_" + geometry.Source
			return geometry, true, nil
		}

		storedStmt := spanner.Statement{
			SQL: `SELECT EncodedRoutePolyline, RouteGeometrySource, StopCount
			      FROM SupplierTruckManifests
			      WHERE DriverId = @did AND RouteId = @rid
			        AND EncodedRoutePolyline IS NOT NULL
			        AND EncodedRoutePolyline != ''
			      ORDER BY UpdatedAt DESC
			      LIMIT 1`,
			Params: map[string]interface{}{"did": driverID, "rid": routeID},
		}
		storedIter := client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, storedStmt)
		storedRow, storedErr := storedIter.Next()
		storedIter.Stop()
		if storedErr == nil {
			var encoded string
			var source spanner.NullString
			var stopCount int64
			if err := storedRow.Columns(&encoded, &source, &stopCount); err == nil && encoded != "" {
				geometry, decodeErr := routing.GeometryFromStoredPolyline(
					routeID,
					encoded,
					source.StringVal,
					int(stopCount),
				)
				if decodeErr == nil {
					if opts.IncludeSteps && builder != nil && len(waypoints) >= 2 {
						detail := builder.BuildDetail(ctx, routeID, waypoints, true)
						geometry.Steps = detail.Steps
					}
					return geometry, true, nil
				}
			}
		}

		var geometry routing.RouteGeometry
		if builder != nil {
			geometry = builder.BuildDetail(ctx, routeID, waypoints, opts.IncludeSteps)
		} else {
			geometry = routing.BuildDenseRouteGeometry(routeID, waypoints)
		}
		return geometry, true, nil
	}
}

// warehouseAnalyticsCountQuery returns a WarehouseAnalyticsQuery backed by
// stale Spanner reads.
func warehouseAnalyticsCountQuery(client *spanner.Client) warehouse.WarehouseAnalyticsQuery {
	return func(ctx context.Context, warehouseID string) (warehouse.WarehouseAnalyticsCounts, error) {
		var counts warehouse.WarehouseAnalyticsCounts
		stmt := spanner.Statement{
			SQL: `SELECT COUNT(*) AS total,
			             COUNTIF(Status = 'COMPLETED') AS completed,
			             COUNTIF(Status = 'CANCELLED') AS cancelled,
			             IFNULL(SUM(CASE WHEN Status = 'COMPLETED' THEN TotalMinor ELSE 0 END), 0) AS revenue
			      FROM Orders WHERE WarehouseId = @wid`,
			Params: map[string]interface{}{"wid": warehouseID},
		}
		iter := client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
		defer iter.Stop()
		row, err := iter.Next()
		if err != nil {
			return counts, fmt.Errorf("warehouse analytics query: %w", err)
		}
		if err := row.Columns(&counts.TotalOrders, &counts.CompletedOrders, &counts.CancelledOrders, &counts.TotalRevenue); err != nil {
			return counts, fmt.Errorf("warehouse analytics scan: %w", err)
		}
		return counts, nil
	}
}

// warehouseOpsOrdersQuery returns warehouse-scoped order rows from Spanner.
func warehouseOpsOrdersQuery(client *spanner.Client) warehouse.WarehouseOpsOrdersQuery {
	return func(ctx context.Context, warehouseID string, limit int) ([]warehouse.OrderRow, error) {
		stmt := spanner.Statement{
			SQL: `SELECT OrderID, RetailerID, Status, TotalMinor, Currency, UpdatedAt
			      FROM Orders WHERE WarehouseId = @wid
			        AND NOT (OrderSource = 'MANUAL_PREORDER' AND Status = 'SCHEDULED')
			      ORDER BY UpdatedAt DESC LIMIT @lim`,
			Params: map[string]interface{}{"wid": warehouseID, "lim": int64(limit)},
		}
		iter := client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
		defer iter.Stop()
		var orders []warehouse.OrderRow
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("warehouse ops orders: %w", err)
			}
			var o warehouse.OrderRow
			var updatedAt time.Time
			if err := row.Columns(&o.OrderID, &o.RetailerID, &o.Status, &o.TotalMinor, &o.Currency, &updatedAt); err != nil {
				return nil, fmt.Errorf("warehouse ops orders scan: %w", err)
			}
			o.UpdatedAt = updatedAt.Format(time.RFC3339Nano)
			orders = append(orders, o)
		}
		if orders == nil {
			orders = []warehouse.OrderRow{}
		}
		return orders, nil
	}
}

// warehouseOpsDriversQuery returns drivers home-noded to a warehouse.
func warehouseOpsDriversQuery(client *spanner.Client) warehouse.WarehouseOpsDriversQuery {
	return func(ctx context.Context, warehouseID string) ([]warehouse.PortalDriver, error) {
		stmt := spanner.Statement{
			SQL: `SELECT d.DriverId, d.Name, COALESCE(d.Phone, ''), d.IsActive, COALESCE(d.OnShift, true),
			             COALESCE(d.UnavailableReason, ''), COALESCE(d.UnavailableNote, ''),
			             COALESCE(d.VehicleId, ''), COALESCE(v.VehicleClass, 'CLASS_B'),
			             COALESCE(v.MaxVolumeVU, 150.0), COALESCE(v.IsActive, FALSE),
			             COALESCE(v.UnavailableReason, ''), COALESCE(v.UnavailableNote, ''),
			             COALESCE(v.Label, v.LicensePlate, '')
			      FROM Drivers@{FORCE_INDEX=Idx_Drivers_ByHomeNode} d
			      LEFT JOIN Vehicles v ON d.VehicleId = v.VehicleId
			      WHERE d.HomeNodeType = 'WAREHOUSE' AND d.HomeNodeId = @wid
			      ORDER BY d.Name`,
			Params: map[string]interface{}{"wid": warehouseID},
		}
		iter := client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
		defer iter.Stop()
		var drivers []warehouse.PortalDriver
		driverIDs := make([]string, 0, 8)
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("warehouse ops drivers: %w", err)
			}
			var d warehouse.PortalDriver
			var vehicleID spanner.NullString
			var unavailableReason, unavailableNote, vehicleUnavailableReason, vehicleUnavailableNote string
			if err := row.Columns(
				&d.DriverID,
				&d.Name,
				&d.Phone,
				&d.IsActive,
				&d.OnShift,
				&unavailableReason,
				&unavailableNote,
				&vehicleID,
				&d.VehicleClass,
				&d.MaxVolumeVU,
				&d.VehicleIsActive,
				&vehicleUnavailableReason,
				&vehicleUnavailableNote,
				&d.VehicleLabel,
			); err != nil {
				return nil, fmt.Errorf("warehouse ops drivers scan: %w", err)
			}
			d.UnavailableReason = strings.TrimSpace(unavailableReason)
			d.UnavailableNote = strings.TrimSpace(unavailableNote)
			d.VehicleUnavailableReason = strings.TrimSpace(vehicleUnavailableReason)
			d.VehicleUnavailableNote = strings.TrimSpace(vehicleUnavailableNote)
			if vehicleID.Valid {
				d.VehicleID = vehicleID.StringVal
			}
			switch {
			case !d.IsActive:
				d.TruckStatus = "INACTIVE"
			case !d.OnShift:
				if strings.EqualFold(d.UnavailableReason, "RETURNING_TO_WAREHOUSE") {
					d.TruckStatus = "RETURNING_TO_WAREHOUSE"
				} else {
					d.TruckStatus = "OFF_SHIFT"
				}
			case d.VehicleID == "":
				d.TruckStatus = "UNASSIGNED"
			case !d.VehicleIsActive:
				d.TruckStatus = "VEHICLE_INACTIVE"
			default:
				d.TruckStatus = "AVAILABLE"
			}
			drivers = append(drivers, d)
			driverIDs = append(driverIDs, d.DriverID)
		}
		if len(driverIDs) > 0 {
			busy, err := warehouseDriversOnActiveManifests(ctx, client, warehouseID, driverIDs)
			if err != nil {
				return nil, err
			}
			for i := range drivers {
				if busy[drivers[i].DriverID] {
					drivers[i].TruckStatus = "IN_TRANSIT"
				}
			}
		}
		if drivers == nil {
			drivers = []warehouse.PortalDriver{}
		}
		return drivers, nil
	}
}

func warehouseDriversOnActiveManifests(ctx context.Context, client *spanner.Client, warehouseID string, driverIDs []string) (map[string]bool, error) {
	out := make(map[string]bool, len(driverIDs))
	if client == nil || warehouseID == "" || len(driverIDs) == 0 {
		return out, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT DISTINCT DriverId
		      FROM SupplierTruckManifests@{FORCE_INDEX=Idx_SupplierManifests_ByWarehouse}
		      WHERE WarehouseId = @wid
		        AND DriverId IN UNNEST(@driverIds)
		        AND State IN ('DRAFT', 'LOADING', 'SEALED', 'DISPATCHED')`,
		Params: map[string]any{
			"wid":       warehouseID,
			"driverIds": driverIDs,
		},
	}
	iter := client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("warehouse active manifest drivers: %w", err)
		}
		var driverID string
		if err := row.Columns(&driverID); err != nil {
			return nil, fmt.Errorf("warehouse active manifest drivers scan: %w", err)
		}
		if driverID != "" {
			out[driverID] = true
		}
	}
}

// warehouseOpsVehiclesQuery returns vehicles home-noded to a warehouse.
func warehouseOpsVehiclesQuery(client *spanner.Client) warehouse.WarehouseOpsVehiclesQuery {
	return func(ctx context.Context, warehouseID string) ([]warehouse.PortalVehicle, error) {
		stmt := spanner.Statement{
			SQL: `SELECT v.VehicleId, COALESCE(v.Label, ''), v.LicensePlate,
			             COALESCE(v.VehicleClass, 'CLASS_B'), COALESCE(v.MaxVolumeVU, 150.0), v.IsActive,
			             COALESCE(v.UnavailableReason, ''), COALESCE(v.UnavailableNote, ''),
			             COALESCE(d.DriverId, ''), COALESCE(d.Name, '')
			      FROM Vehicles@{FORCE_INDEX=Idx_Vehicles_ByHomeNode} v
			      LEFT JOIN Drivers@{FORCE_INDEX=Idx_Drivers_ByHomeNode} d
			        ON d.VehicleId = v.VehicleId
			       AND d.HomeNodeType = 'WAREHOUSE'
			       AND d.HomeNodeId = @wid
			      WHERE v.HomeNodeType = 'WAREHOUSE' AND v.HomeNodeId = @wid
			      ORDER BY v.LicensePlate`,
			Params: map[string]interface{}{"wid": warehouseID},
		}
		iter := client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
		defer iter.Stop()
		var vehicles []warehouse.PortalVehicle
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("warehouse ops vehicles: %w", err)
			}
			var v warehouse.PortalVehicle
			if err := row.Columns(
				&v.VehicleID,
				&v.Label,
				&v.LicensePlate,
				&v.VehicleClass,
				&v.MaxVolumeVU,
				&v.IsActive,
				&v.UnavailableReason,
				&v.UnavailableNote,
				&v.AssignedDriverID,
				&v.AssignedDriverName,
			); err != nil {
				return nil, fmt.Errorf("warehouse ops vehicles scan: %w", err)
			}
			vehicles = append(vehicles, v)
		}
		if vehicles == nil {
			vehicles = []warehouse.PortalVehicle{}
		}
		return vehicles, nil
	}
}

func driverAvailabilityReader(client *spanner.Client) driver.AvailabilityReader {
	return func(ctx context.Context, driverID string) (bool, string, string, bool, error) {
		if client == nil || strings.TrimSpace(driverID) == "" {
			return true, "", "", false, nil
		}
		stmt := spanner.Statement{
			SQL: `SELECT COALESCE(OnShift, true), COALESCE(UnavailableReason, ''), COALESCE(UnavailableNote, '')
			      FROM Drivers WHERE DriverId = @id`,
			Params: map[string]any{"id": driverID},
		}
		row, err := client.Single().Query(ctx, stmt).Next()
		if err == iterator.Done {
			return true, "", "", false, nil
		}
		if err != nil {
			return true, "", "", false, err
		}
		var onShift bool
		var reason, note string
		if err := row.Columns(&onShift, &reason, &note); err != nil {
			return true, "", "", false, err
		}
		return onShift, strings.TrimSpace(reason), strings.TrimSpace(note), true, nil
	}
}

// notificationReaderAdapter bridges notifications.Service to the
// retailer.NotificationReader and driver.DriverNotificationReader interfaces.
type notificationReaderAdapter struct {
	svc *notifications.Service
}

func (a *notificationReaderAdapter) ListForRecipient(ctx context.Context, recipientID string, limit, offset int) ([]any, error) {
	if a == nil || a.svc == nil {
		return []any{}, nil
	}
	notifs, err := a.svc.ListForRecipient(ctx, recipientID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]any, len(notifs))
	for i, n := range notifs {
		out[i] = n
	}
	return out, nil
}

func (a *notificationReaderAdapter) MarkRead(ctx context.Context, recipientID string, notificationIDs []string) error {
	if a == nil || a.svc == nil {
		return nil
	}
	return a.svc.MarkRead(ctx, recipientID, notificationIDs)
}

func (a *notificationReaderAdapter) MarkAllRead(ctx context.Context, recipientID string) error {
	if a == nil || a.svc == nil {
		return nil
	}
	return a.svc.MarkAllRead(ctx, recipientID)
}

func (a *notificationReaderAdapter) UnreadCount(ctx context.Context, recipientID string) (int64, error) {
	if a == nil || a.svc == nil {
		return 0, nil
	}
	return a.svc.UnreadCount(ctx, recipientID)
}

// supplierDashboardCountQuery returns a DashboardCountQuery backed by stale
// Spanner reads for aggregate dashboard KPIs.
func supplierDashboardCountQuery(client *spanner.Client) supplier.DashboardCountQuery {
	return func(ctx context.Context, supplierID string) (supplier.DashboardCounts, error) {
		var counts supplier.DashboardCounts
		stmt := spanner.Statement{
			SQL: `SELECT COUNTIF(Status IN ('PENDING', 'AWAITING_REVIEW')) AS pending_orders,
			             COUNTIF(Status = 'IN_TRANSIT') AS active_deliveries
			      FROM Orders WHERE SupplierID = @sid`,
			Params: map[string]interface{}{"sid": supplierID},
		}
		iter := client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
		defer iter.Stop()
		row, err := iter.Next()
		if err != nil {
			return counts, fmt.Errorf("dashboard count query: %w", err)
		}
		var pendingOrders, activeDeliveries int64
		if err := row.Columns(&pendingOrders, &activeDeliveries); err != nil {
			return counts, fmt.Errorf("dashboard count scan: %w", err)
		}
		counts.PendingOrders = int(pendingOrders)
		counts.ActiveDrivers = int(activeDeliveries)
		return counts, nil
	}
}



// Close tears down long-lived resources in reverse construction order.
func (a *App) Close() {
	if a == nil {
		return
	}
	for i := len(a.cleanup) - 1; i >= 0; i-- {
		if a.cleanup[i] != nil {
			a.cleanup[i]()
		}
	}
	if a.Cache != nil {
		if err := a.Cache.Close(); err != nil {
			slog.Warn("cache close failed", "err", err)
		}
	}
}

func loadSupplierEarningsAuthority(ctx context.Context, repo payment.Repository, supplierID, currency string, now time.Time) (supplier.SupplierEarningsResponse, error) {
	if repo == nil {
		return supplier.SupplierEarningsResponse{}, errors.New("payment repository unavailable")
	}
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	weekStart := dayStart.AddDate(0, 0, -int(dayStart.Weekday()))
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	today, todayAuthoritative, err := sumSupplierEarningsWindow(ctx, repo, supplierID, currency, dayStart, now)
	if err != nil {
		return supplier.SupplierEarningsResponse{}, err
	}
	week, weekAuthoritative, err := sumSupplierEarningsWindow(ctx, repo, supplierID, currency, weekStart, now)
	if err != nil {
		return supplier.SupplierEarningsResponse{}, err
	}
	month, monthAuthoritative, err := sumSupplierEarningsWindow(ctx, repo, supplierID, currency, monthStart, now)
	if err != nil {
		return supplier.SupplierEarningsResponse{}, err
	}

	return supplier.SupplierEarningsResponse{
		Currency:        currency,
		TodayMinor:      today,
		WeekMinor:       week,
		MonthMinor:      month,
		AuthoritySource: "payment_ledger",
		Authoritative:   todayAuthoritative && weekAuthoritative && monthAuthoritative,
		UpdatedAt:       now.Format(time.RFC3339Nano),
	}, nil
}

func sumSupplierEarningsWindow(ctx context.Context, repo payment.Repository, supplierID, currency string, from, to time.Time) (int64, bool, error) {
	rows, err := repo.SummarizeLedgerEntries(ctx, payment.SettlementAuthorityQuery{
		SupplierID:   supplierID,
		OccurredFrom: &from,
		OccurredTo:   &to,
		GroupLimit:   1000,
	})
	if err != nil {
		return 0, false, fmt.Errorf("summarize supplier earnings window: %w", err)
	}
	total := int64(0)
	authoritative := true
	for _, row := range rows {
		if strings.TrimSpace(currency) != "" && !strings.EqualFold(row.Currency, currency) {
			authoritative = false
			continue
		}
		total += payment.SignedSettlementEntryAmount(row.EntryType, row.AmountMinorTotal)
	}
	return total, authoritative, nil
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
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

func shopClosedGraceDuration() time.Duration {
	minutes := envInt("SHOP_CLOSED_GRACE_MINUTES", 5)
	if minutes < 1 {
		minutes = 5
	}
	return time.Duration(minutes) * time.Minute
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


type runtimeSeedRepository struct {
	client *spanner.Client
}

func (r *runtimeSeedRepository) UpsertSupplier(ctx context.Context, s seed.Supplier) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("runtime seed repository: nil client")
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		createdAt, err := existingSeedSupplierCreatedAt(ctx, txn, s.SupplierID)
		if err != nil {
			return err
		}
		mutation := spanner.InsertOrUpdateMap("Suppliers", map[string]any{
			"SupplierId":   s.SupplierID,
			"Name":         s.Name,
			"CountryCode":  s.CountryCode,
			"Currency":     s.Currency,
			"IsConfigured": false,
			"CreatedAt":    createdAt,
			"UpdatedAt":    spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{mutation})
	})
	if err != nil {
		return fmt.Errorf("upsert seed supplier %s: %w", s.SupplierID, err)
	}
	return nil
}

func existingSeedSupplierCreatedAt(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID string) (time.Time, error) {
	row, err := txn.ReadRow(ctx, "Suppliers", spanner.Key{supplierID}, []string{"CreatedAt"})
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return time.Now().UTC(), nil
		}
		return time.Time{}, fmt.Errorf("read seed supplier %s: %w", supplierID, err)
	}
	var createdAt time.Time
	if err := row.Columns(&createdAt); err != nil {
		return time.Time{}, fmt.Errorf("decode seed supplier %s created_at: %w", supplierID, err)
	}
	return createdAt, nil
}

