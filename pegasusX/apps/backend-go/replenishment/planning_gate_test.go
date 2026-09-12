package replenishment

import (
	"os"
	"testing"
)

func TestSkipPlanningOwnedAutoTransfer(t *testing.T) {
	t.Setenv("FACTORY_PLANNING_ENABLED", "")
	if skipPlanningOwnedAutoTransfer("LOW_STOCK") {
		t.Fatal("flag off must not skip")
	}
	t.Setenv("FACTORY_PLANNING_ENABLED", "true")
	if !skipPlanningOwnedAutoTransfer("LOW_STOCK") || !skipPlanningOwnedAutoTransfer("HIGH_VELOCITY") {
		t.Fatal("planning flag owns LOW_STOCK/HIGH_VELOCITY transfers")
	}
	if skipPlanningOwnedAutoTransfer("MEIO_NETWORK") || skipPlanningOwnedAutoTransfer("PREDICTIVE_PUSH") {
		t.Fatal("MEIO/predictive insight reasons stay on replenishment engine")
	}
	_ = os.Unsetenv("FACTORY_PLANNING_ENABLED")
}
