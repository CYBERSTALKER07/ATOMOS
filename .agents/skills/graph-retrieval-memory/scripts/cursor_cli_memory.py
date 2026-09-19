#!/usr/bin/env python3
"""Cursor sessionStart payload: env + bounded GOAL/WORKSPACE injection.

Prints one JSON object on stdout for hooks.json sessionStart.
Paths and excerpts are hypotheses until re-read in code this session.
"""

from __future__ import annotations

import json
import os
import sys
from pathlib import Path

MAX_GOAL_CHARS = 4000
MAX_WORKSPACE_CHARS = 3200
MAX_CONTEXT_CHARS = 7800


def find_void_repo() -> Path:
    env = os.environ.get("VOID_REPO")
    if env:
        p = Path(env).expanduser()
        if (p / ".agents" / "memory" / "GOAL.md").is_file():
            return p.resolve()
    default = Path.home() / "Desktop" / "V.O.I.D"
    if (default / ".agents" / "memory" / "GOAL.md").is_file():
        return default.resolve()
    here = Path(__file__).resolve()
    for parent in [here, *here.parents]:
        if (parent / ".agents" / "memory" / "GOAL.md").is_file():
            return parent
    return default


def find_walker(void_repo: Path) -> Path:
    home = (
        Path.home()
        / ".cursor"
        / "skills"
        / "graph-retrieval-memory"
        / "scripts"
        / "graph_retrieve.py"
    )
    if home.is_file() or home.is_symlink():
        return home
    local = (
        void_repo
        / ".agents"
        / "skills"
        / "graph-retrieval-memory"
        / "scripts"
        / "graph_retrieve.py"
    )
    if local.is_file():
        return local
    return Path(__file__).with_name("graph_retrieve.py")


def clip(text: str, limit: int) -> str:
    text = text.replace("\r\n", "\n").strip()
    if len(text) <= limit:
        return text
    return text[: max(0, limit - 18)].rstrip() + "\n...[truncated]"


def _section_at(lines: list[str], headings: list[tuple[int, str]], idx: int) -> str:
    start = headings[idx][0]
    end = headings[idx + 1][0] if idx + 1 < len(headings) else len(lines)
    return "\n".join(lines[start:end]).strip()


def excerpt_workspace(text: str, limit: int = MAX_WORKSPACE_CHARS) -> str:
    """Project Context plus newest Verified sections. Newest verified wins if clipped."""
    if not text.strip():
        return ""
    lines = text.replace("\r\n", "\n").split("\n")
    headings = [(i, line) for i, line in enumerate(lines) if line.startswith("## ")]
    if not headings:
        return clip(text, limit)

    context = ""
    for n, (_, title) in enumerate(headings):
        if title.lower().startswith("## project context"):
            context = _section_at(lines, headings, n)
            break
    verified = [
        _section_at(lines, headings, n)
        for n, (_, title) in enumerate(headings)
        if title.lower().startswith("## verified")
    ]
    newest = verified[-1] if verified else ""
    older = verified[-2:-1] if len(verified) >= 2 else []

    pieces: list[str] = []
    if context:
        pieces.append(context)
    pieces.extend(older)
    if newest:
        pieces.append(newest)
    if not pieces:
        return clip(text, limit)

    joined = "\n\n".join(pieces)
    if len(joined) <= limit:
        return joined
    if newest and len(pieces) >= 2:
        budget = max(80, limit - len(newest) - 2)
        head = clip("\n\n".join(pieces[:-1]), budget)
        return clip(head + "\n\n" + newest, limit)
    return clip(joined, limit)


def read_text(path: Path) -> str:
    try:
        return path.read_text(encoding="utf-8")
    except OSError:
        return ""


def build_payload(void_repo: Path | None = None) -> dict:
    repo = void_repo or find_void_repo()
    walker = find_walker(repo)
    goal = repo / ".agents" / "memory" / "GOAL.md"
    memory = repo / ".agents" / "memory" / "WORKSPACE.md"
    goal_body = read_text(goal)
    mem_body = read_text(memory)

    header = (
        "Cursor CLI graph-retrieval memory (hypothesis until re-read in code).\n"
        f"Read first: {goal}\n"
        f"Shared memory: {memory}\n"
        f'Walker (any cwd): python3 {walker} -q "<topic>" --hops 2\n'
        "Then open returned paths. Code wins. Persist verified file:line only.\n"
        "Do not treat FEATURES_BY_APP_ROLE or graph runtimeNotes as live status.\n"
        'Prefer: agent --workspace "$HOME/Desktop/V.O.I.D"\n'
    )
    chunks = [
        header,
        "=== GOAL.md ===",
        clip(goal_body, MAX_GOAL_CHARS) if goal_body else "(GOAL.md missing)",
        "",
        "=== WORKSPACE.md (context + latest verified) ===",
        excerpt_workspace(mem_body) if mem_body else "(WORKSPACE.md missing)",
    ]
    ctx = clip("\n".join(chunks), MAX_CONTEXT_CHARS)
    return {
        "env": {
            "VOID_REPO": str(repo),
            "GRAPH_RETRIEVE": str(walker),
            "VOID_MEMORY": str(memory),
            "VOID_GOAL": str(goal),
        },
        "additional_context": ctx,
    }


def main() -> int:
    json.dump(build_payload(), sys.stdout, ensure_ascii=False)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
