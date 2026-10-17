package replenishment

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"google.golang.org/api/iterator"
)

const (
	replayWarmupDays     = 28
	replayMinCalendarDays = 45
	replayMinNonZeroDays  = 14
	replayDefaultDays     = 90
)

// ReplayRequireGateEnabled exits non-zero when v2 misses target ±2pp or holds more stock than legacy.
func ReplayRequireGateEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("SAFETY_STOCK_REPLAY_REQUIRE_GATE")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// ReplayConfig controls a fill-rate replay pass.
type ReplayConfig struct {
	SupplierID         string
	Days               int
	TargetServiceLevel float64
	RequireGate        bool
	LeadDays           float64 // 0 → policy / observed
	LeadSigmaDays      float64 // 0 → policy / observed
}

// PolicyMetrics aggregates one ROP policy across SKUs.
type PolicyMetrics struct {
	UnitFillRate       float64 `json:"unit_fill_rate"`
	CycleServiceLevel  float64 `json:"cycle_service_level"`
	AvgOnHand          float64 `json:"avg_on_hand"`
	DemandUnits        float64 `json:"demand_units"`
	FulfilledUnits     float64 `json:"fulfilled_units"`
	DemandDays         int64   `json:"demand_days"`
	StockoutDays       int64   `json:"stockout_days"`
}

// ReplayResult compares legacy vs §8.2 on historical demand.
type ReplayResult struct {
	OK                 bool          `json:"ok"`
	SupplierID         string        `json:"supplier_id,omitempty"`
	Days               int           `json:"days"`
	WarmupDays         int           `json:"warmup_days"`
	TargetServiceLevel float64       `json:"target_service_level"`
	LeadDays           float64       `json:"lead_days"`
	LeadSigmaDays      float64       `json:"lead_sigma_days"`
	LeadSigmaAssumed   bool          `json:"lead_sigma_assumed"`
	SKUCount           int           `json:"sku_count"`
	SkippedSeries      int           `json:"skipped_series"`
	Legacy             PolicyMetrics `json:"legacy"`
	V2                 PolicyMetrics `json:"v2"`
	PassGate           bool          `json:"pass_gate"`
	GateRequired       bool          `json:"gate_required"`
	GateReason         string        `json:"gate_reason,omitempty"`
}

type seriesMetrics struct {
	DemandUnits   float64
	Fulfilled     float64
	DemandDays    int64
	StockoutDays  int64
	OnHandSum     float64
	ScoredDays    int64
}

// ROPFunc returns reorder point for day index given trailing stats.
type ROPFunc func(dBar, sigmaD float64) float64

