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
	"github.com/pegasusx/pegasusx/apps/backend-go/driverroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/factoryroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/infraroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/orderroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/payloaderroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/paymentroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailerroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplierroutes"
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

	infraroutes.RegisterRoutes(r, infraroutes.Deps{})
	retailerroutes.RegisterRoutes(r, retailerroutes.Deps{
		Service:             app.RetailerService,
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
	})
	driverroutes.RegisterRoutes(r, driverroutes.Deps{
		Service:             app.DriverService,
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
	})
	factoryroutes.RegisterRoutes(r, factoryroutes.Deps{
		Service:             app.FactoryService,
		JWTSecret:           cfg.JWTSecret,
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
		JWTSecret:           cfg.JWTSecret,
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
	})
	supplierroutes.RegisterRoutes(r, supplierroutes.Deps{Service: app.SupplierService, JWTSecret: cfg.JWTSecret})
	paymentroutes.RegisterRoutes(r, paymentroutes.Deps{
		Service:             app.PaymentService,
		JWTSecret:           cfg.JWTSecret,
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
	})
	webhookroutes.RegisterRoutes(r, webhookroutes.Deps{Service: app.PaymentService})
	orderroutes.RegisterRoutes(r, orderroutes.Deps{
		Service:             app.OrderService,
		FirebaseAuthEnabled: cfg.FirebaseAuthEnabled && firebaseVerifier != nil,
		FirebaseVerifier:    firebaseVerifier,
	})
	// TODO: register additional domain *routes packages here.

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
