# pegasusX RETAILER Role — Phased Execution Ledger

**Scope:** pegasusX only · **Parent plan:** `VEGETABLE_PLAN.md` §2.5  
**Last updated:** 2026-06-15 (RT-6P retailer desktop deep UI parity).

## Status model

`TODO` → `IN_PROGRESS` → `WIRED` → `E2E_SSMR_GREEN` → `PROD_CANDIDATE`

---

## Phase RT-1 — Receiving window parity (cross-client)

| ID | Feature | Backend | Desktop | Android | iOS | Status |
|----|---------|---------|---------|---------|-----|--------|
| RT1-01 | Profile GET/PUT receiving windows | `PUT /v1/retailer/profile` + `proximity.ValidateReceivingWindow` | `/settings` edit form | `AccountProfileViewModel` | `AccountProfileView` | **E2E_SSMR_GREEN** (`PX_E2E_RETAILER_RECEIVING_WINDOW_OK`) |
| RT1-02 | Registration capture | `POST /v1/auth/retailer/register` | `/auth/register` wizard | `AuthScreen` | registration flow | **WIRED** |
| RT1-03 | Shared contracts | `RetailerProfileResponse` / `RetailerProfileUpdateRequest` | `lib/types.ts` + `lib/receiving-window.ts` | `Models.kt` | `AppModels.swift` | **WIRED** |

**Exit:** Retailer can set receiving windows at registration and edit later on every client surface; dispatch SLA reads snapshotted windows on new orders.

---

## Phase RT-2 — Production blockers (cancel, auth refresh, card checkout path)

| ID | Feature | Backend | Desktop | Android | iOS | Status |
|----|---------|---------|---------|---------|-----|--------|
| RT2-01 | Catalog products path | `GET /v1/catalog/products` | wired | wired | **WIRED** (`/v1/catalog/products`) | **WIRED** |
| RT2-02 | Order cancel / request-cancel | `POST /v1/order/cancel`, `POST /v1/orders/request-cancel` | `/orders` detail actions | `OrdersViewModel` + fallback | `DashboardView` / `ActiveDeliveriesView` | **WIRED** |
| RT2-03 | Token refresh path | `POST /v1/auth/retailer/refresh` | N/A (web session) | `NetworkModule` | `APIClient` | **WIRED** |
| RT2-04 | Card initiate in checkout | `POST /v1/retailer/card/initiate` + confirm | `CheckoutModal` | checkout flows | card screens | **WIRED** (desktop checkout; mobile card screens pre-existing) |
| RT2-05 | Notification badge (live) | `GET /v1/user/notifications` | `NotificationsProvider` | `NavigationViewModel` | `ContentView` | **WIRED** |
| RT2-06 | SSMR markers | smokecheck | — | — | — | **WIRED** (`PX_E2E_RETAILER_CATALOG_PRODUCTS_OK`, `PX_E2E_RETAILER_CANCEL_OK`, `PX_E2E_RETAILER_CARD_INITIATE_OK`) |

---

## Phase RT-3 — Parity wiring (insights, cards, catalog suppliers, pricing, tracking receipts)

| ID | Feature | Backend | Desktop | Android | iOS | Status |
|----|---------|---------|---------|---------|-----|--------|
| RT3-01 | Category suppliers in catalog | `GET /v1/catalog/categories/{id}/suppliers` | `/catalog` filter chips | `CategorySuppliersScreen` | `CategorySuppliersView` | **WIRED** |
| RT3-02 | Detailed analytics + prediction correct | `GET /v1/retailer/analytics/detailed`, `PATCH /v1/ai/predictions/correct` | `/insights` | `AnalyticsScreen` | `InsightsView` | **WIRED** (desktop detailed + dismiss; mobile pre-existing) |
| RT3-03 | Card deactivate / default | `POST /v1/retailer/card/deactivate`, `/default` | `/settings/cards` | `PegasusApi` | `APIClient` | **WIRED** (desktop UI; mobile API pre-existing) |
| RT3-04 | Settings sidebar / setup | `POST /v1/retailer/setup` | setup page | sidebar → profile + post-register setup | profile | **WIRED** |
| RT3-05 | Pricing rules read-only | `GET /v1/retailer/pricing/rules` | `/settings` | `ProfileScreen` | `ProfileView` | **WIRED** |
| RT3-06 | Tracking `recent_receipts` | `GET /v1/retailer/tracking` | `/tracking` | `TrackingResponse` + VM | `TrackingResponse` model | **WIRED** |
| RT3-07 | Orphan / legacy screens | — | — | — | `InboxView` deprecated; `ArrivalView` uses tracking | **WIRED** |

