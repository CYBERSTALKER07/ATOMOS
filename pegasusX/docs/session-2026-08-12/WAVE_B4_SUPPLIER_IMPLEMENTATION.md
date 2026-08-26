# Wave B4 — Supplier ops truth
> **POINT-IN-TIME SNAPSHOT (2026-08-12) — do not treat as current status.**
> Re-verify any claim against live code before acting. Multiple ecosystem hardening phases have shipped since this audit.


**Date:** 2026-08-13  
**Master:** [`BACKEND_PARITY_MASTER.md`](./BACKEND_PARITY_MASTER.md)

## Changes

### Silent supplier inventory (overlap B2)
- `inventory.AdjustStock` emits `INVENTORY_QUANTITY_UPDATED` in same RW txn (reads warehouse/supplier/product)
- `PATCH /v1/supplier/inventory/policy` RW + `INVENTORY_POLICY_UPDATED` + cache invalidate
- Import apply emits one `INVENTORY_SYNC_COMPLETE` per session (not per-SKU)

### M-P1-14 Scenario publish consumer
- Dispatcher routes `planning.scenario.published.v1` → `handlePlanningEvent` (multi-pod SupplierHub)

### M-P1-5 Credit program lifecycle
- Events: `SUPPLIER_CREDIT_PROGRAM_CHANGED`, `SUPPLIER_CREDIT_TERMS_CHANGED`
- `UpsertProgram` / `UpsertTerms` accept in-txn emit; enable/patch/disable wired
- Dispatcher: program → supplier room; terms → supplier + retailer rooms

### M-P1-4 Control tower
- Events: `CONTROL_TOWER_PLAYBOOK_CHANGED`, `CONTROL_TOWER_RUN_CREATED`, `CONTROL_TOWER_RUN_UPDATED`
- Playbook create/update + run create/update emit in same RW txn
- Dispatcher → SupplierHub only

## Residual (documented)

- ~~Relationship terms+profile dual-write~~ **closed (2026-08-13):** Spanner `UpsertTermsAndProfile` single RW + dual outbox; memory fallback sequential  
- ~~Control tower action mega-txn~~ **closed (2026-08-13):** ApproveRun skips intermediate APPROVED; exception ACK/ASSIGN deferred into run finalize txn; FREEZE_CREDIT compensated if finalize fails  
- ~~Replenishment policy PATCH~~ **closed S-P1-2 (2026-08-13):** `UpsertPolicy` emits `REPLENISHMENT_POLICY_UPDATED` in-txn; `EnsurePolicy` seed remains silent

## Verification

```bash
cd apps/backend-go
go test ./inventory/ ./supplier/ ./credit/ ./controltower/ ./planning/ ./kafka/ ./events/ -count=1
go build -o /tmp/pegasusx-backend .
```
