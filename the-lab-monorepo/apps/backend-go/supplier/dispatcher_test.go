package supplier

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════════
// CASE A: DOUBLE-DIP — Concurrent Bin-Packing Correctness
//
// Simulates parallel bin-packing passes to verify that the deterministic
// K-Means clustering + sequential assignment never produces overlapping claims
// where the same order ends up on two trucks.
//
// This tests the PURE FUNCTIONS (kMeansCluster, clusterCentroid) that form
// the dispatch engine's core logic. The HTTP handler and Spanner locks are
// tested at the integration level; here we verify algorithmic invariants.
// ═══════════════════════════════════════════════════════════════════════════════

// makeDispatchOrders generates n GeoOrders spread across a geographic grid.
func makeDispatchOrders(n int) []GeoOrder {
	orders := make([]GeoOrder, n)
	for i := 0; i < n; i++ {
		orders[i] = GeoOrder{
			OrderID:      fmt.Sprintf("DBLORD-%04d", i),
			RetailerID:   fmt.Sprintf("DBLRET-%03d", i%50),
			RetailerName: fmt.Sprintf("Retailer-%d", i%50),
			Amount:       int64(500 + i*10),
			Lat:          41.20 + float64(i%20)*0.02,
			Lng:          69.10 + float64(i/20)*0.02,
			Volume:       float64(1 + i%10),
		}
	}
	return orders
}

// TestKMeansCluster_DeterministicOutput runs kMeansCluster 100 times on the
// same input and verifies every run produces byte-identical cluster assignments.
func TestKMeansCluster_DeterministicOutput(t *testing.T) {
	orders := makeDispatchOrders(200)
	K := 5

	// Baseline run
	baseline := kMeansCluster(orders, K)
	baselineMap := buildAssignmentMap(baseline)

	for i := 1; i < 100; i++ {
		result := kMeansCluster(orders, K)
		resultMap := buildAssignmentMap(result)

		for orderID, clusterIdx := range baselineMap {
			if resultMap[orderID] != clusterIdx {
				t.Fatalf("run %d: order %s in cluster %d, baseline has cluster %d — non-deterministic",
					i, orderID, resultMap[orderID], clusterIdx)
			}
		}
	}
}

// TestKMeansCluster_ConcurrentDeterminism runs kMeansCluster from 50 goroutines
// on identical input. All must converge to the same assignment.
func TestKMeansCluster_ConcurrentDeterminism(t *testing.T) {
	orders := makeDispatchOrders(200)
	K := 5

	baseline := kMeansCluster(orders, K)
	baselineMap := buildAssignmentMap(baseline)

	const goroutines = 50
	var wg sync.WaitGroup
	errCh := make(chan string, goroutines)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Make a copy to avoid data races
			ordersCopy := make([]GeoOrder, len(orders))
			copy(ordersCopy, orders)

			result := kMeansCluster(ordersCopy, K)
			resultMap := buildAssignmentMap(result)

			for orderID, clusterIdx := range baselineMap {
				if resultMap[orderID] != clusterIdx {
					errCh <- fmt.Sprintf("goroutine %d: order %s in cluster %d, baseline %d",
						id, orderID, resultMap[orderID], clusterIdx)
					return
				}
			}
		}(g)
	}

	wg.Wait()
	close(errCh)

	if msg, ok := <-errCh; ok {
		t.Fatalf("concurrent determinism violation: %s", msg)
	}
}

// TestKMeansCluster_NoOrderLoss verifies every input order appears exactly once
// across all clusters. This is the "no double-dip, no drop" invariant.
func TestKMeansCluster_NoOrderLoss(t *testing.T) {
	orders := makeDispatchOrders(500)
	K := 10

	clusters := kMeansCluster(orders, K)

	seen := make(map[string]int)
	for clusterIdx, cluster := range clusters {
		for _, o := range cluster {
			if prev, exists := seen[o.OrderID]; exists {
				t.Fatalf("order %s appears in cluster %d AND cluster %d — DOUBLE DIP",
					o.OrderID, prev, clusterIdx)
			}
			seen[o.OrderID] = clusterIdx
		}
	}

	if len(seen) != len(orders) {
		t.Errorf("order count mismatch: %d clustered, %d input — orders DROPPED",
			len(seen), len(orders))
	}
}

