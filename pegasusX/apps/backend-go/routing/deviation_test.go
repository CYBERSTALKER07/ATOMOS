package routing

import "testing"

func TestDistanceToPolylineMeters_onSegment(t *testing.T) {
	path := []LatLng{
		{Lat: 41.2995, Lng: 69.2401},
		{Lat: 41.3005, Lng: 69.2411},
	}
	dist := DistanceToPolylineMeters(41.3000, 69.2406, path)
	if dist > 30 {
		t.Fatalf("distance=%f want <= 30", dist)
	}
}

func TestDistanceToPolylineMeters_offSegment(t *testing.T) {
	path := []LatLng{
		{Lat: 41.2995, Lng: 69.2401},
		{Lat: 41.3005, Lng: 69.2411},
	}
	dist := DistanceToPolylineMeters(41.3050, 69.2500, path)
	if dist < 200 {
		t.Fatalf("distance=%f want large off-route distance", dist)
	}
}

func TestWaypointsAhead_skipsPassedStops(t *testing.T) {
	waypoints := []LatLng{
		{Lat: 41.2995, Lng: 69.2401},
		{Lat: 41.3005, Lng: 69.2411},
		{Lat: 41.3015, Lng: 69.2421},
	}
	from := LatLng{Lat: 41.3005, Lng: 69.2411}
	ahead := WaypointsAhead(from, waypoints, 50)
	if len(ahead) != 2 {
		t.Fatalf("len=%d want 2", len(ahead))
	}
	if ahead[0] != from {
		t.Fatalf("first waypoint should be driver position")
	}
	if ahead[1] != waypoints[2] {
		t.Fatalf("second waypoint should be next stop")
	}
}
