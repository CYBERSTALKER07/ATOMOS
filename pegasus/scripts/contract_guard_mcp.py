#!/usr/bin/env python3

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

from guard_utils import added_lines_for_file, changed_files, ensure_git_repo, match_any


CONTRACT_TRIGGER_PATTERNS = [
    "pegasus/apps/backend-go/*routes/**",
    "pegasus/apps/backend-go/**/*routes*.go",
    "pegasus/apps/backend-go/**/kafka/events.go",
    "pegasus/apps/backend-go/webhookroutes/**",
    "pegasus/apps/backend-go/sync/**",
]

ROUTE_TRIGGER_PATTERNS = [
    "pegasus/apps/backend-go/*routes/**",
    "pegasus/apps/backend-go/**/*routes*.go",
]

ROUTE_DOC_SYNC_PATTERNS = [
    "pegasus/context/architecture.md",
    "pegasus/context/architecture-graph.json",
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

RETAILER_CLIENT_SURFACES = {
    "retailer-desktop": ["pegasus/apps/retailer-app-desktop/**"],
    "retailer-android": ["pegasus/apps/retailer-app-android/**"],
    "retailer-ios": ["pegasus/apps/retailer-app-ios/**"],
}

SUPPLIER_CLIENT_SURFACES = {
    "supplier-portal": ["pegasus/apps/admin-portal/**"],
}

DTO_SIGNAL_PATTERNS = [
    "pegasus/apps/backend-go/**/*.go",
    "pegasus/packages/types/**",
    "pegasus/packages/api-client/**",
    "pegasus/packages/validation/**",
]

DTO_CLIENT_PARITY_EXEMPT_PATTERNS = [
    "pegasus/apps/backend-go/kafka/events.go",
]

DTO_SIGNAL_RE = re.compile(
    r"\b(?:type\s+\w+\s+struct|interface\s+\w+|data\s+class|struct\s+\w+|"
    r"Codable|Serializable|SerialName|CodingKeys|json:\")"
)


def has_keyword_dto_signal(
    repo_root: Path,
    file_path: str,
    keyword: str,
    base_sha: str | None,
    head_sha: str | None,
) -> bool:
    if match_any(file_path, DTO_CLIENT_PARITY_EXEMPT_PATTERNS):
        return False

    lowered_keyword = keyword.lower()

    for _, line in added_lines_for_file(repo_root, file_path, base_sha, head_sha):
        stripped = line.strip()
        if not stripped or stripped.startswith("//") or stripped.startswith("*"):
            continue
        if lowered_keyword in stripped.lower() and DTO_SIGNAL_RE.search(stripped):
            return True

    return False


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

    route_changes = [path for path in files if match_any(path, ROUTE_TRIGGER_PATTERNS)]
    route_doc_changes = [path for path in files if match_any(path, ROUTE_DOC_SYNC_PATTERNS)]
    non_route_contract_changes = [
        path for path in trigger_changes if not match_any(path, ROUTE_TRIGGER_PATTERNS)
    ]
    shared_changes = [path for path in files if match_any(path, SHARED_CONTRACT_PATTERNS)]
    consumer_changes = [path for path in files if match_any(path, CONSUMER_SURFACE_PATTERNS)]
    context_sync_changes = [path for path in files if match_any(path, CONTEXT_SYNC_PATTERNS)]
    codebase_focus_changes = sorted(set(trigger_changes + shared_changes + consumer_changes))
    dto_signal_changes = [path for path in files if match_any(path, DTO_SIGNAL_PATTERNS)]

    missing_mcp_files = [
        path for path in MCP_REQUIRED_FILES if not (repo_root / path).exists()
    ]

    failures: list[str] = []

    if non_route_contract_changes and not shared_changes:
        failures.append(
            "Shared contract surface missing. Update at least one of: "
            "packages/types, packages/api-client, packages/validation, packages/optimizer-contract."
        )

    if non_route_contract_changes and not consumer_changes:
        failures.append(
            "Consumer surface missing. Update at least one consuming surface "
            "(web/mobile/worker/kafka-notification layer)."
        )

    if not context_sync_changes:
        failures.append(
            "MCP context sync missing. Update at least one architecture/technology context file."
        )

    if route_changes and not route_doc_changes:
        failures.append(
            "Route changes detected without route-doc sync. Update pegasus/context/architecture.md "
            "or pegasus/context/architecture-graph.json."
        )

    if context_sync_changes and len(codebase_focus_changes) < len(context_sync_changes):
        failures.append(
            "Codebase-first MCP policy violated. Contract-triggered diffs must rely primarily on "
            "real codebase surfaces; context docs are secondary verification. "
            f"codebase_focus={len(codebase_focus_changes)} context_sync={len(context_sync_changes)}."
        )

    retailer_dto_signals = [
        path
        for path in dto_signal_changes
        if has_keyword_dto_signal(
            repo_root, path, "retailer", args.base_sha, args.head_sha
        )
    ]
    supplier_dto_signals = [
        path
        for path in dto_signal_changes
        if has_keyword_dto_signal(
            repo_root, path, "supplier", args.base_sha, args.head_sha
        )
    ]

    if retailer_dto_signals:
        missing_surfaces = [
            surface
            for surface, patterns in RETAILER_CLIENT_SURFACES.items()
            if not any(match_any(path, patterns) for path in files)
        ]
        if missing_surfaces:
            failures.append(
                "Retailer DTO signals detected without cross-client model scan touches for: "
                + ", ".join(missing_surfaces)
                + "."
            )

    if supplier_dto_signals:
        missing_surfaces = [
            surface
            for surface, patterns in SUPPLIER_CLIENT_SURFACES.items()
            if not any(match_any(path, patterns) for path in files)
        ]
        if missing_surfaces:
            failures.append(
                "Supplier DTO signals detected without supplier portal model scan touches for: "
                + ", ".join(missing_surfaces)
                + "."
            )

    if missing_mcp_files:
        failures.append(
            "Missing required AST MCP engine file(s): " + ", ".join(missing_mcp_files)
        )

    print("contract-guard-mcp: trigger changes:")
    for path in trigger_changes:
        print(f"  - {path}")

    print(
        "contract-guard-mcp: context summary "
        f"(codebase_focus={len(codebase_focus_changes)}, context_sync={len(context_sync_changes)})"
    )
    if route_changes:
        print(f"contract-guard-mcp: route changes={len(route_changes)} route-doc-sync={len(route_doc_changes)}")
    if retailer_dto_signals:
        print(f"contract-guard-mcp: retailer dto signals={len(retailer_dto_signals)}")
    if supplier_dto_signals:
        print(f"contract-guard-mcp: supplier dto signals={len(supplier_dto_signals)}")

    if not failures:
        print(
            "contract-guard-mcp: contract, consumer, and codebase-first MCP context checks passed."
        )
        return 0

    print("\ncontract-guard-mcp: FAIL — contract governance violations detected.", file=sys.stderr)
    for item in failures:
        print(f"  - {item}", file=sys.stderr)

    return 1


if __name__ == "__main__":
    raise SystemExit(main())
