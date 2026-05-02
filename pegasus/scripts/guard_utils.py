#!/usr/bin/env python3

from __future__ import annotations

import fnmatch
import os
import re
import subprocess
from pathlib import Path


def run_cmd(cmd: list[str], cwd: Path, allow_fail: bool = False) -> tuple[int, str, str]:
    env = dict(os.environ)
    env.setdefault("GIT_CONFIG_GLOBAL", os.devnull)

    proc = subprocess.run(
        cmd,
        cwd=str(cwd),
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        check=False,
        env=env,
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


def ensure_git_repo(repo_root: Path) -> None:
    if not (repo_root / ".git").exists():
        raise RuntimeError(f"repo root does not contain .git: {repo_root}")


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

    # Local mode (no explicit base/head): evaluate current working tree against HEAD.
    code, out, _ = run_cmd(
        ["git", "diff", "--name-only", "--diff-filter=ACMRTUXB", "HEAD"],
        cwd=repo_root,
        allow_fail=True,
    )
    local_files = {line.strip() for line in out.splitlines() if line.strip()} if code == 0 else set()

    code, out, _ = run_cmd(
        ["git", "ls-files", "--others", "--exclude-standard"],
        cwd=repo_root,
        allow_fail=True,
    )
    if code == 0:
        local_files.update({line.strip() for line in out.splitlines() if line.strip()})

    if local_files:
        return sorted(local_files)

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


def iter_repo_files(repo_root: Path) -> list[str]:
    out: list[str] = []
    for path in repo_root.rglob("*"):
        if not path.is_file():
            continue

        rel = path.relative_to(repo_root).as_posix()
        if rel.startswith(".git/"):
            continue

        out.append(rel)

    return sorted(out)


def parse_added_lines(diff_text: str) -> list[tuple[int, str]]:
    added: list[tuple[int, str]] = []
    current_new_line: int | None = None

    for raw in diff_text.splitlines():
        if raw.startswith("@@"):
            # Example: @@ -10,2 +20,4 @@
            match = re.search(r"\+(\d+)(?:,\d+)?", raw)
            if match:
                current_new_line = int(match.group(1))
            continue

        if raw.startswith("+++") or raw.startswith("---"):
            continue

        if raw.startswith("+"):
            if current_new_line is not None:
                added.append((current_new_line, raw[1:]))
                current_new_line += 1
            continue

        if raw.startswith(" "):
            if current_new_line is not None:
                current_new_line += 1
            continue

        if raw.startswith("-"):
            continue

    return added


def added_lines_for_file(
    repo_root: Path,
    file_path: str,
    base_sha: str | None,
    head_sha: str | None,
) -> list[tuple[int, str]]:
    normalized_base = normalize_sha(base_sha)
    normalized_head = normalize_sha(head_sha) or "HEAD"

    if normalized_base:
        fetch_commit_if_missing(repo_root, normalized_base)
        if normalized_head != "HEAD":
            fetch_commit_if_missing(repo_root, normalized_head)
        diff_args = [
            "git",
            "diff",
            "--unified=0",
            "--diff-filter=ACMRTUXB",
            f"{normalized_base}...{normalized_head}",
            "--",
            file_path,
        ]
    else:
        # Local mode without explicit SHAs: inspect added lines in working tree vs HEAD.
        diff_args = [
            "git",
            "diff",
            "--unified=0",
            "--diff-filter=ACMRTUXB",
            "HEAD",
            "--",
            file_path,
        ]

    code, out, _ = run_cmd(diff_args, cwd=repo_root, allow_fail=True)

    if code != 0 and not normalized_base:
        code, out, _ = run_cmd(
            [
                "git",
                "diff",
                "--unified=0",
                "--diff-filter=ACMRTUXB",
                "HEAD~1...HEAD",
                "--",
                file_path,
            ],
            cwd=repo_root,
            allow_fail=True,
        )

    if code != 0 or not out.strip():
        return []

    return parse_added_lines(out)
