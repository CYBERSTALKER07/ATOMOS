# 04 Algorithms And Planning

> **HISTORICAL / FROZEN — do not plan from this file alone.**
> Current SoT: [`PROD_READINESS_SEQUENCE.md`](../../PROD_READINESS_SEQUENCE.md) · [`ECOSYSTEM_GAP_REGISTER_2026-08-12.md`](../ECOSYSTEM_GAP_REGISTER_2026-08-12.md) · [`FEATURES_BY_APP_ROLE.md`](../../FEATURES_BY_APP_ROLE.md).
> Body is a point-in-time snapshot; re-verify any claim against code before acting.


_Source: subagent `59b342b1-9810-4397-a9a4-4cf3175c0981` from End-Product Reality Report session (2026-08-07)._

# PEGASUSX / ATOMOS — END-PRODUCT REALITY REPORT
## Algorithmic & Planning Core Audit — Ground Truth from Code Only

**Audit basis:** All claims cite `file:line` from the code. Docs were read only to compare documented math vs implemented math; divergences are flagged. Repo root: `/Users/shakhzod/Desktop/V.O.I.D/pegasusX`.

**Verdict legend:** `REAL ALGORITHM LIVE` = real math, wired to a running worker/endpoint · `SIMPLE HEURISTIC` = runs in prod but is rule-based/scored, not optimization/ML · `SKELETON` = code exists but is a no-op, stub, or not deployed · `ABSENT` = nothing in code.

---

## 1. DEMAND FORECASTING

**Verdict: REAL ALGORITHM LIVE** (three classical statistical models + classification + accuracy tracking; no ML)

### What's actually implemented

**Holt-Winters multiplicative seasonality** — `apps/backend-go/planning/forecast/holtwinters.go:78-95` (`FitHW`) and `:97-102` (`HWStep`):
- Level: `L_t = α·(y_t/S_{t-m}) + (1-α)(L_{t-1} + T_{t-1})`
- Trend: `T_t = β·(L_t − L_{t-1}) + (1−β)·T_{t-1}`
- Seasonal: `S_t = γ·(y_t/L_t) + (1−γ)·S_{t-m}`
- Forecast: `ŷ_{t+h} = (L_t + h·T_t)·S_{t-m+h}` (`ForecastHW`, `:120-132`)
- Weekly period `m = DefaultHWPeriod = 7` (`:25`); defaults α=0.3, β=0.05, γ=0.3 (`:23`).

**Croston-SBA (intermittent demand)** — `planning/forecast/croston.go:38-56` (`FitCroston`), `:74-91` (`ForecastCroston`):
- Demand-on-nonzero smoothing: `z' = z_prev + α(d_t − z_prev)`
- Interval smoothing: `p' = p_prev + α(Δ_t − p_prev)`
- SBA bias correction: `ŷ = (1 − α/2)·z/p` (`:86-90`, comment at `:83-85`). Cap ≥ 0.

**SES (smooth/erratic)** — `planning/forecast/ses.go:38-58` (`FitSES`), `:60-68` (`ForecastSES`):
- `s' = s_prev + α(y_t − s_prev)`; forecast horizon = last level.

**Series classification (Syntetos–Boylan–Croston)** — `planning/forecast/classify.go:23-38` (`ClassifySeries`):
- `ADI = days / nonzeroCount` (average demand interval), `CV² = (σ(x≠0)/μ(x≠0))²`
- Rules (`:15-21`): ADI ≥ 1.32 & CV² ≥ 0.49 → Lumpy; ADI ≥ 1.32 → Intermittent; CV² ≥ 0.49 → Erratic; else Smooth.
- Model routing — `fit.go:28-38` (`ForecastSeries`): Smooth/Erratic → Holt-Winters if `fit_hw` else SES; Intermittent/Lumpy → Croston-SBA; low-variance fallback → `TrailingMean7` (`fit.go:45-52`, `series.go:34-49`).

