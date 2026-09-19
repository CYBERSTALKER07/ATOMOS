# Requirement 3 (R3: UI Consistency) — Detailed Investigation Report

## 1. Executive Summary

This investigation surveys the codebase for **Requirement 3 (R3: UI Consistency)** of the PegasusX platform, focusing on:
1. **Control-Tower Web Map & Retailer Android Hex Map**: Identifying MapLibre + Carto vs Mapbox provider usage, removing hardcoded San Francisco coordinates and Mapbox fallback tokens, and establishing dynamic pack-based camera initialization (`mapInitialViewState(pack)`).
2. **Factory & Retailer Mobile Applications**: Auditing Android and iOS codebases for "wired later" or stubbed UI theatre (fake canvases, dummy mock nodes, empty placeholder maps) and defining honest resolutions.
3. **Admin-Portal Frontend**: Auditing `apps/admin-portal` dependencies, types, and UI components to plan migration to `packages/types` and `@pegasusx/ui-kit`.

---

## 2. Task 1: Control-Tower Web Map & Retailer Android Hex Map

### 2.1 Control-Tower Web Map (`HexagonalControlTowerMap.tsx`)

- **File Path**: `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/packages/ui-kit/src/control-tower/HexagonalControlTowerMap.tsx`
- **Current Observations**:
  - **Lines 4, 8**: Imports Mapbox GL and its CSS:
    ```tsx
    import Map from "react-map-gl/mapbox";
    import "mapbox-gl/dist/mapbox-gl.css";
    ```
  - **Lines 10-12**: Defines a hardcoded Mapbox fallback access token:
    ```tsx
    const MAPBOX_ACCESS_TOKEN =
      process.env.NEXT_PUBLIC_MAPBOX_TOKEN ||
      "pk.eyJ1IjoiZGVmYXVsdCIsImEiOiJjbHg1bW14Mm0wMTI2MmpxaXV3eWY2bmM2In0.default";
    ```
  - **Lines 43-50**: Hardcodes San Francisco initial camera coordinates:
    ```tsx
    const initialViewState = {
      longitude: -122.4,
      latitude: 37.74,
      zoom: 11,
      maxZoom: 20,
      pitch: view3D ? 30 : 0,
      bearing: 0,
    };
    ```
  - **Line 67**: Renders Mapbox dark-v11 style:
    ```tsx
    <Map mapboxAccessToken={MAPBOX_ACCESS_TOKEN} mapStyle="mapbox://styles/mapbox/dark-v11" />
    ```

- **Usage Sites in Portals**:
  - `apps/warehouse-portal/app/control-tower/page.tsx:95`: `<HexagonalControlTowerMap data={displayH3Data} />`
  - `apps/supplier-portal/app/(portal)/control-tower/page.tsx:107`: `<HexagonalControlTowerMap data={wsH3Data} />`

- **Standard Implementation Pattern in Codebase**:
  - Standard established in `apps/warehouse-portal/components/FleetLiveMap.tsx` and `apps/retailer-app-desktop/components/tracking/TrackingMap.tsx`:
    - Provider: `react-map-gl/maplibre` and `maplibre-gl`
    - Style: `https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json` (for dark mode) or `https://basemaps.cartocdn.com/gl/positron-gl-style/style.json` (for light mode)
    - Dynamic Pack Camera: `mapInitialViewState(pack)` from `@pegasusx/api-client`
    - No Mapbox token or dependency required.

- **Proposed Changes for `HexagonalControlTowerMap.tsx`**:
  1. Replace `react-map-gl/mapbox` with `react-map-gl/maplibre` and import `maplibregl from "maplibre-gl"`.
  2. Accept optional `pack?: MarketPack | null` in `HexagonalControlTowerMapProps` (or fallback to `readCachedAuthSession()?.pack`).
  3. Derive `initialViewState` using `mapInitialViewState(pack, 11)`:
     ```tsx
     const packView = mapInitialViewState(pack ?? readCachedAuthSession()?.pack, 11);
     const initialViewState = {
       longitude: packView.longitude,
       latitude: packView.latitude,
       zoom: packView.zoom,
       maxZoom: 20,
       pitch: view3D ? 30 : 0,
       bearing: 0,
     };
     ```
  4. Render `<Map mapLib={maplibregl} mapStyle="https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json" />`
  5. Remove all references to `MAPBOX_ACCESS_TOKEN`, `mapbox-gl/dist/mapbox-gl.css`, and San Francisco `(-122.4, 37.74)`.

---

### 2.2 Retailer Android Hex Map (`HexagonalControlTowerMap.kt`)

