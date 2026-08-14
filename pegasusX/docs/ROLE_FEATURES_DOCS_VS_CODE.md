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

**P5 status (2026-08-13):** factory planning OS ported, **flags default off**. `FACTORY_PLANNING_ENABLED` / `FACTORY_BATCHER_ENABLED` env (not money flags). Dedicated `SupplyLanes` + `NetworkOptimizationMode` tables — **do not** hijack `GET /v1/supplier/supply-lanes` (still topology warehouse utilization). P5-D **PARTIAL** — `SYSTEM_PREDICTED` from `DemandForecastBaseline` SUM vs `SupplierInventoryV2`+`ReorderThreshold` (`factory/predictive_push.go`); **not** ORDER-grain `AIPredictions`. Gated by `PlanningEnabled()` (Go default **false**). Dispatch honesty: `pick_n_created_v1` when batcher off; `ffd_nn_lifo_v1` + `HEURISTIC` when on — **not** `dispatch.BinPack`.

**P6 status (2026-08-14 leftover close):** A+F remain. **B/C/E now in X** (payout-policy, entityresolution, SplitManifest naming). **D** UZ-only countrycfg (`checkout_reads_this: false`). **G** loyalty `410 loyalty_not_product`. CRM `Retailers.Email` when set. **Not cloud.**

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
| **P3-D** | `POST /v1/factory/dispatch` honesty | **Done** — `optimizer_class: HEURISTIC` + `dispatch_algo: pick_n_created_v1` when `FACTORY_BATCHER_ENABLED` is off. P5-F adds `ffd_nn_lifo_v1` behind that flag; still **not** `dispatch.BinPack`. |
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

`POST /v1/factory/dispatch` when `FACTORY_BATCHER_ENABLED`: this engine (503 if no Spanner planning); else pick-n + **no invent** (P7-B) + `pick_n_created_v1`.

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
| **P6-C** | **API 2026-08-14** — `entityresolution/` + `POST /v1/supplier/entity-resolution/{resolve,explain}` (RoleAdmin). Result returned directly (no `{status:ok}`). No native UI. |
| **P6-D** | **PARTIAL 2026-08-14** — `GET /v1/country-configs/{code}` UZ seed only; else 404 `country_not_supported`. `GET/PATCH /v1/supplier/country-overrides/{code}`. JSON includes `checkout_reads_this: false`. Checkout does **not** read this. |
| **P6-E** | **PARTIAL 2026-08-14** — `dispatch.ExpandOversizeRoutes` after capacity checks, before persist (`supplier/dispatch_execute.go`, `warehouse/dispatch_execute.go`). Chunks named `AUTO-{driver}-{ts}-A/B`. `MaxWaypointsPerManifest = 25`. |
| **P6-F** | **API 2026-08-13; clients P7-C PARTIAL** — table `FactorySupplyRequestQC`. Factory `GET/POST /v1/factory/supply-requests/{id}/qc`. Parent = `WarehouseSupplyRequests` (factory-scoped). POST does **not** change request `State` (P9-C accept-gate requires PASS). Missing QC row → 200 `{result:""}`. Warehouse GET + POST (P9-C / P11-C). |
| **P6-G** | **GONE 2026-08-14** — `GET /v1/retailer/loyalty/tier` → **410** `loyalty_not_product` (`retailer/mobile_compat.go:136`). Never a fake tier. `Orders.Rating` is queried in laborcapacity but **not in DDL** — do not “fix” that ghost. |

**Non-goals (P6 leftover):** live payout PSP; checkout reading countrycfg; fake loyalty tier; k-means return; Pegasus admin-portal as supplier UI. QC accept-gate + warehouse POST in P9/P11.

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

# Part II — Evidence baseline (2026-08-13)

**Question:** Do living docs match backend + clients?

**Compared:** FEATURES, ECOSYSTEM, ROLE_ROW matrix, Retail OS gate, `BACKEND_PARITY_*` (some P0s later fixed B1–B7).  
**Code:** `*routes/routes.go`, role packages, client nav, `main.go` mounts.

“Wired” on the matrix = happy-path Class A exists. It does **not** mean every FEATURES row is production-complete.

## Scorecard

