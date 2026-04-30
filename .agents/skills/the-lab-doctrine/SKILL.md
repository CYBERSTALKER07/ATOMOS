---
name: the-lab-doctrine
description: Canonical playbook for implementing, extending, or auditing any feature in the V.O.I.D. / The Lab Industries monorepo. Use whenever the user asks to add a feature, endpoint, event, page, screen, or data field — before any code is authored. Captures the role→app matrix, backend package topology, the six-step mutating-handler shape, the dual event path (transactional outbox + EmitNotification), Spanner/cache/idempotency primitives, WebSocket hub discipline, auth scoping, H3 spatial rules, cross-role sync, and the local-simulator + physical-device dev loop so a new feature lands coherently across every surface that consumes it.
version: 1.0.0
---

# The Lab Doctrine — Feature-Complete Authoring Across Every Surface

A feature in this repo is never one file. A new order state, a new driver screen, a new supplier setting, or a new dispatch rule fans out across: Go handler → Spanner txn + outbox → Kafka topic → consumer → WebSocket hub → TS type → Swift `Codable` → Kotlin `@Serializable` → mobile view model → portal page. Skipping any link creates contract drift, zombie caches, or silent failures. This skill captures the shape of every link so a new feature is wired correctly end-to-end the first time.

## When to Use

Invoke at the start of any of these:
- User says "add", "implement", "create", "build", "extend" a feature, endpoint, page, screen, or event.
- User says "wire up", "make this work end-to-end", "push to mobile", "notify the driver/supplier/retailer".
- User asks to add a new Spanner column, a new Kafka event, or a new JWT claim.
- User asks to test locally or on a physical device.
- Before declaring any cross-cutting feature "done" — run the Feature Implementation Checklist at the end of this skill.

Pair with `efficient-code` for code-quality gates and `gap-hunter` for post-change drift sweeps.

## 1. The Role → App Matrix (canonical)

A role is a product, not an app. Every feature for a role must land on EVERY surface in that role's row.

| Role (JWT) | Backend scope | Clients that must stay in sync |
|---|---|---|
| `SUPPLIER` (encoded as `role=ADMIN`) | `claims.ResolveSupplierID()` | `apps/admin-portal` (a.k.a. the Supplier Portal — the primary product surface) |
| `DRIVER` | home-node scoped via `auth.ResolveHomeNode` | `apps/driver-app-android`, `apps/driverappios` |
| `RETAILER` | self-registered | `apps/retailer-app-android`, `apps/retailer-app-ios`, `apps/retailer-app-desktop` |
| `PAYLOAD` | terminal-scoped | `apps/payload-terminal` (only surviving Expo app) |
| `FACTORY_ADMIN` | `auth.GetFactoryScope` | `apps/factory-portal`, `apps/factory-app-android`, `apps/factory-app-ios` |
| `WAREHOUSE_ADMIN` | `auth.RequireWarehouseScope` | `apps/warehouse-portal`, `apps/warehouse-app-android`, `apps/warehouse-app-ios` |

**"ADMIN" in JWT = SUPPLIER in product language.** There is no separate platform-admin identity. Every logged-in user of the admin-portal is a supplier. Legacy route names / variables keeping the word "supplier" are canonical — preserve them.

## 2. Backend Package Topology

Go module: `apps/backend-go`, wired via repo-root `go.work` — no `replace` directives.

- `main.go` is **lifecycle only**: config → `bootstrap.NewApp` → route registration → `ListenAndServe` → graceful shutdown. Target ceiling **200 lines**. Do not grow it. New handlers go in their domain package.
- `bootstrap/` is the composition root. Every Spanner/Redis/Kafka writer, every WS hub, the proximity engine, outbox relay, cron, and priority-guard middleware hangs off `*bootstrap.App`. New app-wide singletons go here — never package-level globals.
- Router: `chi.Router`. Register new routes in a `*routes` package's `RegisterRoutes(r chi.Router, d Deps)`. Do NOT register on `http.DefaultServeMux`.
- Domain packages (business logic): `admin/ analytics/ auth/ cart/ countrycfg/ crypto/ dispatch/ errors/ factory/ fleet/ hotspot/ idempotency/ kafka/ models/ notifications/ order/ outbox/ payment/ pkg/ proximity/ replenishment/ routing/ schema/ secrets/ settings/ storage/ supplier/ telemetry/ vault/ warehouse/ workers/ ws/`.
- Route-composition packages (thin mounts + middleware only): `adminroutes/ airoutes/ authroutes/ catalogroutes/ deliveryroutes/ driverroutes/ factoryroutes/ fleetroutes/ payloaderroutes/ paymentroutes/ sync/ treasury/ userroutes/ warehouseroutes/ webhookroutes/`.

