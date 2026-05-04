#!/usr/bin/env python3

from __future__ import annotations

import argparse
import sys
from pathlib import Path

from guard_utils import changed_files, ensure_git_repo, match_any, run_cmd


ARCH_TRIGGER_PATTERNS = [
    "pegasus/apps/backend-go/*.go",
    "pegasus/apps/backend-go/**/*.go",
    "pegasus/apps/ai-worker/**/*.go",
]

ROUTE_TRIGGER_PATTERNS = [
    "pegasus/apps/backend-go/*routes/**",
    "pegasus/apps/backend-go/**/*routes*.go",
]

ROUTE_DOC_SYNC_PATTERNS = [
    "pegasus/context/architecture.md",
    "pegasus/context/architecture-graph.json",
]

CONTEXT_SYNC_PATTERNS = [
    ".github/ACT.md",
    ".github/copilot-instructions.md",
    ".github/gemini-instructions.md",
    "pegasus/context/architecture.md",
    "pegasus/context/architecture-graph.json",
    "pegasus/context/technology-inventory.md",
    "pegasus/context/technology-inventory.json",
]

MCP_REQUIRED_FILES = [
    ".agents/extensions/ast-engine/engine.mjs",
    ".agents/extensions/ast-engine/mcp-server.mjs",
]


def main() -> int:
    parser = argparse.ArgumentParser(
        description=(
            "Architecture Guard MCP: run architecture boundary checks and enforce context sync + MCP readiness."
        )
    )
    parser.add_argument("--repo-root", default=".", help="Repository root.")
    parser.add_argument("--base-sha", default=None, help="Base commit SHA for diff.")
    parser.add_argument("--head-sha", default=None, help="Head commit SHA for diff.")
    args = parser.parse_args()

    repo_root = Path(args.repo_root).resolve()

    try:
        ensure_git_repo(repo_root)
    except RuntimeError as exc:
        print(f"architecture-guard-mcp: error: {exc}", file=sys.stderr)
        return 2

    files = changed_files(repo_root, args.base_sha, args.head_sha)
    if not files:
        print("architecture-guard-mcp: no changed files detected; skipping.")
        return 0

    trigger_changes = [path for path in files if match_any(path, ARCH_TRIGGER_PATTERNS)]
    if not trigger_changes:
        print("architecture-guard-mcp: no architecture trigger changes detected; passing.")
        return 0

    route_changes = [path for path in files if match_any(path, ROUTE_TRIGGER_PATTERNS)]
    route_doc_sync_changes = [path for path in files if match_any(path, ROUTE_DOC_SYNC_PATTERNS)]
    context_sync_changes = [path for path in files if match_any(path, CONTEXT_SYNC_PATTERNS)]
    codebase_focus_changes = sorted(set(trigger_changes))
    missing_mcp_files = [
        path for path in MCP_REQUIRED_FILES if not (repo_root / path).exists()
    ]

    cmd = [
        "python3",
        "pegasus/scripts/architecture_boundary_guard.py",
        "--repo-root",
        str(repo_root),
    ]
    if args.base_sha:
        cmd.extend(["--base-sha", args.base_sha])
    if args.head_sha:
        cmd.extend(["--head-sha", args.head_sha])

    code, out, err = run_cmd(cmd, cwd=repo_root, allow_fail=True)

    failures: list[str] = []

    if code != 0:
        failures.append("Architecture boundary sub-check failed.")

    if not context_sync_changes:
        failures.append(
            "Architecture context sync missing. Update ACT/instructions/architecture/inventory sync-set files."
        )

    if route_changes and not route_doc_sync_changes:
        failures.append(
            "Route changes detected without route docs sync. Update pegasus/context/architecture.md "
            "or pegasus/context/architecture-graph.json."
        )

    if context_sync_changes and not codebase_focus_changes:
        failures.append(
            "Codebase-first MCP policy violated. Architecture-triggered diffs must rely primarily on "
            "real codebase surfaces; context docs are secondary verification. "
            f"codebase_focus={len(codebase_focus_changes)} context_sync={len(context_sync_changes)}."
        )

    if missing_mcp_files:
        failures.append(
            "Missing required AST MCP engine file(s): " + ", ".join(missing_mcp_files)
        )

    print("architecture-guard-mcp: trigger changes:")
    for path in trigger_changes:
        print(f"  - {path}")

    print(
        "architecture-guard-mcp: context summary "
        f"(codebase_focus={len(codebase_focus_changes)}, context_sync={len(context_sync_changes)})"
    )
    if route_changes:
        print(
            "architecture-guard-mcp: route summary "
            f"(route_changes={len(route_changes)}, route_doc_sync={len(route_doc_sync_changes)})"
        )

    if out.strip():
        print(out.strip())

    if not failures:
        print(
            "architecture-guard-mcp: boundary, context, and codebase-first MCP checks passed."
        )
        return 0

    print("\narchitecture-guard-mcp: FAIL — architecture governance violations detected.", file=sys.stderr)
    for item in failures:
        print(f"  - {item}", file=sys.stderr)

    if err.strip():
        print(err.strip(), file=sys.stderr)

    return 1


if __name__ == "__main__":
    raise SystemExit(main())
