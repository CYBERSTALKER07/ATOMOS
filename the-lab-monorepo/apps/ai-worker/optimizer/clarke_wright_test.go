package optimizer

import (
	"testing"

	contract "optimizercontract"
)

// TestRouteRespectsWindows_TightCloseRejected proves the HH:MM window
// constraint is a hard reject in the solver. We place a stop ~30 km from the
// depot whose receiving window closes 5 minutes after midnight: at the default
// 30 km/h speed the truck needs 60 minutes to arrive, so the window is
// missed and the route must be reported infeasible.
func TestRouteRespectsWindows_TightCloseRejected(t *testing.T) {
	depot := contract.Vehicle{StartLat: 41.30, StartLng: 69.25, AvgSpeedKmph: 30}
	stops := []contract.Stop{
		{
			OrderID:        "o-late",
			Lat:            41.30,
			Lng:            69.60, // ≈ 29 km east of depot → ~58 min @ 30 km/h
			VolumeVU:       1.0,
			WindowOpen:     "00:00",
			WindowClose:    "00:05", // closes 5 min after midnight
			ServiceMinutes: 5,
		},
	}
	if routeRespectsWindows([]int{0}, stops, depot) {
		t.Fatalf("expected late arrival to be rejected, got route accepted")
	}
}

// TestRouteRespectsWindows_GenerousWindowAccepted is the positive control —
// same geography, generous window, must pass.
func TestRouteRespectsWindows_GenerousWindowAccepted(t *testing.T) {
	depot := contract.Vehicle{StartLat: 41.30, StartLng: 69.25, AvgSpeedKmph: 30}
	stops := []contract.Stop{
		{
			OrderID:        "o-ok",
			Lat:            41.30,
			Lng:            69.60,
			VolumeVU:       1.0,
			WindowOpen:     "00:00",
			WindowClose:    "23:59",
			ServiceMinutes: 5,
		},
	}
	if !routeRespectsWindows([]int{0}, stops, depot) {
		t.Fatalf("expected wide window to be accepted, got route rejected")
	}
}

// TestRouteRespectsWindows_PrioritisationOrdering proves the constraint is
// sequence-aware: the same two stops feasible in order [tight, loose] must
// be infeasible when the tight-window stop is sequenced second behind a stop
// that consumes the available time budget.
func TestRouteRespectsWindows_PrioritisationOrdering(t *testing.T) {
	depot := contract.Vehicle{StartLat: 41.30, StartLng: 69.25, AvgSpeedKmph: 30}
	stops := []contract.Stop{
		{ // tight window — must be visited first
			OrderID:        "o-tight",
			Lat:            41.30,
			Lng:            69.30, // ~4 km from depot → 8 min
			VolumeVU:       1.0,
			WindowOpen:     "00:00",
			WindowClose:    "00:30",
			ServiceMinutes: 5,
		},
		{ // generous window — can be visited any time
			OrderID:        "o-loose",
			Lat:            41.30,
			Lng:            69.60, // ~29 km from depot → 58 min
			VolumeVU:       1.0,
			WindowOpen:     "00:00",
			WindowClose:    "23:59",
			ServiceMinutes: 5,
		},
	}
	// First stop tight, then loose: feasible.
	if !routeRespectsWindows([]int{0, 1}, stops, depot) {
		t.Fatalf("expected [tight, loose] sequence to be feasible")
	}
	// First stop loose, then tight: tight stop's window closed long ago by the
	// time the truck doubles back.
	if routeRespectsWindows([]int{1, 0}, stops, depot) {
		t.Fatalf("expected [loose, tight] sequence to be infeasible (tight window closed)")
	}
}
