"""CLI entry: run a one-shot ecosystem quality audit with Deep Agents."""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

from void_deep_agents.factory import create_ecosystem_auditor
from void_deep_agents.paths import (
    default_memory_paths,
    default_skill_paths,
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
        # Last [...] block
        matches = list(re.finditer(r"\[[\s\S]*\]", text))
        if not matches:
            return None
        raw = matches[-1].group(0)
    try:
        data = json.loads(raw)
    except json.JSONDecodeError:
        return None
    return data if isinstance(data, list) else None


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="PegasusX ecosystem auditor orchestra (LangChain Deep Agents)",
    )
    parser.add_argument(
        "message",
        nargs="?",
        default=None,
        help="Audit question or task (default depends on --full)",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Print resolved paths/skills/memory/panels and exit (no LLM call)",
    )
    parser.add_argument(
        "--full",
        action="store_true",
        help="Instruct orchestrator to run all specialist panels",
    )
    parser.add_argument(
        "--panel",
        action="append",
        default=[],
        help="Panel name(s) or comma-separated list (repeatable). Default: all.",
    )
    parser.add_argument(
        "--json-out",
        type=str,
        default="",
        help="Write extracted findings JSON array to this path",
    )
    args = parser.parse_args(argv)

    panels: list[str] | None = None
    if args.panel:
        flat: list[str] = []
        for chunk in args.panel:
            flat.extend(p.strip() for p in chunk.split(",") if p.strip())
        panels = flat

    if args.dry_run:
        print("pegasusx_root:", pegasusx_root())
        print("fs_backend: CompositeBackend(default=pegasusX, routes=/skills/)")
        print("fs_virtual_code: /apps /docs /.agents")
        print("fs_virtual_skills: /skills/")
        print("surfaces:", surfaces_registry_path())
        print("gap_docs:", gap_register_glob())
        print("memory:")
        for m in default_memory_paths(virtual=True):
            print(" -", m)
        print("skills:")
        for s in default_skill_paths(virtual=True):
            print(" -", s)
        print("panels:")
        active = panels if panels is not None else panel_names()
        for p in active:
            print(" -", p)
        if panels is not None:
            print("panels_filter:", ",".join(panels))
        print("fs_probe:")
        probe = probe_filesystem_backend()
        for k, v in probe.items():
            print(f" - {k}: {v}")
        bad = [k for k, v in probe.items() if k not in {"pegasusx_root", "skills_route"} and v != "ok"]
        if bad:
            print("fs_probe_FAILED:", ",".join(bad))
            return 1
        print("fs_probe_ok: true")
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
