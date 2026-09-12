# Backend Role Parity — Master Gap Register

> **POINT-IN-TIME SNAPSHOT (2026-08-12) — do not treat as current status.**
> Re-verify any claim against live code before acting. EH0 and EH1 have shipped since this audit.
> Current SoT: [`FEATURES_BY_APP_ROLE.md`](../FEATURES_BY_APP_ROLE.md) · [`ROLE_ROW_PARITY_MATRIX.md`](../ROLE_ROW_PARITY_MATRIX.md).

**Date:** 2026-08-12  
**Source:** Parallel agents A0–A7 (read-only audits)  
**Protocol:** [`BACKEND_PARITY_PROTOCOL.md`](./BACKEND_PARITY_PROTOCOL.md)  
**Per-role reports:** `BACKEND_PARITY_{SPINE,SUPPLIER,RETAILER,DRIVER,WAREHOUSE,FACTORY,PAYLOAD,PLATFORM_ADMIN}.md`

---

## 1. Fleet status

| Agent | Report | Verdict |
|-------|--------|---------|
| A0 Spine | BACKEND_PARITY_SPINE.md | Happy-path order/payment strong; money idempotency + orphan AR/payout UX events open |
| A1 Supplier | BACKEND_PARITY_SUPPLIER.md | Fleet/pricing/dispatch solid; **silent inventory** + CT + credit program lifecycle fail |
| A2 Retailer | BACKEND_PARITY_RETAILER.md | Claims/create strong; **org-id vs subject**, cash checkout theatre, silent ParentOrders |
| A3 Driver | BACKEND_PARITY_DRIVER.md | Core transition path Class A; **compat silent mutators** + unstable cash keys |
| A4 Warehouse | BACKEND_PARITY_WAREHOUSE.md | Dispatch strong; **WMS stocklots 0 outbox** (77 write sites) |
| A5 Factory | BACKEND_PARITY_FACTORY.md | Manifest lifecycle Class A under Spanner; **demo factory pin** + memory no-op |
| A6 Payload | BACKEND_PARITY_PAYLOAD.md | Seal path Class A under Spanner; **route package orphan**, seal deadlock, home-node |
| A7 Platform | BACKEND_PARITY_PLATFORM_ADMIN.md | MFA/dual-control core solid; **revoke/dunning/approve-audit** P0 |

---

## 2. Cross-role P0 (fix first — Wave B1/B2)

Deduped across agents. Same root cause merged.

| ID | Gap | Roles | Evidence (from agents) | Fix shape |
|----|-----|-------|------------------------|-----------|
| **M-P0-1** | Unstable payment leg idempotency keys embed `newID()` → double cash/credit capture risk | Spine, Driver | `order/service.go` CollectCash `cash-{orderID}-{newID}`; `driver_edges.go` credit-delivery leg | Stable keys: `cash-{orderID}`, `credit-delivery-{orderID}` (match credit-leave pattern) |
| **M-P0-2** | Silent / fake driver mutators return success without Spanner/outbox | Driver | `PATCH /v1/orders/{id}/state` mobile_compat; `POST /v1/delivery/update-order-during-delivery` | Delete or hard-fail; force clients onto `transitionDriverOrder` paths |
| **M-P0-3** | WMS/stocklots + inventory PATCH: Spanner stock writes **zero outbox** | Warehouse, Supplier | `stocklots/*` emits=0; inventory emit callbacks `return nil` | Emit `STOCK_*` / `INVENTORY_*` in same txn; dispatcher → WarehouseHub |
| **M-P0-4** | Retailer order/checkout uses `claims.Subject` as retailer id, not org | Retailer | create/checkout/cancel/card/cash | `ResolveRetailerOrgID` everywhere multi-user JWT |
| **M-P0-5** | Cash checkout ack-only / no Spanner truth | Retailer | retailer checkout cash path | Real PENDING_CASH session + outbox or remove endpoint |
| **M-P0-6** | ParentOrders insert/update silent | Retailer, Spine | multi-supplier checkout | Outbox `PARENT_ORDER_*` same txn |
| **M-P0-7** | Payload: unmounted `payloaderoutes`, seal deadlock, order-only seal silent | Payload | main mounts only one package; `service.go` mutex re-lock | Mount missing routes or delete package; fix seal locking; forbid silent seal |
| **M-P0-8** | Factory data pinned to bootstrap demo factory id, not JWT home node | Factory | bootstrap + spanner repo filters | Resolve factory from claims home-node |
| **M-P0-9** | Memory repos `RunTx` no-op success (factory/payload) | Factory, Payload | repository.go RunTx | Fail closed when Spanner nil under prod/ssmr |
| **M-P0-10** | PLATFORM_ADMIN partner key revoke broken (empty tenant) | Platform | HandleRevokeKey | PreferTenant / query tenant_id for PA |
| **M-P0-11** | Money-flag approve not fail-closed with audit | Platform | featureflags approve | Audit in same success path or roll back |
| **M-P0-12** | Dunning run-once: route allows PA, handler requires ADMIN only | Platform | admin AR dunning | Align RequireRole + handler |
| **M-P0-13** | Claim approve intermediate UNDER_REVIEW silent | Spine, Supplier | claims/service.go emit=nil | Emit CLAIM_UNDER_REVIEW or fold into final transition |
| **M-P0-14** | Payout generate trusts body supplier_id | Spine | payout/handlers.go | PreferTenantSupplierID only |
| **M-P0-15** | Supply-transfer driver arrive: write without outbox | Driver, Warehouse | warehouse/supply_transfer_driver.go | Emit transfer status event |

