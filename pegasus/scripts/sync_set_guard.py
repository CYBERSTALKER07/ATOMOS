#!/usr/bin/env python3

from __future__ import annotations

import argparse
import fnmatch
import subprocess
import sys
from pathlib import Path


SYNC_SET_FILES = [
    ".github/ACT.md",
    ".github/copilot-instructions.md",
    ".github/gemini-instructions.md",
    "pegasus/context/architecture.md",
    "pegasus/context/architecture-graph.json",
    "pegasus/context/technology-inventory.md",
    "pegasus/context/technology-inventory.json",
]

ARCHITECTURE_TRIGGER_PATTERNS = [
    "pegasus/context/architecture.md",
    "pegasus/context/architecture-graph.json",
    "pegasus/context/design-system.md",
    "pegasus/context/purpose.md",
    "pegasus/apps/backend-go/main.go",
    "pegasus/apps/backend-go/bootstrap/**",
    "pegasus/apps/backend-go/*routes/**",
    "pegasus/apps/ai-worker/**",
    "pegasus/infra/**",
    "pegasus/docker-compose.yml",
]

DEPENDENCY_TRIGGER_PATTERNS = [
    "pegasus/**/package.json",
    "pegasus/**/package-lock.json",
    "pegasus/**/pnpm-lock.yaml",
    "pegasus/**/yarn.lock",
    "pegasus/**/go.mod",
    "pegasus/**/go.sum",
    "pegasus/**/Cargo.toml",
    "pegasus/**/Cargo.lock",
    "pegasus/**/build.gradle",
    "pegasus/**/build.gradle.kts",
    "pegasus/**/gradle.properties",
    "pegasus/**/gradle/libs.versions.toml",
    "pegasus/**/Podfile",
    "pegasus/**/Podfile.lock",
    "pegasus/**/Package.swift",
]

TRIGGER_PATTERNS = ARCHITECTURE_TRIGGER_PATTERNS + DEPENDENCY_TRIGGER_PATTERNS


def run_cmd(cmd: list[str], cwd: Path, allow_fail: bool = False) -> tuple[int, str, str]:
    proc = subprocess.run(
        cmd,
        cwd=str(cwd),
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        check=False,
    )
    if proc.returncode != 0 and not allow_fail:
        raise RuntimeError(
            f"command failed ({proc.returncode}): {' '.join(cmd)}\n{proc.stderr.strip()}"
        )
    return proc.returncode, proc.stdout, proc.stderr


def normalize_sha(value: str | None) -> str | None:
    if value is None:
        return None
    cleaned = value.strip()
    return cleaned if cleaned else None


def fetch_commit_if_missing(repo_root: Path, sha: str) -> None:
    code, _, _ = run_cmd(
        ["git", "cat-file", "-e", f"{sha}^{{commit}}"],
        cwd=repo_root,
        allow_fail=True,
    )
    if code == 0:
        return

    run_cmd(
        ["git", "fetch", "--no-tags", "--depth=1", "origin", sha],
        cwd=repo_root,
        allow_fail=True,
    )


def changed_files(repo_root: Path, base_sha: str | None, head_sha: str | None) -> list[str]:
    normalized_base = normalize_sha(base_sha)
    normalized_head = normalize_sha(head_sha) or "HEAD"

    if normalized_base:
        fetch_commit_if_missing(repo_root, normalized_base)
        if normalized_head != "HEAD":
            fetch_commit_if_missing(repo_root, normalized_head)

        code, out, _ = run_cmd(
            [
                "git",
                "diff",
                "--name-only",
                "--diff-filter=ACMRTUXB",
                f"{normalized_base}...{normalized_head}",
            ],
            cwd=repo_root,
            allow_fail=True,
        )
        if code == 0:
            return sorted({line.strip() for line in out.splitlines() if line.strip()})

    code, out, _ = run_cmd(
        ["git", "diff", "--name-only", "--diff-filter=ACMRTUXB", "HEAD~1...HEAD"],
        cwd=repo_root,
        allow_fail=True,
    )
    if code != 0:
        return []

    return sorted({line.strip() for line in out.splitlines() if line.strip()})


def match_any(path: str, patterns: list[str]) -> bool:
    return any(fnmatch.fnmatch(path, pattern) for pattern in patterns)


def main() -> int:
    parser = argparse.ArgumentParser(
        description=(
            "Fail PRs when architecture/dependency changes are missing required sync-set updates."
        )
    )
    parser.add_argument(
        "--repo-root",
        default=".",
        help="Repository root (must contain .git and the canonical pegasus tree).",
    )
    parser.add_argument("--base-sha", default=None, help="Base commit SHA for diff computation.")
    parser.add_argument("--head-sha", default=None, help="Head commit SHA for diff computation.")
    args = parser.parse_args()

    repo_root = Path(args.repo_root).resolve()

    if not (repo_root / ".git").exists():
        print(f"error: repo root does not contain .git: {repo_root}", file=sys.stderr)
        return 2

    files = changed_files(repo_root, args.base_sha, args.head_sha)
    if not files:
        print("sync-set-guard: no changed files detected; skipping.")
        return 0

    trigger_changes = [path for path in files if match_any(path, TRIGGER_PATTERNS)]
    if not trigger_changes:
        print("sync-set-guard: no architecture/dependency trigger changes detected; passing.")
        return 0

    changed_sync_set = sorted([path for path in files if path in SYNC_SET_FILES])
    missing_sync_set = [path for path in SYNC_SET_FILES if path not in files]

    print("sync-set-guard: architecture/dependency trigger changes detected:")
    for path in trigger_changes:
        print(f"  - {path}")

    if not missing_sync_set:
        print("sync-set-guard: all sync-set files updated in this PR.")
        return 0

    print("\nsync-set-guard: FAIL — sync-set drift detected.", file=sys.stderr)
    print("The following required sync-set files were not updated:", file=sys.stderr)
    for path in missing_sync_set:
        print(f"  - {path}", file=sys.stderr)

    print("\nSync-set files changed in this PR:", file=sys.stderr)
    if changed_sync_set:
        for path in changed_sync_set:
            print(f"  - {path}", file=sys.stderr)
    else:
        print("  - (none)", file=sys.stderr)

    print(
        "\nUpdate the full sync-set when architecture/dependency triggers change.",
        file=sys.stderr,
    )
    return 1


if __name__ == "__main__":
    raise SystemExit(main())