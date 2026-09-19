# PegasusX: Full System Parity & Ecosystem Master Plan

**Document Version:** 2.0.0 (Living Master Specification)  
**Date:** 2026-08-20  
**Status:** Canonical Living Master Plan & Architecture Specification  
**Authoritative Tree:** `pegasusX/`  
**Governing Documents:** [`.agents/memory/GOAL.md`](../../.agents/memory/GOAL.md), [`DOCS_SOURCE_OF_TRUTH.md`](./DOCS_SOURCE_OF_TRUTH.md), [`GLOBAL_SCALE_PROGRAM.md`](./GLOBAL_SCALE_PROGRAM.md), [`GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`](./GLOBAL_SCALE_LOCAL_ECOSYSTEM.md), [`ROLE_ROW_PARITY_MATRIX.md`](./ROLE_ROW_PARITY_MATRIX.md)

---

## 1. Executive Summary & Architectural Vision

PegasusX (ATOMOS) is an enterprise-grade logistics and autonomous supply chain operating system. It orchestrates end-to-end multi-party physical trade across six core role rows (**Supplier**, **Retailer**, **Driver**, **Warehouse**, **Factory**, **Payload**) plus **Platform Admin**.

The platform is designed around strict data-plane consistency, request-scoped multi-tenancy, transactional event outbox reliability, local-first geospatial matching, and multi-platform client applications across Web, Desktop, Android, and iOS.

```
                  ┌──────────────────────────────────────────────────┐
                  │               PLATFORM ADMIN PORTAL              │
                  │   Tenants · Flags · Audit · Dead-Letters · Ops   │
                  └─────────────────────────┬────────────────────────┘
                                            │
               ┌────────────────────────────┼────────────────────────────┐
               ▼                            ▼                            ▼
  ┌──────────────────────────┐ ┌──────────────────────────┐ ┌──────────────────────────┐
  │     SUPPLIER PORTAL      │ │      FACTORY PORTAL      │ │     WAREHOUSE PORTAL     │
  │ Web · Tauri · Droid · iOS│ │ Web · Tauri · Droid · iOS│ │ Web · Tauri · Droid · iOS│
  │ Catalog·CRM·Fleet·Finance│ │ LoadingBay·Transfers·QC  │ │ WMS·PickWaves·Dispatch·CQ│
  └────────────┬─────────────┘ └────────────┬─────────────┘ └────────────┬─────────────┘
               │                            │                            │
               │               ┌────────────┴────────────┐               │
               │               ▼                         ▼               │
               │  ┌──────────────────────────┐ ┌──────────────────────┐  │
               │  │     PAYLOAD TERMINAL     │ │      DRIVER APP      │  │
               │  │  Expo · Android · iPad   │ │    Android · iOS     │  │
               │  │ Seal·ShipUnits·Barcodes  │ │ Doorstep·Cash·Nav·GPS│  │
               │  └────────────┬─────────────┘ └──────────┬───────────┘  │
               │               │                          │              │
               └───────────────┼──────────────────────────┼──────────────┘
                               │                          │
                               ▼                          ▼
                  ┌──────────────────────────────────────────────────┐
                  │                   RETAILER APP                   │
                  │              Desktop · Android · iOS             │
                  │    Catalog · Cart · Store Stock · POS · Claims   │
                  └──────────────────────────────────────────────────┘
```

---

## 2. Core Architectural Pillars

### 2.1 True Multi-Tenancy & Tenant Isolation
- **Tenant Partitioning**: All supplier-owned entities in Cloud Spanner enforce `SupplierId STRING(36) NOT NULL` (e.g., `Suppliers`, `Orders`, `SupplierProfiles`, `Warehouses`, `Factories`, `SupplierTruckManifests`, `SupplierPricingRules`).
- **Request-Scoped Isolation**: Extracted cryptographically from signed HS256 JWT claims. Zero trust for body-supplied supplier identifiers.
- **Multi-Supplier Cart Partitioning (`ParentOrders`)**: Retailers can place unified checkout carts spanning multiple suppliers in the same market pack; the backend splits the cart into isolated per-supplier child `Orders` inside a single Spanner transaction.
- **Per-Tenant Identity Provider**: Per-supplier OIDC configuration via `SupplierOIDC` (`spanner.ddl:25-34`), replacing global router auth wrappers with tenant-isolated identity boundaries.

