# Backend Parity — A3 DRIVER
> **POINT-IN-TIME SNAPSHOT (2026-08-12) — do not treat as current status.**
> Re-verify any claim against live code before acting. Multiple ecosystem hardening phases have shipped since this audit.


**Date:** 2026-08-12  
**Tree:** `pegasusX` only  
**Phase:** Backend Class A audit (no implementation)  
**Scope:** `apps/backend-go/driver`, `driverroutes`, `deliveryroutes`, `telemetry`, `telemetryroutes`, order edges (arrive, QR, cash, shop-closed, credit leave, offline idempotency).  
**Clients later:** `driver-app-android`, `driver-app-ios` only.  
**Protocol:** [`BACKEND_PARITY_PROTOCOL.md`](./BACKEND_PARITY_PROTOCOL.md)

---

## 1. Feature inventory (route → service → Class A)

Legend for Class A columns: **Auth** | **Idem** | **Spanner RW** | **Outbox** | **Cache** | **Realtime** | Status.

| Method | Path | Handler | Package | Auth | Idem | Spanner | Outbox | Cache | Realtime | Status |
|--------|------|---------|---------|------|------|---------|--------|-------|----------|--------|
| POST | `/v1/auth/driver/login` | `HandleDriverLogin` | driver | public | — | read/demo | — | — | — | Demo/Firebase login (`auth_login.go`) |
| GET | `/v1/driver/profile` | `HandleProfile` | driver | DRIVER | — | R | — | — | — | Read |
| GET | `/v1/driver/history` | `HandleHistory` | driver | DRIVER | — | R/mem | — | — | — | Read |
| GET | `/v1/driver/earnings` | `HandleEarnings` | driver | DRIVER | — | R | — | — | — | Read |
| GET/PATCH/POST | `/v1/driver/availability` | `HandleAvailability` | driver | DRIVER | optional | **Y** | **Y** in-txn | **Y** | DriverHub+SupplierHub+fleet WS | **PASS** |
| POST | `/v1/driver/ops/rescue/request` | `HandleRescueRequest` | driver | DRIVER | **N** | **Y** | **Y** `RESCUE_REQUESTED` | N | via Kafka only if relayed | **P1** no idem |
| POST | `/v1/driver/ops/rescue/respond` | `HandleRescueRespond` | driver | DRIVER | **N** | **Y** + order reassign | **Y** reassign+RESCUE_* | N | via outbox | **P1** no idem; money/route impact |
| GET | `/v1/driver/pending-collections` | `HandlePendingCollections` | driver | DRIVER | — | R | — | — | — | Read |
| GET | `/v1/driver/open-fiscal` | `HandleOpenFiscal` | driver | DRIVER | — | R | — | — | — | Soft-freeze banner |
| GET | `/v1/driver/manifest-gate` | `HandleManifestGate` | driver | DRIVER | — | R | — | — | — | Issues `offline_nonce` |
| GET | `/v1/driver/manifest`, `/v1/fleet/manifest` | `HandleManifest` | driver | DRIVER | — | R | — | — | — | Offline hashes + nonce |
| GET | `/v1/fleet/orders` | `HandleFleetOrders` | driver | DRIVER | — | R | — | — | — | Fail-closed 503 if query nil |
| GET | `/v1/fleet/route/{routeID}/geometry` | `HandleRouteGeometry` | driver | DRIVER | — | R | — | — | — | Read |
| POST | `/v1/fleet/driver/depart` | `HandleDriverDepart` | driver | DRIVER | optional | via `DepartFn` | via manifest | **Y** manifest key | DriverHub+SupplierHub `MANIFEST_DISPATCHED` | **PASS** when wired; **P0 stub** if `depart==nil` |
| POST | `/v1/fleet/driver/return-complete` | `HandleDriverReturnComplete` | driver | DRIVER | optional | via `ReturnCompleteFn` | via manifest | **Y** | hubs + avail event | **PASS** when wired; fiscal+cash recon gates |
| GET | `/v1/driver/supply-transfers` | warehouse | FACTORY home | — | R | — | — | — | Read |
| POST | `/v1/driver/supply-transfers/{id}/arrive` | warehouse | FACTORY home | **N** | **Y** state only | **N** | N | warehouse WS only | **P0 silent** no outbox |
| POST | `/v1/fleet/route/reorder` | order | DRIVER | optional | **Y** | **Y** `ROUTE_REORDERED` | N | geometry refresh | **PASS** |
| POST | `/v1/fleet/route/request-early-complete` | order | DRIVER | optional | **Y** | **Y** | — | — | See supplier_ops |
| GET | `/v1/orders/{orderID}` | `HandleOrderGet` | driver | DRIVER | — | R | — | — | Ownership fail-closed 404 |
| **PATCH** | **`/v1/orders/{orderID}/state`** | **`HandleOrderStatePatch`** | **driver** | DRIVER | optional | **N** | **N** | **N** | sibling WS only | **P0 SILENT** |
| POST | `/v1/order/validate-qr` | order `HandleValidateQR` | order | DRIVER | **N** | R | N | N | N | Read-only token check |
| POST | `/v1/order/amend` | order | DRIVER | optional | **Y** | **Y** | **Y** | via after | Amend |
| POST | `/v1/fleet/orders/{orderID}/reassign-handshake` | order | DRIVER | optional | **Y** | **Y** | — | — | Handshake |
| POST | `/v1/driver/orders/{orderId}/shop-closed` **and** `/v1/delivery/shop-closed` | order | DRIVER | optional | **Y** | **Y** `SHOP_CLOSED` | order+list | hubs | **PASS** (prox soft) |
| POST | `/v1/driver/orders/{orderId}/partial-offload` **and** `/v1/delivery/partial-offload` | order | DRIVER | optional | **Y** | **Y** partial+returns | order | hubs | **PASS**; GPS required |
| POST | `/v1/driver/orders/{orderId}/credit-leave` | order | DRIVER | optional | **Y** | **Y** status+leg | order | after invalidates | **PASS**; stable leg key |
| POST | `/v1/delivery/proximity-unlock` | order | DRIVER | optional | **Y** | **Y** unlock | order | hubs | **PASS**; 100 m / H3 |
| POST | `/v1/delivery/bypass-offload` | order | DRIVER | optional | **Y** | **Y** bypass | order | — | **PASS** |
| POST | `/v1/delivery/credit-delivery` | order | DRIVER | optional | **Y** | **Y** | order | after | **P1** unstable payment leg key |
| POST | `/v1/delivery/missing-items` / `exception-report` | order | DRIVER | optional | **Y** via amend | **Y** | order | after | **PASS** (via amend path) |
| POST | `/v1/delivery/split-payment` | order | DRIVER | optional | **Y** | **Y** | order | — | Money path |
| POST | `/v1/delivery/confirm-payment-bypass` | order | DRIVER | optional | **Y** | **Y** | order | — | Bypass settle |
| POST | `/v1/ws/ack` | driver | DRIVER | N | N | N | N | N | No-op ack (OK) |
| GET/POST | `/v1/user/notifications*` | driver | DRIVER | — | R | — | — | — | Inbox |
| POST | `/v1/delivery/arrive` | order `HandleMarkArrived` | orderroutes | DRIVER | optional | **Y** | **Y** `ORDER_STATUS_CHANGED` | after | Kafka→hubs | **PASS**; **no GPS** |
| POST | `/v1/order/deliver` | order `HandleSubmitDelivery` | orderroutes | DRIVER | optional | **Y** | status+settlement+payment | after | Kafka | **PASS**; QR+optional geo |
| POST | `/v1/order/confirm-offload` | order | orderroutes | DRIVER | optional | **Y** | **Y** | after | Kafka | **PASS** |
| POST | `/v1/order/complete` | order | orderroutes | DRIVER | optional | **Y** | fiscal path | after | Kafka | **PASS**; GPS required |
| POST | `/v1/order/collect-cash` | order | orderroutes | DRIVER | optional | **Y** | fiscal+cash events | after | Kafka | **PASS Class A**; **P0** leg idem key |
| POST | `/v1/sync/batch` | order | orderroutes | DRIVER | batch+per-order | via SubmitDelivery | via SubmitDelivery | after | Kafka | **PASS** offline handoff only |
| POST | `/v1/delivery/scan-qr` | order | orderroutes | DRIVER | **N** | **Y** | **Y** | after | Kafka | **P1** no HTTP idem |
| POST | `/v1/delivery/verify-handshake` | deliveryroutes | DRIVER | **N** | **N write** | **N** | N | N | **P1** verify-only success |
| POST | `/v1/delivery/update-order-during-delivery` | deliveryroutes | DRIVER | **N** | **N** | **N** | N | N | **P0 SILENT success** |
| POST | `/v1/telemetry/location` | telemetryroutes | DRIVER | N (high freq) | Redis last-loc | **throttled** `DRIVER_LOCATION_UPDATED` | N | TelemetryHub + RetailerHub approach | **PASS** (intentional bus throttle) |
| CRUD | `/v1/drivers*`, `/v1/vehicles*` | driver | ADMIN/WH/FACTORY | — | **Y** | **Y** | — | — | Admin surface, not mobile |

