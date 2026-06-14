# pegasusX FACTORY_ADMIN Role — Phased Execution Ledger

**Scope:** pegasusX only · **Parent plan:** `VEGETABLE_PLAN.md` §2.3  
**Last updated:** 2026-06-14

## Status model

`TODO` → `IN_PROGRESS` → `WIRED` → `E2E_SSMR_GREEN` → `PROD_CANDIDATE`

---

## Phase FA-1 — Analytics backend durability (P1)

| ID | Feature | Backend | Portal | Android | iOS | Status |
|----|---------|---------|--------|---------|-----|--------|
| FA1-01 | Spanner analytics overview | `factory/analytics_spanner.go` → `GET /v1/factory/analytics/overview` | `/analytics` | `AnalyticsScreen` | `AnalyticsView` | **WIRED** |
| FA1-02 | Replenishment insights (shared path) | `warehouseroutes` + `RequireReplenishmentInsightsScope` | `/insights` | `InsightsScreen` | `InsightsView` | **E2E_SSMR_GREEN** (`PX_E2E_FACTORY_INSIGHTS_OK`) |
| FA1-03 | SSMR analytics marker | `runFactoryAnalyticsOverviewE2E` | — | — | — | **E2E_SSMR_GREEN** (`PX_E2E_FACTORY_ANALYTICS_OK`) |
| FA1-04 | Native analytics DTO alignment | wire contract | — | `FactoryAnalyticsOverview` | `FactoryAnalyticsOverview` | **WIRED** |

**Exit:** Factory analytics overview reads Spanner when infra is live; native Android/iOS render KPI overview from dashboard workflow launch (parity with portal `/analytics`).

---

## Cross-role — AI worker freeze locks (WIRED)

| ID | Feature | Backend / worker | Status |
|----|---------|------------------|--------|
| PX-FREEZE-01 | Freeze event fields on wire | `events.DispatchLockEvent` + `entity_type` / `entity_id` / `ttl_seconds` | **WIRED** |
| PX-FREEZE-02 | Producer emits entity scope | `warehouse/service.go` `emitDispatchLockAcquireOutbox` | **WIRED** |
| PX-FREEZE-03 | ai-worker consumer | `apps/ai-worker/freeze_registry.go` + `TopicFreezeLocks` reader | **WIRED** |
| PX-FREEZE-04 | ORDER_CREATED guard | `handleOrderCreated` skips synthesis when frozen | **WIRED** |
| PX-FREEZE-05 | SSMR topic provision | `infra/docker-compose.ssmr.yml` kafka-init + `KAFKA_TOPIC_FREEZE_LOCKS` + `ssmr-smokecheck kafka` expected set | **CLOSED** |

---

## Verification

```bash
cd pegasusX/apps/backend-go && go test ./factory/... ./warehouse/...
cd pegasusX/apps/ai-worker && go test ./...
cd pegasusX && make test-ssmr-infra   # PX_E2E_FACTORY_ANALYTICS_OK, PX_E2E_FACTORY_INSIGHTS_OK
```

---

## Next execution batch

1. ~~Analytics Spanner overview~~ — FA-1 backend
2. ~~ai-worker TopicFreezeLocks consumer~~ — PX-FREEZE-01–04
3. ~~Factory native analytics screens + DTO alignment~~ — FA1-04
4. **Cross-role next** — full import session wizard or Boss-picked role row per `VEGETABLE_PLAN.md` §3
