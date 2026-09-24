# Frontend Shared Monorepo Package Consolidation Survey Report

**Explorer**: Survey Explorer 2 (Frontend Shared Monorepo Explorer)  
**Target Codebase**: `pegasus.x` (Sovereign Core)  
**Requirement Focus**: R2 (Frontend Shared Monorepo Package Consolidation) & Modularization Plan Tasks 4, 5, 6  
**Timestamp**: 2026-09-24T18:22:00+05:00  

---

## 1. Observation

### 1.1 Type System & Contract Drift (`contracts/` vs `packages/types/`)
- **File Inventories & Line Counts**:
  - `contracts/types.ts`: 3,823 lines, 83,693 bytes. Contains 369 exported types/interfaces, including AST-reflected Go structs from `backend/internal/api/` and `backend/internal/models/` (e.g. `AccrueRebateRequest`, `ManifestSealRequest`, `SmartSafeDropRequest`, `ADMCashDropWebhook`, `ConditionContract`, `PickWave`, `PickTask`, `PayrollBatch`).
  - `contracts/regional_types.ts`: 196 lines, 5,022 bytes. Contains 19 exported types covering Uzbekistan nationwide logistics (SOATO regions, mountain pass constraints, feeder shuttle manifests, offline CRDT mutations, and SoftPOS).
  - `packages/types/`: 20 modular files under `src/` (e.g. `admin.ts`, `auto-order.ts`, `compliance.ts`, `warehouse.ts`, `supplier.ts`, `retailer.ts`), with 819 total exports.
  - Cross-analysis revealed that `contracts/` and `packages/types/` overlap on **only 13 types**. **375 types in `contracts/` are completely absent from `packages/types/`**.
- **Ad-Hoc Duplicate Types in Apps**:
  - `apps/warehouse-desktop/lib/dispatch-types.ts` (111 lines) defines local copies of `AvailableDriver`, `RouteStop`, `DispatchRoute`, `DispatchPlanPreview`, `WarehouseFreezeSnapshot`, `OrderReassignRequest`, `ManualDispatchRequest`, and `DispatchCommitRequest` because `@pegasusx/types` lacked them.
  - `apps/supplier-desktop/app/(portal)/catalog/components/types.ts` (117 lines) defines local copies of `ProductPackagingUnit`, `CategoryAttributeField`, `PackagingTemplate`, and `CategoryAttributeSchema`.
- **Architectural Oddities & Broken Re-exports**:
  - `packages/types/forecast-confidence.ts` (106 lines) was placed in the root of `packages/types/` (outside `src/`). It is not exported from `packages/types/index.ts`. Instead, it was bizarrely re-exported through `packages/types/src/fleet.ts` line 76:
    ```ts
    export * from "../forecast-confidence";
    ```
  - In `apps/retailer-desktop/lib/types.ts`, `RetailerAIPredictionsResponse` (which is defined in `packages/types/src/event-payloads.ts:405`) was omitted from re-exports, causing:
    ```
    app/(portal)/dashboard/page.tsx:21:24 - error TS2724: '"../../../lib/types"' has no exported member named 'RetailerAIPredictionsResponse'. Did you mean 'RetailerAIPrediction'?
    ```
  - In `apps/supplier-desktop/lib/__tests__/visualization.test.ts:34`, `const series = { points: [1, 4, 2], source: "live", available: true };` fails typecheck because `source` is inferred as `string` instead of `HistorySeriesSource` union (`"live" | "empty" | "unavailable"`).
  - In `apps/payloader-tablet/src/screens/LoadLedgerScreen.tsx:255`, a syntax error (`error TS1005: ')' expected`) breaks type compilation due to an unclosed ternary operator.
  - `packages/api-client` exists as a 129 KB duplicate of `packages/api-core` and is excluded in `pnpm-workspace.yaml` line 4 (`!packages/api-client`).

