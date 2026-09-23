# Comprehensive Backend Audit: Currency Arithmetic, Naive CRUD, and Enterprise Rigor

**Codebase Target**: `pegasus.x/backend/`  
**Date**: 2026-09-23  
**Auditor**: `teamwork_preview_explorer_survey_11_2`  
**Status**: COMPLETE  

---

## 1. Executive Summary

This report delivers an exhaustive, non-destructive audit of all 83 backend packages in `pegasus.x/backend/internal/`, API route handlers in `internal/api/`, domain models, and database schema mappings against the root **Universal Enterprise Architecture & Engineering Doctrine** (Section 1: Zero-Tolerance for Naive CRUD; Section 5: Strict Two-System Architectural Boundary; Section 8: Strict 64-Bit Integer Tiyin Minor Units).

### Core Findings Breakdown
1. **Financial & Currency Arithmetic**: 
   - Found **21 distinct hotspots** across 15 packages where floating-point math (`float64`, `math.Round`, `math.Ceil`, float division `/ 100.0`, float percentages) is used for monetary values (invoices, VAT, penalties, provisions, volume rebates, unit costs, pricing overrides, and fees) instead of strict 64-bit integer tiyin minor units (`int64`).
   - Found explicit statutory tax violations in `internal/soliq/efactura.go` where Soliq 12% VAT is computed using `math.Round(float64(...))` rather than the mandatory statutory integer formula `(price * 12 + 50) / 100`.
   - Found floating-point DTO fields in `internal/api/handlers_supplier.go` (`VatRate float64`) and `internal/onec/types.go` (`DocumentAmount float64`, `Price float64`, `Amount float64`, `VATAmount float64`).
2. **Naive CRUD & State Machine Bypasses**:
   - Identified **5 critical endpoints** that perform uncoordinated, unauthenticated, or unvalidated state mutations directly against PostgreSQL (e.g. `api/handlers_fleet_driver.go:986`, `epod/repository.go:211`, `doorstep/repository.go:333`, `retailer/repository.go:2828`).
   - Discovered repeated instances of discarded errors (`_, _ = tx.Exec(...)` or `_, _ = r.pool.Exec(...)`) in `api/handlers_payment.go`, `epod/repository.go`, `fleet/repository.go`.
   - Detected **TOCTOU (Time-of-Check to Time-of-Use)** race conditions in `wmsops/repository.go:1554` (checking status in a separate query, then updating without `FOR UPDATE` or conditional `WHERE status IN (...)`).
   - Uncovered split-transaction anti-patterns in `warehouse/service.go:352-364`, `empties/service.go:447-457`, `consignment/service.go:140-146`, and `rebate/service.go:61-71`, where the domain entity is committed in one database call, and the transactional outbox event is emitted in a separate transaction closure (`s.pool.RunInTx`), violating atomic pairing.
   - Identified silent in-memory fallback mechanisms in `internal/order/service.go` (`inMemoryOrders map[string]*models.Order`) and `internal/retailer/repository.go:2881-2896` where failure to read from DB falls back to hardcoded fake data.
3. **Enterprise Rigor Systems (To Preserve Intact)**:
   - Verified **14 world-class subsystems** built with mathematical rigor, formal state machines, and proper concurrency controls:
     - 3L-CVRP longitudinal static moment axle calculation and 11,500 kg road safety gate in `internal/payload/`.
     - Uber H3 spatial discretization and hexagonal surge dispatching in `internal/spatial/`.
     - Mid-shift fleet breakdown rescue hot-swapping in `internal/dispatch/`.
     - Syntetos-Boylan Classification (SBC) and Croston-SBA intermittent demand forecasting in `internal/planning/`.
     - E-IMZO PKCS#7 / CMS digital signature validation with RSA & X.509 in `internal/soliq/`.
     - GS1 Mod-10 check digit algorithms and ZPL-II label synthesis in `internal/gs1core/`.
     - Cash-in-transit (CIT) drawer tracking and depot vault drop rituals in `internal/cashrecon/`.
     - Bilateral trade credit lines with atomic `FOR UPDATE` locking in `internal/credit/`.

---

## 2. Currency & Financial Arithmetic Audit (Floating-Point vs int64 Tiyins)

