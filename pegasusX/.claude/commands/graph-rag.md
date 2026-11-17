# Graph RAG Command (Cursor CLI & Agentic Assistants)

When `/graph-rag` is invoked, the user explicitly requests verified knowledge graph traversal using PegasusX Graph RAG and CodeGraph.

## Target & Arguments:
```
$ARGUMENTS
```

## Mandatory Execution:
1. **Never substitute with raw grep loops or file guessing.**
2. Execute the appropriate tool:
   - For symbol/entity exploration: Call `pegasusx-rag` `graph_explore_symbol` or `codegraph` `codegraph_explore`.
   - For end-to-end flow tracing: Call `pegasusx-rag` `graph_trace_flow` (Mobile → Chi Route → Service → Spanner → Kafka → WS).
   - For impact & blast radius: Call `pegasusx-rag` `graph_blast_radius` or `codegraph` `codegraph_impact`.
   - For Spanner schema audit: Call `pegasusx-rag` `graph_schema_dependencies`.
   - For custom Cypher: Call `pegasusx-rag` `graph_cypher_query`.
3. Report back with verified paths, caller trails, and clickable `file://` links.
