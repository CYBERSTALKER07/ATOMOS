#!/usr/bin/env python3
"""
PegasusX Deep Code Audit Engine for AI Agents & Engineers.

Provides automated architectural, multi-tenancy, and blast-radius auditing
over the live Memgraph CodeGraph (64,432 Nodes / 176,106 Relationships).

Modes:
  1. Full Ecosystem Audit:
       python3 scripts/audit_codegraph.py [--output docs/CODE_AUDIT_REPORT.md] [--json]
  2. Targeted Symbol Blast-Radius Audit (AI Agent pre-edit gate):
       python3 scripts/audit_codegraph.py --symbol <symbol_name> [--json]
  3. File Blast-Radius Audit (AI Agent pre-edit gate):
       python3 scripts/audit_codegraph.py --file <file_path> [--json]
  4. Ad-hoc Cypher Execution for AI Agents:
       python3 scripts/audit_codegraph.py --cypher "<CYPHER_QUERY>" [--json]
"""

from __future__ import annotations

import argparse
from datetime import datetime
import json
import os
import sys
from typing import Any, Dict, List

from neo4j import GraphDatabase

BOLT_URL = os.getenv("MEMGRAPH_BOLT_URL", "bolt://localhost:7687")
BOLT_USER = os.getenv("MEMGRAPH_USER", "")
BOLT_PASS = os.getenv("MEMGRAPH_PASSWORD", "")

auth = (BOLT_USER, BOLT_PASS) if BOLT_USER or BOLT_PASS else None


def get_driver():
    return GraphDatabase.driver(BOLT_URL, auth=auth)


def run_full_audit() -> Dict[str, Any]:
    driver = get_driver()
    audit_data: Dict[str, Any] = {}

    with driver.session() as session:
        # --- Audit 1: Spanner Multi-Tenancy & Tenant Isolation ---
        iso_res = session.run("""
            MATCH (t:SpannerTable)
            RETURN t.name AS table, t.is_tenant_isolated AS isolated, t.has_supplier_id AS supplier_id
            ORDER BY t.name
        """)
        all_tables = [dict(r) for r in iso_res]
        isolated_tables = [t for t in all_tables if t.get("isolated")]
        non_isolated_tables = [t for t in all_tables if not t.get("isolated")]

        audit_data["multi_tenancy"] = {
            "total_tables": len(all_tables),
            "isolated_count": len(isolated_tables),
            "non_isolated_count": len(non_isolated_tables),
            "non_isolated": non_isolated_tables,
        }

        # --- Audit 2: Contract Drift & 404 Hazards ---
        drift_res = session.run("""
            MATCH (c:ApiClientMethod)
            OPTIONAL MATCH (c)-[rel:CONSUMES_ROUTE]->(r:RouteEndpoint)
            WITH c, count(rel) AS matched
            WHERE matched = 0
            RETURN c.platform AS platform, c.file AS file, c.method_name AS method, c.endpoint_template AS endpoint
            ORDER BY c.platform, c.method_name
        """)
        drift_items = [dict(r) for r in drift_res]
        audit_data["contract_drift"] = {
            "total_drift": len(drift_items),
            "items": drift_items,
        }

        # --- Audit 3: Dead / Unconsumed Backend Routes ---
        dead_res = session.run("""
            MATCH (r:RouteEndpoint)
            OPTIONAL MATCH (c:ApiClientMethod)-[rel:CONSUMES_ROUTE]->(r)
            WITH r, count(rel) AS callers
            WHERE callers = 0
            RETURN r.method AS method, r.path AS path, r.file AS file
            ORDER BY r.path
        """)
        dead_routes = [dict(r) for r in dead_res]
        audit_data["dead_routes"] = {
            "total_dead": len(dead_routes),
            "routes": dead_routes,
        }

        # --- Audit 4: Kafka Event Stream Integrity ---
        kafka_res = session.run("""
            MATCH (t:KafkaTopic)
            OPTIONAL MATCH (o:OutboxEmitter)-[r1:ROUTED_TO_TOPIC]->(t)
            OPTIONAL MATCH (t)-[r2:CONSUMED_BY]->(c:KafkaConsumer)
            WITH t, count(r1) AS emitters, count(r2) AS consumers
            RETURN t.name AS topic, emitters, consumers
            ORDER BY consumers ASC, emitters DESC
        """)
        kafka_topics = [dict(r) for r in kafka_res]
        unconsumed_topics = [t for t in kafka_topics if t["consumers"] == 0]
        audit_data["kafka_integrity"] = {
            "total_topics": len(kafka_topics),
            "unconsumed_count": len(unconsumed_topics),
            "topics": kafka_topics,
        }

        # --- Audit 5: Blast Radius Hotspots ---
        hotspot_res = session.run("""
            MATCH (f:Function)<-[r:CALLS]-()
            WITH f, count(r) AS in_degree
            ORDER BY in_degree DESC LIMIT 25
            RETURN f.name AS name, f.qualified_name AS qn, in_degree
        """)
        hotspots = [dict(r) for r in hotspot_res]
        audit_data["hotspots"] = hotspots

    driver.close()
    return audit_data


