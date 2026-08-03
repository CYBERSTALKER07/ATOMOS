#!/usr/bin/env bash
# Fail if retailer release UI still contains Control Tower demo strings.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PATTERN='Mock Data|hardcoded BarMark|fakeH3Pulse|CONTROL_TOWER_SIMULATOR'
# Scan retailer clients only (simulator backend is env-gated separately).
if rg -n --glob '!**/node_modules/**' --glob '!**/.next/**' -e "$PATTERN" \
  "$ROOT/apps/retailer-app-desktop" \
  "$ROOT/apps/retailer-app-android" \
  "$ROOT/apps/retailer-app-ios" 2>/dev/null; then
  echo "FAIL: mock Control Tower strings found in retailer clients" >&2
  exit 1
fi
echo "OK: no mock Control Tower strings in retailer clients"
