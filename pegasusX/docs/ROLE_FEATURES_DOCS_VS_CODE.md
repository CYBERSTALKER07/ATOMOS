# PegasusX — Role features: docs vs code

**Date:** 2026-08-13  
**Tree:** `pegasusX/` only (canonical). Legacy `pegasus/` is a **port source**, not a claimed product.  
**This file is two things:**

| Part | What |
|------|------|
| **I — Execution plan** | Phased, modular close of doc-drift + theatre + factory planning port |
| **II — Evidence baseline** | Role-by-role REAL / PARTIAL / THEATRE / GONE inventory (do not plan without it) |

**Do not start Part I Phase 2+ until Phase 0 proof is green** (same rule as G1–G7).

**Companions:** [`FEATURES_BY_APP_ROLE.md`](./FEATURES_BY_APP_ROLE.md) · [`ROLE_ROW_PARITY_MATRIX.md`](./ROLE_ROW_PARITY_MATRIX.md) · [`MANIFEST_DUAL_PLANE.md`](./MANIFEST_DUAL_PLANE.md) · [`session-2026-08-13/RESIDUAL_REGISTER.md`](./session-2026-08-13/RESIDUAL_REGISTER.md)

---

# Part I — Phased modularized plan

## 0. Doctrine

| Rule | Meaning |
|------|---------|
| **Honesty first** | Theatre → 410/403/501 **or** real persist. Never leave `{status: ok}` / always-`[]` advertised as a feature. |
| **Wire existing** | Prefer X tables/events/flags. Port Pegasus **algorithms**, not package names or dual SoT. |
| **One module = one PR-sized slice** | Independent tests; no “factory OS” mega-PR. |
| **Class A mutators** | JWT → PreferTenant → RW txn + outbox → cache invalidate → hub. |
| **Deploy ≠ code** | Soliq keys, OR-Tools pods, FCM stay in `RESIDUAL_REGISTER`. |
| **Dual plane stays** | `FactoryTruckManifests` ≠ `SupplierTruckManifests`. Pegasus already had both; do not merge. |

**Legend (same as Part II)**

| Tag | Meaning |
|-----|---------|
| **REAL** | Durable + clients on the live path |
| **PARTIAL** | Durable but incomplete |
| **THEATRE** | 200 that does not persist the claim |
| **GONE** | 410/403/501 by product |
| **DOC DRIFT** | Docs still sell it |

---

## 1. Program map (modules × phases)

```
P0  Docs honesty          ── no product code
P1  Theatre kill          ── fail-closed or delete ads
P2  Thin reads            ── real queries or honest empty
P3  Factory ops Class A   ── staff / exception / transfer outbox
P4  Client contract       ── call real paths; unmount unused
P5  Factory planning port ── 6 independent engines (largest)
P6  Supplier extras       ── optional product; not blocking
```

| Phase | Goal | Roles | Effort | Gate |
|-------|------|-------|--------|------|
| **P0** | FEATURES / ECOSYSTEM / matrix match Part II | Docs | S | FEATURES lists no 410/theatre as live |
| **P1** | Kill remaining THEATRE | Retailer, supplier, driver | S–M | `go test` those packages; no always-ok cards/audit |
| **P2** | Warehouse/supplier/driver reads honest | WH, supplier, driver | M | Empty arrays only when query empty |
| **P3** | Factory mutators Class A | Factory | M | Staff + exception + transfer create emit outbox |
| **P4** | Clients hit REAL routes | Retailer AI, payload | S–M | No `/v1/ai/predictions` client; seal-all documented unused **or** wired |
| **P5** | Port factory replenishment OS | Factory + supplier planning | L (6 modules) | Each module has tests + flag |
| **P6** | Optional Pegasus supplier extras | Supplier | M each | Product approve before code |

---

## 2. Phase 0 — Docs honesty (no product code)

**Why first:** FEATURES still sells B2B checkout, saved cards, negotiations, request-cancel, inventory audit, `/v1/ai/predictions` as if they work.

### Modules