The Universal Engineering Doctrine establishes a non-negotiable invariant:
> *"Currency Arithmetic: Strict 64-bit integer tiyin minor unit arithmetic (`int64`). Floating-point arithmetic for currency is strictly prohibited (`float32`, `float64`)."*
> *"VAT Calculation: Statutory 12% VAT applied to taxable items using standard integer round-half-up math (`(price * 12 + 50) / 100`)."*

### Detailed Inventory of Violations

| ID | Package & File | Line Numbers | Observed Code | Violation Description & Risk | Prescribed Architectural Remediation |
|---|---|---|---|---|---|
| **C1** | `internal/soliq/efactura.go` | 131, 198 | `lineVAT := int64(math.Round(float64(lineSubtotal*int64(it.VATPercent)) / 100.0))` <br>`deltaVAT := int64(math.Round(float64(deltaSum*int64(adj.VATRate)) / 100.0))` | Uses `float64` and `math.Round` on tax calculation. Floating-point imprecision creates 1-tiyin rounding drift on large invoice totals, failing Soliq OFD fiscal audit reconciliation. | Replace with pure integer round-half-up math: <br>`lineVAT := (lineSubtotal*int64(it.VATPercent) + 50) / 100`<br>`deltaVAT := (deltaSum*int64(adj.VATRate) + 50) / 100` |
| **C2** | `internal/rebate/rebate.go` | 44, 85, 123 | `RebatePercent float64`<br>`accrualAmount := int64(math.Round(float64(orderVolumeMinor) * (contract.RebatePercent / 100.0)))` | Stores contract rebate percentage as `float64` and converts financial order volume `int64` to `float64` for GL accrual journal entries. | Store contract rebate rate in integer basis points (`RebateBps int64`, where 1% = 100 bps). Calculate accruals via:<br>`accrualAmount := (orderVolumeMinor * contract.RebateBps + 5000) / 10000` |
| **C3** | `internal/consignment/consignment.go` | 24, 44, 94-95 | `ConsignmentStockQty float64`<br>`totalBaseCost := int64(qty * float64(agreement.AgreedBaseCostMinor))` <br>`totalRetailPrice := int64(qty * float64(retailUnitPriceMinor))` | Quantities and base cost are multiplied as float64 and truncated directly to `int64` without rounding. Truncation drops minor unit fractions. | Convert standard packaging units to integer units (`int` or `int64`). If catch weight, use integer thousandths of kg (grams) and integer minor units. |
| **C4** | `internal/matching/matching.go` | 44, 87, 207-214, 249 | `baseExpected := int64(math.Round(grItem.ReceivedQty * float64(poItem.UnitPriceMinor)))`<br>`vatExpected := int64(math.Round(float64(baseExpected) * (poItem.VATRatePercent / 100.0)))`<br>`int64(math.Abs(float64(varianceMinor)))` | 3-way purchase invoice matching calculates expected line total and VAT using float64. Uses `math.Abs(float64(int64))` for integer difference. | Calculate expected lines using integer unit prices. Define `absInt64(x int64) int64`. Use basis points for VAT: `(baseExpected * vatBps + 5000) / 10000`. |
| **C5** | `internal/copa/copa.go` | 22-24, 116-128 | `CardMDRRatePercent = 0.005`<br>`CashLossProvisionRate = 0.001`<br>`cardFee := int64(math.Round(float64(input.CardAmountMinor) * CardMDRRatePercent))` <br>`cashFee := int64(math.Round(float64(input.CashAmountMinor) * CashLossProvisionRate))` <br>`holdingCost := int64(math.Round(float64(input.GrossRevenueMinor) * dailyHoldingRate * float64(holdingDays)))` | Profitability and customer cost-to-serve engine converts financial revenue, card, and cash amounts to `float64` and rounds. | Define fee rates in integer basis points: `CardMDRRateBps = 50` (0.5%), `CashLossProvisionBps = 10` (0.1%). Compute fees via integer math: `(amount * bps + 5000) / 10000`. |
| **C6** | `internal/fscm/dunning.go` | 125-140, 180-181 | `dailyPenalty := float64(invoice.TotalAmountMinor) * CivilCode327DailyRate`<br>`penaltyMinor = int64(math.Round(dailyPenalty * float64(days)))` | Uzbekistan Civil Code Article 327 late penalty (0.1%/day) is computed using `float64` multiplication. | 0.1%/day equals 10 basis points per day. Use integer formula: `penaltyMinor := (invoice.TotalAmountMinor * int64(days) * 10 + 5000) / 10000`. |
| **C7** | `internal/fscm/scoring.go` | 103 | `recommendedLimit := int64(math.Round(float64(metrics.Avg30dDeliveredVolumeMinor) * scoreFactor * tenureMultiplier))` | Credit scoring recommendation uses float64 multiplier on 30-day delivery volume. | Scale score factor and tenure multiplier to integer basis points (e.g. 1.0 = 10000) and perform integer scaling. |
| **C8** | `internal/ar/dunning.go` | 108-110, 119-124 | `dailyRate := (annualRatePct / 100.0) / 365.0`<br>`penalty := float64(balanceMinor) * dailyRate * float64(daysPastDue)`<br>`CalculateBadDebtProvision`: reserves `0.01, 0.05, 0.15, 0.40, 0.80` | AR dunning penalty interest and bad debt reserves use float64 arithmetic on tiyins. | Use integer basis points: 1% = 100 bps, 5% = 500 bps, 15% = 1500 bps, 40% = 4000 bps, 80% = 8000 bps. Compute: `(balance * bps + 5000) / 10000`. |
| **C9** | `internal/supplier/service.go` & `models.go` | `models.go:95-142`, `service.go:1070-1071` | `discountMultiplier := 1.0 - (totalDiscountPct / 100.0)`<br>`unitPriceMinor = int64(math.Round(float64(req.BasePriceMinor) * discountMultiplier))` | Volume tier and contract override discounts are evaluated using floating-point percentages and multiplier float casting. | Represent discounts in basis points (`DiscountBps int64`). Compute discounted price: `unitPriceMinor = req.BasePriceMinor - (req.BasePriceMinor * discountBps + 5000) / 10000`. |
| **C10** | `internal/supplier/models.go` | 361-362 | `originalTotalTiyin := int64(math.Round(expectedTotalNominalWeight * float64(pricePerKgTiyin)))`<br>`adjustedTotalTiyin := int64(math.Round(actualWeightKg * float64(pricePerKgTiyin)))` | Catch-weight adjustment multiplies weight by float price per kg. | Store catch-weight in integer grams (`WeightGrams int64`). Store price per gram (`PricePerGramTiyin int64`) or scale `(Grams * PricePerKgTiyin + 500) / 1000`. |
| **C11** | `internal/inbound/repository.go` | 154-155, 169 | `lineTotal := int64(it.OrderedQty * float64(unitPriceMinor))`<br>`lineVAT := (lineTotal * int64(vatPercent)) / 100` | PO item line total casts unit price to float64. Line VAT uses integer floor truncation rather than round-half-up. | `lineTotal := int64(it.OrderedQty) * unitPriceMinor`. `lineVAT := (lineTotal*int64(vatPercent) + 50) / 100`. |
| **C12** | `internal/fiscal/fx_and_cash.go` & `fleet/fx_index.go` | `fiscal:118-119`, `fleet:58-59` | `ratio := req.FXRateDispatch / req.FXRateCreation`<br>`adjustedGross := int64(math.Round(float64(req.OriginalGrossTiyins) * ratio))` | FX indexation converts gross tiyins to float64, multiplies by floating exchange rate ratio, and rounds. | Store FX rates in integer micro-units (e.g. `12,850.50` UZS/USD = `128505000` micro-UZS). Ratio = `(lockRate * 1000000) / baseRate`. |
| **C13** | `internal/fleet/fuel_theft.go` | 100, 129, 150, 205 | `pricePerLiterMinor := int64(math.Round(float64(txn.TotalAmountMinor) / txn.LitersPurchased))`<br>`loss := int64(math.Round(excessLiters * float64(pricePerLiterMinor)))` | Fuel theft financial loss converts minor amounts to float64 and rounds. | Track fuel in milliliters (`int64(liters * 1000)`). Calculate `pricePerMlTiyin := txn.TotalAmountMinor / milliliters`. |
| **C14** | `internal/payout/calculator.go` | 25, 71 | `reservePct float64`<br>`holdback = int64(math.Round(float64(eligible) * (reservePct / 100.0)))` | Payout holdback reserve converts eligible tiyins to float64. | Accept `reserveBps int64`. Compute: `holdback = (eligible * reserveBps + 5000) / 10000`. |
| **C15** | `internal/payout/rails.go` | 20, 39 | `grossUZS := float64(batch.NetPayoutTiyin) / 100.0`<br>`fmt.Sprintf("Сумма=%.2f\r\n", grossUZS)` | 1C Client-Bank text file generation converts net payout tiyin to float64 for formatting string. | Format directly from integer tiyins: `fmt.Sprintf("Сумма=%d.%02d\r\n", batch.NetPayoutTiyin/100, batch.NetPayoutTiyin%100)`. |
| **C16** | `internal/planning/controltower.go` | 26, 32 | `revenueAtRiskTiyins := int64(incident.DeficitUnits * float64(incident.UnitPriceTiyin))` | Revenue at risk converts unit price to float64. | Deficit units should be integer or scaled; compute `int64(incident.DeficitUnits) * incident.UnitPriceTiyin`. |
| **C17** | `internal/dispatch/shuttle.go` | 73-88 | `costPrimaryMinor := int64(math.Round(in.DistancePrimaryToRetailerKm*float64(in.FuelCostPerKmMinor))) + ...` | Shuttle transfer economics converts fuel and driver hourly cost minor units to float64. | Compute costs using integer meters and seconds, or scale distance to integer meters: `(meters * fuelPerMeterMinor) / 1000`. |
| **C18** | `internal/ump/engine.go` | 95 | `subtotalDelta := int64(math.Round(req.DeltaNumeric * float64(unitPriceMinor)))` | Universal Mutation Protocol delta calculation multiplies float delta by float64 unit price. | If delta is discrete units, use `int64(req.DeltaUnits) * unitPriceMinor`. If catch-weight, use grams. |
| **C19** | `internal/warehouse/service.go` & `qm/quarantine.go` | `warehouse:577`, `qm:79` | `claimAmount := int64(math.Round(shortage * float64(exp.UnitCostMinor)))`<br>`totalValueMinor := int64(math.Round(float64(unitCostMinor) * qty))` | Shortage claim amount and quarantine inventory valuation convert `UnitCostMinor` to float64. | Use integer quantity multiplication: `int64(shortageUnits) * exp.UnitCostMinor`. |
| **C20** | `internal/retailer/repository.go` | 2861-2862, 2883-2893 | `float64(totalMinor)/100.0`, `float64(vatMinor)/100.0`<br>`tot := int64(63000000); vat := (tot * 12) / 112` | In-memory fallback receipt generation uses float64 for UZS strings and hardcoded integers. | Format strings using integer division/modulo: `%d.%02d`. Purge the in-memory fallback entirely. |
| **C21** | `internal/api/handlers_supplier.go` | 1107 | `type OnboardingProductPayload struct { VatRate float64 ... }` | Ingress JSON DTO accepts `vat_rate` as a `float64` without basis point conversion. | Parse `vat_rate` as integer percentage or integer basis points (`VatRateBps int64`), defaulting to 1200 bps (12.00%). |

