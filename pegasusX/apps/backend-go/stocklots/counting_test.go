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

func TestComputeCycleLotAdjustment_EnforcesReservedAndZeroBounds(t *testing.T) {
	// Valid partial deduction above reserved
	nextQoH, st, err := computeCycleLotAdjustment(10, 4, -3, "AVAILABLE")
	if err != nil || nextQoH != 7 || st != "AVAILABLE" {
		t.Fatalf("got qoh=%d st=%s err=%v, want 7/AVAILABLE", nextQoH, st, err)
	}

	// Depletes cleanly when reserved == 0 and delta == -qoh
	nextQoH, st, err = computeCycleLotAdjustment(5, 0, -5, "AVAILABLE")
	if err != nil || nextQoH != 0 || st != "DEPLETED" {
		t.Fatalf("got qoh=%d st=%s err=%v, want 0/DEPLETED", nextQoH, st, err)
	}

	// Rejects clamping below zero
	if _, _, err := computeCycleLotAdjustment(3, 0, -4, "AVAILABLE"); err == nil {
		t.Fatal("expected error when adjustment would go below 0")
	}

	// Rejects dropping QoH below QuantityReserved
	if _, _, err := computeCycleLotAdjustment(10, 6, -5, "AVAILABLE"); err == nil {
		t.Fatal("expected error when adjustment would go below QuantityReserved")
	}

	// Preserves QUARANTINED status on positive adjustment
	nextQoH, st, err = computeCycleLotAdjustment(4, 0, 2, "QUARANTINED")
	if err != nil || nextQoH != 6 || st != "QUARANTINED" {
		t.Fatalf("got qoh=%d st=%s err=%v, want 6/QUARANTINED", nextQoH, st, err)
	}
}
