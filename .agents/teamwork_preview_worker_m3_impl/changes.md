# Milestone 3 (M3: UI Consistency) — Changes Summary

## 1. Control-Tower Web Map Standardization
- **File**: `pegasusX/packages/ui-kit/src/control-tower/HexagonalControlTowerMap.tsx`
  - Replaced legacy Mapbox imports with `react-map-gl/maplibre` and `maplibre-gl`.
  - Removed all hardcoded San Francisco coordinates `(-122.4, 37.74)`, fallback token strings (`pk.eyJ1IjoiZGVmYXVsdC...`), and `mapbox-gl/dist/mapbox-gl.css`.
  - Adopted Carto dark style: `https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json`.
  - Bound the camera dynamically to the active market pack via `mapInitialViewState(pack ?? readCachedAuthSession()?.pack, 11)` from `@pegasusx/api-client`.
  - Added optional `pack?: MarketPack | null` prop to `HexagonalControlTowerMapProps`.

## 2. Mobile UI Theatre Cleanup
- **Android Retailer App**:
  - `pegasusX/apps/retailer-app-android/app/src/main/java/com/pegasusx/retailer/ui/controltower/HexagonalControlTowerMap.kt`: Removed hardcoded San Francisco coordinates (`37.74, -122.4`) in favor of dynamic session market pack center via `com.pegasus.design.sessionMapCenter()`.
  - `pegasusX/apps/retailer-app-android/app/src/main/java/com/pegasusx/retailer/ui/controltower/LiveEKGNetworkGraph.kt`: Removed fake simulated animation canvas and routed directly to the truthful ops pulse view (`ControlTowerScreen.kt`).
  - Preserved live operational counters and empty states in `ControlTowerScreen.kt`.
- **iOS Retailer App**:
  - `pegasusX/apps/retailer-app-ios/retailerapp/retailerapp/Screens/ControlTower/HexagonalControlTowerMap.swift`: Removed "wired later" comment stubs and bound camera position to dynamic market pack coordinates via `packMapCoordinate()`.
  - `pegasusX/apps/retailer-app-ios/retailerapp/retailerapp/Screens/ControlTower/LiveEKGNetworkGraph.swift`: Removed mock hardcoded node arrays and "wired later" stubs, rendering the honest `ControlTowerView()` pulse screen.
  - Preserved live operational signals and honest empty states in `ControlTowerView.swift`.

## 3. Admin-Portal Migration & Types Harmonization
- **File**: `pegasusX/apps/admin-portal/package.json`
  - Added workspace dependencies `"@pegasusx/types": "workspace:*"` and `"@pegasusx/ui-kit": "workspace:*"`.
- **File**: `pegasusX/packages/types/index.ts`
  - Exported canonical Platform Admin DTO interfaces: `Tenant`, `FlagOverride`, `FlagEval`, `AccuracyRow`, `AuditRow`, `MatchQueueItem`, `PartnerKey`, `BillingInvoice`, `BillingFeeSchedule`.
- **File**: `pegasusX/apps/admin-portal/lib/api.ts`
  - Imported canonical DTO types from `@pegasusx/types`, eliminating duplicate local type declarations.
- **File**: `pegasusX/apps/admin-portal/app/globals.css`
  - Imported `@pegasusx/ui-kit/styles/desktop-foundation.css` and `@pegasusx/ui-kit/styles/portal-ui.css`.
- **File**: `pegasusX/apps/admin-portal/components/FlagsPanel.tsx` and UI components
  - Adopted `@pegasusx/ui-kit/portal` UI primitives (`PortalField`, `PortalInput`, `PortalSelect`, `FormAlert`).
  - Preserved all required test-contract markers and assertions: `gs-u-admin-health`, `gs-u-admin-command`, `gs-u-admin-tenants`, `gs-u-admin-flags`, `gs-u-admin-queues`, `gs-u-admin-accuracy`, `gs-u-admin-dlq-count`, `IsRegistered is tenant-register`, `COUNT(*)`, and `no invented mape28 line`.

## 4. Verification
- Zero Mapbox fallback tokens (`pk.eyJ1IjoiZGVmYXVsdC`) in active codebase.
- Zero hardcoded San Francisco coordinates (`37.74`, `-122.4`) in map views.
- Zero "wired later" comments or mock node arrays in mobile applications.
- All test contracts in `admin-portal` (`lib/__tests__/command-dashboard.test.ts`) satisfied.


# Universal Agent & Engineering Guidelines
When developing, designing, or planning, always ensure to account for:
- Gaps, edge cases, and comprehensive feature validation.
- Best practices and optimized integration for Kafka, Redis, Backend, Optimizers, AI, and UI.
- Real-time concepts including WebSockets, webhooks, and their native app equivalents.
- Thorough business logic for features, understanding how the role, app, and ecosystem work together, and engagements with other roles and features.
- Best practices for backend, frontend, and infrastructure libraries/packages. Always prefer existing, high-quality open-source libraries and packages that best suit our features before creating our own.
- Optimal UI infrastructure and UX patterns (e.g., optimal screen positioning for drivers during an active route), applying the same high standards to backend and cloud architecture.
- ALWAYS search the web to find open-source code, libraries, packages, math, algorithms, approaches, and best practices for anything we are doing. If none exist, then create our own.
- Always search the web to get the correct logic, and incorporate edge cases, business logic for features, operations (ops), workflow, data consistency, finance, and AI into everything we do.
