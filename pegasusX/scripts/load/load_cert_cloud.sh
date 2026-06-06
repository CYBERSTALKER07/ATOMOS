#!/usr/bin/env bash
# PX11-A1: load certification against a remote staging/prod API (production SLOs).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

BASE_URL="${PUBLIC_BASE_URL:-}"
if [[ -z "$BASE_URL" ]]; then
  echo "load-cert-cloud-error: set PUBLIC_BASE_URL (e.g. https://api.staging.example.com)" >&2
  exit 1
fi

export BASE_URL
export LOAD_PROFILE="${LOAD_PROFILE:-cert}"
export READ_P99_MS="${READ_P99_MS:-300}"
export ORDER_P99_MS="${ORDER_P99_MS:-800}"

echo "load-cert-cloud: BASE_URL=$BASE_URL profile=$LOAD_PROFILE read_p99=${READ_P99_MS}ms order_p99=${ORDER_P99_MS}ms"
bash scripts/load/load_cert.sh
echo "PX11_CLOUD_LOAD_CERT_OK"
