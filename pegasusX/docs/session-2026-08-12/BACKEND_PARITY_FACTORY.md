# Backend Parity — A5 FACTORY_ADMIN
> **POINT-IN-TIME SNAPSHOT (2026-08-12) — do not treat as current status.**
> Re-verify any claim against live code before acting. Multiple ecosystem hardening phases have shipped since this audit.


**Date:** 2026-08-12  
**Agent:** A5-FACTORY  
**Tree:** `pegasusX` only  
**Phase:** Backend Class A audit (no implementation)  
**Packages:** `factory`, `factoryroutes`, `manifest` (parallel store), `auth/factory_scope.go`  
**Bridge:** `payload` / `payloaderoutes` loading-bay (shared event types; separate Spanner tables)  
**Clients later:** `factory-portal`, `factory-android`, `factory-ios`

---

## 1. Feature inventory (route → service → Class A)

### 1.1 Auth / bootstrap (public)

| Method | Route | Handler | Mutator? | Class A | Notes |
|--------|-------|---------|----------|---------|-------|
| POST | `/v1/auth/factory/login` | `HandleFactoryLogin` | issues JWT | N/A | Spanner staff lookup or demo phone; Role always `FACTORY` + optional `SupplierRole` |
| POST | `/v1/auth/factory/register` | `HandleFactoryRegister` | yes | Partial | Spanner user insert; JWT home node |
| POST | `/v1/auth/factory/refresh` | `HandleFactoryRefresh` | refresh | N/A | |
| POST | `/v1/factory/setup` | `HandleFactorySetup` | **yes** | **FAIL** | Spanner RW for `Factories` / `SupplierUsers` **without outbox** |

Evidence: routes `factoryroutes/routes.go:29-32`; setup silent write `factory/setup.go:117-123`.

### 1.2 Loading-bay (factory + payload roles)

Mounted under `loadingBayRoles = {FACTORY, FACTORY_ADMIN, ADMIN, PAYLOAD}` + `RequireFactoryScope`  
(`factoryroutes/routes.go:34-39`, `77-88`).

| Method | Route | Handler | Spanner RW + outbox | Idem | Cache inv | WS local | Kafka consumer | Class A |
|--------|-------|---------|---------------------|------|-----------|----------|----------------|---------|
| GET | `/v1/factory/manifests` | `HandleManifests` | read (memory/demo) | — | — | — | — | Read |
| GET | `/v1/factory/manifests/{manifestID}` | `HandleManifestDetail` | read | — | — | — | — | Read |
| POST | `…/start-loading` | `HandleManifestStartLoading` → `handleManifestTransition` | via `apply` + `MANIFEST_LOADING_STARTED` | optional key | yes | FactoryHub+SupplierHub | `handleManifestEvent` | **PASS*** |
| POST | `…/seal` | `HandleManifestSeal` | `MANIFEST_SEALED` | optional | yes | yes | yes | **PASS*** |
| GET | `/v1/factory/manifest-exceptions` | `HandleManifestExceptions` | memory list | — | — | — | — | Read (demo) |

\*PASS only when `SpannerRepository` is wired. `inMemoryRepository.RunTx` is a **no-op** (see P0).  
\*Data plane is **bootstrap `factoryNodeID`**, not JWT home node (see P0).  
\*Local WS room is `factory:`+`s.factoryNodeID` (`service.go:719-720`).

### 1.3 Ops (factory roles only)

`factoryRoles = {FACTORY, FACTORY_ADMIN, ADMIN}` + `RequireFactoryScope`  
(`factoryroutes/routes.go:42-75`, `77`, `90-97`).

