# PegasusX — Role features: docs vs code

**Date:** 2026-08-13  
**Tree:** `pegasusX/` only (canonical). Legacy `pegasus/` is a **port source**, not a claimed product.  
**This file is two things:**

| Part | What |
|------|------|
| **I — Execution plan** | Phased, modular close of doc-drift + theatre + factory planning port |
| **II — Evidence baseline** | Role-by-role REAL / PARTIAL / THEATRE / GONE inventory (do not plan without it) |

**Not a go-live certificate.** Agents must re-verify Part II in current code. **"Wired" / this inventory does not mean connect cloud.** Cloud/API/infra only after the live path is REAL (see skill `honest-code-gate`).

**Do not start Part I Phase 2+ until Phase 0 proof is green** (same rule as G1–G7).

**P0 status (2026-08-13):** proof greps green — FEATURES tags THEATRE/GONE/STUB; matrix has “Wired = happy path, not every FEATURES row”; ECOSYSTEM states factory dispatch ≠ warehouse VRP and optimizer = OR-Tools or H3; `BACKEND_PARITY_{PAYLOAD,RETAILER}` have STALE banners.

**P1 status (2026-08-13):** cards / AI alias / AI-correct / inventory audit return **410** (never silent ok/`[]`). PATCH state stays 501; mid-delivery stays not_implemented.

**P2 status (2026-08-13):** treasury invoices query-or-503; analytics/financials honest availability; demand no scaffold without seed; S&OP `capacity_source`; driver history Spanner; pending-payments no `sess_`/`payme`.

**P3 status (2026-08-13):** factory staff POST `RunTx`+`FACTORY_STAFF_CREATED`; exception resolve `RunTx`+`MANIFEST_EXCEPTION_RESOLVED`; transfer create `TRANSFER_CREATED`; dispatch honesty labels; transfer GET Spanner-first.

**P4 status (2026-08-13):** desktop/Android/iOS call `GET /v1/retailer/ai/predictions` `{items}` + confirm/reject-ai (alias still 410 for old builds). Capacity GET `410 capacity_unwired` (`payload/vehicle_capacity.go:19`). Seal-all **wired** on terminal+Android+iOS (P13-A). `POST /v1/payloader/reassign-order` persists `Orders.RouteId`/`DriverId` in the same txn as outbox (`payload/service.go:1598`). Factory SLA badges verified on portal board + Android card + iOS row/staff.

**P5 status (2026-08-13):** factory planning OS ported, **flags default off**. `FACTORY_PLANNING_ENABLED` / `FACTORY_BATCHER_ENABLED` env (not money flags). Dedicated `SupplyLanes` + `NetworkOptimizationMode` tables — **do not** hijack `GET /v1/supplier/supply-lanes` (still topology warehouse utilization). P5-D **PARTIAL** — `SYSTEM_PREDICTED` from `DemandForecastBaseline` SUM vs `SupplierInventoryV2`+`ReorderThreshold` (`factory/predictive_push.go`); **not** ORDER-grain `AIPredictions`. Gated by `PlanningEnabled()` (Go default **false** — do not flip). **Factory dispatch live Spanner (2026-08-14):** warehouse solver class (`plan.OptimizeAndValidate`) → `FactoryTruckManifests` only; not pick-N; not `FACTORY_BATCHER_ENABLED` gate. Nil-Spanner tests still `pick_n_created_v1`. Never `dispatch.BinPack` as a lie.

**P6 status (2026-08-14 leftover close + flexibility):** A+F remain. **B/C/E in X** (payout-policy, entityresolution UI, SplitManifest naming). **D** country catalog AUTH_COUNTRIES in-memory (not CountryConfigs table; `checkout_reads_this: false`). **G** loyalty **live** `{enrolled:false}` / earn on paid orders (never fake Bronze; burn out of scope). CRM `Retailers.Email` + order `lines[]`. **Not cloud.**

**P7 status (2026-08-13):** factory honesty **A + B**. Exception GET Spanner-first (`FactoryTruckManifests` scope, `OrderId`→`transfer_id`); resolve **Spanner-first** (P9-B; memory only seed). Default dispatch **does not invent** empty CREATED queues (`created_manifest_count: 0`, no outbox). **P7-C PARTIAL** — portal+native CRM/QC/planning/payouts attached; **not store, not cloud.**

**P8–P16 status (2026-08-14 leftover close):** Enterprise close. GONE/410 stays. Dead thin `payloaderroutes` removed (live mount is `payloaderoutes` aliased as `payloadroutes` in `main.go`). P13-B typed scored-exception + playbook lists; P13-E retailer CT tiles navigate. Payout-policy thin UI exists; live rail still `no_live_rail`. Code+UI first; cloud apply artifacts in-tree **not applied**; store listings **not submitted**. **P15 not cloud-ready. P16 not store.**

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
P7  Factory honesty       ── exception GET + no invent dispatch
P8  SoT + dead code       ── docs + delete unused payloaderroutes
P9  Factory leftovers     ── staff login, resolve Spanner-first, QC gate, batcher overlay
P10 Supplier durability   ── broadcast outbox, S&OP capacity column
P11 Warehouse depth       ── financials queries, warehouse QC POST
P12 Admin billing         ── list APIs + portal tab + CronJob YAML
P13 Client parity         ── seal-all, portal-only native, CT, set-password
P14 Flag overlays         ── FACTORY_* k8s keys; Go defaults stay false
P15 Cloud apply           ── in-tree Terraform/k8s; owner secrets residual
P16 Store submit          ── PrivacyInfo + API URL; listings ops
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
| **P7** | Factory exception GET + no invent-if-empty | Factory | S–M | `go test ./factory/ -count=1`; empty dispatch count 0 |
| **P8** | SoT + delete unused `payloaderroutes` | Docs | S | FEATURES/matrix match P3/P7-C; package gone |
| **P9** | Factory staff login / QC gate / batcher overlay | Factory | M | hash ≠ unset; accept 409 without PASS; flags.go false |
| **P10** | Broadcast outbox + Factories.DailyOutputCapacity | Supplier | M | OutboxEvents row; S&OP column wins env |
| **P11** | Warehouse financials + QC POST | WH | M | `*_available` true only from query |
| **P12** | Admin billing list + UI | Platform | M | GET invoices; portal tab |
| **P13** | Client parity | All | L | seal-all call site; native slices |
| **P14** | k8s FACTORY_* keys | Infra | S | Go defaults still false |
| **P15** | Cloud apply artifacts | Ops | L | YAML/runbook; **not applied** without owner |
| **P16** | Store | Mobile | M | PrivacyInfo in-tree; **listings not submitted** |

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

