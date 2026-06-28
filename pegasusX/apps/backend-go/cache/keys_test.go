package cache

import "testing"

func TestKeyActiveOptimizationJobsUsesHashTag(t *testing.T) {
	t.Parallel()
	got := KeyActiveOptimizationJobs("sup-1")
	want := "{sup:sup-1}:jobs:active"
	if got != want {
		t.Fatalf("key = %q want %q", got, want)
	}
}

func TestSupplierScopedKey(t *testing.T) {
	t.Parallel()
	got := SupplierScopedKey("sup-1", "dispatch:plan")
	if got != "{sup:sup-1}:dispatch:plan" {
		t.Fatalf("key = %q", got)
	}
}
