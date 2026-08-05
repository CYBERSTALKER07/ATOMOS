# PegasusX Ecosystem — Features by Role (Deep Reference)

**Audience:** operators, product, engineering  
**Grounding:** live monorepo (`apps/backend-go`, role apps, contracts); spatial/dispatch bullets re-aligned 2026-08-05  
**Money unit:** all financial amounts are **integer minor units** (tiyin/cents-style). **Never floats** for money.

> **Authoritative split (2026-08-04, CODE-GROUNDED):** Prefer these over this overview. They were extracted from `*routes/routes.go`, client shells/section enums, and Go packages — not from this file:
>
> 1. [FEATURES_BY_APP_ROLE.md](./FEATURES_BY_APP_ROLE.md) — routes + client nav inventory  
> 2. [ROLE_CAPABILITIES_MATH_LOGIC.md](./ROLE_CAPABILITIES_MATH_LOGIC.md) — formulas from Go  
> 3. [ORDER_FLOW_AND_EDGE_CASES.md](./ORDER_FLOW_AND_EDGE_CASES.md) — `state_machine.go` + route triggers  
> 4. [OPTIMIZER_AND_ROUTING_RUNTIME.md](./OPTIMIZER_AND_ROUTING_RUNTIME.md) — OR-Tools + Google Routes **runtime** (code vs cloud)  
> 5. [PARTNER_API.md](./PARTNER_API.md) — Gate-3 machine identity + `/partner/v1` + outbound webhooks  


---

## Part 0 — What the ecosystem is for

### Problem

Wholesale / B2B distribution in markets like Uzbekistan needs one system where:

1. A **supplier** runs catalog, warehouses, factories, fleet, and finance.  
2. **Retailers** order online (desktop + mobile), often with credit or card.  
3. **Warehouse / factory / payload** staff load trucks correctly.  
4. **Drivers** deliver, collect cash, handle shop-closed / missing / damage.  
5. Money, stock, tax receipts (fiscal), and exceptions stay **consistent** across apps in near real time.

### Solution PegasusX provides

A **multi-tenant logistics + commerce platform**:

| Layer | Technology |
|-------|------------|
| System of record | Cloud Spanner |
| Cache / rate limits / WS fanout | Redis |
| Events | Transactional **outbox** → Kafka → workers + notification dispatcher |
| Realtime | WebSocket hubs per role (`/v1/ws`) |
| API | Go `backend-go` (~418 `/v1` routes) |
| Clients | Portals + Android + iOS per role (see role matrix) |

### Core design laws

1. **Role-row parity** — a feature for a role should land on all clients of that role (portal/desktop + Android + iOS) unless deferred.  
2. **Mutations** — Spanner write + **outbox in same RW transaction** + post-commit cache invalidation + WS/notification.  
3. **Idempotency** — mutating POSTs accept `Idempotency-Key` / `X-Idempotency-Key`.  
4. **Fiscal hard-gate (ADR-009)** — payment capture → `FISCALIZING` → only then `COMPLETED` (or fail/force).  
5. **Inventory** — reserve on order, release on cancel/vet reject/stale timeout.

---

## Part 1 — Architecture spine (all roles)

```
Client (role app)
  → HTTPS /v1/*  (JWT cookie or Bearer)
  → backend-go service layer
  → Spanner ReadWriteTransaction
       ├─ domain rows
       └─ OutboxEvents (same txn)
  → commit
  → cache invalidate (Redis)
  → Outbox relay (worker) → Kafka topics
  → Consumers:
       • order mutator (fiscal apply, payment settle hooks)
       • warehouse mutator
       • notification dispatcher → inbox + FCM
       • ai-worker (import, freeze locks, optional synthesis)
  → WebSocket hub fanout → client silent refresh
```

**Topics (conceptual):** main/orders, realtime, spatial/telemetry, webhooks, freeze-locks, inventory-import, logistics exceptions.

---

## Part 2 — Shared money & stock math

### 2.1 Minor units

| Concept | Formula / rule |
|---------|----------------|
| Display money | `amount_minor / 100` (or currency exponent) for UI only |
| Storage | `int64` minor only |
| Cap | `CapAmount(a, cap) = min(a, cap)` if cap > 0 |
| Credit available | `Available = CreditLimitMinor − CurrentBalanceMinor` |
| Credit utilization bps | `(CurrentBalanceMinor * 10000) / CreditLimitMinor` when limit > 0 |

