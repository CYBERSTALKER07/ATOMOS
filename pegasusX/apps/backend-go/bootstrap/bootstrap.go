// Package bootstrap is the composition root for the pegasusX backend.
//
// It owns construction of every long-lived singleton: Spanner client, Redis
// cache, Kafka writers, WebSocket hubs, services, and middleware. Domain
// packages receive their dependencies through narrow Deps structs; they never
// reach into bootstrap.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/driver"
	"github.com/pegasusx/pegasusx/apps/backend-go/factory"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafka"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/payload"
	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailer"
	"github.com/pegasusx/pegasusx/apps/backend-go/seed"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouse"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

// Config carries the runtime parameters. Loaded from environment by LoadConfig.
type Config struct {
	HTTPPort string

	SpannerEmulatorHost string
	SpannerProject      string
	SpannerInstance     string
	SpannerDatabase     string

	RedisAddr string

	KafkaBrokers   string
	KafkaTopicMain string

	JWTSecret string
	JWTIssuer string

	FirebaseAuthEnabled             bool
	FirebaseProjectID               string
	FirebaseCertsURL                string
	GlobalPayWebhookSecret          string
	AdyenWebhookSecret              string
	StripeWebhookSecret             string
	AirwallexDirectExecutionEnabled bool

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
}

