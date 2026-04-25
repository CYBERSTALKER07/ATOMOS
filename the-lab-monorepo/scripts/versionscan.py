#!/usr/bin/env python3

from __future__ import annotations

import argparse
import datetime as dt
import json
import os
import re
import subprocess
import sys
from pathlib import Path
from typing import Any


IGNORE_DIR_NAMES = {
    ".git",
    "node_modules",
    "dist",
    "build",
    ".next",
    "playwright-report",
    "Pods",
    "DerivedData",
}


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


def to_rel(path: Path, root: Path) -> str:
    return path.resolve().relative_to(root.resolve()).as_posix()


def load_text(path: Path) -> str:
    return path.read_text(encoding="utf-8", errors="ignore")


def is_ignored(path: Path, root: Path) -> bool:
    rel_parts = path.resolve().relative_to(root.resolve()).parts
    return any(part in IGNORE_DIR_NAMES for part in rel_parts)


def git_files(root: Path) -> list[Path]:
    code, out, _ = run_cmd(["git", "ls-files"], cwd=root, allow_fail=True)
    if code != 0:
        return walk_files(root)
    files: list[Path] = []
    for line in out.splitlines():
        if not line:
            continue
        path = root / line
        if path.exists() and path.is_file() and not is_ignored(path, root):
            files.append(path)
    return files


def walk_files(root: Path) -> list[Path]:
    files: list[Path] = []
    for path in root.rglob("*"):
        if not path.is_file():
            continue
        if is_ignored(path, root):
            continue
        files.append(path)
    return files


def scan_api_surface(root: Path, files: list[Path]) -> dict[str, Any]:
    backend_root = root / "apps/backend-go"
    endpoints: list[dict[str, Any]] = []

    http_handle_re = re.compile(r"\bhttp\.HandleFunc\(\s*\"(?P<path>/[^\"]+)\"")
    chi_route_re = re.compile(
        r"\b(?:r|router|rt|api|public|private|supplier|admin|fleet|ws|delivery|payment|warehouse|factory|driver)"
        r"\.(?P<method>HandleFunc|Get|Post|Put|Patch|Delete|MethodFunc|Options)\(\s*\"(?P<path>/[^\"]+)\""
    )

    role_guard_re = re.compile(
        r"Require(?:Role|AnyRole|WarehouseScope|FactoryScope|WarehouseOps|SupplierScope)|ResolveHomeNode|GetWarehouseOps|GetFactoryScope"
    )
    idempotency_re = re.compile(r"idempotency\.Guard\(")

    for path in files:
        if not path.as_posix().startswith(backend_root.as_posix()):
            continue
        if path.suffix != ".go":
            continue
        text = load_text(path)
        lines = text.splitlines()

        for idx, line in enumerate(lines, start=1):
            m_http = http_handle_re.search(line)
            m_chi = chi_route_re.search(line)
            if not m_http and not m_chi:
                continue

            route_path = ""
            method = "UNKNOWN"
            mount = ""
            if m_http:
                route_path = m_http.group("path")
                method = "UNKNOWN"
                mount = "http.HandleFunc"
            else:
                route_path = m_chi.group("path")
                method = m_chi.group("method").upper()
                mount = f"chi.{m_chi.group('method')}"

            context_end = min(len(lines), idx + 8)
            context = "\n".join(lines[idx - 1 : context_end])
            has_role_guard = bool(role_guard_re.search(context))
            has_idempotency = bool(idempotency_re.search(context))

            endpoints.append(
                {
                    "file": to_rel(path, root),
                    "line": idx,
                    "path": route_path,
                    "method": method,
                    "mount": mount,
                    "hasRoleGuardNearby": has_role_guard,
                    "hasIdempotencyNearby": has_idempotency,
                }
            )

    ws_paths = [e for e in endpoints if "/ws" in e["path"] or "websocket" in e["path"]]

    return {
        "endpointCount": len(endpoints),
        "websocketEndpointCount": len(ws_paths),
        "endpoints": endpoints,
    }


