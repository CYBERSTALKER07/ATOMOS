#!/usr/bin/env bash
# PX12-C: fail CI when PX12-touched event types lack declarations in events.go.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

fail() { echo "gap-hunter-gate-error: $1" >&2; exit 1; }

EVENTS="apps/backend-go/events/events.go"
[[ -f "$EVENTS" ]] || fail "missing $EVENTS"

REQUIRED=(
  "EventRouteReordered"
  "EventMissingItemsReported"
  "EventSplitPaymentCreated"
)

for sym in "${REQUIRED[@]}"; do
  if ! grep -q "$sym" "$EVENTS"; then
    fail "$sym not declared in events package"
  fi
done

echo "gap-hunter-gate-ok"
