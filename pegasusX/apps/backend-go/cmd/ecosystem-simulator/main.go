package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", "err", err)
		os.Exit(1)
	}

	base := os.Getenv("API_BASE_URL")
	if base == "" {
		port := cfg.HTTPPort
		if port == "" {
			port = "8080"
		}
		base = "http://localhost:" + port
	}
	base = strings.TrimRight(base, "/")

	sim := NewSimulator(cfg, base)
	slog.Info("Starting Ecosystem E2E Simulation", "base_url", base)

	if err := sim.Run(ctx); err != nil {
		slog.Error("Simulation failed", "err", err)
		os.Exit(1)
	}

	slog.Info("Ecosystem E2E Simulation completed perfectly across all roles!")
}
