package replenishment

import "testing"

func TestClassifyUrgency(t *testing.T) {
	tests := []struct {
		name     string
		tte      float64
		leadDays float64
		want     string
	}{
		{name: "critical at lead boundary", tte: 2.5, leadDays: 2, want: "CRITICAL"},
		{name: "warning between multipliers", tte: 3.5, leadDays: 2, want: "WARNING"},
		{name: "stable beyond warning", tte: 5.0, leadDays: 2, want: "STABLE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyUrgency(tt.tte, tt.leadDays); got != tt.want {
				t.Fatalf("classifyUrgency() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestComputeSuggestedQty(t *testing.T) {
	qty := computeSuggestedQty(skuStock{
		CurrentStock:    10,
		UnfulfilledQty:  5,
		InTransitQty:    2,
		DailyBurnRate:   4,
		FactoryLeadDays: 2,
	}, 12)
	if qty < 1 {
		t.Fatalf("expected positive suggested qty, got %d", qty)
	}
}
