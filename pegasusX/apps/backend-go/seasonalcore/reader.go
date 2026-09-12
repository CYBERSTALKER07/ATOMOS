package seasonalcore

import (
	"context"
	"errors"
	"strings"
	"time"

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

// Multiplier returns the override multiplier if an active window covers on,
// otherwise the matching builtin, otherwise 1.0.
func (r *SpannerOverrideReader) Multiplier(ctx context.Context, supplierID string, on time.Time) (float64, error) {
	if r == nil || r.Client == nil {
		return BuiltinMultiplierFor(on), nil
	}
	sid := strings.TrimSpace(supplierID)
	if sid == "" {
		return BuiltinMultiplierFor(on), nil
	}
	on = on.UTC()
	iter := r.Client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT COALESCE(Multiplier, 0), TemplateId
		      FROM SeasonalTemplateOverrides
		      WHERE SupplierId = @sid AND IsActive = true
		        AND StartDate <= @on AND EndDate >= @on
		      ORDER BY StartDate DESC LIMIT 1`,
		Params: map[string]any{"sid": sid, "on": on},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return BuiltinMultiplierFor(on), nil
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
		return ClampMultiplier(mult), nil
	}
	return ResolveOverrideMultiplier(nil, templateID), nil
}

// BuiltinOnlyReader never hits Spanner (tests / fallback).
type BuiltinOnlyReader struct{}

// Multiplier implements MultiplierReader using builtins only.
func (BuiltinOnlyReader) Multiplier(_ context.Context, _ string, on time.Time) (float64, error) {
	return BuiltinMultiplierFor(on), nil
}
