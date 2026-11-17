package credit

import (
	"context"
	"testing"
)

func TestComputeCreditAdjustment_ReservedAndConverted(t *testing.T) {
	// 1. RESERVED reduced partially: headroom freed via ReservedMinor drop.
	plan, changed, err := computeCreditAdjustment(ReservationReserved, 50_000, 10_000, 20_000, 15_000, 9_000, "ord-1")
	if err != nil || !changed {
		t.Fatalf("reserved partial: changed=%v err=%v", changed, err)
	}
	if plan.NewBalanceMinor != 10_000 || plan.NewReservedMinor != 14_000 || plan.AvailableCreditMinor != 26_000 ||
		plan.NextReservationStatus != ReservationReserved || plan.AdjustInvoice {
		t.Fatalf("unexpected reserved partial plan: %+v", plan)
	}

	// 2. RESERVED reduced to 0: transitions to RELEASED.
	planZero, changed, err := computeCreditAdjustment(ReservationReserved, 50_000, 0, 15_000, 15_000, 0, "ord-2")
	if err != nil || !changed {
		t.Fatalf("reserved zero: changed=%v err=%v", changed, err)
	}
	if planZero.NewReservedMinor != 0 || planZero.AvailableCreditMinor != 50_000 || planZero.NextReservationStatus != ReservationReleased {
		t.Fatalf("unexpected reserved zero plan: %+v", planZero)
	}

	// 3. CONVERTED reduced partially: adjusts CurrentBalanceMinor, frees headroom, triggers AR invoice adjustment.
	convPlan, changed, err := computeCreditAdjustment(ReservationConverted, 50_000, 30_000, 5_000, 20_000, 12_000, "ord-3")
	if err != nil || !changed {
		t.Fatalf("converted partial: changed=%v err=%v", changed, err)
	}
	if convPlan.NewBalanceMinor != 22_000 || convPlan.NewReservedMinor != 5_000 || convPlan.AvailableCreditMinor != 23_000 ||
		convPlan.NextReservationStatus != ReservationConverted || !convPlan.AdjustInvoice {
		t.Fatalf("unexpected converted partial plan: %+v", convPlan)
	}

	// 4. CONVERTED reduced to 0: clears balance and marks reservation CLEARED + adjusts AR invoice.
	convZero, changed, err := computeCreditAdjustment(ReservationConverted, 50_000, 20_000, 0, 20_000, 0, "ord-4")
	if err != nil || !changed {
		t.Fatalf("converted zero: changed=%v err=%v", changed, err)
	}
	if convZero.NewBalanceMinor != 0 || convZero.AvailableCreditMinor != 50_000 ||
		convZero.NextReservationStatus != ReservationCleared || !convZero.AdjustInvoice {
		t.Fatalf("unexpected converted zero plan: %+v", convZero)
	}

	// 5. RELEASED/CLEARED are no-ops.
	for _, st := range []ReservationStatus{ReservationReleased, ReservationCleared} {
		if _, changed, err := computeCreditAdjustment(st, 50_000, 0, 0, 10_000, 5_000, "ord-x"); err != nil || changed {
			t.Fatalf("status %s expected noop, got changed=%v err=%v", st, changed, err)
		}
	}

	// 6. Negative amount fails cleanly.
	if _, _, err := computeCreditAdjustment(ReservationReserved, 50_000, 0, 10_000, 10_000, -1, "ord-neg"); err == nil {
		t.Fatal("expected error for negative reserve amount")
	}
}

func TestAdjustReserveInTxn_NilTxnErrors(t *testing.T) {
	repo := &SpannerRepository{}
	if err := repo.AdjustReserveInTxn(context.Background(), nil, "ord-1", 100); err == nil {
		t.Fatal("expected error for nil txn")
	}
}
