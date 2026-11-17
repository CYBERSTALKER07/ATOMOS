package optimizer

import (
	"math"
	"sort"
	"strconv"
	"strings"

	contract "github.com/pegasusx/pegasusx/packages/optimizer-contract"
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

	isVehicleInUse := func(vi int, excludeRoute int) bool {
		for r, vIdx := range vehicleOf {
			if r != excludeRoute && vIdx == vi && len(routes[r]) > 0 {
				return true
			}
		}
		return false
	}

	findBestVehicle := func(needVU float64, seq []int, excludeRoute int) int {
		// Pass 1: Try unused vehicles first (satisfying volume, constraints, duration, cold chain)
		bestUnused := -1
		bestUnusedCap := math.MaxFloat64
		for vi, v := range vehicles {
			if isVehicleInUse(vi, excludeRoute) {
				continue
			}
			cap := v.MaxVolumeVU * t.tetrisBuffer
			if cap >= needVU && routeRespectsConstraints(seq, stops, v, vehicles) {
				if cap < bestUnusedCap {
					bestUnused = vi
					bestUnusedCap = cap
				}
			}
		}
		if bestUnused != -1 {
			return bestUnused
		}

		// Pass 2: If no unused vehicle fits, fall back to best-fit across all vehicles
		bestAny := -1
		bestAnyCap := math.MaxFloat64
		for vi, v := range vehicles {
			cap := v.MaxVolumeVU * t.tetrisBuffer
			if cap >= needVU && routeRespectsConstraints(seq, stops, v, vehicles) {
				if cap < bestAnyCap {
					bestAny = vi
					bestAnyCap = cap
				}
			}
		}
		return bestAny
	}

	newRoute := func(stopIdx int) bool {
		seq := []int{stopIdx}
		need := stops[stopIdx].VolumeVU
		vi := findBestVehicle(need, seq, -1)
		if vi == -1 {
			return false
		}
		routes = append(routes, seq)
		vehicleOf = append(vehicleOf, vi)
		routeVU = append(routeVU, need)
		routeOf[stopIdx] = len(routes) - 1
		return true
	}

	// 3. Process savings descending: try to merge i and j into a single route.
	for _, p := range pairs {
		i, j := p.i, p.j
		ri, rj := routeOf[i], routeOf[j]

		// Both unassigned: seed a brand-new route with [i, j] if it fits.
		if ri == -1 && rj == -1 {
			combinedVU := stops[i].VolumeVU + stops[j].VolumeVU
			seq := []int{i, j}
			vi := findBestVehicle(combinedVU, seq, -1)
			if vi != -1 {
				routes = append(routes, seq)
				vehicleOf = append(vehicleOf, vi)
				routeVU = append(routeVU, combinedVU)
				routeOf[i] = len(routes) - 1
				routeOf[j] = len(routes) - 1
			} else {
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
			candSeq := append(append([]int{}, routes[routeIdx]...), addIdx)
			need := routeVU[routeIdx] + stops[addIdx].VolumeVU

			// Check if current assigned vehicle still fits and satisfies constraints
			currVi := vehicleOf[routeIdx]
			if currVi >= 0 && vehicles[currVi].MaxVolumeVU*t.tetrisBuffer >= need && routeRespectsConstraints(candSeq, stops, vehicles[currVi], vehicles) {
				routes[routeIdx] = candSeq
				routeVU[routeIdx] = need
				routeOf[addIdx] = routeIdx
				continue
			}

			// Otherwise, try finding another vehicle (preferring unused)
			newVi := findBestVehicle(need, candSeq, routeIdx)
			if newVi == -1 {
				continue
			}
			vehicleOf[routeIdx] = newVi
			routes[routeIdx] = candSeq
			routeVU[routeIdx] = need
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
		targetVehicle := findBestVehicle(mergedVU, merged, ri)
		if targetVehicle == -1 {
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

func isColdStop(s contract.Stop) bool {
	h := strings.ToUpper(strings.TrimSpace(s.HandlingClass))
	return h == "COLD_CHAIN" || h == "CHILLED" || h == "FROZEN" || h == "PERISHABLE"
}

func vehicleSupportsColdChain(v contract.Vehicle, allVehicles []contract.Vehicle) bool {
	vid := strings.ToLower(v.VehicleID)
	if strings.Contains(vid, "cold") || strings.Contains(vid, "reefer") || strings.Contains(vid, "chilled") || strings.Contains(vid, "frozen") || strings.Contains(vid, "fridge") {
		return true
	}
	if strings.Contains(vid, "ambient") || strings.Contains(vid, "dry") {
		return false
	}
	// If fleet contains dedicated cold vehicles, non-cold-tagged vehicles are disqualified for cold stops
	anyColdInFleet := false
	for _, other := range allVehicles {
		ovid := strings.ToLower(other.VehicleID)
		if strings.Contains(ovid, "cold") || strings.Contains(ovid, "reefer") || strings.Contains(ovid, "chilled") || strings.Contains(ovid, "frozen") || strings.Contains(ovid, "fridge") {
			anyColdInFleet = true
			break
		}
	}
	return !anyColdInFleet
}

// routeRespectsConstraints enforces time windows, vehicle shift/route duration bounds,
// hazmat certification, and cold-chain compatibility.
func routeRespectsConstraints(seq []int, stops []contract.Stop, v contract.Vehicle, allVehicles []contract.Vehicle) bool {
	// 1. Hazmat check
	for _, idx := range seq {
		if stops[idx].IsHazardous && !v.HazmatCertified {
			return false
		}
	}

	// 2. Cold-chain check
	for _, idx := range seq {
		if isColdStop(stops[idx]) && !vehicleSupportsColdChain(v, allVehicles) {
			return false
		}
	}

	// 3. Time windows and shift / duration bounds
	speed := v.AvgSpeedKmph
	if speed <= 0 {
		speed = defaultAvgSpeedKmph
	}

	cursorMin := 0.0
	startMin := 0.0
	if sMin, ok := parseHHMM(v.ShiftStart); ok {
		startMin = float64(sMin)
		cursorMin = startMin
	}

	prevLat, prevLng := v.StartLat, v.StartLng
	for _, idx := range seq {
		s := stops[idx]
		distKm := haversineKm(prevLat, prevLng, s.Lat, s.Lng)
		travelMin := (distKm / speed) * 60.0
		cursorMin += travelMin

		openMin, openOK := parseHHMM(s.WindowOpen)
		closeMin, closeOK := parseHHMM(s.WindowClose)
		if openOK && cursorMin < float64(openMin) {
			cursorMin = float64(openMin)
		}
		if closeOK && cursorMin > float64(closeMin) {
			return false
		}
		cursorMin += float64(s.ServiceMinutes)
		prevLat, prevLng = s.Lat, s.Lng
	}

	// Return to depot
	endLat, endLng := v.EndLat, v.EndLng
	if endLat == 0 && endLng == 0 {
		endLat, endLng = v.StartLat, v.StartLng
	}
	distEndKm := haversineKm(prevLat, prevLng, endLat, endLng)
	cursorMin += (distEndKm / speed) * 60.0

	// Check MaxRouteMinutes
	routeDuration := cursorMin - startMin
	if v.MaxRouteMinutes > 0 && routeDuration > float64(v.MaxRouteMinutes) {
		return false
	}

	// Check ShiftEnd
	if eMin, ok := parseHHMM(v.ShiftEnd); ok && cursorMin > float64(eMin) {
		return false
	}

	return true
}

// routeRespectsWindows checks the proposed stop sequence honours every
// retailer's HH:MM receiving window given the vehicle's average speed.
// A window with empty open OR empty close is treated as "no constraint" on
// that boundary (matches Spanner-NULL semantics).
func routeRespectsWindows(seq []int, stops []contract.Stop, v contract.Vehicle) bool {
	return routeRespectsConstraints(seq, stops, v, nil)
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