| Method | Route | Handler | Outbox event(s) | Idem | Class A |
|--------|-------|---------|-----------------|------|---------|
| POST | `/v1/factories` | `HandleCreateFactory` | `FACTORY_CREATED` | **no** | Partial |
| GET | `/v1/factories/{factoryId}` | `HandleGetFactory` | — | — | Read (entity scope) |
| PUT | `/v1/factories/{factoryId}` | `HandleUpdateFactory` | `FACTORY_LOCATION_UPDATED` | **no** | Partial |
| GET | `/v1/factories` | `HandleListFactories` | — | — | Read (supplier scope) |
| GET | `/v1/factory/analytics/overview` | `HandleAnalyticsOverview` | — | — | Read |
| GET | `/v1/factory/dashboard` | `HandleDashboard` | — | — | Read (demo-shaped) |
| GET | `/v1/factory/profile` | `HandleProfile` | — | — | Read |
| GET/PATCH | `/v1/factory/ops/location` | `HandleOpsLocation` | PATCH: `FACTORY_LOCATION_UPDATED` in-txn | PATCH yes | **PASS** (Spanner required) |
| GET | `/v1/factory/transfers` | `HandleTransfers` GET | — | — | Read |
| POST | `/v1/factory/transfers/create` | `HandleTransfers` POST | **nil emit** | yes | **FAIL** silent state |
| GET | `/v1/factory/transfers/{transferID}` | `HandleTransferByID` | — | — | Read |
| POST | `/v1/factory/transfers/{transferID}/transition` | `HandleTransferTransition` | **nil emit** | yes | **FAIL** silent state |
| GET | `/v1/factory/fleet` | `HandleFleet` | — | — | Read |
| GET | `/v1/factory/fleet/live-map` | `HandleFactoryFleetLiveMap` | — | — | Read |
| POST | `…/manifests/{id}/dispatch` | `HandleManifestDispatch` | `MANIFEST_DISPATCHED` | optional | **PASS*** |
| POST | `…/manifests/{id}/complete` | `HandleManifestComplete` | `MANIFEST_COMPLETED` | optional | **PASS*** |
| POST | `/v1/factory/manifests/rebalance` | `HandleManifestRebalance` | `MANIFEST_REBALANCED` (or inject on cross) | optional | **PASS*** / Partial cross |
| POST | `/v1/factory/manifests/cancel-transfer` | `HandleManifestCancelTransfer` | `MANIFEST_ORDER_EXCEPTION` (+ DLQ) | optional | **PASS*** |
| POST | `/v1/factory/manifests/cancel` | `HandleManifestCancel` | `MANIFEST_CANCELLED` | optional | **PASS*** |
| POST | `/v1/factory/manifest-exceptions/{id}/resolve` | `HandleResolveManifestException` | **none** | **no** | **FAIL** silent |
| GET | `/v1/factory/fleet/drivers` | `HandleFleetDrivers` | — | — | Read |
| GET | `/v1/factory/fleet/vehicles` | `HandleFleetVehicles` | — | — | Read |
| GET/POST | `/v1/factory/staff` | `HandleStaff` | POST memory only | **no** | **FAIL** (create silent / non-durable) |
| GET | `/v1/factory/staff/{staffID}` | `HandleStaffDetail` | — | — | Read |
| POST | `/v1/factory/dispatch` | `HandleDispatch` | `MANIFEST_DRAFT_CREATED` | optional | **PASS*** |
| GET | `/v1/factory/supply-requests` | `HandleSupplyRequests` | — | — | Read |
| GET | `…/supply-requests/{id}/fulfill-options` | `HandleSupplyRequestFulfillOptions` | — | — | Read |
| POST | `…/supply-requests/{requestID}/accept` | `HandleAcceptSupplyRequest` | `SUPPLY_REQUEST_ACCEPTED` | optional | **PASS** (Spanner path) |
| PATCH | `/v1/factory/supply-requests/{id}` | `HandleSupplyRequestTransition` | Spanner: `SUPPLY_REQUEST_UPDATE` / fulfill transfers | optional | **PASS** Spanner; **FAIL** memory path |
| GET | `/v1/factory/ws-session` | `WSSessionHandler` | — | — | WS ticket |
| GET | `/v1/factory/pulse` | pulse package | — | — | Read (`pulseroutes`) |

Orphan (not mounted in `factoryroutes`): `HandleTransferDriverUpdate` (`ios_compat.go:97`) — mutator with `emit=nil`.

### 1.4 Class A status legend

