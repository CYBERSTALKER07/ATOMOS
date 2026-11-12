"""
graph_rag.py — PegasusX Graph RAG Engine for AI Agents.

Provides graph traversal, seam tracing, blast radius analysis, and Spanner
schema dependency queries backed by the Memgraph knowledge graph over Bolt.
"""

from __future__ import annotations

import os
import re
from typing import Any, Dict, List, Optional, Set, Tuple

BOLT_URL = os.getenv("MEMGRAPH_BOLT_URL", "bolt://127.0.0.1:7687")
BOLT_USER = os.getenv("MEMGRAPH_USER", "")
BOLT_PASS = os.getenv("MEMGRAPH_PASSWORD", "")

_driver = None


def get_driver():
    global _driver
    if _driver is None:
        try:
            from neo4j import GraphDatabase
        except ImportError:
            raise RuntimeError("neo4j Python package is not installed. Run: uv pip install neo4j")

        auth = (BOLT_USER, BOLT_PASS) if BOLT_USER or BOLT_PASS else None
        _driver = GraphDatabase.driver(BOLT_URL, auth=auth)
    return _driver


def _format_node_display(node) -> Tuple[str, str, str]:
    """Returns (label, display_name, file_context)."""
    labels = list(node.labels) if hasattr(node, "labels") else ["Node"]
    lbl = labels[0] if labels else "Node"
    props = dict(node)

    name = (
        props.get("name")
        or props.get("id")
        or props.get("method_name")
        or props.get("method")
        or props.get("path")
        or props.get("handler")
        or str(node.id)
    )

    file_ctx = props.get("file") or props.get("path") or ""
    return lbl, str(name), str(file_ctx)


def graph_explore_symbol(symbol: str, depth: int = 1, limit: int = 25) -> str:
    """
    Explore a symbol's graph neighborhood in the PegasusX knowledge graph.
    Returns symbol definition, incoming callers/consumers, outgoing calls/mutations,
    Spanner tables, Outbox events, Kafka topics, and WebSocket rooms.
    """
    try:
        driver = get_driver()
    except Exception as e:
        return f"[ERROR] Failed to connect to Memgraph at {BOLT_URL}: {e}\nEnsure Docker container 'pegasusx-codegraph-memgraph' is running."

    clean_sym = symbol.strip().replace("'", "\\'")
    depth = max(1, min(depth, 3))
    limit = max(1, min(limit, 100))

    find_cypher = f"""
    MATCH (n)
    WHERE n.name = '{clean_sym}'
       OR n.id = '{clean_sym}'
       OR n.method = '{clean_sym}'
       OR n.method_name = '{clean_sym}'
       OR n.path = '{clean_sym}'
       OR n.handler = '{clean_sym}'
    RETURN n LIMIT 5
    """

    with driver.session() as session:
        try:
            matched_nodes = [record["n"] for record in session.run(find_cypher)]
        except Exception as e:
            return f"[ERROR] Query failed: {e}"

        if not matched_nodes:
            # Fallback: substring search
            fuzzy_cypher = f"""
            MATCH (n)
            WHERE (n.name CONTAINS '{clean_sym}' OR n.id CONTAINS '{clean_sym}' OR n.path CONTAINS '{clean_sym}')
              AND (n:SpannerTable OR n:RouteEndpoint OR n:ApiClientMethod OR n:ServiceMethod OR n:RepositoryMethod OR n:ArchitectureNode)
            RETURN n LIMIT 5
            """
            matched_nodes = [record["n"] for record in session.run(fuzzy_cypher)]

        if not matched_nodes:
            return f"No nodes found in CodeGraph matching symbol '{symbol}'."

        output_sections = []
        for root_node in matched_nodes:
            lbl, name, file_ctx = _format_node_display(root_node)
            props = dict(root_node)

            props_formatted = []
            for k, v in props.items():
                if k not in ("name", "id", "file", "path"):
                    props_formatted.append(f"`{k}`: {v}")
            props_str = f" ({', '.join(props_formatted)})" if props_formatted else ""

            section = [f"## `{lbl}`: **{name}**{props_str}"]
            if file_ctx:
                section.append(f"**Location:** `{file_ctx}`")

            # Fetch inbound relationships
            node_id_int = int(root_node.element_id) if hasattr(root_node, "element_id") and str(root_node.element_id).isdigit() else root_node.id
            inbound_cypher = f"""
            MATCH (src)-[r]->(tgt)
            WHERE id(tgt) = {node_id_int}
            RETURN src, type(r) AS rel_type
            LIMIT {limit}
            """
            inbound_res = list(session.run(inbound_cypher))
            if inbound_res:
                section.append("\n### Inbound (Who calls / references / consumes this):")
                for rec in inbound_res:
                    src_lbl, src_name, src_file = _format_node_display(rec["src"])
                    rel = rec["rel_type"]
                    loc = f" (`{src_file}`)" if src_file else ""
                    section.append(f"- `[{rel}]` ← **{src_lbl}** `{src_name}`{loc}")

            # Fetch outbound relationships
            outbound_cypher = f"""
            MATCH (src)-[r]->(tgt)
            WHERE id(src) = {node_id_int}
            RETURN tgt, type(r) AS rel_type
            LIMIT {limit}
            """
            outbound_res = list(session.run(outbound_cypher))
            if outbound_res:
                section.append("\n### Outbound (What this calls / mutates / produces):")
                for rec in outbound_res:
                    tgt_lbl, tgt_name, tgt_file = _format_node_display(rec["tgt"])
                    rel = rec["rel_type"]
                    loc = f" (`{tgt_file}`)" if tgt_file else ""
                    section.append(f"- `[{rel}]` → **{tgt_lbl}** `{tgt_name}`{loc}")

            output_sections.append("\n".join(section))

        return "\n\n---\n\n".join(output_sections)


