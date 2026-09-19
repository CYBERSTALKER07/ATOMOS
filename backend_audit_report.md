# PegasusX Go Backend Master Codebase Audit Report
**Target Codebase**: `pegasusX/apps/backend-go` & Associated Client Interfaces (`apps/`)  
**Audit Date**: 2026-08-30  
**Scope**: 8 Specialized Domains (Tracks 1–8)  
**Status**: COMPLETE & AUTHORITATIVE  

---

## 1. Executive Summary

A comprehensive, line-by-line architectural, security, transactional, and concurrency audit of the PegasusX Go backend was performed across all eight primary architectural tracks. The audit systematically verified Google Cloud Spanner Read-Write transactions, multi-tenant cell isolation, role-based identity boundaries, outbox event atomicity, Kafka streaming pipelines, 8-hub WebSocket multiplexing, physical dock operations, IoT hardware integration, and double-entry financial ledgering.

### 1.1 Summary Metric Table

| Track | Domain Area | Critical | High | Medium | Low / Perf | Total Findings |
|---|---|:---:|:---:|:---:|:---:|:---:|
| **Track 1** | Core Infrastructure, Auth, Admin & Middleware | 3 | 7 | 5 | 3 | **18** |
| **Track 2** | Order Lifecycle, Spanner Transactions & State Machines | 2 | 5 | 5 | 0 | **12** |
| **Track 3** | Supplier, Factory & Catalog Domain | 4 | 5 | 5 | 3 | **17** |
| **Track 4** | Retailer, Warehouse & Stock Fulfillment Domain | 3 | 6 | 4 | 1 | **14** |
| **Track 5** | Driver, Fleet, Dispatch & Routing Optimization | 6 | 6 | 5 | 2 | **19** |
| **Track 6** | Payload, Terminal, IoT & Hardware Domain | 4 | 6 | 3 | 1 | **14** |
| **Track 7** | Payments, PSP, Escrow, Invoicing & Financial Integrity | 3 | 4 | 4 | 0 | **11** |
| **Track 8** | Realtime Engine, Outbox Pattern, Kafka & Multi-Hub WebSocket | 3 | 3 | 3 | 1 | **10** |
| **TOTALS** | **Entire PegasusX Backend** | **28** | **42** | **34** | **11** | **105** |

---

### 1.2 Systemic Themes & Cross-Cutting Vulnerabilities

1. **Spanner DDL Schema Drift & Strict NOT NULL Constraint Aborts**:
   - Multiple core mutation flows (such as dispatch compensation in `supplier/dispatch_execute.go:683`, dock exceptions in `payload/exceptions.go:77`, rescue in `warehouse/dispatch_rescue.go:201`, dynamic replan in `routing/replan.go:59`, and ETA updates in `eta/service.go:119`) insert directly into `OutboxEvents` while omitting the mandatory primary key column `SupplierId STRING(64) NOT NULL` (defined in `schema/spanner.ddl:690`). These transactions fail 100% of the time upon commit.
   - Phantom database tables and non-existent columns are referenced in production code paths: `FactoryRawMaterials` in `factory/bom.go:53`, `Routes` / `RouteStops` in `eta/service.go:152`, and `SequenceIndex` on `Orders` in `order/driver_edges.go:94`.
   - Inbound warehouse receiving of short shipments fails with `FAILED_PRECONDITION` because `WarehouseSupplyRequests` inserts omit mandatory `CoverageStartDate` and `ProjectedUnits` columns (`warehouse/receive_items.go:159`).

2. **Multi-Tenant Identity Collisions & Claim Scoping Breakdowns**:
   - Regional cell isolation is completely bypassed when an incoming token omits the `home_cell` claim (`auth/cell_isolation.go:32`), allowing unauthorized cross-cell access.
   - Multi-user authentication in supplier and factory portals executes non-deterministic `LIMIT 1` queries on non-unique phone numbers without scoping by `SupplierId`, intermittently logging staff into competitors' supplier accounts (`supplier/repository_spanner.go:713`, `factory/auth_login.go:207`).
   - Retailer staff authorization is broken across order timelines and status context because the backend compares individual user IDs (`claims.Subject`) directly against organization IDs (`o.RetailerID`), returning 403 Forbidden to legitimate store managers and clerks (`order/status_timeline.go:201`).
   - Multi-user retailer connections are deaf to WebSocket real-time events because connection rooms subscribe to personal user IDs (`"retailer:<user_id>"`) while domain events broadcast to organization rooms (`"retailer:<org_id>"`) (`ws/handler.go:161`).

3. **Inventory Reservation Leaks & Competing Inventory Systems**:
   - Supplier approval of early route completion cancels orders without releasing stock reservations, permanently locking physical inventory (`order/supplier_ops.go:466`).
   - Supplier spreadsheet inventory imports unconditionally reset `QuantityReserved = 0`, wiping out active in-flight order allocations and causing severe overselling (`supplier/import_sessions_apply.go:306`).
   - Two competing inventory models (`InventoryLevels` legacy table vs `SupplierInventoryV2` / `StockLots` WMS) operate simultaneously and continually drift out of sync across portal and warehouse endpoints (`supplier/portal_handlers.go:1049`, `warehouse/demand_products.go:352`).
   - Quarantining a warehouse bin fails to update the status of child `StockLots`, leaving damaged/contaminated goods sellable in ATP calculations (`stocklots/locations.go:114`).

4. **Realtime Event Sourcing & Kafka Pipeline Vulnerabilities**:
   - When a Kafka consumer message fails DLQ delivery and returns `ErrSkipCommit`, consuming the next message on the same partition worker commits the higher offset, permanently dropping the failed event due to Kafka's monotonic offset acknowledgement (`kafka/workerpool/workerpool.go:137`).
   - `granularRoutingKey` dynamically alters Kafka partition keys by appending sub-entity IDs (`":order-123"`), breaking Kafka FIFO ordering for events within the same aggregate root (`outbox/relay.go:280`).
   - WebSocket Hub broadcasting executes synchronous `Send()` loops with 5-second deadlines across all connections, allowing a single degraded mobile client to block Kafka consumer threads and healthy clients for 15+ seconds (`ws/hub.go:235`).
   - Multiple domain services broadcast events to non-existent WebSocket rooms (`"warehouse_ops"`, `"fleet_map"`, `"fleet_broadcast"`) that no client ever joins (`payload/progress.go:57`, `driver/live_tracking.go:56`).

5. **Financial Ledger & Settlement Integrity Flaws**:
   - Chargeback reversal transactions insert ledger entries with hardcoded `AmountMinor: 0` and defaulted currency, corrupting supplier settlement calculations and financial reconciliation (`payment/service.go:645`).
   - Accounts Receivable invoice write-offs zero the in-memory balance before invoking `ApplyCreditNote`, causing the database update to execute with `amount = 0` and leaving uncollectible debt open in Spanner (`ar/treasury_hub.go:189`).
   - Payout batch generation calculates gross captured amounts in a read-only query and then updates settlement slices to `BATCHED` in a separate transaction, permanently stranding concurrent payments from supplier payouts (`payout/payout.go:136`).
   - Cloud Spanner Go SDK cannot scan JSON columns into `[]byte`, causing `WebhookInboxStore.ProcessPending` to fail deserialization on 100% of pending webhooks (`payment/webhook_inbox.go:101`).

6. **Hardware, Cold-Chain & Physical Bridge Inversions**:
   - When an IoT sensor reports an in-transit truck temperature excursion, the cold-chain service mistakenly quarantines all available stock in the originating warehouse rather than the cargo on the truck (`stocklots/coldchain.go:254`).
   - `payload/apply.go:15` performs a full-table read and blind rewrite of every manifest, order, and exception for the supplier on every single mutation, causing extreme lock contention and transaction abort storms in Spanner.
   - Client API contract drift exists across native apps: Expo Terminal calls unmounted `POST /v1/payload/scan`, Android calls unmounted `POST /v1/delivery/missing-items`, and iOS `sealOrder` omits the mandatory `manifest_id` parameter.

---

## 2. Track 1: Core Infrastructure, Auth, Admin & Middleware

### VULN-01: Regional Cell Isolation Bypass via Omitted `home_cell` Claim
- **Location**: `pegasusX/apps/backend-go/auth/cell_isolation.go:32-35`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: The local-first cell router checks `got := strings.ToLower(strings.TrimSpace(c.HomeCell))`. When `got == ""`, the function returns `nil` (ACCEPTED). If an attacker presents a JWT without a `home_cell` claim, regional cell enforcement is bypassed entirely.
- **Ecosystem Blast Radius**: Regional data sovereignty breach; foreign tenant traffic routes into local cell Spanner databases without restriction.
- **Remediation**:
  ```go
  if got == "" || got != current {
      return ErrForeignCellAccess
  }
  ```

### VULN-02: MFA Enrollment Overwrite & Step-Up Bypass on Enrolled Admin Accounts
- **Location**: `pegasusX/apps/backend-go/mfa/service.go:77-98`, `mfa/handlers.go:161-164`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: Routes under `/platform-admin/mfa/` are exempt from step-up validation. `BeginEnroll` unconditionally overwrites existing TOTP secrets in Spanner and resets `Enabled: false`. An attacker with a temporary session token can enroll a new secret, confirm it, and hijack the administrator account.
- **Ecosystem Blast Radius**: Complete takeover of `PLATFORM_ADMIN` identities and administrative controls.
- **Remediation**: Check if the user already has an active MFA secret in `BeginEnroll`; require valid step-up verification or recovery token before allowing re-enrollment.

### VULN-03: No-Op Mutation Protection in `ProtectMutations`
- **Location**: `pegasusX/apps/backend-go/auth/route_guard.go:14-22`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: `ProtectMutations` creates a Chi sub-router group `r.Group(func(gr chi.Router) { mount(gr) })` but fails to attach any mutation guard, CSRF, or authentication middleware to `gr`.
- **Ecosystem Blast Radius**: Endpoints routed via `ProtectMutations` under the assumption of automated protection remain completely unauthenticated and unguarded.
- **Remediation**: Inject the required mutation guard middleware into `gr` before calling `mount(gr)`.

### VULN-04: Unconditional `RoleAdmin` Escalation on OIDC ID Token Exchange
- **Location**: `pegasusX/apps/backend-go/orgoidc/service.go:113-119`
- **Severity**: **HIGH**
- **Root Cause Analysis**: `/v1/auth/oidc/exchange` validates the RSA signature of corporate ID tokens and unconditionally assigns `Role: auth.RoleAdmin` without verifying claims, group memberships, or admin email allowlists.
- **Ecosystem Blast Radius**: Any corporate employee in a supplier organization with a Google/Okta/Azure account gains full administrative rights to catalog, banking, and topology.
- **Remediation**: Map roles dynamically from ID token claims (`groups`, `roles`) or an explicit admin allowlist in `orgoidc.Config`.

### VULN-05: Missing JWKS Caching and Missing `kid` Matching in OIDC Verification
- **Location**: `pegasusX/apps/backend-go/orgoidc/jwks.go:27-57`, `orgoidc/service.go:101-108`
- **Severity**: **HIGH**
- **Root Cause Analysis**: `FetchJWKS` makes an un-cached HTTP GET request on every exchange and iterates through the JWKS keys array returning the first RSA signing key found, completely ignoring the JWT `kid` header.
- **Ecosystem Blast Radius**: SSO authentication fails during IdP key rotation windows; high latency and outbound HTTP saturation.
- **Remediation**: Implement an in-memory JWKS cache with TTL (1 hour) and match `token.Header["kid"]` against `jwk.Kid`.

