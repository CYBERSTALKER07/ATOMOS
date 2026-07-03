# pegasusX SUPPLIER Role — Phased Execution Ledger

**Scope:** pegasusX only · **Reference:** pegasus `admin-portal` (read-only)  
**Parent plan:** `VEGETABLE_PLAN.md` §2.1  
**Last updated:** 2026-06-29 (phase 2 idempotency gap closure + kafka consumer-group dedup fix; **PROD_CANDIDATE**)

## Status model

`TODO` → `IN_PROGRESS` → `WIRED` → `E2E_SSMR_GREEN` → `PROD_CANDIDATE`

---

## Phase 0 — Onboarding gate integrity (P0)

| ID | Feature | Backend | supplier-portal | Android | iOS | Status | Proof |
|----|---------|---------|---------------|---------|-----|--------|-------|
| SP0-01 | JWT `is_registered` claim | `auth/claims.go`, `auth/jwt.go` | `middleware.ts` reads claim | cookie via login | cookie via login | **E2E_SSMR_GREEN** | unit tests |
| SP0-02 | Business setup contract | `supplier/setup.go` `CompleteBusinessSetup` | `/setup/business` | `setupBusiness` API | `setupBusiness` API | **E2E_SSMR_GREEN** | `setup_test.go` |
| SP0-03 | Register → business redirect | `Register` sets `is_registered=false` for minimal wizard | `/auth/register` → `/setup/business` | register flow | register flow | **WIRED** | manual + tests |
| SP0-04 | Login next-step routing | `Login` returns `is_registered` + correct `next_step` | login page | login | login | **WIRED** | `setup_test.go` |
| SP0-05 | Shared types | — | — | — | — | **WIRED** | `packages/types`, `api-client` |

**Exit:** New supplier can register → complete business → billing gate → portal without 400 on business setup.

---

## Phase 1 — Network topology CRUD (P1)

| ID | Feature | Backend | Portal | Native | Status |
|----|---------|---------|--------|--------|--------|
| SP1-01 | Topology editor (`PUT /v1/supplier/topology`) | exists + JSON wire tags | `/topology` editor wired | Android + iOS edit/save | **WIRED** |
| SP1-02 | Warehouse/factory create UI | `PUT topology` | add/remove nodes in topology editor | native edit forms | **WIRED** |
| SP1-03 | Delivery zones + supply lanes | **CLOSED** — read-only `GET /v1/supplier/supply-lanes` derived from topology (warehouse nodes, coverage, co-location); edit via Topology, not lane CRUD |
| SP1-04 | `getSupplierInventory()` in api-client | GET exists | raw fetch today (optional migrate) | — | **WIRED** |

**Dependency:** Phase 0 complete. **Blocks:** warehouse/factory ops for new tenants.

**Exit:** Supplier can add/edit warehouse + factory nodes on portal and native; `GET /v1/supplier/topology` returns snake_case JSON aligned with shared types.

---

## Phase 2 — Dispatch & manifest oversight (P1)

| ID | Feature | Backend | Portal | Native | Status |
|----|---------|---------|--------|--------|--------|
| SP2-01 | Dispatch preview + AUTO execute | `supplierroutes` | `/dispatch` partial | preview screens | **WIRED** |
| SP2-02 | Manifest lifecycle actions | `payloaderroutes` supplier manifests | `/manifests/[id]` start-loading / inject / seal | Android + iOS detail actions | **WIRED** |
| SP2-03 | Manifest exceptions inbox | `GET /v1/supplier/manifest-exceptions` | `/manifest-exceptions` | Android + iOS gate screens | **WIRED** |
| SP2-04 | Fleet live map oversight | `GET /v1/supplier/fleet/live-map` | dashboard + `/fleet` | MapLibre/MapKit | **E2E_SSMR_GREEN** |

**Cross-sync:** warehouse dispatch execute, payload seal, driver assign detection.

**Exit:** Supplier can inspect manifest detail, run loading lifecycle actions, and triage manifest gate exceptions on portal and native.

---

## Phase 3 — Staff, org, treasury depth (P1)

