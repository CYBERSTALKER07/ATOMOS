# Track 2 Audit: Order Lifecycle, Spanner Transactions & State Machines

**Target Codebase**: `apps/backend-go` (and `pegasusX/apps/backend-go`)  
**Auditor**: Codebase Explorer (Track 2 Specialist)  
**Date**: 2026-08-30  
**Status**: COMPLETE  

---

## Executive Summary

A comprehensive, line-by-line architectural and concurrency audit of the PegasusX Go backend was performed, covering the order lifecycle, state machine transitions, Google Cloud Spanner Read-Write transactions, concurrency control, outbox event atomicity, multi-supplier checkouts, driver doorstep edge cases, cancellations, refunds, and fiscal settlement gates.

The audit verified that while the core Spanner transactional design adheres to the outbox pattern inside `ReadWriteTransaction` closures, several **critical and high-severity vulnerabilities** exist in edge transitions, line-item amendments, multi-user authorization, inventory reservation tracking, and build integrity.

---

## Table of Audit Findings

| ID | Category | Severity | Exact Location (`file:line`) | Title / Flaw Summary |
|---|---|---|---|---|
| **F-01** | Build / Runtime | **CRITICAL** | `order/service.go:1321` | Compiler error: Undefined identifier `StatusDraft` breaks `order` build |
| **F-02** | Inventory / Spanner | **CRITICAL** | `order/supplier_ops.go:466-473` | Permanent inventory reservation leak in supplier early complete approval |
| **F-03** | Inventory / Spanner | **HIGH** | `order/warehouse_ops.go:229-248` | Missing inventory reservation reconciliation on warehouse pre-order line edit |
| **F-04** | Inventory / Finance | **HIGH** | `order/negotiation.go:291-311` | Unreconciled inventory & trapped credit hold on quantity negotiation approval |
| **F-05** | Audit / Transactions | **HIGH** | `order/status_timeline.go:62-76` | Non-atomic out-of-transaction write of `OrderStatusTransitions` |
| **F-06** | Auth / Multi-User | **HIGH** | `order/status_timeline.go:201`, `order/status_context.go:38` | Retailer staff authorization break (`claims.Subject` vs `RetailerID` org ID) |
| **F-07** | Data Loss / Splitting | **HIGH** | `order/service.go:1511-1516` | Silent error swallowing during backorder creation in partial fulfillment split |
| **F-08** | Spanner Concurrency | **MEDIUM** | `order/fiscal.go:872-876`, `order/settlement_hardening.go:195-210` | Non-transactional snapshot reads (`spannerClient.Single`) inside RW transaction in `ForceCompleteOrder` |
| **F-09** | Spanner Concurrency | **MEDIUM** | `order/partial_offload.go:320-328` | Blind unversioned Spanner `Apply` overwrite on `Orders` in partial offload |
| **F-10** | State Machine | **MEDIUM** | `order/state_machine.go:14-81` | State machine ignores all `TransitionOpts` (actor, supervisor token, photo URL) |
| **F-11** | Distributed Saga | **MEDIUM** | `order/unified_checkout.go:364-390` | Multi-supplier checkout vulnerable to crash between child order iterations without background reconciler |
| **F-12** | Finance / AR | **MEDIUM** | `order/worker_shop_closed.go:240-250` | Shop-closed worker fails to retry AR invoice creation after post-commit failure |

---

## Detailed Findings

### F-01: Compiler Failure Due to Undefined Identifier `StatusDraft`
- **Location**: `pegasusX/apps/backend-go/order/service.go:1321`
- **Severity**: **CRITICAL (Build Blocker / Runtime Crash)**
- **Observation**:
  ```go
  // order/service.go:1320-1323
  if source == OrderSourceAutoOrder {
      status = StatusDraft
  }
  ```
  Running `go build ./order/...` fails immediately with:
  `order/service.go:1321:13: undefined: StatusDraft`
- **Logic Chain**:
  In `order/service.go:92`, the codebase defines `ConfirmationStatusDraft ConfirmationStatus = "DRAFT"`, but does not define `StatusDraft Status`. In `state_machine.go`, the order status enum does not contain `DRAFT` (valid statuses start at `PENDING`, `SCHEDULED`, or `BACKORDERED`). Writing `status = StatusDraft` references a non-existent identifier in the package.
- **Blast Radius**:
  The entire `order` package cannot compile. Any service importing `order` fails to build.
- **Remediation**:
  Assign `confirmation = ConfirmationStatusDraft` instead of `status = StatusDraft`, keeping `status = StatusPending` or `StatusScheduled`.

---

