#!/usr/bin/env python3

from __future__ import annotations

import argparse
import sys
from pathlib import Path

from guard_utils import changed_files, ensure_git_repo, match_any


CONTRACT_TRIGGER_PATTERNS = [
    "pegasus/apps/backend-go/*routes/**",
    "pegasus/apps/backend-go/**/*routes*.go",
    "pegasus/apps/backend-go/**/kafka/events.go",
    "pegasus/apps/backend-go/webhookroutes/**",
    "pegasus/apps/backend-go/sync/**",
]

SHARED_CONTRACT_PATTERNS = [
    "pegasus/packages/types/**",
    "pegasus/packages/api-client/**",
    "pegasus/packages/validation/**",
    "pegasus/packages/optimizer-contract/**",
]

CONSUMER_SURFACE_PATTERNS = [
    "pegasus/apps/ai-worker/**",
    "pegasus/apps/admin-portal/**",
    "pegasus/apps/factory-portal/**",
    "pegasus/apps/warehouse-portal/**",
    "pegasus/apps/retailer-app-desktop/**",
    "pegasus/apps/*-app-android/**",
    "pegasus/apps/*-app-ios/**",
    "pegasus/apps/driverappios/**",
    "pegasus/apps/payload-terminal/**",
    "pegasus/apps/backend-go/kafka/**",
    "pegasus/apps/backend-go/notifications/**",
]

CONTEXT_SYNC_PATTERNS = [
    "pegasus/context/architecture.md",
    "pegasus/context/architecture-graph.json",
    "pegasus/context/technology-inventory.md",
    "pegasus/context/technology-inventory.json",
]

MCP_REQUIRED_FILES = [
    ".agents/extensions/ast-engine/engine.mjs",
    ".agents/extensions/ast-engine/mcp-server.mjs",
]


def main() -> int:
    parser = argparse.ArgumentParser(
        description=(
            "Contract Guard MCP: enforce contract drift discipline plus AST MCP context readiness."
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
        print(f"contract-guard-mcp: error: {exc}", file=sys.stderr)
        return 2

    files = changed_files(repo_root, args.base_sha, args.head_sha)
    if not files:
        print("contract-guard-mcp: no changed files detected; skipping.")
        return 0

    trigger_changes = [path for path in files if match_any(path, CONTRACT_TRIGGER_PATTERNS)]
    if not trigger_changes:
        print("contract-guard-mcp: no contract trigger changes detected; passing.")
        return 0

    shared_changes = [path for path in files if match_any(path, SHARED_CONTRACT_PATTERNS)]
    consumer_changes = [path for path in files if match_any(path, CONSUMER_SURFACE_PATTERNS)]
    context_sync_changes = [path for path in files if match_any(path, CONTEXT_SYNC_PATTERNS)]

    missing_mcp_files = [
        path for path in MCP_REQUIRED_FILES if not (repo_root / path).exists()
    ]

    failures: list[str] = []

    if not shared_changes:
        failures.append(
            "Shared contract surface missing. Update at least one of: "
            "packages/types, packages/api-client, packages/validation, packages/optimizer-contract."
        )

    if not consumer_changes:
        failures.append(
            "Consumer surface missing. Update at least one consuming surface "
            "(web/mobile/worker/kafka-notification layer)."
        )

    if not context_sync_changes:
        failures.append(
            "MCP context sync missing. Update at least one architecture/technology context file."
        )

    if missing_mcp_files:
        failures.append(
            "Missing required AST MCP engine file(s): " + ", ".join(missing_mcp_files)
        )

    print("contract-guard-mcp: trigger changes:")
    for path in trigger_changes:
        print(f"  - {path}")

    if not failures:
        print("contract-guard-mcp: contract, consumer, and MCP context checks passed.")
        return 0

    print("\ncontract-guard-mcp: FAIL — contract governance violations detected.", file=sys.stderr)
    for item in failures:
        print(f"  - {item}", file=sys.stderr)

    return 1


if __name__ == "__main__":
    raise SystemExit(main())
