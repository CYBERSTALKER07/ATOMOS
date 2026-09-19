package planning

import (
	"context"
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
