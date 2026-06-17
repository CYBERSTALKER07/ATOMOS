package warehouse

import "testing"

func TestWirePortalVehicle(t *testing.T) {
	v := PortalVehicle{
		VehicleID:            "veh-1",
		Label:                "Truck 01",
		LicensePlate:         "01A111AA",
		VehicleClass:         "CLASS_A",
		MaxVolumeVU:          50,
		IsActive:             false,
		UnavailableReason:    "MAINTENANCE",
		AssignedDriverID:     "drv-1",
		AssignedDriverName:   "Jamshid",
	}
	wire := wirePortalVehicle(v)
	if wire["status"] != "UNAVAILABLE" {
		t.Fatalf("status = %v", wire["status"])
	}
	if wire["capacity_vu"] != 50.0 {
		t.Fatalf("capacity_vu = %v", wire["capacity_vu"])
	}
	if wire["assigned_driver_name"] != "Jamshid" {
		t.Fatalf("assigned_driver_name = %v", wire["assigned_driver_name"])
	}
}
