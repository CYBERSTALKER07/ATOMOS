package stocklots

import (
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
)

func TestFilterShelfLife_rejectsShortLots(t *testing.T) {
	delivery := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	cands := []lotCandidate{
		{LotID: "a", Available: 5, Expiry: spanner.NullDate{Valid: true, Date: civil.Date{Year: 2026, Month: 8, Day: 12}}},
		{LotID: "b", Available: 5, Expiry: spanner.NullDate{Valid: true, Date: civil.Date{Year: 2026, Month: 9, Day: 1}}},
		{LotID: "c", Available: 5, Expiry: spanner.NullDate{Valid: false}},
	}
	out := filterShelfLife(cands, true, delivery, 7)
	if len(out) != 1 || out[0].LotID != "b" {
		t.Fatalf("expected only lot b, got %+v", out)
	}
}

func TestFilterShelfLife_nonPerishablePassthrough(t *testing.T) {
	cands := []lotCandidate{{LotID: "x", Available: 1}}
	out := filterShelfLife(cands, false, time.Now(), 30)
	if len(out) != 1 {
		t.Fatalf("expected passthrough, got %d", len(out))
	}
}

func TestLotsEnabledFlag(t *testing.T) {
	SetLotsEnabled(false)
	if LotsEnabled() {
		t.Fatal("expected disabled")
	}
	SetLotsEnabled(true)
	if !LotsEnabled() {
		t.Fatal("expected enabled")
	}
	SetLotsEnabled(false)
}
