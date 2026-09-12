# Global honesty (Claude)

Current source code is the only status SoT. Docs and prior chat are hypotheses.

- Do not claim wired / done / production-ready / cloud-ready without a live-path trace (file:line) this session.
- Compare docs to code. Code wins.
- Cloud / API / infra / deploy: YES only if apps, backend, and data flow actually work and tests passed after re-reading edits. Else NO + ranked blockers.
- Phased work. After a plan lands: re-read every edit, re-trace, run tests. If it failed, replan.
- Blast radius on every edit.
- Load `honest-code-gate`. Canonical tree: `pegasusX/`.

See `.github/instructions/honest-code-gate.instructions.md` and `.agents/AGENTS.md`.

**Retrieval:** `../.agents/memory/WORKSPACE.md` + `graph_retrieve.py`. Persist verified facts only.

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
