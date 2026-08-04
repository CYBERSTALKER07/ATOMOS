# PegasusX — Capabilities: Math, Logic, Algorithms (CODE)

**SOURCE OF TRUTH: Go packages under `apps/backend-go` (not other markdown).**  
Inventory of *which* APIs exist: [FEATURES_BY_APP_ROLE.md](./FEATURES_BY_APP_ROLE.md)  
Order transitions: [ORDER_FLOW_AND_EDGE_CASES.md](./ORDER_FLOW_AND_EDGE_CASES.md)

Each block cites the implementing file/function. Money is `int64` minor units in structs (`UnitPrice` / `TotalMinor` / `*Minor` fields).

---

## 0. Shared engines

### 0.1 Credit available — `credit.Profile.Available`

**File:** `apps/backend-go/credit/types.go`

```
Available = max(0, CreditLimitMinor − CurrentBalanceMinor − ReservedMinor)
```

**Statuses** (`credit.Status`): `INACTIVE`, `ACTIVE`, `FROZEN`, `CLOSED`, `BLACKLISTED`.

**RiskTier** (`LOW|MEDIUM|HIGH|BLOCK`) exist as types; leave-on-credit **does not use them** (see §0.2).

**What / solution:** Headroom for credit orders and credit-leave without over-extension.

**Edges:** Negative headroom clamped to 0; reserved holds block double-spend across open orders.

### 0.2 Credit leave gate — `order.CanLeaveOnCredit`

**File:** `apps/backend-go/order/credit_guard.go`

Logic:

1. `profile != nil` and `profile.Status == ACTIVE`
2. `profile.Available() >= order.TotalMinor`
3. If `cfg.MaxAutoCreditMinor > 0` and `order.TotalMinor > MaxAutoCreditMinor` → reject

Comment in code: *“Credit risk scoring / RiskTier is intentionally not used (Phase A removal).”*

Call sites hardcode `MaxAutoCreditMinor: 50000000` in `retailer_shop_closed.go` and `worker_shop_closed.go`.

### 0.3 Inventory reserve — `order.ReserveLineItemsInTxn`

**File:** `apps/backend-go/order/inventory_reservation.go`

Logic:

1. Aggregate qty by SKU (duplicate lines summed)
2. Read `SupplierInventoryV2` (`QuantityOnHand`, `QuantityReserved`)
3. Require `QuantityOnHand − QuantityReserved >= quantity`
4. Write `QuantityReserved += quantity`
5. Missing SKU → `ErrInventoryExhausted`
6. Marker row `OrderStockReservationMarkers`

**Backfill:** `BackfillScheduledReservations` reserves for `MANUAL_PREORDER` in `SCHEDULED|AUTO_ACCEPTED` missing markers.

### 0.4 Inventory release / stale

**Files:** `inventory_release.go`, `inventory_stale_release.go`

Release: `QuantityReserved = max(0, QuantityReserved − qty)`.

Stale job: select `Status = PENDING` with old `UpdatedAt`, cancel via cancel path (not direct decrement) — avoids double-release with `AWAITING_PAYMENT`.

### 0.5 Dispatch volume & capacity

**Files:** `dispatch/constants.go`, `dispatch/volume.go`, `dispatch/binpack.go`, `dispatch/capacity_recommend.go`

```
TetrisBuffer = 0.95
DefaultTruckVolumeVU = 150.0
defaultUnitVolumeVU = 1.0
effectiveCap = MaxVolumeVU * TetrisBuffer
```

Order volume: sum over lines `max(qty,1) * unitVU` where unitVU from line → product catalog → default 1.0 (`VolumeSourceCatalog` vs `default_1_0`).

`SelectBestVehicle` / binpack reject when order volume exceeds `MaxVolumeVU * TetrisBuffer`. Capacity recommend unselects until under effective cap.

### 0.6 Approach geofence

**File:** `apps/backend-go/proximity/geofence.go`

```
DeliveryApproachRadiusM = 500.0
```

