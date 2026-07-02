#!/usr/bin/env bash
# Sync desktop release secrets (Tauri updater, Authenticode, notarize) into GSM.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SECRETS_FILE="${SECRETS_FILE:-$ROOT/.env.staging.secrets}"
TF_DIR="$ROOT/infra/terraform"

if [[ ! -f "$SECRETS_FILE" ]]; then
  echo "FAIL: secrets file not found: $SECRETS_FILE" >&2
  echo "Copy contracts/desktop-updater/.env.release.example entries into .env.staging.secrets" >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "$SECRETS_FILE"
set +a

cd "$TF_DIR"
OUT="$(terraform output -json 2>/dev/null || echo '{}')"
PROJECT="$(echo "$OUT" | jq -r '.project_id.value // empty')"
if [[ -z "$PROJECT" ]]; then
  echo "FAIL: terraform output project_id — run make phase0-apply first" >&2
  exit 1
fi

put_secret() {
  local gsm_name=$1
  local value=$2
  if [[ -z "$value" || -z "$gsm_name" || "$gsm_name" == "null" ]]; then
    echo "SKIP ${gsm_name:-<empty>} (empty)"
    return 0
  fi
  if ! gcloud secrets describe "$gsm_name" --project "$PROJECT" >/dev/null 2>&1; then
    echo "SKIP $gsm_name (not provisioned — terraform apply)"
    return 0
  fi
  echo -n "$value" | gcloud secrets versions add "$gsm_name" --data-file=- --project "$PROJECT" >/dev/null
  echo "OK   $gsm_name"
}

put_secret "$(echo "$OUT" | jq -r '.tauri_signing_private_key_secret_id.value')" "${TAURI_SIGNING_PRIVATE_KEY:-}"
put_secret "$(echo "$OUT" | jq -r '.tauri_updater_pubkey_secret_id.value')" "${TAURI_UPDATER_PUBKEY:-}"
put_secret "$(echo "$OUT" | jq -r '.windows_codesign_pfx_secret_id.value')" "${WINDOWS_CODESIGN_PFX_B64:-}"
put_secret "$(echo "$OUT" | jq -r '.windows_codesign_password_secret_id.value')" "${WINDOWS_CODESIGN_PASSWORD:-}"

sync_optional() {
  local env_name=$1
  local secret_suffix=$2
  local val="${!env_name:-}"
  if [[ -z "$val" ]]; then
    return 0
  fi
  local prefix
  prefix="$(echo "$OUT" | jq -r '.tenant_slug.value')"
  local gsm="pegasusx-${prefix}-${secret_suffix}"
  put_secret "$gsm" "$val"
}

sync_optional APPLE_NOTARIZE_APPLE_ID apple-notarize-apple-id
sync_optional APPLE_NOTARIZE_TEAM_ID apple-notarize-team-id
sync_optional APPLE_NOTARIZE_APP_PASSWORD apple-notarize-app-password

echo "desktop-release-secrets-sync-ok"
