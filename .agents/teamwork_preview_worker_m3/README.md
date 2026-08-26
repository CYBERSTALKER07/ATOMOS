# Worker M3 Working Directory
Assigned: Milestone 3 (UI Consistency)
- Control-Tower Web Map: In `pegasusX/packages/ui-kit/src/control-tower/HexagonalControlTowerMap.tsx`, convert from Mapbox to MapLibre + Carto dark style (`https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json`), dynamic pack camera via `mapInitialViewState(pack)`, remove Mapbox fallback token (`pk.eyJ1IjoiZGVmYXVsdC...`) and hardcoded SF coordinates (`-122.4, 37.74`).
- Mobile Map Views / UI Theatre:
  - In `retailer-app-android` (`HexagonalControlTowerMap.kt`, `LiveEKGNetworkGraph.kt`), standardize map/camera and remove stubbed SF coordinates / fake graphs; route to truthful `ControlTowerScreen.kt` pulse grid.
  - In `retailer-app-ios` (`HexagonalControlTowerMap.swift`, `LiveEKGNetworkGraph.swift`), remove "wired later" stubs and mock hardcoded nodes, ensuring `ControlTowerView.swift` is the honest destination.
- Admin Portal: Add `@pegasusx/types` and `@pegasusx/ui-kit` to `pegasusX/apps/admin-portal/package.json`, export canonical DTO types in `packages/types/index.ts`, migrate local interfaces in `apps/admin-portal/lib/api.ts` to `@pegasusx/types`, and adopt `@pegasusx/ui-kit` styles/components.
- Run tests: `pnpm --filter @pegasusx/ui-kit typecheck`, `pnpm --filter @pegasusx/admin-portal typecheck`, `pnpm --filter @pegasusx/admin-portal test`.