// App holds every long-lived singleton. Wire new app-wide dependencies here,
// never as package-level globals.
type App struct {
	Config               *Config
	Cache                *cache.Cache
	Idempotency          idempotency.Store
	Supplier             seed.Supplier
	SupplierService      *supplier.Service
	RetailerService      *retailer.Service
	RetailerProximity    *retailer.RetailerProximityService
	DriverService        *driver.Service
	FactoryService       *factory.Service
	PayloadService       *payload.Service
	PaymentService       *payment.Service
	WarehouseService     *warehouse.Service
	OrderService         *order.Service
	RetailerHub          *ws.Hub
	SupplierHub          *ws.Hub
	DriverHub            *ws.Hub
	PayloadHub           *ws.Hub
	WarehouseHub         *ws.Hub
	FactoryHub           *ws.Hub
	TelemetryHub         *ws.Hub
	OutboxRelay          *outbox.Relay
	NotificationConsumer *kafka.Consumer
	Reliability          *ReliabilityMiddleware
	Spanner              *spanner.Client
	cleanup              []func()
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
		KafkaBrokers:                    envOr("KAFKA_BROKERS", "localhost:9092"),
		KafkaTopicMain:                  envOr("KAFKA_TOPIC_MAIN", "pegasusx-main"),
		JWTSecret:                       envOr("JWT_SECRET", "dev-only-change-me"),
		JWTIssuer:                       envOr("JWT_ISSUER", "pegasusx-dev"),
		FirebaseAuthEnabled:             envBool("FIREBASE_AUTH_ENABLED", false),
		FirebaseProjectID:               envOr("FIREBASE_PROJECT_ID", ""),
		FirebaseCertsURL:                envOr("FIREBASE_CERTS_URL", "https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com"),
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
		RequireInfraAdapters:            envBool("REQUIRE_INFRA_ADAPTERS", false),
		ReliabilityEnabled:              envBool("RELIABILITY_MIDDLEWARE_ENABLED", true),
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET required")
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
	if cfg.RequireInfraAdapters {
		log.Info("strict infra adapter mode enabled")
	}

	cacheBackend := cache.Backend(cache.NewInMemoryBackend())
	redisEnabled := false
	if redisBackend, err := newRedisRuntimeAdapter(cfg.RedisAddr); err != nil {
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
		cacheBackend = redisBackend
		redisEnabled = true
		log.Info("redis cache backend enabled", "addr", cfg.RedisAddr)
	}
	if cfg.RequireInfraAdapters && !redisEnabled {
		return nil, fmt.Errorf("require infra adapters: redis unavailable")
	}
	cacheClient := cache.New(cacheBackend, log)
	idemStore := idempotency.NewInMemoryStore()
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
	outboxRelay := outbox.NewRelay(relayStore, outboxPublisher, outbox.RelayConfig{}, log)

	var retailerRepo retailer.Repository
	var supplierRepo supplier.Repository

	if spannerClient != nil {
		retailerRepo = retailer.NewSpannerRepository(spannerClient)
		supplierRepo = supplier.NewSpannerRepository(spannerClient)
		log.Info("retailer and supplier repositories enabled", "backend", "spanner")
	} else {
		retailerRepo = newInMemoryRetailerRepo(outboxAppender)
		supplierRepo = newInMemorySupplierRepo(outboxAppender)
		log.Warn("retailer and supplier repository fallback enabled", "backend", "in-memory")
	}

	var seedRepo seed.Repository
	if spannerClient != nil {
		seedRepo = &runtimeSeedRepository{client: spannerClient}
	}
	supplierSeed, err := seed.EnsureSupplier(ctx, seedRepo,
		cfg.SeedSupplierName, cfg.SeedSupplierCountry, cfg.SeedSupplierCurrency, log)
	if err != nil {
		return nil, fmt.Errorf("seed supplier: %w", err)
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

	retailerSvc := retailer.NewService(retailer.ServiceConfig{
		Repo:        retailerRepo,
		Cache:       cacheClient,
		Idem:        idemStore,
		SupplierID:  supplierSeed.SupplierID,
		CountryCode: cfg.SeedSupplierCountry,
		Log:         log,
	})

	supplierSvc := supplier.NewService(supplier.ServiceConfig{
		Repo:         supplierRepo,
		Cache:        cacheClient,
		Idem:         idemStore,
		SupplierID:   supplierSeed.SupplierID,
		Country:      cfg.SeedSupplierCountry,
		Currency:     cfg.SeedSupplierCurrency,
		JWTSecret:    cfg.JWTSecret,
		JWTIssuer:    cfg.JWTIssuer,
		JWTTTL:       24 * time.Hour,
		CookieSecure: false,
		Log:          log,
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
		orderRepo = newInMemoryOrderRepo(outboxAppender)
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
	orderSvc := order.NewService(order.ServiceConfig{
		Repo:        orderRepo,
		Cache:       cacheClient,
		Warehouse:   orderWarehouseResolver,
		SupplierID:  supplierSeed.SupplierID,
		Currency:    cfg.SeedSupplierCurrency,
		RetailerHub: retailerHub,
		SupplierHub: supplierHub,
		Log:         log,
	})

	var driverRepo driver.Repository
	var factoryRepo factory.Repository
	var payloadRepo payload.Repository
	if spannerClient != nil {
		driverRepo = driver.NewSpannerRepository(spannerClient)
		factoryRepo = factory.NewSpannerRepository(spannerClient)
		payloadRepo = payload.NewSpannerRepository(spannerClient)
		log.Info("factory and payload repositories enabled", "backend", "spanner")
	} else {
		driverRepo = driver.NewInMemoryRepository()
		factoryRepo = factory.NewInMemoryRepository()
		payloadRepo = payload.NewInMemoryRepository()
		log.Warn("factory and payload repository fallback enabled", "backend", "in-memory")
	}

	driverSvc := driver.NewService(driver.ServiceConfig{
		Repo: driverRepo,
	})
	factorySvc := factory.NewService(factory.ServiceConfig{
		Repo:        factoryRepo,
		Cache:       cacheClient,
		SupplierHub: supplierHub,
		FactoryHub:  factoryHub,
		Log:         log,
		SupplierID:  supplierSeed.SupplierID,
		Currency:    cfg.SeedSupplierCurrency,
	})
	payloadSvc := payload.NewService(payload.ServiceConfig{
		Repo:        payloadRepo,
		Cache:       cacheClient,
		SupplierHub: supplierHub,
		PayloadHub:  payloadHub,
		Log:         log,
		SupplierID:  supplierSeed.SupplierID,
		Currency:    cfg.SeedSupplierCurrency,
	})
	paymentSvc := payment.NewService(payment.ServiceConfig{
		Repo:                            paymentRepo,
		Cache:                           cacheClient,
		Idem:                            idemStore,
		SupplierID:                      supplierSeed.SupplierID,
		Currency:                        cfg.SeedSupplierCurrency,
		GlobalPayWebhookSecret:          cfg.GlobalPayWebhookSecret,
		AdyenWebhookSecret:              cfg.AdyenWebhookSecret,
		StripeWebhookSecret:             cfg.StripeWebhookSecret,
		AirwallexDirectExecutionEnabled: cfg.AirwallexDirectExecutionEnabled,
		Log:                             log,
	})
	var warehouseRepo warehouse.Repository
	if spannerClient != nil {
		warehouseRepo = warehouse.NewSpannerRepository(spannerClient)
	} else {
		warehouseRepo = warehouse.NewInMemoryRepository()
	}

	warehouseSvc := warehouse.NewService(warehouse.ServiceConfig{
		Repo:         warehouseRepo,
		Cache:        cacheClient,
		SupplierHub:  supplierHub,
		WarehouseHub: warehouseHub,
		Log:          log,
		SupplierID:   supplierSeed.SupplierID,
		Currency:     cfg.SeedSupplierCurrency,
	})

	var reliabilityMiddleware *ReliabilityMiddleware
	if cfg.ReliabilityEnabled {
		reliabilityMiddleware = NewReliabilityMiddleware(DefaultReliabilityConfig())
		log.Info("reliability middleware enabled")
	}

	var notificationConsumer *kafka.Consumer
	if kafkaEnabled && cfg.KafkaTopicMain != "" {
		dispatcher := kafka.NewNotificationDispatcher(kafka.DispatcherDeps{
			RetailerHub:  retailerHub,
			SupplierHub:  supplierHub,
			DriverHub:    driverHub,
			WarehouseHub: warehouseHub,
		})
		notificationConsumer = kafka.NewConsumer(kafka.ConsumerDeps{
			Brokers: strings.Split(cfg.KafkaBrokers, ","),
			GroupID: "void-notification-dispatcher",
			Topic:   cfg.KafkaTopicMain,
			Handler: dispatcher.HandleEvent,
		})
		cleanup = append(cleanup, func() {
			if err := notificationConsumer.Close(); err != nil {
				log.Warn("notification consumer close failed", "err", err)
			}
		})
		log.Info("notification consumer enabled", "topic", cfg.KafkaTopicMain)
	}

	return &App{
		Config:               cfg,
		Cache:                cacheClient,
		Idempotency:          idemStore,
		Supplier:             supplierSeed,
		SupplierService:      supplierSvc,
		RetailerService:      retailerSvc,
		DriverService:        driverSvc,
		FactoryService:       factorySvc,
		PayloadService:       payloadSvc,
		PaymentService:       paymentSvc,
		WarehouseService:     warehouseSvc,
		OrderService:         orderSvc,
		RetailerHub:          retailerHub,
		SupplierHub:          supplierHub,
		DriverHub:            driverHub,
		PayloadHub:           payloadHub,
		WarehouseHub:         warehouseHub,
		FactoryHub:           factoryHub,
		TelemetryHub:         telemetryHub,
		NotificationConsumer: notificationConsumer,
		OutboxRelay:          outboxRelay,
		Reliability:          reliabilityMiddleware,
		Spanner:              spannerClient,
		cleanup:              cleanup,
	}, nil
}

// ── Scaffold in-memory order repository ────────────────────────────────────
// Production replaces this with a Spanner-backed implementation that runs
// CreateOrder inside a ReadWriteTransaction and writes both the Orders row
// and the OutboxEvents row atomically.

type inMemoryOrderRepo struct {
	mu             sync.RWMutex
	byID           map[string]order.Order
	outboxAppender outboxEventAppender
}

func newInMemoryOrderRepo(outboxAppender outboxEventAppender) *inMemoryOrderRepo {
	return &inMemoryOrderRepo{byID: make(map[string]order.Order), outboxAppender: outboxAppender}
}

func (r *inMemoryOrderRepo) CreateOrder(ctx context.Context, o order.Order, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byID[o.OrderID]; exists {
		return fmt.Errorf("order_id collision: %s", o.OrderID)
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

func (r *inMemoryOrderRepo) GetOrder(_ context.Context, orderID string) (order.Order, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, ok := r.byID[orderID]
	return o, ok, nil
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
	outboxAppender outboxEventAppender
}

func newInMemoryPaymentRepo(outboxAppender outboxEventAppender) *inMemoryPaymentRepo {
	return &inMemoryPaymentRepo{
		sessions:       make(map[string]payment.SessionRecord),
		attempts:       make(map[string]payment.PaymentAttemptRecord),
		chargebacks:    make(map[string]payment.ChargebackRecord),
		reversals:      make(map[string]payment.ReversalRecord),
		webhooks:       make(map[string]payment.WebhookRecord),
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
	_ = ctx
	return nil
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

// inMemoryTxnBuffer collects outbox events for the scaffold path. A future
// Spanner adapter will replace this with a real BufferWrite-backed buffer.
type inMemoryTxnBuffer struct {
	events []outbox.Event
}

func (b *inMemoryTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
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

// ── Scaffold in-memory supplier profile repository ─────────────────────────
// Production swaps this for a Spanner-backed implementation that runs every
// UpdateProfile inside a ReadWriteTransaction and writes the OutboxEvents row
// atomically via the supplied TxnBuffer.

type inMemorySupplierRepo struct {
	mu             sync.RWMutex
	profiles       map[string]supplier.Profile
	authByPhone    map[string]supplier.SupplierAuthRecord
	topologyBySupp map[string]supplier.SupplierTopology
	outboxAppender outboxEventAppender
}

func newInMemorySupplierRepo(outboxAppender outboxEventAppender) *inMemorySupplierRepo {
	return &inMemorySupplierRepo{
		profiles:       make(map[string]supplier.Profile),
		authByPhone:    make(map[string]supplier.SupplierAuthRecord),
		topologyBySupp: make(map[string]supplier.SupplierTopology),
		outboxAppender: outboxAppender,
	}
}

func (r *inMemorySupplierRepo) GetProfile(_ context.Context, supplierID string) (supplier.Profile, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.profiles[supplierID]
	return p, ok, nil
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

func (r *inMemoryOrderRepo) UpdateOrder(ctx context.Context, o order.Order, emit func(outbox.TxnBuffer) error) error {
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
