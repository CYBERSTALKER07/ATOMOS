package optimizer

import (
	"sort"
	"strconv"
	"strings"

	contract "optimizercontract"
)

// planRoutes is the Clarke-Wright Savings driver. Returns assigned routes and
// any stops that could not be placed in any vehicle (orphans).
func planRoutes(stops []contract.Stop, vehicles []contract.Vehicle, t resolvedTunables) ([]contract.Route, []contract.Orphan) {
	if len(stops) == 0 || len(vehicles) == 0 {
		orphans := make([]contract.Orphan, 0, len(stops))
		for _, s := range stops {
			orphans = append(orphans, contract.Orphan{OrderID: s.OrderID, Reason: "no vehicles"})
		}
		return nil, orphans
	}

	// Use the largest vehicle as the depot anchor for savings computation.
	// Initial route per vehicle: empty. We assign stops greedily by trying
	// the smallest-fit vehicle first (Tetris discipline).
	depot := vehicles[len(vehicles)-1]
	depotLat, depotLng := depot.StartLat, depot.StartLng

	type savingsPair struct {
		i, j   int
		saving float64
	}

	// 1. Compute savings(i, j) = d(depot,i) + d(depot,j) - d(i,j).
	pairs := make([]savingsPair, 0, len(stops)*len(stops)/2)
	depotDist := make([]float64, len(stops))
	for i, s := range stops {
		depotDist[i] = haversineKm(depotLat, depotLng, s.Lat, s.Lng)
	}
	for i := 0; i < len(stops); i++ {
		for j := i + 1; j < len(stops); j++ {
			d := haversineKm(stops[i].Lat, stops[i].Lng, stops[j].Lat, stops[j].Lng)
			s := depotDist[i] + depotDist[j] - d
			// Recovery-order priority boost: overflow-bounced orders carry
			// Priority > 0 so they cluster first and get first dibs on volume.
			s += float64(stops[i].Priority + stops[j].Priority)
			pairs = append(pairs, savingsPair{i: i, j: j, saving: s})
		}
	}
	sort.Slice(pairs, func(a, b int) bool {
		if pairs[a].saving != pairs[b].saving {
			return pairs[a].saving > pairs[b].saving
		}
		// Stable tie-break on lexicographic OrderID pair so output is
		// deterministic across solver runs with identical input.
		ai, aj := stops[pairs[a].i].OrderID, stops[pairs[a].j].OrderID
		bi, bj := stops[pairs[b].i].OrderID, stops[pairs[b].j].OrderID
		if ai != bi {
			return ai < bi
		}
		return aj < bj
	})

	// 2. routeOf[i] = index into routes slice; -1 = unassigned.
	routes := make([][]int, 0)
	routeOf := make([]int, len(stops))
	for i := range routeOf {
		routeOf[i] = -1
	}

	// vehicleOf[r] = vehicle index assigned to that route (-1 = unset until
	// the first capacity-fit check on append).
	vehicleOf := make([]int, 0)
	routeVU := make([]float64, 0)

	tryAssignVehicle := func(routeIdx int, addVU float64) bool {
		// Smallest-fit vehicle whose effective capacity covers the new total.
		need := routeVU[routeIdx] + addVU
		// If the route already has a vehicle, just check capacity.
		if vehicleOf[routeIdx] >= 0 {
			cap := vehicles[vehicleOf[routeIdx]].MaxVolumeVU * t.tetrisBuffer
			return need <= cap
		}
		for vi, v := range vehicles {
			if v.MaxVolumeVU*t.tetrisBuffer >= need {
				vehicleOf[routeIdx] = vi
				return true
			}
		}
		return false
	}

	newRoute := func(stopIdx int) bool {
		// New route, no vehicle yet — assign smallest-fit at creation.
		for vi, v := range vehicles {
			if v.MaxVolumeVU*t.tetrisBuffer >= stops[stopIdx].VolumeVU {
				routes = append(routes, []int{stopIdx})
				vehicleOf = append(vehicleOf, vi)
				routeVU = append(routeVU, stops[stopIdx].VolumeVU)
				routeOf[stopIdx] = len(routes) - 1
				return true
			}
		}
		return false
	}

	// 3. Process savings descending: try to merge i and j into a single route.
	for _, p := range pairs {
		i, j := p.i, p.j
		ri, rj := routeOf[i], routeOf[j]

		// Both unassigned: seed a brand-new route with [i, j] if it fits.
		if ri == -1 && rj == -1 {
			combinedVU := stops[i].VolumeVU + stops[j].VolumeVU
			placed := false
			for vi, v := range vehicles {
				if v.MaxVolumeVU*t.tetrisBuffer >= combinedVU {
					routes = append(routes, []int{i, j})
					vehicleOf = append(vehicleOf, vi)
					routeVU = append(routeVU, combinedVU)
					routeOf[i] = len(routes) - 1
					routeOf[j] = len(routes) - 1
					placed = true
					break
				}
			}
			if !placed {
				// Fall through — try as two separate routes below.
				newRoute(i)
				newRoute(j)
			}
			continue
		}

		// One assigned: append the other to that route's tail/head if it
		// keeps capacity, window, and stop-count constraints intact.
		if ri == -1 || rj == -1 {
			var routeIdx, addIdx int
			if ri >= 0 {
				routeIdx, addIdx = ri, j
			} else {
				routeIdx, addIdx = rj, i
			}
			if len(routes[routeIdx]) >= t.maxStopsPerRoute {
				continue
			}
			if !tryAssignVehicle(routeIdx, stops[addIdx].VolumeVU) {
				continue
			}
			if !routeRespectsWindows(append(append([]int{}, routes[routeIdx]...), addIdx), stops, vehicles[vehicleOf[routeIdx]]) {
				continue
			}
			routes[routeIdx] = append(routes[routeIdx], addIdx)
			routeVU[routeIdx] += stops[addIdx].VolumeVU
			routeOf[addIdx] = routeIdx
			continue
		}

		// Both assigned and on the same route — nothing to do.
		if ri == rj {
			continue
		}
		// Different routes: try to merge if combined capacity + windows fit.
		merged := append(append([]int{}, routes[ri]...), routes[rj]...)
		if len(merged) > t.maxStopsPerRoute {
			continue
		}
		mergedVU := routeVU[ri] + routeVU[rj]
		// Pick the smaller-of-the-two vehicle slots; let smallest-fit re-search.
		// Reset vehicleOf and re-resolve.
		var targetVehicle int = -1
		for vi, v := range vehicles {
			if v.MaxVolumeVU*t.tetrisBuffer >= mergedVU {
				targetVehicle = vi
				break
			}
		}
		if targetVehicle == -1 {
			continue
		}
		if !routeRespectsWindows(merged, stops, vehicles[targetVehicle]) {
			continue
		}
		routes[ri] = merged
		routeVU[ri] = mergedVU
		vehicleOf[ri] = targetVehicle
		for _, k := range routes[rj] {
			routeOf[k] = ri
		}
		// Mark rj as drained — we collapse empty routes at the end.
		routes[rj] = nil
		routeVU[rj] = 0
		vehicleOf[rj] = -1
	}

	// 4. Place any leftover unassigned stops as their own route.
	orphans := make([]contract.Orphan, 0)
	for i := range stops {
		if routeOf[i] != -1 {
			continue
		}
		if !newRoute(i) {
			orphans = append(orphans, contract.Orphan{
				OrderID: stops[i].OrderID,
				Reason:  "exceeds largest vehicle capacity",
			})
		}
	}

	// 5. 2-opt local-search refinement on every non-empty route.
	out := make([]contract.Route, 0, len(routes))
	for ri, route := range routes {
		if len(route) == 0 {
			continue
		}
		veh := vehicles[vehicleOf[ri]]
		refined := twoOpt(route, stops, depotLat, depotLng, veh, t)
		out = append(out, buildRoute(refined, stops, depotLat, depotLng, veh))
	}
	return out, orphans
}

