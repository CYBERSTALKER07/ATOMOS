package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// PX §8.4: nightly DemandForecastBaseline vs completed line-unit accuracy (WAPE / bias / TS).
func main() {
	supplierID := flag.String("supplier-id", "", "filter by supplier_id (optional)")
	days := flag.Int("days", 28, "score lookback window in days")
	timeout := flag.Duration("timeout", 20*time.Minute, "job timeout")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if !planning.ForecastAccuracyEnabled() {
		slog.Info("planning accuracy skipped", "reason", "FORECAST_ACCURACY_ENABLED off")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}
	client, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		slog.Error("spanner client", "err", err)
		os.Exit(1)
	}
	defer client.Close()

	svc := &planning.AccuracyService{Client: client, Log: logger}
	written, alerts, err := svc.RunAccuracyPass(ctx, strings.TrimSpace(*supplierID), *days)
	if err != nil {
		slog.Error("accuracy pass failed", "err", err)
		os.Exit(1)
	}
	slog.Info("planning accuracy complete", "written", written, "alerts", alerts, "days", *days)
}

func spannerClientOptions(cfg *bootstrap.Config) []option.ClientOption {
	if strings.TrimSpace(cfg.SpannerEmulatorHost) == "" {
		return nil
	}
	return []option.ClientOption{
		option.WithEndpoint(cfg.SpannerEmulatorHost),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithoutAuthentication(),
	}
}

func spannerDatabasePath(cfg *bootstrap.Config) string {
	return fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		cfg.SpannerProject, cfg.SpannerInstance, cfg.SpannerDatabase)
}
