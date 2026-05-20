package proximity

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ═══════════════════════════════════════════════════════════════════════════════
// ZONE PREVIEW — Real-time density feedback for warehouse coverage planning.
// When a supplier positions or resizes a warehouse zone, this endpoint returns
// the retailers inside the proposed radius, volume stats, and overlap warnings.
// ═══════════════════════════════════════════════════════════════════════════════

// ZonePreview is the response for a proposed warehouse coverage area.
type ZonePreview struct {
	Lat             float64              `json:"lat"`
	Lng             float64              `json:"lng"`
	RadiusKm        float64              `json:"radius_km"`
	RetailersInZone []ZoneRetailer       `json:"retailers_in_zone"`
	RetailerCount   int                  `json:"retailer_count"`
	RecentOrdersDay int64                `json:"recent_orders_day"`
	OverlapWarnings []ZoneOverlapWarning `json:"overlap_warnings"`
}

// ZoneRetailer is a retailer within the proposed zone.
type ZoneRetailer struct {
	RetailerID  string  `json:"retailer_id"`
	Name        string  `json:"name"`
	ShopName    string  `json:"shop_name"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	DistanceKm  float64 `json:"distance_km"`
	OrdersLast7 int64   `json:"orders_last_7"`
}

// ZoneOverlapWarning flags when the proposed zone overlaps with another warehouse.
type ZoneOverlapWarning struct {
	WarehouseID      string  `json:"warehouse_id"`
	WarehouseName    string  `json:"warehouse_name"`
	OverlapRetailers int     `json:"overlap_retailers"`
	DistanceKm       float64 `json:"distance_km"`
}

// HandleZonePreview — GET /v1/supplier/zone-preview?lat=X&lng=Y&radius_km=R
// Returns retailers inside the proposed zone, order volume stats, and overlap warnings.
func HandleZonePreview(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		q := r.URL.Query()
		latStr := q.Get("lat")
		lngStr := q.Get("lng")
		radiusStr := q.Get("radius_km")

		if latStr == "" || lngStr == "" || radiusStr == "" {
			http.Error(w, `{"error":"lat, lng, and radius_km query params required"}`, http.StatusBadRequest)
			return
		}

		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid lat"}`, http.StatusBadRequest)
			return
		}
		lng, err := strconv.ParseFloat(lngStr, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid lng"}`, http.StatusBadRequest)
			return
		}
		radiusKm, err := strconv.ParseFloat(radiusStr, 64)
		if err != nil || radiusKm <= 0 || radiusKm > 500 {
			http.Error(w, `{"error":"invalid radius_km (must be 0 < r <= 500)"}`, http.StatusBadRequest)
			return
		}

		supplierID := claims.ResolveSupplierID()
		ctx := r.Context()

		// Optionally exclude a warehouse (when editing an existing one)
		excludeWH := q.Get("exclude_warehouse_id")

		preview, err := buildZonePreview(ctx, spannerClient, supplierID, lat, lng, radiusKm, excludeWH)
		if err != nil {
			log.Printf("[ZONE-PREVIEW] error for supplier=%s: %v", supplierID, err)
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(preview)
	}
}

// ─── Internal ────────────────────────────────────────────────────────────────

func buildZonePreview(ctx context.Context, client *spanner.Client, supplierID string, lat, lng, radiusKm float64, excludeWH string) (*ZonePreview, error) {
	// 1. Fetch all retailers with coordinates
	retailers, err := fetchRetailersForPreview(ctx, client, lat, lng, radiusKm)
	if err != nil {
		return nil, fmt.Errorf("fetch retailers: %w", err)
	}

	// 2. Compute order volume for retailers in zone (last 7 days)
	retailerIDs := make([]string, len(retailers))
	for i, r := range retailers {
		retailerIDs[i] = r.RetailerID
	}

	var totalOrdersDay int64
	if len(retailerIDs) > 0 {
		orderCounts, dailyAvg, err := fetchOrderVolume(ctx, client, retailerIDs)
		if err != nil {
			log.Printf("[ZONE-PREVIEW] order volume error: %v", err)
		} else {
			totalOrdersDay = dailyAvg
			for i := range retailers {
				if cnt, ok := orderCounts[retailers[i].RetailerID]; ok {
					retailers[i].OrdersLast7 = cnt
				}
			}
		}
	}

	// 3. Check overlaps with existing warehouses
	overlaps, err := checkZoneOverlaps(ctx, client, supplierID, lat, lng, radiusKm, retailers, excludeWH)
	if err != nil {
		log.Printf("[ZONE-PREVIEW] overlap check error: %v", err)
		overlaps = []ZoneOverlapWarning{}
	}

	if retailers == nil {
		retailers = []ZoneRetailer{}
	}

	return &ZonePreview{
		Lat:             lat,
		Lng:             lng,
		RadiusKm:        radiusKm,
		RetailersInZone: retailers,
		RetailerCount:   len(retailers),
		RecentOrdersDay: totalOrdersDay,
		OverlapWarnings: overlaps,
	}, nil
}

