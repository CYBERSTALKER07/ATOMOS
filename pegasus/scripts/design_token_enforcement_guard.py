#!/usr/bin/env python3

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

from guard_utils import added_lines_for_file, changed_files, ensure_git_repo, iter_repo_files, match_any


UI_TARGET_PATTERNS = [
    "pegasus/apps/admin-portal/**",
    "pegasus/apps/factory-portal/**",
    "pegasus/apps/warehouse-portal/**",
    "pegasus/apps/retailer-app-desktop/**",
    "pegasus/packages/ui-kit/**",
]

UI_EXTENSIONS = {".ts", ".tsx", ".js", ".jsx", ".css", ".scss", ".sass", ".less"}

TOKEN_SOURCE_ALLOWLIST = [
    "pegasus/apps/admin-portal/app/globals.css",
    "pegasus/apps/factory-portal/app/globals.css",
    "pegasus/apps/warehouse-portal/app/globals.css",
    "pegasus/apps/retailer-app-desktop/app/globals.css",
    "pegasus/packages/ui-kit/**/tokens*.css",
    "pegasus/packages/ui-kit/**/tokens*.ts",
]

HEX_COLOR_RE = re.compile(r"#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})\b")
RGB_HSL_RE = re.compile(r"\b(?:rgb|rgba|hsl|hsla)\s*\(")
TAILWIND_COLOR_RE = re.compile(
    r"\b(?:bg|text|border|fill|stroke)-"
    r"(?:slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-"
    r"(?:50|100|200|300|400|500|600|700|800|900|950)\b"
)


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
            "Design-token enforcement checker: fail on newly added hardcoded colors on token-governed UI surfaces."
        )
    )
    parser.add_argument("--repo-root", default=".", help="Repository root.")
    parser.add_argument("--base-sha", default=None, help="Base commit SHA for diff.")
    parser.add_argument("--head-sha", default=None, help="Head commit SHA for diff.")
    parser.add_argument(
        "--all-files",
        action="store_true",
        help="Scan full UI target files instead of only added diff lines.",
    )
    args = parser.parse_args()

    repo_root = Path(args.repo_root).resolve()
    changed_only = not args.all_files

    try:
        ensure_git_repo(repo_root)
    except RuntimeError as exc:
        print(f"design-token-guard: error: {exc}", file=sys.stderr)
        return 2

    if changed_only:
        files = changed_files(repo_root, args.base_sha, args.head_sha)
    else:
        files = iter_repo_files(repo_root)

    target_files: list[str] = []
    for path in files:
        if not match_any(path, UI_TARGET_PATTERNS):
            continue

        if Path(path).suffix.lower() not in UI_EXTENSIONS:
            continue

        target_files.append(path)

    if not target_files:
        print("design-token-guard: no UI target file changes detected; passing.")
        return 0

    violations: list[dict[str, str | int]] = []

    for file_path in target_files:
        is_token_source = match_any(file_path, TOKEN_SOURCE_ALLOWLIST)

        for line_no, line in scan_file_lines(
            repo_root, file_path, changed_only, args.base_sha, args.head_sha
        ):
            stripped = line.strip()
            if not stripped:
                continue

            if stripped.startswith("//") or stripped.startswith("/*") or stripped.startswith("*"):
                continue

            if "var(--color-md-" in stripped or "--color-md-" in stripped:
                continue

            if not is_token_source and HEX_COLOR_RE.search(stripped):
                violations.append(
                    {
                        "kind": "hardcoded_hex_color",
                        "file": file_path,
                        "line": line_no,
                        "text": stripped,
                    }
                )

            if not is_token_source and RGB_HSL_RE.search(stripped):
                violations.append(
                    {
                        "kind": "hardcoded_rgb_hsl_color",
                        "file": file_path,
                        "line": line_no,
                        "text": stripped,
                    }
                )

            if TAILWIND_COLOR_RE.search(stripped):
                violations.append(
                    {
                        "kind": "raw_tailwind_color_scale",
                        "file": file_path,
                        "line": line_no,
                        "text": stripped,
                    }
                )

    if not violations:
        print("design-token-guard: no token violations detected; passing.")
        return 0

    print("design-token-guard: FAIL — token policy violations detected.", file=sys.stderr)
    for issue in violations:
        print(
            f"  - {issue['kind']}: {issue['file']}:{issue['line']} :: {issue['text']}",
            file=sys.stderr,
        )

    return 1


if __name__ == "__main__":
    raise SystemExit(main())
