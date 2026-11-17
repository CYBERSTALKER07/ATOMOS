package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/anomalyroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvalroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auditroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/cashreconroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/catalogroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/compliance"
	"github.com/pegasusx/pegasusx/apps/backend-go/connectorroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/controltowerroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/countrycfg"
	"github.com/pegasusx/pegasusx/apps/backend-go/creditnoteroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/creditroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/deliveryroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/demandroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/driverroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/entityresolutionroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/esfroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/etaroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/exportpackroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/factoryroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/featureflags"
	"github.com/pegasusx/pegasusx/apps/backend-go/forecastroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/foundryroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
	"github.com/pegasusx/pegasusx/apps/backend-go/geolocation"
	"github.com/pegasusx/pegasusx/apps/backend-go/globalproductsroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/glroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/iamroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/infraroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/services/billing"
	"github.com/pegasusx/pegasusx/apps/backend-go/laborcapacityroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/mfa"
	"github.com/pegasusx/pegasusx/apps/backend-go/migration"
	"github.com/pegasusx/pegasusx/apps/backend-go/migrationroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/multientityroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/ontologyroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/orderroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/p2proutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner"
	payloadroutes "github.com/pegasusx/pegasusx/apps/backend-go/payloaderoutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/paymentroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/payout"
	"github.com/pegasusx/pegasusx/apps/backend-go/periodroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
	"github.com/pegasusx/pegasusx/apps/backend-go/platformadmin"
	"github.com/pegasusx/pegasusx/apps/backend-go/platformroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/promotionroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/proposalroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/pulseroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/reconroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/replenishment"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailerroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/returnsroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/roiroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/simulator"
	"github.com/pegasusx/pegasusx/apps/backend-go/staffinvite"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
	"github.com/pegasusx/pegasusx/apps/backend-go/storageroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplierroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/syncroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/taxroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetryroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/updateroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/valuationroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouseroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/webhookroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/whatifroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
	"go.uber.org/fx"
)

// HTTPRouterParams groups the dependencies required to construct the unified
// chi.Router via uber-go/fx parameter object injection (fx.In).
type HTTPRouterParams struct {
	fx.In

	Config           *bootstrap.Config
	App              *bootstrap.App
	Logger           *slog.Logger
	SyncCache        cache.Backend `optional:"true"`
	RetailerHub      *ws.Hub       `name:"retailer_hub" optional:"true"`
	SupplierHub      *ws.Hub       `name:"supplier_hub" optional:"true"`
	DriverHub        *ws.Hub       `name:"driver_hub" optional:"true"`
	PayloadHub       *ws.Hub       `name:"payload_hub" optional:"true"`
	WarehouseHub     *ws.Hub       `name:"warehouse_hub" optional:"true"`
	FactoryHub       *ws.Hub       `name:"factory_hub" optional:"true"`
	TelemetryHub     *ws.Hub       `name:"telemetry_hub" optional:"true"`
	PlatformAdminHub *ws.Hub       `name:"platform_admin_hub" optional:"true"`
}

