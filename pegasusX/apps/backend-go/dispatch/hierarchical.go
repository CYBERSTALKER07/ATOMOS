package dispatch

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const (
	// DefaultPartitionTargetStops targets 150-250 stops per spatial subproblem
	// so every OSRM matrix block and OR-Tools solve stays fast and under 4 MiB.
	DefaultPartitionTargetStops = 200
	MinPartitionStops           = 50
	MaxPartitionStops           = 250
	defaultHierarchicalMin      = 201
	defaultRouteStopCap         = 60
	hierarchicalWorkers         = 8
)

// HierarchicalEnabled returns whether spatial partitioning fallback is active.
// Enabled by default; set DISPATCH_HIERARCHICAL_H3=0/false/off to disable explicitly.
func HierarchicalEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("DISPATCH_HIERARCHICAL_H3")))
	if v == "" {
		return true
	}
	return v != "0" && v != "false" && v != "no" && v != "off"
}

// HierarchicalOrderThreshold defaults to 201 (above DefaultPartitionTargetStops);
// override with DISPATCH_HIERARCHICAL_MIN_ORDERS when >= 1.
func HierarchicalOrderThreshold() int {
	raw := strings.TrimSpace(os.Getenv("DISPATCH_HIERARCHICAL_MIN_ORDERS"))
	if raw == "" {
		return defaultHierarchicalMin
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return defaultHierarchicalMin
	}
	return n
}

// EffectiveMaxStopsPerRoute resolves the explicit per-route stop cap so large
// waves (e.g. 2,000 outlets / 40 vans = 50 stops/van) do not silently truncate
// at OR-Tools' historical 25-stop default.
func EffectiveMaxStopsPerRoute(explicitMax, orderCount, fleetCount int) int {
	if explicitMax > 0 {
		return explicitMax
	}
	if raw := strings.TrimSpace(os.Getenv("DISPATCH_MAX_STOPS_PER_ROUTE")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return n
		}
	}
	if fleetCount <= 0 || orderCount <= 0 {
		return MaxWaypointsPerManifest
	}
	avgStops := (orderCount + fleetCount - 1) / fleetCount
	slack := avgStops / 5
	if slack < 10 {
		slack = 10
	}
	capStops := avgStops + slack
	if capStops < MaxWaypointsPerManifest {
		capStops = MaxWaypointsPerManifest
	}
	if capStops > MaxPartitionStops {
		capStops = MaxPartitionStops
	}
	return capStops
}

// SpatialPartitionOptions configures depot + capacity-balanced spatial partitioning.
type SpatialPartitionOptions struct {
	DepotLat             float64
	DepotLng             float64
	MaxStopsPerPartition int
	MaxStopsPerRoute     int
	TetrisBuffer         float64
}

// SpatialSubproblem is one bounded <=250-stop VRP subproblem with disjoint fleet.
type SpatialSubproblem struct {
	Index         int
	DepotLat      float64
	DepotLng      float64
	Orders        []DispatchableOrder
	Fleet         []AvailableDriver
	TotalVolumeVU float64
	StopCount     int
}

// SpatialPartitionPlan holds all spatial subproblems plus deterministically deferred overflow.
type SpatialPartitionPlan struct {
	Subproblems      []SpatialSubproblem
	DeferredOverflow []GeoOrder
	OverflowWarnings []RetailerOverflowWarning
	Warnings         []string
}

type retailerStopGroup struct {
	key        string
	retailerID string
	lat        float64
	lng        float64
	vol        float64
	hasHazmat  bool
	isRecovery bool
	orders     []DispatchableOrder
}

type depotPool struct {
	key        string
	lat        float64
	lng        float64
	drivers    []AvailableDriver
	groups     []retailerStopGroup
	effCapVU   float64
	stopBudget int
}

type spatialBucket struct {
	groups []retailerStopGroup
	stops  int
	vol    float64
}

