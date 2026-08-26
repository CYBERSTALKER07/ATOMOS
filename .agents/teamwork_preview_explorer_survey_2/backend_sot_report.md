# Backend & Contracts Codebase Source-of-Truth (SoT) Report

**Generated Date**: 2026-08-20T17:28:30+05:00  
**Inspector**: Explorer 2 (Backend & Contracts Codebase SoT Inspector)  
**Target Subsystem**: `pegasusX/apps/backend-go/`, `pegasusX/schema/spanner.ddl`, `pegasusX/contracts/`, `pegasusX/packages/types/`, `pegasusX/packages/api-client/`, `pegasusX/apps/backend-go/cmd/ssmr-smokecheck/`

---

## 1. Executive Summary

This report documents the genuine, verified state of the pegasusX backend Go service, Cloud Spanner database schema, cross-role contracts, client SDK packages, and automated end-to-end verification suites.

Every finding below is based directly on code inspection and test execution during this session.

---

## 2. Spanner Schema Architecture (`pegasusX/apps/backend-go/schema/spanner.ddl`)

The authoritative database schema resides at `pegasusX/apps/backend-go/schema/spanner.ddl` (3,648 lines).

### Key Architectural Characteristics:
1. **Multi-Tenancy & Partitioning**:
   - Every core supplier-owned table contains `SupplierId STRING(36) NOT NULL` (e.g., `Suppliers`, `Orders`, `SupplierProfiles`, `SupplierPricingRules`, `SupplierPromotions`, `Warehouses`, `Factories`, `SupplierTruckManifests`).
   - Supports per-supplier OIDC configuration via `SupplierOIDC` (`spanner.ddl:25-34`).
   - `ParentOrders` table (`spanner.ddl:221-239`) enables multi-supplier checkout carts partitioned across distinct supplier orders.
2. **Transactional Outbox Storage**:
   - `OutboxEvents` table (`spanner.ddl:679-697`) with columns `EventId STRING(36)`, `AggregateType STRING(64)`, `AggregateId STRING(64)`, `EventType STRING(64)`, `TopicName STRING(64)`, `SupplierId STRING(36)`, `Payload BYTES(MAX)`, `CreatedAt TIMESTAMP`, `PublishedAt TIMESTAMP`.
   - Index `Idx_OutboxEvents_Unpublished` (`spanner.ddl:696`) indexes `(PublishedAt, CreatedAt)` with `NULL_FILTERED` for polling relay queries.
   - `OutboxDeadLetters` table (`spanner.ddl:698-709`) stores unpublishable poison messages with retry counts and last error details.
3. **Core Domain Subsystems Covered**:
   - **Order & Fulfillment**: `Orders`, `ParentOrders`, `OrderDeliveryProofs`, `OrderConditionReports`, `SupplierTruckManifests`, `ManifestOrders`, `ManifestShipUnits`, `ShopClosedAttempts`, `NegotiationProposals`.
   - **Warehouse & WMS**: `Warehouses`, `WarehouseCoverageCells`, `WarehouseCoverageCities`, `WarehouseBins`, `WarehouseLots`, `WarehousePickWaves`, `WarehousePickTasks`, `WarehouseCycleCounts`, `WarehouseTemperatureReadings`, `WarehouseSupplyRequests`, `WarehouseSupplyRequestItems`.
   - **Retail OS**: `Retailers`, `RetailerLocations`, `RetailerPricingOverrides`, `RetailerCreditProfiles`, `StoreStock`, `StoreStockMovements`, `StoreStockReceiveSessions`, `StoreStockCounts`, `PosRegisters`, `PosSessions`, `PosSales`, `PosHolds`, `RetailerShifts`, `RetailerTimeEntries`, `RetailerSections`, `RetailerAssistTickets`.
   - **Finance & Credit**: `PaymentConfigs`, `PaymentSessions`, `PaymentAttempts`, `PaymentChargebacks`, `PaymentReversals`, `PaymentWebhooks`, `PaymentLedgerEntries`, `ArInvoices`, `CashReconciliations`, `CreditNotes`, `ReverseLogisticsTasks`, `PayoutBatches`, `FxRates`.
   - **Partner & B2B Integration**: `PartnerApiKeys`, `PartnerOAuthClients`, `PartnerWebhooks`, `PartnerDeadLetters`, `PartnerEdiDocuments`, `PartnerEdiProfiles`, `PartnerCoaMaps`, `PartnerSftpConfigs`, `PartnerAs2Configs`, `PartnerExports`.
   - **Platform & Governance**: `PlatformTenants`, `PlatformFeatureFlags`, `PlatformAdminUsers`, `PlatformAdminMFA`, `PlatformAdminAudit`, `TaxRegimes`, `Notifications`, `AuditLog`.

