package stocklots

import "testing"

func TestVarianceQty(t *testing.T) {
	if varianceQty(10, 8) != -2 {
		t.Fatalf("short variance")
	}
	if varianceQty(10, 12) != 2 {
		t.Fatalf("over variance")
	}
	if varianceQty(5, 5) != 0 {
		t.Fatalf("zero variance")
	}
}

func TestCycleCountsEnabledFlag(t *testing.T) {
	SetCycleCountsEnabled(false)
	if CycleCountsEnabled() {
		t.Fatal("expected disabled")
	}
	SetCycleCountsEnabled(true)
	if !CycleCountsEnabled() {
		t.Fatal("expected enabled")
	}
	SetCycleCountsEnabled(false)
}
