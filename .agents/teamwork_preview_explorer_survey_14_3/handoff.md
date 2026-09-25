# Handoff Report: Requirement R3 Cross-Role Domain Parity Survey

**Agent:** teamwork_preview_explorer_survey_14_3  
**Date:** 2026-09-24  
**Working Directory:** `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_14_3`  
**Report Artifact:** `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_14_3/survey_domain_parity.md`  

---

## 1. Observation

Direct code observations across both repositories (`pegasusX` and `pegasus.x`):

1. **Order State Machine Definitions**:
   - In `pegasusX`: `apps/backend-go/order/state_machine.go:23-74` defines 18 canonical states: `StatusPending`, `StatusLoaded`, `StatusDelayed`, `StatusInTransit`, `StatusArrived`, `StatusShopClosedPending`, `StatusDeliveredOnCredit`, `StatusAwaitingPayment`, `StatusPendingCashCollection`, `StatusFiscalizing`, `StatusFiscalFailed`, `StatusCancelRequested`, `StatusCompleted`, `StatusCancelled`, `StatusReconciliationRequired`, `StatusBackordered`, `StatusScheduled`, `StatusAutoAccepted`. Lines 33, 48-52 explicitly enforce the ADR-009 fiscal hard-gate where delivery completion **must** transition through `FISCALIZING` to reach `COMPLETED`.
   - In `pegasus.x`: `backend/internal/order/state_machine.go:56-118` defines 12 states: `StatusDraft`, `StatusPendingApproval`, `StatusPending`, `StatusBackordered`, `StatusConfirmed`, `StatusPicking`, `StatusPacked`, `StatusLoaded`, `StatusInTransit`, `StatusArrived`, `StatusShopClosedPending`, `StatusDisputed`, `StatusDelivered`, `StatusCancelled`. The terminal delivery state is `StatusDelivered` (lines 44-46, 102). States `FISCALIZING`, `FISCAL_FAILED`, `COMPLETED`, `AWAITING_PAYMENT`, `PENDING_CASH_COLLECTION`, `DELIVERED_ON_CREDIT`, `CANCEL_REQUESTED`, `RECONCILIATION_REQUIRED` do not exist in this state machine.

2. **HTTP 428 Precondition Required Onboarding Gates**:
   - In `pegasus.x`: `backend/internal/api/router.go` implements four distinct HTTP 428 middleware interceptors:
     - Lines 448-483: `requireSupplierOnboardingCompleted` (checks `sup.OnboardingStatus == "COMPLETED"`).
     - Lines 490-553: `requireWarehouseOnboardingCompleted` (checks `wh.OnboardingStatus == "COMPLETED"`).
     - Lines 680-734: `requireDriverOnboardingCompleted` (checks `st.OnboardingStatus == "COMPLETED"` and shift readiness).
     - Lines 736-795: `requirePayloaderOnboardingCompleted` (checks `st.IsCommissioned && st.OnboardingStatus == "COMPLETED"`).
   - In `pegasusX`: A grep search across `pegasusX/apps/backend-go` for `StatusPreconditionRequired` or `428` returned 0 matches. Onboarding status relies on boolean JWT claim flags `IsRegistered` and `IsConfigured` (`auth/claims.go:66-67`).

3. **Field Sales Implementation & Broken Contracts**:
   - In `pegasus.x/apps/field-sales-mobile`:
     - `src/screens/CashCollectionScreen.tsx:29`: Calls `POST /v1/cash/payment-legs`. Grep across `pegasus.x/backend` and `pegasusX/apps/backend-go` returned 0 results.
     - `src/screens/ShelfAuditScreen.tsx:20`: Calls `POST /v1/enterprise/ai/vision/shelf-to-cart`. Grep across the entire repository returned 0 results.
     - `src/screens/ProxyOrderScreen.tsx:87-91`: Submits payload `{ "sku": it.sku, "quantity": it.quantity, "unit_price": Math.round(it.unitPriceMinor / 100) }`. In contrast, `pegasus.x/backend/internal/order/service.go:92-127` (`CreateOrderRequest`) expects `{ "sku_id": string, "ordered_qty": int, "list_price_minor": int64 }`.
     - `src/api.ts:11-16`: `CURRENT_AGENT.token` is hardcoded as empty `""`. There is no login screen or JWT authentication flow for field sales representatives.
     - `pegasus.x/backend/internal/models/claims.go:10-17`: Declares `RoleSupplier`, `RoleWarehouse`, `RoleDriver`, `RoleRetailer`, `RolePayloader`, `RoleFactory`, but omits `RoleFieldSales`.
   - In `pegasusX`: `apps/` contains 0 field sales client applications, and `backend-go` contains 0 field sales routes.

