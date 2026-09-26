package replenishment

import (
	"context"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/seasonalcore"
)

// seasonalMultiplierFor returns the builtin calendar multiplier for on
// (no Spanner override). Prefer Engine.resolveSeasonalMultiplier when a
// supplier context and MultiplierReader are available.
func seasonalMultiplierFor(on time.Time) float64 {
	return seasonalcore.BuiltinMultiplierFor(on)
}

// resolveSeasonalMultiplier uses the engine reader when set, else builtins.
func (e *Engine) resolveSeasonalMultiplier(ctx context.Context, supplierID string, on time.Time) float64 {
	if e != nil && e.SeasonalReader != nil {
		if m, err := e.SeasonalReader.Multiplier(ctx, supplierID, on); err == nil && m > 0 {
			return m
		}
	}
	return seasonalMultiplierFor(on)
}
