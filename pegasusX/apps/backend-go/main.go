// Package main is the pegasusX backend lifecycle entrypoint.
//
// main.go is operational only: load config → bootstrap.NewApp → register routes
// → http.Server.ListenAndServe → graceful shutdown. Target ceiling: 200 lines.
// Domain logic lives in domain packages; URL mounts live in *routes packages.
package main

import (
	"context"
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
	"github.com/pegasusx/pegasusx/apps/backend-go/catalogroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/cashreconroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/creditnoteroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/compliance"
	"github.com/pegasusx/pegasusx/apps/backend-go/controltowerroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/creditroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/deliveryroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/demandroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/driverroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/enterprise"
	"github.com/pegasusx/pegasusx/apps/backend-go/etaroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/factoryroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
	"github.com/pegasusx/pegasusx/apps/backend-go/globalproductsroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/services/billing"
	"github.com/pegasusx/pegasusx/apps/backend-go/payout"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
	"github.com/pegasusx/pegasusx/apps/backend-go/geolocation"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/infraroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/laborcapacityroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/orderroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner"
	"github.com/pegasusx/pegasusx/apps/backend-go/payloaderroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/paymentroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/platformroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/platformadmin"
	"github.com/pegasusx/pegasusx/apps/backend-go/featureflags"
	"github.com/pegasusx/pegasusx/apps/backend-go/promotionroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/pulseroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/replenishment"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailerroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/returnsroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/simulator"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplierroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetryroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/updateroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouseroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
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

	// Phase 1/2 Integration: Auth0 Identity Middleware
	if os.Getenv("AUTH0_DOMAIN") != "" {
		auth0Middleware := enterprise.SetupAuth0Middleware()
		r.Use(auth0Middleware)
		slog.Info("Auth0 Enterprise Middleware attached to router")
	}

	if app.Reliability != nil {
		r.Use(app.Reliability.Middleware)
	}
	if app.Idempotency != nil {
		r.Use(idempotency.Middleware(app.Idempotency))
	}
	var firebaseVerifier auth.FirebaseVerifier
	if cfg.FirebaseAuthEnabled {
		if cfg.FirebaseProjectID == "" {
			slog.Warn("firebase auth enabled but FIREBASE_PROJECT_ID is empty")
		} else {
			firebaseVerifier = auth.NewFirebaseTokenVerifier(
				cfg.FirebaseProjectID,
				auth.FirebaseVerifierOptionsForProject(cfg.FirebaseCertsURL),
			)
			slog.Info("firebase auth verifier initialized", "project_id", cfg.FirebaseProjectID)
		}
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
		JWTSecret:      cfg.JWTSecret,
		JWTIssuer:      cfg.JWTIssuer,
	})
	platformadmin.RegisterRoutes(r, app.PlatformAdminHandlers)
	featureflags.RegisterRoutes(r, app.FeatureFlagHandlers)
	pulseroutes.RegisterRoutes(r, pulseroutes.Deps{
		Handlers:            app.PulseHandlers,
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
		AllowAuthBypass:     cfg.AllowAuthBypass,
	})
	retailerroutes.RegisterRoutes(r, retailerroutes.Deps{
		Service:             app.RetailerService,
		PaymentService:      app.PaymentService,
		PromotionService:    app.PromotionService,
		OrderService:        app.OrderService,
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
		AllowAuthBypass:     cfg.AllowAuthBypass,
	})
	driverroutes.RegisterRoutes(r, driverroutes.Deps{
		Service:             app.DriverService,
		WarehouseSvc:        app.WarehouseService,
		OrderService:        app.OrderService,
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
		AllowAuthBypass:     cfg.AllowAuthBypass,
	})
	factoryroutes.RegisterRoutes(r, factoryroutes.Deps{
		Service:             app.FactoryService,
		JWTSecret:           cfg.JWTSecret,
		Spanner:             app.Spanner,
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
	})
	payloaderroutes.RegisterRoutes(r, payloaderroutes.Deps{
		Service:             app.PayloadService,
		OrderService:        app.OrderService,
		JWTSecret:           cfg.JWTSecret,
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
	})
	warehouseroutes.RegisterRoutes(r, warehouseroutes.Deps{
		Service:             app.WarehouseService,
		OrderService:        app.OrderService,
		PayloadService:      app.PayloadService,
		WMSHandler: &stocklots.Handler{
			Spanner: app.Spanner,
		},
		JWTSecret:           cfg.JWTSecret,
		Spanner:             app.Spanner,
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
	})
	returnsDeps := returnsroutes.Deps{
		Service:             app.ReturnsService,
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
	}
	returnsroutes.RegisterRoutes(r, returnsDeps)
	returnsroutes.RegisterDriverRoutes(r, returnsDeps)
	returnsroutes.RegisterSupplierHistory(r, returnsDeps)
	supplierroutes.RegisterRoutes(r, supplierroutes.Deps{
		Service:           app.SupplierService,
		OrderService:      app.OrderService,
		PayloadService:    app.PayloadService,
		NotificationInbox: app.NotificationInbox,
		ComplianceHandler: compliance.NewHandler(app.ComplianceService),
		ExceptionResolve: supplier.ExceptionResolveDeps{
			CashRecon:  app.CashReconService,
			CreditNote: app.CreditNoteService,
			Credit:     app.CreditService,
		},
		JWTSecret:         cfg.JWTSecret,
		Spanner:           app.Spanner,
		SupplierHub:       app.SupplierHub,
		WarehouseHub:      app.WarehouseHub,
	})
	controltowerroutes.RegisterRoutes(r, controltowerroutes.Deps{
		Handlers:            app.ControlTowerHandlers,
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
		AllowAuthBypass:     cfg.AllowAuthBypass,
	})
	promotionroutes.RegisterRoutes(r, promotionroutes.Deps{
		Service:   app.PromotionService,
		JWTSecret: cfg.JWTSecret,
	})
	paymentroutes.RegisterRoutes(r, paymentroutes.Deps{
		Service:             app.PaymentService,
		JWTSecret:           cfg.JWTSecret,
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
		AllowAuthBypass:     cfg.AllowAuthBypass,
	})
	webhookroutes.RegisterRoutes(r, webhookroutes.Deps{Service: app.PaymentService})
	orderroutes.RegisterRoutes(r, orderroutes.Deps{
		Service:             app.OrderService,
		ClaimsService:       app.ClaimsService,
		TaxService:          app.TaxService,
		ComplianceHandler:   compliance.NewHandler(app.ComplianceService),
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
		AllowAuthBypass:     cfg.AllowAuthBypass,
	})
	creditroutes.RegisterRoutes(r, creditroutes.Deps{
		Service:       app.CreditService,
		PolicyService: app.CreditPolicyService,
		ARService:     app.ARService,
		DunningWorker: app.ARDunningWorker,
	})
	if app.ForecastAccuracy != nil || app.ForecastRunner != nil || app.Spanner != nil {
		auth.ProtectMutations(r, auth.MutationGuardConfig{}, func(gr chi.Router) {
			if app.ForecastAccuracy != nil {
				gr.With(auth.RequireRole(auth.RoleAdmin)).Post(
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
		Handlers:            app.CashReconHandlers,
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
		AllowAuthBypass:     cfg.AllowAuthBypass,
	})
	creditnoteroutes.RegisterRoutes(r, creditnoteroutes.Deps{
		Handlers:            app.CreditNoteHandlers,
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
		AllowAuthBypass:     cfg.AllowAuthBypass,
	})
	deliveryroutes.RegisterRoutes(r, deliveryroutes.Deps{
		Service:             app.OrderService,
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
		AllowAuthBypass:     cfg.AllowAuthBypass,
	})
	telemetryroutes.RegisterRoutes(r, telemetryroutes.Deps{
		TelemetryHub:        app.TelemetryHub,
		RetailerHub:         app.RetailerHub,
		LastLocations:       app.DriverLocations,
		DeliveryTokens:      app.OrderService,
		ReturnApproach:      app.ReturnsService,
		SupplierID:          app.Supplier.SupplierID,
		Log:                 slog.Default(),
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
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
		Service:             app.CatalogService,
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
		AllowAuthBypass:     cfg.AllowAuthBypass,
	})
	globalproductsroutes.RegisterRoutes(r, globalproductsroutes.Deps{
		Service:             app.GlobalProductsService,
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
		AllowAuthBypass:     cfg.AllowAuthBypass,
	})
	if app.PartnerHandlers != nil && app.PartnerKeys != nil {
		partner.RegisterPartnerRoutesOpts(r, partner.AuthOptions{
			Keys: app.PartnerKeys, JWTSecret: app.PartnerJWTSecret,
		}, app.PartnerHandlers)
		partner.RegisterAdminKeyRoutes(r, app.PartnerHandlers)
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
		billing.RegisterRoutes(r, &billing.Handlers{Worker: app.BillingInvoiceWorker})
	}
	ws.RegisterRoutes(r, slog.Default(), cfg.JWTSecret, cfg.FirebaseAuthEnabled, firebaseVerifier,
		app.PlatformService,
		app.RetailerHub, app.SupplierHub, app.DriverHub, app.PayloadHub, app.WarehouseHub, app.FactoryHub, app.TelemetryHub,
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
	// Role-row *routes packages each mount /v1/user/notifications with role-specific
	// middleware; chi keeps only the last registration. Mount once here so every
	// authenticated role (supplier cookie, retailer/driver Bearer) resolves via
	// RecipientIDFromClaims after Kafka inbox persistence.
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
