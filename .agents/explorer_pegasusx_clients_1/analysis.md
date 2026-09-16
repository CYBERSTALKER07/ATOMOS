# PegasusX Client Ecosystems & Contract Layers: Deep Architectural Audit

**Investigator:** `explorer_pegasusx_clients`  
**Date:** 2026-09-14  
**Working Directory:** `/Users/shakhzod/Desktop/V.O.I.D/.agents/explorer_pegasusx_clients_1`  
**Target Repository:** `pegasusX/` (Global Enterprise Multi-Tenant Architecture)  
**Verification Tools Used:** `view_file`, `grep_search`, `find_by_name`, `list_dir`, `run_command` (CI verification scripts)

---

## Executive Summary

A comprehensive architectural investigation was conducted across the entire client ecosystem and shared contract layer of `pegasusX/`. The investigation evaluated:
1. **Web Portals & Desktop Apps:** 5 Next.js 15 / React 19 / Tauri v2 applications (`apps/supplier-portal`, `apps/warehouse-portal`, `apps/factory-portal`, `apps/retailer-app-desktop`, `apps/admin-portal`) plus the React Native Expo 55 `apps/payload-terminal`.
2. **Native Mobile Applications:** 6 Kotlin Android applications (`apps/*-android`) and 6 SwiftUI iOS applications (`apps/*-ios`) covering all six business roles (Supplier, Retailer, Driver, Warehouse, Factory, Payload).
3. **Shared Contracts & Type Architecture:** The monorepo packages (`packages/types`, `packages/api-core`, `packages/api-client`), Go AST schema generator (`cmd/gen-contracts`), JSON-Schema contracts (`contracts/events.schema.json`), OpenAPI definitions (`contracts/partner.openapi.yaml`, `contracts/jwt-core.openapi.yaml`), and automated Quicktype code generators.
4. **Realtime, Geospatial & Offline Systems:** Multi-hub WebSocket architecture (`/v1/ws`, `/v1/ws?sv=2`), Server-Sent Events (`/v1/supplier/events`), MapLibre GL + Carto dynamic pack camera bindings (`mapInitialViewState(pack)`), Uber H3 cell polygon rendering, and layered offline mutation queues (Room, SwiftData, SQLite).
5. **Role-Row Parity & Verification Gates:** Full verification via `bash scripts/parity/role_row_contract_check.sh` and `bash scripts/parity/role_row_contract_check_full.sh`, confirming zero unmounted client routes and rigorous Layer A vs Layer B honesty boundaries.

---

## 1. Web Portals & Frontend Architecture

### 1.1 Application Inventory & Framework Specifications

| Application | Path | Frameworks & Core Libraries | Target Runtime | UI / Component Stack |
| :--- | :--- | :--- | :--- | :--- |
| **Supplier Portal** | `apps/supplier-portal` | Next.js `^15.0.0`, React `19.0.0`, Tailwind CSS `^4.0.0`, TypeScript `^5.4.0` | Dual: Web Browser + Desktop (Tauri v2 `@tauri-apps/api: ^2.11.0`) | HeroUI (`@heroui/react: ^3.0.2`), Framer Motion (`^12.42.2`), Lucide React (`^0.511.0`), `@pegasusx/ui-kit` |
| **Warehouse Portal** | `apps/warehouse-portal` | Next.js `^15.0.0` (`--turbopack`), React `19.0.0`, Tailwind CSS `^4.0.0`, TypeScript `^5.4.0` | Dual: Web Browser + Desktop (Tauri v2 `@tauri-apps/api: ^2.11.0`) | HeroUI (`^3.1.0`), Framer Motion (`^12.40.0`), Lucide React (`^1.7.0`), `@pegasusx/ui-kit` |
| **Factory Portal** | `apps/factory-portal` | Next.js `^15.0.0` (`--turbopack`), React `19.0.0`, Tailwind CSS `^4.0.0`, TypeScript `^5.0.0` | Dual: Web Browser + Desktop (Tauri v2 `@tauri-apps/api: ^2.11.0`) | HeroUI (`^3.1.0`), Framer Motion (`^12.38.0`), Lucide React (`^1.7.0`), `@pegasusx/ui-kit` |
| **Retailer Desktop** | `apps/retailer-app-desktop` | Next.js `15.5.12` (`--turbopack`), React `19.1.0`, Tailwind CSS `^4.0.0`, TypeScript `^5.0.0` | Dual: Web Browser + Desktop (Tauri v2 `@tauri-apps/api: ^2.10.1`) | HeroUI (`^3.1.0`), Framer Motion (`^12.40.0`), `qrcode.react`, `@pegasusx/ui-kit` |
| **Admin Portal** | `apps/admin-portal` | Next.js `^15.0.0`, React `19.0.0`, Tailwind CSS `^4.0.0`, TypeScript `^5.4.0` | Web Browser (Console Governance) | Pure Tailwind CSS v4, `@pegasusx/ui-kit`, `@pegasusx/types` |
| **Payload Terminal** | `apps/payload-terminal` | Expo `~55.0.4`, React Native `0.83.2`, React `19.2.0`, NativeWind `^4.2.2`, Tailwind `^3.4.19` | Universal: Web, iOS, Android, Rugged Handhelds | React Native Reanimated `4.2.1`, Expo Camera `~17.0.10`, Expo Haptics `~55.0.8` |

