#!/usr/bin/env bash
# PX91 planning ingest load smoke — 100 concurrent signal ingest requests.
# Usage: BASE_URL=http://localhost:8080 COOKIE='supplier_jwt=...' ./scripts/planning_ingest_load.sh

set -euo pipefail
BASE_URL="${BASE_URL:-http://localhost:8080}"
COOKIE="${COOKIE:-}"
CONCURRENCY="${CONCURRENCY:-100}"

if [[ -z "$COOKIE" ]]; then
  echo "COOKIE required (supplier session)" >&2
  exit 1
fi

tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT

seq 1 "$CONCURRENCY" | xargs -P "$CONCURRENCY" -I{} curl -s -o /dev/null -w "%{http_code}\n" \
  -X POST "$BASE_URL/v1/supplier/planning/signals/ingest" \
  -H "Content-Type: application/json" \
  -H "Cookie: $COOKIE" \
  -H "Idempotency-Key: load-test-{}" \
  -d '{"source":"load_test","payload":{"seq":{}}}' >> "$tmp"

accepted=$(grep -c '^202$' "$tmp" || true)
unavailable=$(grep -c '^503$' "$tmp" || true)
other=$(grep -cvE '^(202|503)$' "$tmp" || true)
echo "accepted=$accepted unavailable=$unavailable other=$other concurrency=$CONCURRENCY"
