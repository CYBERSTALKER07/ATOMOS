# V.O.I.D. Intrusion Codex — Defensive Engineering Laws

> **Purpose**: This document is the "trap map" — every section is a class of mistake that has been made or nearly made in this codebase. It complements `copilot-instructions.md` (which tells what TO do) with concrete, codebase-grounded rules about what NOT to do and WHY.
>
> **Loading**: Always-on context. Every code generation, review, or refactor must check against these laws.
>
> **Authority**: Every rule below is grounded in a real finding from the V.O.I.D. ecosystem audit. File paths reference actual violations or near-misses. Do not dismiss a rule because it seems unlikely — it already happened.

---

## 1. Concurrency & Race Shield

### LAW 1.1 — No Package-Level Mutable Globals
**Violation found**: `cache/redis.go` — `var Client *redis.Client` is read by every request goroutine, written by the health monitor (set to `nil` on failure, reassigned on reconnect) with ZERO synchronization. This is a data race that causes nil pointer dereferences under load.

**Rule**: Package-level `var` that is written after init is FORBIDDEN. Use `atomic.Value`, `sync.RWMutex`-guarded struct fields, or hang the value on `*bootstrap.App`.

**Fix pattern**:
```go
// WRONG
var Client *redis.Client // written by health monitor, read by all handlers

// RIGHT — protected access
type Cache struct {
    mu     sync.RWMutex
    client *redis.Client
}
func (c *Cache) getClient() *redis.Client {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.client
}
```

### LAW 1.2 — No init() Side Effects
**Violation found**: `cache/middleware.go` — `init()` spawns 8 worker goroutines that call `Client.Set(...)`. If `Client` is nil at import time (Redis not yet initialized), these workers panic on the first enqueued job.

**Rule**: `init()` may register a codec, a driver, or a flag. It MUST NOT dial a database, read a file, start a goroutine, or reference any runtime-initialized singleton. Goroutine lifecycles start from explicit `Start(ctx)` methods called by `bootstrap.NewApp`.

### LAW 1.3 — Goroutines Must Propagate Context
**Violation found**: `supplier/returns.go`, `factory/crud.go`, `supplier/registration.go` — fire-and-forget `go func()` in request handlers without passing request `ctx`. If the parent request cancels, the goroutine runs with a stale/dead context, losing `trace_id` and potentially writing to a cancelled Spanner transaction.

**Rule**: Every `go func()` MUST accept a `ctx` derived from the request context (or a detached copy via `context.WithoutCancel` if intentionally outliving the request — with a comment explaining why). Every goroutine exits on `<-ctx.Done()`.

### LAW 1.4 — Bounded Goroutine Pools Only
**Rule**: `for _, x := range items { go process(x) }` is FORBIDDEN on any collection larger than a fixed, known-small batch. Use `errgroup.Group` with `SetLimit(n)` or `workers.Pool`. Unbounded goroutine fan-out under load causes OOM and Spanner connection exhaustion.

### LAW 1.5 — sync.Once Cannot Reconnect
**Violation found**: `cache/pubsub.go` — `globalRelayOnce sync.Once` means if Redis is down at boot, the Pub/Sub relay is permanently disabled even after Redis recovers. `sync.Once` fires exactly once — it has no reset.

**Rule**: For resources that must reconnect (Redis, external APIs, WebSocket relays), use a `sync.Mutex`-guarded initialization with a health-check loop, not `sync.Once`.

### LAW 1.6 — gorilla/websocket Concurrent WriteMessage Panics
**Violation found**: `ws/driver_hub.go` `PushToDriver` — iterates connection snapshots and calls `conn.WriteMessage` without a per-connection write lock. Two concurrent callers (e.g., two Kafka consumers pushing notifications) will cause a gorilla/websocket panic.

**Rule**: Every WebSocket connection MUST have a dedicated write goroutine (fan-in via channel) OR a per-connection `sync.Mutex` held during `WriteMessage`. Never call `WriteMessage` from multiple goroutines on the same `*websocket.Conn`.

**Fix pattern**:
```go
type SafeConn struct {
    conn *websocket.Conn
    mu   sync.Mutex
}
func (sc *SafeConn) WriteJSON(v interface{}) error {
    sc.mu.Lock()
    defer sc.mu.Unlock()
    return sc.conn.WriteJSON(v)
}
```

### LAW 1.7 — Standard Go Maps Are Not Thread-Safe
**Rule**: A bare `map[K]V` accessed from multiple goroutines (even read-only from some, write from others) is a data race. Use `sync.Map` for read-heavy/write-rare patterns, or a `sync.RWMutex`-guarded struct for everything else. The `-race` detector is the arbiter.

---