// NewHTTPRouter constructs and mounts all middleware and domain route trees.
func NewHTTPRouter(p HTTPRouterParams) chi.Router {
	cfg := p.Config
	app := p.App
	log := p.Logger
	if log == nil {
		log = slog.Default()
	}

	retailerHub := coalesceHub(p.RetailerHub, app.RetailerHub)
	supplierHub := coalesceHub(p.SupplierHub, app.SupplierHub)
	driverHub := coalesceHub(p.DriverHub, app.DriverHub)
	payloadHub := coalesceHub(p.PayloadHub, app.PayloadHub)
	warehouseHub := coalesceHub(p.WarehouseHub, app.WarehouseHub)
	factoryHub := coalesceHub(p.FactoryHub, app.FactoryHub)
	telemetryHub := coalesceHub(p.TelemetryHub, app.TelemetryHub)
	platformAdminHub := coalesceHub(p.PlatformAdminHub, app.PlatformAdminHub)

	r := chi.NewRouter()
	r.Use(bootstrap.TraceMiddleware)
	r.Use(telemetry.HTTPMetricsMiddleware)
	r.Use(bootstrap.MaxBytesMiddlewareWithLocalUpload(0, !auth.IsEnforcedEnv() && strings.TrimSpace(os.Getenv("IMPORT_LOCAL_FILE_ROOT")) != ""))
	r.Use(bootstrap.DevCORSMiddleware())
	r.Use(auth.SessionAuth(cfg.JWTSecret))
	// Partner API keys (pxk_*) and OAuth access tokens — attach principal before rate limiting.
	if app.PartnerKeys != nil {
		r.Use(partner.AuthMiddlewareOpts(partner.AuthOptions{
			Keys: app.PartnerKeys, JWTSecret: app.PartnerJWTSecret,
		}))
	}
	r.Use(auth.AttachTenantFromClaims)
	r.Use(auth.RequireTenant(cfg.TenantContextEnforced))

	// GS-I: process-global Auth0 wrap is forbidden (it 401s native HS256).
	// Per-supplier OIDC is orgoidc (attach + /v1/auth/oidc/exchange).
	if os.Getenv("AUTH0_DOMAIN") != "" {
		log.Warn("AUTH0_DOMAIN is set but ignored; GS-I does not wrap the router")
	}

	if app.Reliability != nil {
		r.Use(app.Reliability.Middleware)
	}
	if app.Audit != nil {
		r.Use(audit.AuditMiddleware(app.Audit))
	}
	if app.Idempotency != nil {
		r.Use(app.IdempotencyMiddleware())
	}
	infraroutes.RegisterRoutes(r, app.InfraHealth)

	geocodeSvc := geolocation.NewService(cfg.GoogleMapsAPIKey, app.Cache)
	platformroutes.RegisterRoutes(r, platformroutes.Deps{
		Handler:        app.PlatformHandler,
		GeocodeHandler: geolocation.NewHandler(geocodeSvc),
		TenantRegister: app.TenantRegister,
		INNLookup:      app.INNLookup,
		JWTSecret:      cfg.JWTSecret,
		JWTIssuer:      cfg.JWTIssuer,
	})
	// G4.B: durable PLATFORM_ADMIN password login (public; MFA step-up after token).
	if app.PlatformAdminHandlers != nil {
		login := &platformadmin.LoginDeps{
			Spanner:   app.Spanner,
			JWTSecret: cfg.JWTSecret,
			JWTIssuer: cfg.JWTIssuer,
		}
		r.Post("/v1/auth/platform-admin/login", login.HandleLogin)
	}
	platformadmin.RegisterRoutes(r, app.PlatformAdminHandlers, mfa.RequireStepUp(app.MFAService))
	featureflags.RegisterRoutes(r, app.FeatureFlagHandlers, mfa.RequireStepUp(app.MFAService))
	mfa.RegisterRoutes(r, app.MFAHandlers)

	// G4.C public capabilities honesty (run mode / bus claim).
	r.Get("/v1/health/capabilities", func(w http.ResponseWriter, r *http.Request) {
		mode := bootstrap.NormalizeRunMode(cfg.RunMode)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"run_mode":           mode,
			"api":                cfg.RunsAPI(),
			"workers_on_process": cfg.RunsWorkers(),
			"outbox_relay":       cfg.RunsWorkers(),
			"kafka_consumers":    cfg.RunsWorkers(),
			"full_bus":           mode == bootstrap.RunModeAll,
			"note":               "api-only does not run outbox relay; deploy PEGASUSX_RUN_MODE=worker for full bus",
		})
	})

	pulseroutes.RegisterRoutes(r, pulseroutes.Deps{
		Handlers:        app.PulseHandlers,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	retailerroutes.RegisterRoutes(r, retailerroutes.Deps{
		Service:          app.RetailerService,
		SupplierService:  app.SupplierService,
		PaymentService:   app.PaymentService,
		PromotionService: app.PromotionService,
		OrderService:     app.OrderService,
		JWTSecret:        cfg.JWTSecret,
		JWTIssuer:        cfg.JWTIssuer,
		AllowAuthBypass:  cfg.AllowAuthBypass,
		Spanner:          app.Spanner,
	})
	driverroutes.RegisterRoutes(r, driverroutes.Deps{
		Service:         app.DriverService,
		WarehouseSvc:    app.WarehouseService,
		OrderService:    app.OrderService,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	factoryroutes.RegisterRoutes(r, factoryroutes.Deps{
		Service:   app.FactoryService,
		JWTSecret: cfg.JWTSecret,
		JWTIssuer: cfg.JWTIssuer,
		Spanner:   app.Spanner,
	})
	payloadroutes.RegisterRoutes(r, payloadroutes.Deps{
		Service:      app.PayloadService,
		OrderService: app.OrderService,
		JWTSecret:    cfg.JWTSecret,
	})
	wmsHandler := &stocklots.Handler{
		Spanner:     app.Spanner,
		SoDCheck:    bootstrap.CheckStockAdjustmentSoD(app.IAM),
		AuditRecord: bootstrap.NewStocklotsAuditRecorder(app.Audit),
	}
	warehouseroutes.RegisterRoutes(r, warehouseroutes.Deps{
		Service:        app.WarehouseService,
		DriverService:  app.DriverService,
		OrderService:   app.OrderService,
		PayloadService: app.PayloadService,
		WMSHandler:     wmsHandler,
		JWTSecret:      cfg.JWTSecret,
		JWTIssuer:      cfg.JWTIssuer,
		Spanner:        app.Spanner,
	})
	returnsDeps := returnsroutes.Deps{
		Service: app.ReturnsService,
	}
	returnsroutes.RegisterRoutes(r, returnsDeps)
	returnsroutes.RegisterDriverRoutes(r, returnsDeps)
	returnsroutes.RegisterSupplierHistory(r, returnsDeps)
	storageroutes.Mount(r, app.EvidenceVault)
	taxroutes.Mount(r, app.TaxService)
	var partnerOnecPostings partner.OnecPostingRepository
	if app.PartnerService != nil {
		partnerOnecPostings = app.PartnerService.OnecPostings()
	}
	supplierroutes.RegisterRoutes(r, supplierroutes.Deps{
		Service:         app.SupplierService,
		RetailerService: app.RetailerService,
		StaffInvite: &staffinvite.Handler{
			Secret:         cfg.JWTSecret,
			SeedSupplierID: app.Supplier.SupplierID,
			NodeOwned:      staffinvite.SpannerNodeOwned(app.Spanner),
		},
		OrderService:      app.OrderService,
		PayloadService:    app.PayloadService,
		ComplianceHandler: compliance.NewHandler(app.ComplianceService),
		ExceptionResolve: supplier.ExceptionResolveDeps{
			CashRecon:    app.CashReconService,
			CreditNote:   app.CreditNoteService,
			Credit:       app.CreditService,
			Spanner:      app.Spanner,
			OnecPostings: partnerOnecPostings,
		},
		JWTSecret:    cfg.JWTSecret,
		Spanner:      app.Spanner,
		SupplierHub:  supplierHub,
		WarehouseHub: warehouseHub,
		TelemetryHub: telemetryHub,
		OrgOIDC:      app.OrgOIDC,
	})
	var (
		migrationRepo     migration.Repository
		migrationLive     migration.DryRunLiveOverlayReader
		migrationStore    migration.EntityApplyStore
		migrationReleaser migration.OpenOrderReleaser
	)
	if app.Spanner != nil {
		sr := migration.NewSpannerRepository(app.Spanner)
		migrationRepo = sr
		migrationLive = sr
		migrationStore = migration.NewSpannerApplyStore(app.Spanner)
		migrationReleaser = migration.SpannerOpenOrderReleaser{Client: app.Spanner}
	}
	migrationroutes.RegisterRoutes(r, migrationroutes.Deps{
		Handlers: migration.NewHandlers(migration.HandlerDeps{
			Repo:              migrationRepo,
			AllowAuthBypass:   cfg.AllowAuthBypass,
			JWTSecret:         cfg.JWTSecret,
			LiveReader:        migrationLive,
			ApplyStore:        migrationStore,
			OpenOrderReleaser: migrationReleaser,
		}),
		JWTSecret:       cfg.JWTSecret,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	entityresolutionroutes.RegisterRoutes(r, entityresolutionroutes.Deps{
		Spanner:         app.Spanner,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	countrycfg.RegisterRoutes(r, countrycfg.Deps{
		Spanner:         app.Spanner,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	controltowerroutes.RegisterRoutes(r, controltowerroutes.Deps{
		Handlers:        app.ControlTowerHandlers,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	promotionroutes.RegisterRoutes(r, promotionroutes.Deps{
		Service:   app.PromotionService,
		JWTSecret: cfg.JWTSecret,
	})
	paymentroutes.RegisterRoutes(r, paymentroutes.Deps{
		Service:         app.PaymentService,
		JWTSecret:       cfg.JWTSecret,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	webhookroutes.RegisterRoutes(r, webhookroutes.Deps{Service: app.PaymentService})
	orderroutes.RegisterRoutes(r, orderroutes.Deps{
		Service:           app.OrderService,
		ClaimsService:     app.ClaimsService,
		TaxService:        app.TaxService,
		ComplianceHandler: compliance.NewHandler(app.ComplianceService),
		AllowAuthBypass:   cfg.AllowAuthBypass,
	})
	creditroutes.RegisterRoutes(r, creditroutes.Deps{
		Service:             app.CreditService,
		PolicyService:       app.CreditPolicyService,
		ARService:           app.ARService,
		DunningWorker:       app.ARDunningWorker,
		PromiseBreachWorker: app.ARPromiseBreachWorker,
		StepUp:              mfa.RequireStepUp(app.MFAService),
	})
	if app.ForecastAccuracy != nil || app.ForecastRunner != nil || app.Spanner != nil {
		auth.ProtectMutations(r, auth.MutationGuardConfig{}, func(gr chi.Router) {
			if app.ForecastAccuracy != nil {
				gr.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin)).Post(
					"/v1/admin/planning/accuracy/run-once",
					app.ForecastAccuracy.HandleRunAccuracyOnce,
				)
			}
			if app.ForecastRunner != nil {
				gr.With(auth.RequireRole(auth.RoleAdmin)).Post(
					"/v1/admin/planning/forecast/run-once",
					app.ForecastRunner.HandleRunForecastOnce,
				)
			}
			if app.Spanner != nil {
				replayAPI := &replenishment.FillRateReplayAPI{Client: app.Spanner}
				gr.With(auth.RequireRole(auth.RoleAdmin)).Post(
					"/v1/admin/planning/safety-stock/replay",
					replayAPI.HandleReplay,
				)
			}
		})
	}
	cashreconroutes.RegisterRoutes(r, cashreconroutes.Deps{
		Handlers:        app.CashReconHandlers,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	foundryroutes.RegisterRoutes(r, foundryroutes.Deps{
		Engine: app.FoundryEngine,
	})
	creditnoteroutes.RegisterRoutes(r, creditnoteroutes.Deps{
		Handlers:        app.CreditNoteHandlers,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	glroutes.RegisterRoutes(r, glroutes.Deps{
		Service:         app.GL,
		JWTSecret:       cfg.JWTSecret,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	periodroutes.RegisterRoutes(r, periodroutes.Deps{
		Service:         app.Periods,
		JWTSecret:       cfg.JWTSecret,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	approvalroutes.RegisterRoutes(r, approvalroutes.Deps{
		Service:         app.Approvals,
		JWTSecret:       cfg.JWTSecret,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	valuationroutes.RegisterRoutes(r, valuationroutes.Deps{
		Service:         app.Valuation,
		JWTSecret:       cfg.JWTSecret,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	esfroutes.RegisterRoutes(r, esfroutes.Deps{
		Service:         app.ESF,
		JWTSecret:       cfg.JWTSecret,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	reconroutes.RegisterRoutes(r, reconroutes.Deps{
		Service:         app.Recon,
		JWTSecret:       cfg.JWTSecret,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	auditroutes.RegisterRoutes(r, auditroutes.Deps{
		Service:         app.Audit,
		JWTSecret:       cfg.JWTSecret,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	iamroutes.RegisterRoutes(r, iamroutes.Deps{
		Service:          app.IAM,
		JWTSecret:        cfg.JWTSecret,
		JWTIssuer:        cfg.JWTIssuer,
		SCIMBearerToken:  os.Getenv("SCIM_BEARER_TOKEN"),
		SCIMTenantID:     os.Getenv("SCIM_TENANT_ID"),
		SCIMTenantTokens: iamroutes.ParseSCIMTenantTokens(os.Getenv("SCIM_TENANT_TOKENS")),
		AllowAuthBypass:  cfg.AllowAuthBypass,
	})
	ontologyroutes.RegisterRoutes(r, ontologyroutes.Deps{
		Service:         app.Ontology,
		JWTSecret:       cfg.JWTSecret,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	forecastroutes.RegisterRoutes(r, forecastroutes.Deps{
		Service:          app.ForecastV2,
		ReplenishmentAPI: app.ReplenishmentSuggestions,
		JWTSecret:        cfg.JWTSecret,
		AllowAuthBypass:  cfg.AllowAuthBypass,
	})
	proposalroutes.RegisterRoutes(r, proposalroutes.Deps{
		Service:         app.Proposals,
		JWTSecret:       cfg.JWTSecret,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	roiroutes.RegisterRoutes(r, roiroutes.Deps{
		Service:         app.ROI,
		JWTSecret:       cfg.JWTSecret,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	connectorroutes.RegisterRoutes(r, connectorroutes.Deps{
		Service:         app.Connectors,
		JWTSecret:       cfg.JWTSecret,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	exportpackroutes.RegisterRoutes(r, exportpackroutes.Deps{
		Service:         app.ExportPacks,
		JWTSecret:       cfg.JWTSecret,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	multientityroutes.RegisterRoutes(r, multientityroutes.Deps{
		Service:         app.MultiEntity,
		JWTSecret:       cfg.JWTSecret,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	p2proutes.RegisterRoutes(r, p2proutes.Deps{
		Service:         app.P2P,
		JWTSecret:       cfg.JWTSecret,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	anomalyroutes.RegisterRoutes(r, anomalyroutes.Deps{
		Service:         app.Anomalies,
		JWTSecret:       cfg.JWTSecret,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	whatifroutes.RegisterRoutes(r, whatifroutes.Deps{
		Service:         app.WhatIf,
		JWTSecret:       cfg.JWTSecret,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	stocklots.RegisterHACCPComplianceRoute(r, wmsHandler, cfg.JWTSecret, cfg.AllowAuthBypass)
	deliveryroutes.RegisterRoutes(r, deliveryroutes.Deps{
		Service: app.OrderService,
	})
	telemetryroutes.RegisterRoutes(r, telemetryroutes.Deps{
		TelemetryHub:       telemetryHub,
		RetailerHub:        retailerHub,
		LastLocations:      app.DriverLocations,
		DeliveryTokens:     app.OrderService,
		ReturnApproach:     app.ReturnsService,
		SupplierID:         app.Supplier.SupplierID,
		Log:                log,
		LocationBusEmitter: locationBusEmitter(app),
	})

	updatesBase := strings.TrimSpace(cfg.UpdatesBaseURL)
	if updatesBase == "" {
		updatesBase = "http://localhost:" + cfg.HTTPPort
		log.Warn("UPDATES_BASE_URL unset; using local HTTP origin for updater manifests", "base_url", updatesBase)
	}
	updateroutes.RegisterRoutes(r, updateroutes.Deps{
		BaseURL:        updatesBase,
		DefaultVersion: cfg.UpdatesDefaultVersion,
	})

	demandroutes.RegisterRoutes(r, demandroutes.Deps{
		Service: app.DemandService,
	})

	syncCache := p.SyncCache
	if syncCache == nil && app.Cache != nil {
		syncCache = app.Cache.Backend()
	}
	syncroutes.RegisterRoutes(r, syncroutes.Deps{
		Engine: &syncroutes.SemanticEngine{
			Spanner: app.Spanner,
		},
		Cache: syncCache,
	})

	laborcapacityroutes.RegisterRoutes(r, laborcapacityroutes.Deps{
		Service: app.LaborCapacityService,
	})

	etaroutes.RegisterRoutes(r, etaroutes.Deps{
		Service: app.ETAService,
	})

	catalogroutes.RegisterRoutes(r, catalogroutes.Deps{
		Service:         app.CatalogService,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	globalproductsroutes.RegisterRoutes(r, globalproductsroutes.Deps{
		Service:         app.GlobalProductsService,
		AllowAuthBypass: cfg.AllowAuthBypass,
		StepUp:          mfa.RequireStepUp(app.MFAService),
	})
	if app.PartnerHandlers != nil && app.PartnerKeys != nil {
		partner.RegisterPartnerRoutesOpts(r, partner.AuthOptions{
			Keys: app.PartnerKeys, JWTSecret: app.PartnerJWTSecret,
		}, app.PartnerHandlers)
		partner.RegisterAdminKeyRoutes(r, app.PartnerHandlers, mfa.RequireStepUp(app.MFAService))
	}
	if app.FxRatesHandlers != nil {
		fxrates.RegisterAdminRoutes(r, app.FxRatesHandlers)
		fxrates.RegisterSupplierRoutes(r, app.FxRatesHandlers)
	}
	if app.PayoutService != nil {
		payout.RegisterRoutes(r, &payout.Handlers{Svc: app.PayoutService})
	}
	planning.RegisterAccuracyRoutes(r, app.ForecastAccuracy)
	if app.BillingInvoiceWorker != nil {
		billing.RegisterRoutes(r, &billing.Handlers{Worker: app.BillingInvoiceWorker}, mfa.RequireStepUp(app.MFAService))
	}
	ws.RegisterRoutes(r, log, cfg.JWTSecret,
		app.PlatformService,
		retailerHub, supplierHub, driverHub, payloadHub, warehouseHub, factoryHub, telemetryHub, platformAdminHub,
		ws.RegisterConfig{
			RetailerPromoSuppliers: func(ctx context.Context, retailerID string) []string {
				if app.PromotionAudience == nil {
					return nil
				}
				ids, err := app.PromotionAudience.CartSupplierIDs(ctx, retailerID)
				if err != nil {
					log.WarnContext(ctx, "retailer promo cart suppliers failed", "err", err, "retailer_id", retailerID)
					return nil
				}
				return ids
			},
		},
	)
	if app.NotificationInbox != nil {
		inbox := app.NotificationInbox
		r.Get("/v1/user/notifications", inbox.HandleList)
		r.Post("/v1/user/notifications/read", inbox.HandleMarkRead)
	}
	if app.NotificationPreferences != nil {
		prefs := app.NotificationPreferences
		r.Get("/v1/user/notification-preferences", prefs.HandleGetPreferences)
		r.Patch("/v1/user/notification-preferences", prefs.HandlePatchPreferences)
	}
	if app.AnalyticsHandlers != nil {
		h := app.AnalyticsHandlers
		r.Get("/v1/supplier/route-performance", h.HandleListRoutePerformance)
	}

	gpEnv := strings.ToLower(strings.TrimSpace(cfg.GlobalPayEnv))
	if cfg.EnablePaymentSimulator && gpEnv != "production" {
		simHandler := simulator.NewHandler(
			cfg.GlobalPayWebhookSecret,
			"http://localhost:"+cfg.HTTPPort,
			log,
		)
		r.Route("/sim/globalpay", func(sr chi.Router) {
			simulator.RegisterRoutes(sr, simHandler)
		})
		log.Info("[simulator] Global Pay simulator mounted", "prefix", "/sim/globalpay", "env", gpEnv)
	}

	return r
}

func coalesceHub(primary, fallback *ws.Hub) *ws.Hub {
	if primary != nil {
		return primary
	}
	return fallback
}
