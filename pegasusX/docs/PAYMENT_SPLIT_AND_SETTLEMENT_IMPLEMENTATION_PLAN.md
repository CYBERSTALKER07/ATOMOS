# Payment Split and Settlement Implementation Plan

**Date:** 2026-08-19  
**Scope:** `pegasusX/` only  
**Status:** Implementation plan. The current code remains partially correct; this document is not a completion claim.

## 1. Executive Verdict

The payment system has useful durable primitives, but payment splitting is not yet financially correct end to end.

### Current defects

1. `POST /v1/delivery/split-payment` emits an outbox event but does not create `OrderPaymentLegs` or a payment ledger record.
2. The endpoint accepts arbitrary positive sums instead of requiring the split to equal the outstanding delivered amount.
3. It does not verify that the authenticated driver owns the order.
4. It does not validate individual leg signs, currency, payment status, or overflow.
5. It reads the order outside the write transaction, allowing a race with collection, refund, reassignment, or cancellation.
6. `payout.Repository.SumLegs` combines all currencies into one total and labels the batch with the first currency encountered.
7. The documented per-supplier settlement-slice authority is not present in the live tree. Current payout authority is batch-level `CommissionMinor` and `NetPayoutMinor`.
8. Concurrent payout generation can pass the read-before-insert checks in two requests; the unique index prevents duplicate rows but the loser returns an insertion error instead of replaying the existing batch.

### Target verdict

The corrected path must make these invariants true:

```text
captured_cash + captured_card + captured_credit + approved_exceptions
    >= delivered_gross

cash_minor >= 0
card_minor >= 0
cash_minor + card_minor == outstanding_due_minor

sum(payout_components for one currency)
    == gross_captured - refunds - fees
```

No amount may cross a domain boundary without an explicit currency. No payout batch may contain multiple currencies.

## 2. Existing Code to Adapt

| Concern | Current owner | Required adaptation |
|---|---|---|
| Split-payment route | `order/driver_edges.go` `HandleSplitPayment` | Replace event-only behavior with transactional payment-leg persistence. |
| Driver route/auth | `driverroutes/routes.go` | Keep the route; preserve `DRIVER` role and claim-bound subject. |
| Order payment legs | `order/settlement_hardening.go` | Reuse `PaymentLeg`, `RecordPaymentLeg`, and captured-payment queries. |
| Delivery total | `order/settlement_hardening.go` | Reuse `deliveredGrossFromOrder` and transaction-scoped reads. |
| Order state machine | `order/state_machine.go` and `order/service.go` | Accept split collection only in payment-eligible states. |
| Idempotency | `order/driver_edges.go` and `order/idempotency_guard.go` | Keep body-hash replay semantics; bind the key to the split operation. |
| Durable events | `events`, `outbox`, `kafka/notification_dispatcher.go` | Keep `SPLIT_PAYMENT_CREATED` as a notification-compatible event, but emit it only after durable legs exist. |
| Supplier payout batch | `payout/payout.go`, `payout/store.go` | Make currency a batch dimension and resolve fee snapshots deterministically. |
| Fee schedule | `internal/services/billing/fees.go` | Validate fee bounds and calculate commission in the same currency as the batch. |
| Ledger authority | `payment/repository_spanner.go`, `PaymentLedgerEntries` | Add explicit references to payment legs and settlement components. |
| Clients | Driver Android/iOS, supplier portal/native | Preserve request shape additively; consume durable result and payment state. |

Do not create a second payment state machine, second ledger, or second chargeback product.

## 3. Phase A - Correct Driver Split Payment

### 3.1 Business behavior

The driver may record a single delivery payment composed of cash and card portions. The operation closes only the remaining amount due for the delivered goods. A split is not a new order, a new invoice, or a notification-only fact.

### 3.2 Request contract

Keep the existing request fields for deployed clients:

```json
{
  "order_id": "order-id",
  "cash_minor": 7000,
  "card_minor": 3000,
  "currency": "UZS"
}
```

Additive response fields should include:

```json
{
  "status": "split_recorded",
  "order_id": "order-id",
  "cash_leg_id": "leg-id-or-empty",
  "card_leg_id": "leg-id-or-empty",
  "captured_minor": 10000,
  "remaining_due_minor": 0,
  "currency": "UZS",
  "payment_state": "CAPTURED"
}
```

The response must never claim `CAPTURED` until the corresponding durable legs are committed.