// RunFillRateReplay loads COMPLETED demand and simulates legacy vs v2 ROP policies.
func RunFillRateReplay(ctx context.Context, client *spanner.Client, cfg ReplayConfig) (ReplayResult, error) {
	days := cfg.Days
	if days < replayMinCalendarDays {
		days = replayDefaultDays
	}
	if days > 180 {
		days = 180
	}
	sl := cfg.TargetServiceLevel
	if sl <= 0 {
		sl = 0.98
	}
	requireGate := cfg.RequireGate || ReplayRequireGateEnabled()

	out := ReplayResult{
		OK:                 true,
		SupplierID:         strings.TrimSpace(cfg.SupplierID),
		Days:               days,
		WarmupDays:         replayWarmupDays,
		TargetServiceLevel: sl,
		GateRequired:       requireGate,
	}
	if client == nil {
		return out, errors.New("spanner unavailable")
	}

	end := time.Now().UTC().Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -(days - 1))
	startDate := civil.DateOf(start)
	endDate := civil.DateOf(end)

	leadDays := cfg.LeadDays
	leadSigma := cfg.LeadSigmaDays
	leadAssumed := true
	if out.SupplierID != "" {
		policy, err := LoadPolicy(ctx, client, out.SupplierID)
		if err == nil {
			if cfg.TargetServiceLevel <= 0 && policy.TargetServiceLevel > 0 {
				sl = policy.TargetServiceLevel
				out.TargetServiceLevel = sl
			}
			if leadDays <= 0 {
				leadDays = float64(policy.LeadTimeDays)
			}
			if cfg.LeadSigmaDays == 0 {
				leadSigma = policy.LeadTimeSigmaDays
			}
		}
		if obs, err := ObservedLeadStats(ctx, client, out.SupplierID); err == nil && !obs.Assumed {
			leadDays = obs.MeanDays
			leadSigma = obs.SigmaDays
			leadAssumed = false
		}
	}
	if leadDays <= 0 {
		leadDays = float64(defaultLeadTimeDays)
	}
	if leadSigma <= 0 {
		leadSigma = 1.0
	}
	out.LeadDays = leadDays
	out.LeadSigmaDays = leadSigma
	out.LeadSigmaAssumed = leadAssumed
	leadInt := int(math.Max(1, math.Round(leadDays)))

	series, err := loadReplayDemandSeries(ctx, client, out.SupplierID, start, end)
	if err != nil {
		return out, err
	}

	var legAcc, v2Acc seriesMetrics
	skuCount := 0
	skipped := 0

	for key, dayMap := range series {
		demand := denseDailyDemand(dayMap, startDate, endDate)
		nonZero := 0
		for _, d := range demand {
			if d > 0 {
				nonZero++
			}
		}
		if len(demand) < replayMinCalendarDays || nonZero < replayMinNonZeroDays {
			skipped++
			continue
		}

		openStock := openingStock(demand[:replayWarmupDays], leadDays)
		sigmaFromAcc, _ := residualSigmaFromAccuracy(ctx, client, key.supplierID, key.warehouseID, key.productID, startDate, endDate)

		legacyROP := func(dBar, _ float64) float64 {
			return LegacyReorderPoint(dBar, leadDays)
		}
		v2ROP := func(dBar, sigmaD float64) float64 {
			sd := sigmaD
			assumed := false
			if sd <= 0 {
				sd = math.Max(dBar*0.25, 1)
				assumed = true
			}
			return ComputeReorderPoint(SafetyStockInputs{
				DBar:             dBar,
				SigmaD:           sd,
				SigmaDAssumed:    assumed,
				L:                leadDays,
				SigmaL:           leadSigma,
				LeadSigmaAssumed: leadAssumed,
				ServiceLevel:     sl,
			}).ReorderPoint
		}

		legM := simulateSeries(demand, legacyROP, leadInt, openStock, sigmaFromAcc)
		v2M := simulateSeries(demand, v2ROP, leadInt, openStock, sigmaFromAcc)
		legAcc = accumulate(legAcc, legM)
		v2Acc = accumulate(v2Acc, v2M)
		skuCount++
	}

	out.SKUCount = skuCount
	out.SkippedSeries = skipped
	out.Legacy = toPolicyMetrics(legAcc)
	out.V2 = toPolicyMetrics(v2Acc)
	out.PassGate, out.GateReason = evaluateReplayGate(out.Legacy, out.V2, sl)
	if requireGate && skuCount > 0 && !out.PassGate {
		out.OK = false
		return out, errors.New(out.GateReason)
	}
	return out, nil
}

func evaluateReplayGate(legacy, v2 PolicyMetrics, targetSL float64) (bool, string) {
	if targetSL <= 0 {
		targetSL = 0.98
	}
	minSL := targetSL - 0.02
	if v2.CycleServiceLevel+1e-9 < minSL {
		return false, "gate failed: v2 cycle_service_level below target-2pp"
	}
	if legacy.AvgOnHand > 0 && v2.AvgOnHand > legacy.AvgOnHand*1.02+1e-9 {
		return false, "gate failed: v2 avg_on_hand exceeds legacy by >2%"
	}
	return true, ""
}

