package driver

import (
	"context"
	"testing"
	"time"
)

func TestCashBagVariance_Shortfall(t *testing.T) {
	svc := &Service{
		pendingQuery: func(driverID string) []PendingCollection {
			return []PendingCollection{
				{OrderID: "ord-1", Amount: 50_000, State: "PENDING"},
				{OrderID: "ord-2", Amount: 30_000, State: "PENDING"},
			}
		},
		now: func() time.Time { return time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC) },
	}
	summary, err := svc.GetCashBagSummary(context.Background(), "driver-shortfall")
	if err != nil {
		t.Fatal(err)
	}
	if summary.ExpectedCashMinor != 80_000 {
		t.Fatalf("expected=%d want 80000", summary.ExpectedCashMinor)
	}
	// Declared 75000 vs expected 80000 = shortfall of -5000
	diff := int64(75_000) - summary.ExpectedCashMinor
	if diff >= 0 {
		t.Fatalf("diff=%d should be negative for shortfall", diff)
	}
	if diff != -5_000 {
		t.Fatalf("diff=%d want -5000", diff)
	}
}

func TestCashBagVariance_Overage(t *testing.T) {
	svc := &Service{
		pendingQuery: func(driverID string) []PendingCollection {
			return []PendingCollection{
				{OrderID: "ord-1", Amount: 50_000, State: "PENDING"},
			}
		},
		now: func() time.Time { return time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC) },
	}
	summary, err := svc.GetCashBagSummary(context.Background(), "driver-overage")
	if err != nil {
		t.Fatal(err)
	}
	// Declared 60000 vs expected 50000 = +10000 overage
	diff := int64(60_000) - summary.ExpectedCashMinor
	if diff <= 0 {
		t.Fatalf("diff=%d should be positive for overage", diff)
	}
	if diff != 10_000 {
		t.Fatalf("diff=%d want 10000", diff)
	}
}

func TestCashBagVariance_Balanced(t *testing.T) {
	svc := &Service{
		pendingQuery: func(driverID string) []PendingCollection {
			return []PendingCollection{
				{OrderID: "ord-1", Amount: 100_000, State: "PENDING"},
			}
		},
		now: func() time.Time { return time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC) },
	}
	summary, err := svc.GetCashBagSummary(context.Background(), "driver-balanced")
	if err != nil {
		t.Fatal(err)
	}
	if summary.ExpectedCashMinor != 100_000 {
		t.Fatalf("expected=%d want 100000", summary.ExpectedCashMinor)
	}
	diff := int64(100_000) - summary.ExpectedCashMinor
	if diff != 0 {
		t.Fatalf("diff=%d should be 0 for balanced", diff)
	}
}

func TestCashBagVariance_NoPending(t *testing.T) {
	svc := &Service{
		pendingQuery: func(driverID string) []PendingCollection {
			return nil
		},
		now: func() time.Time { return time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC) },
	}
	summary, err := svc.GetCashBagSummary(context.Background(), "driver-none")
	if err != nil {
		t.Fatal(err)
	}
	if summary.ExpectedCashMinor != 0 {
		t.Fatalf("expected=%d want 0", summary.ExpectedCashMinor)
	}
	if summary.ReconciliationStatus != "PENDING_TURN_IN" {
		t.Fatalf("status=%s want PENDING_TURN_IN", summary.ReconciliationStatus)
	}
}

func TestCashBagVariance_MultipleOrders(t *testing.T) {
	svc := &Service{
		pendingQuery: func(driverID string) []PendingCollection {
			return []PendingCollection{
				{OrderID: "ord-1", Amount: 10_000, State: "PENDING"},
				{OrderID: "ord-2", Amount: 20_000, State: "PENDING"},
				{OrderID: "ord-3", Amount: 30_000, State: "PENDING"},
				{OrderID: "ord-4", Amount: 40_000, State: "PENDING"},
			}
		},
		now: func() time.Time { return time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC) },
	}
	summary, err := svc.GetCashBagSummary(context.Background(), "driver-multi")
	if err != nil {
		t.Fatal(err)
	}
	if summary.ExpectedCashMinor != 100_000 {
		t.Fatalf("expected=%d want 100000", summary.ExpectedCashMinor)
	}
	if len(summary.PendingOrders) != 4 {
		t.Fatalf("len(pending)=%d want 4", len(summary.PendingOrders))
	}
}