**Runner / materialization** — `planning/forecast_runner.go:46-53` (`RunBaselineForecast`): loads sparse demand, builds dense daily series (`series.go:20-31`), applies per-day-of-week seasonal multiplier (`forecast_runner.go:56-59`), writes rows to `DemandForecastBaseline`. Confidence derived from WAPE: `confidence = 1 − clamp(WAPE,0,1)`, floored at 0.05 (`confidenceFromWAPE`, `fit.go:134-143`; residual bands P10/P90 in `fit.go:107-131`).

### Inputs / outputs
- **In:** per-(supplier, warehouse, product) daily demand history (read in `forecast_runner.go`), optional `SeasonalTemplateOverrides` (`seasonalcore/reader.go:26-59`), with hard-coded builtins **holiday_peak ×1.35 (Nov 15–Jan 5), summer_surge ×1.15 (Jun 1–Aug 31)** (`seasonalcore/templates.go:35-36`).
- **Out:** `DemandForecastBaseline` (BaselineQty, P10, P90, ModelUsed, Confidence) consumed by: safety-stock demand mean (`replenishment/safety_stock.go` `AvgBaselineDemand`), reorder suggestion batch (`replenishment/reorder_suggestion_batch.go`), warehouse 90-day plan (`warehouse/plan90_dispatch.go:14-41` reads `SUM(BaselineQty)`), forecast accuracy job.

### Accuracy tracking (the "accuracy tracking systems" commit claim — TRUE)
- `planning/accuracy.go:58-80` (`RunDailyAccuracy`): joins `DemandForecastBaseline` vs actuals, computes **WAPE** (`Σ|err| / Σactual`), **Bias** (`Σerr / Σactual`), **TrackingSignal** (`Σerr / MAD`) — formulas in `planning/forecast/fit.go:147-169` (`WAPEBias`).
- Writes `ForecastAccuracyDaily`; raises alert if `|TrackingSignal| > 4.5` (`accuracy.go`, tracking-signal guard).
- Separate binaries: `cmd/planning-forecast/main.go`, `cmd/planning-accuracy/main.go` (both exist in `apps/backend-go/cmd/`).

### Doc-vs-code check
`docs/FORECAST_ALGO.md` documents Holt-Winters, Croston-SBA, SES, SBC cutoffs 1.32 / 0.49, WAPE/Bias — **matches code**. No divergence found. ML: **ABSENT** (no gradient boosting / NN anywhere in `planning/forecast`).

### Gaps vs design
- No automatic α/β/γ optimization (grid search exists only in test helper `holtwinters_test.go`).
- Seasonality beyond weekly + two hard-coded windows requires manual DB overrides.

---

## 2. SAFETY STOCK / REORDER POINTS

**Verdict: REAL ALGORITHM LIVE** (dual-mode: legacy heuristic default, service-level v2 behind flag)

`apps/backend-go/replenishment/safety_stock.go`:
- **Legacy (default):** `LegacyReorderPoint` (`:145-147`): `ROP = burn·lead·(1 + 15%)`. Toggle `SafetyStockV2Enabled()` reads `SAFETY_STOCK_V2_ENABLED` env (`:29-33`).
- **V2 (service level):** `SafetyStockUnits` (`:94-101`):
  
  `SS = z · √(L·σ_d² + d̄²·σ_L²)` — the classic demand-and-lead-time-variability formula.

  `ComputeReorderPoint` (`:104-106`): `ROP = d̄·L + SS`.
- **Z-score table** — `NormalZ` (`:53-83`): 0.90→1.282, 0.95→1.645, 0.975→1.960, 0.99→2.326, 0.999→3.090 (nearest-bucket interpolation for others).
- **σ_d from forecast residuals** — `ResidualSigmaD` (`:150+`): stddev of signed errors from `ForecastAccuracyDaily` over trailing window (ties safety stock directly to measured forecast error — a genuinely good loop).
- **σ_L / L̄ from observed transfers** — `ObservedLeadStats` reads realized lead times from `FactoryInternalTransfers` (`safety_stock.go`).
- **d̄** — `AvgBaselineDemand` averages `BaselineQty` from `DemandForecastBaseline` (`safety_stock.go`).

