# pegasusX Role-Row Parity Matrix

> **Canonical cross-role spec:** [`FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`](./FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md) — use this matrix for screen-level parity; use the master plan for end-to-end flows, comms, and verification gates.

Last updated: 2026-07-01 (PX91 digital brain sync). Canonical reference: `pegasus/`. Delivery tree: `pegasusX/`.

## Summary

| Role | pegasusX clients | Backend routes | Production v1 capability | UI parity (vs Pegasus) | E2E (SSMR) |
|------|------------------|----------------|--------------------------|------------------------|------------|
| SUPPLIER | supplier-portal, native iOS/Android | supplierroutes, catalogroutes, promotionroutes, returnsroutes, payloaderroutes (manifest lifecycle) | Full ops spine: onboarding, order vet, dispatch preview/execute (CEO override), topology CRUD, catalog-first pricing, inventory import wizard, org-fleet seeding, treasury, analytics, returns; orders+dispatch hub on native | **Wired** — ~43 portal routes + native parity; pegasus multi-tenant extras (CRM, staff, country-overrides) out of scope; exceptions routes exist but removed from primary nav (2026-06-17) | Full SSMR e2e incl. payment + factory |
| RETAILER | desktop, iOS, Android | retailerroutes, orderroutes, catalogroutes | Order lifecycle (manual pre-order Standard vs Scheduled checkout, preorder confirm/edit, **delivery proposal review**, request-cancel), setup wizard, insights dismiss, catalog/search, tracking + dock, unified checkout, **real-time orderable caps** (`orderable_quantities`) | **Wired** — desktop richest; mobile Deliveries hub; Midnight Guard + `PRE_ORDER_*` + `PRE_ORDER_DATE_*` events; quantity steppers clamp to preview caps | Register, order create, tracking, `PX_E2E_MANUAL_PREORDER_OK`, `PX_E2E_DELIVERY_PROPOSAL_OK`, `PX_E2E_CONCURRENT_STOCK_REJECT_OK`, `PX_E2E_CATALOG_OK`, retailer SSMR markers |
| DRIVER | Android, iOS | driverroutes, orderroutes, telemetryroutes | Full delivery edges + reorder (PX12-B); planned route geometry + turn-by-turn + off-route reroute (`GET /v1/fleet/route/{routeID}/geometry`); maps + WS (PX12-F); Firebase phone OTP | **Wired** — live-ops + planned/breadcrumb map overlays; phone OTP + PIN dev on Android/iOS | Telemetry; shop-closed; negotiation; driver edges E2E; `PX_E2E_DRIVER_FIREBASE_OTP_OK` when test token set |
| WAREHOUSE | portal, Android, iOS | warehouseroutes | Pre-order hub (**calendar propose**), stock commitments drill-down, ops settings (lead window, line limits, delivery fee tiers, express toggle), reject/cancel anytime, **atomic stock reservation at create** (REJECT policy) | **Wired** — `/preorders` propose/reject, `/stock-commitments`, checkout policy enforcement, `PRE_ORDER_*` + `PRE_ORDER_DATE_*` WS | `PX_E2E_WAREHOUSE_OPS_POLICY_OK`, `PX_E2E_CHECKOUT_LINE_LIMIT_OK`, `PX_E2E_DELIVERY_FEE_PREVIEW_OK`, `PX_E2E_MANUAL_PREORDER_OK`, `PX_E2E_DELIVERY_PROPOSAL_OK`, `PX_E2E_CONCURRENT_STOCK_REJECT_OK`, `PX_E2E_CHECKOUT_POLICY_GRACE_OK`, `PX_E2E_ORDER_ACCEPTANCE_CLOSED_OK`, dispatch SSMR markers |
| FACTORY | portal, Android, iOS | factoryroutes | Manifests + supply requests + dispatch (PX12-J); Firebase phone OTP | Portal 13 routes; insights P1; manifest exception inbox + transfer create P2; manifest lifecycle + staff detail P3; supply-request transitions + payload override cross-manifest rebalance P4; loading-bay transfer filter/detail/dispatch + transfer state machine P5 on all factory clients; phone OTP + password dev on portal/Android/iOS | Insights + lifecycle + staff + supply transition + payload override rebalance + transfer transition + loading bay + dispatch + exceptions + transfer create SSMR markers; `PX_E2E_FACTORY_FIREBASE_OTP_OK` when test token set |
| PAYLOAD | terminal, iOS, Android | payloaderroutes, platformroutes, returnsroutes | Manifest lifecycle + reassign + device-token + Firebase OTP + manifest barcode scan | **Wired** — lifecycle APIs; manifest-exception + inbound returns idempotency replay; phone OTP + PIN dev on all clients; catalog barcode checklist + inject scan; inbound returns EAN on all three | SSMR payload + lifecycle + reassign + driver gate + payloader device-token sub-markers |

## SUPPLIER — screen map

### Onboarding & auth

| Pegasus (`supplier-portal`) | pegasusX portal | pegasusX native | Backend | Status |
|-----------------------------|-----------------|-----------------|---------|--------|
| `/auth/register` | `/auth/register` | `RegisterScreen` | `POST /v1/auth/supplier/register` | Wired |
| `/auth/login` | `/auth/login` | `LoginScreen` | `POST /v1/auth/supplier/login` | Wired |
| `/setup/business` | `/setup/business` | `BusinessSetupScreen` | `POST /v1/supplier/business/setup` | Wired |
| `/setup/billing` | `/setup/billing` | `BillingScreen` (+ Android More) | `POST /v1/supplier/billing/setup`, `POST /v1/supplier/configure` | Wired — acceptor `SUPPLIER` or `WAREHOUSE` |

