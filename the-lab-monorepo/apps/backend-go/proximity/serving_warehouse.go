package proximity

import (
	"context"
	"fmt"
	"log"

	"backend-go/cache"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ─── Serving Warehouse Resolution ──────────────────────────────────────────────
//
// GetServingWarehouse resolves the exclusive warehouse that should serve a
// retailer for a given supplier. Resolution priority:
//
//   1. H3 polygon membership — retailer's H3Index ∈ warehouse's H3Indexes
//   2. Haversine distance — nearest warehouse within CoverageRadiusKm
//
// If multiple warehouses cover the same retailer via H3, the closest one wins.
// This enforces a single-warehouse-per-supplier-per-retailer model.

// ServingWarehouse represents an exclusive warehouse assignment for a retailer.
type ServingWarehouse struct {
	WarehouseId         string  `json:"warehouse_id"`
	SupplierId          string  `json:"supplier_id"`
	Name                string  `json:"name"`
	DistanceKm          float64 `json:"distance_km"`
	Lat                 float64 `json:"lat"`
	Lng                 float64 `json:"lng"`
	MatchMethod         string  `json:"match_method"` // "H3_POLYGON" | "HAVERSINE" | "GRID_CELL"
	CoverageRadiusKm    float64 `json:"coverage_radius_km"`
	Rerouted            bool    `json:"rerouted,omitempty"`
	OriginalWarehouseId string  `json:"original_warehouse_id,omitempty"`
	LoadPercent         float64 `json:"load_percent,omitempty"`
	LoadWarning         bool    `json:"load_warning,omitempty"`
}

// GetServingWarehouse finds the single warehouse under a supplier that should
// serve a retailer at the given coordinates. Returns nil if no warehouse covers
// the retailer's location.
func GetServingWarehouse(ctx context.Context, client *spanner.Client, supplierID string, retailerLat, retailerLng float64) (*ServingWarehouse, error) {
	if client == nil {
		return nil, fmt.Errorf("spanner client is nil")
	}

	// Edge 17: Check PreferredWarehouseId FIRST — bypasses all geo resolution
	preferred, err := resolvePreferredWarehouse(ctx, client, supplierID, retailerLat, retailerLng)
	if err != nil {
		log.Printf("[SERVING-WH] Preferred warehouse check failed: %v — falling through to geo", err)
	}
	if preferred != nil {
		return preferred, nil
	}

	retailerH3 := LookupCell(retailerLat, retailerLng)

	// Path 1: H3 polygon membership — check if retailer's cell is inside any warehouse's coverage
	match, err := resolveViaH3Polygon(ctx, client, supplierID, retailerH3, retailerLat, retailerLng)
	if err != nil {
		log.Printf("[SERVING-WH] H3 polygon lookup failed: %v — falling back to distance", err)
	}
	if match != nil {
		return applyLoadBalancing(ctx, client, match, supplierID, retailerLat, retailerLng), nil
	}

	// Path 2: Grid cell (Redis O(1) lookup) → existing ResolveWarehouse path
	gridMatch, err := resolveViaGridCell(ctx, supplierID, retailerLat, retailerLng)
	if err != nil {
		log.Printf("[SERVING-WH] Grid cell lookup failed: %v", err)
	}
	if gridMatch != nil {
		sw := &ServingWarehouse{
			WarehouseId:      gridMatch.WarehouseId,
			SupplierId:       gridMatch.SupplierId,
			Name:             gridMatch.Name,
			DistanceKm:       gridMatch.DistanceKm,
			Lat:              gridMatch.Lat,
			Lng:              gridMatch.Lng,
			MatchMethod:      "GRID_CELL",
			CoverageRadiusKm: 0, // Not available from grid cell path
		}
		return applyLoadBalancing(ctx, client, sw, supplierID, retailerLat, retailerLng), nil
	}

	// Path 3: Haversine distance fallback via Spanner
	havMatch, err := resolveViaHaversine(ctx, client, supplierID, retailerLat, retailerLng)
	if err != nil {
		return nil, err
	}
	if havMatch != nil {
		return applyLoadBalancing(ctx, client, havMatch, supplierID, retailerLat, retailerLng), nil
	}
	return nil, nil
}

// loadRerouteThreshold — reroute triggers when warehouse load exceeds this.
const loadRerouteThreshold = 0.9

// loadTargetCeiling — reroute target must be below this to prevent cascading overload.
const loadTargetCeiling = 0.7

// defaultMaxCapacity is used when MaxCapacityThreshold is NULL in Spanner.
const defaultMaxCapacity int64 = 100

// applyLoadBalancing checks if the resolved warehouse is overloaded (>90%).
// If so, it queries sibling warehouses (same supplier, active, on-shift) and
// picks the nearest one with load < 70%. Returns the rerouted warehouse or
// the original with a LoadWarning flag.
func applyLoadBalancing(ctx context.Context, client *spanner.Client, resolved *ServingWarehouse, supplierID string, retailerLat, retailerLng float64) *ServingWarehouse {
	// Fetch MaxCapacityThreshold for the resolved warehouse
	maxCap := getWarehouseMaxCapacity(ctx, client, resolved.WarehouseId)
	load := cache.GetWarehouseLoad(ctx, resolved.WarehouseId, maxCap)
	resolved.LoadPercent = load

	if load < loadRerouteThreshold {
		return resolved // healthy — no reroute needed
	}

	log.Printf("[LOAD-BALANCE] Warehouse %s at %.0f%% load — scanning siblings", resolved.WarehouseId, load*100)

	// Oscillation guard (A-3): If this warehouse was a reroute TARGET within the last 5 minutes,
	// skip rerouting to prevent ping-pong between warehouses. Uses Redis cooldown key.
	if cache.Client != nil {
		cooldownKey := cache.PrefixLBCooldown + resolved.WarehouseId
		exists, err := cache.Client.Exists(ctx, cooldownKey).Result()
		if err == nil && exists > 0 {
			log.Printf("[LOAD-BALANCE] Oscillation guard: warehouse %s was recently a reroute target — skipping", resolved.WarehouseId)
			resolved.LoadWarning = true
			return resolved
		}
	}

	// Query sibling warehouses (same supplier, active, on-shift, different from resolved)
	siblings, err := fetchSiblingWarehouses(ctx, client, supplierID, resolved.WarehouseId)
	if err != nil {
		log.Printf("[LOAD-BALANCE] Failed to fetch siblings: %v — returning overloaded warehouse with warning", err)
		resolved.LoadWarning = true
		return resolved
	}

	// Find the nearest sibling with load < 70% within 2× the resolved warehouse's coverage radius
	maxRerouteDistance := resolved.CoverageRadiusKm * 2
	if maxRerouteDistance < 20 {
		maxRerouteDistance = 20 // minimum 20km reroute radius
	}

	var bestSibling *siblingWarehouse
	var bestDist float64

	for i := range siblings {
		s := &siblings[i]
		sibLoad := cache.GetWarehouseLoad(ctx, s.WarehouseId, s.MaxCapacity)
		if sibLoad >= loadTargetCeiling {
			continue // sibling is too busy
		}

		dist := HaversineKm(retailerLat, retailerLng, s.Lat, s.Lng)
		if dist > maxRerouteDistance {
			continue // too far
		}

		if bestSibling == nil || dist < bestDist {
			bestSibling = s
			bestSibling.load = sibLoad
			bestDist = dist
		}
	}

	if bestSibling == nil {
		log.Printf("[LOAD-BALANCE] No viable sibling for warehouse %s — returning with load warning", resolved.WarehouseId)
		resolved.LoadWarning = true
		return resolved
	}

	log.Printf("[LOAD-BALANCE] Rerouting from %s (%.0f%%) to %s (%.0f%%) — distance %.1fkm",
		resolved.WarehouseId, load*100, bestSibling.WarehouseId, bestSibling.load*100, bestDist)

	// Set oscillation cooldown (A-3): prevent this target from being rerouted FROM for 5 minutes
	if cache.Client != nil {
		cooldownKey := cache.PrefixLBCooldown + bestSibling.WarehouseId
		cache.Client.Set(ctx, cooldownKey, "1", cache.TTLLBCooldown)
	}

	return &ServingWarehouse{
		WarehouseId:         bestSibling.WarehouseId,
		SupplierId:          supplierID,
		Name:                bestSibling.Name,
		DistanceKm:          bestDist,
		Lat:                 bestSibling.Lat,
		Lng:                 bestSibling.Lng,
		MatchMethod:         resolved.MatchMethod,
		CoverageRadiusKm:    bestSibling.CoverageRadiusKm,
		Rerouted:            true,
		OriginalWarehouseId: resolved.WarehouseId,
		LoadPercent:         bestSibling.load,
	}
}

// siblingWarehouse holds candidate warehouses for load-based rerouting.
type siblingWarehouse struct {
	WarehouseId      string
	Name             string
	Lat              float64
	Lng              float64
	CoverageRadiusKm float64
	MaxCapacity      int64
	load             float64 // populated during comparison
}

// fetchSiblingWarehouses returns active, on-shift warehouses for the same supplier,
// excluding the overloaded one.
func fetchSiblingWarehouses(ctx context.Context, client *spanner.Client, supplierID, excludeWarehouseID string) ([]siblingWarehouse, error) {
	stmt := spanner.Statement{
		SQL: `SELECT WarehouseId, Name, Lat, Lng, COALESCE(CoverageRadiusKm, 50) AS CoverageRadiusKm,
		             COALESCE(MaxCapacityThreshold, 100) AS MaxCapacity
		      FROM Warehouses
		      WHERE SupplierId = @sid
		        AND IsActive = true
		        AND COALESCE(IsOnShift, true) = true
		        AND WarehouseId != @excludeId`,
		Params: map[string]interface{}{
			"sid":       supplierID,
			"excludeId": excludeWarehouseID,
		},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var results []siblingWarehouse
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("fetch siblings: %w", err)
		}

		var s siblingWarehouse
		var lat, lng spanner.NullFloat64
		if err := row.Columns(&s.WarehouseId, &s.Name, &lat, &lng, &s.CoverageRadiusKm, &s.MaxCapacity); err != nil {
			continue
		}
		if !lat.Valid || !lng.Valid {
			continue
		}
		s.Lat = lat.Float64
		s.Lng = lng.Float64
		results = append(results, s)
	}

	return results, nil
}

