package predictivepush

import (
	"math"
	"testing"
	"time"
)

func TestCompositeSignalProviderCollectEmpty(t *testing.T) {
	p := &CompositeSignalProvider{}
	out, err := p.Collect(nil, "sup-1", testDay())
	if err != nil {
		t.Fatal(err)
	}
	if out == nil {
		t.Fatal("expected non-nil slice")
	}
	if len(out) != 0 {
		t.Fatalf("expected empty slice from stub-free provider, got %d", len(out))
	}
}

func TestCalculateCrostonSBA_Empty(t *testing.T) {
	res := CalculateCrostonSBA(nil, 7, 0.15)
	if res.Category != "NO_DATA" || res.SuggestedQty != 1 || res.DemandRate != 0 {
		t.Fatalf("unexpected empty result: %+v", res)
	}
}

func TestCalculateCrostonSBA_ZeroDemand(t *testing.T) {
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	pts := []DemandPoint{
		{Date: base, Qty: 0},
		{Date: base.AddDate(0, 0, 1), Qty: 0},
		{Date: base.AddDate(0, 0, 2), Qty: -5},
	}
	res := CalculateCrostonSBA(pts, 7, 0.15)
	if res.Category != "ZERO_DEMAND" || res.SuggestedQty != 1 {
		t.Fatalf("unexpected zero demand result: %+v", res)
	}
}

func TestCalculateCrostonSBA_SingleObservation(t *testing.T) {
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	pts := []DemandPoint{
		{Date: base, Qty: 14},
	}
	res := CalculateCrostonSBA(pts, 7, 0.15)
	if res.Category != "EARLY_SIGNAL" || res.SuggestedQty != 14 {
		t.Fatalf("unexpected single obs result: %+v", res)
	}
	if math.Abs(res.DemandRate-2.0) > 1e-4 {
		t.Fatalf("expected daily demand rate 2.0, got %v", res.DemandRate)
	}
}

func TestCalculateCrostonSBA_Smooth(t *testing.T) {
	// Daily constant orders: low ADI (< 1.32) and low CV2 (< 0.49) -> SMOOTH
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	var pts []DemandPoint
	for i := 0; i < 10; i++ {
		pts = append(pts, DemandPoint{
			Date: base.AddDate(0, 0, i),
			Qty:  10,
		})
	}
	res := CalculateCrostonSBA(pts, 3, 0.15)
	if res.Category != "SMOOTH" {
		t.Fatalf("expected SMOOTH, got %q (ADI=%v, CV2=%v)", res.Category, res.ADI, res.CV2)
	}
	if res.ADI >= 1.32 {
		t.Fatalf("expected ADI < 1.32, got %v", res.ADI)
	}
	if res.CV2 >= 0.49 {
		t.Fatalf("expected CV2 < 0.49, got %v", res.CV2)
	}
	// For constant demand of 10 every 1 day, z=10, p=1, SBA rate = (1 - 0.15/2)*10 = 9.25
	// Horizon 3 days -> 9.25 * 3 = 27.75 -> round to 28
	if res.SuggestedQty != 28 {
		t.Fatalf("expected suggested qty 28, got %d (rate=%v)", res.SuggestedQty, res.DemandRate)
	}
}

func TestCalculateCrostonSBA_Intermittent(t *testing.T) {
	// Orders of 10 every 4 days: high ADI (>= 1.32), low CV2 (< 0.49) -> INTERMITTENT
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	var pts []DemandPoint
	for i := 0; i < 6; i++ {
		pts = append(pts, DemandPoint{
			Date: base.AddDate(0, 0, i*4),
			Qty:  10,
		})
	}
	res := CalculateCrostonSBA(pts, 4, 0.15)
	if res.Category != "INTERMITTENT" {
		t.Fatalf("expected INTERMITTENT, got %q (ADI=%v, CV2=%v)", res.Category, res.ADI, res.CV2)
	}
	if res.ADI < 1.32 {
		t.Fatalf("expected ADI >= 1.32, got %v", res.ADI)
	}
	if res.CV2 >= 0.49 {
		t.Fatalf("expected CV2 < 0.49, got %v", res.CV2)
	}
	// Interval p=4, z=10. SBA rate = (1 - 0.075) * (10/4) = 0.925 * 2.5 = 2.3125
	// Horizon 4 days -> 2.3125 * 4 = 9.25 -> round to 9
	if res.SuggestedQty != 9 {
		t.Fatalf("expected suggested qty 9, got %d (rate=%v)", res.SuggestedQty, res.DemandRate)
	}
}

func TestCalculateCrostonSBA_Erratic(t *testing.T) {
	// Daily orders with high quantity variance: low ADI (< 1.32), high CV2 (>= 0.49) -> ERRATIC
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	qtys := []int64{2, 50, 3, 60, 2, 45, 1, 55}
	var pts []DemandPoint
	for i, q := range qtys {
		pts = append(pts, DemandPoint{
			Date: base.AddDate(0, 0, i),
			Qty:  q,
		})
	}
	res := CalculateCrostonSBA(pts, 2, 0.15)
	if res.Category != "ERRATIC" {
		t.Fatalf("expected ERRATIC, got %q (ADI=%v, CV2=%v)", res.Category, res.ADI, res.CV2)
	}
	if res.ADI >= 1.32 {
		t.Fatalf("expected ADI < 1.32, got %v", res.ADI)
	}
	if res.CV2 < 0.49 {
		t.Fatalf("expected CV2 >= 0.49, got %v", res.CV2)
	}
}

func TestCalculateCrostonSBA_Lumpy(t *testing.T) {
	// Infrequent orders with high quantity variance: high ADI (>= 1.32), high CV2 (>= 0.49) -> LUMPY
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	qtys := []int64{5, 60, 4, 70, 6, 80}
	var pts []DemandPoint
	for i, q := range qtys {
		pts = append(pts, DemandPoint{
			Date: base.AddDate(0, 0, i*5),
			Qty:  q,
		})
	}
	res := CalculateCrostonSBA(pts, 5, 0.15)
	if res.Category != "LUMPY" {
		t.Fatalf("expected LUMPY, got %q (ADI=%v, CV2=%v)", res.Category, res.ADI, res.CV2)
	}
	if res.ADI < 1.32 {
		t.Fatalf("expected ADI >= 1.32, got %v", res.ADI)
	}
	if res.CV2 < 0.49 {
		t.Fatalf("expected CV2 >= 0.49, got %v", res.CV2)
	}
}

func testDay() time.Time {
	return time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
}
