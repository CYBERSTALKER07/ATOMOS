package platformadmin

import (
	"context"
	"errors"
	"testing"
)

func TestTenantLifecycle(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	ctx := context.Background()
	if err := svc.EnsurePending(ctx, TenantSupplier, "sup-new", "New Co"); err != nil {
		t.Fatal(err)
	}
	active, err := svc.IsActive(ctx, TenantSupplier, "sup-new")
	if err != nil || active {
		t.Fatalf("pending should not be active: active=%v err=%v", active, err)
	}
	first, err := svc.Transition(ctx, TransitionInput{
		Actor: "admin-a", TenantType: TenantSupplier, TenantID: "sup-new",
		Status: StatusApproved, KybNotes: "kyb ok", MarketCode: "UZ", HomeCell: "cell-uz",
	})
	if err != nil || first.Status != StatusPending || first.RequestedBy != "admin-a" {
		t.Fatalf("first approve must stay pending: %+v err=%v", first, err)
	}
	active, _ = svc.IsActive(ctx, TenantSupplier, "sup-new")
	if active {
		t.Fatal("requested approve must not be active")
	}
	tnt, err := svc.Transition(ctx, TransitionInput{
		Actor: "admin-b", TenantType: TenantSupplier, TenantID: "sup-new",
		Status: StatusApproved, MarketCode: "UZ", HomeCell: "cell-uz",
	})
	if err != nil || tnt.Status != StatusApproved || tnt.ApprovedBy != "admin-b" {
		t.Fatalf("second approve: %+v err=%v", tnt, err)
	}
	if tnt.MarketCode != "UZ" || tnt.HomeCell != "cell-uz" {
		t.Fatalf("pack=%+v", tnt)
	}
	active, err = svc.IsActive(ctx, TenantSupplier, "sup-new")
	if err != nil || !active {
		t.Fatal("expected approved active")
	}
	if _, err := svc.Transition(ctx, TransitionInput{
		Actor: "admin-b", TenantType: TenantSupplier, TenantID: "sup-new",
		Status: StatusSuspended, KybNotes: "incident",
	}); err != nil {
		t.Fatal(err)
	}
	active, _ = svc.IsActive(ctx, TenantSupplier, "sup-new")
	if active {
		t.Fatal("suspended must not be active")
	}
	if _, err := svc.Transition(ctx, TransitionInput{
		Actor: "admin-b", TenantType: TenantSupplier, TenantID: "sup-new",
		Status: StatusPending,
	}); err == nil {
		t.Fatal("expected illegal transition suspended→pending")
	}
	audit, err := svc.ListAudit(ctx, 10)
	if err != nil || len(audit) < 2 {
		t.Fatalf("audit=%d err=%v", len(audit), err)
	}
}

func TestMissingRowIsNotActive(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	active, err := svc.IsActive(context.Background(), TenantSupplier, "seed")
	if err != nil || active {
		t.Fatal("GS-T3: missing KYB row must not be active")
	}
}

func TestTransitionMissingRowNotFound(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	_, err := svc.Transition(context.Background(), TransitionInput{
		Actor: "admin-a", TenantType: TenantSupplier, TenantID: "missing",
		Status: StatusApproved, MarketCode: "UZ",
	})
	if !errors.Is(err, ErrTenantNotFound) {
		t.Fatalf("err=%v", err)
	}
}

