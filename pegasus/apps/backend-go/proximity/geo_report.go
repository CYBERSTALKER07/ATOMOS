package proximity

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"backend-go/auth"
	"backend-go/cache"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ═══════════════════════════════════════════════════════════════════════════════
// GEO-REPORT: Coverage audit — dead zones, overlaps, and serving assignments
// ═══════════════════════════════════════════════════════════════════════════════

// GeoReport contains the full spatial health audit for a supplier's coverage.
type GeoReport struct {
	TotalRetailers   int                `json:"total_retailers"`
	CoveredRetailers int                `json:"covered_retailers"`
	DeadZones        []DeadZoneRetailer `json:"dead_zones"`
	Overlaps         []OverlapResult    `json:"overlaps"`
	Warehouses       []WarehouseSummary `json:"warehouses"`
}

// DeadZoneRetailer is a retailer with no serving warehouse.
type DeadZoneRetailer struct {
	RetailerID string  `json:"retailer_id"`
	Name       string  `json:"name"`
	ShopName   string  `json:"shop_name"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	H3Index    string  `json:"h3_index"`
}

// WarehouseSummary shows a warehouse's coverage stats in the report.
type WarehouseSummary struct {
	WarehouseId      string  `json:"warehouse_id"`
	Name             string  `json:"name"`
	Lat              float64 `json:"lat"`
	Lng              float64 `json:"lng"`
	CoverageRadiusKm float64 `json:"coverage_radius_km"`
	CellCount        int     `json:"cell_count"`
	RetailersCovered int     `json:"retailers_covered"`
	QueueDepth       int64   `json:"queue_depth"`
	MaxCapacity      int64   `json:"max_capacity"`
	LoadPercent      float64 `json:"load_percent"`
	LoadStatus       string  `json:"load_status"` // GREEN | YELLOW | RED
}

// HandleGeoReport — GET /v1/supplier/geo-report
// Returns coverage health: dead zones (retailers with no serving warehouse),
// cross-supplier overlaps, and per-warehouse stats.
func HandleGeoReport(spannerClient *spanner.Client) http.HandlerFunc {
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

		supplierID := claims.ResolveSupplierID()
		ctx := r.Context()

		report, err := buildGeoReport(ctx, spannerClient, supplierID)
		if err != nil {
			log.Printf("[GEO-REPORT] error for supplier=%s: %v", supplierID, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	}
}

// HandleGetServingWarehouse — GET /v1/supplier/serving-warehouse?retailer_lat=X&retailer_lng=Y
// Returns the exclusive warehouse assignment for a retailer at the given coordinates.
func HandleGetServingWarehouse(spannerClient *spanner.Client, readRouter ReadRouter) http.HandlerFunc {
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

		latStr := r.URL.Query().Get("retailer_lat")
		lngStr := r.URL.Query().Get("retailer_lng")
		if latStr == "" || lngStr == "" {
			http.Error(w, `{"error":"retailer_lat and retailer_lng query params required"}`, http.StatusBadRequest)
			return
		}

		var lat, lng float64
		if _, err := fmt.Sscanf(latStr, "%f", &lat); err != nil {
			http.Error(w, `{"error":"invalid retailer_lat"}`, http.StatusBadRequest)
			return
		}
		if _, err := fmt.Sscanf(lngStr, "%f", &lng); err != nil {
			http.Error(w, `{"error":"invalid retailer_lng"}`, http.StatusBadRequest)
			return
		}

		match, err := GetServingWarehouseWithRouter(r.Context(), spannerClient, readRouter, claims.ResolveSupplierID(), lat, lng)
		if err != nil {
			log.Printf("[SERVING-WH] error for supplier=%s lat=%.6f lng=%.6f: %v", claims.UserID, lat, lng, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if match == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "no warehouse covers this location",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(match)
	}
}

// ─── Internal: Build the full geo report ─────────────────────────────────────

func buildGeoReport(ctx context.Context, client *spanner.Client, supplierID string) (*GeoReport, error) {
	// 1. Fetch all warehouses for this supplier
	warehouses, err := fetchWarehousesForReport(ctx, client, supplierID)
	if err != nil {
		return nil, fmt.Errorf("fetch warehouses: %w", err)
	}

	// 2. Fetch all retailers with coordinates
	retailers, err := fetchRetailersWithCoords(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("fetch retailers: %w", err)
	}

	// 3. Check each retailer against supplier's warehouses
	var deadZones []DeadZoneRetailer
	whCoveredCounts := make(map[string]int) // warehouseID → count

	for _, ret := range retailers {
		served := false
		for wi, wh := range warehouses {
			// Check H3 polygon membership first
			if ret.H3Index != "" && containsCell(wh.H3Indexes, ret.H3Index) {
				served = true
				whCoveredCounts[warehouses[wi].WarehouseId]++
				break
			}
			// Fallback: haversine distance
			dist := HaversineKm(ret.Latitude, ret.Longitude, wh.Lat, wh.Lng)
			if dist <= wh.CoverageRadiusKm {
				served = true
				whCoveredCounts[warehouses[wi].WarehouseId]++
				break
			}
		}
		if !served {
			deadZones = append(deadZones, ret)
		}
	}

	// 4. Check for cross-supplier overlaps
	var allCells []string
	for _, wh := range warehouses {
		allCells = append(allCells, wh.H3Indexes...)
	}
	overlaps, err := CheckCoverageOverlap(ctx, client, supplierID, allCells)
	if err != nil {
		log.Printf("[GEO-REPORT] overlap check error: %v", err)
		overlaps = nil
	}

	// 5. Build warehouse summaries with load data from Redis
	whIDs := make([]string, len(warehouses))
	for i, wh := range warehouses {
		whIDs[i] = wh.WarehouseId
	}
	queueDepths := cache.GetAllWarehouseLoads(ctx, whIDs)

	summaries := make([]WarehouseSummary, len(warehouses))
	for i, wh := range warehouses {
		depth := queueDepths[wh.WarehouseId]
		maxCap := wh.MaxCapacity
		if maxCap <= 0 {
			maxCap = 100
		}
		loadPct := float64(depth) / float64(maxCap)
		if loadPct > 1.0 {
			loadPct = 1.0
		}

		summaries[i] = WarehouseSummary{
			WarehouseId:      wh.WarehouseId,
			Name:             wh.Name,
			Lat:              wh.Lat,
			Lng:              wh.Lng,
			CoverageRadiusKm: wh.CoverageRadiusKm,
			CellCount:        len(wh.H3Indexes),
			RetailersCovered: whCoveredCounts[wh.WarehouseId],
			QueueDepth:       depth,
			MaxCapacity:      maxCap,
			LoadPercent:      loadPct,
			LoadStatus:       cache.LoadStatus(loadPct),
		}
	}

	if deadZones == nil {
		deadZones = []DeadZoneRetailer{}
	}
	if overlaps == nil {
		overlaps = []OverlapResult{}
	}

	return &GeoReport{
		TotalRetailers:   len(retailers),
		CoveredRetailers: len(retailers) - len(deadZones),
		DeadZones:        deadZones,
		Overlaps:         overlaps,
		Warehouses:       summaries,
	}, nil
}

// ─── Data Fetchers ──────────────────────────────────────────────────────────

type warehouseForReport struct {
	WarehouseId      string
	Name             string
	Lat              float64
	Lng              float64
	CoverageRadiusKm float64
	H3Indexes        []string
	MaxCapacity      int64
}

func fetchWarehousesForReport(ctx context.Context, client *spanner.Client, supplierID string) ([]warehouseForReport, error) {
	stmt := spanner.Statement{
		SQL: `SELECT WarehouseId, Name, IFNULL(Lat, 0), IFNULL(Lng, 0), CoverageRadiusKm, H3Indexes, COALESCE(MaxCapacityThreshold, 100)
		      FROM Warehouses WHERE SupplierId = @sid AND IsActive = true`,
		Params: map[string]interface{}{"sid": supplierID},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var results []warehouseForReport
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var wh warehouseForReport
		var h3 []string
		if err := row.Columns(&wh.WarehouseId, &wh.Name, &wh.Lat, &wh.Lng, &wh.CoverageRadiusKm, &h3, &wh.MaxCapacity); err != nil {
			log.Printf("[GEO-REPORT] parse warehouse error: %v", err)
			continue
		}
		wh.H3Indexes = h3
		results = append(results, wh)
	}

	return results, nil
}

func fetchRetailersWithCoords(ctx context.Context, client *spanner.Client) ([]DeadZoneRetailer, error) {
	stmt := spanner.Statement{
		SQL: `SELECT RetailerId, Name, COALESCE(ShopName, ''), IFNULL(Latitude, 0), IFNULL(Longitude, 0), COALESCE(H3Index, '')
		      FROM Retailers
		      WHERE Latitude IS NOT NULL AND Longitude IS NOT NULL
		        AND Latitude != 0 AND Longitude != 0`,
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var results []DeadZoneRetailer
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var r DeadZoneRetailer
		if err := row.Columns(&r.RetailerID, &r.Name, &r.ShopName, &r.Latitude, &r.Longitude, &r.H3Index); err != nil {
			log.Printf("[GEO-REPORT] parse retailer error: %v", err)
			continue
		}
		results = append(results, r)
	}

	return results, nil
}

// containsCell checks if a cell ID exists in a slice.
func containsCell(cells []string, target string) bool {
	for _, c := range cells {
		if c == target {
			return true
		}
	}
	return false
}
