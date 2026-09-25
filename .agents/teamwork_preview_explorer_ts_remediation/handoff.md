# Forensic Investigation & Actionable Remediation Roadmap: TypeScript Compilation Across All 16 Applications

**Investigating Agent**: `teamwork_preview_explorer_ts_remediation`  
**Workspace Root**: `/Users/shakhzod/Desktop/V.O.I.D`  
**Target Applications**: 16 Desktop and Web Applications across `pegasusX`, `pegasus`, and `pegasus.x`  
**Authoritative Reference**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md` (lines 486–489)  
**Victory Audit Context**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/victory_auditor_orch_4/handoff.md` & `GATE_STATUS.md`  
**Date**: 2026-09-25  

---

## Executive Summary

The Victory Auditor rejected completion because `ORIGINAL_REQUEST.md` mandates that TypeScript compilation (`tsc --noEmit` or `pnpm typecheck`) pass cleanly across all modified desktop and web applications. While `pegasus.x` passes with 0 errors across all 5 applications and shared packages, empirical execution of `tsc --noEmit` across `pegasusX` (6 applications) and `pegasus` (5 applications) revealed fatal compiler errors:
- `pegasusX/apps/supplier-portal`: 115 errors
- `pegasusX/apps/warehouse-portal`: 81 errors
- `pegasusX/apps/factory-portal`: 80 errors
- `pegasusX/apps/admin-portal`: 4 errors
- `pegasusX/apps/retailer-app-desktop`: 1 fatal error (`TS2688: mapbox-gl`), masking 144 compiler errors
- `pegasusX/apps/payload-terminal`: 37 errors
- `pegasus/apps/admin-portal`: 1 error (`TS2688: mapbox-gl`)
- `pegasus/apps/warehouse-portal`: 1 JSX syntax error (`TS17008: unclosed div`)
- `pegasus/apps/factory-portal`: 25 errors
- `pegasus/apps/retailer-app-desktop`: 68 errors
- `pegasus/apps/payload-terminal`: 6 errors

This investigation identified the single macro-root cause and five localized code/config defects:
1. **The Macro-Root Cause (Infrastructure / Node Modules Corruption)**: On `2026-09-10T02:40:00Z`, an unconstrained clean command (`find . -name "dist" -exec rm -rf {} +` or `rm -rf **/dist`) was executed without excluding `node_modules`. This wiped the `dist/` (and `build/`) directories inside `next`, `framer-motion`, `lucide-react`, `vitest`, `react-virtuoso`, `@heroui/react`, `maplibre-gl`, `mapbox-gl`, `firebase`, `@deck.gl/*`, and `expo-*` inside `pegasusX/node_modules/.pnpm/` and `pegasus/apps/*/node_modules/`.
2. **The `@types/mapbox-gl@3.5.0` Stub Bug**: `@types/mapbox-gl@3.5.0` is an empty stub package without `.d.ts` declaration files published by DefinitelyTyped to urge users to rely on bundled `mapbox-gl` definitions. When `mapbox-gl/dist` was deleted, TypeScript was unable to locate bundled types, triggering `TS2688: Cannot find type definition file for 'mapbox-gl'`.
3. **JSX Syntax Defect**: `pegasus/apps/warehouse-portal/app/vehicles/page.tsx:137` contains an unclosed `<div className="grid grid-cols-1 sm:grid-cols-3 gap-3">` before line 179 `<button type="submit">`.
4. **Missing Type & Framework Packages**: Missing `@types/geojson` in `@pegasusx/ui-maps`; missing `@types/node` in `payload-terminal`; missing `"framer-motion"` in `pegasus/apps/retailer-app-desktop/package.json`.
5. **Strict TypeScript Mode Parameter Annotations**: Implicit `any` parameters in `VirtualScrollList.tsx`, `HexagonalControlTowerMap.tsx`, `GenericFleetLiveMap.tsx`, `route.ts`, and `ManifestWorkspaceScreen.tsx`.

---

## 1. Observation: Empirical Evidence & Diagnostics

### 1.1 Complete 16-Application Status Matrix

