package order

import (
	"context"
	"errors"
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

func TestGetOrderForTenantIDOR(t *testing.T) {
	repo := &testRepo{
		order: Order{OrderID: "ord-b", SupplierID: "sup-b"},
		found: true,
	}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-a"})

	ctxA := auth.WithTenant(context.Background(), auth.TenantContext{SupplierID: "sup-a", Source: "jwt"})
	_, ok, err := svc.GetOrderForTenant(ctxA, "ord-b")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if ok {
		t.Fatal("cross-tenant order must fail closed as not found")
	}

	ctxB := auth.WithTenant(context.Background(), auth.TenantContext{SupplierID: "sup-b", Source: "jwt"})
	o, ok, err := svc.GetOrderForTenant(ctxB, "ord-b")
	if err != nil || !ok || o.OrderID != "ord-b" {
		t.Fatalf("own tenant: ok=%v err=%v o=%+v", ok, err, o)
	}
}

func TestResolveSupplierIDForCreatePrefersTenant(t *testing.T) {
	svc := NewService(ServiceConfig{SupplierID: "seed"})
	ctx := auth.WithTenant(context.Background(), auth.TenantContext{SupplierID: "sup-live", Source: "jwt"})
	got, err := svc.resolveSupplierIDForCreate(ctx, "")
	if err != nil || got != "sup-live" {
		t.Fatalf("got=%q err=%v", got, err)
	}
	_, err = svc.resolveSupplierIDForCreate(ctx, "other")
	if !errors.Is(err, ErrOrderForbidden) {
		t.Fatalf("mismatch err=%v", err)
	}
}