// getWarehouseMaxCapacity fetches the MaxCapacityThreshold for a single warehouse.
// Returns defaultMaxCapacity (100) if the column is NULL or query fails.
func getWarehouseMaxCapacity(ctx context.Context, client *spanner.Client, warehouseID string) int64 {
	row, err := client.Single().ReadRow(ctx, "Warehouses", spanner.Key{warehouseID}, []string{"MaxCapacityThreshold"})
	if err != nil {
		return defaultMaxCapacity
	}
	var maxCap spanner.NullInt64
	if err := row.Columns(&maxCap); err != nil || !maxCap.Valid {
		return defaultMaxCapacity
	}
	return maxCap.Int64
}

// resolveViaH3Polygon queries Spanner for warehouses whose H3Indexes array
// contains the retailer's H3 cell ID. This is the most precise match method.
func resolveViaH3Polygon(ctx context.Context, client *spanner.Client, supplierID, retailerH3 string, retailerLat, retailerLng float64) (*ServingWarehouse, error) {
	// Spanner does not have a native ARRAY_CONTAINS function. For arrays we use
	// an UNNEST subquery to check membership.
	stmt := spanner.Statement{
		SQL: `SELECT w.WarehouseId, w.Name, w.Lat, w.Lng, w.CoverageRadiusKm
		      FROM Warehouses w
		      WHERE w.SupplierId = @supplierId
		        AND w.IsActive = true
		        AND COALESCE(w.IsOnShift, true) = true
		        AND @h3cell IN UNNEST(w.H3Indexes)`,
		Params: map[string]interface{}{
			"supplierId": supplierID,
			"h3cell":     retailerH3,
		},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var best *ServingWarehouse
	var bestDist float64

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("H3 polygon query: %w", err)
		}

		var whID, name string
		var lat, lng spanner.NullFloat64
		var radiusKm float64

		if err := row.Columns(&whID, &name, &lat, &lng, &radiusKm); err != nil {
			log.Printf("[SERVING-WH] parse error: %v", err)
			continue
		}

		if !lat.Valid || !lng.Valid {
			continue
		}

		dist := HaversineKm(retailerLat, retailerLng, lat.Float64, lng.Float64)
		if best == nil || dist < bestDist {
			best = &ServingWarehouse{
				WarehouseId:      whID,
				SupplierId:       supplierID,
				Name:             name,
				DistanceKm:       dist,
				Lat:              lat.Float64,
				Lng:              lng.Float64,
				MatchMethod:      "H3_POLYGON",
				CoverageRadiusKm: radiusKm,
			}
			bestDist = dist
		}
	}

	return best, nil
}