### 1.2 Inspection of Shared UI Packages (`packages/pulse-ui/` and `packages/ui-kit/`)
- **`packages/pulse-ui/`**:
  - Directory contents: `package.json`, `index.tsx` (104 lines), and `src/` (`index.ts`, `PulseTimeline.tsx`, `usePulse.ts`).
  - `package.json` specifies `"main": "./src/index.ts"` and `"types": "./src/index.ts"`.
  - It declares heavy dependencies that are **never imported in any file in the package**:
    `d3`, `h3-js`, `maplibre-gl`, `react-map-gl`, `@types/d3`, `@types/maplibre-gl`, `@types/react-map-gl`.
  - It contains **zero build or typecheck scripts** (`scripts: {}`) and **no tsconfig.json**.
- **`packages/ui-kit/`**:
  - Exports: `.`, `./portal`, `./auth`, `./pack`, `./control-tower`, `./desktop`, `./tailwind.config`, plus CSS stylesheets.
  - `src/control-tower/` contains **only** `GlassmorphismPanel.tsx` (615 bytes). The primary control tower primitives (Navigation Rail, Density Metric Cards, Context Inspector Drawer) are completely missing.
  - `src/desktop/` contains `DesktopOfflineTray.tsx`, `EnterpriseOnboardingChecklist.tsx`, `PageSkeletons.tsx`, `VirtualScrollList.tsx`. `DetailDrawer` and `NavigationRail` are completely missing.
  - `src/portal/` contains `KpiStat.tsx` (56 lines) with separate props from desktop apps, `PageChrome.tsx`, `StatusStack.tsx`, and `PortalPrimitives.tsx`.
  - `package.json` has `"typecheck": "tsc --noEmit"` but lacks `"build"`.

