# Partial Dispatch Recovery SOP

When warehouse dispatch execute commits some orders but fails on a later chunk, operators must recover without double-assigning.

**Backend signal:** HTTP `409` with `dispatch_partial_commit` (idempotency key released — safe to retry scoped execute).  
**Related:** [`WAREHOUSE_EXCEPTION_SOP.md`](./WAREHOUSE_EXCEPTION_SOP.md), [`REAL_WORLD_CASE_MATRIX.md`](./REAL_WORLD_CASE_MATRIX.md)

---

## What happened

Smart or manual dispatch applied **partial** assignments: earlier chunks wrote manifests/assignments; a later chunk failed (capacity, freeze lock, optimizer timeout, Spanner conflict).

Committed orders are **real** — do not cancel them to “reset” the batch.

---

## Recovery steps

1. Open warehouse portal `/dispatch` (or native Dispatch tab).
2. Review dispatch preview — note which orders already show `LOADED` / assigned on `/orders`.
3. Read capacity modal recommendations:
   - **Accept partial** — proceed with feasible subset.
   - **Force** — audit logged; use only with lead approval.
4. Re-run execute with **same idempotency discipline** (new key if client retried after error).
5. Confirm `DISPATCH_COMMITTED` WS on portal; verify fleet live map shows drivers.

---

## Do not

- Re-dispatch already-committed orders without checking manifest state.
- Run supplier CEO override and warehouse execute simultaneously on same warehouse scope.
- Ignore freeze-lock conflicts — release lock or wait for ai-worker sweep.

---

## Escalation

If partial commit leaves orphaned manifest rows, capture `trace_id`, warehouse id, order ids, and engage engineering with SSMR reproduction steps (`PX_E2E_DISPATCH_CAPACITY_OK`).