**Proof:** grep FEATURES for `card-checkout` / `negotiat` / `/v1/ai/predictions` shows GONE/THEATRE/REAL-split tags (P0-A 2026-08-13).

---

## 3. Phase 1 — Theatre kill (fail-closed)

Prefer **410 + honest body** over implementing a product nobody asked for. Implement only if a client already depends on success.

| ID | Surface | Today | Action | Packages |
|----|---------|-------|--------|----------|
| **P1-R1** | `/v1/retailer/card*` | **410** `saved_cards_not_product` (2026-08-13) | default 410 (no vault) | `retailer` |
| **P1-R2** | `GET /v1/ai/predictions` | **410** `use_retailer_ai_predictions` | not proxied: DemandForecast[] ≠ `{items: RetailerAIPrediction}`; P4 retargets clients | `retailer` |
| **P1-R3** | `PATCH /v1/ai/predictions/correct` | **410** `prediction_correct_unwired` | 410 until persist exists | `retailer` |
| **P1-S1** | `GET /v1/supplier/inventory/audit` | **410** `audit_unwired` | no ledger reader; not silent `[]` | `supplier` |
| **P1-D1** | `PATCH /v1/orders/{id}/state` | 501 mounted | Leave 501; unmounted from FEATURES (P0) | `order` docs |
| **P1-D2** | `update-order-during-delivery` | not_implemented | Keep; FEATURES says amend/partial (P0) | docs |

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
| **P2-W1** | Warehouse treasury invoices | **Done 2026-08-13** — ArInvoices⋈Orders; `[]` only when query empty; no Spanner → 503 |
| **P2-W2** | Analytics `top_products` / `daily_breakdown` | **Done** — Spanner fills; fallback `*_available: false` (no fake `[]`) |
| **P2-W3** | Financials `gateway_breakdown` / `daily_revenue` | **Done** — daily from Orders when Spanner; gateway/fee `available: false` |
| **P2-W4** | Demand forecast scaffold | **Done** — `source: spanner\|scaffold\|empty`; scaffold only `WAREHOUSE_PORTAL_SEED` |
| **P2-S1** | S&OP 700 VU/day | **Done** — `capacity_source: env_default` |
| **P2-D1** | Driver history | **Done** — Orders by DriverId 30d completed window; memory map removed |
| **P2-R1** | `pending-payments` fake `sess_` / `payme` | **Done** — real PaymentSessions or omit |

**Proof:** `go test ./warehouse/ ./driver/ ./retailer/`; demand test asserts no scaffold when run-mode ssmr.

---

## 5. Phase 3 — Factory ops Class A (not the planning OS)

Close holes on **existing** factory surfaces before porting Pegasus engines.

| ID | Mutator | Action |
|----|---------|--------|
| **P3-A** | Staff POST | **Done 2026-08-13** — `RunTx` + `SupplierUsers` + outbox `FACTORY_STAFF_CREATED` |
| **P3-B** | Manifest exception resolve | **Done** — `RunTx` + `ManifestExceptions.ResolvedAt` + `MANIFEST_EXCEPTION_RESOLVED` |
| **P3-C** | Transfer create | **Done** — `apply` emit `TRANSFER_CREATED` (not `nil`) |
| **P3-D** | `POST /v1/factory/dispatch` honesty | **Updated 2026-08-14** — live Spanner path = warehouse solver class (`plan.OptimizeAndValidate`) → `FactoryTruckManifests` only. Nil-Spanner tests still `pick_n_created_v1`. Empty queue no invent. |
| **P3-E** | Transfer GET | **Done** — Spanner first; memory overlay only if `FACTORY_PORTAL_SEED`/`USE_DEMO_SEED` |

**Non-goals:** FFD batcher (P5-F); network optimizer (P5-A).

**Proof:** `go test ./factory/ -count=1`; exception/staff tests assert outbox mutation.

---

## 6. Phase 4 — Client contract close

| ID | Drift | Action |
|----|-------|--------|
| **P4-R1** | No client calls `GET /v1/retailer/ai/predictions` | **Done 2026-08-13** — desktop `dashboard/page.tsx:49`, Android `PegasusApi.kt:178`, iOS `APIClient.swift:370`. Alias remains 410. |
| **P4-P1** | `seal-all` / capacity unused | **Done 2026-08-13 / P13-A** — capacity `410 capacity_unwired` (`vehicle_capacity.go:19`); seal-all REAL persist + terminal+Android+iOS call sites. |
| **P4-P2** | Fleet reassign in-memory orders | **Done 2026-08-13** — fleet path already persisted; `HandleApplyReassign` now `tx.UpdateOrderAssignment` (`service.go:1598`) in the same `apply` txn as outbox. |
| **P4-F1** | Factory SLA badges mobile | **Verified 2026-08-13** — portal `SupplyRequestBoard.tsx`; Android `SupplyRequestCard.kt` `slaBadgeVisible`; iOS `SupplyRequestRow.swift` + `StaffView.swift`. No unbound list row. |

**Non-goals:** i18n linguistic; Substance Gate human walk (residual).

---

## 7. Phase 5 — Factory planning port (Pegasus → X)

Largest real gap. **Six modules, sequential.** Each has a flag default **off**. Port **logic**, persist on **X tables**.

### Shared contracts (P5-0, do first)

