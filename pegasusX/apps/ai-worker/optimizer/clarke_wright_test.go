package optimizer

import (
	"testing"

	contract "github.com/pegasusx/pegasusx/packages/optimizer-contract"
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

func TestPlanRoutes_AssignsDistinctVehiclesWhenFleetSufficient(t *testing.T) {
	// Two stops requiring separate vehicles due to capacity (60 VU each, vehicle cap = 100 * 0.95 = 95).
	// Two identical vehicles in the fleet.
	// Clarke-Wright must assign each route to a distinct physical vehicle.
	vehicles := []contract.Vehicle{
		{VehicleID: "veh-alpha", MaxVolumeVU: 100, StartLat: 41.30, StartLng: 69.25, AvgSpeedKmph: 30},
		{VehicleID: "veh-beta", MaxVolumeVU: 100, StartLat: 41.30, StartLng: 69.25, AvgSpeedKmph: 30},
	}
	stops := []contract.Stop{
		{OrderID: "o1", Lat: 41.31, Lng: 69.26, VolumeVU: 60, ServiceMinutes: 5},
		{OrderID: "o2", Lat: 41.32, Lng: 69.27, VolumeVU: 60, ServiceMinutes: 5},
	}
	tunables := resolvedTunables{
		tetrisBuffer:     0.95,
		twoOptIterations: 10,
		maxStopsPerRoute: 25,
	}

	routes, orphans := planRoutes(stops, vehicles, tunables)
	if len(orphans) > 0 {
		t.Fatalf("expected 0 orphans, got %d", len(orphans))
	}
	if len(routes) != 2 {
		t.Fatalf("expected 2 separate routes, got %d", len(routes))
	}

	// Verify vehicles assigned are distinct
	seenVehicles := make(map[string]bool)
	for _, r := range routes {
		if seenVehicles[r.VehicleID] {
			t.Fatalf("duplicate vehicle %s assigned to multiple concurrent active routes!", r.VehicleID)
		}
		seenVehicles[r.VehicleID] = true
	}
	if len(seenVehicles) != 2 {
		t.Fatalf("expected 2 distinct vehicles, got %d", len(seenVehicles))
	}
}

func TestPlanRoutes_EnforcesMaxRouteMinutes(t *testing.T) {
	// Stop ~30 km from depot requires ~125 min total.
	// Vehicle with MaxRouteMinutes = 60 must reject it (producing orphan).
	vehiclesShort := []contract.Vehicle{
		{VehicleID: "v-tight", MaxVolumeVU: 100, StartLat: 41.30, StartLng: 69.25, AvgSpeedKmph: 30, MaxRouteMinutes: 60},
	}
	stops := []contract.Stop{
		{OrderID: "o-far", Lat: 41.30, Lng: 69.60, VolumeVU: 10, ServiceMinutes: 5},
	}
	tunables := resolvedTunables{
		tetrisBuffer:     0.95,
		twoOptIterations: 10,
		maxStopsPerRoute: 25,
	}

	routes, orphans := planRoutes(stops, vehiclesShort, tunables)
	if len(routes) != 0 || len(orphans) != 1 {
		t.Fatalf("expected stop to orphan due to max_route_minutes bound, got routes=%d orphans=%d", len(routes), len(orphans))
	}

	// With MaxRouteMinutes = 180, it must be accepted.
	vehiclesLong := []contract.Vehicle{
		{VehicleID: "v-long", MaxVolumeVU: 100, StartLat: 41.30, StartLng: 69.25, AvgSpeedKmph: 30, MaxRouteMinutes: 180},
	}
	routesOk, orphansOk := planRoutes(stops, vehiclesLong, tunables)
	if len(routesOk) != 1 || len(orphansOk) != 0 {
		t.Fatalf("expected stop to be routed with generous max_route_minutes, got routes=%d orphans=%d", len(routesOk), len(orphansOk))
	}
}

func TestPlanRoutes_EnforcesColdChainVehicleCompatibility(t *testing.T) {
	// Stop requiring cold chain must be placed only on cold_chain/reefer vehicle
	vehicles := []contract.Vehicle{
		{VehicleID: "veh-ambient-1", MaxVolumeVU: 100, StartLat: 41.30, StartLng: 69.25, AvgSpeedKmph: 30},
		{VehicleID: "veh-reefer-1", MaxVolumeVU: 100, StartLat: 41.30, StartLng: 69.25, AvgSpeedKmph: 30},
	}
	stops := []contract.Stop{
		{OrderID: "o-cold", Lat: 41.31, Lng: 69.26, VolumeVU: 10, HandlingClass: "COLD_CHAIN", ServiceMinutes: 5},
	}
	tunables := resolvedTunables{
		tetrisBuffer:     0.95,
		twoOptIterations: 10,
		maxStopsPerRoute: 25,
	}

	routes, orphans := planRoutes(stops, vehicles, tunables)
	if len(orphans) != 0 || len(routes) != 1 {
		t.Fatalf("expected chilled stop to be assigned, got routes=%d orphans=%d", len(routes), len(orphans))
	}
	if routes[0].VehicleID != "veh-reefer-1" {
		t.Fatalf("expected cold stop on veh-reefer-1, got %s", routes[0].VehicleID)
	}
}