### VULN-06: Denial of Service via Dynamic Bcrypt Hashing on Unauthenticated Login
- **Location**: `pegasusX/apps/backend-go/platformadmin/login.go:50-65`, `145-166`
- **Severity**: **HIGH**
- **Root Cause Analysis**: When an admin login attempt fails DB lookup, `envBootstrapAdmin` dynamically computes `bcrypt.GenerateFromPassword` with default cost. Concurrent unauthenticated requests exhaust CPU cores.
- **Ecosystem Blast Radius**: API CPU starvation leading to complete platform denial of service.
- **Remediation**: Pre-compute `PLATFORM_ADMIN_PASSWORD` bcrypt hash at startup via `sync.Once` and store in memory.

### VULN-07: Prometheus High-Cardinality Metric Explosion in `HTTPMetricsMiddleware`
- **Location**: `pegasusX/apps/backend-go/telemetry/http_metrics.go:63-70`
- **Severity**: **HIGH**
- **Root Cause Analysis**: On 404 / unmatched routes, `RoutePattern()` returns `""`, and the middleware falls back to `route = r.URL.Path` as the Prometheus label value. Random scanner URLs create unbounded metric time series.
- **Ecosystem Blast Radius**: Memory exhaustion and container OOM kills in the Go backend.
- **Remediation**: Fall back to static label `route = "unmatched"` when `RoutePattern()` is empty.

### VULN-08: In-Memory Fixed-Window Rate Limiter Unbounded Memory Leak
- **Location**: `pegasusX/apps/backend-go/bootstrap/reliability_middleware.go:461-503`
- **Severity**: **HIGH**
- **Root Cause Analysis**: `fixedWindowRateLimiter.buckets` stores client rate-limiting keys in a standard Go map with no expiration cleanup or eviction mechanism.
- **Ecosystem Blast Radius**: Continuous memory leak under high IP turnover during fallback in-memory rate limiting.
- **Remediation**: Implement an LRU cache with fixed capacity or a background sweeper goroutine that deletes expired window buckets.

### VULN-09: Token Refresh Rejects Expired Tokens and Duplicates Active Tokens
- **Location**: `pegasusX/apps/backend-go/auth/refresh.go:28-32`, `auth/jwt.go:174-176`
- **Severity**: **HIGH**
- **Root Cause Analysis**: `HandleTokenRefresh` parses tokens using `Parse()`, which returns `ErrInvalidToken` on expired tokens, blocking standard 401 client refresh flows. Furthermore, refreshing active tokens does not revoke the old `jti`, allowing duplicate token trees.
- **Ecosystem Blast Radius**: Mobile and web clients cannot silently refresh expired tokens; exfiltrated tokens can be multiplied indefinitely.
- **Remediation**: Issue dedicated refresh tokens with distinct TTLs and invalidate prior token IDs upon rotation.

### VULN-10: Missing TOTP Replay Prevention Window
- **Location**: `pegasusX/apps/backend-go/mfa/service.go:129-148`, `mfa/totp.go:52-76`
- **Severity**: **HIGH**
- **Root Cause Analysis**: `VerifyCode` validates codes across a 90-second window but does not track consumed time steps. An intercepted OTP can be replayed multiple times within 90 seconds.
- **Ecosystem Blast Radius**: MFA non-repudiation bypass on admin accounts.
- **Remediation**: Store `mfa:used:<subject>:<step>` in Redis with 90-second TTL and reject previously seen steps.

### BUG-11: Background Notification Consumer Leak on Startup Race
- **Location**: `pegasusX/apps/backend-go/main.go:117`, `runtime_workers.go:240-254`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: API server starts an in-process notification consumer if no worker heartbeat is detected during initial boot. This consumer is never terminated when dedicated worker pods come online.
- **Ecosystem Blast Radius**: Duplicate push notifications and SMS messages delivered to drivers and retailers.
- **Remediation**: Implement periodic worker liveness checks to shut down API pod consumers when worker pods become active.

### BUG-12: Overwriting Warehouse Metadata on Staff Registration
- **Location**: `pegasusX/apps/backend-go/warehouse/auth_register.go:142-146`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: `HandleWarehouseRegister` issues an `InsertOrUpdateMap("Warehouses", ...)` mutation updating `CountryCode` and `UpdatedAt` on the warehouse entity when a staff member registers.
- **Ecosystem Blast Radius**: Inadvertent corruption of warehouse configuration and audit metadata.
- **Remediation**: Restrict mutations in `HandleWarehouseRegister` to `WarehouseUsers`.

### BUG-13: Fragile Error Handling in Spanner MFA Repository
- **Location**: `pegasusX/apps/backend-go/mfa/spanner.go:24-27`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: Compares `if err == spanner.ErrRowNotFound` instead of checking `spanner.ErrCode(err) == codes.NotFound`. Wrapped gRPC status errors trigger unexpected 500 errors.
- **Ecosystem Blast Radius**: Admin users without MFA records receive 500 Internal Server Error instead of 200 unconfigured status.
- **Remediation**: Use `if spanner.ErrCode(err) == codes.NotFound { return Record{}, false, nil }`.

### BUG-14: Stateless Staff Invites Lack One-Time-Use Consumption
- **Location**: `pegasusX/apps/backend-go/staffinvite/invite.go:122-163`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: HMAC staff invites are stateless and contain no nonces or consumption tracking in the database, allowing an invite link to be reused multiple times until expiration.
- **Ecosystem Blast Radius**: Unauthorized account registrations if an invite link is shared.
- **Remediation**: Store invite token hashes in Spanner with a `Status: CONSUMED` flag upon first registration.

### BUG-15: Dual-Control Asymmetry in Platform Tenant Lifecycle
- **Location**: `pegasusX/apps/backend-go/platformadmin/service.go:204-228`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: While tenant approval requires two distinct administrators, destructive transitions (`SUSPENDED`, `OFFBOARDED`) can be triggered unilaterally by a single administrator.
- **Ecosystem Blast Radius**: Rogue admin can halt commerce across an entire supplier network without peer approval.
- **Remediation**: Enforce two-party dual control on destructive tenant status transitions.

### PERF-16: Unindexed Full Scan in Telemetry SLO Collector Polling Loop
- **Location**: `pegasusX/apps/backend-go/telemetry/slo_metrics.go:142-152`
- **Severity**: **LOW / PERF**
- **Root Cause Analysis**: `outboxLagP99` runs `SELECT TIMESTAMP_DIFF(PublishedAt, CreatedAt, MILLISECOND) FROM OutboxEvents ... ORDER BY lag_ms DESC LIMIT 100` every 60 seconds without an index on computed `lag_ms`.
- **Ecosystem Blast Radius**: High Spanner CPU usage on telemetry polling loops.
- **Remediation**: Compute lag metrics in-memory inside the outbox relay.

### PERF-17: Nested Retries Duplicate Spanner SDK Internal Retry Loop
- **Location**: `pegasusX/apps/backend-go/spannerutils/retry.go:26-50`
- **Severity**: **LOW / PERF**
- **Root Cause Analysis**: Outer retry loop wraps Spanner SDK's `client.ReadWriteTransaction` without jitter, causing retry amplification under contention.
- **Ecosystem Blast Radius**: Increased transaction abort storms under high concurrency.
- **Remediation**: Rely on Spanner SDK internal retries or add full jitter backoff to outer retries.

### LEAK-18: Spanner ReadOnlyTransaction Session Leak Risk in Context Propagation
- **Location**: `pegasusX/apps/backend-go/spannerutils/retry.go:65-75`
- **Severity**: **LOW / LEAK**
- **Root Cause Analysis**: `*spanner.ReadOnlyTransaction` is passed via `context.Context` without guaranteed `defer txn.Close()` lifecycle management.
- **Ecosystem Blast Radius**: Spanner client session pool depletion under caller error paths.
- **Remediation**: Enforce scoped callback closures with explicit `defer txn.Close()`.

---

## 3. Track 2: Order Lifecycle, Spanner Transactions & State Machines

### F-01: Compiler Failure Due to Undefined Identifier `StatusDraft`
- **Location**: `pegasusX/apps/backend-go/order/service.go:1321`
- **Severity**: **CRITICAL (Build Blocker)**
- **Root Cause Analysis**: `order/service.go:1321` assigns `status = StatusDraft`. The package defines `ConfirmationStatusDraft` for confirmation status, but does not define `StatusDraft` on the `Status` type. `go build ./order/...` fails with `undefined: StatusDraft`.
- **Ecosystem Blast Radius**: Package compilation failure across all services importing `order`.
- **Remediation**: Assign `confirmation = ConfirmationStatusDraft` and keep `status = StatusPending`.

### F-02: Permanent Inventory Reservation Leak in Supplier Early Complete Approval
- **Location**: `pegasusX/apps/backend-go/order/supplier_ops.go:466-473`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: When a supplier approves early route completion, incomplete orders are transitioned to `StatusCancelled` via direct `spanner.UpdateMap("Orders", ...)` without invoking `ReleaseReservationsFromOrderFields`.
- **Ecosystem Blast Radius**: Reserved stock quantities in `SupplierInventoryV2` and `StockLots` are never decremented, ghost-locking physical stock permanently.
- **Remediation**: Call `ReleaseReservationsFromOrderFields(ctx, txn, supplierID, warehouseID, orderSource, lineItemsRaw)` inside the Spanner transaction closure.

### F-03: Missing Inventory Reservation Reconciliation on Warehouse Pre-order Line Edit
- **Location**: `pegasusX/apps/backend-go/order/warehouse_ops.go:229-248`
- **Severity**: **HIGH**
- **Root Cause Analysis**: Warehouse pre-order line edits update `Orders.LineItemsJson` and total amounts, but never reconcile inventory reservations in `SupplierInventoryV2` or `StockLots`.
- **Ecosystem Blast Radius**: Quantity increases lead to overselling; quantity decreases strand ghost inventory reservations.
- **Remediation**: Invoke `s.updatePreorderLines(ctx, current, lineItems, ...)` in `WarehouseEditPreorder`.

### F-04: Unreconciled Inventory & Trapped Credit Hold on Quantity Negotiation Approval
- **Location**: `pegasusX/apps/backend-go/order/negotiation.go:291-311`
- **Severity**: **HIGH**
- **Root Cause Analysis**: When a supplier approves quantity negotiations submitted by drivers, `Orders` is updated with reduced line items, but delta inventory reservations and retailer credit holds are not released.
- **Ecosystem Blast Radius**: Trapped credit limits on retailer accounts and ghost inventory allocations in warehouses.
- **Remediation**: Calculate line item differences and invoke `ReleaseReservationsInTxn` and `s.credit.AdjustReserveInTxn`.

### F-05: Non-Atomic Out-of-Transaction Write of `OrderStatusTransitions`
- **Location**: `pegasusX/apps/backend-go/order/status_timeline.go:62-76`
- **Severity**: **HIGH**
- **Root Cause Analysis**: `persistStatusTransition` executes `s.spannerClient.Apply(...)` after the order mutation transaction has committed. If the server terminates or encounters a network error, the transition log is lost while the order status remains changed.
- **Ecosystem Blast Radius**: Audit trail loss, legal compliance violations, and missing steps in user timeline UIs.
- **Remediation**: Buffer the `OrderStatusTransitions` insert mutation directly into the `ReadWriteTransaction` of `UpdateOrder`.