func toPolicyMetrics(m seriesMetrics) PolicyMetrics {
	pm := PolicyMetrics{
		DemandUnits:    m.DemandUnits,
		FulfilledUnits: m.Fulfilled,
		DemandDays:     m.DemandDays,
		StockoutDays:   m.StockoutDays,
	}
	if m.DemandUnits > 0 {
		pm.UnitFillRate = m.Fulfilled / m.DemandUnits
	}
	if m.DemandDays > 0 {
		pm.CycleServiceLevel = float64(m.DemandDays-m.StockoutDays) / float64(m.DemandDays)
	}
	if m.ScoredDays > 0 {
		pm.AvgOnHand = m.OnHandSum / float64(m.ScoredDays)
	}
	return pm
}

func accumulate(a, b seriesMetrics) seriesMetrics {
	return seriesMetrics{
		DemandUnits:  a.DemandUnits + b.DemandUnits,
		Fulfilled:    a.Fulfilled + b.Fulfilled,
		DemandDays:   a.DemandDays + b.DemandDays,
		StockoutDays: a.StockoutDays + b.StockoutDays,
		OnHandSum:    a.OnHandSum + b.OnHandSum,
		ScoredDays:   a.ScoredDays + b.ScoredDays,
	}
}

type replaySKUKey struct {
	supplierID  string
	warehouseID string
	productID   string
}

func loadReplayDemandSeries(
	ctx context.Context,
	client *spanner.Client,
	supplierID string,
	start, end time.Time,
) (map[replaySKUKey]map[civil.Date]int64, error) {
	out := make(map[replaySKUKey]map[civil.Date]int64)
	start = start.UTC().Truncate(24 * time.Hour)
	end = end.UTC().Truncate(24 * time.Hour)
	endExclusive := end.Add(24 * time.Hour)
	sql := `SELECT SupplierId, WarehouseId, LineItemsJson, UpdatedAt
		FROM Orders
		WHERE Status = 'COMPLETED'
		  AND UpdatedAt >= @start
		  AND UpdatedAt < @end`
	params := map[string]any{"start": start, "end": endExclusive}
	if sid := strings.TrimSpace(supplierID); sid != "" {
		sql += ` AND SupplierId = @sid`
		params["sid"] = sid
	}
	iter := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var sid, whID string
		var raw []byte
		var updatedAt time.Time
		if err := row.Columns(&sid, &whID, &raw, &updatedAt); err != nil {
			return nil, err
		}
		var items []order.LineItem
		if len(raw) == 0 || json.Unmarshal(raw, &items) != nil {
			continue
		}
		day := civil.DateOf(updatedAt.UTC())
		for _, item := range items {
			pid := strings.TrimSpace(item.SKU)
			if pid == "" || item.Quantity <= 0 {
				continue
			}
			key := replaySKUKey{sid, whID, pid}
			if out[key] == nil {
				out[key] = map[civil.Date]int64{}
			}
			out[key][day] += item.Quantity
		}
	}
	return out, nil
}

func denseDailyDemand(dayMap map[civil.Date]int64, start, end civil.Date) []float64 {
	if end.Before(start) {
		return nil
	}
	n := 0
	for d := start; !d.After(end); d = d.AddDays(1) {
		n++
	}
	out := make([]float64, n)
	i := 0
	for d := start; !d.After(end); d = d.AddDays(1) {
		out[i] = float64(dayMap[d])
		i++
	}
	return out
}

func openingStock(warmup []float64, leadDays float64) float64 {
	if len(warmup) == 0 {
		return math.Max(leadDays, 1)
	}
	var sum float64
	for _, v := range warmup {
		sum += v
	}
	mean := sum / float64(len(warmup))
	if mean <= 0 {
		mean = 1
	}
	if leadDays <= 0 {
		leadDays = float64(defaultLeadTimeDays)
	}
	return math.Ceil(mean * leadDays * 1.5)
}

