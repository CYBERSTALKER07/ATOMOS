package cache

import "testing"

func TestSupplierScopedKey(t *testing.T) {
	t.Parallel()
	got := SupplierScopedKey("sup-1", "dispatch:plan")
	if got != "{sup:sup-1}:dispatch:plan" {
		t.Fatalf("key = %q", got)
	}
}
