package cache

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

// ─── Warehouse Geo Cache ──────────────────────────────────────────────────────
//
// Maintains a Redis GEO sorted set of warehouse locations and a SET-per-cell
// index mapping grid cells to warehouse IDs. Supports:
//
//   1. GEOSEARCH — find nearest warehouse(s) to a retailer by radius
//   2. Grid cell lookup — O(1) pre-computed cell→warehouse mapping
//
// Both paths are nil-safe: if Redis is nil, callers fall back to Spanner.

const (
	WarehouseGeoKey       = KeyGeoWarehouses      // Redis GEO sorted set: member = "wh:<id>", lat/lng
	WarehouseCellPrefix   = PrefixWarehouseCell   // SET key per grid cell: "whcell:g7:41.29:69.24" → {warehouseId, ...}
	WarehouseDetailPrefix = PrefixWarehouseDetail // HASH key: "whdetail:<warehouseId>" → {supplierId, name, lat, lng, radiusKm}
	WarehouseGeoTTL       = TTLWarehouseGeo       // Geo entries refreshed daily by cron or on warehouse update
)

// WarehouseGeoEntry represents a warehouse's geo data stored in Redis.
type WarehouseGeoEntry struct {
	WarehouseId string
	SupplierId  string
	Name        string
	Lat         float64
	Lng         float64
	RadiusKm    float64
	H3Cells     []string
}

// IndexWarehouse upserts a warehouse into the Redis geo cache.
// Called on warehouse CREATE/UPDATE and by the nightly refresh cron.
func IndexWarehouse(ctx context.Context, entry WarehouseGeoEntry) error {
	c := GetClient()
	if c == nil {
		return nil
	}

	pipe := c.Pipeline()

	// 1. GEOADD warehouse position
	pipe.GeoAdd(ctx, WarehouseGeoKey, &redis.GeoLocation{
		Name:      warehouseGeoMember(entry.WarehouseId),
		Latitude:  entry.Lat,
		Longitude: entry.Lng,
	})

	// 2. Store warehouse detail hash
	detailKey := WarehouseDetailPrefix + entry.WarehouseId
	pipe.HSet(ctx, detailKey, map[string]interface{}{
		"supplierId": entry.SupplierId,
		"name":       entry.Name,
		"lat":        entry.Lat,
		"lng":        entry.Lng,
		"radiusKm":   entry.RadiusKm,
	})
	pipe.Expire(ctx, detailKey, WarehouseGeoTTL)

	// 3. Map each coverage cell to this warehouse
	for _, cell := range entry.H3Cells {
		cellKey := WarehouseCellPrefix + cell
		pipe.SAdd(ctx, cellKey, entry.WarehouseId)
		pipe.Expire(ctx, cellKey, WarehouseGeoTTL)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		slog.Warn("warehouse geo index failed", "warehouse_id", entry.WarehouseId, "err", err)
		return err
	}

	return nil
}

// RemoveWarehouse removes a warehouse from the Redis geo cache.
// Called on warehouse deactivation or deletion.
func RemoveWarehouse(ctx context.Context, entry WarehouseGeoEntry) error {
	c := GetClient()
	if c == nil {
		return nil
	}

	pipe := c.Pipeline()

	// Remove from GEO set
	pipe.ZRem(ctx, WarehouseGeoKey, warehouseGeoMember(entry.WarehouseId))

	// Remove detail hash
	pipe.Del(ctx, WarehouseDetailPrefix+entry.WarehouseId)

	// Remove from cell indexes
	for _, cell := range entry.H3Cells {
		cellKey := WarehouseCellPrefix + cell
		pipe.SRem(ctx, cellKey, entry.WarehouseId)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// FindWarehousesByCell returns warehouse IDs that cover a specific grid cell.
// This is the O(1) fast path used for retailer→warehouse resolution.
func FindWarehousesByCell(ctx context.Context, cellID string) ([]string, error) {
	c := GetClient()
	if c == nil {
		return nil, nil
	}

	cellKey := WarehouseCellPrefix + cellID
	ids, err := c.SMembers(ctx, cellKey).Result()
	if err != nil {
		return nil, fmt.Errorf("warehouse cell lookup %s: %w", cellID, err)
	}
	return ids, nil
}

// FindNearestWarehouses finds warehouses within radiusKm of the given point.
// Fallback path when cell lookup returns no results (rural areas).
func FindNearestWarehouses(ctx context.Context, lat, lng, radiusKm float64, count int) ([]WarehouseGeoResult, error) {
	c := GetClient()
	if c == nil {
		return nil, nil
	}

	results, err := c.GeoSearchLocation(ctx, WarehouseGeoKey, &redis.GeoSearchLocationQuery{
		GeoSearchQuery: redis.GeoSearchQuery{
			Longitude:  lng,
			Latitude:   lat,
			Radius:     radiusKm,
			RadiusUnit: "km",
			Count:      count,
			Sort:       "ASC", // nearest first
		},
		WithCoord: true,
		WithDist:  true,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("warehouse geo search: %w", err)
	}

	var out []WarehouseGeoResult
	for _, r := range results {
		whID := strings.TrimPrefix(r.Name, "wh:")
		out = append(out, WarehouseGeoResult{
			WarehouseId: whID,
			DistanceKm:  r.Dist,
			Lat:         r.Latitude,
			Lng:         r.Longitude,
		})
	}
	return out, nil
}

// GetWarehouseDetail fetches cached warehouse metadata by ID.
func GetWarehouseDetail(ctx context.Context, warehouseID string) (*WarehouseGeoEntry, error) {
	c := GetClient()
	if c == nil {
		return nil, nil
	}

	detailKey := WarehouseDetailPrefix + warehouseID
	vals, err := c.HGetAll(ctx, detailKey).Result()
	if err != nil || len(vals) == 0 {
		return nil, err
	}

	lat, _ := strconv.ParseFloat(vals["lat"], 64)
	lng, _ := strconv.ParseFloat(vals["lng"], 64)
	radius, _ := strconv.ParseFloat(vals["radiusKm"], 64)

	return &WarehouseGeoEntry{
		WarehouseId: warehouseID,
		SupplierId:  vals["supplierId"],
		Name:        vals["name"],
		Lat:         lat,
		Lng:         lng,
		RadiusKm:    radius,
	}, nil
}

// WarehouseGeoResult is a single result from a geo radius search.
type WarehouseGeoResult struct {
	WarehouseId string
	DistanceKm  float64
	Lat         float64
	Lng         float64
}

func warehouseGeoMember(id string) string {
	return WarehouseGeoMember(id)
}
