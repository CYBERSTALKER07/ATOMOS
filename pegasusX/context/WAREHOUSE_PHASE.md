# pegasusX WAREHOUSE_ADMIN Role — Phased Execution Ledger

**Scope:** pegasusX only · **Reference:** pegasus `warehouse-portal` (read-only)  
**Parent plan:** `VEGETABLE_PLAN.md` §2.2 · `Phase Next: Replenishment + Supply`  
**Last updated:** 2026-06-14

## Status model

`TODO` → `IN_PROGRESS` → `WIRED` → `E2E_SSMR_GREEN` → `PROD_CANDIDATE`

---

## Phase WH-1 — Replenishment insights durability (P1)

| ID | Feature | Backend | Portal | Android | iOS | Status |
|----|---------|---------|--------|---------|-----|--------|
| WH1-01 | Durable `ReplenishmentInsights` DDL | `schema/spanner.ddl` | — | — | — | **WIRED** |
| WH1-02 | List insights from Spanner | `GET /v1/warehouse/replenishment/insights` | `/replenishment` | `ReplenishmentScreen` | `ReplenishmentView` | **WIRED** |
| WH1-03 | Approve → `FactoryInternalTransfers` + outbox | `POST .../insights/{id}/approve` | approve action | approve button | approve button | **WIRED** |
| WH1-04 | Dismiss insight | `POST .../insights/{id}/dismiss` | dismiss action | dismiss button | dismiss button | **WIRED** |
| WH1-05 | SSMR seed + markers | `auth/seed_scope.go` | — | — | — | **WIRED** |
| WH1-06 | Demand forecast uses insight burn | `warehouse/demand_products.go` | `/demand-forecast` | — | — | **WIRED** |

**Exit:** Insights survive pod restart; approve creates durable transfer row; warehouse + factory role rows can list/act on the same Spanner authority.

---

## Verification (warehouse row)

```bash
cd pegasusX/apps/backend-go && go test ./warehouse/...
cd pegasusX && make test-ssmr-infra   # PX_E2E_WAREHOUSE_REPLENISHMENT_OK
cd pegasusX && make parity-contract-full
```

---

## Next execution batch

1. ~~Replenishment insights durability~~ — **this session**
2. **Replenishment engine** — background threshold scanner (`replenishment/engine` port) to auto-generate insights
3. **Native fleet live map** — warehouse Android/iOS consume `/v1/warehouse/ops/fleet/live-map` (portal already wired)
