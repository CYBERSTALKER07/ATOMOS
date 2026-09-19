# End-Product Reality Report — Live Code Audit (`pegasusX/apps`)

**SoT:** `/Users/shakhzod/Desktop/V.O.I.D/pegasusX`  
**Method:** App folders under `apps/` only; docs ignored. Evidence from screens, ViewModels, Retrofit/`APIClient`s, and backend route wiring.

**App inventory (clients):**

| Role | Folders |
|------|---------|
| Retailer | `retailer-app-android`, `retailer-app-ios`, `retailer-app-desktop` (Next + Tauri) |
| Supplier | `supplier-app-android`, `supplier-app-ios`, `supplier-portal` |
| Driver | `driver-app-android`, `driver-app-ios` |
| Payload/Loading | `payload-app-android`, `payload-app-ios`, `payload-terminal` (Expo) |
| Factory | `factory-app-android`, `factory-app-ios`, `factory-portal` |
| Warehouse | `warehouse-app-android`, `warehouse-app-ios`, `warehouse-portal` (Next + Tauri) |
| Admin | `admin-portal` (single-page console) |

---

## Cross-cutting (all roles)

| Capability | Reality in code |
|------------|-----------------|
| **Auth** | Role-specific login routes (`/v1/auth/{retailer\|supplier\|driver\|warehouse\|factory\|payloader}/…`). Portals use cookie/JWT + refresh. Admin: paste `PLATFORM_ADMIN` bearer + MFA gate (`admin-portal/app/page.tsx:55-64`, MFA APIs in `admin-portal/lib/api.ts:150-156`). |
| **WebSockets** | `/v1/ws` (+ role session mint) used by retailer desktop, supplier/warehouse/factory portals, payload apps/terminal, driver, retailer mobile dashboard. |
| **FCM / push** | Clients register `/v1/user/device-token`. Backend `InitFCM` can fall back to **no-op** (`backend-go/bootstrap/bootstrap.go:1351-1357`). `LogTransport` logs instead of delivering when transports unwired (`backend-go/notifications/transport.go:13-27`). |
| **Offline** | Strongest: driver offline queue + telemetry Room; retailer POS offline cash queue; payload offline inject queue; retailer desktop pending POS/checkout flushers. |
| **Barcode / QR** | Payload + warehouse: `EanBarcodeScannerPreview` / DataWedge. Driver: QR scan for delivery. Retailer: QR **display** for dock (`QROverlay`), not general POS barcode checkout camera. |
| **GPS** | Driver `TelemetryService` + `FusedLocationProviderClient` (~10s, adaptive battery) → telemetry socket + offline DAO (`driver-app-android/.../TelemetryService.kt:157-217`). |
| **Payments UI** | Retailer card/cash checkout + POS; supplier treasury/chargebacks; warehouse payment-config; Global Pay **stub only** via `GLOBAL_PAY_STUB_MODE` (never production) (`backend-go/payment/global_pay_executor.go:64-74`). |

---

## 1. Retailer (Android / iOS / Desktop)

### What exists and works

**Android** — large Compose surface + `PegasusApi` (~100+ endpoints): auth/org/location, catalog, cart/checkout (cash/card/unified), orders/claims, AI predictions/preorder, auto-order settings/runs/shadow, POS (open/close/sales/holds) with offline cash sync, stock/local SKUs/sections/shifts/team/capabilities, credit/AR, HQ/reports/pulse/control-tower, dock/tracking, assist, notifications, FCM service.

Evidence: screens under `retailer-app-android/.../ui/screens/*`; API `PegasusApi.kt:68-647`; nav `RetailerNavigation.kt`; FCM `PegasusFirebaseMessagingService.kt:21-43`; POS offline `PosScreen.kt:202-496`.

**iOS** — parity-level SwiftUI screens (Dashboard, Catalog, Cart/Checkout, Orders, POS, AutoOrder, Credit, HQ, ControlTower, Dock/Arrival, DeliveryMap, Reports, etc.) + `APIClient.swift`.

**Desktop** — Next.js + Tauri; nav packs: dashboard, orders, tracking, dock, catalog, procurement, suppliers, credit, auto-order, stock/POS/shifts/sections/assist/insights/control-tower/reports/hq/settings (`RetailerShell.tsx:61-84`). Offline tray + pending POS/checkout flushers in `lib/`.

### Incomplete / weak / decorative