| Status | Meaning |
|--------|---------|
| **PASS*** | Happy path Class A when Spanner repo is live; caveats (bootstrap factory pin, optional idem key, dual local WS) still open |
| **Partial** | Outbox present but missing idempotency and/or scope/WS correctness |
| **FAIL** | Silent Spanner/memory write, no outbox on state machine, or broken apply path |
| Read | Non-mutating |

---

## 2. FactoryHub / realtime plane

### 2.1 Hub construction & subscription

| Piece | Evidence |
|-------|----------|
| Hub name `"factory"` | `bootstrap/bootstrap.go:702`, `service.go:995` (tests) |
| Injected into factory service | `bootstrap/bootstrap.go:958-962` |
| Injected into notification dispatcher | `bootstrap` / `kafka/notification_dispatcher.go:30`, `:769-770` |
| Client subscribe | `ws/handler.go:228-241` `subscribeFactoryAdminRooms` → `factory:{HomeNodeID}` and `factory:{SupplierID}` |
| Supplier derivative | `ws/handler.go:183-184` only if `SupplierRole == FACTORY_ADMIN` |

### 2.2 Local post-commit broadcast (same process)

```705:721:apps/backend-go/factory/service.go
func (s *Service) broadcastFactoryEvent(ctx context.Context, eventType string, data map[string]any) {
	// ...
	if s.supplierHub != nil {
		s.supplierHub.Broadcast(ctx, "supplier:"+s.resolveSupplierScope(ctx), payload)
	}
	if s.factoryHub != nil {
		s.factoryHub.Broadcast(ctx, "factory:"+s.factoryNodeID, payload)
	}
}
```

**Gap:** room uses **process bootstrap** `s.factoryNodeID` (`FACTORY_DEMO_ID` / `"factory-demo-1"`, `bootstrap/bootstrap.go:914-916`), **not** JWT `HomeNodeID` / `auth.FactoryScope`. Multi-factory tenants can join `factory:{their-home-node}` and never receive local fanout if IDs differ.

### 2.3 Kafka → multi-pod fanout

| Event prefix / type | Dispatcher | Rooms |
|---------------------|------------|-------|
| `MANIFEST_*` | `handleManifestEvent` (`notification_dispatcher.go:495-517`) | supplier, `factory:{FactoryID\|SupplierID}`, warehouse, driver, **payload:{SupplierID}**, FCM/inbox |
| `FACTORY_CREATED` | `handleFactoryCreated` (`:597-607`) | supplier + factory |
| `FACTORY_LOCATION_UPDATED` | `handleFactoryLocationUpdated` (`:474-492`) | supplier + factory |
| `SUPPLY_REQUEST_*` / `FACTORY_SUPPLY_*` | `handleSupplyRequestEvent` | supply family |
| `WAREHOUSE_TRANSFER_*` (fulfill) | `handleTransferEvent` | transfer family |

Domain topic routing: manifest family → `TopicDispatch` (`events/topic_routing.go:120-123`).

---

## 3. Loading-bay start / seal events (Class A loop)

### 3.1 Factory path (this role)

| Step | Evidence |
|------|----------|
| Transition | DRAFT→LOADING (`START_LOADING`), LOADING→SEALED (`SEAL`) `service.go:992-999`, `646-697` |
| Spanner tables | `FactoryTruckManifests`, `FactoryInternalTransfers` via `repository_spanner.go:128-262` |
| Outbox in same RW txn | `apply` → `RunTx` buffers `OutboxEvents` (`repository_spanner.go:61-105`, `service.go:1060-1067`) |
| Event types | `MANIFEST_LOADING_STARTED`, `MANIFEST_SEALED` (`events/events.go:179,185`) |
| Local WS | `broadcastFactoryEvent` after commit (`service.go:1082-1094`) |
| Relay consumer | `NotificationDispatcher` prefix `MANIFEST_` → factory + **payload** rooms (`notification_dispatcher.go:194-195,495-515`) |

### 3.2 Payload path (A6 bridge — parallel plane)

