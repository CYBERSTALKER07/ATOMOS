package warehouse

import (
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch"
	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
)

func TestFilterDispatchRowsByOrderIDs(t *testing.T) {
	rows := []dispatch.DispatchableOrder{
		{OrderID: "a"},
		{OrderID: "b"},
		{OrderID: "c"},
	}
	filtered := filterDispatchRowsByOrderIDs(rows, []string{"b", "a"})
	if len(filtered) != 2 {
		t.Fatalf("filtered len = %d want 2", len(filtered))
	}
	if filterDispatchRowsByOrderIDs(rows, nil) == nil || len(filterDispatchRowsByOrderIDs(rows, nil)) != 3 {
		t.Fatalf("nil filter should return all rows")
	}
}

func TestDriverResidualVolumes_WithTopOff(t *testing.T) {
	driver := PortalDriver{MaxVolumeVU: 100}
	top := &manifest.DriverManifestCapacity{
		ManifestID:    "m1",
		TotalVolumeVU: 40,
		MaxVolumeVU:   100,
	}
	used, free, effective := driverResidualVolumes(driver, top)
	if used != 40 || free != 60 || effective != 60*dispatch.TetrisBuffer {
		t.Fatalf("residual = used %v free %v effective %v", used, free, effective)
	}
}

func TestFleetEffectiveCapacityVU(t *testing.T) {
	drivers := []PortalDriver{
		{DriverID: "d1", IsActive: true, TruckStatus: "AVAILABLE", MaxVolumeVU: 100},
	}
	fleetCtx := fleetDispatchContext{InTransit: map[string]bool{}}
	total := fleetEffectiveCapacityVU(drivers, fleetCtx)
	want := 100 * dispatch.TetrisBuffer
	if total != want {
		t.Fatalf("fleet effective = %v want %v", total, want)
	}
}

func TestAutoCapacityWarnings_FleetOverload(t *testing.T) {
	assignment := &dispatch.AssignmentResult{
		Orphans: []dispatch.GeoOrder{{OrderID: "orphan-1"}},
	}
	warnings := autoCapacityWarnings(assignment, nil, fleetDispatchContext{}, 120, 95)
	if len(warnings) == 0 {
		t.Fatalf("expected fleet overload warning")
	}
	if len(warnings[0].SuggestedDeferOrderIDs) != 1 || warnings[0].SuggestedDeferOrderIDs[0] != "orphan-1" {
		t.Fatalf("defer ids = %#v", warnings[0].SuggestedDeferOrderIDs)
	}
}
