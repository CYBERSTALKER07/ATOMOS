"""Minimal CLI for the V.O.I.D / PegasusX deep agent."""

from __future__ import annotations

import argparse
import sys

from void_deep_agents.ecosystem_audit import main as ecosystem_main
from void_deep_agents.factory import create_void_deep_agent


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
        help="Use ecosystem auditor prompt + pegasusX memory/skills",
    )
    parser.add_argument(
        "--no-tools",
        action="store_true",
        help="Do not register the demo echo tool (non-ecosystem mode)",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="With --ecosystem: print paths only",
    )
    args = parser.parse_args(argv)

    if args.ecosystem or args.dry_run:
        eco_argv = [args.message]
        if args.dry_run:
            eco_argv = ["--dry-run"]
        return ecosystem_main(eco_argv)

    tools = [] if args.no_tools else [_echo_tool]
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
