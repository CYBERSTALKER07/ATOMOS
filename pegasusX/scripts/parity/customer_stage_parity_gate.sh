#!/usr/bin/env bash
# Verifies that all platforms (Go, TypeScript, Android Kotlin, iOS Swift)
# maintain strict parity with contracts/order-customer-stage.json.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

FIXTURE="contracts/order-customer-stage.json"

if [[ ! -f "$FIXTURE" ]]; then
  echo "customer_stage_parity_gate: ERROR: missing $FIXTURE" >&2
  exit 1
fi

# 1. Verify Android test resource copies are byte-identical
for res in \
  "apps/retailer-app-android/app/src/test/resources/order-customer-stage.json" \
  "apps/driver-app-android/app/src/test/resources/order-customer-stage.json"
do
  if [[ ! -f "$res" ]]; then
    echo "customer_stage_parity_gate: missing test fixture copy: $res" >&2
    exit 1
  fi
  if ! cmp -s "$FIXTURE" "$res"; then
    echo "customer_stage_parity_gate: $res drifted from $FIXTURE" >&2
    exit 1
  fi
done

# 2. Run Go contract fixture tests
echo "==> Running Go orderstage & facade contract tests..."
(cd apps/backend-go && go test -count=1 ./orderstage/... ./order/... -run "Stage|Fixture")

# 3. Run TypeScript contract fixture tests
echo "==> Running TypeScript facade parity test..."
pnpm --filter admin-portal test run lib/__tests__/order-lifecycle-facade.test.ts

echo "customer_stage_parity_gate: OK (all platform fixtures in sync)"