`Deps` in a `*routes` package is narrow — pass only what the routes need. Never pass `*bootstrap.App` (import cycle + leaks composition concerns). If an inline closure exceeds ~10 lines, lift it to a handler function in the owning domain package.

## 3. The Six-Step Mutating-Handler Shape (mandatory)

Every handler that mutates state follows this order, no exceptions:

```go
func createThing(w http.ResponseWriter, r *http.Request) {
    // 1. AUTH GATE — role + scope resolution (supplier / home node / warehouse / factory).
    claims := auth.MustClaims(r)
    supplierID := claims.ResolveSupplierID()          // never trust request-body supplier_id
    // 2. METHOD GATE — chi does this via r.Post/Put, but internal routers still check.
    // 3. TXN — all writes inside spanner.ReadWriteTransaction.
    _, err := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        if err := insertRow(txn, …); err != nil { return err }
        // 4. OUTBOX — one OutboxEvents row per durable state transition, inside the txn.
        return outbox.EmitJSON(txn, "Driver", driverID, kafka.TopicMain, kafka.DriverCreatedEvent{…})
    })
    if err != nil { /* 5xx + structured log with trace_id */ ; return }
    // 5. CACHE INVALIDATE — after commit. Pub/Sub fans out to peer pods.
    cache.Invalidate(r.Context(), cache.DriversBySupplier(supplierID))
    // 6. BEST-EFFORT UX FAN-OUT — EmitNotification keyed by EventType (see §4).
    kafka.EmitNotification(kafka.EventDriverCreated, kafka.DriverCreatedEvent{…})
    // Structured log with trace_id + aggregate_id.
    slog.InfoContext(ctx, "driver.created", "driver_id", driverID, "supplier_id", supplierID)
    writeJSON(w, 201, resp)
}
```

Skipping any step is a P0 correctness bug. `RequireRole` sits one level above this in the routes package.

## 4. The Event Doctrine — Dual Path (this is the critical one)

V.O.I.D. has **two** Kafka producers with different keying strategies. They coexist on purpose.

### 4a. Transactional Outbox (durable truth)
- **Purpose**: atomicity. A state transition that commits in Spanner MUST either publish the corresponding Kafka event or retry until it does — never diverge.
- **How**: inside a `ReadWriteTransaction`, call `outbox.EmitJSON(txn, aggregateType, aggregateID, topic, payload)`. This writes an `OutboxEvents` row in the same commit as the domain row.
- **Relay**: `outbox.Relay` (started by `bootstrap.NewApp`) tails `OutboxEvents` via stale reads, publishes through `kafka.InitSyncWriter` (`RequiredAcks=all`, `MaxAttempts ≥ 5`), and marks `PublishedAt` on success.
- **Key**: `AggregateID` (driver_id, order_id, manifest_id …). This guarantees per-entity ordering across Kafka partitions.
- **Consumers**: durable projectors (AI worker, analytics, any future typed consumer) read AggregateID-keyed messages.
- **Direct `writer.WriteMessages` from a handler is forbidden** for state-change events. No exceptions.

### 4b. EmitNotification (best-effort UX fan-out)
- **Purpose**: push real-time UX events to WebSocket hubs + FCM/APNs with minimal latency. Durability is provided by 4a; this path is additive.
- **How**: call `kafka.EmitNotification(eventType, payload)` **after** the Spanner commit. `InitNotificationWriter` runs on the same `TopicMain` but the message is keyed by `EventType`.
- **Key**: `EventType` string (e.g. `FLEET_DISPATCHED`, `DRIVER_CREATED`, `FREEZE_LOCK_ACQUIRED`). The `notification_dispatcher` routes switch cases on this key.
- **Tolerance**: fire-and-forget. An occasional drop does not corrupt state because the outbox is canonical. Don't wrap it in retries; don't block the response on it.
- **When to use**: any event whose primary purpose is notifying a human (toast in a portal, push on a phone). When in doubt, emit on BOTH paths — they do not conflict.

### 4c. Event Registration Checklist (new event type)
1. Add the payload struct + constant in `apps/backend-go/kafka/events.go`.
2. Emit inside the mutating handler — outbox for durability, EmitNotification for UX (one or both).
3. Add a switch case + handler in `apps/backend-go/kafka/notification_dispatcher.go`.
4. Add a human-readable formatter in `apps/backend-go/notifications/formatter.go` matching the existing style.
5. Mirror the payload shape in `packages/types/ws-events.ts` (add to the `WSEvent` discriminated union).
6. For each consuming mobile client, add a `Codable` (Swift) / `@Serializable` (Kotlin) model with JSON tags matching the backend struct exactly. Use `gap-hunter` to confirm no shape drift.
7. Subscribe on the correct WS hub: `FleetHub` (supplier telemetry), `DriverHub` (driver-scoped), `RetailerHub` (retailer-scoped), `SupplierHub` (supplier-scoped), `PayloaderHub` (payload-terminal), `TelemetryHub` (driver location).