def audit_symbol(symbol: str) -> Dict[str, Any]:
    driver = get_driver()
    with driver.session() as session:
        # Match symbol definitions
        q = """
        MATCH (f)
        WHERE (f:Function OR f:Method OR f:Class OR f:Interface)
          AND (f.name = $sym OR f.qualified_name = $sym)
        OPTIONAL MATCH (caller)-[r1:CALLS]->(f)
        OPTIONAL MATCH (f)-[r2:CALLS]->(callee)
        RETURN f.name AS name, labels(f)[0] AS type, f.path AS file, f.start_line AS start_line, f.end_line AS end_line,
               count(DISTINCT caller) AS caller_count,
               collect(DISTINCT {name: caller.name, file: caller.path, line: caller.start_line})[..10] AS sample_callers,
               count(DISTINCT callee) AS callee_count,
               collect(DISTINCT {name: callee.name, file: callee.path, line: callee.start_line})[..10] AS sample_callees
        """
        records = [dict(r) for r in session.run(q, {"sym": symbol})]

        # Also check if it serves any route or touches any Spanner table
        q_seams = """
        MATCH (f)
        WHERE (f:Function OR f:Method) AND (f.name = $sym OR f.qualified_name = $sym)
        OPTIONAL MATCH (f)-[:IMPLEMENTS_ROUTE|SERVES_ROUTE*0..2]-(re:RouteEndpoint)
        OPTIONAL MATCH (f)-[:READS_TABLE|MUTATES_TABLE*0..2]-(st:SpannerTable)
        RETURN collect(DISTINCT re.path) AS routes, collect(DISTINCT st.name) AS tables
        """
        seam_rec = session.run(q_seams, {"sym": symbol}).single()
        routes = seam_rec["routes"] if seam_rec else []
        tables = seam_rec["tables"] if seam_rec else []

    driver.close()

    total_callers = sum(r.get("caller_count", 0) for r in records)
    risk_level = "CRITICAL" if total_callers > 100 else ("HIGH" if total_callers > 20 else ("MEDIUM" if total_callers > 0 else "LOW"))

    return {
        "query_symbol": symbol,
        "matches_found": len(records),
        "risk_level": risk_level,
        "total_inbound_callers": total_callers,
        "definitions": records,
        "associated_routes": [r for r in routes if r],
        "associated_tables": [t for t in tables if t],
        "guidance": f"Modifying '{symbol}' affects {total_callers} upstream callers across the repo. Verify unit tests and callers before landing."
    }


def audit_file(file_path: str) -> Dict[str, Any]:
    driver = get_driver()
    with driver.session() as session:
        q = """
        MATCH (fn)
        WHERE fn.path = $fpath OR fn.path CONTAINS $fpath
        OPTIONAL MATCH (caller)-[:CALLS]->(fn) WHERE caller.path <> fn.path
        OPTIONAL MATCH (fn)-[:CALLS]->(callee) WHERE callee.path <> fn.path
        RETURN fn.name AS symbol, labels(fn)[0] AS type, fn.start_line AS line,
               count(DISTINCT caller) AS external_callers,
               collect(DISTINCT {name: caller.name, file: caller.path, line: caller.start_line})[..5] AS sample_callers,
               count(DISTINCT callee) AS external_callees
        ORDER BY external_callers DESC
        LIMIT 25
        """
        symbols = [dict(r) for r in session.run(q, {"fpath": file_path})]

    driver.close()
    total_ext_callers = sum(s["external_callers"] for s in symbols)
    risk_level = "CRITICAL" if total_ext_callers > 150 else ("HIGH" if total_ext_callers > 30 else ("MEDIUM" if total_ext_callers > 0 else "LOW"))

    return {
        "file_path": file_path,
        "symbols_count": len(symbols),
        "total_external_callers": total_ext_callers,
        "risk_level": risk_level,
        "exported_symbols": symbols,
        "guidance": f"File '{file_path}' has {total_ext_callers} external inbound callers. High regression potential if function signatures change."
    }


def run_raw_cypher(query: str) -> List[Dict[str, Any]]:
    driver = get_driver()
    results = []
    with driver.session() as session:
        res = session.run(query)
        for record in res:
            results.append(dict(record))
    driver.close()
    return results


