// Package optimizer is the Phase 2 dispatch solver. It is a pure Go module —
// no Spanner, no Kafka, no logging side effects — so it is unit-testable in
// isolation. The HTTP handler in handler.go is a thin adapter that decodes
// optimizercontract.SolveRequest, calls Solve, and encodes the response.
//
// Algorithm: Clarke-Wright Savings (parallel variant) seeds initial routes by
// merging stop pairs in descending order of savings = d(depot,i) + d(depot,j)
// - d(i,j). Each merged route is then refined with 2-opt local search bounded
// by Tunables.TwoOptIterations. Capacity (TetrisBuffer-adjusted) and HH:MM
// receiving windows are hard constraints — a violating merge is skipped.
package optimizer

import (
	"errors"
	"math"
	"sort"
	"time"

	contract "optimizercontract"
)

// Defaults — overridable per request via SolveRequest.Tunables.
const (
	defaultTetrisBuffer     = 0.95
	defaultTwoOptIterations = 200
	defaultMaxStopsPerRoute = 25
	defaultServiceMinutes   = 5
	defaultAvgSpeedKmph     = 30.0
)

// ErrEmptyFleet bubbles up to the HTTP layer as ErrCodeEmptyFleet/400.
var ErrEmptyFleet = errors.New("optimizer: empty fleet")

// Solve runs Clarke-Wright + 2-opt against req and returns a SolveResponse.
// The function is deterministic for any given (req, time.Now-independent)
// input — cluster ordering ties break on OrderID lexicographic order.
func Solve(req contract.SolveRequest) (contract.SolveResponse, error) {
	start := time.Now()
	resp := contract.SolveResponse{
		V:       contract.V,
		TraceID: req.TraceID,
		Source:  contract.SourceVRP,
	}
	if len(req.Vehicles) == 0 {
		return resp, ErrEmptyFleet
	}

	t := resolveTunables(req.Tunables)

	// 1. Filter stops with non-positive volume — defensive; should be caught
	//    upstream, but the solver must never crash on bad data.
	stops := make([]contract.Stop, 0, len(req.Stops))
	for _, s := range req.Stops {
		if s.VolumeVU <= 0 {
			resp.Orphans = append(resp.Orphans, contract.Orphan{
				OrderID: s.OrderID, Reason: "non-positive volume",
			})
			continue
		}
		if s.ServiceMinutes <= 0 {
			s.ServiceMinutes = defaultServiceMinutes
		}
		stops = append(stops, s)
	}
	considered := len(req.Stops)

	if len(stops) == 0 {
		resp.Stats = contract.Stats{
			ElapsedMs:       int(time.Since(start).Milliseconds()),
			StopsConsidered: considered,
			StopsOrphaned:   considered,
		}
		return resp, nil
	}

	// 2. Sort vehicles small→large; Clarke-Wright assigns largest savings
	//    first, but per-route capacity is the smallest viable truck.
	vehicles := make([]contract.Vehicle, len(req.Vehicles))
	copy(vehicles, req.Vehicles)
	sort.Slice(vehicles, func(i, j int) bool {
		return vehicles[i].MaxVolumeVU < vehicles[j].MaxVolumeVU
	})

	// 3. Build per-vehicle initial routes via Clarke-Wright savings, then
	//    refine each with 2-opt. Stops that cannot be placed in any vehicle
	//    are returned as orphans.
	planned, orphans := planRoutes(stops, vehicles, t)
	resp.Routes = planned
	resp.Orphans = append(resp.Orphans, orphans...)

	// 4. Stats roll-up.
	stops_placed := 0
	utilSum := 0.0
	for _, r := range resp.Routes {
		stops_placed += len(r.Stops)
		utilSum += r.UtilPct
	}
	avgUtil := 0.0
	if len(resp.Routes) > 0 {
		avgUtil = utilSum / float64(len(resp.Routes))
	}
	resp.Stats = contract.Stats{
		ElapsedMs:         int(time.Since(start).Milliseconds()),
		StopsConsidered:   considered,
		StopsPlaced:       stops_placed,
		StopsOrphaned:     len(resp.Orphans),
		VehiclesUsed:      len(resp.Routes),
		AvgUtilisationPct: avgUtil,
	}
	return resp, nil
}

// resolvedTunables holds non-zero solver parameters after default fill-in.
type resolvedTunables struct {
	tetrisBuffer     float64
	twoOptIterations int
	maxStopsPerRoute int
}

func resolveTunables(in *contract.Tunables) resolvedTunables {
	out := resolvedTunables{
		tetrisBuffer:     defaultTetrisBuffer,
		twoOptIterations: defaultTwoOptIterations,
		maxStopsPerRoute: defaultMaxStopsPerRoute,
	}
	if in == nil {
		return out
	}
	if in.TetrisBuffer > 0 && in.TetrisBuffer <= 1 {
		out.tetrisBuffer = in.TetrisBuffer
	}
	if in.TwoOptIterations > 0 {
		out.twoOptIterations = in.TwoOptIterations
	}
	if in.MaxStopsPerRoute > 0 {
		out.maxStopsPerRoute = in.MaxStopsPerRoute
	}
	return out
}

// haversineKm returns the great-circle distance in km between two lat/lng
// points. Used as the cost metric throughout the solver.
func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const earthR = 6371.0
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return earthR * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