### 3.3 Transaction algorithm

Implement a service method owned by `order`, called from `HandleSplitPayment`:

1. Resolve claims and require `RoleDriver`.
2. Read the request body with a bounded size and require `Idempotency-Key` through the existing guard.
3. Normalize `order_id` and currency. Reject empty order ID.
4. Open one `spanner.ReadWriteTransaction`.
5. Read the order inside the transaction using `GetOrderTxn`.
6. Require `order.DriverID == claims.Subject`. A missing assignment must fail closed; do not let any driver collect an unassigned order.
7. Require status in `ARRIVED`, `AWAITING_PAYMENT`, `PENDING_CASH_COLLECTION`, or the explicitly supported credit-settlement state.
8. Resolve the authoritative order currency. If the request currency is empty, use the order currency. If supplied and different, return `currency_mismatch`.
9. Compute delivered gross using `deliveredGrossFromOrder` inside the transaction.
10. Sum already captured legs and settlement exceptions inside the same transaction.
11. Compute `outstanding_due_minor = max(0, delivered_gross - captured - exceptions)` with checked arithmetic.
12. Require `cash_minor >= 0` and `card_minor >= 0`.
13. Use overflow-safe addition and require `cash_minor + card_minor == outstanding_due_minor`. Reject both underpayment and overpayment.
14. If `cash_minor > 0`, insert a `PaymentLeg{MethodCash, StatusCaptured}` with a deterministic idempotency key derived from the request key and order ID.
15. If `card_minor > 0`, do not mark the card leg captured until the provider confirms capture. Use the existing card execution path, or persist `PENDING` and complete it through the provider settlement flow.
16. For a provider-confirmed card leg, persist `CAPTURED`, provider reference, and capture timestamp in the same finalization transaction.
17. Update order/payment state only through the existing order transition helpers. Do not bypass `FISCALIZING` or the fiscal hard gate.
18. Emit `SPLIT_PAYMENT_CREATED` through transactional outbox in the same transaction, including leg IDs, amounts, currency, order ID, supplier ID, retailer ID, and driver ID.
19. Emit the existing best-effort notification path after commit for immediate UI refresh.
20. Invalidate order, payment, retailer tracking, supplier finance, and driver payment caches after commit.
21. Return the committed payment-leg IDs and authoritative totals.

### 3.4 Card execution rule

Cash and card are not symmetric operationally:

- Cash is captured when the driver confirms received cash and passes proximity/state checks.
- Card is captured only after the selected provider confirms the capture.
- A card authorization or redirect is not a captured payment.
- If card capture fails after cash is committed, the transaction must leave a truthful partial state: cash captured, card pending/failed, remaining due, fiscal blocked, and a reconciliation/collection exception.
- The system must never mark the whole order paid merely because the requested split sums to the order total.

### 3.5 Required edge cases

- Negative cash or card amount.
- `cash_minor + card_minor` integer overflow.
- Sum below outstanding due.
- Sum above outstanding due.
- Currency omitted, matching, or mismatching.
- Wrong driver for the order.
- Order has no assigned driver.
- Order is `PENDING`, `LOADED`, `IN_TRANSIT`, `COMPLETED`, or `CANCELLED`.
- Existing captured cash/card legs already cover the order.
- Duplicate request with the same body and idempotency key.
- Same idempotency key with a different body.
- Concurrent cash collection and split collection.
- Concurrent split requests with different idempotency keys.
- Card provider timeout after the provider may have captured the payment.
- Fiscalization timeout after all payment legs are captured.
- Partial offload where delivered gross is less than original order total.

## 4. Phase B - Payment Ledger and Event Contract

### 4.1 Durable facts

`OrderPaymentLegs` remains the order-level payment fact. `PaymentLedgerEntries` remains the provider/payment authority read model. They must be linked rather than competing.

Add additive identifiers where absent:

- `PaymentLegId` or `ReferenceId` on payment ledger entries.
- `OrderId`, `SupplierId`, `RetailerId`, `Currency`, and `Method` on every split event.
- `ProviderReference` for card legs when available.
- `IdempotencyKey` for each logical leg.

Do not change historical rows in place. New fields must be nullable or additive for deployed data.

### 4.2 Event rules

The canonical event producer remains `events.EventSplitPaymentCreated`. Update all source-of-truth locations together:

