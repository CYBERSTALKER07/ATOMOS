"""Minimal CLI for the V.O.I.D / PegasusX deep agent."""

from __future__ import annotations

import argparse
import sys

from void_deep_agents.ecosystem_audit import main as ecosystem_main
from void_deep_agents.factory import create_void_deep_agent, create_void_langchain_agent


def _echo_tool(text: str) -> str:
    """Echo a string back (smoke-test tool)."""
    return text


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Run a V.O.I.D Deep Agent turn")
    parser.add_argument(
        "message",
        nargs="?",
        default="Reply with exactly: deep-agents-ok",
        help="User message to send the agent",
    )
    parser.add_argument(
        "--ecosystem",
        action="store_true",
        help="Use ecosystem auditor orchestrator + specialist panels",
    )
    parser.add_argument(
        "--fleet",
        action="store_true",
        help="Use E2E fleet (confirm until all specialists CONFIRMED)",
    )
    parser.add_argument(
        "--langchain",
        action="store_true",
        help="Use plain LangChain create_agent (no Deep Agents FS/subagents)",
    )
    parser.add_argument(
        "--no-tools",
        action="store_true",
        help="Do not register the demo echo tool (non-ecosystem mode)",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="With --ecosystem/--fleet: print paths/panels only",
    )
    parser.add_argument(
        "--full",
        action="store_true",
        help="With --ecosystem/--fleet: full multi-panel/agent run",
    )
    parser.add_argument(
        "--panel",
        action="append",
        default=[],
        help="With --ecosystem: restrict panels (repeatable / comma-separated)",
    )
    parser.add_argument(
        "--agent",
        action="append",
        default=[],
        dest="fleet_agents",
        help="With --fleet: restrict fleet agents (repeatable / comma-separated)",
    )
    parser.add_argument(
        "--read-only-fleet",
        action="store_true",
        help="With --fleet: deny code writes (reports still writable)",
    )
    parser.add_argument(
        "--json-out",
        type=str,
        default="",
        help="With --ecosystem: write findings JSON to path",
    )
    args = parser.parse_args(argv)

    if (
        args.ecosystem
        or args.fleet
        or args.dry_run
        or args.full
        or args.panel
        or args.fleet_agents
        or args.json_out
        or args.read_only_fleet
    ):
        eco_argv: list[str] = []
        if args.dry_run:
            eco_argv.append("--dry-run")
        if args.fleet:
            eco_argv.append("--fleet")
        if args.full:
            eco_argv.append("--full")
        if args.read_only_fleet:
            eco_argv.append("--read-only-fleet")
        for p in args.panel:
            eco_argv.extend(["--panel", p])
        for a in args.fleet_agents:
            eco_argv.extend(["--agent", a])
        if args.json_out:
            eco_argv.extend(["--json-out", args.json_out])
        if args.message and not args.dry_run:
            eco_argv.append(args.message)
        return ecosystem_main(eco_argv)

    tools = [] if args.no_tools else [_echo_tool]
    if args.langchain:
        agent = create_void_langchain_agent(tools=tools)
    else:
        agent = create_void_deep_agent(tools=tools)

    result = agent.invoke(
        {"messages": [{"role": "user", "content": args.message}]},
    )
    messages = result.get("messages") or []
    if not messages:
        print("No messages in agent result", file=sys.stderr)
        return 1

    last = messages[-1]
    content = getattr(last, "content", last)
    print(content)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
