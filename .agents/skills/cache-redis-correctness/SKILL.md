# Cache & Redis Correctness — Cache Invalidation & Redis Safety

## Description
Prevents stale data serving, cache poisoning, race conditions in cache operations, and Redis client access bugs. Activates when writing or reviewing code that touches `cache.Get`, `cache.Set`, `cache.Delete`, `cache.Invalidate`, Redis operations, TTL configuration, or any mutation handler that should invalidate cached data.

## Trigger Keywords
cache, redis, invalidate, TTL, stale, cache.Get, cache.Set, cache.Delete, cache.Invalidate, cache.Publish, PrefixSupplier, PrefixRetailer, PrefixDriver, PrefixCatalog, PrefixAnalytics, cached, memoize

## Anti-Pattern Catalog

### 1. Mutation Without cache.Invalidate (STALE DATA — SYSTEMIC)
```go
// ❌ WRONG — Spanner updated, cache still serves old data for up to TTL
func (s *Service) UpdateFactory(ctx context.Context, req UpdateFactoryReq) error {
    _, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        return txn.BufferWrite([]*spanner.Mutation{mutation})
    })
    return err // FORGOT cache.Invalidate
}

// ✅ RIGHT — invalidate AFTER commit
func (s *Service) UpdateFactory(ctx context.Context, req UpdateFactoryReq) error {
    _, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        return txn.BufferWrite([]*spanner.Mutation{mutation})
    })
    if err != nil {
        return fmt.Errorf("update factory %s: %w", req.FactoryID, err)
    }
    // Invalidate AFTER successful commit
    s.Cache.Invalidate(ctx,
        cache.Key(cache.PrefixFactory, req.FactoryID),
        cache.Key(cache.PrefixFactoryList, req.SupplierID),
    )
    return nil
}
```
**Real finding**: ~1 `cache.Invalidate` call in the entire codebase (`order/service.go`) vs ~50+ mutation paths. Every `factory/`, `supplier/`, `warehouse/`, `auth/`, `replenishment/` mutation commits to Spanner without invalidating.

**Rule**: EVERY POST/PATCH/PUT/DELETE handler that mutates a cached aggregate MUST call `cache.Invalidate` after Spanner commit. This is the #1 systemic gap in the codebase.

### 2. Pre-Commit Invalidation (RACE CONDITION)
```go
// ❌ WRONG — invalidate before commit → if txn rolls back, cache is empty but DB unchanged
s.Cache.Invalidate(ctx, key)
_, err := s.Spanner.ReadWriteTransaction(ctx, ...)
// If txn fails, cache was invalidated for nothing.
// Worse: another request reads from Spanner, re-populates cache,
// THEN the first request retries and commits, leaving stale cache.

// ✅ RIGHT — invalidate AFTER commit succeeds
_, err := s.Spanner.ReadWriteTransaction(ctx, ...)
if err != nil { return err }
s.Cache.Invalidate(ctx, key) // only after confirmed commit
```

### 3. TTL as Sole Correctness Mechanism (DATA INCONSISTENCY)
```go
// ❌ WRONG — "the cache will expire in 5 minutes" is not correctness
s.Cache.Set(ctx, key, data, 5*time.Minute)
// No Invalidate call on mutation → users see stale data for up to 5 minutes

// ✅ RIGHT — TTL is a safety net, Invalidate is the correctness mechanism
// On read:
s.Cache.Set(ctx, key, data, cache.TTLDefault)
// On mutation:
s.Cache.Invalidate(ctx, key) // immediate consistency
```

### 4. Redis Client Global Access Without Nil Check (PANIC)
```go
// ❌ WRONG — Client can be nil (Redis down or health monitor reconnecting)
result, err := cache.Client.Get(ctx, key).Result()

// ✅ RIGHT — use Cache struct methods (handle nil internally)
result, err := s.Cache.Get(ctx, key)
if err == cache.ErrCacheMiss { /* handle miss */ }

// If you must access Client directly:
client := cache.GetClient()
if client == nil {
    // Redis unavailable — fall through to Spanner
    return nil, cache.ErrCacheUnavailable
}
```
**Real finding**: `cache/redis.go` `var Client *redis.Client` — written by health monitor (nil on failure, new client on reconnect) without mutex. Read by every request goroutine. Data race.

