# 0. Executive Verdict

> **HISTORICAL / FROZEN — do not plan from this file alone.**
> Current SoT: [`PROD_READINESS_SEQUENCE.md`](../../PROD_READINESS_SEQUENCE.md) · [`ECOSYSTEM_GAP_REGISTER_2026-08-12.md`](../ECOSYSTEM_GAP_REGISTER_2026-08-12.md) · [`FEATURES_BY_APP_ROLE.md`](../../FEATURES_BY_APP_ROLE.md).
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
