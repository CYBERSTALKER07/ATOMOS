package order

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPrepareHistoricalOrder_TerminalGuardAndTashkentTimestamps(t *testing.T) {
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)

	ts, err := ParseMigrationTimestamp("2026-10-01 15:04:05")
	if err != nil {
		t.Fatalf("ParseMigrationTimestamp: %v", err)
	}
	// Asia/Tashkent is UTC+5, so 15:04:05 UZT == 10:04:05 UTC.
	if ts.Hour() != 10 || ts.Minute() != 4 {
		t.Fatalf("expected 10:04 UTC from Asia/Tashkent string, got %s", ts.Format(time.RFC3339))
	}

	// Non-terminal statuses must fail closed.
	for _, raw := range []string{"PENDING", "IN_TRANSIT", "PICKING"} {
		if _, _, err := PrepareHistoricalOrder(HistoricalOrderRequest{
			SupplierID:      "sup-1",
			RetailerID:      "ret-1",
			ExternalOrderNo: "HIST-BAD-" + raw,
			RawStatus:       raw,
			TotalMinor:      100000,
			CreatedAt:       ts,
			Now:             now,
		}); !errors.Is(err, ErrInvalidHistoricalStatus) {
			t.Fatalf("status %q got err=%v want ErrInvalidHistoricalStatus", raw, err)
		}
	}

	// Terminal Russian / Uzbek / English statuses map cleanly with OrderSource=MIGRATION & FiscalStatus=HISTORICAL_SKIPPED.
	o, insertMap, err := PrepareHistoricalOrder(HistoricalOrderRequest{
		SupplierID:      "sup-1",
		RetailerID:      "ret-1",
		ExternalOrderNo: "1C-ORD-9001",
		RawStatus:       "ВОЗВРАТ",
		TotalMinor:      450000,
		LineItems: []LineItem{
			{SKU: "prod-1", Name: "Black Tea", Quantity: 10, UnitPrice: 45000},
		},
		CreatedAt: ts,
		Now:       now,
	})
	if err != nil {
		t.Fatalf("PrepareHistoricalOrder: %v", err)
	}
	if o.Status != StatusCancelled || o.Source != OrderSourceMigration || o.FiscalStatus != FiscalStatusHistoricalSkipped {
		t.Fatalf("unexpected historical order attributes: status=%s source=%s fiscal=%s", o.Status, o.Source, o.FiscalStatus)
	}
	if insertMap["OrderSource"] != string(OrderSourceMigration) || insertMap["FiscalStatus"] != FiscalStatusHistoricalSkipped {
		t.Fatalf("unexpected insertMap markers: %+v", insertMap)
	}
}

func TestPrepareAndResolveMigratedOpenOrder_ReleaseGuards(t *testing.T) {
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)

	o, _, err := PrepareMigratedOpenOrder(MigratedOpenOrderRequest{
		SupplierID:         "sup-1",
		RetailerID:         "ret-1",
		WarehouseID:        "wh-1",
		ExternalOrderNo:    "OPEN-1C-42",
		IntendedLiveStatus: StatusScheduled,
		TotalMinor:         250000,
		LineItems:          []LineItem{{SKU: "prod-1", Name: "Tea", Quantity: 5, UnitPrice: 50000}},
		Now:                now,
	})
	if err != nil {
		t.Fatalf("PrepareMigratedOpenOrder: %v", err)
	}
	if o.Status != StatusMigrationOpen || o.Source != OrderSourceMigration {
		t.Fatalf("expected MIGRATION_OPEN / MIGRATION, got status=%s source=%s", o.Status, o.Source)
	}

	// Intended target from WarehouseNotes resolves to SCHEDULED; explicit override works too.
	target, err := ResolveReleaseTargetStatus(o, "")
	if err != nil || target != StatusScheduled {
		t.Fatalf("ResolveReleaseTargetStatus default got %s err=%v want SCHEDULED", target, err)
	}
	targetPending, err := ResolveReleaseTargetStatus(o, StatusPending)
	if err != nil || targetPending != StatusPending {
		t.Fatalf("ResolveReleaseTargetStatus override got %s err=%v want PENDING", targetPending, err)
	}

	// Rolled-back or already-live orders cannot be released again.
	superseded := o
	superseded.SupersededByRollback = true
	if _, err := ResolveReleaseTargetStatus(superseded, StatusPending); !errors.Is(err, ErrOrderNotMigrationHeld) {
		t.Fatalf("expected ErrOrderNotMigrationHeld for superseded order, got %v", err)
	}
	alreadyLive := o
	alreadyLive.Status = StatusPending
	if _, err := ResolveReleaseTargetStatus(alreadyLive, StatusPending); !errors.Is(err, ErrOrderNotMigrationHeld) {
		t.Fatalf("expected ErrOrderNotMigrationHeld for live order, got %v", err)
	}
}

func TestMigrationFiscalExclusion_EvaluateAndSweeperSkip(t *testing.T) {
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)

	snap := EvaluateOpenFiscalOrders(true, []OpenFiscalCandidate{
		{
			OrderID:      "ord-hist-mig",
			Status:       string(StatusFiscalFailed),
			FiscalStatus: FiscalStatusHistoricalSkipped,
		},
		{
			OrderID:      "ord-open-mig",
			Status:       string(StatusMigrationOpen),
			FiscalStatus: FiscalStatusNone,
		},
	})
	if snap.Frozen || snap.Count != 0 || snap.FiscalIssueCount != 0 {
		t.Fatalf("migration orders must be completely excluded from EvaluateOpenFiscalOrders, got %+v", snap)
	}

	histOrder := deliveryTestOrder(StatusFiscalFailed)
	histOrder.OrderID = "ord-hist-sweeper"
	histOrder.Source = OrderSourceMigration
	histOrder.FiscalStatus = FiscalStatusHistoricalSkipped
	histOrder.UpdatedAt = now.Add(-10 * time.Minute)

	repo := &testRepo{found: true, order: histOrder}
	svc := newTestService(repo, now)
	svc.SetFiscalProvider(FakeFiscalProvider{})
	if err := svc.RunFiscalSweeperOnce(context.Background()); err != nil {
		t.Fatalf("RunFiscalSweeperOnce: %v", err)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("expected RunFiscalSweeperOnce to skip MIGRATION orders without updates, got %d update(s)", repo.updateCalls)
	}
}
