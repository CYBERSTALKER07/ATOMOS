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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/catalog"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/optimizerclient"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/plan"
	"github.com/pegasusx/pegasusx/apps/backend-go/driver"
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
	"github.com/pegasusx/pegasusx/apps/backend-go/retailer"
	"github.com/pegasusx/pegasusx/apps/backend-go/seed"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouse"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
	"google.golang.org/api/iterator"
)

// Config carries the runtime parameters. Loaded from environment by LoadConfig.
type Config struct {
	HTTPPort string

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
	InternalAPIKey   string
}

// App holds every long-lived singleton. Wire new app-wide dependencies here,
// never as package-level globals.
type App struct {
	Config                 *Config
	Cache                  *cache.Cache
	Idempotency            idempotency.Store
	Supplier               seed.Supplier
	CatalogService         *catalog.Service
	InventoryService       *inventory.Service
	NotificationService    *notifications.Service
	SupplierService        *supplier.Service
	RetailerService        *retailer.Service
	RetailerProximity      *retailer.RetailerProximityService
	DriverService          *driver.Service
	FactoryService         *factory.Service
	PayloadService         *payload.Service
	PaymentService         *payment.Service
	WarehouseService       *warehouse.Service
	OrderService           *order.Service
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
	PushBridge             *notifications.PushBridge
	Spanner                *spanner.Client
	OptimizerClient        *optimizerclient.Client
	DispatchPlanCounters   *plan.SourceCounters
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
		InternalAPIKey:                  envOr("INTERNAL_API_KEY", "dev-internal-key"),
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
	ws.SetAllowedOrigins(cfg.WebSocketAllowedOrigins)
	if cfg.RequireInfraAdapters {
		log.Info("strict infra adapter mode enabled")
	}

	cacheBackend := cache.Backend(cache.NewInMemoryBackend())
	redisEnabled := false
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
	if redisEnabled {
		adapter, ok := cacheBackend.(redisRuntimeAdapter)
		if !ok {
			log.Warn("cacheBackend is not a redisRuntimeAdapter, cannot use for idempotency store")
		} else {
			idemStore = idempotency.NewRedisStore(adapter.Client())
			log.Info("idempotency redis store enabled", "addr", cfg.RedisAddr)
		}
	}
	memoryOutboxStore := newInMemoryOutboxStore()
	relayStore := outbox.Store(memoryOutboxStore)
	outboxAppender := outboxEventAppender(memoryOutboxStore)
	var spannerClient *spanner.Client
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
		cleanup = append(cleanup, func() {
			client.Close()
		})
		log.Info("spanner outbox store enabled", "database", spannerDatabasePath(cfg))
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
		retailerRepo = newInMemoryRetailerRepo(outboxAppender)
		supplierRepo = newInMemorySupplierRepo(outboxAppender)
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

	var catalogSvc *catalog.Service
	if spannerClient != nil {
		catalogRepo := catalog.NewSpannerRepository(spannerClient)
		catalogSvc = catalog.NewService(catalogRepo, cacheClient, log)
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
	var notifAdapter *notificationReaderAdapter
	if spannerClient != nil {
		notifRepo := notifications.NewSpannerRepository(spannerClient)
		notifSvc = notifications.NewService(notifRepo, cacheClient, log)
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

	var orderRepo order.Repository
	var orderWarehouseResolver order.WarehouseResolver
	if spannerClient != nil {
		orderRepo = order.NewSpannerRepository(spannerClient)
		orderWarehouseResolver = order.NewSpannerWarehouseResolver(spannerClient)
		log.Info("order repository enabled", "backend", "spanner")
	} else {
		orderRepo = newInMemoryOrderRepo(outboxAppender, &retailerReceivingWindowAdapter{repo: retailerRepo})
		log.Warn("order repository fallback enabled", "backend", "in-memory")
	}
	var paymentRepo payment.Repository
	if spannerClient != nil {
		paymentRepo = payment.NewSpannerRepository(spannerClient)
		log.Info("payment repository enabled", "backend", "spanner")
	} else {
		paymentRepo = newInMemoryPaymentRepo(outboxAppender)
		log.Warn("payment repository fallback enabled", "backend", "in-memory")
	}
	supplierSvc.SetEarningsLookup(func(ctx context.Context, supplierID, currency string, now time.Time) (supplier.SupplierEarningsResponse, error) {
		return loadSupplierEarningsAuthority(ctx, paymentRepo, supplierID, currency, now)
	})
	orderSvc := order.NewService(order.ServiceConfig{
		Repo:            orderRepo,
		Cache:           cacheClient,
		Warehouse:       orderWarehouseResolver,
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
	})
	var optimizerCli *optimizerclient.Client
	if strings.TrimSpace(cfg.OptimizerBaseURL) != "" && strings.TrimSpace(cfg.InternalAPIKey) != "" {
		optimizerCli = optimizerclient.New(cfg.OptimizerBaseURL, cfg.InternalAPIKey)
		log.Info("dispatch optimiser client enabled", "base_url", cfg.OptimizerBaseURL)
	}
	dispatchCounters := &plan.SourceCounters{}

	supplierSvc.SetPortalOps(supplier.PortalOpsConfig{
		Spanner:          spannerClient,
		SupplierHub:      supplierHub,
		OptimizerClient:  optimizerCli,
		PlanCounters:     dispatchCounters,
		FallbackDepotLat: cfg.DeliveryZoneCenterLat,
		FallbackDepotLng: cfg.DeliveryZoneCenterLng,
	})
	retailerSvc.SetOrderLifecycle(orderSvc)

	factoryNodeID := strings.TrimSpace(os.Getenv("FACTORY_DEMO_ID"))
	if factoryNodeID == "" {
		factoryNodeID = "factory-demo-1"
	}

	var driverRepo driver.Repository
	var factoryRepo factory.Repository
	var payloadRepo payload.Repository
	if spannerClient == nil {
		log.Error("spanner client required but not configured", "backend", "spanner")
		panic("spanner client required but not configured")
	}
	driverRepo = driver.NewSpannerRepository(spannerClient)
	factoryRepo = factory.NewSpannerRepository(spannerClient, supplierSeed.SupplierID, factoryNodeID)
	payloadRepo = payload.NewSpannerRepository(spannerClient, supplierSeed.SupplierID)
	log.Info("factory and payload repositories enabled", "backend", "spanner")

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
	})
	payloadSvc := payload.NewService(payload.ServiceConfig{
		Repo:        payloadRepo,
		Cache:       cacheClient,
		SupplierHub: supplierHub,
		PayloadHub:  payloadHub,
		DriverHub:   driverHub,
		NotifSvc:    notifSvc,
		Log:         log,
		SupplierID:  supplierSeed.SupplierID,
		Currency:    cfg.SeedSupplierCurrency,
		JWTSecret:   cfg.JWTSecret,
		JWTIssuer:   cfg.JWTIssuer,
	})
	payloadSvc.SetPortalManifestLister(&supplier.ManifestLister{Service: supplierSvc})
	payloadSvc.WarmManifestCache(ctx)
	factorySvc.WarmManifestCache(ctx)
	var driverOrderList driver.DriverOrderQuery
	var driverOrderGet driver.DriverOrderGetQuery
	var driverDepart driver.DepartFn
	var driverReturnComplete driver.ReturnCompleteFn
	if spannerClient != nil {
		driverOrderList = driverOrderListQuery(spannerClient)
		driverOrderGet = driverOrderGetQuery(spannerClient)
		manifestStore := manifest.NewStore(spannerClient)
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
		Depart:         driverDepart,
		ReturnComplete: driverReturnComplete,
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
	})
	paymentSvc := payment.NewService(payment.ServiceConfig{
		Repo:                            paymentRepo,
		Cache:                           cacheClient,
		Idem:                            idemStore,
		SupplierID:                      supplierSeed.SupplierID,
		Currency:                        cfg.SeedSupplierCurrency,
		GlobalPayEnv:                    cfg.GlobalPayEnv,
		GlobalPayServiceID:              cfg.GlobalPayServiceID,
		GlobalPayUsername:               cfg.GlobalPayUsername,
		GlobalPayPassword:               cfg.GlobalPayPassword,
		GlobalPayWebhookSecret:          cfg.GlobalPayWebhookSecret,
		AdyenWebhookSecret:              cfg.AdyenWebhookSecret,
		StripeWebhookSecret:             cfg.StripeWebhookSecret,
		AirwallexDirectExecutionEnabled: cfg.AirwallexDirectExecutionEnabled,
		Log:                             log,
	})
	paymentSvc.BindCartCheckout(orderSvc)
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
		Repo:             warehouseRepo,
		Planner:          orderSvc,
		AnalyticsQuery:   warehouseAnalytics,
		OpsOrders:        whOpsOrders,
		OpsDrivers:       whOpsDrivers,
		OpsVehicles:      whOpsVehicles,
		Cache:            cacheClient,
		Spanner:          spannerClient,
		SupplierHub:      supplierHub,
		WarehouseHub:     warehouseHub,
		Log:              log,
		SupplierID:       supplierSeed.SupplierID,
		Currency:         cfg.SeedSupplierCurrency,
		JWTSecret:        cfg.JWTSecret,
		JWTIssuer:        cfg.JWTIssuer,
		OptimizerClient:  optimizerCli,
		PlanCounters:     dispatchCounters,
		FallbackDepotLat: cfg.DeliveryZoneCenterLat,
		FallbackDepotLng: cfg.DeliveryZoneCenterLng,
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
	outboundCircuits := NewOutboundCircuits()
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
			dispatcher := kafka.NewNotificationDispatcher(kafka.DispatcherDeps{
				RetailerHub:  retailerHub,
				SupplierHub:  supplierHub,
				DriverHub:    driverHub,
				WarehouseHub: warehouseHub,
				FactoryHub:   factoryHub,
				PayloadHub:   payloadHub,
				Push:         pushBridge,
			})
			notificationConsumer = kafka.NewConsumer(kafka.ConsumerDeps{
				Brokers:   strings.Split(cfg.KafkaBrokers, ","),
				GroupID:   "void-notification-dispatcher",
				Topic:     cfg.KafkaTopicMain,
				Handler:   dispatcher.HandleEvent,
				DLQWriter: dlqWriter,
			})
			orderEventConsumer = kafka.NewConsumer(kafka.ConsumerDeps{
				Brokers:   strings.Split(cfg.KafkaBrokers, ","),
				GroupID:   "void-order-mutator",
				Topic:     cfg.KafkaTopicMain,
				Handler:   order.NewEventConsumer(orderSvc, log).HandleEvent,
				DLQWriter: dlqWriter,
			})
			warehouseEventConsumer := kafka.NewConsumer(kafka.ConsumerDeps{
				Brokers:   strings.Split(cfg.KafkaBrokers, ","),
				GroupID:   "void-warehouse-mutator",
				Topic:     cfg.KafkaTopicMain,
				Handler:   warehouse.NewEventConsumer(warehouseSvc, log).HandleEvent,
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
			log.Info("notification consumer enabled", "topic", cfg.KafkaTopicMain, "dlq_topic", cfg.KafkaTopicMainDLQ)
		}
	}

	if spannerClient != nil {
		if err := auth.EnsureDemoScopeLinks(ctx, spannerClient, supplierSeed.SupplierID); err != nil {
			log.Warn("demo scope link seed failed", "err", err)
		}
	}

	return &App{
		Config:                 cfg,
		Cache:                  cacheClient,
		Idempotency:            idemStore,
		Supplier:               supplierSeed,
		CatalogService:         catalogSvc,
		InventoryService:       inventorySvc,
		NotificationService:    notifSvc,
		SupplierService:        supplierSvc,
		RetailerService:        retailerSvc,
		DriverService:          driverSvc,
		FactoryService:         factorySvc,
		PayloadService:         payloadSvc,
		PaymentService:         paymentSvc,
		WarehouseService:       warehouseSvc,
		OrderService:           orderSvc,
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
		PushBridge:             pushBridge,
		Spanner:                spannerClient,
		OptimizerClient:        optimizerCli,
		DispatchPlanCounters:   dispatchCounters,
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

// driverOrderListQuery returns a DriverOrderQuery backed by stale Spanner reads.
func driverOrderListQuery(client *spanner.Client) driver.DriverOrderQuery {
	return func(ctx context.Context, driverID string) ([]driver.DriverOrderView, error) {
		stmt := spanner.Statement{
			SQL: `SELECT o.OrderId, o.RetailerId, COALESCE(r.Name, o.RetailerId), o.Status,
			             o.TotalMinor, o.Lat, o.Lng, COALESCE(o.RouteId, ''),
			             o.LineItemsJson, o.CreatedAt, o.UpdatedAt,
			             COALESCE(mo.SequenceIndex, 0)
			      FROM Orders o
			      LEFT JOIN Retailers r ON r.RetailerId = o.RetailerId
			      LEFT JOIN ManifestOrders mo ON mo.ManifestId = o.ManifestId AND mo.OrderId = o.OrderId
			      WHERE o.DriverId = @did AND o.Status NOT IN ('COMPLETED', 'CANCELLED')
			      ORDER BY COALESCE(mo.SequenceIndex, 999999) ASC, o.CreatedAt ASC
			      LIMIT 50`,
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
				&o.OrderID, &o.RetailerID, &o.RetailerName, &o.Status, &o.TotalMinor,
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
			             o.TotalMinor, o.Lat, o.Lng, COALESCE(o.RouteId, ''),
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
			&o.OrderID, &o.RetailerID, &o.RetailerName, &o.Status, &o.TotalMinor,
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
			SQL: `SELECT d.DriverId, d.Name, COALESCE(d.Phone, ''), d.IsActive,
			             COALESCE(d.VehicleId, ''), COALESCE(v.VehicleClass, 'CLASS_B'),
			             COALESCE(v.MaxVolumeVU, 150.0), COALESCE(v.IsActive, FALSE)
			      FROM Drivers@{FORCE_INDEX=Idx_Drivers_ByHomeNode} d
			      LEFT JOIN Vehicles v ON d.VehicleId = v.VehicleId
			      WHERE d.HomeNodeType = 'WAREHOUSE' AND d.HomeNodeId = @wid
			      ORDER BY d.Name`,
			Params: map[string]interface{}{"wid": warehouseID},
		}
		iter := client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
		defer iter.Stop()
		var drivers []warehouse.PortalDriver
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
			if err := row.Columns(
				&d.DriverID,
				&d.Name,
				&d.Phone,
				&d.IsActive,
				&vehicleID,
				&d.VehicleClass,
				&d.MaxVolumeVU,
				&d.VehicleIsActive,
			); err != nil {
				return nil, fmt.Errorf("warehouse ops drivers scan: %w", err)
			}
			if vehicleID.Valid {
				d.VehicleID = vehicleID.StringVal
			}
			switch {
			case !d.IsActive:
				d.TruckStatus = "INACTIVE"
			case d.VehicleID == "":
				d.TruckStatus = "UNASSIGNED"
			case !d.VehicleIsActive:
				d.TruckStatus = "VEHICLE_INACTIVE"
			default:
				d.TruckStatus = "AVAILABLE"
			}
			drivers = append(drivers, d)
		}
		if drivers == nil {
			drivers = []warehouse.PortalDriver{}
		}
		return drivers, nil
	}
}

