# Execution Plan: PegasusX Gap Closure

## Objectives
Close all remaining Layer A gaps across DevOps/Backend (R1), Geography/Maps/Security (R2), and UI Consistency (R3).

## Milestones
1. **Milestone 1: DevOps & Backend Architecture (R1)** [COMPLETED]
   - Consolidate CI jobs into `.github/workflows/pegasusx-ci.yml`.
   - Fix all occurrences of `reatilerapp` typo.
   - Modularize `bootstrap.go` into `config.go`, `app.go`, `infra.go`, `services.go`, `workers.go`, `queries.go`.
   - Migrate `spanner.Client.Apply` usages to `ReadWriteTransaction` + `txn.BufferWrite`.

2. **Milestone 2: Geography, Maps, and Security (R2)** [COMPLETED]
   - Enforce H3 Resolution 7 in matching writers; use distinct named helpers (`SettlementH3Cell`, `H3CellRes9`) for Resolution 9 in settlement/perimeter logic.
   - Protect geocode endpoints with `RequireAnyAuthenticated` middleware and add country-bias (`components=country:<cc>` / `countrycodes=<cc>`) and country-scoped cache keys.
   - Switch factory fleet list to pull from Spanner `FactoryTruckManifests`, `Vehicles`, and `Drivers`.

3. **Milestone 3: UI Consistency (R3)** [IN-PROGRESS]
   - Standardize `HexagonalControlTowerMap.tsx` in `@pegasusx/ui-kit` to MapLibre + Carto dark style and dynamic pack camera (`mapInitialViewState(pack)`). Eliminate Mapbox fallback token and SF coordinates.
   - Clean up mobile UI theatre in Retailer Android and iOS apps (eliminate fake simulated graphs, "wired later" comments, mock node arrays).
   - Migrate `apps/admin-portal` to `@pegasusx/types` and `@pegasusx/ui-kit`, exporting canonical DTOs from `packages/types`.

4. **Milestone 4: Multi-Agent Review & Challenge** [PENDING]
   - Reviewer 1: Verify Backend, DevOps, Security & Go test suites.
   - Reviewer 2: Verify UI consistency, MapLibre/Carto, mobile theatre removal, Spanner fleet queries, and TypeScript/Jest tests.

5. **Milestone 5: Synthesis & Victory Report** [PENDING]
   - Synthesize all gate results and confirm 100% acceptance criteria met.
   - Send victory report to Sentinel (parent agent).