- **File Path**: `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/retailer-app-android/app/src/main/java/com/pegasusx/retailer/ui/controltower/HexagonalControlTowerMap.kt`
- **Current Observations**:
  - **Lines 7-10**: Imports Google Maps Compose:
    ```kotlin
    import com.google.android.gms.maps.model.CameraPosition
    import com.google.android.gms.maps.model.LatLng
    import com.google.android.gms.maps.model.MapStyleOptions
    import com.google.maps.android.compose.*
    ```
  - **Lines 25-27**: Hardcodes San Francisco camera coordinates:
    ```kotlin
    val cameraPositionState = rememberCameraPositionState {
        position = CameraPosition.fromLatLngZoom(LatLng(37.74, -122.4), 11f)
    }
    ```
  - **Lines 30-74**: Contains a hardcoded dark map style JSON string for Google Maps.
  - **Lines 76-97**: Renders `GoogleMap` with individual `Polygon` overlays for each hex cell.

- **Standard MapLibre Implementation Pattern on Android**:
  - Established in `apps/warehouse-app-android/app/src/main/java/com/pegasusx/warehouse/ui/components/FleetLiveMapLibre.kt`:
    - Uses `org.maplibre.android.maps.MapView` via Compose `AndroidView`.
    - Uses Style URL: `"https://basemaps.cartocdn.com/gl/positron-gl-style/style.json"` or `"https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json"`.
    - Manages Android lifecycle events via `LifecycleEventObserver`.
    - Dynamic camera update using market pack coordinates or bounding box.

- **Proposed Changes for `HexagonalControlTowerMap.kt`**:
  1. Migrate from `GoogleMap` Compose to `MapLibre` (`org.maplibre.android.maps.MapView`) with Carto dark style (`https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json`).
  2. Accept `marketCenterLat: Double?` and `marketCenterLng: Double?` (or session pack center). Default to market pack coordinates instead of San Francisco `(37.74, -122.4)`.
  3. Render H3 polygon GeoJSON features via `GeoJsonSource` and `FillLayer` in MapLibre, matching the performance and rendering standards of the web and warehouse Android apps.

---

## 3. Task 2: Factory and Retailer Mobile Applications UI Theatre Audit

### 3.1 Audit of Factory Mobile Applications

- **Factory Android** (`apps/factory-app-android`):
  - `ui/screens/fleet/FleetScreen.kt`:
    - Cleanly displays vehicle roster (`vehicles`) and live drivers with real GPS coordinates (`liveRoutes`).
    - Uses `FactoryOpsListCard` with status (`LIVE` / `OFFLINE`) and exact latitude/longitude.
    - No fake map theatre or dummy simulation.
  - `ui/screens/location/LocationSettingsScreen.kt` & `ui/components/AddressLocationField.kt`:
    - Truthfully provides forward/reverse geocoding and address suggestion lookup via `GeocodeApi` and fused location provider without dummy map widgets.

- **Factory iOS** (`apps/factory-app-ios`):
  - `FactoryApp/Views/Fleet/FleetView.swift`:
    - Cleanly renders live drivers section with GPS coordinates (`liveRoutes`) and vehicle fleet roster.
    - Truthful empty states and error handling; no misleading canvas or stubbed map.

### 3.2 Audit of Retailer Mobile Applications

- **Retailer Android** (`apps/retailer-app-android`):
  - `ui/controltower/ControlTowerScreen.kt`:
    - Truthfully renders `PulseTile` grid (`Open orders`, `Fulfillment`, `Dock pending`, `POS sessions`, `Open shifts`, `Assist`, `Low stock`, `Variances`, `Sales 7d`) backed by `api.getControlTowerPulse()`.
    - Has honest empty state: `"No live ops signals yet"`.
  - `ui/controltower/LiveEKGNetworkGraph.kt`:
    - **Problem**: Contains an unlinked canvas graph with hardcoded mock layout and fake pulse animations.
    - **Resolution**: Since `ControlTowerScreen.kt` directly uses the truthful `PulseTile` grid, `LiveEKGNetworkGraph.kt` is dead theatre code. Drop/clean this component or ensure it is not used in production navigation.
  - `ui/controltower/HexagonalControlTowerMap.kt`:
    - **Problem**: Uses GoogleMap and hardcoded SF coordinates (analyzed in Task 1).
    - **Resolution**: Standardize to MapLibre + Carto with dynamic pack center, or remove if the pulse grid is the authoritative mobile view.

