package demand

import (
	"testing"
)

func TestCalculateWeatherMultiplier(t *testing.T) {
	tests := []struct {
		name     string
		tempC    float64
		precipMm float64
		want     float64
	}{
		{
			name:     "Normal weather",
			tempC:    25.0,
			precipMm: 0.0,
			want:     1.0,
		},
		{
			name:     "Heat wave (35C) - +10% demand",
			tempC:    35.0,
			precipMm: 0.0,
			want:     1.10,
		},
		{
			name:     "Freezing (-5C) - +20% demand",
			tempC:    -5.0,
			precipMm: 0.0,
			want:     1.20, // 5.0 - (-5.0) = 10 * 0.02 = +0.20
		},
		{
			name:     "Heavy rain (20mm) - -20% demand",
			tempC:    20.0,
			precipMm: 20.0,
			want:     0.80, // 1.0 - 0.20
		},
		{
			name:     "Extreme heat + rain - clamped",
			tempC:    45.0, // +0.30
			precipMm: 0.0,
			want:     1.30, // max is 1.30
		},
		{
			name:     "Extreme cold - clamped",
			tempC:    -20.0, // +0.50 -> 1.50 -> clamp 1.30
			precipMm: 0.0,
			want:     1.30,
		},
		{
			name:     "Extreme rain - clamped",
			tempC:    25.0,
			precipMm: 100.0, // -1.0 -> 0.0 -> clamp 0.70
			want:     0.70,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateWeatherMultiplier(tt.tempC, tt.precipMm)
			// check float equality with small tolerance
			if mathAbs(got-tt.want) > 1e-9 {
				t.Errorf("calculateWeatherMultiplier() = %v, want %v", got, tt.want)
			}
		})
	}
}

func mathAbs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
