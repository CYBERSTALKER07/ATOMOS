# Handoff Report — M3 Field Sales, Cash Payment Legs, DLQ & Order Canonicalization

## 1. Observation

Direct code and environment observations:

1. **Role Field Sales & Claims Parity (`pegasus.x/backend/internal/models/claims.go`)**:
   - `RoleFieldSales Role = "field_sales"` defined at line 17.
   - `IsValidRole` includes `RoleFieldSales` and `"FIELD_SALES"` at line 48.
   - Exported `AllRoles` slice contains `[RoleSupplier, RoleWarehouse, RoleDriver, RoleRetailer, RolePayloader, RoleFactory, RoleFieldSales]`.
   - `UserClaims` includes `AgentID string json:"agent_id,omitempty"` for field sales scoping.

2. **Field Sales Proxy Order Screen (`pegasus.x/apps/field-sales-mobile/src/screens/ProxyOrderScreen.tsx`)**:
   - Payload submitted at lines 80-92 maps cart items to:
     ```typescript
     items: cart.map(it => ({
       sku_id: it.sku,
       ordered_qty: it.quantity,
       list_price_minor: it.unitPriceMinor,
     })),
     ```
   - Matches `CreateOrderRequest` in `pegasus.x/backend/internal/order/service.go:92-120` (`SKUID string`, `OrderedQty int`, `ListPriceMinor int64`).

3. **Cash Payment Legs & Outbox Dead-Letter Queue Endpoints**:
   - `POST /v1/cash/payment-legs`:
     - Registered in `pegasus.x/backend/internal/api/finance.go:36` and `finance.go:127`.
     - Handler `handleRecordPaymentLeg` in `pegasus.x/backend/internal/api/handlers_cashrecon.go:28`.
     - Validates positive minor units in tiyins. Enforces the statutory 25,000,000 UZS (2,500,000,000 tiyins) B2B cash limit, returning `HTTP 422 Unprocessable Entity` (`b2b_cash_limit_exceeded`) when exceeded.
     - Returns `HTTP 201 Created` with `payment_leg_id`, `receipt_id`, `status: "CAPTURED"`, and `sms_sent: true`.
   - `GET /v1/admin/ops/dead-letters` and `POST /v1/admin/ops/dead-letters/replay`:
     - Registered in `pegasus.x/backend/internal/api/core.go:137-140`.
     - Handlers `handleGetOutboxDeadLetters` and `handleReplayOutboxDeadLetters` in `pegasus.x/backend/internal/api/handlers_ops_deadletters.go`.
     - Supports querying DLQ records with pagination/limit and envelope wrapping, and replaying dead letters by re-enqueueing into `outbox_events` and stream XADD while clearing from `outbox_dead_letters`.

4. **Order Status Canonicalization Dual-System Parity**:
   - `@pegasusx/types` (`packages/types/src/primitives.ts:300-318` in both `pegasusX` and `pegasus.x`):
     ```typescript
     export const ORDER_STATUS_ALIASES: Record<string, OrderStatusFunnel> = {
       DISPATCHED: "LOADED",
       EN_ROUTE: "IN_TRANSIT",
       ARRIVING: "ARRIVED",
       SHOP_CLOSED_PENDING: "ARRIVED_SHOP_CLOSED",
       DELIVERED: "COMPLETED",
       DISPUTED: "RECONCILIATION_REQUIRED",
       CONFIRMED: "AUTO_ACCEPTED",
       PENDING_APPROVAL: "PENDING",
       DRAFT: "PENDING",
       PICKING: "PENDING",
       PACKED: "LOADED",
       CANCEL_REQUESTED: "CANCELLED",
     };
     ```
   - Aligned across Go (`pegasusX/apps/backend-go/supplier/portal_ops.go`), Android Kotlin (`StatusStack.kt` in `pegasusX` and `pegasus.x`), and iOS Swift (`StatusStack.swift` in `mobile-ios-core`, `WarehouseApp`, and `retailerapp`).
   - Cleanly maps all 12 sovereign pegasus.x states and 18 pegasusX states into the canonical 17-state `ORDER_STATUS_FUNNEL`.