### 2.2 Transactional State Atomicity (Transactional Outbox Pattern)
- **Zero Ghost State**: Every domain mutation that emits an event writes directly to `OutboxEvents` (`spanner.ddl:679-697`) inside the same Spanner Read-Write transaction.
- **Reliable Relay**: A background poller reads unpublished events (`Idx_OutboxEvents_Unpublished`) and publishes to Kafka partitioned by `AggregateId`.
- **Dead-Letter Recovery**: Poison messages are safely diverted to `OutboxDeadLetters` (`spanner.ddl:698-709`) with error stack traces and retry counts, manageable via the Platform Admin portal.

### 2.3 Local-First Geospatial Intelligence (H3 Resolution 7)
- **Catchment & Clustering**: Real-world locations are mapped to Uber H3 Resolution 7 hexagonal cells (~1.2 km edge length).
- **One Coverage Engine**: `proximity.ResolveServingWarehouse` evaluates country match, store-level service pins (`ServicePins`), supplier regional boundaries (`SupplierRegions`), coverage cells (`WarehouseCoverageCells`), and Haversine distance.
- **Dual Manifest Planes**: Last-mile supplier delivery manifests (`SupplierTruckManifests`) and factory-to-warehouse transfer manifests (`FactoryTruckManifests`) operate on dedicated, unmerged tables.

### 2.4 Realtime WebSocket Multi-Hub Architecture
- **8 Dedicated Hubs**: `RetailerHub`, `SupplierHub`, `DriverHub`, `PayloadHub`, `WarehouseHub`, `FactoryHub`, `TelemetryHub`, and `PlatformAdminHub`.
- **Session Reconciliation**: On reconnect, clients exchange sequence numbers to recover dirty state without full re-fetching (`@pegasusx/ws-refresh-contract`).
- **Telemetry Streaming**: Realtime GPS ingestion at `/v1/driver/location` with Redis caching, dead-reckoning interpolation, and throttled outbox bus replication.

---

## 3. Backend Go Services & 29 Route Packages Map

The backend service entrypoint lives at `pegasusX/apps/backend-go/main.go` (479 lines) and mounts 29 distinct route packages:

| # | Route Package | Base Mount Path | Key Endpoints & Capabilities | Source Evidence |
|---|---|---|---|---|
| 1 | `supplierroutes` | `/v1/supplier/*`, `/v1/auth/supplier/*` | Onboarding, catalog, pricing rules, promotions, inventory async import, S&OP planning, CRM, exceptions resolution. | `supplierroutes/routes.go:71-270` |
| 2 | `retailerroutes` | `/v1/retailer/*`, `/v1/auth/retailer/*`, `/v1/pos/*` | Retailer auth, AI demand predictions, store stock, POS sessions/sales/holds, shifts, sections, assist tickets, auto-order. | `retailerroutes/routes.go:37-287` |
| 3 | `warehouseroutes` | `/v1/warehouse/*`, `/v1/warehouses/*` | Warehouse CRUD, WMS bins/lots, pick waves, cycle counts, cold-chain temperature logs, dispatch preview/execute/rescue. | `warehouseroutes/routes.go:28-205` |
| 4 | `driverroutes` | `/v1/driver/*`, `/v1/fleet/*`, `/v1/delivery/*` | Driver auth, active missions, route geometry, cash turn-in, doorstep QR verification, rescue requests. | `driverroutes/routes.go:41-129` |
| 5 | `factoryroutes` | `/v1/factory/*`, `/v1/factories/*` | Factory management, loading-bay manifests, supply requests, stock transfers, QC fulfillment options. | `factoryroutes/routes.go:23-100` |
| 6 | `payloadroutes` | `/v1/payload/*`, `/v1/payloader/*` | Loading ledger scanning, ship-units, barcode labels, seal-all manifests, variance approvals. | `payloaderoutes/routes.go:21-74` |
| 7 | `orderroutes` | `/v1/order/*`, `/v1/compliance/*` | Order creation, status progression, branded receipts (HTML/PDF), claims adjudication, timeline. | `orderroutes/routes.go:24-99` |
| 8 | `paymentroutes` | `/v1/checkout/*`, `/v1/payment/*` | B2B unified checkout, payment ledger, chargeback & reversal recording, payer CRUD. | `paymentroutes/routes.go:19-68` |
| 9 | `creditroutes` | `/v1/retailer/credit-*`, `/v1/supplier/credit-*`, `/v1/supplier/ar/*` | Credit profile, credit policy relationships, AR invoices, aging summaries, dunning worker. | `creditroutes/routes.go:24-73` |
| 10 | `cashreconroutes` | `/v1/driver/cash-reconciliations`, `/v1/supplier/cash-reconciliations` | Cash bag turn-in, variance recording, reconciliation write-offs. | `cashreconroutes/routes.go:15-35` |
| 11 | `creditnoteroutes`| `/v1/supplier/credit-notes/*`, `/v1/warehouse/reverse-logistics/*` | Manual and auto credit notes, issuance, reverse logistics receiving. | `creditnoteroutes/routes.go:14-32` |
| 12 | `returnsroutes` | `/v1/returns/*`, `/v1/catalog/barcode/*`, `/v1/driver/return-goods` | Inbound return sessions, barcode lookups, return receipt confirmations. | `returnsroutes/routes.go:18-64` |
| 13 | `partner` | `/partner/v1/*`, `/v1/admin/partner-keys/*` | B2B Partner API, OAuth client_credentials, AS2 receive, EDI documents (INVOIC/PRICAT/REMADV), 1C adapters. | `partner/routes.go:11-120` |
| 14 | `platformroutes` | `/v1/platform/*`, `/v1/user/device-token`, `/v1/auth/session` | Client policies, market packs, cells, media upload tickets, tenant registration. | `platformroutes/routes.go:25-56` |
| 15 | `platformadmin` | `/v1/platform-admin/*` | Tenant management, feature flag dual-control, audit trails, outbox dead-letter replay. | `platformadmin/handlers.go:176-200` |
| 16 | `featureflags` | `/v1/platform-admin/flags/*` | Dual-control flag evaluation, pending override approvals, audit logging. | `featureflags/handlers.go:165-181` |
| 17 | `mfa` | `/v1/platform-admin/mfa/*` | TOTP enrollment, confirmation, verification, and step-up enforcement. | `mfa/handlers.go:139-192` |
| 18 | `controltowerroutes` | `/v1/control-tower/*` | Scored exceptions, playbooks, execution runs, automated evaluation. | `controltowerroutes/routes.go:16-33` |
| 19 | `demandroutes` | `/v1/demand/*` | Demand signal ingest, adjustments, POS demand flywheel integration. | `demandroutes/routes.go:18-42` |
| 20 | `laborcapacityroutes`| `/v1/labor-capacity/*` | Driver scores, zone capacities, driver availability scheduling. | `laborcapacityroutes/routes.go:20-32` |
| 21 | `etaroutes` | `/v1/etas/*` | Realtime route ETAs, stop ETAs, recalculation triggers. | `etaroutes/routes.go:18-45` |
| 22 | `globalproductsroutes`| `/v1/global-products/*`, `/v1/admin/product-match-queue/*` | Master global catalog, product offers, match queue resolution. | `globalproductsroutes/routes.go:20-38` |
| 23 | `catalogroutes` | `/v1/catalog/*`, `/v1/products` | Public catalog browsing, category taxonomy, supplier product listings. | `catalogroutes/routes.go:20-39` |
| 24 | `pulseroutes` | `/v1/*/pulse` | Role-tailored live pulse feeds (retailer, supplier, warehouse, driver, payload, factory). | `pulseroutes/routes.go:17-53` |
| 25 | `taxroutes` | `/v1/admin/tax-regimes/*` | Tax regime versioning and rate definitions. | `taxroutes/routes.go:9-19` |
| 26 | `telemetryroutes` | `/v1/driver/location`, `/v1/driver/location/batch` | High-frequency driver GPS ingest, Redis caching, throttled outbox bus emit. | `telemetryroutes/routes.go:59-150` |
| 27 | `updateroutes` | `/v1/updates/ios/*`, `/v1/updates/desktop/*` | OTA updates (iOS manifest.plist, desktop updater.json). | `updateroutes/routes.go:23-26` |
| 28 | `storageroutes` | `/dossiers/*` | Compliance dossier creation and evidence attachment vault. | `storageroutes/routes.go:28-60` |
| 29 | `infraroutes` | `/healthz`, `/ready`, `/metrics`, `/v1/health` | Liveness/readiness probes, Prometheus metrics exporter. | `infraroutes/routes.go:38-46` |

