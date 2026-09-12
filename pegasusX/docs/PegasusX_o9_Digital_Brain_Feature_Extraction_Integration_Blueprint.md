# PegasusX — o9 Digital Brain Feature Extraction & Integration Blueprint

**Document status:** Destination / architecture specification. **Not wiring. Not status.** Re-verify every “existing” claim in code (`file:line`) this session before implementing.  
**Source material:** Full transcripts of o9 videos (Digital Brain, Demand Planners, Data Scientists, Integrations, Control Tower, Why every business needs a Digital Brain, Sales vs Supply Chain friction, S&OE Control Tower) plus author notes 2026-08-18.  
**Audience:** Product, architecture, implementation agents  
**Date:** 2026-08-18  
**Tree:** `pegasusX/`

**First executable slice (approved):** [`DEMAND_CLASS_IBP_SLICE.md`](./DEMAND_CLASS_IBP_SLICE.md) — persist + show SBC demand class already computed by `ForecastSeries`. Do not start GraphCube, Jupyter, SAP connectors, or place-flag flips from this file.

**Companion (problems):** [`PegasusX_o9_Demand_Planning_Problems_Extraction.md`](./PegasusX_o9_Demand_Planning_Problems_Extraction.md)

**Prior O9 gap plans (artifacts, re-verify):** [`../artifacts/PegasusX_O9_Gap_Closure_Implementation_Plan.md`](../artifacts/PegasusX_O9_Gap_Closure_Implementation_Plan.md), [`../artifacts/PegasusX_O9-1_Segmentation_Constrained_Allocation_Plan.md`](../artifacts/PegasusX_O9-1_Segmentation_Constrained_Allocation_Plan.md)

**Constraint:** Zero breakage of existing fiscal, shop-closed, demand, settlement, twin, replan, allocation, compliance, or COD paths. All new behaviour behind feature flags. Money remains int64 minor units. Event-first, shared transactional truth. Factory planning **place** and retailer auto-order **place** stay default-off (`.agents/memory/GOAL.md`).

**Core thesis:** PegasusX will **not** ship separate planning apps. Every o9-style capability is absorbed into the single ecosystem that already moves physical goods (factory → warehouse → truck → doorstep) and money. The Enterprise Knowledge Graph (EKG) equivalent is a **projection** over existing Spanner + Outbox + Kafka — not a second source of truth. Humans stay in the loop on high-value exceptions; routine paths become zero-touch only after shadow evidence.

---

## 1. Extracted o9 Capabilities Catalog

### 1.1 Enterprise Knowledge Graph + Digital Brain (foundational)

**Problem solved**  
Enterprises treat data as siloed tables. Decisions remain lagging, consensus-driven, and blind to external leading indicators. Bullwhip, missed shipments, and surprise demand shifts persist.

**Core logic / algorithm basis**

- Nodes = entities (SKU, Location, Customer/Retailer, Order, Capacity, Event, ExternalSignal).
- Edges = relationships + temporal causality (sellout → order pattern, ILI activity → immunity product sales, price change → volume drop, cut qty → true demand reconstruction).
- Knowledge = continuously updated causal models + contribution factors, not raw history.
- Auto-learning: discrepancy between expected and observed expands the model.

**Workflow / user flow**

1. Continuous ingestion (internal transactional + external signals).
2. Staging / cleansing / feature engineering.
3. GraphCube load (hybrid OLAP cube + graph relations) — **in PegasusX this is a projection, not a GraphCube product**.
4. ML / statistical models consume graph features.
5. Exceptions + contribution insights surface to planners.
6. Approved actions write back into operational systems (allocation, replenishment, pricing, expedite).

**Business logic**  
Convert discrete data → market knowledge + demand knowledge + supply knowledge → predict → prescribe. Break siloed “brains” (sales brain vs supply brain).

**PegasusX mapping**