// resolveViaHaversine queries all active warehouses for a supplier and picks
// the nearest one within its CoverageRadiusKm. Ultimate fallback.
func resolveViaHaversine(ctx context.Context, client *spanner.Client, supplierID string, retailerLat, retailerLng float64) (*ServingWarehouse, error) {
	stmt := spanner.Statement{
		SQL: `SELECT WarehouseId, Name, Lat, Lng, CoverageRadiusKm
		      FROM Warehouses
		      WHERE SupplierId = @supplierId AND IsActive = true
		        AND COALESCE(IsOnShift, true) = true
		      ORDER BY IsDefault DESC`,
		Params: map[string]interface{}{
			"supplierId": supplierID,
		},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var best *ServingWarehouse
	var bestDist float64

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("haversine query: %w", err)
		}

		var whID, name string
		var lat, lng spanner.NullFloat64
		var radiusKm float64

		if err := row.Columns(&whID, &name, &lat, &lng, &radiusKm); err != nil {
			continue
		}

		if !lat.Valid || !lng.Valid {
			continue
		}

		dist := HaversineKm(retailerLat, retailerLng, lat.Float64, lng.Float64)
		if dist > radiusKm {
			continue // outside coverage
		}

		if best == nil || dist < bestDist {
			best = &ServingWarehouse{
				WarehouseId:      whID,
				SupplierId:       supplierID,
				Name:             name,
				DistanceKm:       dist,
				Lat:              lat.Float64,
				Lng:              lng.Float64,
				MatchMethod:      "HAVERSINE",
				CoverageRadiusKm: radiusKm,
			}
			bestDist = dist
		}
	}

	return best, nil
}