### Primary & operations

| Pegasus | pegasusX portal | pegasusX native | Backend | Cross-role | Status |
|---------|-----------------|-----------------|---------|------------|--------|
| `/supplier/dashboard` | `/(portal)/dashboard` | `DashboardScreen` | `GET /v1/supplier/dashboard` | Retailer order KPIs; driver map snippet | Wired |
| `/supplier/orders` | `/(portal)/orders` | `OrdersHubScreen` (Orders \| Dispatch tabs) | `GET /v1/supplier/orders`, `POST /v1/supplier/orders/vet` | **Retailer** places; supplier vets | Wired |
| `/supplier/dispatch` | `/(portal)/dispatch` | Orders hub Dispatch tab + `DispatchPreviewScreen` | `GET/POST /v1/supplier/dispatch/{preview,execute}` | **Warehouse** scope; **Driver** assignment; fingerprint drift warning vs warehouse plan | Wired — CEO override execute; PX-ECS-4F mismatch banner |
| `/supplier/manifests` | `/(portal)/manifests` | `ManifestsScreen` | `GET /v1/supplier/manifests` | **Driver**, **Payload**, **Factory** gate | Wired |
| Manifest detail | `/(portal)/manifests/[id]` | `ManifestDetailScreen` | `POST .../start-loading`, `inject-order`, `seal` | **Payload** lifecycle; **Retailer** orders | Wired |
| Manifest gate exceptions | `/(portal)/manifest-exceptions` | `ManifestExceptionsScreen` | `GET /v1/supplier/manifest-exceptions` | **Payload**, **Warehouse**, **Factory** | Wired — linked from manifests |
| `/fleet`, `/supplier/fleet` | `/(portal)/fleet` | `FleetScreen` + `FleetLiveMapScreen` (publish zone) | `GET/POST /v1/supplier/fleet/*`, `GET /v1/supplier/fleet/live-map`, `GET/POST /v1/supplier/control-tower/zone-overrides` | **Driver** telemetry; **Warehouse** dispatch freeze | Wired — portal + native zone publish (PX90-B5) |
| MEIO network summary | dashboard `MeioNetworkPanel` | `DashboardScreen` / `DashboardView` MEIO card | `GET /v1/supplier/meio/network-summary`, `GET /v1/supplier/replenishment/policies` | **Warehouse** replenishment insights | Wired — PX90-A4/A5 |
| Planning sandbox + S&OP | `PlanningBrainPanel` on analytics | `PlanningBrainSection` on `AnalyticsScreen` / `AnalyticsView` | `GET /v1/supplier/planning/s-and-op`, `POST /v1/supplier/planning/scenarios/run` | Scenario uses driver delivery proofs | Wired — PX90-C4/C5 P2 native |
| Forecast confidence card | `ForecastConfidenceCard` on dashboard + analytics | `ForecastConfidenceView` on dashboard, MEIO/demand screens | `GET /v1/supplier/analytics/demand/today` (+ baseline fields) | **Warehouse** replenishment reads | Wired — PX91-A3; API-first mapper on all native demand surfaces |
| Planning settings (seasonal) | `/(portal)/settings/planning` | `PlanningSettingsScreen` | `GET/POST /v1/supplier/planning/seasonal-overrides` | Isolated forecast blocks | Wired — PX91-A2/A4 |
| Knowledge graph | `/(portal)/analytics/knowledge-graph` | `KnowledgeGraphScreen` | `GET /v1/supplier/knowledge-graph` | Drivers, retailers, orders as nodes | Wired — PX91-B3 |
| Replenishment policies | `/(portal)/operations/replenishment-policies` | `ReplenishmentPoliciesScreen` | `GET /v1/supplier/replenishment/policies` | **Warehouse** touchless path | Wired — PX90-A5 |
| Planning brain (native hub) | — | `PlanningBrainScreen` | planning APIs above | Scenario + S&OP reconcile | Wired — PX91 portal parity on native |
| Promo P&L sandbox | promotions + planning APIs | `PromotionsScreen` P&L swipe | `POST .../planning/promotions/simulate`, `GET .../performance` | **Retailer** checkout impact (read-only) | Wired — PX91-C2/C3; portal + native sandbox (PX-ECS-3A, 4D) |
| Signal ingest ops | planning settings panel | — | `GET .../planning/signals/status`, `POST .../signals/ingest` | Kafka `planning.signal.ingest.v1` → baseline when scoped | Wired — PX-ECS-3E; ingest affects forecast when `product_id` + `warehouse_id` present |
| Sparsity gate | demand analytics | demand screens | `GET .../planning/sparsity-check` | Blocks hallucinated forecast | Wired — PX91-A1 |
| Fleet orders | `/(portal)/fleet/orders` | `FleetOrdersScreen` | `GET /v1/supplier/fleet/orders` | **Driver** ↔ **Retailer** order | Wired |
| Operations / empathy | `/(portal)/operations` | `OperationsScreen` / `OperationsView` | `GET /v1/supplier/empathy/adoption`, `POST /v1/supplier/broadcast`, `POST /v1/supplier/replenishment/trigger`, `POST /v1/supplier/orders/payment-bypass` | Broadcast fans to **supplier + driver/retailer/payload/warehouse/factory** WS rooms by `role`; replenishment → **Warehouse**; bypass → **Retailer** payment | Wired — native More hub + tablet rail |
| Activity feed | dashboard snippet | `ActivityScreen` | `GET /v1/supplier/activity` | Cross-ecosystem events | Wired — native page; portal dashboard only |
| Exceptions hub | `/(portal)/exceptions` | route exists; **not in nav** | `GET /v1/supplier/exceptions` | Multi-role | Wired — deep link only (removed from nav 2026-06-17) |
| Shop closed | `/(portal)/exceptions/shop-closed` | route exists; **not in nav** | `GET/POST /v1/supplier/shop-closed/*` | **Driver** reports → **Retailer** | Wired — deep link only |
| Early route complete | `/(portal)/exceptions/early-complete` | route exists; **not in nav** | `POST /v1/supplier/route/approve-early-complete` | **Driver** | Wired — deep link only |
| Negotiations | `/(portal)/exceptions/negotiations` | disabled stub | `GET /v1/supplier/negotiations/pending` (empty) | **Driver** | **Disabled** ecosystem-wide |

