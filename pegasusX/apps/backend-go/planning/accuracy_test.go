package planning

import (
	"math"
	"testing"

	"cloud.google.com/go/civil"
)

func TestComputeSeriesMetricsWapeBias(t *testing.T) {
	asOf := civil.Date{Year: 2026, Month: 8, Day: 6}
	points := []SeriesPoint{
		{Day: asOf.AddDays(-2), ForecastQty: 10, ActualQty: 10},
		{Day: asOf.AddDays(-1), ForecastQty: 12, ActualQty: 10},
		{Day: asOf, ForecastQty: 8, ActualQty: 10},
	}
	m := ComputeSeriesMetrics(points, asOf)
	// abs errors: 0 + 2 + 2 = 4; actual sum = 30; WAPE = 4/30
	wantWape := 4.0 / 30.0
	if math.Abs(m.Wape7-wantWape) > 1e-9 {
		t.Fatalf("Wape7=%v want %v", m.Wape7, wantWape)
	}
	// signed: 0 + 2 + (-2) = 0
	if math.Abs(m.Bias7) > 1e-9 {
		t.Fatalf("Bias7=%v want 0", m.Bias7)
	}
	if m.SampleDays7 != 3 {
		t.Fatalf("SampleDays7=%d want 3", m.SampleDays7)
	}
	if m.AlertTs {
		t.Fatal("expected no TS alert on balanced series")
	}
}

func TestComputeSeriesMetricsTrackingSignalAlert(t *testing.T) {
	asOf := civil.Date{Year: 2026, Month: 8, Day: 10}
	points := make([]SeriesPoint, 0, 10)
	for i := 9; i >= 0; i-- {
		// Consistently over-forecast by 10 → large |TS|.
		points = append(points, SeriesPoint{
			Day: asOf.AddDays(-i), ForecastQty: 20, ActualQty: 10,
		})
	}
	m := ComputeSeriesMetrics(points, asOf)
	if math.Abs(m.TrackingSignal) <= 4 {
		t.Fatalf("TrackingSignal=%v want |TS|>4", m.TrackingSignal)
	}
	if !m.AlertTs {
		t.Fatal("expected AlertTs")
	}
	// bias = +1.0 (always +10 on actual 10)
	if math.Abs(m.Bias28-1.0) > 1e-9 {
		t.Fatalf("Bias28=%v want 1", m.Bias28)
	}
}

func TestConfidencePctFromWape(t *testing.T) {
	pct, ok := ConfidencePctFromWape(0.25, 7)
	if !ok || pct != 75 {
		t.Fatalf("got pct=%d ok=%v want 75 true", pct, ok)
	}
	if _, ok := ConfidencePctFromWape(0.1, 6); ok {
		t.Fatal("expected insufficient sample")
	}
	pct, ok = ConfidencePctFromWape(1.5, 10)
	if !ok || pct != 0 {
		t.Fatalf("got pct=%d ok=%v want 0 true", pct, ok)
	}
}