---

## Phase RT-4 — Client policy & OTP scaffold

| ID | Feature | Backend | Desktop | Android | iOS | Status |
|----|---------|---------|---------|---------|-----|--------|
| RT4-01 | Client version policy | `GET /v1/platform/client-policy` | `ClientPolicyBanner` | `NavigationViewModel` + profile | `ProfileView` | **WIRED** |
| RT4-02 | Firebase OTP | custom token exchange | — | scaffold + comment | — | **WIRED** (graceful degradation only) |

---

## Phase RT-5 — Cross-client policy banner + native notification inbox parity

| ID | Feature | Backend | Desktop | Android | iOS | Status |
|----|---------|---------|---------|---------|-----|--------|
| RT5-01 | Client policy banner (`role=RETAILER`) | `GET /v1/platform/client-policy` | `ClientPolicyBanner` in dashboard layout | `ClientPolicyBanner` on home + profile | `ClientPolicyBanner` on dashboard + profile | **E2E_SSMR_GREEN** (`PX_E2E_RETAILER_CLIENT_POLICY_OK`) |
| RT5-02 | Native notification inbox | `GET /v1/user/notifications`, `POST /v1/user/notifications/read` | `/notifications` page + `NotificationsProvider` | `NotificationInboxScreen` + `mark_all` | `NotificationInboxView` sheet + `mark_all` | **E2E_SSMR_GREEN** (`PX_E2E_RETAILER_NOTIFICATION_INBOX_OK`) |
| RT5-03 | SSMR markers | smokecheck | — | — | — | **WIRED** |

**Exit:** Retailer client-policy and notification-inbox flows match supplier/warehouse/factory/driver parity pattern on all three surfaces; SSMR emits dedicated retailer markers.

---

## Phase RT-6A — Deep native UI/UX parity (Android)

| ID | Feature | Android | Status |
|----|---------|---------|--------|
| RT6A-01 | Shared KPI / list / status primitives | `RetailerUiComponents.kt` (`RetailerKpiTile`, `RetailerMetricTile`, `RetailerListCard`, `RetailerStatusChip`, `RetailerSectionHeader`, `RetailerTagChip`) | **WIRED** |
| RT6A-02 | Loading/error/empty + runtime banners | `RetailerState.kt` (`RetailerLoadingState`, `RetailerStatePane`, `RetailerRuntimeBanner`) | **WIRED** |
| RT6A-03 | Dashboard overview KPI row + sections | `DashboardScreen` — `RetailerMetricTile` overview row, `RetailerSectionHeader`, `RetailerRuntimeBanner`, pull-to-refresh | **WIRED** |
| RT6A-04 | Catalog grid + empathy states | `CatalogScreen` — 160dp adaptive grid, `RetailerSectionHeader`, `RetailerLoadingState` / `RetailerStatePane` | **WIRED** |
| RT6A-05 | Orders sync banner | `OrdersScreen` — `RetailerRuntimeBanner` + existing tab pager / shimmer / `OrderStatusBadge` | **WIRED** |
| RT6A-06 | Tracking map refresh + list card | `DeliveryMapScreen` — top-bar `IconButton` refresh, `RetailerLoadingState`, `RetailerListCard` bottom sheet | **WIRED** |
| RT6A-07 | Profile KPI row + pricing section | `ProfileScreen` — `RetailerMetricTile` stats, `RetailerSectionHeader` pricing rules, `RetailerRuntimeBanner` | **WIRED** |
| RT6A-08 | Product card tag chips | `ProductCard` — `RetailerTagChip` for variant/size pills | **WIRED** |

**UI audit vs pegasus reference:** pegasus `retailer-app-android` matches pegasusX on dashboard service grid and orders tabs; pegasusX is ahead on catalog all-products browse, sale pricing on `ProductCard`, client-policy banner (RT-5), and notification inbox. RT-6A aligns pegasusX Android with supplier SP-7 / warehouse WH-11A discipline (shared primitives, 160dp adaptive grids, `IconButton` refresh on tracking, empathetic loading copy).

