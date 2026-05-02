#!/usr/bin/env python3

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

from guard_utils import added_lines_for_file, changed_files, ensure_git_repo, match_any


CRITICAL_PATTERNS = [
    "pegasus/apps/backend-go/*.go",
    "pegasus/apps/backend-go/**/*.go",
    "pegasus/apps/ai-worker/**/*.go",
    "pegasus/infra/terraform/**",
    "pegasus/docker-compose.yml",
]

TEST_EVIDENCE_PATTERNS = [
    "tests/**",
    "pegasus/tests/**",
    "pegasus/playwright.config.ts",
    "pegasus/apps/**/*_test.go",
    "pegasus/apps/**/*.test.ts",
    "pegasus/apps/**/*.test.tsx",
    "pegasus/apps/**/*.spec.ts",
    "pegasus/apps/**/*.spec.tsx",
    "pegasus/apps/**/__tests__/**",
]

RULES = [
    ("panic_call", re.compile(r"\bpanic\s*\(")),
    ("sleep_call", re.compile(r"\btime\.Sleep\s*\(")),
    ("background_context", re.compile(r"\bcontext\.Background\s*\(\)")),
    ("insecure_tls", re.compile(r"\bInsecureSkipVerify\s*:\s*true\b")),
]


def main() -> int:
    parser = argparse.ArgumentParser(
        description=(
            "Production Safety Guard: enforce test evidence and block high-risk runtime anti-patterns in critical surfaces."
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
        print(f"production-safety-guard: error: {exc}", file=sys.stderr)
        return 2

    files = changed_files(repo_root, args.base_sha, args.head_sha)
    if not files:
        print("production-safety-guard: no changed files detected; skipping.")
        return 0

    critical_changes = [path for path in files if match_any(path, CRITICAL_PATTERNS)]
    if not critical_changes:
        print("production-safety-guard: no critical production-surface changes detected; passing.")
        return 0

    test_evidence_changes = [path for path in files if match_any(path, TEST_EVIDENCE_PATTERNS)]

    violations: list[dict[str, str | int]] = []

    for file_path in critical_changes:
        for line_no, line in added_lines_for_file(
            repo_root, file_path, args.base_sha, args.head_sha
        ):
            stripped = line.strip()
            if not stripped:
                continue

            if stripped.startswith("//") or stripped.startswith("#"):
                continue

            for kind, pattern in RULES:
                if pattern.search(stripped):
                    violations.append(
                        {
                            "kind": kind,
                            "file": file_path,
                            "line": line_no,
                            "text": stripped,
                        }
                    )

            if "http.Client{" in stripped and "Timeout" not in stripped:
                violations.append(
                    {
                        "kind": "http_client_without_timeout",
                        "file": file_path,
                        "line": line_no,
                        "text": stripped,
                    }
                )

    failures: list[str] = []

    if not test_evidence_changes:
        failures.append(
            "Critical production surfaces changed without test evidence updates."
        )

    if violations:
        failures.append("High-risk production anti-patterns detected in added lines.")

    print("production-safety-guard: critical changes:")
    for path in critical_changes:
        print(f"  - {path}")

    if test_evidence_changes:
        print("production-safety-guard: test evidence changes:")
        for path in test_evidence_changes:
            print(f"  - {path}")

    if not failures:
        print("production-safety-guard: production safety checks passed.")
        return 0

    print("\nproduction-safety-guard: FAIL — production safety violations detected.", file=sys.stderr)
    for item in failures:
        print(f"  - {item}", file=sys.stderr)

    for issue in violations:
        print(
            f"  - {issue['kind']}: {issue['file']}:{issue['line']} :: {issue['text']}",
            file=sys.stderr,
        )

    return 1


if __name__ == "__main__":
    raise SystemExit(main())
