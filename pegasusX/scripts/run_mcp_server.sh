#!/usr/bin/env bash
set -euo pipefail

# run_mcp_server.sh — Universal MCP Server Launcher for Code-Graph-RAG
# Resolves python environment (uv virtualenv or global), checks Memgraph connectivity,
# and starts the stdio MCP server for Cursor, Claude Code, Gemini CLI, Windsurf, Cline, and Zed.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
PEGASUSX_DIR="${REPO_ROOT}/pegasusX"

export TARGET_REPO_PATH="${TARGET_REPO_PATH:-${PEGASUSX_DIR}}"
export MEMGRAPH_BOLT_URL="${MEMGRAPH_BOLT_URL:-bolt://localhost:7687}"
export CYPHER_PROVIDER="${CYPHER_PROVIDER:-google}"
export CYPHER_MODEL="${CYPHER_MODEL:-gemini-2.0-flash}"
if [ -n "${GEMINI_API_KEY:-}" ] && [ -z "${GOOGLE_API_KEY:-}" ]; then
  export GOOGLE_API_KEY="${GEMINI_API_KEY}"
fi

# Locate cgr binary or python environment
if [ -f "${REPO_ROOT}/.venv/bin/cgr" ]; then
  CGR_BIN="${REPO_ROOT}/.venv/bin/cgr"
elif command -v cgr >/dev/null 2>&1; then
  CGR_BIN="cgr"
else
  CGR_BIN="python3 -m codebase_rag.cli"
fi

exec ${CGR_BIN} mcp-server --transport stdio