| Role | Core loop | Docs vs code | Biggest lie / hole |
|------|-----------|--------------|--------------------|
| **Retailer** | Create → track → pay-at-delivery | Core REAL; OS packs coded | Saved-cards/AI-alias/correct **GONE** 410 (P1); loyalty **410** `loyalty_not_product`; B2B 410; CT tiles navigate (P13-E) |
| **Supplier** | Vet → dispatch → catalog/inventory | Core REAL | Inventory audit **GONE** 410 (P1); negotiations 410; S&OP column or env 700 (P10-B); CRM Email when set; payout-policy thin UI, rail `no_live_rail` |
| **Warehouse** | Dispatch execute + WMS | Dispatch REAL; WMS REAL after B2 | Treasury invoices query-or-503 (P2); analytics/financials honest availability (P11 gateway/fee query-or-false); demand scaffold only with seed; QC GET+POST (P9-C) |
| **Factory** | Loading-bay start/seal | Seal/start REAL under Spanner | Dispatch pick ≤2 **default** (`pick_n_created_v1`, **no invent** P7-B); FFD+NN+LIFO behind `FACTORY_BATCHER_ENABLED` (P5-F / P9-D overlay). Planning OS **flag-off default** (P5; env overlay may be on). QC **PARTIAL** portal+native + accept-gate (P7-C / P9-C) |
| **Payload** | Seal / inject / reassign | Seal REAL (`payloaderoutes` mounted) | Dual tables (Pegasus already had both); `seal-all` has terminal+Android+iOS clients (P13-A). Capacity 410. |
| **Driver** | Arrive → QR → cash/credit → complete | Doorstep REAL | `PATCH …/state` 501; history Spanner 30d (P2) |
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

**PARTIAL:** `pending-payments` lists real orders; `session_id`/`gateway` only from PaymentSessions (P2).

### Payment — pay after offload

| Path | Docs | Code |
|------|------|------|
| Unified `items[]` | Checkout | **Creates** order (no capture) |
| Unified `order_id` / B2B | Card/B2B | **410** `payment_before_delivery_removed` |
| `POST /v1/order/card-checkout` | Card | **REAL** session + redirect |
| `POST /v1/order/cash-checkout` | Cash | **REAL** `PENDING_CASH_COLLECTION` |
| `/v1/retailer/card*` | Saved cards | **GONE** `410 saved_cards_not_product` (P1) |
| Credit | Pay on credit | Read + driver credit-leave; no retailer “pay AR” |
| Fiscal | OFD | Pegasus commercial receipts; Soliq not default |

### Rest of FEATURES §1

Auth, catalog, cart, suppliers add/remove (durable favorites, **no** operating-schedule model), cancel pre-dispatch, preorder, shop-closed, claims, pulse, Retail OS packs: **REAL** (POS sale not one txn). Auto-order draft REAL; **place** PARTIAL (flag).  
`GET /v1/ai/predictions` **GONE** `410 use_retailer_ai_predictions` (P1; alias kept for old store builds). Real list is `/v1/retailer/ai/predictions` — desktop/Android/iOS bind `{items}` (P4). Correct PATCH **GONE**. Preorder POST **410**. Loyalty **GONE** `410 loyalty_not_product` (P6-G). Request-cancel **403**. Offline POS deferred.

---

## II.2 Supplier (`ADMIN`)

Portal + Tauri; Android/iOS exist; **no** `supplier-app-desktop`.

**REAL:** register/login/topology/org-fleet, vet, dispatch execute (same engine as WH), catalog, inventory adjust + import, pricing/promos, returns, claims, credit, control tower, shop-closed resolve.

| Feature | Status |
|---------|--------|
| Inventory audit | **GONE** `410 audit_unwired` (P1) |
| Negotiations | **410** |
| Broadcast | PARTIAL — persist `OpsBroadcasts` + outbox then WS (P10-A) |
| S&OP | PARTIAL — `Factories.DailyOutputCapacity` when column sum > 0 (`capacity_source: factories_column`); else `env_default` (`SOP_FACTORY_DAILY_UNITS` default 700). New factories default 700 (P10-B). |
| MEIO | REAL heuristic `cost_aware_v2` / `greedy_capital_v1` |
| Factory planning APIs | **PARTIAL** — `GET/PUT /v1/supplier/network-mode` (includes `planning_enabled`), `POST …/planning/pull-matrix`, `POST …/planning/kill-switch`. Portal `/settings/planning` factory-ops panel + native PlanningSettings (P7-C). Engines no-op unless `FACTORY_PLANNING_ENABLED` (Go default still false; env examples/local set true). `GET /v1/supplier/supply-lanes` **unchanged**. |
| Supplier CRM | **PARTIAL** portal+native (P7-C) — `GET /v1/supplier/crm/retailers` + `/{retailerId}` (P6-A). `Orders.Status` / `TotalMinor`; `Retailers.Email` when set. Empty `[]`; 503 if no Spanner. **Not** warehouse `/ops/crm`. Not App Store. |
| Country overrides / entity resolution / payout-policy | **PARTIAL 2026-08-14** — countrycfg UZ seed + supplier overrides (`checkout_reads_this: false`). Entity-resolution resolve/explain API. Payout-policy GET/PATCH + thin portal/native; batches bank-file; live dispatch `no_live_rail`. |

