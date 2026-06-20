# pegasusX WAREHOUSE_ADMIN Role — Phased Execution Ledger

**Scope:** pegasusX only · **Reference:** pegasus `warehouse-portal` (read-only)  
**Parent plan:** `VEGETABLE_PLAN.md` §2.2 · `Phase Next: Replenishment + Supply`  
**Last updated:** 2026-06-15 (WH-11P portal deep component parity).

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

**Last updated:** 2026-06-15

---

## Phase WH-7 — Production blockers (auth refresh, transfer/order mutations)

| ID | Feature | Backend | Portal | Android | iOS | Status |
|----|---------|---------|--------|---------|-----|--------|
| WH7-01 | Token refresh path | `POST /v1/auth/warehouse/refresh` | `lib/auth.ts` | `NetworkModule` | `APIClient` | **WIRED** (pre-existing; verified) |
| WH7-02 | Order mutations (delay/reject/overflow) | `POST /v1/warehouse/ops/orders/{id}/*` | `/orders/[id]` | `OrderDetailScreen` | `OrderDetailView` | **WIRED** (pre-existing) |
| WH7-03 | Transfer actions | `POST /v1/warehouse/transfers/*` | `/transfers` | `TransferActionsScreen` | `TransferActionsView` | **WIRED** (pre-existing) |
| WH7-04 | Auth refresh on 401 retry | — | `apiFetch` | OkHttp interceptor | `APIClient` | **WIRED** (pre-existing) |

**Exit:** No P0 path bugs found in audit; warehouse row auth + mutations already green via existing SSMR markers.

---

## Phase WH-8 — Parity wiring (notifications inbox, financials, ops depth)

| ID | Feature | Backend | Portal | Android | iOS | Status |
|----|---------|---------|--------|---------|-----|--------|
| WH8-01 | Notification inbox (live) | `GET /v1/user/notifications` | `useNotifications` + panel | `NotificationInboxScreen` | `NotificationInboxView` | **WIRED** |
| WH8-02 | Mark notifications read | `POST /v1/user/notifications/read` | panel actions | inbox mark-all | inbox mark-all | **WIRED** |
| WH8-03 | Ops financials in treasury | `GET /v1/warehouse/ops/financials` | `/treasury` fallback | `TreasuryScreen` | `TreasuryView` | **WIRED** (pre-existing) |
| WH8-04 | Dispatch settings + payment config native | ops routes | portal pages | native screens | native screens | **WIRED** (pre-existing) |
| WH8-05 | Profile / setup portal handoff | setup + JWT profile | `/profile` | portal handoff | portal handoff | **WIRED** (intentional v1) |

**Exit:** Native Android/iOS no longer hand off notifications to portal; unified inbox API wired with graceful 503 copy.

---

## Phase WH-9 — Client policy & platform gating

| ID | Feature | Backend | Portal | Android | iOS | Status |
|----|---------|---------|--------|---------|-----|--------|
| WH9-01 | Client version policy | `GET /v1/platform/client-policy?role=WAREHOUSE` | `ClientPolicyBanner` | dashboard banner | dashboard banner | **WIRED** |
| WH9-02 | SSMR marker | smokecheck | — | — | — | **WIRED** (`PX_E2E_WAREHOUSE_CLIENT_POLICY_OK`) |
| WH9-03 | Firebase OTP | portal + Android + iOS OTP | login OTP | login OTP | login OTP | **WIRED** |

**Exit:** Outdated/force-update surfaces show honest banners on all warehouse clients; SSMR asserts WAREHOUSE role policy tuple.

---

## Phase WH-10 — Dashboard UX parity + live connection (P2)

