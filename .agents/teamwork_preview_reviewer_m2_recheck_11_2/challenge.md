# Adversarial Challenge Report: Milestone 2 Remediation Re-Check

**Reviewer & Adversarial Critic**: `teamwork_preview_reviewer_m2_recheck_11_2`  
**Target Codebase**: `pegasus.x` (Sovereign Lean Single-Tenant National Operating Core)  
**Parent Task**: Milestone 2 Remediation Verification (`d877571c-b5bd-4489-a1cb-441c991bb03d`)  
**Date**: 2026-09-23  

---

## 1. Challenge Summary

**Overall Risk Assessment**: **LOW**  
**Integrity Audit**: **CLEAN (0 Violations)**  
- Zero hardcoded test results embedded in production or test code.
- Zero facade/mock implementations in production packages.
- Zero shortcuts or bypasses of intended business logic.
- All test runs executed live and independently verified with `-race` enabled.

---

## 2. Adversarial Challenges & Stress Testing

### Challenge 1 (High Priority): Double Inventory Deduction & Outbox Event Duplication on Mobile Retries
- **Assumption Challenged**: Calling `TransitionStatus` with `models.StatusDelivered` on an already `DELIVERED` order must be safe, idempotent, and cause zero side effects.
- **Attack Scenario**: 
  1. A driver completes delivery on the mobile app. The network drops the TCP ACK.
  2. The mobile client retries `POST /v1/fleet/driver/orders/{order_id}/complete` 3 times concurrently.
  3. If idempotency guards are missing or weak, `inventory.DeductCommittedStock` could execute multiple times, deducting physical on-hand stock repeatedly and publishing duplicate `order.status_updated` events to the outbox relay and Redis Streams.
- **Code Path Analyzed**:
  - `backend/internal/order/service.go:883-885` (PostgreSQL path):
    ```go
    // Idempotency check: if order is already in target status DELIVERED, treat as idempotent success.
    // Crucially, DO NOT re-execute stock deduction or emit duplicate outbox events.
    if currentStatus == models.StatusDelivered && newStatus == models.StatusDelivered {
        return nil
    }
    ```
  - `backend/internal/order/service.go:859-862` (In-memory path):
    ```go
    if ord.Status == models.StatusDelivered && newStatus == models.StatusDelivered {
        s.mu.Unlock()
        return nil
    }
    ```
  - `backend/internal/order/service.go:925`:
    ```go
    if newStatus == models.StatusDelivered && currentStatus != models.StatusDelivered {
        // DeductCommittedStock loop
    }
    ```
  - `backend/internal/order/state_machine.go:42-49`:
    ```go
    if from == models.StatusDelivered {
        return fmt.Errorf("%w: DELIVERED order cannot be modified", ErrTerminalStateImmutable)
    }
    if from == models.StatusCancelled {
        return fmt.Errorf("%w: CANCELLED order cannot be modified", ErrTerminalStateImmutable)
    }
    ```
- **Stress-Test Analysis & Findings**:
  - `SELECT ... FOR UPDATE` acquires row-level pessimistic lock on the order row.
  - The first transaction transitions `ARRIVED -> DELIVERED`, executes `DeductCommittedStock` once, updates the order status, emits the outbox event, and commits.
  - The second concurrent request acquires the lock, reads `currentStatus = 'DELIVERED'`, and immediately hits line 883.
  - It returns `nil` directly, bypassing `ValidateStatusTransition`, bypassing `DeductCommittedStock`, bypassing order row update, and bypassing `outbox.Emit`.
  - **Result**: **PASS**. Zero double deduction. Zero phantom outbox events.

---

### Challenge 2 (High Priority): Floating-Point Residuals and `math` Package Currency Leaks
- **Assumption Challenged**: All pricing, financial margins, VAT calculations, and penalty fees must strictly utilize 64-bit integer tiyin minor units with standard integer round-half-up math. Zero `math` package imports and zero float casts on currency amounts.
- **Attack Scenario**:
  - Precision drift when calculating `PricePerKgTiyin` from catch-weight nominal kilograms.
  - Floating-point multiplication in COPA route profitability for detour kilometers and on-site duration.
  - Float casts on monetary amounts in 3-way matching and Soliq electronic factura generation.
- **Grep & AST Analysis**:
  - `grep -rn "\"math\"" internal/supplier/ internal/copa/ internal/matching/ internal/soliq/`: **0 matches**.
  - `grep -rn "math.Round" internal/supplier/ internal/copa/ internal/matching/ internal/soliq/`: **0 matches**.
  - `grep -rn "float64(" internal/supplier/service.go internal/copa/copa.go internal/matching/matching.go internal/soliq/efactura.go`:
    - `internal/supplier/service.go:1084, 1087`: Percentage display strings for preview margin label (`marginLabel = fmt.Sprintf("-%.1f%% vs list price", pct)`). Zero currency calculation.
    - `internal/copa/copa.go:140, 141`: Percentage margin KPI metrics (`grossMarginPct`, `netMarginPct`). Zero currency calculation.
    - `internal/matching/matching.go`: Physical item quantities (`OrderedQty`, `ReceivedQty`, `InvoicedQty`), converted via integer milliunits (`* 1000 + 0.5`) before multiplying integer tiyin unit prices.
    - `internal/soliq/efactura.go`: **0 float usages**. Pure 64-bit integer tiyin minor units with `(lineSubtotal*int64(it.VATPercent) + 50) / 100`.
