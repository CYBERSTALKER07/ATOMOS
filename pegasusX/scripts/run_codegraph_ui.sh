#!/usr/bin/env bash
set -euo pipefail

# run_codegraph_ui.sh — Launch PegasusX CodeGraph Studio Dashboard
# Starts the interactive Cytoscape + Starlette web dashboard on http://localhost:3001

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
PEGASUSX_DIR="${REPO_ROOT}/pegasusX"

PORT="${PORT:-3001}"
HOST="${HOST:-127.0.0.1}"

# Locate Python binary
if [ -f "${REPO_ROOT}/.venv/bin/python3" ]; then
  PYTHON_BIN="${REPO_ROOT}/.venv/bin/python3"
elif command -v uv >/dev/null 2>&1 && [ -f "${REPO_ROOT}/.venv/pyvenv.cfg" ]; then
  PYTHON_BIN="uv run python3"
else
  PYTHON_BIN="python3"
fi

echo "[*] Launching PegasusX CodeGraph Studio on http://${HOST}:${PORT}..."
exec ${PYTHON_BIN} "${PEGASUSX_DIR}/tools/codegraph-ui/server.py" --host "${HOST}" --port "${PORT}"
