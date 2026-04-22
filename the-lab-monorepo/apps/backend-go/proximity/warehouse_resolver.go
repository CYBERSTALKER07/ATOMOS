package proximity

import (
	"context"
	"fmt"
	"log"

	"backend-go/cache"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ─── Warehouse Resolver ────────────────────────────────────────────────────────
//
// Resolves which warehouse(s) should fulfill an order for a given retailer.
// Resolution strategy (fastest → slowest):
//
//   1. Grid cell lookup — O(1) Redis SET membership from pre-computed coverage
//   2. Redis GEOSEARCH — nearest warehouse within max radius
//   3. Spanner fallback — direct query when Redis is unavailable
//
// The resolver returns the best warehouse for a supplier+retailer pair.
// For multi-supplier carts, the caller invokes this per-supplier.

// WarehouseMatch represents a resolved warehouse assignment.
type WarehouseMatch struct {
	WarehouseId string
	SupplierId  string
	Name        string
	DistanceKm  float64
	Lat         float64
	Lng         float64
}

// ResolveWarehouse finds the best warehouse under a supplier to fulfill an order
// for a retailer at the given coordinates. Returns nil if no warehouse covers the area.
func ResolveWarehouse(ctx context.Context, spannerClient *spanner.Client, supplierID string, retailerLat, retailerLng float64) (*WarehouseMatch, error) {
	// Path 1: Grid cell lookup (O(1) via Redis)
	match, err := resolveViaGridCell(ctx, supplierID, retailerLat, retailerLng)
	if err != nil {
		log.Printf("[RESOLVER] Grid cell lookup failed: %v — falling back to GEOSEARCH", err)
	}
	if match != nil {
		return match, nil
	}

	// Path 2: Redis GEOSEARCH (nearest within 200km max)
	match, err = resolveViaGeoSearch(ctx, supplierID, retailerLat, retailerLng)
	if err != nil {
		log.Printf("[RESOLVER] GEOSEARCH failed: %v — falling back to Spanner", err)
	}
	if match != nil {
		return match, nil
	}

	// Path 3: Spanner direct query (slowest but always available)
	return resolveViaSpanner(ctx, spannerClient, supplierID, retailerLat, retailerLng)
}

// resolveViaGridCell uses the pre-computed cell→warehouse index.
func resolveViaGridCell(ctx context.Context, supplierID string, lat, lng float64) (*WarehouseMatch, error) {
	cellID := LookupCell(lat, lng)
	warehouseIDs, err := cache.FindWarehousesByCell(ctx, cellID)
	if err != nil {
		return nil, err
	}

	if len(warehouseIDs) == 0 {
		return nil, nil // no coverage at this cell
	}

	// Filter to warehouses belonging to this supplier
	var bestMatch *WarehouseMatch
	var bestDist float64

	for _, whID := range warehouseIDs {
		detail, err := cache.GetWarehouseDetail(ctx, whID)
		if err != nil || detail == nil {
			continue
		}
		if detail.SupplierId != supplierID {
			continue
		}

		dist := HaversineKm(lat, lng, detail.Lat, detail.Lng)
		if bestMatch == nil || dist < bestDist {
			bestMatch = &WarehouseMatch{
				WarehouseId: detail.WarehouseId,
				SupplierId:  detail.SupplierId,
				Name:        detail.Name,
				DistanceKm:  dist,
				Lat:         detail.Lat,
				Lng:         detail.Lng,
			}
			bestDist = dist
		}
	}

	return bestMatch, nil
}

// resolveViaGeoSearch uses Redis GEOSEARCH to find nearest warehouses.
func resolveViaGeoSearch(ctx context.Context, supplierID string, lat, lng float64) (*WarehouseMatch, error) {
	// Search within 200km — generous for Uzbekistan geography
	results, err := cache.FindNearestWarehouses(ctx, lat, lng, 200.0, 20)
	if err != nil {
		return nil, err
	}

	// Filter to this supplier's warehouses
	for _, r := range results {
		detail, err := cache.GetWarehouseDetail(ctx, r.WarehouseId)
		if err != nil || detail == nil {
			continue
		}
		if detail.SupplierId != supplierID {
			continue
		}
		// Check the warehouse actually covers this distance
		if r.DistanceKm > detail.RadiusKm {
			continue
		}
		return &WarehouseMatch{
			WarehouseId: r.WarehouseId,
			SupplierId:  detail.SupplierId,
			Name:        detail.Name,
			DistanceKm:  r.DistanceKm,
			Lat:         r.Lat,
			Lng:         r.Lng,
		}, nil
	}

	return nil, nil
}

// resolveViaSpanner queries Spanner directly as ultimate fallback.
// Computes Haversine distance in application layer (Spanner doesn't have native geo functions).
func resolveViaSpanner(ctx context.Context, client *spanner.Client, supplierID string, retailerLat, retailerLng float64) (*WarehouseMatch, error) {
	if client == nil {
		return nil, fmt.Errorf("spanner client is nil")
	}

	stmt := spanner.Statement{
		SQL: `SELECT WarehouseId, Name, Lat, Lng, CoverageRadiusKm
		      FROM Warehouses
		      WHERE SupplierId = @supplierId AND IsActive = true
		      ORDER BY IsDefault DESC`,
		Params: map[string]interface{}{
			"supplierId": supplierID,
		},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var bestMatch *WarehouseMatch
	var bestDist float64

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("spanner warehouse query: %w", err)
		}

		var warehouseID, name string
		var lat, lng spanner.NullFloat64
		var radiusKm float64

		if err := row.Columns(&warehouseID, &name, &lat, &lng, &radiusKm); err != nil {
			continue
		}

		if !lat.Valid || !lng.Valid {
			continue
		}

		dist := HaversineKm(retailerLat, retailerLng, lat.Float64, lng.Float64)
		if dist > radiusKm {
			continue // outside coverage
		}

		if bestMatch == nil || dist < bestDist {
			bestMatch = &WarehouseMatch{
				WarehouseId: warehouseID,
				SupplierId:  supplierID,
				Name:        name,
				DistanceKm:  dist,
				Lat:         lat.Float64,
				Lng:         lng.Float64,
			}
			bestDist = dist
		}
	}

	return bestMatch, nil
}

// ResolveWarehouseForCart resolves the warehouse for each supplier in a multi-supplier cart.
// Returns a map of supplierID → WarehouseMatch.
func ResolveWarehouseForCart(ctx context.Context, spannerClient *spanner.Client, supplierIDs []string, retailerLat, retailerLng float64) (map[string]*WarehouseMatch, error) {
	results := make(map[string]*WarehouseMatch, len(supplierIDs))

	for _, sid := range supplierIDs {
		match, err := ResolveWarehouse(ctx, spannerClient, sid, retailerLat, retailerLng)
		if err != nil {
			log.Printf("[RESOLVER] Failed to resolve warehouse for supplier %s: %v", sid, err)
			continue
		}
		if match != nil {
			results[sid] = match
		}
	}

	return results, nil
}
