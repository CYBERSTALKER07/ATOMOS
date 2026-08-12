#!/usr/bin/env python3
"""Plain LangChain agent (without Deep Agents harness) for comparison.

Usage:

    source .venv/bin/activate
    export XAI_API_KEY=...
    python examples/langchain_agent.py
"""

from __future__ import annotations

import os
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "src"))

from dotenv import load_dotenv
from langchain.agents import create_agent
from langchain_xai import ChatXAI

load_dotenv()


def add(a: int, b: int) -> int:
    """Add two integers."""
    return a + b


def main() -> None:
    api_key = os.getenv("XAI_API_KEY")
    if not api_key:
        raise SystemExit("Set XAI_API_KEY first (see .env.example)")

    model = ChatXAI(
        model=os.getenv("DEEP_AGENTS_MODEL", "grok-4.5"),
        api_key=api_key,
        temperature=0,
    )
    agent = create_agent(
        model=model,
        tools=[add],
        system_prompt="You are a concise math helper. Use tools for arithmetic.",
    )
    result = agent.invoke(
        {"messages": [{"role": "user", "content": "What is 17 + 25?"}]}
    )
    print(result["messages"][-1].content)


if __name__ == "__main__":
    main()
