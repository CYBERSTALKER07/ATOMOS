#!/usr/bin/env python3
"""Minimal Deep Agents example for V.O.I.D.

Usage (from pegasus/services/deep-agents):

    source .venv/bin/activate
    export XAI_API_KEY=...
    python examples/hello_deep_agent.py
"""

from __future__ import annotations

import sys
from pathlib import Path

# Allow running without `pip install -e .`
ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "src"))

from void_deep_agents import create_void_deep_agent  # noqa: E402


def get_repo_hint() -> str:
    """Return a short description of this monorepo for the agent."""
    return (
        "V.O.I.D monorepo: pegasus (Go backend, Spanner, Kafka) + "
        "pegasusX (portals, native apps). Python Deep Agents live at "
        "pegasus/services/deep-agents."
    )


def main() -> None:
    agent = create_void_deep_agent(
        tools=[get_repo_hint],
        system_prompt=(
            "You are a helpful assistant for the V.O.I.D monorepo. "
            "Use tools when useful. Keep answers short."
        ),
    )

    result = agent.invoke(
        {
            "messages": [
                {
                    "role": "user",
                    "content": "What monorepo am I in? Use the get_repo_hint tool.",
                }
            ]
        }
    )
    print(result["messages"][-1].content)


if __name__ == "__main__":
    main()
