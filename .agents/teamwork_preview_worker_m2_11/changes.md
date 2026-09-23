# Milestone 2 Code Changes — Currency Arithmetic & Domain State Machine Purity

This document details all changes implemented in `pegasus.x` (Sovereign Lean Single-Tenant National Operating Core) to satisfy Milestone 2 (Requirements R1.1 and R1.2).

---

## 1. Currency Arithmetic Hardening (Requirement R1.1 — Zero Float Math)

### 1.1 `backend/internal/soliq/efactura.go`
- **What**: Replaced floating-point VAT calculation with statutory Uzbekistan integer round-half-up minor unit arithmetic:
  - Line VAT calculation: `lineVAT := (lineSubtotal*int64(it.VATPercent) + 50) / 100` (removed `float64(it.VATPercent) / 100.0 * float64(lineSubtotal)`).
  - Credit memo line adjustment: `deltaVAT := (deltaSum*int64(adj.VATRate) + 50) / 100`.
  - Removed `math` package import.
- **Why**: Uzbekistan Soliq OFD tax compliance mandates exact integer round-half-up minor unit arithmetic without IEEE-754 precision loss.

### 1.2 `backend/internal/rebate/rebate.go`
- **What**: Converted percentage-based rebate accruals to integer basis points (`int64`, 1 bp = 0.01%, 10,000 bp = 100%):
  - Added `RebateBps int64` to `RebateRule`.
  - Accrual calculation: `accruedMinor := (orderVolumeMinor*rebateBps + 5000) / 10000`.
  - Removed `math` package import.
- **Why**: Eliminates floating-point multiplication in supplier-retailer volume rebate accruals and reconciliations.

### 1.3 `backend/internal/fscm/dunning.go`
- **What**: Replaced float-based interest penalties with integer basis points arithmetic:
  - Defined `CivilCode327DailyBps = int64(10)` (0.10% per day / 10 basis points per day).
  - Penalty calculation: `penalty := (principal*days*CivilCode327DailyBps + 5000) / 10000`.
  - Removed `math` package import.
- **Why**: Complies with Article 327 of the Civil Code of the Republic of Uzbekistan with deterministic integer tiyin rounding.

### 1.4 `backend/internal/copa/copa.go`
- **What**: Converted fee percentages to basis points constants:
  - `CardMDRRateBps = int64(50)` (0.50% card MDR fee).
  - `CashLossProvisionBps = int64(10)` (0.10% cash leakage reserve).
  - `CostOfCapitalAPRBps = int64(1600)` (16.0% annual cost of capital).
  - Fee calculation: `cardFee := (input.CardAmountMinor*CardMDRRateBps + 5000) / 10000`, `cashFee := (input.CashAmountMinor*CashLossProvisionBps + 5000) / 10000`.
  - Holding cost: `holdingCost := (input.GrossRevenueMinor*CostOfCapitalAPRBps*int64(holdingDays) + 1825000) / 3650000`.
  - Removed `math` package import.
- **Why**: Eliminates float drift in Activity-Based Costing (ABC) drop profitability accounting.

### 1.5 `backend/internal/ar/dunning.go`
- **What**: Defined bad debt reserve basis points:
  - `ReserveCurrentBps = 100` (1%), `Reserve1To30Bps = 500` (5%), `Reserve31To60Bps = 1500` (15%), `Reserve61To90Bps = 4000` (40%), `Reserve90PlusBps = 8000` (80%).
  - Refactored `CalculateBadDebtProvision` to `(balance*bps + 5000) / 10000`.
  - Refactored `CalculateInterestPenalty` to integer daily basis points.
  - Removed `math` package import.
- **Why**: Exact integer accounting for Accounts Receivable provisioning and aging penalties.

### 1.6 `backend/internal/matching/matching.go`
- **What**: Replaced float line total calculation in 3-way matching engine:
  - Converted received quantity to milliunits: `grQtyMilli := int64(grItem.ReceivedQty*1000.0 + 0.5)`.
  - Line expected base: `baseExpected := (grQtyMilli*poItem.UnitPriceMinor + 500) / 1000`.
  - Line expected VAT: `vatExpected := (baseExpected*vatBps + 5000) / 10000`.
  - Replaced `math.Abs` with integer `absInt64`.
  - Removed `math` package import.
- **Why**: Guarantees deterministic 3-way matching between PO, GRN, and AP invoice without rounding discrepancies.

### 1.7 `backend/internal/consignment/consignment.go`
- **What**: Replaced float unit price and total cost calculations:
  - Base cost in milliunits: `baseCostMinor := (soldQtyMilli*rule.SettlementPriceMinor + 500) / 1000`.
  - Retail value in milliunits: `retailValueMinor := (soldQtyMilli*rule.RetailPriceMinor + 500) / 1000`.
  - Removed `math` package import.
- **Why**: Exact integer settlement for consignment inventory sales and commission splits.

### 1.8 `backend/internal/supplier/service.go` & `models.go`
- **What**: 
  - Converted tiered volume discounts to basis points: `totalDiscountBps := int64(rule.DiscountPercent * 100.0)`, `discountAmount := (req.BasePriceMinor*totalDiscountBps + 5000) / 10000`.
  - Converted catch-weight pricing to integer grams and minor units: `actualWeightGrams := int64(actualWeightKg*1000.0 + 0.5)`, `actualPriceMinor := (actualWeightGrams*unitPricePerKgMinor + 500) / 1000`.
  - Added `VatRateBps int64` to `Product` model with backward-compatible `GetVatRateBps()`.
  - Removed `math` package imports.
- **Why**: Zero floating-point drift in supplier catalog pricing and catch-weight items.

