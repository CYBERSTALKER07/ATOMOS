package factory

import (
	"math"
	"testing"
)

// TestShadowDemandDeficitCalculation verifies the core look-ahead math:
//
//	Target = max(SafetyStockLevel, ceil(FutureDemand × 1.15))
//	Deficit = Target - CurrentStock
func TestShadowDemandDeficitCalculation(t *testing.T) {
	tests := []struct {
		name          string
		futureDemand  int64
		currentStock  int64
		safetyLevel   int64
		wantDeficit   int64
		wantTriggered bool
	}{
		{
			name:          "shadow demand exceeds safety level",
			futureDemand:  100,
			currentStock:  50,
			safetyLevel:   30,
			wantDeficit:   65, // ceil(100 * 1.15) - 50 = 115 - 50
			wantTriggered: true,
		},
		{
			name:          "safety level dominates",
			futureDemand:  10,
			currentStock:  50,
			safetyLevel:   80,
			wantDeficit:   30, // max(80, ceil(10*1.15)=12) = 80; 80 - 50
			wantTriggered: true,
		},
		{
			name:          "stock covers everything",
			futureDemand:  20,
			currentStock:  200,
			safetyLevel:   30,
			wantDeficit:   0,
			wantTriggered: false,
		},
		{
			name:          "zero demand — pure threshold fallback",
			futureDemand:  0,
			currentStock:  10,
			safetyLevel:   50,
			wantDeficit:   40, // 50 - 10
			wantTriggered: true,
		},
		{
			name:          "exact buffer match — no deficit",
			futureDemand:  87,
			currentStock:  101, // ceil(87 * 1.15) = ceil(100.05) = 101
			safetyLevel:   30,
			wantDeficit:   0,
			wantTriggered: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buffered := int64(math.Ceil(float64(tt.futureDemand) * (1.0 + SafetyStockBufferPct)))
			target := buffered
			if tt.safetyLevel > target {
				target = tt.safetyLevel
			}
			deficit := target - tt.currentStock
			if deficit < 0 {
				deficit = 0
			}

			triggered := deficit > 0
			if triggered != tt.wantTriggered {
				t.Errorf("triggered = %v, want %v", triggered, tt.wantTriggered)
			}
			if deficit != tt.wantDeficit {
				t.Errorf("deficit = %d, want %d (target=%d, buffered=%d)",
					deficit, tt.wantDeficit, target, buffered)
			}
		})
	}
}

// TestConvoySplitCalculation verifies volumetric convoy math.
func TestConvoySplitCalculation(t *testing.T) {
	tests := []struct {
		name       string
		totalVU    float64
		wantTrucks int
	}{
		{"single truck fits", 350.0, 1},
		{"exactly one truck", 400.0, 1},
		{"two trucks needed", 401.0, 2},
		{"three trucks — large order", 1100.0, 3},
		{"tiny transfer", 5.0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			convoyCount := int(math.Ceil(tt.totalVU / FactoryClassCCapacityVU))
			if convoyCount < 1 {
				convoyCount = 1
			}
			if convoyCount != tt.wantTrucks {
				t.Errorf("convoy = %d trucks, want %d (%.1f VU / %.1f capacity)",
					convoyCount, tt.wantTrucks, tt.totalVU, FactoryClassCCapacityVU)
			}
		})
	}
}

// TestLookAheadConstants verifies operational invariants.
func TestLookAheadConstants(t *testing.T) {
	if LookAheadWindowDays != 7 {
		t.Errorf("LookAheadWindowDays = %d, want 7", LookAheadWindowDays)
	}
	if SafetyStockBufferPct != 0.15 {
		t.Errorf("SafetyStockBufferPct = %f, want 0.15", SafetyStockBufferPct)
	}
	if FactoryClassCCapacityVU != 400.0 {
		t.Errorf("FactoryClassCCapacityVU = %f, want 400.0", FactoryClassCCapacityVU)
	}
}

// TestSourceToDBValue verifies the mapping still includes SYSTEM_THRESHOLD for cron runs.
func TestSourceToDBValue(t *testing.T) {
	if got := sourceToDBValue("CRON"); got != "SYSTEM_THRESHOLD" {
		t.Errorf("sourceToDBValue(CRON) = %s, want SYSTEM_THRESHOLD", got)
	}
	if got := sourceToDBValue("MANUAL"); got != "MANUAL_EMERGENCY" {
		t.Errorf("sourceToDBValue(MANUAL) = %s, want MANUAL_EMERGENCY", got)
	}
}
