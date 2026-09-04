# Graph retrieval + shared memory (all agents / IDEs)

Applies to Grok, Claude Code, Cursor, Copilot, Codex, Gemini, Windsurf, Continue, Cline, and any other agent in this repo.

**Every new session:** open `.agents/memory/GOAL.md` first. That is the final goal. Chat history is gone; this file is not.

## Planes

| Plane | Path | Role |
|-------|------|------|
| **Final goal** | `.agents/memory/GOAL.md` | North star — load first, every session |
| **Shared memory** | `.agents/memory/WORKSPACE.md` | Durable facts **every** IDE can read/write |
| **Grok index** | `~/.grok/memory/` (FTS5+vec0) | Extra search if Grok memory is on |
| **Graph** | `pegasusX/context/architecture-graph.json` | Routing index only (`generatedAt` is null) |
| **Code** | live grep/read | **SoT** |

Do not add Pinecone/Chroma/LangChain RAG. Do not treat docs or graph `runtimeNotes` as status.

## Loop

```
enhance → retrieve (WORKSPACE.md ∥ graph_retrieve.py ∥ grep) → verify file:line → answer → persist verified only
```

```bash
# Cursor CLI / any cwd:
python3 "$HOME/.cursor/skills/graph-retrieval-memory/scripts/graph_retrieve.py" -q "<topic>" --hops 2
# cwd = Desktop/V.O.I.D only:
python3 .agents/skills/graph-retrieval-memory/scripts/graph_retrieve.py -q "<topic>" --hops 2
```

Memory when CLI workspace is `$HOME`: `$HOME/Desktop/V.O.I.D/.agents/memory/{GOAL,WORKSPACE}.md`.

Cursor CLI `sessionStart` injects a **bounded** GOAL + latest WORKSPACE excerpt (`scripts/cursor_cli_memory.py`). That excerpt is not status. Open the files.

Then **open the returned paths**. Graph hits are not evidence.

## Persist

After a live-path verify, append to `.agents/memory/WORKSPACE.md`:

```
## Verified YYYY-MM-DD
- <one sentence> — `path:line`
```

If Grok memory exists, copy the same bullet there so `/memory` can search it. Never persist “wired/done/cloud-ready” without file:line.

## Honesty

Load `honest-code-gate`. Code wins. Living tree: `pegasusX/`.


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