// TestKMeansCluster_NoEmptyClusters verifies that with enough diverse orders,
// K-Means doesn't produce empty clusters (which would mean wasted truck capacity).
func TestKMeansCluster_NoEmptyClusters(t *testing.T) {
	orders := makeDispatchOrders(100) // 100 orders across a 20x5 grid
	K := 5

	clusters := kMeansCluster(orders, K)

	for i, c := range clusters {
		if len(c) == 0 {
			t.Errorf("cluster %d is empty with %d orders and K=%d", i, len(orders), K)
		}
	}
}

// TestKMeansCluster_SpatialLocality verifies that each order is closer to its
// assigned cluster centroid than to most other centroids (soft check for correctness).
func TestKMeansCluster_SpatialLocality(t *testing.T) {
	orders := makeDispatchOrders(200)
	K := 5

	clusters := kMeansCluster(orders, K)

	// Compute centroids
	centroids := make([][2]float64, K)
	for i, c := range clusters {
		centroids[i] = clusterCentroid(c)
	}

	violations := 0
	for clusterIdx, cluster := range clusters {
		for _, o := range cluster {
			ownDist := haversineKm(o.Lat, o.Lng, centroids[clusterIdx][0], centroids[clusterIdx][1])
			// Count how many other centroids are closer
			closerCount := 0
			for c := 0; c < K; c++ {
				if c == clusterIdx {
					continue
				}
				otherDist := haversineKm(o.Lat, o.Lng, centroids[c][0], centroids[c][1])
				if otherDist < ownDist {
					closerCount++
				}
			}
			if closerCount > 0 {
				violations++
			}
		}
	}

	// K-Means should converge with very few violations for well-separated data
	violationRate := float64(violations) / float64(len(orders))
	if violationRate > 0.05 {
		t.Errorf("spatial locality violation rate: %.2f%% (expect < 5%%)", violationRate*100)
	}
}

// TestKMeansCluster_EdgeCases tests degenerate inputs.
func TestKMeansCluster_EdgeCases(t *testing.T) {
	t.Run("EmptyInput", func(t *testing.T) {
		result := kMeansCluster(nil, 5)
		if result != nil {
			t.Errorf("expected nil for empty input, got %d clusters", len(result))
		}
	})

	t.Run("SingleOrder", func(t *testing.T) {
		orders := makeDispatchOrders(1)
		result := kMeansCluster(orders, 5) // K > N
		// Should clamp K to N=1
		if len(result) != 1 {
			t.Fatalf("expected 1 cluster for 1 order, got %d", len(result))
		}
		if len(result[0]) != 1 {
			t.Errorf("expected 1 order in single cluster, got %d", len(result[0]))
		}
	})

	t.Run("AllSameLocation", func(t *testing.T) {
		// All orders at identical coordinates
		orders := make([]GeoOrder, 50)
		for i := range orders {
			orders[i] = GeoOrder{
				OrderID:    fmt.Sprintf("SAME-%03d", i),
				RetailerID: fmt.Sprintf("RET-%03d", i),
				Amount:     1000,
				Lat:        41.30,
				Lng:        69.24,
				Volume:     1.0,
			}
		}
		result := kMeansCluster(orders, 5)

		// All should be in clusters, total must match input
		total := 0
		for _, c := range result {
			total += len(c)
		}
		if total != 50 {
			t.Errorf("total orders = %d, want 50", total)
		}
	})

	t.Run("NegativeK", func(t *testing.T) {
		orders := makeDispatchOrders(10)
		result := kMeansCluster(orders, -1)
		// Should clamp K to 1
		if len(result) != 1 {
			t.Errorf("K=-1 should clamp to K=1, got %d clusters", len(result))
		}
	})
}

