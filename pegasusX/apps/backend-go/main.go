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
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/catalogroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/driverroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/factoryroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/infraroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/orderroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/payloaderroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/paymentroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/platformroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailerroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplierroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetryroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouseroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/webhookroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

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
	if app.OutboxRelay != nil {
		go app.OutboxRelay.Start(ctx)
		slog.Info("outbox relay started")
	}
	if app.Cache != nil {
		go app.Cache.StartInvalidationSubscriber(ctx)
		slog.Info("cache invalidation subscriber started")
	}
	if app.NotificationConsumer != nil {
		go app.NotificationConsumer.Start(ctx)
		slog.Info("notification consumer started")
	}
	startHubRelaySubscribers(ctx, []*ws.Hub{
		app.RetailerHub,
		app.SupplierHub,
		app.DriverHub,
		app.PayloadHub,
		app.WarehouseHub,
		app.FactoryHub,
		app.TelemetryHub,
	})

	r := chi.NewRouter()
	r.Use(bootstrap.TraceMiddleware)
	r.Use(auth.SessionAuth(cfg.JWTSecret))
	if app.Reliability != nil {
		r.Use(app.Reliability.Middleware)
	}
	var firebaseVerifier auth.FirebaseVerifier
	if cfg.FirebaseAuthEnabled {
		if cfg.FirebaseProjectID == "" {
			slog.Warn("firebase auth enabled but FIREBASE_PROJECT_ID is empty")
		} else {
			firebaseVerifier = auth.NewFirebaseTokenVerifier(cfg.FirebaseProjectID, auth.FirebaseTokenVerifierOptions{
				CertsURL: cfg.FirebaseCertsURL,
			})
			slog.Info("firebase auth verifier initialized", "project_id", cfg.FirebaseProjectID)
		}
	}

	infraroutes.RegisterRoutes(r, app.InfraHealth)
	platformroutes.RegisterRoutes(r, platformroutes.Deps{
		Handler:   app.PlatformHandler,
		JWTSecret: cfg.JWTSecret,
		JWTIssuer: cfg.JWTIssuer,
	})
	retailerroutes.RegisterRoutes(r, retailerroutes.Deps{
		Service:             app.RetailerService,
		PaymentService:      app.PaymentService,
		OrderService:        app.OrderService,
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
		AllowAuthBypass:     cfg.AllowAuthBypass,
	})
	driverroutes.RegisterRoutes(r, driverroutes.Deps{
		Service:             app.DriverService,
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
		JWTSecret:           cfg.JWTSecret,
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
	})
	warehouseroutes.RegisterRoutes(r, warehouseroutes.Deps{
		Service:             app.WarehouseService,
		OrderService:        app.OrderService,
		JWTSecret:           cfg.JWTSecret,
		Spanner:             app.Spanner,
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
	})
	supplierroutes.RegisterRoutes(r, supplierroutes.Deps{
		Service:      app.SupplierService,
		OrderService: app.OrderService,
		JWTSecret:    cfg.JWTSecret,
		Spanner:      app.Spanner,
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
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
		AllowAuthBypass:     cfg.AllowAuthBypass,
	})
	telemetryroutes.RegisterRoutes(r, telemetryroutes.Deps{
		TelemetryHub:        app.TelemetryHub,
		LastLocations:       app.DriverLocations,
		SupplierID:          app.Supplier.SupplierID,
		Log:                 slog.Default(),
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
	})
	catalogroutes.RegisterRoutes(r, catalogroutes.Deps{
		Service:             app.CatalogService,
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
		AllowAuthBypass:     cfg.AllowAuthBypass,
	})
	ws.RegisterRoutes(r, slog.Default(), cfg.JWTSecret, cfg.FirebaseAuthEnabled, firebaseVerifier,
		app.PlatformService,
		app.RetailerHub, app.SupplierHub, app.DriverHub, app.PayloadHub, app.WarehouseHub, app.FactoryHub, app.TelemetryHub)

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

func startHubRelaySubscribers(ctx context.Context, hubs []*ws.Hub) {
	for _, hub := range hubs {
		if hub == nil {
			continue
		}
		go hub.StartRelaySubscriber(ctx)
	}
	slog.Info("ws relay subscribers started", "hub_count", len(hubs))
}
