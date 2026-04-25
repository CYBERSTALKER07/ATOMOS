// Package routing implements the Field General AI Route Optimizer (Vector H).
// It offloads the Traveling Salesperson Problem to the Google Maps Directions API
// (optimize:true) and writes the resulting SequenceIndex into Spanner atomically.
//
// ARCHITECTURAL DECISION: TSP is NP-hard. We do not build heuristics.
// Google Maps Enterprise handles the compute; we wire the result into the ledger.
package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
)

// ── Domain Types ─────────────────────────────────────────────────────────────

// MaxWaypointsPerRoute is the hard technical ceiling imposed by the Google Maps
// Directions API. A request with optimize:true supports at most 25 intermediate
// waypoints (origin and destination are separate). Routes exceeding this limit
// are automatically partitioned into sub-routes of MaxWaypointsPerRoute stops.
const MaxWaypointsPerRoute = 25

// DeliveryOrder is the minimal projection the optimizer needs from the Orders table.
// Fetched by GetLoadedOrdersForDriver before calling OptimizeDriverRoute.
type DeliveryOrder struct {
	OrderID      string
	DriverID     string
	ShopLocation string  // WKT "POINT(lon lat)" — parsed to lat/lng below
	ParsedLat    float64 // populated by parseWKT
	ParsedLng    float64 // populated by parseWKT
}

// mapsDirectionsResponse is the shape we decode from the Google Maps
// Directions API. We need waypoint_order for TSP sequencing and legs for
// per-stop ETA / distance data.
type mapsDirectionsResponse struct {
	Status string `json:"status"`
	Routes []struct {
		WaypointOrder []int          `json:"waypoint_order"`
		Legs          []mapsRouteLeg `json:"legs"`
	} `json:"routes"`
}

type mapsRouteLeg struct {
	Duration mapsValue `json:"duration"`
	Distance mapsValue `json:"distance"`
}

type mapsValue struct {
	Value int    `json:"value"` // seconds for duration, meters for distance
	Text  string `json:"text"`
}

// LegETA holds the per-stop ETA data extracted from a Directions API response.
// CumulativeSec is the running total from departure; LegSec/LegM are the
// individual segment values.
type LegETA struct {
	OrderID       string
	SequenceIndex int
	LegSec        int // this leg's duration in seconds
	LegM          int // this leg's distance in meters
	CumulativeSec int // cumulative seconds from depot departure to this stop
}

// RouteETAResult holds the full route ETA data after computing from Directions API response.
type RouteETAResult struct {
	Stops         []LegETA
	ReturnLegSec  int // last leg: final stop → depot
	ReturnLegM    int
	TotalRouteSec int // entire round trip
}

// ── Optimizer ────────────────────────────────────────────────────────────────

