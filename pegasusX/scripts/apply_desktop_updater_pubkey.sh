#!/usr/bin/env bash
# Inject TAURI_UPDATER_PUBKEY into all four desktop tauri.conf.json files.
# Defaults to contracts/desktop-updater/dev.pub (CI / local unsigned builds).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PUBKEY_FILE="${TAURI_UPDATER_PUBKEY_PATH:-$ROOT/contracts/desktop-updater/dev.pub}"
PUBKEY="${TAURI_UPDATER_PUBKEY:-}"

if [[ -z "$PUBKEY" ]]; then
  [[ -f "$PUBKEY_FILE" ]] || {
    echo "apply_desktop_updater_pubkey: missing pubkey file: $PUBKEY_FILE" >&2
    exit 1
  }
  PUBKEY="$(tr -d '\n' <"$PUBKEY_FILE")"
fi

if [[ -z "$PUBKEY" || "$PUBKEY" == "UPDATE_PUBLIC_KEY" ]]; then
  echo "apply_desktop_updater_pubkey: invalid pubkey" >&2
  exit 1
fi

APPS=(
  retailer-app-desktop
  supplier-portal
  warehouse-portal
  factory-portal
)

for app in "${APPS[@]}"; do
  conf="$ROOT/apps/$app/src-tauri/tauri.conf.json"
  [[ -f "$conf" ]] || {
    echo "apply_desktop_updater_pubkey: missing $conf" >&2
    exit 1
  }
  python3 - "$conf" "$PUBKEY" <<'PY'
import json
import sys

path, pubkey = sys.argv[1], sys.argv[2]
with open(path, encoding="utf-8") as fh:
    data = json.load(fh)
data.setdefault("app", {}).setdefault("updater", {})["pubkey"] = pubkey
with open(path, "w", encoding="utf-8") as fh:
    json.dump(data, fh, indent=2)
    fh.write("\n")
PY
  echo "apply_desktop_updater_pubkey: $app"
done

echo "desktop-updater-pubkey-ok"