| ID | Feature | Backend | Portal | Android | iOS | Status |
|----|---------|---------|--------|---------|-----|--------|
| WH10-01 | Fleet status breakdown renders | `GET /v1/warehouse/ops/dashboard` (`fleet_status[]`) | dashboard section | `FleetStatusBreakdown` | `FleetStatusBreakdown` | **WIRED** |
| WH10-02 | KPI ALERT/DONE chips | — | pre-existing | dashboard chips | dashboard chips | **WIRED** |
| WH10-03 | Dashboard load error taxonomy | — | offline/restricted/error | pre-existing retry | pre-existing retry | **WIRED** |
| WH10-04 | WS-aware shell live indicator | warehouse WS hub | `useNotifications.wsState` + topbar | dispatch realtime banner (pre-existing) | dispatch realtime banner (pre-existing) | **WIRED** |

**Exit:** Dashboard fleet chips visible on all clients; portal topbar reflects warehouse WS connection state; no new SSMR markers (UI-only parity).

---

## Phase WH-11 — Deep native UI/UX parity (iOS)

| ID | Feature | Android | iOS | Status |
|----|---------|---------|-----|--------|
| WH11-01 | Shared KPI / list / status primitives | — (iOS-only batch) | `KpiTile`, `WarehouseStatusBadge`, `WarehouseSectionHeader` | **WIRED** |
| WH11-02 | Theme tokens (`statusTint`, `readableMaxWidth`) | — | `LabTheme` + `labReadableWidth()` | **WIRED** |
| WH11-03 | Dashboard KPI grid + fleet status | pre-existing WH-10 | `DashboardView` — adaptive `KpiTile`, section header, state views | **WIRED** |
| WH11-04 | Dispatch status badges + loading copy | — | `DispatchView` — `WarehouseStatusBadge`, `WarehouseLoadingView` | **WIRED** |
| WH11-05 | Replenishment urgency badges | — | `ReplenishmentView` — semantic urgency/status chips | **WIRED** |
| WH11-06 | Supply requests list polish | — | `SupplyRequestsHubView` — badges, retry, filter reload | **WIRED** |
| WH11-07 | Treasury KPI grid + invoice badges | — | `TreasuryView` — `KpiTile`, `WarehouseStatusBadge` | **WIRED** |
| WH11-08 | Fleet live map + transfer actions chrome | — | `FleetLiveMapView`, `TransferActionsView` section headers | **WIRED** |
| WH11-09 | Loading/error/empty states | — | `WarehouseLoadingView` / `WarehouseErrorView` / `WarehouseEmptyView` | **WIRED** |

**UI audit vs pegasus reference:** pegasus `warehouse-app-ios` is thinner (no replenishment/supply/fleet map); pegasusX iOS is ahead on ops depth. WH-11 aligns pegasusX iOS with supplier SP-7 discipline (shared primitives, semantic tints, refresh toolbars, empathetic loading copy).

**Exit:** Primary ops screens use shared warehouse components; no new SSMR markers (UI-only parity).

---

## Phase WH-11A — Deep native UI/UX parity (Android)

| ID | Feature | Android | iOS | Status |
|----|---------|---------|-----|--------|
| WH11A-01 | Shared UI kit (`WarehouseUiComponents`, `WarehouseState`) | KPI tiles, metric tiles, status chips, list cards, section titles, loading/error/empty panes | pre-existing WH-11 | **WIRED** |
| WH11A-02 | Dashboard KPI grid + fleet chips | `DashboardScreen` — `WarehouseKpiTile`, `WarehouseStatusChip` | pre-existing WH-11 | **WIRED** |
| WH11A-03 | Dispatch ops depth | `DispatchScreen` — state panes, section titles, `WarehouseOpsListCard` / status chips on tabs | — | **WIRED** |
| WH11A-04 | Replenishment + supply lists | `ReplenishmentScreen`, `SupplyRequestsScreen` | pre-existing WH-11 | **WIRED** |
| WH11A-05 | Transfers form chrome | `TransferActionsScreen` — `WarehouseSectionTitle` | — | **WIRED** |
| WH11A-06 | Fleet roster (drivers + vehicles) | `DriversScreen`, `VehiclesScreen` | — | **WIRED** |
| WH11A-07 | Treasury / financials KPI grid | `TreasuryScreen` — `WarehouseMetricTile`, `WarehouseOpsListCard` | pre-existing WH-11 | **WIRED** |