// OptimizeDriverRoute is the single public function:
//  1. Builds a Google Maps Directions request with optimize:true.
//  2. Decodes the returned waypoint order (the TSP solution).
//  3. Writes SequenceIndex back to each order row in a single Spanner batch.
//
// Idempotent: re-running overwrites the previous SequenceIndex values, which is
// safe because the cron fires once nightly before drivers pick up their routes.
func OptimizeDriverRoute(
	ctx context.Context,
	client *spanner.Client,
	apiKey string,
	depotLocation string, // "lat,lng" string — the warehouse origin and return point
	orders []DeliveryOrder,
) error {
	if len(orders) == 0 {
		return fmt.Errorf("routing: cannot optimize empty order set")
	}
	if len(orders) == 1 {
		// Single-stop route: sequence is trivially 0. Skip the API call.
		log.Printf("[FieldGeneral] Single-stop route for driver %s — skipping Maps call", orders[0].DriverID)
		return applySequenceToSpanner(ctx, client, []DeliveryOrder{orders[0]}, []int{0}, nil, 0)
	}

	// ── RULE OF 25: Partition if stops exceed the Google Maps waypoint limit ──
	// Google Maps Directions API supports max 25 intermediate waypoints.
	// If a driver has more than 25 stops, partition into sub-routes using a
	// nearest-neighbor spatial pre-sort, then optimize each sub-route independently.
	// Sub-route sequences are stitched into a global SequenceIndex across all orders.
	if len(orders) > MaxWaypointsPerRoute {
		log.Printf("[FieldGeneral] Route for driver %s has %d stops — partitioning into ceil(%d/%d) = %d sub-routes",
			orders[0].DriverID, len(orders), len(orders), MaxWaypointsPerRoute,
			(len(orders)+MaxWaypointsPerRoute-1)/MaxWaypointsPerRoute)
		return optimizePartitionedRoute(ctx, client, apiKey, depotLocation, orders)
	}

	// ── 1. Build waypoints ────────────────────────────────────────────────────
	// Edge 11: Detect duplicate (lat,lng) pairs and add tiny offsets (~1.1m)
	// so Google Maps TSP can distinguish them. Original coords in Spanner untouched.
	type coordKey struct{ lat, lng string }
	seen := make(map[coordKey]int)
	waypoints := make([]string, 0, len(orders))
	for _, o := range orders {
		key := coordKey{fmt.Sprintf("%.6f", o.ParsedLat), fmt.Sprintf("%.6f", o.ParsedLng)}
		count := seen[key]
		seen[key] = count + 1
		lat, lng := o.ParsedLat, o.ParsedLng
		if count > 0 {
			// Offset by ~1.1m per duplicate in alternating directions
			offset := float64(count) * 0.00001
			if count%2 == 0 {
				lat += offset
			} else {
				lng += offset
			}
		}
		waypoints = append(waypoints, fmt.Sprintf("%f,%f", lat, lng))
	}

	// ── 2. Call Google Maps Directions (TSP via optimize:true) ────────────────
	reqURL := fmt.Sprintf(
		"https://maps.googleapis.com/maps/api/directions/json"+
			"?origin=%s&destination=%s"+
			"&waypoints=optimize:true|%s"+
			"&key=%s",
		url.QueryEscape(depotLocation),
		url.QueryEscape(depotLocation), // circular — driver returns to depot
		url.QueryEscape(strings.Join(waypoints, "|")),
		apiKey,
	)

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Get(reqURL)
	if err != nil {
		return fmt.Errorf("routing: Google Maps request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("routing: Google Maps returned HTTP %d", resp.StatusCode)
	}

	var result mapsDirectionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("routing: failed to decode Maps response: %w", err)
	}

	if result.Status != "OK" || len(result.Routes) == 0 {
		// ── FALLBACK: Haversine nearest-neighbor ordering ────────────────────
		// Google Maps returned ZERO_RESULTS or similar — fall back to a greedy
		// nearest-neighbor sort using Haversine distance from the depot.
		log.Printf("[FieldGeneral] Maps rejected request (status=%q) — falling back to Haversine ordering", result.Status)
		return haversineFallbackRoute(ctx, client, depotLocation, orders)
	}

	optimizedOrder := result.Routes[0].WaypointOrder
	if len(optimizedOrder) != len(orders) {
		return fmt.Errorf("routing: waypoint count mismatch: got %d, expected %d", len(optimizedOrder), len(orders))
	}

	// ── 3. Extract per-leg ETA from Directions response ───────────────────────
	// Legs layout for N waypoints with optimize:true and circular depot:
	//   leg[0] = depot → first_stop, leg[1..N-1] = stop-to-stop, leg[N] = last_stop → depot
	legs := result.Routes[0].Legs
	var stopETAs []LegETA
	cumulativeSec := 0
	for i, wpIdx := range optimizedOrder {
		if i < len(legs) {
			cumulativeSec += legs[i].Duration.Value
		}
		stopETAs = append(stopETAs, LegETA{
			OrderID:       orders[wpIdx].OrderID,
			SequenceIndex: i,
			LegSec:        safeGetLegDuration(legs, i),
			LegM:          safeGetLegDistance(legs, i),
			CumulativeSec: cumulativeSec,
		})
	}

	// Return leg is the last leg (final stop → depot)
	returnLegSec := 0
	if len(legs) > len(optimizedOrder) {
		returnLeg := legs[len(optimizedOrder)]
		returnLegSec = returnLeg.Duration.Value
	}

	log.Printf("[FieldGeneral] Route ETA: %d stops, total %d sec + %d sec return", len(stopETAs), cumulativeSec, returnLegSec)

	// ── 4. Write SequenceIndex + ETA to Spanner ──────────────────────────────
	return applySequenceToSpanner(ctx, client, orders, optimizedOrder, stopETAs, returnLegSec)
}

