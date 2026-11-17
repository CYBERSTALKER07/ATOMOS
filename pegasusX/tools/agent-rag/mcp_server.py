import os
import sys

# Crucial for MCP stdio: prevent any 3rd-party library from writing to stdout
os.environ["HF_HUB_DISABLE_SYMLINKS_WARNING"] = "1"
os.environ["TOKENIZERS_PARALLELISM"] = "false"
os.environ["TRANSFORMERS_VERBOSITY"] = "error"
os.environ["TQDM_DISABLE"] = "1"
os.environ["HF_HUB_OFFLINE"] = "1"

from mcp.server.fastmcp import FastMCP
from hybrid_search import hybrid_search, search_dense, find_symbol
from graph_rag import (
    graph_explore_symbol as graph_explore_symbol_impl,
    graph_trace_flow as graph_trace_flow_impl,
    graph_blast_radius as graph_blast_radius_impl,
    graph_schema_dependencies as graph_schema_dependencies_impl,
    graph_cypher_query as graph_cypher_query_impl,
)

# Initialize FastMCP Server
mcp = FastMCP("PegasusX-Agent-RAG")


# ===========================================================================
# 1. Hybrid & Semantic Vector Retrieval Tools
# ===========================================================================

@mcp.tool()
def hybrid_code_search(query: str, layer: str = None, limit: int = 5) -> str:
    """
    Search the PegasusX codebase using hybrid retrieval (BM25 sparse search + Dense vector embeddings + Reciprocal Rank Fusion).
    Excels at finding both conceptual logic and exact symbol / function / table names.
    
    Args:
        query: The natural language or code query (e.g. "OrderStockReservationMarkers", "defaultOrderID", "GlobalPay settlement").
        layer: Optional filter to restrict search to a specific layer ("backend", "infra", "client-web", "mobile-android", "mobile-ios", "python", "shared-packages").
        limit: Number of results to return (default: 5).
    """
    hits = hybrid_search(query, layer=layer, limit=limit)
    if not hits:
        return "No relevant code found in the RAG database for this query."
        
    formatted = []
    for idx, hit in enumerate(hits, start=1):
        meta = hit.get("metadata") or {}
        score = hit.get("rrf_score", 0.0)
        file_path = meta.get("file", "unknown")
        layer_name = meta.get("layer", "unknown")
        symbol = meta.get("symbol", "")
        lang = meta.get("language", "")
        
        sym_header = f" | Symbol: `{symbol}`" if symbol else ""
        formatted.append(
            f"### Result {idx} (RRF Score: {score:.5f})\n"
            f"**File:** `{file_path}` | **Layer:** `{layer_name}`{sym_header}\n"
            f"```{lang}\n{hit['document']}\n```\n"
        )
    return "\n---\n".join(formatted)


@mcp.tool()
def semantic_code_search(query: str, layer: str = None, limit: int = 5) -> str:
    """
    Search the PegasusX codebase using pure dense vector semantic similarity.
    Best for high-level conceptual descriptions (e.g. "reconciliation and settlement flow").
    
    Args:
        query: Conceptual description of what you are looking for.
        layer: Optional filter ("backend", "infra", "client-web", "mobile-android", "mobile-ios", "python", "shared-packages").
        limit: Number of code chunks to return (default: 5).
    """
    hits = search_dense(query, layer=layer, limit=limit)
    if not hits:
        return "No relevant code found in the RAG database for this query."
        
    formatted = []
    for idx, hit in enumerate(hits, start=1):
        meta = hit.get("metadata") or {}
        dist = hit.get("distance", 0.0)
        file_path = meta.get("file", "unknown")
        layer_name = meta.get("layer", "unknown")
        symbol = meta.get("symbol", "")
        lang = meta.get("language", "")
        
        sym_header = f" | Symbol: `{symbol}`" if symbol else ""
        formatted.append(
            f"### Result {idx} (Distance: {dist:.4f})\n"
            f"**File:** `{file_path}` | **Layer:** `{layer_name}`{sym_header}\n"
            f"```{lang}\n{hit['document']}\n```\n"
        )
    return "\n---\n".join(formatted)