func fetchRetailersForPreview(ctx context.Context, client *spanner.Client, centerLat, centerLng, radiusKm float64) ([]ZoneRetailer, error) {
	// Use a bounding box pre-filter for efficiency, then refine with haversine.
	// Approximate: 1° latitude ≈ 111 km
	latDelta := radiusKm / 111.0
	lngDelta := radiusKm / (111.0 * cosDeg(centerLat))

	stmt := spanner.Statement{
		SQL: `SELECT RetailerId, Name, COALESCE(ShopName, ''), IFNULL(Latitude, 0), IFNULL(Longitude, 0)
		      FROM Retailers
		      WHERE Latitude IS NOT NULL AND Longitude IS NOT NULL
		        AND Latitude BETWEEN @minLat AND @maxLat
		        AND Longitude BETWEEN @minLng AND @maxLng`,
		Params: map[string]interface{}{
			"minLat": centerLat - latDelta,
			"maxLat": centerLat + latDelta,
			"minLng": centerLng - lngDelta,
			"maxLng": centerLng + lngDelta,
		},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var results []ZoneRetailer
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var r ZoneRetailer
		if err := row.Columns(&r.RetailerID, &r.Name, &r.ShopName, &r.Latitude, &r.Longitude); err != nil {
			continue
		}

		dist := HaversineKm(centerLat, centerLng, r.Latitude, r.Longitude)
		if dist <= radiusKm {
			r.DistanceKm = dist
			results = append(results, r)
		}
	}

	return results, nil
}