**Route mounts (evidence):**

- `driverroutes/routes.go:48–134` — driver auth + protected DRIVER/FACTORY_DRIVER + admin CRUD.
- `orderroutes/routes.go:45–69` — primary money/delivery edges (arrive, deliver, cash, complete, sync, scan-qr).
- `deliveryroutes/routes.go:21–29` — handshake + mid-delivery update.
- `telemetryroutes/routes.go:115–129` — location POST.

**Bootstrap wiring (depart/return not stubs in prod Spanner):**

- `bootstrap/bootstrap.go:1012–1045` wires `manifestStore.DepartDriver` / `ReturnDriver`.
- `runtime_workers.go:254–259` wires `SpannerLocationBusEmitter` when Spanner present.

---

## 2. Gaps (P0 / P1 / P2) with file:line

### P0 — money / silent state / data integrity

| ID | Finding | Evidence | Impact |
|----|---------|----------|--------|
| D-P0-1 | **`PATCH /v1/orders/{orderID}/state` is a silent mutation.** Returns new state without Spanner update or outbox. Only side effect: optional `OTHER_TRUCK_ON_WAY` WS for split siblings when `orderGet` hits. | `driver/mobile_compat.go:369–429`; mounted `driverroutes/routes.go:92` | Client believes state advanced; supplier/retailer never see transition; fiscal/payment skipped. |
| D-P0-2 | **`POST /v1/delivery/update-order-during-delivery` returns success with no Spanner write / outbox.** Geofence checked then no-op body. | `order/delivery_handshake.go:82–110`; `deliveryroutes/routes.go:28` | Silent “success” contract for mobile mid-stop edits. |
| D-P0-3 | **CollectCash payment leg IdempotencyKey is non-stable** (`cash-{orderID}-{newID}`). Retry after commit but before HTTP response can mint a second CAPTURED cash leg. | `order/service.go:2169–2178` | Double cash capture / money corruption. Contrast credit-leave stable key at `driver_edges.go:307`. |
| D-P0-4 | **`HandleCreditDelivery` payment leg uses `credit-leave-{orderID}-{newID}`** (unstable) vs `HandleCreditLeave` stable `"credit-leave-"+orderID`. Dual credit paths with inconsistent money idempotency. | Unstable: `order/driver_edges.go:503–511`; Stable: `order/driver_edges.go:299–307` | Double credit legs / AR if dual endpoints used. |
| D-P0-5 | **Supply transfer arrive: Spanner state `ARRIVED` with no outbox.** Only warehouse hub broadcast. | `warehouse/supply_transfer_driver.go:102–121` | Factory/warehouse consumers miss durable event on TopicRealtime/Main. Class A hole for factory-driver edge. |
| D-P0-6 | **Depart stub when `DepartFn` nil:** returns `{"status":"departed"}` without manifest/order transitions. | `driver/mobile_compat.go:113–118` | Mitigated in Spanner bootstrap (`bootstrap.go:1019–1028`); still fails Class A for api-only / nil-repo modes. |
| D-P0-7 | **Return-complete stub when `ReturnCompleteFn` nil:** in-memory availability flip only, no Spanner/outbox. | `driver/mobile_compat.go:255–264` | Same mitigation as depart; documented graceful degradation is Class A violation if ever served. |

