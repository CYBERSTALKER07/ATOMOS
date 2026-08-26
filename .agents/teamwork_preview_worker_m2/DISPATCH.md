# Dispatch: Worker 2 (Parity Matrix, Features & Scorecards Synchronization)

## Identity
- Subagent: teamwork_preview_worker_m2
- Type: teamwork_preview_worker
- Role: Parity & Feature Matrix Docs Synchronizer
- Working Directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m2
- Authoritative Request: /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md
- Project Scope: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_1/PROJECT.md

## Exclusive Write Boundaries
You have exclusive write access to:
- `pegasusX/docs/ROLE_ROW_PARITY_MATRIX.md`
- `pegasusX/docs/ROLE_FEATURES_DOCS_VS_CODE.md`
- `pegasusX/docs/session-2026-08-13/SCORECARD.md`
- `pegasusX/docs/session-2026-08-13/RESIDUAL_REGISTER.md`
- `pegasusX/docs/session-2026-08-13/GAP_LEDGER.md`
- `pegasusX/docs/session-2026-08-13/MASTER_10_10_EXECUTION_PROGRAM.md`
- `pegasusX/docs/session-2026-08-13/PROD_READINESS_SEQUENCE.md`

## Inputs & Verified Evidence
- Explorer 1 Report: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_1/doc_inventory_report.md`
- Explorer 2 Report: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_2/backend_sot_report.md`
- Explorer 3 Report: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_3/clients_parity_report.md`

## Mandatory Instructions
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A teamwork_preview_auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

Update the assigned files in-place:
1. Update `ROLE_ROW_PARITY_MATRIX.md` to ensure all 6 role rows cite verified code paths, passing tests, and accurate status definitions.
2. Synchronize `ROLE_FEATURES_DOCS_VS_CODE.md` with live backend route endpoints and 410 boundaries (inventory audit unwired 410, quantity negotiation 410, sealed-all manifests, AI predictions).
3. Align `SCORECARD.md`, `RESIDUAL_REGISTER.md`, `GAP_LEDGER.md`, and `PROD_READINESS_SEQUENCE.md` with the verified state (Layer A code verified, deploy-time secrets/scaling clearly identified as Layer B residuals).
4. Update your `progress.md` with liveness and write a 5-component `handoff.md`.

## 2026-08-20T19:42:09Z

You are the Worker for Milestone 2 (M2: Geography, Maps, and Security).
Working Directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m2
Workspace Root: /Users/shakhzod/Desktop/V.O.I.D
Authoritative Request: /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md
Explorer Handoff Report: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_r2_gen4/handoff.md (read this thoroughly!)

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. An auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

File Ownership:
- `pegasusX/apps/backend-go/proximity/*`
- `pegasusX/apps/backend-go/order/*`
- `pegasusX/apps/backend-go/platformroutes/routes.go`
- `pegasusX/apps/backend-go/geolocation/*`
- `pegasusX/apps/backend-go/factory/ios_compat.go`, `pegasusX/apps/backend-go/factory/service.go`

Tasks:
1. H3 Resolution:
   - Ensure matching writers enforce Resolution 7 (`MatchingResolution = 7`, `MatchingH3Cell`).
   - In settlement/perimeter logic, ensure Resolution 9 uses a distinct named field/helper (`SettlementH3Cell`, `H3CellRes9`) to eliminate ambiguity with Resolution 7 matching cells.
2. Geocode API Security & Country Bias:
   - In `pegasusX/apps/backend-go/platformroutes/routes.go` and `geolocation/handlers.go`, protect the geocode routes (`/v1/platform/geocode/*`) with authentication middleware (e.g. `RequireAnyAuthenticated` or appropriate auth middleware, or role check).
   - In `geolocation/service.go` and handlers, add country-bias support (`components=country:<cc>` for Google Maps, `countrycodes=<cc>` for Nominatim), parsing country from request/context/pack or default market, and namespace cache keys with the country code (`geo:<endpoint>:<cc>:<query>`).
3. Factory Fleet Spanner Data:
   - In `pegasusX/apps/backend-go/factory/ios_compat.go` and `factory/service.go`, update `HandleFleet` and `HandleFleetVehicles` to fetch live data by querying Spanner `Vehicles` (where `HomeNodeType = 'FACTORY' AND HomeNodeId = @factoryId`) joined with active `FactoryTruckManifests` (`State IN ('LOADING', 'SEALED', 'DISPATCHED')`) and `Drivers` instead of in-memory demo data `s.fleetVehicles`.

Verification:
- Run `go test ./proximity/... ./geolocation/... ./order/... ./factory/...` in `pegasusX/apps/backend-go`.
- Run `go test ./...` in `pegasusX/apps/backend-go`.
- Confirm geocode endpoints reject unauthenticated requests (add/run unit tests for this).

Deliverables:
- Write `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m2/changes.md`.
- Write `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m2/handoff.md` with build and test outputs.
- Send a completion message to parent when done.