### 1.2 Desktop Packaging (Tauri v2 Architecture)
Four of the frontend applications (`supplier-portal`, `warehouse-portal`, `factory-portal`, `retailer-app-desktop`) are engineered as hybrid web/desktop apps:
- **Tauri v2 Core:** Built with `@tauri-apps/api: ^2.11.0` and `@tauri-apps/cli: ^2.11.2`.
- **Packaging Scripts:** Configured in `package.json` for Universal macOS (`--target universal-apple-darwin`), Windows x64 (`--target x86_64-pc-windows-msvc`), and Store editions (Mac App Store, Microsoft Store).
- **Plugins:**
  - `@tauri-apps/plugin-sql: ^2.2.0`: Local embedded SQLite database for offline persistence.
  - `@tauri-apps/plugin-deep-link: ^2.4.9`: Handles OS URI schemes (`pegasusx://...`).
  - `@tauri-apps/plugin-updater: ^2.9.0`: Native GCS/CDN signed auto-update mechanism.
  - `@tauri-apps/plugin-fs`, `@tauri-apps/plugin-dialog`, `@tauri-apps/plugin-process`.
- **Shared Desktop Libraries:**
  - `@pegasusx/desktop-bridge`: Abstractions for receipt printing, CSV/Excel file export, deep linking, and runtime detection.
  - `@pegasusx/desktop-cache`: High-level offline persistence wrappers for pending commands, offline checkout, POS sales, and key-value storage.

### 1.3 Control Tower Architecture
Control Tower is not an isolated portal, but an integrated operations hub mounted contextually across portals:
- **Supplier Portal:** `apps/supplier-portal/app/(portal)/control-tower/` backed by `use-control-tower-telemetry.ts` and `__tests__/control-tower-geospatial.test.tsx`.
- **Warehouse Portal:** `apps/warehouse-portal/app/control-tower/` with `use-control-tower-telemetry.ts`.
- **Retailer Desktop:** `apps/retailer-app-desktop/app/(dashboard)/control-tower/`.
- **Shared Components:** `packages/ui-kit/src/control-tower/GlassmorphismPanel.tsx` and `packages/ui-maps/src/HexagonalControlTowerMap.tsx`.
- **Real-Time Telemetry:** Hook `useControlTowerTelemetry` opens a live WebSocket or polling feed, updating driver GPS coordinates, active transit routes, and SLA breach markers with zero full-page re-renders.

### 1.4 State Management & Data Fetching Patterns
- **React Context:** Used for localized session and feature state:
  - `apps/retailer-app-desktop/lib/cart.tsx`: Multi-supplier cart splitting and checkout state.
  - `apps/retailer-app-desktop/lib/ws.tsx`: Global WebSocket lifecycle and typed message subscription dispatcher.
  - `apps/retailer-app-desktop/lib/notifications.tsx`: Real-time notification toaster and badge counts.
  - `apps/admin-portal/lib/session.tsx`: Auth session and tenant token storage.
- **Adaptive Polling (`@pegasusx/api-react/usePolling`):**
  - Stale-while-revalidate data loading.
  - Automatic pause when browser tab is backgrounded (`document.visibilityState !== "visible"`), with configurable `hiddenIntervalMs` (e.g. 60s for fleet tracking).
  - Listens for server-driven backpressure events (`pegasus:backpressure`).
  - In-flight request cancellation via `AbortController`.
- **Desktop SQLite Queues (`@pegasusx/desktop-cache`):**
  - Table `pending_commands` tracks offline mutations (`command_id`, `command_type`, `entity_id`, `known_version`, `payload_json`, `retry_count`, `status`).

---

## 2. Native Mobile Applications

### 2.1 Kotlin Android Applications (`apps/*-android`)

