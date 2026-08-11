package featureflags

import (
	"context"
	"testing"
)

func TestEvaluateEnvAndOverride(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	repo := NewMemoryRepository()
	svc := NewService(repo)
	on, src, err := svc.Evaluate(context.Background(), "AR_INVOICES_ENABLED", "", "")
	if err != nil || !on || src != "env" {
		t.Fatalf("env: on=%v src=%s err=%v", on, src, err)
	}
	if err := svc.SetOverride(context.Background(), Override{
		FlagKey: "AR_INVOICES_ENABLED", TenantType: "SUPPLIER", TenantID: "sup-1",
		Enabled: false, UpdatedBy: "admin", Reason: "soak pause",
	}); err != nil {
		t.Fatal(err)
	}
	on, src, err = svc.Evaluate(context.Background(), "AR_INVOICES_ENABLED", "SUPPLIER", "sup-1")
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
}
