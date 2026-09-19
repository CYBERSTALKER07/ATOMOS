# Backend Parity — PAYLOAD (A6)
> **POINT-IN-TIME SNAPSHOT (2026-08-12) — do not treat as current status.**
> Re-verify any claim against live code before acting. Multiple ecosystem hardening phases have shipped since this audit.


**Date:** 2026-08-12  
**Tree:** `pegasusX` only  
**Phase:** Backend Class A audit (no code changes)  
**Protocol:** [`BACKEND_PARITY_PROTOCOL.md`](./BACKEND_PARITY_PROTOCOL.md)  
**Packages:** `apps/backend-go/payload`, `payloaderoutes`, `payloaderoutes` (orphan), factory loading-bay bridge in `factory` / `factoryroutes`  
**Clients (later):** `payload-terminal` (Expo), `payload-app-android`, `payload-app-ios`

> **STALE AUDIT (2026-08-12).** Do not plan from this file without a code re-verify.  
> payloaderoutes **is mounted** (`apps/backend-go/main.go` import + `RegisterRoutes`). The P0 “orphan richer package / unregistered” finding is **historical**. Dual-plane and fleet-reassign residuals may still apply — re-open those handlers. Living SoT: [`ROLE_FEATURES_DOCS_VS_CODE.md`](../ROLE_FEATURES_DOCS_VS_CODE.md).

---

## Executive summary

Payload has a **well-tested mutator spine** (start-loading, inject, exception, reassign, multi-path seal) that, under Spanner, follows the intended pattern: `apply` → RW txn + in-txn outbox → cache invalidate → PayloaderHub/SupplierHub (and driver hub on seal/reassign). Unit tests assert seam parity for exception + reassign extensively.

However, **Class A is not met end-to-end**. Critical gaps:

| Severity | Finding |
|----------|---------|
| **P0** | Live router mounts **`payloaderoutes` only**; sibling package **`payloaderoutes` is unregistered** but holds routes clients use (`/v1/payload/ws-session`, ship-units, labels). |
| **P0** | `inMemoryRepository.RunTx` is a **no-op** (`return nil` without `fn`/`emit`) — non-Spanner runs claim success without state or outbox. |
| **P0** | `POST /v1/payload/seal` with `manifest_id` **deadlocks** (holds `s.mu` then calls `apply` which re-locks); order-only seal path is a **silent mutation** (no apply/outbox). |
| **P0** | `POST /v1/fleet/reassign` mutates in-memory order routes only, **`emit = nil`**, no Spanner order writes. |
| **P0** | JWT carries `HomeNodeType=WAREHOUSE` + warehouse id, but **reads/writes are supplier-scoped only** — warehouse IDOR / cross-warehouse board leakage. |
| **P1** | Factory loading-bay and payloader manifests are **two separate services/repos** (dual plane); clients merge both HTTP surfaces. Same event *names*, different aggregates. |
| **P1** | Payload outbox `ManifestEvent`s omit **`WarehouseID` / `FactoryID`** → Kafka fanout skips warehouse room; FCM actor is supplier id. |
| **P1** | `OrderRow` status transitions live only in process memory — not in Spanner `Orders` / apply persist path. |

**Overall Class A status: FAIL** (core seal/reassign *intended* design is close under Spanner; production-critical holes block a pass).

---

## 1. Feature inventory (route → service → Class A)

### 1.1 Mounted routes (`payloaderoutes` — **live**)

Registration: [`apps/backend-go/main.go:211-217`](../../apps/backend-go/main.go) → [`apps/backend-go/payloaderroutes/routes.go`](../../apps/backend-go/payloaderroutes/routes.go).

Auth gate: `RequireRole(RolePayload, RoleAdmin)` at [`payloaderroutes/routes.go:68-81`](../../apps/backend-go/payloaderroutes/routes.go). Optional Firebase group when enabled.

