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

	"cloud.google.com/go/spanner"
	"github.com/redis/go-redis/v9"

	"optimizercoreadapter/internal/adapter"
	"optimizercoreadapter/internal/config"
	"optimizercoreadapter/internal/optimizergrpc"
	"optimizercoreadapter/internal/telemetry"
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
	redisClient := newRedisClient(ctx, cfg.RedisAddress)
	if redisClient != nil {
		defer func() {
			if err := redisClient.Close(); err != nil {
				slog.Warn("close optimizer worker redis client", "err", err)
			}
		}()
	}

	worker := adapter.NewWorker(cfg, spannerClient, solverClient, redisClient)
	defer worker.Close()
	metricsServer := startMetricsServer(ctx, cancel, cfg.MetricsAddr)
	defer shutdownMetricsServer(metricsServer)

	slog.Info("optimizer worker started",
		"topic", cfg.KafkaTopic,
		"group", cfg.KafkaGroupID,
		"optimizer_core_addr", cfg.OptimizerCoreAddr,
		"metrics_addr", cfg.MetricsAddr,
	)

	if err := worker.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("optimizer worker exited", "err", err)
		os.Exit(1)
	}

	slog.Info("optimizer worker stopped")
}

func startMetricsServer(ctx context.Context, cancel context.CancelFunc, addr string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", telemetry.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	server := &http.Server{Addr: addr, Handler: mux}
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.ErrorContext(ctx, "optimizer metrics server failed", "addr", addr, "err", err)
			cancel()
		}
	}()
	return server
}

func shutdownMetricsServer(server *http.Server) {
	if server == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Warn("optimizer metrics server shutdown", "err", err)
	}
}

func newRedisClient(ctx context.Context, addr string) redis.UniversalClient {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		slog.WarnContext(ctx, "optimizer worker redis disabled: REDIS_ADDRESS not set")
		return nil
	}

	var client redis.UniversalClient
	if strings.Contains(addr, ",") {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        splitCSV(addr),
			DialTimeout:  3 * time.Second,
			ReadTimeout:  2 * time.Second,
			WriteTimeout: 2 * time.Second,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:         addr,
			DialTimeout:  3 * time.Second,
			ReadTimeout:  2 * time.Second,
			WriteTimeout: 2 * time.Second,
		})
	}

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		slog.WarnContext(ctx, "optimizer worker redis unavailable; continuing without active-set cleanup", "addr", addr, "err", err)
		_ = client.Close()
		return nil
	}

	return client
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}