---

## 4. Spanner Schema Architecture (`schema/spanner.ddl`)

The Cloud Spanner schema contains 3,648 lines of production DDL organized into cohesive domain modules:

1. **Tenancy & Core Identity**:
   - `Suppliers` (`spanner.ddl:1-24`), `SupplierOIDC` (`:25-34`), `SupplierProfiles` (`:35-50`), `SupplierStaff` (`:51-65`), `StaffInvites` (`:66-80`).
2. **Order Lifecycle & Manifests**:
   - `ParentOrders` (`:221-239`), `Orders` (`:240-310`), `OrderItems` (`:311-340`), `OrderDeliveryProofs` (`:341-360`), `OrderConditionReports` (`:361-380`), `SupplierTruckManifests` (`:798-840`), `ManifestOrders` (`:841-860`), `ManifestShipUnits` (`:861-880`).
3. **Warehouse & WMS Engine**:
   - `Warehouses` (`:420-455`), `WarehouseCoverageCells` (`:458-467`), `WarehouseCoverageCities` (`:468-477`), `WarehouseBins` (`:480-510`), `WarehouseLots` (`:511-540`), `WarehousePickWaves` (`:541-570`), `WarehousePickTasks` (`:571-600`), `WarehouseCycleCounts` (`:601-630`), `WarehouseTemperatureReadings` (`:631-650`).
4. **Retail OS & POS**:
   - `Retailers` (`:130-170`), `RetailerLocations` (`:2150-2180`), `StoreStock` (`:2200-2230`), `StoreStockMovements` (`:2231-2260`), `StoreStockReceiveSessions` (`:2261-2290`), `PosRegisters` (`:2350-2380`), `PosSessions` (`:2381-2410`), `PosSales` (`:2411-2440`), `PosHolds` (`:2441-2470`), `RetailerShifts` (`:2500-2530`), `RetailerSections` (`:2560-2580`), `RetailerAssistTickets` (`:2590-2620`).
5. **Finance, Invoices & Ledger**:
   - `PaymentConfigs` (`:651-678`), `PaymentSessions` (`:1200-1230`), `PaymentAttempts` (`:1231-1260`), `PaymentChargebacks` (`:1261-1290`), `PaymentLedgerEntries` (`:1320-1350`), `ArInvoices` (`:1400-1430`), `CashReconciliations` (`:1500-1530`), `CreditNotes` (`:1550-1580`), `PayoutBatches` (`:1650-1680`), `FxRates` (`:1700-1730`).
6. **Partner & B2B Integration**:
   - `PartnerApiKeys` (`:2800-2830`), `PartnerOAuthClients` (`:2831-2850`), `PartnerWebhooks` (`:2860-2890`), `PartnerEdiDocuments` (`:2920-2950`), `PartnerEdiProfiles` (`:2951-2970`), `PartnerAs2Configs` (`:3000-3030`), `PartnerSftpConfigs` (`:3040-3070`).
7. **Transactional Outbox Engine**:
   - `OutboxEvents` (`:679-697`), `OutboxDeadLetters` (`:698-709`), with `Idx_OutboxEvents_Unpublished` (`:696`).

---

## 5. Full 6 Role-Row Client Parity

Every role row possesses a genuine, fully implemented, compiled, and tested application across Web/Desktop, Android, and iOS surfaces:

