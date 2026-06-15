# pegasusX RETAILER Role — Phased Execution Ledger

**Scope:** pegasusX only · **Parent plan:** `VEGETABLE_PLAN.md` §2.5  
**Last updated:** 2026-06-15

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

## Verification

```bash
cd pegasusX/apps/backend-go && go build ./...
cd pegasusX/apps/backend-go && go test ./retailer/...
cd pegasusX && make test-ssmr-infra   # PX_E2E_RETAILER_* markers
cd pegasusX/apps/retailer-app-android && ./gradlew compileDebugKotlin
```

---

## Known remaining gaps (backend / cross-role)

- Card tokenization Spanner persistence may return stub/503 on some stacks — clients show honest errors.
- Quantity negotiation disabled ecosystem-wide (`PX_E2E_NEGOTIATION_SKIPPED`).
- B2B checkout / dock-only payment flows remain desktop-primary; mobile uses handoff copy where applicable.
- Full Firebase phone OTP not implemented — custom-token scaffold only.