---

## 3. Naive CRUD & Concurrency/State Machine Audit

The Universal Engineering Doctrine commands:
> *"Zero-Tolerance for Naive CRUD: Never write dumb table mutations that lack domain state machines, concurrency checks, transactional outbox events, validation guards, or comprehensive telemetry."*

### Detailed Inventory of Naive CRUD & Concurrency Flaws

#### A. Direct Unvalidated State Jumps (State Machine Bypasses)
1. **`internal/api/handlers_fleet_driver.go:978-994` (`handleOrderComplete`)**:
   - **Flaw**: Handler accepts `POST /v1/order/complete` and directly executes:
     ```go
     _, _ = s.pool.Exec(r.Context(), `UPDATE orders SET status = 'COMPLETED', updated_at = $1 WHERE order_id = $2`, now, req.OrderID)
     _, _ = s.pool.Exec(r.Context(), `UPDATE manifest_stops SET status = 'COMPLETED', completed = true, completed_at = $1 WHERE order_id = $2`, now, req.OrderID)
     ```
   - **Vulnerabilities**:
     - No authentication or role check (`auth.GetClaims` is not evaluated).
     - No check on the current status of the order. An order in `DRAFT`, `CANCELLED`, or `CREATED` can jump directly to `COMPLETED`.
     - Completely bypasses `order.Service.TransitionStatus`, so stock is not deducted from inventory, payment is not settled, and no outbox events are emitted.
     - Errors from both `Exec` calls are discarded with `_, _ =`.
   - **Remediation**: Remove direct SQL mutations; route through `order.Service.TransitionStatus(ctx, orderID, models.StatusDelivered, ...)` within an authenticated driver session.

