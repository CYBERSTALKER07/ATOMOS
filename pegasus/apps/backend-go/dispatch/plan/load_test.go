package plan

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"

	"backend-go/dispatch"
)

// TestLoad_Fallback_1000Orders_20Trucks exercises the Phase 1 fallback
// (nil optimiser client) against a synthetic 1000-order × 20-truck workload
// across 4 H3-cell-like clusters. Asserts:
//   - p95 ≤ 2500 ms (matches the optimiser HTTP timeout budget).
//   - orphan rate ≤ 5 % under normal seeding.
//   - no manifest exceeds 95 % of its truck's MaxVolumeVU.
func TestLoad_Fallback_1000Orders_20Trucks(t *testing.T) {
	if testing.Short() {
		t.Skip("load test skipped in -short mode")
	}

	const (
		runs          = 50
		orderCount    = 1000
		truckCount    = 20
		clusterCount  = 4
		latencyP95Cap = 2500 * time.Millisecond
		orphanRateCap = 0.05
	)

	rng := rand.New(rand.NewSource(20260419))
	orders, fleet := seedSyntheticWorkload(rng, orderCount, truckCount, clusterCount)
	totalVolume := 0.0
	for _, o := range orders {
		totalVolume += o.VolumeVU
	}
	totalCapacity := 0.0
	for _, d := range fleet {
		totalCapacity += d.MaxVolumeVU * maxAcceptableUtilFraction
	}
	t.Logf("workload: orders=%d fleet=%d total_vu=%.2f effective_capacity=%.2f utilization=%.1f%%",
		orderCount, truckCount, totalVolume, totalCapacity, 100*totalVolume/totalCapacity)

	latencies := make([]time.Duration, 0, runs)
	var orphanSum int

	for i := 0; i < runs; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t0 := time.Now()
		result, source, err := OptimizeAndValidate(ctx, nil, Job{
			TraceID:    fmt.Sprintf("load-%d", i),
			SupplierID: "load-supplier",
			Orders:     orders,
			Fleet:      fleet,
		})
		elapsed := time.Since(t0)
		cancel()

		if err != nil {
			t.Fatalf("run %d: OptimizeAndValidate err: %v", i, err)
		}
		if source != SourceFallbackPhase1 {
			t.Fatalf("run %d: expected source=%s, got %s", i, SourceFallbackPhase1, source)
		}

		// Capacity invariant: every route ≤ its DispatchRoute.MaxVolume,
		// which BinPack already sets to driver.MaxVolumeVU * TetrisBuffer (95 %).
		for ri, route := range result.Routes {
			if route.LoadedVolume > route.MaxVolume+1e-6 {
				t.Fatalf("run %d: route #%d (driver %s) overloaded: %.2f > %.2f",
					i, ri, route.DriverID, route.LoadedVolume, route.MaxVolume)
			}
		}

		latencies = append(latencies, elapsed)
		orphanSum += len(result.Orphans)
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p50 := latencies[len(latencies)/2]
	p95 := latencies[(len(latencies)*95)/100]
	p99 := latencies[(len(latencies)*99)/100]
	avgOrphans := float64(orphanSum) / float64(runs)
	orphanRate := avgOrphans / float64(orderCount)

	t.Logf("latency: p50=%s p95=%s p99=%s", p50, p95, p99)
	t.Logf("orphans: avg=%.1f rate=%.2f%%", avgOrphans, 100*orphanRate)

	if p95 > latencyP95Cap {
		t.Errorf("p95 latency %s exceeds cap %s", p95, latencyP95Cap)
	}
	if orphanRate > orphanRateCap {
		t.Errorf("orphan rate %.2f%% exceeds cap %.2f%%", 100*orphanRate, 100*orphanRateCap)
	}
}

