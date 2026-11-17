package payload

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func TestSequenceLIFOReversesStops(t *testing.T) {
	got := SequenceLIFO(VehicleRoute{VehicleID: "veh-1", LocationIDs: []string{"a", "b", "c"}})
	if len(got) != 3 || got[0] != "c" || got[1] != "b" || got[2] != "a" {
		t.Fatalf("sequence = %#v", got)
	}
	if empty := SequenceLIFO(VehicleRoute{VehicleID: "veh-1"}); empty != nil {
		t.Fatalf("expected nil for empty stops, got %#v", empty)
	}
}

func TestHandleGetLIFOSequence_RepoUnavailable503(t *testing.T) {
	rr := httptest.NewRecorder()
	(&Service{}).HandleGetLIFOSequence(rr, httptest.NewRequest(http.MethodGet, "/v1/payload/sequence/veh-1", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("nil service repo status = %d want 503 body=%s", rr.Code, rr.Body.String())
	}

	rr2 := httptest.NewRecorder()
	svcMem := NewService(ServiceConfig{Repo: NewInMemoryRepository()})
	svcMem.HandleGetLIFOSequence(rr2, httptest.NewRequest(http.MethodGet, "/v1/payload/sequence/veh-1", nil))
	if rr2.Code != http.StatusServiceUnavailable {
		t.Fatalf("in-memory repo status = %d want 503 body=%s", rr2.Code, rr2.Body.String())
	}

	rr3 := httptest.NewRecorder()
	svcNilClient := NewService(ServiceConfig{Repo: NewSpannerRepository(nil, "sup-1", "")})
	svcNilClient.HandleGetLIFOSequence(rr3, httptest.NewRequest(http.MethodGet, "/v1/payload/sequence/veh-1", nil))
	if rr3.Code != http.StatusServiceUnavailable {
		t.Fatalf("nil spanner client status = %d want 503 body=%s", rr3.Code, rr3.Body.String())
	}
}

type stubLIFORepo struct {
	Repository
	fn func(ctx context.Context, supplierID, id string) (LIFORouteLookup, error)
}

func (s *stubLIFORepo) LookupVehicleRoute(ctx context.Context, supplierID, id string) (LIFORouteLookup, error) {
	return s.fn(ctx, supplierID, id)
}

func TestHandleGetLIFOSequence_MockReaderFound200AndNotFound404(t *testing.T) {
	repo := &stubLIFORepo{
		Repository: NewInMemoryRepository(),
		fn: func(_ context.Context, _ string, id string) (LIFORouteLookup, error) {
			if id == "truck-alpha" {
				return LIFORouteLookup{
					VehicleID:  "truck-alpha",
					RouteID:    "route-alpha",
					ManifestID: "mf-alpha",
					StopIDs:    []string{"ord-1", "ord-2", "ord-3"},
				}, nil
			}
			return LIFORouteLookup{}, ErrRouteNotFound
		},
	}
	svc := NewService(ServiceConfig{Repo: repo, SeedSupplierID: "sup-1"})

	r := chi.NewRouter()
	r.Get("/v1/payload/sequence/{truckID}", svc.HandleGetLIFOSequence)

	// Missing route -> 404
	rr404 := httptest.NewRecorder()
	r.ServeHTTP(rr404, httptest.NewRequest(http.MethodGet, "/v1/payload/sequence/truck-missing", nil))
	if rr404.Code != http.StatusNotFound {
		t.Fatalf("missing route status = %d want 404 body=%s", rr404.Code, rr404.Body.String())
	}

	// Found route -> 200 with LIFO reversed order
	rr200 := httptest.NewRecorder()
	r.ServeHTTP(rr200, httptest.NewRequest(http.MethodGet, "/v1/payload/sequence/truck-alpha", nil))
	if rr200.Code != http.StatusOK {
		t.Fatalf("found route status = %d want 200 body=%s", rr200.Code, rr200.Body.String())
	}
	var payload struct {
		VehicleID       string   `json:"vehicle_id"`
		RouteID         string   `json:"route_id"`
		ManifestID      string   `json:"manifest_id"`
		Stops           []string `json:"stops"`
		LoadingSequence []string `json:"loading_sequence"`
	}
	if err := json.Unmarshal(rr200.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.VehicleID != "truck-alpha" || payload.RouteID != "route-alpha" || payload.ManifestID != "mf-alpha" {
		t.Fatalf("unexpected metadata: %+v", payload)
	}
	if len(payload.LoadingSequence) != 3 || payload.LoadingSequence[0] != "ord-3" || payload.LoadingSequence[1] != "ord-2" || payload.LoadingSequence[2] != "ord-1" {
		t.Fatalf("unexpected loading_sequence: %+v", payload.LoadingSequence)
	}
}

func openPayloadSpannerTestClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping Spanner payload LIFO integration test")
	}
	envOr := func(k, def string) string {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
		return def
	}
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		envOr("SPANNER_PROJECT", "pegasusx-local"),
		envOr("SPANNER_INSTANCE", "pegasusx-instance"),
		envOr("SPANNER_DATABASE", "pegasusx-db"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := spanner.NewClient(ctx, dbPath,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		if requireSpanner {
			t.Fatalf("spanner emulator connect: %v", err)
		}
		t.Skipf("spanner emulator unavailable: %v", err)
	}
	_, probeErr := client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "SupplierTruckManifests", spanner.Key{"__probe__"}, []string{"ManifestId"})
	if probeErr != nil && spanner.ErrCode(probeErr) != codes.NotFound {
		client.Close()
		if requireSpanner {
			t.Fatalf("spanner SupplierTruckManifests unavailable: %v", probeErr)
		}
		t.Skipf("spanner tables unavailable: %v", probeErr)
	}
	t.Cleanup(client.Close)
	return client
}