### 2.2 Inventory

| Concept | Formula |
|---------|---------|
| Available to sell | `QuantityOnHand − QuantityReserved` |
| Reserve on checkout | `QuantityReserved += qty` if available ≥ qty (same Spanner txn as order) |
| Release on cancel / reject / stale | `QuantityReserved = max(0, QuantityReserved − qty)` |
| Stale release | unpaid `PENDING` older than TTL cancelled + released (prevents double-release via careful status guards) |

### 2.3 Claim pricing (post-delivery)

1. **Source of truth** = original order line unit prices (not retailer-supplied arbitrary amounts).  
2. **AggregateClaimLines** merges duplicate SKUs so split rows cannot bypass per-SKU qty caps.  
3. Claim line amount ≈ `qty * unit_price_minor` from order.  
4. Total capped by remaining claimable ceiling (order total − prior claims).  
5. Window: default **`CLAIM_WINDOW_HOURS=48`** from order **COMPLETED** time.  
6. Settlement modes: `LEDGER_ONLY` | `STORE_CREDIT` | `GATEWAY_REFUND` (card partial refund when session is GP).  
7. Cash/COD gateways settle as **INTERNAL/CASH/CREDIT** ledger clawback (no PSP).  
8. Optional `CLAIM_AUTO_APPROVE_MAX_MINOR` auto LEDGER_ONLY under threshold.

### 2.4 Credit risk tier (simplified)

Derived from delinquency count + balance vs limit (`deriveRiskTier` / `EvaluateRisk`): higher delinquency or near-limit balance → higher risk tier; freeze can block new credit orders.

### 2.5 Spatial

- Retailers sit in **H3 cells**; delivery zone is a precomputed set of cells around supplier center + radius + resolution.  
- Driver telemetry updates location; proximity / geo-report use H3 aggregates.  
- Route geometry: **Google Routes → OSRM → dense** (`ROUTING_PROVIDER=auto`); clients render backend polylines.  
- Dispatch plan: **optimizer-core (OR-Tools)** when `OPTIMIZER_BASE_URL` is healthy; else H3 BinPack (`fallback_phase1`). **Cloud SSMR/prod: heuristic-only until sidecar image + replicas ≥ 1.** SoT: [`OPTIMIZER_AND_ROUTING_RUNTIME.md`](./OPTIMIZER_AND_ROUTING_RUNTIME.md).

---

## Part 3 — Order lifecycle (cross-role)

### 3.1 Status graph (canonical)

```
PENDING → LOADED → IN_TRANSIT → ARRIVED
                              ↘ AWAITING_PAYMENT → PENDING_CASH_COLLECTION → FISCALIZING → COMPLETED
                              ↘ DELIVERED_ON_CREDIT → FISCALIZING (when money later) → COMPLETED
FISCALIZING → FISCAL_FAILED ⇄ FISCALIZING (retry)
FISCAL_FAILED → COMPLETED (force-complete, audited)
+ CANCELLED / CANCEL_REQUESTED / DELAYED / SCHEDULED / AUTO_ACCEPTED / BACKORDERED / RECONCILIATION_REQUIRED
```

Enforced by `order.ValidateStatusTransition`. Illegal reverse transitions (e.g. COMPLETED → anything operational) are rejected.

### 3.2 Happy path (ecosystem story)

1. **Retailer** checks out (unified/cash/card/credit).  
2. **Inventory** reserved at fulfilling warehouse.  
3. **Warehouse** (or auto-dispatch) builds routes → **manifest**.  
4. **Payload** seals truck; orders → LOADED/IN_TRANSIT as appropriate.  
5. **Driver** arrives → QR scan → payment choice → collect cash / card / credit.  
6. **Fiscal** receipt → COMPLETED.  
7. Optional **claim** within window → supplier approve → chargeback ledger (+ reverse logistics ticket).

### 3.3 Technical edge cases

| Edge | Behavior |
|------|----------|
| Optimistic concurrency | Status updates may 409/500 with concurrency message; clients retry |
| Idempotent collect-cash | Same key replays without double fiscal |
| Cancel in transit | Often `CANCEL_REQUESTED` then warehouse/supplier decision; inventory rules depend on path |
| Stale pending | Background release unpaid reservations |
| Force complete | Admin only; audited event `ORDER_FORCE_COMPLETED` |
| Split shipment / overflow | Capacity warnings; split shipment events |