## 5. Spanner Patterns

- **Schema of record**: `apps/backend-go/schema/spanner.ddl`. Every new column goes there first; models & DTOs follow. Run `make spanner-init` after edits.
- **Multi-tenant scoping**: every row except `Retailers` carries `SupplierId`. Every `WHERE` clause that returns a list MUST filter by `SupplierId` (or `WarehouseId`/`FactoryId` for node-scoped queries). Missing scope = cross-tenant leak.
- **Index discipline**: every `WHERE` hits a declared secondary index (`Idx_<Table>_By<Column>` or composite). No full-table scans.
- **Stale reads**: for read paths where 15s staleness is acceptable, use `spanner.TimestampBound{StaleRead: 15s}`. Strong reads only when correctness demands.
- **Batch writes**: ≤ 1000 mutations per `ReadWriteTransaction`. Use `InsertOrUpdateMap` for bulk. Never `Apply` for multi-row.
- **Home-node contract**: `Drivers` and `Vehicles` have `HomeNodeType` (`WAREHOUSE`|`FACTORY`), `HomeNodeId`, and the legacy `WarehouseId` field. Always write all three when `HomeNodeType = 'WAREHOUSE'` (keeps legacy readers working).

## 6. Cache & Idempotency

- **Cache-aside**: `cache.Get` before Spanner; on miss, read Spanner, `cache.Set` with a short TTL (default 30s–5m based on volatility). TTL is the safety net, never the correctness mechanism.
- **Invalidation**: every mutation ends with `cache.Invalidate(ctx, keys...)`. Invalidation fans out to peer pods via Redis Pub/Sub. Declare cache keys as helpers in the owning package (`cache.DriversBySupplier(id)`), never inline strings.
- **Idempotency**: all public mutating handlers wrap through `idempotency.Guard` (chi middleware). Clients send `Idempotency-Key` header; guard keys on `(route + key + auth_subject)`. Replays return the cached response verbatim.
- **Webhooks**: idempotency key is the gateway's transaction id (Payme `params.id`, Click `click_trans_id`) — not a client-supplied header.

## 7. WebSocket Hubs & Notification Fan-out

- All hubs expose `Hub.Broadcast(room, payload)` → local fan-out → Redis Pub/Sub relay to peer pods.
- **Fail-open**: Redis Pub/Sub failure MUST NOT panic or return an error. Log + increment `ws_pubsub_failures_total` + continue serving local subscribers. Degraded cross-pod is always preferred over a crashed pod.
- **Auth first**: resolve the JWT (or signed query-string token for native clients) BEFORE `Upgrader.Upgrade`. Bind `(role, supplier_id, home_node_id)` into the connection context. Unauthenticated reads are banned.
- **Heartbeats**: 30s read deadline, 15s ping cadence. Reap dead sockets synchronously.
- **Rooms**: one room per scoped audience (`supplier:<id>`, `driver:<id>`, `retailer:<id>`, `warehouse:<id>`). Never broadcast to "all".

## 8. Auth Gates & Multi-Tenant Scoping

- `auth.RequireRole([]string{...}, handler)` is the outermost gate. Use `RequireRoleWithGrace` only for grace-windowed token rotation.
- `claims.ResolveSupplierID()` — never read `supplier_id` from the request body for authorization. Spoofing via body is a P0.
- `auth.ResolveHomeNode(claims)` → `(homeNodeType, homeNodeId)` for DRIVER / WAREHOUSE_ADMIN / FACTORY_ADMIN paths.
- `auth.GetFactoryScope(claims)` / `auth.RequireWarehouseScope(claims)` for node-scoped handlers.
- Service-to-service calls (AI Worker → backend): `X-Internal-Key` header, constant-time compared against `INTERNAL_API_KEY` env var.

## 9. Spatial Discipline (H3)