**UI audit vs pegasus reference:** pegasus `warehouse-app-android` is thinner (no replenishment/supply/fleet map, plain KPI cards, raw `CircularProgressIndicator` loading). pegasusX Android is ahead on ops depth. WH-11A aligns pegasusX Android with supplier SP-7 discipline (shared primitives, 160dp adaptive grids, `IconButton` refresh, empathetic loading copy).

**Exit:** Primary warehouse Android ops screens share M3 discipline with supplier SP-7 patterns. UI-only — no new SSMR markers.

---

## Phase WH-11P — Deep warehouse-portal UI/UX (portal)

| ID | Feature | pegasus ref | pegasusX portal | Status |
|----|---------|-------------|-----------------|--------|
| WH11P-01 | `PageChrome` skeleton + `EmptyState` | `Skeleton.tsx`, `EmptyState.tsx` | `PageChrome` variants (`dashboard`/`table`/`form`) | **WIRED** |
| WH11P-02 | KPI tile structure (`KpiStatCard`, `PageSection`) | desk KPI grid | `KpiStatCard.tsx`, `PageSection.tsx` | **WIRED** |
| WH11P-03 | Dashboard fleet map + status section chrome | dashboard sections | `PageSection` on map + fleet chips | **WIRED** |
| WH11P-04 | Dispatch control room KPI + sections | dispatch preview layout | KPI grid + `PageSection` orders/drivers/map | **WIRED** |
| WH11P-05 | Replenishment insight cards + empty state | — (pegasusX-only page) | KPI summary + `EmptyState` + table section | **WIRED** |
| WH11P-06 | Treasury KPI grid + invoice section | treasury overview | `KpiStatGrid` + `PageSection` invoices | **WIRED** |
| WH11P-07 | Supply requests skeleton via PageChrome | supply-requests table | `skeletonVariant="table"` | **WIRED** |
| WH11P-08 | Transfers form section chrome | — (pegasusX-only) | `PageSection` + `skeletonVariant="form"` | **WIRED** |

**WH-11P audit gaps (intentional / blocked):** Dashboard linked KPI hover cards (pegasusX ahead of pegasus ref); BentoGrid layout (supplier pattern, not pegasus warehouse); drivers/vehicles CRUD depth; manifests pick-list tabs; dispatch auto-execute (warehouse manual-only by design); analytics chart depth (WH-4 closed on native); Firebase OTP login (WH9 deferred).

**Exit:** Component-level desk tokens, skeleton loaders, KPI structure, and section headers on dashboard, dispatch, replenishment, treasury, supply, transfers. UI-only — no new SSMR.

---

## Next execution batch

1. ~~Replenishment insights durability~~ — WH-1
2. ~~Replenishment engine~~ — WH-2
3. ~~Native fleet live map~~ — WH-3
4. ~~WH-4 analytics parity remediation~~ — WH-4
5. ~~WH-5 native daily revenue chart~~ — WH-5
6. ~~WH-6 CSV import staging~~ — WH-6
7. ~~WH-7/8/9 cross-client parity batch~~ — **CLOSED** (2026-06-15)
8. ~~WH-10 dashboard UX + live connection parity~~ — **CLOSED** (2026-06-15)
9. ~~WH-11 deep native UI/UX parity (iOS component-level)~~ — **CLOSED** (2026-06-15)
10. ~~WH-11P portal deep UI/UX (component-level)~~ — **CLOSED** (2026-06-15)
11. ~~WH-11A Android deep UI/UX parity~~ — **CLOSED** (2026-06-15)
12. ~~WH-12 native ops settings + inventory policy + supply create depth~~ — **CLOSED** (2026-06-15)
13. **Cross-role next** — Boss-picked role row (FACTORY / DRIVER / PAYLOAD per `VEGETABLE_PLAN.md` §3)

---

## Phase WH-13 — Warehouse ops policy E2E (lead days, line limits, delivery fees)

