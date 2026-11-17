package ar

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestScaleInt64Ratio_RemainderBoundAndNoOverflow(t *testing.T) {
	// Exact ratio
	got, err := ScaleInt64Ratio(10_000, 7_000, 10_000)
	if err != nil || got != 7_000 {
		t.Fatalf("got=%d err=%v, want 7000", got, err)
	}

	// Partition sum invariant with remainder rounding: 1/3 + 2/3 == full
	a, err := ScaleInt64Ratio(100, 1, 3)
	if err != nil || a != 33 {
		t.Fatalf("1/3 of 100 got=%d err=%v want 33", a, err)
	}
	b, err := ScaleInt64Ratio(100, 2, 3)
	if err != nil || b != 67 {
		t.Fatalf("2/3 of 100 got=%d err=%v want 67", b, err)
	}
	if a+b != 100 {
		t.Fatalf("a+b=%d want 100", a+b)
	}

	// Half-way tie rounds half-up cleanly without float64 precision loss
	half, err := ScaleInt64Ratio(1, 1, 2)
	if err != nil || half != 1 {
		t.Fatalf("1 * 1 / 2 = %d err=%v want 1", half, err)
	}

	// Large int64 amounts that would overflow naive int64*int64
	large := int64(5_000_000_000_000) // 5e12
	gotLarge, err := ScaleInt64Ratio(large, 3_000_000_000, 10_000_000_000)
	if err != nil || gotLarge != 1_500_000_000_000 {
		t.Fatalf("large scale got=%d err=%v", gotLarge, err)
	}

	// Overflow when result > MaxInt64
	if _, err := ScaleInt64Ratio(math.MaxInt64, 2, 1); err == nil {
		t.Fatal("expected overflow error")
	}

	// Invalid inputs rejected
	if _, err := ScaleInt64Ratio(-1, 1, 2); err == nil {
		t.Fatal("expected negative amt error")
	}
	if _, err := ScaleInt64Ratio(100, 1, 0); err == nil {
		t.Fatal("expected zero total error")
	}
}

func TestComputeInvoiceAdjustment_Scenarios(t *testing.T) {
	openInv := Invoice{
		InvoiceID:      "inv-1",
		SupplierID:     "sup-1",
		RetailerID:     "ret-1",
		OrderID:        "ord-1",
		Status:         StatusOpen,
		PrincipalMinor: 10_000,
		BalanceMinor:   10_000,
		Version:        1,
	}

	// 1. Open invoice partial short waive: 10000 -> 6000
	adj, changed, err := ComputeInvoiceAdjustment(openInv, 10_000, 6_000)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	if adj.NewPrincipalMinor != 6_000 || adj.NewBalanceMinor != 6_000 || adj.BalanceDeltaMinor != -4_000 || adj.NewStatus != StatusOpen {
		t.Fatalf("unexpected open adj: %+v", adj)
	}

	// 2. Open invoice full short waive: 10000 -> 0 => VOID with 0 balance
	adjZero, changed, err := ComputeInvoiceAdjustment(openInv, 10_000, 0)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	if adjZero.NewPrincipalMinor != 0 || adjZero.NewBalanceMinor != 0 || adjZero.BalanceDeltaMinor != -10_000 || adjZero.NewStatus != StatusVoid {
		t.Fatalf("unexpected zero adj: %+v", adjZero)
	}

	// 3. Partially paid invoice: principal 10000, paid 4000 (balance 6000); waive to 7000 => balance 3000 PARTIAL
	partialInv := openInv
	partialInv.Status = StatusPartial
	partialInv.BalanceMinor = 6_000
	adjPartial, changed, err := ComputeInvoiceAdjustment(partialInv, 10_000, 7_000)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	if adjPartial.NewPrincipalMinor != 7_000 || adjPartial.NewBalanceMinor != 3_000 || adjPartial.BalanceDeltaMinor != -3_000 || adjPartial.NewStatus != StatusPartial {
		t.Fatalf("unexpected partial adj: %+v", adjPartial)
	}

	// 4. Partially paid invoice where waive brings new principal below already-paid amount => balance clamped to 0, PAID
	heavyPaid := openInv
	heavyPaid.Status = StatusPartial
	heavyPaid.BalanceMinor = 2_000 // paid 8000
	adjSettled, changed, err := ComputeInvoiceAdjustment(heavyPaid, 10_000, 7_000)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	if adjSettled.NewPrincipalMinor != 7_000 || adjSettled.NewBalanceMinor != 0 || adjSettled.BalanceDeltaMinor != -2_000 || adjSettled.NewStatus != StatusPaid {
		t.Fatalf("unexpected settled adj: %+v", adjSettled)
	}

	// 5. Pro-rata financed fraction (e.g. 8000 principal on 10000 order reduced to 7500 => 6000)
	splitInv := openInv
	splitInv.PrincipalMinor = 8_000
	splitInv.BalanceMinor = 8_000
	adjSplit, changed, err := ComputeInvoiceAdjustment(splitInv, 10_000, 7_500)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	if adjSplit.NewPrincipalMinor != 6_000 || adjSplit.NewBalanceMinor != 6_000 || adjSplit.BalanceDeltaMinor != -2_000 {
		t.Fatalf("unexpected split adj: %+v", adjSplit)
	}

	// 6. Already PAID/VOID invoice does not drift
	paidInv := openInv
	paidInv.Status = StatusPaid
	paidInv.BalanceMinor = 0
	if _, changed, err := ComputeInvoiceAdjustment(paidInv, 10_000, 5_000); err != nil || changed {
		t.Fatalf("expected no change on PAID invoice, got changed=%v err=%v", changed, err)
	}

	// 7. Negative new order total fails
	if _, _, err := ComputeInvoiceAdjustment(openInv, 10_000, -1); err == nil {
		t.Fatal("expected error for negative newOrderTotalMinor")
	}
}

func TestMemoryRepository_AdjustByOrder_Idempotent(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Now().UTC()
	if err := repo.OpenInvoice(context.Background(), Invoice{
		InvoiceID:      "inv-mem-1",
		SupplierID:     "sup-1",
		RetailerID:     "ret-1",
		OrderID:        "ord-mem-1",
		Status:         StatusOpen,
		PrincipalMinor: 12_000,
		BalanceMinor:   12_000,
		Currency:       "UZS",
		CreditLeaveAt:  now,
		DueAt:          now.Add(7 * 24 * time.Hour),
		Version:        1,
	}); err != nil {
		t.Fatalf("open invoice: %v", err)
	}

	inv, changed, err := repo.AdjustByOrder(context.Background(), "ord-mem-1", 12_000, 8_000, "k1")
	if err != nil || !changed {
		t.Fatalf("first adjust: changed=%v err=%v", changed, err)
	}
	if inv.PrincipalMinor != 8_000 || inv.BalanceMinor != 8_000 || inv.Version != 2 {
		t.Fatalf("unexpected inv: %+v", inv)
	}

	// Repeat same key -> no-op
	_, changed, err = repo.AdjustByOrder(context.Background(), "ord-mem-1", 12_000, 8_000, "k1")
	if err != nil || changed {
		t.Fatalf("idempotent replay: changed=%v err=%v", changed, err)
	}
}
