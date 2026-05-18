#!/usr/bin/env python3

from __future__ import annotations

import argparse
import json
import os
import shutil
import subprocess
import sys
import time
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


PYTHON_BIN = sys.executable or "python3"
GIT_BIN = shutil.which("git") or "git"


@dataclass(frozen=True)
class CheckDef:
    id: str
    command: list[str]
    env: dict[str, str] | None = None


def run_cmd(command: list[str], cwd: Path, env: dict[str, str] | None = None) -> dict[str, Any]:
    started = time.time()
    proc_env = dict(os.environ)
    if env:
        proc_env.update(env)

    proc = subprocess.run(
        command,
        cwd=str(cwd),
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        check=False,
        env=proc_env,
    )
    finished = time.time()
    duration_ms = int((finished - started) * 1000)

    return {
        "id": "",
        "command": command,
        "exitCode": proc.returncode,
        "durationMs": duration_ms,
        "status": "pass" if proc.returncode == 0 else "fail",
        "stdout": proc.stdout,
        "stderr": proc.stderr,
    }


def run_git(command: list[str], cwd: Path) -> tuple[int, str, str]:
    cmd = list(command)
    if cmd and cmd[0] == "git":
        cmd[0] = GIT_BIN

    proc = subprocess.run(
        cmd,
        cwd=str(cwd),
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        check=False,
    )
    return proc.returncode, proc.stdout.strip(), proc.stderr.strip()


def normalize_versionscan_output_dir(output_dir_rel: str) -> str:
    clean = output_dir_rel.strip("/")
    if clean.startswith("pegasus/"):
        clean = clean[len("pegasus/") :]
    return f"{clean}/versionscan"


def append_diff_args(command: list[str], base_sha: str | None, head_sha: str | None) -> list[str]:
    out = list(command)
    if base_sha:
        out.extend(["--base-sha", base_sha])
    if head_sha:
        out.extend(["--head-sha", head_sha])
    return out


def detect_default_branch(workspace_root: Path, explicit_branch: str | None) -> str:
    if explicit_branch and explicit_branch.strip():
        return explicit_branch.strip()

    code, out, _ = run_git(["git", "symbolic-ref", "--short", "refs/remotes/origin/HEAD"], workspace_root)
    if code == 0 and out.startswith("origin/"):
        return out.split("origin/", 1)[1]

    return "main"


def resolve_merge_base_scope(
    workspace_root: Path,
    default_branch: str,
) -> tuple[str | None, str | None, str | None]:
    refs_to_try = [f"origin/{default_branch}", default_branch]

    # Warm up git invocation in sandboxed environments where the first call may be flaky.
    run_git(["git", "rev-parse", "--is-inside-work-tree"], workspace_root)

    for ref in refs_to_try:
        for _ in range(2):
            code, out, _ = run_git(["git", "merge-base", "HEAD", ref], workspace_root)
            if code == 0 and out:
                return out, "HEAD", ref

    return None, None, None


def collect_changed_files_for_range(workspace_root: Path, base_sha: str, head_sha: str) -> list[str]:
    code, out, _ = run_git(
        [
            "git",
            "diff",
            "--name-only",
            "--diff-filter=ACMRTUXB",
            f"{base_sha}...{head_sha}",
        ],
        workspace_root,
    )
    if code != 0 or not out:
        return []
    return sorted({line.strip() for line in out.splitlines() if line.strip()})


def build_checks(
    workspace_root: Path,
    pegasus_root: Path,
    output_dir_rel: str,
    with_enforce: bool,
    changed_only: bool,
    base_sha: str | None,
    head_sha: str | None,
    versionscan_changed_files: list[str],
) -> list[CheckDef]:
    versionscan_output_rel = normalize_versionscan_output_dir(output_dir_rel)

    checks: list[CheckDef] = [
        CheckDef(
            id="versionscan_scan",
            command=[
                PYTHON_BIN,
                "pegasus/scripts/versionscan.py",
                "scan",
                "--repo-root",
                str(pegasus_root),
                "--output-dir",
                versionscan_output_rel,
            ],
        ),
        CheckDef(
            id="contract_guard_mcp",
            command=append_diff_args(
                [
                    PYTHON_BIN,
                    "pegasus/scripts/contract_guard_mcp.py",
                    "--repo-root",
                    str(workspace_root),
                ],
                base_sha,
                head_sha,
            ),
        ),
        CheckDef(
            id="architecture_guard_mcp",
            command=append_diff_args(
                [
                    PYTHON_BIN,
                    "pegasus/scripts/architecture_guard_mcp.py",
                    "--repo-root",
                    str(workspace_root),
                ],
                base_sha,
                head_sha,
            ),
        ),
        CheckDef(
            id="design_system_guard_mcp",
            command=append_diff_args(
                [
                    PYTHON_BIN,
                    "pegasus/scripts/design_system_guard_mcp.py",
                    "--repo-root",
                    str(workspace_root),
                ],
                base_sha,
                head_sha,
            ),
        ),
        CheckDef(
            id="production_safety_guard",
            command=append_diff_args(
                [
                    PYTHON_BIN,
                    "pegasus/scripts/production_safety_guard.py",
                    "--repo-root",
                    str(workspace_root),
                ],
                base_sha,
                head_sha,
            ),
        ),
        CheckDef(
            id="visual_test_intelligence_guard",
            command=append_diff_args(
                [
                    PYTHON_BIN,
                    "pegasus/scripts/visual_test_intelligence_guard.py",
                    "--repo-root",
                    str(workspace_root),
                ],
                base_sha,
                head_sha,
            ),
        ),
        CheckDef(
            id="security_guard",
            command=append_diff_args(
                [
                    PYTHON_BIN,
                    "pegasus/scripts/security_guard.py",
                    "--repo-root",
                    str(workspace_root),
                ],
                base_sha,
                head_sha,
            ),
        ),
    ]

    if with_enforce:
        enforce_cmd = [
            PYTHON_BIN,
            "pegasus/scripts/versionscan.py",
            "enforce",
            "--repo-root",
            str(pegasus_root),
            "--output-dir",
            versionscan_output_rel,
        ]
        if changed_only:
            enforce_cmd.append("--changed-only")

        enforce_env: dict[str, str] | None = None
        if changed_only and versionscan_changed_files:
            enforce_env = {
                "VERSIONSCAN_CHANGED_FILES": ",".join(versionscan_changed_files)
            }

        checks.append(
            CheckDef(
                id="versionscan_enforce",
                command=enforce_cmd,
                env=enforce_env,
            )
        )

    return checks


