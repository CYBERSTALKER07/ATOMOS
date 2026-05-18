// Package main is the pegasusX AI worker entrypoint.
//
// Consumes Kafka events (forecast triggers, replenishment signals, ETA
// estimation requests) and writes back through Spanner read-write transactions
// with version gating to reject stale replays.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	slog.Info("pegasusX ai-worker starting")
	<-ctx.Done()
	slog.Info("pegasusX ai-worker shutting down")
}
