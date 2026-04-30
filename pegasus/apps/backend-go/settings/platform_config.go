package settings

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
)

// PlatformConfig provides cached reads from the SystemConfig Spanner table.
// Values are refreshed periodically (every 5 minutes) in the background.
type PlatformConfig struct {
	client *spanner.Client
	mu     sync.RWMutex
	cache  map[string]string
}

// NewPlatformConfig creates a new PlatformConfig and loads values from Spanner.
func NewPlatformConfig(sc *spanner.Client) *PlatformConfig {
	pc := &PlatformConfig{
		client: sc,
		cache:  make(map[string]string),
	}
	pc.refresh()
	go pc.backgroundRefresh()
	return pc
}

// PlatformFeePercent returns the current platform fee percentage (0–100).
// Default: 0% (zero-fee era — Pegasus takes no commission).
func (pc *PlatformConfig) PlatformFeePercent() int64 {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	if v, ok := pc.cache["platform_fee_percent"]; ok {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 0 && n <= 100 {
			return n
		}
	}
	return 0 // zero-fee era
}

// PlatformFeeBasisPoints returns the current platform fee in basis points (0–10000).
// This is the canonical value passed to ComputeSplitRecipients.
func (pc *PlatformConfig) PlatformFeeBasisPoints() int64 {
	return pc.PlatformFeePercent() * 100
}

// DispatchOptimizerTimeoutMs returns the per-call ceiling (milliseconds) the
// dispatch orchestrator passes to the optimizer client. Default: 2500ms,
// matching optimizerclient.DefaultTimeout. Operator override key:
// "dispatch_optimizer_timeout_ms".
func (pc *PlatformConfig) DispatchOptimizerTimeoutMs() int64 {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	if v, ok := pc.cache["dispatch_optimizer_timeout_ms"]; ok {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 100 && n <= 10000 {
			return n
		}
	}
	return 2500
}

// DispatchOptimizerCapacityBufferPct returns the truck-utilisation safety
// buffer (percent). 5.0 means manifests must stay ≤ 95% of MaxVU. Default
// mirrors dispatch.TetrisBuffer (5%). Operator override key:
// "dispatch_optimizer_capacity_buffer_pct".
func (pc *PlatformConfig) DispatchOptimizerCapacityBufferPct() float64 {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	if v, ok := pc.cache["dispatch_optimizer_capacity_buffer_pct"]; ok {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n >= 0 && n <= 50 {
			return n
		}
	}
	return 5.0
}

// DispatchOptimizerAvgSpeedKmh returns the cruising-speed assumption (km/h)
// used by the solver when a vehicle row has no per-truck speed. Default 30
// km/h matches optimizerclient defaultAvgSpeedKmph. Operator override key:
// "dispatch_optimizer_avg_speed_kmh".
func (pc *PlatformConfig) DispatchOptimizerAvgSpeedKmh() float64 {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	if v, ok := pc.cache["dispatch_optimizer_avg_speed_kmh"]; ok {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n >= 5 && n <= 200 {
			return n
		}
	}
	return 30.0
}

// Get returns a config value by key, or empty string if not found.
func (pc *PlatformConfig) Get(key string) string {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.cache[key]
}

// Set upserts a config value in Spanner and updates the local cache.
func (pc *PlatformConfig) Set(ctx context.Context, key, value string) error {
	_, err := pc.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		m := spanner.InsertOrUpdate("SystemConfig",
			[]string{"ConfigKey", "ConfigValue", "UpdatedAt"},
			[]interface{}{key, value, spanner.CommitTimestamp},
		)
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	if err != nil {
		return fmt.Errorf("set config %s: %w", key, err)
	}

	pc.mu.Lock()
	pc.cache[key] = value
	pc.mu.Unlock()

	log.Printf("[PLATFORM_CONFIG] Updated %s = %s", key, value)
	return nil
}

func (pc *PlatformConfig) refresh() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stmt := spanner.Statement{SQL: `SELECT ConfigKey, ConfigValue FROM SystemConfig`}
	iter := pc.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	newCache := make(map[string]string)
	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		var key, val string
		if err := row.Columns(&key, &val); err != nil {
			continue
		}
		newCache[key] = val
	}

	pc.mu.Lock()
	pc.cache = newCache
	pc.mu.Unlock()
}

func (pc *PlatformConfig) backgroundRefresh() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		pc.refresh()
	}
}
