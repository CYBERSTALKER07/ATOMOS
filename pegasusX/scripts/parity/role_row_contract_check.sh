#!/usr/bin/env bash
# PX11-E1: verifies critical api-client exports are referenced by each role row.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

API_CLIENT="$ROOT/packages/api-client/index.ts"
fail() { echo "parity-contract-error: $1" >&2; exit 1; }

[[ -f "$API_CLIENT" ]] || fail "missing packages/api-client/index.ts"

fast_search() {
  local pattern="$1"
  local target="$2"
  if command -v rg >/dev/null 2>&1; then
    rg -q "$pattern" "$target"
  else
    grep -rq --exclude-dir=node_modules --exclude-dir=.next --exclude-dir=build --exclude-dir=dist "$pattern" "$target"
  fi
}

check_ref() {
  local symbol="$1"
  local path="$2"
  fast_search "$symbol" "$API_CLIENT" || fail "api-client missing $symbol"
  fast_search "$symbol" "$path" || fail "$path does not reference $symbol"
}

check_ref "getSupplierNegotiationsPending" "$ROOT/apps/supplier-portal"
check_ref "resolveSupplierNegotiation" "$ROOT/apps/supplier-portal"
check_ref "getRetailerTracking" "$ROOT/apps/retailer-app-desktop"
fast_search "tracking" "$ROOT/apps/retailer-app-android" || fail "retailer-android missing tracking wiring"
fast_search "WebSocket|ws" "$ROOT/apps/driver-app-android/app/src/main/java/com/pegasusx/driver/data/remote" || fail "driver-android missing ws"
fast_search "SupplierWebSocket|/v1/ws" "$ROOT/apps/supplier-app-android" || fail "supplier-android missing ws"
fast_search "SupplierRealtimeClient|/v1/ws" "$ROOT/apps/supplier-app-ios" || fail "supplier-ios missing ws"

echo "role-row-contract-ok"
