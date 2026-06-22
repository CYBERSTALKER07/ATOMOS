package cache

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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
)

// RecordHit increments the hit counter for the key prefix before the first colon.
func RecordHit(key string) {
	cacheHitTotal.WithLabelValues(keyPrefix(key)).Inc()
}

// RecordMiss increments the miss counter for the key prefix before the first colon.
func RecordMiss(key string) {
	cacheMissTotal.WithLabelValues(keyPrefix(key)).Inc()
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
