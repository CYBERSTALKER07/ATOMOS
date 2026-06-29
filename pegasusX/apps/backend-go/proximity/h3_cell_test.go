package proximity

import "testing"

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
