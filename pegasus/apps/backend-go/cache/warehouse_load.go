// Package cache — Warehouse Queue Depth Layer
//
// Tracks active manifest count per warehouse using Redis INCR/DECR counters.
// All operations are nil-safe: if the Redis Client is nil (degraded mode),
// they return zero values without error. Keys auto-expire after 24h to
// prevent stale counter accumulation.
package cache

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

const (
	// warehouseQueuePrefix is the Redis key prefix for warehouse queue depth counters.
	warehouseQueuePrefix = PrefixWarehouseQueue // "wh:queue:"
	// warehouseQueueTTL is the expiry for each counter — auto-resets stale counters.
	warehouseQueueTTL = TTLWarehouseQueue // 24h
)

// warehouseQueueKey returns the Redis key for a warehouse's queue depth.
func warehouseQueueKey(warehouseID string) string {
	return warehouseQueuePrefix + warehouseID
}

// IncrementQueueDepth atomically increments the active manifest count for a warehouse.
// Called after a manifest is created and assigned to this warehouse.
// Sets a 24h TTL on the key (refreshed on each INCR).
func IncrementQueueDepth(ctx context.Context, warehouseID string) {
	c := GetClient()
	if c == nil || warehouseID == "" {
		return
	}
	key := warehouseQueueKey(warehouseID)
	pipe := c.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, warehouseQueueTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		slog.Warn("warehouse load increment failed", "warehouse_id", warehouseID, "err", err)
	}
}

// DecrementQueueDepth atomically decrements the active manifest count for a warehouse.
// Called when an order transitions to COMPLETED. Floors at 0 to prevent negative drift.
func DecrementQueueDepth(ctx context.Context, warehouseID string) {
	c := GetClient()
	if c == nil || warehouseID == "" {
		return
	}
	key := warehouseQueueKey(warehouseID)
	val, err := c.Decr(ctx, key).Result()
	if err != nil {
		slog.Warn("warehouse load decrement failed", "warehouse_id", warehouseID, "err", err)
		return
	}
	// Floor at zero — DECR can go negative if counters drift
	if val < 0 {
		c.Set(ctx, key, 0, warehouseQueueTTL)
	}
}

// GetQueueDepth returns the current active manifest count for a single warehouse.
// Returns 0 if Redis is unavailable or the key does not exist.
func GetQueueDepth(ctx context.Context, warehouseID string) int64 {
	c := GetClient()
	if c == nil || warehouseID == "" {
		return 0
	}
	val, err := c.Get(ctx, warehouseQueueKey(warehouseID)).Int64()
	if err != nil {
		return 0 // key missing or Redis down — degraded mode
	}
	return val
}

// GetWarehouseLoad returns the load factor (0.0–1.0) for a warehouse.
// maxCapacity is the warehouse's MaxCapacityThreshold from Spanner.
// Returns 0.0 if Redis is unavailable, maxCapacity is zero, or the key is missing.
func GetWarehouseLoad(ctx context.Context, warehouseID string, maxCapacity int64) float64 {
	if maxCapacity <= 0 {
		return 0.0
	}
	depth := GetQueueDepth(ctx, warehouseID)
	load := float64(depth) / float64(maxCapacity)
	if load > 1.0 {
		return 1.0
	}
	return load
}

// GetAllWarehouseLoads returns queue depth for multiple warehouses in a single batch.
// Uses Redis MGET for efficiency. Returns a map of warehouseID → queueDepth.
// Missing or errored warehouses default to 0.
func GetAllWarehouseLoads(ctx context.Context, warehouseIDs []string) map[string]int64 {
	result := make(map[string]int64, len(warehouseIDs))
	c := GetClient()
	if c == nil || len(warehouseIDs) == 0 {
		return result
	}

	keys := make([]string, len(warehouseIDs))
	for i, wid := range warehouseIDs {
		keys[i] = warehouseQueueKey(wid)
	}

	vals, err := c.MGet(ctx, keys...).Result()
	if err != nil {
		slog.Warn("warehouse load batch read failed", "err", err)
		return result
	}

	for i, v := range vals {
		if v == nil {
			continue
		}
		str, ok := v.(string)
		if !ok {
			continue
		}
		var n int64
		if _, err := fmt.Sscanf(str, "%d", &n); err == nil && n > 0 {
			result[warehouseIDs[i]] = n
		}
	}

	return result
}

// BulkIncrementQueueDepth increments queue depth for multiple warehouses in a single pipeline.
// Used by the dispatcher when creating multiple manifests across different warehouses.
func BulkIncrementQueueDepth(ctx context.Context, warehouseCounts map[string]int) {
	c := GetClient()
	if c == nil || len(warehouseCounts) == 0 {
		return
	}
	pipe := c.Pipeline()
	for wid, count := range warehouseCounts {
		key := warehouseQueueKey(wid)
		pipe.IncrBy(ctx, key, int64(count))
		pipe.Expire(ctx, key, warehouseQueueTTL)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		slog.Warn("warehouse load bulk increment failed", "err", err)
	}
}

// ResetQueueDepth forces a warehouse's counter to a specific value.
// Used for reconciliation or manual operator correction.
func ResetQueueDepth(ctx context.Context, warehouseID string, value int64) error {
	c := GetClient()
	if c == nil {
		return nil
	}
	return c.Set(ctx, warehouseQueueKey(warehouseID), value, warehouseQueueTTL).Err()
}

// LoadStatus classifies a load factor into a traffic-light status.
func LoadStatus(loadPercent float64) string {
	switch {
	case loadPercent >= 0.9:
		return "RED"
	case loadPercent >= 0.5:
		return "YELLOW"
	default:
		return "GREEN"
	}
}

// ensure redis.Nil is importable for nil-checks elsewhere
var _ = redis.Nil
