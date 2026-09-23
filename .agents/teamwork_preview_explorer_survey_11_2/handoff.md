# Handoff Report: Backend Audit of Currency Arithmetic, Naive CRUD, and Enterprise Rigor

**To**: `teamwork_preview_orchestrator_11` (`d877571c-b5bd-4489-a1cb-441c991bb03d`)  
**From**: `teamwork_preview_explorer_survey_11_2`  
**Working Directory**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_11_2`  
**Target Codebase**: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend`  
**Mission**: Requirements R1.1 & R1.2 — Exhaustive Backend Audit against Universal Engineering Doctrine  
**Date**: 2026-09-23  

---

## 1. Observation

### 1.1 Floating-Point Currency Arithmetic & Monetary Violations
1. **`internal/soliq/efactura.go:131, 198`**:
   - Line 131:
     ```go
     lineVAT := int64(math.Round(float64(lineSubtotal*int64(it.VATPercent)) / 100.0))
     ```
   - Line 198:
     ```go
     deltaVAT := int64(math.Round(float64(deltaSum*int64(adj.VATRate)) / 100.0))
     ```
   - Observed: Float64 casting and `math.Round` on statutory 12% VAT calculations. Violates Universal Engineering Doctrine Rule 8 ("VAT Calculation: Statutory 12% VAT applied to taxable items using standard integer round-half-up math (`(price * 12 + 50) / 100`)").
2. **`internal/rebate/rebate.go:44, 123`**:
   - Line 44: `RebatePercent float64` in `ConditionContract`.
   - Line 123:
     ```go
     accrualAmount := int64(math.Round(float64(orderVolumeMinor) * (contract.RebatePercent / 100.0)))
     ```
   - Observed: Financial volume rebate accrual for GL journal entries converts tiyins to float64.
3. **`internal/consignment/consignment.go:24, 44, 94-95`**:
   - Line 24: `ConsignmentStockQty float64`
   - Line 44: `ConsumedQty float64`
   - Lines 94-95:
     ```go
     totalBaseCost := int64(qty * float64(agreement.AgreedBaseCostMinor))
     totalRetailPrice := int64(qty * float64(retailUnitPriceMinor))
     ```
   - Observed: Direct float64 multiplication on base cost tiyins, truncated to int64 without rounding.
4. **`internal/matching/matching.go:207-214, 249`**:
   - Lines 207-208:
     ```go
     baseExpected := int64(math.Round(grItem.ReceivedQty * float64(poItem.UnitPriceMinor)))
     vatExpected := int64(math.Round(float64(baseExpected) * (poItem.VATRatePercent / 100.0)))
     ```
   - Lines 213-214:
     ```go
     actualLine := int64(math.Round(item.InvoicedQty * float64(item.InvoicedUnitPriceMinor)))
     actualVat := int64(math.Round(float64(actualLine) * (item.VATRatePercent / 100.0)))
     ```
   - Line 249: `int64(math.Abs(float64(varianceMinor))) <= maxAllowableTotalVariance`
   - Observed: 3-way matching invoice validation computes expected totals and VAT using float64 math and casts `int64` variance to `float64` for `math.Abs`.
5. **`internal/copa/copa.go:22-24, 116-128`**:
   - Line 22: `CardMDRRatePercent = 0.005` (0.5%)
   - Line 23: `CashLossProvisionRate = 0.001` (0.1%)
   - Lines 119-120:
     ```go
     cardFee := int64(math.Round(float64(input.CardAmountMinor) * CardMDRRatePercent))
     cashFee := int64(math.Round(float64(input.CashAmountMinor) * CashLossProvisionRate))
     ```
   - Line 128:
     ```go
     holdingCost := int64(math.Round(float64(input.GrossRevenueMinor) * dailyHoldingRate * float64(holdingDays)))
     ```
   - Observed: Profitability engine converts currency amounts to float64.
6. **`internal/fscm/dunning.go:125, 139, 180-181`**:
   - Lines 125-126:
     ```go
     dailyPenalty := float64(invoice.TotalAmountMinor) * CivilCode327DailyRate
     penaltyMinor = int64(math.Round(dailyPenalty * float64(days)))
     ```
   - Observed: Statutory Civil Code 327 late penalty calculated via float64.
