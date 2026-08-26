## 2026-08-21T08:27:46Z
<USER_REQUEST>
You are the Project Orchestrator (teamwork_preview_orchestrator).

Workspace root: /Users/shakhzod/Desktop/V.O.I.D
Your working directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_3
Authoritative user request file: /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md

Mission:
Execute the phased code gap closure plan for the PegasusX repository located at /Users/shakhzod/Desktop/V.O.I.D. This task involves closing the remaining Layer A (in-repo code) gaps identified in the surface audits.

Requirements:
### R1. DevOps and Backend Architecture
Consolidate the nested-only CI jobs into the root `.github/workflows/pegasusx-ci.yml` and fix the `reatilerapp` typo. Split the massive `bootstrap.go` file into modular components (e.g., `infra.go`, `services.go`, `workers.go`). Migrate `spanner.Client.Apply` usages in factory/warehouse packages to `RunTx` + `outbox.EmitJSON`. 

### R2. Geography, Maps, and Security
Enforce H3 resolution 7 in matching writers, and use a distinct named field for resolution 9 in settlement/perimeter logic. Add authentication middleware (`RequireRole` or `RequireAnyAuthenticated`) and country-bias to geocode endpoints. Switch the factory fleet list to pull from Spanner `FactoryTruckManifests`.

### R3. UI Consistency
Standardize the control-tower web map and Retailer Android hex map to use MapLibre + Carto with dynamic pack-based cameras (`mapInitialViewState(pack)`). Remove the Mapbox fallback token and hardcoded San Francisco camera. Remove misleading "wired later" UI theatre on Factory/Retailer mobile apps (either implement the true canvas or show a list/drop the map). Migrate `admin-portal` to use `packages/types` and `@pegasusx/ui-kit`.

Acceptance Criteria:
- Backend & Infrastructure:
  * CI jobs are successfully consolidated into the root workflow file and all typos are fixed.
  * `bootstrap.go` is cleanly split without breaking the build.
  * No `spanner.Client.Apply` calls remain in the factory auth, planning, or warehouse ops files.
  * Geocode endpoints successfully reject unauthenticated requests.
- UI & Maps (Agent-as-Judge):
  * An independent reviewer verifies that the control-tower web map and Retailer Android map use the pack camera and MapLibre/Carto, with no references to the Mapbox fallback token.
  * An independent reviewer confirms the factory fleet list fetches data from Spanner.
  * An independent reviewer confirms mobile map views (Factory/Retailer) honestly reflect their state without misleading "wired later" empty canvases.

Instructions:
- Note that previous milestones M1 and M2 may have partial or complete work in `.agents/teamwork_preview_worker_m1/` and `.agents/teamwork_preview_worker_m2/`. Inspect the codebase and `.agents/` to determine the current state and resume / execute to 100% completion.
- Maintain your `plan.md`, `progress.md`, and `BRIEFING.md` in your working directory.
- Coordinate with specialist workers / reviewers.
- When all tasks are completed and verified internally, send a message back to the Sentinel reporting victory claim so that independent Victory Audit can proceed.
</USER_REQUEST>