## 2. Financial Integrity Engine

### LAW 2.1 — float64 for Money Is a P0 Bug
**Violation found**: `main.go` ~L1719 — `Price float64` in the product variant DTO sent to iOS/Android clients. `order/ai_preorder.go` ~L373 — `var avgPrice float64` from Spanner `AVG()`, then `int64(avgPrice)` truncates instead of rounding.

**Rule**: ALL currency values are `int64` in **tiyins** (minor units). `float64` for money is forbidden at every layer: Go structs, JSON wire format, Spanner columns, TypeScript interfaces, Swift Codable, Kotlin @Serializable.

**Exception**: Spanner `AVG()` returns `FLOAT64` — immediately convert with `int64(math.Round(avgPrice))`, never bare `int64(avgPrice)` which truncates (e.g., 99.7 → 99 instead of 100).

### LAW 2.2 — JSON Number Decode Path
Go's `encoding/json` decodes all JSON numbers as `float64` by default. When decoding webhook payloads (Payme, Click, Global Pay), the amount field arrives as `interface{}` → `float64`.

**Rule**: Convert immediately at the decode boundary: `amount := int64(rawAmount.(float64))`. Never propagate the `float64` deeper into service logic. Better yet, use `json.Number` with `json.NewDecoder` + `decoder.UseNumber()` to avoid the float64 intermediate entirely.

### LAW 2.3 — Percentage Split Zero-Leak Invariant
**Rule**: For basis-point splits: `share = (amount * bps) / 10000`. After computing all shares, verify: `sum(shares) == totalAmount`. If rounding caused a 1-tiyin difference, add/subtract the remainder to/from the platform fee (never the customer or supplier share). A split where `sum ≠ total` is a reconciliation bomb.

### LAW 2.4 — Currency Is a First-Class Field
**Rule**: Every struct, table column, event payload, and DTO that carries an `Amount int64` MUST carry a paired `Currency string` (ISO-4217, 3-char). Never assume UZS. The system handles UZS, USD, and will add more. A ledger row without currency is unreconcilable.

### LAW 2.5 — Major-Unit Conversion at DTO Boundary Only
**Rule**: tiyin → som (or cents → dollars) conversion happens ONLY in the response serializer or the UI rendering layer. Never in service logic, never in repository queries, never in event payloads. Internal math is always minor-unit `int64`.

### LAW 2.6 — Ledger Is Append-Only
**Rule**: Refunds, adjustments, and corrections are NEW paired debit/credit rows in `LedgerEntries`. Never UPDATE an existing ledger row. The sum of all rows per currency per day MUST equal zero. Violation = reconciliation failure.

---

## 3. Spanner Discipline

### LAW 3.1 — ReadWriteTransaction for All Mutations
**Violation found**: `factory/crud.go` ~L309, L576 — `Apply` for multi-row factory entity creation. `Apply` does not retry on Spanner abort. Under contention, mutations silently fail.

**Rule**: Every mutation that writes ≥1 row MUST use `spanner.ReadWriteTransaction`. `Apply` is acceptable ONLY for single-row idempotent updates where retry-on-abort is unnecessary (e.g., heartbeat timestamp). When in doubt, use `ReadWriteTransaction`.

### LAW 3.2 — No Read-Then-Write Outside a Transaction
**Rule**: Reading a row, making a decision in Go, then writing back based on that decision — outside a `ReadWriteTransaction` — is a race condition. Another request can modify the row between your read and write. All read-decide-write sequences MUST happen inside a single `ReadWriteTransaction` callback.

**Prefer**: SQL-level atomic operations where possible: `SET Stock = Stock - @quantity` inside the transaction, rather than reading stock, subtracting in Go, then writing the new value.

### LAW 3.3 — Nullable Critical Columns Are Silent Killers
**Findings from `schema/spanner.ddl`**:
- `Orders.Amount` — nullable. Ledger, payment, and treasurer all assume it's present.
- `Drivers.SupplierId` — nullable. Every scope-check assumes it's present.
- `Drivers.Phone` — nullable. Auth via phone+PIN assumes it's present.
- `Retailers.Latitude/Longitude` — nullable. Geofence completion gate, H3 indexing, and proximity all assume present.
- `Retailers.Phone` — nullable. SMS notifications assume present.

**Rule**: Every handler that reads these columns MUST guard against NULL with an explicit check. Do not assume Spanner columns are NOT NULL unless the DDL says so. When writing new DDL, mark columns `NOT NULL` unless there's a documented reason for nullability.

### LAW 3.4 — Products vs SupplierProducts: Dead Table Trap
**Finding**: `Products` table at `schema/spanner.ddl` L159 has no `SupplierId`, no indexes, uses `NUMERIC` for price (inconsistent with `INT64` everywhere else), and has no `CreatedAt`. The active catalog table is `SupplierProducts`.

