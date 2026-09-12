#!/usr/bin/env bash
# Fail if any desktop tauri.conf.json lacks Tauri 2 plugins.updater wiring,
# or ships the committed dev pubkey without ALLOW_DEV_UPDATER_PUBKEY=1.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APPS=(retailer-app-desktop supplier-portal warehouse-portal factory-portal)
DEV_PUB="$ROOT/contracts/desktop-updater/dev.pub"
ALLOW_DEV="${ALLOW_DEV_UPDATER_PUBKEY:-0}"
failed=0

DEV_CONTENT=""
if [[ -f "$DEV_PUB" ]]; then
  DEV_CONTENT="$(tr -d '\n\r' <"$DEV_PUB")"
fi

for app in "${APPS[@]}"; do
  conf="$ROOT/apps/$app/src-tauri/tauri.conf.json"
  if grep -q 'PLACEHOLDER_PUBLIC_KEY' "$conf"; then
    echo "validate-desktop-updater: placeholder pubkey in $app" >&2
    failed=1
  fi
  if ! grep -q 'pegasusx-ssmr-app-updates' "$conf"; then
    echo "validate-desktop-updater: GCS updater endpoint missing in $app" >&2
    failed=1
  fi
  set +e
  python3 - "$conf" "$DEV_CONTENT" "$ALLOW_DEV" <<'PY'
import json, sys
path, dev_pub, allow_dev = sys.argv[1], sys.argv[2], sys.argv[3].lower()
with open(path, encoding="utf-8") as fh:
    data = json.load(fh)
upd = (data.get("plugins") or {}).get("updater") or {}
pubkey = (upd.get("pubkey") or "").strip()
if not pubkey:
    sys.exit(2)
if not upd.get("endpoints"):
    sys.exit(3)
if (data.get("app") or {}).get("updater"):
    sys.exit(4)
if not (data.get("bundle") or {}).get("createUpdaterArtifacts"):
    sys.exit(5)
if pubkey == "PLACEHOLDER_PUBLIC_KEY":
    sys.exit(6)
if dev_pub and pubkey == dev_pub and allow_dev not in ("1", "true", "yes"):
    sys.exit(7)
# Silent install is a kill-list item for production honesty.
windows = upd.get("windows") if isinstance(upd.get("windows"), dict) else {}
if windows.get("installMode") == "passive" and allow_dev not in ("1", "true", "yes"):
    sys.exit(8)
sys.exit(0)
PY
  code=$?
  set -e
  if [[ "$code" -ne 0 ]]; then
    case "$code" in
      7) echo "validate-desktop-updater: $app still uses committed dev.pub — set production TAURI_UPDATER_PUBKEY or ALLOW_DEV_UPDATER_PUBKEY=1 for local only" >&2 ;;
      8) echo "validate-desktop-updater: $app uses silent installMode=passive — use basic (or ALLOW_DEV_UPDATER_PUBKEY=1)" >&2 ;;
      *) echo "validate-desktop-updater: plugins.updater incomplete in $app (code=$code)" >&2 ;;
    esac
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
