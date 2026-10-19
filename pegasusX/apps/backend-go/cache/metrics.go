package cache

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
)

var (
	cacheHitTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "void_redis_cache_hit_total",
		Help: "Redis-backed cache hits by key prefix (geo, warehouse, etc.).",
	}, []string{"prefix"})

	cacheMissTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "void_redis_cache_miss_total",
		Help: "Redis-backed cache misses by key prefix.",
	}, []string{"prefix"})

	commandDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "void_redis_command_duration_seconds",
		Help:    "Latency of Redis operations by command name and status (redis-observability).",
		Buckets: []float64{0.0002, 0.0005, 0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	}, []string{"command", "status"})

	poolConns = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "void_redis_pool_connections",
		Help: "Active Redis connection pool size by state (total, idle, stale).",
	}, []string{"state"})

	poolHitsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "void_redis_pool_hits_total",
		Help: "Total connections retrieved from pool without creating new ones.",
	})

	poolMissesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "void_redis_pool_misses_total",
		Help: "Total new connections initialized due to pool exhaustion.",
	})

	poolTimeoutsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "void_redis_pool_timeouts_total",
		Help: "Total requests that timed out waiting for a connection from the pool.",
	})

	circuitStateGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "void_redis_circuit_state",
		Help: "Redis circuit breaker operational state (1 for active state, 0 otherwise).",
	}, []string{"state"})

	poolMu       sync.Mutex
	lastHits     uint32
	lastMisses   uint32
	lastTimeouts uint32
)

// RecordHit increments the hit counter for the key prefix before the first colon.
func RecordHit(key string) {
	cacheHitTotal.WithLabelValues(keyPrefix(key)).Inc()
}

// RecordMiss increments the miss counter for the key prefix before the first colon.
func RecordMiss(key string) {
	cacheMissTotal.WithLabelValues(keyPrefix(key)).Inc()
}

// RecordCommandDuration records command execution latency and completion status.
func RecordCommandDuration(command string, duration time.Duration, err error) {
	status := "ok"
	if err == redis.Nil {
		status = "miss"
	} else if err != nil {
		status = "error"
	}
	commandDuration.WithLabelValues(strings.ToLower(command), status).Observe(duration.Seconds())
}

// RecordPoolStats records Redis connection pool statistics into Prometheus gauges/counters.
func RecordPoolStats(stats *redis.PoolStats) {
	if stats == nil {
		return
	}
	poolConns.WithLabelValues("total").Set(float64(stats.TotalConns))
	poolConns.WithLabelValues("idle").Set(float64(stats.IdleConns))
	poolConns.WithLabelValues("stale").Set(float64(stats.StaleConns))

	poolMu.Lock()
	defer poolMu.Unlock()
	if stats.Hits >= lastHits {
		poolHitsTotal.Add(float64(stats.Hits - lastHits))
		lastHits = stats.Hits
	} else {
		lastHits = stats.Hits
	}

	if stats.Misses >= lastMisses {
		poolMissesTotal.Add(float64(stats.Misses - lastMisses))
		lastMisses = stats.Misses
	} else {
		lastMisses = stats.Misses
	}

	if stats.Timeouts >= lastTimeouts {
		poolTimeoutsTotal.Add(float64(stats.Timeouts - lastTimeouts))
		lastTimeouts = stats.Timeouts
	} else {
		lastTimeouts = stats.Timeouts
	}
}

// RecordCircuitState records the active circuit breaker state.
func RecordCircuitState(state string) {
	states := []string{"closed", "open", "half_open"}
	for _, s := range states {
		if strings.EqualFold(s, state) {
			circuitStateGauge.WithLabelValues(s).Set(1)
		} else {
			circuitStateGauge.WithLabelValues(s).Set(0)
		}
	}
}

// PoolStatsProvider is an interface for types that expose Redis PoolStats.
type PoolStatsProvider interface {
	PoolStats() *redis.PoolStats
}

// StartPoolMetricsCollector launches a background goroutine that periodically
// polls pool statistics and updates Prometheus gauges.
func StartPoolMetricsCollector(ctx context.Context, provider PoolStatsProvider, interval time.Duration) {
	if provider == nil || interval <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if stats := provider.PoolStats(); stats != nil {
					RecordPoolStats(stats)
				}
			}
		}
	}()
}

func keyPrefix(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return "unknown"
	}
	if i := strings.IndexByte(key, ':'); i > 0 {
		return key[:i]
	}
	return key
}