### 5. sync.Once for Pub/Sub Relay (PERMANENT FAILURE)
```go
// ❌ WRONG — if Redis is nil at boot, relay is permanently disabled
var globalRelayOnce sync.Once
func startRelay() {
    globalRelayOnce.Do(func() {
        if cache.Client == nil { return } // never retries
    })
}
```
**Real finding**: `cache/pubsub.go` — relay permanently disabled if Redis was down at boot.

**Rule**: For reconnectable subsystems, use mutex-guarded initialization with retry capability (see concurrency-shield skill LAW 1.5).

### 6. init() Workers That Depend on Client (STARTUP PANIC)
**Real finding**: `cache/middleware.go` `init()` spawns 8 workers that call `Client.Set(...)`. Workers panic if a cache write job is enqueued before Redis init completes.

**Rule**: Cache workers must start via explicit `Start(ctx)` method AFTER Redis client is initialized, not in `init()`.

## Cache Key Naming Convention
```
{prefix}:{scope}:{id}

Examples:
  supplier:profile:abc-123
  factory:detail:def-456
  catalog:search:hash-of-query
  driver:profile:ghi-789
  analytics:dashboard:supplier-abc
```
Prefixes are defined in `cache/keys.go`. Use the `cache.Key()` helper to construct keys consistently.

## Known Cache Prefixes and TTLs
| Prefix | TTL | Notes |
|---|---|---|
| `PrefixRateLimit` | `TTLRateLimitDefault` | Per-actor rate limiting |
| `PrefixIdempotency` | 24h (API) / 7d (webhook) | Idempotency key storage |
| `PrefixSupplierProfile` | **NO TTL CONSTANT** — use `TTLDefault` (5m) | |
| `PrefixRetailerProfile` | **NO TTL CONSTANT** | |
| `PrefixDriverProfile` | **NO TTL CONSTANT** | |
| `PrefixFactoryProfile` | **NO TTL CONSTANT** | |
| `PrefixCatalogSearch` | **NO TTL CONSTANT** | |
| `PrefixAnalytics` | **NO TTL CONSTANT** | |
| `PrefixSettings` | **NO TTL CONSTANT** | |

**Rule**: When using a prefix without a TTL constant, use `cache.TTLDefault`. If you find yourself needing a different TTL, add a named constant to `cache/keys.go` — never hardcode a magic number.

## Cross-Pod Invalidation Flow
```
Handler → cache.Invalidate(ctx, keys...)
  ├── DEL keys from local Redis
  └── PUBLISH "cache:invalidate" channel with key list
       ├── Pod A receives → DEL from local in-process cache
       ├── Pod B receives → DEL from local in-process cache
       └── Pod C receives → DEL from local in-process cache
```
This is already wired via `cache.StartInvalidationSubscriber` in `bootstrap.NewApp`. The infrastructure works — handlers just need to call `Invalidate`.

## Mutation Handler Template (Cache-Correct)
```go
func (h *Handler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
    // 1. Auth
    claims := auth.MustGetClaims(r.Context())

    // 2. Decode
    var req UpdateRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil { ... }

    // 3. Mutate (Spanner + Outbox)
    _, err := h.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        // ... business logic ...
        if err := txn.BufferWrite(mutations); err != nil { return err }
        return outbox.EmitJSON(txn, aggregate, id, topic, event, traceID)
    })
    if err != nil { ... }

    // 4. Cache invalidation (AFTER commit)
    h.Cache.Invalidate(ctx,
        cache.Key(cache.PrefixEntity, entityID),
        cache.Key(cache.PrefixEntityList, scopeID),
    )

    // 5. Respond
    writeJSON(w, http.StatusOK, response)
}
```

## Verification Checklist
- [ ] Every mutation handler calls `cache.Invalidate` AFTER Spanner commit
- [ ] No `cache.Invalidate` BEFORE the commit (races with rollback)
- [ ] No hardcoded TTL magic numbers — use named constants from `cache/keys.go`
- [ ] No direct `cache.Client` access — use `*cache.Cache` struct methods
- [ ] No `sync.Once` for reconnectable cache subsystems
- [ ] No `init()` goroutines that depend on Redis client

## Cross-References
- **intrusions.md** §5 (Cache & Redis Correctness) — full finding details
- **gemini-instructions.md** High-Performance Code §2 (Cache Invalidation via Redis Pub/Sub)
- **concurrency-shield** skill — package-level global safety, init() rules
