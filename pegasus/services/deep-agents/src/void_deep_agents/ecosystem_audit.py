"""CLI entry: ecosystem audit OR E2E fleet confirmation."""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

from void_deep_agents.factory import create_e2e_fleet_agent, create_ecosystem_auditor
from void_deep_agents.fleet import fleet_names
from void_deep_agents.paths import (
    default_memory_paths,
    default_skill_paths,
    fleet_filesystem_backend,
    gap_register_glob,
    pegasusx_root,
    probe_filesystem_backend,
    surfaces_registry_path,
)
from void_deep_agents.subagents import panel_names


def _extract_json_array(text: str) -> list | None:
    """Best-effort extract of a JSON findings array from model output."""
    fence = re.search(r"```json\s*(\[[\s\S]*?\])\s*```", text)
    raw = fence.group(1) if fence else None
    if raw is None:
        matches = list(re.finditer(r"\[[\s\S]*\]", text))
        if not matches:
            return None
        raw = matches[-1].group(0)
    try:
        data = json.loads(raw)
    except json.JSONDecodeError:
        return None
    return data if isinstance(data, list) else None


def _parse_panels(raw: list[str]) -> list[str] | None:
    if not raw:
        return None
    flat: list[str] = []
    for chunk in raw:
        flat.extend(p.strip() for p in chunk.split(",") if p.strip())
    return flat


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="PegasusX Deep Agents — ecosystem audit or E2E fleet gate",
    )
    parser.add_argument(
        "message",
        nargs="?",
        default=None,
        help="Audit / fleet task (default depends on --full / --fleet)",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Print resolved paths/skills/memory/panels and exit (no LLM call)",
    )
    parser.add_argument(
        "--fleet",
        action="store_true",
        help="Run E2E delivery fleet (confirm until all specialists CONFIRMED)",
    )
    parser.add_argument(
        "--full",
        action="store_true",
        help="Instruct orchestrator to run all specialist panels/agents",
    )
    parser.add_argument(
        "--panel",
        action="append",
        default=[],
        help="Audit panel name(s) or comma-separated list (repeatable). Default: all.",
    )
    parser.add_argument(
        "--agent",
        action="append",
        default=[],
        dest="fleet_agents",
        help="With --fleet: restrict fleet agents (repeatable / comma-separated).",
    )
    parser.add_argument(
        "--read-only-fleet",
        action="store_true",
        help="With --fleet: deny code writes (reports still writable under /fleet/)",
    )
    parser.add_argument(
        "--json-out",
        type=str,
        default="",
        help="Write extracted findings JSON array to this path (audit mode)",
    )
    args = parser.parse_args(argv)

    panels = _parse_panels(args.panel)
    fleet_agents = _parse_panels(args.fleet_agents)

    if args.dry_run:
        print("pegasusx_root:", pegasusx_root())
        print("mode:", "fleet" if args.fleet else "audit")
        if args.fleet:
            print("fs_backend: CompositeBackend(pegasusX + /skills/ + /fleet/)")
        else:
            print("fs_backend: CompositeBackend(default=pegasusX, routes=/skills/)")
        print("fs_virtual_code: /apps /docs /.agents")
        print("fs_virtual_skills: /skills/")
        if args.fleet:
            print("fs_virtual_fleet: /fleet/")
        print("surfaces:", surfaces_registry_path())
        print("gap_docs:", gap_register_glob())
        print("memory:")
        for m in default_memory_paths(virtual=True):
            print(" -", m)
        print("skills:")
        for s in default_skill_paths(virtual=True):
            print(" -", s)
        if args.fleet:
            print("fleet_agents:")
            active = fleet_agents if fleet_agents is not None else fleet_names()
            for p in active:
                print(" -", p)
        else:
            print("panels:")
            active = panels if panels is not None else panel_names()
            for p in active:
                print(" -", p)
            if panels is not None:
                print("panels_filter:", ",".join(panels))
        print("fs_probe:")
        probe = probe_filesystem_backend(
            fleet_filesystem_backend() if args.fleet else None
        )
        for k, v in probe.items():
            print(f" - {k}: {v}")
        bad = [
            k
            for k, v in probe.items()
            if k not in {"pegasusx_root", "skills_route"} and v != "ok"
        ]
        if bad:
            print("fs_probe_FAILED:", ",".join(bad))
            return 1
        print("fs_probe_ok: true")
        return 0

    if args.fleet:
        message = args.message
        if message is None:
            message = (
                "Deliver the requested feature end-to-end. "
                if not args.full
                else ""
            ) + (
                "Run the FULL E2E fleet. Loop until every specialist writes "
                "CONFIRMED under /fleet/reports/ and /fleet/STATUS.md shows "
                "FLEET_GATE: PASS. Cover implement + wire + business + technical. "
                "Do not stop early."
            )
        elif args.full:
            message = (
                "FULL E2E fleet. Delegate to every fleet agent, loop until unanimous "
                "CONFIRMED, then return FLEET_GATE.\n\n"
                + message
            )

        agent = create_e2e_fleet_agent(
            agents=fleet_agents,
            allow_code_writes=not args.read_only_fleet,
        )
        result = agent.invoke({"messages": [{"role": "user", "content": message}]})
        messages = result.get("messages") or []
        if not messages:
            print("No messages in agent result", file=sys.stderr)
            return 1
        last = messages[-1]
        content = getattr(last, "content", last)
        print(content if isinstance(content, str) else str(content))
        return 0

    message = args.message
    if message is None:
        if args.full or panels is None:
            message = (
                "Run a FULL ecosystem audit across all specialist panels. "
                "Cover business logic, role parity, data-flow, money/fiscal, "
                "kafka/outbox, redis, code quality, architecture, security/tenancy, "
                "cloud, client contracts, and gap-register sync. "
                "Respect resolved_gap_ids. Cite evidence. End with a JSON findings array."
            )
        else:
            message = (
                f"Audit using panels only: {', '.join(panels)}. "
                "Evidence-first. End with a JSON findings array."
            )
    elif args.full:
        message = (
            "FULL multi-panel audit. Delegate to every specialist panel, then merge.\n\n"
            + message
        )

    agent = create_ecosystem_auditor(panels=panels)
    result = agent.invoke(
        {"messages": [{"role": "user", "content": message}]},
    )
    messages = result.get("messages") or []
    if not messages:
        print("No messages in agent result", file=sys.stderr)
        return 1
    last = messages[-1]
    content = getattr(last, "content", last)
    text = content if isinstance(content, str) else str(content)
    print(text)

    if args.json_out:
        findings = _extract_json_array(text)
        out_path = Path(args.json_out)
        out_path.parent.mkdir(parents=True, exist_ok=True)
        if findings is None:
            print(
                f"warning: no JSON findings array found; writing empty list to {out_path}",
                file=sys.stderr,
            )
            findings = []
        out_path.write_text(json.dumps(findings, indent=2) + "\n", encoding="utf-8")
        print(f"wrote_findings: {out_path} ({len(findings)} items)", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
