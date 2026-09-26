package dispatch

import "fmt"

// SplitManifest divides orders for a single driver/truck into chunks of at
// most maxStops. Each overflow chunk gets an alphabetical suffix (A/B/…).
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

// SplitManifestAutoRoute generates AUTO-{driver}-{ts} and splits overflow.
func SplitManifestAutoRoute(driverID, truckID string, orders []GeoOrder, maxStops int, timestampMillis int64) ManifestGroup {
	n := 8
	if len(driverID) < n {
		n = len(driverID)
	}
	routeBase := fmt.Sprintf("AUTO-%s-%d", driverID[:n], timestampMillis%100000)
	return SplitManifest(driverID, truckID, orders, maxStops, routeBase)
}

func splitOrdersIntoChunks(orders []GeoOrder, maxSize int) [][]GeoOrder {
	if maxSize <= 0 {
		maxSize = MaxWaypointsPerManifest
	}
	if len(orders) == 0 {
		return nil
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

// ExpandOversizeRoutes splits any route with more than MaxWaypointsPerManifest
// stops into named AUTO-{driver}-{ts}-A/B chunks. Routes within the cap are
// unchanged. Orphans stay orphans — this only names overflow chunks.
func ExpandOversizeRoutes(routes []DispatchRoute, timestampMillis int64) []DispatchRoute {
	if len(routes) == 0 {
		return routes
	}
	out := make([]DispatchRoute, 0, len(routes))
	for _, route := range routes {
		if len(route.Orders) <= MaxWaypointsPerManifest {
			out = append(out, route)
			continue
		}
		group := SplitManifestAutoRoute(route.DriverID, "", route.Orders, MaxWaypointsPerManifest, timestampMillis)
		for _, chunk := range group.Chunks {
			out = append(out, DispatchRoute{
				DriverID:     route.DriverID,
				MaxVolume:    route.MaxVolume,
				LoadedVolume: chunk.VolumeVU,
				Orders:       chunk.Orders,
				SplitGroupID: group.Chunks[0].RouteID,
				RouteID:      chunk.RouteID,
			})
		}
	}
	return out
}
