package proximity

import (
	"testing"

	"github.com/uber/h3-go/v4"
)

func TestH3CellFromLatLng(t *testing.T) {
	if got := H3CellFromLatLng(0, 0); got != "" {
		t.Fatalf("zero coords = %q want empty", got)
	}
	got := H3CellFromLatLng(41.3111, 69.2797)
	if got == "" {
		t.Fatal("expected non-empty H3 cell for Tashkent coords")
	}
	if len(got) < 10 {
		t.Fatalf("unexpected H3 cell format: %q", got)
	}
}

func TestSettlementH3Cell_Res9(t *testing.T) {
	if got := SettlementH3Cell(0, 0); got != "" {
		t.Fatalf("zero coords = %q want empty", got)
	}
	got := SettlementH3Cell(41.3111, 69.2797)
	if got == "" {
		t.Fatal("expected non-empty settlement H3 cell")
	}
	res9 := H3CellRes9(41.3111, 69.2797)
	if got != res9 {
		t.Fatalf("SettlementH3Cell %q != H3CellRes9 %q", got, res9)
	}
	cell := h3.Cell(h3.IndexFromString(got))
	if !cell.IsValid() || cell.Resolution() != SettlementH3Resolution {
		t.Fatalf("got cell=%q res=%d, want res=%d", got, cell.Resolution(), SettlementH3Resolution)
	}
	matching := MatchingH3Cell(41.3111, 69.2797)
	if got == matching {
		t.Fatalf("settlement cell %q must not equal matching cell %q", got, matching)
	}
}