7. **`internal/ar/dunning.go:108-110, 119-124`**:
   - Lines 108-110:
     ```go
     dailyRate := (annualRatePct / 100.0) / 365.0
     penalty := float64(balanceMinor) * dailyRate * float64(daysPastDue)
     return int64(math.Ceil(penalty))
     ```
   - Lines 119-124: Bad debt provisions computed using float64 multipliers `0.01, 0.05, 0.15, 0.40, 0.80`.
8. **`internal/supplier/service.go:1070-1071` & `models.go:95-142`**:
   - `models.go:97, 99, 114, 141, 142`: `VolumeTier1DiscountPct float64`, `VolumeTier2DiscountPct float64`, `CustomDiscountPct float64`
   - `service.go:1070-1071`:
     ```go
     discountMultiplier := 1.0 - (totalDiscountPct / 100.0)
     unitPriceMinor = int64(math.Round(float64(req.BasePriceMinor) * discountMultiplier))
     ```
9. **`internal/supplier/models.go:361-362`**:
   - ```go
     originalTotalTiyin := int64(math.Round(expectedTotalNominalWeight * float64(pricePerKgTiyin)))
     adjustedTotalTiyin := int64(math.Round(actualWeightKg * float64(pricePerKgTiyin)))
     ```
10. **`internal/inbound/repository.go:154-155`**:
    - ```go
      lineTotal := int64(it.OrderedQty * float64(unitPriceMinor))
      lineVAT := (lineTotal * int64(vatPercent)) / 100
      ```
11. **`internal/fiscal/fx_and_cash.go:118-119` & `fleet/fx_index.go:58-59`**:
    - ```go
      ratio := req.FXRateDispatch / req.FXRateCreation
      adjustedGross := int64(math.Round(float64(req.OriginalGrossTiyins) * ratio))
      ```
12. **`internal/fleet/fuel_theft.go:100, 129, 205`**:
    - ```go
      pricePerLiterMinor := int64(math.Round(float64(txn.TotalAmountMinor) / txn.LitersPurchased))
      loss := int64(math.Round(excessLiters * float64(pricePerLiterMinor)))
      ```
13. **`internal/payout/calculator.go:71` & `rails.go:20, 39`**:
    - `calculator.go:71`: `holdback = int64(math.Round(float64(eligible) * (reservePct / 100.0)))`
    - `rails.go:20, 39`: `grossUZS := float64(batch.NetPayoutTiyin) / 100.0; fmt.Sprintf("Сумма=%.2f\r\n", grossUZS)`
14. **`internal/planning/controltower.go:26, 32`**:
    - `revenueAtRiskTiyins := int64(incident.DeficitUnits * float64(incident.UnitPriceTiyin))`
15. **`internal/dispatch/shuttle.go:73-88`**:
    - `costPrimaryMinor := int64(math.Round(in.DistancePrimaryToRetailerKm*float64(in.FuelCostPerKmMinor))) + ...`
16. **`internal/ump/engine.go:95`**:
    - `subtotalDelta := int64(math.Round(req.DeltaNumeric * float64(unitPriceMinor)))`
17. **`internal/warehouse/service.go:577` & `qm/quarantine.go:79`**:
    - `claimAmount := int64(math.Round(shortage * float64(exp.UnitCostMinor)))`
    - `totalValueMinor := int64(math.Round(float64(unitCostMinor) * qty))`
18. **`internal/retailer/repository.go:2861-2862, 2883-2893`**:
    - `float64(totalMinor)/100.0`, `tot := int64(63000000); vat := (tot * 12) / 112`
19. **`internal/api/handlers_supplier.go:1107`**:
    - `type OnboardingProductPayload struct { ... VatRate float64 json:"vat_rate" }`

### 1.2 Naive CRUD, Concurrency Flaws & State Machine Bypasses
1. **`internal/api/handlers_fleet_driver.go:978-994` (`handleOrderComplete`)**:
   - Directly executes `UPDATE orders SET status = 'COMPLETED'` without authentication, without checking current order status, without `order.Service.TransitionStatus`, without inventory deductions, without outbox events, and discards errors with `_, _ =`.
