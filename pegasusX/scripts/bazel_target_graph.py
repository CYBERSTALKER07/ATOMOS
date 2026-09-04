#!/usr/bin/env python3
"""
PegasusX Bazel / Blaze Build Target Graph & Query Engine.

Models the monorepo using Google Bazel/Blaze target semantics:
  - Every Go package, TypeScript package, and Schema is a Bazel Target:
    e.g. //apps/backend-go/order:order_lib
         //packages/types:types_lib
         //schema:spanner_ddl
  - Extracts intra-monorepo dependency edges:
    (:BazelTarget)-[:DEPENDS_ON]->(:BazelTarget)
  - Implements Bazel query engine:
    rdeps(//..., target) -> Returns exact transitive targets affected by an edit.
    tests(//...)        -> Returns affected test targets in the blast radius.
"""

from __future__ import annotations

import argparse
import glob
import json
import os
import re
import sys
from typing import Any, Dict, List, Set, Tuple

from neo4j import GraphDatabase

BOLT_URL = os.getenv("MEMGRAPH_BOLT_URL", "bolt://localhost:7687")
BOLT_USER = os.getenv("MEMGRAPH_USER", "")
BOLT_PASS = os.getenv("MEMGRAPH_PASSWORD", "")
auth = (BOLT_USER, BOLT_PASS) if BOLT_USER or BOLT_PASS else None

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
REPO_ROOT = os.path.abspath(os.path.join(SCRIPT_DIR, "../.."))
PEGASUSX_DIR = os.path.join(REPO_ROOT, "pegasusX")


def get_memgraph_driver():
    return GraphDatabase.driver(BOLT_URL, auth=auth)


class BazelTargetBuilder:
    def __init__(self, root_dir: str):
        self.root_dir = root_dir
        self.targets: Dict[str, Dict[str, Any]] = {}
        self.dependencies: List[Tuple[str, str]] = []  # (src_target, dep_target)

    def scan_go_packages(self) -> None:
        backend_dir = os.path.join(self.root_dir, "apps/backend-go")
        for root, dirs, files in os.walk(backend_dir):
            go_files = [f for f in files if f.endswith(".go") and not f.endswith("_test.go")]
            test_files = [f for f in files if f.endswith("_test.go")]

            if not go_files:
                continue

            rel_dir = os.path.relpath(root, self.root_dir)
            pkg_name = os.path.basename(root)
            target_id = f"//{rel_dir}:{pkg_name}_lib"
            test_target_id = f"//{rel_dir}:{pkg_name}_test" if test_files else None

            # Collect imports
            imports: Set[str] = set()
            for gf in go_files:
                fpath = os.path.join(root, gf)
                try:
                    content = open(fpath, "r", encoding="utf-8", errors="ignore").read()
                except Exception:
                    continue
                matches = re.findall(r'"github\.com/pegasusx/pegasusx/([^"]+)"', content)
                imports.update(matches)

            self.targets[target_id] = {
                "id": target_id,
                "label": f"//{rel_dir}:{pkg_name}_lib",
                "kind": "go_library",
                "package": rel_dir,
                "srcs_count": len(go_files),
                "has_tests": len(test_files) > 0,
                "test_target": test_target_id,
                "raw_imports": list(imports),
            }

            if test_target_id:
                self.targets[test_target_id] = {
                    "id": test_target_id,
                    "label": test_target_id,
                    "kind": "go_test",
                    "package": rel_dir,
                    "srcs_count": len(test_files),
                    "has_tests": False,
                    "test_target": None,
                    "raw_imports": [rel_dir],
                }
                self.dependencies.append((test_target_id, target_id))

    def scan_ts_packages(self) -> None:
        packages_dir = os.path.join(self.root_dir, "packages")
        for pkg in os.listdir(packages_dir):
            pkg_path = os.path.join(packages_dir, pkg)
            pkg_json = os.path.join(pkg_path, "package.json")
            if os.path.isdir(pkg_path) and os.path.isfile(pkg_json):
                target_id = f"//packages/{pkg}:{pkg}_lib"
                deps: List[str] = []
                try:
                    data = json.load(open(pkg_json))
                    all_deps = {**data.get("dependencies", {}), **data.get("devDependencies", {})}
                    for d in all_deps:
                        if d.startswith("@pegasusx/"):
                            dep_name = d.replace("@pegasusx/", "")
                            deps.append(f"//packages/{dep_name}:{dep_name}_lib")
                except Exception:
                    pass

                self.targets[target_id] = {
                    "id": target_id,
                    "label": target_id,
                    "kind": "ts_project",
                    "package": f"packages/{pkg}",
                    "srcs_count": len(glob.glob(f"{pkg_path}/**/*.ts", recursive=True)),
                    "has_tests": os.path.isdir(os.path.join(pkg_path, "test")),
                    "test_target": f"//packages/{pkg}:{pkg}_test",
                    "raw_imports": deps,
                }

    def resolve_dependencies(self) -> None:
        # Resolve imports to target ids
        pkg_to_target: Dict[str, str] = {}
        for tid, t in self.targets.items():
            if t["kind"] == "go_library":
                pkg_to_target[t["package"]] = tid

        for tid, t in list(self.targets.items()):
            for imp in t.get("raw_imports", []):
                dep_target = pkg_to_target.get(imp)
                if dep_target and dep_target != tid:
                    self.dependencies.append((tid, dep_target))
                elif imp.startswith("//"):
                    self.dependencies.append((tid, imp))

    def build(self) -> Tuple[Dict[str, Dict[str, Any]], List[Tuple[str, str]]]:
        self.scan_go_packages()
        self.scan_ts_packages()
        self.resolve_dependencies()
        return self.targets, self.dependencies


