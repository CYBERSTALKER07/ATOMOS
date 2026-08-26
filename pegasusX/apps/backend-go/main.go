// Package main is the pegasusX backend lifecycle entrypoint.
//
// main.go is operational only: load config → bootstrap.NewApp → register routes
// → http.Server.ListenAndServe → graceful shutdown. Target ceiling: 200 lines.
// Domain logic lives in domain packages; URL mounts live in *routes packages.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
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
	"github.com/pegasusx/pegasusx/apps/backend-go/enterprise"
	"github.com/pegasusx/pegasusx/apps/backend-go/entityresolutionroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/etaroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/factoryroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
	"github.com/pegasusx/pegasusx/apps/backend-go/geolocation"
	"github.com/pegasusx/pegasusx/apps/backend-go/globalproductsroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/infraroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/services/billing"
	"github.com/pegasusx/pegasusx/apps/backend-go/laborcapacityroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/orderroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner"
	"github.com/pegasusx/pegasusx/apps/backend-go/payout"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
	// B2: richer payload routes (ws-session, ship-units, labels).
	"github.com/pegasusx/pegasusx/apps/backend-go/featureflags"
	"github.com/pegasusx/pegasusx/apps/backend-go/mfa"
	payloadroutes "github.com/pegasusx/pegasusx/apps/backend-go/payloaderoutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/paymentroutes"
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
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplierroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/storageroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/taxroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetryroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/updateroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouseroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/webhookroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	// Phase 1/2 Integration: Initialize Datadog APM and Profiler
	enterprise.InitDatadog("pegasusX-backend", "1.0.0")
	defer enterprise.StopDatadog()

	// Phase 1/2 Integration: Initialize HashiCorp HCP Vault Secrets
	if err := enterprise.InitVault(); err != nil {
		slog.Warn("HashiCorp Vault init failed (expected if not configured or running local trial)", "err", err)
	}

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	app, err := bootstrap.NewApp(ctx, cfg)
	if err != nil {
		slog.Error("app bootstrap failed", "err", err)
		os.Exit(1)
	}
	defer app.Close()

	runMode := bootstrap.NormalizeRunMode(cfg.RunMode)
	slog.Info("pegasusX runtime profile", "run_mode", runMode)

	if cfg.RunsWorkers() {
		startBackgroundWorkers(ctx, app)
	}
	if cfg.RunsAPI() {
		startHubRelaySubscribers(ctx, hubList(app))
		// P1-9 run-mode parity: in api-only mode the notification consumer (FCM
		// push + inbox persistence) normally lives on the worker tier. When no
		// worker is live (local/dev or a misconfigured deploy), start it here so
		// push/inbox are not silently lost. The liveness gate prevents double
		// delivery when a worker tier is present.
		startNotificationConsumerIfNoWorker(ctx, app)
	}

	if !cfg.RunsAPI() {
		_ = startWorkerHealthServer(ctx, cfg, app)
		<-ctx.Done()
		slog.Info("shutdown signal received")
		return
	}

	r := chi.NewRouter()
	r.Use(bootstrap.TraceMiddleware)
	r.Use(telemetry.HTTPMetricsMiddleware)
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
		slog.Warn("AUTH0_DOMAIN is set but ignored; GS-I does not wrap the router")
	}

	if app.Reliability != nil {
		r.Use(app.Reliability.Middleware)
	}
	if app.Idempotency != nil {
		r.Use(idempotency.Middleware(app.Idempotency))
	}
	infraroutes.RegisterRoutes(r, app.InfraHealth)
	// SLO metrics (void_outbox_lag_seconds, void_fiscal_success_ratio,
	// void_capture_success_ratio) polled from Spanner into the default
	// Prometheus registry; scraped at /metrics (mounted by infraroutes).
	if app.Spanner != nil {
		slo := telemetry.NewSLOCollector(app.Spanner, nil, slog.Default())
		go slo.Start(ctx, 60*time.Second)
	}
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
		SupplierHub:  app.SupplierHub,
		WarehouseHub: app.WarehouseHub,
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
	creditnoteroutes.RegisterRoutes(r, creditnoteroutes.Deps{
		Handlers:        app.CreditNoteHandlers,
		AllowAuthBypass: cfg.AllowAuthBypass,
	})
	deliveryroutes.RegisterRoutes(r, deliveryroutes.Deps{
		Service: app.OrderService,
	})
	telemetryroutes.RegisterRoutes(r, telemetryroutes.Deps{
		TelemetryHub:   app.TelemetryHub,
		RetailerHub:    app.RetailerHub,
		LastLocations:  app.DriverLocations,
		DeliveryTokens: app.OrderService,
		ReturnApproach: app.ReturnsService,
		SupplierID:     app.Supplier.SupplierID,
		Log:            slog.Default(),
		// P1-10: throttled copy of driver location onto the outbox/Kafka bus so the
		// notification dispatcher + twin consumers are live. Full fidelity stays on
		// WS/Redis; only the bus copy is throttled.
		LocationBusEmitter: locationBusEmitter(app),
	})

	updatesBase := strings.TrimSpace(cfg.UpdatesBaseURL)
	if updatesBase == "" {
		// Non-production default for local OTA smoke only; production rejects empty via ValidateProductionProfile.
		updatesBase = "http://localhost:" + cfg.HTTPPort
		slog.Warn("UPDATES_BASE_URL unset; using local HTTP origin for updater manifests", "base_url", updatesBase)
	}
	updateroutes.RegisterRoutes(r, updateroutes.Deps{
		BaseURL:        updatesBase,
		DefaultVersion: cfg.UpdatesDefaultVersion,
	})

	demandroutes.RegisterRoutes(r, demandroutes.Deps{
		Service: app.DemandService,
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
		// B5 M-P1-11: MFA step-up for PLATFORM_ADMIN partner key issue/list/revoke.
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
	ws.RegisterRoutes(r, slog.Default(), cfg.JWTSecret,
		app.PlatformService,
		app.RetailerHub, app.SupplierHub, app.DriverHub, app.PayloadHub, app.WarehouseHub, app.FactoryHub, app.TelemetryHub, app.PlatformAdminHub,
		ws.RegisterConfig{
			RetailerPromoSuppliers: func(ctx context.Context, retailerID string) []string {
				if app.PromotionAudience == nil {
					return nil
				}
				ids, err := app.PromotionAudience.CartSupplierIDs(ctx, retailerID)
				if err != nil {
					slog.WarnContext(ctx, "retailer promo cart suppliers failed", "err", err, "retailer_id", retailerID)
					return nil
				}
				return ids
			},
		},
	)
	// Live inbox for every authenticated role. Role-row packages must not remount
	// this path (chi last-wins). Fail-closed when NotificationInbox.Service is nil.
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

	// Global Pay local simulator — only mounted when GLOBAL_PAY_ENV != "production".
	// Provides a browser UI for end-to-end payment testing without hitting the real gateway.
	gpEnv := strings.ToLower(strings.TrimSpace(cfg.GlobalPayEnv))
	if gpEnv != "production" && gpEnv != "staging" {
		simHandler := simulator.NewHandler(
			cfg.GlobalPayWebhookSecret,
			"http://localhost:"+cfg.HTTPPort,
			slog.Default(),
		)
		r.Route("/sim/globalpay", func(sr chi.Router) {
			simulator.RegisterRoutes(sr, simHandler)
		})
		slog.Info("[simulator] Global Pay simulator mounted", "prefix", "/sim/globalpay", "env", gpEnv)
	}

	// Start background workers
	go app.OrderService.RunShopClosedWorker(ctx, 30*time.Second)
	go app.DemandService.RunDemandSensingWorker(ctx, 12*time.Hour)

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("pegasusX backend listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http serve failed", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("http shutdown failed", "err", err)
	}
}
