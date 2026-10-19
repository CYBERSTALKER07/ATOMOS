package retailer

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
	if got := svc.resolveSupplierScope(context.Background()); got != "seed" {
		t.Fatalf("fallback got=%q", got)
	}
}