@mcp.tool()
def find_code_symbol(symbol: str, layer: str = None, limit: int = 5) -> str:
    """
    Find exact code symbol declarations (functions, structs, classes, interfaces, tables) across the codebase.
    
    Args:
        symbol: Exact or partial name of the symbol (e.g. "defaultOrderID", "OrdersByStateView", "DeliveryToken").
        layer: Optional layer filter.
        limit: Number of matching declarations to return (default: 5).
    """
    hits = find_symbol(symbol, layer=layer, limit=limit)
    if not hits:
        return f"No symbol declarations found matching '{symbol}'."
        
    formatted = []
    for idx, hit in enumerate(hits, start=1):
        meta = hit.get("metadata") or {}
        file_path = meta.get("file", "unknown")
        layer_name = meta.get("layer", "unknown")
        sym = meta.get("symbol", symbol)
        lang = meta.get("language", "")
        
        formatted.append(
            f"### Declaration {idx}: `{sym}`\n"
            f"**File:** `{file_path}` | **Layer:** `{layer_name}`\n"
            f"```{lang}\n{hit['document']}\n```\n"
        )
    return "\n---\n".join(formatted)


# ===========================================================================
# 2. Graph RAG Tools (Memgraph Knowledge Graph)
# ===========================================================================

@mcp.tool()
def graph_explore_symbol(symbol: str, depth: int = 1, limit: int = 25) -> str:
    """
    Explore a symbol's graph neighborhood in the PegasusX knowledge graph.
    Returns symbol definition, incoming callers/consumers, outgoing calls/mutations,
    Spanner tables, Outbox events, Kafka topics, and WebSocket rooms.

    Args:
        symbol: Exact or partial name of the symbol (e.g. "Suppliers", "Order", "SaveReconciliation", "HandleWarehouseLogin").
        depth: Traversal depth for relationships (default: 1, max: 3).
        limit: Max results per category (default: 25).
    """
    return graph_explore_symbol_impl(symbol=symbol, depth=depth, limit=limit)


@mcp.tool()
def graph_trace_flow(from_symbol: str, to_symbol: str, max_depth: int = 5, limit: int = 5) -> str:
    """
    Trace end-to-end execution and data paths between two symbols in the PegasusX graph.
    Connects mobile client calls -> backend routes -> services -> Spanner tables -> Kafka -> WebSockets.

    Args:
        from_symbol: Starting symbol, client method, route, or event (e.g. "login", "route_POST_/v1/auth/warehouse/login").
        to_symbol: Destination symbol, service method, Spanner table, or Kafka topic (e.g. "Warehouses", "Orders").
        max_depth: Maximum hops between nodes (default: 5, max: 6).
        limit: Max paths to return (default: 5).
    """
    return graph_trace_flow_impl(from_symbol=from_symbol, to_symbol=to_symbol, max_depth=max_depth, limit=limit)


@mcp.tool()
def graph_blast_radius(symbol: str, hops: int = 2, limit: int = 30) -> str:
    """
    Perform a multi-hop blast radius impact analysis on a symbol or database table.
    Shows all affected client apps (iOS, Android, Web), backend HTTP routes, Kafka consumers, and Spanner tables.

    Args:
        symbol: Symbol, table, route, or topic to analyze (e.g. "Orders", "WarehouseProfiles", "spannerTxnBuffer").
        hops: Number of relationship hops to traverse (default: 2, max: 3).
        limit: Maximum impacted symbols to return (default: 30).
    """
    return graph_blast_radius_impl(symbol=symbol, hops=hops, limit=limit)


@mcp.tool()
def graph_schema_dependencies(table_name: str) -> str:
    """
    Analyze dependencies on a Spanner table: multi-tenancy isolation status, tenant partitioning,
    repository methods reading/mutating the table, associated HTTP routes, and client applications.

    Args:
        table_name: Spanner table name (e.g. "Orders", "Suppliers", "Warehouses", "PaymentLedgerEntries").
    """
    return graph_schema_dependencies_impl(table_name=table_name)


@mcp.tool()
def graph_cypher_query(query: str, limit: int = 50) -> str:
    """
    Execute a read-only Cypher query against the PegasusX Memgraph database.
    Mutating clauses (CREATE, SET, DELETE, DROP, MERGE, DETACH) are strictly rejected.

    Args:
        query: Read-only OpenCypher query string (e.g. "MATCH (c:ClientApp)-[:RECEIVED_BY]->(h:WSHubRoom) RETURN c, h").
        limit: Max rows returned (default: 50).
    """
    return graph_cypher_query_impl(query=query, limit=limit)


if __name__ == "__main__":
    mcp.run()