1. `apps/backend-go/events/events.go`.
2. `apps/backend-go/events/types.go`.
3. `contracts/events.schema.json`.
4. `packages/types/ws-events.ts`.
5. Driver Android generated/typed event models.
6. Driver iOS generated/typed event models.
7. `kafka/notification_dispatcher.go` and formatter behavior.

The event is not the ledger. Consumers must never infer captured money from a notification without reading the durable payment authority.

### 4.3 Client behavior

Driver Android and iOS should:

- show `cash captured`, `card pending`, `card captured`, `remaining due`, and `fiscalizing` separately;
- retry with the same idempotency key after reconnect;
- never locally mark payment complete based only on a successful HTTP request if the response says card pending;
- reconcile the payment-leg projection on app resume;
- show stale/offline state when payment authority cannot be refreshed.

Retailer clients should receive the durable payment update through the existing retailer room and refresh pending payment/tracking views. Supplier clients should refresh finance and reconciliation views. No client should calculate supplier payout from the split event.

## 5. Phase C - Currency-Safe Supplier Payout Batches

### 5.1 Current defect

`payout.Repository.SumLegs` reads `Method`, `AmountMinor`, and `Orders.Currency`, stores the first currency, and sums every amount together. This is invalid for a supplier with more than one currency in the period.

### 5.2 Correct model

The smallest safe correction is one payout batch per:

```text
SupplierId + PeriodStart + PeriodEnd + Currency
```

The idempotency key and unique index must include the normalized currency. A batch with mixed currencies must never be rendered or submitted.

### 5.3 Repository algorithm

Replace the scalar `SumLegs` result with either:

```text
[]PayoutCurrencySummary {
  currency,
  gross_captured_minor,
  refunded_minor,
  leg_count
}
```

or require the caller to pass a currency filter and return a conflict if more than one currency is found.

Preferred query shape:

```sql
SELECT l.Method, SUM(l.AmountMinor), o.Currency, COUNT(1)
FROM OrderPaymentLegs l
JOIN Orders o ON o.OrderId = l.OrderId
WHERE o.SupplierId = @supplier_id
  AND l.Status = 'CAPTURED'
  AND l.CapturedAt >= @period_start
  AND l.CapturedAt < @period_end
GROUP BY l.Method, o.Currency
```

The query must use declared supplier/order/capture indexes or be backed by a migration before use at scale.

### 5.4 Batch algorithm

For each currency summary:

1. Resolve the shipped MarketPack currency and verify it matches the historical order currency.
2. Resolve the fee schedule effective for the payout period, not merely the current schedule.
3. Validate `0 <= gmv_bps <= 10000`, non-negative fixed fees, and currency equality.
4. Compute commission with checked multiplication and integer division.
5. Compute `net = gross - refunds - commission` with checked subtraction.
6. Reject negative net and zero-payable batches honestly.
7. Persist a currency-specific draft batch.
8. Render a bank file with the same currency and amount authority.
9. Mark exported/submitted/paid through an idempotent state transition with audit and outbox event.

### 5.5 Concurrent generation

Keep the database unique constraint as the final guard, but make replay behavior deterministic:

1. Attempt the insert with the unique `(supplier, period, currency)` identity.
2. If the insert returns an already-exists conflict, read and return the existing batch.
3. Verify the caller-supplied idempotency key matches the existing request; otherwise return an idempotency conflict.
4. Never create two batches and never make the caller manually retry after a harmless race.

## 6. Phase D - Immutable Settlement Slices

The current `SupplierPayoutPolicies` table describes policy but does not snapshot the evaluated split for each order. Add settlement slices only if the business requires supplier-level and warehouse-local payout ownership.

### 6.1 Model

An immutable `InvoiceSettlementSlice` should contain:

```text
SliceId
InvoiceId
OrderId
SupplierId
WarehouseId
PayoutOwnerType
PayoutOwnerId
GrossMinor
FeePolicyVersion
FeeBps
FeeMinor
NetPayoutMinor
Currency
SettlementTarget
CreatedAt
```

Primary and uniqueness identity must prevent duplicate slices for the same invoice/order/supplier component.

### 6.2 Creation logic

1. During unified checkout, group lines by supplier and resolved settlement target.
2. Resolve policy and currency from the shipped MarketPack and supplier policy.
3. Calculate each component in integer minor units.
4. Allocate the rounding remainder deterministically to the platform fee component, never silently to the supplier or customer.
5. Write invoice, order, slice rows, reservation mutations, and `ORDER_CREATED` outbox records in one transaction.
6. Never recompute historical fee or owner values from current policy during refund or payout.

