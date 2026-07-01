package planning

import "testing"

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
