package replenishment

import (
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

func TestReplenishmentPolicyUpdatedConstant(t *testing.T) {
	if events.EventReplenishmentPolicyUpdated != "REPLENISHMENT_POLICY_UPDATED" {
		t.Fatalf("EventReplenishmentPolicyUpdated = %q", events.EventReplenishmentPolicyUpdated)
	}
	// Prefix routing in notification_dispatcher_parity relies on REPLENISHMENT_*.
	if len(events.EventReplenishmentPolicyUpdated) < len("REPLENISHMENT_") ||
		events.EventReplenishmentPolicyUpdated[:len("REPLENISHMENT_")] != "REPLENISHMENT_" {
		t.Fatalf("event type must keep REPLENISHMENT_ prefix for parity fanout: %q", events.EventReplenishmentPolicyUpdated)
	}
}

func TestDefaultPolicyStable(t *testing.T) {
	p := defaultPolicy("sup-1")
	if p.SupplierID != "sup-1" {
		t.Fatalf("supplier_id = %q", p.SupplierID)
	}
	if p.MinConfidenceScore != 0.85 {
		t.Fatalf("min confidence = %v", p.MinConfidenceScore)
	}
	if p.MaxDailyTransferUnits != 500 {
		t.Fatalf("max daily units = %d", p.MaxDailyTransferUnits)
	}
}