func TestApproveSameActorRejected(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	ctx := context.Background()
	_ = svc.EnsurePending(ctx, TenantSupplier, "sup-1", "Co")
	_, err := svc.Transition(ctx, TransitionInput{
		Actor: "admin-a", TenantType: TenantSupplier, TenantID: "sup-1",
		Status: StatusApproved, MarketCode: "UZ",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Transition(ctx, TransitionInput{
		Actor: "admin-a", TenantType: TenantSupplier, TenantID: "sup-1",
		Status: StatusApproved, MarketCode: "UZ",
	})
	if !errors.Is(err, ErrApproverMustDiffer) {
		t.Fatalf("err=%v", err)
	}
}

func TestApprovePackMismatchRejected(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	ctx := context.Background()
	_ = svc.EnsurePending(ctx, TenantSupplier, "sup-1", "Co")
	_, _ = svc.Transition(ctx, TransitionInput{
		Actor: "admin-a", TenantType: TenantSupplier, TenantID: "sup-1",
		Status: StatusApproved, MarketCode: "UZ", HomeCell: "cell-uz",
	})
	_, err := svc.Transition(ctx, TransitionInput{
		Actor: "admin-b", TenantType: TenantSupplier, TenantID: "sup-1",
		Status: StatusApproved, MarketCode: "UZ", HomeCell: "cell-eu",
	})
	if !errors.Is(err, ErrHomeCellMismatch) && !errors.Is(err, ErrPackCellMismatch) {
		t.Fatalf("err=%v", err)
	}
}

func TestApprovePlannedPackRejected(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	ctx := context.Background()
	_ = svc.EnsurePending(ctx, TenantSupplier, "sup-1", "Co")
	_, err := svc.Transition(ctx, TransitionInput{
		Actor: "admin-a", TenantType: TenantSupplier, TenantID: "sup-1",
		Status: StatusApproved, MarketCode: "EU",
	})
	if !errors.Is(err, ErrMarketNotShipped) {
		t.Fatalf("err=%v", err)
	}
}

func TestApproveEmptyMarketRejected(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	ctx := context.Background()
	_ = svc.EnsurePending(ctx, TenantSupplier, "sup-1", "Co")
	_, err := svc.Transition(ctx, TransitionInput{
		Actor: "admin-a", TenantType: TenantSupplier, TenantID: "sup-1",
		Status: StatusApproved,
	})
	if !errors.Is(err, ErrMarketCodeRequired) {
		t.Fatalf("empty market must not default to UZ: %v", err)
	}
}

func TestSystemBootstrapSingleStepApprove(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	ctx := context.Background()
	_ = svc.EnsurePending(ctx, TenantSupplier, "seed-1", "Seed")
	tnt, err := svc.Transition(ctx, TransitionInput{
		Actor: "system:bootstrap", TenantType: TenantSupplier, TenantID: "seed-1",
		Status: StatusApproved, KybNotes: "seed_auto_approve", MarketCode: "UZ", HomeCell: "cell-uz",
	})
	if err != nil || tnt.Status != StatusApproved {
		t.Fatalf("seed bootstrap: %+v err=%v", tnt, err)
	}
	active, _ := svc.IsActive(ctx, TenantSupplier, "seed-1")
	if !active {
		t.Fatal("seed must be active after system approve")
	}
}

func TestApproveCopiesPackOnHook(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	var gotType, gotID, gotMarket, gotCell string
	svc.OnApproved = func(_ context.Context, tenantType, tenantID, marketCode, homeCell string) error {
		gotType, gotID, gotMarket, gotCell = tenantType, tenantID, marketCode, homeCell
		return nil
	}
	ctx := context.Background()
	_ = svc.EnsurePending(ctx, TenantSupplier, "sup-1", "Co")
	_, _ = svc.Transition(ctx, TransitionInput{
		Actor: "admin-a", TenantType: TenantSupplier, TenantID: "sup-1",
		Status: StatusApproved, MarketCode: "uz",
	})
	if gotID != "" {
		t.Fatal("hook must not run on request")
	}
	_, err := svc.Transition(ctx, TransitionInput{
		Actor: "admin-b", TenantType: TenantSupplier, TenantID: "sup-1",
		Status: StatusApproved, MarketCode: "UZ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotType != TenantSupplier || gotID != "sup-1" || gotMarket != "UZ" || gotCell != "cell-uz" {
		t.Fatalf("hook=%s %s %s %s", gotType, gotID, gotMarket, gotCell)
	}
}