### P1 — realtime / incomplete transitions / contract gaps

| ID | Finding | Evidence | Impact |
|----|---------|----------|--------|
| D-P1-1 | **MarkArrived requires no GPS / geofence.** Any assigned driver can set ARRIVED remotely. | `order/service.go:1767–1784`; `HandleMarkArrived` `2775–2819` only sends `order_id` | Spoofed arrival unlocks shop-closed / offload UI without presence. Approach geo is 500 m elsewhere. |
| D-P1-2 | **Shop-closed proximity is soft:** `ensureProximityUnlocked` error discarded. | `order/shop_closed.go:207–211` | Report shop-closed far from stop if lat/lng zeroed (falls back to order coords at `181–184`) or unlock fails. |
| D-P1-3 | **HTTP idempotency is optional** (`guardIdempotency` no-ops when header/store missing). Money mutators do not use `guardIdempotencyStrict`. | `order/idempotency_guard.go:48–51`, `96–103`; cash/deliver handlers use non-strict | Double-submit without client key is only partially protected by status NoChange, not payment legs. |
| D-P1-4 | **`HandleDeliveryScanQR` / `HandleValidateQR` lack idempotency guard** (scan mutates to AWAITING_PAYMENT). | `order/service.go:2591–2714` | Concurrent scans race; NoChange helps after first commit. |
| D-P1-5 | **Rescue request/respond have no idempotency.** Accept path bulk-reassigns orders. | `driver/rescue.go:14–265` | Double-accept reassign races. |
| D-P1-6 | **`VerifyHandshake` is verify-only** (no session/proof row, no outbox). Plain-token branch empty. | `order/delivery_handshake.go:53–67` | Contract incomplete vs PoD expectations. |
| D-P1-7 | **Credit leave AR invoice open is post-commit fail-open** (log only). | `order/driver_edges.go:353–365`, `546–558` | Order DELIVERED_ON_CREDIT without AR invoice if OpenFromCreditLeave fails. |
| D-P1-8 | **Cash collect AR pay-down fail-open.** | `order/service.go:2186–2196` | Payment captured; AR bookkeeping miss. |
| D-P1-9 | **Dead mobile_compat stubs** (deliver/QR/cash/arrive) still present; not mounted when OrderService wired, but risk if re-routed. | `driver/mobile_compat.go:324–438` | Fake 200 responses if wired by mistake. |
| D-P1-10 | **afterOrderMutation cache keys are retailer/supplier lists only** — not `order:{id}` (credit/shop-closed use separate `invalidateOrderCache`). | `order/service.go:3159–3168` vs `shop_closed.go:813–817` | Stale order detail cache depending on path. |
| D-P1-11 | **Partial offload best-effort second Apply** for `PartialDelivery` flag outside main txn. | `order/partial_offload.go:319–327` | Flag can diverge from line math. |
| D-P1-12 | **Return-complete availability change** broadcasts `DRIVER_AVAILABILITY_CHANGED` on hubs but durable avail may only be in-memory unless ReturnCompleteFn also writes Drivers. | `driver/mobile_compat.go:281–307` | Hub/DB skew if manifest return doesn’t patch OnShift. |