// BuildSpatialSubproblems partitions orders into depot + capacity-balanced
// <=150-250-stop spatial subproblems, pairing each with a proportional, disjoint
// subset of available vans and deterministically deferring hard overflow.
func BuildSpatialSubproblems(orders []DispatchableOrder, fleet []AvailableDriver, opt SpatialPartitionOptions) SpatialPartitionPlan {
	plan := SpatialPartitionPlan{}
	if len(orders) == 0 {
		return plan
	}
	if len(fleet) == 0 {
		plan.DeferredOverflow = sortGeoOrphans(orders)
		plan.Warnings = append(plan.Warnings, "spatial_partition_empty_fleet")
		return plan
	}

	tetris := opt.TetrisBuffer
	if tetris <= 0 || tetris > 1 {
		tetris = TetrisBuffer
	}
	maxPartStops := opt.MaxStopsPerPartition
	if maxPartStops <= 0 {
		maxPartStops = DefaultPartitionTargetStops
	}
	if maxPartStops > MaxPartitionStops {
		maxPartStops = MaxPartitionStops
	}
	maxRouteStops := EffectiveMaxStopsPerRoute(opt.MaxStopsPerRoute, len(orders), len(fleet))

	groups := buildAtomicRetailerGroups(orders, maxPartStops)
	pools := buildDepotPools(fleet, opt.DepotLat, opt.DepotLng, tetris, maxPartStops)
	assignGroupsToDepots(groups, pools, maxRouteStops)

	for _, pool := range pools {
		if len(pool.groups) == 0 || len(pool.drivers) == 0 {
			continue
		}
		keptGroups, overflowOrders := enforcePoolStopBudget(pool.groups, len(pool.drivers)*maxPartStops, pool.lat, pool.lng)
		if len(overflowOrders) > 0 {
			plan.DeferredOverflow = append(plan.DeferredOverflow, overflowOrders...)
		}
		if len(keptGroups) == 0 {
			continue
		}

		buckets := recursiveSpatialBisect(keptGroups, maxPartStops, pool.lat)
		buckets, trimOverflow := capBucketCountToFleet(buckets, len(pool.drivers), maxPartStops, pool.lat, pool.lng)
		if len(trimOverflow) > 0 {
			plan.DeferredOverflow = append(plan.DeferredOverflow, trimOverflow...)
		}

		driverBuckets := allocatePoolDriversProportionally(buckets, pool.drivers, tetris, maxRouteStops)
		for bi, b := range buckets {
			assignedDrivers := driverBuckets[bi]
			if len(assignedDrivers) == 0 {
				for _, g := range b.groups {
					for _, o := range g.orders {
						plan.DeferredOverflow = append(plan.DeferredOverflow, o.ToGeo())
					}
				}
				continue
			}
			flatOrders := make([]DispatchableOrder, 0, b.stops)
			for _, g := range b.groups {
				flatOrders = append(flatOrders, g.orders...)
			}
			sort.Slice(flatOrders, func(i, j int) bool {
				return flatOrders[i].OrderID < flatOrders[j].OrderID
			})
			plan.Subproblems = append(plan.Subproblems, SpatialSubproblem{
				Index:         len(plan.Subproblems),
				DepotLat:      pool.lat,
				DepotLng:      pool.lng,
				Orders:        flatOrders,
				Fleet:         assignedDrivers,
				TotalVolumeVU: b.vol,
				StopCount:     len(flatOrders),
			})
		}
	}

	if len(plan.DeferredOverflow) > 0 {
		sort.Slice(plan.DeferredOverflow, func(i, j int) bool {
			return plan.DeferredOverflow[i].OrderID < plan.DeferredOverflow[j].OrderID
		})
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("deferred_overflow=%d", len(plan.DeferredOverflow)))
	}
	plan.Warnings = append(plan.Warnings, fmt.Sprintf("spatial_partition_subproblems=%d", len(plan.Subproblems)))
	return plan
}

