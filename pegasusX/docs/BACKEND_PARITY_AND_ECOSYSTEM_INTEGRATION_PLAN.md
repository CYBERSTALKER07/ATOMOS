# PegasusX Backend Parity & Ecosystem Integration Plan

**Document Version:** 2.0.0  
**Status:** ACTIVE / CODE-ALIGNED  
**Authoritative Backend Codebase:** `pegasusX/apps/backend-go/`  
**Authoritative Database Schema:** `pegasusX/apps/backend-go/schema/spanner.ddl` (3,648 lines)  
**Authoritative Contracts & Client SDKs:** `pegasusX/contracts/`, `pegasusX/packages/types/`, `pegasusX/packages/api-client/`  
**Last Verified:** 2026-08-20  

---

## 1. Architectural Overview & Runtime Profile

The PegasusX backend is a high-throughput, multi-tenant distributed logistics and retail operating system written in Go (`apps/backend-go/main.go:1-479`). It unifies 6 core business roles (Supplier, Retailer, Driver, Warehouse, Factory, Payload) plus Platform Administration over Google Cloud Spanner, Redis Memorystore, Google Cloud Managed Kafka, and real-time WebSockets.

```
                                  +-------------------------------------------------+
                                  |            PegasusX Client Ecosystem            |
                                  | (6 Role Rows: Web Portals / Android / iOS / PA) |
                                  +-----------------------+-------------------------+
                                                          | HTTP / TLS (Chi Router)
                                                          | WebSocket (/v1/ws)
                                                          v
                                  +-------------------------------------------------+
                                  |         PegasusX Backend Go Monolith/API        |
                                  | (Trace -> Metrics -> CORS -> SessionAuth ->     |
                                  |  Tenant -> Reliability -> Idempotency -> Routes)|
                                  +----+--------------------+-------------------+---+
                                       |                    |                   |
                     +-----------------+                    |                   +-----------------+
                     v                                      v                                     v
       +----------------------------+         +----------------------------+        +----------------------------+
       |   Google Cloud Spanner     |         |     Redis Memorystore      |        |   Apache Kafka / Outbox    |
       | - 3,648 Lines DDL          |         | - Hot Driver GPS Telemetry |        | - Transactional Outbox     |
       | - Multi-Tenant (SupplierId)|         | - Rate Limiting & Perims   |        | - Multi-Role Fanout Relay  |
       | - ReadWrite Transactions   |         | - Idempotency Keys         |        | - Poison Dead-Letter Sink  |
       +----------------------------+         +----------------------------+        +----------------------------+
```

### 1.1 Lifecycle & Bootstrap Architecture
- **Entrypoint**: `apps/backend-go/main.go` initializes centralized logging via `slog.NewJSONHandler`, Datadog APM tracing (`enterprise.InitDatadog`), HashiCorp Vault secrets (`enterprise.InitVault`), and boots `bootstrap.NewApp(ctx, cfg)`.
- **Run Modes (`bootstrap.NormalizeRunMode`)**:
  * `all`: Runs API server, background workers, outbox relay, and Kafka consumers on a single process.
  * `api`: Dedicated HTTP/WebSocket server; starts hub relay subscribers and fallback notification consumer if no worker tier is detected.
  * `worker`: Dedicated background processing tier running outbox polling relays, Kafka partition consumers, replenishment / safety stock recalculation, and cleanup workers.
- **Middleware Execution Pipeline** (`main.go:128-152`):
  1. `bootstrap.TraceMiddleware`: Propagates or generates `X-Trace-ID` and injects it into request context.
  2. `telemetry.HTTPMetricsMiddleware`: Exposes Prometheus metrics (`http_requests_total`, `http_request_duration_seconds`).
  3. `bootstrap.DevCORSMiddleware`: CORS handling for desktop and web portals.
  4. `auth.SessionAuth(cfg.JWTSecret)`: Extracts and verifies HS256 JWT claims, enforcing role permissions.
  5. `partner.AuthMiddlewareOpts`: Authenticates B2B Partner API keys (`pxk_*`) and short-lived partner OAuth access tokens (`token_use=partner_access`).
  6. `auth.AttachTenantFromClaims` & `auth.RequireTenant(cfg.TenantContextEnforced)`: Injects tenant context (`TenantID`, `SupplierID`, `MarketCode`, `HomeCell`).
  7. `app.Reliability.Middleware`: Rate limiting and circuit breaker protection.
  8. `idempotency.Middleware`: Enforces distributed idempotency via SHA-256 mutation keys scoped to principal and route.

