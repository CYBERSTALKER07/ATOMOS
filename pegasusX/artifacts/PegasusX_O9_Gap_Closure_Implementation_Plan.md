# PegasusX — O9-Style Planning Capabilities

## Gap Closure Implementation Plan (Agent-Executable)

**Document status:** Executable specification  
**Audience:** Implementation agents / senior engineers  
**Grounding:** Codebase audit 2026-07-31 (allocation/service.go first-fit, RetailerCreditScores, replenishment/mei_engine.go network rebalance, planning/service.go heuristic scenarios, zone overrides, exception surfaces)  
**Companion to:** PegasusX_New_Feature_Areas_Detailed_Implementation_Plan.md  
**Date:** 2026-07-31  
**Constraint:** Zero breakage of existing fiscal, shop-closed, demand, settlement, twin, replan, allocation wiring, or compliance paths. All new behaviour behind feature flags.

---

## Audit Summary (current vs O9)

| # | O9 Capability | Maturity | Key Location | Gap Severity |
|---|---------------|----------|--------------|--------------|
| 1 | Segmentation + constrained / fair-share allocation | ~0–15% | allocation/service.go, credit scores only | Highest — allocation is naive first-fit; credit tiers unused in allocation |
| 2 | Control tower + automated playbooks | ~25% | zone overrides, exceptions UI, 3 agent actions, touchless | Medium — visibility exists; no rule→action engine |
| 3 | Lightweight MEIO (echelon targets) | ~40% | replenishment/mei_engine.go network rebalance | Medium — network transfers exist; no per-echelon target stock or MEIO-driven reorder |
| 4 | Scenario / what-if (twin-backed) | ~30% | planning/service.go RunScenario heuristic | Lower — sandbox exists; not state-clone simulation |
| 5 | Advanced demand sensing (ML) | ~0% ML / ~40% rules | demand/worker_sensing.go | Deferred |
| 6 | S&OP / IBP | ~20% | GetSAndOP heuristic snapshot | Deferred |

**Recommended execution order:** O9-1 → O9-2 → O9-3 → O9-4.

---

## Global Implementation Rules (apply to every feature)

1. All money is int64 minor units (tiyin). Never float.
2. Every state-changing write happens inside a Spanner ReadWriteTransaction and emits an OutboxEvent in the same transaction.
3. Idempotency keys on every external or retryable mutation.
4. Optimistic concurrency via etag / version columns where records can be concurrently modified.
5. Feature flags (config or DB) for every new worker and every new write path so rollout can be gradual.
6. No direct cross-package table access. Use the owning package's repository / service.
7. Twin, Demand, Compliance, Settlement are projections or consumers — they must not become sources of truth.
8. Tests: unit + integration for every new transition; existing test suites must stay green.
9. Migration style: additive DDL only. Never drop or rename columns that existing code reads.
10. Event names follow existing convention: `domain.action` (e.g. `allocation.fair_share_applied`).
11. Reuse existing credit tiers and OrderLineAllocations — do not reinvent scoring or allocation storage.
12. Allocation remains single-warehouse-per-order unless a later explicit multi-split feature is approved (current wiring rejects multi-warehouse line splits).

---

## TIER 1 — Highest Leverage (O9-1)

### Feature O9-1 — Retailer / SKU Segmentation + Constrained Fair-Share Allocation

#### 1.1 Goal

Replace naïve first-fit warehouse selection with priority-aware, constrained allocation that respects retailer segment, SKU velocity class, credit risk tier, and fair-share rules when stock is scarce. Enable service-level policies that drive safety stock targets and allocation priority without breaking existing single-warehouse-per-order flows.

#### 1.2 When it activates

- On every new order capture path that already calls allocation (flag-gated).
- When stock for a SKU is insufficient to satisfy all open demand in a planning window (fair-share mode).
- When a service policy requires preferential treatment (strategic retailers, high-margin SKUs, etc.).

#### 1.3 Core Behaviour