// BinPackHierarchical replaces the legacy H3 res-5 sequential fleet drain with
// depot + capacity-balanced spatial partitioning solved across parallel BinPack workers.
func BinPackHierarchical(orders []DispatchableOrder, fleet []AvailableDriver, cellLookup func(lat, lng float64) string, opts ...BinPackOptions) *AssignmentResult {
	var opt BinPackOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	if cellLookup == nil {
		cellLookup = H3CellLookup
	}
	maxRouteStops := EffectiveMaxStopsPerRoute(opt.MaxStopsPerRoute, len(orders), len(fleet))
	partPlan := BuildSpatialSubproblems(orders, fleet, SpatialPartitionOptions{
		DepotLat:             opt.Score.DepotLat,
		DepotLng:             opt.Score.DepotLng,
		MaxStopsPerPartition: DefaultPartitionTargetStops,
		MaxStopsPerRoute:     maxRouteStops,
		TetrisBuffer:         TetrisBuffer,
	})

	subResults := make([]*AssignmentResult, len(partPlan.Subproblems))
	workers := hierarchicalWorkers
	if len(partPlan.Subproblems) < workers {
		workers = len(partPlan.Subproblems)
	}
	if workers > 0 {
		jobs := make(chan int, len(partPlan.Subproblems))
		for i := range partPlan.Subproblems {
			jobs <- i
		}
		close(jobs)

		var wg sync.WaitGroup
		wg.Add(workers)
		for w := 0; w < workers; w++ {
			go func() {
				defer wg.Done()
				for idx := range jobs {
					sub := partPlan.Subproblems[idx]
					leafOpt := opt
					leafOpt.MaxStopsPerRoute = maxRouteStops
					if leafOpt.Score.DepotLat == 0 && leafOpt.Score.DepotLng == 0 {
						leafOpt.Score.DepotLat = sub.DepotLat
						leafOpt.Score.DepotLng = sub.DepotLng
					}
					subResults[idx] = BinPack(sub.Orders, sub.Fleet, cellLookup, leafOpt)
				}
			}()
		}
		wg.Wait()
	}

	merged := MergePartitionResults(subResults, partPlan.DeferredOverflow)
	merged.OverflowWarnings = append(merged.OverflowWarnings, partPlan.OverflowWarnings...)
	merged.Warnings = append(merged.Warnings, partPlan.Warnings...)
	merged.Warnings = append(merged.Warnings, "spatial_balanced_partition")
	return merged
}

// MergePartitionResults deterministically merges disjoint subproblem outputs and deferred overflow.
func MergePartitionResults(parts []*AssignmentResult, deferred []GeoOrder) *AssignmentResult {
	merged := &AssignmentResult{
		Routes:  []DispatchRoute{},
		Splits:  []SplitOrder{},
		Orphans: []GeoOrder{},
	}
	for _, part := range parts {
		if part == nil {
			continue
		}
		merged.Routes = append(merged.Routes, part.Routes...)
		merged.Splits = append(merged.Splits, part.Splits...)
		merged.Orphans = append(merged.Orphans, part.Orphans...)
		merged.OverflowWarnings = append(merged.OverflowWarnings, part.OverflowWarnings...)
		merged.Warnings = append(merged.Warnings, part.Warnings...)
	}
	merged.Orphans = append(merged.Orphans, deferred...)
	sort.Slice(merged.Routes, func(i, j int) bool {
		return merged.Routes[i].DriverID < merged.Routes[j].DriverID
	})
	sort.Slice(merged.Orphans, func(i, j int) bool {
		return merged.Orphans[i].OrderID < merged.Orphans[j].OrderID
	})
	return merged
}

func sortGeoOrphans(orders []DispatchableOrder) []GeoOrder {
	out := make([]GeoOrder, len(orders))
	for i, o := range orders {
		out[i] = o.ToGeo()
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].OrderID < out[j].OrderID
	})
	return out
}