### Catalog, inventory & pricing

| Pegasus | pegasusX portal | pegasusX native | Backend | Cross-role | Status |
|---------|-----------------|-----------------|---------|------------|--------|
| `/supplier/inventory` | `/(portal)/inventory` | `InventoryScreen` | `GET/PATCH /v1/supplier/inventory`, `GET .../audit` | Stock for **Retailer** orders | Wired |
| `/inventory/import` | `/(portal)/inventory/import` | `InventoryImportScreen` | `POST/GET /v1/supplier/inventory/imports/*` | Backend idempotency on create/ingest/approve/apply; **Warehouse**-scoped rows; WS to warehouse hub | Wired |
| `/supplier/catalog` | `/(portal)/catalog` | `CatalogScreen` | `GET/POST /v1/catalog/products`, categories | **Retailer** browse/order | Wired |
| Catalog detail | `/(portal)/catalog/[productId]` | `CatalogDetailScreen` | `PUT /v1/catalog/products/{id}`, upload ticket | **Retailer** product view | Wired |
| `/supplier/pricing` | `/(portal)/pricing` → `/(portal)/pricing/[productId]` | `PricingScreen` + `ProductPricingDetailView` (iOS) | `GET/PATCH /v1/supplier/pricing/rules` | **Retailer** list prices | Wired — catalog-first (2026-06-17) |
| `/supplier/pricing/retailer-overrides` | `/(portal)/pricing/retailer-overrides` | `RetailerOverridesScreen` | `GET/POST/DELETE /v1/supplier/pricing/retailer-overrides` | **Retailer**-specific prices | Wired |
| Promotions | `/(portal)/promotions` | `PromotionsScreen` (+ P&L simulate) | `GET/POST/PATCH/DELETE /v1/supplier/promotions`, `POST .../planning/promotions/simulate` | **Retailer** checkout discounts | Wired |

### Network / topology

| Pegasus | pegasusX portal | pegasusX native | Backend | Cross-role | Status |
|---------|-----------------|-----------------|---------|------------|--------|
| Topology editor | `/(portal)/topology` | `TopologyScreen` | `GET/PUT /v1/supplier/topology` | **Warehouse** + **Factory** nodes; factory `H3Cell` backfill on PUT | Wired |
| `/supplier/factories` | `/(portal)/factories` | `FactoriesScreen` (topology PUT) | `PUT /v1/supplier/topology` | **Factory** app staff via org-fleet | Wired — empty-state create CTA (2026-06-17) |
| `/supplier/warehouses` | `/(portal)/warehouses` | `WarehousesScreen` (topology PUT) | `PUT /v1/supplier/topology` | **Warehouse** app; dispatch origin | Wired — empty-state create CTA (2026-06-17) |
| `/supplier/delivery-zones` | `/(portal)/delivery-zones` | `DeliveryZonesScreen` | `GET /v1/supplier/topology` (coverage radii) | **Retailer** delivery eligibility | Wired |
| Supply lanes | `/(portal)/supply-lanes` | `SupplyLanesScreen` | `GET /v1/supplier/supply-lanes` | **Factory** ↔ **Warehouse** lanes | Wired |
| `/supplier/geo-report` | `/(portal)/geo-report` | `GeoReportScreen` | `GET /v1/supplier/supply-lanes` (H3) | Network planning | Wired |

### Intelligence & finance