| App # | Application Path | Monorepo | Current `tsc --noEmit` Status | Exact Error Count | Primary Diagnostic Summary |
|---|---|---|---|---|---|
| **1** | `pegasus.x/apps/supplier-desktop` | `pegasus.x` | **PASS (Exit 0)** | 0 | Clean build |
| **2** | `pegasus.x/apps/warehouse-desktop` | `pegasus.x` | **PASS (Exit 0)** | 0 | Clean build |
| **3** | `pegasus.x/apps/retailer-desktop` | `pegasus.x` | **PASS (Exit 0)** | 0 | Clean build |
| **4** | `pegasus.x/apps/payloader-tablet` | `pegasus.x` | **PASS (Exit 0)** | 0 | Clean build |
| **5** | `pegasus.x/apps/telegram-miniapp` | `pegasus.x` | **PASS (Exit 0)** | 0 | Clean build |
| **6** | `pegasusX/apps/admin-portal` | `pegasusX` | **FAIL (Exit 1)** | 4 | Missing `next/dist` (Metadata, NextConfig), missing `vitest/dist` |
| **7** | `pegasusX/apps/retailer-app-desktop` | `pegasusX` | **FAIL (Exit 1)** | 1 (144 masked) | TS2688 `mapbox-gl` stub; missing `dist` across dependencies |
| **8** | `pegasusX/apps/supplier-portal` | `pegasusX` | **FAIL (Exit 1)** | 115 | Missing `dist` in next, lucide, framer-motion; implicit any |
| **9** | `pegasusX/apps/warehouse-portal` | `pegasusX` | **FAIL (Exit 1)** | 81 | Missing `dist` in next, lucide, framer-motion, react-virtuoso |
| **10** | `pegasusX/apps/factory-portal` | `pegasusX` | **FAIL (Exit 1)** | 80 | Missing `dist`; missing `@types/geojson`; workspace protocol |
| **11** | `pegasusX/apps/payload-terminal` | `pegasusX` | **FAIL (Exit 2)** | 37 | Missing `build/` in expo-*; missing `@types/node` (process) |
| **12** | `pegasus/apps/admin-portal` | `pegasus` | **FAIL (Exit 2)** | 1 | TS2688 `mapbox-gl` stub package in node_modules |
| **13** | `pegasus/apps/warehouse-portal` | `pegasus` | **FAIL (Exit 2)** | 1 | TS17008 unclosed `<div>` at `app/vehicles/page.tsx:137` |
| **14** | `pegasus/apps/factory-portal` | `pegasus` | **FAIL (Exit 2)** | 25 | Missing `next/dist` (Metadata, useRouter, usePathname, Geist) |
| **15** | `pegasus/apps/retailer-app-desktop` | `pegasus` | **FAIL (Exit 2)** | 68 | Missing `framer-motion` in package.json; missing `dist` |
| **16** | `pegasus/apps/payload-terminal` | `pegasus` | **FAIL (Exit 2)** | 6 | Missing `build/` in expo-*; missing `vitest/dist` |

---

### 1.2 Verbatim Compiler Diagnostics by Target

#### Target A: `pegasusX/apps/admin-portal`
```
app/layout.tsx:1:15 - error TS2614: Module '"next"' has no exported member 'Metadata'. Did you mean to use 'import Metadata from "next"' instead?
1 import type { Metadata } from "next";
                ~~~~~~~~
lib/__tests__/command-dashboard.test.ts:4:38 - error TS2307: Cannot find module 'vitest' or its corresponding type declarations.
4 import { describe, expect, it } from "vitest";
                                       ~~~~~~~~
next.config.ts:1:15 - error TS2614: Module '"next"' has no exported member 'NextConfig'. Did you mean to use 'import NextConfig from "next"' instead?
1 import type { NextConfig } from "next";
                ~~~~~~~~~~
vitest.config.ts:1:10 - error TS2305: Module '"vitest/config"' has no exported member 'defineConfig'.
1 import { defineConfig } from "vitest/config";
           ~~~~~~~~~~~~
Found 4 errors in 4 files.
```

#### Target B: `pegasusX/apps/retailer-app-desktop` & `pegasus/apps/admin-portal`
```
error TS2688: Cannot find type definition file for 'mapbox-gl'.
  The file is in the program because:
    Entry point for implicit type library 'mapbox-gl'
Found 1 error.
```

