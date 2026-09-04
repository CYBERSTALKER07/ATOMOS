package factory

import (
	"context"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

func TestInMemoryRunTx_FailClosedInProduction(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	repo := NewInMemoryRepository()
	err := repo.RunTx(context.Background(), func(ctx context.Context, tx FactoryTx) error {
		t.Fatal("should not run")
		return nil
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "in-memory") {
		t.Fatalf("err=%v", err)
	}
}

func TestResolveFactoryNode_PrefersJWTHomeNode(t *testing.T) {
	svc := NewService(ServiceConfig{
		FactoryNodeID:  "factory-demo-1",
		SeedSupplierID: "sup-1",
		Repo:           NewInMemoryRepository(),
	})
	ctx := auth.WithClaims(context.Background(), auth.Claims{
		Subject:      "fac-user",
		Role:         auth.RoleFactoryAdmin,
		HomeNodeType: auth.HomeNodeFactory,
		HomeNodeID:   "factory-real-9",
		SupplierID:   "sup-1",
	})
	got := svc.resolveFactoryNode(ctx)
	if got != "factory-real-9" {
		t.Fatalf("resolveFactoryNode=%q want factory-real-9", got)
	}
	// Without claims falls back to bootstrap demo id.
	if svc.resolveFactoryNode(context.Background()) != "factory-demo-1" {
		t.Fatalf("fallback = %q", svc.resolveFactoryNode(context.Background()))
	}
}

func TestInMemoryRunTx_LocalRunsEmit(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "")
	t.Setenv("REQUIRE_INFRA_ADAPTERS", "")
	repo := NewInMemoryRepository()
	emitted := false
	err := repo.RunTx(context.Background(), func(ctx context.Context, tx FactoryTx) error {
		return nil
	}, func(buf outbox.TxnBuffer) error {
		emitted = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !emitted {
		t.Fatal("expected emit to run locally")
	}
}