func TestHandleGetLIFOSequence_LiveSpanner(t *testing.T) {
	client := openPayloadSpannerTestClient(t)
	ctx := context.Background()
	suffix := time.Now().UnixNano()
	supID := fmt.Sprintf("sup-lifo-%d", suffix)
	manifestID := fmt.Sprintf("mf-lifo-%d", suffix)
	truckID := fmt.Sprintf("truck-lifo-%d", suffix)
	routeID := fmt.Sprintf("route-lifo-%d", suffix)
	rtOnlyRouteID := fmt.Sprintf("rt-only-%d", suffix)
	now := time.Now().UTC()

	muts := []*spanner.Mutation{
		spanner.InsertMap("SupplierTruckManifests", map[string]any{
			"ManifestId":    manifestID,
			"SupplierId":    supID,
			"RouteId":       routeID,
			"TruckId":       truckID,
			"DriverId":      "drv-lifo-1",
			"State":         "LOADING",
			"TotalVolumeVU": float64(30),
			"MaxVolumeVU":   float64(100),
			"StopCount":     int64(3),
			"ReplanCount":   int64(0),
			"CreatedAt":     now,
			"UpdatedAt":     now,
		}),
		spanner.InsertMap("ManifestOrders", map[string]any{
			"ManifestId":    manifestID,
			"OrderId":       "ord-stop-2",
			"SequenceIndex": int64(2),
			"LoadingOrder":  int64(2),
			"VolumeVU":      float64(10),
			"State":         "LOADED",
			"UpdatedAt":     now,
		}),
		spanner.InsertMap("ManifestOrders", map[string]any{
			"ManifestId":    manifestID,
			"OrderId":       "ord-stop-1",
			"SequenceIndex": int64(1),
			"LoadingOrder":  int64(1),
			"VolumeVU":      float64(10),
			"State":         "LOADED",
			"UpdatedAt":     now,
		}),
		spanner.InsertMap("ManifestOrders", map[string]any{
			"ManifestId":    manifestID,
			"OrderId":       "ord-stop-3",
			"SequenceIndex": int64(3),
			"LoadingOrder":  int64(3),
			"VolumeVU":      float64(10),
			"State":         "LOADED",
			"UpdatedAt":     now,
		}),
		spanner.InsertMap("RouteTwins", map[string]any{
			"RouteId":        rtOnlyRouteID,
			"SupplierId":     supID,
			"DriverId":       "drv-rt-1",
			"Status":         "ACTIVE",
			"RemainingStops": int64(2),
			"LastEventAt":    now,
			"UpdatedAt":      now,
		}),
		spanner.InsertMap("StopTwins", map[string]any{
			"RouteId":   rtOnlyRouteID,
			"StopId":    "stop-b",
			"Sequence":  int64(2),
			"Status":    "PENDING",
			"UpdatedAt": now,
		}),
		spanner.InsertMap("StopTwins", map[string]any{
			"RouteId":   rtOnlyRouteID,
			"StopId":    "stop-a",
			"Sequence":  int64(1),
			"Status":    "PENDING",
			"UpdatedAt": now,
		}),
	}
	if _, err := client.Apply(ctx, muts); err != nil {
		t.Fatalf("seed manifest/route twins: %v", err)
	}
	t.Cleanup(func() {
		_, _ = client.Apply(context.Background(), []*spanner.Mutation{
			spanner.Delete("SupplierTruckManifests", spanner.Key{manifestID}),
			spanner.Delete("RouteTwins", spanner.Key{rtOnlyRouteID}),
		})
	})

	repo := NewSpannerRepository(client, supID, "")
	svc := NewService(ServiceConfig{Repo: repo, SeedSupplierID: supID})
	r := chi.NewRouter()
	r.Get("/v1/payload/sequence/{truckID}", svc.HandleGetLIFOSequence)

	// 1. Lookup by TruckId via SupplierTruckManifests + ManifestOrders
	rrTruck := httptest.NewRecorder()
	r.ServeHTTP(rrTruck, httptest.NewRequest(http.MethodGet, "/v1/payload/sequence/"+truckID, nil))
	if rrTruck.Code != http.StatusOK {
		t.Fatalf("truck lookup status = %d body=%s", rrTruck.Code, rrTruck.Body.String())
	}
	var resp1 struct {
		VehicleID       string   `json:"vehicle_id"`
		RouteID         string   `json:"route_id"`
		ManifestID      string   `json:"manifest_id"`
		LoadingSequence []string `json:"loading_sequence"`
	}
	if err := json.Unmarshal(rrTruck.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("decode truck response: %v", err)
	}
	if resp1.VehicleID != truckID || resp1.ManifestID != manifestID || resp1.RouteID != routeID {
		t.Fatalf("unexpected truck lookup response: %+v", resp1)
	}
	if len(resp1.LoadingSequence) != 3 || resp1.LoadingSequence[0] != "ord-stop-3" || resp1.LoadingSequence[1] != "ord-stop-2" || resp1.LoadingSequence[2] != "ord-stop-1" {
		t.Fatalf("unexpected LIFO sequence from manifest: %+v", resp1.LoadingSequence)
	}

	// 2. Fallback lookup via RouteTwins + StopTwins
	rrRT := httptest.NewRecorder()
	r.ServeHTTP(rrRT, httptest.NewRequest(http.MethodGet, "/v1/payload/sequence/"+rtOnlyRouteID, nil))
	if rrRT.Code != http.StatusOK {
		t.Fatalf("route twin lookup status = %d body=%s", rrRT.Code, rrRT.Body.String())
	}
	var resp2 struct {
		LoadingSequence []string `json:"loading_sequence"`
	}
	if err := json.Unmarshal(rrRT.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("decode route twin response: %v", err)
	}
	if len(resp2.LoadingSequence) != 2 || resp2.LoadingSequence[0] != "stop-b" || resp2.LoadingSequence[1] != "stop-a" {
		t.Fatalf("unexpected LIFO sequence from route twin: %+v", resp2.LoadingSequence)
	}

	// 3. Unknown truck -> 404
	rr404 := httptest.NewRecorder()
	r.ServeHTTP(rr404, httptest.NewRequest(http.MethodGet, "/v1/payload/sequence/unknown-truck-999", nil))
	if rr404.Code != http.StatusNotFound {
		t.Fatalf("unknown truck status = %d want 404 body=%s", rr404.Code, rr404.Body.String())
	}
}