- **Existing substrate (hypothesis until re-read):** Spanner entities + Outbox events form the transactional spine. `GET /v1/supplier/knowledge-graph` today is an entity projection (`owned_by` / `catalogs`), not causal EKG.
- **Enhance:** Lightweight Knowledge Projection (materialized views / continuous queries) that maintains causal edges and contribution factors.
- **Do not** invent a separate graph database; project over Spanner + Kafka.

### 1.2 Demand Sensing & Causal Forecasting (leading + lagging indicators)

**Problems solved**

- Bullwhip: stable sellout, crazy ordering → missed shipments, low fill rates.
- Monthly planning drumbeat ignores weekly signals (4–5 week lag).
- Consensus forecast friction between sales / supply / finance.
- History distorted by cut quantities / short shipments.
- Blindness to external drivers (search trends, ILI, holidays that move, movie releases, mobility, price changes, glance views).

**Core logic / algorithm basis**

1. **True demand reconstruction**  
   When shipment history is volatile due to supply cuts:

   ```
   CorrectedDemand(t) = SeasonallyAdjustedLinearInterpolation(Shipments + CutQty, seasonality, trend)
   ```

   Do not simply add CutQty (customers reorder the missed quantity). Use statistical interpolation that respects seasonality.

2. **Causal / multi-driver model**

   ```
   Forecast(t) = Baseline(trend + seasonality + intermittence)
                 + Σ β_i × Driver_i(t)   // price, promo, holiday, ILI, search, mobility, glance views, movie window
                 + residual ML correction
   ```

   Drivers classified as **influenceable** (price, promo, marketing spend) vs **non-influenceable** (seasonality, holidays, weather, ILI, mobility). Contribution factors shown to planners.

3. **Feature selection**  
   Correlation analysis + multicollinearity removal (“less is more”). Model architecture analysis chooses best forecasting level (SKU / category / channel) and time bucket (day / week) per segment.

4. **Tournament + blending**  
   Multiple models compete. Production forecast can be a hybrid / blended forecast (different blends by horizon).

5. **NPI / product feature similarity**  
   Embedding of product attributes → nearest historical analogues. Reduces time to create NPI forecasts.

**Workflow / user flow (Demand Planner)**

1. System surfaces exceptions (bias, high error, sudden driver change).
2. Planner sees driver contribution decomposition + “what if I change price / promo”.
3. Collaborative override with key customers (normalize ordering patterns).
4. Consensus is exception-driven, not full re-forecast every cycle.
5. Move from monthly → weekly (or daily) drumbeat where signals justify it.

**Data Science workflow (platform side)**  
Collect (internal + external) → Cleanse / compliance / outlier rules → Segment (ABC/XYZ, seasonality, intermittence) → Feature engineering → Correlation / impact analysis → Model training + tournament → Publish best recipe(s) to production forecast.

**PegasusX mapping**

- **Existing (re-verify):** `apps/backend-go/demand/worker_sensing.go` (rules-based multipliers: weather, promo, event, holiday). Forecast runner: Croston / SES / Holt–Winters + SBC class (`planning/forecast/`). POS flywheel is a **separate** analytics feed, not forecast actuals.
- **Enhance (after demand-class slice):**
  - CutQty / short-shipment correction path into versioned true-demand history (never mutate original shipments).
  - Expand driver set behind flags: search proxies, ILI-style public health (category-relevant), price history, glance-view if e-comm exists, mobility / foot-traffic proxies.
  - Driver contribution decomposition (influenceable vs not) in Brain UI.
  - ABC/XYZ + forecastability segmentation (ties to O9-1 ServicePolicies). **Demand class persist/show is the first code slice.**
  - Flag-gated tournament / blend (start simple: rules + one statistical + one ML).
  - NPI similarity (attribute embeddings or simpler category+attribute distance first).
- **Data flow:** External signals land via Integration staging → Knowledge Projection → Demand Sensing worker → AdjustedDemand events → Allocation / Replenishment / Twin consumers.