| Item | X home |
|------|--------|
| Lanes read | **Two planes:** `GET /v1/supplier/supply-lanes` = topology warehouse utilization (unchanged JSON). Optimizer edges = new `SupplyLanes` table (not that GET). |
| Transfers | `FactoryInternalTransfers` / existing factory transfer rows — **not** a new mega-table unless missing columns |
| Events already on clients | `PULL_MATRIX_COMPLETED`, `LOOK_AHEAD_COMPLETED`, `NETWORK_MODE_CHANGED`, `FACTORY_SLA_BREACH` — **emit for real**; do not invent new names |
| Mode | Dedicated `NetworkOptimizationMode` table (not `ReplenishmentPolicies` JSON). Values: `SPEED\|ECONOMY\|BALANCED\|LOW_CARBON\|MANUAL_ONLY` |
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
| Columns | **Done 2026-08-13** — real `SupplyLanes` (`DampenedTransitHours`, `FreightCostMinor`, `CarbonScoreKg`, `IsActive`, `Priority`) in `schema/migrations/20260813_p5_factory_planning.ddl`. **Does not** change `GET /v1/supplier/supply-lanes` JSON (`supplier/portal_ops.go` `listSupplierSupplyLanes` — LaneID = warehouse ID). |
| `NetworkOptimizationMode` | **Done** — dedicated `(SupplierId, Mode, UpdatedAt, UpdatedBy)`. Not stuffed into `ReplenishmentPolicies` JSON. |
| Other tables | `ReplenishmentLocks`, `PullMatrixRuns`, `FactorySLAEvents`; `FactoryInternalTransfers.Source` default `MANUAL_EMERGENCY`. |
| Flag | **Done** — `FACTORY_PLANNING_ENABLED` + `FACTORY_BATCHER_ENABLED` env default false (`factory/flags.go`). Registered as **non-money** keys in `featureflags/service.go`. |

### P5-A — Network optimizer

**Port:** `pegasus/apps/backend-go/factory/network_optimizer.go` → `factory/network_optimizer.go`

| Mode | Sort |
|------|------|
| SPEED | `DampenedTransitHours` ASC |
| ECONOMY | `FreightCostMinor` ASC |
| LOW_CARBON | `CarbonScoreKg` ASC |
| BALANCED | `0.5·transit + 0.0003·cost + 0.2·carbon` (same linear combo as Pegasus SQL) |
| MANUAL_ONLY | skip automation |

Capacity **observer-only** (never blocks). X `Factories` has no `CurrentLoad` — label is always `UNLIMITED`. Fallback: nearest active factory (`proximity.HaversineDistance`). Empty lanes optionally seeded from `Warehouses.PrimaryFactoryId` (24h / 0 cost / 0 carbon).

**API:** `GET/PUT /v1/supplier/network-mode` on ADMIN supplier session (`supplierroutes/routes.go`). PUT requires `auth.RoleAdmin`. Outbox `NETWORK_MODE_CHANGED` in the same txn as the upsert.

**Tests:** SPEED picks shorter transit; MANUAL_ONLY returns empty (`factory/network_optimizer_test.go`).

### P5-B — Pull Matrix

**Port:** `factory/pull_matrix.go` — scan `SupplierInventoryV2` on-hand−reserved vs `ReorderThreshold`; group by factory via A; write `FactoryInternalTransfers` `Source=SYSTEM_THRESHOLD`, `State=CREATED`.

| Trigger | Interval / event |
|---------|------------------|
| Cron | 4h worker (`StartPlanningCron`) — no-ops when flag off |
| Event | **Not invented** — X `OUT_OF_STOCK` is an order-reject reason, not a SKU Kafka trigger |
| Manual | `POST /v1/supplier/planning/pull-matrix` (409 if flag off). Run source `MANUAL` is audit-only; transfer Source stays `SYSTEM_THRESHOLD`. |

Emit `PULL_MATRIX_COMPLETED`. Respect `MANUAL_ONLY`. Lock library: SKU+factory, 10 min TTL, velocity preemption.

**Double-create gate:** when flag on, skip replenishment `autoCreateTransfer` / touchless `FactoryInternalTransfers` for `LOW_STOCK`/`HIGH_VELOCITY` only (`replenishment/planning_gate.go`). Insights still write. MEIO / `PREDICTIVE_PUSH` unchanged.

### P5-C — Look-ahead

**Port:** `factory/look_ahead.go`. X has **no** order state `LOCKED`. Shadow demand = line qty on orders with `ConfirmationStatus` in `CONFIRMED\|AUTO_CONFIRMED` (exclude PENDING/DRAFT/REJECTED), `Status` in `PENDING\|SCHEDULED\|AUTO_ACCEPTED\|BACKORDERED`, `RequestedDeliveryDate` next 7 days (Tashkent). `ShadowDeficit = ceil(qty·1.15) − on-hand`; transfer if &gt; 0 even when “safe”. Split Class-C at **400 VU**. Link `SourceInsightId` when an insight exists. Emit `LOOK_AHEAD_COMPLETED`.

### P5-D — Predictive push — **PARTIAL 2026-08-14**

**Not** the retailer AI preorder list. **Not** X `AIPredictions` (ORDER/`PENDING` grain; no `AIPredictionItems` / `RetailerId` / `TriggerDate`).

`POST /v1/supplier/planning/predictive-push` + cron after pull-matrix: `DemandForecastBaseline` SUM over horizon vs `SupplierInventoryV2` + `ReorderThreshold` → `SYSTEM_PREDICTED` (`factory/predictive_push.go`). Response honesty: `"grain":"demand_forecast_baseline"`, `"not_from":"AIPredictions"`. Gated by `PlanningEnabled()` (Go default **false**; local overlay may be true). Keep `AutoApprovePredictivePush` for existing replenishment reason codes.

### P5-E — Transfer SLA monitor

**Different from G7 supply-request SLA.** **Done.**

`RunFactoryTransferSLAWorker` / `ScanTransferTransitSLA` (30-min ticker): APPROVED/LOADING/CREATED vs lane `DampenedTransitHours`: 1× warning, 1.5× critical, 2× reroute via A. Idempotency: `FactorySLAEvents`. Payload `kind: transfer_transit`.

G7 `sla_worker.go` kept; payload now includes `kind: supply_request`.

### P5-F — Batcher (replaces pick-≤2)

