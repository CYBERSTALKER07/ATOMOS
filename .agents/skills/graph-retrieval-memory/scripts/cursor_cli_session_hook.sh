#!/usr/bin/env bash
# Cursor CLI / IDE sessionStart: env + context. Hits are paths, not status.
# User hooks run with cwd ~/.cursor — keep this script path-agnostic.
set -euo pipefail
VOID_REPO="${VOID_REPO:-$HOME/Desktop/V.O.I.D}"
SKILL="${HOME}/.cursor/skills/graph-retrieval-memory"
WALKER="${SKILL}/scripts/graph_retrieve.py"
MEMORY="${VOID_REPO}/.agents/memory/WORKSPACE.md"
GOAL="${VOID_REPO}/.agents/memory/GOAL.md"

export VOID_REPO WALKER MEMORY GOAL

python3 - <<'PY'
import json
import os

void = os.environ["VOID_REPO"]
walker = os.environ["WALKER"]
memory = os.environ["MEMORY"]
goal = os.environ["GOAL"]
ctx = (
    "Cursor CLI graph-retrieval (routing index, not status).\n"
    f"Read first: {goal}\n"
    f"Shared memory: {memory}\n"
    f"Walker (any cwd): python3 {walker} -q \"<topic>\" --hops 2\n"
    "Then open returned paths. Code wins. Persist verified file:line only.\n"
    "Do not treat FEATURES_BY_APP_ROLE or graph runtimeNotes as live status."
)
print(
    json.dumps(
        {
            "env": {
                "VOID_REPO": void,
                "GRAPH_RETRIEVE": walker,
                "VOID_MEMORY": memory,
                "VOID_GOAL": goal,
            },
            "additional_context": ctx,
        }
    )
)
PY