### F-06: Retailer Staff Authorization Break (`claims.Subject` vs `RetailerID` Org ID)
- **Location**: `pegasusX/apps/backend-go/order/status_timeline.go:201`, `order/status_context.go:38`
- **Severity**: **HIGH**
- **Root Cause Analysis**: Code directly compares `if claims.Subject != o.RetailerID`. In multi-user organizations, `claims.Subject` is the user ID (`usr_...`) while `o.RetailerID` is the organization ID (`ret_...`).
- **Ecosystem Blast Radius**: All retailer store managers and staff receive 403 Forbidden on order timeline and status context endpoints.
- **Remediation**: Use `if auth.ResolveRetailerOrgID(claims) != o.RetailerID { return ErrOrderForbidden }`.

### F-07: Silent Error Swallowing During Backorder Creation in Partial Fulfillment Split
- **Location**: `pegasusX/apps/backend-go/order/service.go:1511-1516`
- **Severity**: **HIGH**
- **Root Cause Analysis**: When an order is split into a backorder, `createBackorderOrder` runs asynchronously in a detached goroutine. If it fails, the error is swallowed and only logged as a warning.
- **Ecosystem Blast Radius**: Unfulfilled backordered items vanish from the system without notifying the retailer or scheduling delivery.
- **Remediation**: Create backorder orders synchronously within the transaction or emit an outbox event processed by a durable worker.

### F-08: Non-Transactional Snapshot Reads Inside Spanner RW Transaction in `ForceCompleteOrder`
- **Location**: `pegasusX/apps/backend-go/order/fiscal.go:872-876`, `order/settlement_hardening.go:195-210`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: Inside the `ReadWriteTransaction` closure, `AssertMoneyCoversDelivery` and `getCapturedPaymentMinor` execute queries against `s.spannerClient.Single()`, bypassing transaction lock acquisition.
- **Ecosystem Blast Radius**: Interleaved concurrent payment captures cause dirty read calculations and duplicate settlement exceptions.
- **Remediation**: Pass `txn` directly into all read functions within the transaction closure.

### F-09: Blind Unversioned Spanner `Apply` Overwrite on `Orders` in Partial Offload
- **Location**: `pegasusX/apps/backend-go/order/partial_offload.go:320-328`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: After `UpdateOrderWithTxn` commits the order state, a redundant `spannerClient.Apply` writes `PartialDelivery` and `UpdatedAt` without version checking outside the transaction.
- **Ecosystem Blast Radius**: Overwrites concurrent driver updates and causes timestamp jitter.
- **Remediation**: Remove redundant unversioned `Apply` mutations; state is already committed in transaction.

### F-10: State Machine Ignores All `TransitionOpts`
- **Location**: `pegasusX/apps/backend-go/order/state_machine.go:14-81`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: `ValidateStatusTransition` accepts `TransitionOpts` (`Actor`, `SupervisorToken`, `PhotoURL`) but performs only a simple string lookup, ignoring all option fields.
- **Ecosystem Blast Radius**: State transitions requiring supervisor authorization or proof photos can bypass validation if handlers rely solely on `ValidateStatusTransition`.
- **Remediation**: Implement option validation rules inside `ValidateStatusTransition`.

### F-11: Multi-Supplier Parent Checkout Vulnerability to Process Crash
- **Location**: `pegasusX/apps/backend-go/order/unified_checkout.go:364-390`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: Child orders are created sequentially in individual transactions. If the backend process crashes mid-loop, `compensateParentCheckout` is never invoked, leaving orphan child orders.
- **Ecosystem Blast Radius**: Orphan child orders tie up retailer credit and supplier inventory.
- **Remediation**: Introduce a background saga reconciler or publish an outbox event with timeout compensation.

### F-12: Shop-Closed Worker Fails to Retry AR Invoice Creation After Post-Commit Failure
- **Location**: `pegasusX/apps/backend-go/order/worker_shop_closed.go:240-250`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: `s.ar.OpenFromCreditLeave` is called post-commit. If it fails, the order status has already changed to `StatusDeliveredOnCredit`, causing subsequent worker runs to skip the order.
- **Ecosystem Blast Radius**: Goods delivered on credit without an AR invoice created, leaving debt untracked.
- **Remediation**: Execute `s.ar.OpenFromCreditLeaveInTxn` inside the Spanner `ReadWriteTransaction`.

---

## 4. Track 3: Supplier, Factory & Catalog Domain

### Finding 3.1 (T3-4.1): Missing `SupplierId` on Raw `OutboxEvents` Mutations
- **Location**: `supplier/dispatch_execute.go:683-691`, `712-720`, `322-330`, `payload/exceptions.go:77-84`, `warehouse/dispatch_rescue.go:201-208`, `routing/replan.go:59-66`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: `schema/spanner.ddl:690` defines `OutboxEvents` with `PRIMARY KEY (SupplierId, EventId)` and `SupplierId STRING(64) NOT NULL`. Raw mutations insert without `SupplierId`, causing Spanner to abort the transaction with `column SupplierId cannot be NULL`.
- **Ecosystem Blast Radius**: Compensation routines, dispatch cancellations, and route replanning fail 100% of the time, leaving zombie manifests in `ASSIGNED` status.
- **Remediation**: Always use canonical `outbox.EmitJSON(ctx, txn, ...)` and ensure `SupplierId` is explicitly provided in all raw mutations.

### Finding 3.2 (T3-3.1): Phantom Table & Mock BOM Hardcoding in Production Execution
- **Location**: `factory/bom.go:49-86`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: Factory work order execution queries `FactoryRawMaterials` (which does not exist in `schema/spanner.ddl`) and uses a hardcoded `requested * 2` multiplier with actual stock deduction code commented out.
- **Ecosystem Blast Radius**: Manufacturing batch scheduling and raw material validation fail completely in production.
- **Remediation**: Add `BillOfMaterials` and `FactoryRawInventory` tables to `schema/spanner.ddl`, and implement dynamic BOM explosion with atomic inventory reservation.

### Finding 3.3 (T3-1.1): Multi-Tenant Authentication Leak via Non-Deterministic Phone Lookups
- **Location**: `supplier/repository_spanner.go:713`, `factory/auth_login.go:207-214`, `supplier/repository_spanner_onboarding.go:374`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: Phone lookup queries `SELECT ... FROM SupplierUsers@{FORCE_INDEX=Idx_SupplierUsers_ByPhone} WHERE Phone = @phone LIMIT 1` do not scope by `SupplierId`. Users with identical phone numbers across tenants are authenticated into arbitrary tenant accounts.
- **Ecosystem Blast Radius**: Critical multi-tenant isolation breach allowing unauthorized administrative access across competitors.
- **Remediation**: Require `supplier_id` / `tenant_id` at login or enforce globally unique phone numbers, adding `WHERE SupplierId = @supplier_id` to phone queries.

### Finding 3.4 (T3-3.2): Severe Concurrency Bottleneck in `factory/apply.go`
- **Location**: `factory/apply.go:10-58`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: On every single factory mutation, `apply.go` loads all manifests and transfers for the factory into memory and performs blind `SaveManifest` and `SaveTransfer` writes on every single entity in Spanner.
- **Ecosystem Blast Radius**: $O(N)$ write explosion, extreme transaction lock contention, and lost updates across concurrent factory operators.
- **Remediation**: Refactor `apply.go` to mutate only the specific `ManifestID` or `TransferID` rows affected by the transaction.

### Finding 3.5 (T3-4.2): Inventory Import Wipes Out Active Order Stock Reservations
- **Location**: `supplier/import_sessions_apply.go:306-314`
- **Severity**: **HIGH**
- **Root Cause Analysis**: Spreadsheet import unconditionally writes `QuantityReserved = int64(0)` to `SupplierInventoryV2`, wiping out active in-flight order allocations.
- **Ecosystem Blast Radius**: Available-To-Promise (ATP) calculations artificially surge, causing severe double-allocation and inventory stockouts.
- **Remediation**: Read and preserve existing `QuantityReserved` or use `RollupInventoryV2InTxn` to recompute from active reservations.

### Finding 3.6 (T3-5.1): Order Vetting Blocks Valid Cash-on-Delivery and B2B Credit Orders
- **Location**: `supplier/orders_vet.go:128-136`, `239-270`
- **Severity**: **HIGH**
- **Root Cause Analysis**: `VetOrder` requires `orderPaymentClearedInTxn` to return true (settled payment session). COD and Net-30/60 Trade Credit orders do not settle payments at vetting time, causing approval to fail.
- **Ecosystem Blast Radius**: Suppliers are unable to approve and process Cash-on-Delivery and Trade Credit orders.
- **Remediation**: Check `PaymentMethod` / `PaymentTerms` and bypass the settled payment requirement for `COD` and `SUPPLIER_CREDIT`.

### Finding 3.7 (T3-4.3): Destructive Supplier Topology Updates Orphan Entire Warehouses
- **Location**: `supplier/repository_spanner.go:918-940`
- **Severity**: **HIGH**
- **Root Cause Analysis**: Updating topology executes raw `DELETE FROM Warehouses` and `DELETE FROM Factories`, orphaning all foreign keys, active orders, and inventory records.
- **Ecosystem Blast Radius**: Complete loss of operational history and orphaned inventory rows.
- **Remediation**: Implement upsert/diff reconciliation and mark decommissioned nodes as `IsActive = false`.

### Finding 3.8 (T3-3.3): Factory Manifest Completion Does Not Receive Goods at Warehouse
- **Location**: `factory/service.go:684-735`
- **Severity**: **HIGH**
- **Root Cause Analysis**: When a factory truck completes delivery, internal transfers are marked `COMPLETED`, but `WarehouseSupplyRequests.State` is never updated to `RECEIVED`, and no warehouse inventory is credited.
- **Ecosystem Blast Radius**: Digital inventory at receiving warehouses remains zero despite physical goods delivery.
- **Remediation**: Transition supply requests to `RECEIVED` and invoke `stocklots.PutawayInTxn` at the destination warehouse.

### Finding 3.9 (T3-1.2): Cross-Market Contamination in Catalog Discovery
- **Location**: `catalog/repository.go:266`
- **Severity**: **HIGH**
- **Root Cause Analysis**: `ListDiscoverableProducts` queries `Products WHERE IsActive = TRUE` without filtering by market or country pack code.
- **Ecosystem Blast Radius**: Retailers see foreign currency products (e.g. KZT/USD in Uzbekistan), causing checkout payment and routing failures.
- **Remediation**: Join `Suppliers` and filter by `Suppliers.MarketCode = @market_code`.

### Finding 3.10 (T3-2.2): Retailer Pricing Overrides Disregard Currency
- **Location**: `schema/spanner.ddl:101-115`, `supplier/retailer_pricing.go:111-137`
- **Severity**: **HIGH**
- **Root Cause Analysis**: `RetailerPricingOverrides` stores `CustomPriceMinor` without a `Currency` column, causing prices to be interpreted directly in whatever currency the request specifies.
- **Ecosystem Blast Radius**: Currency corruption on retailer-specific pricing (e.g. 10,000x discrepancy between USD and UZS).
- **Remediation**: Add `Currency STRING(3) NOT NULL` to `RetailerPricingOverrides` and validate matching currency.

