#!/usr/bin/env bash
# Create backend-go-secrets from Secret Manager when External Secrets Operator is unavailable.
set -euo pipefail

: "${GCP_PROJECT_ID:?GCP_PROJECT_ID required}"
NAMESPACE="${K8S_NAMESPACE:-pegasusx}"

prefix="${SECRET_PREFIX:-pegasusx-prod}"

jwt="$(gcloud secrets versions access latest --secret="${prefix}-jwt-secret" --project="${GCP_PROJECT_ID}")"
global_pay="$(gcloud secrets versions access latest --secret="${prefix}-global-pay-webhook-secret" --project="${GCP_PROJECT_ID}")"
adyen="$(gcloud secrets versions access latest --secret="${prefix}-adyen-webhook-secret" --project="${GCP_PROJECT_ID}")"
stripe="$(gcloud secrets versions access latest --secret="${prefix}-stripe-webhook-secret" --project="${GCP_PROJECT_ID}")"

kubectl -n "$NAMESPACE" create secret generic backend-go-secrets \
  --from-literal=jwt-secret="$jwt" \
  --from-literal=global-pay-webhook-secret="$global_pay" \
  --from-literal=adyen-webhook-secret="$adyen" \
  --from-literal=stripe-webhook-secret="$stripe" \
  --dry-run=client -o yaml | kubectl apply -f -

echo "backend-go-secrets-ok namespace=${NAMESPACE}"
