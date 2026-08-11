package platformadmin

import (
	"context"
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
	tnt, err := svc.Transition(ctx, "admin@px", TenantSupplier, "sup-new", StatusApproved, "kyb ok")
	if err != nil || tnt.Status != StatusApproved {
		t.Fatalf("approve: %+v err=%v", tnt, err)
	}
	active, err = svc.IsActive(ctx, TenantSupplier, "sup-new")
	if err != nil || !active {
		t.Fatal("expected approved active")
	}
	if _, err := svc.Transition(ctx, "admin@px", TenantSupplier, "sup-new", StatusSuspended, "incident"); err != nil {
		t.Fatal(err)
	}
	active, _ = svc.IsActive(ctx, TenantSupplier, "sup-new")
	if active {
		t.Fatal("suspended must not be active")
	}
	if _, err := svc.Transition(ctx, "admin@px", TenantSupplier, "sup-new", StatusPending, ""); err == nil {
		t.Fatal("expected illegal transition suspended→pending")
	}
	audit, err := svc.ListAudit(ctx, 10)
	if err != nil || len(audit) < 2 {
		t.Fatalf("audit=%d err=%v", len(audit), err)
	}
}

func TestLegacyTenantWithoutRowIsActive(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	active, err := svc.IsActive(context.Background(), TenantSupplier, "seed")
	if err != nil || !active {
		t.Fatal("legacy missing row should be active")
	}
}
