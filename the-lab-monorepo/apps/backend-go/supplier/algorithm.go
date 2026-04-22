package supplier

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// ═══════════════════════════════════════════════════════════════════════════════
// PURE DISPATCH ALGORITHMS — Spatial clustering, bin-packing, geo math
//
// This file contains zero Spanner/HTTP dependencies. All functions are pure
// computation over in-memory data, safe for unit testing and reuse.
// ═══════════════════════════════════════════════════════════════════════════════

// ── Spatial Clustering Constants ────────────────────────────────────────────

// kMeansMaxIter caps K-Means iterations to keep dispatch latency bounded.
const kMeansMaxIter = 50

// kMeansConvergenceThreshold — centroids that move less than this (degrees)
// between iterations are considered converged.
const kMeansConvergenceThreshold = 0.0001

// maxStopsPerSegment is the maximum number of waypoints per Google Maps URL.
// Google Maps supports up to 25 waypoints. We use 20 to leave headroom for
// origin/destination plus URL length limits.
const maxStopsPerSegment = 20

// ═══════════════════════════════════════════════════════════════════════════════
// DOMAIN TYPES — Algorithm pipeline data structures
//
// GeoOrder / DispatchRoute are aliased from the canonical dispatch/ package
// via dispatch_shim.go. Only supplier-local types live here.
// ═══════════════════════════════════════════════════════════════════════════════

// ── Vehicle Selection (Consolidation-First) ─────────────────────────────────

// VehicleMatch is the result of selectBestVehicle — contains the matched
// driver/vehicle and whether the order overflows the fleet's capacity.
type VehicleMatch struct {
	Driver   availableDriver
	Overflow bool // true when no single vehicle can contain the order
}

// selectBestVehicle implements the "One Order, One Truck" escalation logic.
// Given a total order volume in VU, it finds the smallest vehicle in the fleet
// whose effective capacity (MaxVolumeVU × TetrisBuffer) can contain the order.
//
// Logic:
//
//	SELECT vehicle_id FROM fleet
//	WHERE effective_capacity >= V_total
//	ORDER BY effective_capacity ASC LIMIT 1
//
// If no single vehicle fits, it returns the largest vehicle and sets Overflow=true.
// Returns (nil, false) only when the fleet is empty.
func selectBestVehicle(orderVolumeVU float64, fleet []availableDriver) (*VehicleMatch, bool) {
	if len(fleet) == 0 {
		return nil, false
	}

	// Sort fleet by effective capacity ASC — smallest first
	sorted := make([]availableDriver, len(fleet))
	copy(sorted, fleet)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].MaxVolumeVU < sorted[j].MaxVolumeVU
	})

	// Find the smallest vehicle that fits
	for _, d := range sorted {
		effectiveCap := d.MaxVolumeVU * TetrisBuffer
		if effectiveCap >= orderVolumeVU {
			return &VehicleMatch{Driver: d, Overflow: false}, true
		}
	}

	// No single vehicle fits — return largest + overflow flag
	largest := sorted[len(sorted)-1]
	return &VehicleMatch{Driver: largest, Overflow: true}, true
}

// computeOrderVolume calculates TotalVolumeVU = Σ(qty_i × vol_i) for a set
// of line items. Uses Kahan compensated summation to minimize floating-point
// drift across large item sets.
func computeOrderVolume(quantities []int, volumes []float64) float64 {
	if len(quantities) != len(volumes) {
		return 0
	}
	var sum, compensation float64
	for i := range quantities {
		y := float64(quantities[i])*volumes[i] - compensation
		t := sum + y
		compensation = (t - sum) - y
		sum = t
	}
	return sum
}

// ═══════════════════════════════════════════════════════════════════════════════
// K-MEANS SPATIAL CLUSTERING (Lloyd's Algorithm)
// ═══════════════════════════════════════════════════════════════════════════════