def parse_event_constants(events_file: Path, root: Path) -> dict[str, dict[str, Any]]:
    if not events_file.exists():
        return {}

    text = load_text(events_file)
    constants: dict[str, dict[str, Any]] = {}
    const_re = re.compile(r"\b(Event[A-Za-z0-9_]+)\s*=\s*\"([^\"]+)\"")

    for idx, line in enumerate(text.splitlines(), start=1):
        m = const_re.search(line)
        if not m:
            continue
        constants[m.group(1)] = {
            "eventName": m.group(2),
            "declFile": to_rel(events_file, root),
            "declLine": idx,
            "producerRefs": [],
            "consumerRefs": [],
            "otherRefs": [],
        }
    return constants


def classify_event_reference(line: str) -> str:
    if re.search(r"\bcase\s+Event[A-Za-z0-9_]+\b", line):
        return "consumer"
    if "outbox.Emit" in line:
        return "producer"
    if "EmitNotification(" in line:
        return "producer"
    if re.search(r"\.\s*WriteMessages\s*\(", line):
        return "producer"
    return "other"


def scan_event_graph(root: Path, files: list[Path]) -> dict[str, Any]:
    backend_root = root / "apps/backend-go"
    events_file = backend_root / "kafka/events.go"
    event_map = parse_event_constants(events_file, root)
    if not event_map:
        return {
            "eventCount": 0,
            "events": [],
            "producerOrphans": [],
            "consumerOrphans": [],
        }

    token_re = re.compile(r"\bEvent[A-Za-z0-9_]+\b")

    for path in files:
        if not path.as_posix().startswith(backend_root.as_posix()):
            continue
        if path.suffix != ".go":
            continue
        if path == events_file:
            continue

        lines = load_text(path).splitlines()
        rel = to_rel(path, root)
        for idx, line in enumerate(lines, start=1):
            for token in token_re.findall(line):
                if token not in event_map:
                    continue
                ref = {"file": rel, "line": idx}
                kind = classify_event_reference(line)
                if kind == "producer":
                    event_map[token]["producerRefs"].append(ref)
                elif kind == "consumer":
                    event_map[token]["consumerRefs"].append(ref)
                else:
                    event_map[token]["otherRefs"].append(ref)

    events: list[dict[str, Any]] = []
    producer_orphans: list[str] = []
    consumer_orphans: list[str] = []
    for token, meta in sorted(event_map.items()):
        producers = len(meta["producerRefs"])
        consumers = len(meta["consumerRefs"])
        if producers == 0:
            producer_orphans.append(token)
        if consumers == 0:
            consumer_orphans.append(token)

        events.append(
            {
                "constant": token,
                "eventName": meta["eventName"],
                "declFile": meta["declFile"],
                "declLine": meta["declLine"],
                "producerRefCount": producers,
                "consumerRefCount": consumers,
                "otherRefCount": len(meta["otherRefs"]),
                "producerRefs": meta["producerRefs"],
                "consumerRefs": meta["consumerRefs"],
            }
        )

    return {
        "eventCount": len(events),
        "events": events,
        "producerOrphans": producer_orphans,
        "consumerOrphans": consumer_orphans,
    }