### F-02: Permanent Inventory Reservation Leak in Supplier Early Complete Approval
- **Location**: `pegasusX/apps/backend-go/order/supplier_ops.go:466-473`
- **Severity**: **CRITICAL (Financial / Inventory Drift)**
- **Observation**:
  ```go
  // order/supplier_ops.go:466-473
  mutations := []*spanner.Mutation{
      spanner.UpdateMap("Orders", map[string]any{
          "OrderId":   orderID,
          "Status":    string(StatusCancelled),
          "Version":   version + 1,
          "UpdatedAt": now.UTC(),
      }),
  }
  ```
- **Logic Chain**:
  When a supplier approves an early route completion request, any remaining incomplete orders on the route are transitioned to `StatusCancelled`. However, `HandleApproveEarlyComplete` writes a direct `spanner.UpdateMap("Orders", ...)` mutation without invoking `ReleaseReservationsForOrderInTxn` or `ReleaseReservationsFromOrderFields`. Consequently, inventory reservations in `SupplierInventoryV2.QuantityReserved` and `OrderStockReservationMarkers` / `StockLots` are never decremented.
- **Blast Radius**:
  Available inventory (`QuantityOnHand - QuantityReserved`) permanently decreases on each early completion, causing physical inventory to be ghost-locked and preventing subsequent sales of on-hand goods.
- **Remediation**:
  Load the order line items and source in the transaction and call `ReleaseReservationsFromOrderFields(ctx, txn, supplierID, warehouseID, orderSource, lineItemsRaw)` before buffering the `Orders` mutation.

---

### F-03: Missing Inventory Reservation Reconciliation on Warehouse Pre-order Line Edit
- **Location**: `pegasusX/apps/backend-go/order/warehouse_ops.go:229-248`
- **Severity**: **HIGH (Inventory Inconsistency / Overselling)**
- **Observation**:
  ```go
  // order/warehouse_ops.go:229-248
  lineItems, total, err := s.normalizeAndQuoteLineItems(ctx, req.LineItems, nil)
  ...
  current.LineItems = lineItems
  current.TotalMinor = total
  current.UpdatedAt = s.now()
  if err := s.repo.UpdateOrder(ctx, current, nil, func(txn outbox.TxnBuffer) error {
      return emitPreorderEvent(ctx, txn, events.EventPreOrderEdited, current, "WAREHOUSE_ADMIN", actorID)
  }); err != nil { ... }
  ```
- **Logic Chain**:
  In `repository_spanner.go:228-246`, `UpdateOrder` only releases reservations if `o.Status == StatusCancelled`. When warehouse operators amend line items on scheduled preorders, the change in quantities is saved to `Orders.LineItemsJson`, but `reconcilePreorderReservationsInTxn` is never invoked. (In contrast, the retailer preorder edit in `preorder_service.go:195` properly invokes `s.updatePreorderLines`).
- **Blast Radius**:
  If quantities are increased, excess stock is not reserved (leading to overselling and inventory exhaustion). If quantities are decreased, old reservations remain locked indefinitely.
- **Remediation**:
  Update `WarehouseEditPreorder` to call `s.updatePreorderLines(ctx, current, lineItems, ...)` in `order/preorder_inventory.go`.

---

### F-04: Unreconciled Inventory & Trapped Credit Hold on Quantity Negotiation Approval
- **Location**: `pegasusX/apps/backend-go/order/negotiation.go:291-311`
- **Severity**: **HIGH (Inventory Leak & Trapped Credit)**
- **Observation**:
  ```go
  // order/negotiation.go:304-310
  mutations = append(mutations, spanner.UpdateMap("Orders", map[string]any{
      "OrderId":       orderID,
      "LineItemsJson": updatedRaw,
      "TotalMinor":    total,
      "Version":       version + 1,
      "UpdatedAt":     now.UTC(),
  }))
  ```
- **Logic Chain**:
  When a supplier approves a driver-submitted quantity negotiation, `Orders` is updated with reduced line item quantities and total minor amount. However:
  1. No inventory reservation release is issued for the delta between original and negotiated quantities.
  2. No credit reservation adjustment is made against `OrderCreditReservations` / `RetailerCreditProfiles`.
  3. `OriginalTotalMinor` is not preserved if not already set.
- **Blast Radius**:
  Ghost inventory holds remain in `SupplierInventoryV2`, and the retailer's available credit line is artificially reduced by the original order amount rather than the negotiated amount.
- **Remediation**:
  Calculate line item quantity differences and execute `ReleaseReservationsInTxn` for rejected quantities; adjust credit reservations via `s.credit.AdjustReserveInTxn`.

---