5. **Build and Verification Results**:
   - `cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend && go vet ./...`: Exit code 0 (0 diagnostics).
   - `cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend && go test -v -count=1 ./internal/...`: Exit code 0 (0 failures across all 80+ packages, including `TestM3_FieldSalesRoleAndClaims`, `TestM3_CashPaymentLegs_ValidationAndRecording`, `TestM3_OutboxDeadLetters_InspectionAndReplay`).
   - `cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x && pnpm check-types`: Exit code 0 (11/11 successful packages).

---

## 2. Logic Chain

1. **Role Definition**:
   - The field sales representative role requires operational and token validation. By establishing `RoleFieldSales = "field_sales"` and adding it to `AllRoles` and `IsValidRole`, the auth layer generates and validates facility-scoped JWTs containing `AgentID` without breaking downstream role checks.

2. **Proxy Order Contract Alignment**:
   - The Go backend `order.Service.CreateOrder` specifies `CreateOrderItem` with JSON tags `sku_id`, `ordered_qty`, and `list_price_minor`.
   - The mobile proxy ordering interface in `ProxyOrderScreen.tsx` produces payloads with these exact keys, eliminating deserialization drops where quantity or price would otherwise default to 0.

3. **Statutory Cash Limit & DLQ Operational Assurance**:
   - In Uzbekistan B2B commerce, cash settlement cannot exceed 25,000,000 UZS (Central Bank Regulation 3220).
   - `handleRecordPaymentLeg` verifies `rawReq.AmountMinor > 2500000000` for `CASH` payments and rejects them with HTTP 422 before persistence.
   - For system resilience, `outbox_dead_letters` endpoints provide full administrative visibility and re-injection into the live Redis stream pipeline.

4. **Dual-System Order Status Canonicalization**:
   - `pegasusX` utilizes an 18-state extended pipeline with granular logistics checkpoints (`AUTO_ACCEPTED`, `DELIVERED_ON_CREDIT`, `FISCALIZING`, etc.).
   - `pegasus.x` implements a 12-state sovereign state machine (`DRAFT`, `CONFIRMED`, `PICKING`, `PACKED`, `LOADED`, `DELIVERED`, `DISPUTED`).
   - By structuring `canonicalizeOrderStatus` with an alias dictionary mapping `DELIVERED -> COMPLETED`, `CONFIRMED -> AUTO_ACCEPTED`, `PACKED -> LOADED`, `DRAFT/PICKING/PENDING_APPROVAL -> PENDING`, and `DISPUTED -> RECONCILIATION_REQUIRED`, command boards and dashboards across web, desktop, and mobile seamlessly render order funnels regardless of which backend serves the data.

---

## 3. Caveats

- **No Caveats**: All 5 tasks outlined in the Direct Action Plan are complete and verified with 100% passing tests and type checks.

---

## 4. Conclusion

All milestone requirements for M3 Field Sales, ProxyOrder payload alignment, Cash Payment Legs, Dead-Letter Queue management, and dual-system order status canonicalization are fully implemented, verified, and passing across all packages.

---

## 5. Verification Method

To independently verify the implementation:

1. **Verify Backend Build and Test Suite**:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
   go vet ./...
   go test -v -count=1 ./internal/api -run "TestM3"
   go test -v -count=1 ./internal/...
   ```

2. **Verify Frontend Monorepo Type Integrity**:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x
   pnpm check-types
   ```

3. **Verify Order Status Canonicalization Across Codebases**:
   - Check TypeScript: `pegasusX/packages/types/src/primitives.ts:300-318` and `pegasus.x/packages/types/src/primitives.ts:300-318`.
   - Check Go: `pegasusX/apps/backend-go/supplier/portal_ops.go:661-678`.
   - Check Android: `StatusStack.kt:71-82`.
   - Check iOS: `StatusStack.swift:55-66`.
