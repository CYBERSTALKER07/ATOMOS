#!/usr/bin/env bash
# Phase-5 progress gate: Gate 5 / runtime multi-tenancy Phase 1 foundations.
# Full ADR (docs/MULTI_TENANCY_GATE5_PHASE1.md) spans ~12 weeks; this gate locks
# Week 0–1 spine + early vertical/outbox/rate-limit scaffolding.
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "[1/7] Phase-4 gate (regression) ..."
if [[ "${PHASE5_SKIP_REGRESSION:-}" == "1" ]]; then
  echo "skipping prior gates (PHASE5_SKIP_REGRESSION=1)"
else
  bash scripts/phase4_gate.sh
fi

echo "[2/7] Registration freeze (Week 0) ..."
(cd apps/backend-go && go test ./supplier/ -run 'TestResolveRegistrationSupplierID' -count=1)
grep -q 'ALLOW_MULTI_SUPPLIER_REGISTER' apps/backend-go/supplier/service.go
grep -q 'AllowMultiSupplierRegister' apps/backend-go/bootstrap/bootstrap.go

echo "[3/7] TenantContext spine (Week 1) ..."
(cd apps/backend-go && go test ./auth/ -run 'TestTenant|TestRequireTenant|TestAttachTenant' -count=1)
grep -q 'AttachTenantFromClaims' apps/backend-go/main.go
grep -q 'RequireTenant' apps/backend-go/main.go
grep -q 'withPartnerTenant\|Source: "partner"' apps/backend-go/partner/auth.go

echo "[4/7] Order tenant resolution + IDOR unit path (Weeks 2–3) ..."
(cd apps/backend-go && go test ./order/ -run 'TestGetOrderForTenantIDOR|TestResolveSupplierIDForCreatePrefersTenant|TestResolveSupplierScopePrefersTenant' -count=1)
grep -q 'loadOrderForRequest' apps/backend-go/order/service.go
grep -q 'loadOrderForRequest' apps/backend-go/order/status_context.go
grep -q 'PreferTenantSupplierID\|SeedSupplierID\|seedSupplierID' apps/backend-go/payment/service.go
grep -q 'resolveSupplierID(ctx)' apps/backend-go/payment/retailer_checkout.go
grep -q 'seedSupplierID\|SeedSupplierID' apps/backend-go/order/service.go
grep -q 'resolveSupplierScope' apps/backend-go/order/service.go
grep -q 'runTenantE2E\|PX_E2E_TENANT_REGISTER_FROZEN' apps/backend-go/cmd/ssmr-smokecheck/e2e_tenant.go
grep -q 'case "tenant"' apps/backend-go/cmd/ssmr-smokecheck/main.go
grep -q 'IsRegistered": true' apps/backend-go/auth/seed_scope.go
grep -q 'BackfillSupplierID' apps/backend-go/outbox/backfill.go

echo "[5/7] Outbox fair interleave + SupplierId migration (Week 9 start) ..."
(cd apps/backend-go && go test ./outbox/ -run 'TestFairInterleave|TestSupplierIDFromPayload|TestEventRowMap|TestResolveSupplierID' -count=1)
test -f apps/backend-go/schema/migrations/20260811_outbox_supplier_id.ddl
grep -q 'SupplierId' apps/backend-go/schema/migrations/20260811_outbox_supplier_id.ddl
test -f apps/backend-go/schema/migrations/20260819_outbox_supplier_id_not_null.ddl
grep -q 'ALTER COLUMN SupplierId STRING(64) NOT NULL' apps/backend-go/schema/migrations/20260819_outbox_supplier_id_not_null.ddl
grep -q 'SupplierId       STRING(64)    NOT NULL' apps/backend-go/schema/spanner.ddl
grep -q 'PlatformSupplierID' apps/backend-go/outbox/outbox.go
grep -q 'ResolveSupplierID' apps/backend-go/outbox/outbox.go
grep -q 'SupplierID' apps/backend-go/outbox/outbox.go
grep -q 'SupplierIDFromPayload' apps/backend-go/outbox/outbox.go
grep -q 'SupplierId' apps/backend-go/order/settlement_hardening.go

