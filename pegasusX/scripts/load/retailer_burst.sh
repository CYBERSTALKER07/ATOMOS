#!/usr/bin/env bash
# Lightweight health burst when k6 is not installed. Not a substitute for k6 cert.
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8180}"
CONCURRENCY="${CONCURRENCY:-40}"
REQUESTS="${REQUESTS:-120}"
HEALTH_URL="${BASE_URL%/}/v1/health"

echo "Retailer burst (health smoke): ${REQUESTS} requests @ concurrency ${CONCURRENCY} -> ${HEALTH_URL}"

if command -v vegeta >/dev/null 2>&1; then
  echo "GET ${HEALTH_URL}" | vegeta attack -duration=15s -rate="${CONCURRENCY}/15s" | vegeta report
  exit 0
fi

failures=0
success=0
pids=()

flush_batch() {
  local pid
  # Bash 3.2 + set -u: "${pids[@]}" on an empty array is an error.
  if [[ ${#pids[@]} -eq 0 ]]; then
    return 0
  fi
  for pid in "${pids[@]}"; do
    if wait "$pid"; then
      success=$((success + 1))
    else
      failures=$((failures + 1))
    fi
  done
  pids=()
}

for ((i = 1; i <= REQUESTS; i++)); do
  curl -fsS "$HEALTH_URL" >/dev/null 2>/dev/null &
  pids+=($!)
  if ((${#pids[@]} >= CONCURRENCY)); then
    flush_batch
  fi
done
flush_batch

echo "Burst complete: success=${success} failures=${failures} (install k6 for full retailer/supplier cert mix)"

max_fail=$((REQUESTS / 20))
if [[ "$max_fail" -lt 1 ]]; then
  max_fail=1
fi
if [[ "$failures" -gt "$max_fail" ]]; then
  echo "Health burst failed: ${failures}/${REQUESTS} requests returned non-2xx" >&2
  exit 1
fi
