# pegasusX FACTORY_ADMIN Role — Phased Execution Ledger

**Scope:** pegasusX only · **Parent plan:** `VEGETABLE_PLAN.md` §2.3  
**Last updated:** 2026-06-17 (FA9-03 Firebase OTP).

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
| FA9-03 | Firebase OTP | `firebaseAuth.ts` OTP + emulator | `FirebaseAuthHelper` phone OTP + password dev | `FirebaseAuthHelper` phone OTP + password dev | `PX_E2E_FACTORY_FIREBASE_OTP_OK` (when `FACTORY_FIREBASE_TEST_ID_TOKEN` set) | **WIRED** |

---

## Phase FA-10 — Shell & dashboard UX parity (portal inbox, KPI relabel, native workflow sync)

| ID | Feature | Portal | Android | iOS | Status |
|----|---------|--------|---------|-----|--------|
| FA10-01 | Notification inbox top bar | **NotificationPanel** + `useNotifications` | pre-existing | pre-existing | **WIRED** |
| FA10-02 | Dashboard KPI relabel | Gate Exceptions → `/manifest-exceptions` | pre-existing | pre-existing | **WIRED** |
| FA10-03 | Dashboard workflow launch grid | manifests + exceptions + analytics action cards | pre-existing | **create transfer + insights rows** | **WIRED** |
| FA10-04 | Loading state polish | pre-existing | pre-existing | `FactoryLoadingState` on dashboard | **WIRED** |

**Exit:** Factory portal shell matches native notification/mark-read UX; dashboard KPI grid and workflow launches align across portal + Android + iOS; pegasusX additive screens (manifests, analytics, gate exceptions) surfaced from dashboard on all surfaces.

---

## Phase FA-11 — Deep native UI/UX parity (iOS)

| ID | Feature | Portal | Android | iOS | Status |
|----|---------|--------|---------|-----|--------|
| FA11-01 | Shared KPI / list / status primitives | — | — (iOS-only batch) | `KpiTile`, `FactoryStatusBadge`, `FactorySectionHeader` | **WIRED** |
| FA11-02 | Theme tokens (`statusTint`, `readableMaxWidth`) | — | — | `LabTheme` + `labReadableWidth()` | **WIRED** |
| FA11-03 | Dashboard KPI grid + workflow chrome | pre-existing FA-10 | pre-existing | `DashboardView` — adaptive `KpiTile`, section header, `FactoryErrorView` | **WIRED** |
| FA11-04 | Transfers list status badges + states | — | — | `TransferListView` — `FactoryStatusBadge`, empathetic loading/empty | **WIRED** |
| FA11-05 | Loading bay sections + badges | — | — | `LoadingBayView` — `FactorySectionHeader`, `FactoryStatusBadge`, bay states | **WIRED** |
| FA11-06 | Manifests list polish | — | — | `ManifestsView` — section header, semantic badges, refresh toolbar | **WIRED** |
| FA11-07 | Analytics KPI grid | pre-existing FA-1 | pre-existing | `AnalyticsView` — `KpiTile`, alert chips, section headers | **WIRED** |
| FA11-08 | Fleet + staff roster badges | — | — | `FleetView`, `StaffView` — `FactoryStatusBadge`, refresh toolbars | **WIRED** |
| FA11-09 | Loading/error/empty states | — | — | `FactoryLoadingView` / `FactoryErrorView` + existing `FactoryStateView` | **WIRED** |

**UI audit vs pegasus reference:** pegasus `factory-app-ios` is thinner (no manifests/analytics/insights sheets, raw `ProgressView` loading, plain status capsules). pegasusX iOS is ahead on ops depth (FA-10 workflow rows, notification inbox, client-policy). FA-11 aligns pegasusX iOS with supplier SP-7 / warehouse WH-11 discipline (shared primitives, semantic tints, refresh toolbars, empathetic loading copy).

**Exit:** Primary factory iOS ops screens share SwiftUI component discipline; UI-only — no new SSMR markers.

---

## Phase FA-11A — Deep native UI/UX parity (Android)

