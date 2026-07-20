# Fiscal hard-gate — state transition table vs current SM

**Authority:** ADR-009 · **Live graph today:** `order/state_machine.go`  
**Date:** 2026-07-20

## 1. Current statuses (code)

Live graph: `apps/backend-go/order/state_machine.go` (`ValidateStatusTransition`).  
Money capture: `service.go#CollectCash` sets `NextStatus: StatusCompleted` + `emitPaymentCleared` + `emitOrderFinalized`.

| Status | Role in delivery money path |
|--------|-----------------------------|
| `ARRIVED` | Driver at retailer |
| `AWAITING_PAYMENT` | After QR scan; card/cash selection |
| `PENDING_CASH_COLLECTION` | Retailer chose cash; driver must collect |
| `DELIVERED_ON_CREDIT` | Credit goods without door cash |
| `COMPLETED` | Terminal (today: often set **on collect-cash**) |

### Exact edges today (money-relevant, from SM)

```
ARRIVED                  → AWAITING_PAYMENT | PENDING_CASH_COLLECTION | COMPLETED | DELIVERED_ON_CREDIT | CANCEL_REQUESTED
AWAITING_PAYMENT         → COMPLETED | PENDING_CASH_COLLECTION
PENDING_CASH_COLLECTION  → COMPLETED
DELIVERED_ON_CREDIT      → COMPLETED
```

**Hard-gate removes** every `→ COMPLETED` above that is not preceded by fiscal SUCCESS / force.

## 2. New statuses

| Status | Meaning |
|--------|---------|
| `FISCALIZING` | Payment captured; OFD attempt in flight or pending worker |
| `FISCAL_FAILED` | Last attempt failed; retry or force-complete |

**Not added as order status:** `PAYMENT_CAPTURED`, `FORCE_COMPLETED` (use fiscal row + event).

## 3. Transition table (target)

### 3.1 Unchanged pre-delivery

| From | To | Who / trigger |
|------|-----|----------------|
| `PENDING` | `LOADED`, `CANCELLED`, `DELAYED` | warehouse / cancel |
| `LOADED` | `IN_TRANSIT`, `CANCELLED`, `CANCEL_REQUESTED`, `DELAYED`, `PENDING` | dispatch / payload |
| `IN_TRANSIT` | `ARRIVED`, `CANCELLED`, `CANCEL_REQUESTED`, `PENDING` | driver |
| `SCHEDULED` / `AUTO_ACCEPTED` | (existing preorder edges) | midnight guard |
| `BACKORDERED` | (existing) | stock |

### 3.2 Delivery + payment (changed)

| From | To | Allowed today? | Target hard-gate |
|------|-----|----------------|------------------|
| `ARRIVED` | `AWAITING_PAYMENT` | yes | **keep** (QR / card session) |
| `ARRIVED` | `PENDING_CASH_COLLECTION` | yes | **keep** (confirm-cash without scan edge) |
| `ARRIVED` | `DELIVERED_ON_CREDIT` | yes | **keep** |
| `ARRIVED` | `COMPLETED` | **yes (remove)** | **forbidden** |
| `ARRIVED` | `CANCEL_REQUESTED` | yes | keep |
| `ARRIVED_SHOP_CLOSED` | `AWAITING_PAYMENT`, `DELIVERED_ON_CREDIT` | yes | keep |
| `AWAITING_PAYMENT` | `PENDING_CASH_COLLECTION` | yes | keep |
| `AWAITING_PAYMENT` | `COMPLETED` | **yes (remove)** | **replace with → `FISCALIZING`** after card clear |
| `AWAITING_PAYMENT` | `FISCALIZING` | no | **add** (card clear / capture) |
| `AWAITING_PAYMENT` | `DELIVERED_ON_CREDIT` | optional | allow if credit chosen at door |
| `PENDING_CASH_COLLECTION` | `COMPLETED` | **yes (remove)** | **replace with → `FISCALIZING`** on collect-cash |
| `PENDING_CASH_COLLECTION` | `FISCALIZING` | no | **add** (cash capture + fiscal request) |

### 3.3 Fiscal gate (new)

