# Handoff Report: Track 7 Payments, PSP, Escrow, Invoicing & Financial Integrity Audit

## 1. Observation
Direct observations in the codebase with exact file and line references:
- **`payment/service.go:645-648`**: In `HandleChargebackReversal`, reversal record is created with:
  `ReversalRecord{ ReversalID: s.newID("reversal"), SessionID: strings.TrimSpace(req.SessionID), SupplierID: s.resolveSupplierID(r.Context()), Gateway: executionResult.ResolvedGateway, AmountMinor: 0, Currency: s.currency, CreatedAt: now }`
  When stored via `SaveReversal` -> `buildReversalLedgerEntry` (`repository_spanner.go:607`), `PaymentLedgerEntries` receives `AmountMinor = 0`.
- **`ar/treasury_hub.go:189-196`**: In `WriteOffInvoice`, `inv.BalanceMinor` is set to `0` in memory on line 189, and then on line 194 `s.repo.ApplyCreditNote(ctx, invoiceID, inv.BalanceMinor, idemKey)` is invoked with `0`, leaving `ArInvoices` in Spanner with its original non-zero balance and `Status = OPEN`.
- **`payout/payout.go:136-179` & `payout/store.go:48-65`**: In `GenerateBatch`, summation of payable slices is computed via `SumLegsByCurrency` in an initial read query, and then a separate `Insert` transaction runs `UPDATE InvoiceSettlementSlices SET Status = 'BATCHED' WHERE ... Status = 'UNSETTLED'`. Any slice inserted between the read query and the update is marked `BATCHED` but excluded from the batch's `NetPayoutMinor`.
- **`payment/webhook_inbox.go:101-108`**: In `WebhookInboxStore.ProcessPending`, `row.Columns(&webhookID, &gateway, &recordJSON, &source, &attempts)` attempts to decode a Spanner `JSON NOT NULL` column into `var recordJSON []byte`, which triggers a runtime decoding error in Google Cloud Spanner Go SDK on every row scan.
- **`cashrecon/expected_cash.go:49-50`**: In `ComputeExpectedCashMinor`, driver shift cash is summed using `AND o.CreatedAt >= @start AND o.CreatedAt < @end`, filtering on order placement timestamp instead of cash capture timestamp (`pl.CapturedAt`).
- **`order/settlement_hardening.go:256` & `order/refunds.go:228-236`**: Refund legs created without `CapturedAt` result in `InvoiceSettlementSlices` with zero timestamp (`0001-01-01 00:00:00 UTC`), permanently excluding them from standard payout period queries.
- **`creditnote/service.go:144-146`**: Line calculation uses `(base.LineNetMinor / base.Qty) * qty` with integer division before multiplication.
- **`webhookroutes/routes.go:30-31`**: Payme and Click webhook routes are commented out.
- **`payment/payme_merchant.go:23-34`**: `paymeMerchantMemory` is backed by in-memory Go maps without persistent Spanner storage.

## 2. Logic Chain
1. **Ledger Invariance Breakdown**: Because `HandleChargebackReversal` creates reversals with `AmountMinor: 0` (`payment/service.go:645`), reconciliation calculations (`SignedSettlementEntryAmount`) multiply 0 by +1, producing 0 net credit. The chargeback debit is never cancelled in the financial ledger, corrupting supplier settlement summaries.
2. **Unapplied Write-Offs**: Because `inv.BalanceMinor` is zeroed before calling `ApplyCreditNote` (`ar/treasury_hub.go:189-194`), the deduction applied to Spanner is 0. The invoice remains `OPEN` with full balance in the database while the API caller receives a misleading 200 response with an in-memory zeroed object.
3. **Escrow / Payout Leakage**: Because `payout.GenerateBatch` and `payout.Insert` split slice calculation and slice status locking across separate transactions (`payout/payout.go:136-179`), concurrent order settlement insertions cause slices to be marked `BATCHED` without their monetary value being added to the payout batch.
4. **Webhook Retry Poisoning**: Because `JSON` columns in Spanner Go SDK require `string` or `spanner.NullJSON` targets, scanning into `[]byte` in `WebhookInboxStore.ProcessPending` fails unconditionally on all rows, preventing asynchronous webhook retries.
5. **Driver Cash False Shortfalls**: Because `ComputeExpectedCashMinor` filters on `o.CreatedAt`, any order created on a prior day but fulfilled today is omitted from expected cash calculations, generating false discrepancy alerts during shift close.

## 3. Caveats
- No live cloud credentials or external PSP API endpoints (Stripe/Global Pay/Soliq) were invoked; all findings are derived from static source code and unit test fixtures.
- Local payment gateways (Payme/Click) are intentionally unmounted in `webhookroutes/routes.go` as part of the Layer B launch strategy (Cash + Global Pay + MySoliq + bank-file).
- The analysis assumes Google Cloud Spanner standard Go client semantics for type scanning and transaction isolation.

## 4. Conclusion
Track 7 exhibits strong multi-tenant structure and integer minor-unit discipline, but contains several critical logic defects:
1. Reversal ledger entries with 0 amount in `payment/service.go`.
2. Ineffective write-off logic in `ar/treasury_hub.go`.
3. Non-atomic payout batching in `payout/payout.go`.
4. Spanner `JSON` column decoding failure in `payment/webhook_inbox.go`.
5. Shift cash date filtering flaw in `cashrecon/expected_cash.go`.
6. Zero timestamp refund settlement slices in `order/settlement_hardening.go`.

These defects must be resolved before proceeding with live financial transactions or production billing.

## 5. Verification Method
1. **Unit & Package Tests**: Run `go test -v ./payment/... ./ar/... ./credit/... ./creditnote/... ./cashrecon/... ./payout/... ./tax/... ./fiscal/... ./fxrates/...` inside `pegasusX/apps/backend-go`.
2. **Code Inspection**:
   - Inspect `pegasusX/apps/backend-go/payment/service.go:645-648` to verify reversal amount.
   - Inspect `pegasusX/apps/backend-go/ar/treasury_hub.go:189-196` to verify write-off balance assignment.
   - Inspect `pegasusX/apps/backend-go/payout/store.go:48-65` to verify slice update query.
   - Inspect `pegasusX/apps/backend-go/payment/webhook_inbox.go:101-108` to verify `RecordJson` variable type.
   - Inspect `pegasusX/apps/backend-go/cashrecon/expected_cash.go:49-50` to verify shift date join condition.
3. **SSMR Smokecheck**: Run `cmd/ssmr-smokecheck` to verify end-to-end multi-role financial flows against Spanner emulator.