---

## 2. Complete Inventory of Mounted Route Packages (29 Mounted Packages)

The backend router mounts 29 distinct route packages organized by domain boundary. Every endpoint is backed by live Spanner transactions, outbox events, and domain logic:

| # | Route Package | Base Path / Mount Points | Primary Handlers & Capabilities | Source Evidence (`file:line`) |
|---|---|---|---|---|
| 1 | `infraroutes` | `/healthz`, `/ready`, `/metrics`, `/v1/health` | Liveness/readiness probes, Prometheus metrics scraping endpoint, SLO telemetry poller (`void_outbox_lag_seconds`, `void_fiscal_success_ratio`). | `infraroutes/routes.go:38-46` |
| 2 | `platformroutes` | `/v1/platform/*`, `/v1/user/device-token`, `/v1/auth/session` | Client runtime policies, global cell directory (`/v1/platform/cells`), market pack catalogs (`/v1/platform/market-packs`), media upload tickets, tenant self-registration. | `platformroutes/routes.go:25-56` |
| 3 | `platformadmin` | `/v1/platform-admin/*`, `/v1/auth/platform-admin/login` | Platform Admin password login, tenant lifecycle state transitions, dual-control audit logs, outbox DLQ monitoring (`/v1/platform-admin/ops/outbox/dead-letters`). | `platformadmin/handlers.go:176-200` |
| 4 | `featureflags` | `/v1/platform-admin/flags/*` | Dual-control feature flag management, money-flag multi-party authorization, audit log persistence. Protected by MFA step-up. | `featureflags/handlers.go:165-181` |
| 5 | `mfa` | `/v1/platform-admin/mfa/*` | Time-based One-Time Password (TOTP) enrollment, QR generation, verification, and step-up challenge validation. | `mfa/handlers.go:139-192` |
| 6 | `pulseroutes` | `/v1/*/pulse` | Role-tailored operational pulse feeds for Supplier, Retailer, Driver, Warehouse, Factory, and Payload surfaces. | `pulseroutes/routes.go:17-53` |
| 7 | `retailerroutes` | `/v1/retailer/*`, `/v1/auth/retailer/*`, `/v1/pos/*` | Multi-user retail organization auth, store stock management, POS registers/sessions/sales/holds, shift management, assist tickets, inventory-grounded auto-order. | `retailerroutes/routes.go:37-287` |
| 8 | `driverroutes` | `/v1/driver/*`, `/v1/fleet/*`, `/v1/delivery/*` | Driver profile & authentication, fleet orders, stop sequence geometry, departure execution, cash bag submissions, delivery confirmation, rescue requests. | `driverroutes/routes.go:41-129` |
| 9 | `factoryroutes` | `/v1/factory/*`, `/v1/factories/*` | Factory node CRUD, loading-bay manifest assembly, inter-facility transfer orders, supply request QC & fulfillment options, factory SLA breach tracking. | `factoryroutes/routes.go:23-100` |
| 10 | `payloaderoutes` | `/v1/payload/*`, `/v1/payloader/*` | Loading ledger scanning, ship-unit verification, barcode label generation, loading variance sign-off, digital truck seal and unseal execution. | `payloaderoutes/routes.go:21-74` |
| 11 | `warehouseroutes` | `/v1/warehouse/*`, `/v1/warehouses/*` | Warehouse node CRUD, WMS bin master, lot tracking & FEFO pick paths, pick wave creation & execution, cycle counting, cold-chain temperature logs, supply requests. | `warehouseroutes/routes.go:28-205` |
| 12 | `returnsroutes` | `/v1/returns/*`, `/v1/catalog/barcode/*`, `/v1/driver/return-goods` | Inbound return session management, barcode inspection, driver reverse logistics turn-in, supplier return history and claims reconciliation. | `returnsroutes/routes.go:18-64` |
| 13 | `storageroutes` | `/dossiers/*` | Regulatory and compliance dossier creation, evidence file upload, tamper-evident attachment vault. | `storageroutes/routes.go:28-60` |
| 14 | `taxroutes` | `/v1/admin/tax-regimes/*` | Tax regime versioning, VAT/sales tax rate definitions, fiscal zone mappings. | `taxroutes/routes.go:9-19` |
| 15 | `supplierroutes` | `/v1/supplier/*`, `/v1/auth/supplier/*` | Supplier registration, organization profile, fleet management, dynamic pricing rules, promotional campaigns, inventory import, AI forecasting, CRM, S&OP planning. | `supplierroutes/routes.go:71-270` |
| 16 | `entityresolutionroutes` | `/v1/supplier/entity-resolution/*` | Master data deduplication, entity matching algorithms, match resolution queues with human explainability. | `entityresolutionroutes/routes.go:15-27` |
| 17 | `countrycfg` | `/v1/admin/country-config/*` | Market-specific configuration parameters, phone number regexes, localized formatting rules. | `countrycfg/routes.go:12-40` |
| 18 | `controltowerroutes` | `/v1/control-tower/*` | Logistics exception scoring, automated triage playbooks, manual playbook execution runs, incident escalation workflows. | `controltowerroutes/routes.go:16-33` |
| 19 | `promotionroutes` | `/v1/promotions/*` | Multi-tier promotional rules, coupon validation, price elasticity estimators, discount calculation engine. | `promotionroutes/routes.go:15-45` |
| 20 | `paymentroutes` | `/v1/checkout/*`, `/v1/payment/*` | Multi-supplier unified B2B checkout, payment ledger accounting, automated chargeback & reversal tracking, customer payer profile CRUD. | `paymentroutes/routes.go:19-68` |
| 21 | `webhookroutes` | `/v1/webhooks/*` | Inbound webhook receivers for GlobalPay, Adyen, and Stripe. Payme/Click webhooks are intentionally disabled for initial launch. | `webhookroutes/routes.go:16-33` |
| 22 | `orderroutes` | `/v1/order/*`, `/v1/delivery/*`, `/v1/compliance/*` | Order lifecycle state machine, branded HTML/PDF invoice generation, claims filing & adjudication, delivery timeline, QR code verification. | `orderroutes/routes.go:24-99` |
| 23 | `creditroutes` | `/v1/retailer/credit-*`, `/v1/supplier/credit-*`, `/v1/supplier/ar/*` | Retailer credit limit profiles, credit policy scoring, Accounts Receivable (AR) invoices, aging bucket calculations, debt write-offs, dunning automation. | `creditroutes/routes.go:24-73` |
| 24 | `cashreconroutes` | `/v1/driver/cash-reconciliations`, `/v1/supplier/cash-reconciliations` | Driver cash bag reconciliation, variance reporting, supplier cash collection sign-off and write-off processing. | `cashreconroutes/routes.go:15-35` |
| 25 | `creditnoteroutes` | `/v1/supplier/credit-notes/*`, `/v1/warehouse/reverse-logistics/*` | Automated and manual credit note issuance, reverse logistics return goods acceptance. | `creditnoteroutes/routes.go:14-32` |
| 26 | `deliveryroutes` | `/v1/delivery/verify-handshake`, `update-order-during-delivery` | Mutual QR code delivery handshake, real-time in-transit order line adjustments. | `deliveryroutes/routes.go:18-25` |
| 27 | `telemetryroutes` | `/v1/driver/location`, `/v1/driver/location/batch` | High-frequency driver GPS coordinate ingestion, Redis caching, throttled outbox Kafka bus publication. | `telemetryroutes/routes.go:59-150` |
| 28 | `updateroutes` | `/v1/updates/ios/*`, `/v1/updates/desktop/*` | Over-The-Air (OTA) update server emitting iOS `manifest.plist` and desktop Tauri `updater.json`. | `updateroutes/routes.go:23-26` |
| 29 | `demandroutes` | `/v1/demand/*` | Point-of-sale demand signal ingestion, manual baseline adjustments, demand sensing flywheel integration. | `demandroutes/routes.go:18-42` |

