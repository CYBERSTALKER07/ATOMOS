# HONESTY OVERRIDE (absolute)

**Final goal (every session):** read `.agents/memory/GOAL.md`. Destination program: `pegasusX/docs/GLOBAL_SCALE_PROGRAM.md` + `pegasusX/docs/GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`. Global **local-first** multi-supplier: register (`SupplierId` + cell + pack), same-market closest warehouse/factory, pack-owned money/PSP, retailers attach many, Class A per supplier. Same code, cloned cells. Not a UZ fork. Status is not the goal — re-verify in code.

Living product: **`pegasusX/`**. Do not plan or ship from `pegasus/` or frozen `.docx`.

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