| Issue | Evidence |
|-------|----------|
| Settings rows are no-ops (General / Notifications prefs) | `SettingsSection.kt:68`, `:95` — `onClick = { }` |
| Analytics week-nav buttons no-op | `WeeklySpendCard.kt:100-106` |
| Live tracking can be empty while deliveries exist | Desktop copy: driver location “not available yet” (`tracking/page.tsx:229`) |
| Card path degrades when Global Pay unavailable | `CheckoutModal.tsx:439-493` |

### Features still needed (with how)

#### R1. Notification preferences UI (mobile)
- **Purpose:** Let retailers choose push/email/SMS channels (row already marketed).
- **Why:** Row is dead UI; backend has `/v1/user/notification-preferences` used by supplier apps.
- **Logic:** `GET prefs → PATCH { channel, enabled }`; map to FCM topic / quiet hours.
- **E2E:** Profile → Notifications → toggle → persist → FCM delivery honors prefs.

#### R2. Reliable live tracking when GPS trail missing
- **Purpose:** Show ETA / last-known / “awaiting telemetry” honestly.
- **Why:** UI already admits gap when active deliveries lack driver location.
- **Logic:** Prefer live WS telemetry; else last Spanner ping; else ETA from route geometry + Haversine remaining distance.
- **E2E:** Driver depart → telemetry → retailer map updates; if GPS off → explicit state, not blank map.

#### R3. POS barcode scan-to-cart (if end-product requires it)
- **Purpose:** Scan EAN → add line (today POS is search/price entry + local SKUs).
- **Why:** Warehouse/payload have scanners; retailer POS does not wire camera scan.
- **Logic:** Scan → `GET /v1/catalog/barcode/{ean}` or local-SKU match → cart line.
- **E2E:** Open till → scan → price → complete sale (online or offline cash).

---

## 2. Supplier (Android / iOS / Portal)

### What exists and works

Very broad product surface on all three clients:

- Ops: dashboard/pulse, orders (vet/reassign/bypass), manifests (start-loading/inject/seal), dispatch preview/execute, fleet + live map, exceptions/shop-closed/claims/early-complete
- Catalog/pricing/promotions/inventory import
- Planning brain / S&OP scenarios / replenishment policies / knowledge graph
- Network: warehouses, factories, zones, topology, supply lanes
- Treasury: ledger, payments, cash reconciliations, credit notes/profiles, chargebacks, compliance
- Credit desk: `/credit/collections`, policy, admin-disable (portal)
- Analytics / AI recommendations / earnings

Evidence: `SupplierRoutes` (`SupplierNavigation.kt:101-165`); `SupplierApi.kt` endpoints; portal pages under `supplier-portal/app/(portal)/**`; collections desk `credit/collections/page.tsx:43-73`.

### Incomplete / weak

| Issue | Evidence |
|-------|----------|
| Quantity negotiation UI exists; API **410 by default** | `negotiation_disabled.go:9-24`; clients still have Negotiations screens/pages |
| Control Tower playbooks/scored exceptions need flag | `PlaybookRunsPanel.tsx:61`, `ScoredExceptionsPanel.tsx:38` — `CONTROL_TOWER_PLAYBOOKS_ENABLED` |
| Earnings/payments fall back when settlement endpoint missing | `earnings/page.tsx:167`, `payments/page.tsx:101` |
| Credit **risk scoring product removed** (profiles/limits remain) | `credit/repository.go:616-620` |
| Demand flywheel needs POS/DDL | `analytics/demand/flywheel/page.tsx:35` |

### Features still needed

#### S1. Quantity negotiation — ship or delete
- **Purpose:** Driver proposes qty change; supplier resolves.
- **Why:** Clients still navigate to Negotiations; API returns `410 feature_disabled` unless `QUANTITY_NEGOTIATION_ENABLED=true`.
- **Logic (if enabling):** Propose `{order_id, lines[{sku, qty_proposed}]}` → pending → supplier accept/reject → adjust reserved stock + order total → outbox event. Timeout sweeper cancels stale proposals.
- **E2E:** Driver negotiate → supplier pending list → resolve → retailer/order totals update. **Or** remove Negotiations UI and stop advertising it.

#### S2. Credit risk scoring (or stop implying scores)
- **Purpose:** Rank / auto-limit retailers.
- **Why:** `GetScoresForRetailers` is an empty stub; desk uses limit/balance/delinquency only (`credit/handlers.go:87`).
- **Math (typical):**  
  `utilization = balance / limit`  
  `risk = w1*util + w2*delinquency + w3*days_past_due + w4*(1-pay_velocity)`  
  Tier cutoffs → auto freeze if `risk ≥ T_high` or `util ≥ 1`.