| Item | Factory loading-bay | Payload terminal |
|------|---------------------|------------------|
| Routes | `/v1/factory/manifests/…/start-loading\|seal` | `/v1/payloader/manifests/…` (+ supplier aliases) |
| Roles | FACTORY*, ADMIN, **PAYLOAD** | PAYLOAD, ADMIN |
| Spanner | **`FactoryTruckManifests`** | **`SupplierTruckManifests`** + `ManifestOrders` |
| Events | same type strings | same type strings |
| WS hubs | FactoryHub + SupplierHub | PayloadHub + SupplierHub (+ driver on seal) |

Evidence: factory tables `repository_spanner.go:133-134`; payload tables `payload/repository_spanner.go:142,214`.  
**They are not the same row store.** Payload calling factory start/seal mutates factory truck manifests only. Factory cannot seal warehouse/supplier truck manifests via factoryroutes.

### 3.3 Dual emit / dual room

Both paths emit `MANIFEST_LOADING_STARTED` / `MANIFEST_SEALED` to outbox `TopicMain` then domain-routed; dispatcher fans to factory **and** payload. Correct for multi-role awareness; **incorrect** if clients assume shared mutable state between tables.

---

## 4. Cross-role matrix (who can call what)

| Surface | FACTORY / FACTORY_ADMIN | PAYLOAD | ADMIN | Notes |
|---------|-------------------------|---------|-------|-------|
| Factory auth/login/setup | public | public | public | unauthenticated entry |
| Loading-bay list/detail/start/seal/exceptions list | ✓ | ✓ | ✓ | `loadingBayRoles` |
| Factory ops (dispatch, rebalance, cancel, supply, staff, CRUD factories, location) | ✓ | ✗ | ✓ | `factoryRoles` only |
| Payload board / inject / seal-all / reassign | ✗ | ✓ | ✓ | `payloaderoutes` |
| Factory WS session | ✓ | ✗ | ✓ | factory role group |
| `RequireFactoryScope` pins home node | FACTORY* only | **pass-through** | **pass-through** | `auth/factory_scope.go:35-37` |

### 4.1 Scope enforcement details

```27:62:apps/backend-go/auth/factory_scope.go
// RequireFactoryScope pins factory staff to their JWT home node (factory id).
// RoleFactory / RoleFactoryAdmin only; other roles next.ServeHTTP without scope.
// Rejects query factory_id != claims.HomeNodeID for those roles.
```

| Check | Status |
|-------|--------|
| Body `supplier_id` for factory entity CRUD | Rejected (`RejectBodyScopeOverrides` on create/update) |
| Manifest mutations use JWT factory id | **No** — Spanner filter is repo `factoryNode` from bootstrap |
| Payload on factory loading-bay | Can invoke start/seal without factory home-node pin; still limited to bootstrap factory’s rows |
| `scopedFactoryID` for location | Prefers `GetFactoryScope`, then claims home node (`location_ops.go:174-189`) — **better than manifest path** |

---

## 5. Silent mutations & apply seam

### 5.1 Confirmed silent / broken writers

| # | Mutation | Evidence | Severity |
|---|----------|----------|----------|
| S1 | `inMemoryRepository.RunTx` never runs `fn` or `emit` | `repository.go:48-49` returns `nil` | **P0** api-only / memory fallback: HTTP 200 with **no state change and no outbox** |
| S2 | `POST /v1/factory/setup` Spanner write, no outbox | `setup.go:117-123` | **P0** onboarding state machine silent |
| S3 | Transfer create `emit=nil` | `service.go:896-913` | **P1** (or P0 if transfers drive money/stock later) |
| S4 | Transfer transition `emit=nil` | `ios_compat.go:250-280` | **P1** state machine silent |
| S5 | Exception resolve memory-only, no outbox/idem | `service.go:1855-1903` | **P1** |
| S6 | Staff create memory-only | `service.go:1156-1191` | **P2** / P1 if HR audit required |
| S7 | Supply transition memory path: update memory, **no** `UpdateSupplyRequestState`/outbox | `ios_compat.go:483-505` | **P1** |
| S8 | `HandleTransferDriverUpdate` (unmounted) `emit=nil` | `ios_compat.go:121-137` | **P2** dead code |
| S9 | Cross-manifest rebalance outbox type `MANIFEST_ORDER_INJECTED` but WS type `MANIFEST_REBALANCED` | `rebalance_cross.go:46-87` | **P1** contract split |

