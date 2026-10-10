#!/usr/bin/env python3
"""E2E fleet agent — specialists confirm until FLEET_GATE: PASS.

Usage (from pegasus/services/deep-agents):

    source .venv/bin/activate
    export XAI_API_KEY=...
    # optional: persist reports on disk
    # export FLEET_HOST_DIR=/tmp/pegasusx-fleet
    python examples/e2e_fleet_agent.py "Wire retailer AR pay-down end-to-end"
"""

from __future__ import annotations

import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "src"))

from void_deep_agents import create_e2e_fleet_agent  # noqa: E402


def main() -> None:
    task = (
        " ".join(sys.argv[1:])
        if len(sys.argv) > 1
        else (
            "Dry rehearsal: map the E2E path for order creation coverage "
            "(Spanner→outbox→consumer→clients). Agents must live in the codebase, "
            "write /fleet reports, and loop until all CONFIRMED or blockers listed. "
            "Prefer --read-only style: do not invent patches; cite real gaps."
        )
    )
    agent = create_e2e_fleet_agent(allow_code_writes=False)
    result = agent.invoke({"messages": [{"role": "user", "content": task}]})
    print(result["messages"][-1].content)


if __name__ == "__main__":
    main()
