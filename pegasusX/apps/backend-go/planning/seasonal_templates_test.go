package planning

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/seasonalcore"
)

func TestActiveSeasonalTemplateBuiltinUsesStoredMultiplier(t *testing.T) {
	t.Parallel()
	svc := &Service{}
	tpl, src, err := svc.ActiveSeasonalTemplate(context.Background(), "sup-1", time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if src != "seasonal_template" || tpl == nil || tpl.Multiplier != 1.15 {
		t.Fatalf("got tpl=%+v src=%s", tpl, src)
	}
}

func TestResolveOverrideMultiplierInherit(t *testing.T) {
	t.Parallel()
	if got := seasonalcore.ResolveOverrideMultiplier(nil, "holiday_peak"); got != 1.35 {
		t.Fatalf("got %v", got)
	}
	explicit := 2.0
	if got := seasonalcore.ResolveOverrideMultiplier(&explicit, "holiday_peak"); got != 2.0 {
		t.Fatalf("explicit override lost: %v", got)
	}
}

func TestBuiltinSeasonalTemplatesFromCore(t *testing.T) {
	t.Parallel()
	tpls := BuiltinSeasonalTemplates()
	if len(tpls) != len(seasonalcore.Builtins) {
		t.Fatalf("len=%d", len(tpls))
	}
	for i, tpl := range tpls {
		if tpl.ID != seasonalcore.Builtins[i].ID || tpl.Multiplier != seasonalcore.Builtins[i].Multiplier {
			t.Fatalf("mismatch at %d: %+v vs %+v", i, tpl, seasonalcore.Builtins[i])
		}
	}
}

func TestComposeSeasonalOverrides_DeterministicOverlapAndClamp(t *testing.T) {
	t.Parallel()
	if got := ComposeSeasonalOverrides(nil); got != nil {
		t.Fatalf("expected nil for empty overrides, got %+v", got)
	}

	// Single override preserves exact custom:<id> contract.
	single := ComposeSeasonalOverrides([]SeasonalOverrideRow{
		{OverrideID: "ov-1", SupplierID: "sup-1", TemplateID: "uz_navruz", Name: "Navruz 2026", StartDate: "2026-03-20", EndDate: "2026-03-24", IsActive: true, Multiplier: 1.35},
	})
	if single == nil || single.ID != "custom:ov-1" || single.Multiplier != 1.35 {
		t.Fatalf("single override mismatch: %+v", single)
	}

	// Overlapping Ramadan Hayit (1.45) + Navruz (1.35) supplied out of order -> sorted deterministically & both composed (1.45 * 1.35 = 1.9575).
	overlapA := ComposeSeasonalOverrides([]SeasonalOverrideRow{
		{OverrideID: "ov-navruz", SupplierID: "sup-1", TemplateID: "uz_navruz", Name: "Navruz 2026", StartDate: "2026-03-20", EndDate: "2026-03-24", IsActive: true, Multiplier: 1.35},
		{OverrideID: "ov-hayit", SupplierID: "sup-1", TemplateID: "uz_ramazon_hayit", Name: "Ramazon Hayit 2026", StartDate: "2026-03-20", EndDate: "2026-03-22", IsActive: true, Multiplier: 1.45},
	})
	overlapB := ComposeSeasonalOverrides([]SeasonalOverrideRow{
		{OverrideID: "ov-hayit", SupplierID: "sup-1", TemplateID: "uz_ramazon_hayit", Name: "Ramazon Hayit 2026", StartDate: "2026-03-20", EndDate: "2026-03-22", IsActive: true, Multiplier: 1.45},
		{OverrideID: "ov-navruz", SupplierID: "sup-1", TemplateID: "uz_navruz", Name: "Navruz 2026", StartDate: "2026-03-20", EndDate: "2026-03-24", IsActive: true, Multiplier: 1.35},
	})
	if overlapA == nil || overlapB == nil {
		t.Fatal("expected non-nil composed override")
	}
	if overlapA.ID != overlapB.ID || overlapA.Name != overlapB.Name {
		t.Fatalf("composition order not deterministic: A=%+v B=%+v", overlapA, overlapB)
	}
	if math.Abs(overlapA.Multiplier-1.9575) > 1e-9 {
		t.Fatalf("composed overlap got %v want 1.9575", overlapA.Multiplier)
	}

	// Extreme overlapping multipliers must clamp to MaxMultiplier (2.5).
	clamped := ComposeSeasonalOverrides([]SeasonalOverrideRow{
		{OverrideID: "ov-a", TemplateID: "custom", StartDate: "2027-03-20", EndDate: "2027-03-24", Multiplier: 1.9},
		{OverrideID: "ov-b", TemplateID: "custom", StartDate: "2027-03-21", EndDate: "2027-03-22", Multiplier: 1.8},
	})
	if clamped == nil || clamped.Multiplier != seasonalcore.MaxMultiplier {
		t.Fatalf("expected clamp to %v, got %+v", seasonalcore.MaxMultiplier, clamped)
	}
}
