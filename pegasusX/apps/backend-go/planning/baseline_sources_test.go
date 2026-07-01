package planning

import "testing"

func TestNormalizeBaselineSourceNeverML(t *testing.T) {
	if got := NormalizeBaselineSource("ml"); got != BaselineSourceMovingAverage {
		t.Fatalf("ml mapped to %q want moving_average", got)
	}
	if got := NormalizeBaselineSource("ai_recommendations"); got != BaselineSourceInventoryHint {
		t.Fatalf("ai_recommendations mapped to %q want inventory_hint", got)
	}
}

func TestNormalizeBaselineSourceSeasonal(t *testing.T) {
	if got := NormalizeBaselineSource("", "seasonal_template"); got != BaselineSourceSeasonalTemplate {
		t.Fatalf("got %q", got)
	}
}
