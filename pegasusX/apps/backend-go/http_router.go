package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/cashreconroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/catalogroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/compliance"
	"github.com/pegasusx/pegasusx/apps/backend-go/controltowerroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/countrycfg"
	"github.com/pegasusx/pegasusx/apps/backend-go/creditnoteroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/creditroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/deliveryroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/demandroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/driverroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/entityresolutionroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/etaroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/factoryroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/featureflags"
	"github.com/pegasusx/pegasusx/apps/backend-go/foundryroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
	"github.com/pegasusx/pegasusx/apps/backend-go/geolocation"
	"github.com/pegasusx/pegasusx/apps/backend-go/globalproductsroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/infraroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/services/billing"
	"github.com/pegasusx/pegasusx/apps/backend-go/laborcapacityroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/mfa"
	"github.com/pegasusx/pegasusx/apps/backend-go/orderroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner"
	payloadroutes "github.com/pegasusx/pegasusx/apps/backend-go/payloaderoutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/paymentroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/payout"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
	"github.com/pegasusx/pegasusx/apps/backend-go/platformadmin"
	"github.com/pegasusx/pegasusx/apps/backend-go/platformroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/promotionroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/pulseroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/replenishment"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailerroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/returnsroutes"
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
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouseroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/webhookroutes"
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
	r.Use(bootstrap.MaxBytesMiddleware(0))
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
	if app.Idempotency != nil {
		r.Use(app.IdempotencyMiddleware())
	}
	infraroutes.RegisterRoutes(r, app.InfraHealth)

	geocodeSvc := geolocation.NewService(cfg.GoogleMapsAPIKey, app.Cache)
	platformroutes.RegisterRoutes(r, platformroutes.Deps{
		Handler:        app.PlatformHandler,
		GeocodeHandler: geolocation.NewHandler(geocodeSvc),
		TenantRegister: app.TenantRegister,
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
	warehouseroutes.RegisterRoutes(r, warehouseroutes.Deps{
		Service:        app.WarehouseService,
		DriverService:  app.DriverService,
		OrderService:   app.OrderService,
		PayloadService: app.PayloadService,
		WMSHandler: &stocklots.Handler{
			Spanner: app.Spanner,
		},
		JWTSecret: cfg.JWTSecret,
		JWTIssuer: cfg.JWTIssuer,
		Spanner:   app.Spanner,
	})
	returnsDeps := returnsroutes.Deps{
		Service: app.ReturnsService,
	}
	returnsroutes.RegisterRoutes(r, returnsDeps)
	returnsroutes.RegisterDriverRoutes(r, returnsDeps)
	returnsroutes.RegisterSupplierHistory(r, returnsDeps)
	storageroutes.Mount(r, app.EvidenceVault)
	taxroutes.Mount(r, app.TaxService)
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
			CashRecon:  app.CashReconService,
			CreditNote: app.CreditNoteService,
			Credit:     app.CreditService,
		},
		JWTSecret:    cfg.JWTSecret,
		Spanner:      app.Spanner,
		SupplierHub:  supplierHub,
		WarehouseHub: warehouseHub,
		TelemetryHub: telemetryHub,
		OrgOIDC:      app.OrgOIDC,
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
		Service:       app.CreditService,
		PolicyService: app.CreditPolicyService,
		ARService:     app.ARService,
		DunningWorker: app.ARDunningWorker,
		StepUp:        mfa.RequireStepUp(app.MFAService),
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
