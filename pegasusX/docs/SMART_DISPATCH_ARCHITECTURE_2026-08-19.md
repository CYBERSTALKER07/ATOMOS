# PegasusX Smart Dispatch Architecture

**Date:** 2026-08-19  
**Scope:** `pegasusX/`  
**Status:** Code-grounded architecture explanation. This document is not a release certificate.

## 1. Current Dispatch Pipeline

```text
dispatch request
  → fetch eligible orders
  → fetch available fleet
  → remove freeze-locked orders/drivers
  → build dispatch job
  → optimizer for dense batches
  → validate optimizer result
  → deterministic H3/bin-pack fallback
  → acquire execution freeze lock
  → create/update manifests
  → emit assignment events
```

The shared planner is used by supplier, warehouse, and factory dispatch paths:

- `apps/backend-go/supplier/dispatch_execute.go`;
- `apps/backend-go/warehouse/dispatch_execute.go`;
- `apps/backend-go/factory/dispatch_execute.go`;
- `apps/backend-go/dispatch/plan/optimize.go`.

## 2. Input Snapshot

The dispatch job uses a bounded snapshot containing:

- supplier scope;
- warehouse or home node;
- eligible pending orders;
- retailer coordinates;
- order volume;
- receiving windows;
- available drivers;
- vehicle capacity;
- cold-chain and hazardous constraints;
- order priority and driver score.

The client does not select an arbitrary warehouse or driver as authority. The backend resolves the eligible scope and fleet.

## 3. Freeze-Lock Protection

Before planning, active freeze locks are loaded. Locked orders and drivers are removed from the candidate set through `dispatch/freeze_lock.go`.

Manual dispatch creates a transactional warehouse freeze lock and emits:

```text
FREEZE_LOCK_ACQUIRED
FREEZE_LOCK_RELEASED
```

This prevents a human operator and the optimizer from changing the same dispatch scope concurrently.

## 4. Optimizer Selection

The optimizer is preferred only for dense batches.

Default behavior:

```text
12 or more stops → optimizer attempt
fewer than 12 stops → deterministic dispatch
```

The threshold is controlled by:

```text
DISPATCH_AI_MIN_STOPS
```

The orchestration lives in `apps/backend-go/dispatch/plan/optimize.go`.

## 5. Optimizer Validation

Optimizer output is never trusted directly. PegasusX validates that:

- every route references a known driver;
- driver capacity is valid;
- total route volume fits within the 95% capacity buffer;
- route loaded volume matches the sum of assigned order volumes;
- a non-empty input cannot produce an empty route result.

Invalid or failed optimizer results use the deterministic fallback.

Source labels include:

```text
optimizer
fallback_phase1
fallback_validation_rejected
pure_small_batch
```

Product-facing classification is:

```text
OPTIMAL
HEURISTIC
```

The system must never claim `OPTIMAL` when the fallback handled the route.

## 6. Deterministic Smart-Fit Fallback

The fallback uses the shared dispatch package for:

- H3 spatial grouping;
- capacity-aware bin packing;
- receiving-window compatibility;
- route scoring;
- local search;
- oversize route splitting;
- orphan and no-capacity reporting.

The fallback is the safety path when the optimizer is disabled, unavailable, times out, returns invalid capacity, or returns no usable routes.

## 7. Multi-Objective Scoring

`dispatch.ScoreCandidate` considers:

```text
volume fit
+ spatial fit
+ order priority
+ driver score
+ shop-closed risk
+ receiving-window slack
− empty-mile cost
```

Default weights are:

| Objective | Weight |
|---|---:|
| Volume fit | 25% |
| Spatial fit | 20% |
| Priority | 20% |
| Driver score | 15% |
| Shop-closed risk | 10% |
| Window slack | 5% |
| Empty-mile cost | 5% |

Hard rejection rules apply before scoring:

- cold-chain order on a non-refrigerated vehicle;
- hazardous order on a non-certified vehicle;
- route capacity violation;
- invalid driver or vehicle.

Road-network distance is optional. Haversine remains the fallback and the response must expose the matrix source honestly.

## 8. Execution and Persistence

After a plan is accepted, the execution path:

1. expands oversize routes;
2. acquires the execution freeze lock;
3. creates or updates manifests;
4. assigns orders, drivers, and vehicles;
5. persists route and manifest state;
6. emits assignment and manifest events;
7. invalidates affected caches;
8. broadcasts supplier and warehouse updates;
9. releases the execution lock.

Factory dispatch remains on the factory manifest plane. Supplier/warehouse last-mile dispatch remains on the supplier manifest plane. These planes must not be merged.

## 9. What Is Strong Today

PegasusX Smart Dispatch has:

- deterministic fallback;
- optimizer validation;
- H3 spatial grouping;
- a 95% capacity buffer;
- freeze locks;
- driver and order filtering;
- cold-chain and hazardous restrictions;
- receiving-window awareness;
- explainable optimizer-source labels;
- tests for scoring, volume, fallback, hierarchy, locks, and local search.

## 10. Current Limitations

1. Small batches use heuristics instead of the optimizer.
2. Real SKU volume data is not universal, so some volume remains estimated.
3. Road distance is optional; Haversine is still used when no road matrix is available.
4. External carrier tendering and acceptance are not a complete transport workflow.
5. Fuel, carbon, warehouse loading time, and full service-level objectives are not all hard constraints.
6. Dispatch apply requires more concurrency testing around order assignment, manifest creation, retries, and lock release.
7. The optimizer cannot override payment, fiscal, market, warehouse, or freeze-lock rules.
8. The current live `pegasusX` tree does not expose the legacy queue/apply filenames from earlier Pegasus documentation; verified paths are supplier, warehouse, and factory dispatch execution through `dispatch/plan/optimize.go`.

## 11. Target Evolution

Hard constraints should remain:

```text
scope
market
capacity
fiscal
payment
freeze locks
driver assignment
temperature
receiving windows
```

Soft objectives can expand to:

```text
H3 distance
empty miles
fuel cost
driver score
shop-closed risk
retailer priority
carbon
route balance
warehouse loading time
```

The optimizer may improve the plan, but the deterministic fallback must remain available and must pass the same validation rules before a manifest is persisted.