### F-05: Non-Atomic Out-of-Transaction Write of `OrderStatusTransitions`
- **Location**: `pegasusX/apps/backend-go/order/status_timeline.go:62-76`
- **Severity**: **HIGH (Audit Trail Loss & Dual-Write Inconsistency)**
- **Observation**:
  ```go
  // order/status_timeline.go:62-76
  _, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{
      spanner.InsertMap("OrderStatusTransitions", map[string]any{ ... }),
  })
  ```
- **Logic Chain**:
  `persistStatusTransition` is called from `recordStatusTransitionFromOrder` inside `afterOrderMutation`, which executes *after* the `UpdateOrder` Spanner ReadWriteTransaction has committed. If the server terminates, encounters a network hiccup, or fails Spanner `Apply`, the order status is mutated but the transition log row is permanently lost.
- **Blast Radius**:
  Breaks legal compliance, order history tracking, and causes visual gaps in retailer and supplier timeline UIs.
- **Remediation**:
  Inject the `OrderStatusTransitions` insert mutation directly into the `ReadWriteTransaction` mutation buffer inside `UpdateOrder` and `UpdateOrderWithTxn`.

---

### F-06: Retailer Staff Authorization Break (`claims.Subject` vs `RetailerID` Org ID)
- **Location**: `pegasusX/apps/backend-go/order/status_timeline.go:201` and `pegasusX/apps/backend-go/order/status_context.go:38`
- **Severity**: **HIGH (Broken Retailer Multi-User Access)**
- **Observation**:
  ```go
  // order/status_timeline.go:201 & status_context.go:38
  if claims.Subject != o.RetailerID {
      return ErrOrderForbidden
  }
  ```
- **Logic Chain**:
  Under PegasusX multi-user auth (B3 M-P0-4), `claims.Subject` contains the user's ID (`usr_...`), whereas `o.RetailerID` stores the organization/store ID (`ret_...`). The helper `auth.ResolveRetailerOrgID(claims)` is designed to extract the organization ID from claims. Directly comparing `claims.Subject == o.RetailerID` fails for all staff users whose `Subject != OrgID`.
- **Blast Radius**:
  Retailer staff, store managers, and clerks receive HTTP 403 Forbidden when accessing `/v1/order/{id}/timeline` and `/v1/order/{id}/status-context`.
- **Remediation**:
  Replace `claims.Subject != o.RetailerID` with:
  ```go
  if auth.ResolveRetailerOrgID(claims) != o.RetailerID {
      return ErrOrderForbidden
  }
  ```

---

### F-07: Silent Error Swallowing During Backorder Creation in Partial Fulfillment Split
- **Location**: `pegasusX/apps/backend-go/order/service.go:1511-1516`
- **Severity**: **HIGH (Silent Data Loss & Unfulfilled Orders)**
- **Observation**:
  ```go
  // order/service.go:1511-1516
  if len(invPlan.Backorder) > 0 {
      go func() {
          _, bErr := s.createBackorderOrder(context.Background(), retailerID, orderID, req, invPlan.Backorder)
          if bErr != nil {
              s.log.WarnContext(ctx, "failed to create backorder order", "err", bErr, "order_id", orderID)
          }
      }()
  }
  ```
- **Logic Chain**:
  When an order is partially fulfilled and the remaining items must be split into a backorder, `createBackorderOrder` runs asynchronously in a detached goroutine. If it fails (due to DB constraint, timeout, or credit breach), the error is only logged with `WarnContext` and completely swallowed. The retailer receives a 200 OK response for the primary order, while backordered items vanish from the system without notification or compensation.
- **Blast Radius**:
  Retailers lose ordered stock lines without visibility; store replenishment schedules fail silently.
- **Remediation**:
  Either create the backorder order synchronously or emit an outbox event `EventBackorderCreationRequested` processed by a durable outbox consumer with retry/alert capabilities.

---

### F-08: Non-Transactional Snapshot Reads Inside Spanner RW Transaction in `ForceCompleteOrder`
- **Location**: `pegasusX/apps/backend-go/order/fiscal.go:872-876` and `pegasusX/apps/backend-go/order/settlement_hardening.go:195-210`
- **Severity**: **MEDIUM (Concurrency Lock Bypass / Stale Read in RW Txn)**
- **Observation**:
  ```go
  // order/fiscal.go:872-876 (inside ReadWriteTransaction closure)
  if err := s.AssertMoneyCoversDelivery(ctx, orderID, 0, 0); err != nil {
      delivered, _ := s.getDeliveredGrossMinor(ctx, orderID)
      paid, _ := s.getCapturedPaymentMinor(ctx, orderID)
      exceptions, _ := s.getExceptionsTotalMinor(ctx, orderID)
      ...
  ```