def scan_ws_contracts(root: Path, files: list[Path], api_manifest: dict[str, Any]) -> dict[str, Any]:
    backend_ws_root = root / "apps/backend-go/ws"
    telemetry_hub_file = root / "apps/backend-go/telemetry/hub.go"
    client_ext = {".ts", ".tsx", ".js", ".jsx", ".kt", ".swift"}

    ws_handler_re = re.compile(r"\bfunc\s*(?:\([^)]*\)\s*)?(Handle[A-Za-z0-9_]*Connection|HandleWS|ServeWS)\s*\(")
    ws_server_auth_re = re.compile(r"ExtractTokenFromWSQuery|Authorization|auth\.Extract")

    ws_server_handlers: list[dict[str, Any]] = []
    ws_query_identity_sources: list[dict[str, Any]] = []
    ws_client_refs: list[dict[str, Any]] = []

    identity_qp_re = re.compile(
        r"(?:\?|&)(driver_id|retailer_id|supplier_id|warehouse_id|factory_id|home_node_id)="
    )
    ws_hint_re = re.compile(r"wss?://|/ws/|new\s+WebSocket|URLSessionWebSocketTask|OkHttpClient")

    for path in files:
        text = load_text(path)
        rel = to_rel(path, root)
        lines = text.splitlines()

        if path.as_posix().startswith(backend_ws_root.as_posix()) or path == telemetry_hub_file:
            for idx, line in enumerate(lines, start=1):
                if ws_handler_re.search(line):
                    window = "\n".join(lines[max(0, idx - 1) : min(len(lines), idx + 12)])
                    ws_server_handlers.append(
                        {
                            "file": rel,
                            "line": idx,
                            "symbol": line.strip(),
                            "hasAuthExtractionNearby": bool(ws_server_auth_re.search(window)),
                        }
                    )
                if re.search(r"\b(r\.URL\.Query\(|ExtractTokenFromWSQuery\(|Query\(\)\.Get\()", line):
                    ws_query_identity_sources.append(
                        {
                            "file": rel,
                            "line": idx,
                            "snippet": line.strip(),
                        }
                    )

        if path.suffix not in client_ext:
            continue
        if not rel.startswith("apps/"):
            continue
        if rel.startswith("apps/backend-go/"):
            continue

        for idx, line in enumerate(lines, start=1):
            if "ws" not in line.lower() and "/ws" not in line.lower():
                continue
            if ws_hint_re.search(line):
                ws_client_refs.append(
                    {
                        "file": rel,
                        "line": idx,
                        "snippet": line.strip(),
                        "hasIdentityQueryParam": bool(identity_qp_re.search(line)),
                    }
                )

    ws_paths = [e for e in api_manifest.get("endpoints", []) if "/ws" in e.get("path", "")]
    insecure_client_refs = [r for r in ws_client_refs if r["hasIdentityQueryParam"]]

    return {
        "serverEndpointCount": len(ws_paths),
        "serverHandlers": ws_server_handlers,
        "serverIdentityQueryReads": ws_query_identity_sources,
        "clientReferences": ws_client_refs,
        "clientIdentityQueryParamRefs": insecure_client_refs,
    }


def collect_json_keys_go(files: list[Path], root: Path) -> set[str]:
    keys: set[str] = set()
    tag_re = re.compile(r'json:"([^",\s]+)')
    for path in files:
        if path.suffix != ".go":
            continue
        rel = to_rel(path, root)
        if not rel.startswith("apps/backend-go/") and not rel.startswith("packages/"):
            continue
        for key in tag_re.findall(load_text(path)):
            if key and key != "-":
                keys.add(key)
    return keys


def collect_json_keys_ts(files: list[Path], root: Path) -> set[str]:
    keys: set[str] = set()
    key_re = re.compile(r'["\']([a-z][a-z0-9_]+)["\']\s*:')
    for path in files:
        if path.suffix not in {".ts", ".tsx", ".js", ".jsx"}:
            continue
        rel = to_rel(path, root)
        if not rel.startswith("apps/") and not rel.startswith("packages/"):
            continue
        for key in key_re.findall(load_text(path)):
            keys.add(key)
    return keys


def collect_json_keys_kotlin(files: list[Path], root: Path) -> set[str]:
    keys: set[str] = set()
    serial_re = re.compile(r'@SerialName\("([^"]+)"\)')
    for path in files:
        if path.suffix != ".kt":
            continue
        rel = to_rel(path, root)
        if not rel.startswith("apps/"):
            continue
        for key in serial_re.findall(load_text(path)):
            keys.add(key)
    return keys


