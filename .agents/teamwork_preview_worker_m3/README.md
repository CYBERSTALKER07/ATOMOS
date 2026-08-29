# Worker M3 Working Directory
Assigned: Milestone 3 (UI Consistency)
- Control-Tower Web Map: In `pegasusX/packages/ui-kit/src/control-tower/HexagonalControlTowerMap.tsx`, convert from Mapbox to MapLibre + Carto dark style (`https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json`), dynamic pack camera via `mapInitialViewState(pack)`, remove Mapbox fallback token (`pk.eyJ1IjoiZGVmYXVsdC...`) and hardcoded SF coordinates (`-122.4, 37.74`).
- Mobile Map Views / UI Theatre:
  - In `retailer-app-android` (`HexagonalControlTowerMap.kt`, `LiveEKGNetworkGraph.kt`), standardize map/camera and remove stubbed SF coordinates / fake graphs; route to truthful `ControlTowerScreen.kt` pulse grid.
  - In `retailer-app-ios` (`HexagonalControlTowerMap.swift`, `LiveEKGNetworkGraph.swift`), remove "wired later" stubs and mock hardcoded nodes, ensuring `ControlTowerView.swift` is the honest destination.
- Admin Portal: Add `@pegasusx/types` and `@pegasusx/ui-kit` to `pegasusX/apps/admin-portal/package.json`, export canonical DTO types in `packages/types/index.ts`, migrate local interfaces in `apps/admin-portal/lib/api.ts` to `@pegasusx/types`, and adopt `@pegasusx/ui-kit` styles/components.
- Run tests: `pnpm --filter @pegasusx/ui-kit typecheck`, `pnpm --filter @pegasusx/admin-portal typecheck`, `pnpm --filter @pegasusx/admin-portal test`.


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