### 3.4 Non-technical edge cases

| Edge | Reality |
|------|---------|
| Shop closed | Driver marks shop closed; retailer response window; escalate / credit leave / bypass offload |
| Cash shortfall / overage | Events + fiscal on **received** amount |
| Missing / damaged goods | Exception report + optional photo; claim + return dock ticket |
| Retailer disputes after delivery | Claim window; evidence; supplier adjudication |
| Multi-warehouse supplier | Topology + warehouse resolver chooses fulfillment node |

---

## Part 4 — Role: Retailer

### Clients

- **Desktop** (`retailer-app-desktop` Tauri)  
- **Android** / **iOS**  
- Auth: Firebase phone OTP and/or session JWT  

### Features

#### 4.1 Onboarding & profile

| What | How |
|------|-----|
| Register / login | `/v1/auth/retailer/*` |
| Setup | Address via **Places geocode** (`/v1/platform/geocode/*`), supplier attach |
| Profile / family members | Retailer profile APIs |
| Client policy | Force-update / kill-switch from `/v1/platform/client-policy` |

**Edge cases:** invalid address; outside delivery zone (H3 not in zone set); OTP SMS cost; offline setup.

#### 4.2 Catalog & procurement

| What | Browse products, search suppliers, cart |
| How | Catalog public/list; cart sync; promotions watch |

**Math:** cart line totals sum `qty * unit_price_minor`; checkout preview may apply promotions.

**Edges:** OOS policy (reject / partial / backorder); multi-supplier cart → unified checkout splits.

#### 4.3 Checkout & payment

| Modes | Unified checkout, cash-on-delivery, card (Global Pay initiate/confirm), credit |
| APIs | `/v1/checkout/*`, `/v1/order/*-checkout`, `/v1/retailer/card/*` |

**Edges:** card fail → retry; credit limit breach; gateway degraded event; Airwallex flag-gated off by default.

#### 4.4 Orders & tracking

| What | List/detail, cancel request, accept/reject delivery proposal, preorder confirm/edit/reject |
| Realtime | Pulse + WS silent refresh + tracking API |

**Edges:** cancel only in allowed statuses; proposal expiry; scheduled preorder promote at T-1.

#### 4.5 Auto-order & AI

| What | Rules by global/supplier/category/product/variant; AI predictions confirm/reject |
| APIs | `/v1/retailer/settings/auto-order/*`, `/v1/ai/predictions`, confirm/reject AI order |

**Edges:** AI synthesis can be disabled (`synthesisDisabled` on ai-worker); low confidence forecasts.

#### 4.6 Insights / analytics

Spend by period, categories, weekday patterns — read models from Spanner aggregates.

#### 4.7 Claims (post-delivery)

| What | File claim with lines + optional photo (media ticket → GCS) |
| When | Order COMPLETED + within claim window |
| Types | DAMAGED, MISSING, CONCEALED_DAMAGE, TEMPERATURE, TAMPER, OTHER |

**Edges:** photo required for some types (e.g. DAMAGED); amount capped; cannot claim more than ordered; IDOR across retailers.

#### 4.8 Credit profile (retailer view)

Read own profile: limit, balance, available, status (ACTIVE/FROZEN).

#### 4.9 Notifications

Device token registration + inbox mark-read; FCM when configured.

#### 4.10 Capability packs (Retail OS)

| Pack | Unlocks | Hard deps |
|------|---------|-----------|
| CORE | Procurement, dock, claims, credit view | always on |
| TEAM | Staff invites, roles | — |
| LOCATIONS | Branches, switcher | soft TEAM |
| STORE_STOCK | Store ledger, receive, counts | — |
| SECTIONS | Departments, SKU/staff map | STORE_STOCK |
| POS | Registers, sales, voids | STORE_STOCK |
| SHIFTS | Clock, cash shift recon | TEAM |
| REPORTS_PRO | Ops digests + CSV | — |
| CUSTOMER_ASSIST | Floor help tickets | SECTIONS + TEAM |

API: `GET/POST /v1/retailer/capabilities*`, `GET /v1/retailer/me`. Docs: `RETAILER_CAPABILITY_PACKS.md`.

#### 4.11 Team & locations

Staff roster (`/v1/retailer/org/members`), JWT person subject + `retailer_org_id`. Locations CRUD + switch-location. Docs: capability packs Phase 1–2.

#### 4.12 Store stock

