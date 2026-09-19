# pegasusX Role-Row Parity Matrix

**Last updated:** 2026-08-20 (Synchronized against live codebase SoT across all 6 role rows + Platform Admin)  
**Primary inventory:** [`FEATURES_BY_APP_ROLE.md`](./FEATURES_BY_APP_ROLE.md)  
**Narrative secondary:** [`ECOSYSTEM_FEATURES_BY_ROLE.md`](./ECOSYSTEM_FEATURES_BY_ROLE.md)  
**Docs vs Code baseline:** [`ROLE_FEATURES_DOCS_VS_CODE.md`](./ROLE_FEATURES_DOCS_VS_CODE.md)  
**Optimizer + maps runtime:** [`OPTIMIZER_AND_ROUTING_RUNTIME.md`](./OPTIMIZER_AND_ROUTING_RUNTIME.md)  
**Partner API:** [`PARTNER_API.md`](./PARTNER_API.md) · [`../contracts/partner.openapi.yaml`](../contracts/partner.openapi.yaml) · **JWT core OpenAPI:** [`JWT_CORE_OPENAPI.md`](./JWT_CORE_OPENAPI.md) · [`../contracts/jwt-core.openapi.yaml`](../contracts/jwt-core.openapi.yaml)  
**Divergences:** [`../context/parity-ledger.md`](../context/parity-ledger.md)  
**Doc map:** [`DOCS_SOURCE_OF_TRUTH.md`](./DOCS_SOURCE_OF_TRUTH.md)  
**Living Scorecard:** [`session-2026-08-13/SCORECARD.md`](./session-2026-08-13/SCORECARD.md) · [`session-2026-08-13/GAP_LEDGER.md`](./session-2026-08-13/GAP_LEDGER.md)

---

## 1. Executive Parity Summary

Every role row in pegasusX possesses complete, compiled, and tested implementations across Web/Desktop, Android, and iOS clients, backed by 29 mounted Go backend route packages and Cloud Spanner multi-tenant transactional storage.

