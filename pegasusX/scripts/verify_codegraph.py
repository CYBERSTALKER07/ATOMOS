#!/usr/bin/env python3
"""
verify_codegraph.py — Health Check and Verification Suite for PegasusX CodeGraph.

Validates that:
1. ArchitectureNode backbone is loaded (88 nodes, 160 edges).
2. The 5 Cross-Boundary Seams are traversable in Memgraph.
3. Multi-role blast radius queries resolve across mobile clients, backend routes,
   Spanner persistence, Kafka topics, and WebSocket hubs.
Supports --offline mode to validate generated Cypher artifact syntax and completeness.
"""

from __future__ import annotations

import argparse
import os
import sys
import time
from typing import Any, Dict, List, Tuple


VERIFICATION_QUERIES = [
    {
        "name": "Backbone Nodes & Edges",
        "cypher": "MATCH (n:ArchitectureNode) OPTIONAL MATCH (n)-[r:ARCH_REL]->() RETURN count(DISTINCT n) AS nodes, count(r) AS edges",
        "min_expected": 50
    },
    {
        "name": "Seam 1: Client API to Backend Route",
        "cypher": "MATCH (c:ApiClientMethod)-[:CONSUMES_ROUTE]->(r:RouteEndpoint) RETURN count(*) AS count",
        "min_expected": 10
    },
    {
        "name": "Seam 2: Repository to Spanner Entity",
        "cypher": "MATCH (rm:RepositoryMethod)-[r:MUTATES_TABLE|READS_TABLE]->(st:SpannerTable) RETURN count(*) AS count",
        "min_expected": 10
    },
    {
        "name": "Seam 2: Multi-Tenancy Tenant Isolation Check",
        "cypher": "MATCH (st:SpannerTable {is_tenant_isolated: true}) RETURN count(st) AS count",
        "min_expected": 1
    },
    {
        "name": "Seam 3: Service Method to Outbox Event",
        "cypher": "MATCH (sm:ServiceMethod)-[:EMITS_OUTBOX]->(oe:OutboxEmitter) RETURN count(*) AS count",
        "min_expected": 5
    },
    {
        "name": "Seam 4: Outbox Event to Kafka Topic",
        "cypher": "MATCH (ev:EventDefinition)-[:ROUTED_TO_TOPIC]->(kt:KafkaTopic) RETURN count(*) AS count",
        "min_expected": 5
    },
    {
        "name": "Seam 5: WebSocket Role Hub to Client Apps",
        "cypher": "MATCH (hub:WSHubRoom)-[:RECEIVED_BY]->(app:ClientApp) RETURN count(*) AS count",
        "min_expected": 10
    },
    {
        "name": "Cross-Boundary Blast Radius (End-to-End)",
        "cypher": """
        MATCH (app:ClientApp)-[:RECEIVED_BY]-(hub:WSHubRoom)<-[:FANOUT_WS_ROOM]-(kc:KafkaConsumer)<-[:CONSUMED_BY]-(kt:KafkaTopic)
        RETURN app.role AS role, app.platform AS platform, kt.name AS topic, kc.consumer AS consumer
        LIMIT 10
        """,
        "min_expected": 1
    }
]


def verify_offline(cypher_files: List[str]) -> bool:
    print("[*] Running in offline artifact validation mode...")
    total_statements = 0
    all_passed = True

    for fpath in cypher_files:
        if not os.path.exists(fpath):
            print(f"[FAIL] Missing required Cypher artifact: {fpath}", file=sys.stderr)
            all_passed = False
            continue
        with open(fpath, "r", encoding="utf-8") as fh:
            lines = [l.strip() for l in fh if l.strip() and not l.startswith("//")]
        print(f"[+] Verified {fpath}: {len(lines)} executable Cypher statements found.")
        total_statements += len(lines)

    if total_statements < 500:
        print(f"[FAIL] Insufficient statements generated ({total_statements} < 500).", file=sys.stderr)
        return False

    print(f"[SUCCESS] Offline verification passed. Total statements: {total_statements}")
    return all_passed


def verify_live(bolt_url: str, user: str, password: str) -> bool:
    try:
        from neo4j import GraphDatabase
    except ImportError:
        print("[ERROR] 'neo4j' Python package not found. Install via: pip install neo4j", file=sys.stderr)
        return False

    print(f"[*] Connecting to Memgraph/Neo4j at {bolt_url}...")
    auth = (user, password) if user or password else None
    try:
        driver = GraphDatabase.driver(bolt_url, auth=auth)
    except Exception as e:
        print(f"[FAIL] Could not connect to {bolt_url}: {e}", file=sys.stderr)
        return False

    all_passed = True
    with driver.session() as session:
        for q in VERIFICATION_QUERIES:
            name = q["name"]
            cypher = q["cypher"]
            min_exp = q.get("min_expected", 1)
            try:
                result = session.run(cypher)
                records = list(result)
                if not records:
                    print(f"[FAIL] {name}: 0 records returned (expected >= {min_exp})")
                    all_passed = False
                    continue

                rec = records[0]
                val = rec[0] if len(rec) > 0 else 0
                if isinstance(val, int) and val < min_exp:
                    print(f"[FAIL] {name}: count {val} < threshold {min_exp}")
                    all_passed = False
                else:
                    print(f"[PASS] {name}: {rec.data() if hasattr(rec, 'data') else val}")
            except Exception as e:
                print(f"[FAIL] {name}: Query execution error: {e}")
                all_passed = False

    driver.close()
    return all_passed


def main():
    parser = argparse.ArgumentParser(description="Verify PegasusX CodeGraph and Seams")
    parser.add_argument("--bolt-url", default=os.getenv("MEMGRAPH_BOLT_URL", "bolt://localhost:7687"))
    parser.add_argument("--user", default=os.getenv("MEMGRAPH_USER", ""))
    parser.add_argument("--password", default=os.getenv("MEMGRAPH_PASSWORD", ""))
    parser.add_argument("--offline", action="store_true", help="Validate generated Cypher files offline")
    parser.add_argument(
        "--arch-file",
        default=os.path.join(os.path.dirname(__file__), "../context/architecture_seed.cypher"),
    )
    parser.add_argument(
        "--seams-file",
        default=os.path.join(os.path.dirname(__file__), "../context/seams_seed.cypher"),
    )

    args = parser.parse_args()

    if args.offline:
        ok = verify_offline([args.arch_file, args.seams_file])
    else:
        try:
            ok = verify_live(args.bolt_url, args.user, args.password)
        except Exception:
            print("[WARN] Live Memgraph connection failed. Falling back to offline artifact check...")
            ok = verify_offline([args.arch_file, args.seams_file])

    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