### Finding 3.11 (T3-2.1): Volume Tier Pricing Silently Ignored in Pricing Engine
- **Location**: `pricing/repository.go:28-38`, `pricing/models.go:19`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: `GetActiveUnitPriceMinor` does not accept a quantity parameter, performs `LIMIT 1`, and ignores `PriceListItems.MinQty`.
- **Ecosystem Blast Radius**: Retailers receive arbitrary pricing tiers regardless of order quantity.
- **Remediation**: Accept `quantity int64`, filter `MinQty <= @quantity`, and order by `MinQty DESC`.

### Finding 3.12 (T3-1.3): Unauthenticated Factory Location and Capacity Tampering
- **Location**: `factory/location_ops.go:187-200`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: `scopedFactoryID` accepts `factory_id` from query parameters without verifying `factory.SupplierId == claims.SupplierId` when `HomeNodeID` is absent.
- **Ecosystem Blast Radius**: Insecure Direct Object Reference (IDOR) on factory GPS coordinates and capacity.
- **Remediation**: Verify factory tenant ownership before applying mutations in `HandleOpsLocation`.

### Finding 3.13 (T3-3.4): QC Inspection Failure Fails to Quarantine Supply Request
- **Location**: `factory/qc.go:286-310`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: When a QC inspection result is `FAIL`, `FactorySupplyRequestQC` audit record is inserted, but `WarehouseSupplyRequests.State` is not updated to `QUARANTINE`.
- **Ecosystem Blast Radius**: Defective production batches bypass QC and get scheduled onto delivery manifests.
- **Remediation**: Update `WarehouseSupplyRequests.State = 'QUARANTINE'` and emit `EventFactorySupplyQCRejected`.

### Finding 3.14 (T3-5.2): Replenishment Trigger Creates Empty Stub Supply Requests
- **Location**: `supplier/portal_admin_ops.go:277-347`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: `HandleReplenishmentTrigger` writes `WarehouseSupplyRequests` with 0 projected units and zero line items, causing factory batchers to take no action.
- **Ecosystem Blast Radius**: Automated replenishment fails to trigger manufacturing orders.
- **Remediation**: Insert computed line items into `WarehouseSupplyRequestItems`.

### Finding 3.15 (T3-5.3): Disjoint Inventory Systems (`InventoryLevels` vs `SupplierInventoryV2`)
- **Location**: `supplier/portal_handlers.go:1049-1105`, `supplier/returns.go:281-318`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: Different endpoints mutate different tables (`InventoryLevels` vs `SupplierInventoryV2`), causing inventory views to drift apart.
- **Ecosystem Blast Radius**: Desynchronized inventory between portal, discovery, and warehouse pick waves.
- **Remediation**: Standardize on `SupplierInventoryV2` + `StockLots` and deprecate `InventoryLevels`.

### Finding 3.16 (T3-2.3): Catastrophic $O(N^2)$ Fuzzy Matching and Quadratic Query Spikes
- **Location**: `catalog/repository.go:283-345`, `globalproducts/service.go:303-340`
- **Severity**: **LOW / PERF**
- **Root Cause Analysis**: Nested string distance comparison loops over all catalog items with serial Spanner queries inside the loop.
- **Ecosystem Blast Radius**: CPU exhaustion and timeouts during SKU creation.
- **Remediation**: Implement trigram indexing and batch-read global product records.

### Finding 3.17 (T3-3.5): IoT Telemetry Ingest Incurs Silent Data Loss on Redis Restart
- **Location**: `factory/iot_ingest.go:92-109`
- **Severity**: **LOW / PERF**
- **Root Cause Analysis**: Machine telemetry counts are flushed exclusively to Redis without database persistence.
- **Ecosystem Blast Radius**: Loss of factory OEE and machine run counts on Redis container restart.
- **Remediation**: Periodically persist flush batches to `FactoryMachineTelemetry` in Spanner.

---

## 5. Track 4: Retailer, Warehouse & Stock Fulfillment Domain

### TRK4-001: `CreditViaDefaultPutawayInTxn` Always Fails on Perishable Products
- **Location**: `stocklots/credit_putaway.go:54-61`, `stocklots/lots.go:76-78`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: `CreditViaDefaultPutawayInTxn` leaves `PutawayRequest.ExpiryDate` as `nil`. `lots.go:76` explicitly requires `expiry_date` when `shelfMeta.Perishable == true`.
- **Ecosystem Blast Radius**: Warehouse receiving of perishable customer returns, credit notes, and factory supplies aborts 100% of the time.
- **Remediation**: Compute default fallback expiry date from product shelf life or allow caller-supplied expiry date.

### TRK4-002: Spanner Schema Constraint Violation in Inbound Receiving Short Shipments
- **Location**: `warehouse/receive_items.go:159-178`, `schema/spanner.ddl:509-545`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: Backorder `WarehouseSupplyRequests` inserts omit mandatory `CoverageStartDate`, `CoverageDays`, and `ProjectedUnits` columns, triggering Spanner `FAILED_PRECONDITION`.
- **Ecosystem Blast Radius**: Warehouse operators cannot complete inbound receiving for any shipment containing shortages.
- **Remediation**: Populate all required `NOT NULL` columns in `receive_items.go`.

### TRK4-003: Shop-Closed Order Cancellation Fails when `WMS_LOTS_ENABLED`
- **Location**: `order/shop_closed.go:522, 725`, `order/inventory_release.go:26-30`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: `ReleaseReservationsFromOrderFields` passes empty string `""` for `orderID`. `inventory_release.go:26` throws an error when `WMS_LOTS_ENABLED` and `orderID == ""`.
- **Ecosystem Blast Radius**: Shop-closed cancellations fail with hard error, leaving orders stuck in `SHOP_CLOSED_PENDING` and locking driver manifests.
- **Remediation**: Call `ReleaseReservationsFromOrderFieldsWithID` passing `req.OrderID`.

### TRK4-004: Dual Inventory Repositories & Double-Deduction Math in Legacy Inventory
- **Location**: `inventory/repository.go:29-34, 231-232`, `warehouse/demand_products.go:352`
- **Severity**: **HIGH**
- **Root Cause Analysis**: In `inventory/repository.go:231`, `ReserveForOrder` decrements `QuantityOnHand` AND increments `QuantityReserved`, causing `Available()` (`QoH - QReserved`) to drop by double the order quantity.
- **Ecosystem Blast Radius**: False inventory exhaustion and inaccurate warehouse demand replenishment alerts.
- **Remediation**: Fix `ReserveForOrder` to increment `QuantityReserved` without decrementing `QuantityOnHand`; migrate `demand_products.go` to `SupplierInventoryV2`.

### TRK4-005: Missing Stock Invalidation & Re-Rollup on Bin Quarantine or Deactivation
- **Location**: `stocklots/locations.go:114-184`
- **Severity**: **HIGH**
- **Root Cause Analysis**: `PatchBinInTxn` updates bin location type to `QUARANTINE` or `IsActive = false`, but does not update child `StockLots.Status` or re-rollup `SupplierInventoryV2`.
- **Ecosystem Blast Radius**: Damaged or contaminated goods in quarantined bins remain listed as available ATP and can be picked for orders.
- **Remediation**: Update child `StockLots.Status = 'QUARANTINED'` and execute `RollupInventoryV2InTxn`.

### TRK4-006: Unbatched Multi-Transaction Failure in Retailer Stock Counting
- **Location**: `retailer/store_stock.go:671-686`, `retailer/stock_count_commit.go:298-315`
- **Severity**: **HIGH**
- **Root Cause Analysis**: Stock count commits iterate over SKUs executing individual transactions per SKU. Midway failure leaves partial commits, causing double-adjustments on client retry.
- **Ecosystem Blast Radius**: Retailer store stock balance corruption and incorrect reorder triggers.
- **Remediation**: Wrap all SKU adjustments for the count batch in a single atomic Spanner `ReadWriteTransaction`.

### TRK4-007: Stale Reads on User-Facing Cart APIs
- **Location**: `retailer/repository_cart.go:53, 83`
- **Severity**: **HIGH**
- **Root Cause Analysis**: `ListByRetailer` executes queries with `ExactStaleness(5 * time.Second)`.
- **Ecosystem Blast Radius**: Retailers see ghost items or missing items when navigating from cart to checkout.
- **Remediation**: Use default `StrongRead()` for cart queries.

### TRK4-008: Credit Limit Check-to-Delivery Race Condition
- **Location**: `order/service.go:1492-1498`, `order/credit_guard.go:11-22`
- **Severity**: **HIGH**
- **Root Cause Analysis**: Credit reservation is disabled by default at order creation, and when enabled, runs post-commit.
- **Ecosystem Blast Radius**: Retailers can place multiple large orders exceeding their credit limit; orders get picked and loaded, but fail at doorstep delivery.
- **Remediation**: Enforce `s.credit.ReserveOrderInTxn` inside the order creation transaction.

### TRK4-009: Missing Financial Credit Note & Settlement on Return Inbound Confirmation
- **Location**: `returns/inbound.go:581-606`
- **Severity**: **HIGH**
- **Root Cause Analysis**: `HandleInboundConfirm` updates return status but does not trigger credit note generation, invoice adjustment, or wallet refund.
- **Ecosystem Blast Radius**: Warehouse receives returned stock, but retailer financial balance remains uncredited.
- **Remediation**: Enqueue automatic credit note generation upon warehouse return confirmation.

### TRK4-010: Unauthenticated HTTP Mock in Warehouse Pick Waves
- **Location**: `warehouse/pick_waves.go:48-85`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: Issues unauthenticated HTTP call to `http://localhost:8000/pick-path` bypassing durable Spanner `PickWaves` / `PickTasks`.
- **Ecosystem Blast Radius**: Requests hang or fail when local mock server is unreachable.
- **Remediation**: Route all pick wave generation through `stocklots.CreatePickWaveInTxn`.

### TRK4-011: Arbitrary `LIMIT 200` Truncation in Warehouse Depot Broadcast
- **Location**: `warehouse/ops_broadcast.go:560-566`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: SQL query truncates active retailers to `LIMIT 200`.
- **Ecosystem Blast Radius**: Emergency depot advisories fail to reach retailers beyond the first 200.
- **Remediation**: Page through all active retailers associated with the warehouse.

### TRK4-012: Missing Outbox Events & WebSocket Fanouts in Cycle Counting
- **Location**: `stocklots/counting.go:60-245`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: `CreateCycleCountInTxn` and `ApproveAdjustmentInTxn` write to Spanner without emitting outbox events.
- **Ecosystem Blast Radius**: Warehouse supervisors and ERP integrations miss inventory adjustment notifications.
- **Remediation**: Add `events.EventInventoryAdjusted` outbox emissions inside transactions.

### TRK4-013: Direct `Apply()` Mutation Bypassing Outbox in Auto-Order Worker
- **Location**: `retailer/auto_order_worker.go:665-672`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: `markReorderSuggestionConverted` calls `client.Apply` directly without outbox emission.
- **Ecosystem Blast Radius**: Downstream AI replenishment loops miss converted suggestion telemetry.
- **Remediation**: Use `ReadWriteTransaction` with `outbox.EmitJSON`.