| ID | Module | Files | Done when |
|----|--------|-------|-----------|
| **P0-A** | FEATURES honesty pass | `FEATURES_BY_APP_ROLE.md` | Each THEATRE/GONE row tagged; factory dispatch = stub; AI alias vs real path |
| **P0-B** | Matrix footnote | `ROLE_ROW_PARITY_MATRIX.md` | “Wired = happy path, not every FEATURES row” |
| **P0-C** | ECOSYSTEM factory/optimizer | `ECOSYSTEM_FEATURES_BY_ROLE.md` | Factory dispatch ≠ warehouse VRP; optimizer = OR-Tools **or** H3 heuristic |
| **P0-D** | Stale audits banner | `session-2026-08-12/BACKEND_PARITY_PAYLOAD.md`, `*_RETAILER.md` | Banner: payload **is** mounted; cash checkout REAL (B1) |

**Non-goals:** rewrite frozen `.docx`; invent features.

**Proof:** grep FEATURES for `card-checkout` / `negotiat` / `/v1/ai/predictions` shows GONE/THEATRE tags.

---

## 3. Phase 1 — Theatre kill (fail-closed)

Prefer **410 + honest body** over implementing a product nobody asked for. Implement only if a client already depends on success.

| ID | Surface | Today | Action | Packages |
|----|---------|-------|--------|----------|
| **P1-R1** | `/v1/retailer/card*` | always-ok empty | **410** `saved_cards_not_product` **or** real vault (product decide; default 410) | `retailer` |
| **P1-R2** | `GET /v1/ai/predictions` | `[]` | Proxy to `GET /v1/retailer/ai/predictions` **or** 410 `use_retailer_ai_predictions` | `retailer` |
| **P1-R3** | `PATCH /v1/ai/predictions/correct` | `{status:ok}` | 410 until persist exists | `retailer` |
| **P1-S1** | `GET /v1/supplier/inventory/audit` | always `[]` | Query stocklot/adjust ledger **or** 410 `audit_unwired` | `supplier` |
| **P1-D1** | `PATCH /v1/orders/{id}/state` | 501 mounted | Leave 501; **unmount** from FEATURES as a feature | `order` docs |
| **P1-D2** | `update-order-during-delivery` | not_implemented | Keep; FEATURES must say use amend/partial | docs |

**Non-goals:** loyalty, ratings, Soliq UI, negotiations (stay 410).

**Proof:**

```bash
go test ./retailer/ ./supplier/ ./order/ -count=1
# cards / audit / AI alias: 410 or real rows, never silent ok
```

---

## 4. Phase 2 — Thin reads (real query or honest empty)

| ID | Surface | Action |
|----|---------|--------|
| **P2-W1** | Warehouse treasury invoices | Query `ArInvoices` scoped to warehouse supplier; `[]` only if none |
| **P2-W2** | Analytics `top_products` / `daily_breakdown` | Aggregate from order lines or drop keys + `available: false` |
| **P2-W3** | Financials `gateway_breakdown` / `daily_revenue` | Same |
| **P2-W4** | Demand forecast scaffold | If Spanner empty: `{items:[], source: empty}` — **no seed SKUs** in SSMR/prod (`SeedFallbackAllowed`) |
| **P2-S1** | S&OP 700 VU/day | Label `capacity_source: env_default` (already calibrated) + FEATURES honesty; optional later: factory DailyOutputCapacity column |
| **P2-D1** | Driver history | Spanner `Orders` by `DriverId` (completed window); drop in-memory map as SoT |
| **P2-R1** | `pending-payments` fake `sess_` / `payme` | Real session id or omit; no invented gateway |

**Proof:** `go test ./warehouse/ ./driver/ ./retailer/`; demand test asserts no scaffold when run-mode ssmr.

---

## 5. Phase 3 — Factory ops Class A (not the planning OS)

Close holes on **existing** factory surfaces before porting Pegasus engines.

| ID | Mutator | Action |
|----|---------|--------|
| **P3-A** | Staff POST | Spanner + outbox `FACTORY_STAFF_*` (mirror warehouse staff) |
| **P3-B** | Manifest exception resolve | `apply` + `MANIFEST_EXCEPTION_RESOLVED` outbox |
| **P3-C** | Transfer create | `emit` required (not `nil`); `TRANSFER_CREATED` |
| **P3-D** | `POST /v1/factory/dispatch` honesty | Response field `optimizer_class: HEURISTIC` + `dispatch_algo: pick_n_created_v1`. Do **not** claim bin-pack until P5-F. |
| **P3-E** | Transfer GET | Spanner first; memory overlay only if `SeedFallbackAllowed` |

