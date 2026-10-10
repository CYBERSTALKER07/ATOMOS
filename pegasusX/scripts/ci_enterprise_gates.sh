#!/usr/bin/env bash
# CI entry for enterprise / tenancy progress gates (P2-18).
# Chains Phase 2→5c + analytics-tenancy without re-running nested regressions
# after the first Phase-2 (which still includes Phase-1 unless skipped).
#
# When PHASE2_SKIP_REGRESSION=1 (CI after backend-spanner), this script still
# applies schema so Phase-4/5 Spanner unit paths have a database.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

export SPANNER_EMULATOR_HOST="${SPANNER_EMULATOR_HOST:-localhost:9010}"
export SPANNER_PROJECT="${SPANNER_PROJECT:-pegasusx-local}"
export SPANNER_INSTANCE="${SPANNER_INSTANCE:-pegasusx-instance}"
export SPANNER_DATABASE="${SPANNER_DATABASE:-pegasusx-db}"

echo "==> [ci] Wait for Spanner emulator at ${SPANNER_EMULATOR_HOST} ..."
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

echo "==> [ci] Apply Spanner schema (cmd/setup) ..."
(cd apps/backend-go && go run ./cmd/setup)

echo "==> [ci] Phase-2 gate (includes Phase-1 unless PHASE2_SKIP_REGRESSION=1) ..."
bash scripts/phase2_gate.sh

echo "==> [ci] Phase-3 gate (skip nested Phase-2) ..."
PHASE3_SKIP_REGRESSION=1 bash scripts/phase3_gate.sh

echo "==> [ci] Phase-4 gate (skip nested Phase-3) ..."
PHASE4_SKIP_REGRESSION=1 bash scripts/phase4_gate.sh

echo "==> [ci] Phase-5 gate (skip nested Phase-4) ..."
PHASE5_SKIP_REGRESSION=1 bash scripts/phase5_gate.sh

echo "==> [ci] Phase-5b gate (ParentOrders / multi-supplier) ..."
bash scripts/phase5b_gate.sh

echo "==> [ci] Phase-5c gate (GlobalProducts) ..."
bash scripts/phase5c_gate.sh

echo "==> [ci] Analytics column tenancy gate ..."
bash scripts/analytics_tenancy_gate.sh

echo "ci-enterprise-gates-ok"
