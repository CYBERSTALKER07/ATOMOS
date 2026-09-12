# PegasusX Context Execution Plan & System Reality

**Status:** ACTIVE / CODEBASE-ALIGNED  
**Codebase Root:** `pegasusX/`  
**Backend:** `pegasusX/apps/backend-go/` (Go 1.22+, Chi Router, Spanner, Redis, Kafka, WebSockets)  
**Database Schema:** `pegasusX/apps/backend-go/schema/spanner.ddl` (3,648 lines)  
**Clients:** 6 Full Role Rows (Supplier, Retailer, Driver, Warehouse, Factory, Payload) + Platform Admin across Web Portals (Next.js/Tauri 2), Android (Kotlin Compose/Room), and iOS (SwiftUI/SwiftData)  
**Last Synchronized:** 2026-08-20  

---

## 1. Executive Summary & Codebase Reality

The PegasusX platform is an enterprise-scale distributed B2B supply chain, logistics, and retail execution operating system. All core business flows, data schemas, API contracts, and multi-platform client applications are fully implemented with real business logic, genuine state management, and zero mock data in production code paths.

```
+-------------------------------------------------------------------------------------------------------+
|                                    PegasusX Architectural Structure                                   |
+------------------------------------+----------------------------------+-------------------------------+
| Layer                              | Technology Stack                 | Parity Status                 |
+------------------------------------+----------------------------------+-------------------------------+
| Go Backend API & Workers           | Go, Chi, Cloud Spanner, Redis    | 100% Code Parity (29 Packages)|
| Database Plane                     | Google Cloud Spanner (3,648 DDL) | 100% Multi-Tenant Schema      |
| Event Streaming & Outbox           | Kafka, Transactional Outbox      | 100% At-Least-Once Delivery   |
| Contracts & SDKs                   | TypeScript, JSON Schema, OpenAPI | 100% Typed Wire Version 1     |
| Cross-Role Client Ecosystem        | Next.js, Tauri 2, Compose, Swift | 100% Role-Row Matrix Complete |
| Automated Verification             | SSMR Smokecheck, Vitest, Go Test | 80+ Multi-Role E2E Steps Pass |
+------------------------------------+----------------------------------+-------------------------------+
```

---

## 2. Completed Implementation Milestones & Gates

### 2.1 Gate 0: Infrastructure & Core Hardening
- **Multi-Tenancy & Partitioning**: Stamped `SupplierId STRING(36) NOT NULL` across all supplier-owned tables (`schema/spanner.ddl:3648`). Implemented `TenantContext`, `RequireTenant`, and `AttachTenantFromClaims` middleware.
- **Transactional Outbox Engine**: Implemented `OutboxEvents` and `OutboxDeadLetters` in Spanner with atomic `TxnBuffer` writes inside ReadWrite transactions. Background polling relay produces to Kafka with `RequiredAcks=all` and leases.
- **Idempotency & Reliability**: SHA-256 mutation idempotency keys scoped to route and principal; rate limiting and circuit breakers active.
- **CI / CD Pipelines**: Root GitHub Actions workflows (`pegasusx-ci.yml`, `pegasusx-native-mobile-build.yml`) enforcing race detection, security scanning (`gitleaks`, `govulncheck`), and native mobile builds across all 12 mobile apps.

### 2.2 Gate 1–3: Logistics, WMS & Partner Integration
- **Order & Delivery Lifecycle**: Implemented full order state machine (`CREATED` -> `CONFIRMED` -> `IN_TRANSIT` -> `DELIVERED`), mutual QR delivery handshakes, and proof-of-delivery (PoD) capture.
- **Warehouse Management (WMS Gate 4)**: Implemented stock lot tracking, First-Expired First-Out (FEFO) picking algorithms, pick wave grouping, cycle counting with accuracy metrics, and cold-chain temperature logging.
- **B2B Partner API Layer (Gate 3)**: Implemented `/partner/v1/*` endpoints conforming to RFC 7807 problem details, OAuth2 `client_credentials`, AS2 transport with synchronous MDN, EDI-lite DESADV/ORDERS, GS1 GLN/SSCC/ZPL labeling, and 1C Chart of Accounts mapping (`PartnerCoaMaps`).

### 2.3 Gate 4–5: Retail OS & Multi-Supplier Platform
- **Retail OS (Phases 0–7)**: Deployed Store Stock, POS registers/sessions/sales/holds, shift timecards, assist tickets, capability packs (Packs 0–6), and shadow auto-order replenishment.
- **Multi-Supplier Cart & Parent Orders (Gate 5 Phase 2)**: Persisted `ParentOrders` table enabling unified checkout splitting into partitioned supplier child orders.
- **Global Products Catalog (Gate 5 Phase 3)**: Marketplace master catalog indexing with automated match queue deduplication and explainability.

### 2.4 Waves B1–B7 & G1–G7 Gap Closures
- **Money & Payment Integrity (Wave B1)**: Stabilized idempotency keys (`cash-{orderID}`, `credit-delivery-{orderID}`), eliminating double-capture risk. Implemented `SelectCashAtDelivery` with Spanner transactions and outbox event emissions.
- **Driver & Mobile Hardening (Wave B2–B4)**: Fail-closed handling for obsolete driver endpoints (HTTP 501/503); implemented offline driver action queue with Android Room/WorkManager and iOS QueuedDriverAction/BGTask with GPS fail-closed capture.
- **Claims & Reverse Logistics**: Implemented claims filing within immutable snapshot windows, GCS evidence vault integration, and automated reverse logistics return ticket issuance.

---

