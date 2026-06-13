package routing

import (
	"math"
	"testing"
)

func TestDensifyPath_addsIntermediatePoints(t *testing.T) {
	waypoints := []LatLng{
		{Lat: 41.2995, Lng: 69.2401},
		{Lat: 41.3095, Lng: 69.2501},
	}
	dense := DensifyPath(waypoints, 100)
	if len(dense) < 3 {
		t.Fatalf("expected densified path, got %d points", len(dense))
	}
	if dense[0] != waypoints[0] {
		t.Fatalf("expected first point preserved")
	}
	last := dense[len(dense)-1]
	if last.Lat != waypoints[1].Lat || last.Lng != waypoints[1].Lng {
		t.Fatalf("expected last point preserved")
	}
}

func TestEncodePolyline_roundTripShape(t *testing.T) {
	coords := []LatLng{
		{Lat: 41.2995, Lng: 69.2401},
		{Lat: 41.3000, Lng: 69.2410},
	}
	encoded := EncodePolyline(coords)
	if encoded == "" {
		t.Fatal("expected non-empty polyline")
	}
	decoded, err := DecodePolyline(encoded)
	if err != nil {
		t.Fatalf("decode polyline: %v", err)
	}
	if len(decoded) != len(coords) {
		t.Fatalf("decoded len=%d want %d", len(decoded), len(coords))
	}
	for i := range coords {
		if math.Abs(decoded[i].Lat-coords[i].Lat) > 1e-4 || math.Abs(decoded[i].Lng-coords[i].Lng) > 1e-4 {
			t.Fatalf("point %d drift: got %+v want %+v", i, decoded[i], coords[i])
		}
	}
}

func TestGeometryFromStoredPolyline(t *testing.T) {
	coords := []LatLng{{Lat: 41.2995, Lng: 69.2401}, {Lat: 41.3095, Lng: 69.2501}}
	encoded := EncodePolyline(coords)
	geometry, err := GeometryFromStoredPolyline("route-1", encoded, "manifest_sealed", 2)
	if err != nil {
		t.Fatalf("GeometryFromStoredPolyline: %v", err)
	}
	if geometry.Source != "manifest_sealed" || len(geometry.Coordinates) < 2 {
		t.Fatalf("unexpected geometry: %+v", geometry)
	}
}

func TestBuildDenseRouteGeometry_insufficientWaypoints(t *testing.T) {
	got := BuildDenseRouteGeometry("route-1", []LatLng{{Lat: 1, Lng: 2}})
	if got.Source != "insufficient_waypoints" {
		t.Fatalf("unexpected source %q", got.Source)
	}
}