### P2 — polish / dead code / naming

| ID | Finding | Evidence |
|----|---------|----------|
| D-P2-1 | Negotiate mounted but product-disabled 410. | `driverroutes/routes.go:104–105` |
| D-P2-2 | Location bus event IDs use wall-clock nanos (not UUID) — fine for InsertOrUpdate but collision window tiny. | `telemetryroutes/bus_emitter.go:54–56` |
| D-P2-3 | In-memory availability map still consulted when Spanner reader fails. | `driver/service.go:345–356` |
| D-P2-4 | Demo fleet / demo login env gates remain. | `mobile_compat.go:620`; `auth_login.go` |

---

## 3. Event / consumer matrix

| Event type | Producer | Outbox topic (stored) | Domain dual-route | Consumer fanout | Rooms |
|------------|----------|----------------------|-------------------|-----------------|-------|
| `DRIVER_AVAILABILITY_CHANGED` | availability patch, return-complete broadcast, admin update driver | TopicMain → Dispatch domain | `events/topic_routing.go:124` | Notification dispatcher + immediate fleet WS | `driver:{id}`, `supplier:{id}`, warehouse fleet |
| `DRIVER_LOCATION_UPDATED` | telemetry POST (throttled bus) | TopicRealtime | `topic_routing.go:127` | Dispatcher + twin | WS first: `telemetry:driver:{id}`, `telemetry:supplier:{id}` (`telemetryroutes/routes.go:380–385`) |
| `DRIVER_APPROACHING` / `DELIVERY_ARRIVING` | telemetry next-stop geo | **not outbox** (direct WS) | — | Retailer/Telemetry hubs | `retailer:{id}` (`routes.go:168–198`) |
| `ORDER_STATUS_CHANGED` | arrive, deliver, offload, cash, complete, credit leave… | TopicMain → Orders | `topic_routing.go:103–118` | Dispatcher → DriverHub/Retailer/Supplier | `driver:{id}` etc. (`kafka/notification_dispatcher.go:116`, `728–729`) |
| `SHOP_CLOSED` (+ response/escalate/resolve/timeout/bypass) | shop_closed handlers | TopicMain → Orders | same | Dispatcher + immediate `broadcastShopClosed` | supplier/retailer/driver (`shop_closed.go:797–810`) |
| `PROXIMITY_UNLOCKED` | unlock + ensureProximityUnlocked | TopicMain → Orders | same | Dispatcher + hubs | same |
| `PARTIAL_OFFLOAD` + `SUPPLIER_RETURN_CREATED` | partial offload | TopicMain | Partial → Orders | hubs + replan async | driver/retailer/supplier |
| `CREDIT_LEAVE` / `CREDIT_DELIVERY_MARKED` | credit-delivery path | TopicMain | CreditLeave on Orders domain | Dispatcher | — |
| `SETTLEMENT_REQUIRED` / `PAYMENT_REQUIRED` | deliver / scan-qr / offload | TopicMain → Orders | same | payment + hubs | — |
| Cash variance / fiscal request | CollectCash EmitExtra | TopicMain | Orders/Finance | fiscal worker | — |
| `ROUTE_REORDERED` | fleet reorder | TopicMain → Dispatch | `topic_routing.go:124` | dispatch consumers | — |
| `RESCUE_REQUESTED` / `RESCUE_ACCEPTED` / `ORDER_REASSIGNED` | rescue handlers | TopicMain | reassign → Orders domain | Dispatcher | — |
| `MANIFEST_DISPATCHED` / `MANIFEST_COMPLETED` | depart / return-complete | **direct WS** (plus manifest package may outbox separately) | Manifest* → Dispatch | hubs immediate | driver + supplier |
| Supply transfer ARRIVED | warehouse arrive | **none** | `EventSupplyTransferApproaching` type used only in WS payload | warehouse hub only | `warehouse:{id}` |