// applySequenceToSpanner executes a Spanner batch mutation that sets
// SequenceIndex and baseline ETA on each order row.  The optimizedOrder slice maps:
//
//	optimizedOrder[sequencePos] = indexIntoOrders
//
// i.e. the order that should be delivered *first* is orders[optimizedOrder[0]].
func applySequenceToSpanner(
	ctx context.Context,
	client *spanner.Client,
	orders []DeliveryOrder,
	optimizedOrder []int,
	stopETAs []LegETA,
	returnLegSec int,
) error {
	mutations := make([]*spanner.Mutation, 0, len(orders)+1)

	// ETA lookup by order ID
	etaMap := make(map[string]LegETA, len(stopETAs))
	for _, e := range stopETAs {
		etaMap[e.OrderID] = e
	}

	var driverID string
	for seqIdx, orderIdx := range optimizedOrder {
		if orderIdx < 0 || orderIdx >= len(orders) {
			return fmt.Errorf("routing: invalid waypoint index %d (have %d orders)", orderIdx, len(orders))
		}
		target := orders[orderIdx]
		seqVal := int64(seqIdx)
		driverID = target.DriverID

		cols := []string{"OrderId", "SequenceIndex", "EstimatedDurationSec", "EstimatedDistanceM"}
		vals := []interface{}{target.OrderID, seqVal, nil, nil}

		if eta, ok := etaMap[target.OrderID]; ok {
			vals[2] = int64(eta.CumulativeSec)
			vals[3] = int64(eta.LegM)
		}

		log.Printf("[FieldGeneral] Lock: OrderID=%s → SequenceIndex=%d, ETA=%v sec", target.OrderID, seqIdx, vals[2])

		mutations = append(mutations, spanner.Update("Orders", cols, vals))
	}

	// Write return-leg duration on the driver row (baseline for RETURNING ETA)
	if driverID != "" && returnLegSec > 0 {
		mutations = append(mutations, spanner.Update("Drivers",
			[]string{"DriverId", "ReturnDurationSec"},
			[]interface{}{driverID, int64(returnLegSec)},
		))
	}

	if _, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite(mutations)
	}); err != nil {
		return fmt.Errorf("routing: Spanner batch commit failed: %w", err)
	}

	log.Printf("[FieldGeneral] Committed %d sequence+ETA mutations to Spanner", len(mutations))
	return nil
}

func safeGetLegDuration(legs []mapsRouteLeg, i int) int {
	if i < len(legs) {
		return legs[i].Duration.Value
	}
	return 0
}

func safeGetLegDistance(legs []mapsRouteLeg, i int) int {
	if i < len(legs) {
		return legs[i].Distance.Value
	}
	return 0
}

// ── Spanner Query Helper ──────────────────────────────────────────────────────