echo "[6/7] Warehouse/driver/payload/factory/workers tenant scope + rate limits ..."
grep -q 'TenantFromContext' apps/backend-go/warehouse/dispatch_execute.go
grep -q 'resolveSupplierScope' apps/backend-go/warehouse/ops_dispatch_handlers.go
grep -q 'func (s \*Service) resolveSupplierScope' apps/backend-go/driver/service.go
grep -q 'func (s \*Service) resolveSupplierScope' apps/backend-go/payload/service.go
grep -q 'func (s \*Service) resolveSupplierScope' apps/backend-go/factory/service.go
grep -q 'func (s \*Service) resolveSupplierScope' apps/backend-go/retailer/service.go
grep -q 'TenantFromContext' apps/backend-go/supplier/session_scope.go
grep -q 'PreferTenantSupplierID' apps/backend-go/creditnote/handlers.go
grep -q 'PreferTenantSupplierID' apps/backend-go/cashrecon/handlers.go
grep -q 'JOIN Drivers' apps/backend-go/cashrecon/escalation_worker.go
grep -q 'Source: "worker"' apps/backend-go/cashrecon/escalation_worker.go
grep -q 'tenant:' apps/backend-go/bootstrap/reliability_middleware.go
grep -q 'TENANT_RATE_LIMIT_SHARED\|tenantRateLimitShared' apps/backend-go/bootstrap/reliability_middleware.go
grep -q 'Week 11 fail-closed' apps/backend-go/auth/tenant.go
grep -q 'TENANT_CONTEXT_ENFORCED must be true' apps/backend-go/bootstrap/config_validate.go
grep -q 'PreferTenantSupplierID' apps/backend-go/bootstrap/bootstrap.go
grep -q 'seedSupplierID\|SeedSupplierID' apps/backend-go/warehouse/service.go
grep -q 'seedSupplierID\|SeedSupplierID' apps/backend-go/driver/service.go
grep -q 'seedSupplierID\|SeedSupplierID' apps/backend-go/payload/service.go
grep -q 'seedSupplierID\|SeedSupplierID' apps/backend-go/factory/service.go
grep -q 'seedSupplierID\|SeedSupplierID' apps/backend-go/retailer/service.go
grep -q 'StartSupplierIDBackfill\|BackfillSupplierID' apps/backend-go/runtime_workers.go
grep -q 'OUTBOX_SUPPLIER_BACKFILL' .env.example
grep -q 'backfill-outbox-supplier-id' apps/backend-go/cmd/backfill-outbox-supplier-id/main.go
(cd apps/backend-go && go test ./outbox/ ./bootstrap/ -run 'TestFairInterleave|TestSupplierIDFromPayload|TestEventRowMap|TestSupplierBackfillEnabled|TestReliabilityActorKey_TenantShared' -count=1)
(cd apps/backend-go && go test ./auth/ ./supplier/ ./retailer/ ./cashrecon/ ./payment/ ./warehouse/ ./bootstrap/ -run 'TestPreferTenantSupplierID|TestPreferTenantSupplierIDNoSeedWhenEnforced|TestScopedSupplierIDPrefersTenant|TestResolveSupplierScopePrefersTenant|TestResolveSupplierIDPrefersTenant|TestEscalate|TestValidateProductionProfile_RejectsDisabledTenantEnforcement|TestReliabilityActorKey_TenantShared' -count=1)

echo "[7/7] Env + SSMR markers documented ..."
grep -q 'ALLOW_MULTI_SUPPLIER_REGISTER=false' .env.example
grep -q 'TENANT_CONTEXT_ENFORCED=true' .env.ssmr.example
grep -q 'PX_E2E_TENANT_REGISTER_FROZEN' contracts/ssmr_ecosystem_markers.json
grep -q 'PX_E2E_TENANT_ORDER_ISOLATION' contracts/ssmr_ecosystem_markers.json
grep -q 'PX_E2E_OUTBOX_TENANT_PARTITION' contracts/ssmr_ecosystem_markers.json
grep -q 'PX_E2E_TENANT_RATE_LIMIT' contracts/ssmr_ecosystem_markers.json

echo "phase5-gate-ok"