// simulateSeries runs continuous-review order-up-to-ROP on a demand series.
// sigmaOverride > 0 forces σ_d for all scored days (accuracy path); else rolling residual.
func simulateSeries(demand []float64, ropFn ROPFunc, leadDays int, openStock float64, sigmaOverride float64) seriesMetrics {
	if leadDays < 1 {
		leadDays = 1
	}
	n := len(demand)
	onHand := openStock
	inbound := make([]float64, n+leadDays+2)
	var onOrder float64
	var m seriesMetrics

	for t := 0; t < n; t++ {
		if t < len(inbound) {
			recv := inbound[t]
			if recv > 0 {
				onHand += recv
				onOrder -= recv
				if onOrder < 0 {
					onOrder = 0
				}
			}
		}
		d := demand[t]
		fulfilled := math.Min(d, onHand)
		onHand -= fulfilled
		if onHand < 0 {
			onHand = 0
		}

		dBar := trailingMean(demand, t, 7)
		sigmaD := sigmaOverride
		if sigmaD <= 0 {
			sigmaD = rollingDemandResidualSigma(demand, t, 28)
		}
		rop := ropFn(dBar, sigmaD)
		ip := onHand + onOrder
		if ip <= rop {
			qty := math.Ceil(rop - ip)
			if qty < 1 && dBar > 0 {
				qty = math.Ceil(dBar * float64(leadDays))
			}
			if qty >= 1 {
				arr := t + leadDays
				if arr < len(inbound) {
					inbound[arr] += qty
				}
				onOrder += qty
			}
		}

		if t >= replayWarmupDays {
			m.ScoredDays++
			m.OnHandSum += onHand
			m.DemandUnits += d
			m.Fulfilled += fulfilled
			if d > 0 {
				m.DemandDays++
				if fulfilled+1e-9 < d {
					m.StockoutDays++
				}
			}
		}
	}
	return m
}

func trailingMean(demand []float64, t, window int) float64 {
	if window < 1 {
		window = 7
	}
	from := t - window + 1
	if from < 0 {
		from = 0
	}
	if from > t {
		return 0
	}
	var sum float64
	n := 0
	for i := from; i <= t; i++ {
		sum += demand[i]
		n++
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

func rollingDemandResidualSigma(demand []float64, t, window int) float64 {
	if window < 7 {
		window = 28
	}
	from := t - window + 1
	if from < 0 {
		from = 0
	}
	var residuals []float64
	for i := from; i <= t; i++ {
		mean := trailingMean(demand, i, 7)
		residuals = append(residuals, demand[i]-mean)
	}
	if len(residuals) < 7 {
		return 0
	}
	var mean float64
	for _, r := range residuals {
		mean += r
	}
	mean /= float64(len(residuals))
	var varSum float64
	for _, r := range residuals {
		d := r - mean
		varSum += d * d
	}
	return math.Sqrt(varSum / float64(len(residuals)))
}

func residualSigmaFromAccuracy(
	ctx context.Context,
	client *spanner.Client,
	supplierID, warehouseID, productID string,
	start, end civil.Date,
) (float64, bool) {
	if client == nil || supplierID == "" || productID == "" {
		return 0, false
	}
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
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return 0, false
		}
		var se int64
		if err := row.Columns(&se); err != nil {
			return 0, false
		}
		errs = append(errs, float64(se))
	}
	if len(errs) < 7 {
		return 0, false
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
	return math.Sqrt(varSum / float64(len(errs))), true
}

// TopOffenderSKUs is reserved for future log ranking; kept for API stability.
func TopOffenderSKUs(rates map[string]float64, limit int) []string {
	type pair struct {
		k string
		v float64
	}
	var list []pair
	for k, v := range rates {
		list = append(list, pair{k, v})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].v < list[j].v })
	if limit <= 0 || limit > len(list) {
		limit = len(list)
	}
	out := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, list[i].k)
	}
	return out
}