- Retailers receive a commercial segment (A/B/C or Strategic/Standard/Opportunistic) in addition to the existing RiskTier.
- SKUs receive a velocity / strategic class (A/B/C or Fast/Medium/Slow + Strategic flag).
- Service policies map (RetailerSegment × SkuClass) → allocation priority weight, target service level, max fair-share ratio.
- Allocator:
  - Loads candidate warehouses ordered by nearest + stock + capacity (existing nearest-warehouse ranking remains the base).
  - Scores each open order line by policy weight + credit RiskTier boost/penalty.
  - When aggregate demand ≤ available → normal first-fit / nearest.
  - When aggregate demand > available → fair-share proportional to weight (with floor protection for A-segment).
- Credit score / RiskTier is consulted at allocation time (currently unused there).
- All decisions are recorded on OrderLineAllocations (or additive columns) for audit and twin projection.

#### 1.4 Schema (additive)

```sql
-- Commercial segmentation (orthogonal to existing RiskTier)
CREATE TABLE RetailerSegments (
  RetailerId          STRING(36) NOT NULL,
  Segment             STRING(16) NOT NULL,   -- A | B | C | STRATEGIC | STANDARD | OPPORTUNISTIC
  Reason              STRING(256),
  EffectiveFrom       TIMESTAMP NOT NULL,
  EffectiveTo         TIMESTAMP,
  UpdatedBy           STRING(64) NOT NULL,
  UpdatedAt           TIMESTAMP NOT NULL,
) PRIMARY KEY (RetailerId);

CREATE TABLE SkuClasses (
  SupplierId          STRING(36) NOT NULL,
  Sku                 STRING(64) NOT NULL,
  VelocityClass       STRING(8) NOT NULL,    -- A | B | C
  StrategicFlag       BOOL NOT NULL DEFAULT (FALSE),
  UpdatedAt           TIMESTAMP NOT NULL,
) PRIMARY KEY (SupplierId, Sku);

CREATE TABLE ServicePolicies (
  PolicyId            STRING(36) NOT NULL,
  SupplierId          STRING(36) NOT NULL,
  RetailerSegment     STRING(16) NOT NULL,
  SkuClass            STRING(8) NOT NULL,    -- or '*' for wildcard
  PriorityWeight      INT64 NOT NULL,        -- higher = preferred (e.g. 100, 70, 40)
  TargetServiceLevelBps INT64 NOT NULL,      -- e.g. 9800 = 98%
  MaxFairShareBps     INT64 NOT NULL,        -- cap on share of scarce stock (e.g. 4000 = 40%)
  MinFairShareBps     INT64 NOT NULL,        -- floor protection
  CreditRiskBoost     INT64 NOT NULL,        -- additive weight when RiskTier is low
  Enabled             BOOL NOT NULL DEFAULT (TRUE),
  UpdatedAt           TIMESTAMP NOT NULL,
) PRIMARY KEY (PolicyId);

CREATE INDEX ServicePolicies_BySupplier ON ServicePolicies(SupplierId);

-- Optional audit of allocation decisions (if not already on OrderLineAllocations)
-- Prefer extending existing OrderLineAllocations table with:
--   AllocationMode STRING(16),  -- FIRST_FIT | FAIR_SHARE | POLICY
--   PriorityScore  INT64,
--   FairShareBps   INT64,
--   PolicyId       STRING(36)
```

#### 1.5 Service Contracts

```go
// allocation/service.go (extend existing)
type Service interface {
    // Existing
    AllocateOrder(ctx context.Context, orderID string) error

    // New
    AllocateWithPolicy(ctx context.Context, req AllocateRequest) (*AllocationResult, error)
    PreviewFairShare(ctx context.Context, supplierID string, sku string, horizon time.Duration) (*FairSharePreview, error)
}

// segment/service.go (new thin package or under credit/planning)
type SegmentService interface {
    UpsertRetailerSegment(ctx context.Context, retailerID, segment, reason, actor string) error
    UpsertSkuClass(ctx context.Context, supplierID, sku, velocityClass string, strategic bool) error
    ResolvePolicy(ctx context.Context, supplierID, retailerID, sku string) (*ServicePolicy, error)
}
```

#### 1.6 Integration Points (non-breaking)

