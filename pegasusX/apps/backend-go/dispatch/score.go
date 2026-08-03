package dispatch

import (
	"math"
)

// Default multi-objective weights (sum of positive terms ≈ 0.95; empty-mile subtracted).
const (
	WeightVolumeFit     = 0.25
	WeightSpatialFit    = 0.20
	WeightPriority      = 0.20
	WeightDriverScore   = 0.15
	WeightShopClosed    = 0.10
	WeightWindowSlack   = 0.05
	WeightEmptyMileCost = 0.05
)

// ScoreContext carries depot / clock for pure scoring (no I/O).
type ScoreContext struct {
	DepotLat    float64
	DepotLng    float64
	NowMinutes  int // minutes since midnight; <0 → ignore window slack
	MaxH3Ring   int // spatial normalization; 0 → default 4
	CellLookup  func(lat, lng float64) string
}

// ScoreCandidate ranks assigning `order` onto an existing route or a new route for `driver`.
// When route is nil, scores opening a new route with driver.
// Missing PriorityScore / DriverScore / ShopClosedRisk use neutral mid defaults.
func ScoreCandidate(route *DispatchRoute, order DispatchableOrder, driver AvailableDriver, ctx ScoreContext) float64 {
	maxCap := driver.MaxVolumeVU * TetrisBuffer
	if route != nil && route.MaxVolume > 0 {
		maxCap = route.MaxVolume
	}
	if maxCap <= 0 {
		return -1
	}

	loaded := 0.0
	orderCount := 0
	routeLat, routeLng := ctx.DepotLat, ctx.DepotLng
	if route != nil {
		loaded = route.LoadedVolume
		orderCount = len(route.Orders)
		if orderCount > 0 {
			routeLat, routeLng = routeCentroid(route)
		}
	}
	remaining := maxCap - loaded
	if remaining < order.VolumeVU-1e-9 && (route != nil || order.VolumeVU > maxCap) {
		// Feasibility gate for existing routes; new route overflow handled by caller.
		if route != nil {
			return -1
		}
	}

	volumeFit := clamp01(order.VolumeVU / maxCap)
	if remaining > 0 && route != nil {
		// Prefer assignments that use remaining capacity without extreme waste.
		volumeFit = clamp01(order.VolumeVU / remaining)
		if volumeFit > 1 {
			volumeFit = 1
		}
	}

	spatialFit := 1.0
	if ctx.CellLookup != nil {
		orderCell := ctx.CellLookup(order.Lat, order.Lng)
		routeCell := ctx.CellLookup(routeLat, routeLng)
		if orderCell != "" && routeCell != "" && orderCell == routeCell {
			spatialFit = 1.0
		} else {
			// Haversine proxy normalized to ~max ring distance.
			d := haversineKm(order.Lat, order.Lng, routeLat, routeLng)
			maxKm := 8.0
			if ctx.MaxH3Ring > 0 {
				maxKm = float64(ctx.MaxH3Ring) * 2.5
			}
			spatialFit = clamp01(1.0 - d/maxKm)
		}
	}

	priority := order.PriorityScore
	if priority <= 0 {
		priority = 50
	}
	priorityNorm := clamp01(float64(priority) / 200.0)

	dScore := driver.DriverScore
	if dScore <= 0 {
		dScore = 50
	}
	driverNorm := clamp01(float64(dScore) / 100.0)

	risk := order.ShopClosedRisk
	if risk < 0 {
		risk = 0
	}
	if risk > 1 {
		risk = 1
	}
	shopTerm := 1.0 - risk

	windowSlack := 0.5
	if ctx.NowMinutes >= 0 && HasReceivingWindow(order.ReceivingWindowOpen, order.ReceivingWindowClose) {
		closeM := ParseTimeMinutes(EffectiveWindowClose(order.ReceivingWindowClose))
		if closeM >= 0 {
			slack := closeM - ctx.NowMinutes
			// More slack → higher score (up to 14h).
			windowSlack = clamp01(float64(slack) / (14.0 * 60.0))
		}
	}

	emptyMile := 0.0
	if route == nil {
		// New truck: cost of deadhead from depot to first stop.
		emptyMile = clamp01(haversineKm(ctx.DepotLat, ctx.DepotLng, order.Lat, order.Lng) / 15.0)
	} else if orderCount == 0 {
		emptyMile = clamp01(haversineKm(ctx.DepotLat, ctx.DepotLng, order.Lat, order.Lng) / 15.0)
	} else {
		emptyMile = clamp01(haversineKm(routeLat, routeLng, order.Lat, order.Lng) / 10.0)
	}

	score := WeightVolumeFit*volumeFit +
		WeightSpatialFit*spatialFit +
		WeightPriority*priorityNorm +
		WeightDriverScore*driverNorm +
		WeightShopClosed*shopTerm +
		WeightWindowSlack*windowSlack -
		WeightEmptyMileCost*emptyMile

	// Mild load-balance preference when scores otherwise equal: fewer stops wins slightly.
	score -= 0.001 * float64(orderCount)
	return score
}

// SelectBestScoredVehicle picks the capacity-feasible driver with the best ScoreCandidate for a new route.
func SelectBestScoredVehicle(order DispatchableOrder, fleet []AvailableDriver, ctx ScoreContext) (*VehicleMatch, bool) {
	if len(fleet) == 0 {
		return nil, false
	}
	var best *VehicleMatch
	bestScore := math.Inf(-1)
	var largest AvailableDriver
	largestVU := -1.0
	for _, d := range fleet {
		if d.MaxVolumeVU > largestVU {
			largestVU = d.MaxVolumeVU
			largest = d
		}
		if d.MaxVolumeVU*TetrisBuffer < order.VolumeVU {
			continue
		}
		sc := ScoreCandidate(nil, order, d, ctx)
		if sc > bestScore {
			bestScore = sc
			dd := d
			best = &VehicleMatch{Driver: dd, Overflow: false}
		}
	}
	if best != nil {
		return best, true
	}
	if largestVU < 0 {
		return nil, false
	}
	return &VehicleMatch{Driver: largest, Overflow: true}, true
}

func routeCentroid(route *DispatchRoute) (lat, lng float64) {
	if route == nil || len(route.Orders) == 0 {
		return 0, 0
	}
	var sumLat, sumLng float64
	for _, o := range route.Orders {
		sumLat += o.Lat
		sumLng += o.Lng
	}
	n := float64(len(route.Orders))
	return sumLat / n, sumLng / n
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
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