| Method | Path | Handler | Mut? | Idem | Spanner RW+outbox (when Spanner repo) | Cache | Hub/WS | Class A |
|--------|------|---------|------|------|----------------------------------------|-------|--------|---------|
| POST | `/v1/auth/payloader/login` | `HandlePayloaderLogin` | issue JWT | n/a | n/a (env demo creds) | n/a | n/a | **PARTIAL** (scaffold auth) |
| POST | `/v1/auth/payloader/refresh` | `HandlePayloaderRefresh` | reissue | n/a | n/a | n/a | n/a | **PARTIAL** |
| GET | `/v1/payloader/trucks` | `HandleTrucks` | read | — | hydrate/list mem | — | — | OK read |
| GET | `/v1/payloader/orders` | `HandleOrders` | read | — | mem overlay | — | — | OK read (orders not durable) |
| GET | `/v1/payloader/manifests` | `HandleManifestsList` | read | — | hydrate | — | — | **PARTIAL** (no WH filter) |
| GET | `/v1/payloader/manifests/{id}` | `HandleManifestDetail` | read | — | hydrate | — | — | PARTIAL |
| POST | `/v1/payloader/manifests/{id}/start-loading` | `HandleStartLoading` | **Y** | header | apply + `MANIFEST_LOADING_STARTED` | yes | payload+supplier | **PARTIAL** |
| POST | `/v1/payloader/manifests/{id}/inject-order` | `HandleInjectOrder` | **Y** | header | apply + `MANIFEST_ORDER_INJECTED` | yes | payload+supplier | **PARTIAL** |
| POST | `/v1/payloader/manifests/{id}/seal` | `HandleSealManifest` | **Y** | header | apply + `MANIFEST_SEALED` + pick-wave gate | yes | payload+supplier+driver | **PARTIAL** |
| POST | `/v1/payloader/manifests/seal-all` | `HandleSealAll` | **Y** | header | per-id apply + sealed | yes | same | **PARTIAL** |
| POST | `/v1/payloader/manifests/seal-completed` | `HandleSealCompletedManifests` | **Y** | header | batch apply + sealed | yes | same | **PARTIAL** |
| POST | `/v1/payload/seal` | `HandleSeal` | **Y** | header | **broken / silent** | partial | partial | **FAIL** |
| POST | `/v1/payload/manifest-exception` | `HandleManifestException` | **Y** | header | apply + exception (+ DLQ) | yes | payload+supplier | **PARTIAL** |
| GET | `/v1/payloader/manifest-exceptions` | `HandleManifestExceptions` | read | — | mem list | — | — | OK read |
| POST | `/v1/payloader/recommend-reassign` | `HandleRecommendReassign` | soft | header | no write | — | — | OK (compute) |
| POST | `/v1/payloader/reassign-order` | `HandleApplyReassign` | **Y** | header | apply + `MANIFEST_REBALANCED` | yes | payload+supplier+driver | **PARTIAL** (strong tests) |
| POST | `/v1/fleet/reassign` | `HandleFleetReassign` | **Y** | header | apply **emit nil** | order list only | `ORDER_REASSIGNED` | **FAIL** |
| GET | `/v1/payload/capacity/{vehicleID}` | `HandleVehicleCapacity` | read | — | — | — | — | OK read |
| GET | `/v1/supplier/manifests*` | aliases → same handlers | as above | | | | | same |
| GET/POST | `/v1/user/notifications*` | inbox | soft | | | | | out of payload core |
| POST | `/v1/delivery/exception-report` | order service | **Y** | (order) | order plane | | | order audit |

### 1.2 Orphan routes (`payloaderoutes` — **not mounted**)

[`apps/backend-go/payloaderoutes/routes.go`](../../apps/backend-go/payloaderoutes/routes.go) is **never imported by `main.go`**. Extra surface vs live package:

| Path | Handler | Client use |
|------|---------|------------|
| `GET /v1/payload/ws-session` | `auth.WSSessionHandler` | **Expo** `payload-terminal/hooks/useManifestData.ts:275` |
| `GET …/ship-units` (payloader + payload) | `HandleListShipUnits` | GS1 / labels path |
| `POST …/labels` | `HandleManifestLabels` | ZPL labels |