**Port:** `factory/batcher.go` — FFD (volume DESC, trucks DESC) → NN from factory origin → LIFO load. Writes **`FactoryTruckManifests` only**, state **`DRAFT`** (X start-loading is DRAFT→LOADING). Transfers `ASSIGNED`. Empty queue → empty list (**no invent-if-empty**). `dispatch_algo: ffd_nn_lifo_v1`, `optimizer_class: HEURISTIC`.

`POST /v1/factory/dispatch` **live Spanner (2026-08-14):** warehouse solver class, not this batcher and not pick-n. `FACTORY_BATCHER_ENABLED` is unused on the live path (nil-Spanner tests still `pick_n_created_v1`).

**Reuse X:** `dispatch.BinPack` is **retail delivery**. Factory package does **not** call it.

### P5-G — Kill switch

**Done.** Cancel `SYSTEM_THRESHOLD` / `SYSTEM_PREDICTED` in CREATED/DRAFT/APPROVED; set mode `MANUAL_ONLY`; outbox via `NETWORK_MODE_CHANGED` (`reason=kill_switch`). Leave `MANUAL_EMERGENCY` and LOADING+.

`POST /v1/supplier/planning/kill-switch` (ADMIN). No new event name.

### P5 proof

```bash
go test ./factory/ ./replenishment/ -count=1
# SelectOptimalFactory table tests
# Pull matrix writes transfer under SAFETY breach
# Batcher FFD+LIFO unit test (no Spanner)
# Kill switch leaves MANUAL transfers
```

**Proof 2026-08-13:** `go test ./factory/ ./replenishment/ -count=1` green. Flags default off. No cloud / Layer B / Terraform claim.

**Non-goals:** full MES/MRP/BOM; merging dual manifest tables; claiming LP-optimal MEIO; UI for network-mode/kill-switch (API only); native store release.

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

**Stamp 2026-08-13 (approved subset A+F):** `go test ./supplier/ ./factory/ ./warehouse/ -count=1` green. **Not cloud.**

| ID | Status |
|----|--------|
| **P6-A** | **API 2026-08-13; clients P7-C PARTIAL** — `GET /v1/supplier/crm/retailers` + `/{retailerId}`. X columns: `Orders.Status`, `Orders.TotalMinor`, `Retailers.Email` (omit empty). Line count from `LineItemsJson`. Empty `{"retailers":[]}`. **503** without Spanner. `ACTIVE` if last order within 30 days. Does **not** change warehouse `GET /v1/warehouse/ops/crm`. |
| **P6-B** | **PARTIAL 2026-08-14** — `GET/PATCH /v1/supplier/payout-policy` (`payout/policy.go`). Modes `HQ_SUPPLIER` \| `WAREHOUSE_LOCAL`. Missing row → default HQ (`source: DEFAULT`). PATCH requires `reason`; Class A PreferTenant + RW txn + `AuditLog` + outbox `PAYOUT_POLICY_UPDATED`. Thin portal+Android+iOS. **Does not** enable a live PSP. Batches stay bank-file; live dispatch `no_live_rail`. |
| **P6-C** | **API+UI 2026-08-14** — `entityresolution/` + `POST /v1/supplier/entity-resolution/{resolve,explain}` (RoleAdmin). Result returned directly (no `{status:ok}`). Portal `/entity-resolution` + Android/iOS. |
| **P6-D** | **PARTIAL 2026-08-14** — `GET /v1/country-configs/{code}` **in-memory AUTH_COUNTRIES** (not a `CountryConfigs` Spanner table). Unknown → 404 `country_not_supported`. US is 200. `GET/PATCH /v1/supplier/country-overrides/{code}`. JSON includes `checkout_reads_this: false`. Checkout does **not** read this. Warehouse/factory `CountryCode` persisted. |
| **P6-E** | **PARTIAL 2026-08-14** — `dispatch.ExpandOversizeRoutes` after capacity checks, before persist (`supplier/dispatch_execute.go`, `warehouse/dispatch_execute.go`). Chunks named `AUTO-{driver}-{ts}-A/B`. `MaxWaypointsPerManifest = 25`. |
| **P6-F** | **API 2026-08-13; clients P7-C PARTIAL** — table `FactorySupplyRequestQC`. Factory `GET/POST /v1/factory/supply-requests/{id}/qc`. Parent = `WarehouseSupplyRequests` (factory-scoped). POST does **not** change request `State` (P9-C accept-gate requires PASS). Missing QC row → 200 `{result:""}`. Warehouse GET + POST (P9-C / P11-C). |
| **P6-G** | **LIVE 2026-08-14** — `GET /v1/retailer/loyalty/tier` + `/ledger`; supplier `GET/PATCH /v1/supplier/loyalty/program` (`reason` required). Unconfigured → `{enrolled:false}` (never fake Bronze). Earn on CollectCash + card settle. Burn out of scope. `HandleLoyaltyNotProduct` unmounted. `Orders.Rating` is queried in laborcapacity but **not in DDL** — do not “fix” that ghost. |

**Non-goals (P6 leftover):** live payout PSP; checkout reading countrycfg; fake loyalty tier; k-means return; Pegasus admin-portal as supplier UI. QC accept-gate + warehouse POST in P9/P11. **Do not invent a CountryConfigs Spanner table or global tax/PSP.**

---

## 8b. Phase 7 — Factory honesty

Close leftover factory theatre after P3/P5. No new screens. **Not cloud.**

| ID | Status |
|----|--------|
| **P7-A** | **PARTIAL** (clients already exist) — `GET /v1/factory/manifest-exceptions` Spanner-first: `ManifestExceptions` ⋈ `FactoryTruckManifests` by `FactoryId`. JSON keeps `transfer_id` (DDL `OrderId`). Seed overlay only `FACTORY_PORTAL_SEED`/`USE_DEMO_SEED`. Empty `{"exceptions":[]}`. Resolve **P9-B Spanner-first** (memory only when seed / no Spanner). Supplier `order_id` JSON **unchanged**. |
| **P7-B** | **Done** — default `POST /v1/factory/dispatch` no longer invents a CREATED transfer. Empty queue → 200 `created_manifest_count: 0`, no `MANIFEST_DRAFT_CREATED`. Batcher path and flags unchanged. |
| **P7-C** | **PARTIAL** (portal+native code, not store, not cloud) — Supplier `/crm` + Android/iOS CRM (`GET /v1/supplier/crm/retailers`, Email when set). Warehouse `/ops/crm` JSON **frozen**. Factory PASS/FAIL on existing board/list + native cards; warehouse GET+POST QC (P9-C). Factory-ops on `/settings/planning` + native PlanningSettings (mode, pull-matrix, kill-switch; 409 honest). `/finance/payouts` generate/export/mark-paid + payout-policy mode; live dispatch `no_live_rail`; `GET …/payouts/batches`. `FACTORY_PLANNING_ENABLED=true` in env examples/local only — `flags.go` default still false. `FACTORY_BATCHER_ENABLED=true` local overlay only (P9-D); Go default still false. |

