# PegasusX Parity Ledger (Intentional Divergences & Verified Closures)

**Last Updated:** 2026-08-20  
**Status:** ACTIVE / CODEBASE-ALIGNED  
**Scope:** Backend Go Services, Spanner DDL Schema, Cross-Role Contracts, Client Applications  

---

## 1. Summary of Verified Parity Closures

Every item listed below has been verified against genuine source code implementations in `pegasusX/`:

### 1.1 Backend Parity & Money Path Closures (Waves B1–B7 & G1–G7)
| Feature / Subsystem | Status | Verification Evidence & Implementation Details |
|---|---|---|
| **Stable Payment Idempotency Keys** | **Wired** | Eliminated `newID()` in payment legs; keys are strictly deterministic (`cash-{orderID}`, `credit-delivery-{orderID}`). Located at `order/service.go` and `order/driver_edges.go`. |
| **Fail-Closed Driver Mutators** | **Wired** | Replaced silent/fake driver mutators (`PATCH /v1/orders/{id}/state` compat) with fail-closed handlers (HTTP 501/503), forcing clients onto transactional `transitionDriverOrder` paths. |
| **WMS Stocklot Outbox Emissions** | **Wired** | Every stocklot adjustment, receive, and bin transfer in `stocklots/` emits `STOCK_*` / `INVENTORY_*` events in the same Spanner ReadWriteTransaction as domain mutations. |
| **Retailer Multi-User Organization Auth** | **Wired** | Replaced raw `claims.Subject` lookups with `ResolveRetailerOrgID`, properly scoping orders, carts, and checkout to retailer organizations. |
| **Cash Checkout with Spanner Truth** | **Wired** | Retailer cash checkout routes through `SelectCashAtDelivery`, creating a durable `PENDING_CASH` session in Spanner and queuing outbox events. |
| **Multi-Supplier ParentOrders Outbox** | **Wired** | Multi-supplier checkout atomically inserts `ParentOrders` and child `Orders` and emits `PARENT_ORDER_CREATED` in a single Spanner transaction. |
| **Payload Seal Manifest & Mutex** | **Wired** | Mounted `payloaderoutes` (`main.go:51,222`), resolved seal mutex contention, and ensured `MANIFEST_SEALED` events are emitted to `PayloaderHub`. |
| **Factory Home-Node Scope** | **Wired** | Factory handlers dynamically resolve factory nodes from JWT claims (`HomeNodeID`), removing demo-factory pin fallbacks. |
| **Platform Admin MFA Step-Up** | **Wired** | Enforced TOTP step-up challenges (`mfa.RequireStepUp`) on dual-control flag updates, partner key provisioning, dunning execution, and billing invoices. |
| **Partner API & OAuth2 Client Credentials** | **Wired** | `POST /partner/v1/oauth/token` validates `PartnerApiKeys` and mints short-lived HS256 JWTs (`token_use=partner_access`). Emits RFC 7807 problem details on errors. |
| **Partner AS2 Transport & 1C Chart of Accounts** | **Wired** | `POST /partner/v1/as2` handles synchronous MDN document exchange; `PartnerCoaMaps` enables configurable tenant accounting journals. |
| **Forecast Accuracy & Safety Stock Replay** | **Wired** | `cmd/planning-forecast` and `POST /v1/admin/planning/safety-stock/replay` provide WAPE accuracy metrics and 90-day fill-rate simulation. |

---

## 2. Client Application Parity Closures

| Client Surface | Status | Verified Implementation Details |
|---|---|---|
| **Driver Offline Action Queue** | **Wired** | Android Room + WorkManager and iOS QueuedDriverAction + BGTask orchestrator. Preserves capture-time GPS coordinates, enforces 4.1 flush order, and records unprocessable actions to a local dead-letter table. |
| **Retailer Authorize Bypass Photo** | **Wired** | Retailer Web Desktop, Android, and iOS capture and upload evidence photos to GCS when executing authorization bypasses. |
| **Supplier & Warehouse Return Policy Management** | **Wired** | Supplier and Warehouse Web Portals, Android, and iOS apps configure return policies and grace windows via `@pegasusx/api-client`. |
| **Driver Proof-of-Delivery Photo Gate** | **Wired** | Driver Android and iOS apps mandate photographic evidence capture before allowing credit-leave delivery completion. |
| **Empty Analytics & Spend Charts** | **Closed** | Obsolete demo chart injectors removed; analytics shells render honest empty states or genuine API data feeds. |
| **Multilingual i18n Catalogs** | **Wired** | Web portals and mobile apps consume structured translations (`en`, `ru`, `uz`) with dynamic interpolation placeholders. |

---

## 3. Register of Intentional Divergences & Gated Features

The following features represent intentional design divergences, feature gates, or operational stubs:

| Feature Name | Current State | Code Location | Exit / Activation Criteria |
|---|---|---|---|
| **Quantity Negotiations** | **Product-Deferred (HTTP 410)** | `order/negotiation_disabled.go:22-30` | Enabled when `QUANTITY_NEGOTIATION_ENABLED=true` is set. Clients currently receive HTTP 410 `feature_disabled`. |
| **Inventory Audit Endpoint** | **Product-Deferred (HTTP 410)** | `supplier/portal_handlers.go:1107-1118` | Returns HTTP 410 `audit_unwired`. Replaced by WMS cycle counting and store stock audit sessions. |
| **Payme & Click Webhooks** | **Disabled in Routes** | `webhookroutes/routes.go:26-31` | Routes commented out in router. Initial launch scope is locked to Cash + GlobalPay + MySoliq. |
| **Soliq OFD Fiscal Signer** | **Stub / Pegasus Default** | `apps/backend-go/bootstrap/config.go` | Defaults to `FISCAL_PROVIDER=PEGASUS`. Requires live PKCS#12 certificate volume for `FISCAL_PROVIDER=SOLIQ`. |
| **Retailer Auto-Order Execution** | **Shadow Mode Active** | `retailer/auto_order_handlers.go` | Executes in `shadow` mode, recording recommendations in `AutoOrderShadowLedger` until 80% merchant acceptance is achieved. |
| **Offline POS Mode** | **Product-Deferred** | `retailer/pos_handlers.go` | POS requires active network connectivity in v1. Offline SQLite sync deferred to Retail OS Phase 8. |
| **Global Auth0 Router Wrapping** | **Bypassed** | `main.go:143-145` | Bypassed in favor of native HS256 tokens and per-supplier OIDC (`orgoidc` package). |