**P0 route-mount drift:** docs say `payloaderoutes`; runtime uses `payloaderoutes`; richer package is dead code.

Native Android/iOS WS use `GET /v1/ws` with Bearer (not ticket session) — Expo terminal path is the one that depends on ws-session.

### 1.3 Adjacent role routes (PAYLOAD allowed)

| Area | Mount | Notes |
|------|-------|-------|
| Factory loading-bay | [`factoryroutes/routes.go:34-40,78-88`](../../apps/backend-go/factoryroutes/routes.go) | `loadingBayRoles` includes `RolePayload`; `RequireFactoryScope` **skips** non-factory roles ([`auth/factory_scope.go:34-37`](../../apps/backend-go/auth/factory_scope.go)) so payload JWT (warehouse home node) can enter. |
| Returns inbound | [`returnsroutes/routes.go:32`](../../apps/backend-go/returnsroutes/routes.go) | `RolePayload` allowed |
| Pulse | [`pulseroutes/routes.go:43-44`](../../apps/backend-go/pulseroutes/routes.go) | `GET /v1/payloader/pulse` |

### 1.4 Factory loading-bay bridge (payload-callable)

| Method | Path | Handler | Outbox | Realtime rooms |
|--------|------|---------|--------|----------------|
| GET | `/v1/factory/manifests` | `HandleManifests` | — | factory in-mem list |
| GET | `/v1/factory/manifests/{id}` | `HandleManifestDetail` | — | |
| POST | `…/start-loading` | `HandleManifestStartLoading` | `MANIFEST_LOADING_STARTED` + **FactoryID** | factory+supplier hubs |
| POST | `…/seal` | `HandleManifestSeal` | `MANIFEST_SEALED` + FactoryID | factory+supplier hubs |
| GET | `/v1/factory/manifest-exceptions` | list | — | |

Evidence: transition + outbox [`factory/service.go:1012-1067`](../../apps/backend-go/factory/service.go); outbox fields include `FactoryID: s.factoryNodeID` [`factory/service.go:736-747`](../../apps/backend-go/factory/service.go).

**Bridge parity (Expo + Android + iOS):** all three call factory list/detail/start-loading/seal **and** payloader seal-completed / inject / reassign. Same **backend process**, but **not the same domain model** (factory transfer manifests vs `SupplierTruckManifests`). Class A “single loop” is only name-level, not data-plane unified.

---

## 2. Class A data plane

### 2.1 Intended path (works under Spanner + real repo spy)

```
HTTP (PAYLOAD JWT)
  → payload.Service.Handle*
  → guardIdempotency (optional header)
  → Service.apply → Repository.RunTx
       → mutate in-memory overlay under lock
       → SaveManifest / SaveManifestOrder / SaveException
       → emit → OutboxEvents rows (same Spanner RW txn)
  → cache.Invalidate(payload:manifest*, orders*, exceptions*)
  → broadcastPayloadEvent → PayloaderHub room payload:{supplierID}
                         → SupplierHub room supplier:{supplierID}
  → (seal/reassign) broadcastDriverEvent → driver:{driverID}
  → Relay → Kafka TopicDispatch (MANIFEST_* via topic_routing)
  → NotificationDispatcher.handleManifestEvent
       → supplier / factory / warehouse / driver / payload rooms + FCM
```

Key files:

- Apply seam: [`payload/apply.go:9-80`](../../apps/backend-go/payload/apply.go)
- Spanner RW + outbox: [`payload/repository_spanner.go:46-90`](../../apps/backend-go/payload/repository_spanner.go)
- Hub broadcast: [`payload/service.go:479-520`](../../apps/backend-go/payload/service.go)
- WS subscribe rooms: [`ws/handler.go:199-215`](../../apps/backend-go/ws/handler.go) — `payload:{subject}` and `payload:{supplierID}`
- Kafka fanout: [`kafka/notification_dispatcher.go:495-517`](../../apps/backend-go/kafka/notification_dispatcher.go)
- Topic: `MANIFEST_*` → `TopicDispatch` [`events/topic_routing.go:120-123`](../../apps/backend-go/events/topic_routing.go)