// kMeansCluster partitions orders into K spatial clusters using Lloyd's algorithm.
// If K >= len(orders), each order gets its own cluster.
func kMeansCluster(orders []GeoOrder, K int) [][]GeoOrder {
	n := len(orders)
	if n == 0 {
		return nil
	}
	if K <= 0 {
		K = 1
	}
	if K > n {
		K = n
	}

	// Initialize centroids using K evenly-spaced orders (deterministic seeding)
	centroids := make([][2]float64, K)
	for i := 0; i < K; i++ {
		idx := i * n / K
		centroids[i] = [2]float64{orders[idx].Lat, orders[idx].Lng}
	}

	assignments := make([]int, n)

	for iter := 0; iter < kMeansMaxIter; iter++ {
		// Assign each order to nearest centroid
		for i, o := range orders {
			bestC := 0
			bestDist := math.MaxFloat64
			for c := 0; c < K; c++ {
				d := haversineKm(o.Lat, o.Lng, centroids[c][0], centroids[c][1])
				if d < bestDist {
					bestDist = d
					bestC = c
				}
			}
			assignments[i] = bestC
		}

		// Recompute centroids
		newCentroids := make([][2]float64, K)
		counts := make([]int, K)
		for i, o := range orders {
			c := assignments[i]
			newCentroids[c][0] += o.Lat
			newCentroids[c][1] += o.Lng
			counts[c]++
		}

		converged := true
		for c := 0; c < K; c++ {
			if counts[c] > 0 {
				newCentroids[c][0] /= float64(counts[c])
				newCentroids[c][1] /= float64(counts[c])
			} else {
				// Empty cluster — keep old centroid
				newCentroids[c] = centroids[c]
			}
			dx := newCentroids[c][0] - centroids[c][0]
			dy := newCentroids[c][1] - centroids[c][1]
			if math.Sqrt(dx*dx+dy*dy) > kMeansConvergenceThreshold {
				converged = false
			}
		}
		centroids = newCentroids

		if converged {
			break
		}
	}

	// Build cluster slices
	clusters := make([][]GeoOrder, K)
	for i := 0; i < K; i++ {
		clusters[i] = []GeoOrder{}
	}
	for i, o := range orders {
		clusters[assignments[i]] = append(clusters[assignments[i]], o)
	}

	return clusters
}

// clusterCentroid returns [lat, lng] center of a set of orders.
func clusterCentroid(orders []GeoOrder) [2]float64 {
	if len(orders) == 0 {
		return [2]float64{0, 0}
	}
	var sumLat, sumLng float64
	for _, o := range orders {
		sumLat += o.Lat
		sumLng += o.Lng
	}
	n := float64(len(orders))
	return [2]float64{sumLat / n, sumLng / n}
}

// routeGeoZone generates a human-readable zone label from the route's centroid.
func routeGeoZone(orders []GeoOrder) string {
	c := clusterCentroid(orders)
	return fmt.Sprintf("%.3f,%.3f", c[0], c[1])
}

// ═══════════════════════════════════════════════════════════════════════════════
// GEO UTILITIES
// ═══════════════════════════════════════════════════════════════════════════════

// haversineKm returns the great-circle distance in kilometers between two points.
func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371.0 // Earth radius km
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// nearestNeighborSort reorders stops in-place using a greedy nearest-neighbour
// traversal starting from originLat/originLng. O(n²) on the input slice; safe
// because Rule of 25 caps n at 25 per manifest. Returns the same backing slice.
func nearestNeighborSort(orders []GeoOrder, originLat, originLng float64) []GeoOrder {
	n := len(orders)
	if n <= 1 {
		return orders
	}
	curLat, curLng := originLat, originLng
	for i := 0; i < n-1; i++ {
		bestJ := i
		bestDist := math.MaxFloat64
		for j := i; j < n; j++ {
			d := haversineKm(curLat, curLng, orders[j].Lat, orders[j].Lng)
			if d < bestDist {
				bestDist = d
				bestJ = j
			}
		}
		if bestJ != i {
			orders[i], orders[bestJ] = orders[bestJ], orders[i]
		}
		curLat, curLng = orders[i].Lat, orders[i].Lng
	}
	return orders
}

