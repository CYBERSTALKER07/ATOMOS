#!/usr/bin/env python3

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

from guard_utils import added_lines_for_file, changed_files, ensure_git_repo


CODE_EXTENSIONS = {
    ".go",
    ".py",
    ".ts",
    ".tsx",
    ".js",
    ".jsx",
    ".kt",
    ".swift",
    ".java",
    ".json",
    ".yml",
    ".yaml",
    ".sh",
    ".env",
}

LOCAL_HTTP_ALLOWLIST = (
    "http://localhost",
    "http://127.0.0.1",
    "http://0.0.0.0",
    "http://[::1]",
)

RULES = [
    ("aws_access_key", re.compile(r"\bAKIA[0-9A-Z]{16}\b")),
    ("google_api_key", re.compile(r"\bAIza[0-9A-Za-z\-_]{35}\b")),
    ("private_key_material", re.compile(r"-----BEGIN (?:RSA |EC |OPENSSH |)?PRIVATE KEY-----")),
    ("insecure_tls", re.compile(r"\bInsecureSkipVerify\s*:\s*true\b")),
    (
        "hardcoded_secret_assignment",
        re.compile(r"(?i)\b(password|passwd|secret|api[_-]?key|token)\b\s*[:=]\s*[\"'][^\"']{8,}[\"']"),
    ),
]

SAFE_SECRET_HINTS = ("example", "dummy", "placeholder", "changeme", "test", "ci-")


def is_code_file(path: str) -> bool:
    suffix = Path(path).suffix.lower()
    return suffix in CODE_EXTENSIONS


def main() -> int:
    parser = argparse.ArgumentParser(
        description=(
            "Security Guard: detect obvious secret leaks and insecure runtime/security anti-patterns in added lines."
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
        print(f"security-guard: error: {exc}", file=sys.stderr)
        return 2

    files = changed_files(repo_root, args.base_sha, args.head_sha)
    if not files:
        print("security-guard: no changed files detected; skipping.")
        return 0

    target_files = [path for path in files if is_code_file(path)]
    if not target_files:
        print("security-guard: no code/config file changes detected; passing.")
        return 0

    violations: list[dict[str, str | int]] = []

    for file_path in target_files:
        for line_no, line in added_lines_for_file(
            repo_root, file_path, args.base_sha, args.head_sha
        ):
            stripped = line.strip()
            lowered = stripped.lower()

            if not stripped:
                continue

            if stripped.startswith("//") or stripped.startswith("#") or stripped.startswith("/*"):
                continue

            for kind, pattern in RULES:
                if not pattern.search(stripped):
                    continue

                if kind == "hardcoded_secret_assignment" and any(hint in lowered for hint in SAFE_SECRET_HINTS):
                    continue

                violations.append(
                    {
                        "kind": kind,
                        "file": file_path,
                        "line": line_no,
                        "text": stripped,
                    }
                )

            if "http://" in stripped and not any(item in lowered for item in LOCAL_HTTP_ALLOWLIST):
                violations.append(
                    {
                        "kind": "non_local_http_endpoint",
                        "file": file_path,
                        "line": line_no,
                        "text": stripped,
                    }
                )

    if not violations:
        print("security-guard: no security violations detected; passing.")
        return 0

    print("security-guard: FAIL — security violations detected.", file=sys.stderr)
    for issue in violations:
        print(
            f"  - {issue['kind']}: {issue['file']}:{issue['line']} :: {issue['text']}",
            file=sys.stderr,
        )

    return 1


if __name__ == "__main__":
    raise SystemExit(main())
