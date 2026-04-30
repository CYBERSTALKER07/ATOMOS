package optimizer

import (
	contract "optimizercontract"
)

// twoOpt runs the classic 2-opt swap local-search on a single route. The
// route is improved iff (a) the new sequence shortens total distance AND
// (b) the swap still respects every receiving window. Bounded by
// t.twoOptIterations to keep the solver inside its 2.5 s budget.
//
// The algorithm: for every (i, k) with i < k, reverse the slice [i:k+1] and
// keep the new order if both invariants hold. Repeat until a full pass
// finds no improvement or the iteration budget is exhausted.
func twoOpt(seq []int, stops []contract.Stop, depotLat, depotLng float64, veh contract.Vehicle, t resolvedTunables) []int {
	if len(seq) < 4 {
		return seq
	}
	best := append([]int{}, seq...)
	bestDist := routeDistance(best, stops, depotLat, depotLng)
	iter := 0
	improved := true
	for improved && iter < t.twoOptIterations {
		improved = false
		for i := 1; i < len(best)-2; i++ {
			for k := i + 1; k < len(best)-1; k++ {
				iter++
				if iter >= t.twoOptIterations {
					break
				}
				candidate := append([]int{}, best...)
				reverseSlice(candidate, i, k)
				if !routeRespectsWindows(candidate, stops, veh) {
					continue
				}
				d := routeDistance(candidate, stops, depotLat, depotLng)
				if d < bestDist {
					best = candidate
					bestDist = d
					improved = true
				}
			}
			if iter >= t.twoOptIterations {
				break
			}
		}
	}
	return best
}

func reverseSlice(s []int, i, k int) {
	for i < k {
		s[i], s[k] = s[k], s[i]
		i++
		k--
	}
}

// routeDistance returns the total km of depot → s1 → s2 → ... → sN → depot.
func routeDistance(seq []int, stops []contract.Stop, depotLat, depotLng float64) float64 {
	if len(seq) == 0 {
		return 0
	}
	total := haversineKm(depotLat, depotLng, stops[seq[0]].Lat, stops[seq[0]].Lng)
	for i := 0; i < len(seq)-1; i++ {
		total += haversineKm(
			stops[seq[i]].Lat, stops[seq[i]].Lng,
			stops[seq[i+1]].Lat, stops[seq[i+1]].Lng,
		)
	}
	total += haversineKm(
		stops[seq[len(seq)-1]].Lat, stops[seq[len(seq)-1]].Lng,
		depotLat, depotLng,
	)
	return total
}

// buildRoute materialises a contract.Route from a stop-index sequence.
func buildRoute(seq []int, stops []contract.Stop, depotLat, depotLng float64, veh contract.Vehicle) contract.Route {
	out := contract.Route{
		VehicleID: veh.VehicleID,
		DriverID:  veh.DriverID,
		Stops:     make([]contract.Stop, 0, len(seq)),
	}
	totalVU := 0.0
	speed := veh.AvgSpeedKmph
	if speed <= 0 {
		speed = defaultAvgSpeedKmph
	}
	for _, idx := range seq {
		out.Stops = append(out.Stops, stops[idx])
		totalVU += stops[idx].VolumeVU
	}
	out.TotalVU = totalVU
	if veh.MaxVolumeVU > 0 {
		out.UtilPct = (totalVU / veh.MaxVolumeVU) * 100
	}
	dist := routeDistance(seq, stops, depotLat, depotLng)
	out.DistanceKm = dist
	// Duration = travel + service; travel uses haversine distance and avg speed.
	serviceMin := 0
	for _, idx := range seq {
		serviceMin += stops[idx].ServiceMinutes
	}
	out.DurationMin = int((dist/speed)*60.0) + serviceMin
	return out
}
