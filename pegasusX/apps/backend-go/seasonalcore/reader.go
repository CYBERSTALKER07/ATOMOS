package seasonalcore

import (
	"context"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// MultiplierReader resolves the active calendar multiplier for a supplier/day
// (custom Spanner overrides win over builtins).
type MultiplierReader interface {
	Multiplier(ctx context.Context, supplierID string, on time.Time) (float64, error)
}

// SpannerOverrideReader loads active SeasonalTemplateOverrides.Multiplier.
type SpannerOverrideReader struct {
	Client *spanner.Client
}

// Multiplier composes all active override windows covering on deterministically
// and clamps to [MinMultiplier, MaxMultiplier], otherwise falls back to builtins.
func (r *SpannerOverrideReader) Multiplier(ctx context.Context, supplierID string, on time.Time) (float64, error) {
	if r == nil || r.Client == nil {
		return BuiltinMultiplierFor(on), nil
	}
	sid := strings.TrimSpace(supplierID)
	if sid == "" {
		return BuiltinMultiplierFor(on), nil
	}
	day := civil.DateOf(on.UTC())
	iter := r.Client.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, spanner.Statement{
		SQL: `SELECT COALESCE(Multiplier, 0), TemplateId
		      FROM SeasonalTemplateOverrides
		      WHERE SupplierId = @sid AND IsActive = true
		        AND StartDate <= @on AND EndDate >= @on
		      ORDER BY StartDate ASC, OverrideId ASC`,
		Params: map[string]any{"sid": sid, "on": day},
	})
	defer iter.Stop()
	var activeMults []float64
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return 0, err
		}
		var mult float64
		var templateID string
		if err := row.Columns(&mult, &templateID); err != nil {
			return 0, err
		}
		if mult > 0 {
			activeMults = append(activeMults, ClampMultiplier(mult))
		} else {
			activeMults = append(activeMults, ResolveOverrideMultiplier(nil, templateID))
		}
	}
	if len(activeMults) == 0 {
		return BuiltinMultiplierFor(on), nil
	}
	return ComposeMultipliers(activeMults), nil
}

// BuiltinOnlyReader never hits Spanner (tests / fallback).
type BuiltinOnlyReader struct{}

// Multiplier implements MultiplierReader using builtins only.
func (BuiltinOnlyReader) Multiplier(_ context.Context, _ string, on time.Time) (float64, error) {
	return BuiltinMultiplierFor(on), nil
}
