# Track 2 Audit Handoff Report: Order Lifecycle, Spanner Transactions & State Machines

## 1. Observation
- **Codebase Scope**: Audited `apps/backend-go/order`, `apps/backend-go/schema/spanner.ddl`, and related packages (`retailer/auto_order*.go`, `returns/`, `creditnote/`, `ar/`).
- **Verbatim Compiler Output**:
  ```
  # github.com/pegasusx/pegasusx/apps/backend-go/order
  order/service.go:1321:13: undefined: StatusDraft
  ```
- **Direct Code Observations**:
  1. `order/service.go:1321`: `status = StatusDraft` sets an undefined identifier in package `order`.
  2. `order/supplier_ops.go:466-473`: `HandleApproveEarlyComplete` mutates `Orders` to `StatusCancelled` directly without releasing inventory reservations from `SupplierInventoryV2` or `StockLots`.
  3. `order/warehouse_ops.go:229-248`: `WarehouseEditPreorder` updates `current.LineItems` and calls `UpdateOrder` without running `reconcilePreorderReservationsInTxn`.
  4. `order/negotiation.go:304-310`: `HandleRespondNegotiation` approves reduced quantities without inventory reservation release or credit hold adjustment.
  5. `order/status_timeline.go:62-76`: `persistStatusTransition` executes a standalone `spannerClient.Apply` outside the `UpdateOrder` ReadWriteTransaction.
  6. `order/status_timeline.go:201` & `order/status_context.go:38`: `claims.Subject != o.RetailerID` compares user subject ID to retailer organization ID, returning HTTP 403 Forbidden for retailer staff.
  7. `order/service.go:1511-1516`: `createBackorderOrder` error is logged with `WarnContext` and swallowed in a detached goroutine.
  8. `order/fiscal.go:872-876`: Inside the RW transaction closure of `ForceCompleteOrder`, calls are made to `AssertMoneyCoversDelivery`, `getDeliveredGrossMinor`, and `getCapturedPaymentMinor` which perform non-transactional snapshot reads via `spannerClient.Single()`.
  9. `order/partial_offload.go:320-328`: Executes an unversioned `spannerClient.Apply` modifying `Orders` immediately after `UpdateOrderWithTxn` has already committed.
  10. `order/state_machine.go:14-81`: `ValidateStatusTransition` accepts `TransitionOpts` but never checks `opts.Actor`, `opts.SupervisorToken`, `opts.PhotoURL`, or `opts.SkipProximity`.

## 2. Logic Chain
1. **Compilation Failure**: `order/service.go:1321` refers to `StatusDraft`, which is not declared. This prevents `go build ./order/...` and any package depending on `order` from compiling.
2. **Inventory Leakage**: In Cloud Spanner, inventory reservations are held in `SupplierInventoryV2.QuantityReserved` and `OrderStockReservationMarkers`. Because `UpdateOrder` only automatically releases reservations when status transitions to `StatusCancelled`, state transitions or amendments that modify line items (`WarehouseEditPreorder`, `HandleRespondNegotiation`) or cancel orders via direct `UpdateMap` (`HandleApproveEarlyComplete`) bypass reservation release, permanently locking inventory and falsely reporting stockouts.
3. **Audit Loss**: Emitting outbox events inside the RW transaction guarantees event dispatch, but performing separate `spannerClient.Apply` calls for `OrderStatusTransitions` post-commit breaks dual-write consistency if a failure occurs between the two transactions.
4. **Tenant/User Scope Inconsistency**: Multi-user JWT claims separate individual user identity (`claims.Subject`) from organization identity (`claims.RetailerID` / `auth.ResolveRetailerOrgID`). Handlers comparing `claims.Subject == o.RetailerID` fail for all non-root organization staff.
5. **Concurrency & Read-Write Violations**: Spanner RW transactions require all read operations within the transaction to execute through `txn` to acquire shared/exclusive locks. Using `spannerClient.Single()` inside the transaction closure introduces stale reads and race conditions with concurrent payment capture and settlement calculations.

## 3. Caveats
- **Read-Only Investigation**: In compliance with team protocols, no application source code files were modified; findings and remediation proposals are documented in `.agents/audit_track2_order_spanner/findings.md`.
- **Database Backend**: The analysis focuses on Google Cloud Spanner persistence (`repository_spanner.go` and `spanner.ddl`). In-memory mock repositories used in unit tests may mask transaction isolation and reservation leak issues.

## 4. Conclusion
The Track 2 Order Lifecycle and Spanner Transaction audit is **COMPLETE**. The audit identified **1 Critical Build Blocker**, **1 Critical Inventory Leak**, **5 High-Severity Flaws**, and **5 Medium-Severity Concurrency / Saga Inconsistencies**, accompanied by **5 deep architectural edge-case questions**.

Detailed findings and exact `file:line` references are recorded in `/Users/shakhzod/Desktop/V.O.I.D/.agents/audit_track2_order_spanner/findings.md`.

## 5. Verification Method
1. **Verify Build Error**:
   ```bash
   cd apps/backend-go && go build ./order/...
   ```
2. **Verify Code Locations**:
   - Inspect `order/service.go:1321`
   - Inspect `order/supplier_ops.go:466-473`
   - Inspect `order/warehouse_ops.go:229-248`
   - Inspect `order/negotiation.go:291-311`
   - Inspect `order/status_timeline.go:62-76` & `order/status_timeline.go:201`
   - Inspect `order/fiscal.go:872-876`
   - Inspect `order/partial_offload.go:320-328`
