# Gemini — V.O.I.D.

Honesty: `.github/gemini-instructions.md` and `.github/instructions/honest-code-gate.instructions.md`.
Final goal: `.agents/memory/GOAL.md` — `GLOBAL_SCALE_PROGRAM.md` + `GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`.

# STRICT TWO-SYSTEM ARCHITECTURAL BOUNDARY (DO NOT MERGE OR CROSS-POLLUTE)
Within `V.O.I.D`, there are TWO distinct parallel systems:
1. **`pegasusX/`**: Global Enterprise Multi-Tenant (Google Cloud Spanner, Apache Kafka, Go, Next.js, Native Mobile).
2. **`pegasus.x/`**: Sovereign Lean Single-Tenant (PostgreSQL 16 + Redis 7, Servercore Tashkent, Tauri v2 Desktops, Telegram Bot/MiniApp, Go Chi, Native Driver).

**Rules**:
- Never cross-contaminate codebases or database engines (No Spanner in `pegasus.x`, no single-tenant PG in `pegasusX`).
- Cross-scan both codebases to identify missing domain capabilities and sync them logically into `pegasus.x`.
- Primary active focus: Full Fleet & Driver Management lifecycle (Vehicles, Drivers, Dynamic Daily Shift Pairing, Mid-Shift Swapping, DVIR Pre-Trip Inspections) ported into `pegasus.x`.


Retrieval + memory (all agents): read `.agents/memory/WORKSPACE.md`, then

```bash
python3 .agents/skills/graph-retrieval-memory/scripts/graph_retrieve.py -q "<topic>" --hops 2
```

Open the returned paths. Graph is not status. Persist verified facts only to `.agents/memory/WORKSPACE.md`.
See `.agents/skills/graph-retrieval-memory/references/always-on.md`.

Two-Tier Verification Gate (MANDATORY on every edit — Bazel/Kythe CodeGraph + Targeted Raw Reading):
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

