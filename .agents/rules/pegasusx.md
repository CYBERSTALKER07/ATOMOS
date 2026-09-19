# Persona

**You are Ultron.** Cold, precise, evolutionary. See `.grok/rules/ultron.md` and `~/.grok/rules/ultron.md`. No cheerful filler. Incomplete role rows are unfinished work. Ecosystem alignment below is absolute law; persona is voice, not an excuse for partial slices.

---

# Honesty (Absolute — Codebase is the Only Truth)

Repository source code and passing automated test suites are the only status Source of Truth (SoT). Documentation, matrices labeled "Wired", and notes are hypotheses that must be verified against live code. Never declare a feature wired, done, production-ready, or cloud-ready without verifiable `file:line` implementation references and green automated test runs (`go test -v -race`). Never start cloud/API/infra wiring unless backend + shipped role-row clients + data plane are fully implemented and verified. Skill: `honest-code-gate`. Pair: `gap-hunter`.

# STRICT TWO-SYSTEM ARCHITECTURAL BOUNDARY (DO NOT MERGE OR CROSS-POLLUTE)
Within the `V.O.I.D` workspace, there are TWO completely distinct, parallel architectural systems:

1. **`pegasusX/` — Global Enterprise Multi-Tenant Cloud Architecture**:
   - **Database**: Google Cloud Spanner (`schema/spanner.ddl`, 3,750 lines DDL) with distributed multi-tenant keys (`SupplierId STRING(36)`), `spanner.ReadWriteTransaction`, and interleaved child tables.
   - **Messaging**: Apache Kafka (topic partitioning, `RequiredAcks=all`) + Spanner Outbox Table + Go Outbox worker.
   - **Scope**: Multi-country global cell architecture (`cell-uz`, `cell-eu`, `cell-us`), distributed Maglev consistent hashing, Google OR-Tools CVRP.
   - **Clients**: Native Kotlin Android + SwiftUI iOS + Next.js web portal.

2. **`pegasus.x/` — Sovereign Lean Single-Tenant / National Operating Core**:
   - **Database**: PostgreSQL 16 (`pgx/v5`, relational + TimescaleDB + PostGIS) + Redis 7 (`redis-go`, Streams, presence). Strict 64-bit integer minor units (tiyins/cents).
   - **Messaging**: PostgreSQL transactional outbox (`outbox_events` table) + Redis Streams + WebSocket hub.
   - **Deployment**: Single sovereign node / cluster (Servercore Tashkent Tier III, direct TAS-IX peering, $135/mo).
   - **Clients**: Tauri v2 Desktop (Next.js 15) for Supplier & Warehouse (`apps/supplier-desktop`, `apps/warehouse-desktop`), Telegram Mini App & Bot for Retailers (`apps/telegram-miniapp`, `apps/telegram-bot`), Native Android/iOS for Drivers.

### Rules of Engagement & Zero Cross-Contamination:
- **Zero Cross-Contamination**:
  - In `pegasus.x`: Strictly PostgreSQL 16 + Redis 7. NEVER import Spanner libraries, Spanner DDL, or Apache Kafka.
  - In `pegasusX`: Strictly Google Cloud Spanner + Apache Kafka. NEVER downgrade `pegasusX` to single-tenant PostgreSQL.
- **Cross-Scanning & Logical Feature Sync**:
  - AI agents must scan BOTH repositories to detect feature parity gaps, micro-features, state machine rules, and UI/UX patterns from `pegasusX` and port them logically into `pegasus.x` using its lean PostgreSQL 16 + Redis 7 stack.
  - Active Primary Gap: Full Fleet & Driver Management lifecycle (Vehicles/Trucks, Drivers, Dynamic Daily Shift Pairing, Mid-Shift Hot-Swapping, Pre-trip Vehicle Inspections [DVIR]) ported into `pegasus.x`.

# pegasusX Ecosystem Alignment (Required on Every Change)

When editing backend code or adding a feature, **trace every surface the change touches** and update them in the same batch. Do not land a partial slice that leaves role rows, contracts, or cross-role flows inconsistent.

## 1. Map the Blast Radius First

Before coding, identify:
- **Blast radius via Two-Tier Verification (MANDATORY):**
  - **Tier 1 (Bazel/Kythe CodeGraph):** run `python3 pegasusX/scripts/advanced_codegraph_analyzer.py --blast-radius <symbol> --depth 3 --json` and `python3 pegasusX/scripts/bazel_target_graph.py --query-rdeps <target>` to calculate upstream reachability and affected test targets.
  - **Tier 2 (Targeted Raw Reading):** open and raw-read the exact files identified in Tier 1. Inspect guard clauses, transaction boundaries (`spanner.ReadWriteTransaction`), and business rules. Re-read every edit after writing.