- **Retailer iOS** (`apps/retailer-app-ios`):
  - `retailerapp/Screens/ControlTower/HexagonalControlTowerMap.swift`:
    - **Problem**: Lines 17 & 26 have explicit comments:
      ```swift
      Map(position: $cameraPosition) {
          // Map content wired to API later
      }
      private func generateData() {
          // Wired to API
      }
      ```
    - **Resolution**: Remove this dummy wrapper.
  - `retailerapp/Screens/ControlTower/LiveEKGNetworkGraph.swift`:
    - **Problem**: Lines 76-88 construct a fake static graph (`nodes = [warehouse, retailer1, retailer2, driver1, driver2]`) with dummy Bezier curves.
    - **Resolution**: Drop this mock canvas. `ControlTowerView.swift` already provides an honest, data-backed list of pulse tiles with navigation links.
  - `retailerapp/Screens/DeliveryMapView.swift`:
    - Truthfully displays live tracking orders (`visibleOrders`), fiscal receipt links, QR codes, and fallback empty states (`"No active deliveries with driver location"`).
    - Uses standard `packMapCoordinate()` for pack-aware camera centering.

---

## 4. Task 3: Admin-Portal Frontend Application Migration

### 4.1 Dependency Audit (`apps/admin-portal/package.json`)

- **Current `dependencies`**:
  ```json
  "dependencies": {
    "next": "^15.0.0",
    "react": "19.0.0",
    "react-dom": "19.0.0"
  }
  ```
- **Missing Workspace Dependencies**:
  - `@pegasusx/types`: `"workspace:*"`
  - `@pegasusx/ui-kit`: `"workspace:*"`

### 4.2 Local Types to Migrate (`apps/admin-portal/lib/api.ts`)

| Local Type in `admin-portal/lib/api.ts` | Target Type in `@pegasusx/types` | Status & Migration Path |
|---|---|---|
| `Tenant` | `Tenant` / `PlatformAdminTenant` in `@pegasusx/types` | Move/export canonical interface to `@pegasusx/types` |
| `FlagOverride` | `FlagOverride` in `@pegasusx/types` | Move/export canonical interface to `@pegasusx/types` |
| `FlagEval` | `FlagEval` in `@pegasusx/types` | Move/export canonical interface to `@pegasusx/types` |
| `AccuracyRow` | `AccuracyRow` in `@pegasusx/types` | Move/export canonical interface to `@pegasusx/types` |
| `AuditRow` | `AuditRow` in `@pegasusx/types` | Move/export canonical interface to `@pegasusx/types` |
| `MatchQueueItem` | `MatchQueueItem` in `@pegasusx/types` | Move/export canonical interface to `@pegasusx/types` |
| `PartnerKey` | `PartnerApiKeyMeta` / `PartnerIssuedKey` in `@pegasusx/types` | Align with `PartnerApiKeyMeta` in `@pegasusx/types` |
| `BillingInvoice` | `BillingInvoice` in `@pegasusx/types` | Move/export canonical interface to `@pegasusx/types` |
| `BillingFeeSchedule` | `BillingFeeSchedule` in `@pegasusx/types` | Move/export canonical interface to `@pegasusx/types` |

### 4.3 UI Kit Adoption Strategy

1. **Styles**:
   - Update `apps/admin-portal/app/globals.css` to import `@pegasusx/ui-kit/styles/portal-ui.css` and `@pegasusx/ui-kit/styles/desktop-foundation.css`.
2. **Components**:
   - In `TenantsPanel.tsx`, `FlagsPanel.tsx`, `AuditPanel.tsx`, `MatchQueuePanel.tsx`, `PartnerPanel.tsx`, `OpsPanel.tsx`, `BillingPanel.tsx`, `AccuracyPanel.tsx`:
     - Adopt `PortalSection`, `PortalField`, `PortalInput`, `PortalSelect` for standard forms.
     - Adopt `HealthStrip` / `KpiStat` in `CommandBoard.tsx`.
     - Adopt `StatusStack` or `StatusBadge` for tenant and flag lifecycle badges.
3. **Tests**:
   - Verify `vitest run` passes for `apps/admin-portal/lib/__tests__/command-dashboard.test.ts`.

---

## 5. Verification & Test Plan

1. **TypeScript Build & Typecheck**:
   - `pnpm --filter @pegasusx/ui-kit typecheck`
   - `pnpm --filter @pegasusx/admin-portal typecheck`
   - `pnpm --filter @pegasusx/admin-portal test`
2. **Grep Validation**:
   - Verify zero occurrences of `pk.eyJ1IjoiZGVmYXVsdC` (Mapbox fallback token).
   - Verify zero hardcoded `-122.4` / `37.74` coordinates in map initializers.
   - Verify zero "wired later" comments in mobile app map screens.


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