Bins FLOOR/BACKROOM/QUARANTINE; receive/transfer/adjust/count. Doc: `RETAILER_STORE_STOCK.md`.

#### 4.13 POS

Registers, sessions, sales, tenders, void. Money int64 minor. Doc: `RETAILER_POS.md`.

#### 4.14 Shifts

Clock in/out; shift open/close cash recon; POS may require clock-in. Doc: `RETAILER_SHIFTS.md`.

#### 4.15 Sections / Assist / Reports Pro

Sections + SKU map; assist ticket lifecycle; reports summary/sales/inventory + CSV. Docs: `RETAILER_SECTIONS.md`, `RETAILER_ASSIST.md`, `RETAILER_REPORTS_PRO.md`.

#### 4.16 Control Tower pulse

`GET /v1/retailer/control-tower/pulse` — honest empty/live ops digest (orders, dock, POS, shifts, assist, low stock, 7d sales). **Never** supplier demo ids or mock charts.

---

## Part 5 — Role: Supplier (ADMIN)

### Clients

Portal (primary ops desk), Android, iOS, optional desktop shell.

### Features

#### 5.1 Dashboard, activity, pulse

Live operational KPIs and event feed from pulse APIs + WS.

#### 5.2 Orders

| What | List, detail, vet (accept/reject), reassign recommend/apply, payment-bypass token for drivers |
| Edges | Vet reject must release inventory; bypass only for AWAITING_PAYMENT; IDOR by supplier_id |

#### 5.3 Dispatch & fleet

| What | Dispatch preview/execute (often warehouse-scoped), fleet live map, fleet orders, early-complete approve |
| Algo | Optimizer-core VRPTW-style solve when up; else heuristics; capacity warnings suggest unselect orders |

#### 5.4 Manifests & exceptions

View seals, inject-order visibility, gate exceptions, rebalance/cancel paths (often factory-adjacent).

#### 5.5 Catalog, inventory, import

| What | CRUD products, images (GCS upload ticket), CSV import sessions |
| Flow | Create session → upload → mapping → approve → apply (Kafka import worker) |

**Edges:** bad CSV rows; freeze locks during import; mapping conflicts.

#### 5.6 Pricing & promotions & retailer overrides

Rules + per-retailer price overrides with preview.

#### 5.7 Network topology

Factories, warehouses, delivery zones (H3), supply lanes, geo-report, Places address pickers.

#### 5.8 Org & fleet staff

Org members (FACTORY_ADMIN / WAREHOUSE_ADMIN / …), fleet drivers/vehicles create.

#### 5.9 Finance

| Surface | Purpose |
|---------|---------|
| Ledger | Immutable payment ledger entries |
| Settlement authority | Grouped gateway × entry_type totals |
| Reconciliation mismatches | Detect drift |
| Chargebacks (manual PSP) | Record / reverse |
| **Claim chargebacks** | List `chargeback_clm_*` from claim settle |
| Earnings | Revenue summaries |
| Credit collections | List/freeze retailer credit profiles |

#### 5.10 Claims queue (adjudication)

| What | List OPEN claims, approve with settlement_mode, reject |
| Money | Ledger clawback; optional store credit; optional GP refund |

**Edges:** double approve idempotent chargeback id; INTERNAL cash gateway; over-cap amount; reverse logistics ticket on damage.

#### 5.11 Planning / AI / MEIO

Scenarios, seasonal overrides, knowledge graph, replenishment policies, AI recommendations decide accept/reject, sparsity gates, promo simulate.

#### 5.12 Operations

Broadcast to role rooms (WS), replenishment trigger, payment bypass.

#### 5.13 Returns

Supplier-side returns resolve and history (warehouse dock is physical receive).

---

## Part 6 — Role: Warehouse

### Clients

Portal, Android, iOS — home node warehouse-scoped JWT.

### Features

#### 6.1 Dispatch hub (core)

| What | Select orders, smart/manual mode, drivers/trucks, preview routes, execute, locks, runs history, settings |
| Algo | Optimizer + capacity check + lock exclusion + suggest unselect when overweight |
| Locks | Freeze locks prevent concurrent dispatch; ai-worker freeze registry |

**Edges:** driver on active manifest; vehicle capacity VU; warehouse-scoped vs supplier CEO global.

#### 6.2 Drivers & vehicles

Availability, assignment, live map (MapLibre / MapKit).

#### 6.3 Orders & preorders

Delay, reject, overflow, propose delivery date; preorder edit/reject/propose.