- **E2E:** Nightly job writes scores → collections sort → optional auto-hold → notify retailer.

#### S3. Settlement authority (earnings/payments)
- **Purpose:** Authoritative settlement summary vs ledger-derived fallback.
- **Why:** Portal already warns when settlement endpoint unavailable.
- **E2E:** Treasury page always hits one durable settlement API; no silent derivation without banner.

---

## 3. Driver (Android / iOS)

### What exists and works

Mature delivery OS: login, home, manifest, map + navigation cues, QR scanner, offload review, cash collection, payment waiting, shop-closed waiting, correction, supply transfers, offline verifier, sync queue, notifications, GPS telemetry + boot resume, WS, FCM device token, early-complete / reorder / rescue / split-payment clients.

Evidence: `DriverRoutes` (`DriverNavigation.kt:88-120`); `DriverApi.kt:69-391`; offline catalog endpoints; iOS `APIClient.swift` + `DriverOfflineQueue.swift`; GPS `TelemetryService.kt`.

Backend: when `OrderService` is wired (required — panics if nil, `driverroutes/routes.go:111-114`), real edges mount for arrive, shop-closed, partial-offload, bypass, reorder, early-complete, credit-delivery, missing-items, split-payment, etc. (`driverroutes/routes.go:87-110`, `orderroutes/routes.go:45-53`).

### Incomplete / intentionally broken

| Issue | Evidence |
|-------|----------|
| `PATCH /v1/orders/{id}/state` always **501** (fail-closed) | `mobile_compat.go:374-393`; still in `DriverApi.kt:162` |
| Mid-delivery update API **not implemented** | `delivery_handshake.go:108-109` — no durable writer |
| Negotiation → **410** by default | same as supplier |
| Compat fallbacks return 501 if OrderService unwired | `mobile_compat.go:592-605` (production panics instead) |
| Demo order fallback only if `ALLOW_DRIVER_DEMO_FALLBACK=true` | `mobile_compat.go:523-588` |

### Features still needed

#### D1. Remove or rewire `PATCH …/state`
- **Purpose:** Clients must not call a hard-501 path.
- **Why:** Documented as non-mutating; drivers should use edge routes.
- **E2E:** Client uses arrive / depart / collect-cash / partial-offload only; delete state-patch from API clients.

#### D2. Durable mid-delivery order update
- **Purpose:** Change lines while at stop without full amend/offload path.
- **Why:** `UpdateOrderDuringDelivery` fail-closed (`delivery_handshake.go:82-109`); tests assert `not_implemented`.
- **Logic:** Validate driver assignment + GPS proximity → apply line deltas in Spanner txn → emit outbox → return new `adjusted_total`.
- **E2E:** At ARRIVED → update lines → cash/card uses new total → fiscal uses adjusted amount.

#### D3. Offline → online reconciliation guarantees
- **Purpose:** Queue never silently drops money edges.
- **Why:** Offline catalog includes arrive/cash/partial/bypass/reorder; must map 1:1 to wired OrderService handlers.
- **E2E:** Airplane mode cash collect → reconnect → `/v1/sync/batch` or replay → Spanner + fiscal retry.

---

## 4. Payload / Loading (Android / iOS / Terminal)

### What exists and works

Loading bay workflow across three clients:

- Auth payloader login/refresh
- Trucks list, pulse, orders, manifests (supplier/factory dual paths)
- Start loading → checklist / barcode check → seal / seal-completed
- Inject order, reassign recommend, manifest exceptions, missing-items
- Inbound returns scan session
- WS live sync + offline queue + FCM registration

Evidence: `PayloadApi.kt:49-249`; `HomeViewModel.kt` (WS/FCM/barcode/offline); iOS `WebSocketClient.swift`, `HomeViewModel.swift:300-679`; Expo terminal screens (`ManifestWorkspaceScreen`, `TruckSelectionScreen`, …) + `useManifestData.ts` WS/offline.

### Incomplete / weak

| Issue | Evidence |
|-------|----------|
| Barcode check is **lookup + “SKU on order?”** — not a full pick confirmation ledger | `HomeViewModel.kt:570-596` |
| Seal/exception flows depend on backend/env; DLQ escalation messaging present | `HomeViewModel.kt:832` |
| Terminal is Expo (shared logic) vs native apps — three surfaces to keep in sync | `payload-terminal/package.json` |

### Features still needed

