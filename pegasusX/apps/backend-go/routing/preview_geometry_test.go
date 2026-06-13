package routing

import (
	"context"
	"testing"
)

func TestAttachRouteGeometryToProposedRoutes(t *testing.T) {
	routes := []map[string]any{
		{
			"driver_id": "driver-1",
			"stops": []map[string]any{
				{"lat": 41.31, "lng": 69.25},
				{"lat": 41.32, "lng": 69.26},
			},
		},
	}
	builder := NewGeometryBuilder(nil)
	AttachRouteGeometryToProposedRoutes(context.Background(), builder, LatLng{
		Lat: 41.30,
		Lng: 69.24,
	}, routes)

	geometry, ok := routes[0]["route_geometry"].(RouteGeometryWire)
	if !ok {
		t.Fatalf("route_geometry missing or wrong type: %#v", routes[0]["route_geometry"])
	}
	if geometry.Source != "computed_dense" {
		t.Fatalf("source=%q", geometry.Source)
	}
	if len(geometry.Coordinates) < 2 {
		t.Fatalf("expected coordinates, got %d", len(geometry.Coordinates))
	}
}

func TestFormatStepInstruction(t *testing.T) {
	got := formatStepInstruction("turn", "right", "Amir Temur")
	want := "Turn right onto Amir Temur"
	if got != want {
		t.Fatalf("instruction=%q want %q", got, want)
	}
}