2. **`internal/epod/repository.go:210-212` (`UpdateStopStatus`)**:
   - Executes `_, _ = r.pool.Exec(ctx, 'UPDATE orders SET status = 'DELIVERED', updated_at = NOW() WHERE order_id = $1', orderID)` outside a transaction, without checking if the order was cancelled, without row locking, and discards errors.
3. **`internal/wmsops/repository.go:1550-1563` (`ActionReplenishmentInsight`)**:
   - Performs a read check `if currentStatus != "OPEN" && currentStatus != "PENDING"`, and then executes `UPDATE warehouse_replenishment_insights` without a transaction and without `FOR UPDATE`. Classic TOCTOU concurrency race.
4. **`internal/api/handlers_payment.go:40, 53, 62, 230, 238, 247`**:
   - Six separate queries discard errors using `_, _ = tx.Exec(...)`.
5. **Broken Outbox Atomic Pairing (Split Transactions)**:
   - `internal/warehouse/service.go:352-364`: `UpdateOnboardingStatus` committed in 1st DB call; `outbox.Emit` called in 2nd DB transaction (`RunInTx`).
   - `internal/empties/service.go:447-457`: `SaveBalance` and `RecordMovement` called on repo; `outbox.Emit` called in separate `RunInTx`.
   - `internal/consignment/service.go:140-146`: `SaveAgreement` called on repo; `outbox.Emit` in separate `RunInTx`.
   - `internal/rebate/service.go:61-71`: `RecordAccrual` and `SaveContract` called on repo; `outbox.Emit` in separate `RunInTx`.
6. **Silent In-Memory Fallbacks**:
   - `internal/order/service.go:46, 342, 856-863`: Falls back to `inMemoryOrders` when `s.pool == nil`, mutating memory without validation or outbox.
   - `internal/retailer/repository.go:2881-2896`: Returns hardcoded fake fiscal receipt when DB query fails.

### 1.3 Subsystems Operating with Enterprise Rigor (Leave Intact)
1. `internal/payload/`: 3L-CVRP axle load moments ($W_{\text{steer}}, W_{\text{drive}}$), 11,500 kg single-axle safety limit, 20% steer tractive ratio, supervisor override PIN + bolt seal serial.
2. `internal/spatial/`: Uber H3 hexagonal discretization at Res 7 and Res 8, dead reckoning with traffic penalties, hex cell surge balancing.
3. `internal/dispatch/`: Dynamic fleet breakdown rescue hot-swap transferring undelivered stops via `s.pool.RunInTx` with `outbox.Emit` and Redis Streams.
4. `internal/planning/`: Syntetos-Boylan Classification (SBC), Croston-SBA forecasting, dynamic multi-echelon safety stock ($SS = z \cdot \sqrt{L \sigma_D^2 + D^2 \sigma_L^2}$).
5. `internal/soliq/`: E-IMZO PKCS#7 / CMS cryptographic signature engine with RSA, X.509 validity & legal INN matching.
6. `internal/gs1core/`: Mod-10 check digit verification for EAN-8, GTIN-12, GTIN-13, GTIN-14, SSCC-18, and Zebra ZPL-II label synthesis.
7. `internal/cashrecon/`: CIT driver cash drawer tracking, 25M UZS statutory threshold enforcement, depot smart safe vault drops, and double-entry GL deposit vouchers.
8. `internal/credit/`: Bilateral trade credit lines with `FOR UPDATE` concurrency, overdue debt validation, and `GREATEST(0, ...)` non-negative bounds.
9. `internal/telemetry/`: Lock-free ingestion with `sync.Map` + `atomic.Int64`, Redis geospatial indexing, and bounded memory background pruner.
10. `internal/dock/`: Dock bay appointment scheduling state machine and seal verification.
11. `internal/doorstep/`: 100m GPS proximity handshake, dynamic OTP/QR validation, itemized carton rejections.
12. `internal/epod/`: Electronic proof of delivery with offline SHA-256 payload hashing and digital signature capture.
13. `internal/fscm/`: Multi-factor credit scoring and 4-stage dunning escalation state machine.
14. `internal/claims/`: Damage claims adjudication with `FOR UPDATE` locking, terminal status checks, and basis-point line-item tax calculations.

---

## 2. Logic Chain