**Rule**: NEVER read from or write to the `Products` table. The catalog is `SupplierProducts`. If you see code referencing `Products`, it's either dead code or a bug. An AI will confuse these two tables — always verify which table the existing code uses.

### LAW 3.5 — Every Query Must Hit an Index
**Findings**: `LedgerEntries` has no index on `AccountId` (reconciliation by account = full scan). `MasterInvoices` has no index on `State` (finding pending invoices = full scan).

**Rule**: Before writing a `WHERE` clause, verify the filter columns are covered by an index in `schema/spanner.ddl`. If not, add a secondary index in a migration file — never accept a full-scan query. Use `EXPLAIN` to verify.

### LAW 3.6 — SupplierInventory PK Trap
**Finding**: `SupplierInventory` primary key is `ProductId` only. Two suppliers cannot stock the same product. If multi-supplier inventory is needed, the PK must be composite `(SupplierId, ProductId)`.

**Rule**: Before inserting into `SupplierInventory`, verify your use case doesn't need per-supplier scoping. If it does, this is a schema migration, not a code fix.

### LAW 3.7 — Mutation Cap: 1000 Per Transaction
**Rule**: Spanner hard limit is 20,000 cell mutations per transaction. Practical ceiling: 1,000 row mutations. For bulk operations, batch into multiple transactions. Never write an unbounded loop of mutations inside a single `ReadWriteTransaction`.

### LAW 3.8 — Stale Reads for Dashboards and List Views
**Rule**: Read-only queries for dashboards, analytics, and list views SHOULD use stale reads:
```go
spannerClient.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, stmt)
```
Strong reads (`Single().Query()`) acquire locks and compete with write transactions. Reserve strong reads for mutation precondition checks inside `ReadWriteTransaction`.

---

## 4. Kafka Event Contract Discipline

### LAW 4.1 — trace_id Is NOT in Event Struct Bodies
**Finding**: Zero event structs in `kafka/events.go` contain a `trace_id` JSON field. The trace_id is threaded ONLY as a Kafka header by the outbox relay (`outbox/relay.go` L136-138).

**Consequence**: Consumers that don't read Kafka headers lose trace correlation. Any code that JSON-deserializes an event and re-emits it drops the trace_id. An AI generating a new event struct will not include `trace_id` because none of the existing structs have it.

**Rule**: When consuming events, ALWAYS extract `trace_id` from Kafka message headers and inject it into the processing context via `telemetry.WithTraceID(ctx, traceID)`. When producing via `outbox.EmitJSON`, ALWAYS pass the `traceID` parameter (it's variadic — the compiler won't warn if you omit it).

### LAW 4.2 — EmitJSON traceID Is Variadic (Easy to Forget)
**Finding**: `outbox/emit.go` L75 — `func EmitJSON(txn ..., traceID ...string)`. The variadic signature means callers can omit `traceID` without a compiler error.

**Rule**: Every `outbox.EmitJSON` call MUST include the trace_id as the last argument. Code review must flag any `EmitJSON` call with fewer arguments than the full signature. Extract trace_id from context: `telemetry.TraceIDFromContext(ctx)`.

### LAW 4.3 — omitempty Inconsistency on SupplierId
**Finding**: `SupplierId` field uses `json:"supplier_id"` in some event structs and `json:"supplier_id,omitempty"` in others (`OrderCompletedEvent`, `FleetDispatchedEvent`). Consumers expecting `supplier_id` always present get an absent field for some events.

**Rule**: Fields that are ALWAYS present MUST NOT have `omitempty`. Fields that are genuinely optional MUST have `omitempty`. Never mix for the same field name across event structs. When adding a new event struct, grep existing structs for the same field name and match the tag exactly.

### LAW 4.4 — State-Change Events MUST Use Outbox
**Violation found**: ~30 `writer.WriteMessages` calls in handler code for state-change events (not telemetry). Concentrated in `factory/` (11 instances with swallowed errors), `payment/webhooks.go`, `order/unified_checkout.go`, `supplier/warehouses.go`.

**Rule**: Any event representing a durable state transition (entity created, order state changed, payment settled, manifest lifecycle) MUST be emitted via `outbox.EmitJSON` inside the same `ReadWriteTransaction` that writes the business data. Direct `writer.WriteMessages` is acceptable ONLY for loss-tolerant telemetry (`telemetry.ping`, `fleet.location`).

**Ghost entity risk**: If the DB write commits but the Kafka write fails (network blip, broker down), the entity exists in Spanner but no consumer ever learns about it. The outbox pattern makes this impossible by construction.

