package bootstrap

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/segmentio/kafka-go"
	"golang.org/x/sync/singleflight"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"backend-go/auth"
	"backend-go/bootstrap/spannerrouter"
	"backend-go/cache"
	"backend-go/countrycfg"
	"backend-go/dispatch/optimizerclient"
	"backend-go/dispatch/plan"
	"backend-go/hotspot"
	"backend-go/internal/rpc/optimizergrpc"
	internalKafka "backend-go/kafka"
	"backend-go/notifications"
	"backend-go/order"
	"backend-go/outbox"
	"backend-go/payment"
	"backend-go/proximity"
	"backend-go/replenishment"
	"backend-go/secrets"
	"backend-go/settings"
	"backend-go/simulation"
	"backend-go/storage"
	"backend-go/supplier"
	"backend-go/telemetry"
	"backend-go/vault"
	"backend-go/ws"
	"config"
)

func configureProcessLogger(cfg *config.EnvConfig) {
	handlerOptions := &slog.HandlerOptions{Level: slog.LevelInfo}
	var handler slog.Handler
	if cfg.IsProduction() {
		handler = slog.NewJSONHandler(os.Stdout, handlerOptions)
	} else {
		handler = slog.NewTextHandler(os.Stdout, handlerOptions)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Bridge legacy log.Printf/log.Println call sites through the default
	// slog handler so bootstrap output stays in one consistent format.
	bridge := slog.NewLogLogger(handler, slog.LevelInfo)
	bridge.SetFlags(0)
	bridge.SetPrefix("")
	log.SetFlags(0)
	log.SetPrefix("")
	log.SetOutput(bridge.Writer())
}

// NewApp is the composition root. It initialises every long-lived dependency
// in the order imposed by their construction prerequisites and returns a
// fully-populated *App. Any fatal error during construction returns an
// unwrapped error — callers should log.Fatal on it rather than continue.
//
// The ctx argument controls cancellation of blocking initialisation calls
// (Spanner dial, GCS bucket probe, Secret Manager warmup). A timely cancel
// causes NewApp to surface a context error instead of hanging.
func NewApp(ctx context.Context, cfg *config.EnvConfig) (*App, error) {
	// ── 0. Process logging (text in dev, JSON in production) ─────────────
	configureProcessLogger(cfg)

	// ── 0b. OpenTelemetry TracerProvider ─────────────────────────────────
	// Must init BEFORE Spanner/Redis/Kafka so instrumented clients pick up
	// the global provider. Shutdown is chained into the SIGTERM handler.
	tracerShutdown, err := telemetry.InitProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("otel: %w", err)
	}

	// ── 1. Secret Manager (GCP SM with ENV fallback) ──────────────────────
	secrets.Init(ctx)

	// ── 2. Spanner ────────────────────────────────────────────────────────
	spannerClient, spannerDBName, spannerOpts, err := newSpannerClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("spanner: %w", err)
	}
	hotspot.ConfigureShardCount(cfg.SpannerHotShardCount)

	// SpannerRouter wraps the primary client with optional regional replicas.
	// In single-region mode (the default), it is a no-op: every call returns
	// spannerClient. In multiregion mode, enable_multiregion=true provisions
	// separate Spanner instances per region; add them here when their
	// connection strings are available in cfg.
	//
	// For now, always single-region. The Router API is live so handlers can
	// call SpannerRouter.For(h3Cell) and the routing will activate the moment
	// a second regional client is added without changing any handler code.
	spannerRouter := spannerrouter.NewSingleRegion(spannerClient)

	// ── 3. Cache (Redis) + Kafka writers + GCS ────────────────────────────
	c := cache.New(cfg.RedisAddress)
	c.StartHealthMonitor()
	c.StartInvalidationSubscriber(ctx)
	c.StartCacheWorkers(ctx)

	internalKafka.InitSyncWriter(cfg.KafkaBrokerAddress)
	internalKafka.InitCorrectionWriter(cfg.KafkaBrokerAddress)
	internalKafka.InitNotificationWriter(cfg.KafkaBrokerAddress)

	if err := storage.InitGCS(ctx, cfg.GCSBucketName); err != nil {
		slog.Warn("gcs init failed; image uploads offline", "err", err)
	}

	// ── 4. Payment + vault services ───────────────────────────────────────
	vaultSvc := &vault.Service{Spanner: spannerClient}
	sessionSvc := &payment.SessionService{Spanner: spannerClient, Cache: c}
	cardTokenSvc := &payment.CardTokenService{Spanner: spannerClient}
	cardsClient := payment.NewGlobalPayCardsClient()   // nil when GLOBAL_PAY_GATEWAY_BASE_URL unset
	directClient := payment.NewGlobalPayDirectClient() // nil when GLOBAL_PAY_GATEWAY_BASE_URL unset

	// ── 5. Country/platform/empathy/pricing services ──────────────────────
	countryCfgSvc := countrycfg.NewService(spannerClient)
	countryCfgSvc.AttachInvalidation(c)
	countrycfg.SeedDefaultConfigs(ctx, spannerClient)

	platformCfg := settings.NewPlatformConfig(spannerClient)
	empathySvc := &settings.EmpathyService{Client: spannerClient, Cache: c}
	supplierPricingSvc := supplier.NewPricingService(spannerClient)

	orderSvc := &order.OrderService{
		Client:       spannerClient,
		ReadRouter:   spannerRouter,
		Cache:        c,
		Vault:        vaultSvc,
		SessionSvc:   sessionSvc,
		CardTokenSvc: cardTokenSvc,
		DirectClient: directClient,
		FeeBP:        platformCfg.PlatformFeeBasisPoints(),
		Producer: &kafka.Writer{
			Addr:     kafka.TCP(cfg.KafkaBrokerAddress),
			Topic:    internalKafka.TopicMain,
			Balancer: &kafka.LeastBytes{},
		},
	}

	// ── 6. Proximity + telemetry FleetHub wiring ──────────────────────────
	proxEngine := &proximity.Engine{
		Redis:     cache.GetClient(), // nil-safe
		Spanner:   spannerClient,
		Producer:  orderSvc.Producer,
		ConfigSvc: &countrycfg.ProximityConfigAdapter{Svc: countryCfgSvc},
	}
	telemetry.FleetHub.ProximityEngine = proxEngine
	telemetry.FleetHub.Spanner = spannerClient
	telemetry.FleetHub.Buffer = telemetry.NewGPSBuffer(telemetry.FleetHub)
	if cache.GetClient() == nil {
		slog.Warn("proximity engine degraded; breach detection disabled", "reason", "redis offline")
	}

	// ── 7. Load shedder (priority-aware) ──────────────────────────────────
	backpressure := cache.NewBackpressureEngine(cache.DefaultBackpressureConfig())
	priorityGuard := cache.PrioritySheddingMiddleware(backpressure, 120, 60)

	// ── 8. WebSocket hubs ─────────────────────────────────────────────────
	retailerHub := ws.NewRetailerHub()
	driverHub := ws.NewDriverHub()
	payloaderHub := ws.NewPayloaderHub()
	supplierHub := ws.NewSupplierHub()
	warehouseHub := ws.NewWarehouseHub()

	// ── 9. Communication spine: FCM (primary) + Telegram (fallback) ───────
	fcm := initFCM(spannerClient)
	tg := notifications.NewTelegramClient()

	// ── 10. Firebase Auth (soft-fail into legacy HS256 mode) ──────────────
	if _, fbErr := auth.InitFirebaseAuth(ctx); fbErr != nil {
		slog.Warn("firebase auth init skipped; legacy JWT mode active", "err", fbErr)
	}

	// ── 11. Derived deps that require fcm ─────────────────────────────────
	shopClosedDeps, earlyCompleteDeps, negotiationDeps := buildOrderDeps(fcm, retailerHub, driverHub)

	// ── 12. Downstream services that need everything above ────────────────
	broadcastSvc := &notifications.BroadcastService{Spanner: spannerClient, FCM: fcm}
	reconcilerSvc := payment.NewReconcilerService(cfg, spannerClient)
	gpReconciler := &payment.GlobalPayReconciler{
		Spanner:       spannerClient,
		SessionSvc:    sessionSvc,
		VaultResolver: &vault.PaymentVaultAdapter{Svc: vaultSvc},
		Producer:      orderSvc.Producer,
		DriverHub:     driverHub,
		RetailerHub:   retailerHub,
	}
	replenishEngine := &replenishment.ReplenishmentEngine{Spanner: spannerClient, Producer: orderSvc.Producer}

	// ── 13. Phase 2 dispatch optimiser client (apps/ai-worker) ────────────
	// Nil client is the documented degraded mode — the orchestrator falls
	// straight through to the Phase 1 KMeans + binpack pipeline.
	var optimizerCli *optimizerclient.Client
	if cfg.OptimizerBaseURL != "" {
		optimizerCli = optimizerclient.New(cfg.OptimizerBaseURL, cfg.InternalAPIKey)
	}

	// ── 13a. gRPC optimizer client (preferred over HTTP when OPTIMIZER_GRPC_ADDR is set) ──
	// OPTIMIZER_GRPC_ADDR="" → nil (use HTTP), "xds" → xDS mesh, "host:port" → direct.
	optimizerGRPC, err := optimizergrpc.New(cfg.InternalAPIKey)
	if err != nil {
		return nil, fmt.Errorf("optimizer grpc client: %w", err)
	}

	// ── 13a. Shared dispatch source-attribution counters ──────────────────
	// Hosts atomic tallies for optimizer vs fallback paths. Shared between
	// the shadow/primary dispatch wires and the stealth simulation engine
	// so status endpoints can read a single aggregated source-of-truth.
	dispatchCounts := &plan.SourceCounters{}

	// ── 13b. Stealth simulation harness (gated) ───────────────────────────
	// Only armed when SIMULATION_ENABLED=true. When nil, the /v1/internal/sim
	// routes are not registered — see main.go for the gate.
	var simEngine *simulation.Engine
	if os.Getenv("SIMULATION_ENABLED") == "true" {
		simEngine = simulation.NewEngine(optimizerCli, dispatchCounts)
	}

	return &App{
		Config:            cfg,
		Spanner:           spannerClient,
		SpannerRouter:     spannerRouter,
		SpannerDBName:     spannerDBName,
		SpannerClientOpts: spannerOpts,
		Cache:             c,
		CacheFlight:       &singleflight.Group{},
		KafkaBroker:       cfg.KafkaBrokerAddress,
		Vault:             vaultSvc,
		SessionSvc:        sessionSvc,
		CardTokenSvc:      cardTokenSvc,
		CardsClient:       cardsClient,
		DirectClient:      directClient,
		GPReconciler:      gpReconciler,
		Order:             orderSvc,
		CountryConfig:     countryCfgSvc,
		PlatformConfig:    platformCfg,
		Empathy:           empathySvc,
		SupplierPricing:   supplierPricingSvc,
		ProximityEngine:   proxEngine,
		Replenishment:     replenishEngine,
		Broadcast:         broadcastSvc,
		Reconciler:        reconcilerSvc,
		ShopClosedDeps:    shopClosedDeps,
		EarlyCompleteDeps: earlyCompleteDeps,
		NegotiationDeps:   negotiationDeps,
		FCM:               fcm,
		Telegram:          tg,
		RetailerHub:       retailerHub,
		DriverHub:         driverHub,
		PayloaderHub:      payloaderHub,
		SupplierHub:       supplierHub,
		WarehouseHub:      warehouseHub,
		FleetHub:          telemetry.FleetHub,
		Outbox:            outbox.NewRelay(spannerClient, cfg.KafkaBrokerAddress, 2*time.Second, 100, 0),
		OptimizerClient:   optimizerCli,
		OptimizerGRPC:     optimizerGRPC,
		DispatchOptimizer: dispatchCounts,
		Simulation:        simEngine,
		Backpressure:      backpressure,
		PriorityGuard:     priorityGuard,
		CORSAllowlist:     resolveCORSAllowlist(cfg),
		TracerShutdown:    tracerShutdown,
	}, nil
}

