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
