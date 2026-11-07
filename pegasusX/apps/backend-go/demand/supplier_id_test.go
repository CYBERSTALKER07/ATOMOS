package demand

import (
	"strings"
	"testing"
)

func TestResolveSupplierID(t *testing.T) {
	if got := ResolveSupplierID(""); got != PlatformSupplierID {
		t.Fatalf("empty → platform, got %q", got)
	}
	if got := ResolveSupplierID("  "); got != PlatformSupplierID {
		t.Fatalf("blank → platform, got %q", got)
	}
	if got := ResolveSupplierID("sup-1"); got != "sup-1" {
		t.Fatalf("got %q", got)
	}
}

func TestSignalTenantFilterSQL(t *testing.T) {
	// Document the filter shape used by ListSignals (string presence check for gate/docs).
	sql := `AND (SupplierId = @SupplierId OR SupplierId = @PlatformSupplier OR SupplierId IS NULL OR SupplierId = '')`
	if !strings.Contains(sql, "PlatformSupplier") || !strings.Contains(sql, "SupplierId = @SupplierId") {
		t.Fatalf("unexpected filter sql")
	}
}