| ID | Feature | Backend | Portal | Native | Status |
|----|---------|---------|--------|--------|--------|
| SP3-01 | Org member lifecycle (edit/deactivate) | PATCH/DELETE `/v1/supplier/org/members/{id}` | `/org-fleet` edit + deactivate | Android + iOS deactivate | **WIRED** |
| SP3-02 | Returns resolution | orders `filter=RETURNS` | `/returns` order handoff + context | — | **WIRED** |
| SP3-03 | Payment config / gateway vault | billing setup | `/setup/billing` | billing screen | **WIRED** |
| SP3-04 | Notification inbox recipient fix | `notifications.RecipientIDFromClaims` | `useNotifications.ts` (unchanged path) | — | **WIRED** |

---

## Phase 4 — Intelligence & catalog depth (P2 for single-tenant)

| ID | Feature | Backend | Portal | Native | Status |
|----|---------|---------|--------|--------|--------|
| SP4-01 | Analytics / demand forecast | `/v1/supplier/analytics/*` | `/analytics` + `/analytics/demand` | Android + iOS analytics KPIs | **E2E_SSMR_GREEN** |
| SP4-02 | Retailer pricing overrides | `GET/POST/DELETE /v1/supplier/pricing/retailer-overrides` + promotion quote resolution | portal handoff | Android + iOS API-ready | **E2E_SSMR_GREEN** (`PX_E2E_RETAILER_PRICING_OVERRIDE_OK`) |
| SP4-03 | Inventory CSV import + staging | `POST /v1/supplier/inventory/import` + session wizard + async worker | `/inventory/import` wizard | — | **E2E_SSMR_GREEN** — direct CSV + sync wizard (`PX_E2E_SUPPLIER_IMPORT_WIZARD_OK`) + async upload (`PX_E2E_SUPPLIER_IMPORT_ASYNC_OK`) |
| SP4-04 | Products vs catalog canonical | `/v1/catalog/products` | `/catalog` | — | **WIRED** |

---

## Phase 5 — Cross-cutting depth (analytics / treasury / operations)

| ID | Feature | Backend | Portal | Native | Status |
|----|---------|---------|--------|--------|--------|
| SP5-01 | Native analytics depth | `/v1/supplier/analytics/*` | `/analytics` (reference) | `AnalyticsScreen` / `AnalyticsView` | **WIRED** |
| SP5-02 | Treasury hub depth | earnings + settlement authority | `/treasury` live KPI snapshot | Android/iOS treasury screens (existing) | **WIRED** |
| SP5-03 | Operations / exceptions | empathy, broadcast, replenishment, payment-bypass | `/operations` polished | `OperationsScreen` / `OperationsView` | **E2E_SSMR_GREEN** (`PX_E2E_SUPPLIER_OPERATIONS_OK`, `PX_E2E_SUPPLIER_PAYMENT_BYPASS_OK`) |

---

## Phase 6 — UI/UX parity & cross-cutting client surfaces (SP-6)

| ID | Feature | Backend | Portal | Android | iOS | Status |
|----|---------|---------|--------|---------|-----|--------|
| SP6-01 | Client policy banner (`role=ADMIN`) | `GET /v1/platform/client-policy` | `ClientPolicyBanner` in `SupplierShell` | `ClientPolicyBanner` on dashboard | `ClientPolicyBanner` on dashboard | **E2E_SSMR_GREEN** (`PX_E2E_SUPPLIER_CLIENT_POLICY_OK`) |
| SP6-02 | Native notification inbox | `GET/POST /v1/user/notifications*` | `NotificationPanel` + WS (existing) | `NotificationInboxScreen` | `NotificationInboxView` | **E2E_SSMR_GREEN** (`PX_E2E_SUPPLIER_NOTIFICATION_INBOX_OK`) |
| SP6-03 | Realtime WS refresh | `/v1/ws/supplier` | `useNotifications.ts` + `use-supplier-ws-refresh` | `SupplierWebSocket` + `SupplierRealtimeSignals` | `SupplierRealtimeClient` + hub | **WIRED** (pre-existing) |

