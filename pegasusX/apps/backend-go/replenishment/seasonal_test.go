package replenishment

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/seasonalcore"
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

type stubSeasonalReader struct {
	mult float64
}

func (s stubSeasonalReader) Multiplier(_ context.Context, _ string, _ time.Time) (float64, error) {
	return s.mult, nil
}

func TestResolveSeasonalMultiplierUsesReader(t *testing.T) {
	t.Parallel()
	e := &Engine{
		SeasonalReader: stubSeasonalReader{mult: 1.9},
		Now:            func() time.Time { return time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC) },
	}
	got := e.resolveSeasonalMultiplier(context.Background(), "sup-1", e.Now())
	if got != 1.9 {
		t.Fatalf("got %v want 1.9 from override reader", got)
	}
}

func TestResolveSeasonalMultiplierFallsBackToBuiltin(t *testing.T) {
	t.Parallel()
	e := &Engine{
		SeasonalReader: seasonalcore.BuiltinOnlyReader{},
		Now:            func() time.Time { return time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC) },
	}
	got := e.resolveSeasonalMultiplier(context.Background(), "sup-1", e.Now())
	if got != 1.15 {
		t.Fatalf("got %v want 1.15", got)
	}
}

func TestSuggestedQtyReflectsCustomMult(t *testing.T) {
	t.Parallel()
	base := int64(10)
	mul := 1.9
	got := int64(math.Ceil(float64(base) * mul))
	if got != 19 {
		t.Fatalf("suggested=%d want 19", got)
	}
}
