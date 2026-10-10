package supplier

import (
	"testing"
	"time"
)

func TestAggregateOrderMetrics_FiscalFailedIncrementsChip(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 16, 15, 0, 0, 0, time.UTC)
	got := aggregateOrderMetrics([]SupplierOrder{
		{Status: "FISCAL_FAILED", UpdatedAt: now.Format(time.RFC3339Nano), TotalMinor: 1500},
		{Status: "COMPLETED", UpdatedAt: now.Format(time.RFC3339Nano), CreatedAt: now.Format(time.RFC3339Nano), TotalMinor: 2000, RetailerID: "r1"},
		{Status: "DISPATCHED", UpdatedAt: now.Format(time.RFC3339Nano)},
	}, now)
	if got.ordersByStatus["FISCAL_FAILED"] != 1 {
		t.Fatalf("FISCAL_FAILED=%d want 1 full=%v", got.ordersByStatus["FISCAL_FAILED"], got.ordersByStatus)
	}
	if got.ordersByStatus["LOADED"] != 1 {
		t.Fatalf("DISPATCHED should alias to LOADED, got %d", got.ordersByStatus["LOADED"])
	}
	if got.completedToday != 1 || got.attemptedToday != 3 {
		t.Fatalf("completed/attempted=%d/%d", got.completedToday, got.attemptedToday)
	}
	if len(got.ordersByStatus) != 17 {
		t.Fatalf("funnel keys=%d want 17", len(got.ordersByStatus))
	}
}

func TestManifestStateCounts_UsesDictionary(t *testing.T) {
	t.Parallel()
	got := manifestStateCounts([]SupplierManifestRow{
		{State: "DRAFT"},
		{Status: "SEALED"},
		{State: "unknown"},
	})
	if got["DRAFT"] != 1 || got["SEALED"] != 1 || got["LOADING"] != 0 {
		t.Fatalf("counts=%v", got)
	}
	if len(got) != 6 {
		t.Fatalf("manifest keys=%d want 6", len(got))
	}
}
