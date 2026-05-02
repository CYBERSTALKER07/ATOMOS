#!/usr/bin/env python3

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

from guard_utils import added_lines_for_file, changed_files, ensure_git_repo, match_any


UI_TRIGGER_PATTERNS = [
    "pegasus/apps/admin-portal/**",
    "pegasus/apps/factory-portal/**",
    "pegasus/apps/warehouse-portal/**",
    "pegasus/apps/retailer-app-desktop/**",
    "pegasus/packages/ui-kit/**",
    "pegasus/apps/payload-terminal/**",
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

VISUAL_RULES = [
    ("decorative_gradient", re.compile(r"(?i)(?:bg-gradient-|linear-gradient|radial-gradient|conic-gradient)")),
    ("emoji_icon_usage", re.compile(r"[\U0001F300-\U0001FAFF]")),
]


def main() -> int:
    parser = argparse.ArgumentParser(
        description=(
            "Visual + Test Intelligence Guard: require test evidence on UI changes and block forbidden visual patterns."
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
        print(f"visual-test-intelligence-guard: error: {exc}", file=sys.stderr)
        return 2

    files = changed_files(repo_root, args.base_sha, args.head_sha)
    if not files:
        print("visual-test-intelligence-guard: no changed files detected; skipping.")
        return 0

    ui_changes = [path for path in files if match_any(path, UI_TRIGGER_PATTERNS)]
    if not ui_changes:
        print("visual-test-intelligence-guard: no UI surface changes detected; passing.")
        return 0

    test_evidence_changes = [path for path in files if match_any(path, TEST_EVIDENCE_PATTERNS)]

    violations: list[dict[str, str | int]] = []

    for file_path in ui_changes:
        for line_no, line in added_lines_for_file(
            repo_root, file_path, args.base_sha, args.head_sha
        ):
            stripped = line.strip()
            if not stripped:
                continue

            if stripped.startswith("//") or stripped.startswith("#"):
                continue

            for kind, pattern in VISUAL_RULES:
                if pattern.search(stripped):
                    violations.append(
                        {
                            "kind": kind,
                            "file": file_path,
                            "line": line_no,
                            "text": stripped,
                        }
                    )

    failures: list[str] = []

    if not test_evidence_changes:
        failures.append("UI changed without test evidence updates.")

    if violations:
        failures.append("Forbidden visual pattern detected in added UI lines.")

    print("visual-test-intelligence-guard: UI changes:")
    for path in ui_changes:
        print(f"  - {path}")

    if test_evidence_changes:
        print("visual-test-intelligence-guard: test evidence changes:")
        for path in test_evidence_changes:
            print(f"  - {path}")

    if not failures:
        print("visual-test-intelligence-guard: visual/test checks passed.")
        return 0

    print("\nvisual-test-intelligence-guard: FAIL — visual/test violations detected.", file=sys.stderr)
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
