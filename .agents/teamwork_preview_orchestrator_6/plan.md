# Plan: Dual-System Architecture & Feature Parity Analysis

## Objective
Produce a definitive, production-grade architectural analysis and feature parity comparison between `pegasusX` (Global Enterprise Multi-Tenant) and `pegasus.x` (Sovereign Lean Single-Tenant), complete with detailed Mermaid diagrams, comprehensive tabular parity matrices, and an exhaustive deep dive into the Fleet Management lifecycle gap.

## Acceptance Criteria
- [ ] Comprehensive Architectural Summary detailing components, data flows, deployment, and strict architectural boundaries.
- [ ] At least two detailed Mermaid architecture diagrams (one for pegasusX, one for pegasus.x).
- [ ] Exhaustive tabular Parity Matrix explicitly comparing features across both codebases (Order lifecycle, Catalog, Inventory, Pricing, Finance/Ledger, Dispatch, Telemetry/Tracking, Auth & Tenancy, Client Apps).
- [ ] Dedicated section detailing the Fleet Management lifecycle gap (Vehicles/Trucks, Driver Onboarding & Management, Dynamic Driver-Vehicle Daily Shift Assignments, Mid-Shift Hot-Swapping, and Pre-trip Vehicle Inspections / DVIR).
- [ ] Verification gate passes with independent reviewer approvals.

## Execution Phases

### Phase 1: Parallel Deep Exploration (5 Explorers)
- **Explorer 1 (`explorer_pegasusx_core`)**:
  - Focus: `pegasusX/` backend, Google Cloud Spanner schema (`schema/spanner.ddl`), ReadWrite transactions, Kafka outbox, Go backend architecture (`apps/backend-go/`), Maglev consistent hashing, Google OR-Tools CVRP.
- **Explorer 2 (`explorer_pegasusx_clients`)**:
  - Focus: `pegasusX/` client applications (Web Next.js, Native Kotlin Android, SwiftUI iOS) and role-row contracts (`packages/types`, `packages/api-client`, events schema).
- **Explorer 3 (`explorer_pegasusdotx_core`)**:
  - Focus: `pegasus.x/` backend, PostgreSQL 16 schema/migrations, TimescaleDB, PostGIS, Redis 7 (Streams, cache, presence), Go Chi services, PG transactional outbox, Servercore Tashkent single sovereign node.
- **Explorer 4 (`explorer_pegasusdotx_clients`)**:
  - Focus: `pegasus.x/` client applications: Tauri v2 Desktop (Next.js 15) for Supplier/Warehouse, Telegram Mini App & Bot for Retailers, Native Driver apps.
- **Explorer 5 (`explorer_fleet_lifecycle`)**:
  - Focus: Fleet Management domain in `pegasusX` vs `pegasus.x`: Trucks/Vehicles, Drivers, Shift pairing, Hot-swap reassignments, DVIR inspections. State machines, schemas, and missing capabilities.

### Phase 2: Synthesis & Deliverable Generation (Worker)
- **Worker (`worker_synthesis`)**:
  - Aggregate findings from all 5 Explorer reports.
  - Author `/Users/shakhzod/Desktop/V.O.I.D/DUAL_SYSTEM_ARCHITECTURE_AND_PARITY.md`.
  - Embed rich Mermaid architecture & data flow diagrams.
  - Construct comprehensive tabular parity matrix and dedicated Fleet Management deep dive.

### Phase 3: Independent Review & Verification Gate (2 Reviewers)
- **Reviewer 1 (`reviewer_architecture_parity`)**:
  - Audit architectural accuracy, Mermaid syntax, Spanner vs PG boundaries, data flow validity.
- **Reviewer 2 (`reviewer_gap_fleet`)**:
  - Audit Parity Matrix completeness, verify every functional module, check depth of Fleet Management lifecycle analysis.

### Phase 4: Sign-off & Delivery
- Validate GATE_STATUS.md.
- Send final completion message to sentinel parent.
