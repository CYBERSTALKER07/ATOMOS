package supplier

import (
	"fmt"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════════
// RULE OF 25 — MANIFEST SPLITTING MODULE
//
// Google Maps Directions API enforces a hard ceiling of 25 intermediate
// waypoints per request. When the auto-dispatch engine assigns more than 25
// orders to a single truck, this module splits them into a ManifestGroup
// containing multiple ManifestChunks of ≤25 stops each.
//
// This is a formalization of the inline splitting logic previously in
// dispatcher.go. The Rule of 25 is now a first-class, testable module.
// ═══════════════════════════════════════════════════════════════════════════════

// ManifestChunk / ManifestGroup are aliased from dispatch/ via
// dispatch_shim.go — their shape is shared across supplier, warehouse, and
// factory scopes.

// SplitManifest divides orders for a single driver/truck into chunks of at
// most maxStops. Each chunk gets a unique RouteID with an alphabetical suffix
// when splitting occurs (A, B, C, ...). If maxStops <= 0 it defaults to
// MaxWaypointsPerManifest (25).
//
// The function preserves the input order — chunks are sequential slices,
// not reordered. Route optimization happens downstream in the TSP solver.
func SplitManifest(driverID, truckID string, orders []GeoOrder, maxStops int) ManifestGroup {
	if maxStops <= 0 {
		maxStops = MaxWaypointsPerManifest
	}

	chunks := splitOrdersIntoChunks(orders, maxStops)
	routeBase := fmt.Sprintf("AUTO-%s-%d",
		driverID[:min(8, len(driverID))],
		time.Now().UnixMilli()%100000)

	group := ManifestGroup{
		DriverID:    driverID,
		TruckID:     truckID,
		TotalOrders: len(orders),
		Chunks:      make([]ManifestChunk, len(chunks)),
	}

	for i, chunk := range chunks {
		vol := 0.0
		for _, o := range chunk {
			vol += o.Volume
		}

		suffix := ""
		routeID := routeBase
		if len(chunks) > 1 {
			suffix = alphaIndex(i)
			routeID = routeBase + "-" + suffix
		}

		group.Chunks[i] = ManifestChunk{
			RouteID:  routeID,
			Orders:   chunk,
			VolumeVU: vol,
			Suffix:   suffix,
		}
	}

	return group
}

// splitOrdersIntoChunks divides a slice of GeoOrders into chunks of at most
// maxSize elements. Used by the Rule of 25 manifest splitting logic.
func splitOrdersIntoChunks(orders []GeoOrder, maxSize int) [][]GeoOrder {
	if maxSize <= 0 {
		maxSize = 25
	}
	if len(orders) <= maxSize {
		return [][]GeoOrder{orders}
	}
	numChunks := (len(orders) + maxSize - 1) / maxSize
	chunks := make([][]GeoOrder, 0, numChunks)
	for i := 0; i < len(orders); i += maxSize {
		end := i + maxSize
		if end > len(orders) {
			end = len(orders)
		}
		chunks = append(chunks, orders[i:end])
	}
	return chunks
}

// alphaIndex converts a 0-based index to an alphabetical suffix.
// 0→"A", 25→"Z", 26→"AA", 27→"AB", ... 701→"ZZ", 702→"AAA", ...
func alphaIndex(i int) string {
	result := ""
	for {
		result = string(rune('A'+i%26)) + result
		i = i/26 - 1
		if i < 0 {
			break
		}
	}
	return result
}
