package replenishment

import (
	"context"
	"errors"
	"math"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// SafetyStockV2Enabled gates service-level SS formula vs legacy burn·lead·1.15.
func SafetyStockV2Enabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("SAFETY_STOCK_V2_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// LeadStats holds mean/stddev lead time in days.
type LeadStats struct {
	MeanDays   float64
	SigmaDays  float64
	SampleSize int64
	Assumed    bool
}

// SafetyStockInputs feeds the §8.2 formula.
type SafetyStockInputs struct {
	DBar            float64 // mean daily demand
	SigmaD          float64 // residual demand stdev
	SigmaDAssumed   bool
	L               float64 // lead time days
	SigmaL          float64 // lead time stdev days
	LeadSigmaAssumed bool
	Z               float64
	ServiceLevel    float64
}

// SafetyStockResult is SS + reorder point with audit fields.
type SafetyStockResult struct {
	SafetyStock      float64
	ReorderPoint     float64
	ZAlpha           float64
	SigmaD           float64
	SigmaL           float64
	LeadDays         float64
	DBar             float64
	SigmaDAssumed    bool
	LeadSigmaAssumed bool
	ServiceLevel     float64
}

// NormalZ returns the standard normal quantile for common cycle service levels.
func NormalZ(serviceLevel float64) float64 {
	switch {
	case serviceLevel >= 0.995:
		return 2.576
	case serviceLevel >= 0.99:
		return 2.326
	case serviceLevel >= 0.98:
		return 2.054
	case serviceLevel >= 0.95:
		return 1.645
	case serviceLevel >= 0.90:
		return 1.282
	case serviceLevel >= 0.85:
		return 1.036
	default:
		if serviceLevel <= 0 {
			return 2.054
		}
		// Rough inverse for uncommon levels via Abramowitz-ish clamp.
		return 2.054
	}
}

// SafetyStockUnits = z · √(L·σ_d² + d̄²·σ_L²)
func SafetyStockUnits(in SafetyStockInputs) float64 {
	z := in.Z
	if z <= 0 {
		z = NormalZ(in.ServiceLevel)
	}
	l := math.Max(0, in.L)
	sd := math.Max(0, in.SigmaD)
	sl := math.Max(0, in.SigmaL)
	dbar := math.Max(0, in.DBar)
	inner := l*sd*sd + dbar*dbar*sl*sl
	if inner <= 0 {
		return 0
	}
	return z * math.Sqrt(inner)
}

// ComputeReorderPoint returns SS and ROP = d̄·L + SS.
func ComputeReorderPoint(in SafetyStockInputs) SafetyStockResult {
	z := in.Z
	if z <= 0 {
		z = NormalZ(in.ServiceLevel)
	}
	ss := SafetyStockUnits(in)
	rop := in.DBar*in.L + ss
	return SafetyStockResult{
		SafetyStock:      ss,
		ReorderPoint:     rop,
		ZAlpha:           z,
		SigmaD:           in.SigmaD,
		SigmaL:           in.SigmaL,
		LeadDays:         in.L,
		DBar:             in.DBar,
		SigmaDAssumed:    in.SigmaDAssumed,
		LeadSigmaAssumed: in.LeadSigmaAssumed,
		ServiceLevel:     in.ServiceLevel,
	}
}

// LegacyReorderPoint is burn·lead·(1+0.15).
func LegacyReorderPoint(burn, leadDays float64) float64 {
	if leadDays <= 0 {
		leadDays = float64(defaultLeadTimeDays)
	}
	return burn*leadDays + burn*leadDays*safetyBufferMultiplier
}

// ResidualSigmaD returns stdev of SignedError for a series over the last 28 days.
func ResidualSigmaD(ctx context.Context, client *spanner.Client, supplierID, warehouseID, productID string) (sigma float64, samples int64, ok bool, err error) {
	if client == nil || supplierID == "" || productID == "" {
		return 0, 0, false, nil
	}
	end := civil.DateOf(time.Now().UTC())
	start := end.AddDays(-27)
	sql := `SELECT SignedError FROM ForecastAccuracyDaily
		WHERE SupplierId = @sid AND ProductId = @pid
		  AND ForecastDate BETWEEN @start AND @end`
	params := map[string]any{
		"sid": supplierID, "pid": productID,
		"start": start, "end": end,
	}
	if wh := strings.TrimSpace(warehouseID); wh != "" {
		sql += ` AND WarehouseId = @wh`
		params["wh"] = wh
	}
	iter := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var errs []float64
	for {
		row, nerr := iter.Next()
		if errors.Is(nerr, iterator.Done) {
			break
		}
		if nerr != nil {
			return 0, 0, false, nerr
		}
		var se int64
		if err := row.Columns(&se); err != nil {
			return 0, 0, false, err
		}
		errs = append(errs, float64(se))
	}
	if len(errs) < 7 {
		return 0, int64(len(errs)), false, nil
	}
	var mean float64
	for _, v := range errs {
		mean += v
	}
	mean /= float64(len(errs))
	var varSum float64
	for _, v := range errs {
		d := v - mean
		varSum += d * d
	}
	sigma = math.Sqrt(varSum / float64(len(errs)))
	return sigma, int64(len(errs)), true, nil
}

// ObservedLeadStats computes mean/σ of ReceivedAt−CreatedAt for a supplier (≥10 samples).
func ObservedLeadStats(ctx context.Context, client *spanner.Client, supplierID string) (LeadStats, error) {
	out := LeadStats{Assumed: true, MeanDays: float64(defaultLeadTimeDays), SigmaDays: 1.0}
	if client == nil || strings.TrimSpace(supplierID) == "" {
		return out, nil
	}
	sql := `SELECT CreatedAt, ReceivedAt FROM FactoryInternalTransfers
		WHERE SupplierId = @sid AND ReceivedAt IS NOT NULL AND State = 'RECEIVED'
		  AND (TransferMode IS NULL OR TransferMode != 'INTERNAL')`
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL:    sql,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	var days []float64
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return out, err
		}
		var created, received time.Time
		if err := row.Columns(&created, &received); err != nil {
			return out, err
		}
		d := received.Sub(created).Hours() / 24.0
		if d < 0.5 {
			continue // same-day / INTERNAL-ish noise
		}
		days = append(days, d)
	}
	out.SampleSize = int64(len(days))
	if len(days) < 10 {
		return out, nil
	}
	var mean float64
	for _, v := range days {
		mean += v
	}
	mean /= float64(len(days))
	var varSum float64
	for _, v := range days {
		d := v - mean
		varSum += d * d
	}
	sigma := math.Sqrt(varSum / float64(len(days)))
	if sigma < 0.1 {
		sigma = 0.1
	}
	return LeadStats{MeanDays: mean, SigmaDays: sigma, SampleSize: int64(len(days)), Assumed: false}, nil
}

