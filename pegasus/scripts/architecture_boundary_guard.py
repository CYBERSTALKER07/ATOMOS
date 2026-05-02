#!/usr/bin/env python3

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

from guard_utils import added_lines_for_file, changed_files, ensure_git_repo, iter_repo_files, match_any


ROUTE_FILE_PATTERNS = [
    "pegasus/apps/backend-go/*routes/**",
    "pegasus/apps/backend-go/**/*routes*.go",
]

GO_SCAN_PATTERNS = [
    "pegasus/apps/backend-go/*.go",
    "pegasus/apps/backend-go/**/*.go",
]

WRITE_MESSAGES_ALLOWLIST = [
    "pegasus/apps/backend-go/outbox/**",
    "pegasus/apps/backend-go/telemetry/**",
    "pegasus/apps/backend-go/kafka/**",
]

BOOTSTRAP_REF_RE = re.compile(r'"[^"]*bootstrap[^"]*"|\bbootstrap\.App\b|\*bootstrap\.App\b')
DEFAULT_SERVEMUX_RE = re.compile(r"\bhttp\.(?:HandleFunc|Handle|DefaultServeMux)\b")
WRITE_MESSAGES_RE = re.compile(r"\bWriteMessages\s*\(")


def scan_file_lines(
    repo_root: Path,
    file_path: str,
    changed_only: bool,
    base_sha: str | None,
    head_sha: str | None,
) -> list[tuple[int, str]]:
    if changed_only:
        return added_lines_for_file(repo_root, file_path, base_sha, head_sha)

    absolute = repo_root / file_path
    if not absolute.exists():
        return []

    lines = absolute.read_text(encoding="utf-8", errors="ignore").splitlines()
    return [(idx + 1, line) for idx, line in enumerate(lines)]


def main() -> int:
    parser = argparse.ArgumentParser(
        description=(
            "Architecture boundary checker: enforce backend route and event boundary rules."
        )
    )
    parser.add_argument("--repo-root", default=".", help="Repository root.")
    parser.add_argument("--base-sha", default=None, help="Base commit SHA for diff.")
    parser.add_argument("--head-sha", default=None, help="Head commit SHA for diff.")
    parser.add_argument(
        "--all-files",
        action="store_true",
        help="Scan all backend Go files instead of only added lines in changed files.",
    )
    args = parser.parse_args()

    repo_root = Path(args.repo_root).resolve()
    changed_only = not args.all_files

    try:
        ensure_git_repo(repo_root)
    except RuntimeError as exc:
        print(f"architecture-boundary-guard: error: {exc}", file=sys.stderr)
        return 2

    if changed_only:
        files = changed_files(repo_root, args.base_sha, args.head_sha)
    else:
        files = iter_repo_files(repo_root)

    target_files = [
        path
        for path in files
        if path.endswith(".go") and match_any(path, GO_SCAN_PATTERNS)
    ]

    if not target_files:
        print("architecture-boundary-guard: no backend Go targets detected; passing.")
        return 0

    violations: list[dict[str, str | int]] = []

    for file_path in target_files:
        is_route_file = match_any(file_path, ROUTE_FILE_PATTERNS)
        write_allowed = match_any(file_path, WRITE_MESSAGES_ALLOWLIST)

        for line_no, line in scan_file_lines(
            repo_root, file_path, changed_only, args.base_sha, args.head_sha
        ):
            stripped = line.strip()
            if not stripped or stripped.startswith("//"):
                continue

            if is_route_file and BOOTSTRAP_REF_RE.search(stripped):
                violations.append(
                    {
                        "kind": "routes_bootstrap_leak",
                        "file": file_path,
                        "line": line_no,
                        "text": stripped,
                    }
                )

            if DEFAULT_SERVEMUX_RE.search(stripped):
                violations.append(
                    {
                        "kind": "default_servemux_usage",
                        "file": file_path,
                        "line": line_no,
                        "text": stripped,
                    }
                )

            if WRITE_MESSAGES_RE.search(stripped) and not write_allowed:
                violations.append(
                    {
                        "kind": "direct_kafka_write",
                        "file": file_path,
                        "line": line_no,
                        "text": stripped,
                    }
                )

    if not violations:
        print("architecture-boundary-guard: no boundary violations detected; passing.")
        return 0

    print("architecture-boundary-guard: FAIL — boundary violations detected.", file=sys.stderr)
    for issue in violations:
        print(
            f"  - {issue['kind']}: {issue['file']}:{issue['line']} :: {issue['text']}",
            file=sys.stderr,
        )

    return 1


if __name__ == "__main__":
    raise SystemExit(main())