### 1.3 Agile Planning Drumbeat & Collaborative Normalization

**Problem**  
Monthly S&OP ignores weekly consumer signals. Ordering patterns remain irrational (bullwhip).

**Logic**  
Overlay weeks inside months. Detect divergence between sellout and orders. Share insights with key customers and collaboratively normalize ordering.

**PegasusX mapping**

- Planning cadence already supports weekly / daily via existing workers (flag-gated).
- Add “Customer Ordering Pattern Health” score + collaborative exception workflow (planner ↔ key retailer).
- Surface bullwhip alerts when `|order CV − sellout CV|` exceeds threshold — only when both series exist; honest empty otherwise.

### 1.4 Control Tower + Digital Twin + Prescriptive Playbooks (S&OE)

**Problems solved**  
Real-time disruptions (machine down, capacity loss, JIT policy conflict) are handled too slowly or without financial visibility. High-priority customers suffer while low-priority demand is protected by rigid policies.

**Core logic (from Control Tower video)**

1. Digital Twin models “what is” and “what could be”.
2. Capacity shock detected → system:
   - Identifies affected high-priority demand.
   - Automatically protects high-priority by sacrificing lower-priority.
   - Generates alternate expedite scenarios (source from alternate plant / warehouse).
   - Calculates incremental cost vs revenue-at-risk (int64 minor).
   - Routes for human approval (Director / Planner).
3. Policy exception: JIT plant can be allowed to build-ahead N days when earlier capacity exists; cost of inventory vs lost revenue presented.
4. Decision is rooted in business, financial, and operational constraints simultaneously.

**Workflow**  
Alert → auto-generated scenarios with money impact → approval queue (role-based) → execute (allocation change, expedite, build-ahead exception) → customer impact closed.

**PegasusX mapping**

- **Existing (re-verify):** control-tower playbooks (`controltower/`), zone overrides, exceptions UI, twin (last-mile; factory plane unavailable/unmerged), scenario sandbox (two knobs; publish does not mutate inventory).
- **Enhance with o9 patterns (later slices):**
  - Playbook library: “Capacity Loss → Protect Priority Demand → Expedite Alternatives → Cost/Revenue Trade-off”.
  - Financial impact always shown in pack minor units (revenue at risk, incremental logistics / production cost).
  - Explicit policy-exception engine (JIT override with approval).
- Ties to O9-1 fair-share / priority allocation when that flag is on. Do not merge factory and last-mile trucks.

### 1.5 Digital IBP / Integrated Business Planning 2.0

**Problem**  
Static monthly PowerPoint IBP. Separate revenue brain and supply brain. No continuous trade-off visibility (demand uplift actions vs supply feasibility + cost).

**Logic**

- Revenue scenarios: “do these commercial actions → demand becomes 100 / 120 / 140”.
- Supply scenarios answer feasibility + extra cost for each.
- Management chooses the synchronized plan that maximizes profit / service under constraints.
- Continuous, scenario-native, not monthly batch.

**PegasusX mapping**

- Existing: `GetSAndOP` heuristic (capacity vs open supply-request `ProjectedUnits`, not forecast baseline), scenario service, twin.
- Elevate later to a Digital IBP workspace on **existing** `/planning` (Plan & Brain tabs) that consumes demand drivers, constrained allocation & MEIO, control-tower financial impacts.
- Roles: BU / P&L owner, CFO, CSCO, CRO collaborate on the same scenario objects. Supplier `ADMIN` owns the process.

### 1.6 Data Science Platform & Model Lifecycle

**Capabilities**

- No-code Predict AI entry.
- Native R / Python / PySpark + JupyterHub + Git version control.
- Elastic scale (Hadoop / Spark style parallelization).
- Full process: collect → prep → feature impact → train / tournament → publish.

**PegasusX mapping**

