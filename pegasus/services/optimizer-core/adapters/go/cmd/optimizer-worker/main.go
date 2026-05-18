package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/spanner"

	"optimizercoreadapter/internal/adapter"
	"optimizercoreadapter/internal/config"
	"optimizercoreadapter/internal/optimizergrpc"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	spannerClient, err := spanner.NewClient(ctx, cfg.SpannerDatabase)
	if err != nil {
		slog.Error("create spanner client", "err", err)
		os.Exit(1)
	}
	defer spannerClient.Close()

	solverClient, err := optimizergrpc.New(cfg.OptimizerCoreAddr, 2*time.Second)
	if err != nil {
		slog.Error("create optimizer grpc client", "err", err)
		os.Exit(1)
	}

	worker := adapter.NewWorker(cfg, spannerClient, solverClient)
	defer worker.Close()

	slog.Info("optimizer worker started",
		"topic", cfg.KafkaTopic,
		"group", cfg.KafkaGroupID,
		"optimizer_core_addr", cfg.OptimizerCoreAddr,
	)

	if err := worker.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("optimizer worker exited", "err", err)
		os.Exit(1)
	}

	slog.Info("optimizer worker stopped")
}