| Existing system | How we touch it |
|-----------------|-----------------|
| allocation/service.go | Replace pure first-fit loop with scored + fair-share path behind `ConstrainedAllocationEnabled` flag |
| allocation_wiring.go | Keep single-warehouse-per-order rejection; only change which warehouse is chosen and the audit fields |
| RetailerCreditScores | Read RiskTier at allocation time; do not write |
| Order capture paths | Already call allocation; no new entry points required |
| Reorder suggestions | Later (O9-3) can consume TargetServiceLevelBps |
| Twin / Live Ops | AllocationMode + PriorityScore become visible attributes on route/order twin |
| Compliance / exceptions | Scarce-stock fair-share events can surface as soft exceptions |

#### 1.7 Event flow

- `retailer.segment.updated`
- `sku.class.updated`
- `service_policy.updated`
- `allocation.policy_applied`
- `allocation.fair_share_applied`

#### 1.8 Implementation Sequence (safe)

1. DDL migration only (RetailerSegments, SkuClasses, ServicePolicies + optional columns on OrderLineAllocations).
2. SegmentService + repository (no callers yet). Manual seed API + supplier-portal admin UI for A/B/C tags.
3. Default policies seeded per supplier (A×A = weight 100, C×C = weight 30, etc.).
4. Feature flag `ConstrainedAllocationEnabled` (default off).
5. Extend allocator: when flag off → exact current first-fit behaviour. When on → score + fair-share.
6. Wire RiskTier boost/penalty.
7. Unit tests for fair-share math (protect A-segment floors, respect MaxFairShareBps).
8. Integration: create competing orders against limited stock; assert proportions and audit fields.
9. Portal: segment tags on retailer card + policy editor (simple matrix).
10. Gradual flag rollout per supplier.

#### 1.9 Test Plan

- Unit: fair-share calculation with floors/caps; priority ordering; RiskTier influence.
- Integration: flag-off path identical to current; flag-on produces deterministic fair shares.
- Existing allocation + order + fiscal tests remain green.
- Money / inventory invariants unchanged (allocation never over-allocates available stock).
- Edge: zero policies → graceful fallback to first-fit; unknown segment → treat as C.

#### 1.10 Safety & Rollout

- Flag default off → zero behaviour change.
- Seed only; never auto-tag existing retailers without explicit admin action.
- Fair-share never reduces an already-allocated line; only applies to new allocation decisions.
- Single-warehouse-per-order contract preserved.

**Phase 1.5 (separate):** real cross-order priority allocation under scarcity — queue/sweeper or batch allocator; not part of initial O9-1 landing.

---

## TIER 2 — Operational Multipliers (O9-2)

### Feature O9-2 — Control Tower Playbooks

#### 2.1 Goal

Turn the existing exception surfaces (order exceptions, manifest exceptions, shop-closed, claims, compliance tickets, zone overrides) into a scored, prioritised command centre with rule-driven recommended and auto-executable playbooks.

#### 2.2 Core Behaviour

- Unified exception view that scores severity (impact on revenue, SLA risk, credit exposure, twin delay).
- Playbook engine: ExceptionType + Conditions → RecommendedActions[] (and optional AutoActions when policy allows).
- Actions are drawn from an allow-list that reuses existing governed agent hooks + new thin wrappers:
  - expedite / re-prioritise route
  - freeze credit / open credit note
  - reallocate stock / create supply request
  - zone override (REROUTE, FREEZE_DISPATCH, PRIORITY_BOOST)
  - broadcast template / notify
  - approve insight / open claim
- Human can one-click execute or override; every auto action is audited and feature-flagged.
- Touchless replenishment remains a special case of the same engine.

#### 2.3 Schema (additive)

