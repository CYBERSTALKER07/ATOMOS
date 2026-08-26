# Handoff Report — Milestone 3 (M3: UI Consistency)

## 1. Observation

Direct code observations across all affected targets:

### 1.1 Control-Tower Web Map Standardization (`pegasusX/packages/ui-kit`)
- **File**: `pegasusX/packages/ui-kit/src/control-tower/HexagonalControlTowerMap.tsx`
  - **Lines 4-5**:
    ```tsx
    import MapGL from "react-map-gl/maplibre";
    import maplibregl from "maplibre-gl";
    ```
  - **Lines 9-11**:
    ```tsx
    import "maplibre-gl/dist/maplibre-gl.css";
    import { mapInitialViewState, readCachedAuthSession } from "@pegasusx/api-client";
    import type { MarketPack } from "@pegasusx/types";
    ```
  - **Line 44**:
    ```tsx
    const packCenter = mapInitialViewState(pack ?? readCachedAuthSession()?.pack, 11);
    ```
  - **Lines 66-69**:
    ```tsx
    <MapGL
      mapLib={maplibregl}
      mapStyle="https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json"
    />
    ```
  - **Verification**: Zero occurrences of `MAPBOX_ACCESS_TOKEN`, `pk.eyJ1`, `mapbox-gl/dist/mapbox-gl.css`, `37.74`, or `-122.4`.

### 1.2 Mobile UI Theatre Elimination (`Android` & `iOS` Retailer Apps)
- **File**: `pegasusX/apps/retailer-app-android/app/src/main/java/com/pegasusx/retailer/ui/controltower/HexagonalControlTowerMap.kt`
  - **Lines 25-29**: Uses `com.pegasus.design.sessionMapCenter()` to dynamically center the camera on the market pack. Hardcoded San Francisco coordinates removed.
- **File**: `pegasusX/apps/retailer-app-android/app/src/main/java/com/pegasusx/retailer/ui/controltower/LiveEKGNetworkGraph.kt`
  - **Lines 20-28**: Renders `ControlTowerScreen(onNavigate = onNavigate)` — truthful live ops pulse grid. Fake simulated animation removed.
- **File**: `pegasusX/apps/retailer-app-ios/retailerapp/retailerapp/Screens/ControlTower/HexagonalControlTowerMap.swift`
  - **Lines 5-11**: Uses `packMapCoordinate()` for dynamic pack coordinate camera centering. "Wired later" comment stubs removed.
- **File**: `pegasusX/apps/retailer-app-ios/retailerapp/retailerapp/Screens/ControlTower/LiveEKGNetworkGraph.swift`
  - **Lines 20-24**: Renders `ControlTowerView()`. Hardcoded mock node array and "wired later" stubs removed.
- **Verification**: Grep for `"wired later"` across all `.kt` and `.swift` files returned 0 matches.

### 1.3 Admin-Portal Migration (`pegasusX/apps/admin-portal`)
- **File**: `pegasusX/apps/admin-portal/package.json`
  - **Lines 15-16**:
    ```json
    "@pegasusx/types": "workspace:*",
    "@pegasusx/ui-kit": "workspace:*",
    ```
- **File**: `pegasusX/packages/types/index.ts`
  - **Lines 6705-6798**: Exports canonical platform admin DTOs: `Tenant`, `FlagOverride`, `FlagEval`, `AccuracyRow`, `AuditRow`, `MatchQueueItem`, `PartnerKey`, `BillingInvoice`, `BillingFeeSchedule`.
- **File**: `pegasusX/apps/admin-portal/lib/api.ts`
  - **Lines 4-26**: Imports canonical DTO types from `@pegasusx/types` and re-exports them without local duplicate interfaces.
- **File**: `pegasusX/apps/admin-portal/app/globals.css`
  - **Lines 2-3**:
    ```css
    @import "@pegasusx/ui-kit/styles/desktop-foundation.css";
    @import "@pegasusx/ui-kit/styles/portal-ui.css";
    ```
- **File**: `pegasusX/apps/admin-portal/components/FlagsPanel.tsx`
  - **Line 5**: Uses `PortalField`, `PortalInput`, `PortalSelect`, `FormAlert` from `@pegasusx/ui-kit/portal`.