#### P1. Line-level load confirmation ledger
- **Purpose:** Prove each unit/case was scanned before seal.
- **Why:** Current scan only validates catalog membership on order.
- **Logic:** `required_qty[sku]`, `scanned_qty[sku]`; seal allowed iff `∀ sku: scanned ≥ required` (or variance approved).  
  Variance → `POST …/manifest-exception`.
- **E2E:** Start loading → scan each line → seal blocked until complete → seal emits sealed event → driver can depart.

#### P2. Single client SoT for payloader UX
- **Purpose:** Avoid Android / iOS / Expo drift.
- **Why:** Three implementations of same seal/inject/WS flows.
- **E2E:** Contract tests on `PayloadApi` paths shared by all three.

---

## 5. Factory (Android / iOS / Portal)

### What exists and works

Factory ops: login/setup location, dashboard, manifests (+ detail), loading bay, transfers (list/create/detail), supply requests, payload override / rebalance / cancel-transfer, fleet + live map, staff, insights/replenishment, analytics, manifest-exceptions, notifications, geocode.

Evidence: Android screens list; `FactoryApi.kt` (manifests/fleet/staff/replenishment/…); iOS `Views/*`; portal pages (`factory-portal/app/**`).

### Incomplete / weak

Narrower than warehouse (no full WMS pick-waves/cycle-counts/cold-chain desk). iOS/Android/portal largely aligned; fewer “decorative” stubs found than retailer settings.

### Features still needed

#### F1. Factory cold-chain / lot quarantine (if factory ships temp-sensitive)
- **Purpose:** Same excursion rules as WMS before outbound seal.
- **Why:** Cold chain is warehouse-flagged (`WMS_COLD_CHAIN_ENABLED`); factory clients lack temp ingest UI.
- **Logic:** `excursion = temp ∉ [min_c, max_c]` → quarantine lot → block transfer seal.
- **E2E:** Ingest reading → breach → exception → resolve → release.

#### F2. Cross-dock handoff SLA board
- **Purpose:** Factory→warehouse→payload timing visibility.
- **Why:** Loading bay components reference handoff timeline; needs durable SLA metrics end-to-end.
- **E2E:** Transfer created → ETA → late flag → ops alert.

---

## 6. Warehouse (Android / iOS / Portal)

### What exists and works

Richest ops client set:

Dashboard, orders, drivers/vehicles, inventory/bins/putaway, pick-waves, cycle-counts, dispatch preview/execute/rescue, fleet live map, manifests, preorders, tomorrow board, stock commitments, supply requests, replenishment insights, demand forecast, CRM, returns/reverse logistics, cold chain, labor capacity, exceptions/claims, treasury, payment config, staff, ops/location/return-policy, notifications, barcode scanner, portal handoff.

Evidence: `WarehouseSection.kt:14-48`; `WarehouseApi.kt:10-538`; portal pages (`warehouse-portal/app/**` — control-tower, dispatch, pick-waves, cold-chain, labor-capacity, …); mock data comment removed (`control-tower/page.tsx:30`).

### Incomplete / weak

| Issue | Evidence |
|-------|----------|
| Cold chain UI warns flag must be on | `ColdChainScreen.kt:166`; `stocklots` gated by `WMS_COLD_CHAIN_ENABLED` (`bootstrap.go:1234`) |
| Labor capacity / driver score depend on backend labor module | Android/iOS + portal pages call `/v1/labor-capacity/*` |
| Some heavy features portal-first; mobile “More” hubs | `MoreHubScreen` patterns |

### Features still needed

#### W1. Cold chain always-on for production chilled SKUs
- **Purpose:** Auto-quarantine on excursion.
- **Why:** Feature-flagged off by default; UI tells operators to enable API flag.
- **Logic:** On ingest: if `temp_c < min_c OR temp_c > max_c` → quarantine lots on manifest + raise breach.  
  Release only via exception resolve with reason.
- **E2E:** Flag on → ingest out-of-band temp → lot blocked from pick/dispatch → resolve → re-enter inventory.

#### W2. Labor capacity → dispatch coupling
- **Purpose:** Don’t over-assign routes beyond zone hours / driver scores.
- **Why:** Capacity APIs exist; dispatch must refuse or warn when `required_hours > available_hours`.
- **Math:** `available = Σ driver_available_hours(zone, day)`; `required = Σ route_planned_hours`; block execute if `required > available * (1 + overtime_slack)`.
- **E2E:** Set availability → dispatch preview shows unavailable drivers → execute blocked/warned.

---

## 7. Admin (`admin-portal`)

### What exists and works

Thin break-glass console (~850 LOC UI):

