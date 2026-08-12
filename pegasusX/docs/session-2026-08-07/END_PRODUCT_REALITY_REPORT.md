# 0. Executive Verdict

> **HISTORICAL / FROZEN — do not plan from this file alone.**
> Current SoT: [`PROD_READINESS_SEQUENCE.md`](../PROD_READINESS_SEQUENCE.md) · [`ECOSYSTEM_GAP_REGISTER_2026-08-12.md`](./ECOSYSTEM_GAP_REGISTER_2026-08-12.md) · [`FEATURES_BY_APP_ROLE.md`](../FEATURES_BY_APP_ROLE.md).
> Body is a point-in-time snapshot; re-verify any claim against code before acting.


**PegasusX is not a prototype and not "just a CRUD app."** It is a ~214k-line Go transactional backend (`apps/backend-go`, 812 non-test files, 644 HTTP handler functions), a 155-table Spanner schema (`apps/backend-go/schema/spanner.ddl`) with 87 incremental migrations, 12 native mobile apps and 6 web surfaces, a hand-written 164-method typed API client (`packages/api-client/index.ts`), a real transactional outbox that commits event rows inside the same Spanner transaction as the state mutation (`apps/backend-go/order/repository_spanner.go:28-38`), optimistic concurrency with in-transaction compare-and-swap on `Version` (`apps/backend-go/order/repository_spanner.go:208-215`), money carried as integer minor units with zero float money in any money path, and a real, event-driven integration layer (partner API keys + OAuth2, EDI-lite, AS2 with PKCS#7 crypto, SFTP, GS1, 1C-style journal export). This is the work of engineers who have operated real transactional systems.

**But it is not yet the system the product narrative claims.** Five structural facts dominate everything else in this report:

1. **It is a single-supplier system at runtime, by construction.** The schema is multi-tenant-shaped (`SupplierId` leads most keys) but `bootstrap/bootstrap.go` injects one seed `SupplierID` into ~20 service constructors at process start, and `order.Service` holds it as a private field (`apps/backend-go/order/service.go:352`). Registering a second supplier mints a tenant the request plane cannot isolate (`apps/backend-go/supplier/service.go:442-456`). The data plane can hold ~10 suppliers; the request plane serves exactly 1. Multi-tenancy runtime score: **1/10**. A 12-week Phase-1 migration plan is accepted but uncoded (`docs/MULTI_TENANCY_GATE5_PHASE1.md`).

2. **There is zero machine learning, and the "AI" layer is arithmetic — though it is now honest arithmetic.** The only Python dependency in the solver service is `ortools==9.15.6755` (`services/optimizer-core/requirements.txt`). Forecasting is real classical statistics (Holt-Winters / Croston-SBA / SES with Syntetos-Boylan-Croston classification, `apps/backend-go/planning/forecast/`), safety stock is the correct service-level formula, and the auto-order loop is inventory-grounded — but the legacy `last_order/2` synthesis path still ships as the non-grounded fallback (`apps/ai-worker/synthesis/engine.go:331-343`). No LLM, no CV, no embeddings anywhere; the marketing word "AI" maps to heuristics and SQL pattern mining (`apps/ai-worker/predictivepush/analyzer.go:36-66`).

3. **The platform sells automation that ships turned off.** The highest-value capabilities are flag-gated and default-off: AR invoices and dunning (`AR_INVOICES_ENABLED`, `AR_DUNNING_ENABLED`), safety-stock v2 (`SAFETY_STOCK_V2_ENABLED`), forecast algorithm (`FORECAST_ALGO_ENABLED`), touchless auto-order (`execution_mode` defaults `off`), WMS lots/FEFO/pick-waves/cycle-counts/cold-chain (`WMS_*_ENABLED`), AS2 and SFTP transports (default off and absent from shipped manifests). A stock deployment of the current tree does substantially less than the feature list implies.

4. **Two live correctness bugs sit on the money path, and they are worse than anything the repo's own audit tracked.** (a) Card capture is permanently broken: `CaptureCardPayment` routes to gateway `"GLOBALPAY"` (`apps/backend-go/payment/service.go:653`) while the executor map is keyed `"GLOBAL_PAY"` (`apps/backend-go/payment/execution.go:140`), so every capture call fails — and the order leg is pre-recorded as `CAPTURED` inside the transaction before the PSP call, which then runs fire-and-forget with log-only error handling (`apps/backend-go/order/service.go:1899-1929`). The ledger asserts money moved that never did. (b) The legal fiscal provider cannot work as shipped: the `MY_SOLIQ` adapter's `signer` field is never injected anywhere in the codebase (`apps/backend-go/order/fiscal_provider.go:129`), so enabling `FISCAL_PROVIDER=MY_SOLIQ` yields 100% `FISCAL_FAILED`; the default `PEGASUS` provider issues platform commercial receipts explicitly marked `"tax_ofd": false` (`apps/backend-go/order/fiscal_provider_pegasus.go:13-15,78-79`).

5. **Status integrity is entirely application-level.** Across 155 tables there is exactly one CHECK constraint (`spanner.ddl:1273`). The canonical 18-state order machine (`apps/backend-go/order/state_machine.go:14-81`) is enforced at only 4 call sites (`order/service.go:1523`, `order/service.go:2153`, `order/preorder_sweeper.go:168,241`) while ~65 direct status writes exist across packages; the known-bad paths have hand-rolled guards today, but nothing structural prevents the next ad-hoc writer from creating illegal transitions.

**Can it replace the field sales agent?** It replaces the agent's **order pad and decision loop**, not the agent. The commercial loop (demand signal → proposal → confirmation → credit check → pricing → allocation → dispatch) is genuinely automatable with flags that exist in code today — a store can receive an auto-generated, auto-accepted, auto-dispatched replenishment order with zero human touches when `execution_mode=place` and supplier touchless policies are enabled. But picking, loading, driving, the delivery handshake, cash collection, and every exception path terminate in a human, and the two structural blockers (COD cash collection, off-app dunning) are not gaps that more code alone removes. Honest number: **~35–40% of the agent's job is automatable with what exists today (default config: less, because the autonomy flags ship off), ~60–65% with the P1 gaps closed, and full wipe-out is not on any realistic horizon — the trajectory is a hybrid reduction, not an elimination.**

**Scorecard (this report's assessment, code-grounded):**

| Layer | Score | One-line justification |
|---|---|---|
| Transactional core (outbox, concurrency, money, idempotency) | 8/10 | Same-txn outbox with leased relay and Kafka dedupe; integer money with `math/big` FX; minus: payment-leg idempotency lacks DB uniqueness, capture bug |
| Order & logistics domain model | 8.5/10 | 18-state FSM, two-sided delivery negotiation, volumetric dispatch, COD/credit/split legs, claims ↔ quarantine ↔ reverse logistics |
| Payments correctness | 4/10 | Global Pay integration is real with out-of-band webhook re-verification, but capture routing is broken, the leg is optimistically pre-recorded, and empty-credential stub-success paths fabricate captures/refunds |
| Fiscal / legal compliance | 3/10 | Rigorous FISCALIZING framework around a default provider that is not a tax fiscalizer; legal Soliq path unwired (no signer injection) |
| Credit / AR / collections | 5/10 | Terms, AR open items, aging buckets, dunning step machine, auto-hold all implemented — flag-gated off; scoring deliberately removed; no SMS/email reach |
| Planning algorithms (forecast, safety stock, auto-order) | 6/10 | Real classical statistics wired end-to-end with accuracy tracking; v2 paths flag-gated; no ML |
| Routing / dispatch optimization | 5/10 | Real OR-Tools VRP solver with constraint fidelity built — deployed at `replicas: 0`; live path is H3+bin-pack+2-opt heuristic with Haversine ETAs |
| Integration surface | 7/10 | Partner API + OAuth + EDI-lite + AS2 + SFTP + GS1 + 1C journals are real code with tests; REST order-create not idempotent, no master-data push, webhooks cover 4 of 155 events, Kafka RF=1 |
| Client applications (12 native + 4 portals + desktop) | 8/10 | Almost everything is wired to the backend with real offline queues; the failures are specific: dead warehouse scanner stub, portal-only WMS execution, no supplier offline, no admin console |
| Multi-tenancy (runtime) | 1/10 | One supplier, bound at startup |
| Platform admin / operations console | 2/10 | admin-portal is a retired redirect stub; ~17 `/v1/admin/*` endpoints exist but no tenant lifecycle, no console UI |

---

# 1. Evidence Base, Method, and Scale

**Evidence rule.** Every claim about current implementation in this report traces to code, schema, or configuration in the live tree at `/Users/shakhzod/Desktop/V.O.I.D/pegasusX` (git HEAD `7ee59ba9`, 2026-08-07). Five independent code audits were executed for this report (backend correctness; per-role client apps; integration surface; algorithms/planning; and a documentation-vs-code cross-check), supplemented by direct spot verification. No deleted documentation was used; surviving repo documentation was treated as a claim source to be verified, not as evidence.

**A note on the repo's own documentation.** The tree contains an unusually self-critical audit corpus (`PLATFORM_AUDIT.md`, `context/current_status.md`, `docs/SUBSTANCE_GATE.md`, gap ledgers). Cross-checking found it **mostly accurate at the moment each entry was written** — ~30 marquee claims were re-verified in code — but with three failure modes the reader must know:

- **Temporal layering.** Status banners were updated 2026-08-05→07 while body sections were not. `PLATFORM_AUDIT.md:21` still asserts "no machine-to-machine integration surface at all" while its own §8.9 documents the shipped partner stack; `CREDIT_COLLECTIONS_ENGINE_PLAN.md`'s "current state" verdict predates its own implementation; three gap-closure runbooks (`docs/gap-closure/STAGING_FLAGS.md` step 3, `PRODUCTION_CUTOVER.md`, `STAGING_FOUNDATION.md`) instruct operators to enable `CREDIT_SCORE_ENFORCEMENT_ENABLED`, a flag with **zero references in Go code** (scoring was deliberately removed).
- **Evidence integrity.** The Substance Gate's load-bearing "marker gate PASS" citation (`artifacts/ssmr-e2e-substance-gate-2026-08-04.log`) **does not exist** on disk; role×client "Wired" matrices contradict the project's own Substance Gate signoff where every client cell is DEFERRED; the Gate-0 CI workflows live at the monorepo parent (`.github/workflows/pegasusx-ci.yml`) not where the audit links them.
- **Scale understatement.** Docs cite "411 endpoints / 73 tables / 131,670 Go lines"; measured today: ~731 route registrations, 155 `CREATE TABLE` statements, ~213,719 Go lines in `apps/backend-go` alone, 6,368 source files across the monorepo. The tree is growing fast and the docs lag it.

**Measured scale (this report's measurements):**

| Dimension | Measured | Where |
|---|---|---|
| Go backend | ~213,719 LOC, 1,077 files (812 non-test / 265 test, ~25% test ratio), 644 handler funcs | `apps/backend-go` |
| HTTP surface | ~731 route registrations across role routers + partner API | `apps/backend-go/*routes/`, `main.go:145-400` |
| Schema | 155 tables, 2,624-line DDL + 87 migrations; 13 unique indexes; 1 CHECK constraint | `apps/backend-go/schema/` |
| Mobile | Kotlin ~694 files, Swift ~635 real files (retailer-iOS inflated by ~1,380 vendored SPM checkouts under a typo'd dir `retailerapp/reatilerapp/`) | `apps/*-app-android`, `apps/*-app-ios` |
| Web/desktop | ~815 TS/TSX files across 4 Next.js portals (Tauri-wrapped), retailer desktop, marketing site | `apps/*-portal`, `apps/retailer-app-desktop` |
| Shared typed client | 164 methods, 255 unique `/v1/*` paths, hand-written (not generated) | `packages/api-client/index.ts` |
| Event catalog | 155 event types declared in `contracts/events.schema.json`; 148 referenced from real code | `contracts/events.schema.json`, `apps/backend-go/events/events.go` |
| Deployed staging | GKE `pegasusx-ssmr` live behind `https://api-ssmr.pegasusx.app` with TLS; Spanner migrations applied; smoke PASS 2026-08-04 | `context/current_status.md` §2 |

**Verification status vocabulary used throughout:** **WIRED-LIVE** (code exists, is invoked in a real request/worker path, defaults on or deployed on) · **FLAG-GATED** (implemented and wired, but default-off in shipped config) · **PARTIAL** (real but materially incomplete) · **DECORATIVE** (interface without substance: stub, mock, dead screen) · **ABSENT** (no code).

# 2. Human / Field-Agent Displacement

## 2.1 The 22-step reality check

The field agent's job, decomposed from need-detection to cash in the bank. "Automatable" means a flag, policy, sweeper, or worker exists in code today; each row cites its evidence.

| # | Step | Human needed? | Status today | Evidence |
|---|---|---|---|---|
| 1 | Detect the reorder need | No | **Automated (FLAG-GATED quality)** — demand sensing worker + forecast baseline + predictive push; forecast algo flag-gated off by default | `apps/backend-go/demand/worker_sensing.go`; `apps/backend-go/planning/forecast/`; `apps/ai-worker/predictivepush/analyzer.go:36-66` |
| 2 | Retailer confirms the suggestion | Default yes | **Automatable (FLAG-GATED)** — per-scope auto-order toggles (global/category/supplier/product/variant) + `off\|shadow\|draft\|place` execution mode; default `off` | `apps/backend-go/retailer/auto_order_worker.go`; `retailer/auto_order_policy.go` |
| 3 | Supplier accepts the order | Default yes | **Automatable** — midnight-guard sweeper promotes `SCHEDULED → AUTO_ACCEPTED` | `apps/backend-go/order/preorder_sweeper.go:168` |
| 4 | Credit decision | No | **Automated at placement — limit + status only** (scoring deliberately removed; `RiskTier` blanked) | `apps/backend-go/credit/service.go:49-94` |
| 5 | Price determination | No | **Automated** — override → promotion (basis-points) → price list | `apps/backend-go/promotion/evaluator.go`; `pricing/service.go` |
| 6 | Stock allocation / backorder | No | **PARTIAL** — FEFO/FIFO lot reservation + constrained warehouse selection; **no partial allocation or backorder queue: insufficient stock is a hard error (lost sale)** | `apps/backend-go/stocklots/fefo.go`; `allocation/constrained.go`; `order/inventory_reservation.go:65-92` |
| 7 | Delivery date agreement | Sometimes | **PARTIAL** — auto by default; `NegotiationProposals` two-sided negotiation exists but is flag-gated/product-deferred | `apps/backend-go/order/negotiation.go` |
| 8 | Transfer approval | Sometimes | **Automatable (FLAG-GATED)** — touchless with daily unit budget; CRITICAL urgency escalates by design | `apps/backend-go/replenishment/touchless.go`; `replenishment/policies.go` |
| 9 | Dispatch planning & driver assignment | Optional | **Automatable** — 60s auto-dispatch worker, real closed loop; solver is heuristic in prod (OR-Tools sidecar at 0 replicas) | `apps/backend-go/warehouse/auto_dispatch.go:28,120-131`; `infra/k8s/overlays/prod/kustomization.yaml:44-50` |
| 10 | Physical picking | **YES** | **Software-assisted, portal-only** — pick waves + FEFO + seal gate exist in backend and warehouse-portal; **warehouse mobile apps cannot execute picks**; the Android scanner is a dead stub | `apps/backend-go/stocklots/picking.go`; `warehouse-portal/app/pick-waves/page.tsx`; `warehouse-app-android/.../ScannerViewModel.kt:22,47` |
| 11 | Truck loading | **YES** | Human-driven by design — a 38-endpoint payloader API + loading-bay terminal exist so a human can drive it | `apps/backend-go/payload*/`; `apps/payload-app-android/.../PayloadApi.kt` |
| 12 | Manifest sealing | Yes | PARTIAL — `seal-all` batches the clicks; SSCC minted per ship unit | `packages/api-client` `/v1/payloader/manifests/seal-all`; `gs1/checkdigit.go:142-171` |
| 13 | Driving | **YES** | — | — |
| 14 | QR handshake at the store | **YES — by design** | Geofence auto-detects arrival; the handshake is an intentional two-party control, not a gap | `apps/backend-go/geolocation/`; driver apps |
| 15 | Offload & condition verification | **YES** | Human-reported at dock; post-delivery damage/shortage/concealed file as claims → QUARANTINE + reverse logistics | `apps/backend-go/claims/`; `stocklots/coldchain.go` |
| 16 | Cash collection | **YES — structural** | The most developed manual path in the platform: server-computed expected cash vs declared, accept/write-off, shift-close gate, nightly escalation | `apps/backend-go/cashrecon/service.go:39-161`; `cashrecon/escalation_worker.go` |
| 17 | Card capture | No | **BROKEN TODAY (P0 bug)** — capture routing key mismatch; leg pre-recorded CAPTURED; stub-success when creds empty | `payment/service.go:653` vs `payment/execution.go:140`; `order/service.go:1899-1929`; `payment/global_pay_executor.go:251-258` |
| 18 | Fiscal receipt | No | **Automated shape; NOT legal** — PEGASUS provider issues commercial receipts (`"tax_ofd": false`); legal Soliq OFD adapter exists but its signer is never injected (100% failure if enabled) | `order/fiscal_provider_pegasus.go:78-79`; `order/fiscal_provider.go:129,232-234` |
| 19 | Reconciliation | On exception | PARTIAL — detection automated (cashrecon, settlement exceptions), resolution manual | `apps/backend-go/cashrecon/`; `order/settlement_hardening.go` |
| 20 | Returns disposition | Yes | **Strong** — FileClaim + approve/reject + stock hold + reverse open; human confirms disposition | `apps/backend-go/claims/service.go`; `returns/` |
| 21 | Exceptions (shop closed, delay, overflow, rescue) | **YES** | Every path ends in a human decision; shop-closed timeout worker exists but can mis-record credit debt (P0 risk #4) | `order/worker_shop_closed.go:91-165` |
| 22 | Dunning / collections | Partial | **In-app wired (FLAG-GATED)** — step machine DUE_SOON→…→COLLECTIONS, auto CREDIT_HOLD, FCM+inbox; **off-app SMS/email/WhatsApp absent** | `apps/backend-go/ar/dunning.go:41-74`; `ar/dunning_worker.go` |

## 2.2 What is automatable today — the exact zero-touch path

The complete, code-verified chain for a store replenishment order with no human touch:

1. `demand.RunDemandSensingWorker` (scheduled) computes velocities from order history with day-of-week/payday/signal factors — including a real Open-Meteo weather ingest (`apps/backend-go/demand/worker_weather.go:97-138`) — and upserts `DemandAdjustments`.
2. The after-sensing hook (`apps/backend-go/bootstrap/bootstrap.go:1535`) runs `replenishment.RunBatch` (`replenishment/reorder_suggestion_batch.go`), computing suggested quantities from adjustments + safety stock (service-level formula when `SAFETY_STOCK_V2_ENABLED`).
3. `retailer.RunAutoOrderWorker` (`retailer/auto_order_worker.go`) loads inventory-grounded (R,s,S) proposals from `RetailerStockBalances`, applies the scope policy filter, and — **only when `execution_mode=place`** — writes a real order via `order.Service.Create`.
4. Credit gate runs inline at creation (`credit/service.go:49-94`); inventory is reserved in the same transaction (`order/inventory_reservation.go:65-92`).
5. Supplier-side, `replenishment/touchless.go` auto-approves within policy caps; the 60s auto-dispatch worker commits manifests (`warehouse/auto_dispatch.go:85-90`).

**Where the chain breaks by default:** `execution_mode` defaults to `off`; `draft` requires human confirmation; supplier touchless requires `AutoApproveEnabled` + confidence floor; the forecast algorithm and safety-stock v2 flags are off; and after dispatch, the physical world takes over (steps 10–16 above).

## 2.3 What remains a hard human requirement

- **Physical execution**: picking, loading, driving, offload verification. The platform's correct design choice is to *instrument* these humans (dedicated payload terminal, driver apps, geofenced handshake), not pretend them away.
- **Cash collection**: in a COD-dominant market the driver *is* the collections function. The platform converts that human from sales to logistics; it does not remove him.
- **Relationship commerce**: negotiating shelf space, promo calendars, dispute diplomacy, and reading a store the way a rep does. The negotiation primitive exists for delivery dates only; price/assortment negotiation is absent.
- **Off-app collections**: retailers without the app are unreachable by the dunning engine (no SMS/email/WhatsApp transports).

## 2.4 Realistic trajectory

**Near-term (6–12 months, assuming P0/P1 gaps in §8 close):** the honest product is **"replace the order pad, instrument the rest."** For app-adopting urban retailers, routine replenishment visits largely disappear; the field force re-composes toward exception handling, onboarding, collections, and relationship management. Expect headcount *shift* (sales → logistics/collections), not headcount deletion.

**3–5 year view (if all gaps close):** for the routine-replenishment slice — which is the majority of agent visit volume in FMCG — touchless operation is credible: forecast + inventory-grounded auto-order with proven shadow acceptance, touchless supplier approval, optimized dispatch, fiscalized completion, AR dunning with off-app reach. Even then: cash collection remains structural where COD persists; new-store acquisition, key-account negotiation, and exception recovery remain human. **Realistic outcome is a 50–70% reduction of routine field visits for covered retailers — a hybrid reduction, not a wipe-out.** The market evidence agrees: the best-funded B2B commerce platforms (MaxAB-Wasoko, Udaan, Jumbotail) all converged on monetizing **credit and data on top of instrumented transactions**, not on eliminating the field layer entirely.

# 3. Problem Coverage vs Existing Logistics / Planning Software

## 3.1 Capability-by-capability comparison

Capabilities are marked from PegasusX's **code-verified** state today (W = wired-live, F = flag-gated real code, P = partial/heuristic, D = decorative, – = absent). Enterprise columns reflect the public 2026 state of each category.

| Capability | O9 / Kinaxis | Blue Yonder | ERP+WMS stack (SAP/1C+best-of-breed) | Pure B2B marketplace | **PegasusX today** |
|---|---|---|---|---|---|
| Demand forecasting (ML-grade, multi-signal) | ● Leader | ● Leader | ◐ module | ◐ basic | **F** — classical stats only (HW/Croston/SES + classification + WAPE accuracy); no ML, no POS feed |
| S&OP / IBP (scenario planning, financial reconciliation) | ● | ◐ | ◐ | – | **D** — `GetSAndOP` returns `factories × 700 × 7` (`planning/service.go:252`); `projectStockouts` returns literal strings `sku-projection-1/2` |
| Multi-echelon inventory optimization | ● | ● | ◐ | – | **P** — `RunMEIONetwork` is a two-node greedy donor/receiver swap per SKU (`replenishment/mei_engine.go:168`); echelon targets heuristic |
| Safety stock / service-level inventory | ● | ● | ● | – | **F** — correct `SS = z·√(Lσ_d² + d̄²σ_L²)` with residual-σ loop, flag-gated; legacy heuristic default |
| Warehouse management (lots/FEFO, waves, counts, cold chain) | – (integrates) | ● native WMS | ● | – | **F** — real backend (`stocklots/`: FEFO, pick waves + seal gate, cycle counts ABC, cold-chain quarantine) but **portal-only execution; mobile floor apps can't run it** |
| Transportation / routing optimization | – (integrates) | ● native TMS | ◐ | ◐ 3PL | **P** — OR-Tools VRP with time windows/cold-chain/hazmat constraints built but **0 replicas in prod**; live = H3+bin-pack+2-opt; Haversine ETAs |
| Order capture / B2B storefront | – | ◐ OMS | ◐ | ● | **W** — native apps per role, quoted checkout, idempotency, offline queues |
| Last-mile execution (driver app, POD, geofence, cash) | – | ◐ | – | ◐ | **W** — the strongest layer: telemetry, POD photo/signature, COD reconciliation, rescue |
| Credit / collections / embedded finance | – | – | ◐ AR module | ● (survivors monetize it) | **F** — terms/AR/aging/dunning/auto-hold implemented, flags off; scoring removed; no off-app reach |
| Fiscal compliance (UZ OFD/Soliq) | – | – | ● (1C/local) | – | **P** — framework + hard gate real; legal provider non-functional as shipped; default = non-tax commercial receipt |
| Control tower / real-time visibility | ◐ via partners | ● | ◐ | – | **D/W split** — event-driven twin projection is real (`twin/consumer.go`), but the "live network" dashboard broadcasts **random mock data** (`simulator/control_tower.go:53-79`) |
| Integration (EDI/AS2/API/1C) | ● certified | ● certified | ● | ◐ | **F/P** — EDI-lite + AS2 (uncertified, shipped off) + partner API + 1C journal files; no certified EDIFACT/CommerceML |

## 3.2 What PegasusX actually solves today that the others do not

- **One transactional model from factory floor to shop shelf.** Factory manifests → inter-hub transfers → warehouse loading bay with a dedicated payloader role → volumetric truck packing (VU) → geofenced driver handshake → COD/split/credit delivery with ledger and cash reconciliation → claims/quarantine/reverse → fiscal receipt. O9 and Kinaxis explicitly have no native execution (they integrate to WMS/TMS); Blue Yonder has execution but nothing like the COD field layer, the payloader role, or CIS fiscal plumbing; marketplaces bolt onto 3PLs and never run a warehouse floor.
- **A native app per physical role**, all wired to the same backend: retailer (3 variants), supplier (3), driver (2), payloader (2 + terminal), factory (3), warehouse (3). No competitor in any adjacent category ships this role-complete a client set.
- **COD + credit + split-payment delivery as first-class transaction states** (`PENDING_CASH_COLLECTION`, `DELIVERED_ON_CREDIT`, `FISCALIZING`), with cash reconciliation gating driver shift close (`cashrecon/service.go:152-161`). This is the operational reality of the target market that O9/Kinaxis/BY do not model.

## 3.3 Where it falls short — honestly

- **Planning depth is not in the same league as O9/Kinaxis/BY.** No scenario planning, no concurrent replanning, no financial reconciliation of plans, no ML demand sensing. The S&OP surface is a decorative stub (`planning/service.go:212,252`). A CPG manufacturer would not buy this for planning.
- **Optimization is built-but-not-run.** The OR-Tools solver with constraint fidelity, multi-depot, and OSRM matrices exists in code and sits at `replicas: 0` in every shipped overlay (`infra/k8s/overlays/prod/kustomization.yaml:44-50`). Production routing is a competent heuristic, not optimization.
- **Warehouse execution caps the addressable market despite real backend progress.** Lots/FEFO/pick-waves/cycle-counts/cold-chain are flag-gated and **portal-only**; the floor worker's Android scanner is a dead stub that reports success without any API call (`ScannerViewModel.kt:22,47`, orphaned from navigation). Food/pharma-grade WMS is close in the backend and absent in the aisle.
- **It cannot yet be the system of record for anyone else's money.** Broken card capture, non-tax default receipts, and DB-unenforced payment idempotency (§7) disqualify it as a financial SoT until fixed.

## 3.4 The vertical-depth advantage, stated precisely

The defensible asset is not "a platform connecting suppliers and retailers" (that category exists and is crowded). It is the **single-stack vertical depth with one event bus**: 155 event types flowing from one transactional core to every role app and to the partner surface. For a large distributor who wants their factory, warehouse, fleet, field, and retail customers in one model — that is genuinely rare and is worth more as **deep software for one large distributor** than as a thin marketplace for many. Every adjacent category leader either plans without executing (O9/Kinaxis), executes without owning the B2B field layer (Blue Yonder), or sells without operating (marketplaces).

# 4. Alignment with Systems Big Retailers and Suppliers Already Run

A mid-size Uzbek/CIS distributor runs **1C** for accounting and often stock; a retail chain runs 1C or SAP plus a POS plus possibly a WMS. They will not re-key orders, will not run their inventory in someone else's database, and will not accept a system whose numbers their accountant cannot recognize. This section assesses the machine-to-machine reality from code.

## 4.1 Current state of the machine-to-machine surface

| Channel | Status | Evidence | Honest limitation |
|---|---|---|---|
| Partner REST API | **WIRED-LIVE** — all 20 documented endpoints implemented: orders create/get, catalog, availability, webhooks ×6, exports ×3, CoA ×2, AS2 ×3, EDI docs ×3 | `apps/backend-go/partner/routes.go:17-46`; `contracts/partner.openapi.yaml` | **Order create ignores its contract-required `Idempotency-Key`** (`partner/handlers.go:37-63` vs `partner.openapi.yaml:494-499`) — ERP retries can double-create orders. No amend/cancel endpoints. Catalog is read-only for machines |
| Machine auth | **WIRED-LIVE** — bcrypt `pxk_` API keys + OAuth2 `client_credentials` issuing 15-min HS256 JWTs, scope intersection, live revoke, per-key rate-limit actor | `partner/keys.go:17-35`; `partner/oauth_jwt.go:100-133`; `partner/auth.go:111-127` | HS256 shared-secret only; no mTLS/RS256/JWKS; `RateLimitClass` column exists but is never read |
| Outbound webhooks | **WIRED-LIVE** — HMAC-SHA256 signed, 8-attempt backoff, dead-letter + replay, portal self-service | `partner/delivery.go:34-98` | **Only 4 of 155 event types exposed**: `ORDER_CREATED`, `ORDER_STATUS_CHANGED`, `CLAIM_FILED`, `PAYMENT_CLEARED` (`partner/kafka_handler.go:26-31`). No DELIVERED/DESADV/return/invoice events |
| EDI | **WIRED-LIVE ("EDI-lite")** — real segment codec (UNA dialect), ORDERS inbound parser, ORDRSP/DESADV/INVOIC outbound builders, idempotent ingest on `(tenant, direction, type, external id)`, event-driven outbound | `partner/edi/segment.go:27-145`; `partner/edi_inbound.go:228-265`; `partner/edi_outbound.go:358-389` | Self-declared non-certified dialect; **no CONTRL/APERAK functional acknowledgment** (one-legged loop); zero X12 |
| AS2 | **PARTIAL** — real PKCS#7 sign/encrypt + sync MDN, per-tenant station config, certs via SecretRef | `partner/as2/crypto.go:76-142`; `partner/as2_receive.go:14-88` | Not Drummond-certified; sync MDN only; **default off and absent from shipped manifests** (`partner/as2_flags.go:8-12`; no `PARTNER_*` keys in `kustomize.yaml` configmap) |
| Bulk export / SFTP | **WIRED-LIVE (export)** / **FLAG-GATED (SFTP off)** — CSV/JSON/XML async jobs → GCS signed URL or SFTP push | `partner/export_worker.go:230-249`; `partner/sftp.go` | SFTP default off; host-key verification disabled (`ssh.InsecureIgnoreHostKey()`, `partner/sftp.go:67,136`) |
| 1C accounting | **PARTIAL** — double-entry journal export merging AR + payment ledgers, 1C-style default accounts (Dt 62.01/Kt 90.01; Dt 51.01/Kt 62.01), per-tenant configurable CoA, XML `dialect="1c"` | `partner/export_journals.go:32-57,90-261`; `partner/coa.go:13-17` | No CommerceML 2.x exchange package; only AR-open and cash-settle legs (no VAT breakout, no multi-leg, no COGS); a 1C scheduler cannot consume it without an import script |
| GS1 | **WIRED-LIVE** — GTIN-8/12/13/14 + GLN + SSCC-18 check-digit validation, SSCC minting at seal, ZPL GS1-128 labels, DESADV GIN+BJ from real ship units | `gs1/checkdigit.go:76-171`; `gs1/zpl.go:20-56` | **No DataMatrix** (critical for UZ/CIS marking regimes); no EPCIS; no GS1 registry sync |
| Master-data import | **WIRED-LIVE but human-only** — 9-state import wizard: signed-URL upload, xlsx/csv/tsv, column auto-discovery, mapping, staging, error summaries | `supplier/import_sessions_handlers.go:39-100`; `supplier/import_async.go:331` | **No machine-import endpoint** — a chain cannot push 50k SKUs programmatically |
| Compliance export | **WIRED-LIVE** — CSV export for admin/supplier JWT | `orderroutes/routes.go:92`; `supplierroutes/routes.go:162` | CSV only; no BI sink, no BigQuery (zero references anywhere) |
| Event backbone | **WIRED-LIVE** — transactional outbox → leased relay → Kafka (`RequiredAcks=all`, no auto-create), consumer dedup, DLQ + replay tooling | `outbox/relay.go:155-203`; `outbox/kafka_publisher.go:79-90`; `kafka/dlq_writer.go` | **Kafka is a single broker, replication-factor 1** (`infra/k8s/kafka.yaml:42-47`) — the entire integration backbone dies with one pod; outbox relay has no DLQ of its own |
| SAP / 1C / Odoo / NetSuite connectors | **ABSENT** — zero integration code (grep-verified) | — | Only indirect interchange via files (SFTP/AS2/CSV/XML) |

## 4.2 What must exist for big players to adopt without re-keying

Ranked by blocker severity, each grounded in a verified absence:

**P0 — adoption blockers**
1. **Idempotent REST order create.** Honor `Idempotency-Key` on `POST /partner/v1/orders` with a durable store (the EDI path already has this via unique index; the REST path has nothing).
2. **Master-data sync API.** Partner-key upsert endpoints for products/prices/stock (or EDI PRICAT/INVRPT inbound). Today machines can read catalog but cannot write it.
3. **Webhook event coverage.** Expose the delivery/invoice/manifest/return lifecycle (a configurable per-subscription event filter over the 155-type catalog, most of which already flows through the same outbox→Kafka substrate the webhooks consume).
4. **EDI functional acknowledgments.** CONTRL/APERAK outbound and inbound ORDRSP/INVOIC handling — the loop is currently one-legged.
5. **Enable the transports in shipped manifests.** AS2/SFTP default off and are missing from the configmap and `.env.k8s.example`; the integration layer is invisible in rendered deploys.
6. **Kafka HA.** Three brokers, RF=3, before any enterprise signs against webhook/EDI delivery semantics.

**P1 — enterprise credibility**
7. Certified EDIFACT envelope compliance (UNB/UNZ interchange control) and/or X12; today's dialect is proprietary.
8. 1C CommerceML 2.x exchange package + richer journals (VAT lines, multi-leg, returns/credit-note postings).
9. DataMatrix generation (UZ marking), EPCIS events, GS1 registry sync.
10. M2M auth hardening: mTLS or RS256/JWKS, IP allowlists, wire the unused per-key `RateLimitClass`.
11. Partner order amend/cancel/status-transition endpoints.
12. POS sales ingestion as a demand signal (the auto-order chain's weakest input; weather is wired, POS is explicitly residual).

**P2 — maturity**
13. Async MDN + CEM; Drummond certification.
14. SFTP host-key pinning.
15. Excel exports + BI sink (BigQuery or parquet feed).
16. Partner sandbox + self-serve key onboarding (keys currently require a human JWT session, `partner/routes.go:50-57`).
17. Resolve the 7 declared-but-never-emitted event types (`ALLOCATION_FAIR_SHARE_APPLIED`, `INVENTORY_IMPORT_STATUS_UPDATE`, `RETAILER_CLOCK_IN/OUT`, `RETAILER_SHIFT_OPENED/CLOSED`, `STORE_STOCK_CLAIM_HOLD`).

# 5. Does a True Unified Platform Already Exist?

**The question:** is there any public system that connects quality suppliers and retailers into one transactional platform with near-zero human interaction for routine replenishment, while still supporting the physical execution roles (warehouse floor, loading bay, driver, store receiving)?

## 5.1 The public landscape (verified August 2026)

Four established categories occupy adjacent territory; none occupies the full position:

| Category | Players (2026 status) | What they do | Why they are not the full position |
|---|---|---|---|
| B2B wholesale marketplaces | Udaan (India; FY26 EBITDA burn −40%, city-density + private-label pivot, pre-IPO), MaxAB-Wasoko (Africa; e-commerce retrenching, fintech arm >$180M Egypt turnover now exceeds e-commerce), Jumbotail (India; ~$1B, full-stack + embedded fintech via Solv), TradeDepot (pivot to data/ads), MarketForce (e-commerce arm shut), Ankorstore (0% reorder commission since Jan 2026), Faire | Many-supplier↔many-retailer ordering, logistics via owned/3PL networks, increasingly credit | They **are the merchant** (or a thin layer over one) — not a transactional platform a distributor runs their own chain on. No factory→shelf role apps; physical execution is outsourced or owned, not productized per role |
| Sales-force automation / DMS | FieldAssist (32+ countries, offline-first, route optimization, planogram audits), Botree, PepUpSales | Exactly the "replace the agent's order pad" product | No warehouse/fleet/payment/fiscal depth; they sit on top of the distributor's ERP, they don't run the chain |
| Enterprise planning+execution suites | Blue Yonder (native WMS/TMS + planning; retail/CPG depth), o9/Kinaxis (planning leaders, no native execution) | Deep planning, real execution (BY), enterprise integration | No B2B field-agent/retailer-facing commerce layer, no COD/credit-delivery state model, no CIS fiscal plumbing; $1.5–8M/yr and 9–24-month implementations |
| Agentic order entry | Proton.ai (GA July 2026: reads emails/PDFs/handwritten lists, applies contract pricing, picks warehouse, drafts orders) | Automates order *entry* for distributors' inside sales | Entry-point only; no execution, no platform |

**The clearest lesson from that landscape:** the pure marketplace thesis largely failed on distribution margin alone. Every survivor monetizes **credit, fintech, and data** on top of captured transaction flow (MaxAB-Wasoko: >$20M working-capital loans at claimed 99% repayment underwritten from platform purchase data; Jumbotail: BNPL via NBFC partners; Udaan: private label + density). PegasusX's own audit reached the same conclusion independently, and its code base already contains the credit spine (§7.4) — implemented, flag-gated off.

## 5.2 Verdict on existence

**No public system today occupies the exact position: a multi-tenant transactional platform connecting independent suppliers and retailers, with near-zero-human routine replenishment, that also runs the physical execution roles in the same model.** The closest are the full-stack distributors (Jumbotail, Udaan), which achieve touchless-ish replenishment *as the merchant for their own inventory*, not as a platform others operate. In that narrow sense the whitespace claim survives.

## 5.3 PegasusX's actual position versus that ideal

| Requirement of the ideal | PegasusX reality | Gap |
|---|---|---|
| Many suppliers, many retailers, one transactional platform | **Single-supplier runtime by construction** (seed ID injected into ~20 constructors; second tenant's orders misattributed) | **Existential** — Gate 5 Phase 1 is an accepted plan, uncoded; 150–250 files to touch |
| Near-zero human routine replenishment | Full zero-touch chain exists in code (sensing → reorder → auto-order `place` → touchless approve → auto-dispatch) | **Activation** — every link ships default-off; accuracy/acceptance evidence not yet accumulated to justify auto-flip |
| Physical execution roles supported | The strongest part: driver/payload/warehouse/factory role apps, all wired | **Depth** — WMS floor execution portal-only; dead scanner stub; no item-level scan verification at loading |
| Money and law settled in-platform | COD/credit/split state model + ledger + reconciliation real; fiscal framework real | **Correctness/legality** — card capture broken; legal fiscal provider non-functional; AR flag-gated off; payment idempotency not DB-enforced |
| Enterprise integration without re-keying | Partner API + OAuth + EDI-lite + AS2 + 1C journals exist | **Completeness** — P0 list in §4.2 (idempotency, master-data push, webhook coverage, ACKs, Kafka HA) |
| Quality/trust layer (KYB, ratings, admin governance) | Nothing: no admin console, no tenant approval/suspension, no supplier scorecards | **Absent** — Phase 5 of the tenancy plan; no `PLATFORM_ADMIN` role in `auth/` |

**Bottom line:** PegasusX has built ~70% of the ideal platform's *substance* (the vertical transactional spine, the role apps, the event bus, the credit primitives, the integration skeleton) and ~10% of its *platform property* (multi-tenancy, governance, self-serve onboarding). The distance to the ideal is dominated by one existential gap (runtime multi-tenancy), a set of correctness/legality repairs on the money path, and activation work (flags + evidence), not by missing ambition.

# 6. Per-Role, Per-App, Per-Feature Reality

Method: static code audit of every app under `apps/`, verified against the backend routes it calls. Status vocabulary per §1. API depths are measured (Retrofit endpoint counts for Android, unique `/v1/` path literals for iOS, unique api-client methods for portals).

## 6.1 Retailer — Android (187 kt files) · iOS (144 real Swift files) · Desktop (27 dashboard routes, Tauri)

**Maturity: ~92%. All three variants are WIRED-LIVE; zero mock data found in any variant.**

### What exists and works (WIRED-LIVE unless noted)

| Feature | Evidence |
|---|---|
| Auth (login/register/org memberships/switch) | `retailer-app-android/.../PegasusApi.kt:64-77`; iOS `Services/AuthManager.swift:74,143`; desktop OS-keyring auth |
| Catalog, search, cart, quoted checkout | `PegasusApi.kt:93,133-154`; desktop `components/CheckoutModal.tsx`; idempotency keys per action |
| Orders lifecycle + tracking map (WS live) | `DeliveryTrackingViewModel.kt`, `RetailerWebSocket.kt`; iOS `DeliveryMapView.swift` |
| AI predictions / preorders (confirm/reject with idempotency) | `PegasusApi.kt:174-180`; desktop `lib/api.ts:15-29` |
| Claims (eligibility countdown, file, media upload tickets) | `PegasusApi.kt:112-125`; `FileClaimSheet.kt`; iOS `FileClaimView.swift` |
| Auto-order rules + mode + scopes + shadow inbox | `AutoOrderScreen.kt`; iOS `AutoOrderView.swift`; desktop `auto-order/page.tsx` |
| **POS with offline queue** | `PosScreen.kt` + Room `PendingPosSaleDao.kt`/`PendingPosSaleSync.kt`; iOS `PendingPosStore.swift`; backend `retailer/pos.go`, routes `retailerroutes/routes.go:87-101` (sessions, sales, void, refund, holds) |
| Store stock counts, local SKUs, shifts, sections | `StoreStockScreen.kt`, `LocalSkusScreen.kt`, `ShiftsScreen.kt` |
| Reports/analytics + HQ multi-location (with export) | desktop `hq/page.tsx:72-120` (`/v1/retailer/hq/*`) |
| Payments, saved cards, credit profile | `SavedCardsViewModel.kt`, `CreditProfileViewModel.kt` (`/v1/retailer/credit-profile`) |
| Suppliers discovery/connect, team, locations, setup wizard, capabilities packs | `PegasusApi.kt:151-244` |
| Control tower (hex map, live counts) | `ui/controltower/ControlTowerScreen.kt`; iOS `ControlTowerView.swift:16` |
| Notifications inbox + FCM; auto-updater | `PegasusFirebaseMessagingService.kt`; `service/AutoUpdater.kt:97,147` |
| Offline: Room (catalog/orders/POS/predictions) + workers; iOS file-based replayer; desktop cache + offline tray | `AppDatabase.kt`; `PendingOrderSyncWorker.kt`; iOS `PendingOrderReplayer.swift` |

### Incomplete / decorative / broken

- **Auto-order indicator on My Suppliers is a placeholder** — always shows the icon if the supplier has orders (`MySuppliersScreen.kt:291`). Cosmetic.
- **Retailer-iOS repo hygiene**: ~1,380 vendored SPM build checkouts committed under the app tree, in a directory misspelled `reatilerapp` — build-reproducibility and review-noise hazard.
- **Offline POS contradiction**: client-side offline POS queues are wired (Room/file), yet the project's own status docs list offline POS as product-deferred — the unresolved half is server-side: fiscalization and idempotent acceptance of replayed POS sales. Treat end-to-end offline POS as PARTIAL.
- No E2E-visible refund/return initiation from the retailer side (claims only); HQ export is CSV-only.
- Desktop Control Tower is a simpler surface than mobile parity.

### Missing features that matter (Retailer)

1. **Server-verified offline POS acceptance + fiscalization.**
 *Purpose:* make offline POS sales legally and financially real.
 *Why needed:* the client queues sales offline; without server-side replay acceptance tied to fiscal receipts and idempotent dedup, offline sales are either lost, double-counted, or legally non-compliant in an OFD-mandated market.
 *Logic:* each queued sale carries a deterministic idempotency key (SHA-256 of store+session+line content, as `packages/api-client/idempotency.ts` already does for checkout); server accepts into `PosSales` with unique `(StoreId, IdempotencyKey)`; replayed batch opens a fiscalization leg per sale (same `FISCALIZING` machine as orders); conflicts surface in a shift-close report.
 *End-to-end:* cashier sells offline → Room queue → connectivity returns → `PendingPosSaleSync` replays → server dedups/accepts → fiscal receipt issued per sale → shift close reconciles POS cash + card + queue → HQ reports include the sales.
2. **Cross-supplier cart with order splitting.** *Purpose:* one cart, many suppliers. *Why:* the runtime is single-supplier; `Orders` PK embeds `SupplierId`. *Logic:* new `ParentOrders` table; `ParentOrderId` on `Orders`; split engine fans out per-supplier child orders, each with its own credit check, inventory plan, pricing resolution, warehouse assignment; retailer UI rolls status up. *E2E:* cart mixed SKUs → checkout splits → per-supplier legs flow independently → unified tracking view. (Multi-tenancy Phase 2; 1 new table, 2 altered, 30–50 files.)
3. **Shelf-count / sell-through capture UX feeding auto-order.** *Purpose:* make auto-order inputs truthful. *Why:* the inventory-grounded (R,s,S) proposals decay confidence by stock-record age (`UpdatedAt`); without easy shelf counts, shadow acceptance stays low and `place` never justifiably turns on. *Logic:* count sessions write `RetailerStockBalances` (exists); confidence = f(recency, count frequency); proposals only from stock fresher than threshold. *E2E:* weekly guided count (barcode + qty) → balances fresh → auto-order proposals carry real on-hand → acceptance rate measurable in the shadow ledger (`RetailerAutoOrderShadowProposals`).
4. **KYC / business verification on onboarding.** *Purpose:* credit decisions require verified identity. *Why:* credit limits are granted at placement; today onboarding is self-serve with no verification artifact. *Logic:* document capture → review queue (admin console dependency) → status gate on credit eligibility. *E2E:* register → upload → approved → credit programs visible.
5. **Retailer-initiated refund/return request flow.** *Purpose:* close the post-delivery loop without phone calls. *Why:* today only claims exist; refunds are admin/global-pay paths. *Logic:* claim → approved → credit note (`creditnote/`) → refund leg on original payment method or AR credit. *E2E:* claim approved → retailer chooses refund-vs-credit → credit note issued → money leg or AR balance adjustment → fiscal corrective chain (`CreditNotes.OriginalEhfId/CorrectiveEhfId` schema exists, `spanner.ddl:1696-1698`).

## 6.2 Supplier — Portal (882 files, also desktop via Tauri) · Android (143 kt) · iOS (138 Swift)

**Maturity: ~90%. Zero TODO/mock hits in main sources on any variant. Mobile has NO offline mode (no Room/SwiftData) — the worst offline story among role apps.**

### What exists and works

| Feature | Evidence |
|---|---|
| Auth incl. business setup + billing | `packages/api-client/index.ts:434-462` |
| Orders hub/detail; manifests + exceptions; dispatch preview/execute (MapLibre) | `OrdersHubScreen.kt`; `DispatchPreviewScreen.kt`; portal `(portal)/dispatch` |
| Catalog CRUD + barcode + images; inventory CSV import sessions | `CatalogScreen.kt`; `InventoryImportScreen.kt`; `supplier/import_sessions.go` |
| Fleet: live map, org fleet (drivers/vehicles/members), delivery zones, topology, supply lanes | `FleetLiveMapScreen.kt`, `TopologyScreen.kt`; portal `topology`, `delivery-zones` |
| Claims/chargebacks/credit notes; exceptions (negotiations, shop-closed, early-complete) | `ClaimsScreen.kt`, `ChargebacksScreen.kt`; portal `exceptions/*` |
| Finance: payments, ledger, reconciliation, treasury, earnings | `LedgerScreen.kt`, `ReconciliationScreen.kt`; portal `(portal)/reconciliation` |
| Promotions + performance; retailer price overrides | `PromotionsScreen.kt`; `RetailerOverridesScreen.kt` |
| Planning surfaces (S&OP view, policies, seasonal overrides, forecast accuracy) | `PlanningBrainScreen.kt`; portal `settings/planning`; `GET /v1/supplier/analytics/demand/accuracy` |
| Return policy settings; compliance; notification prefs | `ReturnPolicySettingsScreen.kt` |
| Admin-capable ops inside portal (assign driver, status patch, FX rates, partner keys) | api-client `/v1/admin/*`; `apps/admin-portal/README.md` routing table |

### Incomplete / decorative / broken

- **S&OP planning view renders stub math** — backend `GetSAndOP` returns `factories × 700 × 7` and `projectStockouts` emits literal `sku-projection-%d` strings (`planning/service.go:212,252`). The UI is wired; the substance behind it is placeholder.
- **No offline capability on supplier mobile** — field reps on bad networks lose work.
- iOS dual-target file duplication (`CreateDriverSheet.swift` ×2, `CreateVehicleSheet.swift` ×2) — drift risk.
- Payout execution absent (settlement authority endpoint is a reporting view, `GET /v1/payment/settlement/authority`); refund initiation absent (Gateway-side refund exists for Global Pay only).

### Missing features that matter (Supplier)

1. **Payout execution.** *Purpose:* close the money loop for suppliers. *Why:* collections exist; disbursement does not — a marketplace without payouts cannot monetize. *Logic:* settlement authority view already computes `operating_currency_total_minor`; add payout batches = Σ(captured legs) − Σ(refunds) − commission (fee schedule dependency) per supplier per period; execute via PSP payout rail or bank file export; ledger entries per payout with idempotency keys. *E2E:* period close → batch preview → finance approve → rail execution → supplier statement reconciliation.
2. **Refund initiation (full/partial).** *Purpose:* money reversals without engineering tickets. *Why:* the only "Refund" occurrence in non-test Go reads `AmountRefunded` off a Stripe webhook; GP executor has `executeRefund` but no product surface triggers it. *Logic:* refund ≤ captured amount − prior refunds (cap exists at `payment/service.go:713-716`); create reversal ledger legs; fiscal corrective document; AR credit if credit sale. *E2E:* dispute/claim approve → refund wizard → PSP call (must fix §7 P0-1 first) → reversal legs → fiscal correction → retailer notified.
3. **Pricing authority engine.** *Purpose:* governed pricing — who may set/change prices, within what guardrails. *Why:* `pricing/service.go` is a repository delegate (4 files); the design doc is a self-declared stub; promo engine exists but price governance does not. *Logic:* rule table `(role, scope, delta_limit_bps, margin_floor_bps, approval_required)`; proposed change → policy evaluation → auto-apply or approval task → effective-dated `PriceListItem`; audit row per change. *E2E:* rep proposes −12% → floor check → manager approve → new effective window → checkout quotes respect it.
4. **Supplier mobile offline queue.** *Purpose:* field usability. *Why:* only role with zero offline. *Logic:* reuse `packages/mobile-android-kit` offline contract (ACK 409 / retry 5xx / dead 4xx) already proven in driver. *E2E:* queue mutations → flush on reconnect with capture-time coordinates.
5. **Per-supplier delivery perimeter enforcement (E2).** *Purpose:* multi-supplier correctness. *Why:* `retailer/proximity_service.go:24` uses one global key `ssmr:delivery_perimeter` in production reads; the per-supplier helper exists but is design-only. *Logic:* `PerimeterKeyForSupplier(supplierId)` on all reads/writes. *E2E:* supplier A's zone edits never leak into supplier B's eligibility checks.

## 6.3 Driver — Android (178 kt) · iOS (129 Swift)

**Maturity: ~95% — the most production-hardened role. 56 endpoints per platform; zero mock/TODO hits.**

### What exists and works

| Feature | Evidence |
|---|---|
| Manifest load, arrive/deliver/complete lifecycle (all through the FSM-validated funnel) | `DriverApi.kt:80,109,141,170`; backend `order/service.go:2153` |
| POD: QR validate/scan, signature pad, photo proof (credit leave requires photo, fail-closed) | `DriverApi.kt:123,134`; `SignaturePad.kt`; iOS `SignaturePadView.swift` |
| Cash collection with server-computed expectation | `DriverApi.kt:148`; `CashCollectionViewModel.kt`; `cashrecon/service.go:39-57` |
| Delivery correction/amend/offload review | `CorrectionViewModel.kt`, `OffloadReviewViewModel.kt` |
| Fiscal retry UI (FISCALIZING/FISCAL_FAILED) | `FiscalizingView.kt`, `FiscalFailedView.kt` |
| Telemetry: adaptive filter (15s/20m/15°), WS + Room-buffered sync, boot resume | `TelemetryService.kt`, `TelemetrySyncWorker.kt`, `BootReceiver` |
| Geofencing, route deviation, navigation cue banners | `DriverGeofence.kt`, `RouteDeviation.kt`, `NavigationCueAnnouncer.kt` |
| Offline: Room action queue + verifier + sync-queue UI; iOS SwiftData offline delivery store | `DriverOfflineQueue.kt`; `OfflineSyncWorker.kt` (409 ACK / 5xx retry / 4xx dead); iOS `OfflineDeliveryStore.swift` |
| Earnings/history, availability, rescue/reassign handshake, supply transfers, handoff inbox, scanner (real), notifications | `DriverApi.kt:84,177,191-209`; `RequestRescueSheet.kt` |

### Incomplete / decorative / broken

- **Card collection at the door is hostage to backend P0-1** — the driver can complete a card flow whose capture will silently fail server-side (§7).
- **No turn-by-turn navigation engine** — cue banners over backend geometry only.
- **iOS has no durable offline telemetry buffer** (Android buffers telemetry in Room before WS send); no server-side telemetry ACK frame (OkHttp enqueue ≠ delivery proof).
- iOS target-file duplication (`AutoUpdater.swift` ×2).

### Missing features that matter (Driver)

1. **Server-acknowledged telemetry.** *Purpose:* provable location trail (disputes, insurance, SLA). *Why:* fire-and-forget WS send loses points silently. *Logic:* server ACK frame per batch id; client deletes buffer only on ACK; gap detection metric. *E2E:* telemetry batch → server persist → ACK → client purge.
2. **Turn-by-turn navigation.** *Purpose:* stop-level ETAs and driver efficiency. *Why:* ETAs are Haversine heuristics (`eta/calculator.go:21`); real navigation needs OSRM/Valhalla guidance or SDK integration. *Logic:* route geometry exists (`/v1/driver` route geometry endpoint); add maneuver extraction + voice cues; recompute on deviation (deviation detection already exists). *E2E:* manifest → navigate → auto-advance stop → ETA recompute on each location update (already wired at `eta/service.go:249`).
3. **iOS telemetry durability parity.** *Purpose:* platform parity. *Logic/E2E:* mirror Android's insert-before-send with SwiftData buffer + flush worker.
4. **Offline maps tiles.** *Purpose:* dead-zone operation. *Logic:* prefetch corridor tiles on manifest load; bounded cache.

## 6.4 Payload / Loading — Android (61 kt) · iOS (41 Swift) · Terminal (Expo RN, 33 files)

**Maturity: ~85%. Zero mock/TODO hits. 38/34/15 endpoints across android/iOS/terminal.**

### What exists and works

| Feature | Evidence |
|---|---|
| Payloader auth with token refresh | `PayloadApi.kt:49,52`; `TokenRefreshAuthenticator.kt` |
| Trucks sidebar, manifest list/detail | `PayloadApi.kt:56,81,87` |
| Start loading / seal / seal-completed / **seal-all**; SSCC mint per ship unit at seal | `PayloadApi.kt:90-143`; `gs1/checkdigit.go:142-171` |
| Inject order into manifest; recommend/execute reassign | `PayloadApi.kt:68,74,108,135` |
| Loading checklist per order | `OrderChecklist.kt`; iOS `OrderChecklistSection.swift` |
| Manifest exceptions; missing-items report | `PayloadApi.kt:149-161` |
| Inbound returns (sessions/scan) incl. on terminal | `PayloadApi.kt:201-212`; terminal `inboundReturns.tsx` |
| Offline queue (Room android; OfflineQueue iOS); push | `PayloadDatabase.kt`, `QueuedActionDao.kt` |
| Dual scope: warehouse + supplier manifests | `PayloadApi.kt:115-135` |

### Incomplete / decorative / broken

- **Terminal is a strict subset** (15 endpoints vs 38): no reassign, no seal-all on shared bay devices.
- **Checklist verification is manual taps** — no barcode scan per item; a mis-load is indistinguishable from a correct load.
- **In-memory dev overlay persists with comment-only gating** — `payload/service.go:41-48` keeps an in-memory overlay; the `PAYLOAD_DEV_OVERLAY` env gate exists only as a comment, no code reads it (doc-claimed control that doesn't exist).
- iOS payload app lacks `.xcodeproj` for SPM integration of the shared kit (per repo status docs); minimal test coverage (`ExampleTests.swift` only).
- No weight/temperature capture at loading despite cold-chain backend support.

### Missing features that matter (Payload)

1. **Item-level scan verification at loading.** *Purpose:* loading accuracy — the cheapest place to stop the most expensive errors. *Why:* every mis-load becomes a driver-reported condition report after the truck has driven; scan-at-load catches it at zero transport cost. *Logic:* expected set = manifest lines (sku, qty); scan events decrement expected; seal blocked (or soft-warned, mirroring the existing `pick_wave_warning` soft-warn pattern) while residual > 0; variance auto-creates a manifest exception (endpoint exists). *E2E:* loader scans each case → checklist auto-checks → seal gate → SSCC label → DESADV carries real ship units.
2. **Per-line quantities + hardware scanner on the terminal.** *Purpose:* the shared bay device must do real verification, not taps. *Why:* keyboard-wedge/DataWedge scanning already proven in warehouse kit. *Logic/E2E:* wedge input → line match → qty confirm → same seal gate as mobile.
3. **Cold-chain capture at loading.** *Purpose:* chain-of-custody for temperature before transit. *Why:* backend ingests readings and auto-raises `TEMPERATURE_BREACH` (`stocklots/coldchain.go`); loading-time baseline missing. *Logic:* record bay temp + product band at seal; breach clock starts at seal. *E2E:* seal → baseline reading → in-transit readings → delivery dock reading → quarantine decision already implemented downstream.
4. **Split the 1,700-line god view; generate the iOS Xcode project** (hygiene, unblocks shared-kit adoption).

## 6.5 Factory — Portal (365 files) · Android (77 kt) · iOS (67 Swift)

**Maturity: ~85% as a dispatch hub. This is NOT a manufacturing execution system — no BOM, work orders, or line management anywhere (a scope decision, stated honestly).**

### What exists and works

| Feature | Evidence |
|---|---|
| Auth, dashboard/pulse | `factory-portal/lib/auth.ts:132`; `DashboardScreen.kt` |
| Transfers full lifecycle (create/move/detail/driver assignment) | `factory-portal/app/transfers/page.tsx:39`; `CreateTransferScreen.kt` |
| Loading bay (grid/controls) | `LoadingBayScreen.kt`, `LoadingBayGrid.kt` |
| Manifests + rebalance/cancel; exceptions | `ManifestLifecycle.kt`; portal `manifests/[id]/page.tsx` |
| Supply requests + fulfill options | `SupplyRequestsScreen.kt`; `/v1/factory/supply-requests/fulfill-options` |
| Fleet + live map + drivers/vehicles; staff; payload overrides | `FleetScreen.kt`; `StaffScreen.kt`; `PayloadOverrideScreen.kt` |
| Analytics/insights; handoff timeline; notifications; WS realtime | `AnalyticsScreen.kt`; `FactoryRealtimeClient.kt` |
| Android offline queue | `FactoryOfflineQueue.kt` |

### Incomplete / decorative / broken

- **Factory service holds manifest state in in-memory maps** (`factory/service.go:63-65,271-273`) — restart loses state; flagged in the hardening plan (E6) and still open.
- **S&OP numbers are stub math** (shared `planning/service.go:252` stub).
- iOS/portal online-only; analytics shallow (single overview endpoint).
- `GetSAndOP` + planning-brain screens present a planning capability the backend does not have.

### Missing features that matter (Factory)

1. **Durable factory manifest state.** *Purpose:* correctness across restarts/deploys. *Why:* in-memory maps lose in-flight manifests. *Logic:* persist to the existing Spanner manifest tables; overlay reads become authoritative reads. *E2E:* deploy mid-shift → bay state intact.
2. **Production-decision: MES or not.** *Purpose:* if "factory" remains a dispatch hub, rename it honestly; if it must plan production, that is a new subsystem. *Why:* today the label outruns the substance. *Logic (if MES):* work orders from demand plan (`DemandForecastBaseline` exists per SKU-day), simple capacity check `Σ(work_order_hours) ≤ line_hours`; BOM explosion only if multi-component products are in scope. *E2E:* forecast → weekly production proposal → confirm → material reservation → completion posts finished-goods inventory.
3. **Real S&OP feed.** *Purpose:* replace `factories × 700 × 7`. *Logic:* capacity = Σ active production lines × shifts × rated throughput (needs a `ProductionLines` table); demand = `DemandForecastBaseline` 13-week sum; gap = demand − capacity. *E2E:* S&OP screen shows real gap, drives supply-request urgency.
4. **Transfer lead-time capture completeness.** *Purpose:* σ_L for safety stock v2. *Why:* `FactoryInternalTransfers.ReceivedAt` exists and feeds `ObservedLeadStats` when ≥10 samples; mobile receive-confirm UX ensures every transfer produces a sample. *E2E:* warehouse receive confirm → `ReceivedAt` written → lead stats mature → safety stock stops using assumed σ_L.

## 6.6 Warehouse — Portal (597 files, deepest portal) · Android (110 kt) · iOS (96 Swift)

**Maturity: ~82% overall — portal ~90%, mobile ~70%. The gap between backend capability and floor-worker tooling is the widest in the platform.**

### What exists and works

| Feature | Evidence |
|---|---|
| Auth, dashboard/pulse, orders + ops actions | `warehouse-portal/lib/auth.ts:90`; `OrdersScreen.kt` |
| **Pick waves (create/confirm/waive) + seal gate** — portal | `warehouse-portal/app/pick-waves/page.tsx:34,79,103`; backend `stocklots/picking.go`, `seal_gate.go` |
| **Cycle counts + adjustments (apply-on-approve, ABC)** — portal | `portal/app/cycle-counts/page.tsx:30-238`; backend `stocklots/counting.go` |
| **Bins/lots/putaway (FEFO)** — portal | `portal/app/bins/page.tsx:27-54`; backend `stocklots/fefo.go` |
| **Cold chain ingest + quarantine** — backend | `stocklots/coldchain.go` (`WMS_COLD_CHAIN_ENABLED`) |
| Inventory, stock commitments, replenishment, demand forecast | `InventoryScreen.kt`, `ReplenishmentScreen.kt`, `DemandForecastScreen.kt` |
| Dispatch + locks + rescues + auto-dispatch settings | `DispatchScreen.kt`; portal `dispatch-locks`, `dispatch/rescues` |
| Fleet live map/drivers/vehicles; returns; claims; supply requests; transfers (incl. pick-wave create inside `TransferActionsScreen.kt:340-366`) | portal + android |
| Treasury/payment config; preorders/tomorrow board; CRM; broadcasts; staff; control tower (portal) | portal routes |

### Incomplete / decorative / broken

- **The Android barcode scanner is a dead stub** — `ScannerViewModel.kt:22` (`// TODO: Inject API when available`), `:47` (`// TODO: Dispatch telemetry event to backend` — marks every scan SUCCESS without any call), and `BarcodeScannerScreen.kt` has **zero references from navigation** — an orphaned screen that cannot even be reached. This is the single clearest DECORATIVE feature in the client set.
- **WMS execution is portal-only**: pick waves, cycle counts, bins/lots/putaway have no mobile screens (Android has API methods for some, consumed only inside a transfers screen; iOS has none). Floor workers on mobile cannot execute the WMS that the backend implements.
- **No FEFO/cold-chain UI in any client** (zero hits across apps) — backend capability invisible to operators.
- Android offline queue is SharedPreferences-based (`WarehouseOfflineQueue.kt:64`) — fragile vs Room peers; iOS has no offline store.
- Serial tracking absent (backend too).

### Missing features that matter (Warehouse)

1. **Mobile floor execution (pick waves, putaway, cycle counts) on Android/iOS.**
 *Purpose:* the warehouse is run by people walking aisles with phones/scanners, not by people at desks.
 *Why:* all three flagship WMS capabilities are desk-bound today; without mobile execution the WMS is an aspirational console and the Android scanner stub is a liability.
 *Logic:* pick tasks sorted by zone + serpentine `PickSequence` (backend PR-5 exists); worker claims task → scan lot/location → confirm qty (`QuantityPicked` vs `QuantityRequested`); short-pick → exception; wave complete → seal gate clears. Cycle count: ABC cadence (A monthly/B quarterly/C annually by annual movement value) enqueues counts; variance `> threshold` → `InventoryAdjustments` with mandatory reason + approval; accuracy KPI `1 − Σ|variance|/Σexpected` per warehouse.
 *E2E:* wave released → picker mobile list → serpentine walk with scans → seal → manifest dispatch; count scheduled → mobile count → variance approval → lot QoH + roll-up adjust in one txn (apply-on-approve backend exists).
2. **Fix or delete the scanner stub.** *Purpose:* trust. *Logic:* wire `ScannerViewModel` to `WarehouseApi` (96 endpoints exist) or remove the screen; either way, add a CI grep failing on `TODO: Inject` in main sources.
3. **FEFO/cold-chain operator surfaces.** *Purpose:* expiry-driven allocation must be visible/overridable. *Logic:* lots near expiry list (`ExpiryDate − today < threshold`); allocation preview shows chosen lots; breach alerts route to quarantine actions (backend `TEMPERATURE_BREACH` auto-raise exists).
4. **Room-based offline queue parity + iOS offline store.** *Logic:* adopt `mobile-android-kit` queue contract as driver/payload already do.
5. **Serial tracking.** *Purpose:* pharma/electronics. *Logic:* `SerialNumbers` table keyed to lot + order line; scan at pick and at delivery; warranty/returns by serial.

## 6.7 Platform Admin — dedicated app ABSENT

**Maturity: ~40%.** `apps/admin-portal/` is a retired redirect stub (3 files; `redirect.mjs` exits 1). `apps/supplier-app-desktop/` likewise. Admin capability is real but scattered: ~17 `/v1/admin/*` endpoints (partner keys, FX rates, planning run-once, AR dunning run-once, credit disables) exercised through supplier-portal/warehouse-portal under ADMIN JWT.

**Absent entirely:** tenant/org lifecycle (no approval queue, no suspension, no offboarding — a supplier can self-register and **nobody can approve or remove them**, `supplier/service.go:433-447`), user/role administration UI, feature-flag console, system health/observability, audit-log viewer, support tooling, fee schedule management. No `PLATFORM_ADMIN` role exists in `auth/` at all.

### Missing features that matter (Admin)

1. **Platform admin console.** *Purpose:* the platform cannot be governed. *Why:* multi-tenancy Phase 5 and basic operations both depend on it; today tenant trust is implicit and unenforceable. *Logic:* `PLATFORM_ADMIN` break-glass role; tenant states `PENDING→APPROVED→SUSPENDED→OFFBOARDED` with KYB document collection; every admin action audit-rowed. *E2E:* supplier registers → KYB review → approve → tenant activated; incident → suspend → all tenant tokens denied at middleware.
2. **Feature-flag console.** *Purpose:* the entire autonomy stack is env-flag-gated; operators need runtime control per tenant. *Logic:* flags table + middleware resolver (env default → tenant override); audit + change approval for money-affecting flags (AR, auto-order place, fiscal provider).
3. **Fee schedule + billing ops.** *Purpose:* monetization. *Why:* billing meter schema + event decode are wired (`internal/services/billing/meter_worker.go`) but no fee schedule or invoices exist. *Logic:* fee rules `(per-order fixed | GMV bps | subscription)` per tier; nightly meter → invoice → AR open item (reuse `ArInvoices`) → dunning reuse. *E2E:* tier assign → usage meters → monthly invoice → collection → payout net-of-fees.
4. **Observability & audit surfaces.** *Purpose:* ops trust. *Logic:* outbox lag, relay watchdog state (`outbox/relay.go:88-122` already computes stuck events), DLQ depth, fiscal failure rate, capture success rate — all already measurable from existing tables.

# 7. Correctness, Concurrency, Money, and Fiscal Reality — Verified Register

This section consolidates the backend truth the brief demands be accurately reflected. Items are ranked by severity; all are code-verified with file:line.

## 7.1 P0 — correctness / legality (fix before anything else)

| # | Defect | Evidence | Consequence |
|---|---|---|---|
| 1 | **Card capture permanently broken + fire-and-forget.** `CaptureCardPayment` uses gateway key `"GLOBALPAY"`; executor map is keyed `"GLOBAL_PAY"`; router normalizes only case/whitespace → lookup miss → every capture errors. Callers: post-commit goroutine (log-only) and backorder sweeper (skips). Worse: `CompleteOrder` pre-records the leg `CAPTURED` **in-txn** before any PSP call | `payment/service.go:653`; `payment/execution.go:140,224-236`; `order/service.go:1899-1929`; `order/backorder_sweeper.go:51-55` | **Ledger asserts money collected that never was; silent financial loss; deferred-payment orders stuck BACKORDERED forever** |
| 2 | **Legal fiscalization non-functional as shipped.** `MY_SOLIQ` provider's `signer` field is assigned nowhere; `CreateReceipt` always errors `"mysoliq: no EDSSigner configured"`. Default provider `PEGASUS` issues platform receipts explicitly `"tax_ofd": false`. The 18-state machine's FISCALIZING hard-gate creates false assurance of fiscal compliance | `order/fiscal_provider.go:129,232-234`; `order/fiscal_provider_pegasus.go:13-15,78-79`; gate at `order/service.go:1527-1538` | **Orders complete without legally mandated OFD receipts; flipping the flag halts completions at 100% FISCAL_FAILED** |
| 3 | **Silent PSP stub-success on empty Global Pay credentials.** Capture → `gp_capture_stub_`, refund → `gp_refund_stub_`, status → `gp_status_stub_paid` — all returning nil error. Dormant landmine: `WebhookReconciler` (not yet wired) maps the stub status to a `SignatureValid: true` PAID webhook | `payment/global_pay_executor.go:112-120,251-258,312-320`; `payment/reconciliation.go:57,66-108` | Refunds reported that never happened; if reconciler is wired with empty creds, stuck sessions auto-"pay" themselves |
| 4 | **Shop-closed timeout worker can deliver on credit without recording debt.** Credit-profile read failure is warn-only; order still becomes `DELIVERED_ON_CREDIT`; inline balance math ignores `ReservedMinor` (double-counts vs reserve-at-create) and hardcodes `MaxAutoCreditMinor: 50000000` | `order/worker_shop_closed.go:91-165` | Uncollected debt invisible to AR; credit limit math corrupted |
| 5 | **Payment idempotency not DB-enforced.** `OrderPaymentLegs.IdempotencyKey` has no unique index; `PaymentLedgerEntries` none. Replay protection lives only in Redis with 24h TTL; middleware is header-optional | `spanner.ddl:1667`; `idempotency/middleware.go:45-120`; `bootstrap/bootstrap.go:419-430` | A retry after TTL double-records a financial leg |
| 6 | **AR inert behind default-off flags.** `AR_INVOICES_ENABLED` off → `OpenFromCreditLeave` silently returns nil (no invoice created); `AR_DUNNING_ENABLED` off → no step machine. Local/staging envs enable neither | `ar/service.go:18-21,88-96`; `ar/dunning_worker.go:13-16,66` | Credit leave-behind accrues profile debt while aging/dunning/collections produce nothing — AR blindness by configuration |
| 7 | **Partner REST order create ignores contract-required idempotency** | `partner/handlers.go:37-63` vs `contracts/partner.openapi.yaml:494-499` | ERP retry = duplicate orders |
| 8 | **Kafka single broker, RF=1** | `infra/k8s/kafka.yaml:42-47` | Entire event/webhook/EDI backbone dies with one pod |
| 9 | **Inventory negative-stock clamp hides shrinkage.** `AdjustStock` silently floors at 0 | `inventory/repository.go:166-169` | Stock drift masked; reservation math trusts clamped numbers |
| 10 | **All status integrity is app-level.** 155 tables, 1 CHECK constraint; FSM validator at 4 call sites vs ~65 direct status writes | `spanner.ddl:1273`; `order/state_machine.go:14-81`; call sites `order/service.go:1523,2153`, `order/preorder_sweeper.go:168,241` | One new ad-hoc writer = illegal transitions, invisible to the DB |
| 11 | **Fail-open payer authorization.** Payer GET/PUT check ownership only *if claims exist*; `POST /v1/payers` has no role gate | `payment/crud_handlers.go:52-57,76-81`; `paymentroutes/routes.go:43` | IDOR-shaped exposure under any auth-bypass configuration |
| 12 | **Committed hygiene hazards** — `.env.local` (with `JWT_SECRET`) committed; 90MB compiled `backend-go` binary tracked; `bootstrap.go.bak`, `spanner.ddl.orig` stale artifacts; patch scripts at root | repo root | Secret rotation required; repo confusion |

## 7.2 What is genuinely strong (do not refactor away)

- **Transactional outbox, correctly built**: event rows buffered inside the Spanner closure and committed in the same `txn.BufferWrite` as the state mutation (`order/repository_spanner.go:28-38,187`); relay with lease claiming (`outbox/spanner_store.go:87-170`), 250ms tick, jittered backoff, stuck-event watchdog (`outbox/relay.go:88-203`); Kafka publisher with `RequiredAcks=all`, no auto-topic-creation (`outbox/kafka_publisher.go:79-90`); consumer dedup on `event_id` (`kafka/event_dedup_middleware.go:17-18`).
- **Optimistic concurrency enforced** on orders, credit profiles, inventory, AR dunning (`order/repository_spanner.go:208-215`; `credit/repository.go:238,299,377,485`; `ar/service.go`).
- **Money discipline**: integer minor units end-to-end; basis points for VAT/discounts; `math/big` FX with half-away-from-zero rounding, overflow checks, fail-closed on missing rate (`fxrates/convert.go:101-146`); currency-mismatch gates at checkout, webhook, and payment service.
- **Fail-closed production validation**: dev secrets and memory fallback rejected under `PEGASUSX_ENV=production` (`bootstrap/config_validate.go:11-49`; `REQUIRE_INFRA_ADAPTERS` default true).
- **Payment webhook verification done right**: Global Pay settlement re-verified out-of-band against the gateway before acceptance (`payment/global_pay_webhook.go:80-91`) — defeats forgery even with a leaked secret.
- **Idempotency middleware** scoped by principal+route with SHA-256 body hash, 409-on-conflict semantics (`idempotency/middleware.go:61-86`) — mirrored client-side across web/iOS/Android.
- **Test coverage**: ~25% test-file ratio in the backend including a full FSM transition matrix; Spanner emulator integration tests in CI; only 1 TODO and 3 panics in 812 non-test files; 57 mock/stub hits concentrated in known areas (payment stubs, simulator, fiscal FAKE provider).

## 7.3 The AR/credit engine as implemented (vs plan)

`CREDIT_COLLECTIONS_ENGINE_PLAN.md`'s "current state" is stale — the plan was largely executed. Implemented and verified: credit profiles with limit/balance/reserved/available/version; `CheckOrder` gate (`no_credit_limit`, `credit_limit_breached`); idempotent reservations at order create; same-txn `MarkBalanceInTxn` on credit leave; `SupplierCreditPrograms`/`RetailerPaymentTerms` with policy audit; AR open items with buckets `CURRENT/1_30/31_60/61_90/90_PLUS`; dunning step machine `DUE_SOON(T−3)→OVERDUE→ESC1(+7)→ESC2(+14)→CREDIT_HOLD(+21)→COLLECTIONS(+30)` with grace, monotonic advancement, delinquency bump on first overdue, auto-hold via `HoldRelationship`, inbox+FCM notify — all wired (`bootstrap/bootstrap.go:1247-1270`) behind `AR_*` flags. **Deliberately absent: risk scoring** (removed in Phase A; three gap-closure runbooks still instruct enabling its dead flag — docs error, not code error). **Actually missing:** SMS/email/WhatsApp transports; fee schedule; payout execution; refund initiation.

## 7.4 Fiscal reality, precisely

- Framework: immutable attempt rows, `FISCALIZING/FISCAL_FAILED` states, max-3 attempts, force-complete with closed reason codes (role-gated), Kafka-consumed worker results, cash-bag freeze while fiscal open, late-webhook money guard — all real (`order/fiscal.go`).
- Providers: `ProviderFromEnv` selects `PEGASUS` (default; commercial receipts, `"tax_ofd": false`), `FAKE` (SSMR hooks), `MY_SOLIQ` (real EHF HTTP client with idempotency + permanent/retry classification — **but signer never injected**), `GLOBAL_PAY`. Misconfiguration hard-fails (`hardFailProvider`) rather than silently faking — good.
- `fiscal/uzbekistan.go` is a dead mock referenced only by its own test.
- Tax regime versioning is real: country-scoped, effective-dated VAT regimes with overlap validation (`tax/service.go:38-80`), consumed by fiscal snapshots and credit notes.
- **Net:** you cannot legally close a Soliq-mandated sale today. The distance is small in code (inject an EDS signer, sandbox credentials, then prove `PX_E2E_SOLIQ_SANDBOX_OK`) and large in procurement (credentials, certification).

# 8. Recommendations

## 8.1 Product scope — what to change

1. **Sell deep-single-distributor, not thin-marketplace — for now.** The runtime is single-supplier by construction; the vertical chain (factory→warehouse→fleet→store) is the rare asset. Position as the operating system for one large distributor's whole chain, with the integration layer (§4) as the sales wedge into retail chains. Defer marketplace commerce (Phase 3–5) until Gates below produce evidence.
2. **Monetize credit, not distribution margin.** The market evidence (MaxAB-Wasoko, Udaan, Jumbotail) and the codebase agree: the credit spine is built and dormant. Completing AR activation + off-app dunning + fee schedule + payouts is the revenue path.
3. **Rename or build the "factory" app.** Today it is a dispatch hub; either scope it honestly or fund MES-lite (§6.5).
4. **Treat the Control Tower honestly.** The live-network map broadcasting random data (`simulator/control_tower.go:53-79`) must be env-gated to demo builds only; the real twin projection (`twin/consumer.go`) should feed it.

## 8.2 Architecture — what to change

1. **Route every status write through the validator.** Make `ValidateStatusTransition` the only path (repository-level guard), and add DB-level defense: Spanner CHECK on `Orders.Status` membership at minimum. Today: 4 call sites vs ~65 direct writes.
2. **DB-enforce financial idempotency.** Unique indexes on `OrderPaymentLegs.IdempotencyKey`, `PaymentLedgerEntries` idempotency column; the 24h Redis TTL is a performance layer, not the guarantee.
3. **Kafka HA before enterprise promises.** 3 brokers, RF=3, min.insync.replicas=2; add an outbox-relay DLQ (exhausted publishes currently stay unpublished with only a log line, `outbox/relay.go:135-152`).
4. **Kill the stub-success philosophy.** Any PSP executor returning fabricated success (`gp_*_stub_`) must hard-fail outside explicitly-marked dev envs; the dormant `WebhookReconciler` must refuse stub refs.
5. **Collapse client duplication.** ~12k lines mobile and ~8k web are copy-paste (per repo measurement); fold AutoUpdater/WS clients into `mobile-android-kit`/`mobile-ios-kit` and grow `ui-kit` — it is the prerequisite for accessibility/localization ever being fixed once instead of 4–6 times.
6. **Generate the API client from the OpenAPI contracts** instead of maintaining 164 hand-written methods against 255 paths (drift is inevitable; the contracts now exist).

## 8.3 Prioritized gap list

**P0 — correctness & legality (days-to-2-weeks each; do first):**

| # | Action | Evidence |
|---|---|---|
| P0-1 | Fix capture routing key (`GLOBALPAY`→`GLOBAL_PAY`); remove the in-txn optimistic `CAPTURED` leg — record legs only after PSP confirmation; make backorder sweeper capture path synchronous | `payment/service.go:653`; `order/service.go:1899-1929` |
| P0-2 | Inject an EDS signer into MY_SOLIQ (or hard-block the provider until injectable); obtain Soliq sandbox credentials; prove `PX_E2E_SOLIQ_SANDBOX_OK` | `order/fiscal_provider.go:129,232-234` |
| P0-3 | Remove/gate all `gp_*_stub_*` success paths; guard `WebhookReconciler` against stub refs | `payment/global_pay_executor.go:112-320`; `payment/reconciliation.go:57` |
| P0-4 | Shop-closed worker: fail the transition (not warn) when credit profile unreadable; use `credit.Available()` math incl. `ReservedMinor` | `order/worker_shop_closed.go:91-165` |
| P0-5 | Unique indexes on payment-leg/ledger idempotency keys; honor `Idempotency-Key` on partner order create | `spanner.ddl:1667`; `partner/handlers.go:37-63` |
| P0-6 | Enable `AR_INVOICES_ENABLED`/`AR_DUNNING_ENABLED` wherever credit delivery is enabled — or block credit leave when AR is off (never allow debt without aging) | `ar/service.go:88-96` |
| P0-7 | Kafka RF=3; outbox relay DLQ | `infra/k8s/kafka.yaml:42-47`; `outbox/relay.go:135-152` |
| P0-8 | Fail loudly on negative stock instead of clamping; reconcile report for existing clamps | `inventory/repository.go:166-169` |
| P0-9 | Role-gate `POST /v1/payers`; fail-closed ownership on payer GET/PUT | `payment/crud_handlers.go:52-81` |
| P0-10 | Rotate secrets in committed `.env.local`; purge tracked binaries; delete stale `.bak`/`.orig` artifacts and root patch scripts | repo root |
| P0-11 | Warehouse Android scanner: wire to `WarehouseApi` or delete; CI grep against `TODO: Inject` | `warehouse-app-android/.../ScannerViewModel.kt:22,47` |
| P0-12 | Gate the random-data Control Tower simulator to demo builds | `simulator/control_tower.go:53-79` |

**P1 — structural product truth (2–8 weeks):**

1. **Platform admin console** (tenant lifecycle, KYB, suspension, flags, fees, observability) + `PLATFORM_ADMIN` role — prerequisite for any second tenant and for trust.
2. **Freeze multi-supplier registration** until Gate 5 Phase 1 lands (registration currently mints tenants the runtime misattributes, `supplier/service.go:433-447`).
3. **Master-data sync API** (partner upsert of products/prices/stock) + webhook coverage expansion beyond 4/155 events + EDI CONTRL/APERAK.
4. **WMS mobile floor execution** (pick waves, putaway, counts on Android/iOS) + FEFO/cold-chain operator surfaces.
5. **Off-app dunning transports** (SMS/email/WhatsApp) — the field-agent-displacement blocker for collections.
6. **Payout execution + refund initiation + fee schedule + invoices** (the monetization loop).
7. **Enable the shipped-off integration transports** (AS2/SFTP) in manifests; SFTP host-key pinning.
8. **Supplier mobile offline**; warehouse Room-parity queue; iOS telemetry buffer; server telemetry ACK.
9. **Server-side offline POS acceptance + fiscalization** (close the client-queue/server-deferred contradiction).
10. **Pricing authority engine** (rules + approval + margin floors).

**P2 — planning quality (4–10 weeks):**

1. **Deploy optimizer-core for real** (AR image + replicas ≥1) so routing is optimization, not heuristic; keep the H3/bin-pack fallback as fallback.
2. **Default-on safety-stock v2 and forecast algo** after shadow evidence; **auto-flip auto-order to `place`** only at ≥80% shadow acceptance with human+env signoff (the harness exists: `RetailerAutoOrderShadowProposals` 30-day stats).
3. **Partial allocation / backorder queue** — insufficient stock is currently a hard error and a lost sale (`allocation/service.go`).
4. **POS demand feed + real weather-driven demand adjustments** into forecasting (weather ingest exists; POS feed is the residual).
5. **Real S&OP capacity model** replacing `factories × 700 × 7`; remove `sku-projection-%d` literals.
6. **Serial tracking**; DataMatrix for UZ marking; EPCIS.
7. **Credit risk scoring v2** — only as a product decision (it was deliberately removed); if re-added, data-driven from the now-accumulating `DelinquencyCount`/repayment history, and delete the stale runbook references to the dead flag.

**P3 — scale / enterprise (quarters):**

1. **Multi-tenancy Phase 1** (request-scoped tenancy, ~150–250 files; fail-closed middleware; per-tenant rate limits; outbox partition by tenant) per the accepted ADR — then Phase 2 cross-supplier cart/split.
2. **Certified EDIFACT / certified 1C exchange package / Drummond AS2.**
3. **Global product master** (GTIN-keyed + offers + match queue) as the marketplace prerequisite.
4. **BI/data sink** (BigQuery/parquet) and Excel exports.
5. **Marketplace commerce (Phases 3–5)** — decided on evidence from the above, not on the premise that no competitor exists.

## 8.4 Process — keep the honesty, fix the drift

The repo's audit culture is an asset (its Substance Gate found real theatre). Three mechanical fixes: **(1)** status claims must carry dates and get auto-staled by CI (three runbooks currently instruct enabling a flag that no longer exists); **(2)** evidence artifacts cited by gates must exist at referenced paths (the Substance Gate's marker-gate log is a dangling reference); **(3)** one canonical tree — the docs name `/Users/shakhzod/ATOMOS/pegasusX` canonical while the live, newer tree is this one (202 backend files diverge).

---

# 9. Closing Assessment

The engineering is not the problem. The transactional outbox, the retry-safe closures, out-of-band webhook verification, integer-money discipline with `math/big` FX, idempotency mirrored across three client platforms, the FSM with battle-scar comments, a 25%-test backend with 1 TODO — that is careful, rare work.

The problems are threefold and each is fixable:

**First, the money path has live bugs worse than any tracked gap.** Broken card capture with optimistic ledger writes, a legal fiscal provider that cannot sign, stub-success PSP paths, and AR that ships inert. These are days-to-weeks of work and they gate everything — legality, trust, and monetization.

**Second, the autonomy stack is built and switched off.** Forecast, safety stock, auto-order, dunning, WMS execution, EDI/AS2 — implemented, flag-gated, default-off. The unlock is evidence, not code: run shadow mode, publish acceptance/accuracy numbers, then flip flags with discipline. The shadow harness already exists for exactly this.

**Third, the platform property is missing.** Single-supplier runtime, no admin console, no tenant lifecycle. The schema was designed multi-tenant and the runtime was not; closing that is the largest single investment (150–250 files for Phase 1) and the correct sequencing is after the money path is fixed and the integration layer is complete — building tenant isolation before partner APIs stabilize would rebuild them twice.

Against the six questions this report set out to answer: **(1)** field-agent displacement is real for the order pad (~35–40% today, ~65% at P1-complete) and structural-human for cash and exceptions — a hybrid future, not a wipe-out; **(2)** versus O9/Kinaxis/Blue Yonder/ERP+WMS/marketplaces, PegasusX loses on planning sophistication and wins on vertical factory→shelf transactional depth no category leader has; **(3)** alignment with incumbent 1C/SAP estates is real but incomplete — file-drop and pull-export today, with specific P0/P1 gaps (idempotent writes, master-data push, webhook coverage, certified formats) before a chain adopts without re-keying; **(4)** the exact ideal of a unified transactional platform with near-zero-human routine replenishment plus physical execution roles does not exist publicly — PegasusX has ~70% of its substance and ~10% of its platform property; **(5)** per-role detail shows deeply wired clients with concentrated, fixable weak points; **(6)** the recommendations above sequence the repair: P0 money-path correctness first, then structural truth (admin console, integration completeness, mobile floor execution), then planning quality, then scale.

Fix the money path. Prove the autonomy in shadow. Then decide the marketplace question on evidence.

