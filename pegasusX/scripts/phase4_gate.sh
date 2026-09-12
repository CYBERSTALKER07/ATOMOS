#!/usr/bin/env bash
# Phase-4 exit gate: autonomy on evidence foundations.
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "[1/6] Phase-3 gate (regression) ..."
if [[ "${PHASE4_SKIP_REGRESSION:-}" == "1" ]]; then
  echo "skipping prior gates (PHASE4_SKIP_REGRESSION=1)"
else
  bash scripts/phase3_gate.sh
fi

echo "[2/6] Partial allocation unit path ..."
(cd apps/backend-go && go test ./allocation/ -run 'TestPartialAllocationEnabledEnv' -count=1)

echo "[3/6] S&OP no longer only sku-projection literals ..."
! grep -n 'sku-projection-%d' apps/backend-go/planning/service.go
grep -q 'supply_request_projected\|count_calibrated' apps/backend-go/planning/service.go
grep -q 'CapacityModel' apps/backend-go/planning/service.go

echo "[4/6] Optimizer replicas ≥ 1 on SSMR + staging overlays ..."
grep -A6 'name: optimizer-core' infra/k8s/overlays/ssmr/kustomization.yaml | grep -q 'value: 1'
grep -A6 'name: optimizer-core' infra/k8s/overlays/staging/kustomization.yaml | grep -q 'value: 1'

echo "[5/6] Shadow flags documented + place flip gate ..."
test -f docs/AUTO_ORDER_PLACE_FLIP.md
bash scripts/auto_order_place_flip_check.sh
grep -q 'AUTO_ORDER_SHADOW_ENABLED=true' .env.ssmr.example
grep -q 'PARTIAL_ALLOCATION_ENABLED=true' .env.ssmr.example

echo "[6/6] Planning + allocation packages compile ..."
(cd apps/backend-go && go test ./planning/ ./allocation/ -count=1)

echo "phase4-gate-ok"