**Location bus Class A path (intentional hybrid):**

```
POST /v1/telemetry/location
  → LastLocations (Redis) best-effort
  → emitLocationToBus (5s throttle) → OutboxEvents TopicRealtime  [bus_emitter.go:28–51]
  → TelemetryHub.Broadcast full fidelity  [routes.go:163–166]
  → optional DELIVERY_ARRIVING / DRIVER_APPROACHING WS
```

Relay dual-writes domain topics per `DomainTopicForEventType`; location stays realtime (`topic_routing.go:127`).

---

## 4. Edge-case matrix

| Edge | Server expectation | Pass? | Evidence |
|------|-------------------|-------|----------|
| **Offline handoff sync** | Batch signature = hash(public token); only AWAITING_PAYMENT path; forbid COMPLETED/FISCALIZING offline; per-order idem `sync-batch:{order}:{sig}`; BypassGeofence | **Y** | `order/sync_batch.go:133–161` |
| **Offline QR token** | Manifest gate/manifest returns `offline_nonce`; validate `sha256(nonce+orderID)` or nonce | **Y** | `driver/service.go:742`, `863–869`; `order/service.go:3357–3372` |
| **Offline manifest hashes** | SHA-256 of delivery tokens per order | **Y** | `driver/service.go:810–838` |
| **Idempotency-Key replay** | Same body → stored response; mismatch → 409 | **Y when key present** | `order/idempotency_guard.go:74–91`; `driver/idempotency_guard.go:34–58` |
| **Client timestamp skew >5m** | Reject unless `offlineQueuedAt` set | **Y** | `order/idempotency_guard.go:54–70` |
| **Telemetry accuracy / recordedAt** | Fail closed accuracy>100 m or skew>5 m | **Y** on partial/credit-leave | `order/proximity.go:44–52` |
| **Settlement proximity (cash/credit)** | Unlock required or live ≤100 m / H3 res 9; outer approach 500 m on cash | **Y** cash | `proximity_settlement.go:22–28`, `service.go:2054–2084` |
| **GPS fail-closed complete** | lat/lng required | **Y** | `service.go:1918–1922`, `3443–3447` |
| **GPS fail-closed collect cash** | required geofence | **Y** | `service.go:2077–2084`; test `service_test.go:865+` |
| **GPS fail-closed arrive** | none | **N (gap)** | MarkArrived no Precheck geo |
| **GPS fail-closed shop-closed** | soft | **Partial** | `shop_closed.go:207–211` |
| **Driver ownership** | Subject must equal Order.DriverID; empty assignment → assignment_required | **Y** | `service.go:2305–2310` |
| **Order get IDOR** | Non-owner → 404 | **Y** | `order_get_ownership_test.go` |
| **Double arrive** | Status NoChange when already ARRIVED | **Y** | `transitionDriverOrder` `2245–2252` |
| **Fiscal hard-gate shift end** | return-complete blocked if open fiscal | **Y** | `mobile_compat.go:217–235` |
| **Cash recon gate** | optional flag blocks return | **Y** | `mobile_compat.go:238–252`; test `cash_recon_gate_test.go` |
| **Shop-closed max retries** | cancel after 3 | **Y** | `shop_closed.go:230–231`, `294–300` |
| **Credit leave AR disabled** | fail closed | **Y** | `driver_edges.go:265–270`; money_path_gate_test |
| **Money covers delivery** | AssertMoneyCoversDelivery on cash/complete | **Y** | `service.go:1980`, `2166` |
| **ConfirmOffload no geofence** | none | Documented weaker than cash | `service.go:1869–1881` |
| **SubmitDelivery BypassGeofence** | offline batch sets true | intentional for offline | `sync_batch.go:151–155` |

