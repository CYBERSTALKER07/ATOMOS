package allocation

import (
	"testing"
)

func TestPartialAllocationEnabledEnv(t *testing.T) {
	t.Setenv("PARTIAL_ALLOCATION_ENABLED", "true")
	if !PartialAllocationEnabled() {
		t.Fatal("expected enabled")
	}
	t.Setenv("PARTIAL_ALLOCATION_ENABLED", "")
	if PartialAllocationEnabled() {
		t.Fatal("expected disabled")
	}
}