### 6.3 Refund and chargeback logic

Refunds create reversal slices/ledger rows using the original slice ratios and payout owners. A current policy change must not change the historical refund split.

Chargebacks debit the original payout owner(s), with a deterministic rounding remainder and currency check. If the original slice is missing, fail into reconciliation review instead of guessing.

## 7. Phase E - Client and Operations Wiring

### Supplier portal, Android, and iOS

- Payment authority view shows gross, refunds, fees, net, currency, owner, source, and stale state.
- Split-payment events refresh finance but never replace ledger reads.
- Payout batches are grouped by currency and period.
- Mixed-currency data is shown as separate batches or an explicit reconciliation error.

### Warehouse portal, Android, and iOS

- Warehouse-local settlement ownership is shown only when an immutable slice exists.
- Warehouse finance never derives payout from order count or a notification event.
- Missing authority returns `available: false` or a reconciliation exception, not zero.

### Retailer desktop, Android, and iOS

- Payment timeline distinguishes authorization, captured cash, captured card, refund, chargeback, and fiscal receipt.
- Historical currency is used for receipts, claims, refunds, and disputes.
- Pending payment remains visible after an offline or provider timeout.

### Driver Android and iOS

- Split collection UI sends exact minor amounts and currency.
- Offline retry preserves the original idempotency key and body.
- Card provider pending state blocks false completion.

## 8. Test Matrix

### Split-payment unit tests

- positive exact cash/card split;
- cash-only and card-only compatibility;
- negative leg rejected;
- underpayment rejected;
- overpayment rejected;
- overflow rejected;
- currency mismatch rejected;
- wrong driver rejected;
- unassigned order rejected;
- invalid order state rejected;
- partial-offload delivered total used;
- duplicate same request replays the original response;
- same key with changed body returns conflict.

### Transaction and integration tests

- both legs and outbox commit atomically;
- failed outbox prevents payment-leg commit;
- concurrent split requests produce one durable outcome;
- concurrent cash and split collection cannot over-collect;
- card timeout leaves pending/failed state and no false fiscal completion;
- refund caps against captured leg balances;
- ledger references exactly one payment-leg fact.

### Payout tests

- one currency produces one batch;
- two currencies produce two batches;
- mixed-currency batch cannot be exported;
- fee schedule currency mismatch fails closed;
- invalid negative or over-10000 bps schedule is rejected;
- rounding remainder preserves exact total;
- concurrent batch generation returns the same existing batch;
- refunds are counted once;
- historical policy version is used for a historical slice.

### Required commands

```text
go test ./order ./payment ./payout -count=1
go test ./order -run 'SplitPayment|MoneyPath|Refund' -count=1
go test ./payment -run 'Ledger|Webhook|Settlement|Currency' -count=1
go test ./payout -run 'Payout|Currency|Idempotency' -count=1
```

Spanner-emulator coverage is required for transaction, unique-index, ledger, and payout batch tests. Role-row E2E must cover driver, retailer, supplier, and warehouse clients before rollout.

## 9. Rollout and Safety Gates

1. Add read-only payment-leg and payout-currency diagnostics first.
2. Backfill or quarantine historical mixed-currency payout periods; never silently relabel them.
3. Enable corrected split collection behind a per-cell/per-supplier feature flag.
4. Keep old clients compatible with additive response fields.
5. Observe duplicate attempts, amount mismatches, provider timeouts, fiscal aging, ledger imbalance, mixed currency, and payout conflicts.
6. Require zero unexplained ledger imbalance and successful replay/concurrency drills before widening rollout.
7. Keep live PSP and fiscal credentials as separate operational gates. Keys do not replace missing business logic.

## 10. Completion Definition

This payment work is complete only when:

- split cash/card amounts are durable payment legs;
- order ownership, status, currency, exact due, and overflow are enforced;
- payment legs, ledger entries, fiscal state, and notifications agree;
- payout batches cannot mix currencies;
- fee and payout ownership are immutable for historical orders;
- duplicate, concurrent, offline, webhook, refund, and chargeback paths are replay-safe;
- supplier, warehouse, retailer, driver, and platform read models show the same durable authority;
- all focused tests, Spanner-emulator tests, role-row E2E tests, `go build ./...`, and `go vet ./...` pass.
