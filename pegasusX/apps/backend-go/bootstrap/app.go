// Package bootstrap is the composition root for the pegasusX backend.
//
// It owns construction of every long-lived singleton: Spanner client, Redis
// cache, Kafka writers, WebSocket hubs, services, and middleware. Domain
// packages receive their dependencies through narrow Deps structs; they never
// reach into bootstrap.
package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/allocation"
	"github.com/pegasusx/pegasusx/apps/backend-go/analytics"
	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/cashrecon"
	"github.com/pegasusx/pegasusx/apps/backend-go/catalog"
	"github.com/pegasusx/pegasusx/apps/backend-go/claims"
	"github.com/pegasusx/pegasusx/apps/backend-go/compliance"
	"github.com/pegasusx/pegasusx/apps/backend-go/controltower"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/creditnote"
	"github.com/pegasusx/pegasusx/apps/backend-go/demand"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/optimizerclient"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/plan"
	"github.com/pegasusx/pegasusx/apps/backend-go/driver"
	"github.com/pegasusx/pegasusx/apps/backend-go/eta"
	"github.com/pegasusx/pegasusx/apps/backend-go/factory"
	"github.com/pegasusx/pegasusx/apps/backend-go/featureflags"
	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
	"github.com/pegasusx/pegasusx/apps/backend-go/globalproducts"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/infraroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/services/billing"
	"github.com/pegasusx/pegasusx/apps/backend-go/inventory"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafka"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafkautil"
	"github.com/pegasusx/pegasusx/apps/backend-go/laborcapacity"
	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap/memory"
	"github.com/pegasusx/pegasusx/apps/backend-go/mfa"
	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/orgoidc"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner"
	"github.com/pegasusx/pegasusx/apps/backend-go/payload"
	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
	"github.com/pegasusx/pegasusx/apps/backend-go/payout"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
	"github.com/pegasusx/pegasusx/apps/backend-go/platform"
	"github.com/pegasusx/pegasusx/apps/backend-go/platformadmin"
	"github.com/pegasusx/pegasusx/apps/backend-go/pricing"
	"github.com/pegasusx/pegasusx/apps/backend-go/promotion"
	"github.com/pegasusx/pegasusx/apps/backend-go/pulse"
	"github.com/pegasusx/pegasusx/apps/backend-go/replenishment"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailer"
	"github.com/pegasusx/pegasusx/apps/backend-go/returns"
	"github.com/pegasusx/pegasusx/apps/backend-go/routing"
	"github.com/pegasusx/pegasusx/apps/backend-go/seed"
	"github.com/pegasusx/pegasusx/apps/backend-go/segment"
	"github.com/pegasusx/pegasusx/apps/backend-go/simulator"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
	"github.com/pegasusx/pegasusx/apps/backend-go/storage"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
	"github.com/pegasusx/pegasusx/apps/backend-go/tax"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
	"github.com/pegasusx/pegasusx/apps/backend-go/tenantreg"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouse"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
	"github.com/pegasusx/pegasusx/packages/handoff"
	"github.com/redis/go-redis/v9"
)

// App holds every long-lived singleton. Wire new app-wide dependencies here,
// never as package-level globals.
type App struct {
	Config                *Config
	Cache                 *cache.Cache
	Idempotency           idempotency.Store
	Supplier              seed.Supplier
	CatalogService        *catalog.Service
	GlobalProductsService *globalproducts.Service
	PromotionService      *promotion.Service
	PromotionAudience     *promotion.AudienceResolver
	InventoryService      *inventory.Service
	NotificationService   *notifications.Service
	NotificationInbox     *notifications.InboxHandlers
	SupplierService       *supplier.Service
	OrgOIDC               *orgoidc.Service
	TenantRegister        *tenantreg.Service
	RetailerService       *retailer.Service
	RetailerProximity     *retailer.RetailerProximityService
	DriverService         *driver.Service
	FactoryService        *factory.Service
	PayloadService        *payload.Service
	PaymentService        *payment.Service
	WebhookInbox          *payment.WebhookInboxStore
	// WebhookReconciler polls stuck payment sessions against the gateway and
	// advances them through the outbox (P1-8). Started as a background worker.
	WebhookReconciler    *payment.WebhookReconciler
	WarehouseService     *warehouse.Service
	ReturnsService       *returns.Service
	TaxService           *tax.Service
	ComplianceService    *compliance.Service
	EvidenceVault        *storage.Vault
	ControlTowerService  *controltower.Service
	ControlTowerHandlers *controltower.Handlers
	ControlTowerWorker   *controltower.Worker
	DemandService        *demand.Service
	LaborCapacityService *laborcapacity.Service
	ETAService           *eta.Service
	FirebaseVerifier     auth.FirebaseVerifier
	OrderService         *order.Service
	ClaimsService        *claims.Service
	CreditService        *credit.Service
	CreditPolicyService  *credit.PolicyService
	ARService            *ar.Service
	CashReconHandlers    *cashrecon.Handlers
	CashReconService     *cashrecon.Service
	CashReconEscalation  *cashrecon.EscalationWorker
	CreditNoteHandlers   *creditnote.Handlers
	CreditNoteService    *creditnote.Service
	// BuyerAcceptancePoller polls Soliq EHF buyer clearance for MySoliq-fiscalized
	// orders (P1-6). Nil when Soliq client / credit-note service unavailable.
	BuyerAcceptancePoller  *order.BuyerAcceptancePoller
	HandoffEngine          *handoff.Engine
	DriverLocations        telemetry.LastLocationStore
	RetailerHub            *ws.Hub
	SupplierHub            *ws.Hub
	DriverHub              *ws.Hub
	PayloadHub             *ws.Hub
	WarehouseHub           *ws.Hub
	FactoryHub             *ws.Hub
	TelemetryHub           *ws.Hub
	PlatformAdminHub       *ws.Hub
	OutboxRelay            *outbox.Relay
	NotificationConsumer   *kafka.Consumer
	OrderEventConsumer     *kafka.Consumer
	WarehouseEventConsumer *kafka.Consumer
	ReturnsEventConsumer   *kafka.Consumer
	ClaimsEventConsumer    *kafka.Consumer
	BillingTierConsumer    *kafka.Consumer
	// RedisClient is the raw go-redis client when the Redis backend is enabled,
	// else nil. Used by the api-mode worker-liveness gate (P1-9).
	RedisClient             *redis.Client
	Reliability             *ReliabilityMiddleware
	InfraHealth             infraroutes.Deps
	OutboundCircuits        *OutboundCircuits
	PlatformService         *platform.Service
	PlatformHandler         *platform.Handler
	PlatformAdminService    *platformadmin.Service
	PlatformAdminHandlers   *platformadmin.Handlers
	MFAService              *mfa.Service
	MFAHandlers             *mfa.Handlers
	FeatureFlagService      *featureflags.Service
	FeatureFlagHandlers     *featureflags.Handlers
	PulseHandlers           *pulse.Handlers
	PushBridge              *notifications.PushBridge
	Spanner                 *spanner.Client
	OptimizerClient         *optimizerclient.Client
	DispatchPlanCounters    *plan.SourceCounters
	ReplenishmentEngine     *replenishment.Engine
	FactoryPlanning         *factory.PlanningService
	ReorderSuggestionWorker *replenishment.ReorderSuggestionWorker
	RouteAnalyticsWorker    *analytics.RouteAnalyticsWorker
	AnalyticsHandlers       *analytics.Handlers
	NotificationPreferences *notifications.PreferenceHandlers
	PartnerService          *partner.Service
	PartnerHandlers         *partner.Handlers
	PartnerKeys             partner.KeyRepository
	PartnerJWTSecret        string
	PartnerWebhookDelivery  *partner.DeliveryWorker
	PartnerExportWorker     *partner.ExportWorker
	FxRatesService          *fxrates.Service
	FxRatesHandlers         *fxrates.Handlers
	FxRatesRepo             fxrates.Repository
	PartnerEdiInbound       *partner.EdiInboundWorker
	PartnerEdiOutbound      *partner.EdiOutboundWorker
	PartnerEventConsumer    *kafka.Consumer
	TwinEventConsumer       *kafka.Consumer
	ARDunningWorker         *ar.DunningWorker
	PayoutService           *payout.Service
	BillingInvoiceWorker    *billing.InvoiceWorker
	ForecastAccuracy        *planning.AccuracyService
	ForecastRunner          *planning.ForecastRunner
	cleanup                 []func()
}