### 5.2 Good paths (not silent when Spanner live)

| Mutation | Pattern |
|----------|---------|
| Manifest lifecycle (start/seal/dispatch/complete) | `apply` + in-txn outbox + cache inv + local WS |
| Dispatch plan | `MANIFEST_DRAFT_CREATED` |
| Rebalance / cancel transfer / cancel manifest | outbox + WS + cache |
| Location PATCH | `spannerutils.RunReadWriteTransaction` + outbox + WS |
| Create/Update factory entity | repo RW + outbox + cache |
| Accept / transition supply (Spanner) | RW + outbox; fulfill also inventory/lots + transfer events |

### 5.3 Apply architecture risk (not silent, but corruption)

`apply` loads **all** factory manifests/transfers into process memory, mutates, then `SaveManifest`/`SaveTransfer` **every** row (`apply.go:15-51`). Concurrent requests can clobber each other (last-writer wins on full set). Spanner txn isolation does not fix lost updates across overlapping rewrites.

`manifest.Store.CommitFactory` exists as a cleaner batch API (`manifest/store.go:199-225`) but **factory service does not use it** — dual maintenance surface.

---

## 6. Gaps P0 / P1 / P2 (with file:line)

### P0

1. **Memory `RunTx` no-op** — mutators report success without mutate/outbox.  
   `apps/backend-go/factory/repository.go:48-49`  
   Gated by `ensureMemoryFallbackAllowed` in bootstrap (`bootstrap.go:945-951`) but still a Class A hole for any allowed memory mode.

2. **Factory data plane pinned to bootstrap node, not JWT home node**  
   - Repo: `NewSpannerRepository(..., factoryNodeID)` `bootstrap.go:941`  
   - SQL `WHERE FactoryId = @fid` uses `tx.factoryNode` `repository_spanner.go:133-134,205-206`  
   - Events/WS: `s.factoryNodeID` `service.go:720,741`  
   Multi-factory under one supplier: wrong factory data / wrong room / possible cross-tenant ops relative to login home node.

3. **Setup Spanner write without outbox**  
   `factory/setup.go:117-123` — creates/updates `Factories` and assignment with no `FACTORY_CREATED` / `FACTORY_LOCATION_UPDATED`.

### P1

4. **Payload on factory loading-bay without factory scope** — role pass-through `auth/factory_scope.go:35-37`; intentional bridge but no warehouse/factory linkage check on manifest id.

5. **Separate tables for factory vs payload seals** — same event names, different aggregates; clients can believe Class A loop is shared state when it is dual planes.

6. **Silent transfer create/transition** — `service.go:913`, `ios_compat.go:280`.

7. **Exception resolve silent** — `service.go:1888-1903`.

8. **Supply memory transition silent** — `ios_compat.go:497-505`.

9. **Cross-manifest rebalance event type mismatch** — outbox inject vs WS rebalanced `rebalance_cross.go:46-87`.

10. **Idempotency optional** — `idempotency_guard.go:36-38` no-ops if header missing; Class A expects guard on public mutators (clients may omit).

11. **Create/Update factory entity lack idempotency** — `crud_handlers.go:18-66,99-145`.

12. **Local WS room ≠ JWT subscribe room** when `HomeNodeID != FACTORY_DEMO_ID` — `service.go:720` vs `ws/handler.go:231-240`. Multi-pod Kafka path uses event `FactoryID` (also bootstrap pin) so both local and relay can be wrong for non-demo factories.

13. **Full-table rewrite concurrency** in `apply` — `apply.go:39-50` + `SaveManifest` all rows.

### P2

14. **Staff create non-durable** — `service.go:1181-1191`.

15. **Orphan** `HandleTransferDriverUpdate` not routed.

16. **Login Role always `FACTORY`** even when `SupplierRole` is admin (`auth_login.go:143-155`) — works with routes that allow both; confusing for clients checking `role` only.