Wired as `deliveryGeofenceMeters` in `order/service.go`. Settlement proximity comments in service reference **100 m / H3** unlock path (`ProximityUnlockedAt` / `proximity-unlock` route).

### 0.7 Shop-closed grace

**File:** `order/service.go` `NewService`

If `ShopClosedGrace <= 0` → default **`5 * time.Minute`**.  
Bootstrap sets via `shopClosedGraceDuration()` (`bootstrap/bootstrap.go`).

Worker `worker_shop_closed.go` polls `SHOP_CLOSED_PENDING` where `ShopClosedGraceEndsAt <= now` (limit batch), then `DecideShopClosedTimeout` with `MaxAutoCreditMinor: 50000000`.

### 0.8 Fiscal constants — `order/fiscal.go`

```
FiscalOFDTimeout = 8 * time.Second
```

Cash variance (`emitCashVariance`):

```
shortfall = max(0, expected − received)
overage   = max(0, received − expected)
```

Emit `CASH_SHORTFALL` or `CASH_OVERAGE` only if either > 0.

Force-complete: roles `ADMIN` / `WAREHOUSE_ADMIN` (route in `orderroutes`); statuses handled in `ForceCompleteOrder` (FISCAL_* / reconciliation paths in fiscal.go).

### 0.9 Partial offload — `order/partial_offload.go`

Statuses: `FULL|PARTIAL|NONE|RETURNED`.  
Reasons: `DAMAGED|MISSING|SHOP_REFUSED|CAPACITY|OTHER`.

Invariant in `ApplyPartialOffloadLines`:

```
DeliveredQty + RemainingQty == Quantity  (per touched line)
delivered_minor = Σ UnitPrice × DeliveredQty
remaining_minor = Σ UnitPrice × RemainingQty
```

### 0.10 Claim window — `claims/service.go` + `order/claim_window.go`

```
DefaultPostDeliveryClaimWindow = 48 * time.Hour
```

Env override `CLAIM_WINDOW_HOURS`; supplier/warehouse policy fields snapshotted onto order at COMPLETED (`ClaimWindowHours`, `ClaimWindowEndsAt`, `ClaimWindowPolicySource`).

### 0.11 Claim pricing — `claims/pricing.go`

1. `AggregateClaimLines` merges duplicate SKUs (overflow-safe add).  
2. Order SKU index: weighted avg unit price  
   `unit = valueSum/qty` then half-up if `rem*2 >= qty` (equiv. floor((valueSum+qty/2)/qty)).  
3. `ClaimedQtyBySKU` counts prior claims in `OPEN|UNDER_REVIEW|APPROVED|RESOLVED` (not `REJECTED`).  
4. Require claim qty ≤ order qty − claimed; amount = Σ qty × unit (overflow-checked `mulInt64`/`addInt64`).

### 0.12 Claim eligibility — `claims/eligibility.go`

Requires order status `COMPLETED` (constant `OrderStatusCompleted = "COMPLETED"`). Photo-required types enforced in eligibility logic (DAMAGED family — see function body). Window from snapshot or default 48h.

### 0.13 Idempotency — `order/idempotency_guard.go`

SHA-256 body hash + Idempotency-Key headers; mismatch → conflict; used by `HandleCollectCash` among others.

### 0.14 State machine — `order/state_machine.go`

Canonical transitions only (full table in ORDER_FLOW doc). Same-status allowed. No soft ARRIVED→COMPLETED (comment ADR-009).

---

## 1. Retailer capabilities (logic tied to routes)

