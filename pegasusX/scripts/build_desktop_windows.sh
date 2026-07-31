#!/usr/bin/env bash
# Build one Tauri desktop app for Windows (x86_64-pc-windows-msvc).
set -euo pipefail

APP="${1:-}"
TARGET="${TAURI_WINDOWS_TARGET:-x86_64-pc-windows-msvc}"

if [[ -z "$APP" ]]; then
  echo "Usage: $0 <retailer-app-desktop|supplier-portal|warehouse-portal|factory-portal>" >&2
  exit 1
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

bash scripts/apply_desktop_updater_pubkey.sh

# Prefer CI secret; fall back to local dev signing key for updater artifacts.
if [[ -z "${TAURI_SIGNING_PRIVATE_KEY:-}" && -f "$ROOT/contracts/desktop-updater/dev.key" ]]; then
  export TAURI_SIGNING_PRIVATE_KEY
  TAURI_SIGNING_PRIVATE_KEY="$(tr -d '\n' <"$ROOT/contracts/desktop-updater/dev.key")"
  export TAURI_SIGNING_PRIVATE_KEY_PASSWORD="${TAURI_SIGNING_PRIVATE_KEY_PASSWORD:-}"
fi

case "$APP" in
  retailer-app-desktop) FILTER="@pegasusx/retailer-app-desktop" ;;
  supplier-portal) FILTER="@pegasusx/supplier-portal" ;;
  warehouse-portal) FILTER="@pegasusx/warehouse-portal" ;;
  factory-portal) FILTER="@pegasusx/factory-portal" ;;
  *)
    echo "Unknown desktop app: $APP" >&2
    exit 1
    ;;
esac

pnpm install --frozen-lockfile 2>/dev/null || pnpm install
pnpm --filter "$FILTER" run build:static
pnpm --filter "$FILTER" exec tauri build --target "$TARGET"

echo "desktop-windows-build-ok: $APP"
