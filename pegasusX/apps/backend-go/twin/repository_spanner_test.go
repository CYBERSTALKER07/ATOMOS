package twin

import "testing"

func TestAssembleRouteTwinViews_BatchedStopsAndInventory(t *testing.T) {
	routes := []RouteTwin{
		{RouteID: "route-1", SupplierID: "sup-1", Status: "ACTIVE"},
		{RouteID: "route-2", SupplierID: "sup-1", Status: "ACTIVE"},
	}
	stops := []StopTwin{
		{RouteID: "route-1", StopID: "stop-1a", Sequence: 1, RemainingGrossMinor: 15000},
		{RouteID: "route-1", StopID: "stop-1b", Sequence: 2, RemainingGrossMinor: 25000},
		{RouteID: "route-2", StopID: "stop-2a", Sequence: 1, RemainingGrossMinor: 9000},
	}
	invs := []VehicleInventory{
		{RouteID: "route-1", Sku: "sku-water", QtyOnVehicle: 40},
	}

	views := assembleRouteTwinViews(routes, stops, invs)
	if len(views) != 2 {
		t.Fatalf("views len=%d want 2", len(views))
	}
	if len(views[0].Stops) != 2 || views[0].Stops[0].StopID != "stop-1a" || views[0].Stops[1].StopID != "stop-1b" {
		t.Fatalf("unexpected route-1 stops: %+v", views[0].Stops)
	}
	if len(views[0].Inventory) != 1 || views[0].Inventory[0].Sku != "sku-water" {
		t.Fatalf("unexpected route-1 inventory: %+v", views[0].Inventory)
	}
	if len(views[1].Stops) != 1 || views[1].Stops[0].StopID != "stop-2a" {
		t.Fatalf("unexpected route-2 stops: %+v", views[1].Stops)
	}
	if views[1].Inventory == nil || len(views[1].Inventory) != 0 {
		t.Fatalf("expected non-nil empty inventory slice for route-2, got %#v", views[1].Inventory)
	}
}
