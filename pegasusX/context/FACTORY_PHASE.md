# pegasusX FACTORY_ADMIN Role — Phased Execution Ledger

**Scope:** pegasusX only · **Parent plan:** `VEGETABLE_PLAN.md` §2.3  
**Last updated:** 2026-06-15 (FA-7/8/9 factory cross-client parity batch).

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

---

## Phase FA-7 — Production blockers (auth refresh, manifest/supply mutations)

| ID | Feature | Portal | Android | iOS | Status |
|----|---------|--------|---------|-----|--------|
| FA7-01 | Token refresh | `auth.ts` | `NetworkModule` | `APIClient` | **WIRED** (pre-existing) |
| FA7-02 | Manifest lifecycle mutations | ✓ | ✓ | ✓ | **WIRED** (pre-existing, SSMR green) |
| FA7-03 | Supply/transfer transitions | ✓ | ✓ | ✓ | **WIRED** (pre-existing) |

**Exit:** No P0 path drift; factory JWT refresh on all surfaces.

---

## Phase FA-8 — Parity wiring (notifications inbox, analytics depth)

| ID | Feature | Portal | Android | iOS | Status |
|----|---------|--------|---------|-----|--------|
| FA8-01 | Notification inbox | portal top bar (pre-existing) | **wired native** | **wired native** | **WIRED** (`PX_E2E_FACTORY_NOTIFICATION_INBOX_OK`) |
| FA8-02 | Mark all read | ✓ | ✓ | ✓ | **WIRED** |
| FA8-03 | Analytics overview | `/analytics` | `AnalyticsScreen` | `AnalyticsView` | **WIRED** (FA-1) |

---

## Phase FA-9 — Client policy & platform gating

| ID | Feature | Portal | Android | iOS | SSMR | Status |
|----|---------|--------|---------|-----|------|--------|
| FA9-01 | `GET /v1/platform/client-policy?role=FACTORY` | **ClientPolicyBanner** | dashboard banner | dashboard banner | — | **WIRED** |
| FA9-02 | SSMR marker | — | — | — | `PX_E2E_FACTORY_CLIENT_POLICY_OK` | **WIRED** |
| FA9-03 | Firebase OTP | login TODO | — | — | — | **Open** (deferred) |

---

## Verification

```bash
cd pegasusX/apps/backend-go && go build ./cmd/ssmr-smokecheck/
cd pegasusX/apps/factory-app-android && ./gradlew :app:compileDebugKotlin
cd pegasusX/apps/factory-app-ios && xcodebuild -scheme FactoryAppIOS -destination 'platform=iOS Simulator,name=iPhone 17' CODE_SIGNING_ALLOWED=NO build
cd pegasusX && make test-ssmr-infra   # PX_E2E_FACTORY_* markers
```

---

## Next execution batch

1. ~~Analytics Spanner overview~~ — FA-1 backend
2. ~~ai-worker TopicFreezeLocks consumer~~ — PX-FREEZE-01–04
3. ~~Factory native analytics screens + DTO alignment~~ — FA1-04
4. ~~Import session wizard + async ai-worker consumer~~ — supplier row
5. ~~FA-7/8/9 cross-client parity batch~~ — **CLOSED** (2026-06-15)
6. **Cross-role next** — DRIVER row per `VEGETABLE_PLAN.md` §3
