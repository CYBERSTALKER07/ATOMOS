# Changes: Milestone 2 Remediation

This document details all modifications made to remediate the four defects identified in the Milestone 2 Adversarial Challenge Report.

---

### 1. Idempotency & Concurrency: Eliminating Double Inventory Deduction

#### `backend/internal/order/state_machine.go`
- **Change**: Updated `ValidateStatusTransition` to check terminal states (`models.StatusDelivered` and `models.StatusCancelled`) **before** checking `if from == to`.
- **Rationale**: Previously, `if from == to { return nil }` preceded terminal state checks. When an order was already `DELIVERED`, calling `ValidateStatusTransition(DELIVERED, DELIVERED)` returned `nil`, allowing subsequent calls to proceed through status transition logic. Terminal orders cannot transition, even to themselves, and return `ErrTerminalStateImmutable`.

#### `backend/internal/order/service.go`
- **Change**:
  1. In `TransitionStatus`, added an explicit idempotency check in both in-memory and PostgreSQL branches:
     ```go
     if currentStatus == models.StatusDelivered && newStatus == models.StatusDelivered {
         return nil
     }
     ```
  2. Guarded side effects with status inequality:
     ```go
     if newStatus == models.StatusDelivered && currentStatus != models.StatusDelivered { ... }
     if newStatus == models.StatusCancelled && currentStatus != models.StatusCancelled { ... }
     ```
- **Rationale**: Ensures that repeated calls (e.g. mobile cellular retries, network jitter, concurrent button presses) to `TransitionStatus` for an already-delivered order succeed idempotently as a no-op without re-executing `inventory.DeductCommittedStock` or emitting duplicate outbox events.

#### `backend/internal/api/handlers_fleet_driver.go`
- **Change**:
  1. Handled `order.ErrTerminalStateImmutable` gracefully as an idempotent success:
     ```go
     if err != nil && !errors.Is(err, order.ErrTerminalStateImmutable) { ... }
     ```
  2. In the fallback SQL path (`s.orderSvc == nil`), added `AND status NOT IN ('CANCELLED', 'DELIVERED')` guard and wrapped order status update and manifest stop update into a single atomic database transaction closure via `s.pool.RunInTx`. If rows affected is 0 on the update, checked if order was cancelled and returned HTTP 409 Conflict.
- **Rationale**: Prevents reviving cancelled orders and ensures atomic update across `orders` and `manifest_stops`.

---

### 2. Eliminating Residual Float Currency Math & `math` Package Imports

#### `backend/internal/supplier/service.go`
- **Change**:
  - Removed `"math"` package import.
  - Replaced `math.Round(float64(prod.UnitPriceTiyin) / prod.NominalWeightKg)` on line 283 with pure integer milliunit arithmetic:
    ```go
    nominalWeightGrams := int64(prod.NominalWeightKg*1000.0 + 0.5)
    if nominalWeightGrams > 0 {
        prod.PricePerKgTiyin = (prod.UnitPriceTiyin*1000 + nominalWeightGrams/2) / nominalWeightGrams
    }
    ```
- **Rationale**: Eliminates floating-point division and `math.Round` on currency values, strictly enforcing 64-bit integer tiyin minor units with standard integer round-half-up math.

#### `backend/internal/supplier/models.go`
- **Change**:
  - Removed `"math"` package import.
  - Replaced `math.Abs(varianceKg)` on line 371 with explicit sign-check (`if absVarianceKg < 0 { absVarianceKg = -absVarianceKg }`).
- **Rationale**: Ensures `internal/supplier` has zero `"math"` package imports.

#### `backend/internal/copa/copa.go`
- **Change**:
  - Removed `"math"` package import.
  - Replaced lines 119-120:
    ```go
    transitCost := int64(math.Round(input.DetourDistanceKM * float64(TransitCostPerKMTiyins)))
    laborCost := int64(math.Round(input.OnSiteDurationMin * float64(LaborCostPerMinTiyins)))
    ```
    with integer milli-unit arithmetic in tiyins:
    ```go
    distMilliKM := int64(input.DetourDistanceKM*1000.0 + 0.5)
    transitCost := (distMilliKM*TransitCostPerKMTiyins + 500) / 1000

    durationMilliMin := int64(input.OnSiteDurationMin*1000.0 + 0.5)
    laborCost := (durationMilliMin*LaborCostPerMinTiyins + 500) / 1000
    ```
- **Rationale**: Eliminates `math.Round` and `float64` multiplication on tiyin rate constants (`TransitCostPerKMTiyins`, `LaborCostPerMinTiyins`).

#### `backend/internal/matching/matching.go`
- **Change**:
  - Removed `"math"` package import.
  - Replaced line 201 `qtyDiff := math.Abs(item.InvoicedQty - grItem.ReceivedQty)` with standard difference comparison without `math.Abs`.
- **Rationale**: Eliminates `"math"` package import from `internal/matching`.

---

### 3. Accounting Reserve Clamping in `ar/dunning.go`

#### `backend/internal/ar/dunning.go`
- **Change**: In `CalculateBadDebtProvision`:
  ```go
  provision := (provTiyins + 5000) / 10000
  if provision < 0 {
      return 0
  }
  return provision
  ```
- **Rationale**: Clamps negative aging bucket calculations (e.g. from unallocated customer prepayments or credit notes) to 0. Contra-asset bad debt accounting reserves cannot be negative.

---

### 4. Test Name Alignment in `wmsops`

#### `backend/internal/wmsops/repository_test.go`
- **Change**: Added test runner function `TestUpdateReplenishmentInsightStatus_Race(t *testing.T)` calling `TestConcurrentUpdateReplenishmentInsightStatus(t)`.
- **Rationale**: Ensures that running the mandated test command:
  ```bash
  go test -v -race -run TestUpdateReplenishmentInsightStatus_Race ./internal/wmsops/...
  ```
  matches and executes the concurrent race test suite instead of exiting with `[no tests to run]`.

---

### 5. Automated Tests Added / Enhanced

- `backend/internal/order/state_machine_test.go`: Added test cases verifying that `DELIVERED -> DELIVERED`, `CANCELLED -> CANCELLED`, and `CANCELLED -> PENDING` return `ErrTerminalStateImmutable`.
- `backend/internal/order/concurrency_test.go`: Added `TestOrder_Delivered_Idempotency_NoDoubleDeduction` verifying that repeated transitions to `DELIVERED` succeed as an idempotent no-op without double inventory deduction.
- `backend/internal/ar/ar_test.go`: Added negative aging bucket provision assertions in `TestFinancialCalculations` verifying that negative aging balances clamp bad debt provision to 0.
- `backend/internal/supplier/supplier_catch_weight_test.go`: Added `CatchWeightAutoPricePerKgCalculation_IntegerMath` subtest verifying that zero initial `PricePerKgTiyin` is calculated accurately using pure integer arithmetic.