### 1.3 Inspection of `apps/warehouse-desktop` and Client Apps
- **Triplicated Layout Primitives**:
  1. **Navigation Rail**:
     - `WarehouseShell.tsx` (285 lines in `apps/warehouse-desktop/components/`)
     - `SupplierShell.tsx` (265 lines in `apps/supplier-desktop/components/`)
     - `RetailerShell.tsx` (265 lines in `apps/retailer-desktop/components/`)
     - All 3 implement identical 72px / 260px collapsible Framer Motion `<motion.aside>` navigation rails with brand avatar badges ("W", "S", "R"), system label, role title, `⌘\` collapse toggle, `⌘K` search bar trigger, navigation section maps, `text-[10px] font-mono font-bold uppercase tracking-wider` micro-labels, and theme/logout footer.
  2. **Context Inspector Drawer**:
     - `apps/warehouse-desktop/components/ui/DetailDrawer.tsx` (77 lines)
     - `apps/supplier-desktop/components/ui/DetailDrawer.tsx` (77 lines)
     - `apps/retailer-desktop/components/ui/DetailDrawer.tsx` (77 lines)
     - All 3 files are 100% identical verbatim copies (fixed right drawer, `w-[420px]`, `bg-black/40 backdrop-blur-xs`, scrollable body, sticky footer actions).
  3. **Density Metric Cards**:
     - `KpiStatCard.tsx` and `KpiStatGrid` are duplicated in all 3 apps (`apps/warehouse-desktop/components/KpiStatCard.tsx`, `apps/supplier-desktop/components/KpiStatCard.tsx`, `apps/retailer-desktop/components/KpiStatCard.tsx`).
     - Features: tabular numerals (`font-mono tabular-nums`), embedded SVG sparklines via `guardHistorySeries(spark)`, alert flags, and category bays.
  4. **Other Duplicated Panels**:
     - `NetworkPulsePanel.tsx` is duplicated across all 3 desktop apps (each wrapping `PulseTimeline` from `@pegasusx/pulse-ui`).
     - `ClientPolicyBanner.tsx` (2,405 bytes) is duplicated verbatim across all 3 desktop apps.
     - `globals.css` (44.5 KB) in all 3 desktop apps is a copy of `packages/ui-kit/styles/desktop-foundation.css`.
- **Firebase Dependency Audit**:
  - `apps/warehouse-desktop/package.json` line 31: `"firebase": "^12.19.0"`.
  - `apps/retailer-desktop/package.json` line 26: `"firebase": "^12.19.0"`.
  - `apps/supplier-desktop/package.json` line 30: `"firebase": "^12.19.0"`.
  - In `apps/warehouse-desktop/lib/firebase.ts`, the file contains legacy OTP stubs that are never imported anywhere in `apps/warehouse-desktop`.
  - Actual production authentication across all desktop apps uses `lib/auth.ts`: HTTP cookies (`pegasus_warehouse_jwt`, `pegasus_warehouse_refresh`), `@pegasusx/api-core` session pinning and reconciliation, and Tauri bridge IPC (`storeToken`, `clearStoredToken`).

### 1.4 Frontend Build Verification Status
- Running `pnpm --filter @pegasusx/warehouse-desktop build` (Next.js 15.5.25 production build):
  ```
  ✓ Compiled successfully in 9.3s
  ✓ Linting and checking validity of types
  ✓ Collecting page data
  ✓ Generating static pages (52/52)
  ✓ Collecting build traces
  ✓ Finalizing page optimization
  Exit status: 0
  ```
- Running `pnpm --filter @pegasusx/warehouse-desktop check-types`: **Exits with code 0**.
- Running `pnpm --filter @pegasusx/types check-types`: **Exits with code 0**.
- Running `pnpm --filter @pegasusx/ui-kit typecheck`: **Exits with code 0**.
- Running `pnpm --filter @pegasusx/ui-charts typecheck`: **Exits with code 0**.
- Running `pnpm --filter @pegasusx/ui-maps typecheck`: **Exits with code 0**.
- Running `pnpm --filter @pegasusx/telegram-miniapp check-types`: **Exits with code 0**.
- Running `pnpm run test` (Vitest across monorepo): **Exits with code 0** (10/10 test files in warehouse-desktop, 16/16 in retailer-desktop, 17/17 in supplier-desktop; 152/152 total tests passing).

---

## 2. Logic Chain

1. **Premise 1**: `@pegasusx/types` is the official TypeScript type package linked via `workspace:*` across all desktop, web, and mobile packages. `contracts/types.ts` and `contracts/regional_types.ts` are orphan files that are not imported by any package, yet they contain 375 essential Go backend DTOs.
2. **Premise 2**: Because `@pegasusx/types` lacked these DTOs, application engineers were forced to create duplicate local type definitions (`apps/warehouse-desktop/lib/dispatch-types.ts`, `apps/supplier-desktop/app/(portal)/catalog/components/types.ts`), causing severe contract drift and type divergence between backend and frontend.
3. **Inference 1**: `@pegasusx/types` must be established as the definitive Single Source of Truth (SSOT). All 375 types from `contracts/types.ts` and `contracts/regional_types.ts` must be absorbed into `@pegasusx/types` (specifically under `src/dispatch.ts`, `src/regional.ts`, and `src/contracts.ts`). `contracts/types.ts` and `contracts/regional_types.ts` should then re-export `@pegasusx/types` to guarantee backward compatibility with zero duplication.
4. **Premise 3**: `apps/warehouse-desktop`, `apps/supplier-desktop`, and `apps/retailer-desktop` contain identical implementations of `DetailDrawer.tsx` (Context Inspector Drawer), `KpiStatCard.tsx` (Density Metric Cards), and `Shell.tsx` (Navigation Rail), violating DRY and complicating design token updates.
5. **Inference 2**: These three layout primitives must be extracted into `@pegasusx/ui-kit` (under `src/desktop/` and `src/control-tower/`), where they can be centrally maintained against the V.O.I.D Tactical Control Tower design system tokens.
6. **Premise 4**: `firebase` is listed as a production dependency in `apps/warehouse-desktop/package.json` (as well as `retailer-desktop` and `supplier-desktop`), but grep search proves 0 imports in application code. Production authentication is handled purely by JWT cookies and Tauri IPC.
7. **Inference 3**: `firebase` is completely dead weight and can be safely purged from `package.json`, along with the unused `lib/firebase.ts` stub, eliminating unnecessary bundle overhead and security vulnerability vectors.
8. **Premise 5**: `@pegasusx/types` and `@pegasusx/ui-kit` have inconsistent script names (`check-types` vs `typecheck`), while `@pegasusx/pulse-ui` lacks scripts and tsconfig entirely.
9. **Inference 4**: Adding unified `"build": "tsc --noEmit"` and `"check-types": "tsc --noEmit"` across all shared packages will ensure Turborepo pipeline consistency and clean CI execution.

---

## 3. Caveats

1. **Dead twin directory `packages/api-client`**: `packages/api-client` is an unreferenced duplicate of `packages/api-core`. It is excluded in `pnpm-workspace.yaml`, but should eventually be deleted from disk to prevent confusion.
2. **`apps/payloader-tablet` syntax error**: The syntax error in `apps/payloader-tablet/src/screens/LoadLedgerScreen.tsx:255` is an isolated JSX ternary parenthesis mismatch in an Expo mobile app and does not block desktop builds, but should be fixed for complete monorepo clean-build status.
3. **`apps/supplier-desktop` stale `.next` cache**: Next.js cached type outputs in `.next/types/` can report stale errors if pages were moved or deleted (e.g. `factories/page.js`). Running `rm -rf .next` before `next build` resolves this.

---

## 4. Conclusion

Requirement R2 is thoroughly scoped and ready for implementation. The frontend codebase is stable (warehouse-desktop builds 52 static pages cleanly, and all 152 Vitest unit tests pass), but suffers from type drift, layout triplication, and a stale `firebase` dependency.

### Concrete Consolidation Plan

#### Phase 1: Type System & Contract Unification (Task 4)
1. **Absorb Regional Contracts**:
   - Move `contracts/regional_types.ts` into `packages/types/src/regional.ts`.
   - Export `* from "./src/regional"` in `packages/types/index.ts`.
2. **Absorb Smart Dispatch & VRP Contracts**:
   - Create `packages/types/src/dispatch.ts` containing `DispatchPlanPreview`, `DispatchRoute`, `RouteStop`, `AvailableDriver`, `WarehouseFreezeSnapshot`, `OrderReassignRequest`, `ManualDispatchRequest`, `DispatchCommitRequest`.
   - Export `* from "./src/dispatch"` in `packages/types/index.ts`.
   - Replace local `apps/warehouse-desktop/lib/dispatch-types.ts` with re-exports from `@pegasusx/types`.
3. **Absorb AST-Reflected Enterprise DTOs**:
   - Create `packages/types/src/contracts.ts` containing the AST-reflected Go structs from `contracts/types.ts` (e.g. `AccrueRebateRequest`, `ManifestSealRequest`, `SmartSafeDropRequest`, `ADMCashDropWebhook`, etc.).
   - Export `* from "./src/contracts"` in `packages/types/index.ts`.
4. **Relocate & Fix `forecast-confidence.ts`**:
   - Move `packages/types/forecast-confidence.ts` into `packages/types/src/forecast-confidence.ts`.
   - Export `* from "./src/forecast-confidence"` in `packages/types/index.ts`.
   - Remove the `export * from "../forecast-confidence"` workaround from `packages/types/src/fleet.ts:76`.
5. **Synchronize `contracts/`**:
   - Replace `contracts/types.ts` body with: `export * from "../packages/types/index";`
   - Replace `contracts/regional_types.ts` body with: `export * from "../packages/types/src/regional";`
6. **Fix Client Type Re-exports**:
   - Add `RetailerAIPredictionsResponse` to `apps/retailer-desktop/lib/types.ts`.
   - In `packages/types/package.json`, add `"build": "tsc --noEmit"` and `"check-types": "tsc --noEmit"`.

#### Phase 2: Shared Component Extraction (Task 5)
1. **In `@pegasusx/ui-kit`**:
   - **`ContextInspectorDrawer` / `DetailDrawer`**:
     - Create `packages/ui-kit/src/desktop/DetailDrawer.tsx` (and alias `ContextInspectorDrawer`).
     - Export from `packages/ui-kit/desktop` and `packages/ui-kit`.
     - Update all 3 desktop apps to import `DetailDrawer` from `@pegasusx/ui-kit`.
   - **`NavigationRail`**:
     - Create `packages/ui-kit/src/desktop/NavigationRail.tsx` with generic `NavSection[]`, brand badge, search callback, collapse state, and footer actions.
     - Refactor `WarehouseShell.tsx`, `SupplierShell.tsx`, and `RetailerShell.tsx` to use `NavigationRail`.
   - **`DensityMetricCard` / `KpiStatCard`**:
     - Unify card layout in `packages/ui-kit/src/control-tower/DensityMetricCard.tsx` (re-exported as `KpiStatCard` and `KpiStatGrid`).
     - Standardize tabular numerals (`font-mono tabular-nums`), embedded SVG sparklines (`guardHistorySeries`), and bay/alert flags.
2. **In `@pegasusx/pulse-ui`**:
   - Add `tsconfig.json` extending `../../tsconfig.base.json`.
   - Add scripts `"typecheck": "tsc --noEmit"` and `"build": "tsc --noEmit"`.
   - Purge unused map dependencies (`maplibre-gl`, `react-map-gl`, `@types/maplibre-gl`, `@types/react-map-gl`).
   - Extract `NetworkPulsePanel` into `@pegasusx/pulse-ui` alongside `PulseTimeline`.

#### Phase 3: Purging `firebase` & Token Cleanup (Task 6)
1. Remove `"firebase": "^12.19.0"` from `apps/warehouse-desktop/package.json` (as well as `retailer-desktop` and `supplier-desktop`).
2. Remove `apps/warehouse-desktop/lib/firebase.ts` (0 imports across app).
3. Run `pnpm install` to update `pnpm-lock.yaml`.
4. Verify `grep -rn "firebase" apps/warehouse-desktop/package.json` returns 0 matches.

---

## 5. Verification Method

To independently verify the findings and test the future consolidation:

1. **Verify Contract Drift & Missing Types**:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x
   python3 -c "
   import re, glob
   def get_exports(f):
       return set(re.findall(r'export\s+(?:type|interface|enum|const)\s+([A-Za-z0-9_]+)', open(f).read()))
   c = get_exports('contracts/types.ts') | get_exports('contracts/regional_types.ts')
   p = set().union(*(get_exports(f) for f in glob.glob('packages/types/**/*.ts') if 'node_modules' not in f))
   print(f'Contracts: {len(c)}, Packages/types: {len(p)}, Missing in packages/types: {len(c - p)}')
   "
   ```
   *Expected*: Shows 375 types in contracts that are missing in packages/types.