def collect_json_keys_swift(files: list[Path], root: Path) -> set[str]:
    keys: set[str] = set()
    coding_key_re = re.compile(r'\bcase\s+\w+\s*=\s*"([^"]+)"')
    for path in files:
        if path.suffix != ".swift":
            continue
        rel = to_rel(path, root)
        if not rel.startswith("apps/"):
            continue
        for key in coding_key_re.findall(load_text(path)):
            keys.add(key)
    return keys


def scan_model_parity(root: Path, files: list[Path]) -> dict[str, Any]:
    go_keys = collect_json_keys_go(files, root)
    ts_keys = collect_json_keys_ts(files, root)
    kotlin_keys = collect_json_keys_kotlin(files, root)
    swift_keys = collect_json_keys_swift(files, root)

    client_union = ts_keys | kotlin_keys | swift_keys
    critical_re = re.compile(r"(_id$|_status$|_at$|_type$|_name$|_amount$|_currency$)")

    go_only_critical = sorted([k for k in go_keys if critical_re.search(k) and k not in client_union])

    return {
        "goJsonKeyCount": len(go_keys),
        "tsSnakeKeyCount": len(ts_keys),
        "kotlinSerialNameCount": len(kotlin_keys),
        "swiftCodingKeyCount": len(swift_keys),
        "sharedGoTsKeys": len(go_keys & ts_keys),
        "sharedGoKotlinKeys": len(go_keys & kotlin_keys),
        "sharedGoSwiftKeys": len(go_keys & swift_keys),
        "goOnlyCriticalKeys": go_only_critical,
    }


def add_finding(
    findings: list[dict[str, Any]],
    *,
    file: str,
    line: int,
    rule: str,
    message: str,
    snippet: str,
    severity: str = "error",
) -> None:
    findings.append(
        {
            "file": file,
            "line": line,
            "rule": rule,
            "severity": severity,
            "message": message,
            "snippet": snippet.strip(),
        }
    )


def is_allowed_write_messages_path(rel: str) -> bool:
    allowed = (
        rel.startswith("apps/backend-go/outbox/"),
        rel.startswith("apps/backend-go/kafka/"),
        rel.startswith("apps/backend-go/telemetry/"),
        rel.startswith("apps/ai-worker/"),
    )
    return any(allowed)


def is_allowed_apply_path(rel: str) -> bool:
    allowed = (
        rel.startswith("apps/backend-go/cmd/"),
        rel.startswith("apps/backend-go/tests/"),
        rel.endswith("_test.go"),
    )
    return any(allowed)


def scan_guardrails(root: Path, files: list[Path]) -> dict[str, Any]:
    findings: list[dict[str, Any]] = []

    write_call_re = re.compile(r"\.\s*WriteMessages\s*\(")
    apply_call_re = re.compile(r"\.\s*Apply\s*\(")
    identity_qp_re = re.compile(
        r"(?:\?|&)(driver_id|retailer_id|supplier_id|warehouse_id|factory_id|home_node_id)="
    )
    ws_hint_re = re.compile(r"wss?://|/ws/|new\s+WebSocket|URLSessionWebSocketTask|OkHttpClient")

    for path in files:
        rel = to_rel(path, root)
        lines = load_text(path).splitlines()

        if rel.startswith("apps/backend-go/") and path.suffix == ".go":
            for idx, line in enumerate(lines, start=1):
                if line.strip().startswith("//"):
                    continue

                if write_call_re.search(line) and not is_allowed_write_messages_path(rel):
                    add_finding(
                        findings,
                        file=rel,
                        line=idx,
                        rule="direct_write_messages",
                        message="Direct Kafka WriteMessages call outside outbox/relay or kafka/ infrastructure.",
                        snippet=line,
                    )

                if apply_call_re.search(line) and not is_allowed_apply_path(rel):
                    add_finding(
                        findings,
                        file=rel,
                        line=idx,
                        rule="apply_without_rwt",
                        message="Spanner Apply call detected in app path; prefer ReadWriteTransaction for mutable workflows.",
                        snippet=line,
                        severity="warning",
                    )

        if rel.startswith("apps/") and not rel.startswith("apps/backend-go/") and path.suffix in {
            ".ts",
            ".tsx",
            ".js",
            ".jsx",
            ".kt",
            ".swift",
        }:
            for idx, line in enumerate(lines, start=1):
                if identity_qp_re.search(line):
                    start = max(0, idx - 3)
                    end = min(len(lines), idx + 3)
                    window = "\n".join(lines[start:end])
                    if ws_hint_re.search(window):
                        add_finding(
                            findings,
                            file=rel,
                            line=idx,
                            rule="ws_identity_query_param",
                            message="WebSocket identity passed via query parameter; use token-based claims only.",
                            snippet=line,
                        )

    summary: dict[str, int] = {}
    for finding in findings:
        summary[finding["rule"]] = summary.get(finding["rule"], 0) + 1

    return {
        "findingCount": len(findings),
        "findingSummary": summary,
        "findings": findings,
    }