- Phase 1: keep rules + simple statistical inside Go workers (already present). **This is the current product path.**
- Later: controlled “Model Publish” path where approved Python / R recipes (sandboxed workers or external ML service) write forecast recipes back into Demand / Knowledge Projection.
- Versioned model registry + audit of which recipe produced which forecast version.
- Do **not** make the core transactional path depend on an external notebook runtime. Do **not** add Hadoop/Spark as a required substrate.

### 1.7 Integration Fabric

**Capabilities**

- Protocols: SFTP, REST, SOAP/XML, streaming.
- Connectors + mapping templates (SAP, Snowflake, Oracle, BigQuery…).
- Four steps: Connection → Mapping → Ingestion logic → Oversight / monitoring.
- Staging layer: cleanse, transform, normalize, outlier detection.
- Alerts + metrics on every step.

**PegasusX mapping**

- Existing: Partner API, webhooks, file/EDI, dual-run (GS-P). Re-verify live vs planned (PEPPOL not live).
- Formalize staging tables + transformation jobs that feed the Knowledge Projection.
- External leading-indicator feeds land here first, flag-gated, fail-closed when unkeyed.

### 1.8 Revenue Analytics & Gap Closure

**Logic**  
Distinguish influenceable drivers. Show contribution of each. Optimize pricing, promo, marketing spend allocation to close plan-vs-forecast gaps while respecting supply constraints.

**PegasusX mapping**

- Price history exists in the pricing engine (re-verify).
- Later: “Revenue Gap” on Plan tab that links commercial actions → expected demand lift → constrained supply check → financial outcome. Natural extension of Digital IBP. Not a separate app.

---

## 2. Role × Feature Matrix

| Role | Primary Features | Key Actions | Consumes | Produces |
|------|------------------|-------------|----------|----------|
| **Demand Planner** (supplier planning) | Sensing, causal forecast, exceptions, driver contribution, collaborative override, bullwhip alerts | Review exceptions, adjust drivers, approve consensus, collaborate with key retailers | AdjustedDemand, Knowledge Projection, External signals | Forecast overrides, collaborative notes |
| **Supply / Inventory Planner** (warehouse + supplier) | Constrained allocation (O9-1), MEIO, replenishment, Control Tower playbooks, policy exceptions | Approve expedite / build-ahead, set service policies, review fair-share | Demand, Capacity, Twin, Credit/Segment | Allocation decisions, playbook executions |
| **Data Scientist / Analyst** | Model tournament, feature impact, NPI embeddings, recipe publish | Train / evaluate / publish models | Staging data, historical corrected demand | Model versions, feature importance |
| **Control Tower Operator / S&OE** | Real-time exceptions, digital twin scenarios, financial impact playbooks | Approve / reject scenarios, monitor capacity shocks | Twin state, capacity alerts, priority scores | Expedite orders, policy overrides |
| **Commercial / Revenue Owner** | Revenue analytics, influenceable drivers, gap closure scenarios | Propose pricing / promo actions | Demand drivers, plan vs forecast | Commercial action scenarios |
| **Finance / IBP Facilitator** | Digital IBP workspace, scenario comparison, cost/revenue trade-offs | Run synchronized scenarios, lock plan | All of the above | Locked IBP plan version |
| **Integration / Admin** (platform) | Staging, connectors, monitoring | Configure mappings, monitor health | Source systems | Clean knowledge inputs |
| **Key Retailer (collaborative)** | Ordering pattern health, shared insights | Review & normalize orders | Bullwhip / sellout vs order views | Collaborative commitments |

All roles operate on the **same transactional truth**. No separate planning database. New JWT roles are **not** invented unless a live path needs them — map onto existing supplier ADMIN, warehouse ops, platform admin, retailer first.

---

## 3. Feature Relationships & Consistent Data Flow