- **Logic Chain**:
  `s.AssertMoneyCoversDelivery`, `s.getDeliveredGrossMinor`, `s.getCapturedPaymentMinor`, and `s.getExceptionsTotalMinor` all execute queries against `s.spannerClient.Single()`. Calling `Single()` inside a `ReadWriteTransaction` creates a separate non-transactional read that does not acquire shared locks in the RW transaction.
- **Blast Radius**:
  Concurrent payment captures or settlement exception writes can interleave with force-complete, leading to dirty/stale shortfall calculations and duplicate settlement exception rows.
- **Remediation**:
  Use `AssertMoneyCoversDeliveryTxn`, `getCapturedPaymentMinorTxn`, and `getExceptionsTotalMinorTxn`, passing `txn` directly into all read functions inside the transaction closure.

---

### F-09: Blind Unversioned Spanner `Apply` Overwrite on `Orders` in Partial Offload
- **Location**: `pegasusX/apps/backend-go/order/partial_offload.go:320-328`
- **Severity**: **MEDIUM (Race Condition / Optimistic Concurrency Bypass)**
- **Observation**:
  ```go
  // order/partial_offload.go:320-328
  if s.spannerClient != nil {
      _, _ = s.spannerClient.Apply(ctx, []*spanner.Mutation{
          spanner.UpdateMap("Orders", map[string]any{
              "OrderId":         current.OrderID,
              "PartialDelivery": current.PartialDelivery,
              "UpdatedAt":       now.UTC(),
          }),
      })
  }
  ```
- **Logic Chain**:
  `UpdateOrderWithTxn` on line 274 already committed the order state and incremented `Version`. The subsequent unversioned `spannerClient.Apply` issues a blind `UpdateMap` outside the transaction without version checks, overwriting `UpdatedAt` and potentially clobbering concurrent driver updates.
- **Blast Radius**:
  Can overwrite concurrent status updates and causes timestamp jitter.
- **Remediation**:
  Remove lines 320-328; `PartialDelivery` is already persisted by `UpdateOrderWithTxn`.

---

### F-10: State Machine Ignores All `TransitionOpts`
- **Location**: `pegasusX/apps/backend-go/order/state_machine.go:14-81`
- **Severity**: **MEDIUM (Security & Workflow Validation Bypass)**
- **Observation**:
  ```go
  // order/state_machine.go:14-81
  func ValidateStatusTransition(from, to string, opts TransitionOpts) error {
      ...
      // opts.Actor, opts.SupervisorToken, opts.PhotoURL, opts.SkipProximity are never referenced
  }
  ```
- **Logic Chain**:
  `TransitionOpts` fields are never evaluated in the transition lookup matrix. The state machine performs only a pure string lookup (`switch (from, to)`). Callers passing `TransitionOpts` assume security checks or supervisor authorization are enforced, when they are entirely ignored.
- **Blast Radius**:
  State transitions that should mandate supervisor tokens or photos can be triggered without them if individual endpoint handlers omit explicit checks.
- **Remediation**:
  Add transition-specific guard clauses inside `ValidateStatusTransition` checking required `opts` (e.g. requiring `opts.SupervisorToken` on admin overrides).

---

### F-11: Multi-Supplier Parent Checkout Vulnerability to Process Crash During Sequential Child Order Creation
- **Location**: `pegasusX/apps/backend-go/order/unified_checkout.go:364-390`
- **Severity**: **MEDIUM (Distributed Saga Failure / Orphan Child Orders)**
- **Observation**:
  ```go
  // order/unified_checkout.go:364-390
  for _, supID := range supplierOrder {
      ...
      childResp, err := s.Create(ctx, retailerID, childReq)
      if err != nil {
          s.compensateParentCheckout(ctx, parentID, createdChildIDs)
          return UnifiedCheckoutResponse{}, err
      }
      createdChildIDs = append(createdChildIDs, childResp.OrderID)
  }
  ```
- **Logic Chain**:
  Child orders are created in individual per-supplier Spanner transactions. If the backend process terminates between loop iterations (e.g., node eviction, deployment), `compensateParentCheckout` is never called. The child orders already created remain in `PENDING` status, and `ParentOrders` remains in `INITIAL` status indefinitely without a background saga cleaner.
- **Blast Radius**:
  Orphan child orders tie up retailer credit and supplier inventory while remaining invisible in the retailer cart.
- **Remediation**:
  Introduce a background `ParentOrderReconciler` worker or publish an outbox saga event `EventParentCheckoutInitiated` with a timeout sweeper that cancels uncompleted parent checkout groups.

---