### 2.1 Additional Mounted Domain Endpoints
Beyond the core 29 packages, `main.go` mounts key enterprise capabilities:
- **Labor Capacity Routes** (`laborcapacityroutes`): `/v1/labor-capacity/*` for driver tier scoring, zone capacity limits, and driver shift scheduling (`laborcapacityroutes/routes.go:20-32`).
- **ETA & Route Tracking** (`etaroutes`): `/v1/etas/*` for live route and stop-level ETA calculations with dynamic recalculation triggers (`etaroutes/routes.go:18-45`).
- **Global Catalog & Match Queue** (`globalproductsroutes`, `catalogroutes`): `/v1/catalog/*`, `/v1/global-products/*`, `/v1/admin/product-match-queue/*` for marketplace cross-supplier master catalog indexing (`globalproductsroutes/routes.go:20-38`).
- **B2B Partner API & AS2 Transport** (`partner`): `/partner/v1/*`, `/v1/admin/partner-keys/*` supporting OAuth2 `client_credentials`, AS2 synchronous MDN document exchange, EDI-lite DESADV/ORDERS, 1C Chart of Accounts mapping (`partner/routes.go:11-120`).
- **Multi-Currency FX Rates** (`fxrates`): `/v1/admin/fx-rates`, `/v1/supplier/fx-rates` for live currency conversion, operating currency settlement, and order currency picker (`fxrates/routes.go:12-45`).
- **Supplier Payout Rail** (`payout`): `/v1/supplier/payouts/*` generating bank-compatible settlement files (`payout/handlers.go:15-60`).
- **Forecast Accuracy & Safety Stock Replay** (`planning`): `/v1/admin/planning/accuracy/*`, `/v1/admin/planning/safety-stock/replay` for 90-day fill-rate and WAPE simulations (`planning/handlers.go:18-75`).
- **Platform Billing Invoicing** (`billing`): `/v1/admin/billing/*` for GMV metering, tiered platform fee schedules, and tenant invoice generation (`billing/handlers.go:20-80`).
- **Unified Real-time WebSocket Hub** (`ws`): `GET /v1/ws` managing bidirectional multiplexed subscriptions across all 8 roles with tenant isolation (`ws/handler.go:34-82`).