def graph_trace_flow(from_symbol: str, to_symbol: str, max_depth: int = 5, limit: int = 5) -> str:
    """
    Trace end-to-end execution and data paths between two symbols in the PegasusX graph.
    Connects mobile client calls -> backend routes -> services -> Spanner tables -> Kafka -> WebSockets.
    """
    try:
        driver = get_driver()
    except Exception as e:
        return f"[ERROR] Failed to connect to Memgraph at {BOLT_URL}: {e}"

    src_sym = from_symbol.strip().replace("'", "\\'")
    tgt_sym = to_symbol.strip().replace("'", "\\'")
    max_depth = max(1, min(max_depth, 6))
    limit = max(1, min(limit, 20))

    cypher = f"""
    MATCH (a), (b)
    WHERE (a.name = '{src_sym}' OR a.id = '{src_sym}' OR a.method_name = '{src_sym}' OR a.path = '{src_sym}')
      AND (b.name = '{tgt_sym}' OR b.id = '{tgt_sym}' OR b.method_name = '{tgt_sym}' OR b.path = '{tgt_sym}')
    MATCH path = (a)-[*1..{max_depth}]->(b)
    RETURN path LIMIT {limit}
    """

    with driver.session() as session:
        try:
            records = list(session.run(cypher))
        except Exception as e:
            return f"[ERROR] Cypher query failed: {e}"

        if not records:
            # Try undirected or fuzzy
            alt_cypher = f"""
            MATCH (a), (b)
            WHERE (a.name CONTAINS '{src_sym}' OR a.id CONTAINS '{src_sym}' OR a.path CONTAINS '{src_sym}')
              AND (b.name CONTAINS '{tgt_sym}' OR b.id CONTAINS '{tgt_sym}' OR b.path CONTAINS '{tgt_sym}')
            MATCH path = (a)-[*1..{max_depth}]-(b)
            RETURN path LIMIT {limit}
            """
            try:
                records = list(session.run(alt_cypher))
            except Exception as e:
                return f"[ERROR] Fallback query failed: {e}"

        if not records:
            return f"No path found between '{from_symbol}' and '{to_symbol}' within {max_depth} hops."

        formatted_paths = []
        for idx, rec in enumerate(records, start=1):
            path = rec["path"]
            nodes = path.nodes
            relationships = path.relationships

            path_lines = [f"### Path {idx} ({len(relationships)} hops):"]
            for step_idx in range(len(nodes)):
                n = nodes[step_idx]
                lbl, name, fctx = _format_node_display(n)
                loc_str = f" (`{fctx}`)" if fctx else ""
                path_lines.append(f"**[{step_idx + 1}]** {lbl}: `{name}`{loc_str}")
                if step_idx < len(relationships):
                    rel = relationships[step_idx]
                    path_lines.append(f"   └── **[{rel.type}]** ──>")

            formatted_paths.append("\n".join(path_lines))

        return "\n\n".join(formatted_paths)


