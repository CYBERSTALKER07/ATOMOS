#!/usr/bin/env bash
# Export K8s render env from pegasusX/infra/terraform outputs, then render manifests.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TF_DIR="${ROOT}/infra/terraform"
TAG="${IMAGE_TAG:-latest}"

if ! command -v jq >/dev/null 2>&1; then
  echo "render_k8s_from_terraform: jq required (brew install jq)" >&2
  exit 1
fi

cd "$TF_DIR"
OUT="$(terraform output -json 2>/dev/null || echo '{}')"

if [[ "$OUT" == "{}" ]] || [[ "$(echo "$OUT" | jq 'length')" -eq 0 ]]; then
  cat <<EOF >&2
render_k8s_from_terraform: no terraform outputs — run terraform apply first.

  cd pegasusX/infra/terraform
  terraform apply -var="project_id=v-o-i-d" -var="tenant_slug=prod" -var="enable_gke=true" ...

Then push images to Artifact Registry and re-run this script.
EOF
  exit 1
fi

GAR_URL="$(echo "$OUT" | jq -r '.artifact_registry_url.value // empty')"
RUNTIME_SA="$(echo "$OUT" | jq -r '.backend_runtime_service_account.value // empty')"
JWT_GSM="$(echo "$OUT" | jq -r '.jwt_secret_id.value // empty')"
GLOBAL_PAY_GSM="$(echo "$OUT" | jq -r '.global_pay_webhook_secret_id.value // empty')"
ADYEN_GSM="$(echo "$OUT" | jq -r '.adyen_webhook_secret_id.value // empty')"
STRIPE_GSM="$(echo "$OUT" | jq -r '.stripe_webhook_secret_id.value // empty')"

if [[ -z "$GAR_URL" || -z "$RUNTIME_SA" ]]; then
  echo "render_k8s_from_terraform: artifact_registry_url or backend_runtime_service_account missing — was enable_gke=true on apply?" >&2
  exit 1
fi

# project id is the third path segment: ...pkg.dev/PROJECT/repo
GCP_PROJECT_ID="$(echo "$GAR_URL" | awk -F/ '{print $(NF-1)}')"

export GCP_PROJECT_ID
export GCP_RUNTIME_SA_EMAIL="$RUNTIME_SA"
export BACKEND_IMAGE="${GAR_URL}/backend-go:${TAG}"
export AI_WORKER_IMAGE="${GAR_URL}/ai-worker:${TAG}"
export JWT_SECRET_GSM="$JWT_GSM"
export GLOBAL_PAY_GSM="$GLOBAL_PAY_GSM"
export ADYEN_GSM="$ADYEN_GSM"
export STRIPE_GSM="$STRIPE_GSM"

echo "render_k8s_from_terraform: project=${GCP_PROJECT_ID} tag=${TAG}"
echo "  BACKEND_IMAGE=${BACKEND_IMAGE}"
echo "  AI_WORKER_IMAGE=${AI_WORKER_IMAGE}"
echo "  GCP_RUNTIME_SA_EMAIL=${GCP_RUNTIME_SA_EMAIL}"

bash "${ROOT}/scripts/render_k8s_manifests.sh"