All six Android applications share a common enterprise engineering foundation:
- **Language & Toolchain:** Kotlin 2.x, Android Gradle Plugin 8.x, Compile SDK 35, Target SDK 35, Min SDK 26, Java 17.
- **Architecture:** Jetpack Compose + Compose BOM (`2024.12.01`), AndroidX Navigation Compose (`2.8.5`), AndroidX Lifecycle ViewModel Compose (`2.8.7`).
- **Dependency Injection:** Google Dagger Hilt (`2.59.2`) with KSP (`com.google.devtools.ksp`).
- **Networking & Serialization:** Square Retrofit (`2.11.0`), OkHttp (`4.12.0`), `kotlinx.serialization.json: 1.7.3`, JakeWharton Retrofit Kotlinx Serialization Converter (`1.0.0`).
- **Distribution Flavors:**
  - `enterprise`: Includes CDN OTA background update check (`buildConfigField("boolean", "ENABLE_CDN_OTA", "true")`).
  - `store`: Google Play Store compliant, disables APK sideloading updates (`ENABLE_CDN_OTA=false`), sets official store listing URLs.
- **Shared Monorepo Modules:**
  - `project(":mobile-design")` (`packages/mobile-android-design`): Shared UI theme (`PegasusMonochromeTheme`), design tokens, MarketPack session store, and Cell API router.
  - `project(":mobile-kit")` (`packages/mobile-android-kit`): Shared offline queue contracts (`OfflineEndpointCatalog`, `OfflineHttpSemantics`, `PrefsOfflineQueueStore`, `QueuedMutation`).
  - Shared i18n string catalogs linked via Gradle sourceSets: `res.srcDir(rootProject.file("../../packages/i18n/generated/android"))`.

#### Android App Role Breakdown:

1. **`driver-app-android` (`com.pegasusx.driver`):**
   - 63 screens/composables.
   - Dual telemetry: `TelemetrySocket.kt` connects to `/v1/ws?sv=2` with periodic ping frames and exponential reconnect backoff.
   - Offline mutation engine: `DriverOfflineQueue.kt` backed by Room database (`PegasusDriverDatabase.kt`, `PendingMutationDao`).
   - Background sync: AndroidX WorkManager (`OfflineSyncScheduler.kt`, `OfflineSyncWorker.kt`) triggered automatically upon network reconnect.
   - Hardware: CameraX (`1.4.1`) + Google ML Kit Barcode Scanning (`17.3.0`) for QR doorstep validation; FusedLocationProvider (`21.3.0`) for GPS tracking.
2. **`retailer-app-android` (`com.pegasusx.retailer`):**
   - 40+ composables.
   - Room local persistence (`AppDatabase.kt`) for catalog caching and offline cart.
   - Geospatial: `HexagonalControlTowerMap.kt` renders Uber H3 density hexagons (`com.uber.h3core:H3Core`) on top of Google Maps Compose.
   - Pack Camera: Dynamically centers camera using `com.pegasus.design.sessionMapCenter()` (never hardcoded to San Francisco or Tashkent).
3. **`supplier-app-android` (`com.pegasusx.supplier`):**
   - 61 Compose screens, `SupplierApi.kt` (711 lines Retrofit interface).
   - Realtime: `SupplierWebSocket.kt` maintains persistent connection to `/v1/ws`.
   - Operations: Catalog CRUD, inventory adjustment, batch dispatch triggers, customer CRM, and S&OP planning settings.
4. **`warehouse-app-android` (`com.pegasusx.warehouse`):**
   - 44 screens.
   - WMS inventory scanning, cycle counting, bin-to-bin transfers, pick-wave verification.
   - Offline queue: `WarehouseOfflineQueue.kt` leveraging `PrefsOfflineQueueStore`.
5. **`factory-app-android` (`com.pegasusx.factory`):**
   - 62 Kotlin files.
   - Loading-bay management (`LoadingBayScreen.kt`), transfer dispatch, quality control (QC) inspection recording, and SLA boards.
   - Realtime: `FactoryRealtimeClient.kt` decodes `FACTORY_*` outbox frames.
6. **`payload-app-android` (`com.pegasusx.payload`):**
   - 50 Kotlin files.
   - Manifest board (`ManifestBoard.kt`), barcode scan ledger, truck seal operations (`POST /v1/payloader/manifests/seal-all`), order reassignment.
   - Persistence: `PayloadDatabase.kt` Room database.

---

### 2.2 SwiftUI iOS Applications (`apps/*-ios`)

All six iOS applications are built natively using modern Swift and SwiftUI:
- **Toolchain & Standards:** Swift 5.10 / Swift 6, iOS 17+ deployment target, structured concurrency (`async`/`await`, `Task`, `Sendable`).
- **State & Architecture:** Observable view models (`@Observable` / `ObservableObject`), SwiftData / CoreData for local caches, Keychain-backed secure token storage (`TokenStore.swift`).
- **Shared Swift Packages (`packages/mobile-ios-*`):**
  - `mobile-ios-core`:
    - `PegasusNetworking`: `CellApi.swift`, `MarketPack.swift`, `RealtimeRefresh.swift` (`SilentRefreshModifier`).
    - `PegasusLiveActivities`: `DeliveryActivityAttributes.swift`, `DeliveryLiveActivityWidget.swift`.
    - `PegasusUIKit`: `CollapsibleSidebar.swift`, `PegasusMonochromeTheme.swift`, `StatusStack.swift`.
  - `mobile-ios-kit`:
    - `PegasusKit`: `OfflineHttpSemantics.swift`, `QueuedMutationRecord.swift`, `ReconnectBackoff.swift`.
  - `mobile-ios-barcode`: AVFoundation-based camera barcode scanning views.
