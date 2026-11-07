package demand

import (
	"testing"
)

func TestDensityMultiplier(t *testing.T) {
	if got := densityMultiplier(5); got < 1.05 || got > 1.06 {
		t.Fatalf("min threshold mult = %v", got)
	}
	if got := densityMultiplier(100); got != 1.40 {
		t.Fatalf("clamp high = %v want 1.40", got)
	}
}

func TestParentH3_InvalidReturnsEmpty(t *testing.T) {
	if got := parentH3("not-a-cell", 7); got != "" {
		t.Fatalf("got %q", got)
	}
}