### F-12: Shop-Closed Worker Fails to Retry AR Invoice Creation After Post-Commit Failure
- **Location**: `pegasusX/apps/backend-go/order/worker_shop_closed.go:240-250`
- **Severity**: **MEDIUM (Financial Inconsistency / Uncollectible Debt)**
- **Observation**:
  ```go
  // order/worker_shop_closed.go:240-250
  if err == nil && s.ar != nil && resolvedDecision == DecisionCreditLeave && resolvedOrder != nil && resolvedOrder.TotalMinor > 0 {
      if _, aerr := s.ar.OpenFromCreditLeave(ctx, ar.OpenFromCreditLeaveRequest{ ... }); aerr != nil {
          return aerr
      }
  }
  ```
- **Logic Chain**:
  The Spanner transaction committing `StatusDeliveredOnCredit` and `MarkBalanceInTxn` succeeds on line 232. Then `s.ar.OpenFromCreditLeave` is executed post-commit. If it fails, `aerr` is returned, but on the next worker tick `processShopClosedTimeouts` skips the order because `order.Status` is no longer `SHOP_CLOSED_PENDING`.
- **Blast Radius**:
  Goods are marked as delivered on credit, but no AR invoice is ever created, making the debt untracked in AR.
- **Remediation**:
  Execute `s.ar.OpenFromCreditLeaveInTxn(ctx, txn, ...)` inside the `ReadWriteTransaction`, mirroring `driver_edges.go:332`.

---

## Deep Architectural & Edge-Case Questions

1. **Multi-Supplier Cross-Cell Atomicity & Deadlocks**:
   - In a multi-supplier split order where Supplier A and Supplier B reside in different geographic cells or database shards, how should the distributed saga handle partial network partitions? Currently, in-memory compensation (`compensateParentCheckout`) is vulnerable to process crashes. Should PegasusX introduce an Outbox-backed Saga Coordinator with a timeout-based Dead Letter Queue (DLQ)?
2. **Dynamic Order Line Amendments vs. Distributed Lot Tracking (FEFO/FIFO)**:
   - When orders are amended at the doorstep (`HandleAmendOrder` / `HandlePartialOffload`) or edited by warehouse staff (`WarehouseEditPreorder`), how should the system reconcile individual lot reservations (`StockLots`)? If 3 units from Lot A (expiring tomorrow) and 2 units from Lot B (expiring next month) were allocated, and 2 units are rejected, which lot's reservation should be released first?
3. **Optimistic Locking Overhead under High-Frequency Telemetry vs. Order State Mutations**:
   - `Orders` table uses a single `Version` column for optimistic locking. Driver location telemetry, proximity unlocks, shop-closed logs, and dispatch reassignment all touch the `Orders` row. In high-concurrency environments with rapid driver GPS updates, this causes high Spanner transaction abort/retry rates (`TransactionAbortedError`). Should telemetry and proximity state be decoupled into a child table (`OrderTrackingLive`) interleaved with `Orders`?
4. **Fiscal Offline-Tolerance and Post-Delivery Cash Escrow Settlement**:
   - In regions with unreliable OFD / Soliq connectivity, orders transition to `FISCAL_FAILED` and require manual retry (`RetryFiscal`) or admin force-complete (`ForceCompleteOrder`). During this window, cash collected by the driver is held in physical transit. How should driver shift reconciliation handle orders trapped in `FISCALIZING` or `FISCAL_FAILED` across shift boundaries?
5. **Credit Line Reservation vs. Actual Invoice Finalization Skew**:
   - Credit reservations are checked and held at checkout (`s.credit.Reserve`), but the actual balance deduction and AR invoice generation occur upon delivery (`CompleteOrder` or `HandleCreditLeave`). If an order is delivered in partial quantities (`PartialOffload`), how should the credit hold differential be safely synchronized back to the credit profile without creating race conditions with concurrent orders from the same retailer?

---

## Verification & Independent Reproduction Commands

1. **Verify Undefined Identifier `StatusDraft`**:
   ```bash
   cd apps/backend-go
   go build ./order/...
   # Output: order/service.go:1321:13: undefined: StatusDraft
   ```
2. **Verify Multi-User Retailer Org ID Bug**:
   Inspect `order/status_timeline.go:201` and `order/status_context.go:38`. Compare with `auth.ResolveRetailerOrgID(claims)`.
3. **Verify Inventory Reservation Leaks**:
   Inspect `order/supplier_ops.go:466-473` (`HandleApproveEarlyComplete`), `order/warehouse_ops.go:229-248` (`WarehouseEditPreorder`), and `order/negotiation.go:304-310` (`HandleRespondNegotiation`).
