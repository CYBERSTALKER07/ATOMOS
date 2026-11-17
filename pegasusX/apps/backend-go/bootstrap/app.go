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
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/allocation"
	"github.com/pegasusx/pegasusx/apps/backend-go/analytics"
	"github.com/pegasusx/pegasusx/apps/backend-go/anomalies"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap/memory"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/cashrecon"
	"github.com/pegasusx/pegasusx/apps/backend-go/catalog"
	"github.com/pegasusx/pegasusx/apps/backend-go/claims"
	"github.com/pegasusx/pegasusx/apps/backend-go/compliance"
	"github.com/pegasusx/pegasusx/apps/backend-go/connectors"
	"github.com/pegasusx/pegasusx/apps/backend-go/controltower"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/creditnote"
	"github.com/pegasusx/pegasusx/apps/backend-go/demand"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/optimizerclient"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/plan"
	"github.com/pegasusx/pegasusx/apps/backend-go/driver"
	"github.com/pegasusx/pegasusx/apps/backend-go/esf"
	"github.com/pegasusx/pegasusx/apps/backend-go/eta"
	"github.com/pegasusx/pegasusx/apps/backend-go/exportpacks"
	"github.com/pegasusx/pegasusx/apps/backend-go/factory"
	"github.com/pegasusx/pegasusx/apps/backend-go/featureflags"
	"github.com/pegasusx/pegasusx/apps/backend-go/forecastv2"
	"github.com/pegasusx/pegasusx/apps/backend-go/foundry"
	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/globalproducts"
	"github.com/pegasusx/pegasusx/apps/backend-go/iam"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/infraroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/innlookup"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/services/billing"
	"github.com/pegasusx/pegasusx/apps/backend-go/inventory"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafka"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafkautil"
	"github.com/pegasusx/pegasusx/apps/backend-go/laborcapacity"
	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
	"github.com/pegasusx/pegasusx/apps/backend-go/mfa"
	"github.com/pegasusx/pegasusx/apps/backend-go/multientity"
	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
	"github.com/pegasusx/pegasusx/apps/backend-go/ontology"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/orgoidc"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/p2p"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner"
	"github.com/pegasusx/pegasusx/apps/backend-go/payload"
	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
	"github.com/pegasusx/pegasusx/apps/backend-go/payout"
	"github.com/pegasusx/pegasusx/apps/backend-go/periods"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
	"github.com/pegasusx/pegasusx/apps/backend-go/platform"
	"github.com/pegasusx/pegasusx/apps/backend-go/platformadmin"
	"github.com/pegasusx/pegasusx/apps/backend-go/pricing"
	"github.com/pegasusx/pegasusx/apps/backend-go/promotion"
	"github.com/pegasusx/pegasusx/apps/backend-go/proposals"
	"github.com/pegasusx/pegasusx/apps/backend-go/pulse"
	"github.com/pegasusx/pegasusx/apps/backend-go/recon"
	"github.com/pegasusx/pegasusx/apps/backend-go/replenishment"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailer"
	"github.com/pegasusx/pegasusx/apps/backend-go/returns"
	"github.com/pegasusx/pegasusx/apps/backend-go/roi"
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
	"github.com/pegasusx/pegasusx/apps/backend-go/valuation"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouse"
	"github.com/pegasusx/pegasusx/apps/backend-go/whatif"
	"github.com/pegasusx/pegasusx/apps/backend-go/workerlease"
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
	SMSTransport          *notifications.SMSTransport
	SupplierService       *supplier.Service
	OrgOIDC               *orgoidc.Service
	INNLookup             *innlookup.Service
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
	WebhookReconciler        *payment.WebhookReconciler
	WarehouseService         *warehouse.Service
	ReturnsService           *returns.Service
	TaxService               *tax.Service
	ComplianceService        *compliance.Service
	EvidenceVault            *storage.Vault
	ControlTowerService      *controltower.Service
	ControlTowerHandlers     *controltower.Handlers
	ControlTowerWorker       *controltower.Worker
	DemandService            *demand.Service
	LaborCapacityService     *laborcapacity.Service
	ETAService               *eta.Service
	FirebaseVerifier         auth.FirebaseVerifier
	OrderService             *order.Service
	ClaimsService            *claims.Service
	CreditService            *credit.Service
	CreditPolicyService      *credit.PolicyService
	ARService                *ar.Service
	GL                       *gl.Service
	Periods                  *periods.Service
	Approvals                *approvals.Service
	Valuation                *valuation.Service
	ESF                      *esf.Service
	Recon                    *recon.Service
	ForecastV2               *forecastv2.Service
	ReplenishmentSuggestions *replenishment.SuggestionsAPI
	Proposals                *proposals.Service
	ROI                      *roi.Service
	CashReconHandlers        *cashrecon.Handlers
	Connectors               *connectors.Service
	ExportPacks              *exportpacks.Service
	MultiEntity              *multientity.Service
	P2P                      *p2p.Service
	Anomalies                *anomalies.Service
	WhatIf                   *whatif.Service
	CashReconService         *cashrecon.Service
	CashReconEscalation      *cashrecon.EscalationWorker
	CreditNoteHandlers       *creditnote.Handlers
	CreditNoteService        *creditnote.Service
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
	OutboxPublisher        outbox.Publisher
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
	OntologyEventConsumer   *kafka.Consumer
	BQSinkConsumer          *kafka.Consumer
	ARDunningWorker         *ar.DunningWorker
	ARPromiseBreachWorker   *ar.PromiseBreachWorker
	PayoutService           *payout.Service
	BillingInvoiceWorker    *billing.InvoiceWorker
	ForecastAccuracy        *planning.AccuracyService
	ForecastRunner          *planning.ForecastRunner
	FoundryEngine           *foundry.Engine
	Audit                   *audit.Service
	IAM                     *iam.Service
	Ontology                *ontology.Service
	LeaseManager            workerlease.Manager
	WorkerHolderID          string
	sweeperWg               *sync.WaitGroup
	cancelSweepers          context.CancelFunc
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

	appCtx, cancelSweepers := context.WithCancel(ctx)
	sweeperWg := &sync.WaitGroup{}
	cleanup := make([]func(), 0, 4)
	if spannerCleanup != nil {
		cleanup = append(cleanup, spannerCleanup)
	}

	kafkaAuth := kafkautil.ClientAuth{Mode: cfg.KafkaAuthMode, Username: cfg.KafkaSASLUsername}
	outboxPublisher, kafkaEnabled, kafkaCleanup, err := setupKafkaPublisher(cfg, kafkaAuth, log)
	if err != nil {
		cancelSweepers()
		return nil, err
	}
	if kafkaCleanup != nil {
		cleanup = append(cleanup, kafkaCleanup)
	}

	if cfg.RequireInfraAdapters && spannerClient == nil && !cfg.TestingMode {
		cancelSweepers()
		return nil, fmt.Errorf("require infra adapters: spanner unavailable")
	}

	var leaseMgr workerlease.Manager
	if spannerClient != nil {
		leaseMgr = workerlease.NewSpannerManager(spannerClient)
	} else if cfg.TestingMode || cfg.AllowMemoryFallback {
		leaseMgr = workerlease.NewMemoryManager()
	} else {
		cancelSweepers()
		return nil, fmt.Errorf("workerlease unavailable: spanner unavailable")
	}
	workerHolderID := workerlease.DefaultHolderID("pegasusx-backend")

	outboxRelay := outbox.NewRelay(relayStore, outboxPublisher, outbox.RelayConfig{}, log)

	var seedRepo seed.Repository
	if spannerClient != nil {
		seedRepo = &runtimeSeedRepository{client: spannerClient}
	}
	supplierSeed, err := seed.EnsureSupplier(ctx, seedRepo,
		cfg.SeedSupplierName, cfg.SeedSupplierCountry, cfg.SeedSupplierCurrency, log)
	if err != nil {
		cancelSweepers()
		return nil, fmt.Errorf("seed supplier: %w", err)
	}
	if spannerClient != nil {
		if _, serr := seed.SeedUzbekistanHolidays(ctx, spannerClient, supplierSeed.SupplierID, log); serr != nil {
			log.Warn("uzbekistan seasonal holidays seed skipped or failed", "err", serr)
		}
	}

	var retailerRepo retailer.Repository
	var supplierRepo supplier.Repository

	if spannerClient != nil {
		retailerRepo = retailer.NewSpannerRepository(spannerClient, supplierSeed.SupplierID)
		supplierRepo = supplier.NewSpannerRepository(spannerClient)
		log.Info("retailer and supplier repositories enabled", "backend", "spanner")
	} else {
		if err := cfg.ensureMemoryFallbackAllowed("retailer/supplier repositories"); err != nil {
			cancelSweepers()
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
		catalogSvc.SetIdempotencyStore(idemStore)
		log.Info("catalog service enabled", "backend", "spanner")
		if globalproducts.Enabled() {
			gpRepo := globalproducts.NewSpannerRepository(spannerClient)
			globalProductsSvc = globalproducts.NewService(gpRepo, log)
			if err := globalProductsSvc.EnsureBootstrap(ctx); err != nil {
				log.Warn("global products uom seed failed", "err", err)
			}
			catalogSvc.SetGlobalProductHook(globalproducts.CatalogHook{Svc: globalProductsSvc})
			if cfg.RunsWorkers() {
				workerlease.StartLeasedLoop(appCtx, sweeperWg, leaseMgr, workerlease.Config{
					LeaseName: "sweeper:global_products_match",
					HolderID:  workerHolderID,
					TTL:       90 * time.Second,
					Log:       log,
				}, func(leaseCtx context.Context) {
					globalProductsSvc.StartMatchWorker(leaseCtx, 2*time.Minute)
				})
			}
			log.Info("global products service enabled")
		}
	}

	var inventorySvc *inventory.Service
	if spannerClient != nil {
		inventoryRepo := inventory.NewSpannerRepository(spannerClient)
		inventorySvc = inventory.NewService(inventoryRepo, cacheClient, log)
		if cfg.RunsWorkers() {
			startSpawningSweeper(appCtx, sweeperWg, leaseMgr, "sweeper:inventory_holds", workerHolderID, log, func(leaseCtx context.Context, _ *sync.WaitGroup) {
				inventorySvc.StartHoldSweeper(leaseCtx, 1*time.Minute)
			})
		}
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
	smsTransport := &notifications.SMSTransport{
		Provider:   os.Getenv("SMS_PROVIDER"),
		BaseURL:    os.Getenv("PLAYMOBILE_BASE_URL"),
		Login:      os.Getenv("PLAYMOBILE_LOGIN"),
		Password:   os.Getenv("PLAYMOBILE_PASSWORD"),
		Originator: os.Getenv("PLAYMOBILE_ORIGINATOR"),
		AccountSID: os.Getenv("TWILIO_ACCOUNT_SID"),
		AuthToken:  os.Getenv("TWILIO_AUTH_TOKEN"),
		FromNumber: os.Getenv("TWILIO_FROM"),
		Log:        log,
	}
	smsTransport.SetCircuitBreaker(outboundCircuits.Notification)

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
	var innCache innlookup.CacheStore
	var innTenants innlookup.TenantDirectory
	var innJoinStore innlookup.JoinRequestStore
	if spannerClient != nil {
		st := innlookup.NewSpannerStore(spannerClient)
		innCache = st
		innTenants = st
		innJoinStore = st
	} else {
		memINN := innlookup.NewMemoryStore()
		innCache = memINN
		innTenants = memINN
		innJoinStore = memINN
	}
	innLookupSvc := innlookup.NewService(innlookup.Config{
		Adapter:      innlookup.SoliqAdapterFromEnv(),
		Cache:        innCache,
		Tenants:      innTenants,
		JoinRequests: innJoinStore,
	})

	supplierSvc := supplier.NewService(supplier.ServiceConfig{
		Repo:                       supplierRepo,
		Cache:                      cacheClient,
		Idem:                       idemStore,
		Locations:                  driverLocations,
		InventoryService:           supplierInventory,
		DashboardQuery:             dashboardQuery,
		INNLookup:                  innLookupSvc,
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
	retailerSvc.SetINNLookup(innLookupSvc)
	var oidcStore orgoidc.Store
	if spannerClient != nil {
		oidcStore = orgoidc.NewSpannerStore(spannerClient)
	} else {
		if err := cfg.ensureMemoryFallbackAllowed("orgoidc store"); err != nil {
			cancelSweepers()
			return nil, err
		}
		oidcStore = orgoidc.NewMemoryStore()
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
		INNLookup:      innLookupSvc,
		INNRequired:    cfg.INNVerificationRequired,
		OnJoinRequest: func(ctx context.Context, supplierID string, jr innlookup.JoinRequest) {
			if notifSvc == nil {
				return
			}
			_ = notifSvc.CreateNotification(
				ctx,
				supplierID,
				string(auth.RoleAdmin),
				"JOIN_REQUEST_CREATED",
				"Team access request for matching INN",
				fmt.Sprintf("%s (%s) requested %s access for INN %s", jr.RequesterName, jr.RequesterPhone, jr.RoleSought, jr.NormalizedINN),
				"/settings/team?tab=join-requests",
			)
		},
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
			cancelSweepers()
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
			cancelSweepers()
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
			cancelSweepers()
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
		if err := cfg.ensureMemoryFallbackAllowed("credit policy and AR repositories"); err != nil {
			cancelSweepers()
			return nil, err
		}
		creditPolicyRepo = credit.NewMemoryPolicyRepository()
		arRepo = ar.NewMemoryRepository()
	}
	creditPolicySvc := credit.NewPolicyService(creditPolicyRepo, creditSvc)
	creditSvc.SetPolicyGate(creditPolicySvc)
	arSvc := ar.NewService(arRepo)
	arSvc.SetCreditClearer(creditSvc.ClearBalanceInTxn)
	retailerSvc.SetMigrationAR(&migrationARAdapter{arSvc: arSvc, spanner: spannerClient})
	if notifSvc != nil {
		retailerSvc.SetMigrationNotifier(notifSvc.CreateNotification)
	}
	arDunning := ar.NewDunningWorker(arSvc, log)
	arPromiseBreach := ar.NewPromiseBreachWorker(arSvc, log)
	arDunning.SetDelinquencyBump(func(ctx context.Context, supplierID, retailerID string) error {
		return creditSvc.BumpDelinquency(ctx, retailerID, supplierID)
	})
	arDunning.SetAutoHold(func(ctx context.Context, supplierID, retailerID string) error {
		return creditSvc.FreezeProfile(ctx, retailerID, supplierID, "dunning_credit_hold")
	})
	// Phase 1 money surfaces: payouts (bank-file) + billing fee schedule.
	// Note: feeResolver is shared across payoutSvc, orderSvc, and billingInvoiceWorker;
	// billingInvoiceWorker reconciles settled per-order commissions (COMMISSION_SETTLED / InvoiceSettlementSlices)
	// and refunded GMV (ORDER_REFUNDED / OrderPaymentLegs) so GmvBps is never double-charged.
	var payoutSvc *payout.Service
	var billingInvoiceWorker *billing.InvoiceWorker
	var feeResolver *billing.FeeScheduleResolver
	if spannerClient != nil {
		feeResolver = billing.NewFeeScheduleResolver(spannerClient)
		payout.SettlementJournal = payment.WriteSettlementJournal
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
	var glSvc *gl.Service
	var reconSvc *recon.Service
	cashReconRequired := strings.EqualFold(strings.TrimSpace(os.Getenv("CASH_RECONCILIATION_REQUIRED")), "true")
	if spannerClient != nil {
		complianceSvc = compliance.NewService(compliance.NewSpannerRepository(spannerClient), slog.Default())
		taxSvc = tax.NewService(tax.NewSpannerRepository(spannerClient), cacheClient, slog.Default())
		routingSvc = routing.NewService(spannerClient)
		pricingSvc = pricing.NewService(pricing.NewSpannerRepository(spannerClient))
		cashReconRepo := cashrecon.NewSpannerRepository(spannerClient)
		cashReconSvc = cashrecon.NewService(cashReconRepo, cashReconRepo)
		cashReconHandlers = &cashrecon.Handlers{Svc: cashReconSvc, Idem: idemStore}
		creditNoteRepo := creditnote.NewSpannerRepository(spannerClient)
		creditNoteSvc = creditnote.NewService(creditNoteRepo)
		creditNoteHandlers = &creditnote.Handlers{
			Svc: creditNoteSvc,
			// Last-resort seed only when PreferTenantSupplierID + claims are empty.
			SupplierID: func() string { return supplierSeed.SupplierID },
		}
		glSvc = gl.NewService(spannerClient)
		seedPack, _ := auth.CountryFromContext(ctx, supplierSeed.SupplierID)
		if err := glSvc.SeedDefaults(ctx, supplierSeed.SupplierID, seedPack, cfg.SeedSupplierCurrency); err != nil {
			log.Warn("gl defaults seed skipped or failed", "err", err)
		}
		reconSvc = recon.NewService(spannerClient)
	}
	auditSvc, iamSvc := setupAuditAndIAM(ctx, spannerClient, cfg, log)
	ontSvc := setupOntologyService(spannerClient, cfg, log)

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
		Locations:             driverLocations,
	})
	orderSvc.SetManifestStore(manifestStore)
	orderSvc.SetDoorstepShortfall(payment.WriteDoorstepShortfall)
	if feeResolver != nil {
		orderSvc.SetPerOrderFeeResolver(func(ctx context.Context, supplierID string) (int64, string, error) {
			sched, err := feeResolver.Resolve(ctx, supplierID)
			if err != nil {
				return 0, "", err
			}
			return sched.PerOrderMinor, sched.Currency, nil
		})
	}
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
		if err := cfg.ensureMemoryFallbackAllowed("claims repository"); err != nil {
			cancelSweepers()
			return nil, err
		}
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
	if fn := CheckOrderDraftSoD(iamSvc); fn != nil {
		orderSvc.SetDraftSoDCheck(fn)
	}
	if cfg.RunsWorkers() {
		startSpawningSweeper(appCtx, sweeperWg, leaseMgr, "sweeper:order_preorder", workerHolderID, log, func(leaseCtx context.Context, localWg *sync.WaitGroup) {
			order.StartPreorderSweeperContext(leaseCtx, orderSvc, localWg)
		})
		startSpawningSweeper(appCtx, sweeperWg, leaseMgr, "sweeper:order_fiscal", workerHolderID, log, func(leaseCtx context.Context, localWg *sync.WaitGroup) {
			order.StartFiscalSweeperContext(leaseCtx, orderSvc, localWg)
		})
		startSpawningSweeper(appCtx, sweeperWg, leaseMgr, "sweeper:order_stale_reservation", workerHolderID, log, func(leaseCtx context.Context, localWg *sync.WaitGroup) {
			order.StartStaleReservationSweeperContext(leaseCtx, orderSvc, localWg)
		})
		startSpawningSweeper(appCtx, sweeperWg, leaseMgr, "sweeper:credit_overdue", workerHolderID, log, func(leaseCtx context.Context, localWg *sync.WaitGroup) {
			credit.StartOverdueCreditSweeperContext(leaseCtx, creditSvc, 1*time.Minute, localWg)
		})
		startReconNightlyWorker(appCtx, sweeperWg, leaseMgr, workerHolderID, log, reconSvc)
		startAuditAnchorDailyWorker(appCtx, sweeperWg, leaseMgr, workerHolderID, log, auditSvc)
		startOntologyRebuildNightlyWorker(appCtx, sweeperWg, leaseMgr, workerHolderID, log, ontSvc)
		workerlease.StartLeasedLoop(appCtx, sweeperWg, leaseMgr, workerlease.Config{
			LeaseName: "sweeper:order_negotiation",
			HolderID:  workerHolderID,
			TTL:       60 * time.Second,
			Log:       log,
		}, func(leaseCtx context.Context) {
			orderSvc.StartNegotiationSweeper(leaseCtx)
		})
		workerlease.StartLeasedLoop(appCtx, sweeperWg, leaseMgr, workerlease.Config{
			LeaseName: "sweeper:order_deferred_payment",
			HolderID:  workerHolderID,
			TTL:       60 * time.Second,
			Log:       log,
		}, func(leaseCtx context.Context) {
			orderSvc.StartDeferredPaymentSweeper(leaseCtx, 5*time.Minute)
		})
		workerlease.StartLeasedLoop(appCtx, sweeperWg, leaseMgr, workerlease.Config{
			LeaseName: "sweeper:order_saga_recovery",
			HolderID:  workerHolderID,
			TTL:       60 * time.Second,
			Log:       log,
		}, func(leaseCtx context.Context) {
			order.StartSagaRecoveryWorker(leaseCtx, orderSvc, 30*time.Second)
		})
		if spannerClient != nil {
			workerlease.StartLeasedLoop(appCtx, sweeperWg, leaseMgr, workerlease.Config{
				LeaseName: "sweeper:payment_ledger_reconciler",
				HolderID:  workerHolderID,
				TTL:       60 * time.Second,
				Log:       log,
			}, func(leaseCtx context.Context) {
				payment.StartLedgerReconciler(leaseCtx, payment.NewSpannerLedgerReconciler(spannerClient, log), 5*time.Minute)
			})
			acquired, backfillErr := workerlease.WithLease(ctx, leaseMgr, workerlease.Config{
				LeaseName: "backfill:scheduled_reservations",
				HolderID:  workerHolderID,
				TTL:       2 * time.Minute,
				Log:       log,
			}, func(leaseCtx context.Context) error {
				n, err := orderSvc.BackfillScheduledReservations(leaseCtx, 500)
				if err != nil {
					return err
				}
				if n > 0 {
					log.Info("scheduled reservation backfill complete", "orders", n)
				}
				return nil
			})
			if backfillErr != nil {
				log.Warn("scheduled reservation backfill failed", "err", backfillErr)
			} else if !acquired {
				log.Info("scheduled reservation backfill skipped; lease held by another worker")
			}
		}
	}
	var optimizerCli *optimizerclient.Client
	if strings.TrimSpace(cfg.OptimizerBaseURL) != "" && strings.TrimSpace(cfg.InternalAPIKey) != "" {
		optimizerCli = optimizerclient.New(cfg.OptimizerBaseURL, cfg.InternalAPIKey).WithOSRM(osrmClient).WithBreaker(outboundCircuits.Optimizer)
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
		UnifiedGraph:         planning.AdaptOntologyService(ontSvc, strings.TrimSpace(os.Getenv("CELL_ID"))),
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
		simulator.StartControlTowerSimulationContext(appCtx, telemetryHub, supplierSeed.SupplierID, warehouseNodeID)
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
			cancelSweepers()
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
				Count:            snap.Count,
				OrderIDs:         snap.OrderIDs,
				Frozen:           snap.Frozen,
				RetryQueuedCount: snap.RetryQueuedCount,
				FiscalIssueCount: snap.FiscalIssueCount,
				RetryOrderIDs:    snap.RetryOrderIDs,
			}, nil
		}
	}
	// OFD adapter: cell default from env. GS-M2 fail-closes planned/PEPPOL packs at fiscalize.
	fiscalProvider := order.ProviderFromEnv()
	if msp, ok := fiscalProvider.(*order.MySoliqProvider); ok {
		msp.WithBreaker(outboundCircuits.Soliq)
	}
	orderSvc.SetFiscalProvider(fiscalProvider)
	if spannerClient != nil {
		var fiscalSMS *notifications.SMSTransport
		if strings.TrimSpace(smsTransport.Provider) != "" {
			fiscalSMS = smsTransport
		}
		tgFiscalSender := notifications.NewTelegramBotSenderFromEnv(log)
		tgFiscalSender.Breaker = outboundCircuits.Notification
		orderSvc.SetFiscalNotifier(&order.DefaultFiscalSuccessNotifier{
			Spanner:  spannerClient,
			SMS:      fiscalSMS,
			Telegram: tgFiscalSender,
			Log:      log,
		})
	}
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
		SpannerClient:                   spannerClient,
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
		UzumWebhookSecret:               cfg.UzumWebhookSecret,
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
		cancelSweepers()
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
	orderSvc.SetCreditTermsLookup(creditPolicySvc.ResolveTermsForOrder)
	// Claims chargeback: supplier ledger debit + optional Global Pay partial refund.
	claimsSvc.SetSettler(&claimPaymentSettler{pay: paymentSvc})
	claimsSvc.SetStoreCredit(creditSvc)
	var warehouseRepo warehouse.Repository
	if spannerClient != nil {
		warehouseRepo = warehouse.NewSpannerRepository(spannerClient)
		log.Info("warehouse repository enabled", "backend", "spanner")
	} else {
		if err := cfg.ensureMemoryFallbackAllowed("warehouse repository"); err != nil {
			cancelSweepers()
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
	configureWMSFlagsFromEnv()
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

	var policyRepo platform.PolicyRepository
	var tokenRepo platform.DeviceTokenRepository
	var sessionChecker platform.SessionChecker
	if spannerClient != nil {
		policyRepo = platform.NewSpannerPolicyRepository(spannerClient)
		tokenRepo = platform.NewSpannerDeviceTokenRepository(spannerClient)
		sessionChecker = platform.NewSpannerSessionChecker(spannerClient)
	} else {
		if err := cfg.ensureMemoryFallbackAllowed("platform policy/token repositories"); err != nil {
			cancelSweepers()
			return nil, err
		}
		policyRepo = platform.NewMemoryPolicyRepository()
		tokenRepo = platform.NewMemoryDeviceTokenRepository()
		sessionChecker = platform.NoopSessionChecker{}
	}
	platformSvc := platform.NewService(policyRepo, sessionChecker, log)
	platformHandler := platform.NewHandler(platform.HandlerConfig{
		Service:      platformSvc,
		DeviceTokens: tokenRepo,
		Log:          log,
	})
	var platformAdminRepo platformadmin.Repository
	var featureFlagRepo featureflags.Repository
	var mfaRepo mfa.Repository
	if spannerClient != nil {
		platformAdminRepo = platformadmin.NewSpannerRepository(spannerClient)
		featureFlagRepo = featureflags.NewSpannerRepository(spannerClient)
		mfaRepo = mfa.NewSpannerRepository(spannerClient)
	} else {
		if err := cfg.ensureMemoryFallbackAllowed("platformadmin/featureflags/mfa repositories"); err != nil {
			cancelSweepers()
			return nil, err
		}
		platformAdminRepo = platformadmin.NewMemoryRepository()
		featureFlagRepo = featureflags.NewMemoryRepository()
		mfaRepo = mfa.NewMemoryRepository()
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
	mfaCache := mfa.NewRedisReplayCache(redisClientOrNil(redisAdapter))
	mfaSvc := mfa.NewService(mfaRepo, cfg.JWTIssuer, cfg.PlatformAdminMFARequired, platformAdminSvc, mfaCache)
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
	if retailerSvc != nil {
		retailerSvc.OnRegistered = func(ctx context.Context, retailerID, name string) error {
			return platformAdminSvc.EnsurePending(ctx, platformadmin.TenantRetailer, retailerID, name)
		}
	}

	pushBridge, err := setupPushBridge(cfg, spannerClient, tokenRepo, log)
	if err != nil {
		cancelSweepers()
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
		cancelSweepers()
		return nil, err
	}
	arDunning.SetNotify(dunningNotify)
	arPromiseBreach.SetOnBreach(func(ctx context.Context, supplierID, retailerID, _ string) error {
		if err := creditPolicySvc.HoldRelationship(ctx, supplierID, retailerID, "system:promise_breach", "SYSTEM"); err != nil && !errors.Is(err, credit.ErrProfileNotFound) {
			return err
		}
		if err := creditSvc.FreezeProfile(ctx, retailerID, supplierID, "promise_breach"); err != nil && !errors.Is(err, credit.ErrProfileNotFound) {
			return err
		}
		return nil
	})
	arPromiseBreach.SetNotify(func(ctx context.Context, promise ar.PaymentPromise, outcome string) error {
		title := "Payment promise update"
		body := fmt.Sprintf("Promise %s: %s", promise.PromiseID, outcome)
		switch outcome {
		case ar.PromiseStatusFulfilled:
			title = "To'lov va'dasi bajarildi"
		case ar.PromiseStatusBreached:
			title = "To'lov va'dasi muddati o'tdi"
		}
		if notifSvc != nil {
			if err := notifSvc.CreateNotification(ctx, promise.RetailerID, "RETAILER", "AR_PROMISE_"+outcome, title, body, "/credit"); err != nil {
				log.WarnContext(ctx, "promise retailer notification failed", "promise_id", promise.PromiseID, "err", err)
			}
			if err := notifSvc.CreateNotification(ctx, promise.SupplierID, "ADMIN", "AR_PROMISE_"+outcome, title, body, "/credit/collections"); err != nil {
				log.WarnContext(ctx, "promise supplier notification failed", "promise_id", promise.PromiseID, "err", err)
			}
		}
		if pushBridge != nil {
			pushData := map[string]string{
				"type":        "AR_PROMISE_" + outcome,
				"promise_id":  promise.PromiseID,
				"invoice_id":  promise.InvoiceID,
				"supplier_id": promise.SupplierID,
				"retailer_id": promise.RetailerID,
				"outcome":     outcome,
			}
			pushBridge.NotifyActor(ctx, promise.RetailerID, "RETAILER", pushData)
			pushBridge.NotifyActor(ctx, promise.SupplierID, "ADMIN", pushData)
		}
		return nil
	})

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

	partnerSys, err := setupPartnerSubsystems(spannerClient, cfg, orderSvc, catalogSvc, idemStore, inventorySvc, retailerSvc, log)
	if err != nil {
		cancelSweepers()
		return nil, err
	}
	partnerSvc := partnerSys.svc
	supplierSvc.SetOnecPostings(partnerSvc.OnecPostings())
	partnerHandlers := partnerSys.handlers
	partnerKeys := partnerSys.keys
	partnerJWTSecret := partnerSys.jwtSecret
	partnerDelivery := partnerSys.delivery
	partnerExportWorker := partnerSys.exportWorker
	partnerEdiIn := partnerSys.ediInbound
	partnerEdiOut := partnerSys.ediOutbound

	var consumers *kafkaConsumers
	if kafkaEnabled && cfg.KafkaTopicMain != "" {
		c, err := setupKafkaConsumers(
			cfg, kafkaAuth, redisAdapter, spannerClient, cacheClient, pushBridge,
			notifSvc, retailerSvc, retailerHub, supplierHub, driverHub, warehouseHub,
			factoryHub, payloadHub, orderSvc, warehouseSvc, returnsSvc, claimsSvc,
			partnerSvc, fxSvc, telemetryHub, routingSvc, ontSvc, log,
		)
		if err != nil {
			cancelSweepers()
			return nil, err
		}
		consumers = c
		if c.cleanup != nil {
			cleanup = append(cleanup, c.cleanup)
		}
	}

	var bqSinkConsumer *kafka.Consumer
	if cfg.RunsWorkers() {
		bqc, _, bqCleanup, bqErr := NewBQSinkConsumer(appCtx, cfg, spannerClient, log)
		if bqErr != nil {
			cancelSweepers()
			return nil, bqErr
		}
		bqSinkConsumer = bqc
		if bqCleanup != nil {
			cleanup = append(cleanup, bqCleanup)
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

	controlTowerSvc, controlTowerHandlers, controlTowerWorker := setupControlTowerSubsystem(
		spannerClient, cacheClient, creditNoteSvc, creditSvc, routingSvc,
		notifSvc, returnsSvc, supplierSvc, replenishmentEngine, log,
	)

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
		INNLookup:               innLookupSvc,
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
		GL:                      glSvc,
		Recon:                   reconSvc,
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
		OutboxPublisher:         outboxPublisher,
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
		ARPromiseBreachWorker:   arPromiseBreach,
		PayoutService:           payoutSvc,
		BillingInvoiceWorker:    billingInvoiceWorker,
		ForecastAccuracy:        forecastAccuracy,
		ForecastRunner:          forecastRunner,
		FoundryEngine:           foundry.NewEngine().WithSpannerClient(spannerClient).WithAuditHook(BuildFoundryAuditHook(auditSvc, log)),
		Audit:                   auditSvc,
		IAM:                     iamSvc,
		Ontology:                ontSvc,
		BQSinkConsumer:          bqSinkConsumer,
		SMSTransport:            smsTransport,
		LeaseManager:            leaseMgr,
		WorkerHolderID:          workerHolderID,
		sweeperWg:               sweeperWg,
		cancelSweepers:          cancelSweepers,
		cleanup:                 cleanup,
	}

	if err := SetupWave3A(ctx, cfg, spannerClient, fiscalProvider, app, log); err != nil {
		app.Close()
		return nil, err
	}
	if err := SetupWave3B(ctx, cfg, spannerClient, app, log); err != nil {
		app.Close()
		return nil, err
	}
	if err := SetupWave3C(ctx, cfg, spannerClient, app, log); err != nil {
		app.Close()
		return nil, err
	}
	if err := SetupWave4A(ctx, cfg, spannerClient, app, log); err != nil {
		app.Close()
		return nil, err
	}
	if err := SetupWave4B(ctx, cfg, spannerClient, app, log); err != nil {
		app.Close()
		return nil, err
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
		app.OntologyEventConsumer = consumers.ontologyEventConsumer
	}

	return app, nil
}

// Close tears down long-lived resources with ordered sweeper cancellation before connection teardown.
func (a *App) Close() {
	if a == nil {
		return
	}

	// 1. Cancel background sweepers first so they stop issuing new queries
	if a.cancelSweepers != nil {
		a.cancelSweepers()
	}

	// 2. Wait for background sweeper passes to finish (bounded 5s timeout)
	if a.sweeperWg != nil {
		sweepersDone := make(chan struct{})
		go func() {
			a.sweeperWg.Wait()
			close(sweepersDone)
		}()
		select {
		case <-sweepersDone:
			slog.Info("all background sweepers shut down cleanly")
		case <-time.After(5 * time.Second):
			slog.Warn("timed out waiting for background sweepers to finish")
		}
	}

	// 3. Clean up database, messaging, and connection pools in reverse construction order
	for i := len(a.cleanup) - 1; i >= 0; i-- {
		if a.cleanup[i] != nil {
			a.cleanup[i]()
		}
	}
	if a.Reliability != nil {
		a.Reliability.Close()
	}
	if a.Cache != nil {
		if err := a.Cache.Close(); err != nil {
			slog.Warn("cache close failed", "err", err)
		}
	}
}

// IdempotencyMiddleware returns the configured idempotency middleware.
func (a *App) IdempotencyMiddleware() func(http.Handler) http.Handler {
	if a == nil || a.Idempotency == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	return idempotency.Middleware(a.Idempotency)
}

// IdempotencyMiddleware wraps idempotency.Middleware for bootstrap consumers.
func IdempotencyMiddleware(store idempotency.Store) func(http.Handler) http.Handler {
	return idempotency.Middleware(store)
}

func startSpawningSweeper(
	ctx context.Context,
	wg *sync.WaitGroup,
	leaseMgr workerlease.Manager,
	leaseName string,
	holderID string,
	log *slog.Logger,
	startFn func(leaseCtx context.Context, localWg *sync.WaitGroup),
) {
	workerlease.StartLeasedLoop(ctx, wg, leaseMgr, workerlease.Config{
		LeaseName: leaseName,
		HolderID:  holderID,
		TTL:       60 * time.Second,
		Log:       log,
	}, func(leaseCtx context.Context) {
		var localWg sync.WaitGroup
		startFn(leaseCtx, &localWg)
		<-leaseCtx.Done()
		localWg.Wait()
	})
}

func configureWMSFlagsFromEnv() {
	envTrue := func(key string) bool {
		return strings.EqualFold(strings.TrimSpace(os.Getenv(key)), "true")
	}
	stocklots.SetLotsEnabled(envTrue("WMS_LOTS_ENABLED"))
	stocklots.SetPickWavesEnabled(envTrue("WMS_PICK_WAVES_ENABLED"))
	stocklots.SetCycleCountsEnabled(envTrue("WMS_CYCLE_COUNTS_ENABLED"))
	stocklots.SetPickSShapeEnabled(envTrue("WMS_PICK_SSHAPE_ENABLED"))
	stocklots.SetSealSoftWarnEnabled(envTrue("WMS_SEAL_SOFT_WARN"))
	stocklots.SetColdChainEnabled(envTrue("WMS_COLD_CHAIN_ENABLED"))
	stocklots.SetLoadLedgerEnabled(envTrue("PAYLOAD_LOAD_LEDGER_ENABLED"))
	stocklots.SetLaborCapacityEnforce(envTrue("LABOR_CAPACITY_ENFORCE"))
}
