package planning

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning/forecast"
)

// ForecastAlgoEnabled gates Croston/SES/Holt–Winters baseline writes.
func ForecastAlgoEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("FORECAST_ALGO_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// ForecastRunner materializes DemandForecastBaseline from completed line-unit series.
type ForecastRunner struct {
	Client *spanner.Client
	Log    *slog.Logger
	Now    func() time.Time
}

func (r *ForecastRunner) now() time.Time {
	if r != nil && r.Now != nil {
		return r.Now().UTC()
	}
	return time.Now().UTC()
}

func (r *ForecastRunner) log() *slog.Logger {
	if r != nil && r.Log != nil {
		return r.Log
	}
	return slog.Default()
}

// RunForecastPass forecasts each (supplier, warehouse, product) series and writes baselines for targetDay.
// historyDays defaults to 90 (≥60 required for full fit).
func (r *ForecastRunner) RunForecastPass(ctx context.Context, supplierID string, historyDays int, targetDay time.Time) (written int, skipped int, err error) {
	if r == nil || r.Client == nil {
		return 0, 0, nil
	}
	if !ForecastAlgoEnabled() {
		r.log().Info("forecast algo pass skipped", "reason", "FORECAST_ALGO_ENABLED off")
		return 0, 0, nil
	}
	if historyDays < 60 {
		historyDays = 90
	}
	if targetDay.IsZero() {
		targetDay = r.now().Truncate(24 * time.Hour).AddDate(0, 0, 1) // tomorrow
	} else {
		targetDay = targetDay.UTC().Truncate(24 * time.Hour)
	}
	endHist := r.now().Truncate(24 * time.Hour)
	startHist := endHist.AddDate(0, 0, -(historyDays - 1))

	actuals, err := LoadCompletedActuals(ctx, r.Client, supplierID, startHist, endHist)
	if err != nil {
		return 0, 0, fmt.Errorf("load actuals: %w", err)
	}

	type seriesKey struct {
		SupplierID, WarehouseID, ProductID string
	}
	seriesDays := map[seriesKey]map[civil.Date]int64{}
	for k, qty := range actuals {
		if supplierID != "" && k.SupplierID != supplierID {
			continue
		}
		sk := seriesKey{k.SupplierID, k.WarehouseID, k.ProductID}
		if seriesDays[sk] == nil {
			seriesDays[sk] = map[civil.Date]int64{}
		}
		seriesDays[sk][k.Day] += qty
	}

	planSvc := NewService(r.Client)
	for sk, dayMap := range seriesDays {
		y := forecast.DenseDaily(dayMap, civil.DateOf(startHist), civil.DateOf(endHist))
		res := forecast.ForecastSeries(y)
		point := res.PointForecast
		src := NormalizeBaselineSource(res.BaselineSource)

		// Seasonal multiplier on baseline.
		mult := 1.0
		if tpl, _, terr := planSvc.ActiveSeasonalTemplate(ctx, sk.SupplierID, targetDay); terr == nil && tpl != nil && tpl.Multiplier > 0 {
			mult = tpl.Multiplier
			if math.Abs(mult-1) > 1e-6 {
				src = BaselineSourceMixed
			}
		}
		point *= mult
		low := int64(math.Floor(float64(res.LowUnits) * mult))
		high := int64(math.Ceil(float64(res.HighUnits) * mult))
		if low < 0 {
			low = 0
		}
		qty := int64(math.Round(point))
		if qty < 0 {
			qty = 0
		}
		if high < qty {
			high = qty
		}
		if low > qty {
			low = qty
		}

		confPct := int64(0)
		conf := 0.0
		blocked := res.BlockedReason
		if wape, sample, ok, werr := LoadLatestSeriesWape28(ctx, r.Client, sk.SupplierID, sk.WarehouseID, endHist); werr == nil && ok {
			// G6.A1: demote path fail-closes confidence before predictive push overwrite.
			if ShouldDemote(wape, sample) {
				blocked = AccuracyDemotedReason
				confPct = 5
				conf = 0.05
			} else if pct, have := ConfidencePctFromWape(wape, sample); have {
				confPct = pct
				conf = float64(pct) / 100
			}
		}
		if confPct <= 0 && blocked == "" {
			blocked = "insufficient_history"
		}

		if err := WriteBaselineWithOutbox(ctx, r.Client, r.now(), BaselineWriteInput{
			SupplierID:     sk.SupplierID,
			WarehouseID:    sk.WarehouseID,
			ProductID:      sk.ProductID,
			ForecastDate:   targetDay,
			BaselineQty:    qty,
			LowUnits:       low,
			HighUnits:      high,
			Confidence:     conf,
			ConfidencePct:  confPct,
			Source:         "forecast_algo",
			BaselineSource: src,
			BlockedReason:  blocked,
		}); err != nil {
			r.log().Warn("forecast baseline write failed",
				"supplier_id", sk.SupplierID, "warehouse_id", sk.WarehouseID,
				"product_id", sk.ProductID, "err", err)
			skipped++
			continue
		}
		written++
	}
	r.log().Info("forecast algo pass complete", "written", written, "skipped", skipped, "target", targetDay.Format("2006-01-02"))
	return written, skipped, nil
}