func buildAtomicRetailerGroups(orders []DispatchableOrder, maxPartStops int) []retailerStopGroup {
	byRetailer := make(map[string][]DispatchableOrder)
	for _, o := range orders {
		key := strings.TrimSpace(o.RetailerID)
		if key == "" {
			key = "order:" + strings.TrimSpace(o.OrderID)
		}
		byRetailer[key] = append(byRetailer[key], o)
	}

	keys := make([]string, 0, len(byRetailer))
	for k := range byRetailer {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	groups := make([]retailerStopGroup, 0, len(keys))
	for _, k := range keys {
		slice := byRetailer[k]
		sort.Slice(slice, func(i, j int) bool {
			return slice[i].OrderID < slice[j].OrderID
		})
		for start, chunkIdx := 0, 0; start < len(slice); start, chunkIdx = start+maxPartStops, chunkIdx+1 {
			end := start + maxPartStops
			if end > len(slice) {
				end = len(slice)
			}
			chunk := slice[start:end]
			var latSum, lngSum, volSum float64
			hasHazmat, hasRecovery := false, false
			for _, o := range chunk {
				latSum += o.Lat
				lngSum += o.Lng
				volSum += o.VolumeVU
				if o.IsHazardous || strings.EqualFold(strings.TrimSpace(o.HandlingClass), "HAZMAT") {
					hasHazmat = true
				}
				if o.IsRecovery {
					hasRecovery = true
				}
			}
			subKey := k
			if chunkIdx > 0 {
				subKey = fmt.Sprintf("%s#%d", k, chunkIdx)
			}
			n := float64(len(chunk))
			groups = append(groups, retailerStopGroup{
				key:        subKey,
				retailerID: chunk[0].RetailerID,
				lat:        latSum / n,
				lng:        lngSum / n,
				vol:        volSum,
				hasHazmat:  hasHazmat,
				isRecovery: hasRecovery,
				orders:     chunk,
			})
		}
	}
	return groups
}

func buildDepotPools(fleet []AvailableDriver, defaultLat, defaultLng, tetris float64, maxPartStops int) []*depotPool {
	sortedFleet := make([]AvailableDriver, len(fleet))
	copy(sortedFleet, fleet)
	sort.Slice(sortedFleet, func(i, j int) bool {
		return sortedFleet[i].DriverID < sortedFleet[j].DriverID
	})

	byDepot := make(map[string]*depotPool)
	for _, d := range sortedFleet {
		lat, lng := d.StartLat, d.StartLng
		if lat == 0 && lng == 0 {
			lat, lng = defaultLat, defaultLng
		}
		key := fmt.Sprintf("%.6f,%.6f", lat, lng)
		p, ok := byDepot[key]
		if !ok {
			p = &depotPool{key: key, lat: lat, lng: lng}
			byDepot[key] = p
		}
		p.drivers = append(p.drivers, d)
		p.effCapVU += d.MaxVolumeVU * tetris
		p.stopBudget += maxPartStops
	}

	keys := make([]string, 0, len(byDepot))
	for k := range byDepot {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pools := make([]*depotPool, len(keys))
	for i, k := range keys {
		pools[i] = byDepot[k]
	}
	return pools
}

func assignGroupsToDepots(groups []retailerStopGroup, pools []*depotPool, maxRouteStops int) {
	if len(pools) == 0 {
		return
	}
	if len(pools) == 1 {
		pools[0].groups = append(pools[0].groups, groups...)
		return
	}

	assignedStops := make([]int, len(pools))
	assignedVol := make([]float64, len(pools))

	for _, g := range groups {
		bestIdx := 0
		bestCost := math.Inf(1)
		for pi, p := range pools {
			dist := haversineKM(g.lat, g.lng, p.lat, p.lng)
			volRatio := (assignedVol[pi] + g.vol) / math.Max(1.0, p.effCapVU)
			stopRatio := float64(assignedStops[pi]+len(g.orders)) / float64(max(1, len(p.drivers)*maxRouteStops))
			overload := math.Max(0.0, math.Max(volRatio, stopRatio)-1.0)
			// Prefer nearest depot; penalize only once that depot exceeds its share.
			cost := dist * (1.0 + 4.0*overload)
			if cost < bestCost {
				bestCost = cost
				bestIdx = pi
			}
		}
		pools[bestIdx].groups = append(pools[bestIdx].groups, g)
		assignedStops[bestIdx] += len(g.orders)
		assignedVol[bestIdx] += g.vol
	}
}

func enforcePoolStopBudget(groups []retailerStopGroup, stopBudget int, depotLat, depotLng float64) ([]retailerStopGroup, []GeoOrder) {
	total := 0
	for _, g := range groups {
		total += len(g.orders)
	}
	if total <= stopBudget {
		return groups, nil
	}

	ranked := make([]retailerStopGroup, len(groups))
	copy(ranked, groups)
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].isRecovery != ranked[j].isRecovery {
			return ranked[i].isRecovery
		}
		di := haversineKM(ranked[i].lat, ranked[i].lng, depotLat, depotLng)
		dj := haversineKM(ranked[j].lat, ranked[j].lng, depotLat, depotLng)
		if math.Abs(di-dj) > 1e-6 {
			return di < dj
		}
		return ranked[i].key < ranked[j].key
	})

	kept := make([]retailerStopGroup, 0, len(ranked))
	var overflow []GeoOrder
	used := 0
	for _, g := range ranked {
		if used+len(g.orders) <= stopBudget {
			kept = append(kept, g)
			used += len(g.orders)
		} else {
			for _, o := range g.orders {
				overflow = append(overflow, o.ToGeo())
			}
		}
	}
	return kept, overflow
}