// NewApp constructs the composition root. New singletons attach here.
//
// Persistence policy:
//   - Spanner + Redis + Kafka are required when RequireInfraAdapters=true (default)
//     or PEGASUSX_ENV=production. Bootstrap fails closed; no silent memory path.
//   - In-memory repos/outbox/cache are only used when AllowMemoryFallback=true
//     (ALLOW_MEMORY_FALLBACK=true and non-production, non-strict). Downstream
//     packages depend on interfaces, not concrete stores.
func NewApp(ctx context.Context, cfg *Config) (*App, error) {
	log := slog.Default()
	outboundCircuits := NewOutboundCircuits()
	ws.SetAllowedOrigins(cfg.WebSocketAllowedOrigins)
	if cfg.RequireInfraAdapters {
		log.Info("strict infra adapter mode enabled", "memory_fallback", false)
	} else if cfg.AllowMemoryFallback {
		log.Warn("ALLOW_MEMORY_FALLBACK enabled — not for production traffic")
	}

	setupGCS(ctx, cfg.GCSBucketName, log)

	cacheBackend, redisAdapter, redisEnabled, err := setupRedisCache(ctx, cfg, log)
	if err != nil {
		return nil, err
	}

	cacheClient := cache.New(cacheBackend, log)
	driverLocations := setupDriverLocations(cacheClient)

	idemStore, err := setupIdempotency(cfg, redisAdapter, redisEnabled, log)
	if err != nil {
		return nil, err
	}

	spannerClient, relayStore, outboxAppender, manifestStore, routeGeometryBuilder, osrmClient, spannerOutboxEnabled, spannerCleanup, err := setupSpannerAndRouting(ctx, cfg, outboundCircuits, log)
	if err != nil {
		return nil, err
	}

	cleanup := make([]func(), 0, 3)
	if spannerCleanup != nil {
		cleanup = append(cleanup, spannerCleanup)
	}

	kafkaAuth := kafkautil.ClientAuth{Mode: cfg.KafkaAuthMode, Username: cfg.KafkaSASLUsername}
	outboxPublisher, kafkaEnabled, kafkaCleanup, err := setupKafkaPublisher(cfg, kafkaAuth, log)
	if err != nil {
		return nil, err
	}
	if kafkaCleanup != nil {
		cleanup = append(cleanup, kafkaCleanup)
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
		if err := cfg.ensureMemoryFallbackAllowed("retailer/supplier repositories"); err != nil {
			return nil, err
		}
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
	var globalProductsSvc *globalproducts.Service
	if spannerClient != nil {
		catalogRepo := catalog.NewSpannerRepository(spannerClient)
		catalogSvc = catalog.NewService(catalogRepo, cacheClient, log, promotionSvc, catalog.NewStockEnricher(spannerClient))
		log.Info("catalog service enabled", "backend", "spanner")
		if globalproducts.Enabled() {
			gpRepo := globalproducts.NewSpannerRepository(spannerClient)
			globalProductsSvc = globalproducts.NewService(gpRepo, log)
			if err := globalProductsSvc.EnsureBootstrap(context.Background()); err != nil {
				log.Warn("global products uom seed failed", "err", err)
			}
			catalogSvc.SetGlobalProductHook(globalproducts.CatalogHook{Svc: globalProductsSvc})
			go globalProductsSvc.StartMatchWorker(context.Background(), 2*time.Minute)
			log.Info("global products service enabled")
		}
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
	var notifPrefHandlers *notifications.PreferenceHandlers
	var notifAdapter *notificationReaderAdapter
	if spannerClient != nil {
		notifRepo := notifications.NewSpannerRepository(spannerClient)
		notifSvc = notifications.NewService(notifRepo, cacheClient, log)
		notifPrefHandlers = &notifications.PreferenceHandlers{Repo: notifRepo}
		notifAdapter = &notificationReaderAdapter{svc: notifSvc}
		log.Info("notification service enabled", "backend", "spanner")
	}
	notifInbox = &notifications.InboxHandlers{Service: notifSvc, Log: log}

	firebaseVerifier := newLoginFirebaseVerifier(cfg, log)

	retailerSvc := retailer.NewService(retailer.ServiceConfig{
		Repo:             retailerRepo,
		CartRepo:         cartRepo,
		NotifSvc:         notifAdapter,
		Cache:            cacheClient,
		Idem:             idemStore,
		Locations:        driverLocations,
		Proximity:        retailerProximity,
		SupplierID:       supplierSeed.SupplierID,
		SeedSupplierID:   supplierSeed.SupplierID,
		CountryCode:      cfg.SeedSupplierCountry,
		JWTSecret:        cfg.JWTSecret,
		JWTIssuer:        cfg.JWTIssuer,
		Log:              log,
		FirebaseVerifier: firebaseVerifier,
		// Required for sell-through, reorder suggestions, auto-order bucket/runs, locations Spanner path.
		Spanner: spannerClient,
	})
	retailerSvc.SetTradingPartnerLookup(func(ctx context.Context, supplierID string) (bool, error) {
		if supplierRepo == nil {
			return false, nil
		}
		_, ok, err := supplierRepo.GetProfile(ctx, supplierID)
		return ok, err
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
		Repo:                       supplierRepo,
		Cache:                      cacheClient,
		Idem:                       idemStore,
		Locations:                  driverLocations,
		InventoryService:           supplierInventory,
		DashboardQuery:             dashboardQuery,
		SupplierID:                 supplierSeed.SupplierID,
		SeedSupplierID:             supplierSeed.SupplierID,
		MaxSuppliers:               cfg.MaxSuppliers,
		AllowMultiSupplierRegister: cfg.AllowMultiSupplierRegister,
		Country:                    cfg.SeedSupplierCountry,
		Currency:                   cfg.SeedSupplierCurrency,
		JWTSecret:                  cfg.JWTSecret,
		JWTIssuer:                  cfg.JWTIssuer,
		JWTTTL:                     24 * time.Hour,
		CookieSecure:               false,
		Log:                        log,
	})
	supplier.WireMarketProfileLookup(supplierRepo)
	var oidcStore orgoidc.Store = orgoidc.NewMemoryStore()
	if spannerClient != nil {
		oidcStore = orgoidc.NewSpannerStore(spannerClient)
	}
	orgOIDC := &orgoidc.Service{
		Store:     oidcStore,
		JWTSecret: cfg.JWTSecret,
		JWTIssuer: cfg.JWTIssuer,
		JWTTTL:    24 * time.Hour,
	}
	tenantRegSvc := tenantreg.NewService(tenantreg.Config{
		Repo:           supplierRepo,
		SeedSupplierID: supplierSeed.SupplierID,
		JWTSecret:      cfg.JWTSecret,
		JWTIssuer:      cfg.JWTIssuer,
		JWTTTL:         24 * time.Hour,
		CookieSecure:   false,
		Idem:           idemStore,
		Log:            log,
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
	platformAdminHub := ws.NewHub("platform-admin", cacheBackend, log)
	if promotionSvc != nil {
		promotionSvc.BindRetailerHub(retailerHub)
	}

	var orderRepo order.Repository
	var orderWarehouseResolver order.WarehouseResolver
	if spannerClient != nil {
		orderRepo = order.NewSpannerRepository(spannerClient)
		orderWarehouseResolver = order.NewSpannerWarehouseResolver(spannerClient, redisClientOrNil(redisAdapter))
		log.Info("order repository enabled", "backend", "spanner")
	} else {
		if err := cfg.ensureMemoryFallbackAllowed("order repository"); err != nil {
			return nil, err
		}
		orderRepo = memory.NewOrderRepo(outboxAppender, &memory.RetailerReceivingWindowAdapter{Repo: retailerRepo})
		log.Warn("order repository fallback enabled", "backend", "in-memory")
	}
	var paymentRepo payment.Repository
	if spannerClient != nil {
		paymentRepo = payment.NewSpannerRepository(spannerClient)
		log.Info("payment repository enabled", "backend", "spanner")
	} else {
		if err := cfg.ensureMemoryFallbackAllowed("payment repository"); err != nil {
			return nil, err
		}
		paymentRepo = memory.NewPaymentRepo(outboxAppender)
		log.Warn("payment repository fallback enabled", "backend", "in-memory")
	}
	if paymentRepo != nil {
		repo := paymentRepo
		retailerSvc.SetPaymentSessionByOrder(func(ctx context.Context, orderID string) (string, string, bool, error) {
			session, ok, err := repo.GetSessionByOrderID(ctx, orderID)
			if err != nil || !ok {
				return "", "", ok, err
			}
			return strings.TrimSpace(session.SessionID), strings.TrimSpace(session.Gateway), true, nil
		})
	}
	var creditRepo credit.Repository
	if spannerClient != nil {
		creditRepo = credit.NewSpannerRepository(spannerClient)
		log.Info("credit repository enabled", "backend", "spanner")
	} else {
		if err := cfg.ensureMemoryFallbackAllowed("credit repository"); err != nil {
			return nil, err
		}
		creditRepo = memory.NewCreditRepo(outboxAppender)
		log.Warn("credit repository fallback enabled", "backend", "in-memory")
	}
	creditSvc := credit.NewService(creditRepo)
	if spannerClient != nil {
		creditSvc.SetScoreMetricsProvider(&credit.ARScoreMetrics{Client: spannerClient})
	}
	var creditPolicyRepo credit.PolicyRepository
	var arRepo ar.Repository
	if spannerClient != nil {
		creditPolicyRepo = credit.NewSpannerPolicyRepository(spannerClient)
		arRepo = ar.NewSpannerRepository(spannerClient)
	} else {
		creditPolicyRepo = credit.NewMemoryPolicyRepository()
		arRepo = ar.NewMemoryRepository()
	}
	creditPolicySvc := credit.NewPolicyService(creditPolicyRepo, creditSvc)
	creditSvc.SetPolicyGate(creditPolicySvc)
	arSvc := ar.NewService(arRepo)
	arDunning := ar.NewDunningWorker(arSvc, log)
	// Phase 1 money surfaces: payouts (bank-file) + billing fee schedule.
	var payoutSvc *payout.Service
	var billingInvoiceWorker *billing.InvoiceWorker
	var feeResolver *billing.FeeScheduleResolver
	if spannerClient != nil {
		feeResolver = billing.NewFeeScheduleResolver(spannerClient)
		payoutSvc = payout.NewService(payout.NewRepository(spannerClient))
		payoutSvc.SetCommissionResolver(feeResolver)
		payoutSvc.SetCache(cacheClient)
		billingInvoiceWorker = billing.NewInvoiceWorker(spannerClient, arSvc, feeResolver, log)
	}
	supplierSvc.SetEarningsLookup(func(ctx context.Context, supplierID, currency string, now time.Time) (supplier.SupplierEarningsResponse, error) {
		return loadSupplierEarningsAuthority(ctx, paymentRepo, supplierID, currency, now)
	})
	handoffEngine := handoff.FromEnv()
	var gatewayPolicyReader *payment.SpannerPolicyResolver
	if spannerClient != nil {
		gatewayPolicyReader = payment.NewSpannerPolicyResolver(spannerClient)
	}
	var complianceSvc *compliance.Service
	var taxSvc *tax.Service
	var routingSvc *routing.Service
	var pricingSvc pricing.Service
	var cashReconSvc *cashrecon.Service
	var cashReconHandlers *cashrecon.Handlers
	var creditNoteSvc *creditnote.Service
	var creditNoteHandlers *creditnote.Handlers
	cashReconRequired := strings.EqualFold(strings.TrimSpace(os.Getenv("CASH_RECONCILIATION_REQUIRED")), "true")
	if spannerClient != nil {
		complianceSvc = compliance.NewService(compliance.NewSpannerRepository(spannerClient), slog.Default())
		taxSvc = tax.NewService(tax.NewSpannerRepository(spannerClient), cacheClient, slog.Default())
		routingSvc = routing.NewService(spannerClient)
		pricingSvc = pricing.NewService(pricing.NewSpannerRepository(spannerClient))
		cashReconRepo := cashrecon.NewSpannerRepository(spannerClient)
		cashReconSvc = cashrecon.NewService(cashReconRepo, cashReconRepo)
		cashReconHandlers = &cashrecon.Handlers{Svc: cashReconSvc}
		creditNoteRepo := creditnote.NewSpannerRepository(spannerClient)
		creditNoteSvc = creditnote.NewService(creditNoteRepo)
		creditNoteHandlers = &creditnote.Handlers{
			Svc: creditNoteSvc,
			// Last-resort seed only when PreferTenantSupplierID + claims are empty.
			SupplierID: func() string { return supplierSeed.SupplierID },
		}
	}

	orderSvc := order.NewService(order.ServiceConfig{
		Repo:                  orderRepo,
		Cache:                 cacheClient,
		Warehouse:             orderWarehouseResolver,
		Promotions:            promotionSvc,
		Credit:                creditSvc,
		CommissionResolver:    feeResolver,
		SeedSupplierID:        supplierSeed.SupplierID,
		SupplierID:            supplierSeed.SupplierID,
		SupplierName:          cfg.SeedSupplierName,
		Currency:              cfg.SeedSupplierCurrency,
		CurrencyPickerEnabled: order.OrderCurrencyPickerEnabled(),
		CurrencyAllowlist:     order.ParseCurrencyAllowlist(os.Getenv("ORDER_CURRENCY_ALLOWLIST"), cfg.SeedSupplierCurrency),
		RetailerHub:           retailerHub,
		SupplierHub:           supplierHub,
		DriverHub:             driverHub,
		SpannerClient:         spannerClient,
		ShopClosedGrace:       shopClosedGraceDuration(),
		Log:                   log,
		JWTSecret:             cfg.JWTSecret,
		Handoff:               handoffEngine,
		Idem:                  idemStore,
		Replanner:             routingSvc,
	})
	orderSvc.SetManifestStore(manifestStore)
	if spannerClient != nil {
		segmentSvc := segment.NewService(segment.NewSpannerRepository(spannerClient))
		allocSvc := allocation.NewAllocationService(spannerClient)
		allocSvc.SetSegmentService(segmentSvc)
		allocSvc.SetConstrainedAllocationEnabled(strings.EqualFold(strings.TrimSpace(os.Getenv("CONSTRAINED_ALLOCATION_ENABLED")), "true"))
		orderSvc.SetAllocator(allocSvc)
	}
	orderSvc.SetAllocationRequired(strings.EqualFold(strings.TrimSpace(os.Getenv("ALLOCATION_REQUIRED")), "true"))
	if pricingSvc != nil {
		orderSvc.SetPricingService(pricingSvc)
	}
	if gatewayPolicyReader != nil {
		orderSvc.SetGatewayPolicyReader(gatewayPolicyReader)
	}
	// Logistics claims domain (post-delivery damage / driver OS&D). Additive; optional bridge.
	var claimsRepo claims.Repository
	if spannerClient != nil {
		claimsRepo = claims.NewSpannerRepository(spannerClient)
		log.Info("claims repository enabled", "backend", "spanner")
	} else {
		claimsRepo = claims.NewMemoryRepository()
		log.Warn("claims repository fallback enabled", "backend", "in-memory")
	}
	claimsSvc := claims.NewService(claims.Config{
		Repo:   claimsRepo,
		Orders: orderClaimsLookup{svc: orderSvc},
		Idem:   idemStore,
		Log:    log,
	})
	if creditNoteSvc != nil {
		claimsSvc.SetCreditNotes(creditnote.ClaimsBridge{Svc: creditNoteSvc})
	}
	orderSvc.SetClaimsBridge(&driverClaimsBridge{svc: claimsSvc})
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
		optimizerCli = optimizerclient.New(cfg.OptimizerBaseURL, cfg.InternalAPIKey).WithOSRM(osrmClient)
		log.Info("dispatch optimiser client enabled", "base_url", cfg.OptimizerBaseURL, "osrm_matrix", osrmClient != nil)
	}
	dispatchCounters := &plan.SourceCounters{}

	var replenishmentEngine *replenishment.Engine
	var factoryPlanning *factory.PlanningService
	var reorderSuggestionWorker *replenishment.ReorderSuggestionWorker
	var routeAnalyticsWorker *analytics.RouteAnalyticsWorker
	var analyticsHandlers *analytics.Handlers
	if spannerClient != nil {
		replenishmentEngine = replenishment.NewEngine(spannerClient, log)
		replenishmentEngine.EchelonTargetsEnabled = strings.EqualFold(strings.TrimSpace(os.Getenv("MEIO_ECHELON_TARGETS_ENABLED")), "true")
		factoryPlanning = factory.NewPlanningService(spannerClient, log)
		reorderSuggestionWorker = replenishment.NewReorderSuggestionWorker(spannerClient)
		reorderSuggestionWorker.EchelonTargetsEnabled = replenishmentEngine.EchelonTargetsEnabled
		routeAnalyticsWorker = analytics.NewRouteAnalyticsWorkerFromClient(spannerClient)
		analyticsHandlers = &analytics.Handlers{
			Client: spannerClient,
			SupplierID: func(ctx context.Context) string {
				return auth.PreferTenantSupplierID(ctx, supplierSeed.SupplierID)
			},
		}
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
		FactoryPlanning:      factoryPlanning,
	})
	retailerSvc.SetOrderLifecycle(orderSvc)
	// Auto-order place mode: real order.Create (never mobile_compat). Flag default off.
	retailerSvc.SetOrderCreator(orderSvc)
	retailerSvc.SetAutoOrderPlaceEnabled(strings.EqualFold(strings.TrimSpace(os.Getenv("AUTO_ORDER_PLACE_ENABLED")), "true"))

	factoryNodeID := strings.TrimSpace(os.Getenv("FACTORY_DEMO_ID"))
	if factoryNodeID == "" {
		factoryNodeID = "factory-demo-1"
	}
	warehouseNodeID := strings.TrimSpace(os.Getenv("WAREHOUSE_DEMO_ID"))
	if warehouseNodeID == "" {
		warehouseNodeID = strings.TrimSpace(os.Getenv("SANDBOX_SMOKE_WAREHOUSE_ID"))
	}
	if warehouseNodeID == "" {
		warehouseNodeID = strings.TrimSpace(os.Getenv("SSMR_SMOKE_WAREHOUSE_ID"))
	}
	if warehouseNodeID == "" {
		warehouseNodeID = "wh-demo-1"
	}

	// Fake H3 / network graph telemetry — never on sandbox/production even if flag set.
	ctSim := strings.EqualFold(strings.TrimSpace(os.Getenv("CONTROL_TOWER_SIMULATOR_ENABLED")), "true")
	if ctSim && !auth.IsSandbox() && !auth.IsProduction() {
		simulator.StartControlTowerSimulation(telemetryHub, supplierSeed.SupplierID, warehouseNodeID)
		log.Info("control tower telemetry simulator enabled")
	} else if ctSim {
		log.Warn("control tower simulator ignored on non-dev env", "env", os.Getenv("PEGASUSX_ENV"))
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
		if err := cfg.ensureMemoryFallbackAllowed("driver/factory/payload repositories"); err != nil {
			return nil, err
		}
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
		Repo:             factoryRepo,
		Cache:            cacheClient,
		SupplierHub:      supplierHub,
		FactoryHub:       factoryHub,
		Log:              log,
		Spanner:          spannerClient,
		Locations:        driverLocations,
		SupplierID:       supplierSeed.SupplierID,
		SeedSupplierID:   supplierSeed.SupplierID,
		FactoryNodeID:    factoryNodeID,
		Currency:         cfg.SeedSupplierCurrency,
		JWTSecret:        cfg.JWTSecret,
		JWTIssuer:        cfg.JWTIssuer,
		Idem:             idemStore,
		Planning:         factoryPlanning,
		OptimizerClient:  optimizerCli,
		FirebaseVerifier: firebaseVerifier,
	})
	payloadSvc := payload.NewService(payload.ServiceConfig{
		Repo:             payloadRepo,
		Cache:            cacheClient,
		SupplierHub:      supplierHub,
		PayloadHub:       payloadHub,
		DriverHub:        driverHub,
		NotifSvc:         notifSvc,
		Log:              log,
		SupplierID:       supplierSeed.SupplierID,
		SeedSupplierID:   supplierSeed.SupplierID,
		Currency:         cfg.SeedSupplierCurrency,
		JWTSecret:        cfg.JWTSecret,
		JWTIssuer:        cfg.JWTIssuer,
		ManifestStore:    manifest.NewStore(spannerClient),
		Idem:             idemStore,
		Locations:        driverLocations,
		FirebaseVerifier: firebaseVerifier,
	})
	payloadSvc.SetPortalManifestLister(&supplier.ManifestLister{Service: supplierSvc})
	payloadSvc.SetOrderExpectationReader(orderRepo)
	if spannerClient != nil {
		payloadSvc.SetStaffLookup(payloadStaffLoginLookup(spannerClient))
	}
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
	// Claim / OS&D → warehouse inbound tickets (deduped with amend-created returns).
	claimsSvc.SetReverseLogistics(&returnsClaimsBridge{svc: returnsSvc})
	// Store QUARANTINE hold on physical claims (FLOOR/BACKROOM → QUARANTINE).
	claimsSvc.SetStoreStock(&retailerClaimStockBridge{svc: retailerSvc})
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
	var driverOpenFiscal driver.OpenFiscalLookup
	if spannerClient != nil {
		driverOpenFiscal = func(ctx context.Context, driverID string) (driver.OpenFiscalSnapshot, error) {
			snap, err := order.CountOpenFiscalForDriver(ctx, spannerClient, driverID)
			if err != nil {
				return driver.OpenFiscalSnapshot{}, err
			}
			return driver.OpenFiscalSnapshot{
				Count:    snap.Count,
				OrderIDs: snap.OrderIDs,
				Frozen:   snap.Frozen,
			}, nil
		}
	}
	// OFD adapter: cell default from env. GS-M2 fail-closes planned/PEPPOL packs at fiscalize.
	fiscalProvider := order.ProviderFromEnv()
	orderSvc.SetFiscalProvider(fiscalProvider)
	var buyerAcceptancePoller *order.BuyerAcceptancePoller
	if auth.BuyerAcceptancePollerAllowed() {
		if soliqClient := fiscalProvider.GetSoliqClient(); soliqClient != nil && creditNoteSvc != nil {
			buyerAcceptancePoller = order.NewBuyerAcceptancePoller(soliqClient, orderRepo, log, creditNoteSvc)
			// Default is ON (reverse settlement on reject). Explicit override still honored.
			if v := strings.TrimSpace(os.Getenv("CREDIT_NOTE_AUTO_FROM_BUYER_REJECT")); strings.EqualFold(v, "false") || v == "0" {
				buyerAcceptancePoller.SetAutoCreditNoteOnBuyerReject(false)
			}
			log.Info("buyer acceptance poller constructed (started by worker tier)")
		}
	}
	driverSvc := driver.NewService(driver.ServiceConfig{
		Repo:                       driverRepo,
		Spanner:                    spannerClient,
		Cache:                      cacheClient,
		NotifSvc:                   notifAdapter,
		OrderList:                  driverOrderList,
		HistoryQuery:               driverHistoryListQuery(spannerClient),
		OrderGet:                   driverOrderGet,
		ProfileLookup:              driverProfileLookup,
		AvailabilityReader:         driverAvailReader,
		RouteGeometry:              driverRouteGeometry,
		Depart:                     driverDepart,
		ReturnComplete:             driverReturnComplete,
		OpenFiscal:                 driverOpenFiscal,
		CashReconciliationRequired: cashReconRequired,
		CashReconciliationGate: func(ctx context.Context, driverID string) (bool, error) {
			if cashReconSvc == nil {
				return true, nil
			}
			return cashReconSvc.HasAcceptedReconciliation(ctx, driverID, time.Now().UTC())
		},
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
		SupplierID:       supplierSeed.SupplierID,
		SeedSupplierID:   supplierSeed.SupplierID,
		Currency:         cfg.SeedSupplierCurrency,
		JWTSecret:        cfg.JWTSecret,
		JWTIssuer:        cfg.JWTIssuer,
		Idem:             idemStore,
		FirebaseVerifier: firebaseVerifier,
	})
	if spannerClient != nil {
		driverSvc.SetLoginLookup(driverLoginLookup(spannerClient))
	}
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
		SeedSupplierID:                  supplierSeed.SupplierID,
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
	paymentSvc.BindOrderCashSelector(orderSvc)
	orderSvc.SetPaymentCapturer(paymentSvc)
	orderSvc.SetPaymentRefunder(paymentSvc)

	// P1-8: settlement-vs-captured reconciliation. Polls sessions stuck in
	// AWAITING_PAYMENT and advances them via the gateway status check. Without a
	// runner this was manual-only.
	webhookReconciler := payment.NewWebhookReconciler(paymentRepo, orderRepo, paymentExec, time.Now)

	// Theatre #13: FX rates (ConvertMinor + admin GET/PUT). Operating currency stays UZS.
	var fxRepo fxrates.Repository = fxrates.NewMemoryRepository()
	if spannerClient != nil {
		fxRepo = fxrates.NewSpannerRepository(spannerClient)
	} else if err := cfg.ensureMemoryFallbackAllowed("fx rates repository"); err != nil {
		return nil, err
	}
	fxSvc := fxrates.NewService(fxRepo)
	fxHandlers := fxrates.NewHandlers(fxRepo)
	if err := fxrates.SeedBootstrapRates(ctx, fxRepo, fxrates.SeedOptions{
		OperatingCurrency: cfg.SeedSupplierCurrency,
		USDToUZSScaled:    cfg.FxSeedUSDToUZSScaled,
		Log:               log,
	}); err != nil {
		// Table may be absent until migration; warn and continue (SSMR prints SKIPPED).
		log.Warn("fx rates seed skipped or failed", "err", err)
	}
	paymentSvc.SetFxRates(fxSvc)

	orderSvc.SetARService(arSvc)
	// Claims chargeback: supplier ledger debit + optional Global Pay partial refund.
	claimsSvc.SetSettler(&claimPaymentSettler{pay: paymentSvc})
	claimsSvc.SetStoreCredit(creditSvc)
	var warehouseRepo warehouse.Repository
	if spannerClient != nil {
		warehouseRepo = warehouse.NewSpannerRepository(spannerClient)
		log.Info("warehouse repository enabled", "backend", "spanner")
	} else {
		if err := cfg.ensureMemoryFallbackAllowed("warehouse repository"); err != nil {
			return nil, err
		}
		warehouseRepo = warehouse.NewInMemoryRepository()
		log.Warn("warehouse repository fallback enabled", "backend", "in-memory")
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
	stocklots.SetLotsEnabled(strings.EqualFold(strings.TrimSpace(os.Getenv("WMS_LOTS_ENABLED")), "true"))
	stocklots.SetPickWavesEnabled(strings.EqualFold(strings.TrimSpace(os.Getenv("WMS_PICK_WAVES_ENABLED")), "true"))
	stocklots.SetCycleCountsEnabled(strings.EqualFold(strings.TrimSpace(os.Getenv("WMS_CYCLE_COUNTS_ENABLED")), "true"))
	stocklots.SetPickSShapeEnabled(strings.EqualFold(strings.TrimSpace(os.Getenv("WMS_PICK_SSHAPE_ENABLED")), "true"))
	stocklots.SetSealSoftWarnEnabled(strings.EqualFold(strings.TrimSpace(os.Getenv("WMS_SEAL_SOFT_WARN")), "true"))
	stocklots.SetColdChainEnabled(strings.EqualFold(strings.TrimSpace(os.Getenv("WMS_COLD_CHAIN_ENABLED")), "true"))
	stocklots.SetLoadLedgerEnabled(strings.EqualFold(strings.TrimSpace(os.Getenv("PAYLOAD_LOAD_LEDGER_ENABLED")), "true"))
	stocklots.SetLaborCapacityEnforce(strings.EqualFold(strings.TrimSpace(os.Getenv("LABOR_CAPACITY_ENFORCE")), "true"))
	stocklots.SetTemperatureBreachRaiser(func(ctx context.Context, txn *spanner.ReadWriteTransaction, args stocklots.TemperatureBreachArgs) error {
		return orderSvc.RaiseSystemTemperatureBreachInTxn(ctx, txn, order.SystemTemperatureBreachArgs{
			ManifestID: args.ManifestID,
			ReadingID:  args.ReadingID,
			TempC:      args.TempC,
			MinC:       args.MinC,
			MaxC:       args.MaxC,
			OrderIDs:   args.OrderIDs,
		})
	})

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
		RedisClient:          redisClientOrNil(redisAdapter),
		ManifestStore:        manifestStore,
		RouteGeometryBuilder: routeGeometryBuilder,
		Locations:            driverLocations,
		SupplierHub:          supplierHub,
		WarehouseHub:         warehouseHub,
		DriverHub:            driverHub,
		RetailerHub:          retailerHub,
		Log:                  log,
		SupplierID:           supplierSeed.SupplierID,
		SeedSupplierID:       supplierSeed.SupplierID,
		Currency:             cfg.SeedSupplierCurrency,
		JWTSecret:            cfg.JWTSecret,
		JWTIssuer:            cfg.JWTIssuer,
		FirebaseVerifier:     firebaseVerifier,
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
	var platformAdminRepo platformadmin.Repository = platformadmin.NewMemoryRepository()
	var featureFlagRepo featureflags.Repository = featureflags.NewMemoryRepository()
	var mfaRepo mfa.Repository = mfa.NewMemoryRepository()
	if spannerClient != nil {
		platformAdminRepo = platformadmin.NewSpannerRepository(spannerClient)
		featureFlagRepo = featureflags.NewSpannerRepository(spannerClient)
		mfaRepo = mfa.NewSpannerRepository(spannerClient)
	}
	platformAdminSvc := platformadmin.NewService(platformAdminRepo)
	platformAdminSvc.OnApproved = func(ctx context.Context, tenantType, tenantID, marketCode, homeCell string) error {
		if tenantType != platformadmin.TenantSupplier || supplierRepo == nil {
			return nil
		}
		p, found, err := supplierRepo.GetProfile(ctx, tenantID)
		if err != nil || !found {
			return err
		}
		p.MarketCode = marketCode
		p.HomeCell = homeCell
		if strings.TrimSpace(p.Country) == "" {
			p.Country = marketCode
		}
		if pack, ok := auth.ResolveShippedMarketPack(marketCode); ok && strings.TrimSpace(p.Currency) == "" {
			p.Currency = pack.CurrencyCode
		}
		p.UpdatedAt = time.Now().UTC()
		return supplierRepo.UpdateProfile(ctx, p, nil)
	}
	platformAdminSvc.SetHub(platformAdminHub)
	platformAdminHandlers := &platformadmin.Handlers{
		Svc:       platformAdminSvc,
		JWTSecret: cfg.JWTSecret,
		JWTIssuer: cfg.JWTIssuer,
		Ops: &platformadmin.OpsDeps{
			Spanner:    spannerClient,
			RunMode:    cfg.RunMode,
			RunsAPI:    cfg.RunsAPI(),
			RunsWorker: cfg.RunsWorkers(),
		},
	}
	if spannerClient != nil {
		platformAdminHandlers.Ops.Outbox = outbox.NewSpannerStore(spannerClient)
		_ = platformadmin.EnsureAdminFromEnv(ctx, spannerClient)
	}
	mfaSvc := mfa.NewService(mfaRepo, cfg.JWTIssuer, cfg.PlatformAdminMFARequired, platformAdminSvc)
	mfaHandlers := &mfa.Handlers{Svc: mfaSvc, JWTSecret: cfg.JWTSecret, JWTIssuer: cfg.JWTIssuer}
	featureFlagSvc := featureflags.NewService(featureFlagRepo)
	featureFlagHandlers := &featureflags.Handlers{Svc: featureFlagSvc, Audit: platformAdminSvc}
	retailerSvc.SetPlaceFlagEvaluator(featureFlagSvc)
	stocklots.SetFlagEvaluator(featureFlagSvc) // G2.A seal-class tenant overrides
	retailerSvc.SetSoakBypassAuditor(platformAdminSvc)
	onRegistered := func(ctx context.Context, supplierID, legalName string) error {
		if err := platformAdminSvc.EnsurePending(ctx, platformadmin.TenantSupplier, supplierID, legalName); err != nil {
			return err
		}
		// Seed / first tenant auto-approved so single-tenant SSMR keeps working.
		// T1 never mints the seed id, so this branch does not apply to self-serve.
		if supplierID == supplierSeed.SupplierID {
			pack, ok := auth.ResolveShippedMarketPack(cfg.SeedSupplierCountry)
			if !ok {
				pack, _ = auth.ResolveShippedMarketPack(auth.DefaultMarketCode)
			}
			_, err := platformAdminSvc.Transition(ctx, platformadmin.TransitionInput{
				Actor:      "system:bootstrap",
				TenantType: platformadmin.TenantSupplier,
				TenantID:   supplierID,
				Status:     platformadmin.StatusApproved,
				KybNotes:   "seed_auto_approve",
				MarketCode: pack.Code,
				HomeCell:   pack.HomeCell,
			})
			return err
		}
		return nil
	}
	supplierSvc.OnRegistered = onRegistered
	tenantRegSvc.OnRegistered = onRegistered

	pushBridge, err := setupPushBridge(cfg, spannerClient, tokenRepo, log)
	if err != nil {
		return nil, err
	}

	// Collections substance: dunning auto-hold + delinquency + inbox/FCM fanout.
	arDunning.SetAutoHold(func(ctx context.Context, supplierID, retailerID string) error {
		return creditPolicySvc.HoldRelationship(ctx, supplierID, retailerID, "system:dunning", "SYSTEM")
	})
	arDunning.SetDelinquencyBump(func(ctx context.Context, supplierID, retailerID string) error {
		return creditSvc.BumpDelinquency(ctx, retailerID, supplierID)
	})

	dunningNotify, err := setupDunningNotification(spannerClient, notifSvc, pushBridge, log)
	if err != nil {
		return nil, err
	}
	arDunning.SetNotify(dunningNotify)

	forecastAccuracy := &planning.AccuracyService{Client: spannerClient, Log: log}
	forecastAccuracy.Notify = func(ctx context.Context, supplierID, warehouseID, productID string, ts float64, day civil.Date) error {
		title := "Forecast tracking signal alert"
		body := "Product " + productID + " warehouse " + warehouseID + " |TS|=" + fmt.Sprintf("%.2f", ts) + " on " + day.String()
		if notifSvc != nil {
			_ = notifSvc.CreateNotification(ctx, supplierID, "ADMIN", "FORECAST_TS_ALERT", title, body, "/analytics/demand")
		}
		log.Warn("forecast tracking signal out of control",
			"supplier_id", supplierID, "warehouse_id", warehouseID, "product_id", productID,
			"day", day.String(), "ts", ts)
		return nil
	}
	forecastRunner := &planning.ForecastRunner{Client: spannerClient, Log: log}

	var partnerKeys partner.KeyRepository = partner.NewMemoryKeyRepository()
	var partnerWebhooks partner.WebhookRepository = partner.NewMemoryWebhookRepository()
	var partnerExports partner.ExportRepository = partner.NewMemoryExportRepository()
	var partnerSftp partner.SftpConfigRepository = partner.NewMemorySftpConfigRepository()
	var partnerEdiDocs partner.EdiDocumentRepository = partner.NewMemoryEdiDocumentRepository()
	if spannerClient != nil {
		partnerKeys = partner.NewSpannerKeyRepository(spannerClient)
		partnerWebhooks = partner.NewSpannerWebhookRepository(spannerClient)
		partnerExports = partner.NewSpannerExportRepository(spannerClient)
		partnerSftp = partner.NewSpannerSftpConfigRepository(spannerClient)
		partnerEdiDocs = partner.NewSpannerEdiDocumentRepository(spannerClient)
	}
	partnerSvc := partner.NewService(partnerKeys, partnerWebhooks, orderSvc, catalogSvc, log)
	partnerSvc.SetExportRepos(partnerExports, partnerSftp)
	partnerSvc.SetIdempotencyStore(idemStore)
	partnerSvc.SetInventoryService(inventorySvc)
	if spannerClient != nil {
		partnerSvc.SetPOSFeedSink(partner.NewSpannerPOSFeedSink(spannerClient))
	}
	var partnerAs2 partner.As2ConfigRepository = partner.NewMemoryAs2ConfigRepository()
	if spannerClient != nil {
		partnerAs2 = partner.NewSpannerAs2ConfigRepository(spannerClient)
	}
	partnerSvc.SetAs2Repository(partnerAs2)
	var partnerCoa partner.CoaRepository = partner.NewMemoryCoaRepository()
	if spannerClient != nil {
		partnerCoa = partner.NewSpannerCoaRepository(spannerClient)
	}
	partnerSvc.SetCoaRepository(partnerCoa)
	partnerDelivery := partner.NewDeliveryWorker(partnerWebhooks, log)
	partnerExportWorker := partner.NewExportWorker(partnerExports, partnerSftp, spannerClient, log)
	partnerExportWorker.SetCoaRepository(partnerCoa)
	partnerEdiOut := partner.NewEdiOutboundWorker(partnerEdiDocs, partnerSftp, orderSvc, spannerClient, log)
	partnerEdiOut.SetAs2Repository(partnerAs2)
	partnerSvc.SetEdiRepos(partnerEdiDocs, partnerEdiOut)
	partnerEdiIn := partner.NewEdiInboundWorker(partnerEdiDocs, partnerSftp, partnerSvc, log)
	// G5.A shared EDI profile store (memory default; Spanner when available).
	var partnerEdiProfiles partner.EdiProfileRepository = partner.NewMemoryEdiProfiles()
	if spannerClient != nil {
		partnerEdiProfiles = &partner.SpannerEdiProfiles{Client: spannerClient}
	}
	partnerSvc.SetEdiProfiles(partnerEdiProfiles)
	partnerEdiIn.SetEdiProfiles(partnerEdiProfiles)
	partnerEdiOut.SetEdiProfiles(partnerEdiProfiles)
	partnerEdiIn.SetAckEnqueuer(partnerEdiOut)
	partnerEdiIn.ResolveGeo = func(ctx context.Context, retailerID string) (partner.RetailerGeo, error) {
		loc, err := retailerSvc.EnsurePrimaryLocation(ctx, retailerID)
		if err != nil {
			return partner.RetailerGeo{}, err
		}
		return partner.RetailerGeo{Lat: loc.Lat, Lng: loc.Lng, H3Cell: loc.H3Cell}, nil
	}
	partnerHandlers := &partner.Handlers{Svc: partnerSvc, Delivery: partnerDelivery, EdiInbound: partnerEdiIn}
	partnerJWTSecret := partner.ResolvePartnerJWTSecret(cfg.JWTSecret)
	partnerSvc.SetOAuthJWT(partnerJWTSecret, partner.PartnerOAuthIssuer(), partner.PartnerOAuthTTL())

	var consumers *kafkaConsumers
	if kafkaEnabled && cfg.KafkaTopicMain != "" {
		c, err := setupKafkaConsumers(
			cfg, kafkaAuth, redisAdapter, spannerClient, cacheClient, pushBridge,
			notifSvc, retailerSvc, retailerHub, supplierHub, driverHub, warehouseHub,
			factoryHub, payloadHub, orderSvc, warehouseSvc, returnsSvc, claimsSvc,
			partnerSvc, fxSvc, log,
		)
		if err != nil {
			return nil, err
		}
		consumers = c
		if c.cleanup != nil {
			cleanup = append(cleanup, c.cleanup)
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

	var controlTowerSvc *controltower.Service
	var controlTowerHandlers *controltower.Handlers
	var controlTowerWorker *controltower.Worker
	if spannerClient != nil {
		ctCfg := controltower.ConfigFromEnv()
		ctRepo := controltower.NewSpannerRepository(spannerClient)
		cnSvc := creditNoteSvc
		if cnSvc == nil {
			cnSvc = creditnote.NewService(creditnote.NewSpannerRepository(spannerClient))
		}
		planSvc := planning.NewService(spannerClient).WithCache(cacheClient)
		planSvc.TwinScenarioEnabled = strings.EqualFold(strings.TrimSpace(os.Getenv("TWIN_SCENARIO_ENABLED")), "true")
		segSvc := segment.NewService(segment.NewSpannerRepository(spannerClient))
		ctExecutor := controltower.NewActionExecutor(ctRepo, controltower.ExecutorDeps{
			CreditNotes:   cnSvc,
			Credit:        creditSvc,
			Planning:      planSvc,
			Routing:       routingSvc,
			Notifications: notifSvc,
			Returns:       returnsSvc,
			Segment:       segSvc,
			Log:           log,
		})
		ctEngine := controltower.NewEngine(ctRepo, ctExecutor, segSvc, ctCfg, log)
		controlTowerSvc = controltower.NewService(ctRepo, ctEngine, ctCfg)
		controlTowerHandlers = controltower.NewHandlers(controlTowerSvc, supplierSvc.ScopedSupplierID)
		controlTowerWorker = controltower.NewWorker(controlTowerSvc, log, 3*time.Minute)
		if ctCfg.Enabled {
			log.Info("control tower playbooks enabled", "auto_execute", ctCfg.AutoExecute)
		}
		supplierSvc.SetControlTower(controlTowerSvc)
		if replenishmentEngine != nil {
			replenishmentEngine.SegmentSvc = segSvc
		}
	}

	demandSvc := demand.NewService(spannerClient)
	if demandSvc != nil {
		demandSvc.SetSupplierID(func(ctx context.Context) string {
			return auth.PreferTenantSupplierID(ctx, supplierSeed.SupplierID)
		})
	}
	if demandSvc != nil && reorderSuggestionWorker != nil {
		demandSvc.SetAfterSensingHook(func(ctx context.Context) error {
			return reorderSuggestionWorker.RunBatchAllSuppliers(ctx)
		})
	}

	var cashReconEscalation *cashrecon.EscalationWorker
	if spannerClient != nil && cashReconSvc != nil && notifSvc != nil {
		cashReconEscalation = &cashrecon.EscalationWorker{
			Spanner:    spannerClient,
			Notifier:   notifSvc,
			SupplierID: supplierSeed.SupplierID,
			Now:        func() time.Time { return time.Now().UTC() },
		}
	}

	if rc := redisClientOrNil(redisAdapter); rc != nil {
		auth.SetRevocationStore(auth.NewRedisRevocationStore(rc))
	} else {
		auth.SetRevocationStore(auth.NewMemoryRevocationStore())
	}

	app := &App{
		Config:                  cfg,
		Cache:                   cacheClient,
		Idempotency:             idemStore,
		Supplier:                supplierSeed,
		CatalogService:          catalogSvc,
		GlobalProductsService:   globalProductsSvc,
		PromotionService:        promotionSvc,
		PromotionAudience:       promotionAudience,
		InventoryService:        inventorySvc,
		NotificationService:     notifSvc,
		NotificationInbox:       notifInbox,
		SupplierService:         supplierSvc,
		OrgOIDC:                 orgOIDC,
		TenantRegister:          tenantRegSvc,
		RetailerService:         retailerSvc,
		DriverService:           driverSvc,
		FactoryService:          factorySvc,
		PayloadService:          payloadSvc,
		PaymentService:          paymentSvc,
		WebhookInbox:            webhookInbox,
		WebhookReconciler:       webhookReconciler,
		WarehouseService:        warehouseSvc,
		ReturnsService:          returnsSvc,
		TaxService:              taxSvc,
		ComplianceService:       complianceSvc,
		EvidenceVault:           storage.NewVault(spannerClient),
		ControlTowerService:     controlTowerSvc,
		ControlTowerHandlers:    controlTowerHandlers,
		ControlTowerWorker:      controlTowerWorker,
		DemandService:           demandSvc,
		LaborCapacityService:    laborcapacity.NewService(spannerClient),
		ETAService:              eta.NewService(spannerClient),
		FirebaseVerifier:        firebaseVerifier,
		OrderService:            orderSvc,
		ClaimsService:           claimsSvc,
		CreditService:           creditSvc,
		CreditPolicyService:     creditPolicySvc,
		ARService:               arSvc,
		CashReconHandlers:       cashReconHandlers,
		CashReconService:        cashReconSvc,
		CashReconEscalation:     cashReconEscalation,
		CreditNoteHandlers:      creditNoteHandlers,
		CreditNoteService:       creditNoteSvc,
		BuyerAcceptancePoller:   buyerAcceptancePoller,
		HandoffEngine:           handoffEngine,
		DriverLocations:         driverLocations,
		RetailerHub:             retailerHub,
		SupplierHub:             supplierHub,
		DriverHub:               driverHub,
		PayloadHub:              payloadHub,
		WarehouseHub:            warehouseHub,
		FactoryHub:              factoryHub,
		TelemetryHub:            telemetryHub,
		PlatformAdminHub:        platformAdminHub,
		RedisClient:             redisClientOrNil(redisAdapter),
		OutboxRelay:             outboxRelay,
		Reliability:             reliabilityMiddleware,
		InfraHealth:             infraHealth,
		OutboundCircuits:        outboundCircuits,
		PlatformService:         platformSvc,
		PlatformHandler:         platformHandler,
		PlatformAdminService:    platformAdminSvc,
		PlatformAdminHandlers:   platformAdminHandlers,
		MFAService:              mfaSvc,
		MFAHandlers:             mfaHandlers,
		FeatureFlagService:      featureFlagSvc,
		FeatureFlagHandlers:     featureFlagHandlers,
		PulseHandlers:           pulseHandlers,
		PushBridge:              pushBridge,
		Spanner:                 spannerClient,
		OptimizerClient:         optimizerCli,
		DispatchPlanCounters:    dispatchCounters,
		ReplenishmentEngine:     replenishmentEngine,
		FactoryPlanning:         factoryPlanning,
		ReorderSuggestionWorker: reorderSuggestionWorker,
		RouteAnalyticsWorker:    routeAnalyticsWorker,
		AnalyticsHandlers:       analyticsHandlers,
		NotificationPreferences: notifPrefHandlers,
		PartnerService:          partnerSvc,
		PartnerHandlers:         partnerHandlers,
		PartnerKeys:             partnerKeys,
		PartnerJWTSecret:        partnerJWTSecret,
		PartnerWebhookDelivery:  partnerDelivery,
		PartnerExportWorker:     partnerExportWorker,
		FxRatesService:          fxSvc,
		FxRatesHandlers:         fxHandlers,
		FxRatesRepo:             fxRepo,
		PartnerEdiInbound:       partnerEdiIn,
		PartnerEdiOutbound:      partnerEdiOut,
		ARDunningWorker:         arDunning,
		PayoutService:           payoutSvc,
		BillingInvoiceWorker:    billingInvoiceWorker,
		ForecastAccuracy:        forecastAccuracy,
		ForecastRunner:          forecastRunner,
		cleanup:                 cleanup,
	}

	if consumers != nil {
		app.NotificationConsumer = consumers.notificationConsumer
		app.OrderEventConsumer = consumers.orderEventConsumer
		app.WarehouseEventConsumer = consumers.warehouseEventConsumer
		app.ReturnsEventConsumer = consumers.returnsEventConsumer
		app.ClaimsEventConsumer = consumers.claimsEventConsumer
		app.BillingTierConsumer = consumers.billingTierConsumer
		app.PartnerEventConsumer = consumers.partnerEventConsumer
		app.TwinEventConsumer = consumers.twinEventConsumer
	}

	return app, nil
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