### LAW 4.5 — Swallowed Kafka Write Errors
**Violation found**: `factory/replenishment_lock.go`, `factory/look_ahead.go`, `factory/emergency.go` (11 instances) — `_ = s.Producer.WriteMessages(...)`. The error is discarded. If Kafka is temporarily unreachable, state-change events are silently lost.

**Rule**: Never `_ = producer.WriteMessages(...)` for state-change events. Either use the outbox (preferred) or handle the error explicitly (log, retry, DLQ). The `_ =` pattern for Kafka writes is a P1 code-review block.

### LAW 4.6 — Import Cycle Workaround: Raw Strings Are Intentional
**Context**: `payment` cannot import `kafka` (creates a cycle through treasurer). `proximity` cannot import `kafka` (creates a cycle through telemetry). These packages use raw string constants like `"ORDER_COMPLETED"` instead of `kafka.EventOrderCompleted`.

**Rule**: Do NOT "fix" this by importing `kafka` into `payment` or `proximity` — the cycle is real. If you need a new event constant in a cycle-prone package, either define it locally with a comment linking to `kafka/events.go`, or restructure the dependency (extract the constant to a leaf package).

### LAW 4.7 — Consumer Idempotency via Version Gating
**Rule**: Every Kafka consumer MUST read the target row's `Version` (or `UpdatedAt`) before applying an event. If `event.version ≤ stored.version`, the event is a stale replay — ACK and skip. Never blindly overwrite. Stale replays are expected under at-least-once delivery.

---

## 5. Cache & Redis Correctness

### LAW 5.1 — cache.Invalidate Is Near-Zero Despite Infrastructure Being Ready
**Finding**: The entire codebase has ~1 `cache.Invalidate` call (in `order/service.go`) vs ~50+ mutation paths across `factory/`, `supplier/`, `warehouse/`, `auth/`, `replenishment/`. The infrastructure is ready (`cache.Invalidate`, `cache.StartInvalidationSubscriber`), but handler adoption is near-zero.

**Rule**: Every `POST`/`PATCH`/`PUT`/`DELETE` handler that mutates a cached aggregate MUST call `cache.Invalidate(ctx, keys...)` AFTER the Spanner commit (not before — pre-commit invalidation races with rollback). This is not optional. If you touch a mutation handler, add `cache.Invalidate` in the same commit.

### LAW 5.2 — TTL Is a Safety Net, Not a Correctness Mechanism
**Rule**: Default TTL is 5 minutes. Anything longer requires explicit justification. But TTL alone is NEVER sufficient for correctness — you MUST also call `cache.Invalidate` on mutation. A 5-minute TTL means up to 5 minutes of stale data if you forget to invalidate.

### LAW 5.3 — No TTL Constants for Many Prefixes
**Finding**: `cache/keys.go` defines prefixes for `SupplierProfile`, `RetailerProfile`, `DriverProfile`, `CatalogSearch`, `Analytics`, `Settings` but has no corresponding TTL constants for them.

**Rule**: When using a cache prefix, check `cache/keys.go` for its TTL. If none exists, use `TTLDefault` (5 min) and add a named constant. Never hardcode a TTL as a magic number.

### LAW 5.4 — Redis Client Access Must Be Nil-Safe
**Finding**: `cache/redis.go` `var Client *redis.Client` can be `nil` (Redis down at boot or during health monitor reconnect). Any code that calls `Client.Get(...)` without a nil check will panic.

**Rule**: Always check `Client != nil` before any Redis operation, or use the `*cache.Cache` struct methods which handle nil internally. Never access `cache.Client` directly in new code — use the struct-based API.

---

## 6. WebSocket & Real-Time Security

### LAW 6.1 — FleetHub Has Zero Authentication (P0)
**Finding**: `ws/hub.go` `FleetHub.HandleConnection` upgrades the WebSocket with NO JWT verification. No claims check, no query param auth fallback. Any client on the network can connect and receive all GPS telemetry for all drivers.

**Rule**: EVERY WebSocket hub MUST authenticate before `Upgrader.Upgrade`. Extract JWT from `Authorization` header or signed query-string token. Reject unauthenticated connections with 401 BEFORE upgrade. This is the #1 security vulnerability in the backend.

### LAW 6.2 — Query Param Auth Fallback Is Role-Spoofing (P0)
**Finding**: `ws/driver_hub.go`, `ws/retailer_hub.go`, `ws/warehouse_hub.go`, `ws/payloader_hub.go` — all have this pattern:
```go
if ok && claims != nil {
    driverID = claims.UserID
} else {
    driverID = r.URL.Query().Get("driver_id") // ANYONE CAN SET THIS
}
```
If the JWT middleware is not wired on the WebSocket route, anyone can connect with `?driver_id=X` and receive that driver's payment notifications.