def graph_blast_radius(symbol: str, hops: int = 2, limit: int = 30) -> str:
    """
    Perform a multi-hop blast radius impact analysis on a symbol or database table.
    Shows all affected client apps, backend routes, Kafka consumers, and Spanner tables.
    """
    try:
        driver = get_driver()
    except Exception as e:
        return f"[ERROR] Failed to connect to Memgraph at {BOLT_URL}: {e}"

    clean_sym = symbol.strip().replace("'", "\\'")
    hops = max(1, min(hops, 3))
    limit = max(1, min(limit, 50))

    cypher = f"""
    MATCH (root)
    WHERE (toLower(root.name) = toLower('{clean_sym}') OR toLower(root.id) = toLower('{clean_sym}') OR toLower(root.path) = toLower('{clean_sym}') OR toLower(root.method) = toLower('{clean_sym}'))
    MATCH (m)-[r*1..{hops}]-(root)
    RETURN DISTINCT m, root
    LIMIT {limit}
    """

    with driver.session() as session:
        try:
            records = list(session.run(cypher))
        except Exception as e:
            return f"[ERROR] Blast radius query failed: {e}"

        if not records:
            return f"No dependencies found for '{symbol}' in the graph."

        root_node = records[0]["root"]
        root_lbl, root_name, root_file = _format_node_display(root_node)

        # Categorize affected nodes
        by_category: Dict[str, List[Tuple[str, str, str]]] = {
            "Client Apps & UI Methods": [],
            "Backend HTTP Routes": [],
            "Service & Repository Logic": [],
            "Spanner Database Tables": [],
            "Messaging & Realtime (Kafka & WS)": [],
            "Other Symbols": [],
        }

        for rec in records:
            node = rec["m"]
            lbl, name, fctx = _format_node_display(node)
            item = (lbl, name, fctx)

            if lbl in ("ClientApp", "ApiClientMethod"):
                by_category["Client Apps & UI Methods"].append(item)
            elif lbl == "RouteEndpoint":
                by_category["Backend HTTP Routes"].append(item)
            elif lbl in ("ServiceMethod", "RepositoryMethod", "Function", "Method"):
                by_category["Service & Repository Logic"].append(item)
            elif lbl == "SpannerTable":
                by_category["Spanner Database Tables"].append(item)
            elif lbl in ("KafkaTopic", "KafkaConsumer", "OutboxEmitter", "WSHubRoom", "EventDefinition"):
                by_category["Messaging & Realtime (Kafka & WS)"].append(item)
            else:
                by_category["Other Symbols"].append(item)

        out = [
            f"# Blast Radius Impact Analysis: `{root_lbl}` **{root_name}**",
            f"**Max traversal depth:** {hops} hops | **Total impacted nodes found:** {len(records)}\n",
        ]

        for cat, items in by_category.items():
            if items:
                out.append(f"### {cat} ({len(items)}):")
                # Deduplicate by name
                seen = set()
                for lbl, name, fctx in items:
                    key = (lbl, name)
                    if key in seen:
                        continue
                    seen.add(key)
                    loc = f" (`{fctx}`)" if fctx else ""
                    out.append(f"- **{lbl}**: `{name}`{loc}")
                out.append("")

        return "\n".join(out)