- Paste PLATFORM_ADMIN token
- MFA enroll/confirm/verify
- Tabs: Tenants (KYB transition), Flags (eval/set/approve), Audit, Product match queue, Partner keys/AS2/SFTP/COA + **AR dunning run-once**
- Admin WS refresh hook

Evidence: `admin-portal/app/page.tsx`; panels; `lib/api.ts:82-159`.

### Incomplete / weak

This is **not** a full admin product: no normal login UX, no user management console, no ops dashboards, no billing admin beyond dunning trigger, no multi-page IA.

### Features still needed

#### A1. Real admin identity (not token paste)
- **Purpose:** Secure break-glass with SSO/IdP + MFA (MFA exists; login does not).
- **E2E:** IdP login → MFA → scoped PLATFORM_ADMIN session → audit every mutation.

#### A2. Tenant lifecycle + support tooling
- **Purpose:** Impersonation-safe support, disable tenants, view outbox/DLQ.
- **Why:** Today only status transition + flags + match queue + partner config.
- **E2E:** Search tenant → view health → flag flip with approval → audit row.

#### A3. Collections/dunning as scheduled product (not only run-once)
- **Purpose:** Automated AR cadence.
- **Why:** UI is manual `POST /v1/admin/ar/dunning/run-once` (`PartnerPanel.tsx:119-122`).
- **Logic:** For invoices with `days_past_due ≥ D_n`, send stage-n notice; escalate; freeze credit at stage-final.
- **E2E:** Cron → dunning stages → retailer notify → collections desk sees status.

---

## Backend edges that block “end product” (shared)

| Gap | Status | Refs |
|-----|--------|------|
| Order state patch | Always 501 | `mobile_compat.go:382-393` |
| Mid-delivery update | `not_implemented` | `delivery_handshake.go:108-109` |
| Quantity negotiation | 410 unless env on | `negotiation_disabled.go:22-24` |
| Credit risk scores | Empty stub | `credit/repository.go:616-620` |
| FCM | Can be no-op | `bootstrap.go:1351-1357`, `fcm.go:78-90` |
| Notification LogTransport | Log-only fallback | `transport.go:13-27` |
| Global Pay | Stub only non-prod | `global_pay_executor.go:64-74` |
| Cold chain | Env-gated | `bootstrap.go:1234` |
| Control tower playbooks | Flag-gated | supplier portal panels |

---

## Honesty summary (maturity)

| Role | Client maturity | Backend coupling | Biggest hole |
|------|-----------------|------------------|--------------|
| Retailer | High (3 platforms) | Strong | Dead settings prefs; tracking GPS lag; no POS scan |
| Supplier | High (3 platforms) | Strong | Negotiation UI vs 410; score stub; settlement fallbacks |
| Driver | High (2 platforms) | Strong if OrderService wired | 501 state patch; mid-delivery update missing |
| Payload | High (3 clients) | Strong | Scan ≠ full load ledger; triple-client drift risk |
| Factory | Medium-high | Strong | Narrower ops; no cold-chain desk |
| Warehouse | Highest | Strong | Cold chain / labor flags for full production |
| Admin | **Low (break-glass only)** | Narrow admin APIs | Not an end-user admin product |

**Bottom line:** Operational roles (retailer/supplier/warehouse/driver/payload/factory) have real screens, ViewModels, and wired `/v1/...` clients. The remaining “end product” gaps are concentrated in (1) intentionally fail-closed or flag-gated backend features clients still expose, (2) removed credit-scoring product, (3) push/FCM degradation paths, and (4) admin being a token console rather than a full governance app.

# Universal Agent & Engineering Guidelines
When developing, designing, or planning, always ensure to account for:
- Gaps, edge cases, and comprehensive feature validation.
- Best practices and optimized integration for Kafka, Redis, Backend, Optimizers, AI, and UI.
- Real-time concepts including WebSockets, webhooks, and their native app equivalents.
- Thorough business logic for features, understanding how the role, app, and ecosystem work together, and engagements with other roles and features.
- Best practices for backend, frontend, and infrastructure libraries/packages. Always prefer existing, high-quality open-source libraries and packages that best suit our features before creating our own.
- Optimal UI infrastructure and UX patterns (e.g., optimal screen positioning for drivers during an active route), applying the same high standards to backend and cloud architecture.
- ALWAYS search the web to find open-source code, libraries, packages, math, algorithms, approaches, and best practices for anything we are doing. If none exist, then create our own.
- Always search the web to get the correct logic, and incorporate edge cases, business logic for features, operations (ops), workflow, data consistency, finance, and AI into everything we do.