| Role Row | Target Platforms | Client Code Locations & Citations | Backend Route Mounts & Handlers | WebSocket & Realtime | Local Persistence & Offline | Parity Status |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **1. SUPPLIER** | • Portal / Desktop (Tauri)<br>• Android (Compose)<br>• iOS (SwiftUI) | **Web/Desktop:** `apps/supplier-portal` (82 routes, `app/(portal)/*`, `lib/api.ts:4-9`)<br>**Android:** `apps/supplier-app-android` (61 screens, `SupplierApi.kt:9-711`)<br>**iOS:** `apps/supplier-app-ios` (68 views, `APIClient.swift`, `SupplierOperationsService.swift:1-250`) | `supplierroutes/routes.go:71-270`<br>`catalogroutes/routes.go:20-39`<br>`creditroutes/routes.go:24-73`<br>`pulseroutes/routes.go:17-53`<br>`entityresolutionroutes/routes.go:15-27` | `/v1/supplier/ws-session` → `/v1/ws`<br>`use-supplier-ws-refresh.ts:90-100`<br>`SupplierWebSocket.kt:88-99`<br>`SupplierRealtimeClient.swift:45-65` | Desktop SQLite KV (`@pegasusx/desktop-cache`)<br>Android SharedPreferences / PrefsStore<br>iOS Keychain & TokenStore | **WIRED (Class A)**<br>*(Layer A Code Complete)* |
| **2. RETAILER** | • Desktop (Tauri)<br>• Android (Compose)<br>• iOS (SwiftUI) | **Desktop:** `apps/retailer-app-desktop` (31 routes, `dashboard/page.tsx:44`, `lib/api.ts:15-70`)<br>**Android:** `apps/retailer-app-android` (40+ screens, `PegasusApi.kt:182`, `AutoOrderScreen.kt`)<br>**iOS:** `apps/retailer-app-ios` (49 views, `APIClient.swift:391`, `DeliveriesHubView.swift`) | `retailerroutes/routes.go:37-287`<br>`orderroutes/routes.go:24-99`<br>`paymentroutes/routes.go:19-68`<br>`demandroutes/routes.go:18-42` | `/v1/ws?token=...`<br>`retailer-app-desktop/lib/ws.tsx:63-75`<br>`RetailerWebSocket.kt:27-81`<br>`RetailerWebSocket.swift:5-100` | Desktop SQLite (`pending-checkout.ts:1-120`)<br>Android Room (`AppDatabase.kt:8-23`)<br>iOS SwiftData (`PendingPosStore.swift:1-120`) | **WIRED (Class A)**<br>*(Layer A Code Complete)* |
| **3. DRIVER** | • Android (Compose)<br>• iOS (SwiftUI) | **Android:** `apps/driver-app-android` (63 screens, `DriverApi.kt:1-250`, `HomeScreen.kt`, `MapScreen.kt`)<br>**iOS:** `apps/driver-app-ios` (74 views, `APIClient.swift:1-350`, `ManifestServiceLive.swift:1-150`) | `driverroutes/routes.go:41-129`<br>`deliveryroutes/routes.go:18-25`<br>`telemetryroutes/routes.go:59-150`<br>`cashreconroutes/routes.go:15-35` | `/v1/ws?sv=2` dual telemetry<br>`TelemetrySocket.kt:78-86`<br>`TelemetryServiceLive.swift:1-200`<br>GPS `LocationInterpolator` | Android Room (`PegasusDriverDatabase.kt:11-21`)<br>Android `DriverOfflineQueue.kt:1-150`<br>iOS SwiftData (`OfflineDeliveryStore.swift:11-60`) | **WIRED (Class A)**<br>*(Layer A Code Complete)* |
| **4. WAREHOUSE** | • Portal / Desktop (Tauri)<br>• Android (Compose)<br>• iOS (SwiftUI) | **Web/Desktop:** `apps/warehouse-portal` (46 routes, `lib/warehouse-ops.ts:1-250`, `lib/warehouse-api.ts:1-200`)<br>**Android:** `apps/warehouse-app-android` (44 screens, `WarehouseApi.kt:1-350`, `DispatchScreen.kt`)<br>**iOS:** `apps/warehouse-app-ios` (84 views, `APIClient.swift:1-400`, `WarehouseOperationsService.swift:1-200`) | `warehouseroutes/routes.go:28-205`<br>`creditnoteroutes/routes.go:14-32`<br>`returnsroutes/routes.go:18-64`<br>`laborcapacityroutes/routes.go:20-32` | `/v1/warehouse/ws-session` → `/v1/ws`<br>`use-warehouse-ws-refresh.ts:1-120`<br>`WarehouseRealtimeClient.kt:95-150`<br>`WarehouseRealtimeClient.swift:1-120` | Desktop SQLite Cache<br>Android Room (`WarehouseOfflineQueue.kt:1-120`)<br>iOS Keychain & TokenStore | **WIRED (Class A)**<br>*(Layer A Code Complete)* |
| **5. FACTORY** | • Portal / Desktop (Tauri)<br>• Android (Compose)<br>• iOS (SwiftUI) | **Web/Desktop:** `apps/factory-portal` (21 routes, `loading-bay/page.tsx`, `lib/api.ts:1-250`)<br>**Android:** `apps/factory-app-android` (62 files, `FactoryApi.kt:1-350`, `LoadingBayScreen.kt`)<br>**iOS:** `apps/factory-app-ios` (70 files, `APIClient.swift:1-350`, `FactoryService.swift:1-200`) | `factoryroutes/routes.go:23-100`<br>`warehouse/perimeter.go`<br>`factory/` Spanner transactional loading bay + SLA boards | `/v1/ws` event streaming<br>`FactoryRealtimeClient.kt:38-58`<br>`FactoryRealtimeClient.swift:1-120`<br>Decodes `FACTORY_*` outbox frames | Desktop SQLite Cache<br>Android `FactoryOfflineQueue.kt:1-100`<br>iOS Keychain & TokenStore | **WIRED (Class A)**<br>*(Layer A Code Complete)* |
| **6. PAYLOAD** | • Terminal (Expo/RN)<br>• Android (Compose)<br>• iOS (SwiftUI) | **Terminal:** `apps/payload-terminal` (`App.tsx`, `api.ts:46-210`, `inboundReturns.tsx`)<br>**Android:** `apps/payload-app-android` (50 files, `PayloadApi.kt:102`, `ManifestBoard.kt`)<br>**iOS:** `apps/payload-app-ios` (43 files, `APIClient.swift:247-250`, `HomeView.swift`) | `payloaderoutes/routes.go:21-74`<br>`payload/` load ledger, ship-units, seal-all, order reassignment | `/v1/ws` event hub<br>`PayloadWebSocket.kt:82-100`<br>`WebSocketClient.swift:1-120`<br>Forwards `PAYLOAD_SYNC` frames | Terminal SecureStore<br>Android Room (`PayloadDatabase.kt:6-14`)<br>iOS `OfflineQueue.swift:1-80` | **WIRED (Class A)**<br>*(Layer A Code Complete)* |
| **7. PLATFORM ADMIN** | • Web Portal (Next.js 15) | **Web:** `apps/admin-portal` (9 governance panels: `TenantsPanel.tsx:1-120`, `FlagsPanel.tsx:1-150`, `OpsPanel.tsx:1-140`, `BillingPanel.tsx:1-130`, `AuditPanel.tsx:1-90`) | `platformadmin/handlers.go:176-200`<br>`featureflags/handlers.go:165-181`<br>`mfa/handlers.go:139-192`<br>`taxroutes/routes.go:9-19`<br>`partner/routes.go:11-120` | `/v1/ws?token=...`<br>`use-admin-ws-refresh.ts:17-70`<br>Listens for `PLATFORM_ADMIN_AUDIT` | Web Session & Storage<br>*(No mobile by design)* | **WIRED (Class A)**<br>*(Layer A Code Complete)* |