### 5.1 Supplier Role
- **Web & Desktop (`apps/supplier-portal`)**: Next.js 15 App Router + Tauri 2. 82 portal pages covering Control Tower, Catalog, S&OP Planning, Fleet, Manifests, CRM, and Payouts. Full `@pegasusx/api-client` integration with WebSocket reconnect session reconciliation (`lib/use-supplier-ws-refresh.ts`).
- **Android (`apps/supplier-app-android`)**: Kotlin 2.x, Jetpack Compose, Retrofit (`SupplierApi.kt`, 711 lines), Hilt, Coroutines/Flow, OkHttp WebSocket with exponential backoff. 61 screen composables.
- **iOS (`apps/supplier-app-ios`)**: Swift 6, SwiftUI, Swift Concurrency, `SupplierRealtimeClient.swift` (`URLSessionWebSocketTask`). 68 view components.

### 5.2 Retailer Role
- **Desktop (`apps/retailer-app-desktop`)**: Next.js 15 + Tauri 2. 31 desktop routes covering Store Stock, Local SKUs, POS, Dock, Credit, and HQ. Offline checkout buffer (`pending-checkout.ts`) and SQLite caching (`@pegasusx/desktop-cache`).
- **Android (`apps/retailer-app-android`)**: Jetpack Compose, Room SQLite (`AppDatabase.kt`), WorkManager offline sync, Retrofit (`PegasusApi.kt` calling `/v1/retailer/ai/predictions`). 40+ screens.
- **iOS (`apps/retailer-app-ios`)**: SwiftUI, SwiftData (`PendingPosStore.swift`, `PendingOrderReplayer.swift`), URLSession. 49 screens.

### 5.3 Driver Role
- **Android (`apps/driver-app-android`)**: Jetpack Compose, Room SQLite (v6 `PegasusDriverDatabase.kt`), Foreground Telemetry Service (`TelemetrySocket.kt` on `/v1/ws?sv=2`), dead-reckoning GPS interpolation (`LocationInterpolator.kt`), spoken audio cues (`NavigationCueAnnouncer.kt`). 63 screen files.
- **iOS (`apps/driver-app-ios`)**: SwiftUI, SwiftData (`OfflineDeliveryStore.swift`), CoreLocation background streaming (`TelemetryServiceLive.swift`), AVFoundation voice announcer (`NavigationVoiceAnnouncer.swift`). 74 view files.

### 5.4 Warehouse Role
- **Web & Desktop (`apps/warehouse-portal`)**: Next.js 15 + Tauri 2. 46 routes covering WMS Bins/Lots, Pick Waves, Cycle Counts, Cold Chain, Tomorrow Board, and Dispatch.
- **Android (`apps/warehouse-app-android`)**: Jetpack Compose, Retrofit (`WarehouseApi.kt`), `WarehouseOfflineQueue.kt`, Zebra DataWedge barcode scanning. 44 screens.
- **iOS (`apps/warehouse-app-ios`)**: SwiftUI, Observation, `WarehouseOperationsService.swift`, `WarehouseRealtimeHub.swift`. 84 views.

### 5.5 Factory Role
- **Web & Desktop (`apps/factory-portal`)**: Next.js 15 + Tauri 2. 21 routes covering Loading Bay, Supply Requests, Transfers, and Payload Overrides. Connects to live Spanner `FactoryTruckManifests`.
- **Android (`apps/factory-app-android`)**: Jetpack Compose, Retrofit (`FactoryApi.kt`), `FactoryOfflineQueue.kt`. 62 Kotlin files.
- **iOS (`apps/factory-app-ios`)**: SwiftUI, Observation, `FactoryRealtimeClient.swift`. 70 Swift files.

### 5.6 Payload Role
- **Terminal (`apps/payload-terminal`)**: React Native + Expo (SDK 55), NativeWind, Expo Camera barcode scanning, `POST /v1/payloader/manifests/seal-all` integration.
- **Android (`apps/payload-app-android`)**: Jetpack Compose, Room SQLite (`PayloadDatabase.kt`), OkHttp WebSocket forwarding `PAYLOAD_SYNC`. 50 Kotlin files.
- **iOS (`apps/payload-app-ios`)**: SwiftUI, Observation, `OfflineQueue.swift`, Keychain. 43 Swift files.

