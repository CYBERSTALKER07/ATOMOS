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


## 2026-08-30T06:34:29+05:00

Use a very large team of agents. The team will comprehensively audit the recently built 4-phase ecosystem adaptability system, identify unhandled real-world edge cases (math loops, offline chaos), and implement defensive code to secure the system.

Working directory: /Users/shakhzod/Desktop/V.O.I.D/pegasusX
Integrity mode: development

## Requirements

### R1. Audit Existing Adaptability Phases
Review the offline sync engine, Control Tower Redis tracking, semantic reconciliation queue, and financial draft logic. Identify gaps, race conditions, or missing validation in the current implementation.

### R2. Stress Test & Defensive Implementation
Simulate extreme edge cases (e.g., continuous math recalculation loops, out-of-order offline pushes, conflicting manual fallbacks). Write defensive business logic and tests to handle these scenarios gracefully.

## Verification Resources
- Existing test suites in the `apps/backend-go` monorepo.
- `ssmr-smokecheck` suite for cross-role ecosystem assertions.

## Acceptance Criteria

### Audit & Stability
- [ ] A written audit report is produced detailing at least 3 concrete edge cases or vulnerabilities found in the current adaptability implementation.
- [ ] Code is implemented to address the identified edge cases.
- [ ] Running `go test ./...` in the backend monorepo passes with no regressions.
- [ ] The system does not hang or panic when simulated out-of-order commands are pushed to the sync engine.

## 2026-09-14T09:18:26Z

# Teamwork Project Prompt — Draft

> Status: Launched
> Goal: Craft prompt → get user approval → delegate to teamwork_preview
> Requested team: Large-scale agent team

Use a very large team of agents. Analyze the dual-system codebase (`pegasusX` and `pegasus.x`) to produce a comprehensive architectural overview. The output must include a written summary of key components and data flows, architectural diagrams, and a parity matrix detailing missing features and gaps between the two systems with a deep dive into all areas.

Working directory: /Users/shakhzod/Desktop/V.O.I.D
Integrity mode: development

## Requirements

### R1. Architectural Summary
Produce a detailed written document outlining the architecture of both `pegasusX` (Global Enterprise Multi-Tenant) and `pegasus.x` (Sovereign Lean Single-Tenant). Describe key components, data flow paths, and strict architectural boundaries.

### R2. Architectural Diagrams
Generate detailed Mermaid diagrams visualizing the architecture, components, and data flows for both systems.

### R3. Feature Parity Matrix
Create a comprehensive feature parity matrix documenting the capabilities of both systems. Detail missing features and domain logic gaps that exist in `pegasusX` but are missing from `pegasus.x` (including the Fleet Management lifecycle).

## Acceptance Criteria

### Verification Checklist
- [ ] The output includes at least two Mermaid architecture diagrams (one for pegasusX, one for pegasus.x).
- [ ] The output includes a tabular Parity Matrix explicitly comparing features between both codebases.
- [ ] The summary explicitly identifies the strict architectural boundaries and technologies (e.g., Spanner vs PostgreSQL).
- [ ] The Parity Matrix includes a specific section detailing the "Fleet Management" gap.

---
*Next: when approved → delegate via invoke_subagent (see Delegation Protocol)*

## 2026-09-16T12:25:27Z

Autonomous multi-agent deep architectural audit, feature comparison, and data flow verification across `pegasus`, `pegasusX`, and `pegasus.x` codebases in `/Users/shakhzod/Desktop/V.O.I.D`.

Working directory: /Users/shakhzod/Desktop/V.O.I.D
Integrity mode: development

## Reference Materials
- Ecosystem Master Plan: docs/plans/2026-09-16-ecosystem-deep-audit-plan.md
- Master Architectural Specification: enterprise_ecosystem_deep_dive.md
- Dual-System Parity Atlas: DUAL_SYSTEM_ARCHITECTURE_AND_PARITY.md

## Requirements