// ═══════════════════════════════════════════════════════════════════════════════
// RETAILER-ATOMIC GROUPING
//
// Before K-Means clustering, orders for the same retailer collapse into one
// virtual super-order so the bin-packer treats them as an indivisible unit.
// After force-assignment, each super-order expands back to its child orders
// inheriting the parent's assignment flags. Groups whose combined volume
// exceeds the largest truck's effective capacity stay split — atomicity
// yields only when physically impossible.
// ═══════════════════════════════════════════════════════════════════════════════

// retailerGroupPrefix marks synthetic OrderIDs produced by groupByRetailer so
// they can be identified during expansion. Real OrderIDs are UUIDs and never
// collide with this prefix.
const retailerGroupPrefix = "RGRP-"

// groupOrdersByRetailer collapses orders sharing a RetailerID into a single
// super-order (summed volume, shared lat/lng/window). Returns the collapsed
// slice and a map from synthetic super-order ID to its child orders.
//
// Retailer groups whose summed volume would exceed maxTruckEff are left as
// individual orders — the group cannot fit any single truck, so forcing
// atomicity would strand them.
func groupOrdersByRetailer(orders []GeoOrder, maxTruckEff float64) ([]GeoOrder, map[string][]GeoOrder) {
	if len(orders) == 0 {
		return orders, nil
	}

	byRetailer := make(map[string][]GeoOrder, len(orders))
	keyOrder := make([]string, 0, len(orders))
	for _, o := range orders {
		if _, seen := byRetailer[o.RetailerID]; !seen {
			keyOrder = append(keyOrder, o.RetailerID)
		}
		byRetailer[o.RetailerID] = append(byRetailer[o.RetailerID], o)
	}

	collapsed := make([]GeoOrder, 0, len(orders))
	expansion := make(map[string][]GeoOrder, len(byRetailer))
	for _, rid := range keyOrder {
		group := byRetailer[rid]
		if len(group) == 1 {
			collapsed = append(collapsed, group[0])
			continue
		}
		total := 0.0
		for _, o := range group {
			total += o.Volume
		}
		// Atomicity yields when the group physically cannot fit any truck.
		if maxTruckEff > 0 && total > maxTruckEff {
			collapsed = append(collapsed, group...)
			continue
		}
		head := group[0]
		groupID := retailerGroupPrefix + rid
		collapsed = append(collapsed, GeoOrder{
			OrderID:              groupID,
			RetailerID:           head.RetailerID,
			RetailerName:         head.RetailerName,
			Lat:                  head.Lat,
			Lng:                  head.Lng,
			Volume:               total,
			ReceivingWindowOpen:  head.ReceivingWindowOpen,
			ReceivingWindowClose: head.ReceivingWindowClose,
		})
		expansion[groupID] = group
	}
	return collapsed, expansion
}

// expandRetailerGroups replaces each super-order in the input slice with its
// child orders from the expansion map, propagating Assigned/ForceAssigned/
// CapacityOverflow/LogisticsIsolated flags so downstream Spanner writes and
// orphan reporting see the real OrderIDs. Non-group entries pass through.
func expandRetailerGroups(orders []GeoOrder, expansion map[string][]GeoOrder) []GeoOrder {
	if len(expansion) == 0 || len(orders) == 0 {
		return orders
	}
	out := make([]GeoOrder, 0, len(orders))
	for _, o := range orders {
		children, ok := expansion[o.OrderID]
		if !ok {
			out = append(out, o)
			continue
		}
		for _, c := range children {
			c.Assigned = o.Assigned
			c.ForceAssigned = o.ForceAssigned
			c.CapacityOverflow = o.CapacityOverflow
			c.LogisticsIsolated = o.LogisticsIsolated
			out = append(out, c)
		}
	}
	return out
}

// ═══════════════════════════════════════════════════════════════════════════════
// NAVIGATION URL BUILDERS
// ═══════════════════════════════════════════════════════════════════════════════