func recursiveSpatialBisect(groups []retailerStopGroup, maxPartStops int, depotLat float64) []spatialBucket {
	stops := 0
	vol := 0.0
	for _, g := range groups {
		stops += len(g.orders)
		vol += g.vol
	}
	if stops <= maxPartStops || len(groups) <= 1 {
		return []spatialBucket{{groups: groups, stops: stops, vol: vol}}
	}

	minLat, maxLat := groups[0].lat, groups[0].lat
	minLng, maxLng := groups[0].lng, groups[0].lng
	latSum := 0.0
	for _, g := range groups {
		latSum += g.lat
		if g.lat < minLat {
			minLat = g.lat
		}
		if g.lat > maxLat {
			maxLat = g.lat
		}
		if g.lng < minLng {
			minLng = g.lng
		}
		if g.lng > maxLng {
			maxLng = g.lng
		}
	}
	meanLat := depotLat
	if len(groups) > 0 {
		meanLat = latSum / float64(len(groups))
	}
	latSpan := maxLat - minLat
	lngSpan := (maxLng - minLng) * math.Max(0.2, math.Cos(meanLat*math.Pi/180.0))

	sorted := make([]retailerStopGroup, len(groups))
	copy(sorted, groups)
	splitByLat := latSpan >= lngSpan
	sort.Slice(sorted, func(i, j int) bool {
		if splitByLat {
			if sorted[i].lat != sorted[j].lat {
				return sorted[i].lat < sorted[j].lat
			}
			if sorted[i].lng != sorted[j].lng {
				return sorted[i].lng < sorted[j].lng
			}
		} else {
			if sorted[i].lng != sorted[j].lng {
				return sorted[i].lng < sorted[j].lng
			}
			if sorted[i].lat != sorted[j].lat {
				return sorted[i].lat < sorted[j].lat
			}
		}
		return sorted[i].key < sorted[j].key
	})

	// Choose split ratio floor(k/2) : ceil(k/2) based on required partitions k = ceil(stops/maxPartStops)
	// so total bucket count stays minimal (~150-200 stops/bucket) instead of inflating to a power of 2.
	kParts := (stops + maxPartStops - 1) / maxPartStops
	if kParts < 2 {
		kParts = 2
	}
	leftFraction := float64(kParts/2) / float64(kParts)
	targetLeftStops := float64(stops) * leftFraction
	targetLeftVol := vol * leftFraction
	bestCut := 1
	bestPenalty := math.Inf(1)
	leftStops := 0
	leftVol := 0.0
	for cut := 1; cut < len(sorted); cut++ {
		leftStops += len(sorted[cut-1].orders)
		leftVol += sorted[cut-1].vol
		stopErr := math.Abs(float64(leftStops)-targetLeftStops) / math.Max(1.0, targetLeftStops)
		volErr := 0.0
		if targetLeftVol > 0 {
			volErr = math.Abs(leftVol-targetLeftVol) / targetLeftVol
		}
		penalty := 0.70*stopErr + 0.30*volErr
		if penalty < bestPenalty {
			bestPenalty = penalty
			bestCut = cut
		}
	}

	left := recursiveSpatialBisect(sorted[:bestCut], maxPartStops, depotLat)
	right := recursiveSpatialBisect(sorted[bestCut:], maxPartStops, depotLat)
	return append(left, right...)
}

func capBucketCountToFleet(buckets []spatialBucket, driverCount, maxPartStops int, depotLat, depotLng float64) ([]spatialBucket, []GeoOrder) {
	if len(buckets) <= driverCount {
		return buckets, nil
	}
	// Merge smallest compatible adjacent/global pairs whose stops stay <= maxPartStops.
	for len(buckets) > driverCount {
		bestI, bestJ, bestSum := -1, -1, maxPartStops+1
		for i := 0; i < len(buckets); i++ {
			for j := i + 1; j < len(buckets); j++ {
				sum := buckets[i].stops + buckets[j].stops
				if sum <= maxPartStops && sum < bestSum {
					bestI, bestJ, bestSum = i, j, sum
				}
			}
		}
		if bestI < 0 {
			break
		}
		buckets[bestI].groups = append(buckets[bestI].groups, buckets[bestJ].groups...)
		buckets[bestI].stops += buckets[bestJ].stops
		buckets[bestI].vol += buckets[bestJ].vol
		buckets = append(buckets[:bestJ], buckets[bestJ+1:]...)
	}
	if len(buckets) <= driverCount {
		return buckets, nil
	}

	// Deterministic overflow deferral for buckets exceeding driver count.
	sort.Slice(buckets, func(i, j int) bool {
		if buckets[i].stops != buckets[j].stops {
			return buckets[i].stops > buckets[j].stops
		}
		return bucketDistance(buckets[i], depotLat, depotLng) < bucketDistance(buckets[j], depotLat, depotLng)
	})
	var overflow []GeoOrder
	for _, b := range buckets[driverCount:] {
		for _, g := range b.groups {
			for _, o := range g.orders {
				overflow = append(overflow, o.ToGeo())
			}
		}
	}
	return buckets[:driverCount], overflow
}

