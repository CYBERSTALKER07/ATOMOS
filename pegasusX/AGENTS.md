# HONESTY OVERRIDE (absolute)

**Final goal:** `../.agents/memory/GOAL.md` — `docs/GLOBAL_SCALE_PROGRAM.md` + `docs/GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`. Register + local-first same-market matching + pack as law + Class A per `SupplierId`. No seed, no UZ fork.

This folder is the living product. Do not plan or ship from `../pegasus/` or frozen `.docx`.

- Code opened this session is the only status SoT.
- No wired / done / production-ready / cloud-ready without file:line.
- Cloud / API / infra: YES only if backend + shipped role-row apps + data flow are REAL and tests passed after re-reading edits.
- After a plan lands: re-read, re-trace, test. If it failed, replan.

See `.agents/AGENTS.md` and `.github/instructions/honest-code-gate.instructions.md`.

**Retrieval (all IDEs):** `../.agents/memory/WORKSPACE.md` + `python3 ../.agents/skills/graph-retrieval-memory/scripts/graph_retrieve.py -q "<topic>"`. Graph is a routing index. Persist verified facts only.

**Two-Tier Verification Gate (MANDATORY on every edit — Bazel/Kythe CodeGraph + Targeted Raw Reading):**
1. **Tier 1 — Bazel/Kythe Dynamic CodeGraph (Global Radar):** ALWAYS run BEFORE touching code to discover the mathematical blast radius, reverse dependencies, and taint violations:
   - Blast radius & Kythe callers: `python3 scripts/advanced_codegraph_analyzer.py --blast-radius <symbol> --depth 3 --json` or `python3 scripts/kythe_semantic_adapter.py --xref <symbol> --json`
   - Bazel affected test targets: `python3 scripts/bazel_target_graph.py --query-rdeps <target>`
   - Full ecosystem compiler-grade audit: `make codegraph-advanced-audit`
2. **Tier 2 — Targeted Raw Reading (Local Microscope):** NEVER rely on the graph alone. Open and raw-read the exact files identified in Tier 1:
   - Verify runtime conditionals (`if err != nil`, guard clauses, feature flags).
   - Verify transaction boundaries (`spanner.ReadWriteTransaction` closures, double-entry ledger invariants, atomic outbox pairing).
   - Re-read every edit after writing to ensure zero contract drift or unhandled side-effects.
See `../.agents/skills/codegraph-deep-audit/SKILL.md`.


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
