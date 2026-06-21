package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/infraroutes"
)

func startWorkerHealthServer(ctx context.Context, cfg *bootstrap.Config, app *bootstrap.App) *http.Server {
	r := chi.NewRouter()
	infraroutes.RegisterRoutes(r, app.InfraHealth)

	srv := &http.Server{
		Addr:              ":" + cfg.WorkerHTTPPort,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("pegasusX worker health listening", "addr", srv.Addr, "run_mode", cfg.RunMode)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("worker health serve failed", "err", err)
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("worker health shutdown failed", "err", err)
		}
	}()

	return srv
}