def graph_schema_dependencies(table_name: str) -> str:
    """
    Analyze dependencies on a Spanner table: tenant isolation, repository methods reading/mutating,
    associated routes, and client applications.
    """
    try:
        driver = get_driver()
    except Exception as e:
        return f"[ERROR] Failed to connect to Memgraph at {BOLT_URL}: {e}"

    clean_table = table_name.strip().replace("'", "\\'")

    cypher = f"""
    MATCH (t:SpannerTable)
    WHERE toLower(t.name) = toLower('{clean_table}')
    OPTIONAL MATCH (rm:RepositoryMethod)-[r:MUTATES_TABLE|READS_TABLE]->(t)
    RETURN t, rm, type(r) AS access_type
    """

    with driver.session() as session:
        try:
            records = list(session.run(cypher))
        except Exception as e:
            return f"[ERROR] Schema dependency query failed: {e}"

        if not records or not records[0]["t"]:
            return f"Spanner table '{table_name}' not found in CodeGraph."

        table_node = records[0]["t"]
        props = dict(table_node)
        is_isolated = props.get("is_tenant_isolated", False)
        has_supplier = props.get("has_supplier_id", False)

        out = [
            f"# Spanner Table: `{props.get('name', clean_table)}`",
            f"- **Multi-Tenancy Tenant Isolated:** {'YES (Strictly isolated by Tenant/Supplier ID)' if is_isolated else 'NO / Shared Table'}",
            f"- **Has Supplier ID Column:** {'YES' if has_supplier else 'NO'}\n",
        ]

        reads = []
        mutates = []
        routes_seen = set()
        clients_seen = set()

        for rec in records:
            rm = rec.get("rm")
            access = rec.get("access_type")
            if rm and access:
                rm_props = dict(rm)
                method_name = rm_props.get("method", rm_props.get("id", "unknown"))
                repo_name = rm_props.get("repo", "")
                file_name = rm_props.get("file", "")
                entry = f"- `{repo_name}.{method_name}` (`{file_name}`)"
                if access == "MUTATES_TABLE":
                    mutates.append(entry)
                else:
                    reads.append(entry)

            for route in rec.get("routes", []):
                rp = dict(route)
                r_id = f"{rp.get('method', 'GET')} {rp.get('path', '')}"
                if r_id.strip():
                    routes_seen.add(f"- `{r_id}` → Handler: `{rp.get('handler', '')}` (`{rp.get('file', '')}`)")

            for client in rec.get("clients", []):
                cp = dict(client)
                c_id = f"{cp.get('platform', '')} [{cp.get('role', '')}]: `{cp.get('method_name', '')}`"
                if cp.get("method_name"):
                    clients_seen.add(f"- {c_id} (`{cp.get('file', '')}`)")

        out.append(f"### Mutations ({len(mutates)} Methods):")
        if mutates:
            out.extend(sorted(set(mutates)))
        else:
            out.append("- *No direct repository mutation methods indexed*")

        out.append(f"\n### Reads ({len(reads)} Methods):")
        if reads:
            out.extend(sorted(set(reads)))
        else:
            out.append("- *No direct repository read methods indexed*")

        if routes_seen:
            out.append(f"\n### Associated Route Endpoints ({len(routes_seen)}):")
            out.extend(sorted(list(routes_seen))[:15])

        if clients_seen:
            out.append(f"\n### Associated Client Methods ({len(clients_seen)}):")
            out.extend(sorted(list(clients_seen))[:15])

        return "\n".join(out)


def graph_cypher_query(query: str, limit: int = 50) -> str:
    """
    Execute a read-only Cypher query against the PegasusX Memgraph database.
    Mutating clauses (CREATE, SET, DELETE, DROP, MERGE, DETACH) are rejected.
    """
    try:
        driver = get_driver()
    except Exception as e:
        return f"[ERROR] Failed to connect to Memgraph at {BOLT_URL}: {e}"

    # Security check: disallow mutating Cypher
    disallowed = ["CREATE", "SET", "DELETE", "DROP", "REMOVE", "DETACH", "MERGE", "CALL", "LOAD"]
    tokens = set(re.findall(r"\b[A-Z]+\b", query.upper()))
    for kw in disallowed:
        if kw in tokens:
            return f"[SECURITY ERROR] Mutating keyword '{kw}' is forbidden. This tool is read-only."

    # Enforce limit
    limit = max(1, min(limit, 100))
    clean_query = query.strip().rstrip(";")
    if "LIMIT" not in clean_query.upper():
        clean_query += f" LIMIT {limit}"

    with driver.session() as session:
        try:
            res = list(session.run(clean_query))
        except Exception as e:
            return f"[ERROR] Cypher query failed: {e}\nQuery: {clean_query}"

        if not res:
            return f"Query returned 0 rows.\nQuery: `{clean_query}`"

        # Format as Markdown table
        keys = list(res[0].keys())
        header = "| " + " | ".join(keys) + " |"
        sep = "| " + " | ".join(["---"] * len(keys)) + " |"
        rows = [header, sep]

        for r in res[:limit]:
            row_vals = []
            for k in keys:
                v = r[k]
                if v is None:
                    row_vals.append("")
                elif hasattr(v, "nodes"):  # Path
                    path_str = " -> ".join([_format_node_display(n)[1] for n in v.nodes])
                    row_vals.append(path_str)
                elif hasattr(v, "labels"):  # Node
                    _, name, _ = _format_node_display(v)
                    row_vals.append(name)
                elif isinstance(v, (dict, list)):
                    row_vals.append(str(v).replace("|", "\\|"))
                else:
                    row_vals.append(str(v).replace("|", "\\|"))
            rows.append("| " + " | ".join(row_vals) + " |")

        return f"**Results ({len(res)} rows):**\n\n" + "\n".join(rows)
