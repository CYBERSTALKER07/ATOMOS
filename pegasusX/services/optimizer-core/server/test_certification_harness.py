"""P2-2 optimizer certification harness — JSON corpus under testdata/cert/.

Loads every fixture and asserts invariant expectations. Extends the four
smoke tests in test_contract_solver.py with a durable benchmark corpus.
"""

from __future__ import annotations

import json
from pathlib import Path

import pytest

from contract_solver import solve_contract

CERT_DIR = Path(__file__).resolve().parents[1] / "testdata" / "cert"


def _fixtures() -> list[Path]:
    files = sorted(CERT_DIR.glob("*.json"))
    assert files, f"no certification fixtures in {CERT_DIR}"
    return files


@pytest.mark.parametrize("path", _fixtures(), ids=lambda p: p.stem)
def test_certification_fixture(path: Path):
    fixture = json.loads(path.read_text())
    expect = fixture.get("expect") or {}
    resp = solve_contract(fixture["request"])
    assert resp.get("source") == "OR_TOOLS_VRP"
    assert "routes" in resp and "orphans" in resp

    placed = sum(len(r.get("stops") or []) for r in resp["routes"])
    orphans = resp.get("orphans") or []

    if "min_routes" in expect:
        assert len(resp["routes"]) >= int(expect["min_routes"])
    if "min_placed" in expect:
        assert placed >= int(expect["min_placed"])
    if "max_placed" in expect:
        assert placed <= int(expect["max_placed"])
    if "min_orphans" in expect:
        assert len(orphans) >= int(expect["min_orphans"])
    if "max_stops_per_route" in expect:
        cap = int(expect["max_stops_per_route"])
        for route in resp["routes"]:
            assert len(route.get("stops") or []) <= cap

    if expect.get("prefer_nearest"):
        by_vehicle = {
            r["vehicle_id"]: [s["order_id"] for s in (r.get("stops") or [])]
            for r in resp["routes"]
        }
        if "depot-a" in by_vehicle and "depot-b" in by_vehicle:
            assert "near-a" in by_vehicle["depot-a"]
            assert "near-b" in by_vehicle["depot-b"]
        else:
            assert placed >= 1