2. **`internal/epod/repository.go:210-212` (`UpdateStopStatus`)**:
   - **Flaw**: When a stop is marked delivered, it executes:
     ```go
     if completed {
         _, _ = r.pool.Exec(ctx, `UPDATE orders SET status = 'DELIVERED', updated_at = NOW() WHERE order_id = $1`, orderID)
     }
     ```
   - **Vulnerabilities**:
     - Unconditional status jump without checking if the order was actually loaded on the manifest or already cancelled.
     - Ignores database errors (`_, _ =`).
     - Occurs outside a transaction closure.
   - **Remediation**: Require `pgx.Tx` closure; lock order with `FOR UPDATE`; verify current status is `IN_TRANSIT` or `LOADED`; emit `order.delivered` outbox event.

3. **`internal/retailer/repository.go:2828` (`SelectTenderType`)**:
   - **Flaw**: Executes `_, _ = r.pool.Exec(ctx, 'UPDATE orders SET payment_method = $1 WHERE order_id = $2', req.TenderChoice, orderID)`.
   - **Vulnerabilities**:
     - Does not verify if the order is already paid or completed.
     - Discards error.
   - **Remediation**: Check current payment status (`WHERE order_id = $2 AND payment_status != 'PAID'`); emit outbox event on tender selection.

