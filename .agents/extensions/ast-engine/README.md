# Agent AST Engine (Augment-Style)

This is a local AST retrieval engine designed to give agents Liveblocks/Trigger.dev-style deep context capabilities.

## Concept
Instead of passing millions of tokens to the LLM, the agent invokes this engine via a CLI or MCP tool.
The engine parses TypeScript/Go code, extracts the AST (Abstract Syntax Tree), and builds a dependency graph.

## Components
1. **Indexer**: Scans `apps/` and `packages/` to build a graph of function definitions, classes, and exported interfaces.
2. **Query API**: Allows the agent to query `get_definition("AuthService")` or `get_usages("updateOrder")`.
3. **Sync Watcher**: Keeps the AST graph in sync with live file changes.

## Next Steps
- Integrate `ts-morph` (for TS) and `go/ast` (for Go).
- Create a simple CLI wrapper that the agent can execute to query the codebase.