### 2.2 PayloaderHub contract

| Path | Room | Evidence |
|------|------|----------|
| Service post-commit | `payload:{supplierID}` via `resolveSupplierScope` | [`service.go:506-510`](../../apps/backend-go/payload/service.go) |
| WS dial subscribe | `payload:{JWT.subject}` **and** `payload:{JWT.supplierID}` | [`ws/handler.go:199-214`](../../apps/backend-go/ws/handler.go) |
| Kafka dispatcher | `payload:{supplierID}` + FCM actor `supplierID` role `PAYLOAD` | [`notification_dispatcher.go:776-784`](../../apps/backend-go/kafka/notification_dispatcher.go) |
| Returns gate | also `warehouse:{id}` on PayloadHub | [`notification_dispatcher.go:800-801`](../../apps/backend-go/kafka/notification_dispatcher.go) |

**Mismatch:** frame types after mutators use dual format — `PAYLOAD_SYNC` + typed `PUSH` on payload hub; supplier hub also gets legacy `{type, data}` envelope ([`service.go:482-519`](../../apps/backend-go/payload/service.go)). Kafka fanout sends raw outbox JSON, not `PAYLOAD_SYNC`. Clients must handle both.

### 2.3 Bootstrap wiring

- Spanner: `payload.NewSpannerRepository(client, seedSupplierID, warehouseNodeID)` — **bootstrap warehouse**, not per-JWT ([`bootstrap/bootstrap.go:942`](../../apps/backend-go/bootstrap/bootstrap.go)).
- Hubs injected: SupplierHub, PayloadHub, DriverHub ([`bootstrap.go:974-989`](../../apps/backend-go/bootstrap/bootstrap.go)).
- Memory fallback: `NewInMemoryRepository()` ([`bootstrap.go:950`](../../apps/backend-go/bootstrap/bootstrap.go)).

---

## 3. Mutator deep dives (file:line)

### 3.1 Start loading — PARTIAL Class A

[`service.go:709-804`](../../apps/backend-go/payload/service.go)

| Check | Status | Evidence |
|-------|--------|----------|
| Auth scope | Role only; no warehouse pin | routes gate; no `HomeNodeID` check in handler |
| Idempotency | Yes if header + store | `guardIdempotency` L720; **no `releaseIdempotency` on error** |
| Spanner RW | Yes via apply | L731-773 |
| Outbox | `EventManifestLoadingStarted` | L767-772 |
| Cache | yes | L788 |
| Realtime | payload+supplier | L789-793 |
| Edge | draft-only; 404/409 | L775-785 |

### 3.2 Inject order — PARTIAL

[`service.go:806-940`](../../apps/backend-go/payload/service.go)

- Capacity + loading-state gates; invents order row if missing (L858-863) — **ghost order** risk.
- Outbox `MANIFEST_ORDER_INJECTED` with volume/stop counts (L892-900).
- No warehouse ownership check on target manifest.

### 3.3 Manifest exception — PARTIAL (tests strong)

[`service.go:942-1124`](../../apps/backend-go/payload/service.go); tests [`service_test.go:21-108`](../../apps/backend-go/payload/service_test.go)

- OVERFLOW escalation → second outbox `MANIFEST_DLQ_ESCALATION` (L1058-1067).
- Idempotency replay + release defer present.
- Persist exceptions to Spanner `ManifestExceptions` when repo is Spanner.

### 3.4 Recommend / apply reassign — PARTIAL (apply is best-tested)

Recommend: compute-only [`service.go:1153-1249`](../../apps/backend-go/payload/service.go).

Apply: [`service.go:1251-1592`](../../apps/backend-go/payload/service.go)