---

## 3. Backend Go Services & Routes (`pegasusX/apps/backend-go/`)

The backend entrypoint lives at `pegasusX/apps/backend-go/main.go` (479 lines). It is organized cleanly into modular route packages (*routes), domain packages, and a centralized `bootstrap.NewApp` lifecycle manager.

### 3.1 Route Package Inventory & Verified Endpoints:

| Route Package | Path / Mount Point | Primary Handlers / Purpose | Evidence (`file:line`) |
|---|---|---|---|
| `supplierroutes` | `/v1/supplier/*`, `/v1/auth/supplier/*` | Registration, profile, topology, fleet, pricing rules, promotions, inventory import, AI recommendations, exception resolution, CRM, S&OP planning. | `supplierroutes/routes.go:71-270` |
| `retailerroutes` | `/v1/retailer/*`, `/v1/auth/retailer/*`, `/v1/pos/*` | Multi-org auth, Retail OS capabilities, store stock, POS sales/holds, shifts, sections, assist tickets, auto-order with shadow mode, loyalty. | `retailerroutes/routes.go:37-287` |
| `warehouseroutes` | `/v1/warehouse/*`, `/v1/warehouses/*` | Warehouse CRUD, WMS bins/lots/pick-waves/cycle-counts, temperature logs, dispatch execute/preview/rescue, supply requests. | `warehouseroutes/routes.go:28-205` |
| `driverroutes` | `/v1/driver/*`, `/v1/fleet/*`, `/v1/delivery/*` | Driver login/profile, fleet orders, route geometry, departures, cash bags, delivery execution, arrival, rescue requests. | `driverroutes/routes.go:41-129` |
| `factoryroutes` | `/v1/factory/*`, `/v1/factories/*` | Factory CRUD, loading-bay manifests, transfers, supply request QC & fulfillment options, SLA boards. | `factoryroutes/routes.go:23-100` |
| `payloaderoutes` | `/v1/payload/*`, `/v1/payloader/*` | Loading ledger scanning, ship-units, barcode labels, variance approvals, truck seal/unseal. | `payloaderoutes/routes.go:21-74` |
| `orderroutes` | `/v1/order/*`, `/v1/delivery/*`, `/v1/compliance/*` | Order creation, status transitions, branded receipts (HTML/PDF), claims adjudication, timeline, QR validation. | `orderroutes/routes.go:24-99` |
| `paymentroutes` | `/v1/checkout/*`, `/v1/payment/*` | B2B checkout, unified checkout, payment ledger, chargeback & reversal, payer CRUD. | `paymentroutes/routes.go:19-68` |
| `creditroutes` | `/v1/retailer/credit-*`, `/v1/supplier/credit-*`, `/v1/supplier/ar/*` | Credit profile, credit policy relationships, AR invoices, aging summaries, write-offs, dunning runner. | `creditroutes/routes.go:24-73` |
| `cashreconroutes` | `/v1/driver/cash-reconciliations`, `/v1/supplier/cash-reconciliations` | Driver cash bag turn-in, variance submission, supplier accept/write-off. | `cashreconroutes/routes.go:15-35` |
| `creditnoteroutes`| `/v1/supplier/credit-notes/*`, `/v1/warehouse/reverse-logistics/*` | Manual and auto credit notes, issuance, reverse logistics receiving. | `creditnoteroutes/routes.go:14-32` |
| `returnsroutes` | `/v1/returns/*`, `/v1/catalog/barcode/*`, `/v1/driver/return-goods` | Inbound return sessions, barcode lookup, return confirmation. | `returnsroutes/routes.go:18-64` |
| `partner` | `/partner/v1/*`, `/v1/admin/partner-keys/*` | B2B Partner API, OAuth client_credentials, AS2 receive, EDI documents, SFTP, webhooks, 1C adapters. | `partner/routes.go:11-120` |
| `platformroutes` | `/v1/platform/*`, `/v1/user/device-token`, `/v1/auth/session` | Client policies, market packs, cells, media upload tickets, tenant registration. | `platformroutes/routes.go:25-56` |
| `platformadmin` | `/v1/platform-admin/*` | Tenant transitions, feature flags, audit log, outbox ops summaries. | `platformadmin/handlers.go:176-200` |
| `featureflags` | `/v1/platform-admin/flags/*` | Dual-control flag evaluation, pending override approval, audit logging. | `featureflags/handlers.go:165-181` |
| `mfa` | `/v1/platform-admin/mfa/*` | TOTP enrollment, confirmation, verification, and step-up enforcement. | `mfa/handlers.go:139-192` |
| `controltowerroutes` | `/v1/control-tower/*` | Scored exceptions, playbooks, playbook execution runs, automated evaluation. | `controltowerroutes/routes.go:16-33` |
| `demandroutes` | `/v1/demand/*` | Demand signal ingest, adjustments, POS demand flywheel integration. | `demandroutes/routes.go:18-42` |
| `laborcapacityroutes` | `/v1/labor-capacity/*` | Driver scores, zone capacities, driver availability scheduling. | `laborcapacityroutes/routes.go:20-32` |
| `etaroutes` | `/v1/etas/*` | Realtime route ETAs, stop ETAs, recalculation triggers. | `etaroutes/routes.go:18-45` |
| `globalproductsroutes` | `/v1/global-products/*`, `/v1/admin/product-match-queue/*` | Global master catalog, product offers, match queue resolution. | `globalproductsroutes/routes.go:20-38` |
| `catalogroutes` | `/v1/catalog/*`, `/v1/products` | Public catalog browsing, categories, supplier products, media tickets. | `catalogroutes/routes.go:20-39` |
| `pulseroutes` | `/v1/*/pulse` | Role-tailored live pulse feeds (retailer, supplier, warehouse, driver, payload, factory). | `pulseroutes/routes.go:17-53` |
| `taxroutes` | `/v1/admin/tax-regimes/*` | Tax regime versioning and rate definitions. | `taxroutes/routes.go:9-19` |
| `telemetryroutes`| `/v1/driver/location`, `/v1/driver/location/batch` | High-frequency driver GPS ingest, Redis caching, throttled outbox bus emit. | `telemetryroutes/routes.go:59-150` |
| `updateroutes` | `/v1/updates/ios/*`, `/v1/updates/desktop/*` | OTA updates (iOS manifest.plist, desktop updater.json). | `updateroutes/routes.go:23-26` |
| `storageroutes` | `/dossiers/*` | Compliance dossier creation and evidence attachment vault. | `storageroutes/routes.go:28-60` |
| `infraroutes` | `/healthz`, `/ready`, `/metrics`, `/v1/health` | Liveness/readiness probes, Prometheus metrics exporter. | `infraroutes/routes.go:38-46` |
| `entityresolutionroutes` | `/v1/supplier/entity-resolution/*` | Master data deduplication, entity resolution and explainability. | `entityresolutionroutes/routes.go:15-27` |
| `webhookroutes` | `/v1/webhooks/*` | Inbound webhooks for GlobalPay, Adyen, Stripe. | `webhookroutes/routes.go:16-33` |
| `deliveryroutes`| `/v1/delivery/verify-handshake`, `update-order-during-delivery` | QR delivery handshake and live order item amendments. | `deliveryroutes/routes.go:18-25` |
| `ws` | `GET /v1/ws` | Real-time WebSocket hub routing for all 8 roles. | `ws/handler.go:34-82` |

