#!/usr/bin/env bash
# PX11-E1: verifies critical api-client exports are referenced by each role row.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

API_CLIENT="$ROOT/packages/api-client/index.ts"
fail() { echo "parity-contract-error: $1" >&2; exit 1; }

[[ -f "$API_CLIENT" ]] || fail "missing packages/api-client/index.ts"

check_ref() {
  local symbol="$1"
  local path="$2"
  grep -q "$symbol" "$API_CLIENT" || fail "api-client missing $symbol"
  grep -rq "$symbol" "$path" || fail "$path does not reference $symbol"
}

check_ref "getSupplierNegotiationsPending" "$ROOT/apps/supplier-portal"
check_ref "resolveSupplierNegotiation" "$ROOT/apps/supplier-portal"
check_ref "getRetailerTracking" "$ROOT/apps/retailer-app-desktop"
grep -rq "tracking" "$ROOT/apps/retailer-app-android" || fail "retailer-android missing tracking wiring"
grep -rq "WebSocket\|ws" "$ROOT/apps/driver-app-android/app/src/main/java/com/pegasusx/driver/data/remote" || fail "driver-android missing ws"
grep -rq "SupplierWebSocket\|/v1/ws" "$ROOT/apps/supplier-app-android" || fail "supplier-android missing ws"
grep -rq "SupplierRealtimeClient\|/v1/ws" "$ROOT/apps/supplier-app-ios" || fail "supplier-ios missing ws"

echo "role-row-contract-ok"
