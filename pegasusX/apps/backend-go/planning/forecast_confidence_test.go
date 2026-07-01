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
	if out.BaselineSource != "moving_average" {
		t.Fatalf("source=%q want moving_average", out.BaselineSource)
	}
}

func TestMapBaselineSource(t *testing.T) {
	if got := mapBaselineSource("demand_forecast_baseline"); got != "moving_average" {
		t.Fatalf("got=%q", got)
	}
	if got := mapBaselineSource("", "seasonal_template"); got != "seasonal_template" {
		t.Fatalf("got=%q", got)
	}
}