#### B. TOCTOU (Time-of-Check to Time-of-Use) Vulnerabilities
1. **`internal/wmsops/repository.go:1550-1563` (`ActionReplenishmentInsight`)**:
   - **Flaw**:
     ```go
     // Check status
     err = p.pool.QueryRow(ctx, `SELECT status FROM warehouse_replenishment_insights WHERE insight_id = $1`, insightID).Scan(&currentStatus)
     if currentStatus != "OPEN" && currentStatus != "PENDING" {
         return ErrInsightAlreadyActioned
     }
     // Update status later without lock
     query := `UPDATE warehouse_replenishment_insights SET status = $2, ... WHERE insight_id = $1`
     _, err = p.pool.Exec(ctx, query, insightID, status, reasonCode, targetPOID)
     ```
   - **Vulnerabilities**:
     - High concurrency race condition: Two concurrent warehouse admins can simultaneously read `status = 'OPEN'`, pass the validation check, and both attempt to generate purchase orders for the same insight.
   - **Remediation**: Execute an atomic conditional update:
     ```sql
     UPDATE warehouse_replenishment_insights
     SET status = $2, reason_code = $3, target_po_id = $4, actioned_at = NOW()
     WHERE insight_id = $1 AND status IN ('OPEN', 'PENDING')
     ```
     Verify `tag.RowsAffected() == 1`, returning `ErrInsightAlreadyActioned` if 0 rows were updated.

#### C. Ignored Errors in Mutating Queries (`_, _ = tx.Exec(...)`)
1. **`internal/api/handlers_payment.go:40, 53, 62, 230, 238, 247`**:
   - **Flaw**: Multiple lines execute `_, _ = tx.Exec(...)` when writing payment legs, order status updates, and general ledger journal entries.
   - **Vulnerabilities**:
     - If a database constraint fails (e.g. duplicate key, foreign key failure, null constraint), the error is discarded, and the transaction commits partial state without recording the payment leg or ledger entry.
   - **Remediation**: Every `tx.Exec` call must have its error checked and returned immediately:
     ```go
     if _, err := tx.Exec(...); err != nil {
         return fmt.Errorf("insert payment leg: %w", err)
     }
     ```

2. **`internal/fleet/repository.go:2532, 2823`**:
   - **Flaw**: `_, _ = tx.Exec(ctx, 'UPDATE orders SET status = 'DELIVERED', updated_at = $1 WHERE order_id = $2', now, req.OrderID)`.
   - **Remediation**: Always capture and handle error; ensure row exists (`RowsAffected() > 0`).

