#!/usr/bin/env python3

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

from guard_utils import added_lines_for_file, changed_files, ensure_git_repo, match_any


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

WEBHOOK_FILE_PATTERNS = [
    "pegasus/apps/backend-go/webhookroutes/**/*.go",
    "pegasus/apps/backend-go/**/*webhook*.go",
    "pegasus/apps/backend-go/payment/**/*.go",
]

WEBHOOK_PARSE_RE = re.compile(
    r"\b(?:json\.NewDecoder\s*\(\s*r\.Body\s*\)\.Decode|json\.Unmarshal\s*\(|io\.ReadAll\s*\(\s*r\.Body\s*\)|ReadAll\s*\(\s*r\.Body\s*\))"
)
WEBHOOK_SIGNATURE_RE = re.compile(
    r"(?i)\b(?:verify|signature|hmac|basicauth|validate(?:[A-Za-z_]*signature)?|checksignature)\b"
)

CLIENT_MUTATION_FILE_PATTERNS = [
    "pegasus/apps/admin-portal/**/*.ts",
    "pegasus/apps/admin-portal/**/*.tsx",
    "pegasus/apps/factory-portal/**/*.ts",
    "pegasus/apps/factory-portal/**/*.tsx",
    "pegasus/apps/warehouse-portal/**/*.ts",
    "pegasus/apps/warehouse-portal/**/*.tsx",
    "pegasus/apps/retailer-app-desktop/**/*.ts",
    "pegasus/apps/retailer-app-desktop/**/*.tsx",
]

CLIENT_IDEMPOTENCY_EXEMPT_PATTERNS = [
    "pegasus/apps/*/app/auth/login/**",
    "pegasus/apps/*/app/**/bootstrap/**",
    "pegasus/apps/*/app/bootstrap/**",
]

MUTATION_CALL_RE = re.compile(r"\b(?:fetch\s*\(|\.(?:post|patch|put|delete)\s*\()")
MUTATION_METHOD_RE = re.compile(r"""(?i)\bmethod\s*:\s*["'](?:POST|PATCH|PUT|DELETE)["']""")
IDEMPOTENCY_ENDPOINT_RE = re.compile(
    r"(?i)/v1/[^\"']*(?:order|checkout|payment|refund|cancel|card|manifest|dispatch|wallet)"
)


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
    file_line_cache: dict[str, list[str]] = {}

    for file_path in target_files:
        is_webhook_file = match_any(file_path, WEBHOOK_FILE_PATTERNS)
        is_client_mutation_file = match_any(file_path, CLIENT_MUTATION_FILE_PATTERNS)
        exempt_idempotency = match_any(file_path, CLIENT_IDEMPOTENCY_EXEMPT_PATTERNS)

        absolute = repo_root / file_path
        if absolute.exists():
            file_line_cache[file_path] = absolute.read_text(
                encoding="utf-8", errors="ignore"
            ).splitlines()
        else:
            file_line_cache[file_path] = []

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

            if is_webhook_file and WEBHOOK_PARSE_RE.search(stripped):
                all_lines = file_line_cache[file_path]
                lookback_start = max(0, line_no - 26)
                lookback = all_lines[lookback_start : max(0, line_no - 1)]
                if not any(WEBHOOK_SIGNATURE_RE.search(item) for item in lookback):
                    violations.append(
                        {
                            "kind": "webhook_parse_before_signature_check",
                            "file": file_path,
                            "line": line_no,
                            "text": stripped,
                        }
                    )

            if is_client_mutation_file and not exempt_idempotency:
                if not MUTATION_CALL_RE.search(stripped):
                    continue

                all_lines = file_line_cache[file_path]
                window_start = max(0, line_no - 1)
                window_end = min(len(all_lines), line_no + 10)
                window = "\n".join(all_lines[window_start:window_end])

                if "idempotency-key" in window.lower():
                    continue

                if "auth/login" in window.lower() or "bootstrap" in window.lower():
                    continue

                if not IDEMPOTENCY_ENDPOINT_RE.search(window):
                    continue

                if "fetch(" in stripped and not MUTATION_METHOD_RE.search(window):
                    continue

                violations.append(
                    {
                        "kind": "missing_idempotency_key_header",
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
