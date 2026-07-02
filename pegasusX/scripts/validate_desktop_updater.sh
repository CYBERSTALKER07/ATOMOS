#!/usr/bin/env bash
# Fail if any desktop tauri.conf.json still has the UPDATE_PUBLIC_KEY placeholder.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APPS=(retailer-app-desktop supplier-portal warehouse-portal factory-portal)
failed=0

for app in "${APPS[@]}"; do
  conf="$ROOT/apps/$app/src-tauri/tauri.conf.json"
  if grep -q 'UPDATE_PUBLIC_KEY' "$conf"; then
    echo "validate-desktop-updater: placeholder pubkey in $app" >&2
    failed=1
  fi
  if ! grep -q 'pegasusx-ssmr-app-updates' "$conf"; then
    echo "validate-desktop-updater: GCS updater endpoint missing in $app" >&2
    failed=1
  fi
done

if [[ "$failed" -ne 0 ]]; then
  exit 1
fi

echo "validate-desktop-updater-ok"