- **Apple Ecosystem Features:**
  - **Live Activities & Dynamic Island:** `DeliveryLiveActivityWidget.swift` and `DriverLiveActivityManager.swift` track active order deliveries in real time on the lock screen and Dynamic Island.
  - **Voice Navigation:** `NavigationVoiceAnnouncer.swift` using `AVSpeechSynthesizer` for turn-by-turn audible prompts during deliveries.
  - **Haptics:** `Haptics.swift` using `UIImpactFeedbackGenerator` and `UINotificationFeedbackGenerator`.
  - **Privacy Manifests:** Mandatory `PrivacyInfo.xcprivacy` present across all 6 applications.

#### iOS App Role Breakdown:

1. **`driver-app-ios` (`apps/driver-app-ios`):**
   - 110 Swift source files, 74 views.
   - Telemetry: `TelemetryServiceLive.swift` implements WebSocket telemetry to `/v1/ws?sv=2` with ping/pong loop and exponential reconnect.
   - GPS smoothing: `LocationInterpolator.swift` performs dead-reckoning and smooth marker interpolation.
   - Offline storage: SwiftData `OfflineDeliveryStore.swift` and `DriverOfflineQueue.swift`.
   - Generated WebSocket contract: `driverappios/Generated/PegasusWSEventEnvelope.swift` generated via Quicktype.
2. **`retailer-app-ios` (`apps/retailer-app-ios`):**
   - 49 SwiftUI views.
   - SwiftData `PendingPosStore.swift` for offline POS transactions.
   - Real-time hub: `RetailerWebSocket.swift` listening on `/v1/ws`.
   - Control tower navigation: `ControlTowerHubView.swift` and `TrackingMapView.swift`.
3. **`supplier-app-ios` (`apps/supplier-app-ios`):**
   - 68 SwiftUI views.
   - `SupplierOperationsService.swift` and `SupplierRealtimeClient.swift`.
4. **`warehouse-app-ios` (`apps/warehouse-app-ios`):**
   - 84 SwiftUI views.
   - `WarehouseOperationsService.swift` and `WarehouseRealtimeClient.swift`.
5. **`factory-app-ios` (`apps/factory-app-ios`):**
   - 70 Swift files.
   - `FactoryService.swift`, `FactoryRealtimeClient.swift`, and loading-bay barcode scanning workflows.
6. **`payload-app-ios` (`apps/payload-app-ios`):**
   - 43 Swift files.
   - `APIClient.swift` wired to `seal-all` and manifest ship-unit variance APIs; `OfflineQueue.swift`.

---

## 3. Shared Contracts, Types & Code Generation Pipeline

### 3.1 Single Source of Truth (`events.go`)
The canonical source of truth for all domain events across the PegasusX ecosystem is Go backend file `apps/backend-go/events/events.go`. Event definitions and payload structs are tagged with AST doc directives such as `@Sync` and `@Sync(PayloadStructName)`.

### 3.2 Automated Generator (`cmd/gen-contracts`)
The command `apps/backend-go/cmd/gen-contracts/main.go` parses the Go AST and extracts the canonical event catalog:
- **Flags:**
  - `-source`: Path to `events/events.go`
  - `-mode`: `registry` | `json-schema` | `all`
  - `-schema-out`: Writes the unified JSON-Schema to `contracts/events.schema.json`
  - `-ts-out`: Generates TypeScript interfaces
  - `-strict=true`: Fails execution if any `@Sync` event lacks a mapped payload struct.
- **CI Gate:** Verified by `make gen-contracts-gate` (`scripts/parity/gen_contracts_gate.sh`), which executes `gen-contracts` and asserts that `contracts/events.schema.json` has zero unstaged git diff.

### 3.3 Quicktype Native Model Generation
Both Android and iOS projects wire automated Quicktype tasks to ensure compile-time type safety:
- **Android Gradle Task (`generateWsEventModels`):**
  ```kotlin
  commandLine(
      quicktypeBinary,
      "--lang", "kotlin",
      "--src-lang", "schema",
      "--src", contractsSchemaFile.absolutePath,
      "--package", "com.pegasusx.<role>.generated.contracts",
      "--framework", "kotlinx",
      "--top-level", "PegasusWSEventEnvelope",
      "--out", generatedWsModelFile.absolutePath
  )
  ```
  Integrated in `app/build.gradle.kts` for all 6 Android applications.