### R1. Enterprise Distributed Systems & Infrastructure Audit
Audit and verify the core distributed systems infrastructure across all three codebases (pegasus, pegasusX, pegasus.x), analyzing:
- Persistence & Sharding: Google Cloud Spanner (3,749-line DDL, root SupplierId partitioning, table interleaving) vs. PostgreSQL 16 (pgx/v5 connection pool, 69 migrations, TimescaleDB/PostGIS reality check).
- Messaging & Streaming: Apache Kafka (topic partitioning, hash balancing, consumer groups, RequiredAcks=all) vs. Redis 7 Streams (XADD, consumer groups) and Redis Pub/Sub channels.
- Connection Pooling: Spanner client gRPC session pool vs. PostgreSQL pgxpool (MaxConns: 25, MinConns: 5, lifetime/idle management).
- Load Balancing & Routing: Maglev consistent hashing read-router prototype (Uber H3 Res-7 -> Res-2 bitmask) vs. Sovereign Caddy 2 reverse proxy with direct TAS-IX domestic peering.
- Transactional Outbox & CDC: Atomic outbox pairing (SpannerTxnBuffer vs pgx.Tx), fair multi-tenant interleaving, SKIP LOCKED relays, and dead-letter queue isolation.
- Background Schedulers & Workers: Kafka consumer worker pools, cron schedulers (dispatch plan warmer, replenishment engine, AR dunning, control tower playbooks).

### R2. Feature Matrix & Cross-System Parity Analysis
Map all capabilities and identify functional gaps across the systems:
- Compare domain features across Order Management, Warehouse Operations, Fleet & Driver Management, Factory Production, Double-Entry Financial Accounting, Statutory Uzbekistan Tax/Fiscalization (Soliq 12% VAT, 25M UZS B2B cash limit), and Client Fleet surfaces (Next.js 15 portals, Tauri v2 desktops, Telegram Bot/Mini App, and Native Mobile apps).
- Explicitly trace the primary domain gap: Fleet Management & Driver Shift Lifecycle (Daily Driver-Vehicle Shift Pairing, Pre-trip DVIR Inspections, Mid-shift Hot-swapping).

### R3. Dynamic End-to-End Data Flow Verification
Trace and document step-by-step distributed data flows with exact file:line citations:
- Flow 1: E2E Order Lifecycle & Fulfillment (Checkout -> Reservation -> Wave -> Manifest -> Dispatch -> Doorstep Handover -> Fiscalization -> Payout).
- Flow 2: Fleet Management & Driver Shift Operations (Clock-in -> Vehicle Pairing -> DVIR -> Active Route -> Proof of Delivery).
- Flow 3: Real-time Telemetry & Digital Twin Projection (Driver GPS -> Kalman Smoothing -> Redis Geo -> WebSocket Broadcast -> Control Tower).
- Flow 4: Transactional Outbox Relay, Fair Interleaving & Deduplicated Consumption.
- Flow 5: Algorithmic Planning & S&OP Replenishment (Croston-SBA Intermittent Demand Forecasting, MEIO Dynamic Safety Stock, and Google OR-Tools CVRP).

### R4. Integrity & Architectural Boundary Enforcement
Enforce the Strict Two-System Architectural Boundary:
- Zero Spanner or Kafka libraries/imports inside pegasus.x.
- Zero single-tenant relational downgrades inside pegasusX.
- Zero floating-point arithmetic in financial calculations (strict 64-bit integer tiyins / minor units).
- Balanced double-entry general ledger identity (Debits == Credits).

## Acceptance Criteria

### Technical & Systemic Verification
- [ ] Every architectural dimension (sharding, connection pooling, Kafka, Redis, outbox, load balancing, workers) has an evidence-backed audit section with exact file:line citations.
- [ ] Database schemas (spanner.ddl and 69 PostgreSQL migrations) are fully audited with zero undocumented drift.
- [ ] The Two-System Boundary is verified via automated AST scan: 0 Spanner/Kafka references in pegasus.x, 0 single-tenant PG references in pegasusX.
- [ ] All 5 distributed data flows are fully traced from client ingress down to persistence commits and real-time fanout.
- [ ] Go backend packages compile cleanly and pass tests in both pegasusX/apps/backend-go and pegasus.x/backend.

## 2026-09-16T13:14:29Z

End-to-end implementation of Supplier Sign-Up, Sign-In, Non-Bypassable Phased Onboarding (Product Catalog with MXIK/Tiyins, Cash & Global Pay Corporate Card Gateway), and Post-Onboarding Warehouse/Fleet Management in `pegasus.x`.

