#!/usr/bin/env python3

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

from guard_utils import added_lines_for_file, changed_files, ensure_git_repo, match_any, run_cmd


UI_TRIGGER_PATTERNS = [
    "pegasus/apps/admin-portal/**",
    "pegasus/apps/factory-portal/**",
    "pegasus/apps/warehouse-portal/**",
    "pegasus/apps/retailer-app-desktop/**",
    "pegasus/packages/ui-kit/**",
]

TOKEN_SOURCE_PATTERNS = [
    "pegasus/apps/admin-portal/app/globals.css",
    "pegasus/apps/factory-portal/app/globals.css",
    "pegasus/apps/warehouse-portal/app/globals.css",
    "pegasus/apps/retailer-app-desktop/app/globals.css",
    "pegasus/packages/ui-kit/**/tokens*.css",
    "pegasus/packages/ui-kit/**/tokens*.ts",
]

DESIGN_SYNC_PATTERNS = [
    "pegasus/context/design-system.md",
    "pegasus/context/technology-inventory.md",
    "pegasus/context/technology-inventory.json",
]

MCP_REQUIRED_FILES = [
    ".agents/extensions/ast-engine/engine.mjs",
    ".agents/extensions/ast-engine/mcp-server.mjs",
]

FRONTEND_SCAN_PATTERNS = [
    "pegasus/apps/admin-portal/**/*.ts",
    "pegasus/apps/admin-portal/**/*.tsx",
    "pegasus/apps/factory-portal/**/*.ts",
    "pegasus/apps/factory-portal/**/*.tsx",
    "pegasus/apps/warehouse-portal/**/*.ts",
    "pegasus/apps/warehouse-portal/**/*.tsx",
    "pegasus/apps/retailer-app-desktop/**/*.ts",
    "pegasus/apps/retailer-app-desktop/**/*.tsx",
]

RAW_FETCH_ALLOWLIST_PATTERNS = [
    "pegasus/packages/api-client/**",
    "pegasus/apps/*/lib/auth.ts",
    "pegasus/apps/*/app/auth/login/**",
    "pegasus/apps/*/app/**/bootstrap/**",
    "pegasus/apps/*/app/bootstrap/**",
]

RAW_FETCH_RE = re.compile(r"\bfetch\s*\(")


def main() -> int:
    parser = argparse.ArgumentParser(
        description=(
            "Design System Guard MCP: run design-token enforcement and require design-system sync on token source updates."
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
        print(f"design-system-guard-mcp: error: {exc}", file=sys.stderr)
        return 2

    files = changed_files(repo_root, args.base_sha, args.head_sha)
    if not files:
        print("design-system-guard-mcp: no changed files detected; skipping.")
        return 0

    trigger_changes = [path for path in files if match_any(path, UI_TRIGGER_PATTERNS)]
    if not trigger_changes:
        print("design-system-guard-mcp: no UI/design-system trigger changes detected; passing.")
        return 0

    token_source_changes = [path for path in files if match_any(path, TOKEN_SOURCE_PATTERNS)]
    design_sync_changes = [path for path in files if match_any(path, DESIGN_SYNC_PATTERNS)]
    codebase_focus_changes = sorted(set(trigger_changes + token_source_changes))
    missing_mcp_files = [
        path for path in MCP_REQUIRED_FILES if not (repo_root / path).exists()
    ]

    cmd = [
        "python3",
        "pegasus/scripts/design_token_enforcement_guard.py",
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
        failures.append("Design-token sub-check failed.")

    if token_source_changes and not design_sync_changes:
        failures.append(
            "Design token source changed without design-system context sync. Update pegasus/context/design-system.md."
        )

    raw_fetch_violations: list[tuple[str, int, str]] = []
    frontend_scan_targets = [path for path in files if match_any(path, FRONTEND_SCAN_PATTERNS)]
    for file_path in frontend_scan_targets:
        if match_any(file_path, RAW_FETCH_ALLOWLIST_PATTERNS):
            continue

        for line_no, line in added_lines_for_file(
            repo_root, file_path, args.base_sha, args.head_sha
        ):
            stripped = line.strip()
            if not stripped or stripped.startswith("//") or stripped.startswith("*"):
                continue

            if RAW_FETCH_RE.search(stripped):
                raw_fetch_violations.append((file_path, line_no, stripped))

    if raw_fetch_violations:
        failures.append(
            "New raw fetch() usage detected outside shared API helper. "
            "Use packages/api-client or auth/bootstrap allowlisted surfaces."
        )

    if design_sync_changes and len(codebase_focus_changes) < len(design_sync_changes):
        failures.append(
            "Codebase-first MCP policy violated. Design-triggered diffs must rely primarily on "
            "real codebase surfaces; context docs are secondary verification. "
            f"codebase_focus={len(codebase_focus_changes)} context_sync={len(design_sync_changes)}."
        )

    if missing_mcp_files:
        failures.append(
            "Missing required AST MCP engine file(s): " + ", ".join(missing_mcp_files)
        )

    print("design-system-guard-mcp: trigger changes:")
    for path in trigger_changes:
        print(f"  - {path}")

    print(
        "design-system-guard-mcp: context summary "
        f"(codebase_focus={len(codebase_focus_changes)}, context_sync={len(design_sync_changes)})"
    )

    if out.strip():
        print(out.strip())

    if raw_fetch_violations:
        print("design-system-guard-mcp: raw fetch violations:", file=sys.stderr)
        for file_path, line_no, text in raw_fetch_violations:
            print(f"  - {file_path}:{line_no} :: {text}", file=sys.stderr)

    if not failures:
        print(
            "design-system-guard-mcp: token, design-sync, and codebase-first MCP checks passed."
        )
        return 0

    print("\ndesign-system-guard-mcp: FAIL — design-system governance violations detected.", file=sys.stderr)
    for item in failures:
        print(f"  - {item}", file=sys.stderr)

    if err.strip():
        print(err.strip(), file=sys.stderr)

    return 1


if __name__ == "__main__":
    raise SystemExit(main())