func fetchOrderVolume(ctx context.Context, client *spanner.Client, retailerIDs []string) (map[string]int64, int64, error) {
	if len(retailerIDs) == 0 {
		return nil, 0, nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT RetailerId, COUNT(*) AS OrderCount
		      FROM Orders
		      WHERE RetailerId IN UNNEST(@rids)
		        AND CreatedAt >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY)
		      GROUP BY RetailerId`,
		Params: map[string]interface{}{"rids": retailerIDs},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	counts := make(map[string]int64)
	var total int64
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, 0, err
		}

		var rid string
		var cnt int64
		if err := row.Columns(&rid, &cnt); err != nil {
			continue
		}
		counts[rid] = cnt
		total += cnt
	}

	dailyAvg := total / 7
	return counts, dailyAvg, nil
}

func checkZoneOverlaps(ctx context.Context, client *spanner.Client, supplierID string, lat, lng, radiusKm float64, retailers []ZoneRetailer, excludeWH string) ([]ZoneOverlapWarning, error) {
	// Fetch other active warehouses for this supplier
	sql := `SELECT WarehouseId, Name, IFNULL(Lat, 0), IFNULL(Lng, 0), CoverageRadiusKm
	        FROM Warehouses WHERE SupplierId = @sid AND IsActive = true`
	params := map[string]interface{}{"sid": supplierID}

	if excludeWH != "" {
		sql += " AND WarehouseId != @excludeWH"
		params["excludeWH"] = excludeWH
	}

	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var warnings []ZoneOverlapWarning
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var whID, whName string
		var whLat, whLng, whRadius float64
		if err := row.Columns(&whID, &whName, &whLat, &whLng, &whRadius); err != nil {
			continue
		}

		dist := HaversineKm(lat, lng, whLat, whLng)

		// Check if coverage circles overlap
		if dist < radiusKm+whRadius {
			// Count retailers that fall in both zones
			overlapCount := 0
			for _, r := range retailers {
				distToOther := HaversineKm(r.Latitude, r.Longitude, whLat, whLng)
				if distToOther <= whRadius {
					overlapCount++
				}
			}
			if overlapCount > 0 || dist < radiusKm+whRadius {
				warnings = append(warnings, ZoneOverlapWarning{
					WarehouseID:      whID,
					WarehouseName:    whName,
					OverlapRetailers: overlapCount,
					DistanceKm:       dist,
				})
			}
		}
	}

	if warnings == nil {
		warnings = []ZoneOverlapWarning{}
	}
	return warnings, nil
}

// cosDeg returns cosine of degrees.
func cosDeg(deg float64) float64 {
	const pi = 3.141592653589793
	rad := deg * pi / 180.0
	if rad == 0 {
		return 1.0
	}
	return cosFast(rad)
}

// cosFast is a simple cos approximation; sufficient for bounding box pre-filter.
func cosFast(rad float64) float64 {
	// Use standard library math via the haversine pattern
	// This is just for the bounding box — doesn't need to be exact.
	return 1.0 - rad*rad/2.0 + rad*rad*rad*rad/24.0
}

// ═══════════════════════════════════════════════════════════════════════════════
// VALIDATE COVERAGE — POST /v1/supplier/warehouses/validate-coverage
// Accepts a polygon + H3 resolution from CoverageEditor.tsx and returns the
// computed hex cells, same-supplier overlap conflicts, and retailer count.
// ═══════════════════════════════════════════════════════════════════════════════

type ValidateCoverageRequest struct {
	Polygon      [][2]float64 `json:"polygon"`       // [[lat, lng], ...]
	H3Resolution int          `json:"h3_resolution"` // 7 or 8
	WarehouseID  string       `json:"warehouse_id"`  // optional — exclude self when editing
}

type CoverageConflict struct {
	Hex           string `json:"hex"`
	WarehouseID   string `json:"warehouse_id"`
	WarehouseName string `json:"warehouse_name"`
}

type ValidateCoverageResponse struct {
	Hexes          []string           `json:"hexes"`
	HexesCompacted []string           `json:"hexes_compacted,omitempty"`
	H3Resolution   int                `json:"h3_resolution,omitempty"`
	Conflicts      []CoverageConflict `json:"conflicts"`
	RetailerCount  int                `json:"retailer_count"`
}

// ComputePolygonCoverage returns the H3 cells whose centers fall inside the polygon.
// Invalid resolutions fall back to the canonical supplier coverage resolution.
func ComputePolygonCoverage(polygon [][2]float64, resolution int) ([]string, error) {
	resolution = NormalizeCoverageResolution(resolution)
	if err := ValidateCoveragePolygon(polygon, resolution); err != nil {
		return nil, err
	}

	edgeKm := H3Res7EdgeKm
	if resolution == 8 {
		edgeKm = H3Res7EdgeKm / 2.6457 // res8 ≈ 0.46 km edge
	}
	if cells, err := polygonToCells(polygon, resolution); err == nil && len(cells) > 0 {
		if limitErr := ValidateCoverageCellsCount(len(cells)); limitErr != nil {
			return nil, limitErr
		}
		return cells, nil
	} else if err != nil {
		log.Printf("[POLYGON-COVERAGE] native fill failed; falling back to sampler: %v", err)
	}

	cells, err := computePolygonCoverage(polygon, edgeKm, resolution)
	if err != nil {
		return nil, err
	}
	if err := ValidateCoverageCellsCount(len(cells)); err != nil {
		return nil, err
	}

	return cells, nil
}

// FindCoverageConflicts returns same-supplier warehouses whose H3 coverage overlaps
// the provided cells. excludeWH omits the current warehouse during edit flows.
func FindCoverageConflicts(ctx context.Context, client *spanner.Client, supplierID, excludeWH string, hexes []string) ([]CoverageConflict, error) {
	return findCoverageConflicts(ctx, client, supplierID, excludeWH, hexes)
}

func HandleValidateCoverage(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req ValidateCoverageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		req.H3Resolution = NormalizeCoverageResolution(req.H3Resolution)
		if err := ValidateCoveragePolygon(req.Polygon, req.H3Resolution); err != nil {
			writeCoverageHTTPError(w, err)
			return
		}

		supplierID := claims.ResolveSupplierID()
		ctx := r.Context()

		// 1. Compute grid cells covering the polygon
		hexes, err := ComputePolygonCoverage(req.Polygon, req.H3Resolution)
		if err != nil {
			writeCoverageHTTPError(w, err)
			return
		}
		if err := ValidateCoverageResponseCount(len(hexes)); err != nil {
			writeCoverageHTTPError(w, err)
			return
		}
		hexesCompacted, compactErr := CompactCells(hexes)
		if compactErr != nil {
			log.Printf("[VALIDATE-COVERAGE] compact cells failed: %v", compactErr)
			hexesCompacted = append([]string(nil), hexes...)
		}

		resolutionLabel := strconv.Itoa(req.H3Resolution)
		if len(hexes) > 0 {
			SpatialCompactionRatio.WithLabelValues("validate_coverage", resolutionLabel).Observe(float64(len(hexesCompacted)) / float64(len(hexes)))
		}

		if len(hexesCompacted) > MaxCompactedEgressCells {
			SpatialEgressOverflowTotal.WithLabelValues("validate_coverage", resolutionLabel).Inc()
			log.Printf("[SPATIAL-EGRESS][WARN] zone_type=validate_coverage resolution=%d compacted_cells=%d threshold=%d action=truncate", req.H3Resolution, len(hexesCompacted), MaxCompactedEgressCells)
			hexesCompacted = append([]string(nil), hexesCompacted[:MaxCompactedEgressCells]...)
		}

		if len(hexes) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ValidateCoverageResponse{
				Hexes:          []string{},
				HexesCompacted: []string{},
				H3Resolution:   req.H3Resolution,
				Conflicts:      []CoverageConflict{},
			})
			return
		}

		// 2. Query existing warehouse H3 indexes for overlap detection
		conflicts, err := FindCoverageConflicts(ctx, spannerClient, supplierID, req.WarehouseID, hexes)
		if err != nil {
			log.Printf("[VALIDATE-COVERAGE] conflict query error for supplier=%s: %v", supplierID, err)
			conflicts = []CoverageConflict{}
		}

		// 3. Count retailers inside the polygon
		retailerCount, err := countRetailersInPolygon(ctx, spannerClient, req.Polygon)
		if err != nil {
			log.Printf("[VALIDATE-COVERAGE] retailer count error: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ValidateCoverageResponse{
			Hexes:          hexes,
			HexesCompacted: hexesCompacted,
			H3Resolution:   req.H3Resolution,
			Conflicts:      conflicts,
			RetailerCount:  retailerCount,
		})
	}
}

// computePolygonCoverage generates grid cells whose centers fall inside the polygon.
func computePolygonCoverage(polygon [][2]float64, edgeKm float64, resolution int) ([]string, error) {
	// Compute bounding box
	minLat, maxLat := polygon[0][0], polygon[0][0]
	minLng, maxLng := polygon[0][1], polygon[0][1]
	for _, pt := range polygon[1:] {
		if pt[0] < minLat {
			minLat = pt[0]
		}
		if pt[0] > maxLat {
			maxLat = pt[0]
		}
		if pt[1] < minLng {
			minLng = pt[1]
		}
		if pt[1] > maxLng {
			maxLng = pt[1]
		}
	}

	stepLat := edgeKm / 111.0
	centerLat := (minLat + maxLat) / 2.0
	cosLat := cosDeg(centerLat)
	if cosLat < 1e-10 {
		cosLat = 1e-10
	}
	stepLng := edgeKm / (111.0 * cosLat)
	if stepLat <= 0 || stepLng <= 0 || math.IsNaN(stepLat) || math.IsNaN(stepLng) || math.IsInf(stepLng, 0) {
		return nil, ErrCoverageSamplerLimitExceeded
	}

	latSpan := (maxLat + stepLat) - (minLat - stepLat)
	lngSpan := (maxLng + stepLng) - (minLng - stepLng)
	latSteps := int(math.Ceil(latSpan/stepLat)) + 1
	lngSteps := int(math.Ceil(lngSpan/stepLng)) + 1
	iterations := latSteps * lngSteps
	if iterations > MaxCoverageFallbackIterations {
		return nil, fmt.Errorf("%w: iterations=%d max=%d", ErrCoverageSamplerLimitExceeded, iterations, MaxCoverageFallbackIterations)
	}

	capacity := iterations / 4
	if capacity < 64 {
		capacity = 64
	}
	if capacity > MaxCoverageCells {
		capacity = MaxCoverageCells
	}

	cells := make([]string, 0, capacity)
	seen := make(map[string]struct{}, capacity)

	for lat := minLat - stepLat; lat <= maxLat+stepLat; lat += stepLat {
		for lng := minLng - stepLng; lng <= maxLng+stepLng; lng += stepLng {
			if !pointInPolygon(lat, lng, polygon) {
				continue
			}
			cell, err := safeLatLngToCell(lat, lng, resolution)
			if err != nil {
				continue
			}
			id := cell.String()
			if _, exists := seen[id]; !exists {
				if len(cells) >= MaxCoverageCells {
					return nil, fmt.Errorf("%w: got >=%d", ErrCoverageCellLimitExceeded, MaxCoverageCells)
				}
				seen[id] = struct{}{}
				cells = append(cells, id)
			}
		}
	}

	return cells, nil
}

// pointInPolygon uses ray-casting to test if (lat, lng) is inside the polygon.
func pointInPolygon(lat, lng float64, polygon [][2]float64) bool {
	n := len(polygon)
	inside := false
	j := n - 1
	for i := 0; i < n; i++ {
		yi, xi := polygon[i][0], polygon[i][1]
		yj, xj := polygon[j][0], polygon[j][1]

		if ((yi > lat) != (yj > lat)) &&
			(lng < (xj-xi)*(lat-yi)/(yj-yi)+xi) {
			inside = !inside
		}
		j = i
	}
	return inside
}

func writeCoverageHTTPError(w http.ResponseWriter, err error) {
	status := CoverageErrorStatus(err)
	if status == http.StatusInternalServerError {
		log.Printf("[COVERAGE] unexpected coverage error: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": CoverageErrorMessage(err)})
}

// findCoverageConflicts queries sibling warehouses for overlapping H3 cells.
func findCoverageConflicts(ctx context.Context, client *spanner.Client, supplierID, excludeWH string, hexes []string) ([]CoverageConflict, error) {
	sql := `SELECT w.WarehouseId, w.Name, h3
	        FROM Warehouses w, UNNEST(w.H3Indexes) AS h3
	        WHERE w.SupplierId = @sid
	          AND w.IsActive = true
	          AND h3 IN UNNEST(@hexes)`
	params := map[string]interface{}{
		"sid":   supplierID,
		"hexes": hexes,
	}

	if excludeWH != "" {
		sql += " AND w.WarehouseId != @excludeWH"
		params["excludeWH"] = excludeWH
	}

	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var conflicts []CoverageConflict
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var c CoverageConflict
		if err := row.Columns(&c.WarehouseID, &c.WarehouseName, &c.Hex); err != nil {
			continue
		}
		conflicts = append(conflicts, c)
	}

	if conflicts == nil {
		conflicts = []CoverageConflict{}
	}
	return conflicts, nil
}

// countRetailersInPolygon counts retailers whose coordinates fall inside the polygon.
func countRetailersInPolygon(ctx context.Context, client *spanner.Client, polygon [][2]float64) (int, error) {
	// Bounding box pre-filter
	minLat, maxLat := polygon[0][0], polygon[0][0]
	minLng, maxLng := polygon[0][1], polygon[0][1]
	for _, pt := range polygon[1:] {
		if pt[0] < minLat {
			minLat = pt[0]
		}
		if pt[0] > maxLat {
			maxLat = pt[0]
		}
		if pt[1] < minLng {
			minLng = pt[1]
		}
		if pt[1] > maxLng {
			maxLng = pt[1]
		}
	}

	stmt := spanner.Statement{
		SQL: `SELECT Latitude, Longitude
		      FROM Retailers
		      WHERE Latitude IS NOT NULL AND Longitude IS NOT NULL
		        AND Latitude BETWEEN @minLat AND @maxLat
		        AND Longitude BETWEEN @minLng AND @maxLng`,
		Params: map[string]interface{}{
			"minLat": minLat,
			"maxLat": maxLat,
			"minLng": minLng,
			"maxLng": maxLng,
		},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	count := 0
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}

		var lat, lng float64
		if err := row.Columns(&lat, &lng); err != nil {
			continue
		}
		if pointInPolygon(lat, lng, polygon) {
			count++
		}
	}

	return count, nil
}