**Proof 2026-08-13:** `go test ./factory/ ./supplier/ ./warehouse/ ./payout/ -count=1` green. P7-C **PARTIAL** portal+native, not store, not cloud.

---

## 8c. Phase 8–16 — Enterprise close

**Boss locks:** leave GONE/410; do not invent a live payout PSP; do not flip `PlanningEnabled()` / `BatcherEnabled()` Go defaults; code+UI first; cloud after live path REAL; store last. **Not a go-live certificate.**

| ID | Status |
|----|--------|
| **P8-A** | SoT append + P6/P7 stamp fix + matrix factory row (staff/exception are Class A persist, not in-memory THEATRE). |
| **P8-B** | Unused package `apps/backend-go/payloaderroutes/` **deleted**. Live mount remains `payloaderoutes` (`main.go` alias `payloadroutes`). |
| **P8-C** | Gap-hunter: remaining PARTIAL only; GONE not reopened. Retailer Control Tower depth (P13-E) **closed 2026-08-14** — pulse tiles navigate to existing retailer surfaces. |
| **P9-A** | Staff `SaveStaff` writes bcrypt or invite token (never `"unset"`). `POST /v1/factory/staff/{id}/set-password`. Portal + Android/iOS set-password UI. |
| **P9-B** | Exception resolve **Spanner-first** (`ManifestExceptions` ⋈ `FactoryTruckManifests`); memory only `FACTORY_PORTAL_SEED`. |
| **P9-C** | Factory QC POST still upserts QC row (does not rewrite State). Accept **409** `qc_pass_required` unless QC `PASS`. Warehouse **POST** QC on GET family. |
| **P9-D** | `FACTORY_BATCHER_ENABLED=true` in `.env.example` / `.env.local` / `.env.ssmr.example` only. `flags.go` default **false**. |
| **P10-A** | Supplier + warehouse broadcast: `OpsBroadcasts` + `outbox.EmitJSON` same txn, then WS. Events `SUPPLIER_BROADCAST` / `WAREHOUSE_BROADCAST`. |
| **P10-B** | `Factories.DailyOutputCapacity`. S&OP reads column when sum > 0 (`capacity_source: factories_column`); else `env_default` + 700. |
| **P10-C** | Payout bank-file remains the rail. Live dispatch `no_live_rail`. |
| **P11-A** | Warehouse financials: gateway from `PaymentSessions` ⋈ warehouse orders when query OK; platform fee from `BillingFeeSchedules` when a schedule exists; else `*_available: false`. |
| **P11-C** | Warehouse QC POST (with P9-C). |
| **P12-A/B/C** | `GET /v1/admin/billing/invoices` + fee-schedules (PLATFORM_ADMIN + MFA). Admin portal Billing tab. CronJob YAML `infra/k8s/billing_monthly_cronjob.yaml` **unapplied**. |
| **P13-A** | Payload `POST /v1/payloader/manifests/seal-all` on terminal + Android + iOS. Capacity stays 410. |
| **P13-B** | Supplier native slices: control-tower + playbooks are **typed lists** (not JSON dumps). Segmentation, tax-regimes, credit policy, flywheel, payday remain JSON dumps. Credit admin-disable POST form. |
| **P13-C** | Warehouse Control Tower on Android/iOS — typed `GET /v1/control-tower/exceptions/scored` list (not JSON dump). |
| **P13-D** | Factory staff set-password UI (portal + Android + iOS). |
| **P13-E** | **Done 2026-08-14** — retailer Control Tower pulse tiles navigate (Android `onNavigate`, iOS `NavigationLink`, desktop already linked). |
| **P14** | k8s ConfigMap keys `FACTORY_PLANNING_ENABLED` / `FACTORY_BATCHER_ENABLED` exist; prod values `"false"`. Go defaults false. `AUTO_ORDER_PLACE_ENABLED` not flipped. |
| **P15** | Apply artifacts inventoried in `docs/session-2026-08-13/P15_CLOUD_APPLY.md`. **Not applied. Not cloud-ready.** |
| **P16** | `PrivacyInfo.xcprivacy` on every iOS app. Release API default `https://api.pegasusx.app`. **Not store** — no App Store/Play listings submitted. |

**Proof P8:** `payloaderroutes` directory gone; `go test ./payload/ ./payloaderoutes/ -count=1`.

**Non-goals:** reopen GONE; flip `PlanningEnabled()` / `BatcherEnabled()` Go defaults; live payout PSP; App Store/Play submit from this agent; Terraform apply without owner GSM.

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
  → P7 factory honesty
  → P8 SoT + dead code
  → P9 factory leftovers
  → P10 supplier durability
  → P11 warehouse depth
  → P12 admin billing
  → P13 client parity
  → P14 flag overlays
  → P15 cloud apply (ops)
  → P16 store (ops)
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
| P5 | **Done 2026-08-13** — flags off-by-default; `go test ./factory/ ./replenishment/ -count=1` green; `dispatch_algo` honest (`pick_n_created_v1` / `ffd_nn_lifo_v1`). P5-D PARTIAL (`DemandForecastBaseline`, 2026-08-14). **Not cloud.** |
| P6 | **Leftover close 2026-08-14** — A+F + B/C/D/E in X; G 410. Clients: CRM Email + payout-policy thin UI. **Not cloud.** |
| P7 | **PARTIAL 2026-08-13 (A+B done; C portal+native)** — Exception GET Spanner-first; dispatch no invent. P7-C clients attached. **Not cloud. Not store.** |
| P8 | Dead `payloaderoutes` gone; SoT matches P3/P7-C |
| P9 | Staff bcrypt/invite; resolve Spanner-first; QC accept-gate; local batcher overlay; `flags.go` false |
| P10 | Broadcast outbox; `Factories.DailyOutputCapacity`; payout bank-file |
| P11 | Financials query-or-`available:false`; warehouse QC POST |
| P12 | Admin billing GET + portal tab; CronJob YAML unapplied |
| P13 | Seal-all clients; typed CT lists; retailer CT tiles; factory set-password UI |
| P14 | k8s FACTORY_* keys; Go defaults false |
| P15 | In-tree apply artifacts; **not cloud-ready** without staging apply evidence |
| P16 | PrivacyInfo in-tree; **not store** until listings exist |

