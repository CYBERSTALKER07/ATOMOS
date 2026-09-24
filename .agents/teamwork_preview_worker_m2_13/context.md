# Worker M2 Context: Frontend Shared Monorepo Package Consolidation

Working Directory: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m2_13`
Target Workspace Directory: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`
Authoritative Requirements: `/Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md`
Explorer Findings: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_13_2/handoff.md`
Project Plan: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_13/PROJECT.md`

File Write Ownership (Exclusive):
- `packages/types/` (all files in `src/`, `package.json`, `tsconfig.json`)
- `contracts/` (`index.ts`, `types.ts`, `regional_types.ts`)
- `packages/ui-kit/` (`src/desktop/NavigationRail.tsx`, `src/desktop/DetailDrawer.tsx`, `src/portal/KpiStat.tsx`, `src/index.ts`, `package.json`, `tsconfig.json`)
- `packages/pulse-ui/` (`src/NetworkPulsePanel.tsx`, `src/index.ts`, `package.json`, `tsconfig.json`)
- `apps/warehouse-desktop/package.json` and removal of dead `apps/warehouse-desktop/lib/firebase.ts`
- Clean up duplicate types/imports in `apps/warehouse-desktop`, `apps/supplier-desktop`, `apps/retailer-desktop`, `apps/payloader-tablet` as needed to consume the shared packages cleanly and resolve type errors.

Mission & Tasks (Tasks 4, 5, 6 of modularization-plan.md):
1. **Types Consolidation & Synchronization (Task 4)**:
   - Establish `@pegasusx/types` as the Single Source of Truth (SSOT).
   - Absorb all type definitions from `contracts/types.ts` (3,823 lines) and `contracts/regional_types.ts` into `@pegasusx/types/src/`.
   - Absorb local duplicates like `apps/warehouse-desktop/lib/dispatch-types.ts` and `apps/supplier-desktop/app/(portal)/catalog/components/types.ts` into `@pegasusx/types`.
   - Update `contracts/index.ts` to re-export everything from `@pegasusx/types` for backward compatibility (`export * from '@pegasusx/types'`).
   - Fix `packages/types/` file placement (move or re-export `forecast-confidence.ts` cleanly in `src/` without strange imports).
   - Ensure `@pegasusx/types` has valid `"build": "tsc --noEmit"` and `"typecheck"` scripts.
2. **Extract Shared Control Tower Primitives (Task 5)**:
   - In `@pegasusx/ui-kit`:
     - Extract `NavigationRail` (`src/desktop/NavigationRail.tsx`) unifying the ~90% duplicate code from `WarehouseShell.tsx`, `SupplierShell.tsx`, `RetailerShell.tsx` (collapsible 72px/260px rail, tactical styling, Framer Motion, ⌘K search, tooltips).
     - Extract `DetailDrawer` (`ContextInspectorDrawer`) into `src/desktop/DetailDrawer.tsx` unifying the 100% duplicate code across desktop apps.
     - Consolidate `DensityMetricCard` / `KpiStatCard` and `KpiStatGrid` in `src/portal/KpiStat.tsx`.
     - Export these cleanly in `@pegasusx/ui-kit/src/index.ts`.
     - Ensure `@pegasusx/ui-kit` has `"build": "tsc --noEmit"` and `"typecheck"` scripts.
   - In `@pegasusx/pulse-ui`:
     - Extract `NetworkPulsePanel` into `src/NetworkPulsePanel.tsx`.
     - Export it cleanly from `src/index.ts`.
     - Add `tsconfig.json` and build scripts, cleaning up unused/unimported heavy deps if necessary.
3. **Purge Stale Dependencies & Dead Code (Task 6)**:
   - Remove `"firebase": "^12.19.0"` from `apps/warehouse-desktop/package.json` (and `supplier-desktop`, `retailer-desktop` if present).
   - Delete dead `apps/warehouse-desktop/lib/firebase.ts` (and verify 0 remaining firebase imports in warehouse-desktop).
   - Fix any minor type errors identified by Explorer 2 (e.g., `RetailerAIPredictionsResponse` export, `HistorySeriesSource` literal type inference in supplier-desktop, JSX syntax error in `apps/payloader-tablet/src/screens/LoadLedgerScreen.tsx:255`).
4. **Verification**:
   - `pnpm --filter @pegasusx/types build` -> passes cleanly with 0 type errors.
   - `pnpm --filter @pegasusx/pulse-ui build` (or typecheck) -> passes cleanly.
   - `pnpm --filter @pegasusx/ui-kit build` (or typecheck) -> passes cleanly.
   - `pnpm --filter @pegasusx/warehouse-desktop build` -> compiles cleanly.
   - `grep -rn "firebase" apps/warehouse-desktop/package.json` -> 0 matches.
   - `pnpm test` (or Vitest suite) -> 100% pass.
5. Write comprehensive handoff report to `handoff.md` and report completion via `send_message`.