def build_scan(root: Path) -> dict[str, Any]:
    files = git_files(root)

    api_manifest = scan_api_surface(root, files)
    event_graph = scan_event_graph(root, files)
    ws_manifest = scan_ws_contracts(root, files, api_manifest)
    model_parity = scan_model_parity(root, files)
    guardrails = scan_guardrails(root, files)

    generated_at = dt.datetime.now(dt.timezone.utc).isoformat()

    return {
        "meta": {
            "generatedAt": generated_at,
            "repoRoot": str(root),
            "fileCount": len(files),
            "tool": "versionscan",
            "version": 1,
        },
        "apiManifest": api_manifest,
        "eventGraph": event_graph,
        "wsManifest": ws_manifest,
        "modelParity": model_parity,
        "guardrails": guardrails,
    }


def ensure_output_dir(output_dir: Path) -> None:
    output_dir.mkdir(parents=True, exist_ok=True)


def write_json(path: Path, payload: dict[str, Any]) -> None:
    path.write_text(json.dumps(payload, indent=2, sort_keys=False) + "\n", encoding="utf-8")


def write_outputs(scan: dict[str, Any], output_dir: Path) -> None:
    ensure_output_dir(output_dir)

    write_json(output_dir / "api-manifest.json", scan["apiManifest"])
    write_json(output_dir / "event-graph.json", scan["eventGraph"])
    write_json(output_dir / "ws-manifest.json", scan["wsManifest"])
    write_json(output_dir / "model-parity.json", scan["modelParity"])
    write_json(output_dir / "guardrails.json", scan["guardrails"])

    summary = {
        "meta": scan["meta"],
        "summary": {
            "endpoints": scan["apiManifest"]["endpointCount"],
            "wsEndpoints": scan["apiManifest"]["websocketEndpointCount"],
            "events": scan["eventGraph"]["eventCount"],
            "producerOrphans": len(scan["eventGraph"]["producerOrphans"]),
            "consumerOrphans": len(scan["eventGraph"]["consumerOrphans"]),
            "guardrailFindings": scan["guardrails"]["findingCount"],
            "goOnlyCriticalKeys": len(scan["modelParity"]["goOnlyCriticalKeys"]),
        },
    }
    write_json(output_dir / "report.json", summary)


