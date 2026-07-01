package planning

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ForecastConfidence is the API contract for forecast range + score labeling.
type ForecastConfidence struct {
	LowUnits       int64  `json:"low_units,omitempty"`
	HighUnits      int64  `json:"high_units,omitempty"`
	ConfidencePct  int64  `json:"confidence_pct,omitempty"`
	BaselineSource string `json:"baseline_source,omitempty"`
	BlockedReason  string `json:"blocked_reason,omitempty"`
	Label          string `json:"label,omitempty"`
}

// DemandConfidenceQuery scopes supplier demand confidence aggregation.
type DemandConfidenceQuery struct {
	Granularity     string
	WarehouseID     string
	RetailerID      string
	ForecastDate    time.Time
	FallbackQty     int64
	SourceHint      string
	PredictionCount int64
}

// AggregateDemandConfidence builds confidence from DemandForecastBaseline rows or fallback math.
func AggregateDemandConfidence(ctx context.Context, client *spanner.Client, supplierID string, q DemandConfidenceQuery) (ForecastConfidence, error) {
	if strings.TrimSpace(q.Granularity) == "" {
		q.Granularity = "macro"
	}
	if q.RetailerID != "" || q.Granularity == "micro" {
		return microDemandConfidence(ctx, client, q)
	}
	if client == nil || supplierID == "" {
		return FallbackDemandConfidence(q.FallbackQty, q.SourceHint, q.PredictionCount), nil
	}

	day := q.ForecastDate.UTC().Truncate(24 * time.Hour)
	sql := `SELECT
		COALESCE(SUM(BaselineQty), 0),
		COALESCE(SUM(COALESCE(LowUnits, BaselineQty)), 0),
		COALESCE(SUM(COALESCE(HighUnits, BaselineQty)), 0),
		COALESCE(AVG(Confidence), 0),
		COALESCE(AVG(ConfidencePct), 0),
		MAX(COALESCE(BaselineSource, Source))
	FROM DemandForecastBaseline
	WHERE SupplierId = @sid AND ForecastDate = @day`
	params := map[string]any{"sid": supplierID, "day": day}
	if q.Granularity == "regional" && strings.TrimSpace(q.WarehouseID) != "" {
		sql += ` AND WarehouseId = @wh`
		params["wh"] = strings.TrimSpace(q.WarehouseID)
	}

	iter := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return FallbackDemandConfidence(q.FallbackQty, q.SourceHint, q.PredictionCount), nil
	}
	if err != nil {
		return ForecastConfidence{}, err
	}
	var mid, low, high int64
	var avgConfidence float64
	var avgConfidencePct spanner.NullInt64
	var baselineSource spanner.NullString
	if err := row.Columns(&mid, &low, &high, &avgConfidence, &avgConfidencePct, &baselineSource); err != nil {
		return ForecastConfidence{}, err
	}
	confidencePct := int64(0)
	if avgConfidencePct.Valid && avgConfidencePct.Int64 > 0 {
		confidencePct = avgConfidencePct.Int64
	} else if avgConfidence > 0 {
		confidencePct = int64(math.Round(avgConfidence * 100))
	}
	if mid <= 0 {
		return FallbackDemandConfidence(q.FallbackQty, q.SourceHint, q.PredictionCount), nil
	}
	src := NormalizeBaselineSource(baselineSource.StringVal, q.SourceHint)
	if confidencePct <= 0 {
		confidencePct = defaultConfidencePct(src, q.PredictionCount)
	}
	if low <= 0 || high <= 0 {
		spread := int64(math.Max(1, math.Round(float64(mid)*0.1)))
		low = mid - spread
		if low < 0 {
			low = 0
		}
		high = mid + spread
	}
	label := "standard"
	if q.PredictionCount > 0 && q.PredictionCount < 3 {
		label = "early_signal"
	}
	return ForecastConfidence{
		LowUnits:       low,
		HighUnits:      high,
		ConfidencePct:  confidencePct,
		BaselineSource: src,
		Label:          label,
	}, nil
}

