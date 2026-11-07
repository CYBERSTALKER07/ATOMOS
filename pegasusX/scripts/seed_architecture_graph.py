#!/usr/bin/env python3
"""
seed_architecture_graph.py — Ingest PegasusX architecture-graph.json into Memgraph/Neo4j.

Reads pegasusX/context/architecture-graph.json (88 nodes, 160 edges) and loads it into
Memgraph as the authoritative top-level architectural backbone.
"""

import argparse
import json
import os
import sys
import time
from typing import Any, Dict, List


def load_graph_data(file_path: str) -> Dict[str, Any]:
    if not os.path.exists(file_path):
        raise FileNotFoundError(f"Architecture graph file not found at: {file_path}")
    with open(file_path, "r", encoding="utf-8") as f:
        return json.load(f)


def generate_cypher_statements(data: Dict[str, Any]) -> List[str]:
    statements: List[str] = [
        "// Indexes for fast lookup",
        "CREATE INDEX ON :ArchitectureNode(id);",
        "CREATE INDEX ON :ArchitectureNode(kind);",
        "CREATE INDEX ON :ArchitectureNode(path);",
    ]

    nodes = data.get("nodes", [])
    for node in nodes:
        node_id = node.get("id", "").replace("'", "\\'")
        kind = node.get("kind", "").replace("'", "\\'")
        language = node.get("language", "").replace("'", "\\'")
        path = node.get("path", "").replace("'", "\\'")
        stmt = (
            f"MERGE (n:ArchitectureNode {{id: '{node_id}'}}) "
            f"SET n.kind = '{kind}', n.language = '{language}', n.path = '{path}';"
        )
        statements.append(stmt)

    edges = data.get("edges", [])
    for edge in edges:
        from_id = edge.get("from", "").replace("'", "\\'")
        to_id = edge.get("to", "").replace("'", "\\'")
        kind = edge.get("kind", "").replace("'", "\\'")
        rel_type = "ARCH_REL"
        stmt = (
            f"MATCH (a:ArchitectureNode {{id: '{from_id}'}}), (b:ArchitectureNode {{id: '{to_id}'}}) "
            f"MERGE (a)-[r:{rel_type} {{kind: '{kind}'}}]->(b);"
        )
        statements.append(stmt)

    return statements


def execute_cypher(bolt_url: str, user: str, password: str, statements: List[str]) -> None:
    try:
        from neo4j import GraphDatabase
    except ImportError:
        print("[ERROR] 'neo4j' Python package not found. Install via: pip install neo4j or uv pip install neo4j", file=sys.stderr)
        sys.exit(1)

    print(f"[*] Connecting to graph database at {bolt_url}...")
    auth = (user, password) if user or password else None
    driver = GraphDatabase.driver(bolt_url, auth=auth)

    start_time = time.time()
    node_count = 0
    edge_count = 0

    with driver.session() as session:
        for stmt in statements:
            if stmt.startswith("//"):
                continue
            try:
                session.run(stmt)
                if "MERGE (n:ArchitectureNode" in stmt:
                    node_count += 1
                elif "MERGE (a)-[r:ARCH_REL" in stmt:
                    edge_count += 1
            except Exception as e:
                # Log non-fatal constraint/index warnings
                if "already exists" in str(e).lower():
                    continue
                print(f"[WARN] Error executing: {stmt[:60]}... -> {e}")

    driver.close()
    elapsed = time.time() - start_time
    print(f"[SUCCESS] Seeded {node_count} ArchitectureNodes and {edge_count} ARCH_REL edges in {elapsed:.2f}s.")


def main():
    parser = argparse.ArgumentParser(description="Seed PegasusX architecture-graph.json into Memgraph")
    parser.add_argument(
        "--graph-file",
        default=os.path.join(os.path.dirname(__file__), "../context/architecture-graph.json"),
        help="Path to architecture-graph.json",
    )
    parser.add_argument(
        "--bolt-url",
        default=os.getenv("MEMGRAPH_BOLT_URL", "bolt://localhost:7687"),
        help="Memgraph/Neo4j Bolt URL",
    )
    parser.add_argument("--user", default=os.getenv("MEMGRAPH_USER", ""), help="Database username")
    parser.add_argument("--password", default=os.getenv("MEMGRAPH_PASSWORD", ""), help="Database password")
    parser.add_argument(
        "--export-cypher",
        default="",
        help="Export all Cypher commands to a .cypher file instead of or in addition to connecting",
    )
    parser.add_argument("--dry-run", action="store_true", help="Print stats without executing")

    args = parser.parse_args()
    data = load_graph_data(args.graph_file)
    statements = generate_cypher_statements(data)

    print(f"[*] Loaded {len(data.get('nodes', []))} nodes and {len(data.get('edges', []))} edges from {args.graph_file}")

    if args.export_cypher:
        with open(args.export_cypher, "w", encoding="utf-8") as f:
            f.write("\n".join(statements) + "\n")
        print(f"[+] Exported {len(statements)} Cypher statements to {args.export_cypher}")

    if args.dry_run:
        print("[*] Dry run complete. No database mutations made.")
        return

    if not args.export_cypher or args.bolt_url:
        execute_cypher(args.bolt_url, args.user, args.password, statements)


if __name__ == "__main__":
    main()