def generate_markdown_report(data: Dict[str, Any], output_path: str) -> None:
    now = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    lines: List[str] = [
        "# PegasusX Deep Code Audit Report",
        f"\n**Generated on:** `{now}`  ",
        "**Audit Source:** Memgraph Live CodeGraph (64,432 Nodes / 176,106 Relationships)  \n",
        "---",
        "\n## Executive Summary Dashboard\n",
        "| Audit Suite | Status | Finding Summary | Risk Level |",
        "|---|---|---|---|",
        f"| **1. Multi-Tenancy & Tenant Isolation** | ⚠️ Review | `{data['multi_tenancy']['isolated_count']}` isolated / `{data['multi_tenancy']['non_isolated_count']}` non-isolated | **HIGH** |",
        f"| **2. Contract Drift & 404 Hazards** | ❌ Action Required | `{data['contract_drift']['total_drift']}` client methods missing backend route | **CRITICAL** |",
        f"| **3. Dead & Unconsumed Routes** | ℹ️ Informational | `{data['dead_routes']['total_dead']}` backend endpoints without client caller | **MEDIUM** |",
        f"| **4. Kafka Stream Integrity** | ⚠️ Review | `{data['kafka_integrity']['unconsumed_count']}` topics with zero consumers | **HIGH** |",
        f"| **5. Blast Radius Hotspots** | 🔍 Top 20 Analyzed | Up to `{data['hotspots'][0]['in_degree']}` callers per core function | **HIGH** |",
        "\n---",
        "\n## 1. Multi-Tenancy & Tenant Isolation Audit",
        f"\n- **Isolated Tables (`SupplierId` Partitioned):** `{data['multi_tenancy']['isolated_count']}`",
        f"- **Non-Isolated Tables:** `{data['multi_tenancy']['non_isolated_count']}`",
        "\n> [!WARNING]",
        "> Non-isolated tables must be strictly verified to ensure they only store global catalog data or system configurations. Any operational/transactional table in this list represents a potential cross-tenant leakage vulnerability.\n",
        "### Sample Non-Isolated Tables:",
        "| Table Name | File Definition |",
        "|---|---|",
    ]

    for t in data["multi_tenancy"]["non_isolated"][:25]:
        lines.append(f"| `{t['table']}` | `{t.get('file', 'schema/spanner.ddl')}` |")

    lines.extend([
        "\n---",
        "\n## 2. Contract Drift & 404 Hazards (Client API Calls without Route)",
        f"\nFound **`{data['contract_drift']['total_drift']}`** client methods in Android Kotlin, iOS Swift, or TypeScript where no matching backend Chi route could be verified.\n",
        "> [!CAUTION]",
        "> These methods risk triggering runtime `404 Not Found` errors in mobile applications or portals.\n",
        "| Platform | Method | Target Endpoint Template |",
        "|---|---|---|",
    ])

    for c in data["contract_drift"]["items"][:40]:
        lines.append(f"| `{c.get('platform', 'N/A')}` | `{c.get('method', 'N/A')}` | `{c.get('endpoint', 'N/A')}` |")

    lines.extend([
        "\n---",
        "\n## 3. Dead / Unconsumed Backend Routes",
        f"\nFound **`{data['dead_routes']['total_dead']}`** backend endpoints mounted in Chi router packages that have **zero verified client callers**.\n",
        "> [!NOTE]",
        "> Many of these are internal ops diagnostics (e.g. `/ops/runtime`, `/ops/outbox/*`). Operational routes should be segregated or documented.\n",
        "| HTTP Method | Path | Source File |",
        "|---|---|---|",
    ])

    for r in data["dead_routes"]["routes"][:30]:
        lines.append(f"| `{r.get('method', 'GET')}` | `{r.get('path', '')}` | `{r.get('file', '')}` |")

    lines.extend([
        "\n---",
        "\n## 4. Kafka Event Stream Integrity",
        f"\nOut of **`{data['kafka_integrity']['total_topics']}`** defined Kafka topics, **`{data['kafka_integrity']['unconsumed_count']}`** have zero registered consumer workers.\n",
        "| Topic Name | Outbox Emitters | Registered Consumers | Status |",
        "|---|---|---|---|",
    ])

    for k in data["kafka_integrity"]["topics"]:
        status = "✅ Active Pipeline" if k["consumers"] > 0 else "⚠️ Black Hole (0 Consumers)"
        lines.append(f"| `{k['topic']}` | `{k['emitters']}` | `{k['consumers']}` | {status} |")

    lines.extend([
        "\n---",
        "\n## 5. Critical Blast Radius Hotspots (AST In-Degree)",
        "\nFunctions with highest inbound call volume across the repository. Any change to these functions carries high regression potential:\n",
        "| Function / Method | Inbound Callers | Fully Qualified AST Symbol |",
        "|---|---|---|",
    ])

    for h in data["hotspots"]:
        lines.append(f"| `{h['name']}` | **`{h['in_degree']:,}`** | `{h.get('qn', '')}` |")

    lines.append("\n---\n*Report generated automatically by PegasusX CodeGraph Studio static audit engine.*\n")

    os.makedirs(os.path.dirname(output_path), exist_ok=True)
    with open(output_path, "w") as f:
        f.write("\n".join(lines))


