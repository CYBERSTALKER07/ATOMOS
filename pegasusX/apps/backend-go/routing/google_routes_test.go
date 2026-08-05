package routing

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/pkg/circuit"
)

func TestGoogleRoutesClient_RouteGeometry(t *testing.T) {
	waypoints := []LatLng{
		{Lat: 41.2995, Lng: 69.2401},
		{Lat: 41.3095, Lng: 69.2501},
	}
	encoded := EncodePolyline(waypoints)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != googleRoutesComputePath {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if r.Header.Get("X-Goog-Api-Key") != "test-key" {
			t.Fatalf("missing api key")
		}
		if !strings.Contains(r.Header.Get("X-Goog-FieldMask"), "encodedPolyline") {
			t.Fatalf("field mask=%q", r.Header.Get("X-Goog-FieldMask"))
		}
		body, _ := io.ReadAll(r.Body)
		var req googleRoutesRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.TravelMode != "DRIVE" {
			t.Fatalf("travelMode=%q", req.TravelMode)
		}
		_, _ = w.Write([]byte(`{"routes":[{"polyline":{"encodedPolyline":"` + encoded + `"},"legs":[]}]}`))
	}))
	defer server.Close()

	client := NewGoogleRoutesClient("test-key", server.URL, circuit.New("google-test", circuit.Config{}))
	geometry, err := client.RouteGeometry(context.Background(), "route-1", waypoints, false)
	if err != nil {
		t.Fatalf("RouteGeometry: %v", err)
	}
	if geometry.Source != googleRoutesSource {
		t.Fatalf("source=%q", geometry.Source)
	}
	if len(geometry.Coordinates) < 2 {
		t.Fatalf("coords=%d", len(geometry.Coordinates))
	}
}

func TestGoogleRoutesClient_RouteGeometryWithSteps(t *testing.T) {
	waypoints := []LatLng{
		{Lat: 41.2995, Lng: 69.2401},
		{Lat: 41.3095, Lng: 69.2501},
	}
	encoded := EncodePolyline(waypoints)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("X-Goog-FieldMask"), "navigationInstruction") {
			t.Fatalf("expected steps field mask, got %q", r.Header.Get("X-Goog-FieldMask"))
		}
		_, _ = w.Write([]byte(`{"routes":[{"polyline":{"encodedPolyline":"` + encoded + `"},"legs":[{"steps":[{"distanceMeters":1200,"staticDuration":"90s","navigationInstruction":{"maneuver":"DEPART","instructions":"Head north"},"startLocation":{"latLng":{"latitude":41.2995,"longitude":69.2401}}}]}]}]}`))
	}))
	defer server.Close()

	client := NewGoogleRoutesClient("test-key", server.URL, nil)
	geometry, err := client.RouteGeometry(context.Background(), "route-1", waypoints, true)
	if err != nil {
		t.Fatalf("RouteGeometry: %v", err)
	}
	if len(geometry.Steps) != 1 {
		t.Fatalf("steps=%d", len(geometry.Steps))
	}
	if geometry.Steps[0].DurationS != 90 {
		t.Fatalf("duration=%v", geometry.Steps[0].DurationS)
	}
}

func TestGeometryBuilder_PrefersGoogleOverOSRM(t *testing.T) {
	waypoints := []LatLng{
		{Lat: 41.2995, Lng: 69.2401},
		{Lat: 41.3095, Lng: 69.2501},
	}
	encoded := EncodePolyline(waypoints)

	googleSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"routes":[{"polyline":{"encodedPolyline":"` + encoded + `"}}]}`))
	}))
	defer googleSrv.Close()

	osrmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("OSRM should not be called when Google succeeds")
	}))
	defer osrmSrv.Close()

	builder := NewGeometryBuilder(
		NewGoogleRoutesClient("k", googleSrv.URL, nil),
		NewOSRMClient(osrmSrv.URL, nil),
		RoutingProviderAuto,
	)
	geometry := builder.Build(context.Background(), "r1", waypoints)
	if geometry.Source != googleRoutesSource {
		t.Fatalf("source=%q", geometry.Source)
	}
}

func TestGeometryBuilder_FallsBackToOSRMWhenGoogleFails(t *testing.T) {
	waypoints := []LatLng{
		{Lat: 41.2995, Lng: 69.2401},
		{Lat: 41.3095, Lng: 69.2501},
	}
	encoded := EncodePolyline(waypoints)

	googleSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "quota", http.StatusTooManyRequests)
	}))
	defer googleSrv.Close()

	osrmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":"Ok","routes":[{"geometry":"` + encoded + `","legs":[]}]}`))
	}))
	defer osrmSrv.Close()

	builder := NewGeometryBuilder(
		NewGoogleRoutesClient("k", googleSrv.URL, nil),
		NewOSRMClient(osrmSrv.URL, nil),
		RoutingProviderAuto,
	)
	geometry := builder.Build(context.Background(), "r1", waypoints)
	if geometry.Source != "osrm_driving" {
		t.Fatalf("source=%q", geometry.Source)
	}
}

func TestTrimGoogleWaypoints(t *testing.T) {
	waypoints := make([]LatLng, 40)
	for i := range waypoints {
		waypoints[i] = LatLng{Lat: float64(i), Lng: float64(i)}
	}
	trimmed := trimGoogleWaypoints(waypoints)
	if len(trimmed) != googleRoutesMaxWaypoints {
		t.Fatalf("len=%d want %d", len(trimmed), googleRoutesMaxWaypoints)
	}
	if trimmed[0] != waypoints[0] || trimmed[len(trimmed)-1] != waypoints[len(waypoints)-1] {
		t.Fatalf("origin/destination not preserved")
	}
}

func TestParseRoutingProviderMode(t *testing.T) {
	if ParseRoutingProviderMode("GOOGLE") != RoutingProviderGoogle {
		t.Fatal("google")
	}
	if ParseRoutingProviderMode("osrm") != RoutingProviderOSRM {
		t.Fatal("osrm")
	}
	if ParseRoutingProviderMode("") != RoutingProviderAuto {
		t.Fatal("auto")
	}
}
