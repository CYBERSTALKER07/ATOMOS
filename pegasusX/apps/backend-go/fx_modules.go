package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/enterprise"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

// ObservabilityModule manages Datadog APM/Profiler, HashiCorp Vault initialization,
// and Prometheus SLO background polling via fx.Lifecycle hooks.
var ObservabilityModule = fx.Module("observability",
	fx.Decorate(func(log *slog.Logger) *slog.Logger {
		return log.With("module", "observability")
	}),
	fx.Invoke(registerObservabilityHooks),
)

func registerObservabilityHooks(lc fx.Lifecycle, app *bootstrap.App, log *slog.Logger) {
	var sloCancel context.CancelFunc
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			enterprise.InitDatadog("pegasusX-backend", "1.0.0")
			if err := enterprise.InitVault(); err != nil {
				log.Warn("HashiCorp Vault init failed (expected if not configured or running local trial)", "err", err)
			}
			if app != nil && app.Spanner != nil {
				sloCtx, cancel := context.WithCancel(context.Background())
				sloCancel = cancel
				slo := telemetry.NewSLOCollector(app.Spanner, nil, log)
				go slo.Start(sloCtx, 60*time.Second)
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if sloCancel != nil {
				sloCancel()
			}
			enterprise.StopDatadog()
			return nil
		},
	})
}

// BootstrapModule provides *bootstrap.Config, *bootstrap.App, narrow interface
// projections (cache.Backend, idempotency.Store), and named/grouped *ws.Hub instances.
var BootstrapModule = fx.Module("bootstrap",
	fx.Provide(
		bootstrap.LoadConfig,
		provideBootstrapApp,
		bootstrap.ProvideAppSubgraph,
		fx.Annotate(
			provideCacheBackend,
			fx.As(new(cache.Backend)),
		),
	),
)

func provideBootstrapApp(lc fx.Lifecycle, cfg *bootstrap.Config, log *slog.Logger) (*bootstrap.App, error) {
	ctx := context.Background()
	app, err := bootstrap.NewApp(ctx, cfg)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			runMode := bootstrap.NormalizeRunMode(cfg.RunMode)
			log.Info("pegasusX runtime profile", "run_mode", runMode)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			app.Close()
			return nil
		},
	})
	return app, nil
}

func provideCacheBackend(app *bootstrap.App) cache.Backend {
	if app != nil && app.Cache != nil {
		return app.Cache.Backend()
	}
	return cache.NewInMemoryBackend()
}

// WorkerLifecycleParams uses fx.In and the "ws_hubs" value group to wire
// background workers and WebSocket Redis relay subscribers.
type WorkerLifecycleParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Config    *bootstrap.Config
	App       *bootstrap.App
	Logger    *slog.Logger
	Hubs      []*ws.Hub `group:"ws_hubs"`
}

// WorkersModule manages all background consumers, outbox relays, cron workers,
// and the worker-only health server using non-blocking fx.Lifecycle hooks.
var WorkersModule = fx.Module("workers",
	fx.Decorate(func(log *slog.Logger) *slog.Logger {
		return log.With("module", "workers")
	}),
	fx.Invoke(registerWorkerHooks),
)

func registerWorkerHooks(p WorkerLifecycleParams) {
	var workerCancel context.CancelFunc
	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			workerCtx, cancel := context.WithCancel(context.Background())
			workerCancel = cancel

			if p.Config.RunsWorkers() {
				startBackgroundWorkers(workerCtx, p.App)
			}
			if p.Config.RunsAPI() {
				hubs := p.Hubs
				if len(hubs) == 0 {
					hubs = hubList(p.App)
				}
				startHubRelaySubscribers(workerCtx, hubs)
				startNotificationConsumerIfNoWorker(workerCtx, p.App)
				if p.App.OrderService != nil {
					go p.App.OrderService.RunShopClosedWorker(workerCtx, 30*time.Second)
				}
				if p.App.DemandService != nil {
					go p.App.DemandService.RunDemandSensingWorker(workerCtx, 12*time.Hour)
				}
			} else {
				_ = startWorkerHealthServer(workerCtx, p.Config, p.App)
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if workerCancel != nil {
				workerCancel()
			}
			return nil
		},
	})
}

// HTTPServerParams groups dependencies for constructing the *http.Server.
type HTTPServerParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Config    *bootstrap.Config
	Router    chi.Router
	Logger    *slog.Logger
}

// HTTPModule provides the chi.Router and manages the HTTP server lifecycle.
var HTTPModule = fx.Module("http",
	fx.Decorate(func(log *slog.Logger) *slog.Logger {
		return log.With("module", "http")
	}),
	fx.Provide(
		NewHTTPRouter,
		NewHTTPServer,
	),
	fx.Invoke(func(*http.Server) {}),
)

// NewHTTPServer constructs the *http.Server and registers non-blocking OnStart
// and graceful OnStop hooks on the fx.Lifecycle.
func NewHTTPServer(p HTTPServerParams) *http.Server {
	srv := &http.Server{
		Addr:              ":" + p.Config.HTTPPort,
		Handler:           p.Router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if !p.Config.RunsAPI() {
				p.Logger.Info("HTTP API server skipped in worker-only mode", "run_mode", p.Config.RunMode)
				return nil
			}
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			p.Logger.Info("pegasusX backend listening", "addr", ln.Addr().String())
			go func() {
				if serveErr := srv.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
					p.Logger.Error("http serve failed", "err", serveErr)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if !p.Config.RunsAPI() {
				return nil
			}
			p.Logger.Info("shutting down HTTP server")
			return srv.Shutdown(ctx)
		},
	})

	return srv
}

// AppOptions returns the complete set of uber-go/fx options for the pegasusX backend.
func AppOptions(logger *slog.Logger) []fx.Option {
	if logger == nil {
		logger = slog.Default()
	}
	return []fx.Option{
		fx.Supply(logger),
		fx.WithLogger(func(log *slog.Logger) fxevent.Logger {
			return &fxevent.SlogLogger{Logger: log}
		}),
		fx.StartTimeout(30 * time.Second),
		fx.StopTimeout(20 * time.Second),
		ObservabilityModule,
		BootstrapModule,
		WorkersModule,
		HTTPModule,
	}
}