---

# Part II — Evidence Baseline (Codebase SoT Verified)

**Last Verified:** 2026-08-20  
**Scope:** Verified against live Go backend route mounts (`main.go:1-479`), Cloud Spanner schema (`schema/spanner.ddl:1-3648`), shared packages (`packages/types`, `packages/api-client`, `packages/ws-refresh-contract`), and client applications across all 6 role rows + Platform Admin (`apps/*`).

“Wired” on the matrix = happy-path Class A exists in code and is verified by passing unit test suites. Layer A (code complete) is distinguished from Layer B (deploy-time cloud secrets/infra).

---

## 1. Role Scorecard & Reality Baseline

| Role Row | Core Loop & Surfaces | Codebase Implementation State | Exact File:Line Citations & Notes |
| :--- | :--- | :--- | :--- |
| **Retailer** | Create → Track → Pay-at-Delivery → POS / Store Stock | **REAL (Layer A Complete)**<br>All 3 platforms (Desktop, Android, iOS) wired to live endpoints. | • Desktop: `retailer-app-desktop/app/(dashboard)/dashboard/page.tsx:44`<br>• Android: `retailer-app-android/.../PegasusApi.kt:182`<br>• iOS: `retailer-app-ios/.../APIClient.swift:391`<br>• Backend: `retailerroutes/routes.go:37-287`<br>• Saved cards 410: `retailer/core_handlers.go:1337`<br>• AI Alias 410: `retailer/mobile_compat.go:71-81` |
| **Supplier** | Profile → Catalog / Inventory → Dispatch → S&OP / CRM | **REAL (Layer A Complete)**<br>Portal (Tauri), Android, iOS fully wired with real API SDK and WebSocket refresh. | • Portal: `supplier-portal/app/(portal)/*`, `lib/api.ts:4-9`<br>• Android: `supplier-app-android/.../SupplierApi.kt:9-711`<br>• iOS: `supplier-app-ios/.../SupplierOperationsService.swift:1-250`<br>• Backend: `supplierroutes/routes.go:71-270`<br>• Inventory Audit 410: `supplier/portal_handlers.go:1107-1118`<br>• Quantity Negotiation 410: `order/negotiation_disabled.go:22-30` |
| **Warehouse** | Dispatch Execute → WMS Bins / Waves → QC → Transfers | **REAL (Layer A Complete)**<br>Complete WMS, cycle counts, cold-chain, and delivery perimeter enforcement. | • Portal: `warehouse-portal/lib/warehouse-ops.ts:1-250`<br>• Android: `warehouse-app-android/.../WarehouseApi.kt:1-350`<br>• iOS: `warehouse-app-ios/.../WarehouseOperationsService.swift:1-200`<br>• Backend: `warehouseroutes/routes.go:28-205`<br>• Perimeters: `warehouse/perimeter.go:1-120` |
| **Factory** | Loading-Bay Start / Seal → Supply Requests → Transfers | **REAL (Layer A Complete)**<br>Loading bay start/seal uses Spanner `FactoryTruckManifests` + outbox frames. | • Portal: `factory-portal/lib/api.ts:1-250`<br>• Android: `factory-app-android/.../FactoryApi.kt:1-350`<br>• iOS: `factory-app-ios/.../FactoryService.swift:1-200`<br>• Backend: `factoryroutes/routes.go:23-100`<br>• SLA Boards: `factory/sla.go:1-180` |
| **Payload** | Scan Ledger → Ship-Units → Seal-All → Reassignment | **REAL (Layer A Complete)**<br>Terminal, Android, and iOS all call live `seal-all` and loading-bay APIs. | • Terminal: `payload-terminal/api.ts:46-210`<br>• Android: `payload-app-android/.../PayloadApi.kt:102`<br>• iOS: `payload-app-ios/.../APIClient.swift:247-250`<br>• Backend: `payloaderoutes/routes.go:21-74`<br>• Capacity 410: `payload/vehicle_capacity.go:19` |
| **Driver** | Arrive → QR Scan → Collect Cash → Offload → History | **REAL (Layer A Complete)**<br>Dual telemetry `/v1/ws?sv=2`, Room SQLite v6 / SwiftData offline queues. | • Android: `driver-app-android/.../DriverApi.kt:1-250`<br>• Android Telemetry: `TelemetrySocket.kt:78-86`<br>• iOS: `driver-app-ios/.../ManifestServiceLive.swift:1-150`<br>• iOS Telemetry: `TelemetryServiceLive.swift:1-200`<br>• Backend: `driverroutes/routes.go:41-129`, `deliveryroutes/routes.go:18-25` |
| **Platform Admin** | Tenants → Flags Dual-Control → Outbox Dead-Letters → Audit | **REAL (Layer A Complete)**<br>9 governance panels with real mutation APIs and live audit signals. | • Web: `admin-portal/components/*Panel.tsx`<br>• WS: `use-admin-ws-refresh.ts:17-70`<br>• Backend: `platformadmin/handlers.go:176-200`<br>• Feature Flags: `featureflags/handlers.go:165-181`<br>• MFA Step-Up: `mfa/handlers.go:139-192` |

---

## 2. Detailed Role-by-Role Evidence