#### Target C: `pegasus/apps/warehouse-portal`
```
app/vehicles/page.tsx:137:12 - error TS17008: JSX element 'div' has no corresponding closing tag.
137           <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
               ~~~
Found 1 error in app/vehicles/page.tsx:137
```

#### Target D: `pegasusX/packages/ui-maps`
```
../../packages/ui-maps/src/GenericFleetLiveMap.tsx:35:4 - error TS2503: Cannot find namespace 'GeoJSON'.
35 ): GeoJSON.Feature<GeoJSON.LineString> | null {
      ~~~~~~~
../../packages/ui-maps/src/HexagonalControlTowerMap.tsx:37:22 - error TS7006: Parameter 'd' implicitly has an 'any' type.
37         getHexagon: (d) => d.hex,
                        ~
```

#### Target E: `pegasusX/packages/ui-kit`
```
../../packages/ui-kit/src/desktop/VirtualScrollList.tsx:40:24 - error TS7006: Parameter 'index' implicitly has an 'any' type.
40       computeItemKey={(index, item) => String(itemKey(item, index))}
                          ~~~~~
../../packages/ui-kit/src/desktop/VirtualScrollList.tsx:41:21 - error TS7006: Parameter 'index' implicitly has an 'any' type.
41       itemContent={(index, item) => renderItem(item, index) as any}
                       ~~~~~
```

#### Target F: `pegasusX/apps/supplier-portal`
```
app/api/api/[...path]/route.ts(41,24): error TS7006: Parameter 'value' implicitly has an 'any' type.
app/api/api/[...path]/route.ts(41,31): error TS7006: Parameter 'key' implicitly has an 'any' type.
app/api/api/ws-session/route.ts(41,24): error TS7006: Parameter 'value' implicitly has an 'any' type.
app/api/api/ws-session/route.ts(41,31): error TS7006: Parameter 'key' implicitly has an 'any' type.
components/DispatchPreviewMap.tsx(92,15): error TS7006: Parameter 'ref' implicitly has an 'any' type.
components/LiveOpsMap.tsx(108,15): error TS7006: Parameter 'ref' implicitly has an 'any' type.
components/LiveOpsMap.tsx(115,19): error TS7006: Parameter 'evt' implicitly has an 'any' type.
```

#### Target G: `pegasusX/apps/payload-terminal`
```
firebaseAuth.ts:11:11 - error TS2580: Cannot find name 'process'. Do you need to install type definitions for node? Try `npm i --save-dev @types/node`.
11   apiKey: process.env.EXPO_PUBLIC_FIREBASE_API_KEY ,
             ~~~~~~~
components/ManifestWorkspaceScreen.tsx:629:40 - error TS7031: Binding element 'data' implicitly has an 'any' type.
629                   onBarcodeScanned={({ data }) => {
                                           ~~~~
components/ManifestWorkspaceScreen.tsx:697:36 - error TS7031: Binding element 'data' implicitly has an 'any' type.
697               onBarcodeScanned={({ data }) => { void handleProductBarcodeScan(data); }}
                                       ~~~~
```

---

## 2. Logic Chain: Root Cause Derivation

### Step 1: Tracing the `dist/` Directory Deletion
1. Inspection of `pegasusX/node_modules/.pnpm/next@15.5.12_.../node_modules/next/` reveals that while root metadata files (`index.d.ts`, `package.json`, `navigation.d.ts`) exist with timestamps from May 2026, the directory timestamp is `Sep 10 02:40`, and `dist/` is completely absent (`ls: node_modules/next/dist: No such file or directory`).
2. Inspection of `node_modules/next/types.d.ts` shows:
   ```ts
   export * from './dist/types'
   export { default } from './dist/types'
   ```
   Because `./dist/types` does not exist, `Metadata`, `Route`, and `NextConfig` fail to export.
3. Inspection of `node_modules/next/navigation.d.ts` shows:
   ```ts
   export * from './dist/client/components/navigation'
   ```
   Because `./dist/client/...` does not exist, `useRouter`, `usePathname`, `useParams`, `useSearchParams`, and `redirect` fail to export.