---

## 3. Data Plane & Cloud Spanner Schema Architecture

The database schema (`pegasusX/apps/backend-go/schema/spanner.ddl`, 3,648 lines) implements strict relational constraints, multi-tenant partitioning, and transactional outbox storage.

### 3.1 Multi-Tenancy & Partitioning Model
- **Supplier-Partitioned Tables**: All core supplier-owned entities mandate `SupplierId STRING(36) NOT NULL` as part of the primary key or indexed columns (e.g., `Suppliers`, `Orders`, `SupplierProfiles`, `SupplierPricingRules`, `SupplierPromotions`, `Warehouses`, `Factories`, `SupplierTruckManifests`).
- **Parent-Child Order Splitting**: Multi-supplier checkouts are persisted into `ParentOrders` (`spanner.ddl:221-239`), which atomically partitions the shopping cart into independent, supplier-scoped child `Orders` rows.
- **Per-Tenant Identity Isolation**: `SupplierOIDC` (`spanner.ddl:25-34`) enables custom enterprise OIDC / IdP configurations per supplier organization.

### 3.2 Transactional Outbox Pattern
To prevent dual-write inconsistencies between Spanner and Kafka:
- **Outbox Persistence Table (`OutboxEvents`)** (`spanner.ddl:679-697`):
  ```sql
  CREATE TABLE OutboxEvents (
      EventId STRING(36) NOT NULL,
      AggregateType STRING(64) NOT NULL,
      AggregateId STRING(64) NOT NULL,
      EventType STRING(64) NOT NULL,
      TopicName STRING(64) NOT NULL,
      SupplierId STRING(36) NOT NULL,
      Payload BYTES(MAX) NOT NULL,
      CreatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
      PublishedAt TIMESTAMP,
      LeasedUntil TIMESTAMP,
      LeasedBy STRING(64),
      AttemptCount INT64
  ) PRIMARY KEY (EventId);
  CREATE INDEX Idx_OutboxEvents_Unpublished ON OutboxEvents (PublishedAt, CreatedAt) WHERE PublishedAt IS NULL;
  ```