**Non-goals:** FFD batcher (P5-F); network optimizer (P5-A).

**Proof:** `go test ./factory/ -count=1`; exception/staff tests assert outbox mutation.

---

## 6. Phase 4 — Client contract close

| ID | Drift | Action |
|----|-------|--------|
| **P4-R1** | No client calls `GET /v1/retailer/ai/predictions` | Point desktop/Android/iOS at the real path |
| **P4-P1** | `seal-all` / capacity unused | Wire **or** 410 + unmount from FEATURES live list |
| **P4-P2** | Fleet reassign in-memory orders | Persist `Orders.RouteId`/`DriverId` in same txn as outbox |
| **P4-F1** | Factory SLA badges mobile | Bind `sla_*` already on supply-request JSON (portal done G7) |

**Non-goals:** i18n linguistic; Substance Gate human walk (residual).

---

## 7. Phase 5 — Factory planning port (Pegasus → X)

Largest real gap. **Six modules, sequential.** Each has a flag default **off**. Port **logic**, persist on **X tables**.

### Shared contracts (P5-0, do first)

| Item | X home |
|------|--------|
| Lanes read | Already `GET /v1/supplier/supply-lanes` + `SupplyLanes` (or warehouse–factory pairs) |
| Transfers | `FactoryInternalTransfers` / existing factory transfer rows — **not** a new mega-table unless missing columns |
| Events already on clients | `PULL_MATRIX_COMPLETED`, `LOOK_AHEAD_COMPLETED`, `NETWORK_MODE_CHANGED`, `FACTORY_SLA_BREACH` — **emit for real**; do not invent new names |
| Mode | New `NetworkOptimizationMode` table **or** supplier policy JSON. Values: `SPEED\|ECONOMY\|BALANCED\|LOW_CARBON\|MANUAL_ONLY` |
| Dual plane | Batcher writes **`FactoryTruckManifests` only** |

```
P5-0 schema/mode
  → P5-A SelectOptimalFactory
  → P5-B Pull Matrix (uses A + lock)
  → P5-C Look-ahead (uses A + lock)
  → P5-D Predictive push (uses A; optional)
  → P5-E Transfer SLA monitor (uses A for reroute)
  → P5-F Batcher FFD+NN+LIFO
  → P5-G Kill switch (depends on B/D drafts)
```

P5-lock (SKU+factory, 10 min TTL, velocity priority) is a **library** used by B/C/D — not a user-facing phase.

### P5-0 — Schema + mode

| Work | Notes |
|------|--------|
| Columns | Lane: `DampenedTransitHours`, `FreightCostMinor`, `CarbonScoreKg` if missing |
| `NetworkOptimizationMode` | `(SupplierId, Mode)` |
| Flag | `FACTORY_PLANNING_ENABLED` default false |

### P5-A — Network optimizer

**Port:** `pegasus/apps/backend-go/factory/network_optimizer.go`

| Mode | Sort |
|------|------|
| SPEED | `DampenedTransitHours` ASC |
| ECONOMY | `FreightCostMinor` ASC |
| LOW_CARBON | `CarbonScoreKg` ASC |
| BALANCED | `0.5·transit + 0.3·cost + 0.2·carbon` (normalized) |
| MANUAL_ONLY | skip automation |

Capacity **observer-only** (never blocks). Fallback: nearest active factory (Haversine).

**API:** `GET/PUT /v1/supplier/network-mode` (supplier). Internal: `SelectOptimalFactory(ctx, supplier, warehouse, sku, mode)`.

**Tests:** SPEED picks shorter transit; MANUAL_ONLY returns empty.

### P5-B — Pull Matrix

**Port:** `pull_matrix.go` — scan stock &lt; safety; group by factory via A; write internal transfers.

| Trigger | Interval / event |
|---------|------------------|
| Cron | 4h worker |
| Event | `OUT_OF_STOCK` |
| Manual | `POST /v1/supplier/planning/pull-matrix` (idempotent) |

Emit `PULL_MATRIX_COMPLETED`. Respect `MANUAL_ONLY`.

### P5-C — Look-ahead

**Port:** `look_ahead.go` — `ShadowDemand = Σ(LOCKED+PENDING next 7d) − stock`; `+15%`; transfer if &gt; 0 even when “safe” today. Split Class-C at **400 VU**. Link replenishment id on transfer.

Emit `LOOK_AHEAD_COMPLETED`.