| Pegasus | pegasusX portal | pegasusX native | Backend | Cross-role | Status |
|---------|-----------------|-----------------|---------|------------|--------|
| `/supplier/analytics` | `/(portal)/analytics` | `AnalyticsScreen` | `GET /v1/supplier/analytics/{velocity,revenue,demand/today}` | **Retailer** demand | Wired |
| Demand forecast | `/(portal)/analytics/demand` | `DemandHistoryScreen` | `GET /v1/supplier/analytics/demand/history` | **Retailer** patterns | Wired |
| `/ai/recommendations` | `/ai/recommendations` | `AIRecommendationsScreen` | `GET/POST /v1/supplier/ai/recommendations` | Ops suggestions | Wired |
| `/treasury/*` | `/(portal)/treasury` | `TreasuryHubScreen` | `GET /v1/supplier/earnings`, payment APIs | **Retailer** settlements | Wired |
| `/payments` | `/payments` | `PaymentsScreen` | `GET /v1/payment/settlement/authority`, mismatches | Payment gateway | Wired |
| `/earnings` | `/earnings` | `EarningsScreen` | `GET /v1/supplier/earnings` | **Retailer** order revenue | Wired |
| Ledger | merged in earnings | `LedgerScreen` | `GET /v1/payment/ledger` | **Retailer** payments | Wired — native standalone |
| Reconciliation | `/(portal)/reconciliation` | `ReconciliationScreen` | `GET /v1/payment/reconciliation/mismatches` | **Retailer** payment integrity | Wired |
| Chargebacks | merged in `/earnings` | `ChargebacksScreen` | `POST /v1/payment/chargeback`, reversal | **Retailer** disputes | Wired — native standalone |

### Org, account & settings

| Pegasus | pegasusX portal | pegasusX native | Backend | Cross-role | Status |
|---------|-----------------|-----------------|---------|------------|--------|
| `/org-fleet` | `/org-fleet` | `OrgFleetScreen` | `GET/POST/PATCH/DELETE /v1/supplier/org/members`, `POST /v1/supplier/fleet/{drivers,vehicles}` | Seeds **Warehouse/Factory/Payload** admins + **Driver** fleet | Wired |
| `/supplier/profile` | `/(portal)/profile` | `ProfileScreen` | `GET/PUT /v1/supplier/profile` | — | Wired |
| Business setup | `/(portal)/setup/business` | `BusinessSetupScreen` / `BusinessSetupView` | `POST /v1/supplier/setup/business` | Supplier registration gate | Wired — native More hub + onboarding |
| `/supplier/returns` | `/(portal)/returns` | `ReturnsScreen` | `GET /v1/supplier/returns`, `POST .../resolve` | **Retailer** returns; **Driver** manifest | Wired |
| Notifications | top-bar panel | `NotificationsScreen` | `GET /v1/user/notifications`, `POST .../read` | Cross-role alerts | Wired |
| Network pulse | `NetworkPulsePanel` on dashboard | `SupplierPulseStrip` on dashboard | `GET /v1/supplier/pulse` | Merged inbox + transitions + activity | Wired — portal `@pegasusx/pulse-ui`; native horizontal strip |
| Handoff inbox cards | notifications panel | — | `MetadataJson` on Notifications | Dispatch/manifest/preorder handoffs | Wired — `handoff_metadata` |
| Explain errors | dispatch + API 4xx | — | `platform/explain_status` | Human guidance on operational errors | Wired — `@pegasusx/explain-ui` |

### SUPPLIER — cross-role touchpoint matrix

| Other role | Supplier pages/features that interact |
|------------|----------------------------------------|
| **Retailer** | Orders (vet), catalog, pricing, promotions, retailer overrides, returns, shop-closed resolve, payment bypass, chargebacks, demand analytics, broadcasts |
| **Driver** | Org-fleet CRUD, fleet live map, fleet orders, dispatch execute, manifest visibility, shop-closed, early-complete, order live location, broadcasts |
| **Warehouse** | Topology/warehouses, dispatch origin + scope, inventory import per warehouse, delivery zones, supply lanes, replenishment trigger, org-fleet admins, billing acceptor option, manifest loading gate |
| **Factory** | Topology/factories, supply lanes, org-fleet admins, co-located warehouses, manifest gate exceptions |
| **Payload** | Manifest lifecycle (shared routes), manifest gate exceptions inbox, org-fleet staff seeding, broadcasts to PAYLOAD role |
| **Treasury** | Billing setup, earnings, ledger, payments authority, reconciliation, chargebacks, payment bypass |

**Topology dependency:** SUPPLIER creates warehouse/factory topology only. WAREHOUSE_ADMIN owns day-to-day fleet CRUD, dispatch locks, and capacity overrides. PAYLOAD seals per truck. DRIVER consumes assignment via profile + WS hubs.

## Native shell parity (collapsible icon rail)

All six native role apps use a shared monochrome design system (`packages/mobile-ios-design`, `packages/mobile-android-design`):

| Role | Tablet / regular width | Phone / compact | Theme |
|------|------------------------|-----------------|-------|
| RETAILER | Collapsible sidebar 88↔280pt (iOS) / rail (Android) | Bottom tabs + overflow | B&W; `dynamicColor = false` on Android |
| WAREHOUSE | `CollapsibleSidebar` / `PegasusCollapsibleRail` | 4-tab bottom bar + More | `PegasusMonochromeTheme` |
| FACTORY | `FactoryAdaptiveShell` + rail | 4-tab + More hub | `PegasusMonochromeTheme` |
| SUPPLIER | `CollapsibleSidebar` / `PegasusCollapsibleRail` (5 groups: Primary, Operations, Intelligence, Network, Account) | 4-tab bottom bar (Dashboard, Orders, Fleet, More) + Orders hub (Orders \| Dispatch) | `PegasusMonochromeTheme` |
| PAYLOAD | Collapsible truck list (icon rail ↔ full list) | List-detail scaffold | `PegasusMonochromeTheme` (Android); `TermTheme` (iOS) |
| DRIVER | Collapsible rail (4 tabs) | Bottom tabs + map overlay | B&W; `dynamicColor = false` |

