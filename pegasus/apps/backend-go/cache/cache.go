package cache

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// Cache is a handle over the Redis Resolution Ledger that bootstrap holds on
// behalf of the application. It intentionally delegates to the package-level
// Client/Init/Close/StartHealthMonitor functions so legacy callers that still
// reference cache.Client directly continue to work unchanged. New code should
// prefer GetClient() — the struct is the migration target when full DI happens.
type Cache struct {
	addr string
}

// New constructs a Cache pointed at addr (typically cfg.RedisAddress).
// An empty addr yields a degraded handle whose Client() returns nil —
// consistent with the package-level Init contract.
// L1 eviction is wired here so every invalidation from this process also
// purges the in-process cache tier.
func New(addr string) *Cache {
	Init(addr)
	c := &Cache{addr: addr}
	c.OnInvalidate(l1Evict) // wire L1 eviction for cross-pod invalidation signals
	return c
}

// Client returns the live Redis connection or nil in degraded mode. Thread-safe.
func (c *Cache) Client() redis.UniversalClient { return GetClient() }

// Addr returns the configured Redis address (empty in degraded mode).
func (c *Cache) Addr() string { return c.addr }

// StartHealthMonitor launches the reconnection watchdog goroutine.
func (c *Cache) StartHealthMonitor() { StartHealthMonitor() }

// StartCacheWorkers starts the pool of background cache write workers.
// Call once from bootstrap.NewApp after New() returns.
func (c *Cache) StartCacheWorkers(ctx context.Context) { StartCacheWorkers(ctx) }

// Healthy reports the current connection state (thread-safe).
func (c *Cache) Healthy() bool { return IsRedisHealthy() }

// Close gracefully tears down the connection pool.
func (c *Cache) Close() { Close() }