17. **`manifest` package unused by factory service** for commits — dead dual path risk.

18. **Features doc** mentions `/v1/factory/pulse` via pulse routes (OK) and replenishment insights via warehouse routes — factory `HandleReplenishmentInsights` is demo stub (`ios_compat.go:335-358`) and not mounted on factoryroutes.

---

## 7. Event / consumer matrix

| Event | Emitter (factory) | Outbox topic field | Domain route | Consumers / fanout |
|-------|-------------------|--------------------|--------------|--------------------|
| `MANIFEST_DRAFT_CREATED` | `HandleDispatch` | TopicMain | Dispatch | Supplier, Factory, Payload, WH, Driver |
| `MANIFEST_LOADING_STARTED` | start-loading | TopicMain | Dispatch | same |
| `MANIFEST_SEALED` | seal | TopicMain | Dispatch | same |
| `MANIFEST_DISPATCHED` | dispatch | TopicMain | Dispatch | same |
| `MANIFEST_COMPLETED` | complete | TopicMain | Dispatch | same |
| `MANIFEST_REBALANCED` | rebalance | TopicMain | Dispatch | same |
| `MANIFEST_ORDER_INJECTED` | cross rebalance outbox only | TopicMain | Dispatch | same (WS local uses REBALANCED) |
| `MANIFEST_ORDER_EXCEPTION` | cancel-transfer | TopicMain | Dispatch | same |
| `MANIFEST_DLQ_ESCALATION` | cancel-transfer if escalated | TopicMain | Dispatch | same |
| `MANIFEST_CANCELLED` | cancel | TopicMain | Dispatch | same |
| `FACTORY_CREATED` | CreateFactory | TopicMain | (parity) | Supplier + Factory |
| `FACTORY_LOCATION_UPDATED` | UpdateFactory / location PATCH | TopicMain | | Supplier + Factory |
| `SUPPLY_REQUEST_ACCEPTED` | accept | TopicMain | supply handler | supply fanout |
| `SUPPLY_REQUEST_UPDATE` | PATCH transition Spanner | TopicMain | supply handler | |
| `FACTORY_SUPPLY_REQUEST_UPDATE` | local WS envelope only (accept/transition broadcast) | — local | — | not always outboxed as this type |
| `WAREHOUSE_TRANSFER_CREATED` / `RECEIVED` | fulfill Spanner | TopicMain | transfer handler | WH/factory/supply |

Local post-commit WS does **not** replace Kafka for multi-pod; both are used (local immediate + relay). FCM/inbox via `broadcastFactory` (`notification_dispatcher.go:765-773`).

---

## 8. Edge-case matrix

| Edge | Behavior | Covered? |
|------|----------|----------|
| Double start-loading | `invalid_state` 409 if not DRAFT | Unit + e2e lifecycle |
| Seal non-LOADING | 409 `invalid_state` | same |
| Idempotent rebalance already assigned | 200 `already_assigned`, no extra outbox | `service_test.go` rebalance suite |
| Cancel already cancelled transfer | 200 `already_cancelled` | tests |
| Cancel completed manifest | 409 | code path |
| Transfer ledger/route mismatch on rebalance | 409, no outbox | tests |
| Idempotency key replay | returns stored body when key present | rebalance/cancel/supply/location tests |
| Missing Idempotency-Key | mutation proceeds | by design; Class A gap |
| Concurrent apply on same factory | last full rewrite wins | **not** covered / risk |
| Memory backend | success without write | **not** guarded in handlers |
| JWT factory A vs bootstrap factory B | sees B’s data | **not** rejected |
| Payload seal factory manifest | allowed on factoryroutes; writes FactoryTruck* | intentional bridge |
| Factory seal SupplierTruckManifest | not via factory handlers | separate plane |
| Supply fulfill internal mode | auto RECEIVED + inventory credit in txn | `supply_spanner.go:300-405` |
| Supply forbidden supplier | `resolveSupplierScope` check in transition | `supply_spanner.go:225-227` |