```
External Sources (search, ILI, weather, mobility, ERP, e-comm signals)
        │
        ▼
Integration Staging (cleanse, outlier, normalize, map)
        │
        ▼
Knowledge Projection / EKG-lite
  (nodes + causal edges + contribution factors + corrected history)
        │
        ├──► Demand Sensing Worker ──► AdjustedDemand events
        │         │
        │         ▼
        │    Forecast Tournament / Blend (flag-gated)
        │
        ├──► Segmentation + Service Policies (O9-1)
        │
        ├──► MEIO / Replenishment targets
        │
        └──► Control Tower / Twin
                  │
                  ▼
            Playbook Engine (prescriptive scenarios + money impact)
                  │
                  ▼
            Approval State Machines (human-in-loop)
                  │
                  ▼
            Execution writes (Allocation, Expedite, Build-ahead — flags)
                  │
                  ▼
            Outbox → Kafka → all projections stay consistent
```

**Consistency rules (non-negotiable)**

1. Demand, Allocation, Inventory, Credit, Twin are **projections or consumers**. Source of truth remains transactional tables + Outbox events.
2. Every prescribed action that changes money or stock emits an OutboxEvent inside the same Spanner RW transaction.
3. Feature flags gate every new write path and every new worker.
4. Corrected history is versioned; original shipment / completed-order history is never mutated.
5. Model recipes are versioned; every forecast carries its recipe ID when a recipe path exists.
6. Dual planes stay unmerged (factory trucks vs supplier last-mile trucks).
7. Place stays off: predictive-push / auto-order **preview** only until an explicit dual-control flip.

---

## 4. Prioritized Integration Path

**Immediate (this program’s first code slice — already specified)**

0. **Demand-class honesty** — [`DEMAND_CLASS_IBP_SLICE.md`](./DEMAND_CLASS_IBP_SLICE.md). Persist SBC `smooth` / `erratic` / `intermittent` on `DemandForecastBaseline`; Brain + warehouse forecast show it. Bind existing `ClassifySeries`. No new engines.

**Next (leverage existing O9-1 / O9-2 work — separate PRs, re-verify flags)**

1. **O9-1 enhancement:** Feed ABC/XYZ + forecastability into ServicePolicies. Use corrected demand (cut-qty aware) as input to velocity class.
2. **O9-2 enhancement:** Control Tower playbook patterns (priority protection + expedite + cost vs revenue-at-risk + JIT policy exception). Always show pack minor units.
3. True-demand reconstruction worker (seasonally-aware interpolation of cuts). Versioned series.
4. Driver contribution surface (influenceable vs not) in existing Brain / Demand UI.

**After that (P2 planning quality)**

5. Expand external leading indicators via Integration Staging (flag + honest empty / 501 unkeyed).
6. Simple tournament + blend (rules + statistical + one ML), flag-gated.
7. NPI similarity via attribute embeddings (or simpler analogue SKUs first).
8. Digital IBP on `/planning` that re-uses scenario + twin + financial playbooks. S&OP demand from forecast baseline (one-number) is a named later slice.
9. Collaborative bullwhip / ordering-pattern workflow with key retailers.

**Later (scale + autonomy readiness)**

10. Full Model Publish path for external DS recipes — transactional path must not depend on notebooks.
11. Continuous learning loop (outcome → model expansion).
12. Autonomy flags on playbook execution only after shadow accuracy evidence.

---

## 5. Concrete Algorithm & Logic Snippets

### 5.1 True Demand Correction (from cut-quantity video)

```text
// Do NOT simply Shipments + CutQty
// Customers reorder the missed quantity → double counting risk
CorrectedHistory(t) = SeasonalLinearInterpolation(
  observed_shipments,
  known_cuts,
  seasonal_factors,
  trend
)
```

Store as versioned `CorrectedDemandSeries`; original remains immutable.

### 5.2 Driver Contribution Decomposition

```text
Forecast = Baseline + Σ Contribution_i
Contribution_i = β_i × Driver_i
Classify: Influenceable = {price, promo, marketing}
          NonInfluenceable = {seasonality, holiday, ILI, mobility, search baseline}
```

