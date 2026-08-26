# Handoff Report — Requirement 3 (R3: UI Consistency)

## 1. Observation

Direct observations from the PegasusX codebase:

### 1.1 Control-Tower Web Map Component
- **File**: `pegasusX/packages/ui-kit/src/control-tower/HexagonalControlTowerMap.tsx`
  - **Line 4**: `import Map from "react-map-gl/mapbox";`
  - **Line 8**: `import "mapbox-gl/dist/mapbox-gl.css";`
  - **Lines 10-12**: 
    ```tsx
    const MAPBOX_ACCESS_TOKEN =
      process.env.NEXT_PUBLIC_MAPBOX_TOKEN ||
      "pk.eyJ1IjoiZGVmYXVsdCIsImEiOiJjbHg1bW14Mm0wMTI2MmpxaXV3eWY2bmM2In0.default";
    ```
  - **Lines 43-50**:
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
  - **Line 67**: `<Map mapboxAccessToken={MAPBOX_ACCESS_TOKEN} mapStyle="mapbox://styles/mapbox/dark-v11" />`

### 1.2 Reference MapLibre + Carto Implementations
- **File**: `pegasusX/packages/api-client/market-pack.ts`
  - **Lines 88-95**: Dynamic pack camera helper function `mapInitialViewState(pack, zoom = 12)` computes `{ latitude, longitude, zoom }` from `pack.map_center_lat` and `pack.map_center_lng`.
- **File**: `pegasusX/apps/warehouse-portal/components/FleetLiveMap.tsx`
  - **Lines 6-10**: Imports `MapGL` from `react-map-gl/maplibre`, `maplibregl` from `maplibre-gl`, `mapInitialViewState` and `readCachedAuthSession` from `@pegasusx/api-client`.
  - **Lines 124-131**:
    ```tsx
    initialViewState={{
      ...mapInitialViewState(readCachedAuthSession()?.pack),
      zoom: 11,
      pitch: 0,
    }}
    mapStyle="https://basemaps.cartocdn.com/gl/positron-gl-style/style.json"
    mapLib={maplibregl}
    ```

### 1.3 Retailer Android Hex Map & UI Theatre
- **File**: `pegasusX/apps/retailer-app-android/app/src/main/java/com/pegasusx/retailer/ui/controltower/HexagonalControlTowerMap.kt`
  - **Lines 7-10**: Imports Google Maps Compose (`com.google.maps.android.compose.*`, `com.google.android.gms.maps.model.*`).
  - **Lines 25-27**: Hardcoded San Francisco coordinates:
    `position = CameraPosition.fromLatLngZoom(LatLng(37.74, -122.4), 11f)`
  - **Lines 76-97**: Renders `GoogleMap` instead of MapLibre.
- **File**: `pegasusX/apps/retailer-app-android/app/src/main/java/com/pegasusx/retailer/ui/controltower/LiveEKGNetworkGraph.kt`
  - **Lines 55-69**: Hardcoded radial canvas layout with fake simulated pulse without real backend graph nodes.
- **File**: `pegasusX/apps/retailer-app-ios/retailerapp/retailerapp/Screens/ControlTower/HexagonalControlTowerMap.swift`
  - **Line 17**: `// Map content wired to API later`
  - **Line 26**: `// Wired to API`
- **File**: `pegasusX/apps/retailer-app-ios/retailerapp/retailerapp/Screens/ControlTower/LiveEKGNetworkGraph.swift`
  - **Lines 76-88**: Fake hardcoded node array `nodes = [warehouse, retailer1, retailer2, driver1, driver2]`.

### 1.4 Factory Mobile Apps
- **File**: `pegasusX/apps/factory-app-android/app/src/main/java/com/pegasusx/factory/ui/screens/fleet/FleetScreen.kt`
  - Real vehicle roster and live GPS coordinate list (`liveRoutes` with `lat, lng` and `FactoryOpsListCard`); no fake map theatre.
