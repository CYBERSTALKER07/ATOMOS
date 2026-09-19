#!/usr/bin/env python3
"""Tests for Cursor CLI sessionStart memory payload."""

from __future__ import annotations

import json
import os
import subprocess
import sys
from pathlib import Path

SCRIPT = Path(__file__).with_name("cursor_cli_memory.py")
HOOK = Path(__file__).with_name("cursor_cli_session_hook.sh")


def _write(root: Path, rel: str, body: str) -> Path:
    path = root / rel
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(body, encoding="utf-8")
    return path


def test_payload_injects_goal_and_latest_verified(tmp_path: Path) -> None:
    sys.path.insert(0, str(SCRIPT.parent))
    import cursor_cli_memory as m  # noqa: WPS433

    _write(
        tmp_path,
        ".agents/memory/GOAL.md",
        "# FINAL GOAL\n\nNorth star: local-first multi-supplier.\n",
    )
    _write(
        tmp_path,
        ".agents/memory/WORKSPACE.md",
        (
            "# Shared workspace memory\n\n"
            "## Project Context\n\n"
            "- Living product: pegasusX/\n\n"
            "## Verified 2026-08-15\n\n"
            "- old fact — `old.go:1`\n\n"
            "## Verified 2026-08-16\n\n"
            "- new fact — `new.go:2`\n"
        ),
    )
    payload = m.build_payload(tmp_path)
    ctx = payload["additional_context"]
    assert payload["env"]["VOID_REPO"] == str(tmp_path)
    assert payload["env"]["VOID_GOAL"].endswith("GOAL.md")
    assert payload["env"]["VOID_MEMORY"].endswith("WORKSPACE.md")
    assert "GRAPH_RETRIEVE" in payload["env"]
    assert "North star: local-first multi-supplier." in ctx
    assert "Living product: pegasusX/" in ctx
    assert "new fact" in ctx
    assert "old fact" in ctx
    assert len(ctx) <= m.MAX_CONTEXT_CHARS


def test_missing_files_fail_open(tmp_path: Path) -> None:
    sys.path.insert(0, str(SCRIPT.parent))
    import cursor_cli_memory as m  # noqa: WPS433

    payload = m.build_payload(tmp_path)
    ctx = payload["additional_context"]
    assert "(GOAL.md missing)" in ctx
    assert "(WORKSPACE.md missing)" in ctx
    json.dumps(payload)


def test_excerpt_keeps_newest_verified_when_clipped() -> None:
    sys.path.insert(0, str(SCRIPT.parent))
    import cursor_cli_memory as m  # noqa: WPS433

    body = (
        "## Project Context\n\nctx\n\n"
        "## Verified 2026-08-01\n\n" + ("aaaa\n" * 400) + "\n"
        "## Verified 2026-08-16\n\nKEEP_THIS_BULLET — `keep.go:9`\n"
    )
    excerpt = m.excerpt_workspace(body, limit=800)
    assert "KEEP_THIS_BULLET" in excerpt
    assert len(excerpt) <= 800


def test_cli_and_hook_emit_json() -> None:
    out = subprocess.check_output([sys.executable, str(SCRIPT)], text=True)
    data = json.loads(out)
    assert "additional_context" in data
    assert data["env"]["VOID_GOAL"].endswith("GOAL.md")
    hook_env = os.environ.copy()
    hook_out = subprocess.check_output(["bash", str(HOOK)], text=True, env=hook_env)
    hook_data = json.loads(hook_out)
    assert hook_data["env"]["VOID_MEMORY"].endswith("WORKSPACE.md")
    assert "=== GOAL.md ===" in hook_data["additional_context"]


if __name__ == "__main__":
    import tempfile

    with tempfile.TemporaryDirectory() as td:
        root = Path(td)
        test_payload_injects_goal_and_latest_verified(root / "full")
        test_missing_files_fail_open(root / "empty")
    test_excerpt_keeps_newest_verified_when_clipped()
    test_cli_and_hook_emit_json()
    print("ok")