**Where enforced:**
- `replenishment/engine.go` — warehouse replenishment scan (ROP breach → insight with urgency CRITICAL/WARNING).
- `replenishment/reorder_suggestion_batch.go` — retailer reorder suggestions (`RunBatch`), merges `DemandAdjustments` velocities with ROP v2/legacy.

**Doc check:** `docs/SAFETY_STOCK.md` formula `z·√(Lσ_d² + d̄²σ_L²)`, Z-table, residual σ_d, observed lead stats — **matches code 1:1**.

**Gaps:** v2 is flag-gated and likely off in prod; σ_L requires enough completed internal transfers or falls back to configured lead-time stddev.

---

## 3. AUTO-ORDER / REPLENISHMENT

**Verdict: REAL ALGORITHM LIVE (with policy-gated autonomy)** — full loop exists end-to-end, but the fully-touchless path is narrow.

### Trigger chain (two distinct loops)

**Loop A — Supplier/warehouse replenishment (inventory → internal transfer):**
1. `replenishment.Engine.StartCron` (scheduled worker, started in `runtime_workers.go`) scans warehouse inventory vs burn rates (`replenishment/engine.go`).
2. ROP computed per §2; urgency classified; `ReplenishmentInsights` row written.
3. Auto-execution: `replenishment/touchless.go` `tryTouchlessApprove` — auto-approves insight if supplier policy allows (`replenishment/policies.go`: `AutoApproveEnabled`, `MinConfidence`, `MaxDailyTransferUnits` cap via `CountAutoApprovedToday`, `TargetServiceLevelBps`, lead times) and confidence ≥ threshold → creates `FactoryInternalTransfers` + emits events. "CRITICAL" urgency can also auto-create transfers in engine.
4. Manual path: human approval executes via `replenishment.FulfillApprovedInsight` (also exposed as governed agent action `AgentApproveInsight` in `planning/executor.go:53-59`).

**Loop B — Retailer auto-order (store-side, the one your question targets):**
1. `demand/worker_sensing.go` (scheduled): loads active `DemandSignals`, computes base velocities from order history, applies day-of-week / payday / signal factors, clamps adjustments, upserts `DemandAdjustments`. Weather signal ingestion is real: `demand/worker_weather.go:97-138` hits **Open-Meteo HTTP API** (`provider: "open-meteo"`).
2. Hook: `bootstrap/bootstrap.go:1535` `demandSvc.SetAfterSensingHook(...)` → runs `replenishment/reorder_suggestion_batch.go` `RunBatch` → computes suggested qty from adjustments + safety stock → `RetailerReorderSuggestions`.
3. `retailer/auto_order_worker.go` `RunAutoOrderWorker` (scheduled, in `runtime_workers.go`): per retailer with `RetailerAutoOrderSettings`, loads candidates (inventory-grounded R,s,S proposals, reorder suggestions, or AI predictions), applies scope policy filter (variant/product/category/supplier/global), executes per mode:
   - `off` → nothing; `shadow` → log only; `draft` → creates draft order needing human confirm; **`place` → creates the order directly.**
4. Note: `apps/ai-worker/synthesis/engine.go` explicitly **skips** inserting `AI_PREORDER` drafts when `AUTO_ORDER_INVENTORY_GROUNDED` is enabled (heuristic AI path superseded by inventory-grounded path).

**Doc check:** `docs/AUTO_ORDER.md` modes off/shadow/draft/place, scope policy, candidate chains — **matches code**. `scaffold_auto_order.py` at repo root: **does not exist** (referenced in query; file absent — scaffolding already merged into Go code).

