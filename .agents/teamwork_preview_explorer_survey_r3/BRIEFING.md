# BRIEFING — 2026-08-20T19:36:30Z

## Mission
Investigate Requirement 3 (R3: UI Consistency) across PegasusX: control-tower web map, Retailer Android hex map, Factory/Retailer mobile apps UI theatre, and admin-portal types/ui-kit migration. Produce analysis.md and handoff.md.

## 🔒 My Identity
- Archetype: Explorer
- Roles: Investigation, Synthesis
- Working directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_r3
- Original parent: 5b42a930-75c6-4dc7-9f02-2111f624129e
- Milestone: Survey & Investigation (R3)

## 🔒 Key Constraints
- Read-only investigation — do NOT implement
- Files for content delivery, Messages for coordination
- Handoff must contain 5 sections: Observation, Logic Chain, Caveats, Conclusion, Verification Method

## Current Parent
- Conversation ID: 5b42a930-75c6-4dc7-9f02-2111f624129e
- Updated: 2026-08-20T19:36:30Z

## Investigation State
- **Explored paths**:
  - `packages/ui-kit/src/control-tower/HexagonalControlTowerMap.tsx`
  - `packages/api-client/market-pack.ts`
  - `apps/warehouse-portal/components/FleetLiveMap.tsx`
  - `apps/retailer-app-android/.../controltower/HexagonalControlTowerMap.kt`
  - `apps/retailer-app-android/.../controltower/LiveEKGNetworkGraph.kt`
  - `apps/retailer-app-android/.../controltower/ControlTowerScreen.kt`
  - `apps/retailer-app-ios/.../ControlTower/HexagonalControlTowerMap.swift`
  - `apps/retailer-app-ios/.../ControlTower/LiveEKGNetworkGraph.swift`
  - `apps/retailer-app-ios/.../ControlTower/ControlTowerView.swift`
  - `apps/factory-app-android/.../fleet/FleetScreen.kt`
  - `apps/factory-app-ios/.../Fleet/FleetView.swift`
  - `apps/admin-portal/package.json`
  - `apps/admin-portal/lib/api.ts`
  - `apps/admin-portal/components/*`
  - `packages/types/index.ts`
- **Key findings**:
  - `HexagonalControlTowerMap.tsx` uses Mapbox GL, a fallback token, and hardcoded San Francisco coordinates `(-122.4, 37.74)`. Needs migration to `react-map-gl/maplibre` + Carto dark style with `mapInitialViewState(pack)`.
  - `retailer-app-android`'s `HexagonalControlTowerMap.kt` uses Google Maps and SF coordinates `(37.74, -122.4)`.
  - `retailer-app-ios` contains "wired later" dummy map and hardcoded fake node graph. The truthful UI is `ControlTowerView.swift`'s pulse list.
  - `admin-portal` is missing `@pegasusx/types` and `@pegasusx/ui-kit` in `package.json` and duplicates local type definitions in `lib/api.ts`.
- **Unexplored areas**: None for R3 survey.

## Key Decisions Made
- Fully documented exact file paths, line numbers, and migration strategy in `analysis.md` and `handoff.md`.

## Artifact Index
- `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_r3/analysis.md` — Detailed investigation report
- `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_r3/handoff.md` — 5-section handoff report


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