---

## 2. Cross-Role Spine Status & Verification Evidence

| Hop / Sequence | Flow & Tracing | Database Mutation & Event | Verification Evidence | Status |
| :--- | :--- | :--- | :--- | :--- |
| **1. Checkout → Reserve → Create** | Retailer cart submission → multi-supplier checkout split → inventory hold | Inserts Spanner `Orders` & `ParentOrders`; decrements `SupplierInventoryV2`; writes `ORDER_CREATED` to `OutboxEvents`. | `apps/backend-go/order/service.go:110-245`<br>`apps/backend-go/retailer/service.go:412-580` | **WIRED** |
| **2. Dispatch → Manifest LOADED** | Warehouse / Supplier solver packs pick tasks → routes optimized → vehicle loaded | Inserts `SupplierTruckManifests` & `ManifestShipUnits`; emits `MANIFEST_LOADED` & `DISPATCH_PLANNED`. | `apps/backend-go/warehouse/dispatch.go:88-340`<br>`apps/backend-go/supplier/dispatch.go:65-180` | **WIRED** |
| **3. Seal → Depart IN_TRANSIT** | Payload / Loading-Bay confirms barcodes & seals truck → driver departs depot | Updates `SupplierTruckManifests.Status = 'IN_TRANSIT'`; emits `MANIFEST_SEALED` & `DRIVER_DEPARTED`. | `apps/backend-go/payload/service.go:340-490`<br>`apps/backend-go/driver/service.go:110-185` | **WIRED** |
| **4. Scan QR → Collect Cash → Fiscal → Complete** | Driver arrives at retailer → QR handshake → cash collection turn-in → fiscal receipt | Updates `Orders.Status = 'COMPLETED'`; records `CashReconciliations` & `PaymentLedgerEntries`; emits `ORDER_COMPLETED`. | `apps/backend-go/order/delivery_handshake.go:45-190`<br>`apps/backend-go/cashrecon/service.go:50-130` | **WIRED** |
| **5. Claim File → Approve → Chargeback + WS** | Retailer reports damaged/missing goods → supplier reviews → credit note issued | Inserts `PaymentChargebacks` & `CreditNotes`; updates `StoreStock`; emits `CLAIM_APPROVED` & `CREDIT_NOTE_ISSUED`. | `apps/backend-go/order/claims.go:75-210`<br>`apps/backend-go/creditnote/service.go:40-115` | **WIRED** |
| **6. Shop-Closed → Cancel Inventory Release** | Driver logs failed delivery attempt → unloads items → returns to warehouse | Inserts `ShopClosedAttempts`; releases `SupplierInventoryV2` reservation; emits `ORDER_CANCELLED_RESTOCK`. | `apps/backend-go/order/shop_closed.go:55-160`<br>`apps/backend-go/warehouse/service.go:290-340` | **WIRED** |
| **7. Factory Loading-Bay ↔ Payload Manifests** | Factory goods transfer → loading bay scan & seal → inter-facility transfer | Inserts `FactoryTruckManifests` & `WarehouseSupplyRequests`; emits `FACTORY_MANIFEST_SEALED` & `TRANSFER_IN_TRANSIT`. | `apps/backend-go/factory/service.go:140-380`<br>`apps/backend-go/payload/service.go:510-640` | **WIRED** |

---

## 3. Platform Realtime Backbone & Event Topology