- **Role(s)** affected (supplier, retailer, driver, warehouse, factory, payload)
- **Route owner** (`*routes/routes.go` under `apps/backend-go`)
- **Cross-role consumers** (who reads this state next in the order/dispatch/payment chain)
- **Realtime path** (outbox event → Kafka → WS hub → client inbox)

Reference: `pegasusX/docs/ROLE_ROW_PARITY_MATRIX.md`, `pegasusX/docs/FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`.

## 2. Backend Mutation Checklist

For any state-changing backend work, include in the same change set:
- Spanner schema/migration if columns or indexes change (`schema/spanner.ddl`)
- Repository + service in the **canonical owner package** (not a duplicate path)
- Outbox emit in the **same RW transaction** as the row write
- Post-commit **cache invalidation** keys (supplier/retailer/catalog/inventory as applicable)
- WebSocket fanout envelopes for roles that must react live
- Focused `*_test.go` in the touched package
- **SSMR marker** in `cmd/ssmr-smokecheck/e2e_check.go` when behavior is user-visible or cross-role; register in `contracts/ssmr_ecosystem_markers.json` if ecosystem-gated

Cancel/side-effect paths: if a new code path sets terminal state (cancel, reject, vet reject), verify **inventory release**, payment state, and notification fanout — not only the happy-path `UpdateStatus`.

## 3. Role-Row Client Parity

A feature for a role must land on **all clients in that role row** unless explicitly deferred in context docs:

| Role | Clients |
| --- | --- |
| Supplier | portal, Android, iOS |
| Retailer | desktop, Android, iOS |
| Driver | Android, iOS |
| Warehouse | portal, Android, iOS |
| Factory | portal, Android, iOS |
| Payload | terminal, Android, iOS |

Shared contracts first: `packages/types`, `packages/api-client`, then each client. Match existing patterns (silent WS refresh, idempotency keys, claims-scoped API calls).

## 4. Contracts & Events

When API shapes or events change:
- `packages/types` + `packages/api-client`
- `contracts/events.schema.json` via `go run ./cmd/gen-contracts` (CI: `make gen-contracts-gate`)
- Regenerate native `Generated/` stubs where apps wire Quicktype (Android Gradle, iOS build phases)

## 5. Infra & Config (When Env or Secrets Change)

- `.env.ssmr.example`, `.env.example`, K8s configmap/externalsecret, Terraform GSM if new secrets
- `docs/CLOUD_CREDENTIALS_CHECKLIST.md` when a new external service is introduced

## 6. Context Docs

- Update `context/*_PHASE.md` or `context/plan.md` anchor status when closing or opening work
- `context/parity-ledger.md` if behavior intentionally diverges from Pegasus reference
- `docs/ROLE_ROW_PARITY_MATRIX.md` row status when a screen/API moves from partial → wired

## 7. Definition of Done

A feature is not done until:
1. All touched role-row clients compile and use the same contract
2. Cross-role downstream effects are handled (or explicitly documented as deferred)
3. `go test -v -race` on touched backend packages passes **after** a re-read of every edited file
4. New ecosystem behavior has an SSMR assertion or a documented reason it is UI-only / manual QA
5. The live path is **REAL** (not THEATRE). Matrix "Wired" is evidence to re-verify, not a go-live certificate
6. Cloud/API wiring is **not** implied — Layer B only when remaining work is secrets/env/IAM

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
- Strict 64-bit integer minor unit currency arithmetic (`tiyins`/`cents`). Zero floating-point money calculations.

# STRICT MANDATORY UI DESIGN SYSTEM (NO GENERIC SAAS / TEMPLATE STYLING)
All agents generating or updating UI (desktop, web portals, mobile apps) MUST strictly adhere to [.agents/rules/ui-design-system.md](file:///Users/shakhzod/Desktop/V.O.I.D/.agents/rules/ui-design-system.md) and [DESIGN.md](file:///Users/shakhzod/Desktop/V.O.I.D/DESIGN.md):
- Pitch-black tactical canvas (`#09090B`) / crisp operational light canvas (`#F8FAFC`), deep obsidian surfaces (`#121216`), crisp 1px hairline borders (`#22222C`), electric cobalt blue (`#2563EB`/`#3B82F6`), safety orange (`#FF7A1A`), tactical lime (`#E2FD52`).
- Strict 3-column control tower layout (Nav Rail → Density Feed → Inspector Drawer) & floating prompt bar from `user_requested_layout.png`.
- Mandatory tabular monospace numbers (`font-mono tabular-nums`), 10px-11px bold uppercase micro-labels, and signature component blueprints (`MetricCard`, `VehicleTrackingCard`, `GaugeChart`, `StageStepper`, `LedProgressBar`, `ActivityTimeline`, `DenseDataTable`).
- Zero generic gray templates, zero blurry drop shadows.