---

## 3. Cross-role P1 (Wave B3+)

| ID | Gap | Roles |
|----|-----|-------|
| **M-P1-1** | Orphan events (no dispatcher UX): `AR_INVOICE_*`, `PAYOUT_BATCH_*`, `REFUND_*`, `BUYER_ACCEPTANCE_*` | Spine, Retailer |
| **M-P1-2** | `api` run-mode without worker: notification consumer may start; **outbox relay does not** | All |
| **M-P1-3** | Cart / POS / store-stock mutations without outbox or RetailerHub | Retailer |
| **M-P1-4** | Control tower playbooks/runs silent | Supplier |
| **M-P1-5** | Credit program/terms lifecycle silent (profile side may emit) | Supplier |
| **M-P1-6** | AR aging Apply without outbox | Spine |
| **M-P1-7** | Arrive no GPS hard gate; shop-closed proximity soft | Driver |
| **M-P1-8** | Dual factory vs payload manifest tables (shared event names, different rows) | Factory, Payload |
| **M-P1-9** | Outbox events omit WarehouseID → WH fanout empty for some manifests | Payload |
| **M-P1-10** | ~~Reverse receive body warehouse_id~~ **FIXED Wave B7** | Warehouse |
| **M-P1-11** | MFA step-up missing on partner/match/dunning admin routes | Platform |
| **M-P1-12** | ~~JWT warehouse on payload list/mutate~~ **FIXED Wave B7** | Payload |
| **M-P1-13** | Optional (not required) idempotency on money driver mutators | Driver |
| **M-P1-14** | planning.scenario.published Kafka orphan (local WS only) | Supplier |

---

## 4. What already is Class A (do not regress)

| Loop | Evidence |
|------|----------|
| Order create → outbox ORDER_CREATED → hubs | Spine + Retailer |
| Driver transitionDriverOrder (arrive/cash/complete) RW+outbox+cache | Driver |
| Dispatch execute + freeze locks | Warehouse |
| Factory manifest start/seal/dispatch under Spanner | Factory |
| Payload seal-manifest under Spanner + PayloaderHub | Payload |
| Payment webhooks signature + idempotency + outbox | Spine |
| Claims file (org + residual + outbox) | Retailer |
| JWT jti revoke / logout | Platform/auth |
| Dual-control money flags (set path) | Platform |

---

## 5. Fix waves (backend only)

### Wave B1 — Money integrity (block double-charge / fake success)

**Status: IMPLEMENTED 2026-08-12**