1. **Premise 1**: The Universal Engineering Doctrine (Rule 8) mandates that all currency arithmetic MUST be performed strictly in 64-bit integer tiyin minor units (`int64`), and statutory Soliq 12% VAT MUST use integer round-half-up math `(price * 12 + 50) / 100`.
2. **Observation**: Observations in Section 1.1 (C1 through C21) cite direct code in `soliq`, `rebate`, `consignment`, `matching`, `copa`, `fscm`, `ar`, `supplier`, `inbound`, `fiscal`, `fleet`, `payout`, `planning`, `dispatch`, `ump`, `warehouse`, `qm`, and `retailer` where `float64` is converted to/from minor units and rounded.
3. **Inference 1**: Floating-point operations in monetary calculations cause fractional rounding errors, violating statutory tax reconciliation and double-entry balance sheets.
4. **Premise 2**: Universal Engineering Doctrine (Section 1 & Section 3) strictly forbids naive CRUD: all state mutations must be guarded by domain state machines, concurrency locks (`FOR UPDATE` or atomic conditional WHERE), and must pair the entity update and the outbox event in the exact same database transaction (`pgx.Tx`).
5. **Observation**: Observations in Section 1.2 (1 through 6) demonstrate unauthenticated direct SQL status mutations (`handleOrderComplete`, `UpdateStopStatus`), TOCTOU races in `ActionReplenishmentInsight`, discarded errors in `handlers_payment.go`, and split transactions in `warehouse`, `empties`, `consignment`, and `rebate`.
6. **Inference 2**: In production, these flaws lead to state divergence, lost updates under concurrent access, silent transaction failures, and uncoordinated real-time event streaming across roles.
7. **Premise 3**: Features that already meet Google Principal Engineer standards (advanced mathematical modeling, robust concurrency, and physical logistics laws) should be cataloged and left intact.
8. **Observation**: Observations in Section 1.3 verify 14 subsystems adhering to high-performance distributed systems standards.

---

## 3. Caveats

1. **Third-Party External Protocols**: 1C CommerceML XML and 1C OData REST specifications (`internal/onec/`) require standard decimal currency values in sums/rubles (`DocumentAmount float64`, `Price float64`) as part of external schema compliance. For these external boundary adapters, float conversion is acceptable only at the final serialization layer, provided internal domain structures use `int64` tiyins.
2. **Physical Units**: Values representing GPS coordinates (latitude, longitude), vehicle heading, velocity, vehicle tare weight, and statistical demand parameters (ADI, CV2) are physical/statistical measures, not currency, and appropriately use `float64`.
3. **Catch Weight Representation**: Catch weight quantities (e.g. meat, cheese) involve decimal kilograms (e.g. 1.542 kg). Converting weights to integer grams (`int64`) allows strict integer arithmetic with tiyin prices per gram without float drift.

---

## 4. Conclusion

The `pegasus.x/backend` codebase possesses world-class algorithmic depth in its core logistics engines (axle physics, spatial dispatching, breakdown rescue, intermittent demand forecasting, and cryptographic tax signing). However, significant architectural debt exists in:
1. Floating-point arithmetic on currency in 21 identified areas.
2. Naive CRUD shortcuts in route handlers and repositories bypassing domain state machines.
3. Broken outbox atomic pairing (split transactions) in warehouse, empties, consignment, and rebate services.
4. Silent in-memory fallback repositories in production packages.

Remediating these issues is fully scoped, surgical, and actionable across Milestones 1, 2, and 3 without regressing passing test suites.

---

## 5. Verification Method

### 5.1 Static Scan Verification
1. Verify elimination of floating-point currency math:
   ```bash
   grep -rn "math.Round(float64" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/soliq/
   grep -rn "math.Round(float64" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/rebate/
   ```
   *Expected result*: Zero matches in currency/tax calculation functions.
2. Verify elimination of discarded SQL errors:
   ```bash
   grep -rn "_, _ = tx.Exec" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/api/handlers_payment.go
   ```
   *Expected result*: Zero matches; all errors checked.

### 5.2 Automated Test Execution
Run the full test suite with race detection enabled:
```bash
cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
go test -v -race ./...
```
*Expected result*: 100% passing test suites across all packages with zero race conditions and zero goroutine leaks.