// buildNavigationSegments generates Google Maps directions URLs, splitting into
// multiple segments if the route exceeds maxStopsPerSegment. Each segment is a
// self-contained Google Maps URL with ordered waypoints.
//
// If stops ≤ 25: single URL (backward compatible with legacy behavior).
// If stops > 25: split into ceil(N/20) segments of 20 stops each.
func buildNavigationSegments(orders []GeoOrder) []string {
	if len(orders) == 0 {
		return nil
	}

	// Fast path: fits in a single URL (legacy behavior preserved)
	if len(orders) <= 25 {
		return []string{buildSingleNavigationURL(orders)}
	}

	// Multi-segment: split into chunks of maxStopsPerSegment
	var segments []string
	for i := 0; i < len(orders); i += maxStopsPerSegment {
		end := i + maxStopsPerSegment
		if end > len(orders) {
			end = len(orders)
		}
		segments = append(segments, buildSingleNavigationURL(orders[i:end]))
	}
	return segments
}

// buildSingleNavigationURL generates a single Google Maps directions URL.
// Format: https://www.google.com/maps/dir/{stop1}/{stop2}/.../{stopN}
func buildSingleNavigationURL(orders []GeoOrder) string {
	if len(orders) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("https://www.google.com/maps/dir/")
	for i, o := range orders {
		if i > 0 {
			sb.WriteByte('/')
		}
		sb.WriteString(fmt.Sprintf("%.6f,%.6f", o.Lat, o.Lng))
	}
	return sb.String()
}

// ═══════════════════════════════════════════════════════════════════════════════
// TIME WINDOW & ETA HELPERS
// ═══════════════════════════════════════════════════════════════════════════════

// stopETA holds the estimated arrival at a stop in a route.
type stopETA struct {
	ArrivalMinutes float64 // minutes since midnight
	ArrivalStr     string  // "HH:MM"
}

// computeStopETAs estimates arrival time at each sequential stop using
// haversine distance / 30 km/h + 10 min dwell per stop.
// startLat/startLng is the route origin (cluster centroid).
func computeStopETAs(orders []GeoOrder, startLat, startLng float64) []stopETA {
	const avgSpeedKmh = 30.0
	const dwellMinutes = 10.0
	const defaultDepartureMinutes = 480.0 // 08:00

	etas := make([]stopETA, len(orders))
	prevLat, prevLng := startLat, startLng
	cum := defaultDepartureMinutes

	for i, o := range orders {
		dist := haversineKm(prevLat, prevLng, o.Lat, o.Lng)
		cum += (dist / avgSpeedKmh) * 60.0
		h := int(cum) / 60
		m := int(cum) % 60
		etas[i] = stopETA{
			ArrivalMinutes: cum,
			ArrivalStr:     fmt.Sprintf("%02d:%02d", h, m),
		}
		cum += dwellMinutes
		prevLat, prevLng = o.Lat, o.Lng
	}
	return etas
}

// effectiveWindowClose returns the window close time or "23:59" if not set.
func effectiveWindowClose(wc string) string {
	if wc == "" {
		return "23:59"
	}
	return wc
}

// parseTimeHHMM parses "HH:MM" to minutes since midnight. Returns -1 on failure.
func parseTimeHHMM(t string) int {
	if len(t) < 4 {
		return -1
	}
	var h, m int
	if _, err := fmt.Sscanf(t, "%d:%d", &h, &m); err != nil {
		return -1
	}
	return h*60 + m
}

// parseWKTPoint extracts lat/lng from "POINT(lng lat)" WKT string.
func parseWKTPoint(wkt string) (lat, lng float64) {
	wkt = strings.TrimSpace(wkt)
	if wkt == "" {
		return 0, 0
	}
	wkt = strings.TrimPrefix(wkt, "POINT(")
	wkt = strings.TrimSuffix(wkt, ")")
	parts := strings.Fields(wkt)
	if len(parts) != 2 {
		return 0, 0
	}
	fmt.Sscanf(parts[0], "%f", &lng)
	fmt.Sscanf(parts[1], "%f", &lat)
	return lat, lng
}