---

## Phase 7 portal — Deep supplier-portal UI/UX (SP-7P)

| ID | Feature | pegasus ref | pegasusX portal | Status |
|----|---------|-------------|-----------------|--------|
| SP7P-01 | `PageChrome` skeleton + `EmptyState` | `Skeleton.tsx`, `EmptyState.tsx` | `components/Skeleton.tsx`, `EmptyState.tsx`, `PageChrome` variants | **WIRED** |
| SP7P-02 | KPI tile structure (`KpiStatCard`, `PageSection`) | `md-card-elevated` KPI grid | `KpiStatCard.tsx`, `PageSection.tsx` | **WIRED** |
| SP7P-03 | Analytics depth (AI demand card, charts, header actions) | `supplier/analytics/page.tsx` | `/analytics` + `/analytics/demand` desk tokens | **WIRED** |
| SP7P-04 | Operations form spacing + section chrome | — (pegasusX-only hub) | `/operations` `PageSection` + labeled fields | **WIRED** |
| SP7P-05 | Treasury hub KPI + link cards | payment-config / earnings refs | `/treasury` `KpiStatGrid` + icon link cards | **WIRED** |
| SP7P-06 | Dispatch toolbar (search wired, dead CTA removed) | `supplier/dispatch/page.tsx` | client-side manifest search; link to `/manifests` | **WIRED** |
| SP7P-07 | Dashboard billing gate link fix | `payment-config` | `/setup/billing` (was broken `/settings/billing`) | **WIRED** |
| SP7P-08 | Fleet map chrome | fleet page header | `/fleet` `bento-card-header` parity | **WIRED** |

**SP-7P audit gaps (intentional / blocked):** Fleet CRUD depth (pegasus `supplier/fleet` ~1.5k LOC → pegasusX `/org-fleet`); manifests pick-list tabs + date CSV; dispatch manual control room; analytics `?warehouse_id=` revenue filter (backend gap); per-SKU velocity table (API contract differs); shift toggle on layout.

**Exit:** Component-level desk tokens, skeleton loaders, KPI structure, and section headers on analytics, operations, treasury, dispatch, fleet, manifests. UI-only — no new SSMR.

---

## Phase 7 — Deep native UI/UX parity (SP-7)

| ID | Feature | Android | iOS | Status |
|----|---------|---------|-----|--------|
| SP7-01 | Shared KPI / list / status primitives | `SupplierUiComponents.kt` (`SupplierKpiTile`, `SupplierOpsListCard`, `SupplierStatusChip`, `SupplierMetricTile`, `SupplierLeadingIcon`) | `KpiTile`, `SupplierStatusBadge`, `SupplierSectionHeader` | **WIRED** |
| SP7-02 | Dashboard KPI grid + billing card | `DashboardScreen` — 160dp adaptive grid, icon badges, refresh | `DashboardView` + `KpiTile`, `.refreshable` | **WIRED** |
| SP7-03 | Orders / Fleet list cards | `OrdersScreen`, `FleetScreen` — row cards, status chips, refresh, formatted amounts | refresh toolbar + grouped cards | **WIRED** |
| SP7-04 | More hub ListItem discipline | `MoreScreen` — `SupplierLeadingIcon`, titleSmall/bodySmall | `MoreHubView` section headers | **WIRED** |
| SP7-05 | Operations empathy metrics | `OperationsScreen` — `SupplierMetricTile` grid | `OperationsView` empathy row | **WIRED** |
| SP7-06 | Analytics KPI grid | `AnalyticsScreen` — adaptive grid + parallel fetch | `AnalyticsView` + refresh | **WIRED** |
| SP7-07 | Manifest list + detail KPIs | `ManifestsScreen`, `ManifestDetailScreen` — status chips, volume/orders KPI row | manifest views + refresh | **WIRED** |
| SP7-08 | Loading/error/empty states | `SupplierStatePane` / `SupplierLoadingState` | `SupplierLoadingView` / `SupplierErrorView` | **WIRED** |