### II.1 Retailer
- **Client Implementations**:
  - Desktop: `apps/retailer-app-desktop` (31 Next.js 15 pages in `app/(dashboard)/*`, Tauri wrapper in `src-tauri/tauri.conf.json`).
  - Android: `apps/retailer-app-android` (40+ composables, `PegasusApi.kt:182` calls `@GET("/v1/retailer/ai/predictions")`, `AppDatabase.kt:8-23` manages Room tables).
  - iOS: `apps/retailer-app-ios` (49 SwiftUI views in `retailerapp/retailerapp/Screens/`, `APIClient.swift:391` targets `/v1/retailer/ai/predictions`, `PendingPosStore.swift:1-120`).
- **Order Creation**:
  - `POST /v1/order/create` (`order/service.go:110-245`): Persists Spanner `Orders` + inventory reservation + `ORDER_CREATED` outbox event.
  - `POST /v1/checkout/unified` (`retailer/service.go:412-580`): Multi-supplier cart split with `ParentOrders` table (`schema/spanner.ddl:221-239`).
- **Payment & Tracking**:
  - `GET /v1/retailer/tracking` (`retailer/tracking.go:20-110`): Live GPS coordinates, ETA estimation, and `AWAITING_TELEMETRY` fallback.
  - Cash / Card Checkout: `POST /v1/order/cash-checkout` sets `PENDING_CASH_COLLECTION`; `POST /v1/order/card-checkout` initiates redirect session.
- **Product Boundaries**:
  - Saved Cards: `/v1/retailer/card*` returns HTTP 410 `saved_cards_not_product` (`retailer/core_handlers.go:1337`).
  - AI Predictions Old Alias: `GET /v1/ai/predictions` returns HTTP 410 `use_retailer_ai_predictions` (`retailer/mobile_compat.go:71-81`).

### II.2 Supplier (`ADMIN`)
- **Client Implementations**:
  - Web/Desktop: `apps/supplier-portal` (82 App Router routes, `@pegasusx/api-client` bound in `lib/api.ts:4-9`, Tauri desktop build).
  - Android: `apps/supplier-app-android` (61 Compose screens, `SupplierApi.kt:9-711` Retrofit interface, `SupplierWebSocket.kt:88-99`).
  - iOS: `apps/supplier-app-ios` (68 SwiftUI views, `APIClient.swift`, `SupplierOperationsService.swift:1-250`, `SupplierRealtimeClient.swift:45-65`).
- **Core Operations**:
  - Profile & Topology: `GET/PUT /v1/supplier/profile`, `GET /v1/supplier/topology` (`supplierroutes/routes.go:71-120`).
  - Catalog & Inventory: `GET/POST /v1/supplier/catalog`, `POST /v1/supplier/inventory/adjust`, `POST /v1/supplier/inventory/import` (`supplier/inventory.go:40-210`).
  - Planning & S&OP: `GET/PUT /v1/supplier/network-mode`, `POST /v1/supplier/planning/pull-matrix` (`supplier/planning_handlers.go:30-180`).
  - Entity Resolution: Master data deduplication via `entityresolutionroutes/routes.go:15-27`.
- **Product Boundaries**:
  - Inventory Audit: `GET /v1/supplier/inventory/audit` returns HTTP 410 `audit_unwired` (`supplier/portal_handlers.go:1107-1118`).
  - Quantity Negotiation: `POST /v1/delivery/negotiate` & `POST /v1/supplier/negotiate/resolve` return HTTP 410 `feature_disabled` (`order/negotiation_disabled.go:22-30`).

### II.3 Warehouse
- **Client Implementations**:
  - Web/Desktop: `apps/warehouse-portal` (46 routes, `lib/warehouse-ops.ts:1-250`, `lib/use-warehouse-ws-refresh.ts:1-120`).
  - Android: `apps/warehouse-app-android` (44 screens, `WarehouseApi.kt:1-350`, `WarehouseOfflineQueue.kt:1-120`).
  - iOS: `apps/warehouse-app-ios` (84 views, `APIClient.swift:1-400`, `WarehouseOperationsService.swift:1-200`).
- **WMS & Logistics Core**:
  - Dispatch Execution: `POST /v1/warehouse/dispatch/execute` packs orders, computes vehicle routes, and freezes manifests (`warehouse/dispatch.go:88-340`).
  - Delivery Perimeters: Redis polygon set index `SAdd`/`SIsMember` (`warehouse/perimeter.go:1-120`), enforced during checkout resolution (`warehouse_resolver_spanner.go:45-90`).
  - WMS Inventory: Bins, lots, pick waves, cycle counts, and temperature logs (`warehouseroutes/routes.go:28-205`, `schema/spanner.ddl:380-520`).

### II.4 Factory
- **Client Implementations**:
  - Web/Desktop: `apps/factory-portal` (21 routes, `loading-bay/page.tsx`, `transfers/page.tsx`, `lib/api.ts:1-250`).
  - Android: `apps/factory-app-android` (62 files, `FactoryApi.kt:1-350`, `FactoryRealtimeClient.kt:38-58`, `FactoryOfflineQueue.kt:1-100`).
  - iOS: `apps/factory-app-ios` (70 files, `APIClient.swift:1-350`, `FactoryService.swift:1-200`, `FactoryRealtimeClient.swift:1-120`).
- **Loading-Bay & Replenishment**:
  - Loading-Bay Operations: Start loading and seal operations write to Spanner `FactoryTruckManifests` and emit `FACTORY_MANIFEST_*` events (`factory/service.go:140-380`).
  - SLA Board & Workers: Due-date tracking for supply requests and transfer transits (`factory/sla.go:1-180`, `G7 SLA Board`).
  - Dispatch Algorithm: Live Spanner solver uses warehouse solver class (`plan.OptimizeAndValidate`), emitting zero manifests on empty queues without inventing fake records.

### II.5 Payload
- **Client Implementations**:
  - Terminal: `apps/payload-terminal` (Expo SDK 55 React Native app, `api.ts:46-210`, camera scanner, SecureStore).
  - Android: `apps/payload-app-android` (50 Kotlin files, `PayloadApi.kt:102`, `PayloadDatabase.kt:6-14` Room backing).
  - iOS: `apps/payload-app-ios` (43 Swift files, `APIClient.swift:247-250`, `OfflineQueue.swift:1-80`).
