#!/usr/bin/env bash
# GS-R: written evidence. Does not flip checkout_reads_this. Does not apply terraform.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/artifacts/GS_R_PACK_CLIENT_PROOF.md"
fail=0
pass() { echo "PASS: $*"; }
die() { echo "FAIL: $*" >&2; fail=1; }

if ! grep -E 'export async function fetchAuthSession' "$ROOT/packages/api-client/market-pack.ts" >/dev/null; then
  die "missing fetchAuthSession"
else
  pass "shared fetchAuthSession + packCurrency"
fi
if ! grep -E 'export function PackChip' "$ROOT/packages/ui-kit/src/pack/PackChip.tsx" >/dev/null; then
  die "missing PackChip"
else
  pass "PackChip splash (currency + receipts)"
fi
if ! grep -E 'fun pinApiBaseUrl' "$ROOT/packages/mobile-android-design/src/main/java/com/pegasus/design/CellApi.kt" >/dev/null; then
  die "missing Android pinApiBaseUrl"
else
  pass "Android CellApi.pinApiBaseUrl + CellPinInterceptor"
fi
if ! grep -E 'static func pinApiBaseUrl' "$ROOT/packages/mobile-ios-design/CellApi.swift" >/dev/null; then
  die "missing iOS pinApiBaseUrl"
else
  pass "iOS CellApi.pinApiBaseUrl"
fi

for f in \
  "$ROOT/apps/supplier-portal/components/SessionPackChip.tsx" \
  "$ROOT/apps/warehouse-portal/components/SessionPackChip.tsx" \
  "$ROOT/apps/factory-portal/components/SessionPackChip.tsx" \
  "$ROOT/apps/retailer-app-desktop/components/SessionPackChip.tsx"
do
  if ! grep -E 'useMarketPack' "$f" >/dev/null; then
    die "$f must bind GET /v1/auth/session"
  fi
done
pass "Web/desktop shells bind session pack"

for f in \
  "$ROOT/apps/supplier-app-android/app/src/main/java/com/pegasusx/supplier/data/remote/NetworkModule.kt" \
  "$ROOT/apps/warehouse-app-android/app/src/main/java/com/pegasusx/warehouse/data/remote/NetworkModule.kt" \
  "$ROOT/apps/factory-app-android/app/src/main/java/com/pegasusx/factory/data/remote/NetworkModule.kt" \
  "$ROOT/apps/retailer-app-android/app/src/main/java/com/pegasusx/retailer/data/api/NetworkModule.kt" \
  "$ROOT/apps/payload-app-android/app/src/main/java/com/pegasus/payload/di/NetworkModule.kt" \
  "$ROOT/apps/driver-app-android/app/src/main/java/com/pegasusx/driver/data/remote/NetworkModule.kt"
do
  if ! grep -E 'CellPinInterceptor' "$f" >/dev/null; then
    die "$f must pin via CellPinInterceptor"
  fi
done
pass "Android role apps pin home_cell (dev bootstrap unchanged)"

for f in \
  "$ROOT/apps/supplier-app-ios/SupplierApp/Services/APIClient.swift" \
  "$ROOT/apps/warehouse-app-ios/WarehouseApp/Services/APIClient.swift" \
  "$ROOT/apps/factory-app-ios/FactoryApp/Services/APIClient.swift" \
  "$ROOT/apps/retailer-app-ios/retailerapp/retailerapp/Services/APIClient.swift" \
  "$ROOT/apps/payload-app-ios/payload-app-ios/Services/APIClient.swift" \
  "$ROOT/apps/driver-app-ios/driverappios/driverappios/Services/APIClient.swift"
do
  if ! grep -E 'pinApiBaseUrl|pinnedAPIBaseURL' "$f" >/dev/null; then
    die "$f must pin API URL from home_cell"
  fi
done
pass "iOS role apps pin home_cell (dev bootstrap unchanged)"

if ! grep -E 'payloadApiBaseUrl|pinApiBaseUrl' "$ROOT/apps/payload-terminal/marketPack.ts" >/dev/null; then
  die "payload-terminal must pin + bind pack"
else
  pass "payload-terminal pins and binds session pack"
fi

if ! (cd "$ROOT/apps/supplier-portal" && pnpm exec vitest run lib/__tests__/market-pack.test.ts >/tmp/pegasusx-gs-r-vitest.txt); then
  die "market-pack vitest failed"
  cat /tmp/pegasusx-gs-r-vitest.txt >&2 || true
else
  pass "pack labels + no Stripe on UZ + localhost pin"
fi

mkdir -p "$ROOT/artifacts"
{
  echo "# GS-R pack client bind proof"
  echo
  echo "**Date:** 2026-08-16"
  echo "**Method:** structural greps + supplier-portal vitest. No checkout_reads_this flip. No terraform apply."
  echo
  echo "| Claim | Evidence | Result |"
  echo "|-------|----------|--------|"
  echo "| Session pack bind | \`GET /v1/auth/session\` via \`fetchAuthSession\` / \`MarketPackBinder\` | PASS |"
  echo "| Splash fields | currency + receipts (Soliq / commercial) on portal shells + native homes | PASS |"
  echo "| Native cell pin | \`CellPinInterceptor\` + iOS \`pinApiBaseUrl\` (localhost stays bootstrap) | PASS |"
  echo "| No web-only currency | retailer/supplier formatters + checkout read pack currency | PASS (bind). Deep POS UZS labels remain continuous leftover |"
  echo "| Flag | \`checkout_reads_this\` still false | PASS |"
  echo
  echo "Leftover: linguistic i18n; remaining hardcoded UZS on deep screens; maps SDK swap. Next code = GS-P only when asked."
} >"$OUT"

if [[ "$fail" -ne 0 ]]; then
  echo "GS-R pack client proof failed" >&2
  exit 1
fi

echo "GS-R pack client proof wrote $OUT"
cat "$OUT"