### TRK4-014: `Context.Background()` Usage in Auto-Order Worker Helpers
- **Location**: `retailer/auto_order_worker.go:455, 683`
- **Severity**: **LOW / PERF**
- **Root Cause Analysis**: Worker helpers pass `context.Background()` instead of active execution context.
- **Ecosystem Blast Radius**: Distributed trace propagation and cancellation deadlines are lost.
- **Remediation**: Thread active `ctx` through worker helpers.

---

## 6. Track 5: Driver, Fleet, Dispatch & Routing Optimization

### Finding 5.1 (T5-1.1): Driver PIN Wipe and Attribute Zeroing on REST Update
- **Location**: `driver/repository_crud.go:89`, `driver/crud_handlers.go:123-128`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: `UpdateDriver` executes `spanner.UpdateStruct("Drivers", d)`. `PinHash` is marked `json:"-"`, so `req.PinHash` is always `nil`. `UpdateStruct` sets `PinHash = NULL` and resets boolean flags to zero values.
- **Ecosystem Blast Radius**: Any admin edit to a driver's name or vehicle wipes their PIN, immediately locking the driver out of native mobile apps.
- **Remediation**: Use `spanner.UpdateMap` containing only the explicitly supplied request fields, preserving `PinHash`.

### Finding 5.2 (T5-2.1): Fatal Schema Mismatches in Rescue Service (100% Runtime Failure)
- **Location**: `driver/rescue.go:37, 47, 148, 170, 196, 210, 220`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: References non-existent columns (`TruckStatus`, `Id`, `AssignedWarehouseId` on `Drivers`; `Id` on `Orders`; `LicensePlate` on `Drivers`).
- **Ecosystem Blast Radius**: `POST /v1/driver/ops/rescue/request` and `respond` fail 100% of the time with Spanner query compilation errors.
- **Remediation**: Align all queries to `schema/spanner.ddl` column names (`DriverId`, `HomeNodeId`, `OrderId`) and join `Vehicles`.

### Finding 5.3 (T5-3.1): Fatal SQL Column Mismatch in Route Reordering (`HandleFleetRouteReorder`)
- **Location**: `order/driver_edges.go:94`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: Executes `UPDATE Orders SET SequenceIndex = @seq ... WHERE OrderId = @oid`. Column `SequenceIndex` does not exist on `Orders` table; it exists on `ManifestOrders`.
- **Ecosystem Blast Radius**: Driver manual route reordering crashes with Spanner SQL error; waypoints cannot be reordered.
- **Remediation**: Update `ManifestOrders.SequenceIndex`.

### Finding 5.4 (T5-3.2): Insecure QR Delivery Token Leakage via Telemetry Next-Stop Injection
- **Location**: `telemetryroutes/routes.go:161-193`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: `handleLocation` accepts untrusted `NextStopOrderID` from client, resolves delivery token via `DeliveryTokens.ResolveDeliveryToken`, and broadcasts it without validating driver ownership of the order.
- **Ecosystem Blast Radius**: Malicious drivers can inject arbitrary order IDs to steal delivery tokens and falsify delivery handoffs.
- **Remediation**: Validate in Spanner/cache that `NextStopOrderID` is assigned to `identity.DriverID` before resolving tokens.

### Finding 5.5 (T5-4.1): Fatal Column Mismatches in Route Replan Outbox Mutation
- **Location**: `routing/replan.go:58-67`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: Generates `OutboxEvents` mutation with invalid columns `"Topic"` and `"PayloadJson"` while omitting required `SupplierId`.
- **Ecosystem Blast Radius**: Dynamic route replanning transactions fail 100% of the time upon commit.
- **Remediation**: Use `outbox.EventRowMap(e)` to generate valid mutations.

### Finding 5.6 (T5-4.2): Phantom Tables and Columns in ETA Recalculation Service
- **Location**: `eta/service.go:152, 173, 193-196, 237`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: Queries non-existent tables `Routes`, `RouteStops`, `RetailerScores` and non-existent columns `TotalOrders`, `Retailers.Id`.
- **Ecosystem Blast Radius**: Real-time route ETA recalculations crash with Spanner SQL compilation errors.
- **Remediation**: Rewrite `RecalculateRoute` to read from `SupplierTruckManifests`, `ManifestOrders`, and `StopTwins`.

### Finding 5.7 (T5-1.2): Vehicle Volume and Class Overwritten to Zero on Partial Update
- **Location**: `driver/repository_crud.go:180`, `driver/crud_handlers.go:294-320`
- **Severity**: **HIGH**
- **Root Cause Analysis**: `UpdateVehicle` executes `spanner.UpdateStruct("Vehicles", v)`. Omitted fields default to `0.0` VU and `""` class.
- **Ecosystem Blast Radius**: Vehicle capacity in Spanner is zeroed, rendering vehicles unusable for automated dispatch.
- **Remediation**: Use `spanner.UpdateMap` for vehicle updates.

### Finding 5.8 (T5-1.3): Inability to Set Driver PIN via REST Driver Creation
- **Location**: `driver/crud_handlers.go:29-63`
- **Severity**: **HIGH**
- **Root Cause Analysis**: `HandleCreateDriver` omits PIN handling; `PinHash` is `json:"-"`.
- **Ecosystem Blast Radius**: Drivers created via REST CRUD API cannot log in with phone + PIN.
- **Remediation**: Add `PIN` string field to create request, bcrypt-hash, and persist to `PinHash`.

### Finding 5.9 (T5-1.4): Multi-Tenant Data Leak in Cash Reconciliation Listing
- **Location**: `driver/cash_bag.go:406-417`, `schema/spanner.ddl:1866-1880`
- **Severity**: **HIGH**
- **Root Cause Analysis**: `HandleListCashReconciliations` executes un-scoped `WHERE 1=1`; table lacks `SupplierId` column.
- **Ecosystem Blast Radius**: Cash reconciliation records from all suppliers and warehouses are visible across tenants.
- **Remediation**: Add `SupplierId` to table DDL and enforce tenant filtering.

### Finding 5.10 (T5-2.2): Uncommitted WebSocket Broadcast Inside ReadWriteTransaction
- **Location**: `driver/rescue.go:79`
- **Severity**: **HIGH**
- **Root Cause Analysis**: `s.driverHub.Broadcast` is called inside the `ReadWriteTransaction` closure before commit.
- **Ecosystem Blast Radius**: Aborted or retried transactions send phantom WebSocket events to drivers.
- **Remediation**: Buffer events via `outbox.EmitJSON` or broadcast post-commit.

### Finding 5.11 (T5-2.3): Undeclared Struct Fields in Driver Service
- **Location**: `driver/live_tracking.go:36, 49`, `driver/idempotency_guard.go:78`, `driver/rescue.go:64`
- **Severity**: **HIGH**
- **Root Cause Analysis**: Methods reference `s.redisClient` and `s.warehouseHub` which are undeclared on `driver.Service`.
- **Ecosystem Blast Radius**: Package compilation failure when building driver test packages.
- **Remediation**: Declare `redisClient *redis.Client` and `warehouseHub *ws.Hub` on `driver.Service`.

### Finding 5.12 (T5-3.3): Geofence Verification Bypass in Delivery Submit
- **Location**: `order/service.go:1896-1903`
- **Severity**: **HIGH**
- **Root Cause Analysis**: `if !req.BypassGeofence && (req.Latitude != 0 || req.Longitude != 0)`. Sending `0, 0` skips geofence validation without requiring supervisor token.
- **Ecosystem Blast Radius**: Drivers can submit fraudulent deliveries from any location by omitting GPS coordinates.
- **Remediation**: Require GPS coordinates for `SubmitDelivery` unless authentic supervisor bypass token is verified.

### Finding 5.13 (T5-1.5): Duplicate Cash Reconciliation Records on Retries
- **Location**: `driver/cash_bag.go:162-181`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: Generates new UUID without checking if turn-in was already recorded for `(DriverId, ShiftDate)`.
- **Ecosystem Blast Radius**: Network retries create duplicate reconciliation records for the same shift.
- **Remediation**: Upsert reconciliation row based on unique `(DriverId, ShiftDate)`.

### Finding 5.14 (T5-2.4): Zombie Unauthenticated AI Dispatch Sidecar Path
- **Location**: `driver/ai_dispatch.go:39-71`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: Unauthenticated HTTP call to `http://localhost:8000/fleet-route` duplicating canonical solver in `dispatch/plan`.
- **Ecosystem Blast Radius**: Architectural divergence and unauthenticated routing sidecar risks.
- **Remediation**: Remove file and route through `dispatch/plan/OptimizeAndValidate`.

### Finding 5.15 (T5-2.5): Single-Order Oversized Orphan Inability to Split
- **Location**: `dispatch/binpack.go:180-214`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: When `AllowRetailerSplit=true`, orders are grouped by volume; a single order exceeding max truck volume is never split into sub-orders.
- **Ecosystem Blast Radius**: Bulk orders larger than max vehicle volume can never be dispatched automatically.
- **Remediation**: Implement SKU/line-item level order splitting into sub-orders.

### Finding 5.16 (T5-3.4): Euclidean Distortion in Polyline Deviation Calculation
- **Location**: `routing/deviation.go:49-64`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: Projects GPS coordinates onto polyline segments using raw Euclidean degree math without longitude cosine correction.
- **Ecosystem Blast Radius**: Up to 40% distance distortion causing spurious off-route replan triggers.
- **Remediation**: Scale longitude differences by `math.Cos(lat * math.Pi / 180)`.

### Finding 5.17 (T5-3.5): Unmanaged Background Goroutine in Reassign Handshake
- **Location**: `order/reassign_handshake.go:44`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: Spawns unmanaged `go s.driverHub.Broadcast(context.Background(), ...)` without context cancellation.
- **Ecosystem Blast Radius**: Goroutine leaks under load spikes.
- **Remediation**: Route notifications through transactional outbox.

### Finding 5.18 (T5-4.3): Nanosecond Timestamp Defeating Idempotency in Payment Leg Split
- **Location**: `order/driver_edges.go:974, 988`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: Constructs key with `now.UnixNano()`, generating new keys on Spanner transaction retries.
- **Ecosystem Blast Radius**: Duplicate payment legs recorded on transaction retries.
- **Remediation**: Construct deterministic key `fmt.Sprintf("split-%s-%s", current.OrderID, method)`.

### Finding 5.19 (T5-4.4): Undefined Status Constant in Order Package
- **Location**: `order/service.go:1321`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: Duplicate reference to undefined `StatusDraft` breaking cross-package builds.
- **Ecosystem Blast Radius**: Package compilation failure.
- **Remediation**: Align status enum constants.

---

## 7. Track 6: Payload, Terminal, IoT & Hardware Domain

### TRK6-001: Catastrophic Warehouse Inventory Quarantining on Truck Temperature Excursions
- **Location**: `stocklots/coldchain.go:227-287` (specifically lines 254-285)
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: `quarantineManifestLotsInTxn` looks up the manifest's warehouse (`wid`) and quarantines all lots in that warehouse where `Status = 'AVAILABLE'`. In-transit spoiled truck cargo is not quarantined, while healthy stock in the warehouse is pulled from ATP.
- **Ecosystem Blast Radius**: Total warehouse operational blockage for affected SKUs across all channels; false stockouts.
- **Remediation**: Quarantine manifest-specific ship units / orders (`ManifestShipUnits` / `Orders` condition report `TEMPERATURE_BREACH`); never quarantine warehouse `AVAILABLE` stock for truck transit excursions.

