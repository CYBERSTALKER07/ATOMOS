# BRIEFING — 2026-09-24T18:21:45+05:00

## Mission
Survey and formulate an enterprise consolidation strategy for Requirement R2 (Frontend Shared Monorepo Package Consolidation) in pegasus.x: analyzing contracts vs @pegasusx/types, inspecting pulse-ui & ui-kit, analyzing repeated layouts in apps/warehouse-desktop and other clients, checking firebase dependency, testing frontend builds, and producing a concrete consolidation roadmap.

## 🔒 My Identity
- Archetype: explorer
- Roles: Frontend Shared Monorepo Explorer (Survey Explorer 2)
- Working directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_13_2
- Original parent: 5a4e02a9-b43f-4e55-be49-9194ece2feb7
- Milestone: Survey & Exploration for Modularization R2

## 🔒 Key Constraints
- Read-only investigation — do NOT implement changes in pegasus.x during this phase
- Target strictly pegasus.x (Sovereign Core)
- Adhere to V.O.I.D Tactical UI/UX Control Tower Design System and AGENTS.md rules
- Write findings to handoff.md and report to parent via send_message

## Current Parent
- Conversation ID: 5a4e02a9-b43f-4e55-be49-9194ece2feb7
- Updated: 2026-09-24T18:21:45+05:00

## Investigation State
- **Explored paths**:
  - `contracts/types.ts` & `contracts/regional_types.ts`
  - `packages/types/` (index.ts, src/*.ts, forecast-confidence.ts, package.json, tsconfig.json)
  - `packages/pulse-ui/` (src/PulseTimeline.tsx, src/usePulse.ts, package.json)
  - `packages/ui-kit/` (src/control-tower, src/desktop, src/portal, src/pack, styles/desktop-foundation.css, package.json, tailwind.config.ts)
  - `apps/warehouse-desktop` (package.json, components/WarehouseShell.tsx, components/KpiStatCard.tsx, components/ui/DetailDrawer.tsx, components/NetworkPulsePanel.tsx, lib/firebase.ts, lib/auth.ts, lib/types.ts, lib/dispatch-types.ts, app/globals.css)
  - `apps/supplier-desktop` (package.json, components/SupplierShell.tsx, components/KpiStatCard.tsx, components/ui/DetailDrawer.tsx, lib/forecast-confidence.ts, lib/types.ts, lib/__tests__/visualization.test.ts)
  - `apps/retailer-desktop` (package.json, components/RetailerShell.tsx, components/KpiStatCard.tsx, components/ui/DetailDrawer.tsx, lib/types.ts, app/(portal)/dashboard/page.tsx)
  - `apps/payloader-tablet` (src/screens/LoadLedgerScreen.tsx)
  - `apps/telegram-miniapp`
  - `pnpm-workspace.yaml`, `package.json`, `turbo.json`, `tsconfig.base.json`
- **Key findings**:
  1. Contracts vs Types: 388 types in `contracts/` vs 819 in `packages/types/` with only 13 overlapping types. 375 contracts (including Go DTO models like `AccrueRebateRequest`, `ManifestSealRequest`, `RouteFeasibility`, etc.) are missing from `@pegasusx/types`. Apps have declared ad-hoc local duplicates (`apps/warehouse-desktop/lib/dispatch-types.ts`, `apps/supplier-desktop/app/(portal)/catalog/components/types.ts`).
  2. Bizarre coupling: `packages/types/forecast-confidence.ts` is placed outside `src/`, excluded from `index.ts`, and re-exported via `packages/types/src/fleet.ts:76`.
  3. UI Duplication:
     - Navigation Rail is ~90% duplicated across `WarehouseShell.tsx`, `SupplierShell.tsx`, and `RetailerShell.tsx` (~270 lines each).
     - Context Inspector Drawer (`DetailDrawer.tsx`) is 100% identical across all 3 desktop apps (77 lines).
     - Density Metric Cards (`KpiStatCard.tsx` + `KpiStatGrid`) duplicated across all 3 desktop apps, while `ui-kit/portal/KpiStat.tsx` exists separately.
     - `NetworkPulsePanel.tsx` duplicated across all 3 desktop apps.
     - `globals.css` (44.5 KB) in all 3 desktop apps duplicates `packages/ui-kit/styles/desktop-foundation.css`.
  4. Firebase: Stale, unused dependency in `apps/warehouse-desktop/package.json` line 31 (and also in `retailer-desktop` and `supplier-desktop`). `lib/firebase.ts` is dead code with 0 imports; production auth uses JWT cookies, `@pegasusx/api-core`, and Tauri bridge.
  5. Monorepo Builds:
     - `pnpm --filter @pegasusx/warehouse-desktop build` (Next.js 15) passes 100% (52/52 static pages generated).
     - `pnpm run test` passes 100% (all 6 vitest suites pass).
     - `@pegasusx/types` has `check-types` but lacks `build` script.
     - `@pegasusx/pulse-ui` lacks tsconfig and scripts, has unimported deps (`d3`, `h3-js`, `maplibre-gl`).
     - Stale type errors exist in `retailer-desktop` (missing `RetailerAIPredictionsResponse` export), `supplier-desktop` (`HistorySeriesSource` literal type inference in visualization test), and `payloader-tablet` (syntax error TS1005 in `LoadLedgerScreen.tsx:255`).
  6. Duplicate `packages/api-client`: Dead twin of `packages/api-core` excluded in `pnpm-workspace.yaml`.
- **Unexplored areas**: None. All objectives investigated with deep evidence.

## Key Decisions Made
- Formulated 4-part consolidation strategy:
  1. Synchronize contracts into `@pegasusx/types` as SSOT; make `contracts/` re-export `@pegasusx/types`.
  2. Extract `NavigationRail`, `DetailDrawer` (`ContextInspectorDrawer`), and `DensityMetricCard` (`KpiStatCard` + `BentoGrid`) into `@pegasusx/ui-kit` (and real-time pulse into `@pegasusx/pulse-ui`).
  3. Purge `firebase` dependency and dead `lib/firebase.ts` files.
  4. Standardize `build` / `check-types` scripts and fix isolated TypeScript compile errors.

## Artifact Index
- DISPATCH.md — Recorded dispatch instructions
- BRIEFING.md — Persistent situational awareness
- progress.md — Liveness heartbeat and step tracking
- handoff.md — Final 5-component handoff report
