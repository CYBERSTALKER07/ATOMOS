package replenishment

import (
	"os"
	"strings"
)

func factoryPlanningEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("FACTORY_PLANNING_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// skipPlanningOwnedAutoTransfer avoids double SYSTEM_THRESHOLD trucks when P5 pull matrix is on.
// Insights still persist. MEIO / PREDICTIVE_PUSH reason codes are unchanged.
func skipPlanningOwnedAutoTransfer(reason string) bool {
	if !factoryPlanningEnabled() {
		return false
	}
	switch strings.ToUpper(strings.TrimSpace(reason)) {
	case "LOW_STOCK", "HIGH_VELOCITY":
		return true
	default:
		return false
	}
}
