# PegasusX Vegetable Plan (Granular Feature-by-Feature Execution Ledger)

**Scope:** pegasusX/ ONLY. pegasus/ is read-only reference for UI layouts, design systems, component patterns, code/arch approaches, and feature ideas that are suitable for a **single-tenant** ecosystem (one primary SUPPLIER, thousands of retailers, multiple warehouses/factories under that supplier).  
**Never edit pegasus/.** All implementation, new files, and status updates happen exclusively inside pegasusX/.

**Last updated:** 2026-06-15 (retailer RT-6P desktop portal deep UI parity).  
**Owner:** Boss / F.R.I.D.A.Y. (single-tenant SSMR/PX12 focus).  
**Status model (per item):** `TODO` | `IN_PROGRESS` | `WIRED` | `E2E_SSMR_GREEN` | `PROD_CANDIDATE` | `BLOCKED` | `DEFERRED` (intentional v1 delta).  
**Reconciliation rule:** After every edit batch, update this file + `context/plan.md` + relevant ledgers/gap docs + diagrams in the **same commit set**. Code is ground truth.

**Core Principles (from Boss directive + existing doctrine):**
- Every feature/capability must be **end-to-end complete** across the **whole ecosystem** before marking done: backend (with outbox/Kafka/WS/cache), all affected role clients (web + native + desktop where applicable for the role), cross-role interactions/confirmations, events, sync, offline/reconnect where relevant.
- Phased + smart: respect hard dependencies. Some features are useless or unsafe without predecessors (e.g. dispatch capacity recs require fleet CRUD + topology + home-node scoping + freeze locks + preview/execute paths + UI on warehouse portal + native warehouse apps + supplier oversight + driver notification).
- Single-tenant simplifications encouraged where they reduce complexity without losing capability (fixed/seeded supplier, no supplier-switcher UIs, simpler discovery always returning the one supplier, home-node always resolved, MAX_SUPPLIERS policy for controlled growth).
- UI/Layout/Design/Component replication: copy good patterns from pegasus reference (BentoGrid, shells, PageChrome/desk-page patterns, M3 discipline via custom tokens in pegasusX/packages/ui-kit, native SwiftUI/Compose mirrors). pegasusX already has `packages/ui-kit`, `packages/types`, `packages/api-client` — enhance/extend these as the canonical pegasusX surface rather than duplicating per-app. Maintain UI freeze on existing surfaces unless explicitly discussed.
- New feature ideas (even if good in pegasus and single-tenant suitable): **first discuss** with Boss. Do not implement speculatively. List in "Candidates for Discussion" section.
- Verification: backend changes + SSMR (docker) green with new/updated `PX_E2E_*` markers first. Then role-row clients together. Then full parity-contract-full / gap-hunter-gate / validate-launch-readiness. Simulate external services (Spanner/Redis/Kafka via docker, webhooks via local, FCM/APNs via noop or emulator) for green pass now; wire real cloud later.
- 90-day style horizon (or longer — no rush, quality > speed). Grouped so one phase gives "more context" across connected surfaces.
- Backend-first per cluster, then all clients for affected roles in lockstep.

**Related (always read before touching a phase):**
- `context/plan.md` (canonical phased anchors PX-0…PX-12+)
- `docs/FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`
- `docs/ROLE_ROW_PARITY_MATRIX.md`
- `context/parity-ledger.md`
- `docs/DEPLOYMENT_READINESS_GAP_LEDGER.md`
- `docs/BACKEND_ECOSYSTEM_READINESS.md`
- `docs/LAUNCH_READINESS_RUNBOOK.md` + qa/ runbooks
- Relevant `assets/diagrams/pegasusx-*.mmd`
- SSMR `make test-ssmr-infra` and focused subchecks.

---

## 1. Backend Feature Inventory (Master List for All Surfaces)