- **Resolution 7** everywhere. Do not introduce other resolutions without a documented reason.
- **Proximity queries**: `h3.GridDisk(cell, k)` + `WHERE H3Cell IN UNNEST(@cells)` against `Idx_<Table>_ByH3Cell`. Never `ST_Distance` full scans. Never raw Haversine in a hot path.
- **Tashkent timezone**: `proximity.TashkentLocation` is the canonical zone for all business-hours / shift calculations. Do NOT build `time.LoadLocation("Asia/Tashkent")` ad hoc — the helper caches it.
- **Dispatch constants**: `TetrisBuffer = 0.95` (bin-pack safety margin), Nearest-Neighbor route ordering, LIFO loading stack. Changing any of these requires an architectural verification pass.

## 10. Cross-Role Sync Protocol (the "fan-out checklist")

When a feature lands for a role, walk every client in that role's row (§1):

1. **API client**: endpoint & request/response shape in `packages/api-client` (or per-app local client).
2. **Shared types**: `packages/types`, Swift `Codable`, Kotlin `@Serializable`, TS interfaces — JSON tags match the backend exactly.
3. **View model / repository**: each client's data layer fetches + maps the new field or state.
4. **UI surfaces**: the feature renders on EVERY client in the row, or is feature-flagged uniformly if rollout is staggered.
5. **Navigation parity**: new portal page → mobile equivalent (or explicit "manage on desktop" handoff + deep-link).
6. **WebSocket / push coverage**: every client subscribes to the same hub room OR receives the same FCM/APNs channel.
7. **Offline / reconnect**: mobile caches locally + reconciles on reconnect; web handles WS drop with toast + auto-retry.
8. **Version-gated wire**: additive fields for backward compat; removals only after the oldest deployed client has been updated one release ahead.

Acceptable to ship one client first **only** when: backend contract is back-compat, feature is flagged, a tracked deadline exists for the un-updated client, and the surface is hidden or labeled "coming soon" on the un-updated client.


## 11. Local & Physical-Device Testing

Backend binds `0.0.0.0:$BACKEND_PORT` — LAN-reachable out of the box. Client-side dev-endpoint resolution is standardized per platform:

| Client | Override mechanism | Fallback |
|---|---|---|
| Next.js portals (`admin-portal`, `warehouse-portal`, `factory-portal`, `retailer-app-desktop`) | `NEXT_PUBLIC_API_URL` + `NEXT_PUBLIC_WS_URL` in `.env.local` | `http://localhost:8080` |
| Expo (`payload-terminal`) | `EXPO_PUBLIC_API_URL` in `.env` | `__DEV__ ? localhost : prod` |
| iOS (`driverappios`, `retailer-app-ios`, `warehouse-app-ios`, `factory-app-ios`) | Scheme env var `LAB_DEV_HOST` (Edit Scheme → Run → Arguments → Environment Variables). Accepts bare host, `host:port`, or full URL. | `http://localhost:8080` |
| Android (`*-android`) | `dev.host=…` in `local.properties` → `BuildConfig.API_BASE_URL` | `10.0.2.2` (AVD loopback) |
| Backend | N/A (binds `0.0.0.0`) | listens on `$BACKEND_PORT` |

**Bootstrap**:
```
make env-up          # Kafka + Redis + Spanner emulators
make spanner-init    # create instance + database + apply DDL
make seed            # deterministic test matrix
make run-backend     # native Go (binds :8080)
make dev-ip          # prints Mac LAN IP
make dev-devices     # prints per-client recipe with your LAN IP pre-filled
```

**Simulator path** (iOS Simulator / Android AVD): leave all overrides unset — fallbacks resolve to `localhost` / `10.0.2.2`, both of which reach the Mac's `:8080`.

**Physical-device path** (driver phone on same Wi-Fi as Mac):
1. `make dev-ip` → copy the LAN IP (e.g. `192.168.1.42`).
2. iOS: Xcode → Scheme → Run → Environment Variables → `LAB_DEV_HOST=192.168.1.42`. Rerun.
3. Android: `echo 'dev.host=192.168.1.42' >> apps/<app>-android/local.properties` → Gradle re-reads on next build.
4. Expo: `cp apps/payload-terminal/.env.example apps/payload-terminal/.env` and set `EXPO_PUBLIC_API_URL=http://192.168.1.42:8080`.
5. Next.js portals (if accessed from a phone browser): set `NEXT_PUBLIC_API_URL` to the LAN URL in `.env.local`.

Never hardcode LAN IPs in source — only in gitignored `local.properties`, `.env.local`, or scheme env vars.

## 12. Testing Layers

