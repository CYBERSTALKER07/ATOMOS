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

// acklamQuantile computes Peter J. Acklam's rational approximation for the inverse standard normal CDF (Φ⁻¹(p)).
// It achieves relative error < 1.15e-9 across the entire domain (0, 1).
func acklamQuantile(p float64) float64 {
	a1 := -3.969683028665376e+01
	a2 := 2.209460984245205e+02
	a3 := -2.759285104469687e+02
	a4 := 1.383577518672690e+02
	a5 := -3.066479806614716e+01
	a6 := 2.506628277459239e+00

	b1 := -5.447609879822406e+01
	b2 := 1.615858368580409e+02
	b3 := -1.556989798598866e+02
	b4 := 6.680131188771972e+01
	b5 := -1.328068155288572e+01

	c1 := -7.784894002430293e-03
	c2 := -3.223964580411365e-01
	c3 := -2.400758277161838e+00
	c4 := -2.549732539343734e+00
	c5 := 4.374664141464968e+00
	c6 := 2.938163982698783e+00

	d1 := 7.784695709041462e-03
	d2 := 3.224671290700398e-01
	d3 := 2.445134137142996e+00
	d4 := 3.754408661907416e+00

	const pLow = 0.02425
	const pHigh = 1.0 - pLow // 0.97575

	if p < pLow {
		q := math.Sqrt(-2.0 * math.Log(p))
		return (((((c1*q+c2)*q+c3)*q+c4)*q+c5)*q+c6) /
			((((d1*q+d2)*q+d3)*q+d4)*q + 1.0)
	} else if p <= pHigh {
		q := p - 0.5
		r := q * q
		return (((((a1*r+a2)*r+a3)*r+a4)*r+a5)*r+a6) * q /
			(((((b1*r+b2)*r+b3)*r+b4)*r+b5)*r + 1.0)
	} else {
		q := math.Sqrt(-2.0 * math.Log(1.0-p))
		return -(((((c1*q+c2)*q+c3)*q+c4)*q+c5)*q+c6) /
			((((d1*q+d2)*q+d3)*q+d4)*q + 1.0)
	}
}

// NormalZ returns the standard normal quantile for common cycle service levels using
// Peter J. Acklam's continuous rational approximation for the inverse CDF Φ⁻¹(p).
// Boundary rules:
// - Non-positive or NaN serviceLevel falls back to default 0.98 (~2.0537).
// - Service levels in (0, 0.50] return 0.0 (no safety stock buffer below median demand).
// - Service levels above 0.9999 are clamped to 0.9999 for numerical stability.
func NormalZ(serviceLevel float64) float64 {
	if math.IsNaN(serviceLevel) || serviceLevel <= 0 {
		return acklamQuantile(0.98)
	}
	if serviceLevel <= 0.50 {
		return 0.0
	}
	p := serviceLevel
	if p > 0.9999 {
		p = 0.9999
	}
	return acklamQuantile(p)
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
