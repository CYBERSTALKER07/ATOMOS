# Original User Request

## Initial Request — 2026-08-20T17:21:08+05:00

Read all `.md` and `.docx` files across the entire repository (/Users/shakhzod/Desktop/V.O.I.D), compare them to the actual codebase (source of truth, particularly pegasusX/ and actual implementation code), and update the documentation in place to align with the current implementation.

Ensure:
1. R1: Comprehensive audit of all `.md` and `.docx` files in the repository.
2. R2: Codebase synchronization — determine actual state of features, schemas, configurations. Identify discrepancies where docs claim "done" or "wired" without implementation.
3. R3: In-place updates — correct obsolete/false claims and document new implementations accurately.
4. Maintain your plan.md, progress.md, and BRIEFING.md in your working directory.
5. Coordinate with specialists and ensure high integrity.
6. When work is complete, send a message back to report completion so independent verification can proceed.

## Update — 2026-08-20T18:46:58Z

# Teamwork Project Prompt

> Requested team: Full team

Read all markdown and docx documentation files in `/Users/shakhzod/Desktop/V.O.I.D`, compare them against the actual codebase implementation, and update the documentation to accurately reflect the real codebase.

Working directory: /Users/shakhzod/Desktop/V.O.I.D
Integrity mode: development

## Requirements

### R1. Documentation Conversion
Identify all `.docx` files in the repository and convert them to Markdown (`.md`) format.

### R2. Codebase Alignment
Analyze the current source code implementation. Update all documentation files (both existing `.md` files and the newly converted ones) so that they accurately describe the codebase's current state, architecture, and behavior. Remove outdated information.

## Acceptance Criteria

### Documentation format
- [ ] No `.docx` files remain in the active documentation directories (they are either deleted or moved to an archive folder).
- [ ] New `.md` files exist for all converted `.docx` files.

### Content Accuracy (Agent-as-Judge)
- [ ] An independent reviewer agent confirms that a random sample of the updated documentation accurately matches the logic in the corresponding source code files.
- [ ] The updated documentation contains no references to deprecated features that no longer exist in the code.

## 2026-08-20T19:24:12Z

# Teamwork Project Prompt

> Requested team: Full team

Execute the phased code gap closure plan for the PegasusX repository located at `/Users/shakhzod/Desktop/V.O.I.D`. This task involves closing the remaining Layer A (in-repo code) gaps identified in the surface audits.

Working directory: /Users/shakhzod/Desktop/V.O.I.D
Integrity mode: development

## Requirements

### R1. DevOps and Backend Architecture
Consolidate the nested-only CI jobs into the root `.github/workflows/pegasusx-ci.yml` and fix the `reatilerapp` typo. Split the massive `bootstrap.go` file into modular components (e.g., `infra.go`, `services.go`, `workers.go`). Migrate `spanner.Client.Apply` usages in factory/warehouse packages to `RunTx` + `outbox.EmitJSON`. 

### R2. Geography, Maps, and Security
Enforce H3 resolution 7 in matching writers, and use a distinct named field for resolution 9 in settlement/perimeter logic. Add authentication middleware (`RequireRole` or `RequireAnyAuthenticated`) and country-bias to geocode endpoints. Switch the factory fleet list to pull from Spanner `FactoryTruckManifests`.

### R3. UI Consistency
Standardize the control-tower web map and Retailer Android hex map to use MapLibre + Carto with dynamic pack-based cameras (`mapInitialViewState(pack)`). Remove the Mapbox fallback token and hardcoded San Francisco camera. Remove misleading "wired later" UI theatre on Factory/Retailer mobile apps (either implement the true canvas or show a list/drop the map). Migrate `admin-portal` to use `packages/types` and `@pegasusx/ui-kit`.

## Acceptance Criteria

### Backend & Infrastructure
- [ ] CI jobs are successfully consolidated into the root workflow file and all typos are fixed.
- [ ] `bootstrap.go` is cleanly split without breaking the build.
- [ ] No `spanner.Client.Apply` calls remain in the factory auth, planning, or warehouse ops files.
- [ ] Geocode endpoints successfully reject unauthenticated requests.

### UI & Maps (Agent-as-Judge)
- [ ] An independent reviewer verifies that the control-tower web map and Retailer Android map use the pack camera and MapLibre/Carto, with no references to the Mapbox fallback token.
- [ ] An independent reviewer confirms the factory fleet list fetches data from Spanner.
- [ ] An independent reviewer confirms mobile map views (Factory/Retailer) honestly reflect their state without misleading "wired later" empty canvases.



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

## 2026-08-30T00:18:00Z

# Teamwork Project Prompt — Draft

> Status: Launched.
> Goal: Craft prompt → get user approval → delegate to teamwork_preview
> Requested team: Very large team (Maximal parallel exploration of the codebase)

Use a very large team of agents. Conduct a comprehensive, line-by-line review of the entire PegasusX Go backend. Question every feature, identify architectural gaps, logical bugs, and inconsistencies across the codebase.

Working directory: ~/teamwork_projects/pegasusx_audit
Integrity mode: development

## Requirements

### R1. Line-by-Line Ecosystem Audit
Analyze the Go backend codebase (`apps/backend-go/`) line-by-line, focusing on general architecture, logic bugs, and ecosystem consistency. Ensure that role-row parity, Spanner transactions, Kafka outbox events, and WebSocket multi-hub broadcasts are correctly integrated and consistent across domains.

### R2. Comprehensive Report Generation
Produce a detailed Markdown report (`backend_audit_report.md`) documenting all findings, logical inconsistencies, and open architectural questions for every feature analyzed. Do not implement code fixes; focus entirely on surfacing issues and questions.

## Acceptance Criteria

### Audit Depth & Formatting
- [ ] The report explicitly covers and analyzes code from at least 5 major role domains (e.g., supplier, retailer, warehouse, factory, driver, payload, order).
- [ ] Every identified bug or architectural gap cites the exact file path and line number(s).
- [ ] The report evaluates whether the codebase adheres to the ecosystem alignment rules (e.g., Spanner Tx + Kafka Outbox + WebSocket fanout).

### Architectural Inquiry
- [ ] The report includes a dedicated "Open Questions" section that surfaces at least 5 deep architectural or edge-case questions regarding unhandled scenarios or state inconsistencies in the codebase.