// seedSyntheticWorkload generates orderCount orders distributed across
// clusterCount geographic clusters, plus truckCount drivers with mixed
// capacity tiers sized so total effective fleet capacity ~= 1.5× total order
// volume (leaves headroom; orphan rate should stay well under 5 %).
func seedSyntheticWorkload(rng *rand.Rand, orderCount, truckCount, clusterCount int) ([]dispatch.DispatchableOrder, []dispatch.AvailableDriver) {
	// Cluster centres roughly within Tashkent metro bbox.
	type center struct{ lat, lng float64 }
	centers := make([]center, clusterCount)
	for i := range centers {
		centers[i] = center{
			lat: 41.20 + rng.Float64()*0.30,
			lng: 69.10 + rng.Float64()*0.30,
		}
	}

	orders := make([]dispatch.DispatchableOrder, orderCount)
	totalVolume := 0.0
	for i := 0; i < orderCount; i++ {
		c := centers[i%clusterCount]
		// ~3 km spread per cluster (1 deg lat ≈ 111 km).
		lat := c.lat + (rng.Float64()-0.5)*0.05
		lng := c.lng + (rng.Float64()-0.5)*0.05
		vu := 0.5 + rng.Float64()*4.0 // 0.5–4.5 VU per order
		totalVolume += vu
		orders[i] = dispatch.DispatchableOrder{
			OrderID:      fmt.Sprintf("order-%05d", i),
			RetailerID:   fmt.Sprintf("retailer-%04d", i%200),
			RetailerName: fmt.Sprintf("Retailer %d", i%200),
			Amount:       int64(10000 + rng.Intn(50000)),
			Lat:          lat,
			Lng:          lng,
			VolumeVU:     vu,
		}
	}

	// Size fleet so per-truck capacity ≈ 1.5 × (totalVolume / truckCount) to
	// guarantee feasibility headroom.
	targetPerTruck := 1.5 * totalVolume / float64(truckCount)
	fleet := make([]dispatch.AvailableDriver, truckCount)
	for i := 0; i < truckCount; i++ {
		// Tiered: small (0.7×), medium (1.0×), large (1.4×).
		var mult float64
		switch i % 3 {
		case 0:
			mult = 0.7
		case 1:
			mult = 1.0
		default:
			mult = 1.4
		}
		fleet[i] = dispatch.AvailableDriver{
			DriverID:     fmt.Sprintf("driver-%03d", i),
			DriverName:   fmt.Sprintf("Driver %d", i),
			VehicleID:    fmt.Sprintf("vehicle-%03d", i),
			VehicleClass: "TRUCK",
			MaxVolumeVU:  targetPerTruck * mult,
		}
	}
	return orders, fleet
}

// TestValidateAssignment_OverloadedRouteRejected proves the 95 % Tetris-buffer
// guard fires when an optimiser response over-packs a truck. This is the
// post-solve safety net that catches solver bugs (or a malicious / buggy
// AI worker) before an overloaded manifest reaches a driver.
func TestValidateAssignment_OverloadedRouteRejected(t *testing.T) {
	fleet := []dispatch.AvailableDriver{
		{DriverID: "drv-1", VehicleID: "veh-1", MaxVolumeVU: 100.0},
	}
	// Truck cap = 100, 95 % buffer = 95. Pack it to 96 → must be rejected.
	overloaded := &dispatch.AssignmentResult{
		Routes: []dispatch.DispatchRoute{{
			DriverID: "drv-1",
			Orders: []dispatch.GeoOrder{
				{OrderID: "o-1", Volume: 50.0},
				{OrderID: "o-2", Volume: 46.0},
			},
			LoadedVolume: 96.0,
		}},
	}
	if reason := validateAssignment(overloaded, fleet); reason == "" {
		t.Fatalf("expected validateAssignment to reject 96/100 (>95%%) overload, got accepted")
	}

	// Same fleet, packed to exactly the buffer (95) → must pass.
	atBuffer := &dispatch.AssignmentResult{
		Routes: []dispatch.DispatchRoute{{
			DriverID:     "drv-1",
			Orders:       []dispatch.GeoOrder{{OrderID: "o-3", Volume: 95.0}},
			LoadedVolume: 95.0,
		}},
	}
	if reason := validateAssignment(atBuffer, fleet); reason != "" {
		t.Fatalf("expected validateAssignment to accept 95/100 (=95%%), got rejection: %s", reason)
	}
}

// TestValidateAssignment_UnknownDriverRejected covers the second failure mode:
// the optimiser returned a route assigned to a driver not in the fleet. This
// indicates contract drift between the planner and the solver and must never
// reach a manifest write.
func TestValidateAssignment_UnknownDriverRejected(t *testing.T) {
	fleet := []dispatch.AvailableDriver{
		{DriverID: "drv-real", VehicleID: "veh-1", MaxVolumeVU: 100.0},
	}
	ghost := &dispatch.AssignmentResult{
		Routes: []dispatch.DispatchRoute{{
			DriverID:     "drv-ghost",
			Orders:       []dispatch.GeoOrder{{OrderID: "o-1", Volume: 10.0}},
			LoadedVolume: 10.0,
		}},
	}
	if reason := validateAssignment(ghost, fleet); reason == "" {
		t.Fatalf("expected validateAssignment to reject unknown driver, got accepted")
	}
}
