package supplier

import (
	"testing"
	"time"
)

// ── dayKey ──────────────────────────────────────────────────────────────────

func TestDayKey_Monday(t *testing.T) {
	mon := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	if got := dayKey(mon); got != "mon" {
		t.Fatalf("dayKey(Monday) = %q, want %q", got, "mon")
	}
}

func TestDayKey_Sunday(t *testing.T) {
	sun := time.Date(2024, 1, 14, 12, 0, 0, 0, time.UTC)
	if got := dayKey(sun); got != "sun" {
		t.Fatalf("dayKey(Sunday) = %q, want %q", got, "sun")
	}
}

func TestDayKey_Wednesday(t *testing.T) {
	wed := time.Date(2024, 1, 17, 12, 0, 0, 0, time.UTC)
	if got := dayKey(wed); got != "wed" {
		t.Fatalf("dayKey(Wednesday) = %q, want %q", got, "wed")
	}
}

func TestDayKey_Saturday(t *testing.T) {
	sat := time.Date(2024, 1, 13, 12, 0, 0, 0, time.UTC)
	if got := dayKey(sat); got != "sat" {
		t.Fatalf("dayKey(Saturday) = %q, want %q", got, "sat")
	}
}

// ── parseHHMM ──────────────────────────────────────────────────────────────

func TestParseHHMM_Valid(t *testing.T) {
	ref := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	got, err := parseHHMM("09:30", ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Hour() != 9 || got.Minute() != 30 {
		t.Fatalf("expected 09:30, got %02d:%02d", got.Hour(), got.Minute())
	}
}

func TestParseHHMM_Midnight(t *testing.T) {
	ref := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	got, err := parseHHMM("00:00", ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Hour() != 0 || got.Minute() != 0 {
		t.Fatalf("expected 00:00, got %02d:%02d", got.Hour(), got.Minute())
	}
}

func TestParseHHMM_PreservesDate(t *testing.T) {
	ref := time.Date(2024, 3, 20, 15, 45, 0, 0, time.UTC)
	got, err := parseHHMM("10:00", ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Day() != 20 || got.Month() != 3 || got.Year() != 2024 {
		t.Fatalf("date should match ref; got %v", got)
	}
}

func TestParseHHMM_PreservesLocation(t *testing.T) {
	loc := time.FixedZone("UTC+5", 5*60*60)
	ref := time.Date(2024, 1, 15, 12, 0, 0, 0, loc)
	got, err := parseHHMM("14:30", ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Location().String() != loc.String() {
		t.Fatalf("location should match ref; got %v", got.Location())
	}
}

func TestParseHHMM_InvalidNoColon(t *testing.T) {
	ref := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	_, err := parseHHMM("0930", ref)
	if err == nil {
		t.Fatal("expected error for missing colon")
	}
}

func TestParseHHMM_InvalidLetters(t *testing.T) {
	ref := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	_, err := parseHHMM("ab:cd", ref)
	if err == nil {
		t.Fatal("expected error for non-numeric input")
	}
}

func TestParseHHMM_Empty(t *testing.T) {
	ref := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	_, err := parseHHMM("", ref)
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

// ── resolveIsActive ─────────────────────────────────────────────────────────

func TestResolveIsActive_ManualOff_AlwaysFalse(t *testing.T) {
	schedule := `{"mon":{"open":"00:00","close":"23:59"}}`
	mon := time.Date(2024, 1, 15, 7, 0, 0, 0, time.UTC) // 12:00 Tashkent Monday
	if resolveIsActive(schedule, true, mon) {
		t.Fatal("manualOff=true should always return false")
	}
}

func TestResolveIsActive_EmptySchedule_AlwaysOpen(t *testing.T) {
	if !resolveIsActive("", false, time.Now()) {
		t.Fatal("empty schedule should mean always open")
	}
}

func TestResolveIsActive_BareObject_AlwaysOpen(t *testing.T) {
	if !resolveIsActive("{}", false, time.Now()) {
		t.Fatal("empty object should mean always open")
	}
}

func TestResolveIsActive_NullString_AlwaysOpen(t *testing.T) {
	if !resolveIsActive("null", false, time.Now()) {
		t.Fatal("null string should mean always open")
	}
}

func TestResolveIsActive_MalformedJSON_FailOpen(t *testing.T) {
	if !resolveIsActive("{broken", false, time.Now()) {
		t.Fatal("malformed JSON should fail open (return true)")
	}
}

func TestResolveIsActive_WithinWindow(t *testing.T) {
	// Monday 09:00-18:00 Tashkent. Now = Monday 07:00 UTC = 12:00 Tashkent
	schedule := `{"mon":{"open":"09:00","close":"18:00"}}`
	now := time.Date(2024, 1, 15, 7, 0, 0, 0, time.UTC)
	if !resolveIsActive(schedule, false, now) {
		t.Fatal("12:00 Tashkent should be within 09:00-18:00")
	}
}

func TestResolveIsActive_OutsideWindow(t *testing.T) {
	// Monday 09:00-18:00 Tashkent. Now = Monday 14:00 UTC = 19:00 Tashkent
	schedule := `{"mon":{"open":"09:00","close":"18:00"}}`
	now := time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC)
	if resolveIsActive(schedule, false, now) {
		t.Fatal("19:00 Tashkent should be outside 09:00-18:00")
	}
}

func TestResolveIsActive_DayNotInSchedule_Closed(t *testing.T) {
	// Schedule only has "mon", today is Tuesday
	schedule := `{"mon":{"open":"09:00","close":"18:00"}}`
	tue := time.Date(2024, 1, 16, 7, 0, 0, 0, time.UTC) // Tuesday 12:00 Tashkent
	if resolveIsActive(schedule, false, tue) {
		t.Fatal("Tuesday not in schedule should return false")
	}
}

func TestResolveIsActive_ExactOpenTime_Active(t *testing.T) {
	// !local.Before(openTime) is true when equal
	schedule := `{"mon":{"open":"12:00","close":"18:00"}}`
	now := time.Date(2024, 1, 15, 7, 0, 0, 0, time.UTC) // 12:00 Tashkent
	if !resolveIsActive(schedule, false, now) {
		t.Fatal("exactly at open time should be active")
	}
}

func TestResolveIsActive_ExactCloseTime_Inactive(t *testing.T) {
	// local.Before(closeTime) is false when equal → inactive
	schedule := `{"mon":{"open":"09:00","close":"12:00"}}`
	now := time.Date(2024, 1, 15, 7, 0, 0, 0, time.UTC) // 12:00 Tashkent
	if resolveIsActive(schedule, false, now) {
		t.Fatal("exactly at close time should be inactive")
	}
}

func TestResolveIsActive_MultipleDays(t *testing.T) {
	schedule := `{"mon":{"open":"09:00","close":"18:00"},"tue":{"open":"10:00","close":"20:00"}}`
	// Tuesday 08:00 UTC = 13:00 Tashkent → within tue window 10:00-20:00
	tue := time.Date(2024, 1, 16, 8, 0, 0, 0, time.UTC)
	if !resolveIsActive(schedule, false, tue) {
		t.Fatal("13:00 Tashkent should be within tue 10:00-20:00")
	}
}

func TestResolveIsActive_WhitespaceSchedule_AlwaysOpen(t *testing.T) {
	if !resolveIsActive("   ", false, time.Now()) {
		t.Fatal("whitespace-only schedule should mean always open")
	}
}