### ⛳ Can the system auto-generate a store replenishment order with zero human touch? — **YES, conditionally.**
Exact path: `demand.RunDemandSensingWorker` (`demand/worker_sensing.go`) → `afterSensingHook` (`bootstrap/bootstrap.go:1535`) → `replenishment.RunBatch` (`replenishment/reorder_suggestion_batch.go`) → `retailer.RunAutoOrderWorker` (`retailer/auto_order_worker.go`) → candidate load → scope filter → **`executePlace`** mode writes a real order.
**Chain breaks / human gates:** (a) retailer must have `ExecutionMode = "place"` (default is `off`); (b) if mode is `draft`, a human must confirm the draft; (c) supplier-side Loop A touchless requires `AutoApproveEnabled` + confidence floor in `replenishment/policies.go`; otherwise the insight waits for manual approve or the governed-agent approval (`planning/executor.go:53`).

---

## 4. ROUTING / DISPATCH OPTIMIZATION

**Verdict: REAL ALGORITHM (Python OR-Tools CP-SAT VRP) — but SKELETON IN PROD (replicas: 0); live path is SIMPLE HEURISTIC (H3 + scored BinPack + 2-opt).**

### The real solver (built, not deployed)
- `services/optimizer-core/server/contract_solver.py` — Python, **Google OR-Tools `pywrapcp`**: multi-depot VRP, capacity dimension, max-stops, **time windows**, max route duration, vehicle eligibility (cold-chain / hazmat). First solution `PATH_CHEAPEST_ARC`, metaheuristic `GUIDED_LOCAL_SEARCH`.
- Contract: `packages/optimizer-contract` (shared `SolveRequest`/`SolveResponse` types used by both Go client and Rust/Python servers).
- Go client: `apps/backend-go/dispatch/optimizerclient/client.go` — builds `contract.SolveRequest`, **distance matrix built in Go** (OSRM via `routing/osrm.go` with circuit breaker, Haversine fallback), posts to optimizer-core, maps to `dispatch.AssignmentResult`.
- Orchestration: `dispatch/plan/optimize.go` — calls solver only for dense batches **≥ `DISPATCH_AI_MIN_STOPS` (default 12)**; validates result; **falls back to H3 BinPack** on failure/timeout/empty/validation rejection.
- **Deployment reality:** `infra/k8s/optimizer-core/deployment.yaml:9` `replicas: 1`, but prod overlay patches it to **0**: `infra/k8s/overlays/prod/kustomization.yaml:44-50` ("Keep scaled to 0 until a real optimizer-core AR image exists"). Same in ssmr overlay (`overlays/ssmr/kustomization.yaml:13-31`, "Heuristic fallback until replicas ≥ 1"). **So the OR-Tools solver never runs in prod today.**

### The actually-live heuristic
- `dispatch/binpack.go` — "Smart Fit": H3 res-7 cell grouping, retailer orders atomic (no-split), first-fit at 95% capacity (`TetrisBuffer = 0.95`), oversized orders split-if-allowed else orphaned.
- Multi-objective candidate scoring — `dispatch/score.go`: weighted blend of volume fit, spatial fit, order priority, driver score, shop-closed risk, window slack, empty-mile cost.
- Local search — `dispatch/localsearch.go:6-16`: **2-opt + bounded 3-opt** on stop order (`twoOptTour`, `threeOptBounded(…,24)`), nearest-neighbor seed (`:19-26`), Haversine distances. `ResequenceStops` used for continuous replan with freeze-locks.
- **Rust "cpsat.rs" is NOT CP-SAT** — `services/optimizer-core/server-rust/src/solver/cpsat.rs` is a greedy priority-sort + swap local search for factory slot assignment; `vrp.rs` is nearest-neighbor + 2-opt. Neither is wired into dispatch prod path (Rust service also not the deployed one).
- Doc check: `docs/OPTIMIZER_AND_ROUTING_RUNTIME.md` explicitly states Python OR-Tools is the real solver, Go owns the distance matrix, fallback always on, optimizer-core **not deployed to prod GKE** — **doc matches code/k8s exactly.** `docs/AUTO_DISPATCH_IMPROVEMENT_PLAN.md` self-describes the system as "not AI magic… hybrid deterministic" — honest and accurate.

