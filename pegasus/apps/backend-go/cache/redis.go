// Package cache provides a singleton Redis client backed by Google Cloud Memorystore.
// It is the Resolution Ledger for the Dead Letter Queue subsystem:
// every successfully replayed Kafka offset is recorded in a Redis SET so that
// the DLQ portal can filter it out permanently — preventing admin double-fires.
package cache

import (
	"context"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// clientMu guards all mutations of Client so the health monitor goroutine and
// request goroutines never race on the pointer.
var clientMu sync.RWMutex

// Client is the process-wide Redis connection pool.
// New code should use GetClient() for thread-safe access. Direct assignment is
// only safe in test setup where the health monitor is not running.
var Client redis.UniversalClient

// GetClient returns the live Redis client or nil in degraded mode. Thread-safe.
func GetClient() redis.UniversalClient {
	clientMu.RLock()
	c := Client
	clientMu.RUnlock()
	return c
}

// setClient stores c as the live client. Resets the Pub/Sub relay on nil so
// it is re-created after a recovery.
func setClient(c redis.UniversalClient) {
	clientMu.Lock()
	Client = c
	clientMu.Unlock()
	if c == nil {
		resetRelay()
	}
}

// DLQResolvedKey is the Redis SET key that stores all offset integers
// which have been successfully replayed to the main Kafka topic.
const DLQResolvedKey = KeyDLQResolved

// Init boots the Redis client from addr. Comma-separated addrs → cluster mode;
// a single addr → single-node. If addr is empty or the initial ping fails,
// Init logs a warning and leaves Client as nil — callers degrade gracefully.
func Init(addr string) {
	if addr == "" {
		log.Println("[REDIS] REDIS_ADDRESS not set — Resolution Ledger running in degraded mode (no DLQ de-duplication).")
		return
	}

	redisAddr = addr // Store for health monitor reconnection

	c := newUniversalClient(addr)
	if c == nil {
		return
	}
	setClient(c)
	redisHealthy.Store(true)
	log.Println("[REDIS] Memorystore connection established. Resolution Ledger: ACTIVE.")
}

// newUniversalClient dials a single-node or cluster client and returns it after
// a successful ping. Returns nil on failure.
func newUniversalClient(addr string) redis.UniversalClient {
	var c redis.UniversalClient
	if strings.Contains(addr, ",") {
		// Cluster mode — comma-separated node addresses.
		c = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        strings.Split(addr, ","),
			DialTimeout:  3 * time.Second,
			ReadTimeout:  2 * time.Second,
			WriteTimeout: 2 * time.Second,
		})
	} else {
		c = redis.NewClient(&redis.Options{
			Addr:         addr,
			PoolSize:     4000,
			MinIdleConns: 100,
			DialTimeout:  3 * time.Second,
			ReadTimeout:  2 * time.Second,
			WriteTimeout: 2 * time.Second,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := c.Ping(ctx).Err(); err != nil {
		log.Printf("[REDIS] Ping failed (%v) — Resolution Ledger degraded. DLQ replay will work but de-duplication is disabled.", err)
		_ = c.Close()
		return nil
	}
	return c
}

// MarkResolved records a Kafka DLQ offset as successfully replayed.
// It is idempotent — calling it twice for the same offset is safe.
func MarkResolved(ctx context.Context, offset int64) error {
	c := GetClient()
	if c == nil {
		return nil // degraded mode — skip silently
	}
	pipe := c.Pipeline()
	pipe.SAdd(ctx, DLQResolvedKey, offset)
	pipe.Expire(ctx, DLQResolvedKey, TTLDLQResolved) // Auto-evict old offsets after 7 days
	_, err := pipe.Exec(ctx)
	return err
}

// IsResolved returns true if the given offset has already been replayed.
func IsResolved(ctx context.Context, offset int64) bool {
	c := GetClient()
	if c == nil {
		return false
	}
	result, err := c.SIsMember(ctx, DLQResolvedKey, offset).Result()
	if err != nil {
		return false
	}
	return result
}

// ResolvedOffsets returns the current set of all resolved offset strings.
// Used by ListDLQMessages to batch-filter in a single round-trip.
func ResolvedOffsets(ctx context.Context) map[string]bool {
	resolved := make(map[string]bool)
	c := GetClient()
	if c == nil {
		return resolved
	}
	members, err := c.SMembers(ctx, DLQResolvedKey).Result()
	if err != nil {
		log.Printf("[REDIS] Failed to fetch resolved offsets: %v — showing full DLQ", err)
		return resolved
	}
	for _, m := range members {
		resolved[m] = true
	}
	return resolved
}

// PurgeResolutionLedger wipes the resolved offsets SET.
// ADMIN-ONLY — exposed for operational resets when the DLQ topic itself is reset.
func PurgeResolutionLedger(ctx context.Context) error {
	c := GetClient()
	if c == nil {
		return nil
	}
	return c.Del(ctx, DLQResolvedKey).Err()
}

// ═══════════════════════════════════════════════════════════════════════════════
// Redis Health Monitor (I-1)
// ═══════════════════════════════════════════════════════════════════════════════

// redisAddr is stored at Init time for reconnection attempts.
var redisAddr string

// RedisHealthy reports whether the Redis client is currently connected and responsive.
var redisHealthy atomic.Bool

// IsRedisHealthy returns the current health state (thread-safe).
func IsRedisHealthy() bool {
	return redisHealthy.Load()
}

// StartHealthMonitor pings Redis every 30 seconds. After 3 consecutive failures,
// it calls setClient(nil) for graceful degradation. On recovery, it reconnects.
func StartHealthMonitor() {
	if redisAddr == "" {
		return // no Redis configured — nothing to monitor
	}

	go func() {
		consecutiveFailures := 0
		for {
			time.Sleep(30 * time.Second)

			if c := GetClient(); c != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				err := c.Ping(ctx).Err()
				cancel()

				if err != nil {
					consecutiveFailures++
					log.Printf("[REDIS_HEALTH] Ping failed (%d/3): %v", consecutiveFailures, err)
					if consecutiveFailures >= 3 {
						log.Println("[REDIS_HEALTH] 3 consecutive failures — setting Client=nil for graceful degradation")
						setClient(nil)
						redisHealthy.Store(false)
					}
				} else {
					if consecutiveFailures > 0 {
						log.Printf("[REDIS_HEALTH] Recovered after %d failures", consecutiveFailures)
					}
					consecutiveFailures = 0
					redisHealthy.Store(true)
				}
			} else {
				// Attempt reconnection via newUniversalClient (supports cluster).
				if c := newUniversalClient(redisAddr); c != nil {
					setClient(c)
					redisHealthy.Store(true)
					consecutiveFailures = 0
					log.Println("[REDIS_HEALTH] Reconnected successfully")
				}
			}
		}
	}()
}

// Close gracefully shuts down the Redis connection pool.
// Call during application shutdown to prevent connection leaks.
func Close() {
	if c := GetClient(); c != nil {
		if err := c.Close(); err != nil {
			log.Printf("[REDIS] Close error: %v", err)
		} else {
			log.Println("[REDIS] Connection pool closed.")
		}
		setClient(nil)
	}
}
