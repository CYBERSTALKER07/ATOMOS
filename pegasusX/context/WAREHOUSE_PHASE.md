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

## Phase WH-2 — Replenishment engine (P1)

| ID | Feature | Backend | Portal | Android | iOS | Status |
|----|---------|---------|--------|---------|-----|--------|
| WH2-01 | Threshold + burn-rate scanner | `replenishment/engine.go` | — | — | — | **WIRED** |
| WH2-02 | Cron (4h default, env override) | `main.go` + `REPLENISHMENT_CRON_DISABLED` | — | — | — | **WIRED** |
| WH2-03 | Manual trigger runs engine | `POST /v1/supplier/replenishment/trigger` | supplier ops | — | — | **WIRED** |
| WH2-04 | CRITICAL auto-transfer + outbox | `replenishment/engine.go` | — | — | — | **WIRED** |
| WH2-05 | Unit tests (urgency math) | `replenishment/engine_test.go` | — | — | — | **WIRED** |

**Adaptations vs pegasus:** `LineItemsJson` burn/unfulfilled aggregation (no `OrderLineItems` table); `SupplierInventoryV2.QuantityOnHand`; `Products.UnitVolumeVU`; default 2-day factory lead; per-SKU in-transit skipped (aggregate `FactoryInternalTransfers` only).

**Exit:** Engine writes `ReplenishmentInsights` rows; supplier trigger returns `insights_generated` / `transfers_created`; CRITICAL insights auto-approve + create transfer atomically.

---

## Phase WH-3 — Native fleet live map (P1)

| ID | Feature | Backend | Portal | Android | iOS | Status |
|----|---------|---------|--------|---------|-----|--------|
| WH3-01 | Fleet live map API | `GET /v1/warehouse/ops/fleet/live-map` | `FleetLiveMapPanel` | `FleetLiveMapSection` | `FleetLiveMapSection` | **WIRED** |
| WH3-02 | Full-screen map route | — | dispatch + dashboard | `FleetLiveMapScreen` | `FleetLiveMapView` | **WIRED** |
| WH3-03 | WS-accelerated refresh | warehouse hub events | `use-warehouse-fleet-live-map` | `WarehouseRealtimeSignals` | `WarehouseRealtimeHub` | **WIRED** |
| WH3-04 | SSMR marker | `e2e_check.go` | — | — | — | **WIRED** |

**Exit:** Warehouse Android/iOS render sealed/dispatched manifest polylines + animated driver markers; 15s poll + WS bump; SSMR asserts `PX_E2E_WAREHOUSE_FLEET_LIVE_MAP_OK`.

---

## Verification (warehouse row)

```bash
cd pegasusX/apps/backend-go && go test ./warehouse/... ./replenishment/...
cd pegasusX && make test-ssmr-infra   # PX_E2E_WAREHOUSE_REPLENISHMENT_OK, PX_E2E_WAREHOUSE_FLEET_LIVE_MAP_OK
cd pegasusX && make parity-contract-full
```

---

## Next execution batch

1. ~~Replenishment insights durability~~ — WH-1
2. ~~Replenishment engine~~ — WH-2
3. ~~Native fleet live map~~ — **WH-3 verified this session** (implementation pre-existed; SSMR marker + ledger closed)
4. **Cross-role next** — factory/warehouse analytics native depth, or Boss-picked role row per `VEGETABLE_PLAN.md` §3 (notification inbox persistence **WIRED** — see `SUPPLIER_PHASE.md` NI-01–NI-04)
