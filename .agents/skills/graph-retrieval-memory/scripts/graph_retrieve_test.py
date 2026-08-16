#!/usr/bin/env python3
import json
import subprocess
import sys
from pathlib import Path

SCRIPT = Path(__file__).with_name("graph_retrieve.py")


def test_order_hits_backend():
    out = subprocess.check_output(
        [sys.executable, str(SCRIPT), "-q", "order fiscal", "--json", "--hops", "1"],
        text=True,
    )
    data = json.loads(out)
    assert data["generatedAt"] is None
    assert "routing-index-not-status" in data["honesty"]
    ids = {h["id"] for h in data["hits"]}
    assert "order-service" in ids or "backend-go" in ids, ids


if __name__ == "__main__":
    test_order_hits_backend()
    print("ok")