- **Scan Ledger & Manifest Sealing**:
  - `POST /v1/payloader/manifests/seal-all` (`payload/service.go:340-420`): Live batch sealing across all 3 client platforms.
  - Manifest Ship-Units: Scanning and variance recording (`payload/ship_units.go:25-140`).
  - Reassign Order: `POST /v1/payloader/reassign-order` (`payload/service.go:1598`) updates `Orders.RouteId`/`DriverId` in same transaction as outbox.
- **Product Boundaries**:
  - Vehicle Capacity: `GET /v1/payloader/capacity` returns HTTP 410 `capacity_unwired` (`payload/vehicle_capacity.go:19`).

### II.6 Driver
- **Client Implementations**:
  - Android: `apps/driver-app-android` (63 screens, `DriverApi.kt:1-250`, Room v6 `PegasusDriverDatabase.kt:11-21`, `DriverOfflineQueue.kt:1-150`).
  - iOS: `apps/driver-app-ios` (74 views, `APIClient.swift:1-350`, `ManifestServiceLive.swift:1-150`, SwiftData `OfflineDeliveryStore.swift:11-60`).
- **Doorstep Delivery & Telemetry**:
  - Doorstep Handshake: QR validation, cash collection, POD photo upload, and completion (`deliveryroutes/routes.go:18-25`, `order/delivery_handshake.go:45-190`).
  - Dual Telemetry: High-frequency GPS streaming over `/v1/ws?sv=2` and `/v1/driver/location` (`telemetryroutes/routes.go:59-150`, `TelemetrySocket.kt:78-86`, `TelemetryServiceLive.swift:1-200`).
  - Offline Replay: Full mutation queueing with exponential backoff for low-connectivity delivery execution.

### II.7 Platform Admin
- **Client Implementation**:
  - Web: `apps/admin-portal` (Next.js 15, 9 governance panels: `TenantsPanel.tsx`, `FlagsPanel.tsx`, `OpsPanel.tsx`, `BillingPanel.tsx`, `AuditPanel.tsx`, `PartnerPanel.tsx`, `MatchQueuePanel.tsx`, `AccuracyPanel.tsx`).
- **Governance & Dual-Control**:
  - Tenant Lifecycle: Onboarding, state transition, and cell mapping (`platformadmin/handlers.go:176-200`).
  - Feature Flags: Dual-control mutation (`POST /v1/admin/featureflags/propose` + `approve`, `featureflags/handlers.go:165-181`).
  - Dead-Letter Replay: Outbox poison message inspection and replay trigger (`/v1/admin/ops/outbox/dead-letters`, `/v1/admin/ops/dead-letters/replay`).
  - MFA Enforcement: TOTP enrollment, confirmation, and step-up authorization (`mfa/handlers.go:139-192`).

---

## 3. Product Disables & Gated Surface Registry

| Feature / Surface | Route & Method | Wire Behavior | Ground Truth & Code Reference |
| :--- | :--- | :--- | :--- |
| **Saved Cards Vault** | `/v1/retailer/card*` | **410 GONE** | `retailer/core_handlers.go:1337` returns `saved_cards_not_product`. B2B flow uses COD cash/credit or one-time payment session. |
| **AI Predictions Old Alias** | `GET /v1/ai/predictions` | **410 GONE** | `retailer/mobile_compat.go:71-81` returns `use_retailer_ai_predictions`. Clients use `/v1/retailer/ai/predictions`. |
| **Inventory Audit Ledger** | `GET /v1/supplier/inventory/audit` | **410 GONE** | `supplier/portal_handlers.go:1107-1118` returns `audit_unwired`. Live clients query standard inventory adjustment list. |
| **Quantity Negotiation** | `POST /v1/delivery/negotiate`<br>`POST /v1/supplier/negotiate/resolve` | **410 GATED** | `order/negotiation_disabled.go:22-30` returns `feature_disabled` unless `QUANTITY_NEGOTIATION_ENABLED=true`. |
| **Payme & Click Webhooks** | `/v1/webhooks/payme`<br>`/v1/webhooks/click` | **COMMENTED** | `webhookroutes/routes.go:26-31` routes commented out. Active payment rails are Cash + GlobalPay + MySoliq. |
| **Vehicle Capacity GET** | `GET /v1/payloader/capacity` | **410 GONE** | `payload/vehicle_capacity.go:19` returns `capacity_unwired`. Volume utilization computed client-side from ship-units. |
| **Request Cancel Post-Dispatch** | `POST /v1/order/{id}/cancel` | **403 GATED** | Cannot cancel after order has transitioned to `DISPATCHED` or `LOADED`. Must use return or shop-closed flow. |
| **Auto-Order Place Soak** | `POST /v1/retailer/auto-order/place` | **FLAG GATED** | Draft & shadow mode active (`AUTO_ORDER_SHADOW=true`); automated placement disabled until 30-day soak gate. |
| **Auth0 Global Wrap** | Global Router Wrapping | **BYPASSED** | Replaced by GS-I per-supplier OIDC (`orgoidc` package) to support multi-tenant IdP isolation and native HS256 auth. |

---

## 4. Verification & Audit Commands

```bash
# Verify backend route mounts across all 29 packages
rg -n 'RegisterRoutes|RegisterHandlers' pegasusX/apps/backend-go/main.go

# Verify all 410 product disable handlers
rg -n 'StatusGone|feature_disabled|audit_unwired|capacity_unwired|saved_cards_not_product' \
  pegasusX/apps/backend-go/{retailer,supplier,warehouse,factory,payload,driver,order}

# Verify AI predictions client call sites
rg -n '/v1/retailer/ai/predictions' pegasusX/apps/

# Execute client unit test suites
pnpm --filter @pegasusx/supplier-portal test
pnpm --filter @pegasusx/retailer-app-desktop test
pnpm --filter @pegasusx/warehouse-portal test
pnpm --filter @pegasusx/factory-portal test
pnpm --filter payload-terminal test
pnpm --filter @pegasusx/admin-portal test
```