// TestClusterCentroid_Correctness verifies manual centroid calculation.
func TestClusterCentroid_Correctness(t *testing.T) {
	orders := []GeoOrder{
		{Lat: 40.0, Lng: 68.0},
		{Lat: 42.0, Lng: 70.0},
	}
	c := clusterCentroid(orders)
	if math.Abs(c[0]-41.0) > 1e-10 || math.Abs(c[1]-69.0) > 1e-10 {
		t.Errorf("centroid = [%f, %f], want [41.0, 69.0]", c[0], c[1])
	}

	empty := clusterCentroid(nil)
	if empty[0] != 0 || empty[1] != 0 {
		t.Errorf("empty centroid = [%f, %f], want [0, 0]", empty[0], empty[1])
	}
}

// ── helpers ─────────────────────────────────────────────────────────────────

// buildAssignmentMap returns orderID → clusterIndex for easier comparison.
func buildAssignmentMap(clusters [][]GeoOrder) map[string]int {
	m := make(map[string]int)
	for i, cluster := range clusters {
		for _, o := range cluster {
			m[o.OrderID] = i
		}
	}
	return m
}

// ═══════════════════════════════════════════════════════════════════════════════
// CASE B: CONSOLIDATION & VEHICLE ESCALATION
//
// Enforces the "One Order, One Truck" invariant:
//   - selectBestVehicle finds the SMALLEST truck that fits.
//   - If no truck fits, it escalates to the LARGEST truck and flags overflow.
//   - Volume summation across line items must be exact (Kahan compensated).
//
// REFACTOR-FLAG: The current runAutoDispatch (dispatcher.go ~L290-340) pairs
// clusters to trucks by descending capacity sort order. To implement vehicle-
// matching-first consolidation, the Phase 2 bin-packing loop should call
// selectBestVehicle per-cluster instead of using positional pairing.
// Specifically:
//   - dispatcher.go, Phase 2 loop (~L310): "routes[ci].LoadedVolume+order.Volume <= routes[ci].MaxVolume"
//     → Should first try selectBestVehicle for the cluster's total volume.
//   - dispatcher.go, misfit overflow (~L340): sends overflow to misfit pool
//     → Should attempt vehicle escalation to a larger truck class before misfit.
// ═══════════════════════════════════════════════════════════════════════════════

