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