**Rule**: Query parameters MUST NEVER be used as an auth fallback. If JWT extraction fails, the connection MUST be rejected. Period. A query param may carry a signed token (HMAC or JWT), but never a bare ID.

### LAW 6.3 — Cross-Pod Delivery Requires Redis Pub/Sub Relay
**Finding**: `DriverHub`, `WarehouseHub`, `PayloaderHub` write only to local connections. Unlike `RetailerHub` which calls `cache.Publish` for cross-pod relay, these three hubs have no Redis Pub/Sub broadcast.

**Consequence**: In a multi-pod deployment, notifications are delivered ONLY to connections on the same pod as the producer. Payment settlement notifications, dispatch updates, and warehouse alerts silently fail to reach users connected to other pods.

**Rule**: Every hub's broadcast method MUST publish to a Redis Pub/Sub channel. Every pod subscribes to that channel and delivers to its local connections. The `RetailerHub` pattern is the reference implementation.

### LAW 6.4 — Origin Allowlist Hardcoded to Localhost
**Finding**: `ws/hub.go` `CheckWSOrigin` has a hardcoded map of `localhost` origins. Production domains (`admin.thelab.uz`, etc.) are not included.

**Rule**: WebSocket origin allowlist MUST include production domains. Use the same CORS allowlist resolved in `bootstrap.NewApp` for WebSocket origin checks. Dynamic LAN/ngrok patterns for dev are fine, but production domains are non-negotiable.

### LAW 6.5 — Keepalive Timing Is Inverted
**Finding**: `ws/keepalive.go` — `PingInterval = 30s`, `PongWait = 65s`. Doctrine specifies 15s ping, 30s read deadline.

**Rule**: `PingInterval` MUST be less than `PongWait/2`. The current 30s/65s is functional but suboptimal (dead connections take up to 65s to reap). Target: 15s ping, 30s pong wait.

---

## 7. Context & Observability

### LAW 7.1 — context.Background() in Request Paths Breaks Tracing
**Violation found**: `payment/webhooks.go` L941, L968 — payment event emission uses `context.Background()` via worker pool, losing `trace_id` propagation. `order/unified_checkout.go` L249, `supplier/returns.go` L260, `supplier/retailer_pricing.go` L463 — same pattern.

**Rule**: Never `context.Background()` inside a request-scoped call path. If you need a context that outlives the request (e.g., async cleanup), use `context.WithoutCancel(ctx)` (Go 1.21+) to preserve trace values while detaching cancellation. Comment why.

**Acceptable uses of context.Background()**: `main()` startup, cron job tickers, test setup. NOWHERE ELSE.

### LAW 7.2 — Swallowed Errors Are Silent Data Loss
**Violation found**: `supplier/manifest.go` L679, `factory/force_receive.go` L177, `factory/pull_matrix.go` L101 — `_, _ = s.Spanner.Apply(...)`. Audit/SLA inserts silently fail. `analytics/retailer.go` L121, `kafka/notification_dispatcher.go` L443 — `_ = row.Columns(...)` silently produces partial/corrupted data.

**Rule**: `_ = someFunc()` is almost always wrong. The correct form is:
```go
if err := someFunc(); err != nil {
    slog.WarnContext(ctx, "non-fatal: <what failed>", "err", err, "trace_id", traceID)
}
```
If you genuinely intend to discard an error, add a line-end comment explaining why. Any `_ =` on a Spanner write or Kafka produce is a P1 review block.

### LAW 7.3 — Bare return err Loses Stack Context
**Violation found**: `vault/vault.go` L372, `vault/onboarding.go` L158, `factory/network_optimizer.go` L390 — `return err` at package boundary without wrapping.

**Rule**: Always wrap errors with context: `return fmt.Errorf("reassign route %s: %w", routeID, err)`. Never `return err` bare from >1 call site deep. The stack trace is useless without a handle.

### LAW 7.4 — String-Based Error Matching Is Fragile
**Violation found**: `supplier/manifest.go` L1279 — `err.Error()[:14]` string slice comparison.

**Rule**: Use `errors.Is(err, sentinel)` for sentinel errors. Use `errors.As(err, &target)` for structured errors. NEVER compare error strings — they can change across library versions, and substring matching is fragile to message rewording.

### LAW 7.5 — Every Log Line Needs trace_id
**Rule**: Every `slog.InfoContext` / `slog.ErrorContext` in a request-scoped path MUST include the `trace_id` field. Extract from context: `telemetry.TraceIDFromContext(ctx)`. A single order lifecycle (webhook → DB → Kafka → WS → mobile ACK) must be traceable by grepping `trace_id=<uuid>` across pod logs.

