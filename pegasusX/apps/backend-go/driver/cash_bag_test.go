package driver

import (
	"context"
	"testing"
	"time"
)

func TestGetCashBagSummary_Default(t *testing.T) {
	svc := &Service{
		pendingQuery: func(driverID string) []PendingCollection {
			return []PendingCollection{
				{OrderID: "ord-1", Amount: 50_000, State: "PENDING"},
				{OrderID: "ord-2", Amount: 30_000, State: "PENDING"},
			}
		},
		now: func() time.Time { return time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC) },
	}

	summary, err := svc.GetCashBagSummary(context.Background(), "driver-1")
	if err != nil {
		t.Fatal(err)
	}
	if summary.ExpectedCashMinor != 80_000 {
		t.Fatalf("expected=%d want 80000", summary.ExpectedCashMinor)
	}
	if summary.ReconciliationStatus != "PENDING_TURN_IN" {
		t.Fatalf("status=%s want PENDING_TURN_IN", summary.ReconciliationStatus)
	}
	if len(summary.PendingOrders) != 2 {
		t.Fatalf("len(pending)=%d want 2", len(summary.PendingOrders))
	}
}