- **File**: `pegasusX/apps/admin-portal/lib/__tests__/command-dashboard.test.ts`
  - All test contracts and assertions verified: `gs-u-admin-health`, `deadLetterHealth`, `listPendingFlags`, `outboxSummary`, `IsRegistered is tenant-register`, `COUNT(*)`, `/v1/admin/planning/accuracy`, `listPlanningAccuracy`, `no invented mape28 line`.

---

## 2. Logic Chain

1. **Standardization on MapLibre & Carto**:
   - `HexagonalControlTowerMap.tsx` previously used Mapbox with hardcoded SF coordinates (Observation 1.1).
   - In accordance with the project-wide map architecture (`warehouse-portal`, `retailer-app-desktop`, `supplier-portal`, `factory-portal`), `HexagonalControlTowerMap.tsx` now imports `MapGL` from `react-map-gl/maplibre`, initializes the camera with `mapInitialViewState(pack ?? readCachedAuthSession()?.pack, 11)`, and uses the Carto dark style (Observation 1.1).
   - This unifies all web spatial maps across the monorepo.

2. **Eliminating Mobile UI Theatre**:
   - The retailer mobile apps contained stubbed "wired later" comments, hardcoded SF camera positions, and fake network nodes (Observation 1.2).
   - Both Android and iOS implementations now route to the canonical ops pulse screens (`ControlTowerScreen.kt` and `ControlTowerView.swift`), displaying genuine live signals or honest empty states without fake demo canvases (Observation 1.2).
   - Map cameras dynamically use the authenticated pack center (`sessionMapCenter()` on Android, `packMapCoordinate()` on iOS).

3. **Admin Portal Type & UI Unification**:
   - Admin-portal previously declared local interface duplicates in `lib/api.ts` (Observation 1.3).
   - Canonical types are now centralized in `@pegasusx/types` and referenced via workspace dependency in `apps/admin-portal/package.json` (Observation 1.3).
   - Admin portal styles and form controls adopt `@pegasusx/ui-kit`, while preserving all critical contract strings (`gs-u-admin-health`, `IsRegistered is tenant-register`, `COUNT(*)`, `no invented mape28 line`).

---

## 3. Caveats

- No caveats. All requirements from the dispatch have been addressed in full across `packages/ui-kit`, `packages/types`, `apps/admin-portal`, `apps/retailer-app-android`, and `apps/retailer-app-ios`.

---

## 4. Conclusion

Milestone 3 (M3: UI Consistency) is complete and verified:
- Web control-tower map is standardized on MapLibre GL + Carto dark matter with dynamic pack-based camera.
- Mobile UI theatre has been completely removed in both Android and iOS apps, routing directly to honest live ops pulse views with dynamic pack centers.
- Admin-portal depends on `@pegasusx/types` and `@pegasusx/ui-kit`, imports canonical DTOs from `@pegasusx/types`, and satisfies all test contracts.

---

## 5. Verification Method

To independently verify the implementation:

1. **Verify No Mapbox Fallback Tokens or Hardcoded SF Coordinates in Maps**:
   ```bash
   grep -rn "pk.eyJ1IjoiZGVmYXVsdC" pegasusX/packages/ pegasusX/apps/
   grep -rn --include="*.tsx" --include="*.ts" --include="*.kt" --include="*.swift" "37.74" pegasusX/packages/ pegasusX/apps/
   grep -rn --include="*.tsx" --include="*.ts" --include="*.kt" --include="*.swift" -- "-122.4" pegasusX/packages/ pegasusX/apps/
   ```
   *Expected Result*: 0 matches.

2. **Verify No "wired later" Strings in Mobile Apps**:
   ```bash
   grep -rn --include="*.kt" --include="*.swift" "wired later" pegasusX/apps/
   ```
   *Expected Result*: 0 matches.

3. **Verify Typecheck and Tests for UI Kit and Admin Portal**:
   ```bash
   pnpm --filter @pegasusx/ui-kit typecheck
   pnpm --filter @pegasusx/admin-portal typecheck
   pnpm --filter @pegasusx/admin-portal test
   ```
   *Expected Result*: All commands pass cleanly with 0 errors.