---

## 5. Class A checklist summary (driver mutators)

| # | Check | Driver verdict |
|---|--------|----------------|
| 1 | Auth scope | **Mostly pass.** Role gates on routes; order transitions enforce assigned driver. Login uses demo env when Firebase off. |
| 2 | Idempotency | **Partial.** Guards optional; cash/credit-delivery legs non-stable; rescue/scan-qr weak. |
| 3 | Spanner RW | **Most money edges pass.** Failures: state patch, update-during-delivery, (stub depart/return). |
| 4 | Outbox in-txn | **Core lifecycle pass.** Failures: state patch, update-during-delivery, supply-transfer arrive, WS-only approach events. |
| 5 | Cache after commit | **Partial.** Availability + invalidateOrderCache on many paths; afterOrderMutation list keys only. |
| 6 | Realtime | **Pass** via Kafka dispatcher + selective direct hubs (availability, shop-closed, telemetry). |
| 7 | Edge cases | Offline handoff solid; GPS inconsistent across arrive vs cash; dual credit endpoints. |
| 8 | Tests | Unit: availability outbox/WS, cash geofence, mark arrived, ownership, cash recon gate. E2E markers in `cmd/ssmr-smokecheck/e2e_driver.go`. |

---

## 6. Silent mutations (explicit list)

