#!/usr/bin/env bash
# Phase-0 exit gate: money-path correctness and legal safety.
# Proves against the Spanner emulator:
#   (a) capture failure never writes CAPTURED
#   (b) duplicate idempotency key never double-records (unique index)
#   (c) empty Global Pay credentials produce hard errors (unit)
#   (d) shop-closed credit debt is always recorded
# plus repo hygiene gates (gitleaks, TODO:Inject, mock control tower,
# Kafka HA contracts, tracked-secret/tfstate hygiene).
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

export SPANNER_EMULATOR_HOST="${SPANNER_EMULATOR_HOST:-localhost:9010}"
export SPANNER_PROJECT="${SPANNER_PROJECT:-pegasusx-local}"
export SPANNER_INSTANCE="${SPANNER_INSTANCE:-pegasusx-instance}"
export SPANNER_DATABASE="${SPANNER_DATABASE:-pegasusx-db}"

echo "[1/7] Waiting for Spanner emulator at ${SPANNER_EMULATOR_HOST} ..."
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

echo "[2/7] Applying schema (incl. payment idempotency unique indexes) ..."
(cd apps/backend-go && go run ./cmd/setup)

echo "[3/7] Money-path gate tests (order + payment) ..."
(cd apps/backend-go && PARITY_REQUIRE_SPANNER=1 go test ./order/ ./payment/ -run 'TestMoneyPathGate|TestWorkerShopClosed' -count=1 -v)

echo "[4/7] Hygiene greps ..."
bash scripts/ci_fail_todo_inject.sh
bash scripts/ci_no_mock_control_tower.sh

echo "[5/7] Secrets scan ..."
if command -v gitleaks >/dev/null 2>&1; then
  gitleaks dir --config .gitleaks.toml .
else
  echo "gitleaks binary not found locally; CI runs gitleaks-action (secrets job)" >&2
fi

echo "[6/7] Kafka HA contracts ..."
bash scripts/ci_kafka_ha_gate.sh

echo "[7/7] Repo hygiene ..."
bash scripts/ci_repo_hygiene_gate.sh

echo "money-path-gate-ok"
