#!/usr/bin/env bash
# Fail if any desktop tauri.conf.json lacks Tauri 2 plugins.updater wiring.
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
  if ! python3 - "$conf" <<'PY'
import json, sys
with open(sys.argv[1], encoding="utf-8") as fh:
    data = json.load(fh)
upd = (data.get("plugins") or {}).get("updater") or {}
if not upd.get("pubkey"):
    sys.exit(2)
if not upd.get("endpoints"):
    sys.exit(3)
if (data.get("app") or {}).get("updater"):
    # Legacy Tauri v1 location — must be migrated
    sys.exit(4)
if not (data.get("bundle") or {}).get("createUpdaterArtifacts"):
    sys.exit(5)
sys.exit(0)
PY
  then
    code=$?
    echo "validate-desktop-updater: plugins.updater incomplete in $app (code=$code)" >&2
    failed=1
  fi
  cargo_toml="$ROOT/apps/$app/src-tauri/Cargo.toml"
  if ! grep -q 'tauri-plugin-updater' "$cargo_toml"; then
    echo "validate-desktop-updater: tauri-plugin-updater missing in $app Cargo.toml" >&2
    failed=1
  fi
  caps="$ROOT/apps/$app/src-tauri/capabilities/default.json"
  if ! grep -q 'updater:default' "$caps"; then
    echo "validate-desktop-updater: updater:default permission missing in $app" >&2
    failed=1
  fi
done

if [[ "$failed" -ne 0 ]]; then
  exit 1
fi

echo "validate-desktop-updater-ok"