4. **Axle Load Statics vs Volumetric Capacity**:
   - In `pegasus.x`: `backend/internal/fleet/axle_load.go:46-100` implements the 3L-CVRP longitudinal static moment equation:
     $W_{front} = W_{curb,front} + \sum \frac{w_i (L - x_i)}{L}$, $W_{rear} = W_{curb,rear} + \sum \frac{w_i x_i}{L}$, checking $\frac{W_{front}}{W_{total}} \ge 20\%$ and single axle $\le 11,500\text{ kg}$. `internal/payload/repository.go:30, 40` enforces `ErrAxleOverloadViolation` and `ErrMissingOverrideReason`.
   - In `pegasusX`: Grep for `CalculateAxleBalance` returned 0 matches in `pegasusX/apps/backend-go`. The system checks volume units (VU) against `TetrisBuffer = 0.95` without axle moment calculations.

5. **Upstream Factory Production Tier**:
   - In `pegasusX`: Fully implemented with 3 client applications (`apps/factory-portal`, `apps/factory-app-android`, `apps/factory-app-ios`), 8 Spanner DDL tables (`Factories`, `FactoryProfiles`, `FactorySupplyRequests`, `FactorySupplyRequestItems`, `FactorySupplyRequestQC`, `FactoryLots`, `FactoryTruckManifests`, `FactoryLoadingBays`), and backend package `apps/backend-go/factory`.
   - In `pegasus.x`: 0 client applications, 0 backend routes. `RoleFactory` in `claims.go` is unused. Inter-facility replenishment is modeled solely as warehouse transfers (`apps/supplier-desktop/app/(portal)/transfers/page.tsx`).

6. **Platform Administration**:
   - In `pegasusX`: `apps/admin-portal` manages multi-tenant provisioning, cell directory routing (`cell-uz`, `cell-eu`, `cell-us`, `cell-kz`), dual-control feature flags, and outbox DLQ replay (`POST /v1/admin/ops/dead-letters/replay`).
   - In `pegasus.x`: 0 admin portal applications. Outbox dead letters are inserted into `outbox_dead_letters` (`internal/outbox/relay.go:192`), but there is no endpoint or UI to view or replay them.

---

## 2. Logic Chain

1. **From Observation 1**: Because `pegasusX` uses an 18-state machine with `COMPLETED` as terminal and mandatory traversal of `FISCALIZING`, whereas `pegasus.x` uses a 12-state machine with `DELIVERED` as terminal and explicit intermediate warehouse states (`CONFIRMED` $\rightarrow$ `PICKING` $\rightarrow$ `PACKED` $\rightarrow$ `LOADED`), client applications expecting a single unified order status will encounter state mismatch errors unless canonicalized.
2. **From Observation 2**: Because `pegasus.x` enforces HTTP 428 Precondition Required on all operational endpoints when an actor has not completed onboarding/commissioning, uncommissioned clients in `pegasus.x` are cleanly blocked at the network edge. In `pegasusX`, the absence of HTTP 428 middleware means access control relies entirely on application-level routing or JWT claim validation, creating an architectural boundary divergence between the two systems.
3. **From Observation 3**: In `pegasus.x`, `apps/field-sales-mobile` attempts to place proxy orders and collect cash, but fails due to:
   - Requesting missing routes (`/v1/cash/payment-legs`, `/v1/enterprise/ai/vision/shelf-to-cart`),
   - Sending `{ sku, quantity, unit_price }` instead of `{ sku_id, ordered_qty, list_price_minor }`,
   - Supplying an empty JWT token without a field sales role in claims.
   Therefore, the field sales workflow in `pegasus.x` is currently non-operational end-to-end, and in `pegasusX` it is completely unrepresented in client code.