- Rich conflict matrix (capacity, mutable state, driver/route mismatch, already_assigned noop).
- Outbox `MANIFEST_REBALANCED` with from/to manifest & route (L1497-1511).
- Driver WS `MANIFEST_AMENDED` (L1574-1580).
- **Orders table not updated** — only overlay `OrderRow` + `ManifestOrders`.
- Extensive unit coverage in `service_test.go` (capacity, noop, replay, mismatch, etc.).

### 3.5 Seal paths

| Handler | Path | Class A |
|---------|------|---------|
| `HandleSealManifest` | `/v1/payloader/manifests/{id}/seal` | PARTIAL — apply+outbox+driver+SSCC [`1712-1814`](../../apps/backend-go/payload/service.go) |
| `HandleSealCompletedManifests` | `…/seal-completed` | PARTIAL — batch; pick-wave blocked rows; [`1594-1710`](../../apps/backend-go/payload/service.go) |
| `HandleSealAll` | `…/seal-all` | PARTIAL — same; clients mostly unwired ([FEATURES_BY_APP_ROLE](../FEATURES_BY_APP_ROLE.md)) |
| `HandleSeal` | `/v1/payload/seal` | **FAIL** — see §4 |

`sealManifestLocked` sets orders to `DISPATCHED` in memory only ([`service.go:436-470`](../../apps/backend-go/payload/service.go)).

Pick-wave gate: `stocklots.AssertManifestPickReady` ([`service.go:432-434`](../../apps/backend-go/payload/service.go)).

### 3.6 Fleet reassign — FAIL (silent)

[`fleet_compat.go:9-84`](../../apps/backend-go/payload/fleet_compat.go)

```go
err = s.apply(r.Context(), func() error {
    // only mutates s.orders[i].RouteID
}, nil) // emit == nil → no outbox
```

Orders are **not** in `apply` persist set (only manifests / manifestOrders / exceptions). Even Spanner path does not durable-write order route.

### 3.7 Auth / warehouse JWT

Login mints ([`auth_login.go:104-112`](../../apps/backend-go/payload/auth_login.go)):

- `Role: PAYLOAD`
- `SupplierRole: RoleWarehouseAdmin`
- `HomeNodeType: HomeNodeWarehouse`
- `HomeNodeID: warehouseID` (env demo / SSMR seed)

Refresh preserves claims ([`auth_refresh.go:48-64`](../../apps/backend-go/payload/auth_refresh.go)).

**No payload handler** calls `auth.GetWarehouseScope`, `EffectiveWarehouseID`, or filters `ListManifests` by warehouse. Spanner list:

```sql
FROM SupplierTruckManifests WHERE SupplierId = @sid
```

([`repository_spanner.go:138-143`](../../apps/backend-go/payload/repository_spanner.go)).

Write stamps bootstrap `warehouseID` if set ([`repository_spanner.go:211-213`](../../apps/backend-go/payload/repository_spanner.go)), not JWT home node.

---

## 4. Gaps P0 / P1 / P2

### P0 — money/auth/silent state / corruption-class

| ID | Gap | Evidence | Impact |
|----|-----|----------|--------|
| P0-1 | **`payloaderoutes` not registered**; `payloaderoutes` is live | `main.go:45,211` vs package `payloaderoutes/` | Expo `GET /v1/payload/ws-session` 404; ship-units/labels unreachable |
| P0-2 | **In-memory `RunTx` no-op** | `repository.go:32-34` `return nil` | Mutations appear OK (nil err) with **zero state change and zero outbox** when Spanner absent |
| P0-3 | **`HandleSeal` deadlock** with `manifest_id` | Locks `s.mu` at `service.go:1848` then `apply` re-locks `service.go:41` / RLock `:60` | Primary client seal path (`payload-terminal` `useManifestActions` → `/v1/payload/seal`) can hang |
| P0-4 | **Order-only `/v1/payload/seal` silent mutation** | `service.go:1913-1954`: mutate under mutex, **no apply**, outbox type `ORDER_DISPATCHED` (not `MANIFEST_*`) only via WS, no Kafka | Driver/order plane not durable; Class A broken |
| P0-5 | **Fleet reassign silent** | `fleet_compat.go:41-64` emit nil; orders not persisted | Native fleet reassign lies about durability |
| P0-6 | **Warehouse scope not enforced** | JWT warehouse vs supplier-wide list/mutate | Cross-warehouse IDOR for multi-WH suppliers |