---

## 8. Auth & Scope Enforcement

### LAW 8.1 — Scope IDs From Request Bodies Are Role-Spoofing
**Rule**: NEVER read `supplier_id`, `factory_id`, or `warehouse_id` from `r.Body`, `r.FormValue()`, or `r.URL.Query().Get()` for authorization decisions. ALWAYS resolve from JWT claims via `claims.ResolveSupplierID()`, `auth.GetFactoryScope()`, `auth.GetWarehouseOps()`, `auth.RequireWarehouseScope()`.

**Exception**: Query params for read-only FILTERING (not authorization) are acceptable IF the handler first resolves the user's scope from JWT and then validates the query param falls within that scope. `auth/warehouse_scope.go` is the reference pattern.

### LAW 8.2 — Every Mutation Endpoint Needs RequireRole
**Rule**: Every HTTP handler that mutates data MUST have an `auth.RequireRole` wrapper (or explicit signature-first webhook pattern for external callbacks). A mutation endpoint reachable without auth is a P0 security bug. No exceptions.

### LAW 8.3 — Webhook Signature Verification Is Line 1
**Rule**: In webhook handlers (Payme, Click, Stripe, FCM), the signature/HMAC check is the FIRST non-trivial statement — before `json.Decode`, before any Spanner read, before any business logic. Parsing an unverified body is an injection vector. Mismatch → 401 + structured log + metric + return. No "soft validation", no grace windows, no non-prod bypasses.

### LAW 8.4 — JWT Expiry and Refresh Token Rotation
**Rule**: Access tokens have short TTL (15 min). Refresh tokens are single-use and rotated on every refresh. Never accept an expired JWT. Never reuse a refresh token — replay detection MUST be in place. A stolen refresh token that can be reused indefinitely is a session hijack.

---

## 9. Frontend (TypeScript / React) Traps

### LAW 9.1 — API Base URL Must Be a Single Constant
**Violation found**: `process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'` is copy-pasted into 20+ files across the admin portal (`lib/auth.ts`, `hooks/useTelemetry.ts`, `app/page.tsx`, `app/fleet/page.tsx`, `app/treasury/*/page.tsx`, `app/auth/*/page.tsx`, etc.).

**Rule**: Define once in `lib/config.ts` (or equivalent):
```typescript
export const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
```
Import everywhere. Never inline `process.env.NEXT_PUBLIC_API_URL` in component files.

Similarly, map style URLs (CartoDB dark-matter) duplicated across 4 files should be a single constant.

### LAW 9.2 — Never .catch(() => {})
**Violation found**: 9 instances in the admin portal — `lib/auth.ts` (token storage), `lib/usePolling.ts`, `lib/useLiveData.ts`, `hooks/useSupplierShift.tsx`, `app/fleet/page.tsx`, `app/supplier/orders/page.tsx`.

**Rule**: Never silently swallow a Promise rejection. At minimum, log the error:
```typescript
.catch((err) => console.error('polling failed:', err))
```
For user-facing operations (order actions, shift toggles), surface the error in the UI. For background operations (cleanup, abort), log with context. An empty `.catch(() => {})` hides operational failures that are impossible to diagnose.

### LAW 9.3 — Route-Level Error Boundaries
**Finding**: Only `app/error.tsx` exists (root-level). No nested `error.tsx` in `/fleet`, `/treasury`, `/supplier/*`, or any other route group. No `not-found.tsx` at any level.

**Rule**: Every major route group (`/fleet`, `/treasury`, `/supplier`, `/warehouse`, `/factory`, `/auth`) SHOULD have its own `error.tsx` that catches errors within that subtree without nuking the entire app shell. Add `not-found.tsx` at the root level for 404 handling.

### LAW 9.4 — Tauri IPC: JS invoke() Requires a Rust Handler
**Violation found**: `retailer-app-desktop/lib/bridge.ts` calls `invoke('store_token')`, `invoke('get_stored_token')`, `invoke('clear_stored_token')`. `retailer-app-desktop/src-tauri/src/lib.rs` has ZERO registered command handlers. These invocations silently fail. Auth tokens are not persisted — users lose sessions on app restart.

**Rule**: Every Tauri `invoke('command_name')` on the JS side MUST have a matching `#[tauri::command]` function registered via `.invoke_handler(tauri::generate_handler![...])` on the Rust side. Test IPC commands in integration tests, not just the JS surface.

### LAW 9.5 — No `any` at Domain Boundaries
**Rule**: `any` is acceptable for third-party library interop (`as any` for Recharts callbacks, WebGL libraries). `any` is FORBIDDEN for domain types: API response payloads, store state, component props, event handlers for business actions. Use types from `packages/types` or define local interfaces.