- **File**: `pegasusX/apps/factory-app-ios/FactoryApp/Views/Fleet/FleetView.swift`
  - Renders truthful list of live drivers with GPS coordinates and vehicles; no fake map theatre.

### 1.5 Admin-Portal Application
- **File**: `pegasusX/apps/admin-portal/package.json`
  - Lacks `@pegasusx/types` and `@pegasusx/ui-kit` in `dependencies`.
- **File**: `pegasusX/apps/admin-portal/lib/api.ts`
  - Defines local interfaces for `Tenant`, `FlagOverride`, `FlagEval`, `AccuracyRow`, `AuditRow`, `MatchQueueItem`, `PartnerKey`, `BillingInvoice`, `BillingFeeSchedule`.
- **File**: `pegasusX/apps/admin-portal/lib/__tests__/command-dashboard.test.ts`
  - Passes 4 tests with `pnpm --filter @pegasusx/admin-portal test`.

---

## 2. Logic Chain

1. **Map Standard Harmonization**:
   - In `HexagonalControlTowerMap.tsx`, Mapbox GL is used with a fallback token and hardcoded San Francisco coordinates `(-122.4, 37.74)` (Observation 1.1).
   - Across the rest of the web apps (`warehouse-portal`, `retailer-app-desktop`, `supplier-portal`), the standard is MapLibre GL + Carto basemaps with dynamic pack cameras (`mapInitialViewState(pack)`) (Observation 1.2).
   - Therefore, `HexagonalControlTowerMap.tsx` must be converted to `react-map-gl/maplibre` + Carto dark style (`https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json`), using `mapInitialViewState(pack)` and eliminating Mapbox tokens and SF coordinates.

2. **Android Hex Map & UI Theatre Cleanup**:
   - `retailer-app-android`'s `HexagonalControlTowerMap.kt` uses Google Maps with hardcoded SF coordinates `(37.74, -122.4)` (Observation 1.3).
   - `retailer-app-ios`'s `HexagonalControlTowerMap.swift` and `LiveEKGNetworkGraph.swift` have stubbed "wired later" comments and hardcoded fake nodes (Observation 1.3).
   - The truthful primary UI in both `retailer-app-android` (`ControlTowerScreen.kt`) and `retailer-app-ios` (`ControlTowerView.swift`) is the live ops pulse grid/list, which accurately renders real backend counters or honest empty states.
   - Therefore, dummy stubbed theatre should be replaced or dropped in favor of the honest pulse view, and any remaining Android hex map component must standardize to MapLibre + Carto with dynamic pack coordinates.

3. **Admin-Portal Types & UI-Kit Integration**:
   - `apps/admin-portal/package.json` currently omits `@pegasusx/types` and `@pegasusx/ui-kit` (Observation 1.5).
   - `apps/admin-portal/lib/api.ts` contains duplicate local TypeScript types that belong in `@pegasusx/types` (Observation 1.5).
   - Therefore, adding `@pegasusx/types: "workspace:*"` and `@pegasusx/ui-kit: "workspace:*"` to `apps/admin-portal/package.json`, exporting canonical DTOs from `packages/types`, and adopting `@pegasusx/ui-kit` UI components will enforce system-wide UI/type consistency.

---

## 3. Caveats

- In `apps/admin-portal/lib/__tests__/command-dashboard.test.ts`, tests check regex patterns on source files (`board.toMatch(/gs-u-admin-health/)`, `board.toMatch(/IsRegistered is tenant-register/)`, `ops.toMatch(/COUNT\(\*\)/)`, `accuracy.toMatch(/no invented mape28 line/)`). Any refactoring of `admin-portal` components must preserve these test-contract string and testid markers.
- In `HexagonalControlTowerMap.tsx`, the component is used in `warehouse-portal` and `supplier-portal`. Updating it in `@pegasusx/ui-kit/src/control-tower` immediately harmonizes both portals.

---

## 4. Conclusion & Step-by-Step Implementation Strategy for Worker