```sql
CREATE TABLE Playbooks (
  PlaybookId          STRING(36) NOT NULL,
  SupplierId          STRING(36) NOT NULL,   -- or '*' for platform
  Name                STRING(128) NOT NULL,
  ExceptionType       STRING(64) NOT NULL,   -- SHOP_CLOSED | CLAIM | STOCKOUT | DELAY | FISCAL | ...
  ConditionsJson      JSON,                   -- severity >= X, segment = A, age > 2h, etc.
  RecommendedActionsJson JSON NOT NULL,      -- ordered list of action specs
  AutoActionsJson     JSON,                   -- subset that may run without human if policy allows
  Priority            INT64 NOT NULL,
  Enabled             BOOL NOT NULL DEFAULT (TRUE),
  UpdatedAt           TIMESTAMP NOT NULL,
) PRIMARY KEY (PlaybookId);

CREATE INDEX Playbooks_ByType ON Playbooks(ExceptionType, Enabled);

CREATE TABLE PlaybookExecutions (
  ExecutionId         STRING(36) NOT NULL,
  PlaybookId          STRING(36) NOT NULL,
  ExceptionRef        STRING(128) NOT NULL,  -- e.g. order:xxx or ticket:yyy
  Mode                STRING(16) NOT NULL,   -- RECOMMENDED | AUTO | MANUAL
  ActionsTakenJson    JSON,
  Actor               STRING(64) NOT NULL,   -- system or user
  Status              STRING(16) NOT NULL,   -- SUCCESS | PARTIAL | FAILED
  CreatedAt           TIMESTAMP NOT NULL,
) PRIMARY KEY (ExecutionId);
```

#### 2.4 Service Contracts

```go
// planning/playbook or controltower/playbook
type PlaybookService interface {
    Match(ctx context.Context, exception ExceptionView) ([]*PlaybookMatch, error)
    Execute(ctx context.Context, playbookID, exceptionRef string, actions []ActionSpec, actor string) (*PlaybookExecution, error)
    ListOpenScored(ctx context.Context, supplierID string) ([]*ScoredException, error)
}
```

#### 2.5 Integration Points

- Reuse existing exception list endpoints and zone override API.
- Reuse `POST /v1/supplier/planning/agent/invoke` allow-list; extend carefully with new action types behind the same governance.
- Exception Command Centre UI gains "Recommended" panel and one-click execute.
- Control Tower (`/control-tower`) surfaces the top scored exceptions.

#### 2.6 Implementation Sequence

1. DDL + PlaybookService (no auto-execution yet).
2. Seed 4–6 starter playbooks (shop-closed, stockout, delayed route, open claim, fiscal ticket).
3. Scoring function (simple weighted formula first).
4. Feature flag `PlaybookEngineEnabled`.
5. Wire Match into exception list API (additive field).
6. Execute path (human-triggered only first).
7. Auto path for low-risk actions (touchless-style) behind second flag.
8. UI: recommended actions + execution history.

#### 2.7 Safety

- Auto actions only from explicitly enabled playbooks and only the AutoActionsJson subset.
- Every execution writes PlaybookExecutions + outbox event.
- Existing exception flows continue to work if engine is off.

---

## TIER 3 — Inventory Intelligence (O9-3)

### Feature O9-3 — Echelon Targets + MEIO-Driven Replenishment

#### 3.1 Goal

Extend the existing network MEIO (`RunMEIONetwork`) so that it also maintains per-echelon target stock (central / regional / retailer-facing) driven by service-level policies from O9-1, and feeds those targets into reorder suggestions and allocation priority.

#### 3.2 Core Behaviour

- Echelon model: Warehouse → (optional Regional Hub) → Retailer.
- For each (Supplier, Sku, Echelon, Segment) compute TargetStock = demand × coverage days × service-level factor.
- Network MEIO continues to recommend inter-warehouse transfers; additionally writes echelon targets into a new table.
- Reorder suggestion worker (or ProcessSuggestion) consumes TargetStock instead of (or in addition to) the generic formula.
- Allocation can bias toward warehouses that are above their target (or protect those below).

#### 3.3 Schema (additive)

```sql
CREATE TABLE EchelonTargets (
  SupplierId          STRING(36) NOT NULL,
  Sku                 STRING(64) NOT NULL,
  WarehouseId         STRING(36) NOT NULL,   -- or special RETAILER for forward positions
  Echelon             STRING(16) NOT NULL,   -- CENTRAL | REGIONAL | FORWARD
  TargetQty           INT64 NOT NULL,
  SafetyQty           INT64 NOT NULL,
  ServiceLevelBps     INT64 NOT NULL,
  HorizonDays         INT64 NOT NULL,
  ComputedAt          TIMESTAMP NOT NULL,
  Source              STRING(32) NOT NULL,   -- MEIO | MANUAL | POLICY
) PRIMARY KEY (SupplierId, Sku, WarehouseId, Echelon);
```

#### 3.4 Integration

