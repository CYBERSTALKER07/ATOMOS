## 2026-08-21T00:42:09Z
You are the Worker for Milestone 3 (M3: UI Consistency).
Working Directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m3
Workspace Root: /Users/shakhzod/Desktop/V.O.I.D
Authoritative Request: /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md
Explorer Handoff Report: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_r3/handoff.md (read this thoroughly!)

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. An auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

File Ownership:
- `pegasusX/packages/ui-kit/*`
- `pegasusX/packages/types/*`
- `pegasusX/apps/admin-portal/*`
- `pegasusX/apps/retailer-app-android/*`
- `pegasusX/apps/retailer-app-ios/*`

Tasks:
1. Control-Tower Web Map Standardization:
   - In `pegasusX/packages/ui-kit/src/control-tower/HexagonalControlTowerMap.tsx`:
     * Replace Mapbox imports with `react-map-gl/maplibre` and `maplibre-gl`.
     * Remove `MAPBOX_ACCESS_TOKEN`, `mapbox-gl/dist/mapbox-gl.css`, and hardcoded SF coordinates `(-122.4, 37.74)`.
     * Use Carto dark style (`https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json`).
     * Use dynamic pack camera `mapInitialViewState(pack)` from `@pegasusx/api-client` (or passed pack prop).
2. Mobile UI Theatre Cleanup:
   - In `pegasusX/apps/retailer-app-android/app/src/main/java/com/pegasusx/retailer/ui/controltower/HexagonalControlTowerMap.kt` and `LiveEKGNetworkGraph.kt`, remove hardcoded SF coordinates and fake simulated graphs, routing to the truthful pulse grid in `ControlTowerScreen.kt`.
   - In `pegasusX/apps/retailer-app-ios/retailerapp/retailerapp/Screens/ControlTower/HexagonalControlTowerMap.swift` and `LiveEKGNetworkGraph.swift`, remove "wired later" comments and mock hardcoded node arrays, making `ControlTowerView.swift` the honest destination.
3. Admin-Portal Migration:
   - In `pegasusX/apps/admin-portal/package.json`, add `"@pegasusx/types": "workspace:*"` and `"@pegasusx/ui-kit": "workspace:*"`.
   - In `pegasusX/packages/types/index.ts`, export canonical DTO types (`Tenant`, `FlagOverride`, `FlagEval`, `AccuracyRow`, `AuditRow`, `MatchQueueItem`, `BillingInvoice`, `BillingFeeSchedule`, etc.).
   - In `pegasusX/apps/admin-portal/lib/api.ts`, import types from `@pegasusx/types` and remove duplicate local interface declarations.
   - In `pegasusX/apps/admin-portal/app/globals.css` and components, adopt `@pegasusx/ui-kit` styling and UI components while preserving all test-contract strings (e.g. `gs-u-admin-health`, `tenant-register`, `COUNT(*)`, etc.).

Verification:
- Run `pnpm --filter @pegasusx/ui-kit typecheck`
- Run `pnpm --filter @pegasusx/admin-portal typecheck`
- Run `pnpm --filter @pegasusx/admin-portal test`
- Grep to verify no Mapbox fallback tokens (`pk.eyJ1IjoiZGVmYXVsdC`) or hardcoded SF coordinates in map views remain.
- Grep to verify no "wired later" strings remain in mobile apps.

Deliverables:
- Write `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m3/changes.md`.
- Write `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m3/handoff.md` with build and test outputs.
- Send a completion message to parent when done.


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
