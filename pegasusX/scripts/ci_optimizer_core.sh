#!/usr/bin/env bash
# Automated CI test runner for services/optimizer-core (Python & Rust VRP solvers).
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "==> Testing PegasusX Optimizer Core Service..."

SERVER_DIR="$ROOT_DIR/services/optimizer-core/server"
if [[ ! -d "$SERVER_DIR" ]]; then
  echo "Error: $SERVER_DIR does not exist" >&2
  exit 1
fi

cd "$SERVER_DIR"

# Determine python / pytest executable
PYTEST_CMD=""
if [[ -f ".venv/bin/pytest" ]]; then
  PYTEST_CMD=".venv/bin/pytest"
elif command -v pytest >/dev/null 2>&1; then
  PYTEST_CMD="pytest"
elif command -v python3 >/dev/null 2>&1; then
  echo "Setting up virtual environment for optimizer-core tests..."
  python3 -m venv .venv
  .venv/bin/pip install --upgrade pip
  .venv/bin/pip install -r requirements.txt pytest
  PYTEST_CMD=".venv/bin/pytest"
else
  echo "Error: Python 3 not available to test optimizer-core" >&2
  exit 1
fi

echo "Running pytest on $SERVER_DIR..."
"$PYTEST_CMD" -v

# If cargo is installed, run Rust solver tests
RUST_DIR="$ROOT_DIR/services/optimizer-core/server-rust"
if [[ -d "$RUST_DIR" ]] && command -v cargo >/dev/null 2>&1; then
  echo "Running cargo test on $RUST_DIR..."
  (cd "$RUST_DIR" && cargo test)
fi

echo "optimizer-core-ok"
