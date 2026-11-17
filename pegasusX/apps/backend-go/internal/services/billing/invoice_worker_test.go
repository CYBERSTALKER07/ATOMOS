package billing

import (
	"context"
	"testing"
	"time"
)

func TestInvoiceWorker_RecordOrderRefunded_Guards(t *testing.T) {
	ctx := context.Background()

	// Nil worker
	var nilWorker *InvoiceWorker
	if err := nilWorker.RecordOrderRefunded(ctx, "sup-1", "ord-1", 100, time.Time{}); err == nil {
		t.Fatal("expected error on nil worker")
	}

	// Nil client
	worker := NewInvoiceWorker(nil, nil, nil, testDiscardLogger())
	if err := worker.RecordOrderRefunded(ctx, "sup-1", "ord-1", 100, time.Time{}); err == nil {
		t.Fatal("expected error on nil client")
	}

	// Empty supplier / order ID / non-positive refund returns nil (no-op)
	if err := worker.RecordOrderRefunded(ctx, "", "ord-1", 100, time.Time{}); err != nil {
		t.Fatalf("expected nil on empty supplier, got %v", err)
	}
	if err := worker.RecordOrderRefunded(ctx, "sup-1", "", 100, time.Time{}); err != nil {
		t.Fatalf("expected nil on empty order ID, got %v", err)
	}
	if err := worker.RecordOrderRefunded(ctx, "sup-1", "ord-1", 0, time.Time{}); err != nil {
		t.Fatalf("expected nil on zero refund, got %v", err)
	}
	if err := worker.RecordOrderRefunded(ctx, "sup-1", "ord-1", -50, time.Time{}); err != nil {
		t.Fatalf("expected nil on negative refund, got %v", err)
	}
}

func TestInvoiceWorker_RecordOrderCompletedWithCommission_Guards(t *testing.T) {
	ctx := context.Background()

	var nilWorker *InvoiceWorker
	if err := nilWorker.RecordOrderCompletedWithCommission(ctx, "sup-1", "ord-1", 1000, 50, time.Time{}); err == nil {
		t.Fatal("expected error on nil worker")
	}

	worker := NewInvoiceWorker(nil, nil, nil, testDiscardLogger())
	if err := worker.RecordOrderCompletedWithCommission(ctx, "sup-1", "ord-1", 1000, 50, time.Time{}); err == nil {
		t.Fatal("expected error on nil client")
	}

	// Empty supplier / order ID returns nil
	if err := worker.RecordOrderCompletedWithCommission(ctx, "", "ord-1", 1000, 50, time.Time{}); err != nil {
		t.Fatalf("expected nil on empty supplier, got %v", err)
	}
	if err := worker.RecordOrderCompletedWithCommission(ctx, "sup-1", "", 1000, 50, time.Time{}); err != nil {
		t.Fatalf("expected nil on empty order ID, got %v", err)
	}
}

func TestFeeSchedule_MonthlyPlatformFee(t *testing.T) {
	s := FeeSchedule{
		PerOrderMinor:            250,
		GmvBps:                   175,
		MonthlySubscriptionMinor: 50000,
	}

	// 0 orders
	if got := s.MonthlyPlatformFee(0); got != 50000 {
		t.Fatalf("0 orders fee = %d, want 50000", got)
	}

	// 40 orders
	// 40 * 250 + 50000 = 10000 + 50000 = 60000
	if got := s.MonthlyPlatformFee(40); got != 60000 {
		t.Fatalf("40 orders fee = %d, want 60000", got)
	}
}

func TestComputeMonthlyBillingFee_ExcessSettledCommissionCredited(t *testing.T) {
	s := FeeSchedule{
		PerOrderMinor:            250,
		GmvBps:                   100, // 1%
		MonthlySubscriptionMinor: 50000,
	}
	// orderCount = 40 -> MonthlyPlatformFee = 40 * 250 + 50000 = 60000 minor
	// gmvMinor = 1,000,000 -> totalGmvFee = 1,000,000 * 100 / 10000 = 10000 minor

	// Case 1: settledCommission = 15,000 > totalGmvFee (10,000)
	// excess = 5,000
	// fee = platformFee (60,000) - excess (5,000) = 55,000
	fee := computeMonthlyBillingFee(s, 40, 1_000_000, 15_000)
	if fee != 55000 {
		t.Fatalf("expected 55,000 with excess commission credit, got %d", fee)
	}

	// Case 2: settledCommission = 80,000 (excess 70,000 exceeds platform fee 60,000)
	// fee should be clamped at 0
	fee = computeMonthlyBillingFee(s, 40, 1_000_000, 80_000)
	if fee != 0 {
		t.Fatalf("expected 0 when excess commission exceeds platform fee, got %d", fee)
	}

	// Case 3: settledCommission = 6,000 < totalGmvFee (10,000)
	// uncollected = 4,000
	// fee = platformFee (60,000) + uncollected (4,000) = 64,000
	fee = computeMonthlyBillingFee(s, 40, 1_000_000, 6_000)
	if fee != 64000 {
		t.Fatalf("expected 64,000 when commission uncollected, got %d", fee)
	}

	// Case 4: settledCommission = 0
	// uses MonthlyFee = platformFee (60,000) + totalGmvFee (10,000) = 70,000
	fee = computeMonthlyBillingFee(s, 40, 1_000_000, 0)
	if fee != 70000 {
		t.Fatalf("expected 70,000 when 0 settled commission, got %d", fee)
	}
}
