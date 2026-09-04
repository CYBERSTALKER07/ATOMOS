# Track 7 Comprehensive Audit: Payments, PSP, Escrow, Invoicing & Financial Integrity

**Audited Subsystems:**
- `pegasusX/apps/backend-go/payment` & `paymentroutes` (Payment sessions, PSP execution, webhooks, idempotency, ledger)
- `pegasusX/apps/backend-go/ar` & `creditroutes` (Accounts Receivable, Invoices, Dunning, Treasury Hub, Write-offs)
- `pegasusX/apps/backend-go/credit` (Retailer credit profiles, limits, reservations, CAS balance adjustments, risk scoring)
- `pegasusX/apps/backend-go/creditnote` & `creditnoteroutes` (Credit notes, VAT calculations, line items, reverse logistics)
- `pegasusX/apps/backend-go/cashrecon` & `cashreconroutes` (Driver cash reconciliation, shift expected cash, discrepancies)
- `pegasusX/apps/backend-go/payout` (Supplier settlement batches, commission deduction, bank file export, live rail interface)
- `pegasusX/apps/backend-go/tax`, `fiscal`, `soliq` (Tax regimes, VAT calculation, E-IMZO PKCS#12 signing, Soliq client)
- `pegasusX/apps/backend-go/fxrates` (Integer scaled currency conversion, cross-currency validation)
- `pegasusX/apps/backend-go/schema/spanner.ddl` (Financial entity schemas, indexes, constraints)

---

## 1. Executive Summary

Track 7 implements the multi-tenant financial foundation of PegasusX, enforcing integer minor units (e.g. UZS tiyin, USD cents), immutable payment ledgering, pay-at-delivery cash/card collection, credit leave-behind with AR open items, and bank-file supplier payouts.

While the architecture demonstrates strong patterns (atomic Spanner RW transactions with transactional outbox event buffering, strict tenant isolation via `SupplierId`, and fail-closed currency/pack validation), this line-by-line review surfaced several **critical vulnerabilities, financial integrity leaks, and state machine race conditions** that require immediate remediation before live deployment.

---

## 2. Ranked Findings & Code Review

### Finding T7-01: Reversal Ledger Entry Injected with Zero Amount and Defaulted Currency
- **File & Lines:** `pegasusX/apps/backend-go/payment/service.go:645-648`, `pegasusX/apps/backend-go/payment/repository_spanner.go:601-613`
- **Severity:** CRITICAL (Financial Ledger Integrity / Bookkeeping Invariance)
- **Description:**
  When a chargeback reversal is processed in `HandleChargebackReversal`, the `ReversalRecord` is initialized with hardcoded `AmountMinor: 0` and `Currency: s.currency` (`service.go:645-646`), completely ignoring the original session's `AmountMinor` and `Currency`. When `SaveReversal` commits the transaction, `buildReversalLedgerEntry` writes a row into `PaymentLedgerEntries` with `AmountMinor = 0` and `EntryType = CHARGEBACK_REVERSAL_RECORDED`.
  Because `SignedSettlementEntryAmount` computes `0 * 1 = 0`, the reversal completely fails to restore the supplier's settlement balance in reconciliation and financial reports (`HandleSettlementAuthority` and `HandleReconciliationMismatches`).
- **Blast Radius:**
  - `PaymentLedgerEntries` table contains corrupted 0-value reversal rows.
  - Supplier reconciliation queries will show persistent false-positive debits for chargebacks that were legitimately reversed.
  - Supplier payout calculations and settlement authority summaries will underpay suppliers.
- **Recommendation:**
  Retrieve the original chargeback or session amount and currency from `s.repo.GetSession` / `s.repo.HasChargebackForOrder` and populate `AmountMinor: session.AmountMinor` and `Currency: session.Currency` in `ReversalRecord`.

---

### Finding T7-02: Silent Mutation Failure in Invoice Write-Off
- **File & Lines:** `pegasusX/apps/backend-go/ar/treasury_hub.go:189-196`
- **Severity:** CRITICAL (Accounts Receivable / Balance Inconsistency)
- **Description:**
  In `Service.WriteOffInvoice`, the invoice balance is mutated in memory to zero on line 189:
  ```go
  inv.Status = StatusVoid
  inv.BalanceMinor = 0
  inv.UpdatedAt = s.now().UTC()
  ```
  Immediately afterwards, on line 194, it attempts to clear the database balance by calling:
  ```go
  if err := s.repo.ApplyCreditNote(ctx, invoiceID, inv.BalanceMinor, idemKey); err != nil
  ```
  Because `inv.BalanceMinor` was just set to `0`, `ApplyCreditNote` is called with `amountMinor = 0`. In `applyPaymentInTxn` (`ar/service.go:756`), `newBal := bal - 0 = bal`, so the invoice balance in Spanner remains untouched, its status remains `OPEN`, and the returned `&inv` object falsely indicates to the caller that the invoice was voided and zeroed out.
- **Blast Radius:**
  - Invoices intended to be written off remain `OPEN` with full balances in Spanner.
  - Dunning workers will continue dunning retailers for written-off debt.
  - Retailers will be wrongfully placed on `CREDIT_HOLD` / delinquency lock for uncollectible debts that managers already approved for write-off.
- **Recommendation:**
  Capture the existing balance before zeroing the struct (`origBalance := inv.BalanceMinor`), pass `origBalance` to `ApplyCreditNote`, and explicitly execute a transaction that updates `Status = StatusVoid` in `ArInvoices`.

---

### Finding T7-03: Payout Batch Settlement Slice Money Leak Race Condition
- **File & Lines:** `pegasusX/apps/backend-go/payout/payout.go:136-179`, `pegasusX/apps/backend-go/payout/store.go:48-65`
- **Severity:** CRITICAL (Supplier Payouts / Escrow Underpayment)
- **Description:**
  In `Service.GenerateBatch`, payout calculation occurs across non-atomic steps:
  1. `SumLegsByCurrency` runs a read-only Single query over `InvoiceSettlementSlices` where `Status = 'UNSETTLED' AND CreatedAt >= @start AND CreatedAt < @end` to compute `sum.Captured`, `sum.Commission`, and `sum.NetPayout`.
  2. For each currency, `s.repo.Insert(ctx, b)` opens a new ReadWriteTransaction which inserts `PayoutBatches` and runs an unqualified UPDATE:
     ```sql
     UPDATE InvoiceSettlementSlices
     SET Status = 'BATCHED', PayoutBatchId = @b
     WHERE SupplierId = @s AND Currency = @c AND Status = 'UNSETTLED'
       AND CreatedAt >= @start AND CreatedAt < @end
     ```
  If a new order settlement slice is created between step 1 and step 2, the SQL `UPDATE` in step 2 will mark the new slice as `BATCHED` with `PayoutBatchId = b.BatchID`. However, its money amount was NOT included in `b.GrossCapturedMinor` or `b.NetPayoutMinor` calculated in step 1.
  Because the slice is now marked `BATCHED`, it will never be included in any future payout batch, resulting in permanent, untracked underpayment of the supplier.
- **Blast Radius:**
  - Suppliers lose payout funds on orders captured concurrently during batch generation.
  - Reconciliation between `PayoutBatches.NetPayoutMinor` and `SUM(InvoiceSettlementSlices.NetPayoutMinor WHERE PayoutBatchId = batch_id)` will permanently fail.
- **Recommendation:**
  Perform the summation and the status update in the **exact same Spanner ReadWriteTransaction**, querying the slices inside `txn.Query` and locking the slice IDs directly (`WHERE SliceId IN UNNEST(@slice_ids)`).

---

### Finding T7-04: Permanent Webhook Inbox Deserialization Scan Failure
- **File & Lines:** `pegasusX/apps/backend-go/payment/webhook_inbox.go:101-108`
- **Severity:** HIGH (Payment Webhook Reliability / Fault Tolerance)
- **Description:**
  In `WebhookInboxStore.ProcessPending`, the query retrieves pending webhook records from `WebhookInbox`:
  ```go
  var webhookID, gateway, source string
  var recordJSON []byte
  var attempts int64
  if err := row.Columns(&webhookID, &gateway, &recordJSON, &source, &attempts); err != nil {
      continue
  }
  ```
  In `schema/spanner.ddl:642`, `RecordJson` is defined as `JSON NOT NULL`. In the Cloud Spanner Go SDK, `JSON` columns cannot be scanned into a raw `[]byte` slice (which is reserved for `BYTES` columns); scanning `JSON` into `[]byte` returns `spanner: cannot decode JSON into []byte`.
  This causes `row.Columns` to error on every row, triggering `continue`. As a result, `ProcessPending` can NEVER deserialize or retry any failed webhooks, permanently breaking the webhook retry mechanism.
- **Blast Radius:**
  - All webhooks that encountered temporary database contention or network errors during initial receipt are stranded in `WebhookInbox` forever.
  - Orders awaiting webhook clearance will remain stuck in `AWAITING_PAYMENT` or `FISCALIZING`.
- **Recommendation:**
  Change `var recordJSON []byte` to `var recordJSON string` or `var recordJSON spanner.NullJSON` (or `var recordJSON spanner.NullString`), and parse `[]byte(recordJSON)`.

---

### Finding T7-05: Zero Timestamp on Settlement Slices Created from Uncaptured Legs
- **File & Lines:** `pegasusX/apps/backend-go/order/settlement_hardening.go:256`, `pegasusX/apps/backend-go/order/refunds.go:228-236`
- **Severity:** HIGH (Financial Settlement / Payout Period Exclusion)
- **Description:**
  When a refund is recorded in `order/refunds.go:228-236`, the created `PaymentLeg` has `CapturedAt: spanner.NullTime{Valid: false}` (unpopulated).
  When `RecordPaymentLeg` processes this leg in `settlement_hardening.go:256`:
  ```go
  payout.GenerateSettlementSlice(ctx, s.commissionResolver, s.newID(), leg.OrderID, supplierID, leg.LegID, amount, currency, leg.CapturedAt.Time)
  ```
  `leg.CapturedAt.Time` evaluates to Go's zero `time.Time` (`0001-01-01 00:00:00 UTC`). The resulting `InvoiceSettlementSlices` record is written to Spanner with `CreatedAt = 0001-01-01`.
  When `payout.SumLegsByCurrency` queries slices for any standard billing period `[start, end)` (e.g. `2026-08-01` to `2026-08-31`), the slice is filtered out because `0001-01-01 < start`. The refund clawback is never applied against supplier payout batches.
- **Blast Radius:**
  - Refund deductions and clawbacks fail to be deducted from supplier payouts.
  - Platform suffers monetary loss by overpaying suppliers for refunded orders.
- **Recommendation:**
  In `settlement_hardening.go`, if `!leg.CapturedAt.Valid || leg.CapturedAt.Time.IsZero()`, fall back to `leg.CreatedAt` or `s.now()`.

---

### Finding T7-06: Driver Cash Shift Filtered by Order Creation Time Instead of Collection Time
- **File & Lines:** `pegasusX/apps/backend-go/cashrecon/expected_cash.go:49-50`
- **Severity:** HIGH (Driver Operations / Cash Reconciliation Integrity)
- **Description:**
  In `ComputeExpectedCashMinor`, expected cash for a driver's shift is calculated via:
  ```sql
  SELECT COALESCE(SUM(pl.AmountMinor), 0)
  FROM OrderPaymentLegs pl
  JOIN Orders o ON o.OrderId = pl.OrderId
  WHERE o.DriverId = @driverId
    AND pl.Method = @method
    AND pl.Status = @status
    AND o.CreatedAt >= @start
    AND o.CreatedAt < @end
  ```
  The SQL query filters by `o.CreatedAt` (when the customer placed the order), NOT by `pl.CapturedAt` or `pl.CreatedAt` (when the driver delivered the order and collected cash).
  Orders placed yesterday evening and delivered this morning will be omitted from today's shift expected cash, falsely marking the driver as having an unexplained cash surplus. Conversely, orders placed today for delivery tomorrow will be included in today's expected cash before the driver has even delivered them, falsely accusing the driver of cash theft/shortfall.
- **Blast Radius:**
  - False discrepancy flags generated on driver shift close.
  - Unnecessary cash reconciliation disputes (`CashReconciliations.DifferenceMinor != 0`) escalating to finance teams.
- **Recommendation:**
  Change the filter from `o.CreatedAt >= @start AND o.CreatedAt < @end` to `pl.CapturedAt >= @start AND pl.CapturedAt < @end` (or `pl.CreatedAt`).

---

### Finding T7-07: Unchecked Version Overwrite in AR Invoice Aging Recomputation
- **File & Lines:** `pegasusX/apps/backend-go/ar/service.go:876-947`
- **Severity:** HIGH (Concurrency / Optimistic Locking Violation)
- **Description:**
  In `SpannerRepository.RecomputeAging`, the worker runs a single query to load open invoices with their current `Version` (`rec.ver`). Then, for each invoice needing an aging bucket transition, it opens a separate ReadWriteTransaction:
  ```go
  spanner.UpdateMap("ArInvoices", map[string]any{
      "InvoiceId":   rec.id,
      "AgingBucket": rec.bucket,
      "Version":     rec.ver + 1,
      "UpdatedAt":   spanner.CommitTimestamp,
  })
  ```
  Inside the ReadWriteTransaction, it does NOT read `ArInvoices` to check if `Version == rec.ver`. If a payment arrived concurrently (e.g. `ApplyPayment` updated `BalanceMinor` to 0, `Status` to `PAID`, and `Version` to `rec.ver + 1`), `RecomputeAging` blindly overwrites `Version` with `rec.ver + 1` (reverting the version counter) and emits `EventARInvoiceAgingUpdated` for a paid invoice.
- **Blast Radius:**
  - Optimistic locking version sequence in `ArInvoices` gets corrupted.
  - Downstream event consumers receive misleading aging update events for already settled invoices.
- **Recommendation:**
  Inside the ReadWriteTransaction, read `ArInvoices.Version` and `ArInvoices.Status`. If `Status` is no longer `OPEN`/`PARTIAL` or `Version != rec.ver`, abort or re-evaluate the transition.

---

### Finding T7-08: Premature Integer Division in Manual Credit Note Calculation
- **File & Lines:** `pegasusX/apps/backend-go/creditnote/service.go:144-146`
- **Severity:** MEDIUM (Rounding & Tax Precision)
- **Description:**
  In `Service.CreateManual`, line item amounts are calculated as:
  ```go
  line.LineNetMinor = (base.LineNetMinor / base.Qty) * qty
  line.LineVatMinor = (base.LineVatMinor / base.Qty) * qty
  line.LineGrossMinor = (base.LineGrossMinor / base.Qty) * qty
  ```
  Because integer division precedes multiplication (`base.LineNetMinor / base.Qty`), non-divisible numbers suffer truncation error. For example, for 3 items with total net 10,000 minor units, returning 2 items calculates `(10000 / 3) * 2 = 3333 * 2 = 6666`, losing 1 minor unit. Furthermore, `LineNetMinor + LineVatMinor` can drift from `LineGrossMinor`.
- **Blast Radius:**
  - Credit note totals drift by minor units from original invoice proportions.
  - Tax audit discrepancies between Soliq OFD declarations and credit note ledger amounts.
- **Recommendation:**
  Multiply before dividing: `line.LineNetMinor = (base.LineNetMinor * qty) / base.Qty`, and enforce `LineGrossMinor = LineNetMinor + LineVatMinor`.

---

### Finding T7-09: Multi-Currency FX Scaling Omits Differing Decimal Exponents
- **File & Lines:** `pegasusX/apps/backend-go/fxrates/convert.go:114-146`
- **Severity:** MEDIUM (Multi-Currency Conversion)
- **Description:**
  `applyRate` converts integer minor units using `amount * rateScaled / scale` directly without adjusting for decimal exponents between base and quote currencies.
  While this functions correctly when both currencies have 2 decimals (e.g. USD cents ↔ UZS tiyin), converting between 2-decimal currencies and 0-decimal currencies (e.g. JPY, KRW, VND) or 3-decimal currencies (e.g. KWD, BHD, OMR) results in values that are miscalculated by factors of 100 or 1,000.
- **Blast Radius:**
  - International multi-currency pricing or settlement will produce severely inaccurate conversions for zero-decimal or three-decimal currencies.
- **Recommendation:**
  Incorporate ISO-4217 currency exponent lookup (`math.Pow10(expTo - expFrom)`) into `applyRate`.

---

### Finding T7-10: Adyen Webhook Acknowledgment Signature & Batch Handling
- **File & Lines:** `pegasusX/apps/backend-go/payment/adyen_webhook.go:42-59`, `186-192`
- **Severity:** MEDIUM (PSP Integration Protocol Compliance)
- **Description:**
  1. In `HandleAdyenWebhook`, the response sent back to Adyen is `json.Marshal(map[string]any{"status": "accepted", "gateway": "adyen", "processed_items": processed})`. Standard Adyen webhook endpoints require the literal string `"[accepted]"` in the HTTP response body. If Adyen's webhook system does not recognize the JSON response, it will treat the webhook delivery as failed and keep retrying.
  2. If `NotificationItems` contains multiple items in a single request and item #2 fails during database commit, an error is returned to Adyen after item #1 was already committed. When Adyen retries the entire HTTP payload, item #1 is reprocessed or re-checked via idempotency guard while item #2 is retried.
- **Blast Radius:**
  - Repeated duplicate webhook deliveries and spam in server logs.
- **Recommendation:**
  Return `w.Write([]byte("[accepted]"))` for Adyen webhook responses as per Adyen specifications.

---

### Finding T7-11: Multiple Active Checkout Sessions Permitted for Single Order
- **File & Lines:** `pegasusX/apps/backend-go/payment/retailer_checkout.go:395-464`
- **Severity:** MEDIUM (State Machine / Session Sprawl)
- **Description:**
  When `HandleOrderCardCheckout` is called repeatedly for the same order (e.g. rapid user clicks or network retries), `initCheckoutSession` checks if an existing session exists in `PAYMENT_REQUIRED`, but rather than returning the existing active session's redirect URL, it mints a brand new `SessionID` (`psess_...`) and inserts a new row into `PaymentSessions` and `PaymentAttempts`.
  In `schema/spanner.ddl:570-582`, `PaymentSessions` has `PRIMARY KEY (SessionId)`, and `OrderId` has a non-unique index `Idx_PaymentSessions_ByOrderCreated`. An order can thus accumulate multiple parallel active sessions in `PAYMENT_REQUIRED` across different gateways.
- **Blast Radius:**
  - Retailer clients may receive different checkout URLs on retries.
  - Webhooks matching by `OrderId` might update a session that differs from the one the retailer actually paid on.
- **Recommendation:**
  If an existing session exists in `PAYMENT_REQUIRED` for the same gateway and amount, reuse and return the existing session and attempt rather than generating new rows.

---

## 3. Double-Entry Ledger & Spanner Balance Invariance Audit

| Area | Current Implementation | Risk / Gap | Verification Status |
|---|---|---|---|
| **Payment Ledger** (`PaymentLedgerEntries`) | Single-entry event log with `AmountMinor` and `EntryType`. Unique index on `(Gateway, EntryType, ReferenceId)`. | Not a balanced dual-entry accounting ledger (no paired Debit/Credit accounts). Reversals currently insert `0` amounts (Finding T7-01). | ❌ Balance Invariance Violated on Reversals |
| **AR Ledger** (`ArLedgerEntries`) | Dual-entry with `OPEN` (`+amount`) and `PAYMENT` (`-amount`). Unique index on `IdempotencyKey`. | WriteOff currently passes 0 amount and fails to update database (Finding T7-02). | ❌ Write-off broken |
| **Credit Reservations** (`OrderCreditReservations`) | CAS state machine: `RESERVED` → `CONVERTED` / `RELEASED` → `CLEARED`. | Properly synchronized with `RetailerCreditProfiles.ReservedMinor` and `CurrentBalanceMinor`. | ✅ Verified Atomic in Spanner RW Txn |
| **Settlement Slices** (`InvoiceSettlementSlices`) | Immutable slices: `GrossMinor`, `CommissionMinor`, `NetPayoutMinor`. Unique per payment leg. | Batching update is detached from summation query (Finding T7-03); zero timestamp on refund legs (Finding T7-05). | ❌ Payout undercount risk |

---

## 4. Deep Architectural & Edge-Case Open Questions

1. **Gateway Timeout & Pending State Machine Ambiguity:**
   When an external PSP gateway call times out or returns HTTP 504 during `CHECKOUT_CAPTURE` (e.g. Global Pay or Payme API drops connection mid-flight), the payment session remains in `INITIATED` or `PAYMENT_REQUIRED`. Does the system have an automated background reconciler that issues a `STATUS_CHECK` before allowing a second capture or cancel, or does it risk double-charging the customer's card if the driver retries?
2. **Multi-Leg Partial Delivery & Split Tender Settlement:**
   When an order is partially delivered with split tender (e.g. 50% paid via Card at delivery, 30% paid in Cash, 20% rejected as buyer damage), how are `InvoiceSettlementSlices` and `CreditNotes` reconciled against the single parent order? If a `CASH_SHORTFALL` exception is recorded for the driver while a `CreditNote` is issued for the buyer reject, is the commission calculated on the gross delivered amount or the net collected cash?
3. **Multi-Tenant Currency Boundary Enforcement in Supplier Payouts:**
   If a regional hub manages suppliers in multiple countries (e.g. Uzbekistan UZS and Kazakhstan KZT), how does `PayoutBatches` handle cross-border tax withholding and bank routing fees? Currently, `PayoutBatches` groups by `(SupplierId, PeriodStart, PeriodEnd, Currency)`. If a supplier operates with multi-currency catalog items, can a supplier have multiple simultaneous payout batches in different currencies for the same period without colliding on bank instructions?
4. **Offline Driver Payment Collection & Retroactive Cash Reconciliation:**
   If a driver operates in an offline rural zone with zero cellular connectivity, collects cash from 10 retailers, and completes shifts offline, the driver app buffers transactions in local SQLite. When the driver syncs at the end of the shift, all 10 cash payments are submitted concurrently. How does the server prevent out-of-order race conditions where the driver's shift close `SubmitReconciliation` is processed before the individual `CollectCash` payment legs are committed to Spanner?
5. **Irrevocable Credit Leave-Behind vs Late Buyer Chargeback:**
   When an order is delivered on credit (`StatusDeliveredOnCredit`), an AR invoice is opened and credit balance is converted in Spanner. If the retailer subsequently disputes the delivery 14 days later via a chargeback (`POST /v1/payment/chargeback`), `HandleChargeback` records a chargeback against the supplier, but `ArInvoices` remains `OPEN` with full balance. Who owns the credit risk—does the platform absorb the uncollectible AR debt, or does a secondary clawback debit the supplier's payout batch twice (once via chargeback ledger and once via AR default)?

---

## 5. Summary of Recommended Immediate Actions

1. **Fix Reversal Amount & Currency:** Update `HandleChargebackReversal` in `payment/service.go:645` to populate `AmountMinor` and `Currency` from the original session rather than hardcoded 0.
2. **Fix Invoice Write-Off:** Update `WriteOffInvoice` in `ar/treasury_hub.go:189-196` to pass the true original balance to `ApplyCreditNote` and execute a database update setting `Status = StatusVoid`.
3. **Atomic Payout Batching:** Rewrite `payout.GenerateBatch` to query unsettled slices and update their status to `BATCHED` inside the **same Spanner ReadWriteTransaction**.
4. **Fix Webhook Inbox JSON Scanning:** Change `recordJSON` in `WebhookInboxStore.ProcessPending` from `[]byte` to `string` or `spanner.NullJSON`.
5. **Fix Driver Shift Cash Bounds:** Change `ComputeExpectedCashMinor` in `cashrecon/expected_cash.go` to filter on `pl.CapturedAt` rather than `o.CreatedAt`.
6. **Fix Settlement Slice Refund Timestamp:** Ensure refund payment legs populate `CapturedAt` or fall back to `CreatedAt` in `settlement_hardening.go`.
7. **Fix Integer Division in Credit Notes:** Use `(base.LineNetMinor * qty) / base.Qty` in `creditnote/service.go`.
8. **Add Currency Decimal Exponent Scaling:** Update `fxrates/convert.go` to support non-2-decimal currency exponents.