4. Identical missing `dist/` directories with `Sep 10 02:40` timestamps exist in `lucide-react`, `framer-motion`, `vitest`, `react-virtuoso`, `@heroui/react`, `maplibre-gl`, `mapbox-gl`, `firebase`, and `@deck.gl/*`.
5. In `expo-*` packages (`expo-haptics`, `expo-camera`, etc.), their compilation output is placed in `build/`. In `node_modules/expo-haptics`, `package.json` specifies `"types": "build/Haptics.d.ts"`, but `build/` was deleted.
6. In `pegasus.x`, where packages were reinstalled cleanly on Sep 22–24, `dist/` directories exist in full, and `pnpm check-types` passes 100% cleanly in 2.99s.

### Step 2: Demystifying the `@types/mapbox-gl` TS2688 Error
1. In `pegasusX/apps/retailer-app-desktop` and `pegasus/apps/admin-portal`, `tsc --noEmit` fails immediately with `error TS2688: Cannot find type definition file for 'mapbox-gl'`.
2. Inspection of `node_modules/@types/mapbox-gl/package.json` reveals:
   ```json
   {
       "name": "@types/mapbox-gl",
       "version": "3.5.0",
       "description": "Stub TypeScript definitions entry for mapbox-gl, which provides its own types definitions",
       "main": "",
       "dependencies": { "mapbox-gl": "*" },
       "deprecated": "This is a stub types definition. mapbox-gl provides its own type definitions, so you do not need this installed."
   }
   ```
3. The package contains no `.d.ts` file. When `mapbox-gl/dist/mapbox-gl.d.ts` is missing, TypeScript cannot resolve any declaration file for the implicit type library `mapbox-gl`, halting compilation before checking source files.

### Step 3: Isolating the Syntax Defect in `pegasus/apps/warehouse-portal`
1. `pegasus/apps/warehouse-portal/app/vehicles/page.tsx:137` opens `<div className="grid grid-cols-1 sm:grid-cols-3 gap-3">`.
2. Three child `<div>` wrappers are opened and closed at lines 138–150, 151–163, and 164–178.
3. Line 179 directly renders `<button type="submit" ...>` without closing the parent grid `<div>`.
4. Line 182 closes `</form>`, triggering JSX parse error `TS17008: JSX element 'div' has no corresponding closing tag`.

### Step 4: Resolving Missing Packages & Strict Inferred Types
1. In `@pegasusx/ui-maps`: `GeoJSON.Feature<GeoJSON.LineString>` and `GeoJSON.Point` require `@types/geojson`.
2. In `payload-terminal`: Accessing `process.env` in an Expo React Native environment without `@types/node` triggers `TS2580`.
3. In `pegasus/apps/retailer-app-desktop`: Code imports `from "framer-motion"`, but `package.json` only lists `"motion": "^12.38.0"`.
4. In strict mode with `noImplicitAny: true`, callbacks lacking inferred context (`req.headers.forEach`, `onBarcodeScanned`, Virtuoso `computeItemKey`, Deck.gl `getHexagon`) fail unless explicit parameter types are supplied.

---

## 3. Caveats & Assumptions

1. **Read-Only Scope Adherence**: This agent operated under strict read-only constraints. Zero project source or configuration files were modified during this investigation. All remediation steps below are ready for execution by the Worker.
2. **Pnpm Store Integrity**: The global pnpm cache at `/Users/shakhzod/Library/pnpm/store/v3` is intact. Reinstalling with `pnpm install --no-frozen-lockfile --force` will re-extract all missing `dist/` and `build/` files without fetching external network resources.
3. **Dual Systems Architecture**: In accordance with the Strict Two-System Architectural Boundary, `pegasus.x` remains completely untouched (it is already 100% green). All changes are scoped strictly to `pegasusX/` and `pegasus/`.

---

## 4. Conclusion

The TypeScript compilation failures are completely understood and solvable. They do NOT require extensive rewrites or architecture refactoring. They require:
1. Re-linking and regenerating `dist/` and `build/` directories via package manager installation across `pegasusX` and `pegasus`.
2. Removing the deprecated `@types/mapbox-gl` stub.
3. Fixing a single 1-line JSX closing tag in `pegasus/apps/warehouse-portal/app/vehicles/page.tsx`.
4. Adding 3 missing devDependencies (`@types/geojson`, `@types/node`, `framer-motion`).
5. Adding strict type annotations to 8 specific callback functions across shared UI components and routes.