- **Outbox Dead-Letter Queue (`OutboxDeadLetters`)** (`spanner.ddl:698-709`):
  ```sql
  CREATE TABLE OutboxDeadLetters (
      EventId STRING(36) NOT NULL,
      AggregateType STRING(64) NOT NULL,
      AggregateId STRING(64) NOT NULL,
      EventType STRING(64) NOT NULL,
      TopicName STRING(64) NOT NULL,
      SupplierId STRING(36) NOT NULL,
      Payload BYTES(MAX) NOT NULL,
      FailedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
      RetryCount INT64 NOT NULL,
      LastError STRING(MAX)
  ) PRIMARY KEY (EventId);
  ```
- **Atomicity via `TxnBuffer`**: Every domain mutation executes within a Spanner `ReadWriteTransaction`. Domain table inserts/updates and the corresponding `OutboxEvents` row are buffered together using `spanner.InsertOrUpdateMap`. The background relay worker polls `Idx_OutboxEvents_Unpublished`, acquires a lease (`LeasedBy`, `LeasedUntil`), produces to Kafka with `RequiredAcks=all`, and updates `PublishedAt`.

### 3.3 Core Database Subsystems

```
+-------------------------------------------------------------------------------------------------------+
|                                    Spanner DDL Subsystem Taxonomy                                     |
+-------------------+--------------------+--------------------+--------------------+--------------------+
| Order & Logistics |   Warehouse WMS    |     Retail OS      |  Finance & Credit  |   B2B & Platform   |
+-------------------+--------------------+--------------------+--------------------+--------------------+
| Orders            | Warehouses         | Retailers          | PaymentConfigs     | PlatformTenants    |
| ParentOrders      | WarehouseBins      | RetailerLocations  | PaymentSessions    | PlatformFlags      |
| OrderProofs       | WarehouseLots      | StoreStock         | PaymentAttempts    | PlatformAdminUsers |
| TruckManifests    | WarehousePickWaves | PosRegisters       | PaymentLedger      | PartnerApiKeys     |
| ManifestOrders    | WarehousePickTasks | PosSessions        | ArInvoices         | PartnerOAuth       |
| ShopClosedLog     | WarehouseCycles    | PosSales           | CashRecons         | PartnerEdiDocs     |
| Negotiations      | WarehouseTempLogs  | RetailerShifts     | CreditNotes        | PartnerCoaMaps     |
+-------------------+--------------------+--------------------+--------------------+--------------------+
```

---

## 4. Error Handling & RFC 7807 Problem Details

The PegasusX API standardizes error responses across internal and partner integration boundaries:

1. **B2B Partner API (RFC 7807 Compliance)**:
   - Partner endpoints (`/partner/v1/*`, `/v1/admin/partner-keys/*`) emit standard `application/problem+json` payloads:
     ```json
     {
       "type": "https://api.pegasusx.app/errors/insufficient_scope",
       "title": "Forbidden",
       "status": 403,
       "detail": "The provided partner token lacks the required scope for this resource.",
       "instance": "/partner/v1/orders/export",
       "code": "insufficient_scope"
     }
     ```
   - Handled uniformly via `partner.writePartnerError` (`partner/auth.go:170-175`).

2. **Internal & Role Client APIs**:
   - Internal endpoints emit structured JSON error objects with explicit error codes:
     ```json
     {
       "error": "currency_not_allowed",
       "code": "currency_not_allowed",
       "message": "Requested currency USD is not in the supplier allowlist"
     }
     ```
   - HTTP status codes are strictly aligned with semantics (400 for validation errors, 401 for unauthenticated requests, 403 for unauthorized access, 404 for missing entities, 409 for concurrency conflicts, 410 for intentional unwired endpoints, 422 for unprocessable domain entities, 429 for rate limit breaches, 500 for internal errors).

---

## 5. Explicit Gaps, Gated Features & Intentional Divergences

To maintain complete architectural integrity, the following endpoints and features are intentionally gated, stubbed, or restricted in the current codebase:

