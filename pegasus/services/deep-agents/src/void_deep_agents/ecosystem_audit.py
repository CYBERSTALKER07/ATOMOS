"""CLI entry: run a one-shot ecosystem quality audit with Deep Agents."""

from __future__ import annotations

import argparse
import sys

from void_deep_agents.factory import create_ecosystem_auditor
from void_deep_agents.paths import (
    default_memory_paths,
    default_skill_paths,
    gap_register_glob,
    pegasusx_root,
    surfaces_registry_path,
)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="PegasusX ecosystem auditor (LangChain Deep Agents)",
    )
    parser.add_argument(
        "message",
        nargs="?",
        default=(
            "Summarize how to audit one feature end-to-end using the coverage "
            "rule. List skill names available and where surfaces.yaml lives. "
            "Do not invent gap statuses."
        ),
        help="Audit question or task",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Print resolved paths/skills/memory and exit (no LLM call)",
    )
    args = parser.parse_args(argv)

    if args.dry_run:
        print("pegasusx_root:", pegasusx_root())
        print("surfaces:", surfaces_registry_path())
        print("gap_docs:", gap_register_glob())
        print("memory:")
        for m in default_memory_paths():
            print(" -", m)
        print("skills:")
        for s in default_skill_paths():
            print(" -", s)
        return 0

    agent = create_ecosystem_auditor()
    result = agent.invoke(
        {"messages": [{"role": "user", "content": args.message}]},
    )
    messages = result.get("messages") or []
    if not messages:
        print("No messages in agent result", file=sys.stderr)
        return 1
    last = messages[-1]
    print(getattr(last, "content", last))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