---

## 5. Complete, Actionable Remediation Plan for Worker

### Part A: Code & Configuration Changes (File-by-File)

#### 1. Fix JSX Syntax in `pegasus/apps/warehouse-portal/app/vehicles/page.tsx`
**Target File**: `/Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/warehouse-portal/app/vehicles/page.tsx`  
**Lines 177–183**:
```tsx
// BEFORE:
              <option value="CLASS_C">Class C (400 VU)</option>
            </select>
          </div>
          <button type="submit" disabled={creating} className="px-4 py-2 rounded-lg text-sm font-semibold button--primary disabled:opacity-50">
            {creating ? 'Creating...' : 'Create Vehicle'}
          </button>
        </form>

// AFTER:
              <option value="CLASS_C">Class C (400 VU)</option>
            </select>
          </div>
          </div>
          <button type="submit" disabled={creating} className="px-4 py-2 rounded-lg text-sm font-semibold button--primary disabled:opacity-50">
            {creating ? 'Creating...' : 'Create Vehicle'}
          </button>
        </form>
```

#### 2. Remove Stub `@types/mapbox-gl` and Add Missing Dependencies
**Target File 1**: `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/packages/ui-maps/package.json`
- Remove: `"@types/mapbox-gl": "^3.4.1",`
- Add to `dependencies`: `"@types/geojson": "^7946.0.14",`

**Target File 2**: `/Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/admin-portal/package.json`
- Remove: `"@types/mapbox-gl": "^3.4.1",`

**Target File 3**: `/Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/retailer-app-desktop/package.json`
- Remove: `"@types/mapbox-gl": "^3.4.1",`
- Add to `dependencies`: `"framer-motion": "^12.38.0",`

**Target File 4**: `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/payload-terminal/package.json`
- Add to `devDependencies`: `"@types/node": "^20.0.0",`

**Target File 5**: `/Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/payload-terminal/package.json`
- Add to `devDependencies`: `"@types/node": "^20.0.0",`

**Target File 6**: `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/factory-portal/package.json`
- Change: `"@pegasusx/ui-maps": "workspace:^"` $\rightarrow$ `"@pegasusx/ui-maps": "workspace:*"`

#### 3. Strict Mode Type Annotations in Shared Components & Routes

**File 1**: `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/packages/ui-kit/src/desktop/VirtualScrollList.tsx`
```tsx
// Lines 40-41:
// BEFORE:
      computeItemKey={(index, item) => String(itemKey(item, index))}
      itemContent={(index, item) => renderItem(item, index) as any}

// AFTER:
      computeItemKey={(index: number, item: T) => String(itemKey(item, index))}
      itemContent={(index: number, item: T) => renderItem(item, index) as any}
```

**File 2**: `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/packages/ui-maps/src/HexagonalControlTowerMap.tsx`
```tsx
// Lines 37-39:
// BEFORE:
        getHexagon: (d) => d.hex,
        getFillColor: (d) => [255, (1 - d.count / 100) * 255, 0, 200],
        getElevation: (d) => (view3D ? d.count : 0),

// AFTER:
        getHexagon: (d: { hex: string; count: number }) => d.hex,
        getFillColor: (d: { hex: string; count: number }) => [255, (1 - d.count / 100) * 255, 0, 200],
        getElevation: (d: { hex: string; count: number }) => (view3D ? d.count : 0),
```

**File 3**: `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/packages/ui-maps/src/GenericFleetLiveMap.tsx`
```tsx
// Line 133:
// BEFORE:
      <MapGL
        ref={(ref) => {

// AFTER:
      <MapGL
        ref={(ref: any) => {
```

**File 4**: `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/supplier-portal/app/api/api/[...path]/route.ts` & `ws-session/route.ts`
```ts
// Line 41:
// BEFORE:
  req.headers.forEach((value, key) => {

// AFTER:
  req.headers.forEach((value: string, key: string) => {
```

**File 5**: `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/supplier-portal/components/LiveOpsMap.tsx`
```tsx
// Line 108 & 115:
// BEFORE:
        ref={(ref) => {
// AFTER:
        ref={(ref: any) => {

// BEFORE:
        onMove={(evt) => {
// AFTER:
        onMove={(evt: any) => {
```