// CheckCoverageOverlap finds warehouses owned by OTHER suppliers whose
// H3Indexes overlap with the given set of cells. Used to detect polygon
// conflicts when a supplier creates or updates warehouse coverage.
func CheckCoverageOverlap(ctx context.Context, client *spanner.Client, excludeSupplierID string, cells []string) ([]OverlapResult, error) {
	if client == nil || len(cells) == 0 {
		return nil, nil
	}

	// Sample up to 50 cells to keep the query bounded
	sample := cells
	if len(sample) > 50 {
		sample = sample[:50]
	}

	stmt := spanner.Statement{
		SQL: `SELECT w.WarehouseId, w.SupplierId, w.Name, cell
		      FROM Warehouses w, UNNEST(w.H3Indexes) AS cell
		      WHERE w.SupplierId != @excludeSid
		        AND w.IsActive = true
		        AND cell IN UNNEST(@cells)
		      LIMIT 100`,
		Params: map[string]interface{}{
			"excludeSid": excludeSupplierID,
			"cells":      sample,
		},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	seen := map[string]bool{}
	var results []OverlapResult

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("overlap query: %w", err)
		}

		var whID, sid, name, cell string
		if err := row.Columns(&whID, &sid, &name, &cell); err != nil {
			continue
		}

		key := whID + ":" + cell
		if seen[key] {
			continue
		}
		seen[key] = true

		results = append(results, OverlapResult{
			WarehouseId:     whID,
			SupplierId:      sid,
			WarehouseName:   name,
			OverlappingCell: cell,
		})
	}

	return results, nil
}

