#!/usr/bin/env bash
# Phase-5c progress gate: Gate 5 / §8.10 Phase 3 GlobalProducts master.
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "[1/5] ADR + flag ..."
test -f docs/MULTI_TENANCY_GATE5_PHASE3.md
grep -q 'GLOBAL_PRODUCTS_ENABLED=true' .env.ssmr.example
grep -q 'GLOBAL_PRODUCTS_ENABLED' .env.example

echo "[2/5] Schema GlobalProducts + offers + match queue + UoM ..."
test -f apps/backend-go/schema/migrations/20260818_global_products.ddl
grep -q 'CREATE TABLE GlobalProducts' apps/backend-go/schema/migrations/20260818_global_products.ddl
grep -q 'CREATE TABLE SupplierProductOffers' apps/backend-go/schema/migrations/20260818_global_products.ddl
grep -q 'CREATE TABLE ProductMatchQueue' apps/backend-go/schema/migrations/20260818_global_products.ddl
grep -q 'CREATE TABLE UnitsOfMeasure' apps/backend-go/schema/migrations/20260818_global_products.ddl
grep -q 'CREATE TABLE GlobalProducts' apps/backend-go/schema/spanner.ddl
grep -q 'Idx_GlobalProducts_ByGtin' apps/backend-go/schema/spanner.ddl

echo "[3/5] Package + routes + catalog hook ..."
grep -q 'MatchAndLink' apps/backend-go/globalproducts/service.go
grep -q 'OnProductUpserted' apps/backend-go/globalproducts/service.go
grep -q 'SetGlobalProductHook' apps/backend-go/catalog/service.go
grep -q 'global-products' apps/backend-go/globalproductsroutes/routes.go
grep -q 'GlobalProductsService' apps/backend-go/bootstrap/bootstrap.go
grep -q 'globalproductsroutes.RegisterRoutes' apps/backend-go/main.go

echo "[4/5] Unit tests + smoke wiring ..."
(cd apps/backend-go && go test ./globalproducts/ -count=1)
grep -q 'PX_E2E_GLOBAL_PRODUCT_GTIN_LINK' contracts/ssmr_ecosystem_markers.json
grep -q 'PX_E2E_GLOBAL_PRODUCT_FUZZY_QUEUE' contracts/ssmr_ecosystem_markers.json
grep -q 'PX_E2E_GLOBAL_PRODUCT_OFFERS_COMPARE' contracts/ssmr_ecosystem_markers.json
grep -q 'runGlobalProductsSmokeCheck\|PX_E2E_GLOBAL_PRODUCT_GTIN_LINK' apps/backend-go/cmd/ssmr-smokecheck/e2e_global_products.go
grep -q 'case "global-products"' apps/backend-go/cmd/ssmr-smokecheck/main.go

echo "[5/5] Audit + progress docs ..."
grep -q 'MULTI_TENANCY_GATE5_PHASE3\|GlobalProducts' PLATFORM_AUDIT.md
test -f docs/session-2026-08-07/PHASE5_PHASE3_PROGRESS.md

echo "phase5c-gate-ok"