| From | To | Trigger |
|------|-----|---------|
| `FISCALIZING` | `COMPLETED` | Worker: fiscal attempt `SUCCESS` |
| `FISCALIZING` | `FISCAL_FAILED` | Worker: OFD fail after retries |
| `FISCAL_FAILED` | `FISCALIZING` | `POST .../fiscal/retry` new attempt |
| `FISCAL_FAILED` | `COMPLETED` | `POST .../force-complete` (**ADMIN or WAREHOUSE_ADMIN** + reason) only |

### 3.4 Credit (§9.1: fiscal only when money received)

| From | To | Trigger |
|------|-----|---------|
| `DELIVERED_ON_CREDIT` | `FISCALIZING` | Settlement capture (cash/card) — money received |
| `DELIVERED_ON_CREDIT` | `COMPLETED` | **Forbidden** without money+fiscal, except audited force |

**No door fiscal** on credit leave-behind.

### 3.5 Terminal / reconciliation (mostly unchanged)

| From | To | Notes |
|------|-----|--------|
| `COMPLETED` | ∅ | terminal |
| `CANCELLED` | `RECONCILIATION_REQUIRED` | keep |
| `RECONCILIATION_REQUIRED` | `COMPLETED`, `CANCELLED` | `COMPLETED` here still requires fiscal policy review if money moved |

## 4. CollectCash / card-clear behavior change

### Today

```
PENDING_CASH_COLLECTION --CollectCash--> COMPLETED
  + PAYMENT_CLEARED + ORDER_FINALIZED in same path
```

### Target

```
PENDING_CASH_COLLECTION --CollectCash--> FISCALIZING
  + ledger capture
  + OrderFiscalReceipts(PENDING)
  + FISCAL_RECEIPT_REQUESTED (outbox)
  + Orders.FiscalStatus=PENDING
  − no ORDER_FINALIZED yet

Worker success:
  FISCALIZING --> COMPLETED
  + attempt SUCCESS
  + FISCAL_RECEIPT_SUCCEEDED
  + PAYMENT_CLEARED (if not emitted at capture — prefer once)
  + ORDER_FINALIZED
```

**Payment-cleared timing (agent rule):** Emit `PAYMENT_CLEARED` **once** at capture (money in hand) **or** once at fiscal success — not both. **Preferred:** emit `PAYMENT_CLEARED` at capture (ledger truth), keep `ORDER_FINALIZED` only at fiscal SUCCESS / force. Tracking must distinguish “paid” vs “fiscalized.”

## 5. Driver / API response contract

| Order status | Driver UI |
|--------------|-----------|
| `FISCALIZING` | “Waiting for fiscal receipt…” |
| `FISCAL_FAILED` | “Fiscal failed — Retry / call supervisor” |
| `COMPLETED` + fiscal SUCCESS | Done + show receipt id if needed |
| `COMPLETED` + FORCE_SKIPPED | Done with ops exception flag |

HTTP collect-cash response: `state=FISCALIZING`, `attempt_id`, not `COMPLETED`.

## 6. Validation function sketch

```go
// After hard-gate:
// ARRIVED: remove StatusCompleted from allowed next
// AWAITING_PAYMENT: remove StatusCompleted; add StatusFiscalizing
// PENDING_CASH_COLLECTION: remove StatusCompleted; add StatusFiscalizing
// add cases StatusFiscalizing, StatusFiscalFailed
```

Regression tests must include:

- `PENDING_CASH_COLLECTION → COMPLETED` **denied**
- `ARRIVED → COMPLETED` **denied**
- `FISCALIZING → COMPLETED` **allowed**
- `FISCAL_FAILED → COMPLETED` **allowed only via force path** (enforced in service, not only SM)

## 7. SSMR impact

| Marker | Update |
|--------|--------|
| Lifecycle vertical cash path | Expect `FISCALIZING` then worker → `COMPLETED` |
| `PX_E2E_FISCAL_*` | New markers (ADR-009) |
| Fake OFD | Success by default; fail when `X-SSMR-Fiscal-Fail: 1` or order id prefix |

## 8. Implementation checklist for agents

- [x] Apply `20260720_order_fiscal_receipts.ddl` (SSMR emulator verified 2026-07-20)
- [ ] Constants + SM + tests
- [ ] Events triple-lock
- [ ] CollectCash / card clear rewrite
- [ ] Worker + fake provider
- [ ] Retry + force-complete APIs
- [ ] Dispatcher fanout (explicit cases — no silent drop)
- [ ] Driver/retailer/supplier UI
- [ ] Update lifecycle vertical + full SSMR
