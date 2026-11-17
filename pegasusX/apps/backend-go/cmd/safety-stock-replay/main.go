package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/replenishment"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// PX §8.2 residual: offline 90-day fill-rate replay (legacy 1.15 vs service-level SS).
func main() {
	supplierID := flag.String("supplier-id", "", "filter by supplier_id (optional)")
	days := flag.Int("days", 90, "history lookback days")
	targetSL := flag.Float64("target-service-level", 0, "override TargetServiceLevel (0=policy/default 0.98)")
	requireGate := flag.Bool("require-gate", false, "exit 1 unless v2 hits target-2pp at ≤legacy+2% OH")
	timeout := flag.Duration("timeout", 25*time.Minute, "job timeout")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

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

	replayCfg := replenishment.ReplayConfig{
		SupplierID:         strings.TrimSpace(*supplierID),
		Days:               *days,
		TargetServiceLevel: *targetSL,
		RequireGate:        *requireGate || replenishment.ReplayRequireGateEnabled(),
	}
	result, err := replenishment.RunFillRateReplay(ctx, client, replayCfg)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(result)
	if err != nil {
		slog.Error("fill-rate replay failed", "err", err)
		os.Exit(1)
	}
	slog.Info("fill-rate replay complete",
		"sku_count", result.SKUCount,
		"legacy_fill", result.Legacy.UnitFillRate,
		"v2_fill", result.V2.UnitFillRate,
		"legacy_cycle_sl", result.Legacy.CycleServiceLevel,
		"v2_cycle_sl", result.V2.CycleServiceLevel,
		"legacy_avg_oh", result.Legacy.AvgOnHand,
		"v2_avg_oh", result.V2.AvgOnHand,
		"pass_gate", result.PassGate,
	)
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
