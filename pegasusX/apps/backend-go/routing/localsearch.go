package routing

import "math"

// geoStop is a minimal stop for pure resequence (mirrors dispatch.GeoOrder fields used in 2-opt).
type geoStop struct {
	OrderID string
	Lat     float64
	Lng     float64
}

// resequenceStops applies nearest-neighbor + 2-opt (same algorithm as dispatch.ResequenceStops).
// Duplicated here to avoid import cycle: dispatch → order → manifest → routing → dispatch.
func resequenceStops(stops []geoStop, depotLat, depotLng float64) []geoStop {
	if len(stops) <= 2 {
		return stops
	}
	seeded := nearestNeighborTour(stops, depotLat, depotLng)
	return twoOptTour(seeded, depotLat, depotLng)
}

func nearestNeighborTour(stops []geoStop, depotLat, depotLng float64) []geoStop {
	n := len(stops)
	remaining := make([]geoStop, n)
	copy(remaining, stops)
	out := make([]geoStop, 0, n)
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

func twoOptTour(stops []geoStop, depotLat, depotLng float64) []geoStop {
	n := len(stops)
	if n < 4 {
		return stops
	}
	best := make([]geoStop, n)
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

func twoOptSwap(stops []geoStop, i, j int) []geoStop {
	out := make([]geoStop, len(stops))
	copy(out, stops[:i+1])
	for k := 0; k <= j-(i+1); k++ {
		out[i+1+k] = stops[j-k]
	}
	copy(out[j+1:], stops[j+1:])
	return out
}

func tourLength(stops []geoStop, depotLat, depotLng float64) float64 {
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

func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const r = 6371.0
	toRad := math.Pi / 180
	dLat := (lat2 - lat1) * toRad
	dLng := (lng2 - lng1) * toRad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*toRad)*math.Cos(lat2*toRad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return r * c
}
