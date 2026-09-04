package warehouse

import (
	"context"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestResolveSupplierScopePrefersTenant(t *testing.T) {
	svc := NewService(ServiceConfig{SupplierID: "seed"})
	ctx := auth.WithTenant(context.Background(), auth.TenantContext{SupplierID: "sup-live", Source: "jwt"})
	if got := svc.resolveSupplierScope(ctx); got != "sup-live" {
		t.Fatalf("got=%q", got)
	}
}

func TestResolveSupplierScopeNoSeedWhenEnforced(t *testing.T) {
	t.Setenv("TENANT_CONTEXT_ENFORCED", "true")
	svc := NewService(ServiceConfig{SupplierID: "seed"})
	ctx := auth.WithClaims(context.Background(), auth.Claims{Subject: "u", Role: auth.RoleAdmin})
	if got := svc.resolveSupplierScope(ctx); got != "" {
		t.Fatalf("authenticated without supplier must not seed, got=%q", got)
	}
}
