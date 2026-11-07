package proximity

import (
	"testing"
	"time"
)

func TestPackLocation_UZ(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	loc := PackLocation()
	if loc == nil || loc.String() != "Asia/Tashkent" {
		t.Fatalf("loc=%v", loc)
	}
}

func TestPackLocation_PlannedNil(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "EU")
	if PackLocation() != nil {
		t.Fatal("planned pack must not expose a Tashkent location")
	}
	if !PackTodayStart(time.Now()).IsZero() {
		t.Fatal("planned pack today start must be zero")
	}
}
