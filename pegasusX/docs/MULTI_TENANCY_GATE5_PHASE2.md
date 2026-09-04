# ADR: Gate 5 / §8.10 Phase 2 — Multi-supplier cart / ParentOrders

**Status:** Accepted — Wired (backend)  
**Date:** 2026-08-11  
**Deciders:** Platform engineering  
**Related:** [MULTI_TENANCY_GATE5_PHASE1.md](./MULTI_TENANCY_GATE5_PHASE1.md); [PLATFORM_AUDIT.md](../PLATFORM_AUDIT.md) §8.10 Phase 2; [PHASE5_PHASE2_PROGRESS.md](./session-2026-08-07/PHASE5_PHASE2_PROGRESS.md) 

---

## Context

Phase 1 delivered request-scoped tenancy and live IDOR/outbox/rate-limit markers. Cart rows already carry per-line `SupplierId`, but reads and `UnifiedCheckout` still collapse to one trading-partner tenant and one child `Orders` row.

Audit Phase 2: one cart, many suppliers → parent rollup + per-supplier child orders (credit, inventory, warehouse, dispatch each stay tenant-scoped).

## Decision

1. **Reopen multi-supplier mint** under `ALLOW_MULTI_SUPPLIER_REGISTER` + `MAX_SUPPLIERS` (SSMR default on / 10). Production stays fail-closed unless ops sets the flag. Phase 1 `TENANT_CONTEXT_ENFORCED` remains on.
2. Introduce **`ParentOrders`** and `Orders.ParentOrderId`. Children remain the operational unit.
3. When `MULTI_SUPPLIER_CHECKOUT_ENABLED` (default on for `PEGASUSX_ENV=ssmr`): group checkout lines by product/line `SupplierId`, create one parent + N children (including N=1), stamp `ParentOrderId`, return `parent_order_id` + `supplier_orders[]`.
4. Line/`Products.SupplierId` is authoritative at checkout — JWT trading-partner must not override mixed-cart lines.
5. **All-or-nothing MVP:** if any leg fails, cancel any children already created in the attempt and fail the checkout.
6. Retailer multi-partner UI is **out of scope** (backend + SSMR proof only).

## Non-goals

| Out | Why |
|-----|-----|
| Retailer catalog/cart multi-partner UX | Follow-up client program |
| Cross-supplier escrow / single PSP invoice | Each child keeps own credit/payment path |
| GlobalProducts / marketplace / KYB console | §8.10 Phases 3–5 / Gate 6 |
| Partial-commit split (some suppliers succeed) | Follow-up after soak |

## Flags

| Flag | SSMR | Notes |
|------|------|--------|
| `ALLOW_MULTI_SUPPLIER_REGISTER` | `true` | Dynamic ecosystem mint |
| `MAX_SUPPLIERS` | `10` | Cap |
| `MULTI_SUPPLIER_CHECKOUT_ENABLED` | default on when `PEGASUSX_ENV=ssmr` | Split + ParentOrders |

## Success criteria

1. Second supplier registers under flag (`PX_E2E_MULTI_SUPPLIER_REGISTER_OK`).
2. Mixed-supplier cart checkout → 1 parent + 2 children (`PX_E2E_PARENT_ORDER_SPLIT_OK`).
3. Supplier A cannot read supplier B child (`PX_E2E_PARENT_ORDER_ISOLATION_OK`).
4. `phase5b_gate.sh` / unit suite green.
5. Audit §8.10 Phase 2 marked Wired (backend); UI residual noted.

## References

- [`order/unified_checkout.go`](../apps/backend-go/order/unified_checkout.go)
- [`retailer/repository_cart.go`](../apps/backend-go/retailer/repository_cart.go)
- [`supplier/service.go`](../apps/backend-go/supplier/service.go) `resolveRegistrationSupplierID`
