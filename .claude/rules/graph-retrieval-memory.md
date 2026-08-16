# Graph retrieval + shared memory (Claude Code)

Read `.agents/memory/WORKSPACE.md`. Run the graph walker, then open live code.

```
enhance → retrieve (WORKSPACE.md ∥ graph_retrieve.py ∥ grep) → verify file:line → persist verified only
```

```bash
python3 .agents/skills/graph-retrieval-memory/scripts/graph_retrieve.py -q "<topic>" --hops 2
```

Do not stand up a side vector store. Graph is a routing index. Code wins.

Skill: `.agents/skills/graph-retrieval-memory/`