### P5-D — Predictive push (optional)

**Not** the retailer AI preorder list. Read supplier `AIPredictions` where status WAITING and within safety-stock days → `SYSTEM_PREDICTED` transfers. Touchless uses existing `AutoApprovePredictivePush`.

If `AIPredictions` grain is retailer-preorder only, **skip** this module and document — do not fake SYSTEM_PREDICTED.

### P5-E — Transfer SLA monitor

**Different from G7 supply-request SLA.**

Pegasus: 30-min cron on APPROVED/LOADING vs `DampenedTransitHours`: 1× warning, 1.5× Kafka critical, 2× auto-reroute via A.

X: new worker `RunFactoryTransferSLAWorker`. Keep G7 request SLA. Emit existing `FACTORY_SLA_BREACH` with `kind: transfer_transit` vs `kind: supply_request`.

### P5-F — Batcher (replaces pick-≤2)

**Port:** `batcher.go` — FFD (volume DESC, trucks DESC) → NN from factory origin → LIFO load. Writes `FactoryTruckManifests` + LOADING.

`POST /v1/factory/dispatch` when `FACTORY_BATCHER_ENABLED`: this engine; else keep pick-n + `dispatch_algo` label.

**Reuse X:** `dispatch.BinPack` is **retail delivery**. Do not call it for factory transfers. Port FFD locally under `factory/batcher.go`.

### P5-G — Kill switch

Cancel `SYSTEM_THRESHOLD` / `SYSTEM_PREDICTED` drafts; set mode `MANUAL_ONLY`. Emergency / user transfers still work.

`POST /v1/supplier/planning/kill-switch` + audit outbox.

### P5 proof

```bash
go test ./factory/ ./replenishment/ -count=1
# SelectOptimalFactory table tests
# Pull matrix writes transfer under SAFETY breach
# Batcher FFD+LIFO unit test (no Spanner)
# Kill switch leaves MANUAL transfers
```

**Non-goals:** full MES/MRP/BOM; merging dual manifest tables; claiming LP-optimal MEIO.

---

## 8. Phase 6 — Supplier extras (product-gated, modular)

Each is **optional**. Do not start without a product yes.

| ID | Feature | Port from | X home |
|----|---------|-----------|--------|
| **P6-A** | Supplier CRM | `pegasus/.../supplier/crm.go` | `supplier/crm.go` + `/v1/supplier/crm/retailers` |
| **P6-B** | Payout policy self-service | `payout-policy` routes | `GET/PATCH /v1/supplier/payout-policy` (treasury already exists internally) |
| **P6-C** | Entity resolution | `entityresolution/` | New package, supplier-scoped |
| **P6-D** | Country config + overrides | `countrycfg` | Only if multi-country is product |
| **P6-E** | Rule of 25 chunker | `SplitManifest` | `dispatch` / `routing` — `AUTO-{driver}-{ts}-A/B` |
| **P6-F** | Supply-request QC | `FactorySupplyRequestQC` | Factory + warehouse |
| **P6-G** | Loyalty / ratings | `retailer/phase5.go` | **Default skip** (Retail OS has no UX) |

**Non-goals:** Pegasus admin-portal as supplier UI; k-means return (X H3+score is the replacement).

---

## 9. Cross-role execution order

```
P0 docs
  → P1 theatre (R/S/D in parallel)
  → P2 thin reads (W/S/D/R in parallel)
  → P3 factory ops Class A
  → P4 clients
  → P5-0 … P5-G (serial)
  → P6 only if approved
```

Role walk after each phase: FEATURES row + one client path. Same as G-program.

---

## 10. Explicit non-goals (whole program)

- Live Soliq/EDS/PSP/FCM secret cutover  
- OR-Tools prod replica flip  
- Full SAP IDoc / Drummond  
- Quantity negotiation / offline POS / Soliq OFD UI  
- Merging factory + payload manifest tables  
- o9/Kinaxis APS rewrite  

---

## 11. Exit criteria

| Phase | Exit |
|-------|------|
| P0 | FEATURES/matrix/ECOSYSTEM do not advertise theatre as live |
| P1 | No always-ok cards/audit/AI-correct |
| P2 | No scaffold demand in ssmr/prod; driver history Spanner |
| P3 | Factory staff/exception/transfer create Class A |
| P4 | Retailer clients use real AI path |
| P5 | Planning flags off-by-default; tests green; dispatch_algo honest |
| P6 | Each extra has its own proof or stays “not in X” |