### Step 1: Update `HexagonalControlTowerMap.tsx` in `@pegasusx/ui-kit`
- **File**: `pegasusX/packages/ui-kit/src/control-tower/HexagonalControlTowerMap.tsx`
- Replace Mapbox imports with MapLibre (`react-map-gl/maplibre` and `maplibre-gl`).
- Import `mapInitialViewState` and `readCachedAuthSession` from `@pegasusx/api-client` (or accept `pack?: MarketPack | null`).
- Initialize camera using `mapInitialViewState(pack ?? readCachedAuthSession()?.pack, 11)`.
- Use Carto dark style: `https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json`.
- Delete `MAPBOX_ACCESS_TOKEN`, `mapbox-gl/dist/mapbox-gl.css`, and SF coordinates `(-122.4, 37.74)`.

### Step 2: Clean Up Mobile UI Theatre & Update Android Hex Map
- **Android**:
  - `pegasusX/apps/retailer-app-android/app/src/main/java/com/pegasusx/retailer/ui/controltower/HexagonalControlTowerMap.kt`: Convert to MapLibre / Carto with dynamic pack center, or remove stubbed GoogleMap SF coordinates.
  - `pegasusX/apps/retailer-app-android/app/src/main/java/com/pegasusx/retailer/ui/controltower/LiveEKGNetworkGraph.kt`: Ensure navigation routes to the truthful `ControlTowerScreen.kt` pulse grid.
- **iOS**:
  - `pegasusX/apps/retailer-app-ios/retailerapp/retailerapp/Screens/ControlTower/HexagonalControlTowerMap.swift`: Remove "wired later" stub.
  - `pegasusX/apps/retailer-app-ios/retailerapp/retailerapp/Screens/ControlTower/LiveEKGNetworkGraph.swift`: Remove mock hardcoded nodes. Ensure `ControlTowerView.swift` remains the canonical pulse destination.

### Step 3: Migrate `admin-portal` to `@pegasusx/types` and `@pegasusx/ui-kit`
- **File**: `pegasusX/apps/admin-portal/package.json`
  - Add `"@pegasusx/types": "workspace:*"` and `"@pegasusx/ui-kit": "workspace:*"` to `dependencies`.
- **File**: `pegasusX/packages/types/index.ts`
  - Export canonical DTO types: `Tenant`, `FlagOverride`, `FlagEval`, `AccuracyRow`, `AuditRow`, `MatchQueueItem`, `BillingInvoice`, `BillingFeeSchedule`.
- **File**: `pegasusX/apps/admin-portal/lib/api.ts`
  - Replace local interface declarations with imports from `@pegasusx/types`.
- **File**: `pegasusX/apps/admin-portal/app/globals.css` & components
  - Import `@pegasusx/ui-kit/styles/portal-ui.css` and `@pegasusx/ui-kit/styles/desktop-foundation.css`.
  - Adopt `@pegasusx/ui-kit` primitives (`PortalSection`, `PortalField`, `PortalInput`, `PortalSelect`, `HealthStrip`, `StatusBadge`/`StatusStack`).

---

## 5. Verification Method

1. **Verify UI Kit & Admin Portal Typechecks and Tests**:
   ```bash
   pnpm --filter @pegasusx/ui-kit typecheck
   pnpm --filter @pegasusx/admin-portal typecheck
   pnpm --filter @pegasusx/admin-portal test
   ```

2. **Verify Elimination of Mapbox Fallback Token and Hardcoded SF Coordinates**:
   ```bash
   grep -rn "pk.eyJ1IjoiZGVmYXVsdC" pegasusX/
   grep -rn --include="*.tsx" --include="*.ts" --include="*.kt" --include="*.swift" "37.74" pegasusX/
   grep -rn --include="*.tsx" --include="*.ts" --include="*.kt" --include="*.swift" -- "-122.4" pegasusX/
   ```
   *Expected result*: No matches in map components or control tower files.

3. **Verify Elimination of "wired later" Theatre**:
   ```bash
   grep -rn --include="*.swift" --include="*.kt" "wired later" pegasusX/apps/
   ```
   *Expected result*: Zero matches.
