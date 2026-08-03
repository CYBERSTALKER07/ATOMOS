# ADR-009: Fiscal hard-gate at delivery

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



**Status:** Accepted (§9 product locks signed off 2026-07-20)  
**Date:** 2026-07-20  
**Context:** Uzbekistan delivery-time cash/card must produce a durable fiscal receipt (OFD / `my.soliq.uz` class). ADR-001 keeps **pay-at-delivery only**. Today `CollectCash` and card-clear paths can move orders to `COMPLETED` and emit `PAYMENT_CLEARED` / `ORDER_FINALIZED` **without** a reconstructible fiscal receipt ID.

## Decision

1. **Hard-gate:** After payment is **captured** at delivery (cash geofenced collect or card webhook clear), the order must enter **`FISCALIZING`**. It may become **`COMPLETED` only if** required fiscal attempt(s) are `SUCCESS`, or an audited **force-complete** marks fiscal `FORCE_SKIPPED`.
2. **Async OFD:** Fiscalization is **never** synchronous on the driver HTTP path. Capture writes ledger + outbox `FISCAL_RECEIPT_REQUESTED` in the **same** Spanner RW txn; a worker calls OFD.
3. **Immutable attempts:** Every attempt (success/fail/force) is a new row in `OrderFiscalReceipts`. No in-place mutation of success rows.
4. **No soft-complete in production:** Completing while OFD is down without force path is forbidden.
5. **Integer Tiyin only:** All fiscal amounts are `INT64` minor units (`UZS`).
6. **Fiscal only when money is received** — never on unpaid credit leave-behind.

### §9 product locks — **Boss signed off**

| # | Decision | **Locked value** | Notes |
|---|----------|------------------|-------|
| 1 | Credit delivery fiscal | **Only when money is received** | `DELIVERED_ON_CREDIT` does **not** fiscalize at the door. Fiscal starts on settlement capture (cash/card clear). |
| 2 | OFD aggregation | **Both: per-supplier receipt + order total** | Today: 1 order = 1 retailer = 1 supplier → one supplier receipt **is** the order total. Model receipts **per `SupplierId`**; order-level `FiscalStatus` / amount is the rollup (sum of successful supplier legs). Future multi-supplier: N supplier OFD docs + one order total view — no rework of PK. |
| 3 | Force-complete actors | **`ADMIN` + `WAREHOUSE_ADMIN`** | `reason_code` + actor audit required. Driver never force-completes. |
| 4 | OFD provider | **Fake for now** | Interface + fake provider for SSMR/local. Real `my.soliq.uz` / OFD adapter later behind `FISCAL_PROVIDER` flag. |

### Non-decisions (explicit)

- Pay-at-delivery unchanged (ADR-001).
- No pre-pay fiscalization.
- No floating-point money.
- No microservices for fiscal (worker mode in modular monolith is enough — ADR-003).

## Mapping to current code (gap)

| Today (`order/service.go`) | Problem |
|----------------------------|---------|
| `CollectCash` → `NextStatus: COMPLETED` + `PAYMENT_CLEARED` + `ORDER_FINALIZED` same path | Completes without fiscal |
| `AWAITING_PAYMENT` → `COMPLETED` (card/scan paths) | Same |
| `PENDING_CASH_COLLECTION` → `COMPLETED` | Same |
| `ARRIVED` → `COMPLETED` allowed in SM | Soft path must be **removed** under hard-gate |

## Target state machine (order `Status`)

Reuse existing payment states; add fiscal terminalizing states. **Do not** add `PAYMENT_CAPTURED` — capture is expressed by leaving payment states into `FISCALIZING`.

```
# Pre-delivery (unchanged)
PENDING → LOADED → IN_TRANSIT → ARRIVED | …

# Delivery payment (existing)
ARRIVED → AWAITING_PAYMENT | PENDING_CASH_COLLECTION | DELIVERED_ON_CREDIT | CANCEL_REQUESTED
# REMOVE: ARRIVED → COMPLETED (hard-gate)

AWAITING_PAYMENT → PENDING_CASH_COLLECTION | FISCALIZING | DELIVERED_ON_CREDIT
PENDING_CASH_COLLECTION → FISCALIZING

# Fiscal hard-gate (new)
FISCALIZING → COMPLETED | FISCAL_FAILED
FISCAL_FAILED → FISCALIZING | COMPLETED   # COMPLETED only via force-complete

# Credit (§9.1: fiscal only when money received)
DELIVERED_ON_CREDIT → FISCALIZING   # only after settlement cash/card capture
# REMOVE soft DELIVERED_ON_CREDIT → COMPLETED without money+fiscal (except audited force)

# Terminal
COMPLETED → ∅
```

**Force-complete:** order becomes `COMPLETED` with `OrderFiscalReceipts.Status=FORCE_SKIPPED` and outbox `ORDER_FORCE_COMPLETED`. No separate `FORCE_COMPLETED` status (keeps clients/tracking simpler).

## Events (triple-lock — ADR-004)