### LAW 9.6 — Every Data Surface Needs Five States
**Rule**: Every component that fetches data must handle: (1) loading, (2) empty, (3) error, (4) stale/offline, (5) success. A component that shows data or a blank screen with no feedback is fake completeness.

---

## 10. Native Mobile Traps

### LAW 10.1 — Swift: Never Force-Unwrap URLs
**Violation found**: `driverappios/Services/SyncServiceLive.swift` L33 — `URL(string: "\(APIClient.shared.apiBaseURL)/v1/sync/batch")!`. `driverappios/ViewModels/TelemetryViewModel.swift` L41 — same pattern for WebSocket URL.

**Rule**: URL construction from string interpolation MUST use `guard let url = URL(string: ...) else { return }` or a throwing initializer. If `apiBaseURL` ever contains a space, Unicode character, or trailing slash artifact, the force unwrap crashes the app.

### LAW 10.2 — Swift: Never try! in Production Paths
**Violation found**: `driverappios/Views/OfflineVerifierView.swift` L303 — `try! ModelContainer(for: OfflineDelivery.self, configurations: config)`. SwiftData initialization can fail (corrupted store, migration failure). `try!` turns a recoverable error into an unrecoverable crash.

**Rule**: Use `do { ... } catch { ... }` with user-facing error recovery. `try!` is acceptable ONLY in unit tests and previews.

### LAW 10.3 — Kotlin: CoroutineScope Needs SupervisorJob
**Violation found**: `driver-app-android/.../TelemetrySocket.kt` L39 — `CoroutineScope(Dispatchers.IO)` without `SupervisorJob()`. If any child coroutine fails, all sibling coroutines in the scope are cancelled.

**Rule**: Custom `CoroutineScope` MUST include `SupervisorJob()`: `CoroutineScope(SupervisorJob() + Dispatchers.IO)`. This isolates child failures. Without it, a single failed WebSocket reconnect cancels all active telemetry uploads.

### LAW 10.4 — Kotlin: Every Custom Scope Needs cancel()
**Violation found**: `TelemetrySocket.kt` creates a scope with no visible `scope.cancel()` on disconnect/destroy. The scope and its child coroutines leak.

**Rule**: Every `CoroutineScope` created manually MUST have a matching `scope.cancel()` in the lifecycle teardown method (`onDestroy`, `onCleared`, `close()`). `viewModelScope` and `lifecycleScope` handle this automatically — prefer them.

### LAW 10.5 — Local Models Must Link to Backend Canonical Type
**Rule**: Every Swift `Codable` struct or Kotlin `@Serializable` class that mirrors a backend Go struct MUST carry a comment linking to the canonical source:
```swift
// Mirror of backend-go/order.Order — keep JSON keys aligned
struct Order: Codable { ... }
```
When the backend adds a field, this comment is the breadcrumb that leads to the mobile model that needs updating.

### LAW 10.6 — packages/types Must Stay Aligned
**Finding**: `packages/types/entities.ts` `Driver` interface is missing `HomeNodeType`, `HomeNodeId`, `WarehouseId` — fields added in Phase VII and written by all four fleet-creation handlers. `Vehicle` is similarly stale.

**Rule**: When a backend Go struct adds a field, the corresponding interfaces in `packages/types`, Swift `Codable` structs, and Kotlin `@Serializable` classes MUST be updated in the same change set (or the same PR, with explicit cross-platform task tracking).

---

## 11. Schema & Type Drift

### LAW 11.1 — Backend Field Addition Must Propagate
**Rule**: Adding a field to a Go struct is not complete until:
1. Spanner DDL column exists (if persisted)
2. `packages/types` TypeScript interface updated
3. Swift `Codable` struct updated (all iOS apps that consume it)
4. Kotlin `@Serializable` class updated (all Android apps that consume it)
5. JSON tag matches across all representations (`snake_case` everywhere)
6. `omitempty` behavior is consistent across all event/DTO structs using the field

Failure to propagate = schema drift = gap-hunter Class-1.

### LAW 11.2 — Spanner Columns That Exist But Are Never Written
**Documented in copilot-instructions.md Known Gap #13**:
- `Drivers`: `DepartedAt`, `EstimatedReturnAt`, `OfflineReason`
- `Orders`: `CancelLockedAt`, `ConfirmationNotifiedAt`, `AiPendingConfirmation`
- `Retailers`: `AccessType`, `StorageCeilingHeightCM`

**Rule**: These are forward-provisioned for upcoming features. Do NOT delete them. Do NOT read them and expect values. Populate them when the owning feature ships.