**Exit:** Primary retailer Android screens share M3 discipline with supplier/warehouse native patterns. UI-only — no new SSMR markers.

---

## Phase RT-6 — Deep native UI/UX parity (iOS)

| ID | Feature | Backend | Desktop | Android | iOS | Status |
|----|---------|---------|---------|---------|-----|--------|
| RT6-01 | Shared KPI / list / status primitives | — | — | — | `KpiTile`, `RetailerStatusBadge`, `RetailerSectionHeader` | **WIRED** |
| RT6-02 | Theme tokens (`statusTint`, `readableMaxWidth`) | — | — | — | `AppTheme.statusTint`, `retailerCard()`, `retailerReadableWidth()` | **WIRED** |
| RT6-03 | Dashboard KPI grid + loading copy | — | — | — | `DashboardView` — adaptive `KpiTile`, `RetailerLoadingView` / `RetailerErrorView` | **WIRED** |
| RT6-04 | Catalog section headers + error states | — | — | — | `CatalogView` — `RetailerSectionHeader`, `.refreshable`, `RetailerErrorView` | **WIRED** |
| RT6-05 | Orders status badges | — | — | — | `OrdersView` + `OrderCardView` — `RetailerStatusBadge` | **WIRED** |
| RT6-06 | Arrival / tracking polish | `GET /v1/retailer/tracking` | — | — | `ArrivalView` — LIVE/WAITING badges, approaching banner, `RetailerEmptyView` | **WIRED** |
| RT6-07 | Profile KPI grid | — | — | — | `ProfileView` — `KpiTile` stats row | **WIRED** |
| RT6-08 | Product card sale badge | — | — | — | `ProductCardView` — `RetailerStatusBadge` for sale offers | **WIRED** |
| RT6-09 | Global refresh toolbar | — | — | — | `ContentView` — `arrow.clockwise` triggers `RetailerRefreshCenter` | **WIRED** |
| RT6-10 | Loading/error/empty states | — | — | — | `RetailerLoadingView` / `RetailerErrorView` / `RetailerEmptyView` | **WIRED** |

**UI audit vs pegasus reference:** pegasusX iOS is ahead on catalog browse chips (Categories/All/Suppliers), tracking-based `ArrivalView`, sale pricing on `ProductCardView`, and client-policy banners (RT-5). pegasus reference has confirm/reject arrival actions (deprecated — pegasusX uses delivery handoff). RT-6 aligns pegasusX iOS with supplier SP-7 / warehouse WH-11 discipline (shared primitives, semantic tints, refresh toolbar, empathetic loading copy).

**Exit:** Primary retailer iOS screens share SwiftUI discipline with supplier/warehouse native patterns. UI-only — no new SSMR markers.

---

## Phase RT-6P — Deep retailer-app-desktop UI/UX (portal)

| ID | Feature | pegasus ref | pegasusX desktop | Status |
|----|---------|-------------|------------------|--------|
| RT6P-01 | `PageChrome` skeleton + desk `Skeleton` | inline HeroUI pulse | `components/Skeleton.tsx`, `PageChrome.tsx` | **WIRED** |
| RT6P-02 | KPI / section chrome (`KpiStatCard`, `PageSection`) | BentoGrid KPI cards | `KpiStatCard.tsx`, `PageSection.tsx` | **WIRED** |
| RT6P-03 | Dashboard section polish | Quick reorder + AI restock panels | `PageSection` + `PageSkeleton` loading | **WIRED** |
| RT6P-04 | Catalog product grid + category suppliers | product cards + filters | `PageSection` browse grid, skeleton chips/loaders | **WIRED** |
| RT6P-05 | Orders queue + detail chrome | split list/detail | `PageSection` queue + detail, `ListRowSkeleton` | **WIRED** |
| RT6P-06 | Tracking map + recent receipts | map empty + receipts strip | `PageSection` map/receipts, `EmptyState` on map | **WIRED** |
| RT6P-07 | Notifications inbox loaders | list skeleton | `ListRowSkeleton` on initial load | **WIRED** |