#### 6.4 Inventory & stock commitments & ops settings

Bin/stock views; policies; pick/pack automation toggles.

#### 6.5 Returns inbound

List OPEN physical returns; barcode scan (EAN-13 checksum); confirm disposition **RESTOCK / WRITE_OFF / …**; sessions.

**Edges:** wrong warehouse; barcode not found; claim-linked tickets without barcode.

#### 6.6 Replenishment & supply requests

Insights → request factory supply; transfers receive/force-receive.

#### 6.7 Treasury / payment config / CRM / analytics / demand forecast / staff

Ops board, exceptions, tomorrow board, control tower.

#### 6.8 Broadcast templates

Warehouse ops messaging to drivers/retailers.

---

## Part 7 — Role: Factory

### Clients

Portal, Android, iOS.

### Features

#### 7.1 Dashboard & analytics

Pulse of production/loading state.

#### 7.2 Loading bay & manifests

Start loading → load orders → seal → dispatch to warehouse/payload chain.

#### 7.3 Payload override / rebalance / cancel transfer

Hot-path corrections when truck contents wrong.

#### 7.4 Supply requests

Accept warehouse requests; fulfill options.

#### 7.5 Transfers

Create and transition factory↔warehouse stock moves.

#### 7.6 Staff & fleet

Factory-local drivers/vehicles for supply legs.

#### 7.7 Manifest exceptions

List/act on seal exceptions.

**Edges:** concurrent seal; volume VU overflow; cancel after LOADED.

---

## Part 8 — Role: Payload (payloader)

### Clients

Terminal (web/Electron-style), Android, iOS.

### Features

#### 8.1 Truck board

List trucks/manifests for warehouse; pick truck; see LOADED orders checklist.

#### 8.2 Seal

Per-manifest or seal-all / seal-completed; transitions orders toward driver-ready; emits `MANIFEST_SEALED`.

#### 8.3 Inject order

Late order into open manifest (capacity checks).

#### 8.4 Recommend / apply reassign

Suggest move order to another truck/route; apply.

#### 8.5 Exceptions

Report cannot-load / wrong SKU etc. → exception queue + optional supplier inject path.

#### 8.6 Inbound returns (dock)

Physical OPEN returns list (claim-linked).

#### 8.7 Notifications + WS

Live board updates.

**Edges:** seal without all picks; reassign after seal; dual-wire missing items; device offline.

---

## Part 9 — Role: Driver

### Clients

Android (Google Maps Compose), iOS (MapKit). **No web portal.**

### Features

#### 9.1 Auth & shift

Login; availability on/off (blocks dispatch when off).

#### 9.2 Manifest / rides list

Today’s stops; offline hashes/nonces for verification; early complete request (supplier approve).

#### 9.3 Navigation map

Live location telemetry POST; planned route geometry; camera lock; order picker sheets.

#### 9.4 Doorstep flow

| Step | API |
|------|-----|
| Arrive | `/v1/delivery/arrive` |
| QR scan | retailer QR → `/v1/delivery/scan-qr` → AWAITING_PAYMENT |
| Confirm cash | retailer `/v1/delivery/confirm-cash` |
| Collect cash | `/v1/order/collect-cash` → FISCALIZING |
| Card / split / credit | card capture, split payment, credit delivery |
| Fiscal retry / open fiscal | retry failed OFD; open fiscal counts |
| Complete | after fiscal SUCCESS |

#### 9.5 Exceptions

| Report | Notes |
|--------|-------|
| Missing items | Dual wire legacy + exception-report |
| Damage / wrong item | Photo URL often required |
| Shop closed | Starts grace + retailer response protocol |
| Bypass offload | Supervised exception |
| Return goods | Bring stock back |

#### 9.6 Offload / cash UI

Summary, fiscalizing spinner, fiscal failed, collect cash form.

#### 9.7 Supply transfers

Factory supply leg arrive.

#### 9.8 Rescue

Request/respond rescue dispatch.

#### 9.9 Earnings / history / profile

Driver payouts snapshot; past routes.

**Driver edge cases (real world):**

- Shop closed / no one answers  
- Partial delivery / short cash  
- Card terminal offline → cash or credit leave  
- Phone offline → offline verify hashes  
- Wrong QR / replay QR  
- Open fiscal freezes shift end  
- GPS spoof / bad telemetry  
- Multi-stop resequence  

---

