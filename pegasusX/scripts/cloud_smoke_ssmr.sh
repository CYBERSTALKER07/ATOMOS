#!/usr/bin/env bash
# PX11-A1: SSMR-equivalent smoke against a deployed pegasusX API.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

BASE_URL="${PUBLIC_BASE_URL:-}"
if [[ -z "$BASE_URL" ]]; then
  echo "cloud-smoke-error: set PUBLIC_BASE_URL" >&2
  exit 1
fi

BASE_URL="${BASE_URL%/}"
echo "cloud-smoke: health check $BASE_URL"
curl -fsS "$BASE_URL/healthz" | grep -q '"status":"ok"'

echo "cloud-smoke: client-policy"
curl -fsS "$BASE_URL/v1/platform/client-policy?role=DRIVER&platform=ios&version=1.0.0" | grep -q '"minimum_version"'

if [[ -x apps/backend-go/cmd/ssmr-smokecheck/ssmr-smokecheck ]] || command -v go >/dev/null 2>&1; then
  export PUBLIC_BASE_URL="$BASE_URL"
  echo "cloud-smoke: optional full e2e (requires seeded staging data)"
  echo "  run locally: PUBLIC_BASE_URL=$BASE_URL go run ./apps/backend-go/cmd/ssmr-smokecheck e2e"
fi

echo "PX11_CLOUD_SMOKE_OK"