**Status:** **CLOSED** (2026-06-17) — Spanner policy columns, `GET/PATCH /v1/warehouse/ops/settings`, checkout preview/create enforcement, retailer clamp + fee totals, driver `delivery_fee_minor` on fleet orders, supplier `sale_unit`/`units_per_pack`, SSMR markers `PX_E2E_WAREHOUSE_OPS_POLICY_OK`, `PX_E2E_CHECKOUT_LINE_LIMIT_OK`, `PX_E2E_DELIVERY_FEE_PREVIEW_OK`.

| ID | Surface | Notes |
|----|---------|-------|
| WH13-01 | Backend | `WarehouseOpsPolicy` loaders, line min/max, haversine fee tiers, pack multiples Phase B |
| WH13-02 | Retailer row | Desktop/Android/iOS preview fields + delivery fee in checkout total |
| WH13-03 | Driver row | Fleet order DTO + fee badge (cash collection awareness) |
| WH13-04 | Supplier catalog | `sale_unit`, `units_per_pack` persisted + portal already sends `units_per_case` |
| WH13-05 | Warehouse clients | Ops settings models synced; pre-order edit/reject on Android preorders hub |

---

## Phase WH-14 — Pre-order hub + delivery-date negotiation

**Status:** **CLOSED** (2026-06-17) — `ProposedDeliveryDate` columns, `PENDING_WAREHOUSE` confirmation status, warehouse `propose-delivery` + retailer accept/reject-proposal, active orders list excludes `SCHEDULED` manual pre-orders, `PRE_ORDER_DATE_*` notifications with `/orders/{id}?review=1`, role-row calendar propose + review sheet, SSMR `PX_E2E_DELIVERY_PROPOSAL_OK`, `PX_E2E_DELIVERY_PROPOSAL_REJECT_CANCEL_OK`.

| ID | Surface | Notes |
|----|---------|-------|
| WH14-01 | Backend | `delivery_proposal.go`, warehouse/retailer routes, T-2 lock, inventory release on reject |
| WH14-02 | Warehouse row | Android DatePicker propose, iOS sheet, portal actions column, WS refresh |
| WH14-03 | Retailer row | Review Delivery badge, accept/reject proposal, push deep link `?review=1` |
| WH14-04 | Contracts | `packages/types`, `api-client`, `events.schema.json`, i18n notification copy |

---

## Phase WH-15 — Concurrent stock checkout (REJECT + reservations)

**Status:** **CLOSED** (2026-06-17) — `QuantityReserved` incremented at create for **all** non-backorder orders (including `SCHEDULED` manual pre-orders); idempotent `OrderStockReservationMarkers` backfill on bootstrap; checkout preview exposes `orderable_quantities` = `min(available, line_max)`; SSMR `PX_E2E_CONCURRENT_STOCK_REJECT_OK` (parallel creates, one winner).

| ID | Surface | Notes |
|----|---------|-------|
| WH15-01 | Backend | `ReserveLineItemsInTxn`, removed `StatusScheduled` skip, `BackfillScheduledReservations` |
| WH15-02 | Schema | `OrderStockReservationMarkers` DDL + migration `20250625_order_stock_reservation_markers.ddl` |
| WH15-03 | Preview | `orderable_quantities`, line-min block when stock &lt; min |
| WH15-04 | SSMR | Parallel retailer creates exceed on-hand → exactly one 201 + one `inventory_exhausted` |

---

## Phase WH-12 — Native ops depth + ecosystem parity (P1/P2)

**Status:** **CLOSED** (2026-06-15) — ops settings, per-SKU inventory policy, enriched supply-request create on Android + iOS; nav README + notifications label fix; WAREHOUSE row **Wired** in parity matrix.