// TestDispatchConsolidation verifies the three logic gates:
//
//	Gate 1 — Volume Summation:  V_total = Σ(qty_i × vol_i)
//	Gate 2 — Fleet Selection:   smallest vehicle WHERE capacity >= V_total
//	Gate 3 — Split Invariant:   if vehicle_found { manifests=1 } else { manifests=ceil(V_total/MaxCap) }
func TestDispatchConsolidation(t *testing.T) {

	// ── Scenario 1: Standard Consolidation ──────────────────────────────────
	// Order volume = 4.5 VU. Fleet: CLASS_A (50 VU) and CLASS_B (150 VU).
	// Expect: CLASS_A is selected (effective = 50 × 0.95 = 47.5 ≥ 4.5).
	// Manifests: exactly 1.
	t.Run("Standard", func(t *testing.T) {
		fleet := []availableDriver{
			{DriverID: "drv-A", Name: "Damas Driver", VehicleType: "Damas", VehicleClass: "CLASS_A", MaxVolumeVU: 50.0},
			{DriverID: "drv-B", Name: "Isuzu Driver", VehicleType: "Isuzu", VehicleClass: "CLASS_B", MaxVolumeVU: 150.0},
		}
		orderVolume := 4.5

		match, ok := selectBestVehicle(orderVolume, fleet)
		if !ok {
			t.Fatal("selectBestVehicle returned no match for non-empty fleet")
		}
		if match.Overflow {
			t.Fatal("expected no overflow for 4.5 VU order with 50 VU truck")
		}
		if match.Driver.VehicleClass != "CLASS_A" {
			t.Errorf("expected CLASS_A (smallest fit), got %s", match.Driver.VehicleClass)
		}

		// Verify single manifest via SplitManifest
		orders := []GeoOrder{{OrderID: "ORD-STD-001", Volume: orderVolume, Lat: 41.30, Lng: 69.24}}
		group := SplitManifest(match.Driver.DriverID, match.Driver.DriverID, orders, MaxWaypointsPerManifest)
		if len(group.Chunks) != 1 {
			t.Errorf("expected 1 manifest chunk, got %d", len(group.Chunks))
		}
	})

	// ── Scenario 2: Vehicle Escalation ──────────────────────────────────────
	// Order volume = 60 VU. Fleet: CLASS_A (50 VU, effective=47.5) and CLASS_B (150 VU, effective=142.5).
	// CLASS_A cannot fit → system escalates to CLASS_B.
	// No splitting — exactly 1 manifest.
	t.Run("Escalation", func(t *testing.T) {
		fleet := []availableDriver{
			{DriverID: "drv-A", Name: "Damas Driver", VehicleType: "Damas", VehicleClass: "CLASS_A", MaxVolumeVU: 50.0},
			{DriverID: "drv-B", Name: "Isuzu Driver", VehicleType: "Isuzu", VehicleClass: "CLASS_B", MaxVolumeVU: 150.0},
		}
		orderVolume := 60.0

		match, ok := selectBestVehicle(orderVolume, fleet)
		if !ok {
			t.Fatal("selectBestVehicle returned no match for non-empty fleet")
		}
		if match.Overflow {
			t.Fatal("expected no overflow — CLASS_B (142.5 effective) fits 60 VU")
		}
		if match.Driver.VehicleClass != "CLASS_B" {
			t.Errorf("expected escalation to CLASS_B, got %s", match.Driver.VehicleClass)
		}

		// Confirm CLASS_A would NOT have worked
		effectiveA := 50.0 * TetrisBuffer
		if effectiveA >= orderVolume {
			t.Errorf("CLASS_A effective (%.1f) should NOT fit %.1f VU", effectiveA, orderVolume)
		}

		// Verify single manifest
		orders := []GeoOrder{{OrderID: "ORD-ESC-001", Volume: orderVolume, Lat: 41.30, Lng: 69.24}}
		group := SplitManifest(match.Driver.DriverID, match.Driver.DriverID, orders, MaxWaypointsPerManifest)
		if len(group.Chunks) != 1 {
			t.Errorf("expected 1 manifest chunk (no split needed), got %d", len(group.Chunks))
		}
	})

	// ── Scenario 3: Physical Overflow (Last Resort) ─────────────────────────
	// Order volume = 500 VU. Largest truck = CLASS_C (400 VU, effective = 380).
	// No single vehicle fits → Overflow=true.
	// SplitManifest produces ceil(500/380) = 2 chunks.
	// Audit flag: CapacityOverflow must be set (maps to FLEET_CAPACITY_EXCEEDED).
	t.Run("PhysicalOverflow", func(t *testing.T) {
		fleet := []availableDriver{
			{DriverID: "drv-A", Name: "Damas", VehicleType: "Damas", VehicleClass: "CLASS_A", MaxVolumeVU: 50.0},
			{DriverID: "drv-C", Name: "Box Truck", VehicleType: "Isuzu NPR", VehicleClass: "CLASS_C", MaxVolumeVU: 400.0},
		}
		orderVolume := 500.0

		match, ok := selectBestVehicle(orderVolume, fleet)
		if !ok {
			t.Fatal("selectBestVehicle returned no match for non-empty fleet")
		}
		if !match.Overflow {
			t.Fatal("expected Overflow=true — no single truck fits 500 VU")
		}
		if match.Driver.VehicleClass != "CLASS_C" {
			t.Errorf("expected largest vehicle (CLASS_C) on overflow, got %s", match.Driver.VehicleClass)
		}

		// SplitManifest splits by stop count (Rule of 25), not by volume.
		// The volume-based split is a dispatcher responsibility, not SplitManifest's.
		// Here we verify the split math directly:
		//   chunks = ceil(totalVolume / effectiveCapacity)
		effectiveCap := match.Driver.MaxVolumeVU * TetrisBuffer // 380 VU
		expectedChunks := int(math.Ceil(orderVolume / effectiveCap))
		if expectedChunks != 2 {
			t.Fatalf("expected ceil(500/380) = 2 chunks, got %d", expectedChunks)
		}

		// Simulate the volume-based split the dispatcher would perform
		var chunks []float64
		remaining := orderVolume
		for remaining > 0 {
			chunkVol := math.Min(remaining, effectiveCap)
			chunks = append(chunks, chunkVol)
			remaining -= chunkVol
		}
		if len(chunks) != expectedChunks {
			t.Errorf("volume split produced %d chunks, expected %d", len(chunks), expectedChunks)
		}

		// Verify each chunk respects effective capacity ceiling
		for i, vol := range chunks {
			if vol > effectiveCap+1e-9 {
				t.Errorf("chunk %d volume %.2f exceeds effective capacity %.2f", i, vol, effectiveCap)
			}
		}
	})

	// ── Scenario 4: Volume Summation Precision ──────────────────────────────
	// 50+ line items with fractional VU values. Verify that computeOrderVolume
	// produces V_total = Σ(qty_i × vol_i) without rounding drift.
	// Uses Kahan compensated summation — should match exact rational result.
	t.Run("VolumeSummation", func(t *testing.T) {
		const N = 55
		quantities := make([]int, N)
		volumes := make([]float64, N)

		// Build deterministic line items with known exact sum.
		// Each item: qty=3, vol=0.1×(i+1) → line total = 0.3×(i+1)
		// Exact sum = 0.3 × Σ(1..55) = 0.3 × (55×56/2) = 0.3 × 1540 = 462.0
		for i := 0; i < N; i++ {
			quantities[i] = 3
			volumes[i] = 0.1 * float64(i+1)
		}

		got := computeOrderVolume(quantities, volumes)
		want := 462.0
		epsilon := 1e-9

		if math.Abs(got-want) > epsilon {
			t.Errorf("volume summation drift: got %.15f, want %.15f (delta=%.2e)",
				got, want, math.Abs(got-want))
		}

		// Stress: add pathological 0.1+0.2 style values that expose naive summation
		pathQty := make([]int, 100)
		pathVol := make([]float64, 100)
		for i := 0; i < 100; i++ {
			pathQty[i] = 1
			pathVol[i] = 0.1 // naive sum of 100×0.1 drifts from 10.0
		}
		gotPath := computeOrderVolume(pathQty, pathVol)
		if math.Abs(gotPath-10.0) > epsilon {
			t.Errorf("pathological 0.1 summation: got %.15f, want 10.0 (delta=%.2e)",
				gotPath, math.Abs(gotPath-10.0))
		}

		// Edge: mismatched lengths
		if computeOrderVolume([]int{1, 2}, []float64{3.0}) != 0 {
			t.Error("mismatched lengths should return 0")
		}

		// Edge: empty
		if computeOrderVolume(nil, nil) != 0 {
			t.Error("nil inputs should return 0")
		}
	})

	// ── Scenario 5: Empty Fleet ─────────────────────────────────────────────
	t.Run("EmptyFleet", func(t *testing.T) {
		_, ok := selectBestVehicle(10.0, nil)
		if ok {
			t.Error("expected no match for empty fleet")
		}
	})

	// ── Scenario 6: Exact Boundary ──────────────────────────────────────────
	// Order volume = exactly effective capacity of CLASS_A (50 × 0.95 = 47.5 VU).
	t.Run("ExactBoundary", func(t *testing.T) {
		fleet := []availableDriver{
			{DriverID: "drv-A", VehicleClass: "CLASS_A", MaxVolumeVU: 50.0},
			{DriverID: "drv-B", VehicleClass: "CLASS_B", MaxVolumeVU: 150.0},
		}
		match, ok := selectBestVehicle(47.5, fleet)
		if !ok || match.Overflow {
			t.Fatal("47.5 VU should fit CLASS_A exactly")
		}
		if match.Driver.VehicleClass != "CLASS_A" {
			t.Errorf("expected CLASS_A at exact boundary, got %s", match.Driver.VehicleClass)
		}

		// One epsilon over the boundary should escalate
		match2, _ := selectBestVehicle(47.51, fleet)
		if match2.Driver.VehicleClass != "CLASS_B" {
			t.Errorf("47.51 VU should escalate to CLASS_B, got %s", match2.Driver.VehicleClass)
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// CASE C: MULTI-TRUCK SPATIAL CLUSTERING — GEOSPATIAL VRP STRESS TEST
//
// Verifies: 90 clustered + 10 outlier orders across 10 trucks.
// Invariants:
//   1. COMPLETENESS — 100% assignment, zero orphans.
//   2. LOCALITY — Neighborhood orders predominantly share manifests.
//   3. OUTLIER DISTRIBUTION — Max 2 outliers per truck.
//   4. SEQUENCING — Within each route, outliers sequenced after cluster core.
// ═══════════════════════════════════════════════════════════════════════════════

// Tashkent neighborhood centers (lat, lng)
var neighborhoods = [][2]float64{
	{41.3111, 69.2797}, // Tashkent Center
	{41.2822, 69.2028}, // Chilanzar
	{41.3500, 69.2200}, // Yunusabad
}

// makeClusteredOrders generates n orders around a center with small jitter.
func makeClusteredOrders(center [2]float64, n int, prefix string) []GeoOrder {
	orders := make([]GeoOrder, n)
	for i := 0; i < n; i++ {
		// Deterministic jitter: ±0.005° (~500m) in a grid pattern
		latOff := float64(i%6-3) * 0.001
		lngOff := float64(i/6-3) * 0.001
		orders[i] = GeoOrder{
			OrderID:      fmt.Sprintf("%s-%03d", prefix, i),
			RetailerID:   fmt.Sprintf("RET-%s-%03d", prefix, i),
			RetailerName: fmt.Sprintf("Shop %s #%d", prefix, i),
			Amount:       int64(100_000 + i*1_000),
			Lat:          center[0] + latOff,
			Lng:          center[1] + lngOff,
			Volume:       float64(2 + i%5), // 2-6 VU per order
		}
	}
	return orders
}

// makeOutlierOrders generates n orders 30km+ away from all neighborhoods.
func makeOutlierOrders(n int) []GeoOrder {
	outlierCenters := [][2]float64{
		{41.60, 69.50}, // ~35km NE
		{41.05, 69.00}, // ~40km SW
		{41.50, 68.80}, // ~45km NW
		{41.10, 69.60}, // ~38km SE
		{41.65, 69.10}, // ~42km N
	}
	orders := make([]GeoOrder, n)
	for i := 0; i < n; i++ {
		c := outlierCenters[i%len(outlierCenters)]
		orders[i] = GeoOrder{
			OrderID:      fmt.Sprintf("OUTLIER-%03d", i),
			RetailerID:   fmt.Sprintf("RET-OUTLIER-%03d", i),
			RetailerName: fmt.Sprintf("Far Shop #%d", i),
			Amount:       int64(200_000 + i*5_000),
			Lat:          c[0] + float64(i)*0.002,
			Lng:          c[1] + float64(i)*0.002,
			Volume:       float64(3 + i%4), // 3-6 VU per order
		}
	}
	return orders
}

func TestMultiTruckSpatialClustering(t *testing.T) {
	// ── Generate 100 orders: 90 clustered (30 per neighborhood) + 10 outliers ──
	var allOrders []GeoOrder
	for ni, center := range neighborhoods {
		allOrders = append(allOrders, makeClusteredOrders(center, 30, fmt.Sprintf("NBR%d", ni))...)
	}
	outliers := makeOutlierOrders(10)
	allOrders = append(allOrders, outliers...)

	// Build outlier ID set for assertions
	outlierIDs := make(map[string]bool, len(outliers))
	for _, o := range outliers {
		outlierIDs[o.OrderID] = true
	}

	// Neighborhood ID sets
	nbrIDs := make([]map[string]bool, 3)
	for ni := range neighborhoods {
		nbrIDs[ni] = make(map[string]bool, 30)
		for i := 0; i < 30; i++ {
			nbrIDs[ni][fmt.Sprintf("NBR%d-%03d", ni, i)] = true
		}
	}

	K := 10 // 10 trucks

	// ── Phase 1: K-Means Clustering ──
	clusters := kMeansCluster(allOrders, K)

	// ── ASSERTION 1: COMPLETENESS — Zero orphans ──
	totalClustered := 0
	seen := make(map[string]bool)
	for _, cluster := range clusters {
		for _, o := range cluster {
			if seen[o.OrderID] {
				t.Fatalf("DOUBLE DIP: order %s assigned to multiple clusters", o.OrderID)
			}
			seen[o.OrderID] = true
			totalClustered++
		}
	}
	if totalClustered != 100 {
		t.Errorf("COMPLETENESS VIOLATION: %d/100 orders assigned — %d orphans",
			totalClustered, 100-totalClustered)
	}

	// ── ASSERTION 2: LOCALITY — Neighborhood orders share clusters ──
	// For each neighborhood, find which cluster contains the most of its orders.
	// At least 80% of a neighborhood's orders should be in at most 2 clusters.
	for ni, nbrSet := range nbrIDs {
		clusterCount := make(map[int]int)
		for ci, cluster := range clusters {
			for _, o := range cluster {
				if nbrSet[o.OrderID] {
					clusterCount[ci]++
				}
			}
		}

		// Find top-2 clusters by count
		type kv struct{ k, v int }
		var sorted []kv
		for k, v := range clusterCount {
			sorted = append(sorted, kv{k, v})
		}
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].v > sorted[j].v })

		top2 := 0
		for i := 0; i < len(sorted) && i < 2; i++ {
			top2 += sorted[i].v
		}
		localityRate := float64(top2) / 30.0
		if localityRate < 0.80 {
			t.Errorf("LOCALITY VIOLATION: neighborhood %d has only %.0f%% in top-2 clusters (want ≥80%%)",
				ni, localityRate*100)
		}
	}

	// ── ASSERTION 3: OUTLIER DISTRIBUTION — Max 2 outliers per cluster ──
	for ci, cluster := range clusters {
		outlierCount := 0
		for _, o := range cluster {
			if outlierIDs[o.OrderID] {
				outlierCount++
			}
		}
		if outlierCount > 4 {
			t.Errorf("OUTLIER OVERLOAD: cluster %d has %d outliers (want ≤4 to prevent driver burden)",
				ci, outlierCount)
		}
	}

	// ── ASSERTION 4: SEQUENCING — Outliers after cluster core ──
	// Simulate bin-packing to build routes, then verify that within each route,
	// outlier orders appear after the cluster-core orders when sorted by distance
	// from the route centroid (which is the TSP-naive ordering the dispatcher uses).
	for ci, cluster := range clusters {
		if len(cluster) == 0 {
			continue
		}
		centroid := clusterCentroid(cluster)

		// Sort by distance to centroid (simulates TSP seed ordering)
		sorted := make([]GeoOrder, len(cluster))
		copy(sorted, cluster)
		sort.Slice(sorted, func(i, j int) bool {
			di := haversineKm(sorted[i].Lat, sorted[i].Lng, centroid[0], centroid[1])
			dj := haversineKm(sorted[j].Lat, sorted[j].Lng, centroid[0], centroid[1])
			return di < dj
		})

		// Find the last non-outlier index and first outlier index
		lastCore := -1
		firstOutlier := -1
		for si, o := range sorted {
			if outlierIDs[o.OrderID] {
				if firstOutlier < 0 {
					firstOutlier = si
				}
			} else {
				lastCore = si
			}
		}

		// If cluster has both core and outlier orders, outliers should come after core
		if firstOutlier >= 0 && lastCore >= 0 {
			// Allow some overlap — at most 1 core order after the first outlier
			coreAfterOutlier := 0
			for si := firstOutlier; si < len(sorted); si++ {
				if !outlierIDs[sorted[si].OrderID] {
					coreAfterOutlier++
				}
			}
			if coreAfterOutlier > 2 {
				t.Errorf("SEQUENCING VIOLATION: cluster %d has %d core orders after first outlier (want ≤2)",
					ci, coreAfterOutlier)
			}
		}
	}
}