- `replenishment/mei_engine.go` → after network pass, also upsert EchelonTargets.
- Reorder suggestions read EchelonTargets when present; fall back to current formula.
- Allocation scoring can add a "below-target" boost.
- Feature flag `MEIOEchelonTargetsEnabled`.

#### 3.5 Sequence

1. DDL.
2. Extend MEIO engine to write targets (still network-first).
3. Update reorder suggestion path to prefer targets.
4. Optional allocation bias.
5. Portal: MeioNetworkPanel already exists — surface target vs on-hand.

---

## TIER 4 — What-If Depth (O9-4)

### Feature O9-4 — Twin-Backed Scenario Sandbox

#### 4.1 Goal

Upgrade the existing heuristic `RunScenario` into a proper what-if that clones (or snapshots) relevant twin + inventory + capacity state, applies shocks, and reports service-level, stockout, and cash impact.

#### 4.2 Core Behaviour

- Snapshot current twin routes, inventory balances, open orders, capacity for a supplier/zone.
- Apply shocks: warehouse offline, demand ±X%, lead-time +Y days, fleet reduction, promo uplift.
- Re-run lightweight allocation + MEIO projection + ETA model on the snapshot.
- Return structured impact: SLA risk delta, stockout SKUs (real), capacity breaches, projected revenue at risk.
- Keep the current heuristic path as fallback when `TwinScenarioEnabled` is off or snapshot is too large.

#### 4.3 Implementation Notes

- Prefer read-only projection tables or a short-lived Spanner database / in-memory clone; never mutate live state.
- Cache results (existing 15 min Redis pattern can stay).
- Portal PlanningBrainPanel already consumes the API — only response shape enrichment needed.

#### 4.4 Sequence

1. Snapshot helper (inventory + open demand + twin routes).
2. Shock application layer.
3. Projection runner reusing allocation + MEIO logic.
4. Feature flag + API response versioning.
5. UI: show real SKU stockouts and side-by-side before/after.

---

## Recommended Execution Roadmap

| Phase | Features | Dependency notes | Est. effort signal |
|-------|----------|-------------------|------------------|
| O9-A | O9-1 Segmentation + Constrained Allocation | Credit tiers + allocation wiring already live | Smallest conceptual add; highest leverage |
| O9-B | O9-2 Control Tower Playbooks | Exceptions + zone overrides + agent allow-list exist | Medium; pure orchestration |
| O9-C | O9-3 Echelon MEIO Targets | Network MEIO + reorder suggestions exist | Builds on O9-1 service levels |
| O9-D | O9-4 Twin Scenario | Twin + heuristic scenario exist | Deeper; do after allocation is policy-aware |

---

## Agent Instructions for Safe Landing

When implementing any feature above:

1. Read the owning package's existing service, repository, and state machine (`allocation/service.go`, `allocation_wiring.go`, `replenishment/mei_engine.go`, `planning/service.go`, credit score worker) before writing.
2. Add DDL in a new dated migration file only.
3. Wrap every new write path in the same Spanner + Outbox transaction style.
4. Feature-flag the first caller that connects a new domain to an existing flow. Default off.
5. Run the full `go test ./...` (or at least allocation + order + replenishment + planning + credit packages) before claiming done.
6. Preserve single-warehouse-per-order contract and all existing event names consumed by mobile / portal.
7. Do not let Twin, Demand, or Compliance become sources of truth for allocation or targets.
8. Document new events in the central events schema / contract package.
9. Seed default policies and segments conservatively; never auto-reclassify production retailers without an explicit admin path.

---

## Success Criteria (per feature)

- Existing tests green; new tests cover happy path + scarcity / fair-share / policy-miss edge cases.
- Flag-off behaviour is bit-identical to today's first-fit / heuristic paths.
- No increase in fiscal hard-gate failures or inventory oversell.
- Allocation never exceeds available stock; fair-share respects floors and caps.
- Playbook auto-actions are fully audited and reversible where possible.
- MEIO targets are recommendations only until reorder path is explicitly switched.
- Scenario sandbox never mutates live twin or inventory.

---

**End of plan.**

This document is the single source of truth for closing the O9-style planning gaps identified in the 2026-07-31 codebase audit. Implement O9-1 first unless business priority overrides.