- **iOS Xcode Build Phases:**
  Configured in `.xcodeproj` shell script build phases:
  ```bash
  go run ./cmd/gen-contracts -source events -mode json-schema -schema-out "$SCHEMA_PATH" -pretty=true
  quicktype --lang swift --src-lang schema --src "$SCHEMA_PATH" --top-level PegasusWSEventEnvelope --just-types --out "$OUTPUT_PATH"
  ```
  Output stored in `apps/<role>-app-ios/.../Generated/PegasusWSEventEnvelope.swift`.

### 3.4 Monorepo Shared Packages
1. **`packages/types`:** 16 domain files defining TypeScript types for all entities:
   - `primitives.ts`, `supplier.ts`, `warehouse.ts`, `claims.ts`, `compliance.ts`, `auto-order.ts`, `events.ts`, `event-payloads.ts`, `envelope.ts`, `fleet.ts`, `market.ts`, `admin.ts`, `partner.ts`, `problem-detail.ts`, `notifications.ts`, `base.ts`.
2. **`packages/api-core` (`packages/api-client` symlink):**
   - `index.ts` (128 KB): Exhaustive, type-safe fetch bindings for all backend routes.
   - `idempotency.ts` (31 KB): Synchronous SHA-256 deterministic key builders (`driverDeliverKey`, `driverCollectCashKey`, `orderCreateKey`, etc.) guaranteeing idempotency across reconnects.
   - `market-pack.ts` (5.2 KB): Core market pack parsing (`MarketPack`, `AuthSession`), fiscal receipt formatting, currency formatting (`formatPackMoney`), and camera centering (`packMapCenter`, `mapInitialViewState`).
   - `cell-api.ts`: Decodes `home_cell` claim from JWT tokens (`cell-uz`, `cell-eu`, `cell-us`) and pins the API base URL.
3. **`packages/ws-refresh-contract`:**
   - Defines event sets (`ORDER_STATUS_REFRESH_EVENTS`, `DISPATCH_REFRESH_EVENTS`, `RETURN_REFRESH_EVENTS`, `FAILED_DELIVERY_REFRESH_EVENTS`, `PREORDER_REFRESH_EVENTS`, `INVENTORY_REFRESH_EVENTS`).
   - Provides `parseSSEEventData` and event discrimination helpers used across all web and mobile clients for cache invalidation.
4. **`contracts/partner.openapi.yaml` & `contracts/jwt-core.openapi.yaml`:**
   - `jwt-core.openapi.yaml`: OpenAPI 3.0.3 spec (~45 operations) covering human JWT authentication, session discovery, and core operations.
   - `partner.openapi.yaml`: OpenAPI 3.0.3 spec (1,271 lines) for B2B partner machine integrations (OAuth2 client credentials, catalog sync, order ingestion, webhook rotation, EDI-lite/AS2). Validated by `make partner-openapi-gate`.

---

## 4. Realtime, Geospatial & Offline Sync Engines

### 4.1 Realtime Communication Matrix

| Role / Surface | Protocol & Path | Invalidation & Refresh Hook | Reconnect & Recovery |
| :--- | :--- | :--- | :--- |
| **Supplier** | SSE `/v1/supplier/events` & WS `/v1/ws` | `useSupplierWsRefresh.ts` (debounced by event type) | Native SSE `retry:` directive; reconnect triggers `runSupplierSessionReconcile()` |
| **Retailer Desktop** | WebSocket `/v1/ws?token=...` | `lib/ws.tsx` (`WebSocketProvider`, `subscribe()`) | Exponential backoff with random jitter (base 3s, max 60s); advances `reconnectEpoch` |
| **Driver (Mobile)** | Dual Telemetry `/v1/ws?sv=2` & REST `/v1/driver/location` | `TelemetrySocket.kt` (Android) / `TelemetryServiceLive.swift` (iOS) | Active 30s ping loop; connection state flow; auto-reconnect on OS network regain |
| **Warehouse** | WS `/v1/ws` & SSE `/v1/warehouse/events` | `useWarehouseWsRefresh.ts` / `WarehouseRealtimeClient` | Advances dirty cache epoch; triggers background revalidation |
| **Factory** | WebSocket `/v1/ws` | `FactoryRealtimeClient.kt` / `FactoryRealtimeClient.swift` | Listens for `FACTORY_MANIFEST_*` and `TRANSFER_*` outbox frames |
| **Platform Admin** | WebSocket `/v1/ws?token=...` | `useAdminWsRefresh.ts` | Listens for `PLATFORM_ADMIN_AUDIT` signals; triggers optimistic panel refresh |
| **Native UI Hooks** | Invalidation bus | `SilentRefreshModifier` (SwiftUI) / `RealtimeRefresh.kt` (Compose) | ViewModels implement `load(silent: true)` to reload data without flashing full-screen loading spinners |