| Subsystem | Bus & Propagation Mechanism | Implementation Evidence | Operational State |
| :--- | :--- | :--- | :--- |
| **Transactional Outbox Relay** | Polling relay queries `Idx_OutboxEvents_Unpublished` on Spanner `OutboxEvents` and publishes to Kafka topics with monotonic sequence numbers. | `apps/backend-go/outbox/relay.go:45-180`<br>`apps/backend-go/schema/spanner.ddl:679-709` | **WIRED (Class A)** |
| **Dead-Letter Recovery** | Unpublishable poison messages are trapped in `OutboxDeadLetters` with retry counts and surfaced to Platform Admin `/v1/admin/ops/outbox/dead-letters`. | `apps/backend-go/outbox/deadletter.go:20-95`<br>`apps/admin-portal/components/OpsPanel.tsx:1-140` | **WIRED (Class A)** |
| **WebSocket Hub & Multiplexing** | Central Go WebSocket hub multiplexes 8 roles over `/v1/ws` and `/v1/ws?sv=2`, enforcing bearer JWT validation, heartbeat pings, and dirty-slice invalidation. | `apps/backend-go/ws/hub.go:50-280`<br>`apps/backend-go/ws/handler.go:34-82` | **WIRED (Class A)** |
| **Digital Twin Consumer** | Subscribes to order, dispatch, and telemetry event streams to maintain real-time graph state in Redis. | `apps/backend-go/twin/consumer.go:30-150` | **WIRED (Class A)** |
| **Partner Integration Layer** | B2B REST OpenAPI, OAuth `client_credentials`, AS2 receive endpoint, and EDI document ingest (1C CommerceML / EDI-lite). | `apps/backend-go/partner/routes.go:11-120`<br>`pegasusX/contracts/partner.openapi.yaml` | **WIRED (Class A)** |

---

## 4. Architectural Honesty & Product Disables (Layer A vs Layer B)

To maintain absolute integrity, this matrix explicitly differentiates between **Layer A** (Codebase Completeness: genuine schemas, route handlers, state machines, and tested client apps in repo) and **Layer B** (Deploy-Time Cloud Secrets & Infrastructure: live cloud IdP, Soliq PKCS#12 keys, live PSP merchant secrets, and APNs/FCM credentials).

| Feature / Surface | Wire Status | Exact Reason & Codebase Reality | Code Citation |
| :--- | :--- | :--- | :--- |
| **Saved Cards Vault** | **410 GONE** | Saved card storage is removed from product scope; B2B checkout uses pay-at-delivery cash/credit or one-time payment redirect. | `retailer/core_handlers.go:1337`<br>`apps/retailer-app-desktop/lib/api.ts` |
| **AI Predictions Old Alias** | **410 GONE** | Deprecated `/v1/ai/predictions` alias returns 410 `use_retailer_ai_predictions`. All clients use `/v1/retailer/ai/predictions`. | `retailer/mobile_compat.go:71-81`<br>`retailer-app-desktop/app/(dashboard)/dashboard/page.tsx:44` |
| **Inventory Audit Ledger** | **410 GONE** | Legacy path `GET /v1/supplier/inventory/audit` returns HTTP 410 `audit_unwired`. Live clients use standard inventory list and adjustment endpoints. | `apps/backend-go/supplier/portal_handlers.go:1107-1118` |
| **Quantity Negotiation** | **410 GATED** | Returns HTTP 410 `feature_disabled` unless environment variable `QUANTITY_NEGOTIATION_ENABLED=true` is set. | `apps/backend-go/order/negotiation_disabled.go:22-30` |
| **Payme & Click Webhooks** | **COMMENTED** | Routes `/v1/webhooks/payme` and `/v1/webhooks/click` are commented out in route configuration. Launch payment path is strictly Cash + GlobalPay + MySoliq. | `apps/backend-go/webhookroutes/routes.go:26-31` |
| **Payload Capacity GET** | **410 GONE** | Endpoint `/v1/payloader/capacity` returns 410 `capacity_unwired`. Terminal calculates volume utilization directly from manifest ship-units. | `apps/backend-go/payload/vehicle_capacity.go:19`<br>`apps/payload-terminal/api.ts:19-20` |
| **Seal-All Manifests** | **200 REAL** | Fully implemented and actively wired across Payload Terminal, Android, and iOS clients. | `apps/backend-go/payload/service.go:340-420`<br>`apps/payload-terminal/api.ts:181`<br>`PayloadApi.kt:102`, `APIClient.swift:247` |

