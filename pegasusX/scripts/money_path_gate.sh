#!/usr/bin/env bash
# Phase-0 exit gate: money-path correctness and legal safety.
# Proves against the Spanner emulator:
#   (a) capture failure never writes CAPTURED
#   (b) duplicate idempotency key never double-records (unique index)
#   (c) empty Global Pay credentials produce hard errors (unit)
#   (d) shop-closed credit debt is always recorded
# plus repo hygiene gates (gitleaks, TODO:Inject, mock control tower).
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

export SPANNER_EMULATOR_HOST="${SPANNER_EMULATOR_HOST:-localhost:9010}"
export SPANNER_PROJECT="${SPANNER_PROJECT:-pegasusx-local}"
export SPANNER_INSTANCE="${SPANNER_INSTANCE:-pegasusx-instance}"
export SPANNER_DATABASE="${SPANNER_DATABASE:-pegasusx-db}"

echo "[1/5] Waiting for Spanner emulator at ${SPANNER_EMULATOR_HOST} ..."
for i in $(seq 1 60); do
  if (echo >/dev/tcp/${SPANNER_EMULATOR_HOST%:*}/${SPANNER_EMULATOR_HOST#*:}) 2>/dev/null; then
    break
  fi
  if [[ "$i" -eq 60 ]]; then
    echo "Spanner emulator not reachable (start it: make infra-up)" >&2
    exit 1
  fi
  sleep 1
done

echo "[2/5] Applying schema (incl. payment idempotency unique indexes) ..."
(cd apps/backend-go && go run ./cmd/setup)

echo "[3/5] Money-path gate tests (order + payment) ..."
(cd apps/backend-go && PARITY_REQUIRE_SPANNER=1 go test ./order/ ./payment/ -run 'TestMoneyPathGate|TestWorkerShopClosed' -count=1 -v)

echo "[4/5] Hygiene greps ..."
bash scripts/ci_fail_todo_inject.sh
bash scripts/ci_no_mock_control_tower.sh

echo "[5/5] Secrets scan ..."
if command -v gitleaks >/dev/null 2>&1; then
  gitleaks dir --config .gitleaks.toml .
else
  echo "gitleaks binary not found locally; CI runs gitleaks-action (secrets job)" >&2
fi

echo "money-path-gate-ok"
