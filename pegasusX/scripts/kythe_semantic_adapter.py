#!/usr/bin/env python3
"""
PegasusX Google Kythe Semantic Schema Adapter.

Converts and enriches CodeGraph nodes into the official Google Kythe Schema:
  - Kythe VName (Vector Name):
      { corpus: "pegasusx", root: "", path: <rel_path>, signature: <sig>, language: <lang> }
  - Kythe Standard Edge Kinds:
      /kythe/edge/defines/binding   (Anchor/File -> Definition)
      /kythe/edge/ref/call          (Callsite -> Function)
      /kythe/edge/childof           (Child -> Parent Scope)
      /kythe/edge/typed             (Function/Variable -> Type)
      /kythe/edge/generates         (BazelTarget -> Artifact)
      /kythe/edge/depends           (BazelTarget -> Dependency)
"""

from __future__ import annotations

import argparse
import json
import os
import sys
from typing import Any, Dict, List, Optional

from neo4j import GraphDatabase

BOLT_URL = os.getenv("MEMGRAPH_BOLT_URL", "bolt://localhost:7687")
BOLT_USER = os.getenv("MEMGRAPH_USER", "")
BOLT_PASS = os.getenv("MEMGRAPH_PASSWORD", "")
auth = (BOLT_USER, BOLT_PASS) if BOLT_USER or BOLT_PASS else None


def get_memgraph_driver():
    return GraphDatabase.driver(BOLT_URL, auth=auth)


def enrich_nodes_with_kythe_vnames():
    driver = get_memgraph_driver()
    with driver.session() as session:
        print("[*] Annotating Function/Method nodes with Kythe VNames...")
        session.run("""
        MATCH (f:Function)
        WHERE f.vname_signature IS NULL
        SET f.kythe_corpus = 'pegasusx',
            f.kythe_language = CASE
              WHEN f.path ENDS WITH '.go' THEN 'go'
              WHEN f.path ENDS WITH '.ts' OR f.path ENDS WITH '.tsx' THEN 'typescript'
              WHEN f.path ENDS WITH '.kt' THEN 'kotlin'
              WHEN f.path ENDS WITH '.swift' THEN 'swift'
              ELSE 'unknown'
            END,
            f.vname_signature = coalesce(f.qualified_name, f.name),
            f.vname_path = f.path,
            f.vname_root = ''
        """)

        print("[*] Annotating AST CALLS with Kythe /kythe/edge/ref/call...")
        session.run("""
        MATCH ()-[r:CALLS]->()
        SET r.kythe_edge_kind = '/kythe/edge/ref/call'
        """)

        print("[*] Annotating Bazel DEPENDS_ON with Kythe /kythe/edge/depends...")
        session.run("""
        MATCH ()-[r:DEPENDS_ON]->()
        SET r.kythe_edge_kind = '/kythe/edge/depends'
        """)

        # Count
        v_cnt = session.run("MATCH (n) WHERE n.kythe_corpus = 'pegasusx' RETURN count(n) AS cnt").single()["cnt"]
        e_call_cnt = session.run("MATCH ()-[r:CALLS {kythe_edge_kind: '/kythe/edge/ref/call'}]->() RETURN count(r) AS cnt").single()["cnt"]
        e_dep_cnt = session.run("MATCH ()-[r:DEPENDS_ON {kythe_edge_kind: '/kythe/edge/depends'}]->() RETURN count(r) AS cnt").single()["cnt"]

    driver.close()
    print(f"[+] Successfully annotated {v_cnt} nodes with Kythe VNames.")
    print(f"[+] Annotated {e_call_cnt} AST calls with Kythe '/kythe/edge/ref/call'.")
    print(f"[+] Annotated {e_dep_cnt} Bazel targets with Kythe '/kythe/edge/depends'.")


def query_kythe_xrefs(symbol: str) -> Dict[str, Any]:
    """
    Implements Kythe cross-references query:
      - Definitions
      - Callers (/kythe/edge/ref/call)
      - Enclosing Compilation Unit
    """
    driver = get_memgraph_driver()
    with driver.session() as session:
        res = session.run("""
        MATCH (fn)
        WHERE (fn:Function OR fn:Method OR fn:RouteEndpoint)
          AND (fn.name = $sym OR fn.qualified_name = $sym)
        OPTIONAL MATCH (caller)-[r:CALLS]->(fn)
        OPTIONAL MATCH (pkg:BazelTarget)
        WHERE fn.path STARTS WITH pkg.package
        RETURN fn.name AS name, fn.path AS path, fn.kythe_language AS lang,
               fn.vname_signature AS signature,
               collect(DISTINCT pkg.id)[0] AS bazel_package,
               collect(DISTINCT {
                 caller: caller.name,
                 path: caller.path,
                 vname: caller.vname_signature
               }) AS callers
        LIMIT 10
        """, {"sym": symbol}).single()
    driver.close()

    if not res:
        return {"symbol": symbol, "found": False}

    return {
        "symbol": symbol,
        "found": True,
        "kythe_vname": {
            "corpus": "pegasusx",
            "path": res["path"],
            "signature": res["signature"],
            "language": res["lang"]
        },
        "bazel_compilation_target": res["bazel_package"],
        "kythe_ref_calls_count": len(res["callers"]),
        "sample_callers": res["callers"][:10]
    }


def main():
    parser = argparse.ArgumentParser(description="Google Kythe Semantic Schema Adapter for PegasusX")
    parser.add_argument("--index", action="store_true", help="Annotate CodeGraph with Kythe VNames and edge kinds")
    parser.add_argument("--xref", help="Query Kythe cross-references for a symbol")
    parser.add_argument("--json", action="store_true", help="Output machine-readable JSON")
    args = parser.parse_args()

    if args.index:
        enrich_nodes_with_kythe_vnames()
        return

    if args.xref:
        data = query_kythe_xrefs(args.xref)
        if args.json:
            print(json.dumps(data, indent=2))
        else:
            print(f"\n[*] KYTHE CROSS-REFERENCES FOR: {args.xref}")
            if not data.get("found"):
                print("    Symbol not found.")
                return
            v = data["kythe_vname"]
            print(f"    Kythe VName: [corpus: {v['corpus']}, lang: {v['language']}, path: {v['path']}]")
            print(f"    Signature:   {v['signature']}")
            print(f"    Bazel Unit:  {data['bazel_compilation_target']}")
            print(f"    Callers:     {data['kythe_ref_calls_count']} /kythe/edge/ref/call sites")
            for c in data["sample_callers"][:8]:
                print(f"      - {c['caller']} ({c['path']})")
        return

    enrich_nodes_with_kythe_vnames()


if __name__ == "__main__":
    main()