| ID | Fix | Evidence |
|----|-----|----------|
| M-P0-1 ✅ | Stable keys `cash-{orderID}`, `credit-leave-{orderID}` | `order/service.go` CollectCash; `order/driver_edges.go` credit-delivery |
| M-P0-2 ✅ | Silent mutators fail-closed (501/503, not 200) | `driver/mobile_compat.go` state/collect/arrive; `order/delivery_handshake.go` mid-delivery |
| M-P0-5 ✅ | Cash checkout → `SelectCashAtDelivery` (Spanner + outbox) | `order.SelectCashAtDelivery`; `payment.HandleOrderCashCheckout`; bootstrap bind |
| M-P0-14 ✅ | Payout generate uses PreferTenantSupplierID, not body | `payout/handlers.go` |
| M-P0-13 ✅ | `CLAIM_UNDER_REVIEW` outbox on approve intermediate | `claims/service.go`; `events.EventClaimUnderReview` |
| M-P1-1 ✅ | AR + payout events in notification_dispatcher + formatters | `kafka/notification_dispatcher.go`; `notifications/formatter.go` |

Tests: `order/wave_b1_money_test.go` + packages order/payment/payout/claims/kafka/notifications/driver/bootstrap green.

**Owners:** A0 + A2 + A3 packages.

### Wave B2 — Logistics truth (stock + load + scope)

**Status: IMPLEMENTED 2026-08-13** — see [`WAVE_B2_LOGISTICS_IMPLEMENTATION.md`](./WAVE_B2_LOGISTICS_IMPLEMENTATION.md)

| ID | Fix |
|----|-----|
| M-P0-3 ✅ | WMS putaway/pick/cycle/temp + inventory qty/policy outbox; dispatcher WMS fanout |
| M-P0-7 ✅ | Mount `payloaderoutes`; seal no deadlock; order-only seal 400; fleet reassign emits |
| M-P0-8 ✅ | `resolveFactoryNode` from JWT home-node |
| M-P0-9 ✅ | Memory RunTx fail-closed in prod/ssmr |
| M-P0-15 ✅ | Supply-transfer arrive + `SUPPLY_TRANSFER_ARRIVED` outbox |
| M-P1-9 ✅ | `WarehouseID` on payload seal events when JWT home-node set |

**Owners:** A4 + A5 + A6 + A0.

### Wave B3 — Retailer multi-user + parent order bus

**Status: IMPLEMENTED 2026-08-13** — see [`WAVE_B3_RETAILER_IMPLEMENTATION.md`](./WAVE_B3_RETAILER_IMPLEMENTATION.md)

| ID | Fix |
|----|-----|
| M-P0-4 ✅ | `ResolveRetailerOrgID` on create/unified/preview/cancel/card/cash/confirm-cash + ownership gates |
| M-P0-6 ✅ | ParentOrders insert/update + `PARENT_ORDER_*` outbox; dispatcher retailer fanout |
| M-P1-3 ✅ | Cart `CART_SYNC_UPDATED` producer; POS/STORE_STOCK dispatcher cases |

**Owners:** A2 + A0.

### Wave B4 — Supplier ops truth

**Status: IMPLEMENTED 2026-08-13** — see [`WAVE_B4_SUPPLIER_IMPLEMENTATION.md`](./WAVE_B4_SUPPLIER_IMPLEMENTATION.md)

| ID | Fix |
|----|-----|
| Silent inventory ✅ | Supplier AdjustStock + inventory policy + import apply outbox (reuse B2 event types) |
| M-P1-4 ✅ | Control tower playbook/run outbox + SupplierHub dispatcher |
| M-P1-5 ✅ | Credit program + terms lifecycle outbox + dispatcher |
| M-P1-14 ✅ | `planning.scenario.published.v1` → `handlePlanningEvent` |

**Owners:** A1.

### Wave B5 — Platform admin break-glass

**Status: IMPLEMENTED 2026-08-13** — see [`WAVE_B5_PLATFORM_IMPLEMENTATION.md`](./WAVE_B5_PLATFORM_IMPLEMENTATION.md)

| ID | Fix |
|----|-----|
| M-P0-10 ✅ | Partner key revoke: PA requires tenant_type/tenant_id; admin-portal passes tenant |
| M-P0-11 ✅ | Money-flag approve reverts to PENDING if audit fails |
| M-P0-12 ✅ | Dunning run-once allows PLATFORM_ADMIN |
| M-P1-11 ✅ | MFA step-up on partner keys, match queue, dunning run-once |

