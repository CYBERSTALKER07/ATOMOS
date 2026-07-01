#!/usr/bin/env bash
# PX-PROD-0: sync boss handoff secrets into GSM (names from terraform output).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TF_DIR="$ROOT/infra/terraform"
SECRETS_FILE="${SECRETS_FILE:-$ROOT/.env.staging.secrets}"

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

put_secret() {
	local gsm_name=$1
	local value=$2
	if [[ -z "$value" ]]; then
		echo "SKIP $gsm_name (empty)"
		return 0
	fi
	echo -n "$value" | gcloud secrets versions add "$gsm_name" --data-file=- --project "$(terraform output -raw project_id)" >/dev/null
	echo "OK   $gsm_name"
}

JWT_GSM="$(echo "$OUT" | jq -r '.jwt_secret_id.value')"
GLOBAL_PAY_GSM="$(echo "$OUT" | jq -r '.global_pay_webhook_secret_id.value')"
ADYEN_GSM="$(echo "$OUT" | jq -r '.adyen_webhook_secret_id.value')"
STRIPE_GSM="$(echo "$OUT" | jq -r '.stripe_webhook_secret_id.value')"
MAPS_GSM="$(echo "$OUT" | jq -r '.google_maps_api_key_secret_id.value')"
KAFKA_GSM="$(echo "$OUT" | jq -r '.kafka_bootstrap_secret.value')"

put_secret "$JWT_GSM" "${JWT_SECRET:-}"
put_secret "$GLOBAL_PAY_GSM" "${GLOBAL_PAY_WEBHOOK_SECRET:-}"
put_secret "$ADYEN_GSM" "${ADYEN_WEBHOOK_SECRET:-}"
put_secret "$STRIPE_GSM" "${STRIPE_WEBHOOK_SECRET:-}"
put_secret "$MAPS_GSM" "${GOOGLE_MAPS_API_KEY:-}"

if [[ -n "${KAFKA_BROKERS:-}" ]]; then
	put_secret "$KAFKA_GSM" "${KAFKA_BROKERS}"
elif [[ -n "${kafka_bootstrap_servers:-}" ]]; then
	put_secret "$KAFKA_GSM" "${kafka_bootstrap_servers}"
fi

# Optional payment rails — create ad-hoc GSM secrets if not in terraform module
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
	if gcloud secrets describe "$gsm" --project "$(terraform output -raw project_id)" >/dev/null 2>&1; then
		put_secret "$gsm" "$val"
	else
		echo "SKIP $gsm (not provisioned by terraform — add manually if needed)"
	fi
}

sync_optional INTERNAL_API_KEY internal-api-key
sync_optional GLOBAL_PAY_SERVICE_ID global-pay-service-id
sync_optional GLOBAL_PAY_USERNAME global-pay-username
sync_optional GLOBAL_PAY_PASSWORD global-pay-password
sync_optional PAYME_WEBHOOK_SECRET payme-webhook-secret
sync_optional CLICK_WEBHOOK_SECRET click-webhook-secret

echo "phase0-secrets-ok"
exit 0