1. **`HandleOrderStatePatch`** — HTTP 200 state echo, **zero Spanner** (`driver/mobile_compat.go:369–429`).  
2. **`UpdateOrderDuringDelivery`** — HTTP 200 after geofence only (`order/delivery_handshake.go:107–110`).  
3. **`ArriveSupplyTransfer`** — Spanner write **without outbox** (`warehouse/supply_transfer_driver.go:102–106`).  
4. **Depart / return-complete nil-fn stubs** — success without durable transition (`driver/mobile_compat.go:113–118`, `255–264`).  
5. **Dead stubs** `HandleOrderDeliver|ValidateQR|ConfirmOffload|Complete|CollectCash|DeliveryArrive` — always-OK JSON (`mobile_compat.go:324–438`); not mounted when OrderService wired (`driverroutes/routes.go:87–115` panics if OrderService nil).  
6. **`HandleWSAck`** — intentional no-op (`mobile_compat.go:76–87`).  
7. **`VerifyHandshake`** — intentional verify-only (no write); still incomplete PoD (`delivery_handshake.go:63–67`).

---

## 7. PoD / QR / cash / credit — server contract (for clients)

| Flow | Endpoint(s) | Preconditions | Proof / GPS | Post-state | Events |
|------|-------------|---------------|-------------|------------|--------|
| Arrive | `POST /v1/delivery/arrive` | DRIVER owns order, IN_TRANSIT | **none** | ARRIVED | `ORDER_STATUS_CHANGED` |
| Proximity unlock | `POST /v1/delivery/proximity-unlock` | ARRIVED… pre-fiscal | ≤100 m or H3 or force token; telemetry age ≤2 m | ProximityUnlockedAt set | `PROXIMITY_UNLOCKED` |
| QR validate | `POST /v1/order/validate-qr` | ownership | token vs stored / offline | no change | none |
| QR scan / deliver | `POST /v1/delivery/scan-qr`, `/v1/order/deliver` | assignment | token (+ offline nonce); deliver optional geo unless coords sent | AWAITING_PAYMENT | status + SETTLEMENT/PAYMENT_REQUIRED |
| Offline batch | `POST /v1/sync/batch` | signature | token hash; **no fiscal offline** | AWAITING_PAYMENT | same as deliver |
| Confirm offload | `POST /v1/order/confirm-offload` | transition ok | **no geo** | AWAITING_PAYMENT | same |
| Collect cash | `POST /v1/order/collect-cash` | payment states | unlock **or** 100 m + 500 m outer; amount | FISCALIZING | payment+fiscal+variance |
| Complete (card) | `POST /v1/order/complete` | AWAITING_PAYMENT/credit | lat/lng required | FISCALIZING | card leg + settle |
| Shop closed | shop-closed routes | ARRIVED; photo required | accuracy if GPS present; **prox soft** | SHOP_CLOSED_PENDING | `SHOP_CLOSED` |
| Credit leave | `/credit-leave` | ARRIVED; AR invoices on; GPS Validate 100 m | ensureProximityUnlocked hard | DELIVERED_ON_CREDIT | status (+ AR open post) |
| Credit delivery | `/credit-delivery` | ARRIVED or shop-closed pending | requireProximityUnlocked; photo | DELIVERED_ON_CREDIT | CREDIT_LEAVE + CREDIT_DELIVERY_MARKED |
| Partial offload | partial-offload | ARRIVED… | location Validate + unlock | lines/totals; returns | PARTIAL_OFFLOAD |