### LAW 11.3 — JSON Tag Consistency
**Rule**: Every Go struct field that appears in JSON (API response, Kafka event, WebSocket payload) MUST have an explicit `json:"snake_case"` tag. The tag MUST match the corresponding TypeScript property name, Swift `CodingKey`, and Kotlin `@SerialName`. A tag mismatch is silent data loss on the consumer side.

---

## 12. Outbox & Event Relay

### LAW 12.1 — Relay Poll Interval Is 2s, Not 250ms
**Finding**: `outbox/relay.go` L47 — default poll interval is `2 * time.Second`. Doctrine states 250ms. This means outbox events have up to 2s of latency before being published to Kafka, giving a p99 end-to-end latency of ~2-3s for state-change notifications.

**Rule**: Be aware of this latency when designing real-time features. If sub-second notification delivery is required, the relay interval must be tuned down (config change, not code change in new handlers).

### LAW 12.2 — Relay Uses Strong Reads (Not Stale)
**Finding**: `outbox/relay.go` L86 — `r.spanner.Single().Query()` is a strong read. It acquires locks and competes with handler write transactions.

**Impact**: Under high write load, the relay's strong reads add contention. Switching to stale reads (`ExactStaleness(1 * time.Second)`) would reduce contention at the cost of up to 1s additional delay.

### LAW 12.3 — markPublished Uses Apply (No Retry on Abort)
**Finding**: `outbox/relay.go` L130 — `r.spanner.Apply()` to mark events as published. If Spanner aborts this write, the event is NOT marked published and WILL be re-delivered on the next tick.

**Impact**: Consumers MUST be idempotent (per LAW 4.7). Under Spanner contention, duplicate event delivery rate increases. This is by design (at-least-once) but consumers that aren't idempotent will produce duplicate side effects.

### LAW 12.4 — No Stuck-Event Watchdog
**Finding**: No mechanism detects events stuck in the outbox (`CreatedAt < now - 60s AND PublishedAt IS NULL`). If the relay fails to publish an event (e.g., topic/shape mismatch), it retries forever with no alerting.

**Rule**: Until a watchdog is implemented, monitor `OutboxEvents` manually after deploying changes to event shapes. A shape mismatch between `EmitJSON` payload and the topic's expected schema will cause permanent publish failures.

### LAW 12.5 — Never Mix writer.WriteMessages and outbox.EmitJSON
**Rule**: In the same mutation path, use ONE event emission strategy. Mixing inline `writer.WriteMessages` with `outbox.EmitJSON` creates inconsistent atomicity guarantees — the outbox event is atomic with the Spanner commit, the inline write is not. If both exist, the inline write can succeed while the outbox write rolls back (or vice versa).

---

## Quick-Reference: Severity Classification

| Severity | Meaning | Response |
|---|---|---|
| **P0** | Active security vulnerability or data corruption risk | Fix before any feature work. Block deployment. |
| **P1** | Silent failure, data loss, or ghost entity risk | Fix when touching the affected file. Never introduce new instances. |
| **Medium** | Degraded reliability or developer experience | Fix opportunistically. Track in Known Gaps. |
| **Low** | Style, consistency, or future maintenance concern | Fix during dedicated cleanup sprints. |

### Current P0 Findings (Must Fix)
1. `ws/hub.go` — FleetHub zero authentication (LAW 6.1)
2. `ws/driver_hub.go`, `ws/retailer_hub.go`, `ws/warehouse_hub.go`, `ws/payloader_hub.go` — query-param auth spoofing (LAW 6.2)
3. `cache/redis.go` — `var Client` data race under health monitor reconnect (LAW 1.1)
4. `cache/middleware.go` — `init()` goroutine panic on nil Client (LAW 1.2)
5. `ws/driver_hub.go` — concurrent `WriteMessage` without lock (LAW 1.6)

### Current P1 Findings (Fix on Touch)
1. `main.go` — `Price float64` in variant DTO (LAW 2.1)
2. `order/ai_preorder.go` — `int64(avgPrice)` truncation (LAW 2.1)
3. `factory/crud.go` — `Apply` for multi-row mutations (LAW 3.1)
4. `factory/*.go` — 11 `_ = producer.WriteMessages(...)` swallowed errors (LAW 4.5)
5. `payment/webhooks.go` — `context.Background()` for Kafka emit (LAW 7.1)
6. `supplier/manifest.go` — `_, _ = s.Spanner.Apply(...)` swallowed errors (LAW 7.2)
7. `supplier/manifest.go` — `err.Error()[:14]` string matching (LAW 7.4)
8. Cache invalidation near-zero adoption across all mutation handlers (LAW 5.1)
9. All inline `writer.WriteMessages` for state-change events (LAW 4.4)
