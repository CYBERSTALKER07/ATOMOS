---
applyTo: "**"
---

# Graph retrieval + shared memory (Copilot / VS Code)

Read `.agents/memory/GOAL.md` first (final goal survives a new chat). Then `.agents/memory/WORKSPACE.md`. Walk the graph, then open live files.

```
enhance → retrieve (WORKSPACE.md ∥ graph_retrieve.py ∥ grep) → verify file:line → persist verified only
```

```bash
python3 .agents/skills/graph-retrieval-memory/scripts/graph_retrieve.py -q "<topic>" --hops 2
```

Graph (`pegasusX/context/architecture-graph.json`) is a routing index, not status. Code wins. Do not add a side vector DB.

Full loop: `.agents/skills/graph-retrieval-memory/references/always-on.md`
Skill: `.agents/skills/graph-retrieval-memory/SKILL.md`
