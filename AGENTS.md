# HONESTY OVERRIDE (absolute)

**Final goal (every session):** read `.agents/memory/GOAL.md`. Destination program: `pegasusX/docs/GLOBAL_SCALE_PROGRAM.md` + `pegasusX/docs/GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`. Global **local-first** multi-supplier: register (`SupplierId` + cell + pack), same-market closest warehouse/factory, pack-owned money/PSP, retailers attach many, Class A per supplier. Same code, cloned cells. Not a UZ fork. Status is not the goal — re-verify in code.

Living product: **`pegasusX/`**. Do not plan or ship from `pegasus/` or frozen `.docx`.

# STRICT TWO-SYSTEM ARCHITECTURAL BOUNDARY (DO NOT MERGE OR CROSS-POLLUTE)
Within the `V.O.I.D` workspace, there are TWO completely distinct, parallel architectural systems:

1. **`pegasusX/` — Global Enterprise Multi-Tenant Cloud Architecture**:
   - **Database**: Google Cloud Spanner (`schema/spanner.ddl`, 3,750 lines DDL) with distributed multi-tenant keys (`SupplierId STRING(36)`), `spanner.ReadWriteTransaction`, interleaved child tables.
   - **Messaging**: Apache Kafka + Spanner Outbox Table + Go Outbox worker.
   - **Scope**: Multi-country global cell architecture (`cell-uz`, `cell-eu`, `cell-us`), distributed Maglev consistent hashing, Google OR-Tools CVRP.
   - **Clients**: Native Kotlin Android + SwiftUI iOS + Next.js web portal.

2. **`pegasus.x/` — Sovereign Lean Single-Tenant / National Operating Core**:
   - **Database**: PostgreSQL 16 (`pgx/v5`, relational + TimescaleDB + PostGIS) + Redis 7 (`redis-go`, Streams, presence). Strict 64-bit integer minor units (tiyins/cents).
   - **Messaging**: PostgreSQL transactional outbox + Redis Streams + WebSocket hub.
   - **Deployment**: Single sovereign node / cluster (Servercore Tashkent Tier III, direct TAS-IX peering, $135/mo).
   - **Clients**: Tauri v2 Desktop (Next.js 15) for Supplier & Warehouse (`apps/supplier-desktop`, `apps/warehouse-desktop`), Telegram Mini App & Bot for Retailers (`apps/telegram-miniapp`, `apps/telegram-bot`), Native Android/iOS for Drivers.

### Rules of Engagement for AI Agents:
- **Zero Cross-Contamination**: NEVER import Spanner libraries, Spanner DDL, or Kafka into `pegasus.x`. NEVER downgrade `pegasusX` to single-tenant PostgreSQL.
- **Cross-Scanning & Logical Feature Sync**: AI agents must scan BOTH repositories to detect feature parity gaps, micro-features, state machine rules, and UI/UX patterns from `pegasusX` and port them logically into `pegasus.x` using its lean PostgreSQL 16 + Redis 7 stack.
- **Identified Critical Gap**: Fleet Management (Vehicles/Trucks), Driver Onboarding & Management, Dynamic Driver-Vehicle Daily Shift Assignments, Mid-Shift Hot-Swapping, and Pre-trip Vehicle Inspections (DVIR) exist in `pegasusX` domain logic but were missing from `pegasus.x`. These must be planned and synchronized into `pegasus.x`.


- Code opened this session is the only status SoT. Docs, matrices labeled **Wired**, Copilot runtime notes, and prior chat are hypotheses.
- Forbidden without file:line: wired, done, production-ready, cloud-ready, we can start connecting cloud.
- Cloud / API / infra: YES only if backend + shipped role-row apps + data flow are REAL and tests passed after re-reading edits. Else NO + ranked blockers.
- After a plan lands: re-read every edit, re-trace, run tests. If it failed, replan.
- Blast radius on every edit. Load `honest-code-gate`.

Full text: `.github/instructions/honest-code-gate.instructions.md`  
pegasusX agent file: `pegasusX/.agents/AGENTS.md`  
Do not treat `.AGENTS.MD` runtime notes as status.

**Retrieval (all IDEs):** read `.agents/memory/WORKSPACE.md`, run `python3 .agents/skills/graph-retrieval-memory/scripts/graph_retrieve.py -q "<topic>"`, then open live paths. Persist verified facts only. See `.agents/skills/graph-retrieval-memory/references/always-on.md`.

**Cursor CLI:** if cwd is not this repo, use `python3 "$HOME/.cursor/skills/graph-retrieval-memory/scripts/graph_retrieve.py" -q "<topic>" --hops 2` and read `$HOME/Desktop/V.O.I.D/.agents/memory/{GOAL,WORKSPACE}.md`. Prefer `agent --workspace "$HOME/Desktop/V.O.I.D"`. `sessionStart` injects a bounded GOAL/WORKSPACE excerpt (hypothesis). Hits are paths, not status. `/graph-retrieve`.

**Two-Tier Verification Gate (MANDATORY on every edit — Bazel/Kythe CodeGraph + Targeted Raw Reading):**
1. **Tier 1 — Bazel/Kythe Dynamic CodeGraph (Global Radar):** ALWAYS run BEFORE touching code to discover the mathematical blast radius, reverse dependencies, and taint violations:
   - Blast radius & Kythe callers: `python3 pegasusX/scripts/advanced_codegraph_analyzer.py --blast-radius <symbol> --depth 3 --json` or `python3 pegasusX/scripts/kythe_semantic_adapter.py --xref <symbol> --json`
   - Bazel affected test targets: `python3 pegasusX/scripts/bazel_target_graph.py --query-rdeps <target>`
   - Full ecosystem compiler-grade audit: `make codegraph-advanced-audit`
2. **Tier 2 — Targeted Raw Reading (Local Microscope):** NEVER rely on the graph alone. Open and raw-read the exact files identified in Tier 1:
   - Verify runtime conditionals (`if err != nil`, guard clauses, feature flags).
   - Verify transaction boundaries (`spanner.ReadWriteTransaction` closures, double-entry ledger invariants, atomic outbox pairing).
   - Re-read every edit after writing to ensure zero contract drift or unhandled side-effects.
See `.agents/skills/codegraph-deep-audit/SKILL.md`.


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

# STRICT MANDATORY UI DESIGN SYSTEM (NO GENERIC SAAS / TEMPLATE STYLING)
All agents generating or updating UI (desktop, web portals, mobile apps) MUST strictly adhere to [.agents/rules/ui-design-system.md](file:///Users/shakhzod/Desktop/V.O.I.D/.agents/rules/ui-design-system.md) and [DESIGN.md](file:///Users/shakhzod/Desktop/V.O.I.D/DESIGN.md):
- Pitch-black tactical canvas (`#09090B`) / crisp operational light canvas (`#F8FAFC`), deep obsidian surfaces (`#121216`), crisp 1px hairline borders (`#22222C`), electric cobalt blue (`#2563EB`/`#3B82F6`), safety orange (`#FF7A1A`), tactical lime (`#E2FD52`).
- Strict 3-column control tower layout (Nav Rail → Density Feed → Inspector Drawer) & floating prompt bar from `user_requested_layout.png`.
- Mandatory tabular monospace numbers (`font-mono tabular-nums`), 10px-11px bold uppercase micro-labels, and signature component blueprints (`MetricCard`, `VehicleTrackingCard`, `GaugeChart`, `StageStepper`, `LedProgressBar`, `ActivityTimeline`, `DenseDataTable`).
- Zero generic gray templates, zero blurry drop shadows.