Working directory: /Users/shakhzod/Desktop/V.O.I.D
Integrity mode: development

## Requirements

### R1. Minimal Supplier Sign-Up & Sign-In with STIR Legal Deduplication
- Implement `POST /v1/auth/supplier/register` accepting:
  - `company_name` (Legal company name)
  - `tax_id` (Uzbekistan 9-digit STIR/INN)
  - `phone` (`+998XXXXXXXXX`)
  - `password` (bcrypt hashed)
- Enforce STIR uniqueness in PostgreSQL; return HTTP 409 Conflict if already registered.
- Set initial state: `onboarding_status = 'PENDING'`.
- Implement `POST /v1/auth/supplier/login`:
  - Returns JWT claims and current `onboarding_status` with `next_step: "/onboarding/products"`.

### R2. Non-Bypassable Onboarding Gate & Phased Wizard
- Build middleware `RequireSupplierOnboardingCompleted`:
  - If `onboarding_status != 'COMPLETED'`, block all operational supplier endpoints with HTTP 428 Precondition Required (`onboarding_incomplete`).
  - Whitelist `/v1/auth/*` and `/v1/supplier/onboarding/*`.
- Step 1: Product Catalog Management:
  - `POST /v1/supplier/onboarding/products` (Add, Edit, Delete).
  - Enforce: Name, unique EAN-13 barcode, 17-digit statutory MXIK code, package code, units_per_case, unit_price_tiyin (64-bit integer), 12% VAT.
  - Gate requirement: At least 1 active product required to complete step.
- Step 2: Payment Gateway Configuration:
  - `POST /v1/supplier/onboarding/payment`.
  - Cash enabled by default.
  - Global Pay (`GLOBAL_PAY`) corporate card gateway setup with service ID, secret key, and corporate card BIN validation (B2B corporate cards only).
- Step 3: Complete Onboarding:
  - `POST /v1/supplier/onboarding/complete`: Transitions `onboarding_status` to `'COMPLETED'`, unblocks the gate, and emits outbox/WebSocket event.

### R3. Warehouse & Fleet Management Hub
- `POST/GET/PUT/DELETE /v1/supplier/warehouses`:
  - Mandatory `latitude` and `longitude` (`DOUBLE PRECISION`) for retailer proximity and driver dispatch routing.
  - Invalidate Redis proximity cache and emit `warehouse.relocated` on coordinate updates.
  - Block warehouse deletion with HTTP 409 Conflict if on-hand stock > 0 or active orders exist.
- Fleet & Dock Logistics:
  - `POST/GET /v1/supplier/warehouses/{id}/trucks` (license plate, capacity kg/m³, fuel type).
  - `POST/GET /v1/supplier/warehouses/{id}/payloaders` (name, phone, warehouse_id).

### R4. Zero Mock Data & Pure PostgreSQL 16 Persistence
- Purge `MemoryRepository` and hardcoded seeds from `pegasus.x/backend/internal/supplier/repository.go`.
- Create PostgreSQL migration `069_supplier_onboarding_and_globalpay.sql` for all new columns and tables.
- All endpoints query and write directly to PostgreSQL 16 via `pgxpool`.
- Full test suite passing with `go test -v -race`.

## Acceptance Criteria

### Verification & Quality Gates
- [ ] Automated migration `069_supplier_onboarding_and_globalpay.sql` applies cleanly to PostgreSQL 16.
- [ ] Sign-up with duplicate STIR returns HTTP 409 Conflict; valid registration creates records with `onboarding_status = 'PENDING'`.
- [ ] Calling operational supplier endpoints with uncompleted onboarding returns HTTP 428 Precondition Required.
- [ ] Product creation rejects prices not in 64-bit integer tiyins or invalid 17-digit MXIK codes.
- [ ] Payment gateway setup correctly configures Cash and Global Pay (`GLOBAL_PAY`) with corporate card attributes.
- [ ] Adding a warehouse requires valid GPS coordinates; deleting a warehouse with stock returns HTTP 409 Conflict.
- [ ] Zero mock data or fallback memory repositories remain in the supplier package.
- [ ] `go test -v -race ./...` passes across `pegasus.x/backend`.