Portal-only (no Android): none of the P13-B hrefs remain portal-only. Control-tower + playbooks are typed lists; other P13-B slices may still dump JSON. Planning settings + CRM + payouts exist on portal **and** native (P7-C). Retailer Control Tower tiles navigate (P13-E).

---

## II.3 Warehouse

Dispatch execute = strongest mutator (idempotency + outbox + freeze). WMS REAL after B2. Transfers/supply/returns REAL.

| Feature | Status |
|---------|--------|
| CRM / staff / payment-config | REAL — warehouse CRM JSON (`business_name` / `total_orders` / `total_revenue` / `last_order_date`) **unchanged**. Portal+native last_order + load-error honesty (P7-C). |
| Treasury invoices | **PARTIAL** — ArInvoices⋈Orders; empty only if query empty; 503 if Spanner missing (P2) |
| Analytics top_products / daily | **PARTIAL** — `loadOpsAnalytics` when Spanner OK; fallback `*_available: false` (P2) |
| Financials gateway/daily | daily_revenue from Orders when Spanner; `gateway_breakdown` from `PaymentSessions` ⋈ warehouse orders when query OK; `platform_fee` from `BillingFeeSchedules` when a schedule exists; else `*_available: false` (P11-A) |
| Demand forecast | Spanner first (`source: spanner`); scaffold only `WAREHOUSE_PORTAL_SEED`; else `source: empty` (P2) |
| Factory QC | **PARTIAL** GET+POST (P9-C / P11-C) — warehouse `GET/POST /v1/warehouse/supply-requests/{id}/qc` on portal detail + Android/iOS. |

---

## II.4 Factory

Loading-bay start/seal/complete/rebalance: REAL under Spanner + payload JWT on bay routes.

`POST /v1/factory/dispatch` **default (flag off):** pick ≤2 `CREATED` transfers, first driver/vehicle, DRAFT, **no invent-if-empty** (P7-B) — empty queue returns `created_manifest_count: 0` with no outbox. Labels `optimizer_class=HEURISTIC` + `dispatch_algo=pick_n_created_v1`. **When `FACTORY_BATCHER_ENABLED`:** FFD+NN+LIFO → `FactoryTruckManifests` `DRAFT`, **no invent-if-empty**, `dispatch_algo=ffd_nn_lifo_v1`. **Not** warehouse VRP / `dispatch.BinPack`.

Supply-request accept REAL. G7 SLA board = **request due-date** (`kind: supply_request`). Transfer-transit 1×/1.5×/2× is a **second** worker (`kind: transfer_transit`) behind `FACTORY_PLANNING_ENABLED`.

Supply-request QC **PARTIAL** portal+native (P7-C / P9-C) — factory `GET/POST /v1/factory/supply-requests/{id}/qc` on board/list + Android/iOS cards. Table `FactorySupplyRequestQC`. POST does not change `WarehouseSupplyRequests.State`. Accept **409** unless QC `PASS`. Outbox `FACTORY_SUPPLY_REQUEST_UPDATE`. Missing QC row → 200 `{result:""}`.

Staff POST **PARTIAL** — `SupplierUsers` + `FACTORY_STAFF_CREATED` (P3). Password is bcrypt or invite (P9-A), never `"unset"`. Set-password UI portal+native (P13-D). Exception GET **PARTIAL** — Spanner `ManifestExceptions` ⋈ `FactoryTruckManifests` (P7-A); `transfer_id` ← `OrderId`; seed overlay only with portal seed. Resolve **PARTIAL** — Spanner-first (P9-B); memory only with seed; `RunTx` + outbox (P3). Transfer create emits `TRANSFER_CREATED`. Transfer GET Spanner-first.