#### D. Split-Transaction Anti-Patterns (Broken Outbox Invariant)
The doctrine requires:
> *"Transactional Outbox: Entity state mutation and outbox event MUST be written in the exact same database transaction (`pgx.Tx` in PostgreSQL 16)."*

In several packages, the entity is modified first in one transaction/autocommit, and then `outbox.Emit` is called in a completely separate transaction:
1. **`internal/warehouse/service.go:352-364` (`CompleteOnboarding`)**:
   - Line 352: `s.repo.UpdateOnboardingStatus(ctx, warehouseID, "COMPLETED")` (1st DB operation).
   - Line 357: `s.pool.RunInTx(ctx, func(tx pgx.Tx) error { return outbox.Emit(...) })` (2nd DB transaction).
2. **`internal/empties/service.go:447-457` (`RecordMovement`)**:
   - Line 447: `s.repo.SaveBalance(ctx, balance)`
   - Line 451: `s.repo.RecordMovement(ctx, result)`
   - Line 456: `s.pool.RunInTx(ctx, func(tx pgx.Tx) error { return outbox.Emit(...) })`
3. **`internal/consignment/service.go:140-146` (`InboundConsignmentReceive`)**:
   - Line 140: `s.repo.SaveAgreement(ctx, agreement)`
   - Line 145: `s.pool.RunInTx(ctx, func(tx pgx.Tx) error { return outbox.Emit(...) })`
4. **`internal/rebate/service.go:61-71` (`RecordAccrualsForOrder`)**:
   - Line 61: `s.repo.RecordAccrual(...)`
   - Line 64: `s.repo.SaveContract(...)`
   - Line 70: `s.pool.RunInTx(ctx, func(tx pgx.Tx) error { return outbox.Emit(...) })`

**Impact**: If the process crashes, the network disconnects, or the pod restarts between step 1 and step 2, the entity state change is committed in the database, but the outbox event is **never emitted**. Downstream roles (finance, warehouse, retailer, driver) will never receive the update, causing state divergence.  
**Remediation**: Refactor repository methods to accept an existing `tx pgx.Tx` (or combine the mutation and outbox write in a single `pool.RunInTx` closure).

#### E. Silent In-Memory Fallbacks in Production Code
1. **`internal/order/service.go:46, 342, 856-863`**:
   - If `s.pool == nil`, `order.Service` falls back to `s.inMemoryOrders[req.OrderID] = createdOrder`.
   - In `TransitionStatus`, when `s.pool == nil`, it modifies `inMemoryOrders` without validation, without inventory side effects, and without outbox events.
2. **`internal/retailer/repository.go:2881-2896`**:
   - When fetching a fiscal receipt, if `r.pool.QueryRow` fails, it falls back to hardcoded mock strings ("FS-SOLIQ-UZ-983192083", `tot := int64(63000000)`).
   - This violates the Zero Mock Data Policy. If the DB fails, the repository must fail closed and return the error.

---

## 4. Enterprise Rigor Catalog (Subsystems to Preserve Intact)

The audit identified **14 battle-tested, high-performance packages** that adhere to Google Principal Engineer standards. These packages must be strictly preserved without breaking their mathematical or algorithmic integrity:

### 1. `internal/payload/` (3L-CVRP Longitudinal Axle Physics & Road Safety Gate)
- **Mathematical Invariant**:
  $$W_{\text{steer}} = W_{\text{curb,steer}} + \sum \frac{w_i (L - x_i)}{L}, \quad W_{\text{drive}} = W_{\text{curb,drive}} + \sum \frac{w_i x_i}{L}$$
- **Features**:
  - Enforces Uzbekistan statutory single-axle weight limit of 11,500 kg (`ErrAxleOverloadViolation`).
  - Enforces minimum 20.0% tractive steering axle ratio (`MinSteerAxleShareRatio`).
  - Mechanically bounds pallet centers of gravity across cargo bed $x \in [-500, L + 1500]$ mm.
  - Implements formal Supervisor Override workflow requiring 4-digit PIN / 14-digit PINFL, mandatory reason code, and digital bolt seal serial number (`SEAL-UZ-XXXXXX`).