**UI audit vs pegasus reference:** No native supplier app in `pegasus/apps/`; Android patterns mirrored from `factory-app-android` (`KpiMetricCard`, icon badges) and `warehouse-app-android` (`AssistChip` status, 160dp grid, refresh top-bar). iOS aligned with factory/warehouse SwiftUI (semantic tints, section headers, adaptive KPI grids).

---

## Phase 8 — Ecosystem parity (portal + Android + iOS) — **WIRED** (2026-06-15)

| ID | Feature | Portal | Android | iOS | Status |
|----|---------|--------|---------|-----|--------|
| SP8-01 | Unified nav map (`SupplierSection` / More hub) | `SupplierShell` retailer-overrides link | `SupplierRoutes` + `MoreScreen` | `SupplierSection` + `MoreHubView` + iPad sidebar | **WIRED** |
| SP8-02 | Native onboarding (register → business → billing) | existing | `RegisterScreen`, `BusinessSetupScreen` | `RegisterView`, `BusinessSetupView` | **WIRED** |
| SP8-03 | Order vetting | `/orders` | `OrdersScreen` + `OrdersViewModel` | `OrdersView` + `OrdersViewModel` | **WIRED** |
| SP8-04 | Inventory adjust + CSV import | `/inventory`, `/inventory/import` | `InventoryScreen`, `InventoryImportScreen` | `InventoryView`, `InventoryImportView` | **WIRED** |
| SP8-05 | Retailer price overrides | `/pricing/retailer-overrides` | `RetailerOverridesScreen` | `RetailerOverridesView` | **WIRED** |
| SP8-06 | Chargebacks native | `/earnings` + `/payments` | `ChargebacksScreen` | `ChargebacksView` | **WIRED** |
| SP8-07 | Treasury hub + demand history | `/treasury`, `/analytics/demand` | `TreasuryHubScreen`, `DemandHistoryScreen` | `TreasuryHubView`, `DemandHistoryView` | **WIRED** |
| SP8-08 | Factories / warehouses browse | `/factories`, `/warehouses` | `FactoriesScreen`, `WarehousesScreen` | `FactoriesView`, `WarehousesView` | **WIRED** |
| SP8-09 | Catalog product detail | `/catalog/[productId]` | `CatalogDetailScreen` | `CatalogDetailView` | **WIRED** |
| SP8-10 | Profile edit | `/profile` PUT | `ProfileScreen` (read) | `ProfileView` (read) | **WIRED** portal; native read-only |
| SP8-11 | ViewModels (onboarding, orders, inventory, treasury) | N/A | Hilt ViewModels | `@Observable` ViewModels | **WIRED** |

**Edge case E1 (dispatch execute):** Supplier `POST /v1/supplier/dispatch/execute` retained as CEO override on portal + native; warehouse row owns fleet CRUD per topology dependency diagram.

## Intentional single-tenant deltas (do not close)

- Platform control center, DLQ, KYC, country config (pegasus multi-tenant admin)
- pegasus ~59-route multi-tenant admin extras (CRM, staff, country-overrides) — out of scope for single-tenant pegasusX
- Quantity negotiation disabled ecosystem-wide

---

## Verification (supplier row)

```bash
cd pegasusX/apps/backend-go && go test ./supplier/...
cd pegasusX && make test-ssmr-infra   # PX_E2E_ORDER_OK umbrella
cd pegasusX && make parity-contract-full
```

---

## Next execution batch (Boss-approved start: supplier)

