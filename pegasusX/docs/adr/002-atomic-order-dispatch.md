# ADR-002: Atomic whole-order dispatch

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



**Status:** Accepted  
**Date:** 2026-06-22  
**Context:** Splitting one retailer order across multiple trucks causes operational chaos (partial deliveries, reconciliation, driver disputes).

## Decision

Dispatch bin-packing assigns **whole orders only**. An order either fits entirely on one truck route or becomes an **orphan** with an explicit capacity warning. Smart dispatch distributes orders across the fleet automatically; manual mode assigns a single truck.

Background dispatch plan warming pre-solves the full undispatched pool; subset selection filters cached routes without re-solving.

## Consequences

- `dispatch/binpack.go` never chunks `OrderChunk` across routes.
- Warehouse UI: smart mode hides truck picker; manual mode requires explicit driver.
- `plan_fingerprint` / `plan_stale` guards execute path against preview drift.
- `DISPATCH_PLAN_UPDATED` WS event invalidates client previews.

## References

- `dispatch/binpack.go`, `warehouse/dispatch_plan_warmer.go`, `warehouse/dispatch_plan_filter.go`
- `packages/ws-refresh-contract` — `DISPATCH_PLAN_UPDATED`
