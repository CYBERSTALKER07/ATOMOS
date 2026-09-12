#!/usr/bin/env bash
# Retail OS unit gate (Phase 7). Run from monorepo root or apps/backend-go.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT/apps/backend-go"
go test ./retailer/ ./retailerroutes/ -count=1
# Fail if demo supplier id reappears in retailer clients
if grep -R "sup-demo-1" \
  "$ROOT/apps/retailer-app-desktop" \
  "$ROOT/apps/retailer-app-android/app/src" \
  "$ROOT/apps/retailer-app-ios/retailerapp/retailerapp" \
  --include='*.tsx' --include='*.ts' --include='*.kt' --include='*.swift' \
  2>/dev/null; then
  echo "FAIL: sup-demo-1 found in retailer apps" >&2
  exit 1
fi
echo "OK: retailer OS unit gate"