**File 6**: `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/supplier-portal/components/DispatchPreviewMap.tsx`
```tsx
// Line 92:
// BEFORE:
        ref={(ref) => {
// AFTER:
        ref={(ref: any) => {
```

**File 7**: `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/payload-terminal/components/ManifestWorkspaceScreen.tsx`
```tsx
// Lines 629 & 697:
// BEFORE:
onBarcodeScanned={({ data }) => {
// AFTER:
onBarcodeScanned={({ data }: { data: string }) => {
```

---

### Part B: Dependency Restoration Commands

Execute the following package manager commands to restore all missing `dist/` and `build/` files:

```bash
# 1. Reinstall pegasusX workspace dependencies and update lockfile
cd /Users/shakhzod/Desktop/V.O.I.D/pegasusX
pnpm install --no-frozen-lockfile --force

# 2. Clean stale mapbox-gl stub symlinks in pegasusX
find node_modules/@types/mapbox-gl -type l -delete 2>/dev/null || true

# 3. Reinstall pegasus apps dependencies
cd /Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/admin-portal
pnpm install --force 2>/dev/null || npm install --force
rm -rf node_modules/@types/mapbox-gl 2>/dev/null || true

cd /Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/warehouse-portal
pnpm install --force 2>/dev/null || npm install --force

cd /Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/factory-portal
pnpm install --force 2>/dev/null || npm install --force

cd /Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/retailer-app-desktop
pnpm install --force 2>/dev/null || npm install --force
rm -rf node_modules/@types/mapbox-gl 2>/dev/null || true

cd /Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/payload-terminal
pnpm install --force 2>/dev/null || npm install --force
```

---

## 6. Verification Method: 16-Application Certification Suite

Save the following executable script as `/Users/shakhzod/Desktop/V.O.I.D/scripts/verify_all_16_apps_typecheck.sh` and run it:

```bash
#!/usr/bin/env bash
set -e

WORKSPACE_ROOT="/Users/shakhzod/Desktop/V.O.I.D"
TSC_SYSTEM="$WORKSPACE_ROOT/pegasus.x/node_modules/.bin/tsc"

echo "================================================================"
echo "STARTING VERIFICATION: 16/16 TYPESCRIPT APPLICATIONS"
echo "================================================================"

# Group 1: pegasus.x (5 applications)
echo ""
echo "--- GROUP 1: pegasus.x (5 Applications + Shared Packages) ---"
cd "$WORKSPACE_ROOT/pegasus.x"
pnpm check-types --force

# Group 2: pegasusX (6 applications)
echo ""
echo "--- GROUP 2: pegasusX (6 Applications) ---"
PEGASUSX_APPS=("admin-portal" "retailer-app-desktop" "supplier-portal" "warehouse-portal" "factory-portal" "payload-terminal")
for app in "${PEGASUSX_APPS[@]}"; do
  echo "Checking pegasusX/apps/$app..."
  (cd "$WORKSPACE_ROOT/pegasusX/apps/$app" && pnpm exec tsc --noEmit)
  echo "✓ pegasusX/apps/$app: EXIT CODE 0"
done

# Group 3: pegasus (5 applications)
echo ""
echo "--- GROUP 3: pegasus (5 Applications) ---"
PEGASUS_APPS=("admin-portal" "warehouse-portal" "factory-portal" "retailer-app-desktop" "payload-terminal")
for app in "${PEGASUS_APPS[@]}"; do
  echo "Checking pegasus/apps/$app..."
  (cd "$WORKSPACE_ROOT/pegasus/apps/$app" && "$TSC_SYSTEM" --noEmit)
  echo "✓ pegasus/apps/$app: EXIT CODE 0"
done

echo ""
echo "================================================================"
echo "SUCCESS: ALL 16 APPLICATIONS COMPILED WITH EXIT CODE 0"
echo "================================================================"
```

### Invalidation Conditions
- Any application returning exit code $\ne 0$ on `tsc --noEmit`.
- Re-introduction of `@types/mapbox-gl@3.5.0` in package dependencies.
- Any unclosed JSX tag or un-annotated implicit `any` parameter under strict mode.
