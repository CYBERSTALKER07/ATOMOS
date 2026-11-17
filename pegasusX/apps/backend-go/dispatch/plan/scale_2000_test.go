package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	contract "github.com/pegasusx/pegasusx/packages/optimizer-contract"

	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/optimizerclient"
)

func seed2000Outlet40VanWave() ([]dispatch.DispatchableOrder, []dispatch.AvailableDriver) {
	rng := rand.New(rand.NewSource(20261113))
	const (
		outletCount = 2000
		vanCount    = 40
	)
	orders := make([]dispatch.DispatchableOrder, outletCount)
	totalVol := 0.0
	for i := 0; i < outletCount; i++ {
		vu := 1.0 + rng.Float64()*2.0
		totalVol += vu
		orders[i] = dispatch.DispatchableOrder{
			OrderID:      fmt.Sprintf("w-ord-%04d", i),
			RetailerID:   fmt.Sprintf("w-ret-%04d", i),
			RetailerName: fmt.Sprintf("Shop %d", i),
			Lat:          41.20 + rng.Float64()*0.25,
			Lng:          69.10 + rng.Float64()*0.25,
			VolumeVU:     vu,
			TotalMinor:   int64(250_000 + rng.Intn(750_000)),
		}
	}
	// Plenty of volume capacity so stop-count and partitioning are the primary tests.
	vanCap := (totalVol / float64(vanCount)) * 1.6
	fleet := make([]dispatch.AvailableDriver, vanCount)
	for i := 0; i < vanCount; i++ {
		fleet[i] = dispatch.AvailableDriver{
			DriverID:     fmt.Sprintf("w-drv-%02d", i),
			DriverName:   fmt.Sprintf("Driver %02d", i),
			VehicleID:    fmt.Sprintf("w-veh-%02d", i),
			VehicleClass: "VAN",
			StartLat:     41.311,
			StartLng:     69.240,
			MaxVolumeVU:  vanCap,
		}
	}
	return orders, fleet
}

