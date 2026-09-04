# BRIEFING — 2026-08-21T00:42:09Z

## Mission
Complete Milestone 3 (M3: UI Consistency) for PegasusX: harmonize Control Tower Web Map to MapLibre + Carto dark style + dynamic pack camera, clean up Mobile UI theatre (Android & iOS) to route truthfully to pulse grids, and migrate admin-portal to use @pegasusx/types and @pegasusx/ui-kit.

## 🔒 My Identity
- Archetype: worker
- Roles: implementer, qa, specialist
- Working directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m3
- Original parent: 5b42a930-75c6-4dc7-9f02-2111f624129e
- Milestone: M3 (UI Consistency)

## 🔒 Key Constraints
- DO NOT CHEAT: Genuine implementations only, no hardcoding test expectations, dummy facades, or fabricating verification outputs.
- Adhere strictly to file ownership:
  * `pegasusX/packages/ui-kit/*`
  * `pegasusX/packages/types/*`
  * `pegasusX/apps/admin-portal/*`
  * `pegasusX/apps/retailer-app-android/*`
  * `pegasusX/apps/retailer-app-ios/*`
- Maintain admin-portal test contracts (`gs-u-admin-health`, `tenant-register`, `COUNT(*)`, etc.).
- Follow minimal change principle and verify all builds/tests before completion.

## Current Parent
- Conversation ID: 5b42a930-75c6-4dc7-9f02-2111f624129e
- Updated: 2026-08-21T00:42:09Z

## Task Summary
- **What to build**:
  1. Standardize `pegasusX/packages/ui-kit/src/control-tower/HexagonalControlTowerMap.tsx` with MapLibre + Carto dark + dynamic `mapInitialViewState(pack)`.
  2. Clean up mobile UI theatre in Retailer Android (`HexagonalControlTowerMap.kt`, `LiveEKGNetworkGraph.kt`) and iOS (`HexagonalControlTowerMap.swift`, `LiveEKGNetworkGraph.swift`) to honestly route to pulse grids/views.
  3. Migrate `admin-portal`: add `@pegasusx/types` & `@pegasusx/ui-kit` to `package.json`, export canonical DTOs in `@pegasusx/types`, update `admin-portal/lib/api.ts`, update globals.css and components.
- **Success criteria**:
  * `pnpm --filter @pegasusx/ui-kit typecheck` passes
  * `pnpm --filter @pegasusx/admin-portal typecheck` passes
  * `pnpm --filter @pegasusx/admin-portal test` passes
  * No Mapbox fallback tokens or hardcoded SF coords in map views
  * No "wired later" strings in mobile apps
- **Interface contracts**: `pegasusX/packages/types/index.ts`, `pegasusX/packages/api-client/market-pack.ts`
- **Code layout**: Monorepo under `pegasusX/`

## Key Decisions Made
- Replace Mapbox with `react-map-gl/maplibre` and `maplibre-gl` in UI kit.
- Move admin-portal DTO types to `@pegasusx/types`.

## Change Tracker
- **Files modified**: [TBD]
- **Build status**: [TBD]
- **Pending issues**: None

## Quality Status
- **Build/test result**: [TBD]
- **Lint status**: [TBD]
- **Tests added/modified**: [TBD]

## Loaded Skills
None requested.

## Artifact Index
- `.agents/teamwork_preview_worker_m3/DISPATCH.md` — Assignment instructions
- `.agents/teamwork_preview_worker_m3/BRIEFING.md` — Agent working memory
- `.agents/teamwork_preview_worker_m3/progress.md` — Liveness & progress tracking
- `.agents/teamwork_preview_worker_m3/changes.md` — Detailed record of modifications
- `.agents/teamwork_preview_worker_m3/handoff.md` — Final 5-component handoff report


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