UI always shows the stack and the levers the commercial team can actually pull. Missing drivers → honest unavailable, not a fake stack.

### 5.3 Control Tower Financial Playbook

```text
on CapacityShock(plant, lost_capacity, duration):
  affected = identify_open_demand_impacted()
  high_pri, low_pri = partition_by_service_policy(affected)
  protect(high_pri)
  scenarios = generate_expedite_alternatives(low_pri)
  for s in scenarios:
    s.incremental_cost_minor = calc_extra_logistics_production(s)
    s.revenue_at_risk_minor = calc_lost_revenue_if_not(s)
  route_for_approval(scenarios, role=supplier ADMIN or warehouse ops)
```

Same-market / dual-plane / geography_incomplete rules still apply. Do not auto-place factory transfers.

### 5.4 NPI Similarity

```text
embedding = embed(SKU_attributes)  // Phase 1: category + attribute hash / TF-IDF; neural later
neighbours = top_k(historical_skus with good history)
NPI_forecast = weighted_average(neighbours.history, weights=similarity × volume)
```

### 5.5 Fair-Share under Constraint (O9-1, flag-gated)

When aggregate demand > available:

```text
weight = ServicePolicy.PriorityWeight × CreditRiskBoost
share = available × (weight / Σ weights)   // with floor for A-segment
```

One warehouse per order remains product law until explicitly changed.

---

## 6. What We Explicitly Do **Not** Copy

- Separate planning database or GraphCube as source of truth.
- Multi-year big-bang transformation. Keep Chakri’s “fast innovation / Lego block” mindset — **one slice per PR**.
- Full Hadoop/Spark dependency for core path.
- Removing human approval on high-value or cash-impacting decisions until shadow evidence exists.
- Anything that breaks fiscal, Soliq, COD, shop-closed, or single-warehouse-per-order (until explicitly approved later).
- Flipping `checkout_reads_this`, terraform apply, live PSP keys, `FACTORY_PLANNING_ENABLED`, `AUTO_ORDER_PLACE_ENABLED`.

---

## 7. Success Metrics (tied to o9 claims — measure after a slice ships)

- Forecast accuracy uplift on sensed horizon (beat naive / previous baseline; FVA is a **later** metric because override layers do not exist yet).
- Time spent in forecast consensus process down (exception-based).
- Fill rate / service level on constrained SKUs up via fair-share + priority (O9-1 flag on).
- Mean time to resolve capacity / disruption exceptions down (playbook + approval).
- NPI forecast creation time down.
- Zero regression on existing transactional, fiscal, and settlement paths.

---

## 8. Next Concrete Actions for Agents

1. **Execute** [`DEMAND_CLASS_IBP_SLICE.md`](./DEMAND_CLASS_IBP_SLICE.md) first.
2. Update [`../artifacts/PegasusX_O9-1_Segmentation_Constrained_Allocation_Plan.md`](../artifacts/PegasusX_O9-1_Segmentation_Constrained_Allocation_Plan.md) to consume corrected demand + ABC/XYZ from sensing — **after** class persist exists.
3. Flesh O9-2 playbooks with capacity-loss and JIT-exception patterns + financial impact objects (pack minor units).
4. Design additive schema (later PRs) for: `CorrectedDemandSeries`, `DemandDriverContributions`, `ExternalSignalIngest`, `ModelRecipeVersion`, playbook financial impact fields. DemandAssumption / ForecastLayer / RiskOpportunity objects are specified in the companion problems doc.
5. Extend Integration Staging contract for leading-indicator feeds (flag + fail-closed).
6. Map new surfaces onto existing roles before inventing JWT roles.
7. Keep every new write path behind feature flags and emit OutboxEvents in the same RW txn.

---

**End of blueprint.**  
This file is the catalog for turning o9 video transcripts into integrated capabilities inside PegasusX. It is **not** a certificate that those capabilities are wired. Subsequent demand, control tower, allocation, and IBP work should reference this extraction **and** re-verify live code.