### TRK6-002: Full-Table Read & Blind Rewrite on Every Single Mutation in `payload/apply.go`
- **Location**: `payload/apply.go:15-81`, `payload/repository_spanner.go:48-76`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: On every single barcode scan or exception, `apply.go` loads all manifests, all manifest orders, and all exceptions for the entire supplier, and then executes blind `InsertOrUpdateMap` writes for EVERY entity in Spanner.
- **Ecosystem Blast Radius**: Total database lockup under concurrent loading bay operations, transaction abort storms, high Spanner CPU.
- **Remediation**: Refactor to entity-scoped mutations (`tx.SaveManifestOrder(manifestID, orderID)`); eliminate full-table reading/rewriting.

### TRK6-003: Dead / Broken Dock Damage Handler with Fatal Schema Incompatibilities
- **Location**: `payload/exceptions.go:21-116`
- **Severity**: **HIGH**
- **Root Cause Analysis**: `HandleDockDamage` is unrouted in `payloaderoutes/routes.go`, targets non-existent table `LoadLedger`, uses invalid column `Orders.TotalPrice`, and passes string payload to `BYTES` column in `OutboxEvents`.
- **Ecosystem Blast Radius**: Crashes with Spanner SQL and type errors if ever invoked.
- **Remediation**: Rewrite using `ManifestLoadLines`, `Orders.TotalMinor`, and `outbox.EmitJSON`, or remove obsolete file.

### TRK6-004: Fleet Reassign Mutates `Orders` but Desynchronizes `ManifestOrders` and Volumes
- **Location**: `payload/fleet_compat.go:47-74`
- **Severity**: **HIGH**
- **Root Cause Analysis**: `HandleFleetReassign` updates `Orders.RouteId` and `DriverId`, but fails to update `ManifestOrders` or adjust `TotalVolumeVU` on source and destination `SupplierTruckManifests`.
- **Ecosystem Blast Radius**: Double-dispatching of orders, incorrect truck volume capacities, and corrupted driver manifest views.
- **Remediation**: Execute full manifest rebalancing adjusting source and destination manifests and `ManifestOrders` atomically.

### TRK6-005: Client-Backend Contract Violation: Expo Terminal Calls Unmounted `POST /v1/payload/scan`
- **Location**: `apps/payload-terminal/api.ts:251-261` vs `payloaderoutes/routes.go:40`
- **Severity**: **HIGH**
- **Root Cause Analysis**: Terminal calls `/v1/payload/scan`; backend mounts `/v1/payloader/manifests/{manifestID}/load-ledger/scan`.
- **Ecosystem Blast Radius**: Scan progress reporting returns HTTP 404 on Expo Terminal clients.
- **Remediation**: Update `apps/payload-terminal/api.ts` to call the canonical route.

### TRK6-006: Android App Contract Drift: `missingItems` Calls Unmounted Endpoint
- **Location**: `apps/payload-app-android/.../PayloadApi.kt:187-191` vs `payloaderoutes/routes.go:64-66`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: Android app calls `POST v1/delivery/missing-items`; backend mounts `/v1/delivery/exception-report`.
- **Ecosystem Blast Radius**: Missing item reports fail with HTTP 404 from Android hardware terminals.
- **Remediation**: Align Retrofit endpoint to `/v1/delivery/exception-report`.

### TRK6-007: iOS App Contract Drift: `sealOrder` Omits Required `manifest_id`
- **Location**: `apps/payload-app-ios/.../APIClient.swift:268-276` vs `payload/service.go:1977-1983`
- **Severity**: **HIGH**
- **Root Cause Analysis**: iOS client calls `POST v1/payload/seal` without `manifest_id`, which backend strictly enforces as mandatory.
- **Ecosystem Blast Radius**: Payload sealing fails with HTTP 400 `manifest_id_required` on all iOS devices.
- **Remediation**: Update iOS `sealOrder` to serialize `manifestId`.

### TRK6-008: Non-Transactional `client.Apply` for GS1 Ship Units Bypasses Outbox
- **Location**: `payload/ship_units.go:181-193`
- **Severity**: **HIGH**
- **Root Cause Analysis**: `insertShipUnit` writes `ManifestShipUnits` using standalone `client.Apply` outside parent transaction without buffering outbox event.
- **Ecosystem Blast Radius**: Downstream automated sorting systems and EDI/ASN generators miss ship-unit events.
- **Remediation**: Move ship unit insertion into parent Spanner transaction with `EventShipUnitsGenerated`.

### TRK6-009: Invalid GS1 SSCC Serial Number Calculation Breaches GS1-128 Spec
- **Location**: `payload/ship_units.go:161-168`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: Formats 64-bit uint64 FNV hash directly as decimal, overflowing the fixed 18-digit SSCC standard length and generating invalid check digits.
- **Ecosystem Blast Radius**: Dock-printed barcode labels fail industrial handheld scanner verification.
- **Remediation**: Generate bounded sequential serial references constrained by prefix length.

### TRK6-010: Destination Retailer GLN Hardcoded to Blank in ZPL Label Generator
- **Location**: `payload/ship_units.go:302-318`
- **Severity**: **LOW**
- **Root Cause Analysis**: `labelGLNs` initializes `toGLN = ""` and returns without populating from `Retailers.Gln`.
- **Ecosystem Blast Radius**: Printed crate/pallet labels fail GS1-128 compliance audits at Tier-1 enterprise retailers.
- **Remediation**: Query destination `Retailers.Gln` or `RetailerLocations.Gln`.

### TRK6-011: Multi-Pod State Divergence in `stocklots.LoadLine` Memory Fallback
- **Location**: `stocklots/load_ledger.go:41-45, 127-164`
- **Severity**: **HIGH**
- **Root Cause Analysis**: Falls back to in-process memory map `memLoadRows` across multi-pod deployments.
- **Ecosystem Blast Radius**: Worker scanning on Pod A is invisible to seal on Pod B, causing intermittent `ErrLoadLedgerIncomplete` errors.
- **Remediation**: Disallow silent in-memory fallbacks in production; persist durably to Cloud Spanner `ManifestLoadLines`.

### TRK6-012: In-Transit IoT Telemetry Pipelines Lack Sensor Modalities
- **Location**: `telemetryroutes/routes.go:70-124`, `telemetry/location_store.go:38-50`
- **Severity**: **HIGH**
- **Root Cause Analysis**: Telemetry models only GPS coordinates; lacks humidity (% RH), 3-axis accelerometer shock (G-force), angular tilt, and tamper switch contacts.
- **Ecosystem Blast Radius**: Inability to monitor high-value/fragile cargo or trigger automated tamper alarms.
- **Remediation**: Add `POST /v1/telemetry/sensors` supporting multimodal frames with automated threshold alerting.

### TRK6-013: Absence of Hardware-Level Authentication
- **Location**: `payload/auth_login.go:38-117`
- **Severity**: **HIGH**
- **Root Cause Analysis**: Terminal authentication relies solely on user PINs without mutual TLS (mTLS), TPM hardware signing, or device cert validation.
- **Ecosystem Blast Radius**: Any actor with a stolen PIN can authenticate as a terminal from untrusted arbitrary networks.
- **Remediation**: Implement device certificate registration and mTLS validation middleware (`auth.RequireDeviceCert`).

### TRK6-014: Hardcoded Regional Code (`UZ-TAS`) and Static Reassignment Distances
- **Location**: `payload/manifest_list.go:58, 103`, `payload/tablet_wire.go:42`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: Hardcodes `RegionCode: "UZ-TAS"` and static `DistanceKm: 1.2`.
- **Ecosystem Blast Radius**: Breaks multi-region cell isolation and displays inaccurate proximity metrics to loading bay operators.
- **Remediation**: Derive `RegionCode` dynamically from node pack and compute real Haversine distances.

---

## 8. Track 7: Payments, PSP, Escrow, Invoicing & Financial Integrity

### Finding 7.1 (T7-01): Reversal Ledger Entry Injected with Zero Amount and Defaulted Currency
- **Location**: `payment/service.go:645-648`, `payment/repository_spanner.go:601-613`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: `HandleChargebackReversal` initializes `ReversalRecord` with hardcoded `AmountMinor: 0` and `Currency: s.currency`, ignoring the original session amount. `SaveReversal` writes a 0-amount row into `PaymentLedgerEntries`.
- **Ecosystem Blast Radius**: Chargeback reversals fail to restore supplier settlement balances, causing underpayment in payout batches and financial reconciliation reports.
- **Remediation**: Retrieve original session amount and currency and populate `ReversalRecord` accurately.

### Finding 7.2 (T7-02): Silent Mutation Failure in Invoice Write-Off
- **Location**: `ar/treasury_hub.go:189-196`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: `WriteOffInvoice` sets `inv.BalanceMinor = 0` in memory on line 189, and then calls `ApplyCreditNote(ctx, invoiceID, inv.BalanceMinor, idemKey)` on line 194 with `0` amount. Database balance in Spanner remains untouched and status remains `OPEN`.
- **Ecosystem Blast Radius**: Invoices intended to be written off remain open; dunning workers continue dunning retailers and place uncollectible accounts on credit hold.
- **Remediation**: Capture original balance before zeroing struct (`origBalance := inv.BalanceMinor`), pass `origBalance` to `ApplyCreditNote`, and execute Spanner update setting `Status = StatusVoid`.

### Finding 7.3 (T7-03): Payout Batch Settlement Slice Money Leak Race Condition
- **Location**: `payout/payout.go:136-179`, `payout/store.go:48-65`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: Payout generation computes gross totals in a read-only query and then runs an unqualified `UPDATE InvoiceSettlementSlices SET Status = 'BATCHED' WHERE ...` in a separate transaction. Payments arriving between steps are marked `BATCHED` but omitted from payout totals.
- **Ecosystem Blast Radius**: Permanent, untracked underpayment of suppliers on orders captured concurrently during batch generation.
- **Remediation**: Perform summation and status updates within the **same Spanner ReadWriteTransaction** locking slice IDs directly (`WHERE SliceId IN UNNEST(@slice_ids)`).

### Finding 7.4 (T7-04): Permanent Webhook Inbox Deserialization Scan Failure
- **Location**: `payment/webhook_inbox.go:101-108`
- **Severity**: **HIGH**
- **Root Cause Analysis**: `ProcessPending` scans `RecordJson` (defined as `JSON` in Spanner DDL) into `var recordJSON []byte`. Cloud Spanner Go SDK rejects decoding JSON into `[]byte`, causing `row.Columns` to error on 100% of rows.
- **Ecosystem Blast Radius**: Webhook retry worker cannot deserialize or process any pending webhooks; orders awaiting payment clearance remain stuck in `AWAITING_PAYMENT`.
- **Remediation**: Change `var recordJSON []byte` to `var recordJSON string` or `spanner.NullJSON`.

### Finding 7.5 (T7-05): Zero Timestamp on Settlement Slices Created from Uncaptured Legs
- **Location**: `order/settlement_hardening.go:256`, `order/refunds.go:228-236`
- **Severity**: **HIGH**
- **Root Cause Analysis**: Refund payment legs have `CapturedAt: spanner.NullTime{Valid: false}`. `RecordPaymentLeg` passes `leg.CapturedAt.Time` (`0001-01-01`), writing slices with `CreatedAt = 0001-01-01`.
- **Ecosystem Blast Radius**: Refund slices fall outside billing period queries `[start, end)` and are excluded from payout batch deductions, causing platform monetary loss by overpaying suppliers.
- **Remediation**: Fall back to `leg.CreatedAt` or `s.now()` when `CapturedAt` is invalid/zero.