// GetLoadedOrdersForDriver fetches all orders with State='LOADED' or 'DISPATCHED' for a given
// driver, ordered by ascending SequenceIndex (NULL last — unsequenced fallback).
// Called by the cron job to build the optimizer input slice.
func GetLoadedOrdersForDriver(ctx context.Context, client *spanner.Client, driverID string) ([]DeliveryOrder, error) {
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, DriverId, ShopLocation
		      FROM Orders
		      WHERE DriverId = @driverID AND State IN ('LOADED', 'DISPATCHED')
		      ORDER BY SequenceIndex ASC NULLS LAST`,
		Params: map[string]interface{}{
			"driverID": driverID,
		},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var orders []DeliveryOrder
	for {
		row, err := iter.Next()
		if err != nil {
			break // iterator.Done or real error — caller checks Next() contract
		}
		var o DeliveryOrder
		var shopLoc spanner.NullString
		if err := row.Columns(&o.OrderID, &o.DriverID, &shopLoc); err != nil {
			return nil, fmt.Errorf("routing: row parse failed: %w", err)
		}
		if shopLoc.Valid {
			o.ShopLocation = shopLoc.StringVal
			lat, lng, parseErr := parseWKTPoint(shopLoc.StringVal)
			if parseErr != nil {
				log.Printf("[FieldGeneral] WARN: could not parse ShopLocation for order %s: %v", o.OrderID, parseErr)
				continue // skip orders with corrupt geometry rather than crashing
			}
			o.ParsedLat = lat
			o.ParsedLng = lng
		}
		orders = append(orders, o)
	}
	return orders, nil
}

// GetActiveDriverIDs returns the distinct set of driver IDs that currently have
// at least one order in LOADED state.  Used by the nightly cron to know which
// drivers need route optimization before the morning dispatch window.
func GetActiveDriverIDs(ctx context.Context, client *spanner.Client) ([]string, error) {
	stmt := spanner.Statement{
		SQL: `SELECT DISTINCT DriverId FROM Orders WHERE State IN ('LOADED', 'DISPATCHED')`,
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var ids []string
	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		var driverID string
		if err := row.Columns(&driverID); err != nil {
			return nil, fmt.Errorf("routing: driver id parse failed: %w", err)
		}
		ids = append(ids, driverID)
	}
	return ids, nil
}

// parseWKTPoint extracts latitude and longitude from a WKT POINT string.
// Format: "POINT(longitude latitude)" — note Spanner stores (lng lat) per WKT spec.
func parseWKTPoint(wkt string) (lat, lng float64, err error) {
	wkt = strings.TrimSpace(wkt)
	if !strings.HasPrefix(wkt, "POINT(") || !strings.HasSuffix(wkt, ")") {
		return 0, 0, fmt.Errorf("invalid WKT: %q", wkt)
	}
	inner := wkt[6 : len(wkt)-1] // strip "POINT(" and ")"
	parts := strings.Fields(inner)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected 2 coordinates, got %d in %q", len(parts), wkt)
	}
	if _, err = fmt.Sscanf(parts[0], "%f", &lng); err != nil {
		return 0, 0, fmt.Errorf("invalid longitude: %w", err)
	}
	if _, err = fmt.Sscanf(parts[1], "%f", &lat); err != nil {
		return 0, 0, fmt.Errorf("invalid latitude: %w", err)
	}
	return lat, lng, nil
}

// ── Haversine Fallback (ZERO_RESULTS recovery) ──────────────────────────────

// haversineFallbackRoute produces a nearest-neighbor ordering when Google Maps
// is unavailable or returns ZERO_RESULTS. Greedy: always visit the closest
// unvisited stop from the current position, starting from the depot.
func haversineFallbackRoute(
	ctx context.Context,
	client *spanner.Client,
	depotLocation string,
	orders []DeliveryOrder,
) error {
	// Parse depot coordinates from "lat,lng" format
	var depotLat, depotLng float64
	parts := strings.Split(depotLocation, ",")
	if len(parts) == 2 {
		fmt.Sscanf(parts[0], "%f", &depotLat)
		fmt.Sscanf(parts[1], "%f", &depotLng)
	}

	// Greedy nearest-neighbor algorithm
	visited := make([]bool, len(orders))
	sequence := make([]int, 0, len(orders))
	curLat, curLng := depotLat, depotLng

	for range orders {
		bestIdx := -1
		bestDist := math.MaxFloat64
		for i, o := range orders {
			if visited[i] {
				continue
			}
			d := haversineKm(curLat, curLng, o.ParsedLat, o.ParsedLng)
			if d < bestDist {
				bestDist = d
				bestIdx = i
			}
		}
		if bestIdx < 0 {
			break
		}
		visited[bestIdx] = true
		sequence = append(sequence, bestIdx)
		curLat = orders[bestIdx].ParsedLat
		curLng = orders[bestIdx].ParsedLng
	}

	log.Printf("[FieldGeneral] Haversine fallback: %d stops sequenced from depot (%.4f,%.4f)",
		len(sequence), depotLat, depotLng)

	// Write to Spanner with RoutingMethod = HAVERSINE_FALLBACK
	mutations := make([]*spanner.Mutation, 0, len(sequence))
	for seqIdx, orderIdx := range sequence {
		mutations = append(mutations, spanner.Update("Orders",
			[]string{"OrderId", "SequenceIndex", "RoutingMethod"},
			[]interface{}{orders[orderIdx].OrderID, int64(seqIdx), "HAVERSINE_FALLBACK"},
		))
	}

	if _, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite(mutations)
	}); err != nil {
		return fmt.Errorf("routing: Haversine fallback Spanner commit failed: %w", err)
	}

	log.Printf("[FieldGeneral] Committed %d Haversine-sequenced mutations to Spanner", len(mutations))
	return nil
}

// haversineKm computes the great-circle distance between two points in kilometers.
func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

// ── Rule of 25: Partitioned Route Optimization ──────────────────────────────
//
// When a driver has more than MaxWaypointsPerRoute stops, we:
//  1. Spatially pre-sort all stops using nearest-neighbor from the depot
//  2. Slice into partitions of MaxWaypointsPerRoute
//  3. Optimize each partition independently via Google Maps Directions
//  4. Stitch the resulting sequences into a global SequenceIndex
//
// This ensures each Google Maps call respects the 25-waypoint limit while
// the driver gets an end-to-end optimized route across all partitions.

func optimizePartitionedRoute(
	ctx context.Context,
	client *spanner.Client,
	apiKey string,
	depotLocation string,
	orders []DeliveryOrder,
) error {
	// Step 1: Spatial pre-sort using nearest-neighbor from depot
	var depotLat, depotLng float64
	parts := strings.Split(depotLocation, ",")
	if len(parts) == 2 {
		fmt.Sscanf(parts[0], "%f", &depotLat)
		fmt.Sscanf(parts[1], "%f", &depotLng)
	}

	sorted := nearestNeighborSort(orders, depotLat, depotLng)
	totalStops := len(sorted)
	numPartitions := (totalStops + MaxWaypointsPerRoute - 1) / MaxWaypointsPerRoute

	log.Printf("[FieldGeneral:Rule25] Partitioning %d stops into %d sub-routes for driver %s",
		totalStops, numPartitions, orders[0].DriverID)

	// Step 2–3: Optimize each partition and collect the global sequence
	globalMutations := make([]*spanner.Mutation, 0, totalStops)
	globalSeq := 0

	for p := 0; p < numPartitions; p++ {
		start := p * MaxWaypointsPerRoute
		end := start + MaxWaypointsPerRoute
		if end > totalStops {
			end = totalStops
		}
		partition := sorted[start:end]

		// Determine the sub-route depot:
		// - First partition starts from the warehouse depot
		// - Subsequent partitions start from the last stop of the previous partition
		subDepot := depotLocation
		if p > 0 {
			prev := sorted[start-1]
			subDepot = fmt.Sprintf("%f,%f", prev.ParsedLat, prev.ParsedLng)
		}

		log.Printf("[FieldGeneral:Rule25] Sub-route %d/%d: %d stops (global seq %d–%d)",
			p+1, numPartitions, len(partition), globalSeq, globalSeq+len(partition)-1)

		// Try Google Maps optimization for this partition
		subOrder, subETAs, returnSec, err := optimizeSinglePartition(ctx, apiKey, subDepot, partition)
		if err != nil {
			// Fallback: use the pre-sorted order as-is
			log.Printf("[FieldGeneral:Rule25] Sub-route %d optimization failed: %v — using spatial pre-sort", p+1, err)
			for i, o := range partition {
				globalMutations = append(globalMutations, spanner.Update("Orders",
					[]string{"OrderId", "SequenceIndex", "RoutePartition", "RoutingMethod"},
					[]interface{}{o.OrderID, int64(globalSeq + i), int64(p), "PARTITION_FALLBACK"},
				))
			}
			globalSeq += len(partition)
			continue
		}

		// Apply the optimized sub-order with global offset
		etaMap := make(map[string]LegETA, len(subETAs))
		for _, e := range subETAs {
			etaMap[e.OrderID] = e
		}

		for subSeq, orderIdx := range subOrder {
			if orderIdx < 0 || orderIdx >= len(partition) {
				continue
			}
			target := partition[orderIdx]
			cols := []string{"OrderId", "SequenceIndex", "RoutePartition", "EstimatedDurationSec", "EstimatedDistanceM"}
			vals := []interface{}{target.OrderID, int64(globalSeq + subSeq), int64(p), nil, nil}

			if eta, ok := etaMap[target.OrderID]; ok {
				vals[3] = int64(eta.CumulativeSec)
				vals[4] = int64(eta.LegM)
			}

			globalMutations = append(globalMutations, spanner.Update("Orders", cols, vals))
		}

		// Write return-leg for the last partition only
		if p == numPartitions-1 && returnSec > 0 {
			driverID := partition[0].DriverID
			if driverID != "" {
				globalMutations = append(globalMutations, spanner.Update("Drivers",
					[]string{"DriverId", "ReturnDurationSec"},
					[]interface{}{driverID, int64(returnSec)},
				))
			}
		}

		globalSeq += len(partition)
	}

	// Step 4: Commit all mutations atomically
	if _, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite(globalMutations)
	}); err != nil {
		return fmt.Errorf("routing: partitioned Spanner batch commit failed: %w", err)
	}

	log.Printf("[FieldGeneral:Rule25] Committed %d sequence mutations across %d partitions",
		len(globalMutations), numPartitions)
	return nil
}

// optimizeSinglePartition calls Google Maps Directions API for a single sub-route
// and returns (waypointOrder, legETAs, returnLegSec, error).
func optimizeSinglePartition(
	ctx context.Context,
	apiKey string,
	depotLocation string,
	orders []DeliveryOrder,
) ([]int, []LegETA, int, error) {
	if apiKey == "" {
		return nil, nil, 0, fmt.Errorf("no API key")
	}

	// Build waypoints with duplicate offset handling
	type coordKey struct{ lat, lng string }
	seen := make(map[coordKey]int)
	waypoints := make([]string, 0, len(orders))
	for _, o := range orders {
		key := coordKey{fmt.Sprintf("%.6f", o.ParsedLat), fmt.Sprintf("%.6f", o.ParsedLng)}
		count := seen[key]
		seen[key] = count + 1
		lat, lng := o.ParsedLat, o.ParsedLng
		if count > 0 {
			offset := float64(count) * 0.00001
			if count%2 == 0 {
				lat += offset
			} else {
				lng += offset
			}
		}
		waypoints = append(waypoints, fmt.Sprintf("%f,%f", lat, lng))
	}

	reqURL := fmt.Sprintf(
		"https://maps.googleapis.com/maps/api/directions/json"+
			"?origin=%s&destination=%s"+
			"&waypoints=optimize:true|%s"+
			"&key=%s",
		url.QueryEscape(depotLocation),
		url.QueryEscape(depotLocation),
		url.QueryEscape(strings.Join(waypoints, "|")),
		apiKey,
	)

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Get(reqURL)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("Maps request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, 0, fmt.Errorf("Maps returned HTTP %d", resp.StatusCode)
	}

	var result mapsDirectionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, 0, fmt.Errorf("failed to decode Maps response: %w", err)
	}

	if result.Status != "OK" || len(result.Routes) == 0 {
		return nil, nil, 0, fmt.Errorf("Maps status: %s", result.Status)
	}

	optimizedOrder := result.Routes[0].WaypointOrder
	if len(optimizedOrder) != len(orders) {
		return nil, nil, 0, fmt.Errorf("waypoint count mismatch: got %d, expected %d", len(optimizedOrder), len(orders))
	}

	legs := result.Routes[0].Legs
	var stopETAs []LegETA
	cumulativeSec := 0
	for i, wpIdx := range optimizedOrder {
		if i < len(legs) {
			cumulativeSec += legs[i].Duration.Value
		}
		stopETAs = append(stopETAs, LegETA{
			OrderID:       orders[wpIdx].OrderID,
			SequenceIndex: i,
			LegSec:        safeGetLegDuration(legs, i),
			LegM:          safeGetLegDistance(legs, i),
			CumulativeSec: cumulativeSec,
		})
	}

	returnLegSec := 0
	if len(legs) > len(optimizedOrder) {
		returnLegSec = legs[len(optimizedOrder)].Duration.Value
	}

	return optimizedOrder, stopETAs, returnLegSec, nil
}

// nearestNeighborSort spatially pre-sorts orders using greedy nearest-neighbor
// from the depot. Returns a new slice in visit order (does not mutate input).
func nearestNeighborSort(orders []DeliveryOrder, depotLat, depotLng float64) []DeliveryOrder {
	n := len(orders)
	if n == 0 {
		return nil
	}

	sorted := make([]DeliveryOrder, 0, n)
	visited := make([]bool, n)
	curLat, curLng := depotLat, depotLng

	for range orders {
		bestIdx := -1
		bestDist := math.MaxFloat64
		for i, o := range orders {
			if visited[i] {
				continue
			}
			d := haversineKm(curLat, curLng, o.ParsedLat, o.ParsedLng)
			if d < bestDist {
				bestDist = d
				bestIdx = i
			}
		}
		if bestIdx < 0 {
			break
		}
		visited[bestIdx] = true
		sorted = append(sorted, orders[bestIdx])
		curLat = orders[bestIdx].ParsedLat
		curLng = orders[bestIdx].ParsedLng
	}

	return sorted
}