- **Result**: **PASS**. Zero residual floats on money. Pure integer minor units verified.

---

### Challenge 3 (Minor Finding / Defense-in-Depth): Terminal State Handling for Cancelled Orders in Driver Handler
- **Assumption Challenged**: A driver cannot complete an order that has been `CANCELLED` post-dispatch.
- **Attack Scenario**:
  - An order is cancelled by an administrator while the truck is in transit (`IN_TRANSIT -> CANCELLED`).
  - The driver arrives at the retailer and submits `POST /v1/fleet/driver/orders/{order_id}/complete`.
- **Observation in `handlers_fleet_driver.go:1017`**:
  ```go
  if err := s.orderSvc.TransitionStatus(r.Context(), orderID, models.StatusDelivered, opts); err != nil {
      if errors.Is(err, order.ErrInvalidStatusTransition) {
          _ = s.orderSvc.TransitionStatus(r.Context(), orderID, models.StatusArrived, opts)
          err = s.orderSvc.TransitionStatus(r.Context(), orderID, models.StatusDelivered, opts)
      }
      if err != nil && !errors.Is(err, order.ErrTerminalStateImmutable) {
          response.Error(w, http.StatusBadRequest, "invalid_status_transition", err.Error())
          return
      }
  }
  ```
  - In `orderSvc.TransitionStatus`:
    `currentStatus` is `CANCELLED`.
    Line 883 condition is false.
    Line 887 calls `ValidateStatusTransition(CANCELLED, DELIVERED)` which returns `ErrTerminalStateImmutable`.
    The transaction rolls back, preserving the database order status as `CANCELLED`.
  - In `handlers_fleet_driver.go:1017`:
    `!errors.Is(err, order.ErrTerminalStateImmutable)` evaluates to `false`.
    The handler suppresses the error and proceeds to line 1052: `UPDATE manifest_stops SET status = 'COMPLETED' ...` and returns `HTTP 200 OK`.
  - In the fallback SQL path (`s.orderSvc == nil`, lines 1033-1045):
    The handler explicitly checks if `currentStatus == "CANCELLED"` and returns `HTTP 409 Conflict ("cannot complete a cancelled order")`.
- **Blast Radius**:
  - Low. The `orders` table is not corrupted (it remains `CANCELLED`). However, the `manifest_stops` row is marked `COMPLETED` and the HTTP caller receives `200 OK` rather than an explicit `409 Conflict`.
- **Mitigation Recommendation**:
  - In `handlers_fleet_driver.go`, either inspect current order status or query `GetOrder` before completing, or refine the error check to only treat `DELIVERED` as idempotent success rather than broad suppression of `ErrTerminalStateImmutable`.

---

### Challenge 4 (Medium Priority): Negative Balance Clamping in Bad Debt Provisions
- **Assumption Challenged**: Under accounting standards, an allowance for doubtful accounts (contra-asset) cannot become negative if a customer has unapplied credits or prepayments.
- **Verification**:
  - `backend/internal/ar/dunning.go:142`:
    ```go
    provision := (provTiyins + 5000) / 10000
    if provision < 0 {
        return 0
    }
    return provision
    ```
  - Verified with unit tests in `backend/internal/ar/ar_test.go` (`TestFinancialCalculations`), confirming that negative bucket balances return a clamped provision of `0`.
- **Result**: **PASS**.

---

### Challenge 5 (Medium Priority): Replenishment Insight Concurrency Race Condition
- **Assumption Challenged**: `UpdateReplenishmentInsightStatus` must safely serialize concurrent actions and prevent race conditions.
- **Verification**:
  - Ran `go test -v -race -count=1 -run TestUpdateReplenishmentInsightStatus_Race ./internal/wmsops/...`.
  - 20 concurrent workers raced to approve/reject the same insight.
  - Invariant verified: exactly 1 worker succeeds, 19 receive `ErrInsightAlreadyActioned`.
  - Zero data races detected under `-race`.
- **Result**: **PASS**.

---

## 3. Stress Test Results Summary

| Test Suite / Target | Command | Result | Duration | Race Conditions |
| :--- | :--- | :--- | :--- | :--- |
| `internal/wmsops` | `go test -v -race -count=1 -run TestUpdateReplenishmentInsightStatus_Race ./internal/wmsops/...` | **PASS** | 1.517s | 0 |
| `internal/order/...` | `go test -v -race -count=1 ./internal/order/...` | **PASS** | 1.323s | 0 |
| `internal/api/...` | `go test -v -race -count=1 ./internal/api/...` | **PASS** | 45.667s | 0 |
| `internal/supplier/...` | `go test -v -race -count=1 ./internal/supplier/...` | **PASS** | 2.895s | 0 |
| `internal/copa/...` | `go test -v -race -count=1 ./internal/copa/...` | **PASS** | 1.285s | 0 |
| `internal/ar/...` | `go test -v -race -count=1 ./internal/ar/...` | **PASS** | 1.289s | 0 |
| `internal/matching/...` | `go test -v -race -count=1 ./internal/matching/...` | **PASS** | 1.451s | 0 |
| `internal/soliq/...` | `go test -v -race -count=1 ./internal/soliq/...` | **PASS** | 2.066s | 0 |

---

## 4. Adversarial Conclusion

The remediation performed by Worker 2 successfully resolves all core defects from the Milestone 2 review. The system enforces strict 64-bit integer tiyin minor unit arithmetic, eliminates double inventory deduction, prevents duplicate outbox events, and exhibits zero race conditions across all evaluated modules.