### Finding 7.6 (T7-06): Driver Cash Shift Filtered by Order Creation Time Instead of Collection Time
- **Location**: `cashrecon/expected_cash.go:49-50`
- **Severity**: **HIGH**
- **Root Cause Analysis**: `ComputeExpectedCashMinor` filters by `o.CreatedAt` (order placement time) instead of `pl.CapturedAt` (cash collection time).
- **Ecosystem Blast Radius**: Orders placed yesterday but delivered today are omitted from today's expected cash (false surplus); orders placed today for tomorrow are included today (false shortage).
- **Remediation**: Filter by `pl.CapturedAt >= @start AND pl.CapturedAt < @end`.

### Finding 7.7 (T7-07): Unchecked Version Overwrite in AR Invoice Aging Recomputation
- **Location**: `ar/service.go:876-947`
- **Severity**: **HIGH**
- **Root Cause Analysis**: Aging worker queries version in read step and blindly updates `Version = rec.ver + 1` in write step without checking if concurrent payments modified version or status in between.
- **Ecosystem Blast Radius**: Corrupts optimistic locking version counters and emits aging update events for already paid invoices.
- **Remediation**: Read `ArInvoices.Version` and `Status` inside the ReadWriteTransaction and abort if modified.

### Finding 7.8 (T7-08): Premature Integer Division in Manual Credit Note Calculation
- **Location**: `creditnote/service.go:144-146`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: Computes `(base.LineNetMinor / base.Qty) * qty`. Integer division before multiplication causes truncation loss and rounding drift from `LineGrossMinor`.
- **Ecosystem Blast Radius**: Minor unit drift in credit note totals; tax audit discrepancies in OFD reporting.
- **Remediation**: Multiply before dividing: `(base.LineNetMinor * qty) / base.Qty`.

### Finding 7.9 (T7-09): Multi-Currency FX Scaling Omits Differing Decimal Exponents
- **Location**: `fxrates/convert.go:114-146`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: `applyRate` assumes all currencies have 2 decimal places, miscalculating conversions with 0-decimal (JPY, KRW) or 3-decimal (KWD, BHD) currencies by factors of 100 or 1,000.
- **Ecosystem Blast Radius**: Severe currency conversion inaccuracies in non-2-decimal international markets.
- **Remediation**: Incorporate ISO-4217 decimal exponent scaling (`math.Pow10(expTo - expFrom)`).

### Finding 7.10 (T7-10): Adyen Webhook Acknowledgment Signature & Batch Handling
- **Location**: `payment/adyen_webhook.go:42-59`, `186-192`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: Returns JSON response rather than literal string `"[accepted]"` required by Adyen webhook protocol.
- **Ecosystem Blast Radius**: Adyen treats responses as delivery failures and repeatedly re-delivers duplicate webhooks.
- **Remediation**: Return `w.Write([]byte("[accepted]"))` on Adyen webhooks.

### Finding 7.11 (T7-11): Multiple Active Checkout Sessions Permitted for Single Order
- **Location**: `payment/retailer_checkout.go:395-464`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: Rapid user clicks generate multiple parallel sessions in `PAYMENT_REQUIRED` rather than returning the existing active session.
- **Ecosystem Blast Radius**: Retailers receive differing checkout URLs on retries; webhooks risk matching mismatched sessions.
- **Remediation**: Reuse existing active session in `PAYMENT_REQUIRED` for identical gateway and amount.

---

## 9. Track 8: Realtime Engine, Outbox Pattern, Kafka & Multi-Hub WebSocket Fanout

### Finding 8.1 (T8-01): Multi-User Retailer WebSocket Subscription Broken (Deaf to Real-Time Events)
- **Location**: `apps/backend-go/ws/handler.go:161-172`, `apps/backend-go/ws/sse.go:136-140`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: `subscribeRetailerRooms` subscribes connections to `"retailer:" + ident.Subject` (personal user ID `usr_...`). `NotificationDispatcher` broadcasts domain events to `"retailer:" + retailerID` (organization ID `ret_...`). The connection never matches the room.
- **Ecosystem Blast Radius**: All retailer staff desktop/mobile apps receive zero real-time updates for orders, payments, deliveries, and disputes.
- **Remediation**: Resolve organization ID via `auth.ResolveRetailerOrgID(ident)` and subscribe to `"retailer:" + orgID`.

### Finding 8.2 (T8-02): Kafka Workerpool Offset Commit Skips Invalidate on Subsequent Message Commit
- **Location**: `apps/backend-go/kafka/workerpool/workerpool.go:137-158`, `apps/backend-go/kafka/consumer.go:151-161`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: When a message fails DLQ delivery and returns `ErrSkipCommit`, the worker skips commit for Offset N, but continues processing Offset N+1. Committing Offset N+1 tells Kafka that all offsets up to N+1 are acknowledged, permanently dropping message N.
- **Ecosystem Blast Radius**: Unrecoverable loss of business-critical state change events during temporary broker or DLQ errors.
- **Remediation**: Halt message consumption on the partition channel when `ErrSkipCommit` occurs and trigger circuit breaker/backoff.

### Finding 8.3 (T8-03): Kafka Granular Routing Key Fragmenting Per-Entity Event Order
- **Location**: `apps/backend-go/outbox/relay.go:280-312`, `apps/backend-go/outbox/kafka_publisher.go:88`
- **Severity**: **CRITICAL**
- **Root Cause Analysis**: `granularRoutingKey` dynamically appends sub-entity IDs (`":order-123"`), causing events for the same aggregate root (`AggregateID: "ret-1"`) to hash to different Kafka partitions.
- **Ecosystem Blast Radius**: Breaks FIFO ordering across dependent domain events; downstream consumers process child events before parent creation.
- **Remediation**: Ensure routing keys strictly represent deterministic partition affinity boundaries (`e.AggregateID` or tenant key).

### Finding 8.4 (T8-04): Synchronous Fan-Out Head-of-Line Blocking in WebSocket Hub
- **Location**: `apps/backend-go/ws/hub.go:235-266`, `apps/backend-go/ws/connection.go:108-117`
- **Severity**: **HIGH**
- **Root Cause Analysis**: `fanoutLocal` iterates through room connections synchronously with a 5-second `c.Send` timeout. A slow mobile client blocks the broadcast caller (including Kafka consumer threads) for 15+ seconds.
- **Ecosystem Blast Radius**: Cascading latency spikes across Kafka consumer groups and delayed notifications for all healthy clients.
- **Remediation**: Implement asynchronous write pumps with bounded per-connection send queues (`send chan []byte`) and drop/reap slow consumers.

### Finding 8.5 (T8-05): Dual-Write Topic Poisoning and Replay Duplication in Outbox Relay
- **Location**: `apps/backend-go/outbox/relay.go:220-261`, `apps/backend-go/events/topic_routing.go:146-163`
- **Severity**: **HIGH**
- **Root Cause Analysis**: In dual-write mode, if publish to Topic 2 fails, retry loops restart from Topic 1, sending duplicate messages to Topic 1. If retries exhaust, event is dead-lettered despite successful publish to Topic 1.
- **Ecosystem Blast Radius**: Duplicate message storms on primary topics and erroneous dead-lettering of successfully published events.
- **Remediation**: Track per-topic publication status across retry attempts.

### Finding 8.6 (T8-06): Outbox Polling Contention and Multi-Tenant Starvation
- **Location**: `apps/backend-go/outbox/spanner_store.go:114-190`, `apps/backend-go/outbox/fair.go:8-52`
- **Severity**: **HIGH**
- **Root Cause Analysis**: Poller queries index range inside a ReadWriteTransaction, causing lock conflicts across replicas. Query orders strictly by `CreatedAt`, allowing a single bulk-upload tenant to starve other tenants in the 500-candidate window.
- **Ecosystem Blast Radius**: Multi-pod relay thrashing and severe latency spikes for low-volume tenants during batch imports.
- **Remediation**: Read candidates in snapshot read, compute fair claims, and claim rows via conditional mutations; partition polling by tenant.

### Finding 8.7 (T8-07): Ghost Broadcasts to Non-Existent WebSocket Rooms
- **Location**: `payload/progress.go:57`, `driver/live_tracking.go:56`, `payload/exceptions.go:104`, `driver/rescue.go:79`, `retailer/shelf_intelligence.go:66`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: Domain handlers broadcast to unnamespaced rooms (`"warehouse_ops"`, `"fleet_map"`, `"fleet_broadcast"`) that no client ever joins.
- **Ecosystem Blast Radius**: Real-time dock scan progress, driver live tracking, and shelf alerts fail to appear on frontend dashboards.
- **Remediation**: Standardize on canonical tenant-scoped rooms (`"warehouse:" + warehouseID`, `"driver:" + driverID`).

### Finding 8.8 (T8-08): Build & Unit Test Failures in `ws` and `order` Packages
- **Location**: `apps/backend-go/ws/hub_test.go:119`, `apps/backend-go/order/service.go:1321`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: Test mock `publishFailBackend` misses `IncrBy`/`DecrBy` methods; `order/service.go` references undefined `StatusDraft`.
- **Ecosystem Blast Radius**: `go test ./ws/...` and `go test ./order/...` compilation failures.
- **Remediation**: Add stub methods to test mock and fix status constant.

### Finding 8.9 (T8-09): Missing Reconnection Message Replay / Catch-up Mechanism
- **Location**: `apps/backend-go/ws/handler.go:44-80`, `apps/backend-go/ws/sse.go:191-271`
- **Severity**: **MEDIUM**
- **Root Cause Analysis**: Does not inspect `Last-Event-ID` or `?since_seq=` on reconnect; events emitted during cellular disconnection are lost.
- **Ecosystem Blast Radius**: Drivers and POS cashiers see stale state after passing through cellular dead zones until manual reload.
- **Remediation**: Support delta sync on reconnect via Redis stream ring buffer or notifications inbox.

### Finding 8.10 (T8-10): Full Table Scan in Outbox Supplier ID Backfill Query
- **Location**: `apps/backend-go/outbox/backfill.go:20-26`, `50-60`
- **Severity**: **LOW**
- **Root Cause Analysis**: `BackfillSupplierID` queries `WHERE (SupplierId IS NULL OR SupplierId = '')` without supporting index, forcing full table scans every 5 minutes.
- **Ecosystem Blast Radius**: High Spanner CPU utilization under large historical outbox sizes.
- **Remediation**: Batch mutations and paginate via primary key cursor or bounded time range.

---

## 10. Deep Architectural & Systemic Open Questions

The following 12 architectural dilemmas represent deep cross-cutting concurrency, multi-tenant partitioning, and reliability challenges that must be addressed during core hardening:

### 1. Multi-Cell Cross-Market Data Consistency vs. Distributed Outbox Lag
- **Problem**: In PegasusX's local-first architecture, regional instances (e.g. `cell-uz`, `cell-eu`, `cell-kz`) operate with independent Spanner databases. If a multinational supplier updates master SKU pricing or inventory in `cell-uz` while a retailer in `cell-eu` is placing an order, how is catalog state synchronized across cell boundaries without introducing multi-second outbox replication lag or cross-region write latencies?
- **Architectural Consideration**: Should cross-cell catalog synchronization rely on an asynchronous global event mesh (Kafka MirrorMaker 2 / EventMesh) with eventual consistency, or should global products use a centralized global directory shard with regional read-through caches?

