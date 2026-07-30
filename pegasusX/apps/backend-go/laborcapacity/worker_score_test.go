package laborcapacity

import (
	"math"
	"testing"
)

func TestComputeScore_Perfect(t *testing.T) {
	score := computeScore(1.0, 1.0, 0.0, 0.0, 1.0)
	if score != 100.0 {
		t.Errorf("perfect score = %v, want 100", score)
	}
}

func TestComputeScore_Worst(t *testing.T) {
	score := computeScore(0.0, 0.0, 1.0, 1.0, 0.0)
	if score != 0.0 {
		t.Errorf("worst score = %v, want 0", score)
	}
}

func TestComputeScore_AllDefaults(t *testing.T) {
	// New driver with no data: onTime=0, completion=0, damage=0, shopClosed=0, feedback=0.5
	score := computeScore(0.0, 0.0, 0.0, 0.0, 0.5)
	// 0.35*0 + 0.25*0 + 0.20*1 + 0.10*1 + 0.10*0.5 = 0.35 → 35
	expected := 35.0
	if math.Abs(score-expected) > 0.01 {
		t.Errorf("defaults score = %v, want %v", score, expected)
	}
}

func TestComputeScore_GoodDriver(t *testing.T) {
	score := computeScore(0.95, 0.98, 0.02, 0.05, 0.8)
	// 0.35*0.95 + 0.25*0.98 + 0.20*0.98 + 0.10*0.95 + 0.10*0.8
	// = 0.3325 + 0.245 + 0.196 + 0.095 + 0.08 = 0.9485 → 94.85
	expected := 94.85
	if math.Abs(score-expected) > 0.1 {
		t.Errorf("good driver score = %v, want ~%v", score, expected)
	}
}

func TestComputeScore_Clamped(t *testing.T) {
	// Should not exceed 100 even with weird inputs
	score := computeScore(1.5, 1.5, -0.5, -0.5, 1.5)
	if score > 100 {
		t.Errorf("score %v exceeds 100", score)
	}
}

func TestComputeScore_Formula(t *testing.T) {
	tests := []struct {
		name      string
		onTime    float64
		complete  float64
		damage    float64
		shopClose float64
		feedback  float64
		want      float64
	}{
		{"mid_range", 0.5, 0.5, 0.5, 0.5, 0.5, 50.0},
		{"high_ontime", 1.0, 0.5, 0.0, 0.0, 0.5, 72.5},
		{"zero_feedback", 0.8, 0.8, 0.1, 0.1, 0.0, 65.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeScore(tt.onTime, tt.complete, tt.damage, tt.shopClose, tt.feedback)
			if math.Abs(got-tt.want) > 0.5 {
				t.Errorf("computeScore(%v,%v,%v,%v,%v) = %v, want ~%v",
					tt.onTime, tt.complete, tt.damage, tt.shopClose, tt.feedback, got, tt.want)
			}
		})
	}
}