func newParallelWaveMockServer(t testing.TB, enforceMaxPartition int, trackMaxInFlight *atomic.Int32) *httptest.Server {
	t.Helper()
	var inFlight atomic.Int32
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cur := inFlight.Add(1)
		defer inFlight.Add(-1)
		if trackMaxInFlight != nil {
			for {
				prev := trackMaxInFlight.Load()
				if cur <= prev || trackMaxInFlight.CompareAndSwap(prev, cur) {
					break
				}
			}
		}
		// Tiny simulated solver latency to observe genuine worker parallelism.
		time.Sleep(10 * time.Millisecond)

		raw, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(raw) >= (4 << 20) {
			http.Error(w, "payload exceeds 4 MiB", http.StatusRequestEntityTooLarge)
			return
		}
		var req contract.SolveRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(req.Stops) > enforceMaxPartition {
			http.Error(w, fmt.Sprintf("subproblem stops %d exceeds %d", len(req.Stops), enforceMaxPartition), http.StatusBadRequest)
			return
		}
		if req.Tunables == nil || req.Tunables.MaxStopsPerRoute <= 0 {
			http.Error(w, "missing explicit max_stops_per_route", http.StatusBadRequest)
			return
		}
		if len(req.DistanceMatrixM) != len(req.Vehicles)+len(req.Stops) {
			http.Error(w, "matrix dimension mismatch", http.StatusBadRequest)
			return
		}

		maxStops := req.Tunables.MaxStopsPerRoute
		routes := make([]contract.Route, len(req.Vehicles))
		for i, v := range req.Vehicles {
			routes[i] = contract.Route{
				VehicleID: v.VehicleID,
				DriverID:  v.DriverID,
			}
		}

		var orphans []contract.Orphan
		vCursor := 0
		for _, s := range req.Stops {
			placed := false
			for attempt := 0; attempt < len(routes); attempt++ {
				vi := (vCursor + attempt) % len(routes)
				maxCap := req.Vehicles[vi].MaxVolumeVU * maxAcceptableUtilFraction
				if len(routes[vi].Stops) < maxStops && routes[vi].TotalVU+s.VolumeVU <= maxCap {
					routes[vi].Stops = append(routes[vi].Stops, contract.Stop{
						OrderID:    s.OrderID,
						RetailerID: s.RetailerID,
						Lat:        s.Lat,
						Lng:        s.Lng,
						VolumeVU:   s.VolumeVU,
					})
					routes[vi].TotalVU += s.VolumeVU
					placed = true
					vCursor = (vi + 1) % len(routes)
					break
				}
			}
			if !placed {
				orphans = append(orphans, contract.Orphan{
					OrderID: s.OrderID,
					Reason:  "max_stops_per_route",
				})
			}
		}

		nonEmpty := make([]contract.Route, 0, len(routes))
		placedCount := 0
		for i, rt := range routes {
			if len(rt.Stops) == 0 {
				continue
			}
			capVU := req.Vehicles[i].MaxVolumeVU
			if capVU > 0 {
				rt.UtilPct = (rt.TotalVU / capVU) * 100.0
			}
			placedCount += len(rt.Stops)
			nonEmpty = append(nonEmpty, rt)
		}

		resp := contract.SolveResponse{
			V:       contract.V,
			TraceID: req.TraceID,
			Source:  "OR_TOOLS_VRP",
			Routes:  nonEmpty,
			Orphans: orphans,
			Stats: contract.Stats{
				ElapsedMs:       10,
				StopsConsidered: len(req.Stops),
				StopsPlaced:     placedCount,
				StopsOrphaned:   len(orphans),
				VehiclesUsed:    len(nonEmpty),
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestScale_2000Outlets_40Vans_ParallelOptimizer(t *testing.T) {
	orders, fleet := seed2000Outlet40VanWave()
	var maxInFlight atomic.Int32
	srv := newParallelWaveMockServer(t, dispatch.MaxPartitionStops, &maxInFlight)
	defer srv.Close()

	cli := optimizerclient.New(srv.URL, "scale-key")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t0 := time.Now()
	res, source, err := OptimizeAndValidate(ctx, cli, Job{
		TraceID:          "scale-2000-40",
		SupplierID:       "sup-scale",
		DepotLat:         41.311,
		DepotLng:         69.240,
		Orders:           orders,
		Fleet:            fleet,
		MaxStopsPerRoute: 60,
	})
	elapsed := time.Since(t0)

	if err != nil {
		t.Fatalf("OptimizeAndValidate: %v", err)
	}
	if source != SourceOptimizer {
		t.Fatalf("expected source=%s, got %s (warnings=%v)", SourceOptimizer, source, res.Warnings)
	}
	if elapsed > solverBudget {
		t.Fatalf("solve took %s > solverBudget %s", elapsed, solverBudget)
	}
	if maxInFlight.Load() < 2 {
		t.Fatalf("expected parallel optimizerclient worker execution (max_in_flight >= 2), got %d", maxInFlight.Load())
	}

	placed := 0
	seenDrivers := make(map[string]bool, len(fleet))
	for _, r := range res.Routes {
		if len(r.Orders) > 60 {
			t.Fatalf("route driver=%s has %d stops > MaxStopsPerRoute=60", r.DriverID, len(r.Orders))
		}
		if seenDrivers[r.DriverID] {
			t.Fatalf("duplicate driver %s in merged routes", r.DriverID)
		}
		seenDrivers[r.DriverID] = true
		placed += len(r.Orders)
	}
	if placed != len(orders) || len(res.Orphans) != 0 {
		t.Fatalf("expected all %d orders placed cleanly with 60 stops/van cap, got placed=%d orphans=%d", len(orders), placed, len(res.Orphans))
	}

	// Deterministic overflow deferral check: tight MaxStopsPerRoute=40 across 40 vans = 1600 max routable -> 400 clean orphans.
	resOverflow, sourceOverflow, err := OptimizeAndValidate(ctx, cli, Job{
		TraceID:          "scale-2000-40-overflow",
		SupplierID:       "sup-scale",
		DepotLat:         41.311,
		DepotLng:         69.240,
		Orders:           orders,
		Fleet:            fleet,
		MaxStopsPerRoute: 40,
	})
	if err != nil || sourceOverflow != SourceOptimizer {
		t.Fatalf("overflow run err=%v source=%s", err, sourceOverflow)
	}
	overflowPlaced := 0
	for _, r := range resOverflow.Routes {
		if len(r.Orders) > 40 {
			t.Fatalf("overflow route driver=%s has %d > 40 stops", r.DriverID, len(r.Orders))
		}
		overflowPlaced += len(r.Orders)
	}
	if overflowPlaced+len(resOverflow.Orphans) != len(orders) {
		t.Fatalf("accounting mismatch: placed=%d + orphans=%d != %d", overflowPlaced, len(resOverflow.Orphans), len(orders))
	}
	if len(resOverflow.Orphans) < 400 {
		t.Fatalf("expected >= 400 deferred overflow stops at 40x40 cap, got %d", len(resOverflow.Orphans))
	}
	for i := 1; i < len(resOverflow.Orphans); i++ {
		if resOverflow.Orphans[i-1].OrderID >= resOverflow.Orphans[i].OrderID {
			t.Fatalf("orphans not deterministically sorted at index %d", i)
		}
	}
}

func BenchmarkOptimizeAndValidate_2000Outlets_40Vans(b *testing.B) {
	orders, fleet := seed2000Outlet40VanWave()
	srv := newParallelWaveMockServer(b, dispatch.MaxPartitionStops, nil)
	defer srv.Close()
	cli := optimizerclient.New(srv.URL, "bench-key")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, source, err := OptimizeAndValidate(context.Background(), cli, Job{
			TraceID:          fmt.Sprintf("bench-%d", i),
			SupplierID:       "sup-bench",
			DepotLat:         41.311,
			DepotLng:         69.240,
			Orders:           orders,
			Fleet:            fleet,
			MaxStopsPerRoute: 60,
		})
		if err != nil || source != SourceOptimizer || res == nil {
			b.Fatalf("bench failed: err=%v source=%s", err, source)
		}
	}
}