### 2. Zero-Downtime Secret Rotation & Cryptographic Key Management
- **Problem**: `auth/jwt.go` relies on a single static `JWT_SECRET` string. Rotating this secret immediately invalidates all active session tokens, abruptly logging out thousands of mobile drivers, POS cashiers, and warehouse operators worldwide.
- **Architectural Consideration**: How should the token verification subsystem support dual-secret keyrings (`JWT_SECRET_CURRENT`, `JWT_SECRET_PREVIOUS`) or an internal JWKS endpoint with cryptographic key IDs (`kid`) and asymmetric RSA/ECDSA signing to enable seamless rolling key rotation?

### 3. Distributed Multi-Tenant Fleet & Inventory Isolation vs. 3PL Pooling
- **Problem**: Current vehicle and driver schemas enforce strict `SupplierId` tenancy. However, in emerging markets, 3PL logistics carriers and private vehicle fleets frequently serve multiple non-competing suppliers concurrently on the same delivery run.
- **Architectural Consideration**: How should vehicle capacity (`MaxVolumeVU`), shift schedules, and route waypoints be partitioned in Spanner and OR-Tools solvers to support multi-supplier shared 3PL capacity without leaking customer order metadata between competing suppliers?

### 4. High-Volume Ephemeral Telemetry Ingestion vs. Spanner Commit Saturation
- **Problem**: When 10,000 active delivery drivers report GPS location pings every 1–3 seconds, writing throttled updates to Spanner `OutboxEvents` (even at 5-second intervals) creates 2,000 writes/sec, saturating Spanner mutation limits and generating lock contention on the `OutboxEvents` index.
- **Architectural Consideration**: Should ephemeral driver GPS telemetry bypass Cloud Spanner entirely and stream exclusively through Redis Streams / Kafka partitioned by DriverId to WebSocket hubs, with Spanner only recording durable milestone events (e.g. `GEOFENCE_ARRIVED`, `STOP_DELIVERED`)?

### 5. Multi-Supplier Distributed Saga Coordination & Crash Recovery
- **Problem**: Multi-supplier parent checkouts (`unified_checkout.go`) split carts into child orders across suppliers. If the backend process crashes mid-loop between creating child order 1 and child order 2, the in-memory compensation routine (`compensateParentCheckout`) is lost, stranding orphan child orders in `PENDING` status.
- **Architectural Consideration**: How should an Outbox-backed Saga Coordinator with durable state tracking in `ParentOrders` and a timeout-based Dead Letter Queue (DLQ) sweeper be structured to guarantee atomic all-or-nothing checkout compensation across process crashes?

### 6. Strict Lot FEFO Depletion vs. Warehouse Travel Distance Optimization
- **Problem**: Strict First-Expired-First-Out (FEFO) picking mandates picking the lot with the earliest expiry date, which may be located in Aisle Z, Bin 40, while an identical SKU with 2 days later expiry is located in Aisle A, Bin 1. Strict FEFO multiplies picker walking distances and slows dispatch waves.
- **Architectural Consideration**: How should the WMS picking engine balance perishable shelf-life thresholds (e.g. "pick earliest lot within 14 days of expiry, but allow nearest bin within a 30-day tolerance window") with 2D S-shape travel distance optimization?

### 7. Multi-Tier Bill of Materials (BOM) Dynamic Allocation & Component Exhaustion
- **Problem**: When two manufacturing production batches requiring a shared raw material are scheduled concurrently, the absence of row-level locks or pessimistic reservation queues causes both batches to evaluate available stock simultaneously, leading to material exhaustion mid-production.
- **Architectural Consideration**: What transaction isolation and reservation protocol should be enforced when expanding multi-tier BOM recipes to guarantee atomic material allocation across parallel factory lines?

### 8. Offline Driver / POS Reconciliation Race Conditions with Concurrent Online Mutations
- **Problem**: A driver completes deliveries offline in a basement depot, scanning QR codes and collecting cash. Concurrently, the retailer contacts customer support and cancels the order online. When the driver re-establishes connectivity and syncs, the offline physical delivery conflicts with the online cancellation.
- **Architectural Consideration**: What deterministic conflict resolution hierarchy (e.g. physical goods custody transfer vs financial authorization) should govern offline sync conflict resolution, and how are compensatory financial adjustments recorded?

### 9. Double-Entry Accounting Invariance & Multi-Leg Split Tender Settlement
- **Problem**: An order is delivered with split tender (e.g. 60% Card, 30% Cash, 10% damaged item credit note). Driver cash collections undergo shift reconciliation with potential shortages, card payments clear through external PSP gateways with interchange fees, and credit notes adjust AR balances.
- **Architectural Consideration**: How should `PaymentLedgerEntries` be migrated from a single-entry event stream to a formal double-entry general ledger (Debit/Credit asset, liability, revenue, and expense accounts) to guarantee balance invariance across partial deliveries and chargebacks?

### 10. Hardware Gateway Identity, Device Attestation & Offline Cryptographic Locks
- **Problem**: Smart loading docks, cold-chain trucks, and depot locker cages operate with hardware microcontrollers in environments with intermittent connectivity.
- **Architectural Consideration**: What asymmetric cryptographic challenge-response protocol (e.g. BLE Time-based One-Time Password / TOTP or TPM-signed lease tokens) should be deployed so hardware controllers can verify driver and warehouse personnel authorization offline without real-time backend round-trips?

### 11. Kafka Topic Re-Partitioning & Monotonic Consumer Offset Integrity
- **Problem**: When Kafka topic partitions scale from 12 to 48 partitions under traffic growth, key-to-partition hashes change. In-flight messages for an entity may be split across old and new partitions, breaking FIFO ordering during partition rebalance.
- **Architectural Consideration**: How should partition worker pools handle partition rebalances and per-partition pausing to prevent monotonic offset commit data loss when individual message handlers encounter transient failures?

### 12. Poison Pill Isolation & Automated DLQ Replay Safety
- **Problem**: When an event with corrupted payload triggers a panic or unrecoverable handler error, it is moved to `OutboxDeadLetters`.
- **Architectural Consideration**: What tooling, schema validation, and tenant-isolated replay workflows must be established to inspect, patch, and safely replay dead-lettered events without re-poisoning consumer worker pools or generating duplicate downstream side effects?

---

## 11. Prioritized Remediation Roadmap & Strategy

```
+-----------------------------------------------------------------------------------+
|                           PEGASUSX REMEDIATION PHASES                             |
+-----------------------------------------------------------------------------------+
| Phase 0 (P0): Build Blockers & Fatal Schema Aborts (Immediate 24-48h)             |
|   - Fix undefined `StatusDraft` in `order/service.go:1321`                        |
|   - Fix test suite mocks in `ws/hub_test.go:119`                                  |
|   - Add mandatory `SupplierId` to all raw `OutboxEvents` mutations                |
|   - Fix fatal schema mismatches in `driver/rescue.go` and `eta/service.go`        |
|   - Populate NOT NULL columns in `warehouse/receive_items.go`                     |
|   - Fix `coldchain.go` inverse quarantine bug                                     |
+-----------------------------------------------------------------------------------+
| Phase 1 (P0/P1): Security, Multi-Tenant Isolation & Auth (Week 1)                |
|   - Enforce non-empty `home_cell` in `auth/cell_isolation.go`                     |
|   - Require step-up / prevent secret overwrite in `mfa/service.go`                |
|   - Scope supplier/factory user phone lookups by `SupplierId`                     |
|   - Map OIDC ID token claims to roles in `orgoidc/service.go`                     |
|   - Pre-hash admin password to prevent bcrypt DoS in `platformadmin/login.go`     |
|   - Fix driver PIN wipe on REST update in `driver/repository_crud.go`             |
|   - Validate QR delivery token driver ownership in `telemetryroutes/routes.go`    |
+-----------------------------------------------------------------------------------+
| Phase 2 (P1): Transactional Integrity, Inventory & Financial Ledgers (Week 2)     |
|   - Release stock reservations on early route completion (`supplier_ops.go`)      |
|   - Preserve `QuantityReserved` during spreadsheet imports (`import_sessions.go`) |
|   - Fix zero-amount chargeback reversal ledger entries (`payment/service.go`)     |
|   - Fix invoice write-off balance clear in `ar/treasury_hub.go`                   |
|   - Make payout batch calculation and slice update atomic in `payout/payout.go`   |
|   - Fix `WebhookInbox` JSON scanning into string/NullJSON                         |
|   - Fix driver cash shift filtering by `CapturedAt` in `cashrecon`                |
+-----------------------------------------------------------------------------------+
| Phase 3 (P1/P2): Realtime Engine, Kafka & WebSocket Hub (Week 3)                  |
|   - Fix multi-user retailer WebSocket subscription to org ID (`ws/handler.go`)    |
|   - Fix Kafka workerpool monotonic offset commit data loss on DLQ failure         |
|   - Restore aggregate-level partition keys in `outbox/relay.go`                   |
|   - Implement async write queues in `ws/hub.go` to eliminate HoL blocking         |
|   - Clean up ghost WebSocket broadcast room names across all domain packages      |
|   - Eliminate full-table read/rewrite in `payload/apply.go` and `factory/apply.go`|
+-----------------------------------------------------------------------------------+
| Phase 4 (P2/P3): Client Contract Parity, Hardware & Optimization (Week 4)         |
|   - Fix Expo Terminal (`/v1/payload/scan`), Android, and iOS route drift          |
|   - Unify `InventoryLevels` and `SupplierInventoryV2` single source of truth      |
|   - Standardize GS1 SSCC serial number calculations                               |
|   - Implement multimodal IoT sensor telemetry pipelines (shock, tilt, tamper)     |
|   - Implement hardware device certificate / mTLS authentication                   |
+-----------------------------------------------------------------------------------+
```

---

## 12. Verification & Regression Testing Guide

To independently verify the identified findings and validate fixes across all 8 tracks:

```bash
# 1. Verify Go Backend Compilation & Package Build Integrity
cd /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go
go build ./...

# 2. Run Comprehensive Unit & Race Condition Tests Across All Touched Packages
go test -v -race ./auth/... ./bootstrap/... ./mfa/... ./platformadmin/... \
  ./order/... ./supplier/... ./factory/... ./catalog/... ./pricing/... \
  ./warehouse/... ./stocklots/... ./retailer/... ./returns/... ./credit/... \
  ./driver/... ./dispatch/... ./routing/... ./eta/... ./payload/... \
  ./payment/... ./ar/... ./creditnote/... ./cashrecon/... ./payout/... ./tax/... \
  ./outbox/... ./kafka/... ./ws/... ./telemetry/...

# 3. Verify Offline Schema Drift & Cloud Spanner DDL Parity
go run ./cmd/schema-drift -offline

# 4. Verify Cross-Role Contract Generation & Linter Gates
make gen-contracts-gate

# 5. Run Multi-Tenant Role Matrix Verification on Spanner Emulator
SPANNER_EMULATOR_HOST=localhost:9010 go run ./cmd/verify-multitenant
```
