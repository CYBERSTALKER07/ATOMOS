# Cursor CLI

User-level skill: `~/.cursor/skills/graph-retrieval-memory`  
User rule: `~/.cursor/rules/graph-retrieval-memory.mdc` (`alwaysApply: true`)  
Slash command: `/graph-retrieve` → `~/.cursor/commands/graph-retrieve.md` (also `<repo>/.cursor/commands/`)  
User hook: `~/.cursor/hooks.json` `sessionStart` → `./hooks/graph-retrieval-session.sh`  
Project hook: `<repo>/.cursor/hooks.json` `sessionStart` → `.cursor/hooks/graph-retrieval-session.sh`

`sessionStart` prints JSON from `scripts/cursor_cli_memory.py`: `env` (`VOID_REPO`, `GRAPH_RETRIEVE`, `VOID_MEMORY`, `VOID_GOAL`) plus a **bounded** `additional_context` excerpt of `GOAL.md` and the latest `WORKSPACE.md` verified section(s). Excerpt is hypothesis. Open the files. Code wins.

This CLI session’s workspace may be `$HOME`, not V.O.I.D. Do **not** run
`.agents/skills/...` from home. Use the skill path (walker finds the graph via `__file__`):

```bash
python3 "$HOME/.cursor/skills/graph-retrieval-memory/scripts/graph_retrieve.py" -q "<topic>" --hops 2
python3 "$HOME/.cursor/skills/graph-retrieval-memory/scripts/graph_retrieve.py" -q "<topic>" --json
python3 "$HOME/.cursor/skills/graph-retrieval-memory/scripts/graph_retrieve_test.py"
python3 "$HOME/.cursor/skills/graph-retrieval-memory/scripts/cursor_cli_memory_test.py"
```

Shared memory (all IDEs):

```
$HOME/Desktop/V.O.I.D/.agents/memory/GOAL.md
$HOME/Desktop/V.O.I.D/.agents/memory/WORKSPACE.md
```

Prefer `agent --workspace "$HOME/Desktop/V.O.I.D"` so project `AGENTS.md` loads.
Hits are paths. Open them. Code wins.


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
