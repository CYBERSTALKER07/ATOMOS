# Graph retrieve (Cursor CLI)

Hits are **paths**, not status. `generatedAt` is null. Code wins.

1. Read `.agents/memory/GOAL.md` then `.agents/memory/WORKSPACE.md`.
2. Run (repo cwd):

```bash
python3 .agents/skills/graph-retrieval-memory/scripts/graph_retrieve.py -q "$ARGUMENTS" --hops 2
```

From any cwd:

```bash
python3 "$HOME/.cursor/skills/graph-retrieval-memory/scripts/graph_retrieve.py" -q "$ARGUMENTS" --hops 2
```

3. Open every returned path this session. Grep if the graph misses (e.g. `fiscal retry`).
4. Persist only after file:line verify to `.agents/memory/WORKSPACE.md`.
