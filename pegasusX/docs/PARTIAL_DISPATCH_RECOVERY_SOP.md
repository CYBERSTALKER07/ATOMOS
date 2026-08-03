# Partial Dispatch Recovery SOP

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



When warehouse dispatch execute commits some orders but fails on a later chunk, the backend **auto-compensates** (rolls back committed routes) and operators recover by retrying.

**Backend signal:** HTTP `500` with body `{"error":"dispatch_partial_commit", "committed_routes", "failed_chunk", "total_chunks", "total_routes", "detail", "compensated"}`. The idempotency key is released on failure — a retry with the same or a new key is safe.  
**Related:** [`WAREHOUSE_EXCEPTION_SOP.md`](./WAREHOUSE_EXCEPTION_SOP.md), [`REAL_WORLD_CASE_MATRIX.md`](./REAL_WORLD_CASE_MATRIX.md)

---

## What happened

Smart or manual dispatch applied **partial** assignments: earlier chunks wrote manifests/assignments; a later chunk failed (capacity, freeze lock, optimizer timeout, Spanner conflict).

The backend immediately runs **partial-commit compensation** that unwinds the already-committed routes. The response's `compensated` flag tells you whether the rollback succeeded.

---

## Recovery steps

1. Read the error body: `compensated: true` means state was rolled back cleanly — the batch is back to pre-dispatch state.
2. Open warehouse portal `/dispatch` (or native Dispatch tab) and re-check the dispatch preview; committed assignments should have disappeared after compensation.
3. Address the failure cause from `detail` (capacity, freeze lock, optimizer timeout, Spanner conflict) before retrying.
4. Re-run execute with **same idempotency discipline** (the failed attempt released its key; use a fresh key for the retry).
5. Confirm `DISPATCH_COMMITTED` WS on portal; verify fleet live map shows drivers.

---

## Do not

- Re-dispatch already-committed orders without checking manifest state.
- Run supplier CEO override and warehouse execute simultaneously on same warehouse scope.
- Ignore freeze-lock conflicts — release lock or wait for ai-worker sweep.

---

## Escalation

If the response reports `compensated: false`, compensation failed and orphaned manifest rows may remain: capture `trace_id`, warehouse id, order ids, and engage engineering with SSMR reproduction steps (`PX_E2E_DISPATCH_CAPACITY_OK`). Do not re-dispatch that warehouse scope until engineering confirms cleanup.
