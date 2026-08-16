# Cursor CLI

User-level skill: `~/.cursor/skills/graph-retrieval-memory`  
User rule: `~/.cursor/rules/graph-retrieval-memory.mdc` (`alwaysApply: true`)  
Slash command: `/graph-retrieve` → `~/.cursor/commands/graph-retrieve.md`

This CLI session’s workspace may be `$HOME`, not V.O.I.D. Do **not** run
`.agents/skills/...` from home. Use the skill path (walker finds the graph via `__file__`):

```bash
python3 "$HOME/.cursor/skills/graph-retrieval-memory/scripts/graph_retrieve.py" -q "<topic>" --hops 2
python3 "$HOME/.cursor/skills/graph-retrieval-memory/scripts/graph_retrieve.py" -q "<topic>" --json
python3 "$HOME/.cursor/skills/graph-retrieval-memory/scripts/graph_retrieve_test.py"
```

Shared memory (all IDEs):

```
$HOME/Desktop/V.O.I.D/.agents/memory/GOAL.md
$HOME/Desktop/V.O.I.D/.agents/memory/WORKSPACE.md
```

Prefer `agent --workspace "$HOME/Desktop/V.O.I.D"` so project `AGENTS.md` loads.
Hits are paths. Open them. Code wins.
