# Orchestration Plan: PegasusX Phased Code Gap Closure

## Objectives
Close remaining Layer A code gaps in the PegasusX repository located at `/Users/shakhzod/Desktop/V.O.I.D`.

## Requirements Breakdown & Milestones

### Milestone 1 (M1): DevOps and Backend Architecture
1. Consolidate nested-only CI jobs into root `.github/workflows/pegasusx-ci.yml` and fix `reatilerapp` typo.
2. Split massive `bootstrap.go` into modular components (`infra.go`, `services.go`, `workers.go`, etc.) without breaking build.
3. Migrate `spanner.Client.Apply` usages in factory/warehouse packages to `RunTx` + `outbox.EmitJSON`.

### Milestone 2 (M2): Geography, Maps, and Security
1. Enforce H3 resolution 7 in matching writers, and use distinct named field for resolution 9 in settlement/perimeter logic.
2. Add authentication middleware (`RequireRole` or `RequireAnyAuthenticated`) and country-bias to geocode endpoints.
3. Switch factory fleet list to pull from Spanner `FactoryTruckManifests`.

### Milestone 3 (M3): UI Consistency
1. Standardize control-tower web map and Retailer Android hex map to use MapLibre + Carto with dynamic pack-based cameras (`mapInitialViewState(pack)`).
2. Remove Mapbox fallback token and hardcoded San Francisco camera.
3. Remove misleading "wired later" UI theatre on Factory/Retailer mobile apps (implement true canvas or show list/drop map).
4. Migrate `admin-portal` to use `packages/types` and `@pegasusx/ui-kit`.

### Phase 4: Independent Verification & Review
- Comprehensive builds and tests across Go services, mobile Android/iOS components, and web frontends.
- Independent verification by reviewer agents.
- Victory reporting to Sentinel.
