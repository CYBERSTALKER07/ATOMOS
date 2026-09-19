#!/usr/bin/env python3
"""Neighborhood walk over pegasusX/context/architecture-graph.json.

This is a routing index, not a status oracle. generatedAt is null; runtimeNotes
are stale. Always open the returned paths in the current session.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from collections import defaultdict, deque
from pathlib import Path


def find_graph() -> Path:
    here = Path(__file__).resolve()
    for p in [here, *here.parents]:
        cand = p / "pegasusX" / "context" / "architecture-graph.json"
        if cand.is_file():
            return cand
    raise SystemExit("architecture-graph.json not found under pegasusX/context/")


def tokens(q: str) -> list[str]:
    parts = re.findall(r"[a-zA-Z0-9_./-]{2,}", q.lower())
    stop = {
        "the", "and", "for", "with", "from", "that", "this", "into", "onto",
        "our", "how", "what", "where", "does", "live", "file",
    }
    return [p for p in parts if p not in stop]


def score_node(node: dict, toks: list[str]) -> int:
    hay = " ".join(
        str(node.get(k, "")) for k in ("id", "kind", "language", "path")
    ).lower()
    s = 0
    for t in toks:
        if t in hay:
            s += 3 if t in (node.get("id") or "").lower() else 1
            if t in (node.get("path") or "").lower():
                s += 2
    return s


def walk(graph: dict, seeds: list[str], hops: int) -> list[dict]:
    by_id = {n["id"]: n for n in graph["nodes"]}
    adj: dict[str, set[str]] = defaultdict(set)
    edge_list = []
    for e in graph["edges"]:
        a, b = e.get("from"), e.get("to")
        if a and b:
            adj[a].add(b)
            adj[b].add(a)
            edge_list.append(e)

    seen: dict[str, int] = {}
    q: deque[tuple[str, int]] = deque((s, 0) for s in seeds if s in by_id)
    for s, d in list(q):
        seen[s] = d
    while q:
        nid, d = q.popleft()
        if d >= hops:
            continue
        for nxt in adj[nid]:
            if nxt not in seen:
                seen[nxt] = d + 1
                q.append((nxt, d + 1))

    out = []
    for nid, dist in sorted(seen.items(), key=lambda x: (x[1], x[0])):
        n = by_id[nid]
        incident = [
            e
            for e in edge_list
            if e.get("from") == nid or e.get("to") == nid
        ]
        out.append(
            {
                "id": nid,
                "kind": n.get("kind"),
                "path": n.get("path"),
                "language": n.get("language"),
                "hops": dist,
                "edges": [
                    {
                        "from": e.get("from"),
                        "to": e.get("to"),
                        "kind": e.get("kind"),
                    }
                    for e in incident[:12]
                ],
            }
        )
    return out


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--query", "-q", required=True)
    ap.add_argument("--hops", type=int, default=2)
    ap.add_argument("--limit", type=int, default=16)
    ap.add_argument("--json", action="store_true")
    args = ap.parse_args()

    path = find_graph()
    graph = json.loads(path.read_text())
    toks = tokens(args.query)
    if not toks:
        print("empty query after tokenization", file=sys.stderr)
        return 2

    ranked = sorted(
        ((score_node(n, toks), n) for n in graph["nodes"]),
        key=lambda x: -x[0],
    )
    seeds = [n["id"] for s, n in ranked if s > 0][:8]
    if not seeds:
        print(
            json.dumps(
                {
                    "graph": str(path),
                    "warning": "no node matched; graph is a routing index only",
                    "nodes": len(graph["nodes"]),
                    "edges": len(graph["edges"]),
                    "generatedAt": graph.get("generatedAt"),
                    "hits": [],
                },
                indent=2,
            )
            if args.json
            else f"no graph hits for {args.query!r} in {path}"
        )
        return 0

    hits = walk(graph, seeds, max(0, args.hops))[: args.limit]
    payload = {
        "graph": str(path),
        "generatedAt": graph.get("generatedAt"),
        "honesty": "routing-index-not-status; open paths this session; code wins",
        "query": args.query,
        "seeds": seeds,
        "hits": hits,
    }
    if args.json:
        print(json.dumps(payload, indent=2))
        return 0

    print(f"# graph retrieve  query={args.query!r}  generatedAt={graph.get('generatedAt')}")
    print("# NOT status. Open these paths. Code wins.\n")
    for h in hits:
        print(f"{h['hops']}hop  {h['id']:32}  {h['kind']:12}  {h['path']}")
        for e in h["edges"][:4]:
            print(f"      {e['from']} --{e['kind']}--> {e['to']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