1. ~~Phase 0 onboarding~~ — **done**
2. ~~Phase 1 topology editor~~ — **done**
3. ~~Phase 2 manifest actions + manifest-exceptions~~ — **done**
4. ~~Phase 3 staff/org lifecycle + notification inbox fix~~ — **done this session**
5. ~~Phase 4 intelligence & catalog depth (analytics, CSV import, retailer pricing overrides, import session wizard + async worker)~~ — **E2E_SSMR_GREEN** (`PX_E2E_SUPPLIER_ANALYTICS_OK`, `PX_E2E_SUPPLIER_INVENTORY_IMPORT_OK`, `PX_E2E_SUPPLIER_IMPORT_WIZARD_OK`, `PX_E2E_SUPPLIER_IMPORT_ASYNC_OK`, `PX_E2E_RETAILER_PRICING_OVERRIDE_OK`)
6. ~~Phase 5 cross-cutting depth (native analytics, treasury hub KPIs, operations broadcast/bypass)~~ — **E2E_SSMR_GREEN** (`PX_E2E_SUPPLIER_OPERATIONS_OK`, `PX_E2E_SUPPLIER_PAYMENT_BYPASS_OK`)
7. ~~Phase 6 UI/UX parity (client-policy + native notification inbox)~~ — **E2E_SSMR_GREEN** (`PX_E2E_SUPPLIER_CLIENT_POLICY_OK`, `PX_E2E_SUPPLIER_NOTIFICATION_INBOX_OK`)
8. ~~Phase 7 portal deep UI/UX (SP-7P component-level)~~ — **WIRED** (SP7P-01–SP7P-08; UI-only)
9. ~~Phase 7 deep native UI/UX (Android + iOS component-level parity)~~ — **WIRED** (SP7-01–SP7-08)
10. ~~Phase 8 ecosystem parity (native onboarding, vet, inventory, overrides, chargebacks, treasury hub)~~ — **WIRED**
11. **Cross-role next** — Boss-picked role row per `VEGETABLE_PLAN.md` §3

---

## Cross-role — Notification inbox persistence (WIRED)

| ID | Feature | Backend | Portal / clients | Status |
|----|---------|---------|------------------|--------|
| NI-01 | Dispatcher → Spanner inbox | `kafka/notification_inbox.go` `persistInbox`; `ShouldPersistInboxEvent` skips telemetry | — | **WIRED** |
| NI-02 | Supplier inbox routes | `supplierroutes` GET/POST `/v1/user/notifications` | `useNotifications.ts` | **WIRED** |
| NI-03 | Retailer / driver / payload read paths | existing `retailerroutes` / `driverroutes` / `payloaderroutes` | native inbox screens | **WIRED** (read; rows now populate) |
| NI-04 | SSMR | `runNotificationInboxE2E` polls supplier + retailer inbox after order flow | — | **E2E_SSMR_GREEN** (`PX_E2E_NOTIFICATION_INBOX_OK`) |
| NI-05 | ShopClosed + fleet inbox copy | `notifications/formatter.go` + `inbox_format.go` explicit cases | supplier portal inbox | native inbox read paths | **WIRED** | `inbox_format_test`; SSMR asserts `SHOP_CLOSED` row after report |
| NI-06 | Manifest lifecycle inbox copy | `formatter.go` + `inbox_format.go` for all 10 `MANIFEST_*` events | supplier portal inbox | native inbox read paths | **WIRED** | `inbox_format_test`; SSMR asserts `MANIFEST_SEALED` after factory seal |
| NI-07 | Retailer price override inbox | `handleRetailerPriceOverride` + `FormatRetailerPriceOverride` | retailer inbox | native inbox read paths | **E2E_SSMR_GREEN** | SSMR asserts `RETAILER_PRICE_OVERRIDE` after supplier override create |

**Note:** `notification_dispatcher.go` already fans out `SHOP_CLOSED*` and `DRIVER_CREATED` to WS + inbox — NI-05 adds operator-grade titles/deep links (not a new consumer). NI-06 covers remaining manifest lifecycle events (`MANIFEST_DRAFT_CREATED` through `MANIFEST_COMPLETED`) with `/manifests/{id}` or `/manifest-exceptions` deep links.

---

## Enterprise production readiness (2026-06-18)

**Row status:** `PROD_CANDIDATE` — idempotency on dispatch execute, WS stale-while-revalidate, treasury native parity, topology PUT create UX, Global Pay circuit breaker on backend checkout path. Boss must provision Global Pay.UZ credentials and GCS bucket per `docs/CLOUD_CREDENTIALS_CHECKLIST.md`.