| Constant | Wire type |
|----------|-----------|
| `EventFiscalReceiptRequested` | `FISCAL_RECEIPT_REQUESTED` |
| `EventFiscalReceiptSucceeded` | `FISCAL_RECEIPT_SUCCEEDED` |
| `EventFiscalReceiptFailed` | `FISCAL_RECEIPT_FAILED` |
| `EventOrderForceCompleted` | `ORDER_FORCE_COMPLETED` |

Payloads: integer `amount_minor`, `attempt_id`, `order_id`, **`supplier_id`** (leg), `retailer_id`, `trace_id`. Success includes `fiscal_receipt_id`, `fiscal_qr`. Failure includes `error_code`, `error_message`. Force includes `reason_code`, `actor_id`.

**Emit discipline:**

| Moment | Same RW txn |
|--------|-------------|
| Cash collect / card clear (capture) | Ledger as today + insert fiscal attempt(s) `PENDING` **per supplier leg** + order → `FISCALIZING` + `FISCAL_RECEIPT_REQUESTED` (per leg). Prefer **`PAYMENT_CLEARED` once at capture**. **Do not** emit `ORDER_FINALIZED` yet. |
| Worker OFD success (leg) | Attempt → `SUCCESS` + `FISCAL_RECEIPT_SUCCEEDED`. When **all required supplier legs** SUCCESS (or forced): order → `COMPLETED` + `ORDER_FINALIZED`. Today one supplier ⇒ one leg completes the gate. |
| Worker OFD fail | Attempt → `FAILED` + order → `FISCAL_FAILED` + `FISCAL_RECEIPT_FAILED`. |
| Force-complete | Attempt `FORCE_SKIPPED` (+ remaining open legs skipped as policy) + order → `COMPLETED` + `ORDER_FORCE_COMPLETED` + `ORDER_FINALIZED`. |

## Storage

- Table `OrderFiscalReceipts` (see migration `20260720_order_fiscal_receipts.ddl`) — each row is a **supplier-scoped** attempt; include `SupplierId`.
- Order denorm rollup: `FiscalStatus`, `LatestFiscalReceiptId`, `LatestFiscalAttemptId`, `FiscalizedAt` (order **total** view for UI).
- v1 single-supplier: one successful row’s amount == order total; UI may still show “supplier receipt” + “order total” as the same numbers.

## Worker

- Consume `FISCAL_RECEIPT_REQUESTED` (group e.g. `void-fiscal-worker`).
- `FiscalProvider` interface; SSMR uses fake success/fail by order id or env.
- Idempotency key = `attempt_id`.
- Retries with backoff; DLQ after N failures (order remains `FISCAL_FAILED`).

## API (additive)

| Endpoint | Role | Behavior |
|----------|------|----------|
| Existing collect-cash / card clear | DRIVER | Capture → `FISCALIZING`; response exposes fiscal pending |
| `POST /v1/order/{id}/fiscal/retry` | DRIVER, ADMIN, WAREHOUSE_ADMIN | New attempt if `FISCAL_FAILED` |
| `POST /v1/order/{id}/force-complete` | **ADMIN, WAREHOUSE_ADMIN** | `reason_code` required; audited |
| Tracking / receipts | RETAILER, SUPPLIER | Per-supplier fiscal QR when SUCCESS + order total status |

## Clients

- Driver: wait / retry / escalate; no silent complete; no force.
- Retailer: fiscal QR or “pending fiscal”; order total.
- Supplier / warehouse admin: fiscal status; force-complete with reason.

## Metrics

- `void_fiscal_attempts_total{status}`
- `void_fiscal_pending_age_seconds`
- `void_fiscal_force_complete_total`

## SSMR markers

- `PX_E2E_FISCAL_CASH_OK`
- `PX_E2E_FISCAL_CARD_OK`
- `PX_E2E_FISCAL_FAIL_RETRY_OK`
- `PX_E2E_FISCAL_FORCE_OK`
- `PX_E2E_FISCAL_NO_SOFT_COMPLETE`

## Consequences

- Lifecycle vertical and full SSMR must be updated: cash path no longer expects immediate `COMPLETED`.
- Notification dispatcher must fan out fiscal events (prefix or explicit cases — avoid silent multi-pod drop).
- `docs/PAYMENT_EXCEPTION_SOP.md` and delivery SOPs gain fiscal failure / force procedures.
- Compliance: PSP multi-merchant letter remains non-code (plan F5).

## Implementation order

1. DDL + denormalized columns  
2. Events triple-lock  
3. State machine + unit tests  
4. Collect-cash / card-clear emit fiscal request (no complete)  
5. Fake OFD + worker  
6. Retry / force APIs  
7. WS fanout  
8. Client UI  
9. Cash variance (F2) follow-up  
10. Export (F4) follow-up  

## References

- ADR-001 pay-at-delivery  
- ADR-003 modular monolith (worker mode)  
- ADR-004 event triple-lock  
- `order/state_machine.go`, `order/service.go` (`CollectCash`)  
- Boss plan: Fiscal Hard-Gate (2026-07-20)
