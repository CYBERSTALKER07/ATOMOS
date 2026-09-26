#!/usr/bin/env bash
# Phase-5b progress gate: Gate 5 / §8.10 Phase 2 ParentOrders + multi-supplier checkout.
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "[1/6] ADR + env reopen ..."
test -f docs/MULTI_TENANCY_GATE5_PHASE2.md
grep -q 'ALLOW_MULTI_SUPPLIER_REGISTER=true' .env.ssmr.example
grep -q 'MAX_SUPPLIERS=10' .env.ssmr.example
grep -q 'MULTI_SUPPLIER_CHECKOUT_ENABLED' .env.ssmr.example
grep -q 'MULTI_SUPPLIER_CHECKOUT_ENABLED' .env.example
grep -q 'ALLOW_MULTI_SUPPLIER_REGISTER=true' .env.ssmr || true

echo "[2/6] Schema ParentOrders + Orders.ParentOrderId ..."
test -f apps/backend-go/schema/migrations/20260817_parent_orders.ddl
grep -q 'CREATE TABLE ParentOrders' apps/backend-go/schema/migrations/20260817_parent_orders.ddl
grep -q 'ParentOrderId' apps/backend-go/schema/migrations/20260817_parent_orders.ddl
grep -q 'CREATE TABLE ParentOrders' apps/backend-go/schema/spanner.ddl
grep -q 'Idx_Orders_ByParentOrder' apps/backend-go/schema/spanner.ddl

echo "[3/6] Cart ListByRetailerAll + clear-all + preserve SupplierId ..."
grep -q 'ListByRetailerAll' apps/backend-go/retailer/repository_cart.go
grep -q 'ClearCartAll' apps/backend-go/retailer/repository_cart.go
grep -q 'HandleCartClear' apps/backend-go/retailer/core_handlers.go
grep -q 'Preserve explicit line SupplierId' apps/backend-go/retailer/core_handlers.go

echo "[4/6] Split engine + parent rollup ..."
(cd apps/backend-go && go test ./order/ -run 'TestUnifiedCheckout_|TestRollupParentStatus' -count=1)
grep -q 'MultiSupplierCheckoutEnabled' apps/backend-go/order/multi_supplier_checkout.go
grep -q 'unifiedCheckoutMultiSupplier\|ParentOrderID' apps/backend-go/order/unified_checkout.go
grep -q 'HandleGetParentOrder' apps/backend-go/order/parent_orders.go
grep -q 'parent-orders' apps/backend-go/orderroutes/routes.go

echo "[5/6] SSMR markers + parent-order smokecheck ..."
grep -q 'PX_E2E_MULTI_SUPPLIER_REGISTER' contracts/ssmr_ecosystem_markers.json
grep -q 'PX_E2E_PARENT_ORDER_SPLIT' contracts/ssmr_ecosystem_markers.json
grep -q 'PX_E2E_PARENT_ORDER_ISOLATION' contracts/ssmr_ecosystem_markers.json
grep -q 'runParentOrderE2E\|PX_E2E_PARENT_ORDER_SPLIT' apps/backend-go/cmd/ssmr-smokecheck/e2e_parent_order.go
grep -q 'case "parent-order"' apps/backend-go/cmd/ssmr-smokecheck/main.go
grep -q 'runParentOrderE2E' apps/backend-go/cmd/ssmr-smokecheck/e2e_check.go

echo "[6/6] Audit + progress docs ..."
grep -q 'MULTI_TENANCY_GATE5_PHASE2' PLATFORM_AUDIT.md
test -f docs/session-2026-08-07/PHASE5_PHASE2_PROGRESS.md

echo "phase5b-gate-ok"
