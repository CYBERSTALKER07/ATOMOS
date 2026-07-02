#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

APPS=(
  retailer-app-desktop
  supplier-portal
  warehouse-portal
  factory-portal
)

fail() { echo "validate-desktop-clients-error: $1" >&2; exit 1; }

if ! command -v pnpm >/dev/null 2>&1; then
  fail "pnpm is required"
fi

pnpm install --frozen-lockfile 2>/dev/null || pnpm install

echo "desktop-bridge: unit tests"
pnpm --filter @pegasusx/desktop-bridge test

echo "desktop-cache: unit tests"
pnpm --filter @pegasusx/desktop-cache test

for app in "${APPS[@]}"; do
  dir="apps/$app"
  [[ -d "$dir" ]] || fail "missing $dir"
  echo "$app: typecheck"
  pnpm --filter "@pegasusx/$app" typecheck
  echo "$app: static export (TAURI_BUILD=1)"
  pnpm --filter "@pegasusx/$app" run build:static
done

echo "desktop-clients-ok"
