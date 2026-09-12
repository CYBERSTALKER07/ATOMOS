package payload

import (
	"context"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

func TestInMemoryRunTx_FailClosedInSSMR(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "ssmr")
	t.Setenv("REQUIRE_INFRA_ADAPTERS", "")
	repo := NewInMemoryRepository()
	err := repo.RunTx(context.Background(), func(ctx context.Context, tx PayloadTx) error {
		t.Fatal("fn should not run when memory blocked")
		return nil
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "in-memory") {
		t.Fatalf("err=%v want in-memory blocked", err)
	}
}

func TestInMemoryRunTx_LocalExecutesFn(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "local")
	t.Setenv("REQUIRE_INFRA_ADAPTERS", "false")
	t.Setenv("ALLOW_MEMORY_FALLBACK", "true")
	repo := NewInMemoryRepository()
	ran := false
	err := repo.RunTx(context.Background(), func(ctx context.Context, tx PayloadTx) error {
		ran = true
		return nil
	}, func(buf outbox.TxnBuffer) error {
		return buf.BufferOutbox(context.Background(), outbox.Event{EventID: "e1"})
	})
	if err != nil {
		t.Fatalf("local RunTx: %v", err)
	}
	if !ran {
		t.Fatal("expected fn to execute in local memory mode")
	}
}