func microDemandConfidence(ctx context.Context, client *spanner.Client, q DemandConfidenceQuery) (ForecastConfidence, error) {
	retailerID := strings.TrimSpace(q.RetailerID)
	if retailerID == "" {
		return FallbackDemandConfidence(q.FallbackQty, q.SourceHint, q.PredictionCount), nil
	}
	sparsity, err := CanForecast(ctx, client, retailerID)
	if err != nil {
		return ForecastConfidence{}, err
	}
	if !sparsity.Allowed {
		return ForecastConfidence{
			BlockedReason: sparsity.BlockedReason,
			Label:         sparsity.Label,
		}, nil
	}
	fallback := FallbackDemandConfidence(q.FallbackQty, q.SourceHint, q.PredictionCount)
	if fallback.ConfidencePct > 0 {
		fallback.ConfidencePct = ApplyConfidenceCap(sparsity, fallback.ConfidencePct)
	}
	if sparsity.Label != "" {
		fallback.Label = sparsity.Label
	}
	return fallback, nil
}

// FallbackDemandConfidence derives range/score when baseline rows are absent.
func FallbackDemandConfidence(mid int64, sourceHint string, predictionCount int64) ForecastConfidence {
	if predictionCount == 0 && mid <= 0 {
		return ForecastConfidence{
			BlockedReason:  "no_predictions",
			Label:          "insufficient_history",
			BaselineSource: NormalizeBaselineSource("", sourceHint),
		}
	}
	if mid <= 0 {
		mid = predictionCount
		if mid <= 0 {
			mid = 1
		}
	}
	spread := int64(math.Max(1, math.Round(float64(mid)*0.1)))
	src := NormalizeBaselineSource("", sourceHint)
	confidence := defaultConfidencePct(src, predictionCount)
	label := "standard"
	if predictionCount < 3 {
		label = "early_signal"
	}
	return ForecastConfidence{
		LowUnits:       mid - spread,
		HighUnits:      mid + spread,
		ConfidencePct:  confidence,
		BaselineSource: src,
		Label:          label,
	}
}

// ProductForecastBreakdown returns demand_breakdown JSON fields for warehouse/factory insight cards.
func ProductForecastBreakdown(ctx context.Context, client *spanner.Client, supplierID, warehouseID, productID string, day time.Time) map[string]any {
	if client == nil || supplierID == "" || productID == "" {
		return nil
	}
	day = day.UTC().Truncate(24 * time.Hour)
	sql := `SELECT BaselineQty, LowUnits, HighUnits, ConfidencePct, Confidence, BaselineSource, Source, BlockedReason
		FROM DemandForecastBaseline
		WHERE SupplierId = @sid AND ForecastDate = @day AND ProductId = @pid`
	params := map[string]any{"sid": supplierID, "day": day, "pid": productID}
	if strings.TrimSpace(warehouseID) != "" {
		sql += ` AND WarehouseId = @wh`
		params["wh"] = strings.TrimSpace(warehouseID)
	}
	sql += ` ORDER BY BaselineQty DESC LIMIT 1`

	iter := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return nil
	}
	if err != nil {
		return nil
	}
	var qty int64
	var low, high, confidencePct spanner.NullInt64
	var confidence float64
	var baselineSource, source, blocked spanner.NullString
	if err := row.Columns(&qty, &low, &high, &confidencePct, &confidence, &baselineSource, &source, &blocked); err != nil {
		return nil
	}
	out := map[string]any{
		"baseline_source": NormalizeBaselineSource(baselineSource.StringVal, source.StringVal),
		"label":           "standard",
	}
	if blocked.StringVal != "" {
		out["blocked_reason"] = blocked.StringVal
		out["label"] = "insufficient_history"
	}
	if low.Valid && high.Valid {
		out["low_units"] = low.Int64
		out["high_units"] = high.Int64
	} else if qty > 0 {
		spread := int64(math.Max(1, math.Round(float64(qty)*0.1)))
		out["low_units"] = qty - spread
		out["high_units"] = qty + spread
	}
	if confidencePct.Valid && confidencePct.Int64 > 0 {
		out["confidence_pct"] = confidencePct.Int64
	} else if confidence > 0 {
		out["confidence_pct"] = int64(math.Round(confidence * 100))
	}
	return out
}

func defaultConfidencePct(src string, predictionCount int64) int64 {
	switch NormalizeBaselineSource(src) {
	case BaselineSourceSeasonalTemplate:
		return 75
	case BaselineSourceInventoryHint:
		return 72
	default:
		if predictionCount >= 5 {
			return 72
		}
		return 65
	}
}
