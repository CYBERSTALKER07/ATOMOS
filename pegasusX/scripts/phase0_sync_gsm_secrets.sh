#!/usr/bin/env bash
# PX-PROD-0 / P0-8: sync handoff secrets into GSM (names from terraform output).
# Ensures every ExternalSecret remoteRef has ≥1 enabled version so ESO atomic sync can succeed.
# Unused PSP rails get the documented stub "unused-rail-placeholder" when env is empty.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TF_DIR="$ROOT/infra/terraform"
SECRETS_FILE="${SECRETS_FILE:-$ROOT/.env.staging.secrets}"
UNUSED_RAIL_PLACEHOLDER="${UNUSED_RAIL_PLACEHOLDER:-unused-rail-placeholder}"

if [[ ! -f "$SECRETS_FILE" ]]; then
	echo "FAIL: secrets file not found: $SECRETS_FILE" >&2
	echo "Copy .env.staging.secrets.example and fill values." >&2
	exit 1
fi

set -a
# shellcheck disable=SC1090
source "$SECRETS_FILE"
set +a

cd "$TF_DIR"
OUT="$(terraform output -json 2>/dev/null || echo '{}')"
if [[ "$OUT" == "{}" ]] || [[ "$(echo "$OUT" | jq 'length')" -eq 0 ]]; then
	echo "FAIL: no terraform outputs — run make phase0-apply first" >&2
	exit 1
fi

PROJECT_ID="$(terraform output -raw project_id)"

put_secret() {
	local gsm_name=$1
	local value=$2
	if [[ -z "$gsm_name" || "$gsm_name" == "null" ]]; then
		echo "SKIP (empty gsm name)" >&2
		return 0
	fi
	if [[ -z "$value" ]]; then
		echo "SKIP $gsm_name (empty value)"
		return 0
	fi
	echo -n "$value" | gcloud secrets versions add "$gsm_name" --data-file=- --project "$PROJECT_ID" >/dev/null
	echo "OK   $gsm_name"
}

# Add a version only when the secret has zero enabled versions (ESO boot).
ensure_version() {
	local gsm_name=$1
	local value=$2
	if [[ -z "$gsm_name" || "$gsm_name" == "null" ]]; then
		return 0
	fi
	local n
	n="$(gcloud secrets versions list "$gsm_name" --project "$PROJECT_ID" --filter='state:ENABLED' --format='value(name)' 2>/dev/null | wc -l | tr -d ' ')"
	if [[ "${n:-0}" -gt 0 ]]; then
		echo "HAVE $gsm_name (enabled versions=$n)"
		return 0
	fi
	put_secret "$gsm_name" "$value"
}

JWT_GSM="$(echo "$OUT" | jq -r '.jwt_secret_id.value')"
GLOBAL_PAY_WEBHOOK_GSM="$(echo "$OUT" | jq -r '.global_pay_webhook_secret_id.value')"
ADYEN_GSM="$(echo "$OUT" | jq -r '.adyen_webhook_secret_id.value')"
STRIPE_GSM="$(echo "$OUT" | jq -r '.stripe_webhook_secret_id.value')"
MAPS_GSM="$(echo "$OUT" | jq -r '.google_maps_api_key_secret_id.value')"
KAFKA_GSM="$(echo "$OUT" | jq -r '.kafka_bootstrap_secret.value')"
INTERNAL_GSM="$(echo "$OUT" | jq -r '.internal_api_key_secret_id.value')"
PAYME_GSM="$(echo "$OUT" | jq -r '.payme_webhook_secret_id.value')"
CLICK_GSM="$(echo "$OUT" | jq -r '.click_webhook_secret_id.value')"
GP_SERVICE_GSM="$(echo "$OUT" | jq -r '.global_pay_service_id_secret_id.value')"
GP_USER_GSM="$(echo "$OUT" | jq -r '.global_pay_username_secret_id.value')"
GP_PASS_GSM="$(echo "$OUT" | jq -r '.global_pay_password_secret_id.value')"
REDIS_GSM="$(echo "$OUT" | jq -r '.redis_auth_secret_id.value')"

