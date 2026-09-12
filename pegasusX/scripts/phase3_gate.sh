#!/usr/bin/env bash
# Phase-3 exit gate: operational truth (admin, flags, WMS API, SLOs).
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "[1/5] Phase-2 gate (regression) ..."
if [[ "${PHASE3_SKIP_REGRESSION:-}" == "1" ]]; then
  echo "skipping phase-2 (PHASE3_SKIP_REGRESSION=1)"
else
  bash scripts/phase2_gate.sh
fi

echo "[2/5] Platform admin tenant lifecycle ..."
(cd apps/backend-go && go test ./platformadmin/ -count=1)

echo "[3/5] Feature-flag evaluation + money-flag reason ..."
(cd apps/backend-go && go test ./featureflags/ -count=1)

echo "[4/5] PLATFORM_ADMIN role + routes present ..."
grep -q 'RolePlatformAdmin' apps/backend-go/auth/claims.go
grep -q 'platform-admin/tenants' apps/backend-go/platformadmin/handlers.go
grep -q 'platform-admin/flags' apps/backend-go/featureflags/handlers.go
grep -q 'ws-session' apps/backend-go/platformadmin/handlers.go
grep -q 'PlatformAdminRoom' apps/backend-go/ws/rooms.go
grep -q 'RolePlatformAdmin' apps/backend-go/ws/handler.go
grep -q 'confirmPickTask' apps/warehouse-app-android/app/src/main/java/com/pegasusx/warehouse/data/remote/WarehouseApi.kt
echo "platform-admin-routes-ok"

echo "[5/5] SLO docs + observability alert stubs ..."
test -f docs/PLATFORM_SLOS.md
grep -q 'outbox_lag_high' infra/terraform/observability.tf
grep -q 'fiscal_success_low' infra/terraform/observability.tf
grep -q 'capture_success_low' infra/terraform/observability.tf
grep -q 'outbox_relay_restarts_high' infra/terraform/observability.tf
grep -q 'outbox_dlq_depth_nonzero' infra/terraform/observability.tf
grep -q 'partner_webhook_success_low' infra/terraform/observability.tf
echo "slo-stubs-ok"

echo "phase3-gate-ok"
