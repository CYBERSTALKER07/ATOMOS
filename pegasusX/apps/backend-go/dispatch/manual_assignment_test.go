package dispatch

import "testing"

func TestBuildManualAssignmentCapacityWarning(t *testing.T) {
	rows := []DispatchableOrder{
		{OrderID: "o1", VolumeVU: 60},
		{OrderID: "o2", VolumeVU: 50},
	}
	assignment := BuildManualAssignment(rows, []ManualRouteInput{{
		DriverID: "drv-1",
		OrderIDs: []string{"o1", "o2"},
	}}, map[string]float64{"drv-1": 100})
	if assignment == nil || len(assignment.Routes) != 1 {
		t.Fatalf("expected one route, got %#v", assignment)
	}
	warnings := ManualCapacityWarnings(assignment.Routes, map[string]float64{"drv-1": 100})
	if len(warnings) != 1 {
		t.Fatalf("expected capacity warning, got %d", len(warnings))
	}
	if warnings[0].ExcessVU <= 0 {
		t.Fatalf("expected positive excess, got %v", warnings[0].ExcessVU)
	}
}
