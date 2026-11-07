#!/usr/bin/env bash
# Fail if any client/backend source still carries a "TODO: Inject" placeholder —
# the marker of an orphaned screen/ViewModel that silently does nothing.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
if rg -n --glob '!**/node_modules/**' --glob '!**/.next/**' --glob '!**/build/**' \
  -e 'TODO: Inject' "$ROOT/apps" "$ROOT/packages" 2>/dev/null; then
  echo "FAIL: 'TODO: Inject' placeholders found — wire the dependency or delete the screen" >&2
  exit 1
fi
echo "OK: no 'TODO: Inject' placeholders"