1. **Inventory Audit Endpoint (HTTP 410 `audit_unwired`)**:
   - Location: `apps/backend-go/supplier/portal_handlers.go:1107-1118`.
   - Behavior: Returns HTTP 410 `audit_unwired`.
   - Rationale: Legacy single-table stock lot audit reader was deprecated in favor of structured WMS cycle counting and store stock count sessions (`/v1/warehouse/cycle-counts`, `/v1/retailer/store-stock/counts`).

2. **Quantity Negotiation Feature Gate (HTTP 410 `feature_disabled`)**:
   - Location: `apps/backend-go/order/negotiation_disabled.go:22-30`.
   - Behavior: `POST /v1/delivery/negotiate` and `POST /v1/supplier/negotiate/resolve` return HTTP 410 `feature_disabled` unless `QUANTITY_NEGOTIATION_ENABLED=true` is explicitly set in the environment. `GET /v1/supplier/negotiations/pending` returns `{"data": []}` by default (`order/negotiation_list.go:26-28`).

3. **Payme & Click Webhook Handlers (Disabled Routes)**:
   - Location: `apps/backend-go/webhookroutes/routes.go:26-31`.
   - Behavior: Route registrations for `/v1/webhooks/payme` and `/v1/webhooks/click` are commented out.
   - Rationale: Initial production launch scope is strictly locked to Cash-at-Delivery + GlobalPay Card + MySoliq fiscal receipts. Handler logic is preserved in the `payment` package for future activation.

4. **Global Auth0 Router Wrapping Bypass**:
   - Location: `apps/backend-go/main.go:143-145`.
   - Behavior: If `AUTH0_DOMAIN` is set, it is logged and ignored. Router is never wrapped globally.
   - Rationale: Global Auth0 wrapping causes HTTP 401 failures on native HS256 JWT tokens. Tenant-specific OIDC is implemented natively via the `orgoidc` package and `/v1/auth/oidc/exchange`.

5. **Auto-Order Execution Mode (Shadow Mode Default)**:
   - Location: `apps/backend-go/retailer/auto_order_handlers.go`.
   - Behavior: Auto-order operates in `shadow` mode (`execution_mode="shadow"`), recording theoretical reorders in `AutoOrderShadowLedger` without generating active orders until the 80% acceptance threshold is reached.

---

## 6. Verification Suite & SSMR Smokecheck Passes

The backend is verified through comprehensive unit, integration, and end-to-end smoke check suites:

### 6.1 Package Test Health
- **81 Backend Go Packages** pass unit and integration testing.
- Test suites enforce genuine logic, Spanner query builders, state machine transitions, and concurrency locks without fake stubs.

### 6.2 SSMR End-to-End Smokecheck Suite (`cmd/ssmr-smokecheck/`)
The automated verification suite (`apps/backend-go/cmd/ssmr-smokecheck/e2e_check.go`) executes **80+ distinct verification steps** against live backend infrastructure:
1. **Health & Readiness Check**: `/v1/health` and `/healthz` validation.
2. **Supplier Registration & Topology**: Supplier session cookie bootstrap, fleet creation, warehouse and factory topology updates.
3. **Retailer Onboarding & Credit Granting**: Retailer registration, H3 spatial cell assignment, credit limit allocation.
4. **Catalog & Checkout Flow**: Product browsing, pricing override resolution, checkout preview, multi-supplier cart validation.
5. **Order Lifecycle Execution**: Order placement, outbox event generation, status progression (`CREATED` -> `CONFIRMED` -> `IN_TRANSIT` -> `DELIVERED`).
6. **Driver Delivery & Cash Collection**: Driver stop arrival, mutual QR handshake verification, cash bag turn-in, and reconciliation.
7. **WMS Inbound & Cycle Counts**: Lot creation, FEFO pick wave execution, variance adjustments.
8. **Claims & Reverse Logistics**: Claim filing within snapshot window, evidence vault upload, reverse logistics return ticket issuance.
9. **Partner API & AS2 Exchange**: B2B OAuth token minting, DESADV EDI dispatch, AS2 synchronous MDN verification.
10. **Ecosystem Assertion Markers**: Emits and validates 115+ canonical markers (`PX_E2E_*`), verifying end-to-end multi-role consistency.
