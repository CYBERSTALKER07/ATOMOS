#!/usr/bin/env bash
# Phase-1 exit gate: money and law complete.
#   1. Phase-0 money-path gate still green (invariants must not regress)
#   2. AR: credit leave opens invoice; AR off rejects credit leave (fail-closed)
#   3. Refunds: cap, reversal legs, idempotency, provider-failure truthfulness,
#      fiscal corrective chain (Spanner emulator)
#   4. Payouts: net math, replay idempotency, bank-file fail-closed
#   5. Billing: fee schedule resolution, monthly AR invoice, idempotent re-run
#   6. Soliq recorded-contract suite (submit + corrective, golden-stable)
#   7. Global Pay simulator: refund + capture happy paths
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

export SPANNER_EMULATOR_HOST="${SPANNER_EMULATOR_HOST:-localhost:9010}"
export SPANNER_PROJECT="${SPANNER_PROJECT:-pegasusx-local}"
export SPANNER_INSTANCE="${SPANNER_INSTANCE:-pegasusx-instance}"
export SPANNER_DATABASE="${SPANNER_DATABASE:-pegasusx-db}"

echo "[1/7] Phase-0 money-path gate (regression) ..."
bash scripts/money_path_gate.sh

echo "[2/7] AR activation rules (order, emulator) ..."
(cd apps/backend-go && PARITY_REQUIRE_SPANNER=1 go test ./order/ -run 'TestMoneyPathGate_ShopClosed' -count=1)

echo "[3/7] Refunds (order, emulator) ..."
(cd apps/backend-go && PARITY_REQUIRE_SPANNER=1 go test ./order/ -run 'TestRefund_' -count=1)

echo "[4/7] Payouts (emulator) ..."
(cd apps/backend-go && PARITY_REQUIRE_SPANNER=1 go test ./payout/ -count=1)

echo "[5/7] Billing fee schedule + monthly AR invoice (emulator) ..."
(cd apps/backend-go && PARITY_REQUIRE_SPANNER=1 go test ./internal/services/billing/ -count=1)

echo "[6/7] Soliq recorded-contract suite (incl. corrective) ..."
(cd apps/backend-go && go test ./order/ -run 'TestMySoliqContract|TestSoliqSigner' -count=1)

echo "[7/7] Global Pay simulator: refund + capture happy paths ..."
(cd apps/backend-go && go test ./payment/ -run 'TestGlobalPayRefundAgainstSimulator|TestGlobalPayCaptureAgainstSimulator' -count=1)

echo "phase1-gate-ok"