**Owners:** A7.

### Wave B6 — Money fail-closed

**Status: IMPLEMENTED 2026-08-13** — see [`WAVE_B6_MONEY_FAILCLOSED.md`](./WAVE_B6_MONEY_FAILCLOSED.md)

| ID | Fix |
|----|-----|
| SPINE-P0-4 ✅ | AR open same txn as credit leave (`OpenFromCreditLeaveInTxn`); disabled AR fail-closed |
| S-P0-4 partial ✅ | Claim approve/reject HTTP Idempotency-Key (UNDER_REVIEW emit was B1) |
| M-P1-6 ✅ | AR aging bucket change + `AR_INVOICE_AGING_UPDATED` outbox |
| Credit-leave event ✅ | `HandleCreditLeave` emits `CREDIT_LEAVE` |
| Refund/BA bus ✅ | Dispatcher for `REFUND_*` + `BUYER_ACCEPTANCE_*` |

**Owners:** A0 + A2 packages.

### Wave B7 — Scope & stubs (fail-closed)

**Status: IMPLEMENTED 2026-08-13** — see [`WAVE_B7_SCOPE_STUBS.md`](./WAVE_B7_SCOPE_STUBS.md)

| ID | Fix |
|----|-----|
| R-P0-3 ✅ | Retailer cancel/create/unified stubs → **503** `order_service_unwired` |
| D-P0-6 ✅ | Depart nil fn → **503** `depart_unwired` (+ release idempotency) |
| D-P0-7 ✅ | Return-complete nil fn → **503** `return_complete_unwired` |
| WH-P0-3 / M-P1-10 ✅ | Reverse receive home-node pin + `WAREHOUSE_ADMIN` + ops scope |
| WH-P0-4 ✅ | Stocklots by-id membership (`warehouse_scope_forbidden`) |
| WH-P0-5 ✅ | Returns inbound scan → `RETURN_SCAN_RECEIVED` outbox |
| FAC-P0-3 ✅ | Factory setup same-txn factory create/location outbox |
| PL-P0-6 / M-P1-12 ✅ | Payload list/detail/seal/start-loading warehouse scope |

**Owners:** A2 + A3 + A4 + A5 + A6 packages.

---

## 6. Outbox density signal (orchestrator grep)

| Package | Emit sites | Spanner write sites | Signal |
|---------|------------|---------------------|--------|
| stocklots | **0** | ~77 | **P0 silent stock** |
| retailer | ~12 | ~93 | Many silent (cart/POS) |
| supplier | ~20 | ~99 | Inventory/CT gaps |
| warehouse | ~24 | ~91 | WMS off bus |
| order | ~69 | ~207 | Core path good; edges mixed |
| claims / ar / payout | high ratio | — | Mostly evented |

---

## 7. Next execution step

**Implement Wave B1 first** (money keys + silent driver + payout tenant + claim emit).  
After B1: `go test ./order ./payment ./claims ./driver ./payout -count=1` + gap re-grep.

UI/platform app parity is **out of scope** until B1–B2 P0s close — backend contract must be truthful first.

---

## 8. Report index

| File |
|------|
| [BACKEND_PARITY_PROTOCOL.md](./BACKEND_PARITY_PROTOCOL.md) |
| [BACKEND_PARITY_SPINE.md](./BACKEND_PARITY_SPINE.md) |
| [BACKEND_PARITY_SUPPLIER.md](./BACKEND_PARITY_SUPPLIER.md) |
| [BACKEND_PARITY_RETAILER.md](./BACKEND_PARITY_RETAILER.md) |
| [BACKEND_PARITY_DRIVER.md](./BACKEND_PARITY_DRIVER.md) |
| [BACKEND_PARITY_WAREHOUSE.md](./BACKEND_PARITY_WAREHOUSE.md) |
| [BACKEND_PARITY_FACTORY.md](./BACKEND_PARITY_FACTORY.md) |
| [BACKEND_PARITY_PAYLOAD.md](./BACKEND_PARITY_PAYLOAD.md) |
| [BACKEND_PARITY_PLATFORM_ADMIN.md](./BACKEND_PARITY_PLATFORM_ADMIN.md) |