### 1.9 `backend/internal/payout/calculator.go` & `rails.go`
- **What**:
  - Converted reserve withholdings to basis points: `reserveWithheld := (eligible*reserveBps + 5000) / 10000`.
  - Refactored 1C Client-Bank payment order file export in `rails.go` to format currency using integer division and modulo (`%d.%02d`) without floating-point conversion.
  - Removed `math` package import.
- **Why**: Prevents float precision issues during bank wire export and payout reconciliation.

### 1.10 `backend/internal/api/handlers_supplier.go`
- **What**: Updated `OnboardingProductPayload`, `handleSupplierOnboardingCreateProduct`, and `handleSupplierUpdateProduct` to support `vat_rate_bps` while maintaining backward compatibility with legacy `vat_rate`.
- **Why**: Full cross-tier contract alignment for basis points tax rates.

---

## 2. Domain State Machine & Concurrency Purity (Requirement R1.2)

### 2.1 `backend/internal/api/handlers_fleet_driver.go`
- **What**: Refactored `handleOrderComplete`:
  - Replaced naive direct SQL `UPDATE orders SET status = 'COMPLETED'` with domain state machine call:
    `s.orderSvc.TransitionStatus(r.Context(), orderID, models.StatusDelivered, opts)`.
  - Added transition recovery: handles moving from `IN_TRANSIT` through `ARRIVED` to `DELIVERED`.
  - Added proper error handling and propagation instead of silent ignore.
  - Added missing imports: `errors`, `fmt`, `models`, `order`.
- **Why**: Eliminates illegal status jumps, bypassing state transitions, and unvalidated status updates.

### 2.2 `backend/internal/epod/repository.go`
- **What**:
  - In `UpdateStopStatus`: added transition guard:
    `UPDATE orders SET status = 'DELIVERED', updated_at = NOW() WHERE order_id = $1 AND status NOT IN ('CANCELLED', 'DELIVERED')`.
  - Handled errors on all `tx.Exec` calls (vehicle active status transition, payment leg creation, manifest completion check).
  - Fixed syntax error (duplicate closing brace).
- **Why**: Prevents terminal CANCELLED orders from being overwritten as DELIVERED, and guarantees atomic transactional integrity.

### 2.3 `backend/internal/wmsops/repository.go`
- **What**: In `UpdateReplenishmentInsightStatus`:
  - Eliminated TOCTOU (Time-of-Check to Time-of-Use) race condition: replaced separate `SELECT status` followed by `UPDATE` with a single atomic conditional update:
    ```sql
    UPDATE warehouse_replenishment_insights
    SET status = $2, reason_code = $3, target_po_id = $4, actioned_at = NOW()
    WHERE insight_id = $1 AND status IN ('OPEN', 'PENDING')
    ```
  - Checked `tag.RowsAffected() == 0`, returning `ErrInsightAlreadyActioned` or `ErrInsightNotFound`.
  - Added concurrent race test `TestConcurrentUpdateReplenishmentInsightStatus` in `repository_test.go` verifying 20 concurrent goroutines racing on the same insight.
- **Why**: Guarantees zero double-actioning of replenishment purchase order insights under high concurrency.

### 2.4 `backend/internal/api/handlers_payment.go`
- **What**:
  - In `handleProcessHandover`: wrapped all database operations in `s.pool.RunInTx`, checked every `tx.Exec` error (order delivery, cash payment leg, card payment leg, retailer debt, credit account update, retailer wallet, ledger entries, and outbox event), and returned HTTP 500 on transaction rollback.
  - In GlobalPay webhook handler: checked every `tx.Exec` error on order payment status, payment leg, ledger journal entry, postings, and outbox event.
- **Why**: Eliminates silent data loss where payment leg or debt insertion failed but the HTTP response returned success.

### 2.5 `backend/internal/fleet/repository.go`
- **What**:
  - In `ReleaseAssignment`: checked errors when setting driver offline and vehicle standby.
  - In `SwapVehicle`: checked errors when redirecting open manifests and orders to the replacement vehicle.
  - In `AssignReliefDriver`: checked errors when relieving the old driver and assigning the new driver to manifests/orders.
  - In `SplitPayment`: checked transaction begin, each `tx.Exec`, `tx.QueryRow`, transition guard on orders (`status NOT IN ('CANCELLED', 'DELIVERED')`), and `tx.Commit`.
  - In `OrderDeliver`: checked errors on manifest stop and order delivery with status guards.
- **Why**: Ensures all vehicle and driver mutations fail-closed and rollback cleanly on error.

### 2.6 `backend/internal/ewm/` (Repository Quarantine & Postgres Persistence)
- **What**:
  - Created `backend/internal/ewm/repository.go` implementing `PostgresRepository` with full CRUD for `sku_velocity_assignments` and `cross_dock_allocations` tables on PostgreSQL 16.
  - Added fail-closed constructors: `NewPostgresRepository` (returns error on nil pool) and `MustNewPostgresRepository` (panics on nil pool).
  - Quarantined `MemoryRepository` into `backend/internal/ewm/repository_mock.go` strictly for isolated unit testing and smokecheck verification.
  - In `backend/internal/ewm/service.go`: removed `MemoryRepository`, updated `NewService` to instantiate `PostgresRepository` when pool is provided and fail-closed when both are nil. Added `NewTestService`.
  - In `backend/internal/api/router.go`: wired `ewm.MustNewPostgresRepository(pool)` when pool is non-nil.
  - In `backend/cmd/smokecheck/main.go`: updated to use `ewm.NewTestService(nil)`.
  - In `backend/internal/ewm/ewm_test.go`: added `TestNewPostgresRepository_FailClosedOnNilPool` and `TestNewService_FailClosedOnNilPoolAndRepo`.
- **Why**: Complies with the Zero Mock Data Policy, ensuring production uses PostgreSQL 16 persistence while unit tests and smokechecks remain isolated.