**RT-6P audit gaps (intentional / blocked):** pegasusX desktop ahead on category-suppliers API, AI order confirm/reject, tracking `recent_receipts`, client-policy banner (RT-5); Firebase phone OTP scaffold only; card Spanner persistence may 503; negotiation disabled ecosystem-wide; B2B dock checkout desktop-primary; procurement page depth vs pegasus not in RT-6P scope; insights/settings/cards use existing BentoGrid (no PageChrome migration — minimal diff).

**Exit:** Component-level desk tokens, skeleton loaders, KPI/section structure on dashboard, catalog, orders, tracking, notifications. UI-only — no new SSMR.

---

## Phase RT-8 — Ecosystem parity (desktop + Android + iOS)

| ID | Feature | Desktop | Android | iOS | Status |
|----|---------|---------|---------|-----|--------|
| RT8-01 | Preorder confirm / edit on Orders | `/orders` detail actions | `OrderedCard` + `OrdersViewModel` | `OrdersViewModel` + `OrderCardView` | **WIRED** |
| RT8-02 | Request-cancel for in-flight orders | `/orders` | `OrdersViewModel` fallback | `OrdersViewModel` + `OrderDetailSheet` | **WIRED** |
| RT8-03 | Post-register setup wizard | `/setup` | `AuthViewModel` post-register | `SetupView` + `needsSetup` gate | **WIRED** |
| RT8-04 | Insights prediction dismiss | `/insights` | `AnalyticsScreen` | `InsightsView` | **WIRED** |
| RT8-05 | Canonical nav map + READMEs | — | `README.md` + dock tab | `RetailerSection.swift` | **WIRED** |
| RT8-06 | Orphan cleanup | — | removed `profile/AutoOrderScreen` | wired `SearchView`; removed `InboxView` | **WIRED** |
| RT8-07 | iOS ViewModels | — | — | `OrdersViewModel`, `OnboardingViewModel`, `InsightsViewModel` | **WIRED** |
| RT8-08 | Pending checkout on reconnect | `PendingCheckoutFlusher` | `PendingOrderSyncWorker` | `PendingOrderReplayer` | **WIRED** (verified) |

**Exit:** Retailer row marked **Wired** in `ROLE_ROW_PARITY_MATRIX.md`; builds green on Android + iOS; `go test ./retailer/...` pass.

---

## Verification

```bash
cd pegasusX/apps/backend-go && go build ./...
cd pegasusX/apps/backend-go && go test ./retailer/...
cd pegasusX && make test-ssmr-infra   # PX_E2E_RETAILER_* markers
cd pegasusX/apps/retailer-app-android && ./gradlew compileDebugKotlin
```

---

## Phase RT-7 — Manual pre-order + delivery intent (Standard vs Scheduled)

| ID | Feature | Backend | Desktop | Android | iOS | Status |
|----|---------|---------|---------|---------|-----|--------|
| RT7-01 | Two-mode checkout (`STANDARD` / `SCHEDULED`, express, deliver-before) | `order/preorder_policy.go`, `Create`, unified checkout | `CheckoutModal` delivery intent | cart/checkout API fields | checkout API fields | **WIRED** |
| RT7-02 | Preorder confirm/edit gates | `Status=SCHEDULED`, `MANUAL_PREORDER`, `DRAFT` | `/orders` detail actions | `needsPreorderAction` | `needsManualPreorderAction` | **WIRED** |
| RT7-03 | Midnight Guard + `PRE_ORDER_*` events | `order/preorder_sweeper.go`, outbox | WS refresh | WS refresh | WS refresh | **WIRED** |
| RT7-04 | SSMR | smokecheck | — | — | — | **WIRED** (`PX_E2E_MANUAL_PREORDER_OK`) |

**Exit:** Retailer can place Standard (ASAP + optional deliver-before) or Scheduled (T+3+) pre-orders; warehouse sees commitments; SSMR green.

---

## Known remaining gaps (backend / cross-role)

- Card tokenization Spanner persistence may return stub/503 on some stacks — clients show honest errors.
- Quantity negotiation disabled ecosystem-wide (`PX_E2E_NEGOTIATION_SKIPPED`).
- B2B checkout / dock-only payment flows remain desktop-primary; mobile uses handoff copy where applicable.
- Full Firebase phone OTP not implemented — custom-token scaffold only.
