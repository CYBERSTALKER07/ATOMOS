package featureflags

import (
	"context"
	"testing"
)

func TestEvaluateEnvAndOverride(t *testing.T) {
	t.Setenv("PROMO_RULES_ENABLED", "true") // non-money flag: applies immediately
	repo := NewMemoryRepository()
	svc := NewService(repo)
	on, src, err := svc.Evaluate(context.Background(), "PROMO_RULES_ENABLED", "", "")
	if err != nil || !on || src != "env" {
		t.Fatalf("env: on=%v src=%s err=%v", on, src, err)
	}
	if err := svc.SetOverride(context.Background(), Override{
		FlagKey: "PROMO_RULES_ENABLED", TenantType: "SUPPLIER", TenantID: "sup-1",
		Enabled: false, UpdatedBy: "admin", Reason: "test",
	}); err != nil {
		t.Fatal(err)
	}
	on, src, err = svc.Evaluate(context.Background(), "PROMO_RULES_ENABLED", "SUPPLIER", "sup-1")
	if err != nil || on || src != "tenant_override" {
		t.Fatalf("override: on=%v src=%s err=%v", on, src, err)
	}
}

func TestMoneyFlagRequiresReason(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	err := svc.SetOverride(context.Background(), Override{
		FlagKey: "AUTO_ORDER_PLACE_ENABLED", TenantType: "SUPPLIER", TenantID: "s1",
		Enabled: true, UpdatedBy: "a",
	})
	if err == nil {
		t.Fatal("expected reason_required")
	}
	err = svc.SetOverride(context.Background(), Override{
		FlagKey: "AUTO_ORDER_SOAK_GATE_DISABLED", TenantType: "RETAILER", TenantID: "r1",
		Enabled: true, UpdatedBy: "a",
	})
	if err == nil {
		t.Fatal("expected reason_required for soak-gate break-glass")
	}
	if !MoneyAffectingFlags["AUTO_ORDER_SOAK_GATE_DISABLED"] {
		t.Fatal("AUTO_ORDER_SOAK_GATE_DISABLED must be money-affecting")
	}
}

// Money flags are dual-controlled: a set is PENDING and does not change runtime
// evaluation until a DIFFERENT admin approves it.
func TestMoneyFlagDualControl(t *testing.T) {
	ctx := context.Background()
	t.Setenv("AUTO_ORDER_PLACE_ENABLED", "false")
	svc := NewService(NewMemoryRepository())

	// Setter requests enable; must stay PENDING and not take effect.
	if err := svc.SetOverride(ctx, Override{
		FlagKey: "AUTO_ORDER_PLACE_ENABLED", TenantType: "SUPPLIER", TenantID: "s1",
		Enabled: true, UpdatedBy: "setter-1", Reason: "pilot cohort",
	}); err != nil {
		t.Fatal(err)
	}
	on, src, _ := svc.Evaluate(ctx, "AUTO_ORDER_PLACE_ENABLED", "SUPPLIER", "s1")
	if on || src != "env" {
		t.Fatalf("pending override must not take effect: on=%v src=%s", on, src)
	}

	// Same actor cannot approve their own change.
	if err := svc.ApproveOverride(ctx, "AUTO_ORDER_PLACE_ENABLED", "SUPPLIER", "s1", "setter-1"); err == nil {
		t.Fatal("self-approval must be rejected")
	}

	// A different approver activates it.
	if err := svc.ApproveOverride(ctx, "AUTO_ORDER_PLACE_ENABLED", "SUPPLIER", "s1", "approver-2"); err != nil {
		t.Fatalf("approve: %v", err)
	}
	on, src, _ = svc.Evaluate(ctx, "AUTO_ORDER_PLACE_ENABLED", "SUPPLIER", "s1")
	if !on || src != "tenant_override" {
		t.Fatalf("approved override must take effect: on=%v src=%s", on, src)
	}

	// Approving a non-pending override fails.
	if err := svc.ApproveOverride(ctx, "AUTO_ORDER_PLACE_ENABLED", "SUPPLIER", "s1", "approver-3"); err == nil {
		t.Fatal("re-approving an active override must fail")
	}
}
