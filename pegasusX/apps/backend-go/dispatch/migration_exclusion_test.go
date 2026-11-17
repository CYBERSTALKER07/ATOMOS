package dispatch

import (
	"strings"
	"testing"
)

func TestDispatchableEligibilitySQL_ExcludesMigrationHeldAndHistoricalOrders(t *testing.T) {
	if !strings.Contains(dispatchableEligibilitySQL, "o.Status != 'MIGRATION_OPEN'") {
		t.Fatalf("dispatchableEligibilitySQL must exclude o.Status = MIGRATION_OPEN, got:\n%s", dispatchableEligibilitySQL)
	}
	if !strings.Contains(dispatchableEligibilitySQL, "COALESCE(o.OrderSource, '') != 'MIGRATION'") {
		t.Fatalf("dispatchableEligibilitySQL must exclude o.OrderSource = MIGRATION, got:\n%s", dispatchableEligibilitySQL)
	}
}