// routeRespectsWindows checks the proposed stop sequence honours every
// retailer's HH:MM receiving window given the vehicle's average speed.
// A window with empty open OR empty close is treated as "no constraint" on
// that boundary (matches Spanner-NULL semantics).
func routeRespectsWindows(seq []int, stops []contract.Stop, v contract.Vehicle) bool {
	speed := v.AvgSpeedKmph
	if speed <= 0 {
		speed = defaultAvgSpeedKmph
	}
	// Cursor in minutes-since-midnight starting at 00:00 of the dispatch day.
	// We model arrival relative to t=0; absolute clock time is irrelevant —
	// only window minutes matter.
	cursorMin := 0.0
	prevLat, prevLng := v.StartLat, v.StartLng
	for _, idx := range seq {
		s := stops[idx]
		distKm := haversineKm(prevLat, prevLng, s.Lat, s.Lng)
		travelMin := (distKm / speed) * 60.0
		cursorMin += travelMin
		openMin, openOK := parseHHMM(s.WindowOpen)
		closeMin, closeOK := parseHHMM(s.WindowClose)
		// Wait until window opens (driver idles outside the shop).
		if openOK && cursorMin < float64(openMin) {
			cursorMin = float64(openMin)
		}
		// Late arrival → infeasible.
		if closeOK && cursorMin > float64(closeMin) {
			return false
		}
		// Service the stop.
		cursorMin += float64(s.ServiceMinutes)
		prevLat, prevLng = s.Lat, s.Lng
	}
	return true
}

// parseHHMM returns minutes-since-midnight and ok=true on a valid "HH:MM",
// or (0,false) on empty / malformed. Solver-side validation is permissive —
// the backend already canonicalises via proximity.ValidateReceivingWindow.
func parseHHMM(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, false
	}
	hh, err := strconv.Atoi(parts[0])
	if err != nil || hh < 0 || hh > 23 {
		return 0, false
	}
	mm, err := strconv.Atoi(parts[1])
	if err != nil || mm < 0 || mm > 59 {
		return 0, false
	}
	return hh*60 + mm, true
}