| Capability | What it does | Solution | Logic / math | Edges | Evidence |
|------------|--------------|----------|--------------|-------|----------|
| Checkout + reserve | Create order, hold stock | Prevent oversell | §0.3 in same txn as create | OOS → `inventory_exhausted`; empty warehouseID no-op reserve | `inventory_reservation.go`, `paymentroutes` checkout |
| Credit checkout / leave | Use trade credit | Buy without doorstep cash | §0.1–0.2 | Frozen/blacklisted; max auto 50_000_000 minor | `credit/*`, `credit_guard.go` |
| Shop-closed respond | Resolve closed shop | Unstick delivery | Routes → status via retailer_shop_closed handlers | Grace 5m default; credit leave needs ACTIVE+Available | `retailer_shop_closed.go`, `worker_shop_closed.go` |
| Preorder confirm/edit/reject | Future-dated commitment | Plan ahead of T-1 | ConfirmationStatus + SCHEDULED/AUTO_ACCEPTED → PENDING promote | Edit/cancel locks in preorder_policy/sweeper | `preorder_*.go` |
| Claims | Post-delivery redress | Fair chargeback amounts | §0.10–0.12 | Not COMPLETED; window; SKU caps; overflow | `claims/*` |
| Cart / quote | Sync cart, price quote | Cross-device cart | Sum `qty * unit_price_minor` | Stale prices | retailer cart/quote handlers |
| POS / stock / shifts | Store OS packs | In-store ops | int64 tender amounts; stock movements | Capability pack gates on JWT | `retailerroutes` stock/pos/shifts |
| Auto-order / AI | Rules + predictions | Reduce manual reorders | Settings patches + AI_PREORDER source | confirm/reject-ai routes | retailerroutes AI + auto-order |
| Confirm cash | Retailer confirms COD intent | Align with driver collect | `POST /v1/delivery/confirm-cash` RoleRetailer | Race with driver collect | `orderroutes` |

---

## 2. Supplier (`ADMIN`) capabilities

| Capability | What / solution | Logic / math | Edges | Evidence |
|------------|-----------------|--------------|-------|----------|
| Vet orders | Accept/reject before load | Reject → cancel + release inventory | Concurrent vet | `supplierroutes` vet + inventory_release |
| Dispatch preview/execute | Build routes under capacity | §0.5 TetrisBuffer / binpack / optional optimizer | Over-cap unselect; freeze locks (warehouse) | `dispatch/*`, supplier dispatch routes |
| Payment bypass | Token for doorstep | Driver confirms bypass | Scope to awaiting payment | supplier payment-bypass + driver confirm-payment-bypass |
| Claims approve/reject | Adjudicate OPEN claims | Pricing from order lines §0.11 | Double settle idempotency | orderroutes claims + claims service |
| Credit profiles/program | Set limits/terms/hold | §0.1 Available | Policy v2 flags in credit package | `creditroutes` |
| Cash recon accept/write-off | Close driver cash bags | Ledger entries | Shortfall write-off | `cashreconroutes` |
| Inventory import | CSV → apply | Session pipeline + Kafka worker | Bad rows; freeze locks | supplier inventory import + ai-worker |
| Control tower / playbooks | Scored exceptions | `/v1/control-tower/*` RoleAdmin | Portal-first UI | `controltowerroutes` |
| Negotiations | Qty negotiate resolve | Routes exist; may return disabled | Product flag | supplier negotiations routes |

---

## 3. Warehouse capabilities

| Capability | What / solution | Logic / math | Edges | Evidence |
|------------|-----------------|--------------|-------|----------|
| Dispatch hub | Depot-owned plan/execute | Same §0.5; locks `/dispatch-lock*` | Role WarehouseAdmin for most ops | `warehouseroutes` |
| Order delay/reject/overflow/propose | Reshape commitment | Status DELAYED / cancel / proposal PENDING_WAREHOUSE | Inventory release on reject | warehouse ops order routes + order package |
| Returns inbound | Physical dock | Scan/confirm sessions | Wrong WH; barcode | `returnsroutes` |
| Reverse logistics | Claim-linked receive | Task receive | **RoleWarehouse only** | `creditnoteroutes` |
| Claims adjudicate | Local approve/reject | Same claims service | WarehouseAdmin | orderroutes |
| Reassign | Move order without payload UI | Capacity check | Concurrent seal | warehouse reassign routes |
| Return policy CRUD | Policy document | Admin/WarehouseAdmin | No dedicated shell href observed | warehouseroutes return-policy |

