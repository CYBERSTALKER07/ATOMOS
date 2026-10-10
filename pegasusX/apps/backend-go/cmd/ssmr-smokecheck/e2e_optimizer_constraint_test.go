package main

import "testing"

func TestVehicleForStop_ColdChainAssignment(t *testing.T) {
	routes := []optimizerRoute{
		{VehicleID: "dry-truck", Stops: []optimizerRouteStop{{OrderID: "dry-fixture-2"}}},
		{VehicleID: "reefer", Stops: []optimizerRouteStop{{OrderID: "cold-fixture-1"}}},
	}
	if got := vehicleForStop(routes, "cold-fixture-1"); got != "reefer" {
		t.Fatalf("got %q want reefer", got)
	}
}

func TestVehicleForStop_ConstraintViolationDetectable(t *testing.T) {
	routes := []optimizerRoute{
		{VehicleID: "dry-truck", Stops: []optimizerRouteStop{{OrderID: "cold-fixture-1"}}},
	}
	got := vehicleForStop(routes, "cold-fixture-1")
	if got == "" {
		t.Fatal("expected assignment")
	}
	if got == "reefer" {
		t.Fatal("fixture should demonstrate violation path")
	}
}

func TestVehicleForStop_Missing(t *testing.T) {
	if got := vehicleForStop(nil, "cold-fixture-1"); got != "" {
		t.Fatalf("got %q want empty", got)
	}
}