Rail never fully hides on tablet — collapsed state keeps an 88dp/pt icon column.

## Cool Features Program (Waves 0–3)

| Feature | Supplier | Warehouse | Retailer | Driver | Payload | Factory |
|---------|----------|-----------|----------|--------|---------|---------|
| Pulse timeline | portal dashboard | portal dashboard | desktop tracking (optional) | Android + iOS home strip | Android + iOS home strip | portal dashboard |
| Explain status banners | API 4xx | dispatch page | tracking errors | manifest-gate (Android/iOS) | seal errors + batch row explain (Android/iOS/terminal) | loading-bay, transfers, manifest detail, exceptions |
| Handoff inbox cards | notifications | notifications | notifications | native inbox + primary_link nav | native inbox + primary_link nav | notifications |
| Exception weather map | portal fleet tab + native fleet | — | — | — | — | — |
| Broadcast templates | portal operations + native | portal operations + native (depot-scoped, custom saved) | — | — | — | — |
| Override impact preview | portal + native retailer overrides | portal operations read-only preview | — | — | — | — |
| Production lane board | — | — | — | — | — | portal supply-requests board + native |
| Fulfill mode helper | — | — | — | — | — | portal + native FULFILL sheet |

## Intentional deltas

- Single deploy defaults to one seeded supplier; `MAX_SUPPLIERS` (default 10) allows additional registrations.
- No Rust optimizer sidecar in pegasusX v1 (dispatch preview + ai-worker only).
- Firebase bearer optional; SSMR uses JWT bearer + cookie for smoke auth.

## Engine parity (PX11-E1 — critical paths)

| Path | Backend | WS fan-out | FCM fallback | Native clients |
|------|---------|------------|--------------|----------------|
| Order lifecycle | `orderroutes` + outbox | All role hubs | Driver + retailer | All retailer + driver apps |
| Payment / webhooks | `paymentroutes` + `webhookroutes` | Supplier + retailer | Retailer tokens | Desktop + mobile |
| Shop-closed | `order/shop_closed.go` | Dispatcher | Yes | Driver wait-state VMs |
| Negotiation | `order/negotiation.go` | Dispatcher | Yes | **DEFERRED v2** — `order/negotiation_disabled.go` returns 410; SSMR `PX_E2E_NEGOTIATION_SKIPPED` |
| Manifest gate | `payloaderroutes` + Spanner manifests | `MANIFEST_*` events | Payload/factory hubs | Payload + factory row |
| Client version policy | `GET /v1/platform/client-policy` | `SYSTEM_APP_OUTDATED` on WS | N/A | Driver + supplier native |
| Supplier realtime | `supplier:` WS room | Kafka dispatcher | N/A | Portal + **PX11** native WS |

## Feature parity (PX11-E2 — UI phase, portal depth > native)

| Surface | Pegasus reference depth | pegasusX native row | Notes |
|---------|-------------------------|---------------------|-------|
| Supplier portal | ~59 routes (pegasus multi-tenant) | Portal + native full single-tenant parity | Native register, vet, inventory, overrides, chargebacks, treasury hub |
| Retailer desktop | Full procurement | Desktop richest; mobile tracking-first | Intentional until E2 |
| Factory / warehouse | Full portal | Portal + mobile ops dashboards | Treasury depth portal-only |

## Verification commands

```bash
cd pegasusX && make test-ssmr-infra
cd pegasusX && make validate-launch-readiness
cd pegasusX && bash scripts/parity/role_row_contract_check.sh
cd pegasusX && make backend-build
```

## Barcode catalog + inbound return gate parity (2026-06-15)

Policy: [`pegasus/docs/BARCODE_SCANNING.md`](../../pegasus/docs/BARCODE_SCANNING.md) — EAN/GTIN only at return gate; supplier catalog `Products.Barcode` required for scan match.

| Surface | Catalog EAN capture | Inbound gate scan | History tab | EAN on rows | Idempotency-Key | Offline scan queue |
|---------|--------------------|--------------------|-------------|---------------|-----------------|-------------------|
| supplier-portal `/catalog` | Manual create + inline edit + checksum validation | — | — | — | — | — |
| supplier-app Android/iOS | Camera + manual (`CatalogBarcodeField`) | — | — | — | — | — |
| payload-terminal inbound | — | Manual + camera | Yes | Yes | Yes | Yes (SecureStore queue) |
| payload-app Android/iOS | — | Camera + manual | Yes | Yes | Yes | Yes (Android VM queue; iOS `OfflineQueue`) |
| warehouse-portal `/returns` | — | Wedge/manual + Enter | Yes | Yes | Yes | — (desktop wedge) |
| warehouse-app Android/iOS | — | Camera + manual | Yes | Yes | Yes | — |

Shared primitives: `@pegasusx/validation` `normalizeEanBarcode`, `@pegasusx/api-client` `warehouseInboundScanKey` / `payloadInboundScanKey`, Android `mobile-android-barcode-scanner`, iOS `mobile-ios-barcode`.

