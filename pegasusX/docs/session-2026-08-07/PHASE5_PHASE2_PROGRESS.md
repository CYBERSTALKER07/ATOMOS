# Gate 5 / §8.10 Phase 2 — Progress

> **HISTORICAL / FROZEN — session progress note; do not treat as current gap SoT.**
> Living residuals: [`../PROD_READINESS_SEQUENCE.md`](../PROD_READINESS_SEQUENCE.md) · [`ECOSYSTEM_GAP_REGISTER_2026-08-12.md`](./ECOSYSTEM_GAP_REGISTER_2026-08-12.md) · [`../DOCS_SOURCE_OF_TRUTH.md`](../DOCS_SOURCE_OF_TRUTH.md).


**Date:** 2026-08-11  
**ADR:** [`docs/MULTI_TENANCY_GATE5_PHASE2.md`](../MULTI_TENANCY_GATE5_PHASE2.md)  
**Gate:** `bash scripts/phase5b_gate.sh` → `phase5b-gate-ok`  
**Focused smoke:** `go run ./cmd/ssmr-smokecheck parent-order` → **all three markers OK** (2026-08-11 live SSMR)

## Live proof (2026-08-11)

| Step | Result |
|------|--------|
| `make ssmr-infra-up` | Stack healthy (`/v1/health` 200) |
| ParentOrders DDL | Applied via setup + `make apply-parent-orders-ddl` (checksum match) |
| `ssmr-smokecheck parent-order` | `PX_E2E_MULTI_SUPPLIER_REGISTER_OK`, `PX_E2E_PARENT_ORDER_SPLIT_OK`, `PX_E2E_PARENT_ORDER_ISOLATION_OK` |

## Shipped (backend slice)

| Area | Status |
|------|--------|
| ADR + SSMR env reopen (`ALLOW_MULTI_SUPPLIER_REGISTER`, `MAX_SUPPLIERS=10`) | Done |
| `ParentOrders` migration + `spanner.ddl` mirror + `Orders.ParentOrderId` | Done |
| Cart `ListByRetailerAll` / `ClearCartAll`; POST preserves explicit line `SupplierId` | Done |
| `MULTI_SUPPLIER_CHECKOUT_ENABLED` split in `UnifiedCheckout` (parent even for N=1; all-or-nothing) | Done |
| `GET /v1/retailer/parent-orders/{id}` on-read rollup | Done |
| Markers + `parent-order` smokecheck + early wire in full e2e | Done |
| Audit §8.10 Phase 2 → Wired (backend); UI residual noted | Done |

## Markers

| Marker | Meaning |
|--------|---------|
| `PX_E2E_MULTI_SUPPLIER_REGISTER_OK` / `_SKIPPED` | Second supplier mint under flag |
| `PX_E2E_PARENT_ORDER_SPLIT_OK` / `_SKIPPED` | Mixed cart → 1 parent + 2 children |
| `PX_E2E_PARENT_ORDER_ISOLATION_OK` / `_SKIPPED` | Supplier A cannot read supplier B child |

## Residuals

- Retailer multi-partner catalog/cart UI (explicit non-goal this program)
- Partial-commit split (follow-up); cross-supplier escrow (out of scope)

## Next forks (pick one)

1. ~~**Enterprise Phase 0** — money-path correctness~~ → **Wired** — see [`PHASE0_MONEY_PATH_PROGRESS.md`](./PHASE0_MONEY_PATH_PROGRESS.md) (`money-path-gate-ok`)
2. ~~**Enterprise Phase 1** — money and law~~ → **Wired (backend/simulator)** — see [`PHASE1_MONEY_LAW_PROGRESS.md`](./PHASE1_MONEY_LAW_PROGRESS.md) (`phase1-gate-ok`); owner creds residual
3. ~~**Enterprise Phase 2** — partner integration~~ → **Wired** — see [`PHASE2_COMPLETION.md`](./PHASE2_COMPLETION.md) (`phase2-gate-ok`); Phase 6 cert residual
4. ~~**§8.10 Phase 3** — GlobalProducts master~~ → **Wired (backend)** — see [`PHASE5_PHASE3_PROGRESS.md`](./PHASE5_PHASE3_PROGRESS.md) (`phase5c-gate-ok`)
5. ~~**Enterprise Phase 3** — operational truth~~ → **Wired (backend/API/SLO stubs)** — see [`PHASE3_PROGRESS.md`](./PHASE3_PROGRESS.md) (`phase3-gate-ok`); UI/offline/live monitoring residual
6. ~~**Enterprise Phase 4 autonomy**~~ → **Wired (foundations)** — see [`PHASE4_COMPLETION.md`](./PHASE4_COMPLETION.md) (`phase4-gate-ok`); soak/prod/place-flip residual
7. ~~**Enterprise Phase 5 / tenancy soak**~~ → **Wired** — see [`PHASE5_PROGRESS.md`](./PHASE5_PROGRESS.md) (`phase5-gate-ok`); Outbox NOT NULL closed; analytics column residual
8. ~~**Analytics column tenancy**~~ → **Wired (RoutePerformanceAnalytics)** — see [`ANALYTICS_COLUMN_TENANCY_PROGRESS.md`](./ANALYTICS_COLUMN_TENANCY_PROGRESS.md) (`analytics-tenancy-gate-ok`)
9. **§8.10 Phase 4–5 / Enterprise Phase 6** — marketplace commerce / KYB / cert (decision-gated)
10. **Client 10/10 residuals** — deferred
