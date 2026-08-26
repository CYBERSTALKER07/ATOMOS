## 2026-08-20T19:25:36Z
Execute the phased code gap closure plan for the PegasusX repository located at /Users/shakhzod/Desktop/V.O.I.D to close remaining Layer A (in-repo code) gaps.

Requirements:
1. R1. DevOps and Backend Architecture:
   - Consolidate nested-only CI jobs into root `.github/workflows/pegasusx-ci.yml` and fix the `reatilerapp` typo.
   - Split the massive `bootstrap.go` file into modular components (e.g., `infra.go`, `services.go`, `workers.go`).
   - Migrate `spanner.Client.Apply` usages in factory/warehouse packages to `RunTx` + `outbox.EmitJSON`.

2. R2. Geography, Maps, and Security:
   - Enforce H3 resolution 7 in matching writers, and use a distinct named field for resolution 9 in settlement/perimeter logic.
   - Add authentication middleware (`RequireRole` or `RequireAnyAuthenticated`) and country-bias to geocode endpoints.
   - Switch the factory fleet list to pull from Spanner `FactoryTruckManifests`.

3. R3. UI Consistency:
   - Standardize the control-tower web map and Retailer Android hex map to use MapLibre + Carto with dynamic pack-based cameras (`mapInitialViewState(pack)`).
   - Remove the Mapbox fallback token and hardcoded San Francisco camera.
   - Remove misleading "wired later" UI theatre on Factory/Retailer mobile apps (either implement the true canvas or show a list/drop the map).
   - Migrate `admin-portal` to use `packages/types` and `@pegasusx/ui-kit`.

Acceptance Criteria:
- Backend & Infrastructure:
  * CI jobs consolidated into root workflow file and typos fixed.
  * `bootstrap.go` cleanly split without breaking the build.
  * No `spanner.Client.Apply` calls remain in factory auth, planning, or warehouse ops files.
  * Geocode endpoints reject unauthenticated requests.
- UI & Maps:
  * Control-tower web map and Retailer Android map use pack camera and MapLibre/Carto, no Mapbox fallback token references.
  * Factory fleet list fetches data from Spanner.
  * Mobile map views honestly reflect their state without misleading "wired later" empty canvases.
