package replenishment

import (
	"testing"
	"time"
)

func TestSeasonalMultiplierFor(t *testing.T) {
	t.Parallel()
	if got := seasonalMultiplierFor(time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)); got != 1.15 {
		t.Fatalf("summer=%v want 1.15", got)
	}
	if got := seasonalMultiplierFor(time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)); got != 1.35 {
		t.Fatalf("holiday=%v want 1.35", got)
	}
	if got := seasonalMultiplierFor(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)); got != 1.0 {
		t.Fatalf("off=%v want 1.0", got)
	}
}