Planning OS **ported, flags default off** (P5-0…G). Env examples/local set `FACTORY_PLANNING_ENABLED=true` (P7-C) and `FACTORY_BATCHER_ENABLED=true` (P9-D); Go default still false. P5-D **PARTIAL** — `DemandForecastBaseline` grain, not `AIPredictions`. Factory-ops panel on supplier `/settings/planning` + native PlanningSettings — **not** `/planning` S&OP. **Not cloud. Not store.**

Insights: factory clients call **warehouse** replenishment insights — intentional.

---

## II.5 Payload

`main.go` mounts **`payloaderoutes`** (richer). Dual plane = `FactoryTruckManifests` vs `SupplierTruckManifests` — **Pegasus already had both**; X added `manifest_domain` + payload package extras (load ledger, seal-all, reassign, ship-units).

Seal requires `manifest_id`. `POST /v1/fleet/reassign` and `POST /v1/payloader/reassign-order` persist `Orders.RouteId`/`DriverId` in the same txn as outbox (P4). `seal-all`: REAL persist; terminal + Android + iOS call sites (P13-A). Capacity GET **GONE** `410 capacity_unwired`.

---

## II.6 Driver

Doorstep REAL on `orderroutes`. Arrive: no GPS. Partial/complete: GPS. Cash/credit: stable idempotency. Depart/return 503 if fn nil.

`PATCH …/state` **501**. Mid-delivery update **not_implemented**. Negotiate **410**. History **Spanner Orders** (30d completed window, P2). Earnings REAL if wired.

X is strictly ahead of Pegasus on driver (Pegasus had no `driver/` package).

---

## II.7 Platform admin

Login+MFA, tenants, flags dual-control, partner, outbox + **dead-letters**: REAL. Honest `{available:false}` without Spanner. Billing **PARTIAL** — `GET /v1/admin/billing/invoices` + fee-schedules + portal tab (P12); CronJob YAML unapplied; worker still needs `AR_INVOICES_ENABLED`.

---

## II.8 Product disables (all docs must say)

| Feature | Behavior |
|---------|----------|
| Quantity negotiation | 410 |
| Pre-delivery card/B2B | 410 |
| Saved cards `/v1/retailer/card*` | 410 `saved_cards_not_product` |
| `GET /v1/ai/predictions` alias | 410 `use_retailer_ai_predictions` |
| Request-cancel after dispatch | 403 |
| Soliq OFD | Not default |
| Auto-order place | Flag off |
| Airwallex | Flag off |

---

## II.9 What living docs get wrong

**P0 honesty pass 2026-08-13** tagged FEATURES, appended the matrix “Wired = happy path” footnote, stated factory dispatch ≠ warehouse VRP in ECOSYSTEM, and stamped STALE banners on the 2026-08-12 payload/retailer audits. Residual code holes below are **P1+**, not remaining doc ads.

1. Matrix “Wired” ≠ every FEATURES row — **footnote added (P0-B)**.  
2. FEATURES listed B2B, cards, negotiations, request-cancel, inventory audit, `/v1/ai/predictions` as live — **P0 tagged; P1 cards/AI-alias/audit are 410 GONE**.  
3. `BACKEND_PARITY_PAYLOAD` (2026-08-12): `payloaderoutes` **is** mounted — **STALE banner (P0-D)**.  
4. Retailer cash checkout ack-only audit is **stale** (B1) — **STALE banner (P0-D)**.  
5. ECOSYSTEM optimizer — already OR-Tools **or** H3; factory dispatch now explicitly **not** warehouse VRP (P0-C).  
6. Warehouse analytics “always `[]`” in this Part II was **overstated** — Spanner path fills `top_products` / `daily_breakdown`.

---

## II.10 Re-verify

```bash
rg -n 'RegisterRoutes' pegasusX/apps/backend-go/main.go

rg -n 'entries.: \[\]any|cards.: \[\]any|StatusGone|order_service_unwired|not_implemented' \
  pegasusX/apps/backend-go/{retailer,supplier,warehouse,factory,payload,driver,order,payment}

rg -n '/v1/ai/predictions|/v1/retailer/ai/predictions' pegasusX/apps
```

Do not plan from frozen `.docx`. Use `pegasus/` only as a **port source** for P5/P6.
