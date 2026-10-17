#!/usr/bin/env bash
# Create backend-go-secrets from Secret Manager when External Secrets Operator is unavailable.
# Key set matches ExternalSecret / Deployment secretKeyRef (P0-8 — 12 keys).
set -euo pipefail

: "${GCP_PROJECT_ID:?GCP_PROJECT_ID required}"
NAMESPACE="${K8S_NAMESPACE:-pegasusx}"
# Default matches live GSM tenant (ssmr-shaped); override for a future prod tenant.
prefix="${SECRET_PREFIX:-pegasusx-ssmr}"

access() {
	local suffix=$1
	gcloud secrets versions access latest --secret="${prefix}-${suffix}" --project="${GCP_PROJECT_ID}"
}

jwt="$(access jwt-secret)"
internal="$(access internal-api-key)"
global_pay_webhook="$(access global-pay-webhook-secret)"
adyen="$(access adyen-webhook-secret)"
stripe="$(access stripe-webhook-secret)"
payme="$(access payme-webhook-secret)"
click="$(access click-webhook-secret)"
gp_service="$(access global-pay-service-id)"
gp_user="$(access global-pay-username)"
gp_pass="$(access global-pay-password)"
maps="$(access google-maps-api-key)"
redis="$(access redis-auth)"

kubectl -n "$NAMESPACE" create secret generic backend-go-secrets \
  --from-literal=jwt-secret="$jwt" \
  --from-literal=internal-api-key="$internal" \
  --from-literal=global-pay-webhook-secret="$global_pay_webhook" \
  --from-literal=adyen-webhook-secret="$adyen" \
  --from-literal=stripe-webhook-secret="$stripe" \
  --from-literal=payme-webhook-secret="$payme" \
  --from-literal=click-webhook-secret="$click" \
  --from-literal=global-pay-service-id="$gp_service" \
  --from-literal=global-pay-username="$gp_user" \
  --from-literal=global-pay-password="$gp_pass" \
  --from-literal=google-maps-api-key="$maps" \
  --from-literal=redis-password="$redis" \
  --dry-run=client -o yaml | kubectl apply -f -

echo "backend-go-secrets-ok namespace=${NAMESPACE} prefix=${prefix}"