func bucketDistance(b spatialBucket, depotLat, depotLng float64) float64 {
	if len(b.groups) == 0 {
		return 0
	}
	var lat, lng float64
	for _, g := range b.groups {
		lat += g.lat
		lng += g.lng
	}
	n := float64(len(b.groups))
	return haversineKM(lat/n, lng/n, depotLat, depotLng)
}

func allocatePoolDriversProportionally(buckets []spatialBucket, drivers []AvailableDriver, tetris float64, maxRouteStops int) [][]AvailableDriver {
	bCount := len(buckets)
	dCount := len(drivers)
	out := make([][]AvailableDriver, bCount)
	if bCount == 0 || dCount == 0 {
		return out
	}

	totalEffCap := 0.0
	for _, d := range drivers {
		totalEffCap += d.MaxVolumeVU * tetris
	}
	avgVanCap := math.Max(1e-6, totalEffCap/float64(dCount))

	counts := make([]int, bCount)
	bucketHazmat := make([]bool, bCount)
	for i, b := range buckets {
		counts[i] = 1
		for _, g := range b.groups {
			if g.hasHazmat {
				bucketHazmat[i] = true
				break
			}
		}
	}

	// Assign extra vans iteratively using highest current stop/volume deficit ratio (D'Hondt)
	// so every partition satisfies MaxStopsPerRoute and capacity proportionally.
	remaining := dCount - bCount
	for remaining > 0 {
		bestIdx := 0
		bestRatio := -1.0
		for i, b := range buckets {
			stopRatio := float64(b.stops) / float64(counts[i]*max(1, maxRouteStops))
			volRatio := b.vol / (float64(counts[i]) * avgVanCap)
			ratio := math.Max(stopRatio, volRatio)
			if ratio > bestRatio+1e-9 || (math.Abs(ratio-bestRatio) <= 1e-9 && b.stops > buckets[bestIdx].stops) {
				bestRatio = ratio
				bestIdx = i
			}
		}
		counts[bestIdx]++
		remaining--
	}

	// Place hazmat drivers into hazmat-needing partitions first, then distribute
	// remaining drivers sorted by capacity in snake order so tiers balance evenly.
	pool := make([]AvailableDriver, len(drivers))
	copy(pool, drivers)
	sort.Slice(pool, func(i, j int) bool {
		if pool[i].MaxVolumeVU != pool[j].MaxVolumeVU {
			return pool[i].MaxVolumeVU > pool[j].MaxVolumeVU
		}
		return pool[i].DriverID < pool[j].DriverID
	})

	used := make([]bool, len(pool))
	for bi := 0; bi < bCount; bi++ {
		if !bucketHazmat[bi] {
			continue
		}
		for di, d := range pool {
			if !used[di] && d.HazmatCertified && len(out[bi]) < counts[bi] {
				out[bi] = append(out[bi], d)
				used[di] = true
			}
		}
	}

	var leftover []AvailableDriver
	for di, d := range pool {
		if !used[di] {
			leftover = append(leftover, d)
		}
	}

	li := 0
	forward := true
	for li < len(leftover) {
		progressed := false
		for step := 0; step < bCount && li < len(leftover); step++ {
			bi := step
			if !forward {
				bi = bCount - 1 - step
			}
			if len(out[bi]) < counts[bi] {
				out[bi] = append(out[bi], leftover[li])
				li++
				progressed = true
			}
		}
		if !progressed {
			break
		}
		forward = !forward
	}
	return out
}

func haversineKM(lat1, lng1, lat2, lng2 float64) float64 {
	const earthR = 6371.0
	const rad = math.Pi / 180.0
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	if a < 0 {
		a = 0
	} else if a > 1 {
		a = 1
	}
	return earthR * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