- **Unit**: `go test ./<package>/... -count=1`. Use mocks for Spanner/Kafka writers where possible.
- **Spanner-emulator tests**: run via the `test-with-spanner` skill (sets `STORJ_TEST_SPANNER=run:spanner_emulator`). Required for packages that exercise real DDL (`metabase`, integration harnesses).
- **E2E (Playwright)**: `npm run test:e2e:<project>` from repo root. Projects: `admin-portal`, `retailer-desktop`, `factory-portal`, `warehouse-portal`, `api`, `cross-role`. Always run the role-specific project for targeted fan-out checks and `cross-role` before declaring a multi-surface feature done.
- **Mobile**: iOS `xcodebuild test`, Android `./gradlew test`. Scheme env vars (`LAB_DEV_HOST`) propagate into test runs, so tests can target the local backend.
- **AI Worker**: `cd apps/ai-worker && go test ./... -count=1`.

Before declaring a phase complete: `go build ./...` + `go vet ./...` + the relevant domain tests + gap-hunter sweep.

## 13. Anti-Patterns (Reject on Sight)

- `writer.WriteMessages(...)` from a mutating handler for a state-change event. → Use `outbox.EmitJSON` inside the txn.
- `supplierID := req.SupplierID` used for authorization. → Use `claims.ResolveSupplierID()`.
- `time.LoadLocation("Asia/Tashkent")` ad hoc. → `proximity.TashkentLocation`.
- `ST_Distance` or raw Haversine in a query hot path. → `h3.GridDisk` + `WHERE H3Cell IN UNNEST(@cells)`.
- New event type without a `notification_dispatcher` switch case + formatter + TS/Swift/Kotlin mirror. → Run §4c.
- New handler without `idempotency.Guard` + `cache.Invalidate` + structured `slog` with `trace_id`. → Apply §3.
- New package-level global for a Spanner/Kafka/Redis client. → Add to `*bootstrap.App`.
- Hardcoded `http://192.168.x.x:8080` in source. → Use platform override (§11).
- `log.Printf` in new code. → `slog.InfoContext(ctx, …)` with `trace_id`.
- A feature that renders on one client in a role's row but not another, without a feature flag and a tracked deadline. → Run §10 before shipping.
- `main.go` > 200 lines. → Extract to a domain or `*routes` package.

## 14. Feature Implementation Checklist

Before declaring any feature done, walk this list top to bottom:

- [ ] **Schema**: new columns in `spanner.ddl` + model struct + DTO + migration applied via `make spanner-init`.
- [ ] **Handler**: six-step shape (§3). Auth gate, method gate, txn, outbox, cache invalidate, slog with trace_id.
- [ ] **Event**: payload struct + constant in `kafka/events.go` + dispatcher case + formatter (§4c).
- [ ] **Route**: mounted in the correct `*routes` package with `RequireRole` + `idempotency.Guard` + `priorityGuard`.
- [ ] **TS mirror**: `packages/types/ws-events.ts` discriminated-union entry.
- [ ] **Mobile mirrors**: Swift `Codable` (all iOS apps in the role's row), Kotlin `@Serializable` (all Android apps).
- [ ] **WS subscription**: correct hub + room; every client in the role's row subscribes.
- [ ] **UI surfaces**: rendered on every client in the role's row (or feature-flagged with a tracked deadline).
- [ ] **States**: loading, empty, offline, stale, permission-restricted handled on every surface.
- [ ] **Cache keys**: declared as helpers; invalidated on mutation.
- [ ] **Tests**: unit for the handler; E2E for the role; gap-hunter sweep for contract drift.
- [ ] **Build**: `go build ./...` + `go vet ./...` clean at repo root.
- [ ] **Dev-loop verified**: feature works in simulator AND on a physical device with `LAB_DEV_HOST` / `dev.host` set.
- [ ] **Docs-in-code**: doc.go in the owning package updated if the package's contract changed; no README edits unless the user asked.

## Related Skills
- `efficient-code` — per-line quality gates (index-backed reads, bounded goroutines, circuit breakers).
- `gap-hunter` — post-change drift sweep (contract, dead code, schema, wire-shape, enforcement).
- `test-with-spanner` — how to run Spanner-emulator-backed unit tests.
- `swiftui-pro` — iOS-specific review discipline.

## Source Material (cite when ambiguous)
- `.github/gemini-instructions.md` — F.R.I.D.A.Y. protocol + role matrix + backend topology.
- `pegasus/.coderabbit.yaml` — review invariants (Rule of 25, H3 discipline, outbox, SupplierId scoping).
- `apps/backend-go/schema/spanner.ddl` — schema of record.
- `apps/backend-go/kafka/events.go` — event type catalog.
- `apps/backend-go/outbox/relay.go` — outbox relay contract + "no direct WriteMessages" rule.
- `apps/backend-go/supplier/doc.go`, `apps/backend-go/warehouse/doc.go`, etc. — per-package patterns.
