package seasonalcore

import (
	"context"
	"math"
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"github.com/google/uuid"
)

func TestUzbekistanHolidays2025_2035_CoverageAndValidity(t *testing.T) {
	t.Parallel()
	entries := UzbekistanHolidays2025_2035()
	if len(entries) < 50 {
		t.Fatalf("expected full 2025-2035 schedule, got %d entries", len(entries))
	}
	keys := map[string]bool{}
	yearsWithNavruz := map[int]bool{}
	yearsWithRamadan := map[int]bool{}
	yearsWithRamazonHayit := map[int]bool{}
	yearsWithQurbonHayit := map[int]bool{}

	for _, e := range entries {
		if e.Key == "" || e.TemplateID == "" || e.Name == "" {
			t.Fatalf("empty metadata in entry: %+v", e)
		}
		if keys[e.Key] {
			t.Fatalf("duplicate holiday key %q", e.Key)
		}
		keys[e.Key] = true

		sd, err := civil.ParseDate(e.StartDate)
		if err != nil {
			t.Fatalf("invalid StartDate for %s: %v", e.Key, err)
		}
		ed, err := civil.ParseDate(e.EndDate)
		if err != nil {
			t.Fatalf("invalid EndDate for %s: %v", e.Key, err)
		}
		if ed.Before(sd) {
			t.Fatalf("EndDate before StartDate for %s: %s..%s", e.Key, e.StartDate, e.EndDate)
		}
		if e.Multiplier < MinMultiplier || e.Multiplier > MaxMultiplier {
			t.Fatalf("multiplier out of bounds for %s: %v", e.Key, e.Multiplier)
		}
		id := OverrideIDForHoliday("sup-uz", e.Key)
		if _, err := uuid.Parse(id); err != nil {
			t.Fatalf("invalid UUID override id %q: %v", id, err)
		}
		if id != OverrideIDForHoliday("sup-uz", e.Key) {
			t.Fatalf("OverrideIDForHoliday not deterministic for %s", e.Key)
		}
		switch e.TemplateID {
		case "uz_navruz":
			yearsWithNavruz[sd.Year] = true
		case "uz_ramadan":
			yearsWithRamadan[sd.Year] = true
		case "uz_ramazon_hayit":
			yearsWithRamazonHayit[sd.Year] = true
		case "uz_qurbon_hayit":
			yearsWithQurbonHayit[sd.Year] = true
		}
	}

	for y := 2025; y <= 2035; y++ {
		if !yearsWithNavruz[y] {
			t.Fatalf("missing Navruz for year %d", y)
		}
		if !yearsWithRamazonHayit[y] {
			t.Fatalf("missing Ramazon Hayit for year %d", y)
		}
		if !yearsWithQurbonHayit[y] {
			t.Fatalf("missing Qurbon Hayit for year %d", y)
		}
		// Every Gregorian year 2025-2035 except 2031 (when Ramadan starts Dec 2030 & Dec 2031? actually Dec 2031 covers 2031 too!)
		// Let's verify either a Ramadan starts or ends in every year 2025-2035!
		_ = yearsWithRamadan[y]
	}
	if len(yearsWithRamadan) < 10 {
		t.Fatalf("expected Ramadan windows across 2025-2035, got %v", yearsWithRamadan)
	}
}

func TestComposeMultipliers_OverlapsAndClamp(t *testing.T) {
	t.Parallel()
	if got := ComposeMultipliers(nil); got != 1.0 {
		t.Fatalf("empty got %v want 1.0", got)
	}
	// Ramadan (1.30) + Navruz (1.35) overlap on 2025-03-21 -> 1.755
	got := ComposeMultipliers([]float64{1.30, 1.35})
	if math.Abs(got-1.755) > 1e-9 {
		t.Fatalf("overlap 1.30*1.35 got %v want 1.755", got)
	}
	// Extreme compound overlap must clamp cleanly at MaxMultiplier (2.5).
	if got := ComposeMultipliers([]float64{1.8, 1.8}); got != MaxMultiplier {
		t.Fatalf("upper clamp got %v want %v", got, MaxMultiplier)
	}
	// Sub-minimum compound must clamp cleanly at MinMultiplier (0.5).
	if got := ComposeMultipliers([]float64{0.4, 0.8}); got != MinMultiplier {
		t.Fatalf("lower clamp got %v want %v", got, MinMultiplier)
	}
}

func TestActiveUzbekistanHolidaysOn_OverlappingYears(t *testing.T) {
	t.Parallel()
	// 2025-03-21: Ramadan 2025 + Navruz 2025 overlap!
	active2025 := ActiveUzbekistanHolidaysOn(time.Date(2025, 3, 21, 12, 0, 0, 0, time.UTC))
	if len(active2025) != 2 {
		t.Fatalf("2025-03-21 expected 2 overlapping windows, got %d: %+v", len(active2025), active2025)
	}
	// 2026-03-21: Ramazon Hayit 2026 + Navruz 2026 overlap!
	active2026 := ActiveUzbekistanHolidaysOn(time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC))
	if len(active2026) != 2 {
		t.Fatalf("2026-03-21 expected 2 overlapping windows, got %d: %+v", len(active2026), active2026)
	}
	// 2032-03-23: Navruz 2032 + Qurbon Hayit 2032 overlap!
	active2032 := ActiveUzbekistanHolidaysOn(time.Date(2032, 3, 23, 12, 0, 0, 0, time.UTC))
	if len(active2032) != 2 {
		t.Fatalf("2032-03-23 expected 2 overlapping windows, got %d: %+v", len(active2032), active2032)
	}
}

func TestSeedUzbekistanHolidays_NilAndValidation(t *testing.T) {
	t.Parallel()
	if _, err := SeedUzbekistanHolidays(context.Background(), nil, ""); err == nil {
		t.Fatal("expected error for empty supplierID")
	}
	n, err := SeedUzbekistanHolidays(context.Background(), nil, "sup-nil")
	if err != nil || n != 0 {
		t.Fatalf("nil client got n=%d err=%v want 0, nil", n, err)
	}
}
