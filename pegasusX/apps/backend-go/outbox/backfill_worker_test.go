package outbox

import (
	"os"
	"testing"
)

func TestSupplierBackfillEnabledDefaults(t *testing.T) {
	t.Setenv("OUTBOX_SUPPLIER_BACKFILL", "")
	t.Setenv("PEGASUSX_ENV", "ssmr")
	if !SupplierBackfillEnabled() {
		t.Fatal("expected enabled for ssmr")
	}
	t.Setenv("PEGASUSX_ENV", "local")
	if SupplierBackfillEnabled() {
		t.Fatal("expected disabled for local without override")
	}
	t.Setenv("OUTBOX_SUPPLIER_BACKFILL", "true")
	if !SupplierBackfillEnabled() {
		t.Fatal("expected override on")
	}
	t.Setenv("OUTBOX_SUPPLIER_BACKFILL", "0")
	if SupplierBackfillEnabled() {
		t.Fatal("expected override off")
	}
	_ = os.Unsetenv("OUTBOX_SUPPLIER_BACKFILL")
}
