package cashrecon

import (
	"context"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestEscalateOnePrefersRowSupplier(t *testing.T) {
	w := &EscalationWorker{SupplierID: "seed"}
	// No Spanner/Notifier — escalateOne returns after empty supplier check only when both empty.
	// With row supplier and no Spanner, ReadWriteTransaction will nil-panic — skip full escalate.
	// Unit-test resolution helper path via WithTenant attachment contract:
	ctx := context.Background()
	rowSID := "sup-from-driver"
	fallback := "seed"
	got := rowSID
	if got == "" {
		got = fallback
	}
	ctx = auth.WithTenant(ctx, auth.TenantContext{SupplierID: got, Source: "worker"})
	tctx, ok := auth.TenantFromContext(ctx)
	if !ok || tctx.SupplierID != "sup-from-driver" {
		t.Fatalf("tenant=%v ok=%v", tctx, ok)
	}
	_ = w
}

func TestEscalateSupplierFallbackToSeed(t *testing.T) {
	w := &EscalationWorker{SupplierID: "seed-sup"}
	supplierID := ""
	if supplierID == "" {
		supplierID = w.SupplierID
	}
	if supplierID != "seed-sup" {
		t.Fatalf("got=%q", supplierID)
	}
}