---

# Part II — Evidence baseline (2026-08-13)

**Question:** Do living docs match backend + clients?

**Compared:** FEATURES, ECOSYSTEM, ROLE_ROW matrix, Retail OS gate, `BACKEND_PARITY_*` (some P0s later fixed B1–B7).  
**Code:** `*routes/routes.go`, role packages, client nav, `main.go` mounts.

“Wired” on the matrix = happy-path Class A exists. It does **not** mean every FEATURES row is production-complete.

## Scorecard

| Role | Core loop | Docs vs code | Biggest lie / hole |
|------|-----------|--------------|--------------------|
| **Retailer** | Create → track → pay-at-delivery | Core REAL; OS packs coded; several listed APIs theatre | Saved cards + `/v1/ai/predictions` empty alias; B2B 410 |
| **Supplier** | Vet → dispatch → catalog/inventory | Core REAL | Inventory **audit always `[]`**; negotiations 410; S&OP 700 VU/day default |
| **Warehouse** | Dispatch execute + WMS | Dispatch REAL; WMS REAL after B2 | Treasury **invoices always `[]`**; analytics breakdowns empty; demand can scaffold |
| **Factory** | Loading-bay start/seal | Seal/start REAL under Spanner | **Dispatch pick ≤2**; staff POST memory; exception no outbox; **no planning OS** |
| **Payload** | Seal / inject / reassign | Seal REAL (`payloaderoutes` mounted) | Dual tables (Pegasus already had both); `seal-all` unused |
| **Driver** | Arrive → QR → cash/credit → complete | Doorstep REAL | `PATCH …/state` 501; history in-memory |
| **Platform admin** | Tenants / flags / MFA | REAL | Ops empty if Spanner nil — honest |

---

## II.1 Retailer

**Clients:** desktop (Tauri), Android, iOS.

### Create — REAL

| Path | Persistence |
|------|-------------|
| `POST /v1/order/create` | Spanner `Orders` + reserve + `ORDER_CREATED` |
| `POST /v1/checkout/unified` + `items[]` | Same Create per supplier; parent order outbox (B3) |

Retailer `HandleCreateOrder` **unmounted**; hit → **503** `order_service_unwired`. Multi-supplier is sequential Create + compensate. Credit reserve at create only if `CREDIT_RESERVE_AT_CREATE=1`. Unified success does **not** clear server cart.

### Tracking — REAL

`GET /v1/retailer/tracking` — Spanner + GPS + geometry; missing telem → `AWAITING_TELEMETRY`.

**PARTIAL:** `pending-payments` fabricates `session_id = "sess_"+orderId` and `gateway: "payme"`.

### Payment — pay after offload

| Path | Docs | Code |
|------|------|------|
| Unified `items[]` | Checkout | **Creates** order (no capture) |
| Unified `order_id` / B2B | Card/B2B | **410** `payment_before_delivery_removed` |
| `POST /v1/order/card-checkout` | Card | **REAL** session + redirect |
| `POST /v1/order/cash-checkout` | Cash | **REAL** `PENDING_CASH_COLLECTION` |
| `/v1/retailer/card*` | Saved cards | **THEATRE** |
| Credit | Pay on credit | Read + driver credit-leave; no retailer “pay AR” |
| Fiscal | OFD | Pegasus commercial receipts; Soliq not default |

### Rest of FEATURES §1

Auth, catalog, cart, suppliers add/remove (durable favorites, **no** operating-schedule model), cancel pre-dispatch, preorder, shop-closed, claims, pulse, Retail OS packs: **REAL** (POS sale not one txn). Auto-order draft REAL; **place** PARTIAL (flag).  
`GET /v1/ai/predictions` **THEATRE** `[]` — real is `/v1/retailer/ai/predictions` (no client). Correct PATCH theatre. Preorder POST **410**. Loyalty/ratings **not in X**. Request-cancel **403**. Offline POS deferred.

---

## II.2 Supplier (`ADMIN`)

Portal + Tauri; Android/iOS exist; **no** `supplier-app-desktop`.

**REAL:** register/login/topology/org-fleet, vet, dispatch execute (same engine as WH), catalog, inventory adjust + import, pricing/promos, returns, claims, credit, control tower, shop-closed resolve.