### P1 — realtime / incomplete transitions / contract split

| ID | Gap | Evidence |
|----|-----|----------|
| P1-1 | Dual factory vs payload planes | Separate `factory.Service` + `payload.Service`; clients merge HTTP (`payload-terminal/api.ts`, Android `PayloadApi.kt`, iOS `APIClient.swift`) |
| P1-2 | Payload outbox omits `WarehouseID`/`FactoryID` | e.g. seal emit `service.go:1746-1755` vs factory `manifestOutboxFields` includes FactoryID |
| P1-3 | Kafka warehouse fanout empty for payload seals | `handleManifestEvent` uses `e.WarehouseID` → `broadcastWarehouse` no-ops |
| P1-4 | Order status not durable | `OrderRow` only; apply saves manifests/orders-on-manifest/exceptions only (`apply.go:62-78`) |
| P1-5 | `apply` rewrites **all** manifests every mutation | write amplification + last-writer races across pods |
| P1-6 | Idempotency stuck in-progress | Several mutators lack `defer releaseIdempotency` (start-loading, inject, seal*, reassign apply) |
| P1-7 | Inject can invent orders | `service.go:858-863` |
| P1-8 | Demo login not real staff table | `auth_login.go:44-74` env phone/PIN |
| P1-9 | Factory seal/start does not update payload overlay | bridge is HTTP dual-call, not shared txn |
| P1-10 | FCM payload actor = supplierID | `broadcastPayload` — not worker subject |

### P2 — polish / dead code

| ID | Gap | Evidence |
|----|-----|----------|
| P2-1 | Dual package names `payloaderoutes` / `payloaderoutes` | protocol table lists both; only one mounted |
| P2-2 | Production dual-write / overlay acknowledged | `service.go:41-48` comment |
| P2-3 | `HandleDeviceTokenNoop` | inbox.go — device token global platformroutes |
| P2-4 | seal-all / capacity API-only | FEATURES_BY_APP_ROLE client note |

---

## 5. Event / consumer matrix

| Event type | Emitter (payload) | Topic | Consumers / fanout | Notes |
|------------|-------------------|-------|--------------------|-------|
| `MANIFEST_LOADING_STARTED` | start-loading | Dispatch | ND handleManifestEvent → supplier, factory room, warehouse (empty), driver, payload | No warehouse_id |
| `MANIFEST_ORDER_INJECTED` | inject-order | Dispatch | same | |
| `MANIFEST_ORDER_EXCEPTION` | manifest-exception | Dispatch | same | |
| `MANIFEST_DLQ_ESCALATION` | exception overflow | Dispatch | same | |
| `MANIFEST_REBALANCED` | reassign-order | Dispatch | same + driver room via DriverID field if set | |
| `MANIFEST_SEALED` | seal* (good paths) | Dispatch | same + post-commit driver `MANIFEST_DISPATCHED` WS | SSCC after commit (best-effort) |
| `ORDER_DISPATCHED` | order-only seal | **WS only** (not outbox) | PayloaderHub only | Not in topic_routing as sealed path |
| `ORDER_REASSIGNED` | fleet reassign | **WS only** | payload hub | No outbox |
| Factory `MANIFEST_*` | factory loading-bay | Dispatch | same ND; **FactoryID** set | Parallel universe of manifests |

Relay path is shared outbox infrastructure (not re-audited here; Spine A0 owns bus).

---

## 6. Edge-case matrix

