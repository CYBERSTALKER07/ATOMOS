package warehouse

import (
	"testing"
)

func TestDriverAssignmentGuard_BlocksActiveOrders(t *testing.T) {
	err := driverAssignmentGuard(driverAssignmentState{DriverID: "drv-1"}, 2)
	if err == nil {
		t.Fatal("expected guard error")
	}
	fleetErr, ok := err.(*FleetMutationError)
	if !ok || fleetErr.Code != "driver_active_orders" {
		t.Fatalf("unexpected fleet error: %+v", err)
	}
}

func TestResolveVehicleMaxVU_ExplicitWins(t *testing.T) {
	if got := resolveVehicleMaxVU("CLASS_B", 220); got != 220 {
		t.Fatalf("expected 220, got %v", got)
	}
}

func TestVehicleClassMaxVU_DefaultClassB(t *testing.T) {
	if got := vehicleClassMaxVU("CLASS_B"); got != 150 {
		t.Fatalf("expected 150, got %v", got)
	}
}