def write_report(report_path: Path, payload: dict[str, Any]) -> None:
    report_path.parent.mkdir(parents=True, exist_ok=True)
    report_path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description=(
            "Run Sprint-1 enterprise execution baseline checks and emit a machine-readable report."
        )
    )
    parser.add_argument(
        "--repo-root",
        default=".",
        help="Repository root containing .git and pegasus/",
    )
    parser.add_argument(
        "--output-dir",
        default="pegasus/.execution/sprint1",
        help="Output directory relative to repository root",
    )
    parser.add_argument(
        "--with-enforce",
        action="store_true",
        help="Also run VersionScan enforce mode",
    )
    parser.add_argument(
        "--changed-only",
        action="store_true",
        help="When --with-enforce is set, restrict enforce checks to changed files",
    )
    parser.add_argument("--base-sha", default=None, help="Optional base commit SHA for guard diff scope")
    parser.add_argument("--head-sha", default=None, help="Optional head commit SHA for guard diff scope")
    parser.add_argument(
        "--diff-mode",
        default="auto",
        choices=["auto", "working-tree"],
        help="Diff strategy when base/head are omitted (auto=merge-base, working-tree=local dirty tree)",
    )
    parser.add_argument(
        "--default-branch",
        default=None,
        help="Default branch name used in auto diff mode (fallback: origin/HEAD or main)",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    repo_root = Path(args.repo_root).resolve()
    workspace_root = repo_root

    if not (workspace_root / ".git").exists() and (workspace_root.parent / ".git").exists():
        workspace_root = workspace_root.parent

    if not (workspace_root / ".git").exists():
        print(
            f"sprint1-execution-gate: workspace root does not contain .git: {workspace_root}",
            file=sys.stderr,
        )
        return 2

    pegasus_root = workspace_root / "pegasus"
    if not pegasus_root.exists():
        pegasus_root = workspace_root

    base_sha = args.base_sha
    head_sha = args.head_sha
    diff_source = "explicit" if base_sha or head_sha else "working-tree"
    default_branch = detect_default_branch(workspace_root, args.default_branch)
    default_ref: str | None = None

    if not base_sha and not head_sha and args.diff_mode == "auto":
        auto_base, auto_head, auto_ref = resolve_merge_base_scope(workspace_root, default_branch)
        if auto_base and auto_head:
            base_sha = auto_base
            head_sha = auto_head
            default_ref = auto_ref
            diff_source = "merge-base"
        else:
            diff_source = "working-tree-fallback"

    versionscan_changed_files: list[str] = []
    if args.changed_only and base_sha and head_sha:
        versionscan_changed_files = collect_changed_files_for_range(workspace_root, base_sha, head_sha)

    checks = build_checks(
        workspace_root=workspace_root,
        pegasus_root=pegasus_root,
        output_dir_rel=args.output_dir,
        with_enforce=args.with_enforce,
        changed_only=args.changed_only,
        base_sha=base_sha,
        head_sha=head_sha,
        versionscan_changed_files=versionscan_changed_files,
    )

    results: list[dict[str, Any]] = []

    for check in checks:
        outcome = run_cmd(check.command, cwd=workspace_root, env=check.env)
        outcome["id"] = check.id
        results.append(outcome)

    failed = [r for r in results if r["status"] == "fail"]
    passed = [r for r in results if r["status"] == "pass"]

    report = {
        "generatedAt": datetime.now(timezone.utc).isoformat(),
        "repoRoot": str(workspace_root),
        "pegasusRoot": str(pegasus_root),
        "outputDir": args.output_dir,
        "diffScope": {
            "mode": args.diff_mode,
            "source": diff_source,
            "baseSha": base_sha,
            "headSha": head_sha,
            "defaultBranch": default_branch,
            "defaultRef": default_ref,
            "changedFileCount": len(versionscan_changed_files),
        },
        "summary": {
            "total": len(results),
            "passed": len(passed),
            "failed": len(failed),
            "status": "pass" if not failed else "fail",
        },
        "checks": results,
    }

    report_path = workspace_root / args.output_dir / "gate-report.json"
    write_report(report_path, report)

    print(
        json.dumps(
            {
                "status": report["summary"]["status"],
                "total": report["summary"]["total"],
                "passed": report["summary"]["passed"],
                "failed": report["summary"]["failed"],
                "report": str(report_path),
            },
            indent=2,
        )
    )

    if failed:
        print("\nsprint1-execution-gate: failing checks:", file=sys.stderr)
        for item in failed:
            print(f"- {item['id']} (exit={item['exitCode']})", file=sys.stderr)
        return 1

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