| ID | Feature | Portal | Android | iOS | Status |
|----|---------|--------|---------|-----|--------|
| FA11A-01 | Shared UI kit (`FactoryUiComponents`, `FactoryState`) | — | KPI tiles, metric tiles, status chips, list cards, section titles, loading/error/empty panes | pre-existing FA-10 | **WIRED** |
| FA11A-02 | Dashboard KPI grid + hero metrics | pre-existing FA-10 | `DashboardScreen` — `FactoryKpiTile`, `FactoryMetricTile`, 160dp adaptive grid | pre-existing FA-10 | **WIRED** |
| FA11A-03 | Transfers pipeline | — | `TransferListScreen` — state panes, `FactoryStatusChip`, `FactoryMetricTile` | — | **WIRED** |
| FA11A-04 | Loading bay queues | — | `LoadingBayScreen` — `FactorySectionHeader`, `FactoryInlineEmptyState`, shared chips/metrics | — | **WIRED** |
| FA11A-05 | Manifest list | pre-existing | `ManifestListScreen` — `FactoryOpsListCard`, summary card | pre-existing | **WIRED** |
| FA11A-06 | Analytics overview | pre-existing FA-1 | `AnalyticsScreen` — `FactoryKpiTile` grid, `FactorySectionTitle` | pre-existing FA-1 | **WIRED** |
| FA11A-07 | Fleet + staff rosters | — | `FleetScreen`, `StaffScreen` — summary KPI rows, `FactoryOpsListCard` | — | **WIRED** |

**UI audit vs pegasus reference:** pegasus `factory-app-android` matches pegasusX on transfer/loading-bay/fleet/staff workflow and state panes; pegasusX is ahead on manifests, analytics, gate exceptions, notification inbox, and client-policy banner (FA-7/8/9/10). FA-11A aligns pegasusX Android with supplier SP-7 / warehouse WH-11A discipline (shared primitives, 160dp adaptive grids, `IconButton` refresh, empathetic loading copy, semantic status chips).

**Exit:** Primary factory Android ops screens share M3 discipline with supplier/warehouse native patterns. UI-only — no new SSMR markers.

---

## Phase FA-11P — Deep factory-portal UI/UX (portal)

| ID | Feature | pegasus ref | pegasusX portal | Status |
|----|---------|-------------|-----------------|--------|
| FA11P-01 | `PageChrome` skeleton + `EmptyState` | `Skeleton.tsx`, `EmptyState.tsx` | `PageChrome` variants (`dashboard`/`table`/`form`) | **WIRED** |
| FA11P-02 | KPI tile structure (`KpiStatCard`, `PageSection`) | desk KPI grid | `KpiStatCard.tsx`, `PageSection.tsx` | **WIRED** |
| FA11P-03 | Transfers pipeline KPI + filter section | transfers table layout | `PageChrome` + `KpiStatGrid` + `PageSection` | **WIRED** |
| FA11P-04 | Loading bay kanban + bay KPIs | loading-bay kanban | `PageChrome` + `KpiStatGrid` + `PageSection` columns | **WIRED** |
| FA11P-05 | Manifests lifecycle table | — (pegasusX-only page) | KPI summary + `PageSection` pipeline table | **WIRED** |
| FA11P-06 | Gate exceptions inbox + DLQ KPIs | — (pegasusX-only page) | `PageChrome` + runtime banner + `KpiStatGrid` | **WIRED** |
| FA11P-07 | Analytics overview KPI grid | — (pegasusX-only page) | `PageChrome` + `KpiStatGrid` + exceptions CTA | **WIRED** |
| FA11P-08 | Supply requests queue + transitions | supply-requests table | `PageChrome` + `KpiStatGrid` + `PageSection` | **WIRED** |

**FA-11P audit gaps (intentional / blocked):** Dashboard hero/action-card layout (pegasusX ahead post FA-10); pegasus ref lacks manifests/analytics/gate-exceptions nav (pegasusX additive); fleet/staff/insights/payload-override still on legacy inline headers; `daily_activity` chart depth (API returns array, no chart component yet).

**Exit:** Component-level desk tokens, skeleton loaders, KPI structure, and section headers on transfers, loading bay, manifests, gate exceptions, analytics, and supply. UI-only — no new SSMR.

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
6. ~~FA-10 shell/dashboard UX parity~~ — **CLOSED** (2026-06-15)
7. ~~FA-11 iOS deep UI/UX parity~~ — **CLOSED** (2026-06-15)
8. ~~FA-11A Android deep UI/UX parity~~ — **CLOSED** (2026-06-15)
9. ~~FA-11P portal deep UI/UX (component-level)~~ — **CLOSED** (2026-06-15)
10. ~~FA9-03 Firebase phone OTP (portal + Android + iOS)~~ — **CLOSED** (2026-06-17)
11. **Cross-role next** — per `VEGETABLE_PLAN.md` §3 (warehouse WH9 Firebase OTP deferred)