---

## 8. Proposed fixes (audit only — do not implement here)

1. **Remove or hard-wire `HandleOrderStatePatch`** to real `transitionDriverOrder` / forbid arbitrary state strings; delete silent path.  
2. **Implement or 501** `UpdateOrderDuringDelivery`; never return success without txn+outbox.  
3. **Stable payment leg keys:** `cash-{orderID}` (or `cash-{orderID}-{attempt}` from client idem key); align credit-delivery with `credit-leave-{orderID}`.  
4. **Supply transfer arrive:** emit `EventSupplyTransferApproaching` (or ARRIVED) in same Spanner txn as state update.  
5. **Fail closed when Depart/ReturnComplete nil** (503) instead of fake OK.  
6. **GPS on arrive** (optional product decision): require approach geofence or proximity unlock before ARRIVED.  
7. **Shop-closed:** make proximity hard or require prior unlock.  
8. **`guardIdempotencyStrict` on money mutators** (cash, complete, credit, deliver, scan-qr).  
9. **Delete dead mobile_compat order stubs** to prevent future mis-mount.  
10. **Rescue:** idempotency + SupplierID on outbox row map consistency.  
11. **Credit leave:** same-txn AR open or compensating outbox command (no fail-open debt hole).  
12. **afterOrderMutation:** always invalidate `order:{id}`.

---

## 9. Package map (absolute paths)

| Area | Path |
|------|------|
| Routes | `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/driverroutes/routes.go` |
| Delivery routes | `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/deliveryroutes/routes.go` |
| Order driver mounts | `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/orderroutes/routes.go` |
| Telemetry | `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/telemetryroutes/routes.go`, `bus_emitter.go` |
| Driver service | `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/driver/service.go`, `mobile_compat.go`, `rescue.go`, `idempotency_guard.go` |
| Order edges | `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/order/service.go`, `driver_edges.go`, `shop_closed.go`, `partial_offload.go`, `proximity_settlement.go`, `sync_batch.go`, `delivery_handshake.go` |
| Hubs | `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/kafka/notification_dispatcher.go` |
| Events | `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/events/events.go`, `topic_routing.go` |

---

## 10. Bottom line

Driver **money and delivery lifecycle** on `orderroutes` + `order.Service.transitionDriverOrder` is largely Class A compliant (Spanner RW + in-txn outbox + cache after + Kafka→DriverHub). Telemetry Class A is intentionally hybrid (full WS + throttled outbox `DRIVER_LOCATION_UPDATED`).

**Blocking Class A failures for A3:** silent `PATCH .../state`, silent mid-delivery update, unstable cash/credit-delivery payment leg keys, supply-transfer arrive without outbox, and optional-idempotency on money endpoints. Arrive without GPS and soft shop-closed proximity are product/security P1s before treating PoD as fail-closed end-to-end.

**Clients (android/ios) must send:** `Idempotency-Key` on all mutators; GPS with accuracy ≤100 m and fresh `recordedAt`/`client_timestamp` for cash/credit/partial; offline `offlineQueuedAt` when replaying; never trust `PATCH /orders/{id}/state` for real transitions until fixed.