2. **Verify Duplicated UI Primitives**:
   ```bash
   diff -u apps/warehouse-desktop/components/ui/DetailDrawer.tsx apps/supplier-desktop/components/ui/DetailDrawer.tsx
   diff -u apps/warehouse-desktop/components/ui/DetailDrawer.tsx apps/retailer-desktop/components/ui/DetailDrawer.tsx
   ```
   *Expected*: Zero diff (files are identical).

3. **Verify Firebase Stale Presence & Zero Usages**:
   ```bash
   grep -rn "firebase" apps/warehouse-desktop/package.json
   grep -rnI --exclude-dir={node_modules,.next} "from.*firebase" apps/warehouse-desktop/
   ```
   *Expected*: Line 31 in `package.json`, only `lib/firebase.ts` imports it, and 0 pages/components import `lib/firebase.ts`.

4. **Verify Frontend Builds & Typechecks**:
   ```bash
   pnpm --filter @pegasusx/types run check-types
   pnpm --filter @pegasusx/ui-kit run typecheck
   pnpm --filter @pegasusx/warehouse-desktop run check-types
   pnpm --filter @pegasusx/warehouse-desktop run build
   pnpm run test
   ```
   *Expected*: All commands exit with code 0; warehouse-desktop builds all 52 static routes.
