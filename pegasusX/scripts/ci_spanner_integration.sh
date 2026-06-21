#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

export SPANNER_EMULATOR_HOST="${SPANNER_EMULATOR_HOST:-localhost:9010}"
export SPANNER_PROJECT="${SPANNER_PROJECT:-pegasusx-local}"
export SPANNER_INSTANCE="${SPANNER_INSTANCE:-pegasusx-instance}"
export SPANNER_DATABASE="${SPANNER_DATABASE:-pegasusx-db}"
export JWT_SECRET="${JWT_SECRET:-ci-spanner-integration-secret}"

echo "Waiting for Spanner emulator at ${SPANNER_EMULATOR_HOST} ..."
for i in $(seq 1 60); do
  if (echo >/dev/tcp/${SPANNER_EMULATOR_HOST%:*}/${SPANNER_EMULATOR_HOST#*:}) 2>/dev/null; then
    break
  fi
  if [[ "$i" -eq 60 ]]; then
    echo "Spanner emulator not reachable" >&2
    exit 1
  fi
  sleep 1
done

echo "Applying Spanner schema via cmd/setup ..."
(cd apps/backend-go && go run ./cmd/setup)

echo "Running Spanner integration tests (PARITY_REQUIRE_SPANNER=1) ..."
(cd apps/backend-go && PARITY_REQUIRE_SPANNER=1 go test ./outbox/... -count=1 -short)

echo "spanner-integration-ok"
