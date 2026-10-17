package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning/forecast"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// PX §8.1: nightly Croston / SES / Holt–Winters baseline materialization.
// -mode=forecast (default) writes DemandForecastBaseline
// -mode=backtest compares algo vs 7-day mean WAPE
func main() {
	supplierID := flag.String("supplier-id", "", "filter by supplier_id (optional)")
	days := flag.Int("days", 90, "history lookback days")
	mode := flag.String("mode", "forecast", "forecast or backtest")
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

	switch strings.ToLower(strings.TrimSpace(*mode)) {
	case "backtest":
		if err := runBacktest(ctx, client, strings.TrimSpace(*supplierID), *days, logger); err != nil {
			slog.Error("backtest failed", "err", err)
			os.Exit(1)
		}
	default:
		if !planning.ForecastAlgoEnabled() {
			slog.Info("planning forecast skipped", "reason", "FORECAST_ALGO_ENABLED off")
			return
		}
		runner := &planning.ForecastRunner{Client: client, Log: logger}
		written, skipped, err := runner.RunForecastPass(ctx, strings.TrimSpace(*supplierID), *days, time.Time{})
		if err != nil {
			slog.Error("forecast pass failed", "err", err)
			os.Exit(1)
		}
		slog.Info("planning forecast complete", "written", written, "skipped", skipped)
		if planning.SeasonalEstimateEnabled() && strings.TrimSpace(*supplierID) != "" {
			planSvc := planning.NewService(client)
			est, estErr := planSvc.EstimateCalendarMultipliers(ctx, strings.TrimSpace(*supplierID), time.Time{}, true)
			if estErr != nil {
				slog.Warn("seasonal estimate failed", "err", estErr)
			} else {
				slog.Info("seasonal estimate complete",
					"suggestions", len(est.Suggestions),
					"persisted_drafts", est.PersistedDrafts,
				)
			}
		}
	}
}

func runBacktest(ctx context.Context, client *spanner.Client, supplierID string, days int, log *slog.Logger) error {
	if days < 60 {
		days = 90
	}
	end := time.Now().UTC().Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -(days - 1))
	actuals, err := planning.LoadCompletedActuals(ctx, client, supplierID, start, end)
	if err != nil {
		return err
	}
	type sk struct{ S, W, P string }
	series := map[sk]map[civil.Date]int64{}
	for k, qty := range actuals {
		key := sk{k.SupplierID, k.WarehouseID, k.ProductID}
		if series[key] == nil {
			series[key] = map[civil.Date]int64{}
		}
		series[key][k.Day] += qty
	}

	var improved, total int
	var sumAlgoWAPE, sumMeanWAPE float64
	originDays := 90
	if days < originDays {
		originDays = days - 14
	}
	if originDays < 14 {
		originDays = 14
	}

	for key, dayMap := range series {
		y := forecast.DenseDaily(dayMap, civil.DateOf(start), civil.DateOf(end))
		if forecast.NonZeroCount(y) < 14 || len(y) < 60 {
			continue
		}
		var algoPred, meanPred, acts []float64
		// Rolling origin on last originDays (fit on prefix, predict next).
		from := len(y) - originDays
		if from < 60 {
			from = 60
		}
		for t := from; t < len(y); t++ {
			prefix := y[:t]
			res := forecast.ForecastSeries(prefix)
			algoPred = append(algoPred, res.PointForecast)
			meanPred = append(meanPred, forecast.TrailingMean7(prefix))
			acts = append(acts, y[t])
		}
		if len(acts) == 0 {
			continue
		}
		aw, _ := forecast.WAPEBias(algoPred, acts)
		mw, _ := forecast.WAPEBias(meanPred, acts)
		total++
		sumAlgoWAPE += aw
		sumMeanWAPE += mw
		// Algo beats mean by >15% relative WAPE reduction.
		if mw > 0 && aw <= mw*0.85 {
			improved++
		}
		_ = key
	}

	pct := 0.0
	if total > 0 {
		pct = 100 * float64(improved) / float64(total)
	}
	log.Info("forecast backtest complete",
		"series", total,
		"improved_gt_15pct_wape", improved,
		"improved_pct", math.Round(pct*10)/10,
		"avg_algo_wape", sumAlgoWAPE/math.Max(1, float64(total)),
		"avg_mean_wape", sumMeanWAPE/math.Max(1, float64(total)),
	)

	require := strings.ToLower(strings.TrimSpace(os.Getenv("FORECAST_ALGO_REQUIRE_GATE")))
	gateOn := require == "1" || require == "true" || require == "yes" || require == "on"
	if gateOn && total > 0 && pct < 80 {
		return fmt.Errorf("gate failed: improved on %.1f%% of series (need >=80%%)", pct)
	}
	return nil
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
