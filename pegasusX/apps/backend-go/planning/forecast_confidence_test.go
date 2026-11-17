package planning

import (
	"context"
	"testing"
	"time"
)

func TestFallbackDemandConfidenceBlockedWhenEmpty(t *testing.T) {
	out := FallbackDemandConfidence(0, "ai_recommendations", 0)
	if out.Label != "insufficient_history" {
		t.Fatalf("label=%q want insufficient_history", out.Label)
	}
	if out.BlockedReason != "no_predictions" {
		t.Fatalf("blocked=%q want no_predictions", out.BlockedReason)
	}
}

func TestFallbackDemandConfidenceRange(t *testing.T) {
	out := FallbackDemandConfidence(100, "demand_forecast_baseline", 4)
	if out.LowUnits != 90 || out.HighUnits != 110 {
		t.Fatalf("range=%d-%d want 90-110", out.LowUnits, out.HighUnits)
	}
	if out.BaselineSource != BaselineSourceMovingAverage {
		t.Fatalf("source=%q want moving_average", out.BaselineSource)
	}
}

func TestNormalizeBaselineSourceDemandBaseline(t *testing.T) {
	if got := NormalizeBaselineSource("demand_forecast_baseline"); got != BaselineSourceMovingAverage {
		t.Fatalf("got=%q", got)
	}
}

func TestAggregateDemandConfidence_NilClient(t *testing.T) {
	q := DemandConfidenceQuery{
		ForecastDate: time.Now(),
		FallbackQty:  50,
		SourceHint:   "demand_forecast_baseline",
	}
	out, err := AggregateDemandConfidence(context.Background(), nil, "sup-1", q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.LowUnits != 45 || out.HighUnits != 55 {
		t.Fatalf("unexpected range: %d-%d", out.LowUnits, out.HighUnits)
	}
}