def read_changed_files(root: Path) -> set[str]:
    changed: set[str] = set()

    env_list = os.environ.get("VERSIONSCAN_CHANGED_FILES", "").strip()
    if env_list:
        for item in env_list.split(","):
            candidate = item.strip()
            if candidate:
                changed.add(candidate)

    base_ref = os.environ.get("GITHUB_BASE_REF", "").strip()
    if base_ref:
        run_cmd(["git", "fetch", "origin", base_ref, "--depth=1"], cwd=root, allow_fail=True)
        code, out, _ = run_cmd(
            ["git", "diff", "--name-only", f"origin/{base_ref}...HEAD"],
            cwd=root,
            allow_fail=True,
        )
        if code == 0:
            for line in out.splitlines():
                if line.strip():
                    changed.add(line.strip())

    for diff_cmd in (
        ["git", "diff", "--name-only", "--diff-filter=ACMR"],
        ["git", "diff", "--name-only", "--cached", "--diff-filter=ACMR"],
    ):
        code, out, _ = run_cmd(diff_cmd, cwd=root, allow_fail=True)
        if code != 0:
            continue
        for line in out.splitlines():
            if line.strip():
                changed.add(line.strip())

    return changed


def filter_findings_for_enforcement(
    findings: list[dict[str, Any]],
    *,
    changed_only: bool,
    changed_files: set[str],
) -> list[dict[str, Any]]:
    block_rules = {
        "direct_write_messages",
        "ws_identity_query_param",
        "apply_without_rwt",
    }

    out: list[dict[str, Any]] = []
    for finding in findings:
        if finding["rule"] not in block_rules:
            continue
        if changed_only and finding["file"] not in changed_files:
            continue
        out.append(finding)
    return out


def print_report(scan: dict[str, Any]) -> None:
    summary = {
        "endpoints": scan["apiManifest"]["endpointCount"],
        "wsEndpoints": scan["apiManifest"]["websocketEndpointCount"],
        "events": scan["eventGraph"]["eventCount"],
        "producerOrphans": len(scan["eventGraph"]["producerOrphans"]),
        "consumerOrphans": len(scan["eventGraph"]["consumerOrphans"]),
        "guardrailFindings": scan["guardrails"]["findingCount"],
        "goOnlyCriticalKeys": len(scan["modelParity"]["goOnlyCriticalKeys"]),
    }
    print(json.dumps(summary, indent=2))


def command_scan(args: argparse.Namespace) -> int:
    root = Path(args.repo_root).resolve()
    output_dir = (root / args.output_dir).resolve()

    scan = build_scan(root)
    write_outputs(scan, output_dir)
    print_report(scan)
    return 0


def command_enforce(args: argparse.Namespace) -> int:
    root = Path(args.repo_root).resolve()
    output_dir = (root / args.output_dir).resolve()

    scan = build_scan(root)
    write_outputs(scan, output_dir)
    print_report(scan)

    findings = scan["guardrails"]["findings"]
    changed_files = read_changed_files(root) if args.changed_only else set()

    if args.changed_only and not changed_files:
        print("VersionScan enforce: no changed files detected; skipping block rules.")
        return 0

    blocking = filter_findings_for_enforcement(
        findings,
        changed_only=args.changed_only,
        changed_files=changed_files,
    )

    if not blocking:
        print("VersionScan enforce: passed.")
        return 0

    print("VersionScan enforce: blocking findings detected:")
    for finding in blocking:
        print(
            f"- {finding['rule']}: {finding['file']}:{finding['line']} :: {finding['message']}"
        )
    return 1


def build_arg_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="VersionScan manifest and guardrail scanner")
    sub = parser.add_subparsers(dest="command", required=True)

    for name in ("scan", "enforce"):
        p = sub.add_parser(name)
        p.add_argument("--repo-root", default=".", help="Repository root to scan")
        p.add_argument(
            "--output-dir",
            default=".versionscan",
            help="Output directory (relative to repo root)",
        )
        if name == "enforce":
            p.add_argument(
                "--changed-only",
                action="store_true",
                help="Enforce block rules only on changed files",
            )

    return parser


def main() -> int:
    parser = build_arg_parser()
    args = parser.parse_args()

    if args.command == "scan":
        return command_scan(args)
    if args.command == "enforce":
        return command_enforce(args)

    parser.error("unknown command")
    return 2


if __name__ == "__main__":
    sys.exit(main())
