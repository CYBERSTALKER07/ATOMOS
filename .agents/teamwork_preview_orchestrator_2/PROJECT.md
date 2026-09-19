# Project: PegasusX Layer A Code Gap Closure

## Architecture
PegasusX multi-package system comprising:
- Go backend services (bootstrap, factory, warehouse, matching, settlement, geocode APIs)
- Web frontends (`admin-portal`, `control-tower`)
- Mobile applications (Factory and Retailer apps, Android/iOS)
- Shared packages (`packages/types`, `@pegasusx/ui-kit`)
- CI/CD workflows (`.github/workflows/`)

## Feature Inventory
| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| 1 | CI Consolidation & Typo Fix | Consolidate nested CI jobs into root `pegasusx-ci.yml`, fix `reatilerapp` typo | M1 | ORIGINAL_REQUEST |
| 2 | Split bootstrap.go | Modularize bootstrap.go into infra, services, workers | M1 | ORIGINAL_REQUEST |
| 3 | Spanner Outbox Migration | Migrate `spanner.Client.Apply` in factory/warehouse to `RunTx` + `outbox.EmitJSON` | M1 | ORIGINAL_REQUEST |
| 4 | H3 Resolution Standardization | Res 7 in matching writers, distinct named field for res 9 in settlement/perimeter | M2 | ORIGINAL_REQUEST |
| 5 | Geocode Auth & Country Bias | Add auth middleware & country-bias to geocode endpoints | M2 | ORIGINAL_REQUEST |
| 6 | Factory Fleet Spanner Manifests | Switch factory fleet list to pull from Spanner `FactoryTruckManifests` | M2 | ORIGINAL_REQUEST |
| 7 | Map Standardization | Control-tower & Retailer Android map to MapLibre + Carto with pack camera | M3 | ORIGINAL_REQUEST |
| 8 | Remove Fallback Token/SF Cam | Remove Mapbox fallback token and hardcoded San Francisco camera | M3 | ORIGINAL_REQUEST |
| 9 | Honest Mobile Map Views | Remove misleading "wired later" UI theatre on Factory/Retailer mobile apps | M3 | ORIGINAL_REQUEST |
| 10 | Admin Portal Types/UI Kit | Migrate `admin-portal` to use `packages/types` and `@pegasusx/ui-kit` | M3 | ORIGINAL_REQUEST |

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| 0 | Survey & Exploration | Locate exact files and examine implementation gaps | none | IN_PROGRESS |
| 1 | M1: DevOps & Backend Architecture | CI consolidation, bootstrap.go split, Spanner outbox migration | M0 | PLANNED |
| 2 | M2: Geography, Maps & Security | H3 res 7/9, geocode auth & bias, factory fleet Spanner | M0 | PLANNED |
| 3 | M3: UI Consistency | MapLibre/Carto, pack camera, honest mobile map, admin portal ui-kit | M0 | PLANNED |
| 4 | M4: Final Review & Verification | End-to-end verification, review pass, victory declaration | M1, M2, M3 | PLANNED |

## Interface Contracts
- Go backend: Packages under `pegasusX/` or root Go modules maintain package exports, compilation, and unit test pass.
- Web apps: TypeScript builds succeed without broken imports.
- Mobile: Kotlin/Compose/Swift/React Native views cleanly compile and accurately reflect real logic.


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