def sync_bazel_targets_to_memgraph(targets: Dict[str, Dict[str, Any]], deps: List[Tuple[str, str]]) -> None:
    driver = get_memgraph_driver()
    with driver.session() as session:
        # Create constraint on BazelTarget
        try:
            session.run("CREATE CONSTRAINT ON (t:BazelTarget) ASSERT t.id IS UNIQUE")
        except Exception:
            pass

        print(f"[*] Ingesting {len(targets)} Bazel targets into Memgraph...")
        batch = []
        for t in targets.values():
            batch.append({
                "id": t["id"],
                "label": t["label"],
                "kind": t["kind"],
                "package": t["package"],
                "srcs_count": t["srcs_count"],
                "has_tests": t["has_tests"],
            })

        session.run("""
        UNWIND $batch AS item
        MERGE (t:BazelTarget {id: item.id})
        SET t.label = item.label,
            t.kind = item.kind,
            t.package = item.package,
            t.srcs_count = item.srcs_count,
            t.has_tests = item.has_tests
        """, {"batch": batch})

        print(f"[*] Ingesting {len(deps)} Bazel DEPENDS_ON edges...")
        dep_batch = [{"src": s, "tgt": t} for s, t in deps]
        session.run("""
        UNWIND $batch AS item
        MATCH (src:BazelTarget {id: item.src})
        MATCH (tgt:BazelTarget {id: item.tgt})
        MERGE (src)-[r:DEPENDS_ON]->(tgt)
        """, {"batch": dep_batch})

    driver.close()
    print("[+] Bazel target DAG successfully synced with Memgraph.")


def query_bazel_rdeps(target_id: str, depth: int = 5) -> Dict[str, Any]:
    """
    Implements Bazel query 'rdeps(//..., target_id)':
    Finds all targets that transitively depend on this target.
    """
    driver = get_memgraph_driver()
    with driver.session() as session:
        q = f"""
        MATCH (target:BazelTarget {{id: $tid}})
        CALL {{
          WITH target
          MATCH path = (dependent:BazelTarget)-[:DEPENDS_ON*1..{depth}]->(target)
          RETURN dependent, length(path) AS distance, [n IN nodes(path) | n.id] AS trace
          LIMIT 100
        }}
        RETURN target.id AS target, target.kind AS kind,
               collect(DISTINCT {{
                 id: dependent.id,
                 kind: dependent.kind,
                 package: dependent.package,
                 distance: distance
               }}) AS rdeps_targets
        """
        try:
            rec = session.run(q, {"tid": target_id}).single()
        except Exception:
            # Fallback for size calculation
            q_fallback = f"""
            MATCH (target:BazelTarget {{id: $tid}})
            MATCH path = (dependent:BazelTarget)-[:DEPENDS_ON*1..{depth}]->(target)
            RETURN target.id AS target, target.kind AS kind,
                   collect(DISTINCT {{
                     id: dependent.id,
                     kind: dependent.kind,
                     package: dependent.package,
                     distance: size(relationships(path))
                   }}) AS rdeps_targets
            """
            rec = session.run(q_fallback, {"tid": target_id}).single()

    driver.close()
    if not rec:
        return {"target": target_id, "found": False, "rdeps_count": 0, "dependents": []}

    deps = rec["rdeps_targets"]
    test_targets = [d for d in deps if d["kind"] == "go_test"]
    lib_targets = [d for d in deps if d["kind"] != "go_test"]

    return {
        "target": target_id,
        "found": True,
        "kind": rec["kind"],
        "total_transitive_dependents": len(deps),
        "affected_library_targets": lib_targets,
        "affected_test_targets_to_run": test_targets,
    }


def main():
    parser = argparse.ArgumentParser(description="PegasusX Bazel Build Target Graph Engine")
    parser.add_argument("--sync", action="store_true", help="Sync Bazel targets and dependencies into Memgraph")
    parser.add_argument("--query-rdeps", help="Run Bazel rdeps query on target (e.g. //apps/backend-go/order:order_lib)")
    parser.add_argument("--depth", type=int, default=5, help="Max depth for rdeps traversal")
    parser.add_argument("--json", action="store_true", help="Output machine-readable JSON")
    args = parser.parse_args()

    builder = BazelTargetBuilder(PEGASUSX_DIR)
    targets, deps = builder.build()

    if args.sync:
        sync_bazel_targets_to_memgraph(targets, deps)
        return

    if args.query_rdeps:
        res = query_bazel_rdeps(args.query_rdeps, args.depth)
        if args.json:
            print(json.dumps(res, indent=2))
        else:
            print(f"\n[*] BAZEL RDEPS QUERY RESULTS FOR: {args.query_rdeps}")
            print(f"    Total Transitive Dependents: {res['total_transitive_dependents']}")
            print(f"    Affected Test Targets To Execute ({len(res['affected_test_targets_to_run'])}):")
            for t in res["affected_test_targets_to_run"]:
                print(f"      - [Distance {t['distance']}] {t['id']}")
            print(f"    Affected Upstream Library Targets ({len(res['affected_library_targets'])}):")
            for l in res["affected_library_targets"][:10]:
                print(f"      - [Distance {l['distance']}] {l['id']}")
        return

    # Default: summary
    print(f"[*] Bazel Target Graph Builder:")
    print(f"    Discovered {len(targets)} Bazel build targets across monorepo.")
    print(f"    Discovered {len(deps)} target dependency edges.")
    print(f"    Run with --sync to push DAG into Memgraph.")


if __name__ == "__main__":
    main()
