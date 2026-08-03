package dispatch

import "math"

// ImproveRoutesLocalSearch runs 2-opt (then bounded 3-opt) on each route's stop order.
// Pure: never changes capacity load, membership, or atomic retailer grouping beyond order of stops.
// Tour length is non-increasing.
func ImproveRoutesLocalSearch(result *AssignmentResult, ctx ScoreContext) {
	if result == nil {
		return
	}
	depotLat, depotLng := ctx.DepotLat, ctx.DepotLng
	for i := range result.Routes {
		result.Routes[i].Orders = twoOptTour(result.Routes[i].Orders, depotLat, depotLng)
		result.Routes[i].Orders = threeOptBounded(result.Routes[i].Orders, depotLat, depotLng, 24)
	}
}

// ResequenceStops applies nearest-neighbor seed + 2-opt for continuous replan (remaining stops only).
func ResequenceStops(stops []GeoOrder, depotLat, depotLng float64) []GeoOrder {
	if len(stops) <= 2 {
		return stops
	}
	seeded := nearestNeighborTour(stops, depotLat, depotLng)
	return twoOptTour(seeded, depotLat, depotLng)
}

func nearestNeighborTour(stops []GeoOrder, depotLat, depotLng float64) []GeoOrder {
	n := len(stops)
	if n == 0 {
		return stops
	}
	remaining := make([]GeoOrder, n)
	copy(remaining, stops)
	out := make([]GeoOrder, 0, n)
	lat, lng := depotLat, depotLng
	for len(remaining) > 0 {
		best := 0
		bestD := math.Inf(1)
		for i, s := range remaining {
			d := haversineKm(lat, lng, s.Lat, s.Lng)
			if d < bestD {
				bestD = d
				best = i
			}
		}
		out = append(out, remaining[best])
		lat, lng = remaining[best].Lat, remaining[best].Lng
		remaining = append(remaining[:best], remaining[best+1:]...)
	}
	return out
}

func twoOptTour(stops []GeoOrder, depotLat, depotLng float64) []GeoOrder {
	n := len(stops)
	if n < 4 {
		return stops
	}
	best := make([]GeoOrder, n)
	copy(best, stops)
	bestLen := tourLength(best, depotLat, depotLng)
	improved := true
	for improved {
		improved = false
		for i := 0; i < n-2; i++ {
			for j := i + 2; j < n; j++ {
				if i == 0 && j == n-1 {
					continue
				}
				cand := twoOptSwap(best, i, j)
				candLen := tourLength(cand, depotLat, depotLng)
				if candLen+1e-9 < bestLen {
					best = cand
					bestLen = candLen
					improved = true
				}
			}
		}
	}
	return best
}

func twoOptSwap(stops []GeoOrder, i, j int) []GeoOrder {
	out := make([]GeoOrder, len(stops))
	copy(out, stops[:i+1])
	// reverse i+1..j
	for k := 0; k <= j-(i+1); k++ {
		out[i+1+k] = stops[j-k]
	}
	copy(out[j+1:], stops[j+1:])
	return out
}

func threeOptBounded(stops []GeoOrder, depotLat, depotLng float64, maxIters int) []GeoOrder {
	n := len(stops)
	if n < 6 || maxIters <= 0 {
		return stops
	}
	best := make([]GeoOrder, n)
	copy(best, stops)
	bestLen := tourLength(best, depotLat, depotLng)
	iters := 0
	for a := 0; a < n-4 && iters < maxIters; a++ {
		for b := a + 2; b < n-2 && iters < maxIters; b++ {
			for c := b + 2; c < n && iters < maxIters; c++ {
				iters++
				// One useful 3-opt reconnection: reverse middle segment b+1..c
				cand := twoOptSwap(best, b, c)
				candLen := tourLength(cand, depotLat, depotLng)
				if candLen+1e-9 < bestLen {
					best = cand
					bestLen = candLen
				}
			}
		}
	}
	return best
}

func tourLength(stops []GeoOrder, depotLat, depotLng float64) float64 {
	if len(stops) == 0 {
		return 0
	}
	total := haversineKm(depotLat, depotLng, stops[0].Lat, stops[0].Lng)
	for i := 1; i < len(stops); i++ {
		total += haversineKm(stops[i-1].Lat, stops[i-1].Lng, stops[i].Lat, stops[i].Lng)
	}
	total += haversineKm(stops[len(stops)-1].Lat, stops[len(stops)-1].Lng, depotLat, depotLng)
	return total
}