## Cross-platform operational parity (2026-06-15)

Per-role surfaces that must stay wired end-to-end (portal + native + terminal where applicable):

| Capability | SUPPLIER | RETAILER | DRIVER | WAREHOUSE | FACTORY | PAYLOAD |
|------------|----------|----------|--------|-----------|---------|---------|
| Returns / dispute resolve | portal + Android + iOS | — | — | gate (see above) | — | gate inbound |
| Fleet live map | portal + Android + iOS | — | — | portal + Android + iOS | — | — |
| Dispatch preview/execute | portal + Android + iOS | — | — | portal + Android + iOS | portal + Android + iOS | — |
| Handoff timeline (preorder → seal) | — | — | — | portal + Android + iOS dispatch | portal + Android + iOS loading-bay | — |
| Supply requests | — | — | arrive (Android + iOS) | portal + Android + iOS | portal + Android + iOS | — |
| Manifest lifecycle | portal read | — | — | — | portal + Android + iOS | terminal + Android + iOS |
| Reassign (redispatch) | — | — | — | — | — | terminal + Android + iOS (`/v1/payloader/reassign-order`) |
| Treasury / invoices | portal | desktop | — | portal + Android + iOS | portal | — |
| Notifications inbox | portal + Android + iOS (dashboard bell on both native) | mobile + desktop | Android + iOS | portal + Android + iOS | portal + Android (+ iOS sheet) | Android + iOS |
| Client policy banner | all clients (global shell) | all clients (global shell) | all clients (global shell) | all clients (global shell) | portal + native | all clients (global shell) |
| Idempotency on mutations | dispatch, resolve, broadcast, payment-bypass, import wizard, topology PUT, replenishment trigger | checkout, orders, profile, setup, supplier favorites, reject-ai, edit-preorder | delivery edges, amend, transition, supply arrive | dispatch, supply, inbound gate, order delay/reject/overflow | manifest, transfer | seal, reassign, inbound |
| Dock inbound queue (supplier-grouped, QR reveal) | — | desktop + Android + iOS | — | — | — | — |

**Intentional portal-only deferrals (v1):** supplier empathy adoption depth on native; warehouse supply forecast create form depth on native (create from Dispatch tab); factory iOS analytics/exceptions as dashboard sheets not tabs.

**Intentional nav removals (2026-06-17):** supplier exceptions hub, shop-closed, and early-complete removed from portal sidebar and native More/rail — routes and APIs remain for deep links and SSMR.

**Recently closed gaps (2026-07-02):** Supplier native More hub — chargebacks, business setup, and operations wired end-to-end on Android + iOS (removed web portal handoff section; tablet rail includes business setup).

**Recently closed gaps (2026-07-01): PX-ECS ecosystem sync** — promo P&L on supplier native (4D); dispatch fingerprint mismatch warning supplier portal + native (4F); handoff timeline on warehouse/factory dispatch + loading-bay **portal + Android + iOS** (4E); supplier dashboard forecast confidence on native (2E polish); signal ingest ops panel + baseline projection truth in docs (3E, 3G); planning outcomes + traceability panels (3B, 3D). **PX91 digital brain** — sparsity gate, seasonal overrides, signal ingest→Kafka, promo P&L sandbox, shadow deploy on ai-worker; supplier portal + native PlanningBrain, confidence cards, EKG, replenishment policies; warehouse/factory `ForecastConfidenceView`; retailer sparsity “Insufficient history” badge on prediction cards; session reconcile planning paths; SSMR markers in `e2e_plan90.go`. **pegasus P1** — federated planning read APIs + `/planning` admin page; supplier UI parity **pending** (P2).

**Recently closed gaps (2026-06-29):** **Supplier gap closure (phase 2)** — pricing rules PATCH, inventory PATCH, profile PUT, business setup, configure, retailer price overrides, and promotion mutations honor Redis idempotency replay; contract keys wired on supplier-portal + Android/iOS (phase 1 import/topology/broadcast unchanged). **Retailer gap closure** — `PUT /v1/retailer/profile`, `POST /v1/retailer/setup`, supplier favorite add/remove, `POST /v1/retailer/orders/reject-ai`, and `POST /v1/orders/edit-preorder` honor Redis idempotency replay on the retailer service (checkout/cancel/confirm paths already guarded in `order/`); `retailerProfileUpdateKey`, `retailerSetupKey`, `retailerRejectAIKey`, `retailerEditPreorderKey` wired on desktop settings/setup + Android/iOS profile, onboarding, suppliers, and order flows. **Supplier ecosystem gap closure** — import wizard backend idempotency; returns resolve idempotency + `outbox.EmitJSON`; cross-role broadcast WS fanout; factory `H3Cell` on topology PUT; deterministic idempotency keys on portal/Android/iOS (org-fleet, import, chargebacks); dispatch partial-commit structured error; vet order `Deprecation` header (route retained for SSMR). **Warehouse gap closure** — `PATCH /v1/warehouse/ops/location` now uses RW txn + `H3Cell` + idempotency replay + outbox `WAREHOUSE_LOCATION_UPDATED`; portal settings/location + native location PATCH send deterministic idempotency keys; warehouse-portal `Suspense` boundary for `useSearchParams` pages. **Driver gap closure** — `POST/PATCH /v1/driver/availability` and `PATCH /v1/orders/{id}/state` honor Redis idempotency replay; `driverAvailabilityKey` wired on Android/iOS session end / availability mutations. **Payload gap closure** — `POST /v1/payload/manifest-exception` now uses Redis idempotency replay (clients already sent keys); shared `returns` inbound scan/confirm handlers honor `payloadInboundScanKey` / `warehouseInboundScanKey` replay; `payloadManifestExceptionKey` exported from `@pegasusx/api-client`. **Factory gap closure** — `PATCH /v1/factory/supply-requests/{id}` (incl. FULFILL transfer create) and legacy `POST …/accept` honor Redis idempotency replay; `PATCH /v1/factory/ops/location` mirrors warehouse hardening (RW txn + `H3Cell` + outbox `FACTORY_LOCATION_UPDATED` + WS fanout); `factorySupplyRequestTransitionKey`, `factorySupplyRequestAcceptKey`, `factoryOpsLocationKey` wired on factory-portal + Android + iOS.