### 4.2 Geospatial Architecture & Map Standardization
- **Standard Map Engine:** MapLibre GL (`maplibre-gl: ^5.19.0`) and `react-map-gl/maplibre: ^8.1.0`.
- **Basemap Style:** Carto Positron vector tiles (`https://basemaps.cartocdn.com/gl/positron-gl-style/style.json`).
- **Dynamic Camera Centering (`mapInitialViewState`):**
  - Reads `pack` from `readCachedAuthSession()?.pack` (or `com.pegasus.design.sessionMapCenter()` in mobile).
  - Inspects `map_center_lat` and `map_center_lng`.
  - **Honesty Rule:** If pack status is not `"shipped"` or coordinates are `(0, 0)`, returns default global view (`{ latitude: 0, longitude: 0, zoom: 1 }`). **Never invents a hardcoded Tashkent or San Francisco camera.**
- **Hexagonal Spatial Aggregations:** Uber H3 (`h3-js: ^4.5.0` on Web, `com.uber.h3core:H3Core` on Android) generates geographic boundaries for hex density layers (resolution 7 for dispatch perimeters, resolution 9 for settlement clusters).
- **Mapbox Elimination:** Zero fallback tokens or Mapbox SDK dependencies exist in production client code.

### 4.3 Offline Mutation Queues & Caching Strategy

```
┌────────────────────────────────────────────────────────────────────────┐
│                        OFFLINE MUTATION PIPELINE                       │
├───────────────────┬────────────────────────────┬───────────────────────┤
│    Client Role    │     Storage Substrate      │    Replay Trigger     │
├───────────────────┼────────────────────────────┼───────────────────────┤
│ Driver (Android)  │ Room SQLite                │ WorkManager Worker    │
│                   │ (PendingMutationEntity)    │ on Network Regain     │
├───────────────────┼────────────────────────────┼───────────────────────┤
│ Driver (iOS)      │ SwiftData / SQLite         │ Background Task /     │
│                   │ (OfflineDeliveryStore)     │ Network Monitor       │
├───────────────────┼────────────────────────────┼───────────────────────┤
│ Warehouse/Factory │ Android SharedPreferences  │ Foreground Lifecycle  │
│ (Android)         │ (PrefsOfflineQueueStore)   │ Replay Hook           │
├───────────────────┼────────────────────────────┼───────────────────────┤
│ Retailer Desktop  │ Tauri SQLite Plugin        │ Background Reconnect  │
│                   │ (pending_commands)         │ Worker                │
├───────────────────┼────────────────────────────┼───────────────────────┤
│ Retailer (iOS)    │ SwiftData                  │ Foreground Replay     │
│                   │ (PendingPosStore)          │ Scheduler             │
├───────────────────┼────────────────────────────┼───────────────────────┤
│ Payload Terminal  │ Expo SecureStore           │ Manual / Connection   │
│                   │ (Encrypted KV)             │ Restored              │
└───────────────────┴────────────────────────────┴───────────────────────┘
```

#### Driver Offline Priority Hierarchy (`DriverOfflineActionCatalog`):
Both Android and iOS enforce the exact same deterministic priority order for queued actions:
1. **Priority 10 (Proximity):** `v1/delivery/proximity-unlock` (max age 120s; discarded if stale).
2. **Priority 20 (Delivery Decisions):** `v1/delivery/shop-closed`, `v1/delivery/partial-offload`, `v1/order/deliver`.
3. **Priority 30 (Financial Collections):** `v1/order/collect-cash`, `v1/delivery/credit-delivery`.
4. **Priority 40 (General Operations):** `v1/delivery/arrive`, `v1/order/confirm-offload`, `v1/order/complete`, `v1/delivery/split-payment`, `v1/delivery/bypass-offload`, `v1/delivery/confirm-payment-bypass`, `v1/fleet/driver/depart`, `v1/fleet/driver/return-complete`, `v1/driver/cash-reconciliations`, `v1/fleet/route/reorder`, `v1/driver/availability`, and fiscal retries.

#### Queue Poisoning Prevention:
- **Fail-Closed on Business Errors:** HTTP 4xx errors (e.g., `401 Unauthorized`, `403 Forbidden`, `422 Invalid Transition`) are marked as un-retryable (`isNetworkEnqueueable() = false`) and are discarded or flagged as `DEAD`.
- **Retryable HTTP Codes:** Only transport failures (network offline, timeouts, connection drops) and transient server errors (`502`, `503`, `504`, `429`) are allowed to remain in the queue with exponential backoff.