# Prefer real env values when provided.
put_secret "$JWT_GSM" "${JWT_SECRET:-}"
put_secret "$GLOBAL_PAY_WEBHOOK_GSM" "${GLOBAL_PAY_WEBHOOK_SECRET:-}"
put_secret "$ADYEN_GSM" "${ADYEN_WEBHOOK_SECRET:-}"
put_secret "$STRIPE_GSM" "${STRIPE_WEBHOOK_SECRET:-}"
put_secret "$MAPS_GSM" "${GOOGLE_MAPS_API_KEY:-}"
put_secret "$INTERNAL_GSM" "${INTERNAL_API_KEY:-}"
put_secret "$PAYME_GSM" "${PAYME_WEBHOOK_SECRET:-}"
put_secret "$CLICK_GSM" "${CLICK_WEBHOOK_SECRET:-}"
put_secret "$GP_SERVICE_GSM" "${GLOBAL_PAY_SERVICE_ID:-}"
put_secret "$GP_USER_GSM" "${GLOBAL_PAY_USERNAME:-}"
put_secret "$GP_PASS_GSM" "${GLOBAL_PAY_PASSWORD:-}"
put_secret "$REDIS_GSM" "${REDIS_AUTH:-${REDIS_PASSWORD:-}}"

if [[ -n "${KAFKA_BROKERS:-}" ]]; then
	put_secret "$KAFKA_GSM" "${KAFKA_BROKERS}"
elif [[ -n "${kafka_bootstrap_servers:-}" ]]; then
	put_secret "$KAFKA_GSM" "${kafka_bootstrap_servers}"
fi

# Atomic ESO: every ExternalSecret remoteRef needs ≥1 version.
# Core secrets without env fall back to placeholder only when ENSURE_ES_STUBS=1 (default).
ENSURE_ES_STUBS="${ENSURE_ES_STUBS:-1}"
if [[ "$ENSURE_ES_STUBS" == "1" ]]; then
	echo "Ensuring ExternalSecret remoteRefs have ≥1 enabled version..."
	# Core — prefer real values; stub only if still empty (ops must replace before production).
	ensure_version "$JWT_GSM" "${JWT_SECRET:-$UNUSED_RAIL_PLACEHOLDER}"
	ensure_version "$INTERNAL_GSM" "${INTERNAL_API_KEY:-$UNUSED_RAIL_PLACEHOLDER}"
	ensure_version "$MAPS_GSM" "${GOOGLE_MAPS_API_KEY:-$UNUSED_RAIL_PLACEHOLDER}"
	ensure_version "$GLOBAL_PAY_WEBHOOK_GSM" "${GLOBAL_PAY_WEBHOOK_SECRET:-$UNUSED_RAIL_PLACEHOLDER}"
	ensure_version "$GP_SERVICE_GSM" "${GLOBAL_PAY_SERVICE_ID:-$UNUSED_RAIL_PLACEHOLDER}"
	ensure_version "$GP_USER_GSM" "${GLOBAL_PAY_USERNAME:-$UNUSED_RAIL_PLACEHOLDER}"
	ensure_version "$GP_PASS_GSM" "${GLOBAL_PAY_PASSWORD:-$UNUSED_RAIL_PLACEHOLDER}"
	ensure_version "$REDIS_GSM" "${REDIS_AUTH:-${REDIS_PASSWORD:-$UNUSED_RAIL_PLACEHOLDER}}"
	# Unused PSP rails — stub is expected until that rail goes live.
	ensure_version "$ADYEN_GSM" "${ADYEN_WEBHOOK_SECRET:-$UNUSED_RAIL_PLACEHOLDER}"
	ensure_version "$STRIPE_GSM" "${STRIPE_WEBHOOK_SECRET:-$UNUSED_RAIL_PLACEHOLDER}"
	ensure_version "$PAYME_GSM" "${PAYME_WEBHOOK_SECRET:-$UNUSED_RAIL_PLACEHOLDER}"
	ensure_version "$CLICK_GSM" "${CLICK_WEBHOOK_SECRET:-$UNUSED_RAIL_PLACEHOLDER}"
fi

echo "phase0-secrets-ok"
exit 0
