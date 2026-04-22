package cache

import (
	"context"
	"log"
	"strings"
	"sync"
)

// ─── Pub/Sub Cache Invalidation (V.O.I.D. Phase VII) ──────────────────────────
//
// When a handler mutates a cached aggregate it must call Invalidate so every
// pod's view of the key is refreshed. Today Redis is the only cache tier, so
// DEL by itself is sufficient for correctness; the Pub/Sub "kill signal"
// message is also emitted so a future in-process L1 cache (added by handlers
// that need microsecond reads) can subscribe and drop its local entries.
//
// Channel: invalidationChannel below.
// Payload: comma-separated key list. Callers should keep keys compact.

const invalidationChannel = "cache:invalidate"

// invalidationHooks receive locally-originated AND peer-pod invalidations.
// Register via OnInvalidate. Handlers run on the Pub/Sub relay goroutine — be
// fast, non-blocking, and log-safe.
var (
	invMu    sync.RWMutex
	invHooks []func(keys []string)
)

// Invalidate deletes keys from Redis AND publishes a kill-signal on the
// invalidation channel. Peer pods (and any local handler registered with
// OnInvalidate) drop their L1 entries when they receive the message.
//
// Fail-open: a Redis DEL error does NOT prevent the Pub/Sub announce and does
// NOT propagate to the caller beyond a log line. An invalidation that
// partially succeeds is still safer than a mutation with no invalidation at
// all — downstream reads will eventually re-populate from the source of truth.
func (c *Cache) Invalidate(ctx context.Context, keys ...string) {
	if len(keys) == 0 {
		return
	}
	// Evict L1 first — any subsequent read that hits between L1 eviction and
	// Redis DEL goes to Redis (or falls through to Spanner); never serves stale.
	l1Evict(keys)
	if rc := GetClient(); rc != nil {
		if err := rc.Del(ctx, keys...).Err(); err != nil {
			log.Printf("[CACHE] Invalidate DEL %v failed: %v", keys, err)
		}
	}
	// Announce to peers even if the local DEL fails — their copies may still
	// be live and must be evicted.
	Publish(ctx, invalidationChannel, []byte(strings.Join(keys, ",")))
	// Fire local hooks synchronously (originating pod): keeps in-process L1
	// caches in sync immediately without waiting for the Pub/Sub echo.
	invMu.RLock()
	hooks := append([]func([]string){}, invHooks...)
	invMu.RUnlock()
	for _, h := range hooks {
		h(keys)
	}
}

// OnInvalidate registers a hook that fires on every invalidation — both ones
// originating on this pod (via Invalidate) and ones relayed from peer pods
// (via StartInvalidationSubscriber). Intended as the integration point for
// future in-process L1 caches.
func (c *Cache) OnInvalidate(h func(keys []string)) {
	if h == nil {
		return
	}
	invMu.Lock()
	invHooks = append(invHooks, h)
	invMu.Unlock()
}

// StartInvalidationSubscriber subscribes this pod to the invalidation channel
// and dispatches incoming messages to registered OnInvalidate hooks.
// Safe to call even in degraded mode (no Redis client) — it becomes a no-op.
// Typically called once from bootstrap.NewApp.
func (c *Cache) StartInvalidationSubscriber(ctx context.Context) {
	if Client == nil {
		log.Println("[CACHE] Invalidation subscriber disabled: no Redis client")
		return
	}
	Subscribe(invalidationChannel, func(channel string, payload []byte) {
		if len(payload) == 0 {
			return
		}
		keys := strings.Split(string(payload), ",")
		invMu.RLock()
		hooks := append([]func([]string){}, invHooks...)
		invMu.RUnlock()
		for _, h := range hooks {
			h(keys)
		}
	})
	log.Println("[CACHE] Invalidation subscriber online on", invalidationChannel)
}