---

## 5. Role-Row Parity Audit & Contract Verification

### 5.1 Parity Audit Script Execution Results
During this investigation session, the canonical repository verification scripts were executed directly in terminal:

1. **`bash scripts/parity/role_row_contract_check.sh`:**
   - **Result:** `role-row-contract-ok` (Exit Code 0).
   - Validates that critical API client symbols (`getSupplierNegotiationsPending`, `resolveSupplierNegotiation`, `getRetailerTracking`, etc.) are actively imported and referenced by each corresponding role-row application.
2. **`bash scripts/parity/role_row_contract_check_full.sh`:**
   - **Result:** `role-row-contract-full-ok` (Exit Code 0).
   - Dynamically scans every `/v1/...` route registration in `apps/backend-go` across all 29 route packages and matches them against all HTTP calls made across every web, desktop, and mobile application. Confirms zero dangling or unmounted API calls.

### 5.2 Role-Row Implementation Status Matrix

| Role Row | Web / Desktop Client | Native Android Client | Native iOS Client | Backend Route Packages | Realtime Backbone | Parity Status |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **1. Supplier** | `apps/supplier-portal` (82 App Router routes, Tauri desktop) | `apps/supplier-app-android` (61 Compose screens, `SupplierApi.kt`) | `apps/supplier-app-ios` (68 SwiftUI views, `SupplierOperationsService.swift`) | `supplierroutes`, `catalogroutes`, `creditroutes`, `pulseroutes`, `entityresolutionroutes` | SSE `/v1/supplier/events` & WS `/v1/ws` | **WIRED (Class A)** |
| **2. Retailer** | `apps/retailer-app-desktop` (31 routes, Tauri desktop) | `apps/retailer-app-android` (40+ composables, `PegasusApi.kt`) | `apps/retailer-app-ios` (49 SwiftUI views, `PendingPosStore.swift`) | `retailerroutes`, `orderroutes`, `paymentroutes`, `demandroutes` | WS `/v1/ws` | **WIRED (Class A)** |
| **3. Driver** | N/A (Dedicated Mobile by design) | `apps/driver-app-android` (63 screens, Room offline queue) | `apps/driver-app-ios` (74 views, SwiftData offline store, Dynamic Island) | `driverroutes`, `deliveryroutes`, `telemetryroutes`, `cashreconroutes` | WS `/v1/ws?sv=2` (dual telemetry) | **WIRED (Class A)** |
| **4. Warehouse** | `apps/warehouse-portal` (46 routes, Tauri desktop) | `apps/warehouse-app-android` (44 screens, `WarehouseApi.kt`) | `apps/warehouse-app-ios` (84 views, `WarehouseOperationsService.swift`) | `warehouseroutes`, `creditnoteroutes`, `returnsroutes`, `laborcapacityroutes` | WS `/v1/ws` & SSE `/v1/warehouse/events` | **WIRED (Class A)** |
| **5. Factory** | `apps/factory-portal` (21 routes, Tauri desktop) | `apps/factory-app-android` (62 files, `FactoryApi.kt`) | `apps/factory-app-ios` (70 files, `FactoryService.swift`) | `factoryroutes`, warehouse perimeter, Spanner loading bay | WS `/v1/ws` | **WIRED (Class A)** |
| **6. Payload** | `apps/payload-terminal` (Expo SDK 55, Universal RN) | `apps/payload-app-android` (50 files, `PayloadApi.kt`) | `apps/payload-app-ios` (43 files, `APIClient.swift`) | `payloaderoutes` (alias `payloadroutes`) | WS `/v1/ws` | **WIRED (Class A)** |
| **7. Platform Admin** | `apps/admin-portal` (Next.js 15, 9 governance panels) | N/A (Desktop Web only by design) | N/A (Desktop Web only by design) | `platformadmin`, `featureflags`, `mfa`, `taxroutes`, `partner` | WS `/v1/ws` (`PLATFORM_ADMIN_AUDIT`) | **WIRED (Class A)** |

### 5.3 Architectural Honesty & Disabled Endpoints (Layer A vs Layer B)

To avoid misleading claims of "production-readiness" or "cloud-wiring" where underlying services are intentionally un-scoped or disabled, the PegasusX codebase enforces clear boundary contracts:

