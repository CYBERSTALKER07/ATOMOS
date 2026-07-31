package replenishment

import "testing"

func TestComputeEchelonTarget(t *testing.T) {
	target, safety := ComputeEchelonTarget(10, 14, 9800)
	if target <= 0 {
		t.Fatalf("expected positive target, got %d", target)
	}
	if safety <= 0 {
		t.Fatalf("expected positive safety, got %d", safety)
	}
}

func TestSuggestedQtyFromTarget(t *testing.T) {
	if qty := SuggestedQtyFromTarget(100, 40); qty != 60 {
		t.Fatalf("expected 60, got %d", qty)
	}
	if qty := SuggestedQtyFromTarget(50, 80); qty != 0 {
		t.Fatalf("expected 0, got %d", qty)
	}
}