### 2. `internal/spatial/` (Uber H3 Discretization & Hex Dispatch Engine)
- **Features**:
  - Implements hexagonal spatial discretization at Resolution 7 (~1.2 km cell radius) and Resolution 8 (~460 m cell radius).
  - Dead reckoning transit ETA calculation factoring velocity (0-180 km/h), vehicle heading degrees (0-360), traffic congestion penalty (0.0 to 0.85), and doorstep dwell time.
  - Spatial surge balancing: Computes supply-demand ratio per hex cell and classifies zones into `BALANCED`, `HIGH_DEMAND_SURGE`, or `SURPLUS_CAPACITY`.

### 3. `internal/dispatch/` (Fleet Breakdown Rescue Hot-Swapping)
- **Features**:
  - Dynamic mid-shift rescue hot-swap without cancelling retailer orders: transfers undelivered stops from a broken truck to an active or idle rescue vehicle.
  - Atomic transaction closure (`s.pool.RunInTx` in `service.go:830-924`):
    - Creates rescue manifest (`manifests`).
    - Inserts transferred stops (`manifest_stops`).
    - Records audit trail in `manifest_stop_transfers`.
    - Updates order driver and vehicle assignment.
    - Updates rescue driver state to `IN_TRANSIT`.
    - Emits `FLEET_BREAKDOWN_RESCUED` via `outbox.Emit(ctx, tx, ...)`.
    - Publishes to Redis Streams and PubSub channels.

### 4. `internal/planning/` (Croston-SBA Intermittent Demand & Dynamic Safety Stock)
- **Features**:
  - Syntetos-Boylan Classification (SBC): Categorizes demand history into Smooth, Intermittent, Erratic, or Lumpy based on Average Demand Interval ($\text{ADI} \le 1.32$) and Squared Coefficient of Variation ($\text{CV}^2 \le 0.49$).
  - Croston's method with Syntetos-Boylan approximation ($\alpha_{\text{SBA}} = 0.15$).
  - Dynamic multi-echelon safety stock ($SS = z_{0.95} \cdot \sqrt{L \cdot \sigma_D^2 + D^2 \cdot \sigma_L^2}$).

### 5. `internal/soliq/` (E-IMZO PKCS#7 / CMS Cryptographic Signature Engine)
- **Features**:
  - True cryptographic verification of electronic invoices under Uzbekistan tax law.
  - Deterministic canonical JSON serialization for SHA-256 digest computation.
  - PKCS#7 / CMS signed data verification with RSA signatures and X.509 certificate expiry & legal INN matching.

### 6. `internal/gs1core/` (Packaging Hierarchy & Mod-10 Check Digit Verification)
- **Features**:
  - Full Mod-10 check digit algorithms for EAN-8, GTIN-12 (UPC-A), GTIN-13 (EAN-13), GTIN-14 (ITF-14), GLN, and SSCC-18.
  - Zebra ZPL-II logistics label generator for pallet SSCC barcodes and product identification.

### 7. `internal/cashrecon/` (Driver CIT Drawer & Vault Drop Rituals)
- **Features**:
  - Real-time driver cash drawer tracking in tiyins (`current_cash_drawer_minor`).
  - Cash-in-transit (CIT) threshold alerts against statutory 25,000,000 UZS cash ceilings.
  - Mid-shift depot smart safe drops (`RecordMidShiftVaultDropReq`) and end-of-shift bank deposit reconciliation with double-entry general ledger postings.

### 8. `internal/credit/` (Bilateral Trade Credit Lines & Concurrency Controls)
- **Features**:
  - Concurrency safety: Uses `SELECT ... FOR UPDATE` on `supplier_retailer_credit_lines`.
  - Invariant validation: Checks credit limit headroom, overdue invoices, and net days before allowing credit order placement.
  - Non-negative debt updates guarded by `GREATEST(0, utilized_debt_minor - $1)`.

### 9. `internal/telemetry/` (Lock-Free Ingestion & Geospatial Indexing)
- **Features**:
  - Lock-free throttled fanout using `sync.Map` and `atomic.Int64` timestamps to prevent Redis flood.
  - Background memory-bounded pruner evicting inactive GPS beacons.
  - Haversine geofence arrival verification.

### 10. `internal/dock/` (Dock Bay Appointment Scheduling & State Machine)
- **Features**:
  - Formal dock bay state transitions (`SCHEDULED` -> `CHECKED_IN` -> `LOADING` -> `SEALED` -> `DEPARTED`).
  - Digital bolt seal serial logging and dock progress metrics.