| Edge | Behavior | Covered? |
|------|----------|----------|
| Double seal same manifest | `manifest_not_sealable` / already_sealed in seal-all | Handler yes; concurrency multi-pod weak (full table rewrite) |
| Seal while not LOADED/LOADING | blocked | yes |
| Pick wave incomplete | 409 / pick_wave_blocked | yes (stocklots gate) |
| Inject over capacity | `volume_conflict` | yes |
| Inject already assigned other truck | `order_already_assigned` | yes |
| Reassign to sealed target | `target_manifest_not_mutable` / unavailable | tests yes |
| Reassign capacity exceeded | 409 + explain | tests yes |
| Reassign replay | noop depth stable | tests yes |
| Exception not loading | 409 | yes |
| OVERFLOW escalation threshold=3 | dual event | tests yes |
| Idempotency key replay | cached body | exception test; others partial |
| Idempotency no header | unguarded double-submit | by design |
| Cancel sealed truck | no payload cancel API | deferred to factory cancel (not RolePayload on cancel routes) |
| Memory backend | silent no-op success | **uncovered hole** |
| Cross-warehouse JWT | full supplier board | **uncovered hole** |
| `/v1/payload/seal` concurrent | deadlock risk | **uncovered hole** |

---

## 7. Factory ↔ payload bridge parity (Expo + native)

| Capability | Expo terminal | Android | iOS | Backend |
|------------|---------------|---------|-----|---------|
| Payloader board (trucks/orders/manifests) | yes | yes | yes | `payloaderoutes` |
| Supplier alias inject/seal | yes | yes | yes | same handlers |
| Factory list/detail/start/seal | yes | yes | yes | `factoryroutes` loading-bay + RolePayload |
| seal-completed batch | yes | yes | yes | payloader |
| recommend + reassign-order | yes | yes | yes | payloader |
| fleet reassign | (partial) | yes | yes | **silent** backend |
| `/v1/payload/seal` | yes | (varies) | (varies) | **broken path** |
| WS | ws-session ticket (Expo) vs `/v1/ws` Bearer (native) | native OK if JWT valid | native OK | Expo ticket route **unmounted** |

**Conclusion:** Client surface is intentionally dual-backend (factory + payload). Backend does **not** provide a unified Class A transaction across both. Same process, same Kafka type names, **different tables/overlays**.

---

## 8. Silent mutations checklist

| Mutation | Spanner durable? | Outbox same txn? | Verdict |
|----------|------------------|------------------|---------|
| start-loading / inject / exception / reassign-order / seal-manifest / seal-all / seal-completed | Yes if Spanner repo | Yes | OK pattern under Spanner |
| Same mutators + memory repo | **No** (RunTx no-op) | **No** | **Silent** |
| `/v1/payload/seal` order-only | No | No | **Silent** |
| `/v1/payload/seal` manifest_id | Deadlock / unreliable | intended yes | **Broken** |
| fleet reassign | No (orders) | No | **Silent** |
| Order status DISPATCHED/LOADED/PENDING | No Orders table write | No order events | **Silent** vs order plane |
| afterSealAssignSSCC | side-effect post-commit | not in seal txn | Acceptable best-effort; failures only logged |
| route geometry persist | post-commit warn | n/a | Acceptable best-effort |

---

## 9. Warehouse scope for payload JWT

| Layer | Behavior | File:line |
|-------|----------|-----------|
| Login claims | `HomeNodeWarehouse` + env warehouse id | `auth_login.go:104-111` |
| Refresh | returns `warehouse_id: claims.HomeNodeID` | `auth_refresh.go:63` |
| Middleware on payload routes | **none** (no `RequireWarehouseScope` / ops scope) | `payloaderoutes/routes.go` |
| List manifests | SupplierId only | `repository_spanner.go:140-143` |
| Save manifests | optional static bootstrap WarehouseId | `repository_spanner.go:211-213` |
| Outbox | no WarehouseID | e.g. `service.go:1746-1755` |
| Client store | Expo/Android persist warehouse for UI only | not enforced server-side |

**Required for Class A:** pin all payload reads/writes to `claims.HomeNodeID` when `RolePayload` (and/or `SupplierRole=WAREHOUSE_ADMIN` home warehouse), include `warehouse_id` on every `ManifestEvent`, and construct Spanner repo scope per request (or filter in SQL) — not process-global seed warehouse.

