package dispatch

import "fmt"

// SplitManifest divides orders for a single driver/truck into chunks of at
// most maxStops. Each chunk gets a unique RouteID with an alphabetical suffix.
// If maxStops <= 0 it defaults to MaxWaypointsPerManifest (25).
func SplitManifest(driverID, truckID string, orders []GeoOrder, maxStops int, routeBase string) ManifestGroup {
	if maxStops <= 0 {
		maxStops = MaxWaypointsPerManifest
	}

	chunks := splitOrdersIntoChunks(orders, maxStops)

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
			suffix = AlphaIndex(i)
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

// SplitManifestAutoRoute generates a route base from the driver ID and
// splits using the standard naming convention.
func SplitManifestAutoRoute(driverID, truckID string, orders []GeoOrder, maxStops int, timestampMillis int64) ManifestGroup {
	routeBase := fmt.Sprintf("AUTO-%s-%d",
		driverID[:min(8, len(driverID))],
		timestampMillis%100000)
	return SplitManifest(driverID, truckID, orders, maxStops, routeBase)
}

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

// AlphaIndex converts a 0-based index to an alphabetical suffix.
// 0→"A", 25→"Z", 26→"AA", ...
func AlphaIndex(i int) string {
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