### ETA computation
- `eta/calculator.go:21` `CalculateETAs(now, driverLat, driverLng, profile, stops, shopClosedRates)` — **pure Haversine heuristic**: per-leg distance → drive time using historical speed profile, + average stop service time, × congestion factor, × shop-closed-rate adjustment; widens windows & lowers confidence on thin data.
- Consumed by `eta/service.go:95` (initial) and `:249` (recompute on driver location update; deletes+rewrites `RouteETAs` in a txn, emits outbox event). OSRM used for route geometry/matrix (`routing/osrm.go`), **not** for ETAs. Google Routes API: configured in env but not the ETA path.

### Auto-dispatch
- `warehouse/auto_dispatch.go:28` `StartAutoDispatchWorker` — ticks every `WAREHOUSE_AUTO_DISPATCH_INTERVAL_SEC` (default 60s, `:120-130`), lists warehouses with `AutoDispatchEnabled`, debounced via cache, calls `ExecuteDispatch{Mode: "AUTO", AcceptPartial: true}` (`:85-90`) → commits manifests, broadcasts `DISPATCH_COMMITTED` WS event (`:108-116`).

---

## 5. ALLOCATION

**Verdict: REAL ALGORITHM LIVE (two modes), but partial allocation / backorders: ABSENT.**

- `allocation/service.go` — `AllocateOrderTxn` with two strategies:
  - **First-fit (legacy):** `allocateFirstFit` — first warehouse with stock wins.
  - **Policy-constrained (v2):** `allocation/constrained.go` — resolves line context via `segment.Service` (retailer risk tier, SKU class, priority score), generates warehouse candidates, sorts by score + slack, picks best. Flag: `CONSTRAINED_ALLOCATION_ENABLED` wired in `bootstrap/bootstrap.go:765`.
- **Lot-level reservation is FEFO/FIFO, separate layer:** `stocklots/fefo.go` `ReserveFEFOInTxn` — FEFO for perishables (shelf-life filtered), FIFO otherwise; called from `order/inventory_reservation.go:45` during order placement; allocation itself invoked at `order/allocation_wiring.go:132`.
- **Partial/backorder:** allocation returns "insufficient stock" error — **no partial line allocation, no backorder queue.** The only split logic in the system is dispatch-time volume splitting (`binpack.go`), which is a delivery concern, not inventory allocation.
- Bonus: `replenishment/mei_engine.go` `RunMEIONetwork` — multi-echelon network balance scan (per-warehouse days-cover, critical/warning SKU counts, transfer recommendations between warehouses) and `replenishment/echelon_targets.go:42` `ComputeEchelonTarget`: `target = ⌈burn·horizon·(serviceLevelBps/10000)⌉` — heuristic echelon targets, not true MEIO optimization (no network flow solver).

---

## 6. PRICING AUTHORITY

**Verdict: SKELETON (pricing) / SIMPLE HEURISTIC LIVE (promotions).**

- `pricing/` is 4 small files. `pricing/service.go` just delegates to repository: resolve price from `PriceList`/`PriceListItem` (`pricing/models.go` — PriceListId, Sku, UnitPriceMinor, MinQty, effective window). **No rule engine, no authority/approval workflow, no margin floors.**
- `docs/PRICING_AUTHORITY_RULES.md` is literally a stub: *"Operational stub — expand before production hypercare."* — documented design vs code: **both empty.**
- **Promotion engine is real:** `promotion/evaluator.go` `PickBestPromotion` (selects best active promo: active date window, retailer scope, product/category scope, min qty / min subtotal) and `ApplyQuote` (applies discount in basis points to cart lines); CRUD + checkout quote + list-price resolution in `promotion/service.go:35-186`.

---

## 7. CREDIT DECISIONING

**Verdict: SIMPLE HEURISTIC LIVE (limits + dunning state machine); credit scoring: REMOVED/ABSENT.**