---

## 10. Tests inventory (payload package)

| Area | Status |
|------|--------|
| Exception outbox + hub + cache | yes (`TestHandleManifestException_EscalationSeamParity`) |
| Exception idempotency replay | yes |
| Apply reassign seam parity + many conflicts | extensive (`service_test.go`) |
| Seal-completed explain rows | yes |
| HandleSeal deadlock / order-only silent | **no** |
| Fleet reassign outbox | **no** |
| Warehouse scope IDOR | **no** |
| Memory repo no-op | **no** |
| Route mount ws-session | **no** (integration) |
| Factory bridge shared state | **no** |

---

## 11. Proposed fixes (audit only — do not implement here)

1. **Mount one routes package:** either register `payloaderoutes` instead of/in addition to `payloaderoutes`, or merge extras (`ws-session`, ship-units, labels) into the live package and delete the orphan.
2. **Fix `inMemoryRepository.RunTx`** to invoke `fn` + in-memory outbox buffer (mirror factory/tests spy), never silent success.
3. **Rewrite `HandleSeal`:** drop outer `s.mu.Lock`; always go through `apply`+`MANIFEST_SEALED`; remove or properly outbox order-only path (prefer require `manifest_id`).
4. **Fleet reassign:** delegate to `HandleApplyReassign` semantics (manifest move + outbox) or emit `MANIFEST_REBALANCED` / order route events in-txn.
5. **Warehouse pin:** middleware or handler-level `claims.HomeNodeID` filter on list/mutate; set `WarehouseID` on all payload `ManifestEvent`s from JWT; Spanner `WHERE SupplierId=@sid AND WarehouseId=@wid`.
6. **Persist order linkage:** write order route/manifest/status through Spanner (or document intentional deferral that warehouse dispatch remains SoT for order state).
7. **Idempotency:** `defer releaseIdempotency` on all mutators that guard.
8. **Bridge:** document dual planes as intentional **or** make factory loading-bay thin proxy into payload service for shared `SupplierTruckManifests` when role is PAYLOAD (single SoT).
9. **apply write set:** save only dirty manifests/orders to avoid full-table rewrite races.
10. **Tests:** deadlock seal, memory repo, warehouse IDOR, fleet outbox, mounted route smoke for ws-session.

---

## 12. Class A scorecard (role)

| Checklist item | Result |
|----------------|--------|
| 1 Auth scope (tenant/home-node, not body supplier) | **FAIL** — role gate OK; warehouse home node unused |
| 2 Idempotency on public mutators | **PARTIAL** — supported; not mandatory; incomplete release |
| 3 Spanner RW for writes | **PARTIAL** — Spanner path yes; memory no-op; orders not written |
| 4 Outbox same txn for transitions | **PARTIAL** — main mutators yes; seal order path / fleet no |
| 5 Cache invalidate after commit | **PASS** on main mutators |
| 6 Realtime dispatcher + hub/FCM | **PARTIAL** — local hub OK; Kafka warehouse/FCM incomplete; Expo ticket route missing |
| 7 Edge cases | **PARTIAL** — reassign strong; seal/scope weak |
| 8 Tests | **PARTIAL** — reassign/exception strong; P0 paths untested |

**Role verdict: Class A FAIL — do not treat PAYLOAD as production-hardened until P0-1…P0-6 are closed.**

---

## 13. Absolute paths (primary evidence)

- `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/payload/service.go`
- `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/payload/apply.go`
- `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/payload/repository.go`
- `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/payload/repository_spanner.go`
- `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/payload/auth_login.go`
- `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/payload/fleet_compat.go`
- `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/payload/service_test.go`
- `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/payloaderoutes/routes.go`
- `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/payloaderoutes/routes.go`
- `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/factoryroutes/routes.go`
- `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/factory/service.go`
- `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/ws/handler.go`
- `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/kafka/notification_dispatcher.go`
- `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/main.go`
- `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/bootstrap/bootstrap.go`