---

## 4. Contracts & Client SDKs

1. **JSON Schema Event Registry**:
   - `pegasusX/contracts/events.schema.json` contains 6,122 lines of schema definitions.
   - Generated mechanically from `pegasusX/apps/backend-go/events/` by `cmd/gen-contracts` (`main.go:88-150`).
2. **OpenAPI Specifications**:
   - `pegasusX/contracts/jwt-core.openapi.yaml`
   - `pegasusX/contracts/partner.openapi.yaml`
3. **Ecosystem Marker Registries**:
   - `contracts/ssmr_ecosystem_markers.json` (597 lines) and `contracts/sandbox_ecosystem_markers.json` define canonical assertion markers (`PX_E2E_*`).
4. **TypeScript SDK & Contracts**:
   - `pegasusX/packages/types/index.ts` (6,682 lines) defines all DTO interfaces and wire version constants (`WireVersion = 1`).
   - `pegasusX/packages/api-client/` (3,669 lines `index.ts`) provides full HTTP client bindings, session reconciliation, idempotency headers, and cell URL pinning (`cell-api.ts:7-12`).
5. **Native Mobile Generated Contracts**:
   - Android: `PegasusWSEventEnvelope.kt` (`driver-app-android/app/src/main/java/com/pegasusx/driver/generated/contracts/PegasusWSEventEnvelope.kt`).
   - iOS: `PegasusWSEventEnvelope.swift` (`driver-app-ios/driverappios/driverappios/Generated/PegasusWSEventEnvelope.swift`).