**Recently closed gaps (2026-06-17):** supplier topology create UX (empty-state CTA + `PUT /v1/supplier/topology` on portal/Android/iOS); catalog-first pricing with per-product detail; orders+dispatch combined hub on native; Android rail full section parity (`ORG_FLEET`, treasury suite, `INVENTORY_IMPORT`, `RETAILER_OVERRIDES`); `PegasusCollapsibleRail` expanded-drawer layout fix (supplier/warehouse/factory tablet nav).

**Recently closed gaps (2026-06-15):** warehouse smart dispatch AUTO commit (portal + Android + iOS) with post-solve capacity modal (accept partial / force); residual truck capacity (`free_volume_vu`, DRAFT/LOADING top-off); dispatch plan Redis cache + `plan_fingerprint`; manual dispatch Apply suggestion on portal; warehouse global client-policy banner (Android/iOS login + nav shell); warehouse Android/iOS AutoUpdater on outdated policy; warehouse native order delay/reject/overflow idempotency; warehouse iOS APNs push registration; warehouse portal driver CRUD + dispatch-lock scope idempotency fixes; driver global client-policy banner (Android/iOS login + nav shell); driver Android/iOS AutoUpdater on outdated policy; driver amend + transition-state idempotency; driver iOS APNs push registration + offline flush on network restore + map WS reconnect reconcile; payload global client-policy banner (Android/iOS/Expo login + app shell); payload Expo push token registration (`EXPO` + login `firebase_token`); payload Android AutoUpdater wired on outdated policy; payload iOS AutoUpdater in main target; retailer dock queue; supplier dispatch route map; supplier idempotency + policy; payload seal-completed idempotency alignment across terminal + iOS.

## SSMR fleet / dispatch / payload feature IDs (2026-06-14)

| ID | Capability | Owner row | Backend | Clients | E2E marker |
|----|------------|-----------|---------|---------|------------|
| PX-FLEET-001 | Warehouse driver+vehicle CRUD, assign/re-sign, active-route guard | WAREHOUSE_ADMIN | `warehouse/ops_portal.go`, `warehouse/fleet_guards.go`, `warehouse/fleet_ops.go` | warehouse-portal `/drivers`, native ops | `PX_E2E_WAREHOUSE_FLEET_MGMT_OK` |
| PX-DISP-002 | Manual dispatch capacity warnings + suggested unselect + force audit | WAREHOUSE_ADMIN | `warehouse/dispatch_execute.go`, `dispatch/capacity_recommend.go` | warehouse-portal + warehouse Android/iOS dispatch | `PX_E2E_DISPATCH_CAPACITY_OK` |
| PX-DISP-003 | Smart dispatch AUTO commit + residual fleet VU + plan cache + accept_partial | WAREHOUSE_ADMIN | `warehouse/dispatch_execute.go`, `warehouse/dispatch_plan_cache.go`, `manifest/store.go` | warehouse-portal + warehouse Android/iOS dispatch Smart Dispatch button | preview `plan_fingerprint` + execute `mode:AUTO` |
| PX-PAY-003 | Per-truck seal + aggregate `seal-completed` batch | PAYLOAD | `payload/service.go`, `payloaderroutes` | payload-terminal + payload Android/iOS tablet apps | `PX_E2E_PAYLOAD_SEAL_FLOWS_OK` |
| PX-REAS-004 | Durable payload reassign + recommend | PAYLOAD | `payload/service.go` | payload-terminal | `PX_E2E_REASSIGN_FLOWS_OK` |
| PX-DRV-005 | Driver profile `vehicle_id` + WS assign detection | DRIVER | `driver/service.go`, bootstrap profile lookup | driver Android/iOS profile VM | `PX_E2E_DRIVER_ASSIGN_DETECTION_OK` |

### Topology dependency (SUPPLIER root)

SUPPLIER creates warehouse/factory topology only (`PUT /v1/supplier/topology`). WAREHOUSE_ADMIN owns fleet CRUD, dispatch, and capacity overrides. FACTORY_ADMIN fulfills supply requests and loading-bay transfers. PAYLOAD seals per truck then batch-activates drivers. DRIVER consumes assignment via profile + hubs.

Diagrams: `pegasusX/assets/diagrams/pegasusx-supplier-topology-dependency.mmd`, `pegasusx-fleet-dispatch-capacity-flow.mmd`, `pegasusx-payload-seal-multi-truck.mmd`.

