---
name: graph-retrieval-memory
description: >-
  Graph-based retrieval loop with shared WORKSPACE.md memory for every
  IDE/agent including Cursor CLI (`agent`). Use for RAG, retrieve, graph
  retrieval, /graph-retrieve, agent memory, remember, recall, what do we know,
  or status/wiring questions. Not a new vector database. From any cwd use
  ~/.cursor/skills/graph-retrieval-memory/scripts/graph_retrieve.py.
---

# Graph retrieval + actual memory

Do **not** stand up LangChain/Pinecone/Chroma. Every IDE uses the same four planes:

| Plane | What it is | Status? |
|-------|------------|---------|
| **Shared memory** | `.agents/memory/WORKSPACE.md` (all agents) | Hypothesis until re-read |
| **Grok index** | `~/.grok/memory/` FTS5+vec0 (Grok only) | Hypothesis until re-read |
| **Architecture graph** | `pegasusX/context/architecture-graph.json` (88 nodes / 160 edges) | **Routing index only** (`generatedAt: null`, notes stale) |
| **Live code** | grep / read / codebase index | **SoT** |

Always-on text (copied into Copilot/Cursor/Claude/Gemini/Windsurf/Cline/Codex): `references/always-on.md`.

## When this applies

- RAG / graph retrieval / “use memory” / “what did we decide”
- Status, wiring, “where does X live”, blast-radius, role-row fan-out
- After a verified finding you want to **keep** across sessions

## The loop (mandatory)

```
enhance → retrieve (memory ∥ graph ∥ code) → verify (code wins) → answer → persist verified only
```

### 1. Enhance

Name: role, package, route prefix, table, event, client. Example: `fiscal retry` → `order.HandleFiscalRetry`, `FISCAL_PROVIDER`, `MarketPack.fiscal_adapter`.

### 2. Retrieve (parallel)

```bash
# Graph neighborhood (paths to open — not verdicts)
# Cursor CLI / any cwd (home workspace is not V.O.I.D):
python3 "$HOME/.cursor/skills/graph-retrieval-memory/scripts/graph_retrieve.py" -q "<enhanced>" --hops 2
# Only when cwd is Desktop/V.O.I.D:
python3 .agents/skills/graph-retrieval-memory/scripts/graph_retrieve.py -q "<enhanced>" --hops 2

# Shared memory (every IDE)
#   $HOME/Desktop/V.O.I.D/.agents/memory/WORKSPACE.md
#   $HOME/Desktop/V.O.I.D/.agents/memory/GOAL.md
# Grok extra (if [memory] enabled):
#   ~/.grok/memory/MEMORY.md
```

Cursor CLI details: `references/cursor-cli.md`. Slash: `/graph-retrieve`. First-turn excerpt: `scripts/cursor_cli_memory.py` via `sessionStart` (hypothesis until re-read).

Then grep/read the **paths the graph returned**. Never stop at the JSON.

### 3. Verify

Load `honest-code-gate`. A graph node path that you did not open this session is not evidence. Docs vs code: **code wins**. Stale `runtimeNotes` in the graph are N10 drift.

### 4. Answer

Use the honest skeleton for status questions. Cite `file:line`.

### 5. Persist (actual memory)

Only after verify. Append a dated bullet to **`.agents/memory/WORKSPACE.md` first** (every IDE sees it):

```
## Verified <YYYY-MM-DD>
- <fact> — `path:line` (opened this session)
```

If Grok memory tools exist, copy the same bullet so `/memory` can search it. Do **not** write unverified “wired/done/cloud-ready”.

Grok first-turn injection needs a session started **after** `[memory] enabled`. Other IDEs only have WORKSPACE.md — that is enough.

## Anti-patterns

- Embedding the whole `docs/` tree into a side vector store
- Treating `FEATURES_BY_APP_ROLE` or graph `runtimeNotes` as live status
- Writing chat summaries into MEMORY.md without file:line
- Claiming first-turn injection in a session that started before memory was enabled (need `/new`)