## Part 10 — Platform features (all roles)

| Feature | Purpose |
|---------|---------|
| Geocode/Places | Autocomplete, place_id, reverse, forward |
| Media upload ticket | Signed GCS PUT for claims/OS&D (placeholder if signBlob fails) |
| Notifications inbox | Durable per-user notifications |
| Device tokens | FCM registration |
| Client policy | Min version, force update |
| Desktop/iOS updater | Tauri / plist manifests |
| Webhooks | GP, Adyen, Stripe, Payme, Click |
| Health/ready | K8s probes |

---

## Part 11 — Non-technical ecosystem edge cases (business)

| Situation | Product response |
|-----------|------------------|
| Retailer cannot pay card | Cash COD or credit if profile allows |
| Driver cannot find shop | Shop-closed flow; reschedule / credit / return |
| Warehouse short stock after order | Overflow / split / cancel with inventory release |
| Truck full | Capacity warning; unselect orders; second trip |
| Supplier disputes claim | Reject with note; no chargeback |
| Supplier accepts claim | Ledger clawback; optional store credit or card refund |
| Tax audit | PEGASUS platform receipts now; Soliq OFD later |
| Fraud multi-claim | Caps, window, evidence, IDOR, supplier scope |
| Staff phone reuse | Org member conflict; idempotent create |
| Multi-country later | Currency + gateway matrix already multi-gateway |

---

## Part 12 — Technical edge cases (platform)

| Situation | Mitigation |
|-----------|------------|
| Double-click pay/collect | Idempotency keys |
| Concurrent status updates | Optimistic concurrency + retry |
| Pod restart mid-mutation | Spanner txn atomicity + outbox |
| Kafka lag | Workers scaled; smoke may force-complete fiscal |
| Redis down | Ready fails; degrade carefully |
| GCS signBlob IAM missing | Placeholder media URLs (pilot) |
| Unknown payment gateway on claim | INTERNAL/CREDIT/CASH executors for ledger |
| WS multi-pod | Redis relay subscribers |
| Rate limit abuse | Reliability middleware (IP/class limits) |
| Stale cache | Invalidate on mutation |

---

## Part 13 — Algorithms & optimizers (summary)

| Area | Algorithm / approach |
|------|----------------------|
| Dispatch | Code: OR-Tools via `optimizer-core` + `optimizerclient`; cloud often heuristic (`fallback_phase1`) until sidecar deployed. Capacity checks + suggest-unselect. See [`OPTIMIZER_AND_ROUTING_RUNTIME.md`](./OPTIMIZER_AND_ROUTING_RUNTIME.md). |
| Routing geometry | Google Routes (primary) → OSRM → dense; clients render backend polyline only |
| Delivery zone | H3 disk/ring precompute around lat/lng/radius/resolution |
| Demand forecast | History-based series per warehouse SKU (Spanner or scaffold) |
| Credit risk | Rule tiers from delinquency + utilization |
| Claim pricing | Aggregate lines, order unit prices, cumulative caps, time window |
| Inventory | Conservative reserve (on-hand − reserved) |
| Idempotency | Key + body hash store for replay |
| Maglev-style notes | Separate skill/docs for LB selection; not the order state machine |

---

## Part 14 — Role → client matrix

| Role | Portal/Desktop | Android | iOS |
|------|----------------|---------|-----|
| Supplier | portal (+ desktop) | ✓ | ✓ |
| Retailer | desktop | ✓ | ✓ |
| Warehouse | portal | ✓ | ✓ |
| Factory | portal | ✓ | ✓ |
| Driver | — | ✓ | ✓ |
| Payload | terminal | ✓ | ✓ |

---

## Part 15 — How to extend safely

1. Map blast radius (roles, routes, consumers, WS).  
2. Schema + owner package only.  
3. Outbox in same txn.  
4. Cache keys.  
5. Contracts (`packages/types`, api-client, events.schema).  
6. All role-row clients.  
7. Tests + SSMR marker for cross-role behavior.  

---

## Part 16 — Document limits

This manual is **complete at product/feature level** for the monorepo as inspected. It is not a line-by-line dump of every handler. For handler lists see route packages under `apps/backend-go/*routes`. For live money path proof see SSMR claims smoke (`PX_E2E_CLAIMS_ALL_OK`, 2026-07-29).

---

*End of deep reference. Update when new roles/features land; keep money in minor units and lifecycle graph in sync with `order/state_machine.go`.*
