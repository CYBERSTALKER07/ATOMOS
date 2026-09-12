#!/usr/bin/env python3
"""Plain LangChain agent via create_void_langchain_agent.

Usage:

    source .venv/bin/activate
    export XAI_API_KEY=...
    python examples/langchain_agent.py
"""

from __future__ import annotations

import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "src"))

from void_deep_agents import create_void_langchain_agent  # noqa: E402


def add(a: int, b: int) -> int:
    """Add two integers."""
    return a + b


def main() -> None:
    agent = create_void_langchain_agent(
        tools=[add],
        system_prompt="You are a concise math helper. Use tools for arithmetic.",
    )
    result = agent.invoke(
        {"messages": [{"role": "user", "content": "What is 17 + 25?"}]}
    )
    print(result["messages"][-1].content)


if __name__ == "__main__":
    main()