- **Limit check** — `credit/service.go` `CheckOrder`: `allowed ⇐ currentBalance + reservedAmount + orderAmount ≤ creditLimit`, plus profile status gates: Blacklisted / Frozen / Inactive / Closed → deny. Programs/terms via `credit/policy.go` (`SupplierCreditProgram`, `RetailerPaymentTerms`, `CreditPolicyAudit` on enable/disable; effective-terms resolution; `PolicyV2Enabled` flag).
- **Reservation lifecycle** — `credit/reserve.go:12-40` `Reserve` (idempotent per order, emits `CreditProfileChanged`), `ReleaseReserve`, `ConvertReserve` (reservation → balance on credit leave).
- **Scoring — explicitly removed:** `credit/service.go` contains `p.RiskTier = "" // scoring product removed`. No PD model, no scorecard, no formula. Delinquency is a **count** (`BumpDelinquencyCount`), not a score.
- **Dunning** — `ar/dunning.go` `DesiredDunningStep`: step machine DUE_SOON → OVERDUE → ESCALATED → CREDIT_HOLD → COLLECTIONS driven by (now vs dueDate + gracePeriod); `ShouldBumpDelinquency`, `ShouldAutoHold`. Worker `ar/dunning_worker.go` started when `AR_DUNNING_ENABLED` (`runtime_workers.go:121`); hooks wired in `bootstrap/bootstrap.go:1248-1254` (auto-hold → credit freeze, delinquency bump, notify).
- **Aging buckets** — computed in `ar/service.go` (bucket keys 1_30 / 31_60 / 61_90 / 90_PLUS from invoice dueAt).
- Doc check: `CREDIT_COLLECTIONS_ENGINE_PLAN.md` describes a scoring/collections engine — **diverges from code: scoring portion unimplemented (deliberately removed); dunning machine implemented as documented.**

---

## 8. LABOR / WAREHOUSE PLANNING

**Verdict: SIMPLE HEURISTIC LIVE (driver scoring + zone capacity); pick waves: ABSENT.**

- `laborcapacity/worker_score.go` (nightly, in `runtime_workers.go`): driver score = weighted sum of on-time rate, completion rate, damage rate (inverse), shop-closed rate (inverse), feedback score, stops/hour.
- `laborcapacity/worker_capacity.go`: zone capacity snapshot = Σ(driver available hours × stops/hour × driver score) vs assigned orders → used/available capacity.
- `warehouseops/`: receiving/putaway/count tasks exist (`stocklots/counting.go`, `picking.go`, `seal_gate.go`) but **no pick-wave planning math** (no wave batching algorithm, no labor standards).
- 90-day demand-driven dispatch planning: `warehouse/plan90_dispatch.go:14-41` aggregates `DemandForecastBaseline` into per-product forward demand.

---

## 9. AI WORKER

**Verdict: SIMPLE HEURISTIC LIVE. LLM calls: ABSENT. Vision/planogram: ABSENT.**

- `apps/ai-worker/main.go` — Kafka consumer + HTTP server. Handlers: synthesis engine, predictive push, planning ingest, and an **optimizer endpoint** (`optimizer.Handler`) — a **Clarke-Wright Savings** heuristic (`apps/ai-worker/optimizer/clarke_wright.go:11-40`: `saving(i,j) = d(depot,i)+d(depot,j)−d(i,j)`, greedy merge, smallest-fit vehicle). This is *another* VRP heuristic living in ai-worker; the dispatch prod path uses `optimizerclient` → optimizer-core (§4), not this one.
- `synthesis/engine.go` — supplier recommendations + optional `AI_PREORDER` drafts from **recency/volume/value heuristic boosts** (no model inference). Skips preorder insert when `AUTO_ORDER_INVENTORY_GROUNDED` on.
- `predictivepush/analyzer.go:36-66` — SQL pattern mining: orders grouped by (retailer, product, supplier, day-of-week) over 8 weeks, `HAVING COUNT(*) ≥ 4`, confidence threshold 0.75 → predicted demand events. Pure SQL heuristic — decent, but not ML.
- **No LLM anywhere:** `.env.example` has no OPENAI/ANTHROPIC/GEMINI keys; grep finds no LLM client. Document parsing/vision: none. `docs/PLANOGRAM_VISION_PLAN.md` is explicitly an *implementation plan* — planogram CV (YOLO sidecar) is **not implemented** ("Later: sidecar CV").

---