| ID | Feature | Portal | Android | iOS |
|----|---------|--------|---------|-----|
| WH12-01 | Ops settings | ref `/settings` | `OpsSettingsScreen` | `OpsSettingsView` |
| WH12-02 | Inventory policy | ref `/inventory` | `InventoryScreen` policy picker | `InventoryView` policy picker |
| WH12-03 | Supply create depth | ref `/supply-requests/new` | `CreateSupplyRequestDialog` (Dispatch + Supply) | `CreateSupplyRequestSheet` |
| WH12-04 | Nav README + notification label fix | — | README + `NOTIFICATIONS` section | README + `notifications` section |
| WH12-05 | Matrix Wired | — | — | `ROLE_ROW_PARITY_MATRIX` WAREHOUSE → **Wired** |

**Build proof:**

```bash
cd pegasusX/apps/backend-go && go test ./warehouse/... ./replenishment/...
cd pegasusX/apps/warehouse-app-android && ./gradlew :app:compileDebugKotlin
cd pegasusX/apps/warehouse-app-ios && xcodegen generate && xcodebuild -scheme WarehouseAppIOS build
```

---

## Phase WH-6 — Import staging write path (P1)

**Status:** **CLOSED** — direct CSV import persists `SupplierImportSessions` + `SupplierImportStagedRows`; warehouse `import_anomaly_queue` populates from `validation_errors` (warehouse-scoped).

| ID | Item | Backend | SSMR | Status |
|----|------|---------|------|--------|
| WH6-01 | Staging DDL | `20250615_supplier_import_staging.ddl` | — | **CLOSED** |
| WH6-02 | Anomaly read path | `warehouse/analytics_spanner.go` | decode shape | **CLOSED** |
| WH6-03 | Import staging write | `supplier/inventory_import_staging.go` | `session_id` + anomaly assert | **CLOSED** |
| WH6-04 | Product validation on import | `validateImportedProduct` | bad SKU row skipped + staged | **CLOSED** |

---

## Phase WH-5 — Analytics native UI depth (P2)

**Status:** **CLOSED** — daily chart, 30d default, import freshness/anomaly cards wired (staging DDL `20250615_supplier_import_staging.ddl`).

| ID | Item | Portal | Android | iOS | Status |
|----|------|--------|---------|-----|--------|
| WH5-01 | Daily revenue chart | Recharts bar | `DailyRevenueChart` | `DailyRevenueChart` | **CLOSED** |
| WH5-02 | Default period 30d | yes | yes | yes | **CLOSED** |
| WH5-03 | iOS `daily_breakdown` decode | — | — | `DailyMetric` + `chartDaily` | **CLOSED** |
| WH5-04 | Full import freshness / anomaly cards | full | full KPI + meta | full KPI + meta | **CLOSED** — staging DDL + Spanner anomaly scan |

---

## Phase WH-4 — Analytics native parity remediation

**Status:** **REMEDIATION_COMPLETE** (P0 + P1 + P2) — see `context/WAREHOUSE_ANALYTICS_PARITY.md`.

| ID | Finding | Portal | Android | iOS | Status |
|----|---------|--------|---------|-----|--------|
| WH4-AUDIT-01 | Analytics period filter | live | live | live | **CLOSED** — `analytics_spanner.go` |
| WH4-AUDIT-02 | Analytics field population | partial | partial | partial | **CLOSED** — top/daily/fleet/import anomaly via staging write path |
| WH4-AUDIT-03 | Replenishment OPEN-only gate | yes | yes | yes | **CLOSED** |
| WH4-AUDIT-04 | Demand forecast insights fallback | yes | yes | yes | **CLOSED** |
| WH4-AUDIT-05 | Demand forecast Series tab | no | yes | yes | native-ahead (keep) |
| WH4-AUDIT-06 | SSMR `PX_E2E_WAREHOUSE_ANALYTICS_OK` | — | — | — | **CLOSED** |

---

## Enterprise production readiness (2026-06-18)

**Row status:** `PROD_CANDIDATE` — AUTO dispatch + freeze locks, supply request idempotency, fleet live map WS, preorders/stock commitments. OSRM geometry via `ROUTING_OSRM_URL` in k8s configmap.