---

## 5. Test Suite Verification & Status

Running `go test ./...` in `pegasusX/apps/backend-go` reveals:
- **81 packages pass unit and integration tests**.
- **3 specific package failures** were identified with exact root causes:

| Package | Test File & Line | Failure Type | Root Cause Details |
|---|---|---|---|
| `promotion` | `promotion/lifecycle_test.go:7:2` | Build failure | `"cloud.google.com/go/spanner" imported and not used` in test file. |
| `orgoidc` | `orgoidc/service_test.go:97, 141` | Test failure | Mock clock in `service_test.go:45` is fixed at `2026-08-16`, causing issued JWTs to be expired when evaluated against current system time (`2026-08-20`). |
| `payment` | `payment/currency_mismatch_test.go:59, 105`, `payment/execution_test.go:43` | Test failure | Tests expect live credentials or `GLOBAL_PAY_STUB_MODE=true` when initializing checkout without simulator fallback. |

### SSMR Smoke Check Suite (`cmd/ssmr-smokecheck/`):
- `cmd/ssmr-smokecheck` tests pass (`ok github.com/pegasusx/pegasusx/apps/backend-go/cmd/ssmr-smokecheck 6.853s`).
- `e2e_check.go` executes 80+ distinct multi-role verification steps against live backend configurations.

---

## 6. Detailed Gaps, Partial, Stubbed, or Gated Flows

The following areas are intentionally restricted, gated, or stubbed in the codebase:

1. **Inventory Audit Endpoint**:
   - `GET /v1/supplier/inventory/audit` at `apps/backend-go/supplier/portal_handlers.go:1107-1118` returns HTTP 410 `audit_unwired`.
   - Reason: No backend adjust/stocklot ledger reader is wired for this legacy path; clients must use standard inventory list/adjust APIs.
2. **Quantity Negotiation Feature Gate**:
   - `POST /v1/delivery/negotiate` and `POST /v1/supplier/negotiate/resolve` at `apps/backend-go/order/negotiation_disabled.go:22-30` return HTTP 410 `feature_disabled` unless `QUANTITY_NEGOTIATION_ENABLED=true` is set.
   - `GET /v1/supplier/negotiations/pending` returns an empty array `{"data": []}` by default (`order/negotiation_list.go:26-28`).
3. **Payme & Click Webhook Handlers**:
   - In `apps/backend-go/webhookroutes/routes.go:26-31`, routes for `/v1/webhooks/payme` and `/v1/webhooks/click` are commented out.
   - Reason: The launch payment path is strictly scoped to Cash + GlobalPay + MySoliq. Handlers exist in `payment` package but routes are deliberately inactive.
4. **Global Auth0 Router Wrapping**:
   - In `apps/backend-go/main.go:143-145`, `AUTH0_DOMAIN` is ignored if present.
   - Reason: GS-I per-supplier OIDC (`orgoidc` package) replaced global router wrapping to allow native HS256 and per-tenant IdP isolation.
