# Project: PegasusX Gap Closure

## Architecture
- Monorepo containing Go backend (`pegasusX/apps/backend-go`), TypeScript packages (`packages/types`, `packages/ui-kit`, `packages/api-client`), web portals (`apps/admin-portal`, `apps/warehouse-portal`, etc.), and mobile apps (Android & iOS).

## Feature Inventory
| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| 1 | CI Consolidation & Typo Fix | Consolidate nested CI jobs into root workflow and fix `reatilerapp` typo | M1 | R1 |
| 2 | Bootstrap Modularization | Split `bootstrap.go` into `config.go`, `app.go`, `infra.go`, `services.go`, `workers.go`, `queries.go` | M1 | R1 |
| 3 | Spanner Transaction Migration | Migrate `spanner.Client.Apply` usages to `RunTx` + `BufferWrite` | M1 | R1 |
| 4 | H3 Resolution Disambiguation | Enforce Res 7 in matching writers; distinct named helpers for Res 9 in settlement | M2 | R2 |
| 5 | Geocode Auth & Country Bias | Add auth middleware, country-bias params, and country-scoped cache keys | M2 | R2 |
| 6 | Factory Fleet Spanner Data | Switch factory fleet list to pull from Spanner `FactoryTruckManifests` | M2 | R2 |
| 7 | Control-Tower Web Map | Standardize `HexagonalControlTowerMap.tsx` to MapLibre + Carto + pack camera | M3 | R3 |
| 8 | Mobile UI Theatre Cleanup | Remove fake maps/theatre in Android & iOS Retailer apps | M3 | R3 |
| 9 | Admin-Portal Migration | Migrate `admin-portal` to `@pegasusx/types` and `@pegasusx/ui-kit` | M3 | R3 |

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| 1 | DevOps & Backend Architecture | CI consolidation, typo fix, bootstrap modularization, Spanner Apply | none | DONE |
| 2 | Geography, Maps, Security | H3 Res7/9, Geocode Auth & Country Bias, Factory Fleet Spanner | M1 | DONE |
| 3 | UI Consistency | MapLibre/Carto, Mobile UI theatre cleanup, Admin-Portal migration | M1, M2 | DONE |
| 4 | Verification & Review | Multi-agent review and challenge | M3 | DONE |
| 5 | Victory Reporting | Final report to Sentinel | M4 | DONE |


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
