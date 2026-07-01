#!/usr/bin/env bash
# Render pegasusX K8s manifests with image URLs and GCP placeholders replaced.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

usage() {
  cat <<EOF
Usage: export required vars, then run this script.

Required:
  GCP_PROJECT_ID          GCP project id
  GCP_RUNTIME_SA_EMAIL    Runtime GSA email (terraform output backend_runtime_service_account)
  BACKEND_IMAGE           Container image for backend-go (incl. pegasusx-setup)
  AI_WORKER_IMAGE         Container image for ai-worker

Optional:
  JWT_SECRET_GSM, GLOBAL_PAY_GSM, ADYEN_GSM, STRIPE_GSM  Secret Manager secret ids
  GOOGLE_MAPS_GSM                                        Google Maps API key secret id
  OUT_DIR                                                 Output directory (default: artifacts/k8s-rendered)

Quick start:
  cp .env.k8s.example .env.k8s   # edit values
  set -a && source .env.k8s && set +a
  bash scripts/render_k8s_manifests.sh
  kubectl apply -f artifacts/k8s-rendered/

Local docker images (after make docker-build):
  export BACKEND_IMAGE=pegasusx-backend:local
  export AI_WORKER_IMAGE=pegasusx-ai-worker:local
EOF
}

if [[ -f "$ROOT/.env.k8s" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT/.env.k8s"
  set +a
fi

missing=()
[[ -z "${GCP_PROJECT_ID:-}" ]] && missing+=("GCP_PROJECT_ID")
[[ -z "${GCP_RUNTIME_SA_EMAIL:-}" ]] && missing+=("GCP_RUNTIME_SA_EMAIL")
[[ -z "${BACKEND_IMAGE:-}" ]] && missing+=("BACKEND_IMAGE")
[[ -z "${AI_WORKER_IMAGE:-}" ]] && missing+=("AI_WORKER_IMAGE")

if ((${#missing[@]} > 0)); then
  echo "render_k8s_manifests: missing: ${missing[*]}" >&2
  echo >&2
  usage >&2
  exit 1
fi

JWT_SECRET_GSM="${JWT_SECRET_GSM:-pegasusx-prod-jwt-secret}"
GLOBAL_PAY_GSM="${GLOBAL_PAY_GSM:-pegasusx-prod-global-pay-webhook-secret}"
ADYEN_GSM="${ADYEN_GSM:-pegasusx-prod-adyen-webhook-secret}"
STRIPE_GSM="${STRIPE_GSM:-pegasusx-prod-stripe-webhook-secret}"
GOOGLE_MAPS_GSM="${GOOGLE_MAPS_GSM:-pegasusx-prod-google-maps-api-key}"

SPANNER_PROJECT_VAL="${SPANNER_PROJECT:-pegasusx-prod}"
SPANNER_INSTANCE_VAL="${SPANNER_INSTANCE:-pegasusx-instance}"
SPANNER_DATABASE_VAL="${SPANNER_DATABASE:-pegasusx-db}"
REDIS_ADDR_VAL="${REDIS_ADDR:-redis.pegasusx.svc.cluster.local:6379}"
KAFKA_BROKERS_VAL="${KAFKA_BROKERS:-kafka.pegasusx.svc.cluster.local:9092}"

OUT_DIR="${OUT_DIR:-${ROOT}/artifacts/k8s-rendered}"
mkdir -p "$OUT_DIR"

render() {
  local src=$1
  local dest=$2
  sed \
    -e "s|PEGASUSX_BACKEND_GO_IMAGE_PLACEHOLDER|${BACKEND_IMAGE}|g" \
    -e "s|PEGASUSX_AI_WORKER_IMAGE_PLACEHOLDER|${AI_WORKER_IMAGE}|g" \
    -e "s|PEGASUSX_GCP_RUNTIME_SA_PLACEHOLDER|${GCP_RUNTIME_SA_EMAIL}|g" \
    -e "s|PEGASUSX_GCP_PROJECT_PLACEHOLDER|${GCP_PROJECT_ID}|g" \
    -e "s|PEGASUSX_JWT_SECRET_GSM_NAME_PLACEHOLDER|${JWT_SECRET_GSM}|g" \
    -e "s|PEGASUSX_GLOBAL_PAY_WEBHOOK_GSM_NAME_PLACEHOLDER|${GLOBAL_PAY_GSM}|g" \
    -e "s|PEGASUSX_ADYEN_WEBHOOK_GSM_NAME_PLACEHOLDER|${ADYEN_GSM}|g" \
    -e "s|PEGASUSX_STRIPE_WEBHOOK_GSM_NAME_PLACEHOLDER|${STRIPE_GSM}|g" \
    -e "s|PEGASUSX_GOOGLE_MAPS_API_KEY_GSM_NAME_PLACEHOLDER|${GOOGLE_MAPS_GSM}|g" \
    -e "s|SPANNER_PROJECT: \"pegasusx-prod\"|SPANNER_PROJECT: \"${SPANNER_PROJECT_VAL}\"|g" \
    -e "s|SPANNER_INSTANCE: \"pegasusx-instance\"|SPANNER_INSTANCE: \"${SPANNER_INSTANCE_VAL}\"|g" \
    -e "s|SPANNER_DATABASE: \"pegasusx-db\"|SPANNER_DATABASE: \"${SPANNER_DATABASE_VAL}\"|g" \
    -e "s|REDIS_ADDR: \"redis.pegasusx.svc.cluster.local:6379\"|REDIS_ADDR: \"${REDIS_ADDR_VAL}\"|g" \
    -e "s|KAFKA_BROKERS: \"kafka.pegasusx.svc.cluster.local:9092\"|KAFKA_BROKERS: \"${KAFKA_BROKERS_VAL}\"|g" \
    "$src" >"$dest"
}

render infra/k8s/namespace.yaml "$OUT_DIR/namespace.yaml"
render infra/k8s/serviceaccount.yaml "$OUT_DIR/serviceaccount.yaml"
render infra/k8s/backend-go/configmap.yaml "$OUT_DIR/backend-go-configmap.yaml"
render infra/k8s/backend-go/deployment.yaml "$OUT_DIR/backend-go-deployment.yaml"
render infra/k8s/backend-go/service.yaml "$OUT_DIR/backend-go-service.yaml"
render infra/k8s/backend-go/migrate-job.yaml "$OUT_DIR/backend-go-migrate-job.yaml"
render infra/k8s/ai-worker/configmap.yaml "$OUT_DIR/ai-worker-configmap.yaml"
render infra/k8s/ai-worker/deployment.yaml "$OUT_DIR/ai-worker-deployment.yaml"
render infra/k8s/ai-worker/service.yaml "$OUT_DIR/ai-worker-service.yaml"
render infra/k8s/external-secrets/backend-go-externalsecret.yaml "$OUT_DIR/backend-go-externalsecret.yaml"

echo "k8s-render-ok out=${OUT_DIR}"