// AvgBaselineDemand returns average BaselineQty over last 7 days for SKU/WH.
func AvgBaselineDemand(ctx context.Context, client *spanner.Client, supplierID, warehouseID, productID string) (float64, bool, error) {
	if client == nil || supplierID == "" || productID == "" {
		return 0, false, nil
	}
	end := civil.DateOf(time.Now().UTC())
	start := end.AddDays(-6)
	sql := `SELECT COALESCE(AVG(BaselineQty), 0), COUNT(*) FROM DemandForecastBaseline
		WHERE SupplierId = @sid AND ProductId = @pid AND ForecastDate BETWEEN @start AND @end`
	params := map[string]any{"sid": supplierID, "pid": productID, "start": start, "end": end}
	if wh := strings.TrimSpace(warehouseID); wh != "" {
		sql += ` AND WarehouseId = @wh`
		params["wh"] = wh
	}
	iter := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	var avg float64
	var n int64
	if err := row.Columns(&avg, &n); err != nil {
		return 0, false, err
	}
	if n == 0 || avg <= 0 {
		return 0, false, nil
	}
	return avg, true, nil
}

// InTransitBySKU sums SuggestedQuantity for open transfers linked to insights at a warehouse.
func InTransitBySKU(ctx context.Context, client *spanner.Client, warehouseID string) (map[string]int64, error) {
	out := map[string]int64{}
	if client == nil || strings.TrimSpace(warehouseID) == "" {
		return out, nil
	}
	sql := `SELECT i.ProductId, COALESCE(SUM(i.SuggestedQuantity), 0)
		FROM FactoryInternalTransfers t
		JOIN ReplenishmentInsights i ON t.SourceInsightId = i.InsightId
		WHERE i.WarehouseId = @wh
		  AND t.State NOT IN ('RECEIVED', 'CANCELLED', 'FAILED')
		  AND t.SourceInsightId IS NOT NULL
		GROUP BY i.ProductId`
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL:    sql,
		Params: map[string]any{"wh": warehouseID},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return out, err
		}
		var sku string
		var qty int64
		if err := row.Columns(&sku, &qty); err != nil {
			return out, err
		}
		if sku != "" && qty > 0 {
			out[sku] += qty
		}
	}
	return out, nil
}
