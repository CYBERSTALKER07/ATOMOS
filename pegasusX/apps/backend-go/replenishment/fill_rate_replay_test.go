package replenishment

import (
	"math"
	"testing"
)

func TestSimulateSeriesConstantDemandHighFill(t *testing.T) {
	demand := make([]float64, 60)
	for i := range demand {
		demand[i] = 10
	}
	lead := 2
	open := openingStock(demand[:replayWarmupDays], float64(lead))
	legacy := simulateSeries(demand, func(dBar, _ float64) float64 {
		return LegacyReorderPoint(dBar, float64(lead))
	}, lead, open, 0)
	v2 := simulateSeries(demand, func(dBar, sigmaD float64) float64 {
		sd := sigmaD
		if sd <= 0 {
			sd = math.Max(dBar*0.25, 1)
		}
		return ComputeReorderPoint(SafetyStockInputs{
			DBar: dBar, SigmaD: sd, L: float64(lead), SigmaL: 1, ServiceLevel: 0.98,
		}).ReorderPoint
	}, lead, open, 2.0)

	legM := toPolicyMetrics(legacy)
	v2M := toPolicyMetrics(v2)
	if legM.UnitFillRate < 0.95 {
		t.Fatalf("legacy fill=%v", legM.UnitFillRate)
	}
	if v2M.UnitFillRate < 0.95 {
		t.Fatalf("v2 fill=%v", v2M.UnitFillRate)
	}
	// Extra σ buffer → v2 should hold at least as much inventory on average.
	if v2M.AvgOnHand+1e-6 < legM.AvgOnHand {
		t.Fatalf("expected v2 avg OH >= legacy; v2=%v legacy=%v", v2M.AvgOnHand, legM.AvgOnHand)
	}
}

func TestSimulateSeriesVolatileV2ProtectsBetter(t *testing.T) {
	demand := make([]float64, 70)
	for i := range demand {
		if i%5 == 0 {
			demand[i] = 40
		} else {
			demand[i] = 4
		}
	}
	lead := 2
	open := openingStock(demand[:replayWarmupDays], float64(lead))
	legacy := toPolicyMetrics(simulateSeries(demand, func(dBar, _ float64) float64 {
		return LegacyReorderPoint(dBar, float64(lead))
	}, lead, open, 0))
	v2 := toPolicyMetrics(simulateSeries(demand, func(dBar, sigmaD float64) float64 {
		sd := sigmaD
		if sd <= 0 {
			sd = math.Max(dBar*0.25, 1)
		}
		return ComputeReorderPoint(SafetyStockInputs{
			DBar: dBar, SigmaD: sd, L: float64(lead), SigmaL: 1.5, ServiceLevel: 0.98,
		}).ReorderPoint
	}, lead, open, 12))

	if v2.CycleServiceLevel+1e-9 < legacy.CycleServiceLevel {
		t.Fatalf("v2 cycle SL %v should be >= legacy %v on volatile series", v2.CycleServiceLevel, legacy.CycleServiceLevel)
	}
}

func TestDenseDailyAndOpeningStock(t *testing.T) {
	if got := openingStock([]float64{10, 10, 10}, 2); got != math.Ceil(10*2*1.5) {
		t.Fatalf("open=%v", got)
	}
	short := make([]float64, 20)
	m := simulateSeries(short, func(dBar, _ float64) float64 { return LegacyReorderPoint(math.Max(dBar, 1), 2) }, 2, 10, 0)
	// Short series still scores after warmup if len > warmup; 20 < 28 → no scored days.
	if m.ScoredDays != 0 {
		t.Fatalf("expected no scored days for short series, got %d", m.ScoredDays)
	}
}

func TestEvaluateReplayGate(t *testing.T) {
	legacy := PolicyMetrics{CycleServiceLevel: 0.90, AvgOnHand: 100}
	v2ok := PolicyMetrics{CycleServiceLevel: 0.97, AvgOnHand: 95}
	pass, reason := evaluateReplayGate(legacy, v2ok, 0.98)
	if !pass {
		t.Fatalf("expected pass, reason=%s", reason)
	}
	v2low := PolicyMetrics{CycleServiceLevel: 0.90, AvgOnHand: 95}
	pass, _ = evaluateReplayGate(legacy, v2low, 0.98)
	if pass {
		t.Fatal("expected fail on low cycle SL")
	}
	v2heavy := PolicyMetrics{CycleServiceLevel: 0.99, AvgOnHand: 120}
	pass, _ = evaluateReplayGate(legacy, v2heavy, 0.98)
	if pass {
		t.Fatal("expected fail on higher avg OH")
	}
}
