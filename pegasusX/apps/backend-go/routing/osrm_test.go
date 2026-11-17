package routing

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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

func TestOSRMClient_DistanceMatrix(t *testing.T) {
	points := []LatLng{
		{Lat: 41.2995, Lng: 69.2401},
		{Lat: 41.3095, Lng: 69.2501},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/table/v1/driving/") {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if !strings.Contains(r.URL.RawQuery, "annotations=distance") {
			t.Fatalf("expected annotations=distance, got %s", r.URL.RawQuery)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if len(body) >= (4 << 20) {
			t.Fatalf("payload too large: %d", len(body))
		}
		var req osrmTablePostRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}
		if len(req.Coordinates) != 2 || len(req.Sources) != 2 || len(req.Destinations) != 2 {
			t.Fatalf("unexpected request payload: %+v", req)
		}
		_, _ = w.Write([]byte(`{"code":"Ok","distances":[[0,1500],[1600,0]]}`))
	}))
	defer server.Close()

	client := NewOSRMClient(server.URL, circuit.New("osrm-table-test", circuit.Config{}))
	matrix, err := client.DistanceMatrix(context.Background(), points)
	if err != nil {
		t.Fatalf("DistanceMatrix: %v", err)
	}
	if matrix[0][1] != 1500 || matrix[1][0] != 1600 {
		t.Fatalf("matrix=%v", matrix)
	}
}

func TestOSRMClient_DistanceMatrix_ChunkedPOST(t *testing.T) {
	const n = 125
	points := make([]LatLng, n)
	for i := range points {
		points[i] = LatLng{
			Lat: 41.20 + float64(i)*0.001,
			Lng: 69.10 + float64(i)*0.002,
		}
	}

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if len(raw) >= (4 << 20) {
			t.Errorf("request body %d exceeds 4 MiB", len(raw))
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		var req osrmTablePostRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			t.Errorf("unmarshal body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if len(req.Coordinates) > osrmMaxSingleTableCoords {
			t.Errorf("chunk coordinates %d exceeds %d", len(req.Coordinates), osrmMaxSingleTableCoords)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		requests.Add(1)

		dist := make([][]float64, len(req.Sources))
		for i, si := range req.Sources {
			sc := req.Coordinates[si]
			row := make([]float64, len(req.Destinations))
			for j, di := range req.Destinations {
				dc := req.Coordinates[di]
				// Deterministic asymmetric distance encoding coordinates.
				dLat := math.Round((dc[1] - sc[1]) * 1000)
				dLng := math.Round((dc[0] - sc[0]) * 1000)
				if dLat == 0 && dLng == 0 {
					row[j] = 0
				} else {
					row[j] = math.Abs(dLat)*10 + math.Abs(dLng)*20 + 1
				}
			}
			dist[i] = row
		}
		_ = json.NewEncoder(w).Encode(osrmTableResponse{Code: "Ok", Distances: dist})
	}))
	defer server.Close()

	client := NewOSRMClient(server.URL, nil)
	matrix, err := client.DistanceMatrix(context.Background(), points)
	if err != nil {
		t.Fatalf("DistanceMatrix: %v", err)
	}
	if requests.Load() != 9 {
		t.Fatalf("expected 9 chunked POST calls (3x3 of 50), got %d", requests.Load())
	}
	if len(matrix) != n || len(matrix[0]) != n {
		t.Fatalf("unexpected dimensions %dx%d", len(matrix), len(matrix[0]))
	}
	for i := 0; i < n; i++ {
		if matrix[i][i] != 0 {
			t.Fatalf("diagonal [%d][%d] = %d, want 0", i, i, matrix[i][i])
		}
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			want := int(math.Abs(float64(j-i))*10 + math.Abs(float64(j-i)*2)*20 + 1)
			if matrix[i][j] != want {
				t.Fatalf("matrix[%d][%d] = %d, want %d", i, j, matrix[i][j], want)
			}
		}
	}
}

func TestOSRMClient_DistanceMatrix_BodyBoundBisection(t *testing.T) {
	points := make([]LatLng, 6)
	for i := range points {
		points[i] = LatLng{Lat: 41.3 + float64(i)*0.001, Lng: 69.2 + float64(i)*0.001}
	}
	var maxSeen atomic.Int64
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		for {
			cur := maxSeen.Load()
			if int64(len(raw)) <= cur || maxSeen.CompareAndSwap(cur, int64(len(raw))) {
				break
			}
		}
		calls.Add(1)
		var req osrmTablePostRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		dist := make([][]float64, len(req.Sources))
		for i, si := range req.Sources {
			sc := req.Coordinates[si]
			dist[i] = make([]float64, len(req.Destinations))
			for j, di := range req.Destinations {
				dc := req.Coordinates[di]
				// Return -1 for point[0]->point[0] to verify negative -> 0 normalization.
				if sc[1] == points[0].Lat && dc[1] == points[0].Lat {
					dist[i][j] = -1
				} else {
					dist[i][j] = 100
				}
			}
		}
		_ = json.NewEncoder(w).Encode(osrmTableResponse{Code: "Ok", Distances: dist})
	}))
	defer server.Close()

	client := &OSRMClient{
		baseURL:           server.URL,
		tableChunkCoords:  6,
		maxTableBodyBytes: 220, // Force recursive bisection well below 6 coords
	}
	matrix, err := client.DistanceMatrix(context.Background(), points)
	if err != nil {
		t.Fatalf("DistanceMatrix: %v", err)
	}
	if calls.Load() <= 1 {
		t.Fatalf("expected recursive bisection to trigger multiple POST calls, got %d", calls.Load())
	}
	if maxSeen.Load() >= 220 {
		t.Fatalf("saw payload %d bytes >= 220 limit", maxSeen.Load())
	}
	if len(matrix) != 6 || matrix[0][0] != 0 {
		t.Fatalf("unexpected matrix=%v", matrix)
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
