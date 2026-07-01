package planning

import (
	"testing"
	"time"
)

func TestTemplateActiveOn_HolidayPeak(t *testing.T) {
	tpl := builtinSeasonalTemplates[0]
	if !templateActiveOn(tpl, time.Date(2026, 12, 20, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("expected holiday peak active in December")
	}
	if templateActiveOn(tpl, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("expected holiday peak inactive in March")
	}
}

func TestApplyConfidenceCap(t *testing.T) {
	result := SparsityResult{Allowed: true, ConfidenceCapPct: 60}
	if got := ApplyConfidenceCap(result, 85); got != 60 {
		t.Fatalf("expected cap 60, got %d", got)
	}
	blocked := SparsityResult{Allowed: false}
	if got := ApplyConfidenceCap(blocked, 85); got != 0 {
		t.Fatalf("expected 0 for blocked, got %d", got)
	}
}

func TestScenarioCacheKey(t *testing.T) {
	key := ScenarioCacheKey("sup-1", "8:10.0:7")
	if key != "planning:scenario:sup-1:8:10.0:7" {
		t.Fatalf("unexpected key %s", key)
	}
}