// warehouseOpsVehiclesQuery returns vehicles home-noded to a warehouse.
func warehouseOpsVehiclesQuery(client *spanner.Client) warehouse.WarehouseOpsVehiclesQuery {
	return func(ctx context.Context, warehouseID string) ([]warehouse.PortalVehicle, error) {
		stmt := spanner.Statement{
			SQL: `SELECT VehicleId, COALESCE(Label, ''), LicensePlate,
			             COALESCE(VehicleClass, 'CLASS_B'), COALESCE(MaxVolumeVU, 150.0), IsActive
			      FROM Vehicles@{FORCE_INDEX=Idx_Vehicles_ByHomeNode}
			      WHERE HomeNodeType = 'WAREHOUSE' AND HomeNodeId = @wid
			      ORDER BY LicensePlate`,
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
			if err := row.Columns(&v.VehicleID, &v.Label, &v.LicensePlate, &v.VehicleClass, &v.MaxVolumeVU, &v.IsActive); err != nil {
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

// notificationReaderAdapter bridges notifications.Service to the
// retailer.NotificationReader and driver.DriverNotificationReader interfaces.
type notificationReaderAdapter struct {
	svc *notifications.Service
}

func (a *notificationReaderAdapter) ListForRecipient(ctx context.Context, recipientID string, limit int) ([]any, error) {
	if a == nil || a.svc == nil {
		return []any{}, nil
	}
	notifs, err := a.svc.ListForRecipient(ctx, recipientID, limit)
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

// ── Scaffold in-memory order repository ────────────────────────────────────
// Production replaces this with a Spanner-backed implementation that runs
// CreateOrder inside a ReadWriteTransaction and writes both the Orders row
// and the OutboxEvents row atomically.

type inMemoryOrderRepo struct {
	mu             sync.RWMutex
	byID           map[string]order.Order
	outboxAppender outboxEventAppender
	windows        order.ReceivingWindowReader
}

func newInMemoryOrderRepo(outboxAppender outboxEventAppender, windows order.ReceivingWindowReader) *inMemoryOrderRepo {
	return &inMemoryOrderRepo{
		byID:           make(map[string]order.Order),
		outboxAppender: outboxAppender,
		windows:        windows,
	}
}

type retailerReceivingWindowAdapter struct {
	repo retailer.Repository
}

func (a *retailerReceivingWindowAdapter) GetReceivingWindows(ctx context.Context, retailerID string) (string, string, error) {
	if a == nil || a.repo == nil {
		return "", "", nil
	}
	ret, ok, err := a.repo.GetRetailer(ctx, retailerID)
	if err != nil {
		return "", "", err
	}
	if !ok {
		return "", "", nil
	}
	return ret.ReceivingWindowOpen, ret.ReceivingWindowClose, nil
}

func (r *inMemoryOrderRepo) CreateOrder(ctx context.Context, o *order.Order, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if o == nil {
		return fmt.Errorf("create order: nil aggregate")
	}
	if _, exists := r.byID[o.OrderID]; exists {
		return fmt.Errorf("order_id collision: %s", o.OrderID)
	}
	if r.windows != nil {
		open, closeWindow, err := r.windows.GetReceivingWindows(ctx, o.RetailerID)
		if err != nil {
			return err
		}
		if err := order.SnapshotReceivingWindowsOnOrder(o, open, closeWindow); err != nil {
			return err
		}
	}
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.byID[o.OrderID] = *o
	return nil
}

func (r *inMemoryOrderRepo) GetOrder(_ context.Context, orderID string) (order.Order, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, ok := r.byID[orderID]
	return o, ok, nil
}

func (r *inMemoryOrderRepo) ListRetailerOrders(_ context.Context, retailerID string, limit int) ([]order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 25
	}
	items := make([]order.Order, 0, limit)
	for _, orderRecord := range r.byID {
		if orderRecord.RetailerID != strings.TrimSpace(retailerID) {
			continue
		}
		items = append(items, orderRecord)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (r *inMemoryOrderRepo) ListWarehouseOrdersByDeliveryWindow(_ context.Context, warehouseID string, from, to time.Time, limit int) ([]order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 200
	}
	items := make([]order.Order, 0, limit)
	for _, orderRecord := range r.byID {
		if orderRecord.WarehouseID != strings.TrimSpace(warehouseID) || orderRecord.RequestedDeliveryDate == nil {
			continue
		}
		requested := orderRecord.RequestedDeliveryDate.UTC()
		if requested.Before(from.UTC()) || !requested.Before(to.UTC()) {
			continue
		}
		items = append(items, orderRecord)
	}
	sort.Slice(items, func(i, j int) bool {
		left := items[i].RequestedDeliveryDate
		right := items[j].RequestedDeliveryDate
		if left == nil || right == nil {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		if left.Equal(*right) {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		return left.After(*right)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (r *inMemoryOrderRepo) ListDueAutoConfirmOrders(_ context.Context, before time.Time, limit int) ([]order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 50
	}
	items := make([]order.Order, 0, limit)
	for _, orderRecord := range r.byID {
		if orderRecord.AutoConfirmAt == nil || orderRecord.Source != order.OrderSourceAIPreorder || orderRecord.ConfirmationStatus != order.ConfirmationStatusPending {
			continue
		}
		if orderRecord.AutoConfirmAt.After(before.UTC()) {
			continue
		}
		items = append(items, orderRecord)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].AutoConfirmAt.Before(*items[j].AutoConfirmAt)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

// ── Scaffold in-memory payment repository ─────────────────────────────────
// Used when Spanner is unavailable in local/fallback environments.

type inMemoryPaymentRepo struct {
	mu             sync.RWMutex
	sessions       map[string]payment.SessionRecord
	attempts       map[string]payment.PaymentAttemptRecord
	chargebacks    map[string]payment.ChargebackRecord
	reversals      map[string]payment.ReversalRecord
	webhooks       map[string]payment.WebhookRecord
	ledgerEntries  map[string]payment.LedgerEntryRecord
	outboxAppender outboxEventAppender
}

func newInMemoryPaymentRepo(outboxAppender outboxEventAppender) *inMemoryPaymentRepo {
	return &inMemoryPaymentRepo{
		sessions:       make(map[string]payment.SessionRecord),
		attempts:       make(map[string]payment.PaymentAttemptRecord),
		chargebacks:    make(map[string]payment.ChargebackRecord),
		reversals:      make(map[string]payment.ReversalRecord),
		webhooks:       make(map[string]payment.WebhookRecord),
		ledgerEntries:  make(map[string]payment.LedgerEntryRecord),
		outboxAppender: outboxAppender,
	}
}

func (r *inMemoryPaymentRepo) CreateSession(ctx context.Context, s payment.SessionRecord, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.sessions[s.SessionID] = s
	r.ledgerEntries["pledger_session_"+s.SessionID] = payment.LedgerEntryRecord{
		LedgerEntryID: "pledger_session_" + s.SessionID,
		SessionID:     s.SessionID,
		OrderID:       s.OrderID,
		SupplierID:    s.SupplierID,
		RetailerID:    s.RetailerID,
		Gateway:       strings.ToUpper(strings.TrimSpace(s.Gateway)),
		EntryType:     "SESSION_" + strings.ToUpper(strings.TrimSpace(s.Status)),
		AmountMinor:   s.AmountMinor,
		Currency:      strings.ToUpper(strings.TrimSpace(s.Currency)),
		ReferenceID:   s.SessionID,
		Source:        "payment.session",
		OccurredAt:    s.UpdatedAt,
		CreatedAt:     s.UpdatedAt,
	}
	_ = ctx
	return nil
}

func (r *inMemoryPaymentRepo) CreateSessionWithAttempt(ctx context.Context, s payment.SessionRecord, a payment.PaymentAttemptRecord, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.sessions[s.SessionID] = s
	r.attempts[a.AttemptID] = a
	r.ledgerEntries["pledger_session_"+s.SessionID] = payment.LedgerEntryRecord{
		LedgerEntryID: "pledger_session_" + s.SessionID,
		SessionID:     s.SessionID,
		OrderID:       s.OrderID,
		SupplierID:    s.SupplierID,
		RetailerID:    s.RetailerID,
		Gateway:       strings.ToUpper(strings.TrimSpace(s.Gateway)),
		EntryType:     "SESSION_" + strings.ToUpper(strings.TrimSpace(s.Status)),
		AmountMinor:   s.AmountMinor,
		Currency:      strings.ToUpper(strings.TrimSpace(s.Currency)),
		ReferenceID:   s.SessionID,
		Source:        "payment.session",
		OccurredAt:    s.UpdatedAt,
		CreatedAt:     s.UpdatedAt,
	}
	_ = ctx
	return nil
}

func (r *inMemoryPaymentRepo) SaveAttempt(ctx context.Context, a payment.PaymentAttemptRecord, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.attempts[a.AttemptID] = a
	_ = ctx
	return nil
}

func (r *inMemoryPaymentRepo) SaveChargeback(ctx context.Context, c payment.ChargebackRecord, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.chargebacks[c.ChargebackID] = c
	r.ledgerEntries["pledger_chargeback_"+c.ChargebackID] = payment.LedgerEntryRecord{
		LedgerEntryID: "pledger_chargeback_" + c.ChargebackID,
		OrderID:       c.OrderID,
		SupplierID:    c.SupplierID,
		RetailerID:    c.RetailerID,
		Gateway:       strings.ToUpper(strings.TrimSpace(c.Gateway)),
		EntryType:     "CHARGEBACK_RECORDED",
		AmountMinor:   c.AmountMinor,
		Currency:      strings.ToUpper(strings.TrimSpace(c.Currency)),
		ReferenceID:   c.ChargebackID,
		Source:        "payment.chargeback",
		OccurredAt:    c.CreatedAt,
		CreatedAt:     c.CreatedAt,
	}
	_ = ctx
	return nil
}

func (r *inMemoryPaymentRepo) SaveReversal(ctx context.Context, rev payment.ReversalRecord, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.reversals[rev.ReversalID] = rev
	r.ledgerEntries["pledger_reversal_"+rev.ReversalID] = payment.LedgerEntryRecord{
		LedgerEntryID: "pledger_reversal_" + rev.ReversalID,
		SessionID:     rev.SessionID,
		SupplierID:    rev.SupplierID,
		Gateway:       strings.ToUpper(strings.TrimSpace(rev.Gateway)),
		EntryType:     "CHARGEBACK_REVERSAL_RECORDED",
		AmountMinor:   rev.AmountMinor,
		Currency:      strings.ToUpper(strings.TrimSpace(rev.Currency)),
		ReferenceID:   rev.ReversalID,
		Source:        "payment.chargeback_reversal",
		OccurredAt:    rev.CreatedAt,
		CreatedAt:     rev.CreatedAt,
	}
	_ = ctx
	return nil
}

func (r *inMemoryPaymentRepo) SaveWebhook(ctx context.Context, w payment.WebhookRecord, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.webhooks[w.WebhookID] = w
	r.ledgerEntries["pledger_webhook_"+w.WebhookID] = payment.LedgerEntryRecord{
		LedgerEntryID: "pledger_webhook_" + w.WebhookID,
		SessionID:     w.SessionID,
		OrderID:       w.OrderID,
		SupplierID:    w.SupplierID,
		RetailerID:    w.RetailerID,
		Gateway:       strings.ToUpper(strings.TrimSpace(w.Gateway)),
		EntryType:     "WEBHOOK_" + strings.ToUpper(strings.TrimSpace(w.Status)),
		AmountMinor:   w.AmountMinor,
		Currency:      strings.ToUpper(strings.TrimSpace(w.Currency)),
		ReferenceID:   w.TransactionID,
		Source:        "payment.webhook",
		OccurredAt:    w.ReceivedAt,
		CreatedAt:     w.ReceivedAt,
	}
	_ = ctx
	return nil
}

func (r *inMemoryPaymentRepo) ListLedgerEntries(_ context.Context, q payment.LedgerQuery) ([]payment.LedgerEntryRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	limit := q.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	supplierID := strings.TrimSpace(q.SupplierID)
	orderID := strings.TrimSpace(q.OrderID)
	sessionID := strings.TrimSpace(q.SessionID)
	gateway := strings.ToUpper(strings.TrimSpace(q.Gateway))
	entryType := strings.ToUpper(strings.TrimSpace(q.EntryType))
	items := make([]payment.LedgerEntryRecord, 0, len(r.ledgerEntries))
	for _, entry := range r.ledgerEntries {
		if supplierID != "" && entry.SupplierID != supplierID {
			continue
		}
		if orderID != "" && entry.OrderID != orderID {
			continue
		}
		if sessionID != "" && entry.SessionID != sessionID {
			continue
		}
		if gateway != "" && strings.ToUpper(strings.TrimSpace(entry.Gateway)) != gateway {
			continue
		}
		if entryType != "" && strings.ToUpper(strings.TrimSpace(entry.EntryType)) != entryType {
			continue
		}
		if q.OccurredFrom != nil && entry.OccurredAt.Before(q.OccurredFrom.UTC()) {
			continue
		}
		if q.OccurredTo != nil && entry.OccurredAt.After(q.OccurredTo.UTC()) {
			continue
		}
		items = append(items, entry)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].OccurredAt.Equal(items[j].OccurredAt) {
			return items[i].CreatedAt.After(items[j].CreatedAt)
		}
		return items[i].OccurredAt.After(items[j].OccurredAt)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (r *inMemoryPaymentRepo) SummarizeLedgerEntries(_ context.Context, q payment.SettlementAuthorityQuery) ([]payment.SettlementAuthorityRow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	groupLimit := q.GroupLimit
	if groupLimit <= 0 || groupLimit > 1000 {
		groupLimit = 200
	}

	supplierID := strings.TrimSpace(q.SupplierID)
	gateway := strings.ToUpper(strings.TrimSpace(q.Gateway))
	entryType := strings.ToUpper(strings.TrimSpace(q.EntryType))
	groups := make(map[string]payment.SettlementAuthorityRow)

	for _, entry := range r.ledgerEntries {
		if supplierID != "" && entry.SupplierID != supplierID {
			continue
		}
		if gateway != "" && strings.ToUpper(strings.TrimSpace(entry.Gateway)) != gateway {
			continue
		}
		if entryType != "" && strings.ToUpper(strings.TrimSpace(entry.EntryType)) != entryType {
			continue
		}
		if q.OccurredFrom != nil && entry.OccurredAt.Before(q.OccurredFrom.UTC()) {
			continue
		}
		if q.OccurredTo != nil && entry.OccurredAt.After(q.OccurredTo.UTC()) {
			continue
		}

		key := strings.Join([]string{
			strings.ToUpper(strings.TrimSpace(entry.Gateway)),
			strings.ToUpper(strings.TrimSpace(entry.EntryType)),
			strings.ToUpper(strings.TrimSpace(entry.Currency)),
		}, "|")
		row := groups[key]
		if row.Gateway == "" {
			row = payment.SettlementAuthorityRow{
				Gateway:          strings.ToUpper(strings.TrimSpace(entry.Gateway)),
				EntryType:        strings.ToUpper(strings.TrimSpace(entry.EntryType)),
				Currency:         strings.ToUpper(strings.TrimSpace(entry.Currency)),
				FirstOccurredAt:  entry.OccurredAt,
				LastOccurredAt:   entry.OccurredAt,
				EntryCount:       0,
				AmountMinorTotal: 0,
			}
		}
		row.EntryCount++
		row.AmountMinorTotal += entry.AmountMinor
		if entry.OccurredAt.Before(row.FirstOccurredAt) {
			row.FirstOccurredAt = entry.OccurredAt
		}
		if entry.OccurredAt.After(row.LastOccurredAt) {
			row.LastOccurredAt = entry.OccurredAt
		}
		groups[key] = row
	}

	rows := make([]payment.SettlementAuthorityRow, 0, len(groups))
	for _, row := range groups {
		rows = append(rows, row)
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Gateway == rows[j].Gateway {
			if rows[i].EntryType == rows[j].EntryType {
				return rows[i].Currency < rows[j].Currency
			}
			return rows[i].EntryType < rows[j].EntryType
		}
		return rows[i].Gateway < rows[j].Gateway
	})

	if len(rows) > groupLimit {
		rows = rows[:groupLimit]
	}
	return rows, nil
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

// ── Scaffold in-memory retailer repository ─────────────────────────────────
// Replaced wholesale by the Spanner-backed implementation once available.
// Keeps the build green and lets the retailer registration handler exercise
// its happy path under unit tests.

type inMemoryRetailerRepo struct {
	mu             sync.RWMutex
	byID           map[string]retailer.Retailer
	byPhone        map[string]string // phone -> id
	outboxAppender outboxEventAppender
}

func newInMemoryRetailerRepo(outboxAppender outboxEventAppender) *inMemoryRetailerRepo {
	return &inMemoryRetailerRepo{
		byID:           make(map[string]retailer.Retailer),
		byPhone:        make(map[string]string),
		outboxAppender: outboxAppender,
	}
}

func (r *inMemoryRetailerRepo) CreateRetailer(ctx context.Context, ret retailer.Retailer, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byPhone[ret.Phone]; exists {
		return fmt.Errorf("retailer phone already registered")
	}
	if emit != nil {
		// Scaffold txn: persist the row and the outbox event in-memory together.
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	if ret.CreatedAt.IsZero() {
		ret.CreatedAt = time.Now().UTC()
	}
	r.byID[ret.RetailerID] = ret
	r.byPhone[ret.Phone] = ret.RetailerID
	return nil
}

func (r *inMemoryRetailerRepo) FindByPhone(_ context.Context, phone string) (retailer.Retailer, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byPhone[phone]
	if !ok {
		return retailer.Retailer{}, false, nil
	}
	return r.byID[id], true, nil
}

func (r *inMemoryRetailerRepo) GetRetailer(_ context.Context, retailerID string) (retailer.Retailer, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ret, ok := r.byID[retailerID]
	return ret, ok, nil
}

func (r *inMemoryRetailerRepo) UpdateRetailer(ctx context.Context, ret retailer.Retailer, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byID[ret.RetailerID]; !exists {
		return fmt.Errorf("retailer not found")
	}
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	ret.UpdatedAt = time.Now().UTC()
	r.byID[ret.RetailerID] = ret
	r.byPhone[ret.Phone] = ret.RetailerID
	_ = ctx
	return nil
}

func (r *inMemoryRetailerRepo) ListRetailersBySupplier(_ context.Context, supplierID string) ([]retailer.Retailer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]retailer.Retailer, 0, len(r.byID))
	for _, ret := range r.byID {
		if supplierID != "" && ret.SupplierID != supplierID {
			continue
		}
		out = append(out, ret)
	}
	return out, nil
}

func (r *inMemoryRetailerRepo) GetSupplierPricingRule(_ context.Context, _ string) (retailer.SupplierPricingRule, bool, error) {
	return retailer.SupplierPricingRule{}, false, nil
}

func (r *inMemoryRetailerRepo) ListTrackingOrders(_ context.Context, _ string, _ int) ([]retailer.TrackingOrder, error) {
	return []retailer.TrackingOrder{}, nil
}

func (r *inMemoryRetailerRepo) ListRecentReceipts(_ context.Context, _ string, _ int) ([]retailer.TrackingOrder, error) {
	return []retailer.TrackingOrder{}, nil
}

// inMemoryTxnBuffer collects outbox events for the scaffold path. A future
// Spanner adapter will replace this with a real BufferWrite-backed buffer.
type inMemoryTxnBuffer struct {
	events []outbox.Event
	audits []outbox.AuditEntry
}

func (b *inMemoryTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

func (b *inMemoryTxnBuffer) BufferAudit(_ context.Context, e outbox.AuditEntry) error {
	b.audits = append(b.audits, e)
	return nil
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

// ── Scaffold in-memory supplier profile repository ─────────────────────────
// Production swaps this for a Spanner-backed implementation that runs every
// UpdateProfile inside a ReadWriteTransaction and writes the OutboxEvents row
// atomically via the supplied TxnBuffer.

type inMemorySupplierRepo struct {
	mu                  sync.RWMutex
	profiles            map[string]supplier.Profile
	authByPhone         map[string]supplier.SupplierAuthRecord
	topologyBySupp      map[string]supplier.SupplierTopology
	orgMembersBySupp    map[string][]supplier.SupplierOrgMember
	fleetDriversBySupp  map[string][]supplier.SupplierFleetDriver
	fleetVehiclesBySupp map[string][]supplier.SupplierFleetVehicle
	pricingBySupp       map[string]supplier.SupplierPricingRule
	aiRecommendations   map[string]supplier.AIRecommendation
	outboxAppender      outboxEventAppender
}

func newInMemorySupplierRepo(outboxAppender outboxEventAppender) *inMemorySupplierRepo {
	return &inMemorySupplierRepo{
		profiles:            make(map[string]supplier.Profile),
		authByPhone:         make(map[string]supplier.SupplierAuthRecord),
		topologyBySupp:      make(map[string]supplier.SupplierTopology),
		orgMembersBySupp:    make(map[string][]supplier.SupplierOrgMember),
		fleetDriversBySupp:  make(map[string][]supplier.SupplierFleetDriver),
		fleetVehiclesBySupp: make(map[string][]supplier.SupplierFleetVehicle),
		pricingBySupp:       make(map[string]supplier.SupplierPricingRule),
		aiRecommendations:   make(map[string]supplier.AIRecommendation),
		outboxAppender:      outboxAppender,
	}
}

func (r *inMemorySupplierRepo) GetProfile(_ context.Context, supplierID string) (supplier.Profile, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.profiles[supplierID]
	return p, ok, nil
}

func (r *inMemorySupplierRepo) CountSuppliers(_ context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.profiles) == 0 {
		return 1, nil
	}
	return int64(len(r.profiles)), nil
}

func (r *inMemorySupplierRepo) UpdateProfile(ctx context.Context, p supplier.Profile, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.profiles[p.SupplierID] = p
	if _, ok := r.topologyBySupp[p.SupplierID]; !ok && (p.WarehouseLat != 0 || p.WarehouseLng != 0) {
		warehouseName := strings.TrimSpace(p.WarehouseName)
		if warehouseName == "" {
			warehouseName = "Primary Warehouse"
		}
		topology := supplier.SupplierTopology{
			Warehouses: []supplier.WarehouseNode{{
				WarehouseID:      "wh_primary_" + strings.TrimSpace(p.SupplierID),
				Name:             warehouseName,
				Lat:              p.WarehouseLat,
				Lng:              p.WarehouseLng,
				CoverageRadiusKm: 10,
				IsActive:         true,
				IsOnShift:        true,
				CreatedAt:        p.RegisteredAt,
				UpdatedAt:        p.UpdatedAt,
			}},
			Factories: make([]supplier.FactoryNode, 0, p.FactoryCount),
		}
		for i := 0; i < p.FactoryCount; i++ {
			topology.Factories = append(topology.Factories, supplier.FactoryNode{
				FactoryID: "fc_" + strings.TrimSpace(p.SupplierID) + "_" + strconv.Itoa(i+1),
				Name:      "Factory " + strconv.Itoa(i+1),
				Lat:       p.WarehouseLat,
				Lng:       p.WarehouseLng,
				IsActive:  true,
				CreatedAt: p.RegisteredAt,
				UpdatedAt: p.UpdatedAt,
			})
		}
		r.topologyBySupp[p.SupplierID] = topology
	}
	if strings.TrimSpace(p.AuthPasswordHash) != "" && strings.TrimSpace(p.Phone) != "" {
		userID := strings.TrimSpace(p.AuthUserID)
		if userID == "" {
			userID = "root_" + p.SupplierID
		}
		r.authByPhone[strings.TrimSpace(p.Phone)] = supplier.SupplierAuthRecord{
			UserID:       userID,
			SupplierID:   p.SupplierID,
			Phone:        strings.TrimSpace(p.Phone),
			PasswordHash: p.AuthPasswordHash,
			IsConfigured: p.IsConfigured,
		}
	}
	return nil
}

func (r *inMemorySupplierRepo) GetAuthByPhone(_ context.Context, phone string) (supplier.SupplierAuthRecord, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.authByPhone[strings.TrimSpace(phone)]
	return rec, ok, nil
}

func (r *inMemorySupplierRepo) GetTopology(_ context.Context, supplierID string) (supplier.SupplierTopology, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	topo, ok := r.topologyBySupp[supplierID]
	if !ok {
		return supplier.SupplierTopology{}, nil
	}
	return cloneSupplierTopology(topo), nil
}

func (r *inMemorySupplierRepo) ReplaceTopology(ctx context.Context, supplierID string, topology supplier.SupplierTopology, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.topologyBySupp[supplierID] = cloneSupplierTopology(topology)
	return nil
}

func (r *inMemorySupplierRepo) ListOrgMembers(_ context.Context, supplierID string) ([]supplier.SupplierOrgMember, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rows := r.orgMembersBySupp[supplierID]
	return append([]supplier.SupplierOrgMember(nil), rows...), nil
}

func (r *inMemorySupplierRepo) CreateOrgMember(ctx context.Context, member supplier.CreateOrgMemberParams, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.orgMembersBySupp[member.SupplierID] = append(r.orgMembersBySupp[member.SupplierID], supplier.SupplierOrgMember{
		UserID:              member.UserID,
		SupplierID:          member.SupplierID,
		Name:                member.Name,
		Email:               member.Email,
		Phone:               member.Phone,
		SupplierRole:        member.SupplierRole,
		AssignedWarehouseID: member.AssignedWarehouseID,
		AssignedFactoryID:   member.AssignedFactoryID,
		IsActive:            member.IsActive,
		CreatedAt:           member.CreatedAt,
		UpdatedAt:           member.UpdatedAt,
	})
	return nil
}

func (r *inMemorySupplierRepo) ListFleetDrivers(_ context.Context, supplierID string) ([]supplier.SupplierFleetDriver, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rows := r.fleetDriversBySupp[supplierID]
	return append([]supplier.SupplierFleetDriver(nil), rows...), nil
}

func (r *inMemorySupplierRepo) CreateFleetDriver(ctx context.Context, driverParams supplier.CreateFleetDriverParams, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.fleetDriversBySupp[driverParams.SupplierID] = append(r.fleetDriversBySupp[driverParams.SupplierID], supplier.SupplierFleetDriver{
		DriverID:     driverParams.DriverID,
		SupplierID:   driverParams.SupplierID,
		Name:         driverParams.Name,
		Phone:        driverParams.Phone,
		HomeNodeType: driverParams.HomeNodeType,
		HomeNodeID:   driverParams.HomeNodeID,
		VehicleID:    driverParams.VehicleID,
		IsActive:     driverParams.IsActive,
		CreatedAt:    driverParams.CreatedAt,
		UpdatedAt:    driverParams.UpdatedAt,
	})
	return nil
}

func (r *inMemorySupplierRepo) ListFleetVehicles(_ context.Context, supplierID string) ([]supplier.SupplierFleetVehicle, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rows := r.fleetVehiclesBySupp[supplierID]
	return append([]supplier.SupplierFleetVehicle(nil), rows...), nil
}

func (r *inMemorySupplierRepo) CreateFleetVehicle(ctx context.Context, vehicle supplier.CreateFleetVehicleParams, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.fleetVehiclesBySupp[vehicle.SupplierID] = append(r.fleetVehiclesBySupp[vehicle.SupplierID], supplier.SupplierFleetVehicle{
		VehicleID:    vehicle.VehicleID,
		SupplierID:   vehicle.SupplierID,
		Label:        vehicle.Label,
		LicensePlate: vehicle.LicensePlate,
		HomeNodeType: vehicle.HomeNodeType,
		HomeNodeID:   vehicle.HomeNodeID,
		VehicleClass: vehicle.VehicleClass,
		MaxVolumeVU:  vehicle.MaxVolumeVU,
		IsActive:     vehicle.IsActive,
		CreatedAt:    vehicle.CreatedAt,
		UpdatedAt:    vehicle.UpdatedAt,
	})
	return nil
}

func (r *inMemorySupplierRepo) GetPricingRule(_ context.Context, supplierID string) (supplier.SupplierPricingRule, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rule, ok := r.pricingBySupp[supplierID]
	return rule, ok, nil
}

func (r *inMemorySupplierRepo) UpsertPricingRule(ctx context.Context, rule supplier.SupplierPricingRule, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}

	existing, exists := r.pricingBySupp[rule.SupplierID]
	if exists {
		rule.RuleVersion = existing.RuleVersion + 1
	} else if rule.RuleVersion <= 0 {
		rule.RuleVersion = 1
	}
	if strings.TrimSpace(rule.Currency) == "" {
		if profile, ok := r.profiles[rule.SupplierID]; ok && strings.TrimSpace(profile.Currency) != "" {
			rule.Currency = profile.Currency
		}
	}
	if rule.UpdatedAt.IsZero() {
		rule.UpdatedAt = time.Now().UTC()
	}
	r.pricingBySupp[rule.SupplierID] = rule
	return nil
}

func (r *inMemorySupplierRepo) ListAIRecommendations(_ context.Context, supplierID string, query supplier.AIRecommendationQuery) ([]supplier.AIRecommendation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	status := strings.ToUpper(strings.TrimSpace(query.Status))
	items := make([]supplier.AIRecommendation, 0, limit)
	for _, item := range r.aiRecommendations {
		if item.SupplierID != supplierID {
			continue
		}
		if status != "" && !strings.EqualFold(item.Status, status) {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].GeneratedAt > items[j].GeneratedAt })
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (r *inMemorySupplierRepo) RecordAIRecommendationDecision(ctx context.Context, supplierID string, decision supplier.AIRecommendationDecision, emit func(outbox.TxnBuffer, supplier.AIRecommendation) error) (supplier.AIRecommendation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.aiRecommendations[strings.TrimSpace(decision.RecommendationID)]
	if !ok || item.SupplierID != supplierID {
		return supplier.AIRecommendation{}, supplier.ErrAIRecommendationNotFound
	}
	item.Status = supplierDecisionStatus(decision.Decision)
	item.Decision = decision.Decision
	item.DecisionNote = strings.TrimSpace(decision.Note)
	item.DecidedBy = strings.TrimSpace(decision.DecidedBy)
	item.DecidedAt = decision.DecidedAt.UTC().Format(time.RFC3339Nano)
	item.UpdatedAt = item.DecidedAt
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn, item); err != nil {
			return supplier.AIRecommendation{}, err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return supplier.AIRecommendation{}, err
			}
		}
	}
	r.aiRecommendations[item.RecommendationID] = item
	return item, nil
}

func supplierDecisionStatus(decision string) string {
	if strings.EqualFold(decision, "REOPENED") {
		return "PENDING"
	}
	return strings.ToUpper(strings.TrimSpace(decision))
}

func cloneSupplierTopology(src supplier.SupplierTopology) supplier.SupplierTopology {
	out := supplier.SupplierTopology{
		Warehouses: make([]supplier.WarehouseNode, 0, len(src.Warehouses)),
		Factories:  make([]supplier.FactoryNode, 0, len(src.Factories)),
	}
	out.Warehouses = append(out.Warehouses, src.Warehouses...)
	out.Factories = append(out.Factories, src.Factories...)
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

func (r *inMemoryOrderRepo) ListManifestOrders(_ context.Context, manifestID string) ([]order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []order.Order
	for _, o := range r.byID {
		if o.ManifestID == manifestID {
			out = append(out, o)
		}
	}
	return out, nil
}

func (r *inMemoryOrderRepo) UpdateOrder(ctx context.Context, o order.Order, _ []order.DeliveryProofArtifact, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byID[o.OrderID]; !exists {
		return fmt.Errorf("order not found: %s", o.OrderID)
	}
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.byID[o.OrderID] = o
	return nil
}