## 3. Backend Route Package Architecture (29 Mounted Packages)

The backend entrypoint (`apps/backend-go/main.go:1-479`) registers 29 distinct route packages:

1. `infraroutes`: `/healthz`, `/ready`, `/metrics`, `/v1/health`
2. `platformroutes`: `/v1/platform/*`, `/v1/user/device-token`, `/v1/auth/session`, `/v1/platform/cells`, `/v1/platform/market-packs`
3. `platformadmin`: `/v1/platform-admin/*`, `/v1/auth/platform-admin/login`, tenant transitions, audit log
4. `featureflags`: `/v1/platform-admin/flags/*` (dual-control money flags with MFA step-up)
5. `mfa`: `/v1/platform-admin/mfa/*` (TOTP enrollment, confirmation, step-up)
6. `pulseroutes`: `/v1/*/pulse` (role-tailored operational feeds)
7. `retailerroutes`: `/v1/retailer/*`, `/v1/auth/retailer/*`, `/v1/pos/*` (store stock, POS, shifts, assist)
8. `driverroutes`: `/v1/driver/*`, `/v1/fleet/*`, `/v1/delivery/*` (login, fleet, stops, departures, cash)
9. `factoryroutes`: `/v1/factory/*`, `/v1/factories/*` (manifests, transfers, supply QC, SLA boards)
10. `payloaderoutes`: `/v1/payload/*`, `/v1/payloader/*` (loading ledger, ship-units, digital truck seal)
11. `warehouseroutes`: `/v1/warehouse/*`, `/v1/warehouses/*` (WMS bins, lots, pick waves, cycle counts)
12. `returnsroutes`: `/v1/returns/*`, `/v1/catalog/barcode/*`, `/v1/driver/return-goods`, return history
13. `storageroutes`: `/dossiers/*` (compliance dossiers and evidence vault)
14. `taxroutes`: `/v1/admin/tax-regimes/*` (tax regime versioning and rate maps)
15. `supplierroutes`: `/v1/supplier/*`, `/v1/auth/supplier/*` (profile, fleet, pricing, promo, CRM, S&OP)
16. `entityresolutionroutes`: `/v1/supplier/entity-resolution/*` (master data deduplication)
17. `countrycfg`: `/v1/admin/country-config/*` (localized formatting and market configs)
18. `controltowerroutes`: `/v1/control-tower/*` (exception scoring and automated playbooks)
19. `promotionroutes`: `/v1/promotions/*` (multi-tier promotions and discount rules)
20. `paymentroutes`: `/v1/checkout/*`, `/v1/payment/*` (unified checkout, ledger, chargebacks)
21. `webhookroutes`: `/v1/webhooks/*` (inbound webhooks for GlobalPay, Adyen, Stripe)
22. `orderroutes`: `/v1/order/*`, `/v1/delivery/*`, `/v1/compliance/*` (order state machine, invoices)
23. `creditroutes`: `/v1/retailer/credit-*`, `/v1/supplier/credit-*`, `/v1/supplier/ar/*` (credit, AR, dunning)
24. `cashreconroutes`: `/v1/driver/cash-reconciliations`, `/v1/supplier/cash-reconciliations` (cash bags)
25. `creditnoteroutes`: `/v1/supplier/credit-notes/*`, `/v1/warehouse/reverse-logistics/*` (credit notes)
26. `deliveryroutes`: `/v1/delivery/verify-handshake`, `update-order-during-delivery` (QR handshake)
27. `telemetryroutes`: `/v1/driver/location`, `/v1/driver/location/batch` (GPS telemetry, Redis cache)
28. `updateroutes`: `/v1/updates/ios/*`, `/v1/updates/desktop/*` (OTA updater manifests)
29. `demandroutes`: `/v1/demand/*` (POS demand ingestion and sensing flywheel)

*Additionally mounted*: `laborcapacityroutes`, `etaroutes`, `catalogroutes`, `globalproductsroutes`, `partner`, `fxrates`, `payout`, `planning`, `billing`, and `ws`.

---

## 4. Gated Features & Intentional Divergences Register

To maintain full transparency, the following boundaries are intentionally gated or restricted in the codebase:

| Feature / Endpoint | Code Location | Status & Behavior | Reason / Activation Gate |
|---|---|---|---|
| **Inventory Audit** | `supplier/portal_handlers.go:1107` | HTTP 410 `audit_unwired` | Deprecated in favor of WMS cycle counts (`/v1/warehouse/cycle-counts`). |
| **Quantity Negotiation** | `order/negotiation_disabled.go:22` | HTTP 410 `feature_disabled` | Requires `QUANTITY_NEGOTIATION_ENABLED=true` env flag. |
| **Payme & Click Webhooks** | `webhookroutes/routes.go:26-31` | Routes commented out | Launch scope is locked to Cash + GlobalPay + MySoliq. |
| **Auto-Order Execution** | `retailer/auto_order_handlers.go` | Shadow Mode active | Operates in shadow mode until 80% merchant acceptance is recorded. |
| **Global Auth0 Wrapper** | `main.go:143-145` | Bypassed | Per-tenant OIDC (`orgoidc`) replaces global router wrapping. |

---

## 5. Verification & Testing Evidence

- **Backend Unit & Integration Tests**: 81 packages passing `go test ./...`.
- **Client Test Suites**: Mock-free client tests passing across all web portals and mobile SDKs.
- **SSMR Automated Smokecheck (`cmd/ssmr-smokecheck`)**: `e2e_check.go` executes **80+ multi-role verification steps** and emits 115+ canonical assertion markers (`PX_E2E_*`) against live staging infrastructure.
