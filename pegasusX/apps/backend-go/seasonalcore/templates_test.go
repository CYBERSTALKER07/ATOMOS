package seasonalcore

import (
	"testing"
	"time"
)

func TestBuiltinMultiplierFor(t *testing.T) {
	t.Parallel()
	if got := BuiltinMultiplierFor(time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)); got != 1.15 {
		t.Fatalf("summer=%v want 1.15", got)
	}
	if got := BuiltinMultiplierFor(time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)); got != 1.35 {
		t.Fatalf("holiday=%v want 1.35", got)
	}
	if got := BuiltinMultiplierFor(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)); got != 1.0 {
		t.Fatalf("off=%v want 1.0", got)
	}
}

func TestResolveOverrideMultiplier(t *testing.T) {
	t.Parallel()
	explicit := 1.8
	if got := ResolveOverrideMultiplier(&explicit, "custom"); got != 1.8 {
		t.Fatalf("explicit=%v", got)
	}
	tooHigh := 9.0
	if got := ResolveOverrideMultiplier(&tooHigh, "custom"); got != MaxMultiplier {
		t.Fatalf("clamp high=%v", got)
	}
	if got := ResolveOverrideMultiplier(nil, "holiday_peak"); got != 1.35 {
		t.Fatalf("inherit holiday=%v", got)
	}
	if got := ResolveOverrideMultiplier(nil, "summer_surge"); got != 1.15 {
		t.Fatalf("inherit summer=%v", got)
	}
	if got := ResolveOverrideMultiplier(nil, "custom"); got != DefaultOverrideMultiplier {
		t.Fatalf("default=%v", got)
	}
}

func TestBuiltinByIDParity(t *testing.T) {
	t.Parallel()
	holiday, ok := BuiltinByID("holiday_peak")
	if !ok || holiday.Multiplier != 1.35 {
		t.Fatalf("holiday_peak missing or wrong: %+v", holiday)
	}
	summer, ok := BuiltinByID("summer_surge")
	if !ok || summer.Multiplier != 1.15 {
		t.Fatalf("summer_surge missing or wrong: %+v", summer)
	}
}