// newSpannerClient dials the Spanner API (emulator-friendly) and returns
// the client alongside the database URI and the option.ClientOption slice
// used to dial — both surfaced so the inline DDL migrations in main can
// build a matching DatabaseAdminClient without re-deriving endpoint state.
// The session pool config matches the legacy main.go values — preserved
// verbatim to avoid any behavioural drift during the refactor.
func newSpannerClient(ctx context.Context, cfg *config.EnvConfig) (*spanner.Client, string, []option.ClientOption, error) {
	emulatorAddr := os.Getenv("SPANNER_EMULATOR_HOST")
	if emulatorAddr == "" {
		emulatorAddr = "localhost:9010"
		os.Setenv("SPANNER_EMULATOR_HOST", emulatorAddr)
	}
	opts := []option.ClientOption{
		option.WithEndpoint(emulatorAddr),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithoutAuthentication(),
	}
	dbName := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		cfg.SpannerProject, cfg.SpannerInstance, cfg.SpannerDatabase)
	client, err := spanner.NewClientWithConfig(ctx, dbName,
		spanner.ClientConfig{
			NumChannels: 100,
			SessionPoolConfig: spanner.SessionPoolConfig{
				MinOpened:           1000,
				MaxOpened:           4000,
				WriteSessions:       0.2,
				HealthCheckInterval: 5 * time.Minute,
			},
		},
		opts...,
	)
	if err != nil {
		return nil, "", nil, err
	}
	return client, dbName, opts, nil
}