---

## 4. Factory capabilities

| Capability | What / solution | Logic / math | Edges | Evidence |
|------------|-----------------|--------------|-------|----------|
| Manifest load/seal/dispatch/complete | Outbound correctness | Manifest mutations + outbox events | Concurrent seal | `factoryroutes` |
| Rebalance / cancel-transfer | Hot correction | Manifest mutation | After LOADED | factory manifests routes |
| Supply requests / transfers | Factory↔warehouse stock | Transfer state machine in factory package | FACTORY_DRIVER arrive via driver API | factory + driver supply-transfers |
| Insights | Demand pull view | Calls warehouse replenishment insights | Cross-package scope | warehouse insights route |

---

## 5. Payload capabilities

| Capability | What / solution | Logic / math | Edges | Evidence |
|------------|-----------------|--------------|-------|----------|
| Seal / seal-completed / seal-all | Driver-ready truck | Manifest seal transitions | seal-all registered; verify client use | `payloaderroutes` |
| Inject order | Late add | Capacity `GET /v1/payload/capacity/{vehicleID}` | Over VU | payload capacity + inject |
| Reassign | Suggest/apply truck move | Capacity + recommend | After seal | recommend/reassign routes |
| Exceptions | Cannot-load etc. | Exception rows + optional delivery exception-report | Dual paths | payload manifest-exception |
| Inbound returns | Dock | Shared returns inbound | Role PAYLOAD | returnsroutes |

---

## 6. Driver capabilities

| Capability | What / solution | Logic / math | Edges | Evidence |
|------------|-----------------|--------------|-------|----------|
| Arrive | Mark ARRIVED | Geofence §0.6 | `geofence_violation` | `POST /v1/delivery/arrive` |
| Proximity unlock | Unlock settlement | 100 m / H3 / unlock token (service comments) | `proximity_required` | proximity-unlock + service |
| QR / offload / deliver | → AWAITING_PAYMENT | ValidateStatusTransition | Replay QR | scan-qr, deliver, confirm-offload |
| Collect cash / complete | → FISCALIZING | §0.8 variance; ADR-009 | Idempotency §0.13; short cash | collect-cash, complete, fiscal.go |
| Credit leave | → DELIVERED_ON_CREDIT | §0.2 | Cannot COMPLETED until FISCALIZING | credit-leave, credit-delivery |
| Partial offload | Line qty adjust | §0.9 | Invalid status after fiscalizing | partial_offload.go |
| Shop closed | → SHOP_CLOSED_PENDING | Grace §0.7 | Timeout worker matrix | shop-closed handlers + worker |
| Bypass offload | Supervised → AWAITING_PAYMENT | Token + photo opts in TransitionOpts | Token mismatch | bypass-offload |
| Fiscal retry | FISCAL_FAILED → FISCALIZING | OFD 8s timeout | Max attempts in fiscal package | fiscal/retry |
| Offline sync | Batch upload | `/v1/sync/batch` | Clock skew checks on cash | sync + idempotency |
| Supply transfer arrive | Factory leg | Driver/FACTORY_DRIVER | Wrong transfer id | supply-transfers |
| Force-complete | N/A for driver | — | RoleAdmin/WarehouseAdmin only | orderroutes |

---

## 7. Outbox / concurrency (platform)

Mutations that change order status typically: Spanner RW txn + outbox emit (`emitOrderStatusChanged` in `order/service.go`) + version optimistic concurrency. Illegal transitions return `ErrInvalidStatusTransition`.

---

## 8. Rebuild rule

When updating this file:

1. Open the cited `.go` file.  
2. Copy formulas/constants from source.  
3. Do not import wording from other `.md` files.

Known code nuances:

- `RiskTier` types remain; `CanLeaveOnCredit` ignores them.  
- Claim prior-qty statuses ≠ residual claim money filters (see `claims/residual.go` if present).  
- Shop-closed wire status string is `SHOP_CLOSED_PENDING`.