This is the "list of all the features in the backend" that apps/roles depend on. Grouped by vertical for phase planning. Status reflects audit of current code + existing plans/ledgers as of 2026-06-14. Many are already `E2E_SSMR_GREEN` or `WIRED`; active work (user's recent dispatch/fleet/payload changes) reflected as `IN_PROGRESS`.

### 1.1 Auth, Identity, Onboarding, Topology, Billing (Foundation)
- Supplier register (4-step: account+phone intl, location/warehouse+billing addr, business/tax/fleet, categories) + billing gate (`/setup/billing`).
- Supplier login/refresh/session (cookie + JWT ADMIN), ws-session mint for native.
- Retailer self-reg + seeded supplier linkage + profile (receiving windows).
- Driver/vehicle/org-member CRUD (supplier creates; warehouse/factory ops scoped create via home node).
- Topology: warehouses + factories + co-locate (transfer_mode TRUCK|INTERNAL, co_locate_with_factory_id).
- Home-node scoping + role enforcement (WAREHOUSE_ADMIN / FACTORY_ADMIN via SupplierRole + home_node resolution; no body spoofing).
- Status: mostly `E2E_SSMR_GREEN` / `WIRED` (see PX1 anchors). Single-tenant: one seeded supplier by default.

### 1.2 Order Lifecycle + Exceptions (ShopClosed, Negotiation)
- PENDING → LOADED → IN_TRANSIT → ARRIVED → COMPLETED (geofence gated for COMPLETE).
- Shop-closed protocol (driver report → retailer open-now/negotiate → supplier resolve/escalate).
- Negotiation (propose/counter/resolve on delivery terms).
- Driver edges (reorder, bypass-offload, credit, missing-items, split-payment).
- Status: `E2E_SSMR_GREEN` for core + shop-closed + negotiation (PX9-A1, PX10-B, PX12-B1). Outbox + dispatcher + all role hubs.

### 1.3 Catalog, Inventory, Pricing, Replenishment/Supply
- Catalog browse/search (categories, all-products, suppliers, per-supplier), My Suppliers connect/remove.
- Inventory (supplier/warehouse views + patches), pricing rules (supplier).
- Supply requests (warehouse SUBMITTED → factory ACK → IN_PRODUCTION → READY → FULFILL; co-locate auto).
- Transfers (truck path manifest; internal auto-receive + inventory credit).
- Replenishment insights (durable, not demo).
- Status: core `WIRED`/`E2E` (PX12-C1 etc.); replenishment insights durability **WIRED** (WH1–WH2).

### 1.4 Fleet (Drivers, Vehicles, Assign, Guards)
- CRUD drivers/vehicles (supplier + node-admin scopes).
- Assign to home node / active routes.
- Fleet guards (active-route, capacity, home-node enforcement).
- Status: recent hardening — `E2E_SSMR_GREEN` (`PX-FLEET-001`).

### 1.5 Dispatch, Capacity, Execute, Locks (Warehouse Ops Core)
- Dispatch preview (geo-batch H3 + Tetris capacity fit).
- Capacity recommendations + warnings + suggested unselect + force audit (capacity_recommend.go).
- Dispatch lock (freeze for manual ops vs AI).
- Execute (manifest split, assignment, outbox).
- Dispatch settings (tunable).
- Status: capacity + execute `E2E_SSMR_GREEN` (`PX-DISP-002`). Preview/locks `E2E_SSMR_GREEN`.

### 1.6 Telemetry, Live Maps, Tracking
- Driver location POST → TelemetryHub (driver + supplier rooms).
- Live fleet maps (animated markers, route geometry, stale flags) on supplier/warehouse portals + native.
- Planned vs actual, breadcrumbs/trails (additive in some clients).
- Status: core `E2E_SSMR_GREEN` (PX5-A2, PX12-F). Client trails/pills ahead in pegasusX per parity audits.

### 1.7 Manifest + Payload Lifecycle (Factory/Payload/Driver Gate)
- Manifest DRAFT → LOADING → SEALED → DISPATCHED → COMPLETED (per-truck + aggregate seal-complete batch).
- Payload: list/detail, start-loading, inject order, seal multi-truck, exceptions, reassign (recommend + durable apply).
- Driver manifest gate (pre-seal block).
- Device tokens + policy.
- Status: durability `E2E_SSMR_GREEN` (PX9-B, PX-PAY-003); multi-truck seal `PX_E2E_PAYLOAD_SEAL_FLOWS_OK`.

### 1.8 Payment, Ledger, Treasury, Webhooks
- Checkout unified (card/cash), sessions, webhooks (signature-first + idempotency), chargeback/reversal.
- Double-entry ledger, earnings, disputes, settlement.
- Status: `E2E_SSMR_GREEN` (PX3, PX-ECO-002). Scaffold for some providers; prod SDKs P2.

### 1.9 Analytics, Insights, AI Recs, Geo
- Supplier dashboard KPIs, earnings, activity, geo-report (H3), AI recs.
- Warehouse demand forecast, replenishment insights.
- Factory analytics/insights.
- Status: many `WIRED`; deeper native analytics + durable insights partial (`IN_PROGRESS` per P5/P6 in parity).

### 1.10 Realtime, Events, Notifications, Cache, Outbox, Platform
- 7 WS hubs + Redis relay + signed fallback tokens.
- Transactional outbox for all mutating state changes (TopicMain + FreezeLocks).
- Kafka consumers (notification_dispatcher ~90 handlers, warehouse supply consumer, ai-worker freeze/locks).
- Cache invalidate post-commit (Redis Pub/Sub).
- Client policy / version gating / SYSTEM_APP_OUTDATED / safe deferral.
- FCM/APNs for driver + retailer.
- Status: core `implemented` (PX9-D, PX11-B/C, PX12-C2 etc.); **durable notification inbox E2E_SSMR_GREEN** (unified `/v1/user/notifications` mount + strong Spanner reads); **ai-worker TopicFreezeLocks consumer WIRED** (`apps/ai-worker/freeze_registry.go`); **SSMR kafka-init provisions `pegasusx-freeze-locks`** (2026-06-15). Remaining: deeper native analytics screens (P2 UI freeze).

### 1.11 Cross-Cutting (Scaffold vs Durable, Single-Tenant, Infra)
- Scaffold in-memory fallbacks (bootstrap + per-domain) for SSMR/dev; strict `REQUIRE_INFRA_ADAPTERS=true` forces Spanner/Redis/Kafka.
- Single-tenant seeds + MAX_SUPPLIERS + fixed supplier discovery.
- Status: intentional and documented. No "real" gaps here — expected.

**Overall Backend Health (audit 2026-06-14):** Vast majority of production paths use outbox + RW txn + post-commit invalidate. Inline Kafka only in telemetry (by design) or legacy paths (migrate on touch). SSMR passes with markers. Real gaps are durability for a few analytics/insights + ai-worker consumers + full cross-client wiring for the newest dispatch/payload work.

---

## 2. Role → App → Backend Features + UI + Sync + Status (The Vegetable Breakdown)

For each role, list its apps (all must stay in sync per doctrine). For each app/surface: concrete backend features it consumes (cross-ref section 1), UI/layout replication targets (pegasus ref, read-only; target pegasusX/packages/ui-kit + per-app compose), required cross-ecosystem sync (events/WS/roles that must also execute or confirm), E2E criteria, current status, phase.

**Rule:** When a phase touches one, plan the minimal set of other apps/roles that must move in the same change set for E2E (e.g. warehouse dispatch change → warehouse-portal + warehouse-android + warehouse-ios + supplier oversight surfaces + driver notification + payload if manifest-adjacent + backend + events).

### 2.1 SUPPLIER Role
**JWT:** ADMIN (primary supplier scope).  
**Apps:** supplier-portal (primary, richest), supplier-app-android, supplier-app-ios, supplier-app-desktop (Tauri via supplier-portal or dedicated thin).  
**Primary backend:** supplierroutes + supplier/* + shared (orders, dispatch preview, fleet, manifests, replenishment, exceptions, treasury, ai).

#### supplier-portal (Next.js + Tauri desktop)
- **Depends on backend features (1.x):** 1.1 (onboard/topology/billing), 1.2 (orders + shopclosed + negotiate), 1.3 (catalog/inv/pricing/replenish), 1.4 (fleet), 1.5 (dispatch preview/execute), 1.6 (live map), 1.7 (manifests cross-read), 1.8 (earnings/payments/treasury), 1.9 (dashboard/analytics/ai/geo), 1.10 (ws session, supplier rooms, broadcast).
- **UI/Layout/Design targets from pegasus ref (replicate/maintain in pegasusX/packages/ui-kit + this app):** BentoGrid (components/BentoGrid.tsx) for dashboard anchor/stat/list/control cells; SupplierShell / split auth + circle stepper; PageChrome / desk-page for ops lists (orders, fleet, dispatch, manifests, exceptions, treasury); M3 tokens via globals.css + ui-kit (no @material/web); dense tables/kanban, org-fleet map, sparklines. Flat URLs (intentional single-tenant). See parity-ledger for 2026-06 parity claims.
- **Cross-sync obligations:** Supplier WS room for nearly everything; must see warehouse dispatch actions, factory manifest progress, payload seals, driver telemetry, retailer orders. Outbox events consumed by dispatcher for notifications to other roles.
- **E2E criteria:** Full ops spine visible + actionable; live map animated; dispatch preview/execute (with capacity); exceptions (shopclosed/negotiate) resolvable; treasury live; WS + Kafka fanout verified in SSMR; `PX_E2E_ORDER_OK` etc. umbrella.
- **Current status:** Phases 0–7 **WIRED** on portal + native ops slice (see `context/SUPPLIER_PHASE.md` SP0–SP7 / SP-7P). SP6 **E2E_SSMR_GREEN** (client-policy + native notification inbox). SP-7P portal deep component parity (skeleton loaders, KPI tiles, section chrome, analytics/ops/treasury/dispatch polish) — UI-only, no new SSMR.
- **Phase:** Supplier portal row v1 complete for scoped phases. SP4-03 import session wizard **E2E_SSMR_GREEN** (sync `/ingest` + async GCS/local worker via `INVENTORY_IMPORT_UPLOADED`; `PX_E2E_SUPPLIER_IMPORT_WIZARD_OK`, `PX_E2E_SUPPLIER_IMPORT_ASYNC_OK`).

#### supplier-app-android + supplier-app-ios (native ops slice)
- **Depends on:** Subset of above — fleet/orders/manifests/exceptions/shopclosed/negotiate/dispatch-preview/activity/ledger/replenish-trigger + profile + ws.
- **UI/Layout:** Mirror pegasus driver/retailer native patterns where applicable, but supplier ops (Kotlin/Compose M3 or SwiftUI). Additive features already noted in parity audits (connection state, etc.). Use pegasusX packages for shared models.
- **Cross-sync:** Same supplier rooms + push (FCM/APNs optional for supplier v1).
- **E2E:** Ops panels functional + realtime + offline-tolerant.
- **Status:** `implemented` (PX8-A3, PX11-B2, PX12-H) per ledgers; thinner than portal by design. **SP-7 (2026-06-15):** Android + iOS deep UI pass — shared KPI/list/status components, 160dp adaptive grids, refresh toolbar actions, `SupplierStatePane` / `SupplierLoadingState` (Android) and `SupplierLoadingView` / `SupplierErrorView` (iOS); primary ops screens aligned with factory/warehouse native M3/SwiftUI discipline — see `context/SUPPLIER_PHASE.md` SP7.
- **Phase:** Update together with any supplier-portal backend-facing change that affects ops.

(Desktop via Tauri on supplier-portal shell — parity maintained.)

### 2.2 WAREHOUSE_ADMIN Role
**JWT:** ADMIN + WAREHOUSE_ADMIN + home_node.  
**Apps:** warehouse-portal, warehouse-app-android, warehouse-app-ios.  
**Primary backend:** warehouseroutes + warehouse/* + shared dispatch/fleet/supply/replenish.

#### warehouse-portal (Next.js + Tauri)
- **Depends on:** 1.3 (supply requests, transfers, replenishment), 1.4 (fleet CRUD + guards), 1.5 (dispatch preview/execute/capacity/locks/settings), 1.6 (ops fleet live-map), 1.7 (manifest visibility?), 1.9 (demand-forecast, insights), 1.2 (order mutations), 1.10 (warehouse rooms + supplier fanout).
- **UI/Layout from pegasus ref:** WarehouseShell / KPI dashboard grid (not always Bento); PageChrome + desk-table / dense ops tables for orders/dispatch/manifests/supply-requests/dispatch-locks/transfers; order detail mutation panel (`/orders/[id]`); transfer action panel (pegasusX-only under Operations). Replicate good patterns (chrome, headers) from pegasus warehouse-portal. Enhance pegasusX ui-kit.
- **Cross-sync:** Warehouse room; interacts with factory (supply), payload (via manifests), driver (assign/execution), supplier (oversight). Supply WS + Kafka consumers. Dispatch locks freeze AI.
- **E2E criteria:** Supply full state machine + WS; dispatch lock + capacity + execute end-to-end with fleet map updates + driver notification; transfers receive + inventory; order mutations visible to retailer/driver; `PX_E2E_WAREHOUSE_*` + replenishment markers.
- **Status:** Portal + ops `WIRED`/`E2E` (PX4-A1, PX12-I). **Replenishment insights durability (2026-06-14):** Spanner-backed `ReplenishmentInsights` + approve/dismiss — see `context/WAREHOUSE_PHASE.md` WH1-*. **Replenishment engine + native fleet live map:** WH2–WH3. **WH-7/8/9 closed (2026-06-15):** client-policy banners (portal/Android/iOS), native notification inbox (Android/iOS), auth/mutation paths verified — see `context/WAREHOUSE_PHASE.md`. **WH-10 closed (2026-06-15):** dashboard fleet-status chips + KPI ALERT/DONE on native, portal `fleet_status[]` decode fix, WS-aware shell live indicator — see `context/WAREHOUSE_PHASE.md`. **WH-11P closed (2026-06-15):** portal deep component parity (skeleton loaders, `KpiStatCard`/`PageSection`, dispatch control room, replenishment insight cards, treasury KPI grid) — UI-only, no new SSMR — see `context/WAREHOUSE_PHASE.md`. SSMR: `PX_E2E_WAREHOUSE_CLIENT_POLICY_OK` + existing warehouse markers.

#### warehouse-app-android + warehouse-app-ios
- **Depends on:** Same core ops slice (supply, dispatch, fleet map, transfers, order mutations, insights).
- **UI/Layout:** Compose/SwiftUI mirrors of portal patterns + native maps (MapLibre/MapKit). Additive: connection state, trails per prior audits. pegasusX ui-kit tokens where shared.
- **Cross-sync:** Same as portal; native must participate in dispatch/transfer flows.
- **E2E:** Native order detail mutations + transfer actions + dispatch participation green (see PX12-I, PX_E2E_WAREHOUSE_ORDER_MUTATION_OK etc.).
- **Status:** Shells + basic wiring `implemented`; **WH-10 (2026-06-15):** native dashboard fleet-status row + KPI alert chips aligned with portal. **WH-11 (2026-06-15):** iOS deep UI pass — shared `KpiTile` / `WarehouseStatusBadge` / `WarehouseSectionHeader`, `WarehouseLoadingView` / `WarehouseErrorView`, semantic `LabTheme.statusTint`, adaptive KPI grids + refresh toolbars on dispatch/replenishment/supply/treasury/fleet — see `context/WAREHOUSE_PHASE.md` WH11. **WH-11A (2026-06-15):** Android deep UI pass — shared `WarehouseUiComponents` / `WarehouseState` (`WarehouseKpiTile`, `WarehouseMetricTile`, `WarehouseStatusChip`, `WarehouseOpsListCard`, `WarehouseLoadingState`, `WarehouseStatePane`), 160dp adaptive grids + `IconButton` refresh on dashboard/dispatch/replenishment/supply/transfers/fleet/treasury — see `context/WAREHOUSE_PHASE.md` WH11A. Full feature parity with portal for ops surfaces advancing with current dispatch work.
- **Phase:** Move in lockstep with portal for any warehouse ops change.

### 2.3 FACTORY_ADMIN Role
**JWT:** ADMIN + FACTORY_ADMIN + home_node.  
**Apps:** factory-portal, factory-app-android, factory-app-ios.  
**Primary backend:** factoryroutes + factory/* + manifest + supply + transfers.

#### factory-portal + native (android/ios)
- **Depends on:** 1.3 (supply queue + transitions + fulfill), 1.7 (manifest lifecycle full: draft/loading/seal/dispatch/complete + rebalance/override + exceptions + payload integration), 1.4/1.5 (fleet/transfer/dispatch visibility?), 1.9 (analytics/insights), 1.10 (factory rooms + MANIFEST_* fanout).
- **UI/Layout from pegasus ref:** Factory shell/dashboard with WorkflowLaunchRow, KPI cards (relabel to "Gate exceptions" etc. as in pegasusX audits); ManifestList/Detail, ManifestExceptions, StaffDetail, LoadingBay, PayloadOverride, CreateTransfer, TransferList/Detail, Insights. Replicate pegasus factory-portal layouts/patterns into pegasusX factory-portal + enhance ui-kit. Native: identical structure + adaptive (iPad split, Android two-pane).
- **Cross-sync:** Factory room + supplier + warehouse (transfers) + payload (manifest execution gate) + driver (via manifest). Full MANIFEST_* events + outbox. Payload override rebalance affects multiple manifests.
- **E2E criteria:** Supply ACK→FULFILL + transfer; manifest full lifecycle + seal + dispatch + complete; payload override + exceptions; driver gate; `PX_E2E_FACTORY_*` + manifest markers.
- **Status:** Lifecycle durability `implemented` (PX4-A2, PX9-B); role-row UI parity `implemented` (PX12-J) with pegasusX often ahead (additive screens). **FA-7/8/9 closed** (2026-06-15): client-policy banners (portal/Android/iOS), native notification inbox (Android/iOS), auth/mutation paths verified. **FA-10 closed** (2026-06-15): portal notification top bar + mark-read, dashboard Gate Exceptions KPI relabel, workflow launch grid parity (manifests/exceptions/analytics portal; create transfer + insights iOS) — see `context/FACTORY_PHASE.md`. SSMR: `PX_E2E_FACTORY_CLIENT_POLICY_OK`, `PX_E2E_FACTORY_NOTIFICATION_INBOX_OK` + existing factory markers.
- **Phase:** Payload/manifest adjacent work moves factory row together.

### 2.4 DRIVER Role
**Apps:** driver-app-android, driver-app-ios (canonical driverappios structure). No desktop.  
**Primary backend:** driverroutes + driver/* + telemetry + order edges + manifest-gate.

- **Depends on:** 1.6 (telemetry + tracking), 1.2 (edges: reorder/bypass/credit/missing/split + shopclosed + negotiation), 1.7 (manifest gate + execution), 1.4 (assign/profile vehicle), 1.10 (driver rooms + telemetry hub + FCM + client policy).
- **UI/Layout from pegasus ref:** Pegasus driver Android/iOS are the reference; pegasusX audits show pegasusX at parity or ahead (additive WsConnectionPill, LocationTrail polyline, live socket observers, ManifestScreen wrapping, Login additive). Use native platform patterns (Compose M3, SwiftUI HIG + SF Symbols). No pegasus graft needed — maintain/extend the good additive patterns.
- **Cross-sync:** Driver WS + supplier/warehouse telemetry rooms; manifest seal from payload/factory blocks gate; shopclosed/negotiate involves retailer + supplier; order mutations from warehouse visible.
- **E2E criteria:** Full delivery path + edges + telemetry + offline queue + shopclosed + negotiation + gate + profile assign detection; `PX_E2E_DELIVERY_OK`, `PX_E2E_TELEMETRY_OK`, `PX_E2E_DRIVER_EDGES_OK`, `PX_E2E_DRIVER_ASSIGN_DETECTION_OK`.
- **Status:** Core + edges + telemetry `E2E_SSMR_GREEN` / `implemented` (PX5, PX12-B/F). Design parity closed with pegasusX ahead on connection UX. **DR-7/8/9 closed** (2026-06-15): client-policy banners (Android/iOS home), native notification inbox verified with driver JWT SSMR — see `context/DRIVER_PHASE.md`. SSMR: `PX_E2E_DRIVER_CLIENT_POLICY_OK`, `PX_E2E_DRIVER_NOTIFICATION_INBOX_OK` + existing delivery markers.
- **Phase:** Any manifest/payload/dispatch change that affects execution must include driver row.

### 2.5 RETAILER Role
**Apps:** retailer-app-desktop (richest procurement), retailer-app-android, retailer-app-ios.  
**Primary backend:** retailerroutes + orderroutes + catalogroutes + retailer/* + payment.

- **Depends on:** 1.3 (catalog + my-suppliers + search), 1.2 (order create + tracking + post-order decisions + shopclosed), 1.8 (checkout unified + tracking + disputes), 1.10 (retailer room + FCM + policy), 1.1 (reg + profile + receiving windows).
- **UI/Layout from pegasus ref:** Desktop procurement (richest); mobile catalog-first + tracking. Replicate browse chips (Categories/All/Suppliers), flat grid, search, connect-vendor sheet, receiving window editors, unified checkout. pegasusX parity-ledger notes desktop + mobile catalog/supplier parity closed (2026-06). Use pegasusX ui-kit for web; native platform for mobile.
- **Cross-sync:** Retailer room; supplier sees orders/exceptions; driver/shopclosed negotiation; payment webhooks settle orders visible to supplier.
- **E2E criteria:** Reg → catalog → my-suppliers → checkout (card/cash) → tracking → receive/confirm; shopclosed flow; receiving windows respected; `PX_E2E_CATALOG_OK`, order/payment markers.
- **Status:** Register/order/tracking/catalog suppliers/auto-order `E2E` (PX2, PX12-G/B3). **RT-2/RT-3/RT-4 closed** (2026-06): cancel + request-cancel (desktop/Android), card initiate checkout, catalog path fix (iOS), token refresh, insights/cards/category-suppliers/pricing/tracking receipts, client-policy banners — see `context/RETAILER_PHASE.md`. **RT-5 closed** (2026-06-15): client-policy banner parity (portal/Android/iOS home) + native notification inbox verified with retailer JWT SSMR — see `context/RETAILER_PHASE.md`. **RT-6 closed** (2026-06-15): iOS deep UI pass — see `context/RETAILER_PHASE.md` RT6. **RT-6A closed** (2026-06-15): Android deep UI pass — shared `RetailerUiComponents` / `RetailerState`, 160dp adaptive catalog grids, tracking refresh toolbar, empathy loading/empty — see `context/RETAILER_PHASE.md` RT6A. **RT-6P closed** (2026-06-15): retailer-app-desktop deep component parity (`Skeleton`, `PageChrome`, `KpiStatCard`, `PageSection`; dashboard/catalog/orders/tracking/notifications polish) — UI-only, no new SSMR — see `context/RETAILER_PHASE.md` RT6P. SSMR: `PX_E2E_RETAILER_RECEIVING_WINDOW_OK`, `PX_E2E_RETAILER_CATALOG_PRODUCTS_OK`, `PX_E2E_RETAILER_CANCEL_OK`, `PX_E2E_RETAILER_CARD_INITIATE_OK`, `PX_E2E_RETAILER_CLIENT_POLICY_OK`, `PX_E2E_RETAILER_NOTIFICATION_INBOX_OK`. **Open:** card Spanner persistence (503 scaffold), negotiation disabled, B2B/dock desktop-only, full Firebase OTP, desktop preorder edit UI.
- **Phase:** Backend card persistence + negotiation re-enable when product directs; iOS Xcode smoke on device.

### 2.6 PAYLOAD Role
**Apps:** payload-terminal (Expo), payload-app-ios (iPad), payload-app-android (tablet).  
**Primary backend:** payloaderroutes + payload/* + platform (device token) + manifest.

- **Depends on:** 1.7 (manifest full lifecycle + reassign + exceptions + driver gate + multi-truck seal + aggregate seal-complete), 1.10 (payload rooms + device token + policy), 1.4/1.5 (fleet visibility for loading?).
- **UI/Layout from pegasus ref:** PayloadTerminal Expo + tablet natives. pegasusX audits: byte-identical themes/components except data-layer scoping (`/v1/payloader/*`) + adaptive polish (iPad split column, Android ListDetailPaneScaffold auto-open). Replicate/maintain the good adaptive + state panels. pegasusX packages for shared.
- **Cross-sync:** Payload room + factory (manifest control) + supplier + warehouse + driver gate. MANIFEST_* events critical. Seal batch affects fleet/driver.
- **E2E criteria:** Login → manifest list/detail → loading/inject/seal (multi-truck) → dispatch → complete; reassign recommend+apply durable; exceptions; driver gate; device token; `PX_E2E_PAYLOAD_OK` + sub-markers (LIFECYCLE, REASSIGN, DRIVER_GATE, DEVICE_TOKEN, SEAL_FLOWS, CLIENT_POLICY).
- **Status:** Lifecycle + reassign + seal flows `E2E_SSMR_GREEN` (PX4-A3, PX9-B, PX-PAY-003, PX-REAS-004). **PL-1/2/3 closed** (2026-06-15): API path audit green, notification inbox + exceptions panels on all clients, client-policy banners (terminal/Android/iOS), SSMR `PX_E2E_PAYLOAD_CLIENT_POLICY_OK`, payloader JWT refresh (`PX_E2E_PAYLOAD_AUTH_REFRESH_OK`) — see `context/PAYLOAD_PHASE.md`. **PL-4 closed** (2026-06-15): manifest KPI headers (Android tactical grid, iOS existing), connection/queue chrome on terminal sidebar, multi-truck batch seal UI synced terminal/Android/iOS — see `context/PAYLOAD_PHASE.md`. **Open:** Firebase phone OTP UI (PIN remains production path).
- **Phase:** Any manifest/payload change moves the entire payload row + factory + supplier oversight + driver gate consumers.

---

## 3. Phased Execution Roadmap (Smart, Dependency-Respecting, ~90 Days or as Needed)

Phases are **vertical clusters**. Each phase = backend E2E (SSMR first) + all affected role apps/clients updated together + full verification + status marks in this doc + main plan + ledgers.

**Current Active (as of 2026-06-14):** Cross-role depth after supplier SP0–SP5 green (Boss-picked role row).

### Phase Closed / Supplier SP5 (Analytics / Treasury / Operations depth) — **E2E_SSMR_GREEN**
- **Status:** Closed. Native analytics KPIs, portal treasury hub snapshot, operations broadcast/bypass on portal + Android/iOS; `PX_E2E_SUPPLIER_OPERATIONS_OK`, `PX_E2E_SUPPLIER_PAYMENT_BYPASS_OK`.

### Phase Closed / PX-DISP-FLEET-2026-06 (Dispatch Capacity, Fleet Guards/Ops, Payload Multi-Seal) — **E2E_SSMR_GREEN**
- **Status:** Closed. SSMR markers `PX_E2E_DISPATCH_CAPACITY_OK`, `PX_E2E_WAREHOUSE_FLEET_MGMT_OK`, `PX_E2E_PAYLOAD_SEAL_FLOWS_OK` green; capacity recs, fleet guards, multi-truck seal paths wired.
- **Exit met:** Role-row clients show capacity warnings; fleet guards block bad assigns; multi-truck seal works with driver gate + supplier visibility.

### Phase Closed / PX-REPLEN-2026-06 (Replenishment + Supply + Transfers + Insights Durability) — **E2E_SSMR_GREEN**
- **Status:** Closed. WH1–WH2 replenishment durability, `PX_E2E_REPLENISH_OK`, `PX_E2E_REPLENISH_COLOCATE_OK`, `PX_E2E_WAREHOUSE_REPLENISHMENT_OK` green.

### Subsequent Phases (example order; adjust live):
- ~~Analytics/Insights/Treasury depth (supplier native + portal)~~ — **E2E_SSMR_GREEN** (SP5).
- ~~Exception depth (supplier broadcast/payment-bypass)~~ — **E2E_SSMR_GREEN** (`PX_E2E_SUPPLIER_OPERATIONS_OK`, `PX_E2E_SUPPLIER_PAYMENT_BYPASS_OK`).
- AI recs consumer wiring + remaining native analytics depth (factory/warehouse if gaps).
- Full ai-worker consumers (TopicFreezeLocks, any remaining). ~~notification inbox surfaces~~ **WIRED**. ~~TopicFreezeLocks consumer~~ **WIRED** (see `FACTORY_PHASE.md` PX-FREEZE-*).
- Platform/client-policy + version gating + safe update deferral across all native + web (if not 100%).
- Performance/hardening/observability cutover prep (load certs, chaos, DR).
- Any "Candidates" approved by Boss (see below).

**General phase template (use for all):**
1. Read this Vegetable Plan + main context/plan.md + relevant gap/ledger + diagrams.
2. Backend changes (RW txn + outbox + invalidate + structured logs + trace_id).
3. Update packages/types + api-client.
4. Update **all** affected clients for the roles (no partial role rows).
5. Add/enhance SSMR markers + focused tests.
6. Run full verification gates.
7. Update statuses everywhere + commit message references anchors.
8. If new cross-role interaction discovered: add to obligations and re-execute affected prior surfaces if needed.

---

## 4. Gaps Identified in Current Audit (2026-06-14)
- **Real functional gaps (non-scaffold):** XLSX discovery in async worker (CSV/TSV + local/GCS path **E2E_SSMR_GREEN** via `apps/ai-worker/import_worker.go` + `IMPORT_LOCAL_FILE_ROOT`).
- **Scaffold comments:** Expected and documented (bootstrap + domain repos). Not bugs — dev/SSMR path. Strict mode + cloud cutover removes them.
- **UI/TO DOs in clients:** Mostly dev conveniences (recaptcha TODOs in web logins, Firebase SPM comments in retailer iOS) — non-blocking for core flows.
- **Intentional v1 deltas (P2, do not close unless Boss directs):** Supplier portal depth vs full pegasus ~59 (some ops portal-only); no Rust sidecar; thinner native vs portal for supplier; FCM mainly driver+retailer; payme/click scaffolds.
- **Missing apps?** None — all six roles have their declared surfaces present and substantially wired (see ls + parity-ledger). supplier-app-desktop appears as Tauri shell or thin redirect.
- **Parity with pegasus UI/layouts:** High per existing audits (2026-06). pegasusX often at parity or ahead (additive live connection UX, trails, adaptive layouts). When touching, replicate/extend via ui-kit + native mirrors. No wholesale rewrite.

**No unauthorized features introduced in this plan creation.** All items pulled from existing master plan, parity ledger, role matrix, recent code changes, and Boss directive for completeness.

---

## 5. Candidates for Discussion (Before Any Implementation)
(Features/patterns/arch from pegasus reference or elsewhere that look useful for single-tenant but not yet in pegasusX plan/ledgers.)
- [List will grow during execution; e.g. "Pegasus advanced H3 compaction helpers for geo-report if they reduce payload size further for single-tenant maps" — discuss first.]
- Any new dispatch heuristic, treasury split, or AI insight not already in pegasusX replenishment/analytics.
- Open-source tools or best practices (e.g. better client-side H3 lib, specific map render optimizations, observability sidecars) — evaluate for pegasusX cloud cost/SLO profile.

**Rule:** Add here during any phase audit. Do not code until Boss explicit approval + plan update.

---

## 6. How to Use This Plan (Execution Protocol)
- Before any work: read this file + main context/plan.md.
- Pick a phase (or sub-anchor). Confirm no blockers.
- Execute backend E2E (docker SSMR green, new markers) **before** or **with** client changes.
- Touch **all** apps in affected role rows (and cross-role observers) in the coordinated set.
- Replicate good pegasus UI/layouts/components only into pegasusX/ (ui-kit first, then compose).
- If a change implies a new sync obligation (new event, new WS consumer, new role confirmation step), implement the full chain or explicitly defer with Boss sign-off and ledger entry.
- After batch: run gates, update **all** status markers in this file + linked docs, add checkpoint.
- "After execution you should mark whether it's implemented or no" — done here via the status model.

**Boss:** This is the living granular vegetable breakdown you asked for. It starts from backend features, maps them to every role/app that must consume/participate for E2E, calls out UI replication targets, sync requirements, single-tenant notes, and phases with dependency respect. Ready for the next phase execution directive (current dispatch/fleet/payload work is the natural first to close).

All future coding stays in pegasusX/. pegasus/ only for reference reads.

Status of this plan creation: initial version complete from audit. Gaps section reflects reality (mostly durability + in-flight work + intentional deltas). No new un-discussed features added.

Next step on your word, Chief. We can execute the current dispatch phase to green, or adjust the roadmap.