## Native realtime refresh contract (2026-06-17)

All pegasusX native role apps share **stale-while-revalidate** over WebSocket — clients never consume Kafka directly.

| Pattern | Android | iOS |
| --- | --- | --- |
| Refresh signal | `*RealtimeSignals.refreshTick` / `reconnectTick` | `*RealtimeHub.refreshEpoch` / `reconnectEpoch` |
| Reload API | `load(silent: Boolean = false)` | `load(silent: Bool = false)` |
| Compose/SwiftUI hook | `RealtimeRefreshEffect` (`mobile-android-design`) | `silentRealtimeRefresh` (`mobile-ios-design`) |
| Loading UI rule | `showFullScreenLoading(loading, hasData)` | `loading && items.isEmpty` |
| **Anti-pattern** | `key(refreshEpoch) { NavHost/Screen }` | `.id("tab-\(refreshEpoch)")` on tabs |

**Manual test matrix (per app):** rapid tab switch (no bounce); background→foreground silent reload; WS event in-place update; pull-to-refresh still shows indicator; cold start shows skeleton.

## UI / motion parity (Phase 4, 2026-06-17)

Shared design packages now own cross-app chrome:

| Concern | Android (`mobile-android-design`) | iOS (`mobile-ios-design`) | Web (`@pegasusx/motion-tokens`) |
| --- | --- | --- | --- |
| Motion | `PegasusMotionTokens`, `PegasusAnim` | `PegasusAnim`, `PegasusMotionDuration` | `duration`, `easing`, `spring`, `motionVariants` |
| Loading / empty / error | `PegasusLoadingState`, `PegasusStatePane`, `PegasusStateKind` | `PegasusLoadingView`, `PegasusErrorView`, `PegasusEmptyView` | portal `FactoryPageState`, `EmptyState` (per-portal) |
| Spacing | `PegasusSpacing` | `PegasusMonochromeTheme.spacing*` | `ui-kit` desktop foundation |
| Realtime refresh | `RealtimeRefreshEffect` | `silentRealtimeRefresh` | `usePolling` in `@pegasusx/api-client` |

**App wiring (representative):**
- Supplier Android/iOS state panes → typealias to `Pegasus*` shared components
- Retailer Android `MotionTokens` → typealias `PegasusMotionTokens`
- Supplier iOS `SupplierAnim` → typealias `PegasusAnim`

**Desktop stale-while-revalidate:** supplier/warehouse/factory portals migrated from raw `setInterval` to `usePolling` (visibility-aware, abort-safe, backpressure-aware). Pair with `loading: isLoading && !data` in hooks.

**pegasus vs pegasusX component gaps (native):**

| Component | pegasus (legacy) | pegasusX |
| --- | --- | --- |
| Collapsible rail / sidebar | per-app | `PegasusCollapsibleRail` / `CollapsibleSidebar` |
| Monochrome theme | mixed teal/purple | `PegasusMonochromeTheme` |
| List loading bounce | `key(refreshEpoch)` remounts | `load(silent:)` + shared state panes |
| Motion tokens | duplicated per app | `PegasusMotionTokens` / `PegasusAnim` |
| Desktop polling | `setInterval` in page effects | `usePolling` shared hook |

## Desktop capabilities (Tauri portals — 2026-07-02)

| Capability | retailer-app-desktop | supplier-portal | warehouse-portal | factory-portal |
| --- | --- | --- | --- | --- |
| SQLite offline cache | yes | yes | yes | — |
| Offline tray | yes | yes | yes | yes |
| Native CSV export (save dialog) | via bridge | yes | yes | yes |
| Treasury print/PDF | — | yes | yes | — |
| Deep link scheme | `pegasusx-retailer://` | `pegasusx-supplier://` | `pegasusx-warehouse://` | `pegasusx-factory://` |
| Single instance | yes | yes | yes | yes |
| Tauri Android | — | **deprecated** (use `supplier-app-android`) | — | — |
| Virtualized order lists (3A) | yes | yes | yes | — |
| Fleet map 3D view | optional toggle (default 2D) | optional | optional | — |

### Desktop polling refresh strategy (PX-DESK-3E)

| Surface | Mechanism | Interval (visible) | Hidden tab |
| --- | --- | --- | --- |
| Supplier dashboard KPIs | `usePolling` | 60s | pause |
| Supplier fleet live map | `usePolling` + WS | 15s | 60s (`hiddenIntervalMs`) |
| Supplier dispatch preview | `usePolling` + WS | 30s | pause |
| Warehouse fleet live map | `usePolling` + WS | 15s | 60s |
| Factory home / supply requests | `usePolling` + WS | 30–60s | pause |
| Retailer dock queue | `useLiveData` + WS | 15s poll + reconcile on reconnect | SQLite cache on Tauri |

Prefer WebSocket-driven refresh where role inbox events exist; polling is fallback / map geometry only.

Reference: [`context/plan_desktop.md`](../context/plan_desktop.md), [`docs/qa/PX-DESK_MANUAL_QA.md`](./qa/PX-DESK_MANUAL_QA.md), [`docs/adr/008-desktop-tauri-strategy.md`](./adr/008-desktop-tauri-strategy.md).
