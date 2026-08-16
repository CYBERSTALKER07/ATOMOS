# Gemini — V.O.I.D.

Honesty: `.github/gemini-instructions.md` and `.github/instructions/honest-code-gate.instructions.md`.
Final goal: `.agents/memory/GOAL.md` — `GLOBAL_SCALE_PROGRAM.md` + `GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`.

Retrieval + memory (all agents): read `.agents/memory/WORKSPACE.md`, then

```bash
python3 .agents/skills/graph-retrieval-memory/scripts/graph_retrieve.py -q "<topic>" --hops 2
```

Open the returned paths. Graph is not status. Persist verified facts only to `.agents/memory/WORKSPACE.md`.
See `.agents/skills/graph-retrieval-memory/references/always-on.md`.