| Feature | Status |
|---------|--------|
| Inventory audit | **THEATRE** always `[]` |
| Negotiations | **410** |
| Broadcast | PARTIAL — WS only, no outbox |
| S&OP | PARTIAL — `SOP_FACTORY_DAILY_UNITS` default 700 |
| MEIO | REAL heuristic `cost_aware_v2` / `greedy_capital_v1` |
| CRM / country overrides / entity resolution / payout-policy | **Not in X** |

Portal-only (no Android): control-tower, playbooks, segmentation, tax-regimes, credit policy, flywheel, payday, planning settings.

---

## II.3 Warehouse

Dispatch execute = strongest mutator (idempotency + outbox + freeze). WMS REAL after B2. Transfers/supply/returns REAL.

| Feature | Status |
|---------|--------|
| CRM / staff / payment-config | REAL |
| Treasury invoices | always `[]` |
| Analytics top_products / daily | always `[]` |
| Financials gateway/daily | always `[]`; `platform_fee` = 0 |
| Demand forecast | Spanner **or scaffold** |
| Factory QC | **Not in X** |

---

## II.4 Factory

Loading-bay start/seal/dispatch/complete/rebalance: REAL under Spanner + payload JWT on bay routes. **Not** Pegasus FFD+NN+LIFO.

`POST /v1/factory/dispatch`: ≤2 `CREATED` transfers, first driver/vehicle, DRAFT. Supply-request accept REAL. G7 SLA board = **request due-date**, not transit 1×/1.5×/2×.

Staff POST **memory**. Exception resolve **no outbox**. Transfer create `emit=nil`. Planning OS **not in X** (P5).

Insights: factory clients call **warehouse** replenishment insights — intentional.

---

## II.5 Payload

`main.go` mounts **`payloaderoutes`** (richer). Dual plane = `FactoryTruckManifests` vs `SupplierTruckManifests` — **Pegasus already had both**; X added `manifest_domain` + payload package extras (load ledger, seal-all, reassign, ship-units).

Seal requires `manifest_id`. Fleet reassign emits outbox but order overlay still in-memory. `seal-all` / capacity: **no client**.

---

## II.6 Driver

Doorstep REAL on `orderroutes`. Arrive: no GPS. Partial/complete: GPS. Cash/credit: stable idempotency. Depart/return 503 if fn nil.

`PATCH …/state` **501**. Mid-delivery update **not_implemented**. Negotiate **410**. History **in-memory**. Earnings REAL if wired.

X is strictly ahead of Pegasus on driver (Pegasus had no `driver/` package).

---

## II.7 Platform admin

Login+MFA, tenants, flags dual-control, partner, outbox + **dead-letters**: REAL. Honest `{available:false}` without Spanner.

---

## II.8 Product disables (all docs must say)

| Feature | Behavior |
|---------|----------|
| Quantity negotiation | 410 |
| Pre-delivery card/B2B | 410 |
| Request-cancel after dispatch | 403 |
| Soliq OFD | Not default |
| Auto-order place | Flag off |
| Airwallex | Flag off |

---

## II.9 What living docs get wrong

1. Matrix “Wired” ≠ every FEATURES row.  
2. FEATURES still lists B2B, cards, negotiations, request-cancel, inventory audit, `/v1/ai/predictions` as live.  
3. `BACKEND_PARITY_PAYLOAD` (2026-08-12): `payloaderoutes` **is** mounted now.  
4. Retailer cash checkout ack-only audit is **stale** (B1).  
5. ECOSYSTEM treats optimizer-core as always-on solver — runtime is OR-Tools **or** H3 heuristic.  
6. ECOSYSTEM “factory dispatch engine” is warehouse/supplier VRP, not `POST /v1/factory/dispatch`.

---

## II.10 Re-verify

```bash
rg -n 'RegisterRoutes' pegasusX/apps/backend-go/main.go

rg -n 'entries.: \[\]any|cards.: \[\]any|StatusGone|order_service_unwired|not_implemented' \
  pegasusX/apps/backend-go/{retailer,supplier,warehouse,factory,payload,driver,order,payment}

rg -n '/v1/ai/predictions|/v1/retailer/ai/predictions' pegasusX/apps
```

Do not plan from frozen `.docx`. Use `pegasus/` only as a **port source** for P5/P6.
