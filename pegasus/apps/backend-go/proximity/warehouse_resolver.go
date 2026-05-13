package proximity

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"

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

type warehouseResolverCandidate struct {
	match      *WarehouseMatch
	h3Distance int
}

type roundRobinSequenceProvider func(ctx context.Context, key string) (int64, error)

// ResolveWarehouse finds the best warehouse under a supplier to fulfill an order
// for a retailer at the given coordinates. Returns nil if no warehouse covers the area.
func ResolveWarehouse(ctx context.Context, spannerClient *spanner.Client, supplierID string, retailerLat, retailerLng float64) (*WarehouseMatch, error) {
	return ResolveWarehouseWithRouter(ctx, spannerClient, nil, supplierID, retailerLat, retailerLng)
}

// ResolveWarehouseWithRouter mirrors ResolveWarehouse but allows H3-aware
// read-client routing when a read router is provided.
func ResolveWarehouseWithRouter(ctx context.Context, spannerClient *spanner.Client, readRouter ReadRouter, supplierID string, retailerLat, retailerLng float64) (*WarehouseMatch, error) {
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
	readClient := readClientForRetailer(spannerClient, readRouter, retailerLat, retailerLng)
	return resolveViaSpanner(ctx, readClient, supplierID, retailerLat, retailerLng)
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

	// Filter to warehouses belonging to this supplier.
	// For equal H3 ring distances, selection is round-robin so one warehouse
	// isn't repeatedly preferred by iteration order.
	candidates := make([]warehouseResolverCandidate, 0, len(warehouseIDs))

	for _, whID := range warehouseIDs {
		detail, err := cache.GetWarehouseDetail(ctx, whID)
		if err != nil || detail == nil {
			continue
		}
		if detail.SupplierId != supplierID {
			continue
		}

		dist := HaversineKm(lat, lng, detail.Lat, detail.Lng)
		candidates = append(candidates, warehouseResolverCandidate{
			match: &WarehouseMatch{
				WarehouseId: detail.WarehouseId,
				SupplierId:  detail.SupplierId,
				Name:        detail.Name,
				DistanceKm:  dist,
				Lat:         detail.Lat,
				Lng:         detail.Lng,
			},
			h3Distance: warehouseH3Distance(cellID, detail.Lat, detail.Lng),
		})
	}

	return resolveWarehouseByH3Distance(ctx, supplierID, cellID, candidates, nextWarehouseTieSequence), nil
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

	retailerCell := LookupCell(retailerLat, retailerLng)

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

	candidates := make([]warehouseResolverCandidate, 0, 8)

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

		candidates = append(candidates, warehouseResolverCandidate{
			match: &WarehouseMatch{
				WarehouseId: warehouseID,
				SupplierId:  supplierID,
				Name:        name,
				DistanceKm:  dist,
				Lat:         lat.Float64,
				Lng:         lng.Float64,
			},
			h3Distance: warehouseH3Distance(retailerCell, lat.Float64, lng.Float64),
		})
	}

	return resolveWarehouseByH3Distance(ctx, supplierID, retailerCell, candidates, nextWarehouseTieSequence), nil
}

func resolveWarehouseByH3Distance(ctx context.Context, supplierID, retailerCell string, candidates []warehouseResolverCandidate, nextSequence roundRobinSequenceProvider) *WarehouseMatch {
	if len(candidates) == 0 {
		return nil
	}

	minDistance := math.MaxInt32
	tied := make([]warehouseResolverCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.h3Distance < minDistance {
			minDistance = candidate.h3Distance
			tied = tied[:0]
			tied = append(tied, candidate)
			continue
		}
		if candidate.h3Distance == minDistance {
			tied = append(tied, candidate)
		}
	}

	if len(tied) == 1 {
		return tied[0].match
	}

	return roundRobinWarehouseTie(ctx, supplierID, retailerCell, tied, nextSequence)
}

func roundRobinWarehouseTie(ctx context.Context, supplierID, retailerCell string, tied []warehouseResolverCandidate, nextSequence roundRobinSequenceProvider) *WarehouseMatch {
	if len(tied) == 0 {
		return nil
	}

	sort.Slice(tied, func(i, j int) bool {
		return tied[i].match.WarehouseId < tied[j].match.WarehouseId
	})

	if nextSequence == nil {
		return tied[0].match
	}

	key := cache.PrefixWarehouseTieRR + supplierID + ":" + retailerCell
	sequence, err := nextSequence(ctx, key)
	if err != nil {
		log.Printf("[RESOLVER] round-robin tie-break failed for %s: %v", key, err)
		return tied[0].match
	}
	if sequence <= 0 {
		return tied[0].match
	}
	index := int((sequence - 1) % int64(len(tied)))
	return tied[index].match
}

func nextWarehouseTieSequence(ctx context.Context, key string) (int64, error) {
	client := cache.GetClient()
	if client == nil {
		return 1, nil
	}
	sequence, err := client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	_ = client.Expire(ctx, key, cache.TTLWarehouseTieRR).Err()
	return sequence, nil
}

func warehouseH3Distance(retailerCell string, warehouseLat, warehouseLng float64) int {
	warehouseCell := LookupCell(warehouseLat, warehouseLng)
	distance := H3GridDistance(retailerCell, warehouseCell)
	if distance < 0 {
		return math.MaxInt32
	}
	return distance
}

// ResolveWarehouseForCart resolves the warehouse for each supplier in a multi-supplier cart.
// Returns a map of supplierID → WarehouseMatch.
func ResolveWarehouseForCart(ctx context.Context, spannerClient *spanner.Client, supplierIDs []string, retailerLat, retailerLng float64) (map[string]*WarehouseMatch, error) {
	return ResolveWarehouseForCartWithRouter(ctx, spannerClient, nil, supplierIDs, retailerLat, retailerLng)
}

// ResolveWarehouseForCartWithRouter mirrors ResolveWarehouseForCart with
// optional H3-routed Spanner reads.
func ResolveWarehouseForCartWithRouter(ctx context.Context, spannerClient *spanner.Client, readRouter ReadRouter, supplierIDs []string, retailerLat, retailerLng float64) (map[string]*WarehouseMatch, error) {
	results := make(map[string]*WarehouseMatch, len(supplierIDs))

	for _, sid := range supplierIDs {
		match, err := ResolveWarehouseWithRouter(ctx, spannerClient, readRouter, sid, retailerLat, retailerLng)
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
