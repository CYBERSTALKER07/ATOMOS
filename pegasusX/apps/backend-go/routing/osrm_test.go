package routing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/pkg/circuit"
)

func TestOSRMClient_RouteGeometry(t *testing.T) {
	waypoints := []LatLng{
		{Lat: 41.2995, Lng: 69.2401},
		{Lat: 41.3095, Lng: 69.2501},
	}
	encoded := EncodePolyline(waypoints)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/route/v1/driving/69.240100,41.299500;69.250100,41.309500" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"code":"Ok","routes":[{"geometry":"` + encoded + `","distance":1200,"legs":[{"steps":[]}] }]}`))
	}))
	defer server.Close()

	client := NewOSRMClient(server.URL, circuit.New("osrm-test", circuit.Config{}))
	geometry, err := client.RouteGeometry(context.Background(), "route-1", waypoints, false)
	if err != nil {
		t.Fatalf("RouteGeometry: %v", err)
	}
	if geometry.Source != "osrm_driving" {
		t.Fatalf("source=%q", geometry.Source)
	}
	if len(geometry.Coordinates) < 2 {
		t.Fatalf("expected coordinates, got %d", len(geometry.Coordinates))
	}
}

func TestOSRMClient_RouteGeometryWithSteps(t *testing.T) {
	waypoints := []LatLng{
		{Lat: 41.2995, Lng: 69.2401},
		{Lat: 41.3095, Lng: 69.2501},
	}
	encoded := EncodePolyline(waypoints)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "steps=true") {
			t.Fatalf("expected steps=true query, got %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"code":"Ok","routes":[{"geometry":"` + encoded + `","distance":1200,"legs":[{"steps":[{"distance":1200,"duration":90,"name":"Amir Temur","mode":"driving","maneuver":{"type":"depart","modifier":"north","location":[69.2401,41.2995]}}]}]}]}`))
	}))
	defer server.Close()

	client := NewOSRMClient(server.URL, circuit.New("osrm-test", circuit.Config{}))
	geometry, err := client.RouteGeometry(context.Background(), "route-1", waypoints, true)
	if err != nil {
		t.Fatalf("RouteGeometry: %v", err)
	}
	if len(geometry.Steps) != 1 {
		t.Fatalf("steps=%d want 1", len(geometry.Steps))
	}
	if geometry.Steps[0].Instruction == "" {
		t.Fatalf("expected instruction")
	}
}

func TestGeometryBuilder_FallsBackToDense(t *testing.T) {
	waypoints := []LatLng{
		{Lat: 41.2995, Lng: 69.2401},
		{Lat: 41.3095, Lng: 69.2501},
	}
	builder := NewGeometryBuilder(nil, nil, RoutingProviderAuto)
	geometry := builder.Build(context.Background(), "route-1", waypoints)
	if geometry.Source != "computed_dense" {
		t.Fatalf("source=%q", geometry.Source)
	}
}
