#!/usr/bin/env bash
# Inject TAURI_UPDATER_PUBKEY into all four desktop tauri.conf.json files
# under plugins.updater (Tauri 2).
#
# Fail-closed: does NOT default to contracts/desktop-updater/dev.pub.
# For local/dev CI unsigned builds only:
#   ALLOW_DEV_UPDATER_PUBKEY=1 bash scripts/apply_desktop_updater_pubkey.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEV_PUB="$ROOT/contracts/desktop-updater/dev.pub"
ALLOW_DEV="${ALLOW_DEV_UPDATER_PUBKEY:-0}"
PUBKEY="${TAURI_UPDATER_PUBKEY:-}"
PUBKEY_FILE="${TAURI_UPDATER_PUBKEY_PATH:-}"

if [[ -z "$PUBKEY" && -n "$PUBKEY_FILE" ]]; then
  [[ -f "$PUBKEY_FILE" ]] || {
    echo "apply_desktop_updater_pubkey: missing pubkey file: $PUBKEY_FILE" >&2
    exit 1
  }
  PUBKEY="$(tr -d '\n\r' <"$PUBKEY_FILE")"
fi

if [[ -z "$PUBKEY" ]]; then
  if [[ "$ALLOW_DEV" == "1" || "$ALLOW_DEV" == "true" || "$ALLOW_DEV" == "yes" ]]; then
    [[ -f "$DEV_PUB" ]] || {
      echo "apply_desktop_updater_pubkey: missing $DEV_PUB" >&2
      exit 1
    }
    PUBKEY="$(tr -d '\n\r' <"$DEV_PUB")"
    echo "apply_desktop_updater_pubkey: WARNING using committed dev.pub (ALLOW_DEV_UPDATER_PUBKEY=1)" >&2
  else
    echo "apply_desktop_updater_pubkey: set TAURI_UPDATER_PUBKEY or TAURI_UPDATER_PUBKEY_PATH (production), or ALLOW_DEV_UPDATER_PUBKEY=1 for local/dev only" >&2
    exit 1
  fi
fi

if [[ -z "$PUBKEY" || "$PUBKEY" == "PLACEHOLDER_PUBLIC_KEY" ]]; then
  echo "apply_desktop_updater_pubkey: invalid pubkey" >&2
  exit 1
fi

# Refuse to inject the committed dev key into release configs unless explicitly allowed.
if [[ -f "$DEV_PUB" ]]; then
  DEV_CONTENT="$(tr -d '\n\r' <"$DEV_PUB")"
  if [[ "$PUBKEY" == "$DEV_CONTENT" ]]; then
    if [[ "$ALLOW_DEV" != "1" && "$ALLOW_DEV" != "true" && "$ALLOW_DEV" != "yes" ]]; then
      echo "apply_desktop_updater_pubkey: refusing committed dev.pub without ALLOW_DEV_UPDATER_PUBKEY=1" >&2
      exit 1
    fi
  fi
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
app_cfg = data.get("app") or {}
if isinstance(app_cfg, dict) and "updater" in app_cfg:
    del app_cfg["updater"]
data["app"] = app_cfg
plugins = data.setdefault("plugins", {})
updater = plugins.setdefault("updater", {})
updater["pubkey"] = pubkey
# Prefer visible installer UI over silent "passive" (audit kill-list).
if updater.get("windows") is None:
    updater["windows"] = {}
if isinstance(updater["windows"], dict):
    updater["windows"]["installMode"] = "basic"
if "endpoints" not in updater or not updater["endpoints"]:
    raise SystemExit(f"missing plugins.updater.endpoints in {path}")
with open(path, "w", encoding="utf-8") as fh:
    json.dump(data, fh, indent=2)
    fh.write("\n")
PY
  echo "apply_desktop_updater_pubkey: $app"
done

echo "desktop-updater-pubkey-ok"
