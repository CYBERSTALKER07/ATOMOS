# Graph retrieval + memory (always on — Grok)

Same loop as every other IDE. Canonical: `.agents/skills/graph-retrieval-memory/references/always-on.md`

0. Read `.agents/memory/GOAL.md` (final goal — survives `/new`).
1. Read `.agents/memory/WORKSPACE.md` (and `~/.grok/memory/` if indexed).
2. `python3 .agents/skills/graph-retrieval-memory/scripts/graph_retrieve.py -q "<topic>"`
3. Open returned paths. Code wins.
4. Persist verified facts to `.agents/memory/WORKSPACE.md` (and Grok memory if on).

Do not add a side vector database.