| Feature / Surface | Route & Method | Wire Behavior | Ground Truth & Code Reference |
| :--- | :--- | :--- | :--- |
| **Saved Cards Vault** | `/v1/retailer/card*` | **HTTP 410 GONE** | `retailer/core_handlers.go:1337` returns `saved_cards_not_product`. B2B flow uses COD cash/credit or one-time payment redirect. |
| **AI Predictions Legacy Alias** | `GET /v1/ai/predictions` | **HTTP 410 GONE** | `retailer/mobile_compat.go:71-81` returns `use_retailer_ai_predictions`. Clients actively target `/v1/retailer/ai/predictions`. |
| **Inventory Audit Ledger** | `GET /v1/supplier/inventory/audit` | **HTTP 410 GONE** | `supplier/portal_handlers.go:1107-1118` returns `audit_unwired`. Live clients query standard inventory adjustment list. |
| **Quantity Negotiation** | `POST /v1/delivery/negotiate`<br>`POST /v1/supplier/negotiate/resolve` | **HTTP 410 GATED** | `order/negotiation_disabled.go:22-30` returns `feature_disabled` unless environment variable `QUANTITY_NEGOTIATION_ENABLED=true` is set. |
| **Payme & Click Webhooks** | `/v1/webhooks/payme`<br>`/v1/webhooks/click` | **COMMENTED** | `webhookroutes/routes.go:26-31` routes are commented out. Active payment rails are Cash + GlobalPay + MySoliq. |
| **Vehicle Capacity GET** | `GET /v1/payloader/capacity` | **HTTP 410 GONE** | `payload/vehicle_capacity.go:19` returns `capacity_unwired`. Volume utilization is computed directly from manifest ship-units. |
| **Post-Dispatch Order Cancel** | `POST /v1/order/{id}/cancel` | **HTTP 403 GATED** | Orders in `DISPATCHED` or `LOADED` status cannot be cancelled via standard endpoint; must follow formal return or shop-closed workflows. |
| **Auto-Order Automatic Execution** | `POST /v1/retailer/auto-order/place` | **FLAG GATED** | Shadow mode is active (`AUTO_ORDER_SHADOW=true`), but automatic order placement is disabled pending the 30-day soak gate. |
| **Factory Planning & Batcher** | Planning cron & batcher solver | **FLAG GATED (OFF)** | Go flags `PlanningEnabled()` and `BatcherEnabled()` default to `false` in production profile to avoid unauthorized automated replenishments. |

---

## 6. Discovered Gaps & Actionable Technical Debt

During testing and inspection, two specific technical debt items were uncovered:

1. **`apps/admin-portal` TypeScript Typecheck Drift:**
   - Running `pnpm --filter @pegasusx/admin-portal typecheck` produced errors:
     - `app/layout.tsx(1,15): error TS2614: Module '"next"' has no exported member 'Metadata'.`
     - `next.config.ts(1,15): error TS2614: Module '"next"' has no exported member 'NextConfig'.`
     - `lib/__tests__/command-dashboard.test.ts(4,38): error TS2307: Cannot find module 'vitest'.`
   - *Cause:* In Next.js 15, `Metadata` and `NextConfig` should be imported from `"next"` or `"next/types"` depending on TS configuration, and vitest types were not hoisted into the local tsconfig.
2. **Vitest CLI Node Resolution in pnpm virtual store:**
   - Running `vitest run` directly inside `apps/admin-portal` under Node v25 failed due to an ESM module resolution error looking for `vitest/dist/cli.js`.

---

## 7. Synthesis: Dual-System Comparison Context

This investigation directly informs the dual-system analysis required between `pegasusX` (Global Enterprise Multi-Tenant) and `pegasus.x` (Sovereign Lean Single-Tenant):

| Architectural Dimension | `pegasusX/` Client Architecture | `pegasus.x/` Target Equivalents |
| :--- | :--- | :--- |
| **Desktop Frontends** | Next.js 15 + React 19 + HeroUI + Tauri v2 (`supplier-portal`, `warehouse-portal`, `factory-portal`, `retailer-app-desktop`) | Next.js 15 + Tauri v2 (`apps/supplier-desktop`, `apps/warehouse-desktop`) |
| **Mobile Frontends** | 6 Native Kotlin Android + 6 Native SwiftUI iOS applications | Telegram Mini App + Bot (`apps/telegram-*`) for Retailers; Native Android/iOS for Drivers |
| **Contract Synchronization** | Monorepo `packages/types` + AST generator `gen-contracts` + Quicktype Kotlin/Swift stubs + JSON-Schema | Shared Go types + TypeScript interfaces for Tauri apps |
| **Realtime Transport** | WebSocket Hub `/v1/ws` & `/v1/ws?sv=2` backed by Kafka + Spanner Outbox | WebSocket Hub backed by PostgreSQL Outbox + Redis Streams |
| **Fleet & Driver Lifecycle** | Complete dynamic shift assignments, mid-shift truck swapping, DVIR vehicle pre-trip inspection, and route reordering across both Android & iOS | **Critical Gap in `pegasus.x`:** Needs logical port of Fleet & Driver management entities and UI flows. |