### 5.7 Platform Admin Role
- **Web Console (`apps/admin-portal`)**: Next.js 15. 9 governance panels: Tenants, Feature Flags (dual-control), Outbox Dead-Letters & Replay, Billing, Audit Logs, Partner Keys, Product Match Queue.

---

## 6. Shared Packages Architecture (`pegasusX/packages/`)

```
pegasusX/packages/
├── types/                     # 6,682 lines DTOs, interfaces, RFC 7807 ProblemDetails
├── api-client/                # 3,669 lines unified HTTP client, backoff jitter, idempotency
├── ws-refresh-contract/       # Canonical WebSocket refresh rules & dashboardDirtySlice()
├── desktop-bridge/            # Tauri IPC bridge (printing, deep-links, updater)
├── desktop-cache/             # SQLite local persistence & offline mutation queues
├── ui-kit/                    # Design tokens & portal primitives (StatusStack, KpiStat)
├── mobile-android-kit/        # Shared Android network backoff & prefs offline store
├── mobile-ios-kit/            # Shared iOS network backoff & queued mutation models
├── mobile-android-design/     # Monochrome design tokens & Jetpack Compose components
├── mobile-ios-design/         # Monochrome design tokens & SwiftUI components
├── pulse-ui/                  # Realtime pulse feed UI primitives
├── explain-ui/                # Machine learning explainability visualizations
└── validation/                # Runtime schema validation helpers
```

---

## 7. Quality Gates & Verified Test Results

All client workspaces and backend services are verified passing:

1. **Backend Go Core (`apps/backend-go`)**: 80+ packages pass unit and integration tests; `cmd/ssmr-smokecheck` executes 80+ multi-role end-to-end checks cleanly.
2. **Client Workspace Unit Suites (Vitest)**:
   - `@pegasusx/desktop-bridge`: 3 test files, 17 tests passed.
   - `@pegasusx/desktop-cache`: 1 test file, 7 tests passed.
   - `@pegasusx/supplier-portal`: 17 test files, 56 tests passed.
   - `@pegasusx/retailer-app-desktop`: 17 test files, 93 tests passed.
   - `@pegasusx/warehouse-portal`: 10 test files, 21 tests passed.
   - `@pegasusx/factory-portal`: 9 test files, 21 tests passed.
   - `payload-terminal`: 3 test files, 19 tests passed.
   - `@pegasusx/admin-portal`: 1 test file, 4 tests passed.
3. **Android Test Suites**: 39 Kotlin test classes covering state machines, contract decoders, offline queues, and view models.
4. **iOS Test Suites**: 28 Swift test classes covering XCTest/Swift-Testing assertions.

---

## 8. Honest Product Boundaries (HTTP 410 Gated Flows)

To uphold the absolute honesty standard, intentionally restricted or gated boundaries return RFC 7807 HTTP 410:

- `GET /v1/supplier/inventory/audit`: Returns HTTP 410 `audit_unwired`.
- `POST /v1/delivery/negotiate`: Returns HTTP 410 `feature_disabled` unless `QUANTITY_NEGOTIATION_ENABLED=true`.
- `POST /v1/retailer/card*`: Returns HTTP 410 `saved_cards_not_product`.
- `GET /v1/payloader/capacity`: Returns HTTP 410 `capacity_unwired` (VU computed directly from manifest items).
- `POST /v1/webhooks/payme` & `click`: Commented out in `webhookroutes` (launch payment path scoped to Cash + GlobalPay + MySoliq).

---

## 9. Global Scale & Multi-Market Roadmap

The global-scale roadmap decomposes into structured phases:

- **GS-A**: Auth session & MarketPack catalog resolution (shipped).
- **GS-T**: Self-serve multi-supplier tenant registration (shipped).
- **GS-M**: MarketPack reading for checkout, fiscal, and proximity (shipped).
- **GS-C**: Regional cell scaffolding (`cell-uz`, `cell-eu`, `cell-us`) (scaffolded).
- **GS-I**: Per-tenant enterprise OIDC integration (shipped).
- **GS-L**: Local-first warehouse matching & ServicePins (L0–L4 shipped).
- **GS-K**: Pack-owned PSP catalog & honest executors (K1–K3 shipped).
- **GS-U**: Full client visualization program (U0–U9 shipped).