### 11. `internal/doorstep/` (100m Proximity Handshake & Bilateral Inspection)
- **Features**:
  - OTP/QR dynamic token exchange between driver app and retailer terminal.
  - 100-meter GPS proximity validation before allowing offload handover.
  - Itemized damaged carton rejection with photo proof capture.

### 12. `internal/epod/` (Electronic Proof of Delivery & Tamper-Evident Signatures)
- **Features**:
  - Offline sync engine with tamper-evident SHA-256 payload hashing.
  - Digital signature capture, photo proof recording, and GPS arrival confirmation.

### 13. `internal/fscm/` (Credit Risk Scoring & Dunning State Machine)
- **Features**:
  - Multi-factor credit scoring: punctuality, volume, tenure, discrepancy, card payment ratio.
  - 4-stage dunning escalation state machine (Reminder -> Warning -> Late Penalty Accrual -> Pre-Litigation Talabnoma).

### 14. `internal/claims/` (Damage Claims Adjudication & Tax Accounting)
- **Features**:
  - `FOR UPDATE` row locking during claim adjudication.
  - State machine check preventing re-adjudication of finalized claims.
  - Integrates line-item basis-point tax calculations (`fiscal.CalculateLineTaxes`) and automatic quarantine lot creation.

---

## 5. Architectural Recommendations & Fix Strategy for Milestones 1-3

### Phase 1: Fail-Closed Constructors & Elimination of Fallback Repositories (Milestone 1)
1. **Refactor Constructors**: In `internal/consignment`, `internal/rebate`, `internal/payout`, `internal/wmsops`, and `internal/order`, ensure `NewService` and `NewRepository` require a non-nil `*db.Pool`. If `pool == nil`, return `nil, errors.New("database pool is required")` (fail-closed).
2. **Move In-Memory Repositories**: Move any `MemoryRepository` definitions strictly into `*_test.go` files for unit tests.
3. **Purge Fake Fallbacks**: Remove `r.pool == nil` fallback logic in `retailer/repository.go:2881` and `order/service.go:342`.

### Phase 2: Elimination of Floating-Point Currency Arithmetic (Milestone 2)
1. **Soliq E-Factura VAT**: Update `internal/soliq/efactura.go` lines 131 and 198 to use pure integer arithmetic: `(lineSubtotal * int64(it.VATPercent) + 50) / 100`.
2. **Rebates & Accruals**: Convert `RebatePercent` in `internal/rebate/rebate.go` to `RebateBps int64`. Compute accruals with `(orderVolumeMinor * contract.RebateBps + 5000) / 10000`.
3. **FSCM Civil Code 327 Penalties**: Update `internal/fscm/dunning.go` to use integer basis points (10 bps/day) instead of `float64` multiplication.
4. **COPA Rates & Fees**: Replace `float64` fee rates in `internal/copa/copa.go` with integer basis points (`CardMDRRateBps = 50`, `CashLossProvisionBps = 10`).
5. **Supplier Discounts & Catch Weight**: Represent discounts in basis points (`VolumeTierDiscountBps int64`). Compute catch-weight adjustments using integer grams.
6. **API DTOs**: Update `internal/api/handlers_supplier.go` to parse `vat_rate` as integer basis points or validate strict integer tiyins.

### Phase 3: Transactional Outbox Atomic Pairing & State Machine Enforcement (Milestone 3)
1. **Atomic Transaction Closures**: Refactor `warehouse/service.go`, `empties/service.go`, `consignment/service.go`, and `rebate/service.go` so that the entity mutation and `outbox.Emit` execute within the **exact same `pgx.Tx` transaction**.
2. **Eliminate Naive CRUD Handlers**:
   - Rewrite `api/handlers_fleet_driver.go:978` (`handleOrderComplete`) to require authentication, validate the order state machine via `order.Service.TransitionStatus`, and emit the completion outbox event.
   - Guard `epod/repository.go:211` status updates with row-level locks and transition checks.
   - Refactor `wmsops/repository.go:1558` to use atomic conditional `UPDATE ... WHERE insight_id = $1 AND status IN ('OPEN', 'PENDING')`.
3. **Check All Exec Errors**: Replace all `_, _ = tx.Exec(...)` with explicit error handling and rollback propagation.
