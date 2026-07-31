# O9-1 — Segmentation + Constrained Allocation (extract)

**Full spec:** [PegasusX_O9_Gap_Closure_Implementation_Plan.md](./PegasusX_O9_Gap_Closure_Implementation_Plan.md) — Tier 1 (O9-1)

## Repo-realistic constraints

- **No multi-warehouse line splits** — respects `order/allocation_wiring.go` rejection.
- **One warehouse per order** — unchanged.
- Feature flag `ConstrainedAllocationEnabled` (default **false**).

## Phase 1 (this track)

- DDL: `RetailerSegments`, `SkuClasses`, `ServicePolicies`, audit columns on `OrderLineAllocations` (or `AllocationDecisions`).
- Segmentation bootstrap from existing `RetailerCreditScores` + order volume (admin seed path; no silent auto-tag).
- Priority helper: segment + SKU class + `RiskTier` → `PriorityScore`.
- Decision logging in `allocation/service.go`.
- Flag + tests; staging with flag on (observe audit only if fair-share deferred).

## Phase 1.5 (separate PR)

- Real **cross-order** priority allocation under global scarcity (queue/sweeper or batch allocator).

## Implementation order

1. DDL  
2. Segmentation bootstrap  
3. Priority helper  
4. Decision logging in `allocation/service.go`  
5. Flag + tests  
6. Staging with flag on (observe only)

## Key files to read first

- `apps/backend-go/allocation/service.go`
- `apps/backend-go/order/allocation_wiring.go`
- `apps/backend-go/credit/retailer_credit_score_worker.go`
- `apps/backend-go/bootstrap/bootstrap.go` (flag wiring pattern from `ALLOCATION_REQUIRED`)