E2E markers (smoke): `PX_E2E_FACTORY_*` in `cmd/ssmr-smokecheck/e2e_factory.go` (lifecycle, loading-bay, supply, payload-override start-loading).

---

## 9. Per-mutation checklist summary (protocol §)

| Check | Factory core lifecycle | Transfers / exceptions / staff | Setup / location | Supply Spanner |
|-------|------------------------|--------------------------------|------------------|----------------|
| 1 Auth scope | Role gate yes; **home-node data pin no** | Role yes; weak entity pin | Location yes; setup JWT | Supplier check on transition |
| 2 Idempotency | Optional header | mixed / missing resolve | location yes; setup no | optional |
| 3 Spanner RW | yes (Spanner repo) | transfer via apply; resolve no | setup/location yes | yes |
| 4 Outbox in txn | yes | **no** for transfer/resolve/staff | setup **no**; location yes | yes |
| 5 Cache invalidate | yes | transfer yes; resolve no | entity yes | no dedicated keys |
| 6 Realtime | local + Kafka | incomplete | location local+Kafka | local + Kafka |
| 7 Edge cases | good unit coverage | partial | thin | partial |
| 8 Tests | strong on rebalance/cancel/start | supply/location idem tests | — | transition idem |

---

## 10. Proposed fixes (audit only — do not implement here)

1. **Fix `inMemoryRepository.RunTx`** to execute `fn` + in-memory outbox buffer (or hard-fail mutators when Spanner nil in non-dev).  
2. **Resolve factory id per request**: `auth.EffectiveFactoryID(ctx)` → Spanner `WHERE FactoryId=@jwt` and WS room `factory:{jwt}`; keep bootstrap only as seed/demo default.  
3. **Emit outbox on setup** (`FACTORY_CREATED` / `FACTORY_LOCATION_UPDATED`) in the same RW txn as Factories write.  
4. **Transfer create/transition**: emit typed events (e.g. factory transfer state) in-txn; or document intentional deferral if purely UI scaffold.  
5. **Exception resolve**: durable flag + outbox + idempotency.  
6. **Align cross-rebalance** outbox type with WS (`MANIFEST_REBALANCED` or dual-emit both).  
7. **Require Idempotency-Key** on all public POSTs (or server-generate from body hash + actor).  
8. **Narrow apply** to single-row / batch updates (or call `manifest.Store.CommitFactory`) to kill full-table rewrite races.  
9. **Document dual loading-bay planes** for clients: factory-portal uses FactoryTruck*; payload-terminal uses SupplierTruck*; shared events only.  
10. **Payload on factory routes**: either require factory_id claim linkage or warehouse-factory mapping; do not leave unrestricted pass-through without IDOR review.  
11. **Staff create**: Spanner `SupplierUsers` or drop POST until durable.  
12. **Remove or mount** `HandleTransferDriverUpdate` consistently.

---

## 11. Role fleet pointer

| Agent | Role | Primary | Routes |
|-------|------|---------|--------|
| A5-Factory | FACTORY_ADMIN / FACTORY | factory, manifest | factoryroutes |
| A6-Payload | PAYLOAD | payload (+ factory loading-bay bridge) | payloaderoutes |

Protocol SoT: `docs/session-2026-08-12/BACKEND_PARITY_PROTOCOL.md`.  
Client SoT: `docs/FEATURES_BY_APP_ROLE.md` §4 Factory.

---

## 12. Verdict

Factory **manifest lifecycle mutators** (dispatch → start-loading → seal → dispatch → complete, rebalance, cancel*) implement the Class A spine **when Spanner is live**: JWT role gate → RW txn + outbox → relay → Kafka → FactoryHub/PayloadHub/FCM → cache invalidate, with solid unit tests.

They are **not fully Class A** for production multi-factory because:

1. Data and WS rooms are **bootstrap-scoped**, not JWT home-node-scoped.  
2. Memory fallback **silently no-ops** apply.  
3. Setup, transfers, exception resolve, staff, and memory supply paths are **silent** or non-durable.  
4. Payload bridge shares **events**, not **tables**.

**Highest priority remediation:** P0 items 1–3 (memory RunTx, factory id pin, setup outbox).
