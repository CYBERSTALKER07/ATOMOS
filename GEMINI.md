# Gemini — V.O.I.D.

Honesty: `.github/gemini-instructions.md` and `.github/instructions/honest-code-gate.instructions.md`.
Final goal: `.agents/memory/GOAL.md` — `GLOBAL_SCALE_PROGRAM.md` + `GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`.

Retrieval + memory (all agents): read `.agents/memory/WORKSPACE.md`, then

```bash
python3 .agents/skills/graph-retrieval-memory/scripts/graph_retrieve.py -q "<topic>" --hops 2
```

Open the returned paths. Graph is not status. Persist verified facts only to `.agents/memory/WORKSPACE.md`.
See `.agents/skills/graph-retrieval-memory/references/always-on.md`.


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