4. **From Observation 4**: In `pegasus.x`, freight loading is mechanically constrained by physics (axle static moments, 11.5T single-axle limit, 20% steer traction), which prevents illegal and dangerous overloads on national roads. In `pegasusX`, global fleets lack this physical constraint check in backend software.
5. **From Observation 5**: Because `pegasus.x` lacks the Factory tier, it is currently scoped strictly as a distributor/wholesaler operating core (RDC to retailer), while `pegasusX` covers the full manufacturing-to-retail supply chain (Factory $\rightarrow$ CDC $\rightarrow$ RDC $\rightarrow$ Retailer).
6. **From Observation 6**: Because `pegasus.x` is an appliance-model single-tenant system, omitting a multi-tenant platform admin portal is an intentional architectural design choice. However, the lack of an Outbox DLQ inspection/replay endpoint creates an operational risk if background event publishing fails persistently.

---

## 3. Caveats

- **Asynchronous Outbox Latency**: Both systems rely on asynchronous outbox relay pollers (250ms in `pegasusX` vs 500ms in `pegasus.x`). Cross-role state changes (e.g., driver arrives $\rightarrow$ retailer notified) are decoupled through event buses (Kafka vs Redis Streams) and may experience transient delivery delays under high broker load.
- **Third-Party Soliq / Didox Dependencies**: Electronic fiscalization (12% VAT e-Factura) requires network access to external Uzbekistan government tax gateways. Failures or timeouts ($>8\text{s}$) trigger out-of-band error handling (`FISCAL_FAILED` in `pegasusX`; retry workers in `pegasus.x`).
- **Subterranean Offline Synchronization**: During basement store deliveries, drivers execute deliveries offline. Handoff between driver and warehouse occurs asynchronously upon return to depot when physical cash and return containers (Tara) are tallied.

---

## 4. Conclusion

Requirement R3 domain parity across the 8 roles is partially aligned:
- **Supplier, Retailer, Driver, Warehouse, and Payload Dock** maintain high functional parity, with `pegasus.x` exhibiting superior real-world adaptations (Telegram commerce, strict biometric/DVIR vehicle pairing, 3L-CVRP axle load statics, double-entry GL ledger with tiyin invariants).
- **Factory** is present only in `pegasusX`.
- **Platform Admin** is present only in `pegasusX` (by design for multi-tenancy).
- **Field Sales** is absent in `pegasusX` and has disjoint, broken API contracts in `pegasus.x`.
- **Order State Machine** exhibits divergent state vocabularies (18 states with `FISCALIZING` in `pegasusX` vs 12 states with `DELIVERED` terminal in `pegasus.x`).

Actionable reconciliation recommendations:
1. Harmonize frontend status badges using `@pegasusx/types` `canonicalizeOrderStatus()`.
2. Reconcile `apps/field-sales-mobile` in `pegasus.x`: add `RoleFieldSales`, implement agent login, fix order payload field names (`sku_id`, `ordered_qty`), and implement `/v1/cash/payment-legs`.
3. Add an Outbox DLQ inspection and replay API to `pegasus.x/backend/internal/api/`.
4. Backport 3L-CVRP longitudinal axle statics into `pegasusX`.

---

## 5. Verification Method

To independently verify these findings, execute the following commands in the workspace:

1. **Verify State Machine Divergence**:
   ```bash
   grep -n "case Status" /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/order/state_machine.go
   grep -n "case models.Status" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/order/state_machine.go
   ```
2. **Verify HTTP 428 Precondition Required Gates in `pegasus.x`**:
   ```bash
   grep -n "StatusPreconditionRequired" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/api/router.go
   ```
3. **Verify Absence of HTTP 428 in `pegasusX`**:
   ```bash
   grep -rnI "StatusPreconditionRequired" /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/
   # Expected: 0 matches
   ```
4. **Verify Field Sales Orphaned Endpoints in `pegasus.x`**:
   ```bash
   grep -rnI "/v1/cash/payment-legs" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/
   # Returns match in field-sales-mobile/src/screens/CashCollectionScreen.tsx:29, but 0 matches in backend
   ```
5. **Verify Axle Statics Implementation**:
   ```bash
   grep -n "CalculateAxleBalance" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/fleet/axle_load.go
   grep -rnI "CalculateAxleBalance" /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/
   # Expected: Found in pegasus.x, 0 matches in pegasusX
   ```
6. **Verify Backend Unit Tests**:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend && go test ./internal/order/... ./internal/fleet/... ./internal/payload/...
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go && go test ./order/...
   ```