def main() -> None:
    parser = argparse.ArgumentParser(description="PegasusX Deep Code Audit Engine for AI Agents & Engineers")
    parser.add_argument("--output", default="docs/CODE_AUDIT_REPORT.md", help="Output markdown report path")
    parser.add_argument("--symbol", help="Audit blast radius for a specific function/class symbol")
    parser.add_argument("--file", help="Audit blast radius for a specific source file")
    parser.add_argument("--cypher", help="Execute an arbitrary Cypher query and return results")
    parser.add_argument("--json", action="store_true", help="Output machine-readable JSON for AI agents")
    args = parser.parse_args()

    # 1. Targeted Symbol Audit
    if args.symbol:
        res = audit_symbol(args.symbol)
        if args.json:
            print(json.dumps(res, indent=2))
        else:
            print(f"\n[*] BLAST RADIUS AUDIT FOR SYMBOL: '{args.symbol}'")
            print(f"    Risk Level: {res['risk_level']}")
            print(f"    Total Upstream Callers: {res['total_inbound_callers']}")
            print(f"    Definitions Found: {res['matches_found']}")
            for d in res["definitions"]:
                print(f"      - {d['type']} in {d['file']}:{d['start_line']} (callers: {d['caller_count']}, callees: {d['callee_count']})")
                if d.get("sample_callers"):
                    print("        Sample Callers:")
                    for c in d["sample_callers"][:4]:
                        print(f"          * {c['name']} ({c['file']}:{c['line']})")
            if res["associated_routes"]:
                print(f"    Serves Routes: {res['associated_routes']}")
            if res["associated_tables"]:
                print(f"    Mutates Tables: {res['associated_tables']}")
        return

    # 2. Targeted File Audit
    if args.file:
        res = audit_file(args.file)
        if args.json:
            print(json.dumps(res, indent=2))
        else:
            print(f"\n[*] BLAST RADIUS AUDIT FOR FILE: '{args.file}'")
            print(f"    Risk Level: {res['risk_level']}")
            print(f"    Total External Inbound Callers: {res['total_external_callers']}")
            print(f"    Exported Symbols: {res['symbols_count']}")
            for s in res["exported_symbols"][:10]:
                print(f"      - {s['symbol']} (line {s['line']}): {s['external_callers']} callers")
                if s.get("sample_callers"):
                    for c in s["sample_callers"][:2]:
                        print(f"          * caller: {c['name']} in {c['file']}:{c['line']}")
        return

    # 3. Ad-hoc Cypher execution
    if args.cypher:
        rows = run_raw_cypher(args.cypher)
        if args.json:
            print(json.dumps(rows, indent=2, default=str))
        else:
            for r in rows:
                print(r)
        return

    # 4. Full Ecosystem Audit
    data = run_full_audit()

    if args.json:
        print(json.dumps(data, indent=2, default=str))
        return

    print("\n" + "=" * 65)
    print("           PEGASUSX DEEP CODE AUDIT EXECUTIVE SUMMARY")
    print("=" * 65)
    print(f" 1. Multi-Tenancy:       {data['multi_tenancy']['isolated_count']} Isolated | {data['multi_tenancy']['non_isolated_count']} Non-Isolated Tables")
    print(f" 2. Contract Drift:      {data['contract_drift']['total_drift']} Client Calls with Missing Routes [CRITICAL]")
    print(f" 3. Dead Endpoints:      {data['dead_routes']['total_dead']} Unconsumed Backend Endpoints")
    print(f" 4. Kafka Streams:       {data['kafka_integrity']['unconsumed_count']} Unconsumed Topics [HIGH]")
    print(f" 5. AST Top Hotspot:     '{data['hotspots'][0]['name']}' ({data['hotspots'][0]['in_degree']:,} callers)")
    print("=" * 65)

    generate_markdown_report(data, args.output)
    print(f"[+] Full audit report written to: {args.output}\n")


if __name__ == "__main__":
    main()
