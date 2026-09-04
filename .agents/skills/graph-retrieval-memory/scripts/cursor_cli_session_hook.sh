#!/usr/bin/env bash
# Cursor CLI / IDE sessionStart: env + bounded GOAL/WORKSPACE injection.
# User hooks run with cwd ~/.cursor; project hooks run from repo root.
# Resolve the real script so this file can be a symlink.
set -euo pipefail
export VOID_REPO="${VOID_REPO:-$HOME/Desktop/V.O.I.D}"

SELF="$(python3 -c 'import os,sys; print(os.path.realpath(sys.argv[1]))' "$0")"
DIR="$(dirname "$SELF")"
HELPER="$DIR/cursor_cli_memory.py"
if [[ ! -f "$HELPER" ]]; then
  HELPER="${HOME}/.cursor/skills/graph-retrieval-memory/scripts/cursor_cli_memory.py"
fi
if [[ ! -f "$HELPER" ]]; then
  HELPER="${VOID_REPO}/.agents/skills/graph-retrieval-memory/scripts/cursor_cli_memory.py"
fi
exec python3 "$HELPER"