## 10. SIMULATOR / TWIN

**Verdict: TWIN = SIMPLE HEURISTIC LIVE (event projection); SIMULATOR = SKELETON (mocks for dev/demo).**

- `twin/service.go` + `twin/consumer.go` — Kafka event consumer projecting RouteTwin/StopTwin state, with Prometheus metrics (`twin_update_success_total`, latency histogram). A read-model projection, not a physics/policy simulator.
- `simulator/global_pay.go:1-8` — **Global Pay payment gateway mock** (HMAC-signed, same URL surface as `checkout-api.globalpay.uz`), mounted only when `GLOBAL_PAY_ENV ∈ {local, dev}`; register-guarded against prod.
- `simulator/control_tower.go:53-79` — **fake telemetry**: `generateMockNetwork()` / `generateMockH3Data()` broadcast **random** (`rand.Intn`) warehouse/retailer/driver node statuses and H3 densities over WebSocket every 2s for the Control Tower UI. The dashboard's "live network" is literally random data.
- `cmd/ecosystem-simulator/main.go` — a load/demo driver hitting the HTTP API.
- `demand/density_worker.go:8-10` — real event-density aggregation is a **no-op tick** ("Until a real event-source aggregation exists… no mock H3 rows written").

---

## ALGORITHM GAP LIST (ranked by business impact)

1. **OR-Tools VRP solver deployed at 0 replicas** — prod runs H3+BinPack+2-opt heuristic; route cost/density gains from real constraint optimization unrealized. (`infra/k8s/overlays/prod/kustomization.yaml:44-50`)
2. **No partial allocation / backorder logic** — insufficient stock = hard error; lost sales instead of partial fulfillment. (`allocation/service.go`)
3. **Credit scoring removed** — limit-only decisioning; no risk-based limits/pricing; `RiskTier` blanked. (`credit/service.go` "scoring product removed"; `CREDIT_COLLECTIONS_ENGINE_PLAN.md` unimplemented scoring)
4. **Safety Stock v2 flag-gated** — prod likely on legacy `burn·lead·1.15`; the good z-score/residual-σ loop dormant. (`replenishment/safety_stock.go:29-33`, `:145-147`)
5. **Touchless auto-order off by default** — execution mode defaults `off`; full loop only live for opted-in retailers/suppliers. (`retailer/auto_order_worker.go`, `replenishment/policies.go`)
6. **No ML anywhere** — forecasting is classical stats only; no promo/uplift/price-elasticity models; ai-worker is heuristics + SQL. (`planning/forecast/*`, `apps/ai-worker/*`)
7. **Pricing authority engine absent** — no rule engine, approval workflow, or margin guardrails; doc is a stub. (`pricing/service.go`, `docs/PRICING_AUTHORITY_RULES.md`)
8. **Control Tower "live" map is random mock data** — demo-grade telemetry broadcast as if real. (`simulator/control_tower.go:53-110`)
9. **No pick-wave / labor-standards planning** — capacity is driver-score aggregates only. (`laborcapacity/*`)
10. **Demand density worker is a no-op** — H3 demand density signals never computed. (`demand/density_worker.go:8-10`)
11. **Rust optimizer misnomer** — `cpsat.rs` is greedy+swap, not CP-SAT; duplicate Clarke-Wright heuristic in ai-worker vs OR-Tools in optimizer-core = solver sprawl. (`server-rust/src/solver/cpsat.rs`, `ai-worker/optimizer/clarke_wright.go`)
12. **Planogram vision / CV absent** — plan only. (`docs/PLANOGRAM_VISION_PLAN.md`)

**Bottom line:** the forecasting → safety-stock → replenishment → auto-order spine is genuinely implemented classical OR/statistics with a working (policy-gated) touchless path — a store *can* receive an auto-generated order untouched by humans when `place` mode is enabled. Dispatch is a solid deterministic heuristic in prod with a real OR-Tools solver built but unshipped. Credit scoring, partial allocation, pricing authority, ML, and CV are the real gaps; the Control Tower's "digital twin" visual is a random-data mock.