// OverlapResult describes a detected coverage overlap with another supplier's warehouse.
type OverlapResult struct {
	WarehouseId     string `json:"warehouse_id"`
	SupplierId      string `json:"supplier_id"`
	WarehouseName   string `json:"warehouse_name"`
	OverlappingCell string `json:"overlapping_cell"`
}

// ── Edge 17: Preferred Warehouse Resolution ───────────────────────────────────

// resolvePreferredWarehouse checks if the retailer nearest to these coordinates
// has a PreferredWarehouseId set. If so, verifies the warehouse is active + on-shift.
func resolvePreferredWarehouse(ctx context.Context, client *spanner.Client, supplierID string, retailerLat, retailerLng float64) (*ServingWarehouse, error) {
	// Find the retailer that matches these coordinates (within ~50m)
	stmt := spanner.Statement{
		SQL: `SELECT RetailerId, PreferredWarehouseId FROM Retailers
		      WHERE PreferredWarehouseId IS NOT NULL
		      AND ABS(Latitude - @lat) < 0.0005 AND ABS(Longitude - @lng) < 0.0005
		      LIMIT 1`,
		Params: map[string]interface{}{"lat": retailerLat, "lng": retailerLng},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return nil, nil // No retailer with preferred warehouse near these coords
	}
	var retailerID, prefWH string
	if err := row.Columns(&retailerID, &prefWH); err != nil {
		return nil, nil
	}

	// Verify the preferred warehouse is active and on-shift under this supplier
	whRow, err := client.Single().ReadRow(ctx, "Warehouses", spanner.Key{prefWH},
		[]string{"SupplierId", "Name", "Lat", "Lng", "IsActive", "IsOnShift", "CoverageRadiusKm"})
	if err != nil {
		log.Printf("[SERVING-WH] Preferred warehouse %s not found: %v", prefWH, err)
		return nil, nil // Fall through to geo resolution
	}
	var whSID, whName string
	var whLat, whLng, coverageKm float64
	var isActive, isOnShift bool
	if err := whRow.Columns(&whSID, &whName, &whLat, &whLng, &isActive, &isOnShift, &coverageKm); err != nil {
		return nil, nil
	}

	// Supplier must match and warehouse must be operational
	if whSID != supplierID || !isActive || !isOnShift {
		log.Printf("[SERVING-WH] Preferred warehouse %s for retailer %s is offline/wrong supplier — falling through",
			prefWH, retailerID)
		return nil, nil
	}

	log.Printf("[SERVING-WH] Using preferred warehouse %s (%s) for retailer %s", prefWH, whName, retailerID)
	return &ServingWarehouse{
		WarehouseId:      prefWH,
		SupplierId:       whSID,
		Name:             whName,
		DistanceKm:       HaversineKm(retailerLat, retailerLng, whLat, whLng),
		Lat:              whLat,
		Lng:              whLng,
		MatchMethod:      "PREFERRED",
		CoverageRadiusKm: coverageKm,
	}, nil
}
