# PegasusX — Features by App / Role

**SOURCE OF TRUTH: CODE ONLY (not other markdown).**  
Extracted 2026-08-04; client parity re-synced 2026-08-12; **G7 regen 2026-08-13** (admin ops/dead-letters, factory SLA board, planning accuracy MAPE, partner EDI/1C/ASN); **P0 honesty 2026-08-13**; **P5 factory planning 2026-08-13** (API+workers, flags default off, no new screens); **P6 supplier extras 2026-08-13** (A+F API; clients P7-C); **leftover close 2026-08-14** (payout-policy, entityresolution, countrycfg UZ, SplitManifest naming, loyalty 410, P5-D DemandForecastBaseline, CRM Email, typed CT lists, retailer CT tiles); **P7 factory honesty 2026-08-13** (exception GET Spanner-first; dispatch no invent; C portal+native); **P8–P16 enterprise close 2026-08-13** (dead `payloaderoutes` removed; live `payloaderoutes`; staff login; QC gate; broadcast outbox; S&OP column; billing list; seal-all clients; **not cloud, not store**). Doc map: [`DOCS_SOURCE_OF_TRUTH.md`](./DOCS_SOURCE_OF_TRUTH.md).

**Honesty (P0):** this file is a **route + nav inventory**. A listed path is not a live product feature. Tags: **REAL** (durable + clients), **PARTIAL** (durable but incomplete), **THEATRE** (200 that does not persist the claim), **GONE** (410/403/501 by product), **STUB** (mounted but not the advertised engine). Re-verify in handlers — see [`ROLE_FEATURES_DOCS_VS_CODE.md`](./ROLE_FEATURES_DOCS_VS_CODE.md).

Extracted from:

- JWT roles: [`apps/backend-go/auth/claims.go`](../apps/backend-go/auth/claims.go)
- HTTP mounts: every `apps/backend-go/*routes/routes.go` (612 route registrations scanned)
- Client nav: `*Shell.tsx`, `*Section.kt` / `WarehouseSection.swift`, `DriverRoutes`, `RetailerNavigation.kt`
- Apps tree: `apps/*` (stubs noted from their `package.json`)

Companion (also code-grounded): [ROLE_CAPABILITIES_MATH_LOGIC.md](./ROLE_CAPABILITIES_MATH_LOGIC.md) · [ORDER_FLOW_AND_EDGE_CASES.md](./ORDER_FLOW_AND_EDGE_CASES.md) · [PROD_READINESS_SEQUENCE.md](./PROD_READINESS_SEQUENCE.md)

---

## 0. Apps on disk

| Dir | Status (from code/package.json) |
|-----|----------------------------------|
| `supplier-portal` | Live Next.js + Tauri 2 supplier/ADMIN UI (web + desktop shell) |
| `supplier-app-android` / `supplier-app-ios` | Live |
| `supplier-app-desktop` | **Does not exist** — desktop = `supplier-portal` Tauri |
| `admin-portal` | Live Next **PLATFORM_ADMIN** console: login+MFA, tenants / flags (+ dual-control approve) / audit / match queue / partner keys·AS2·SFTP / **ops outbox + Spanner dead-letters** / **billing invoices + fee-schedules** (P12) |
| `retailer-app-desktop` / `-android` / `-ios` | Live (desktop = Next 15 + Tauri 2) |
| `warehouse-portal` / `-android` / `-ios` | Live |
| `factory-portal` / `-android` / `-ios` | Live |
| `payload-terminal` / `-android` / `-ios` | Live |
| `driver-app-android` / `-ios` | Live (no portal) |
| `backend-go` | API |
| `ai-worker` | Workers (import/freeze/synthesis) |
| `handoff-service` | Handoff service |
| `marketing-site` | Marketing (not ops features) |

## 0.1 JWT roles (`auth/claims.go`)

```
ADMIN, PLATFORM_ADMIN, RETAILER, DRIVER, PAYLOAD,
FACTORY, FACTORY_ADMIN, FACTORY_DRIVER,
WAREHOUSE_ADMIN, WAREHOUSE
```

Scope fields on claims: `SupplierID`, `HomeNodeType`/`HomeNodeID`, retailer `RetailerOrgID`/`RetailerRole`/`CapabilityPacks`/`ActiveLocationID`. Comment in package: single-tenant `ResolveSupplierID` returns seeded supplier for authenticated callers.

---

## 1. Retailer — APIs (`retailerroutes` 145 + related)

Role gate: `RoleRetailer` (plus shared catalog/payment/order/claim routes).

### Clients (nav from code)

| Client | Evidence | Surfaces |
|--------|----------|----------|
| Desktop | `RetailerShell.tsx` hrefs | `/`, `/dashboard`, `/catalog`, `/my-suppliers`, `/orders`, `/tracking`, `/dock`, `/procurement`, `/auto-order`, `/insights`, **`/control-tower`**, `/reports`, `/hq`, `/credit`, `/stock`, `/stock/local-skus`, `/pos`, `/shifts`, `/sections`, `/assist`, `/settings` |
| Android | `RetailerNavigation.kt` + `SidebarMenu.kt` | `PROCUREMENT`, `CONTROL_TOWER`, `CREDIT`, `HQ`, `DOCK`, `ANALYTICS`, `AI_PREDICTIONS`→`FUTURE_DEMAND`, `AUTO_ORDER`, `CART`, `NOTIFICATIONS`, `ACCOUNT_PROFILE`, `FAMILY_MEMBERS`, `CAPABILITIES`, `TEAM`, `LOCATIONS`, `STORE_STOCK`, `LOCAL_SKUS`, `POS`, `SHIFTS`, `SECTIONS`, `REPORTS_PRO`, `ASSIST`, `SETUP_WIZARD` |
| iOS | `ContentView.swift` / `Screens/` | Full Retail OS + `HqView`, `CreditPartnersView`, `ControlTower`, `ReportsProView`, `PosView`, pulse on `DashboardView` |

### Feature inventory = registered routes

**Auth / identity**

| Feature | Methods |
|---------|---------|
| Login / refresh / register | `POST /v1/auth/retailer/{login,refresh,register}` |
| Memberships / select-org / switch-org | `GET …/memberships`, `POST …/select-org`, `POST …/switch-org` |
| Me + capabilities | `GET /v1/retailer/me`, `GET/POST /v1/retailer/capabilities*` |
| Switch location | `POST /v1/auth/retailer/switch-location` |
| Setup / profile | `POST /v1/retailer/setup`, `GET/PUT /v1/retailer/profile` |
| Family members (+ migrate-to-team) | `/v1/retailer/family-members*` |
| Org members / locations | `/v1/retailer/org/members*`, `/v1/retailer/locations*` |
| Notifications | `/v1/user/notifications*` |

**Procurement**

| Feature | Methods |
|---------|---------|
| Suppliers attach/detach | `/v1/retailer/suppliers*` |
| Cart sync | `GET/POST /v1/retailer/cart/sync` |
| Checkout quote / promo watch | `POST /v1/retailer/checkout/quote`, `POST /v1/retailer/promotions/watch` |
| Pricing rules (read) | `GET /v1/retailer/pricing/rules` |
| Cash/card checkout | **REAL** — `POST /v1/order/cash-checkout` and `POST /v1/order/card-checkout` (B1 cash → `PENDING_CASH_COLLECTION` + outbox; card opens a GP session). Not saved-cards. |
| Unified/preview/B2B checkout | `POST /v1/checkout/{unified,preview,b2b}` (`paymentroutes`). Unified `items[]` **REAL** create (no capture). Preview dry-run. **B2B GONE** — `410 payment_before_delivery_removed`. |
| Catalog browse | `GET /v1/catalog/*`, `GET /v1/products` (`catalogroutes`) |
| Saved cards | **GONE** — `/v1/retailer/card*`, `GET /v1/retailer/cards` `410 saved_cards_not_product` (P1; no vault). |
| Loyalty | **GONE** — `GET /v1/retailer/loyalty/tier` `410 loyalty_not_product` (`retailer/mobile_compat.go:136`). Never a fake tier. |

**Orders / delivery / AI**

| Feature | Methods |
|---------|---------|
| Order list | `GET /v1/orders`, `GET /v1/retailers/{id}/orders` |
| Cancel / request-cancel | `POST /v1/order/cancel` **REAL** pre-dispatch. `POST /v1/orders/request-cancel` **GONE** — hard `403 cancel_not_allowed`. |
| Preorder confirm/edit/reject | `POST /v1/orders/{confirm,edit,reject}-preorder` |
| Delivery proposal accept/reject | `POST /v1/orders/{accept,reject}-delivery-proposal` |
| Shop-closed respond | `POST /v1/retailer/shop-closed-response`, `POST /v1/retailer/orders/{id}/shop-closed/respond` |
| Confirm cash (doorstep) | `POST /v1/delivery/confirm-cash` (`orderroutes`, RoleRetailer) |
| Tracking / active fulfillment / pending payments | `GET /v1/retailer/{tracking,active-fulfillment}` **REAL**. `pending-payments` **PARTIAL** — real orders; `session_id`/`gateway` only when a PaymentSessions row exists (P2; no `sess_` / `payme`). |
| Pulse / control-tower pulse | `GET /v1/retailer/pulse`, `GET /v1/retailer/control-tower/pulse` |
| Auto-order settings + run | `/v1/retailer/settings/auto-order*` including `…/run` |
| AI predictions / confirm / reject | `GET /v1/retailer/ai/predictions` **REAL** `{items: RetailerAIPrediction}` (pending AI preorders, not SKU DemandForecast). Desktop `dashboard/page.tsx` / `insights/page.tsx`; Android `PegasusApi.kt` `getRetailerAIPredictions`; iOS `APIClient.getRetailerAIPredictions`. Confirm/reject-ai POSTs from those apps. `GET /v1/ai/predictions` **GONE** `410 use_retailer_ai_predictions` (alias kept for old store builds). `PATCH /v1/ai/predictions/correct` **GONE** `410 prediction_correct_unwired`. Procurement/auto-order SKU steppers do **not** map this list. |
| Reorder suggestions / sell-through | `GET /v1/retailer/reorder-suggestions`, `GET /v1/retailer/insights/sell-through` |
| Claims file/list/eligibility | `/v1/orders/{id}/claims*`, `claim-eligibility` (`orderroutes`) |
| Receipt / QR / timeline / status-context | orderroutes retailer GETs |
| Credit profile / relationships / AR | `creditroutes`: `/v1/retailer/credit-profile`, `credit-relationships`, `ar/invoices` |

**Retail OS**

| Feature | Methods |
|---------|---------|
| Store stock | `/v1/retailer/stock*` (receive/transfer/adjust/counts) |
| Local SKUs | `/v1/retailer/local-skus*` |
| POS | `/v1/retailer/registers`, `/v1/retailer/pos/*` |
| Time / shifts | `/v1/retailer/time/*`, `/v1/retailer/shifts*` |
| Sections | `/v1/retailer/sections*` |
| Assist tickets | `/v1/retailer/assist/tickets*` |
| Reports | `/v1/retailer/reports/*` |
| Franchise HQ | `/v1/retailer/hq/*` |
| Analytics expenses | `/v1/retailer/analytics/{expenses,detailed}` |

### Parity gaps (nav vs routes)

| Observation | Code basis |
|-------------|------------|
| HQ / Credit / AR | **Parity closed** — desktop `/hq` `/credit`; Android `CREDIT`/`HQ`; iOS `HqView` / `CreditPartnersView` (AR under credit) |
| Control Tower | Android/iOS first-class; desktop `/control-tower` in `RetailerShell` nav (R4.2). Pulse tiles navigate to orders/deliveries/dock/POS/shifts/assist/stock/reports (P13-E). |
| Reports / pulse | Desktop deep reports + mobile Reports Pro CSV share + dashboard pulse strip (`/v1/retailer/reports/export`, `/v1/retailer/pulse`) |
| POS holds | Mobile park/list/resume/void when `POS_HOLDS_ENABLED` (pilot default on; set `false` to hide) |

---

## 2. Supplier (`ADMIN`) — APIs (`supplierroutes` + finance packs)

### Clients

| Client | Evidence | Surfaces (abbrev.) |
|--------|----------|-------------------|
| Portal (+ Tauri desktop) | `SupplierShell.tsx` | dashboard, orders, dispatch, fleet*, manifests, exceptions, catalog, inventory(+import), pricing*, promotions, topology/factories/warehouses/zones/lanes, ledger/payments/chargebacks*/reconciliation/treasury*/earnings/compliance, credit/*, control-tower, playbooks, planning, segmentation, tax-regimes, AI, analytics*, replenishment*, returns, org-fleet, activity, ops/map, … |
| Android / iOS | `SupplierSection.kt` (+ iOS peers) | `DASHBOARD`, `ORDERS`, `FLEET*`, `MANIFESTS`, `DISPATCH_PREVIEW`, `ACTIVITY`, `ORG_FLEET`, `TREASURY_HUB`, `LEDGER`, `PAYMENTS`, `CHARGEBACKS`, `RECONCILIATION`, `OPERATIONS`, `ANALYTICS`, `AI_RECOMMENDATIONS`, topology/catalog/inventory/pricing/promotions/returns/earnings/profile/setup, … |
| `supplier-app-desktop` | — | **Does not exist** — use `supplier-portal` Tauri (see §0) |

### Feature inventory (route groups)

| Domain | Route prefix / examples |
|--------|-------------------------|
| Auth / setup | `/v1/auth/supplier/*`, `/v1/supplier/{configure,business/setup,billing/setup,profile,settings}` |
| Dashboard / activity / pulse | `/v1/supplier/{dashboard,activity}`, `/v1/supplier/pulse` |
| Orders | `/v1/supplier/orders`, `…/vet`, `…/payment-bypass`, receipts |
| Shop-closed ops | `/v1/supplier/shop-closed/{active,resolve}` |
| Negotiations | **GONE** (default) — `/v1/supplier/negotiations/pending` empty list; `…/negotiate/resolve` `410 feature_disabled` unless `QUANTITY_NEGOTIATION_ENABLED`. |
| Early-complete | `/v1/supplier/route/approve-early-complete` |
| Reassign | `/v1/supplier/{recommend-reassign,reassign-order}` |
| Dispatch / fleet | `/v1/supplier/dispatch/*`, `/v1/supplier/fleet/*`. Oversize routes split after capacity checks (`dispatch.ExpandOversizeRoutes`, `AUTO-{driver}-{ts}-A/B`, cap 25). |
| Manifests / exceptions | `/v1/supplier/manifests`, `exceptions*`, `manifest-exceptions`, `ops/exception-map` |
| Catalog CRUD | `catalogroutes` POST/PUT products (RoleAdmin) |
| Inventory + import + audit | `/v1/supplier/inventory*` adjust/import **REAL**. `GET /v1/supplier/inventory/audit` **GONE** `410 audit_unwired` (P1). |
| Pricing / promotions | `/v1/supplier/pricing/*`, `/v1/supplier/promotions*` |
| Topology | `/v1/supplier/topology`, `supply-lanes` (**topology utilization JSON unchanged** — not optimizer `SupplyLanes` rows), factories/warehouses via supplier APIs |
| Factory planning (P5 / P7-C) | `GET/PUT /v1/supplier/network-mode`, `POST /v1/supplier/planning/pull-matrix`, `POST /v1/supplier/planning/kill-switch`, `POST /v1/supplier/planning/predictive-push` — ADMIN. Portal `/settings/planning` factory-ops panel + native PlanningSettings. Engines no-op unless `FACTORY_PLANNING_ENABLED` (Go default false). P5-D `SYSTEM_PREDICTED` from `DemandForecastBaseline` (not `AIPredictions`); `"grain":"demand_forecast_baseline"`. |
| Supplier CRM (P6-A / P7-C) | `GET /v1/supplier/crm/retailers`, `GET …/crm/retailers/{retailerId}` — ADMIN, **PARTIAL** portal+native (`/crm`, Android, iOS). `Orders.Status` + `TotalMinor`; `Retailers.Email` when set. Empty `{"retailers":[]}`; **503** without Spanner. Does **not** change warehouse `/ops/crm`. **Not store.** |
| Payout batches (P7-C) | `GET/POST /v1/supplier/payouts/batches`, GET by id, export, mark-paid, dispatch. Portal `/finance/payouts` + Android/iOS. Live `dispatch` with `live:true` → `no_live_rail`. |
| Payout policy (P6-B) | `GET/PATCH /v1/supplier/payout-policy` — ADMIN. Modes `HQ_SUPPLIER` \| `WAREHOUSE_LOCAL`. PATCH requires `reason`. Thin portal+Android+iOS. **Not** a live PSP. |
| Entity resolution (P6-C) | `POST /v1/supplier/entity-resolution/{resolve,explain}` — ADMIN. Result direct (no `{status:ok}`). API-only. |
| Country cfg (P6-D) | `GET /v1/country-configs/{code}` UZ only; else 404 `country_not_supported`. `GET/PATCH /v1/supplier/country-overrides/{code}`. `checkout_reads_this: false`. |
| Finance | `/v1/payment/{ledger,chargeback*,settlement/authority,reconciliation/mismatches}`, claim-chargebacks, cash-reconciliations, credit-notes |
| Credit program | `/v1/supplier/credit-{profiles,program*,relationships*}`, admin disable |
| Claims adjudicate | `/v1/supplier/claims`, `/v1/claims/{id}/{approve,reject}` |
| Compliance | `/v1/compliance/*`, tax-regimes |
| Planning / AI / MEIO / twin | `/v1/supplier/{ai,planning,meio,segmentation,knowledge-graph}*`, twin routes, replenishment*. S&OP **PARTIAL** — `Factories.DailyOutputCapacity` when column sum > 0 (`capacity_source: factories_column`); else `capacity_source: env_default` (`SOP_FACTORY_DAILY_UNITS` default 700) (P10-B). |
| Control tower | `/v1/control-tower/*`, zone-overrides |
| Returns / broadcast | `/v1/supplier/returns*`, `/v1/supplier/broadcast` (P10-A persist + outbox then WS) |
| Demand signals | `demandroutes` `/v1/demand/signals*` |

### Parity gaps (portal shell vs Android section)

P13-B native slices exist on Android `SupplierSection` + iOS `SupplierSection`: `/control-tower` and `/settings/playbooks` are **typed lists** (scored exceptions / playbooks). Segmentation, tax-regimes, credit policy, flywheel, payday remain JSON dumps. `/settings/planning`, `/crm`, `/finance/payouts` remain from P7-C (payouts include payout-policy mode). P13-E retailer Control Tower tiles navigate.

---

## 3. Warehouse — APIs (`warehouseroutes` 78 + returns/creditnote/claims)

Roles: `WAREHOUSE`, `WAREHOUSE_ADMIN` (many ops routes require Admin/WarehouseAdmin).

### Clients

| Client | Evidence |
|--------|----------|
| Portal | `WarehouseShell.tsx`: `/`, dispatch*, orders, preorders, tomorrow-board, drivers/vehicles, inventory, **`/bins`**, **`/pick-waves`**, **`/cycle-counts`**, **`/cold-chain`**, **`/labor-capacity`**, stock-commitments, products, manifests, fleet-live-map, replenishment, demand-forecast, supply-requests, transfers, returns, claims, exceptions, rescues, analytics, crm, treasury, payment-config, staff, settings (incl. return-policy embed), operations, control-tower, dispatch-locks |
| Android | `WarehouseSection.kt`: `DASHBOARD`, `ORDERS`, `DRIVERS`, `VEHICLES`, `INVENTORY`, `DISPATCH`, `ANALYTICS`, `TREASURY`, `STAFF`, `MANIFESTS`, `DISPATCH_SETTINGS`, `FLEET_LIVE_MAP`, `TRANSFER_ACTIONS` (pick-wave/cycle nested), `PRODUCTS`, `PREORDERS`, `TOMORROW_BOARD`, `STOCK_COMMITMENTS`, `SUPPLY_REQUESTS`, `REPLENISHMENT`, `DEMAND_FORECAST`, `RETAILERS`, `RETURNS`, **`COLD_CHAIN`**, **`LABOR_CAPACITY`**, `RETURN_POLICY`, `EXCEPTIONS`, **`CONTROL_TOWER`**, `CLAIMS`, `RESCUES`, `PAYMENT_CONFIG`, `OPS_SETTINGS`, `LOCATION_SETTINGS`, `NOTIFICATIONS`, `PORTAL_*` handoff |
| iOS | Peer screens incl. `ReturnPolicySettingsView`, **`ColdChainView`**, **`LaborCapacityView`**, **Control Tower** typed scored-exception list (`GET /v1/control-tower/exceptions/scored`); pick/cycle via Transfer Actions |

### Feature inventory

| Domain | Routes |
|--------|--------|
| Auth / setup | `/v1/auth/warehouse/*`, `/v1/warehouse/setup` |
| Dispatch | `/v1/warehouse/ops/dispatch/{preview,execute,settings,runs*}`, rescue preview/propose, tracking, dispatch-locks |
| Reassign | `/v1/warehouse/{recommend-reassign,reassign-order}` |
| Orders / preorders | delay/reject/overflow/propose-delivery; preorder edit/reject |
| Board / exceptions / broadcast | `/ops/board`, `/ops/exceptions`, broadcast templates |
| Fleet | drivers/vehicles/staff, live-map |
| Inventory / commitments / products / settings / location | `/ops/inventory*`, stock-commitments, products, settings, location |
| Replenishment / demand / supply / transfers | insights, supply-requests, transfers emergency/receive/force-receive **REAL**. Demand/forecast **PARTIAL** — Spanner first (`source: spanner` even if empty); scaffold only when `WAREHOUSE_PORTAL_SEED`; else `source: empty` (P2). `GET/POST /v1/warehouse/supply-requests/{id}/qc` portal+native (P9-C / P11-C). |
| Returns inbound | `returnsroutes` `/v1/returns/inbound*` (+ `/ops/returns`) |
| Reverse logistics | `GET/POST /v1/warehouse/reverse-logistics*` (**RequireRole Warehouse only**) |
| Return policy | `GET/PUT /v1/warehouse/return-policy` (WarehouseAdmin/Admin) |
| Claims | shared orderroutes claims approve/reject (WarehouseAdmin) |
| Treasury / financials / payment-config / CRM / analytics | `/ops/{treasury,financials,payment-config,crm,analytics}`. CRM/payment-config **REAL**. Warehouse CRM JSON (`retailers[]` with `business_name` / `total_orders` / `total_revenue` / `last_order_date`) **unchanged**. Portal+native last_order + load-error honesty (P7-C). Treasury invoices **PARTIAL** — ArInvoices⋈Orders by warehouse (P2); `[]` only when query empty; no Spanner → 503. Analytics **PARTIAL** — Spanner fills breakdowns; fallback `*_available: false`. Financials `daily_revenue` from Orders when Spanner; `gateway_breakdown` from `PaymentSessions` ⋈ warehouse orders when query OK; `platform_fee` from `BillingFeeSchedules` when a schedule exists; else `*_available: false` (P11-A). |
| Pulse | `/v1/warehouse/ops/pulse` |

### Parity gaps

| Observation | Code |
|-------------|------|
| Control Tower | Portal shell + Android `CONTROL_TOWER` + iOS `controlTower` (P13-C) — typed `GET /v1/control-tower/exceptions/scored` list |
| Cold-chain / labor-capacity | Portal + Android/iOS (R4.1) — `ColdChainScreen`/`LaborCapacityScreen`, `ColdChainView`/`LaborCapacityView` |
| Pick waves / cycle counts / bins | Portal dedicated routes; mobile nested under Transfer Actions |
| Return policy | Routes + Android `RETURN_POLICY` + iOS settings; portal embeds under settings (no `/return-policy` href) |
| Reverse-logistics role = `WAREHOUSE` not `WAREHOUSE_ADMIN` | `creditnoteroutes` |
| `PORTAL_SETUP` / `PORTAL_PROFILE` / `PORTAL_SEARCH` on Android | explicit portal handoff |

---

## 4. Factory — APIs (`factoryroutes` 39)

Roles: `FACTORY`, `FACTORY_ADMIN` (+ ADMIN). `FACTORY_DRIVER` uses driver routes.

### Clients

| Client | Evidence |
|--------|----------|
| Portal | `FactoryShell.tsx`: `/`, loading-bay, manifests, manifest-exceptions, payload-override, transfers, supply-requests, fleet, staff, insights, analytics, settings/location |
| Android | `FactorySection.kt`: `DASHBOARD`, `LOADING_BAY`, `TRANSFERS`, `FLEET`, `STAFF`, `LOCATION`, `SUPPLY_REQUESTS`, `PAYLOAD_OVERRIDE`, `MANIFESTS`, `MANIFEST_EXCEPTIONS`, `INSIGHTS`, `ANALYTICS`, `NOTIFICATIONS` |

### Feature inventory

| Domain | Routes |
|--------|--------|
| Auth / setup / profile / location | `/v1/auth/factory/*`, `/v1/factory/setup`, profile, ops/location |
| Dashboard / analytics / pulse | `/v1/factory/{dashboard,analytics/overview}`, `/v1/factory/pulse` |
| Manifest lifecycle | start-loading, seal, complete **REAL** under Spanner. `POST /v1/factory/dispatch` **STUB default** — pick ≤2 `CREATED` transfers, first driver/vehicle, DRAFT; **does not invent** if the queue is empty (`created_manifest_count: 0`, P7-B); `optimizer_class=HEURISTIC` + `dispatch_algo=pick_n_created_v1`. When `FACTORY_BATCHER_ENABLED`: FFD+NN+LIFO → `FactoryTruckManifests` `DRAFT`, **no invent-if-empty**, `dispatch_algo=ffd_nn_lifo_v1` (P5-F). **Not** warehouse VRP / `dispatch.BinPack`. |
| Rebalance / cancel | manifests rebalance, cancel-transfer, cancel |
| Exceptions | GET **PARTIAL** — Spanner `ManifestExceptions` ⋈ `FactoryTruckManifests` (P7-A); JSON `transfer_id` ← `OrderId`; empty `[]` when query/seed empty. Resolve **PARTIAL** — Spanner-first (P9-B); memory only `FACTORY_PORTAL_SEED`; `RunTx` + `MANIFEST_EXCEPTION_RESOLVED` (P3). Supplier exception JSON (`order_id`) **unchanged**. |
| Transfers / supply-requests | GET **PARTIAL** — Spanner `FactoryInternalTransfers` first; memory overlay only `FACTORY_PORTAL_SEED`/`USE_DEMO_SEED` (P3). Transition + supply accept **REAL**/PARTIAL. Transfer create **PARTIAL** — persist via `apply` + `TRANSFER_CREATED` outbox (P3). |
| Supply-request QC (P6-F / P7-C / P9-C) | `GET/POST /v1/factory/supply-requests/{id}/qc` **PARTIAL** portal+native (board/list + Android/iOS cards). Table `FactorySupplyRequestQC` on `WarehouseSupplyRequests`. PASS/FAIL upsert; does **not** change request State; accept **409** unless QC `PASS`. Outbox `FACTORY_SUPPLY_REQUEST_UPDATE`. Missing QC row → 200 `result: ""`. **Not store.** |
| **SLA board (G7.1)** | `GET /v1/factory/sla-board` + `sla_*` on supply-requests; portal badges — request due-date (`kind: supply_request`). Transfer-transit 1×/1.5×/2× is a **second** worker (`kind: transfer_transit`) behind `FACTORY_PLANNING_ENABLED` (P5-E) — not this board. |
| Fleet / staff | fleet/drivers/vehicles reads. Staff POST **PARTIAL** — `SupplierUsers` via `RunTx` + `FACTORY_STAFF_CREATED` outbox (P3); password bcrypt/invite (P9-A), never `"unset"`. Set-password UI portal+native (P13-D). GET Spanner-first (`AssignedFactoryId`); memory overlay only with seed. |
| Insights | clients call `/v1/warehouse/replenishment/insights` (warehouse route allows factory roles in handler gate) |

No dedicated `FACTORY_DRIVER` app directory — role allowed on `driverroutes`.

---

## 5. Payload — APIs (`payloaderoutes` 27)

Role: `PAYLOAD` (+ ADMIN).

### Clients

| Client | Evidence |
|--------|----------|
| Terminal | `apps/payload-terminal` Expo app |
| Android | `PayloadRoot.kt` (home + inbound) |
| iOS | Root home + inbound views |

### Feature inventory

| Feature | Routes |
|---------|--------|
| Auth | `/v1/auth/payloader/{login,refresh}` |
| Board | `/v1/payloader/{trucks,orders,manifests}` |
| Load / seal | start-loading, seal, seal-completed, `/v1/payload/seal` |
| Inject | `…/inject-order` |
| Reassign | recommend-reassign, reassign-order, `/v1/fleet/reassign` |
| Exceptions | `/v1/payload/manifest-exception`, list, `/v1/delivery/exception-report` |
| Alias supplier manifests | `/v1/supplier/manifests*` same handlers |
| Pulse / notifications | `/v1/payloader/pulse`, user notifications |
| Inbound returns | `returnsroutes` (PAYLOAD allowed on inbound) |

**Client usage notes:** terminal + native apps use inbound + seal-completed / per-order seal + **`POST …/seal-all`** (P13-A: terminal, Android, iOS). **`GET /v1/payload/capacity/{vehicleID}` is GONE** `410 capacity_unwired` (hardcoded theatre; VU lives on the manifest).

---

## 6. Driver — APIs (`driverroutes` 49 + `orderroutes` doorstep + telemetry)

Roles: `DRIVER`, `FACTORY_DRIVER`.

### Clients

| Client | Evidence |
|--------|----------|
| Android | `DriverRoutes`: `LOGIN`, `MAIN`, `SCANNER`, `NOTIFICATIONS`, `CORRECTION`, `OFFLOAD_REVIEW`, `PAYMENT_WAITING`, `CASH_COLLECTION`, `SHOP_CLOSED_WAITING`, `OFFLINE_VERIFIER`, `SYNC_QUEUE`, `SUPPLY_TRANSFERS` |
| iOS | Matching Views under driver-app-ios |
| Portal | **none** under `apps/` |

### Feature inventory

| Domain | Routes |
|--------|--------|
| Auth / profile / availability / pulse | `/v1/auth/driver/login`, `/v1/driver/{profile,availability,pulse,earnings}`. `GET /v1/driver/history` **PARTIAL** — Spanner Orders by DriverId (30d completed window, P2); `[]` when none / query unset. |
| Manifest / fleet | manifest, manifest-gate, `/v1/fleet/{manifest,orders}`, geometry, depart, return-complete, reorder, request-early-complete |
| Telemetry | `POST /v1/telemetry/location` |
| Doorstep (orderroutes) | arrive, proximity-unlock, shop-closed, partial-offload, scan-qr, deliver, confirm-offload, collect-cash, complete, fiscal/retry |
| Doorstep (driverroutes aliases) | shop-closed, partial-offload, credit-leave, validate-qr, amend, credit-delivery, missing-items, exception-report, split-payment, confirm-payment-bypass, bypass-offload. `POST /v1/delivery/negotiate` **GONE** — default `410 feature_disabled`. |
| Sync offline | `POST /v1/sync/batch` |
| Supply transfers | `/v1/driver/supply-transfers*` |
| Rescue | `/v1/driver/ops/rescue/{request,respond}` |
| Cash recon | `/v1/driver/cash-reconciliations` |
| Return goods | `GET /v1/driver/return-goods` |
| Open fiscal / pending collections | `/v1/driver/open-fiscal`, `pending-collections` |
| Handshake | `deliveryroutes` verify-handshake. `update-order-during-delivery` **GONE** — `not_implemented`; use amend / missing-items / partial-offload. |
| `PATCH /v1/orders/{id}/state` | **GONE** — mounted `501 use_delivery_edges`; not a feature. Use arrive / collect-cash / credit-leave / partial-offload / depart. |
| Force-complete | **not** driver — `RoleAdmin`, `RoleWarehouseAdmin` only |

---

## 7. Platform routes (all roles)

| Package | Routes |
|---------|--------|
| `platformroutes` | client-policy/config, media upload-ticket, device-token, auth refresh/logout, **GS-A** `GET /v1/auth/session` (JWT + market pack; `checkout_reads_this: false`), `GET /v1/platform/market-packs` (+ `/{code}`) |
| `webhookroutes` | global-pay, adyen, stripe, payme, click |
| `updateroutes` | iOS plist + desktop updater.json |
| `infraroutes` | healthz/ready (+ G4 `GET /v1/health/capabilities` optimizer honesty) |
| `etaroutes` / `laborcapacityroutes` | ETA + labor capacity helpers |
| `platformadmin` (G4–G7 / P12) | login, MFA, tenants, flags, audit, match queue, partner, `/ops/outbox/{summary,events,dead-letters}`, `/ops/runtime`, `GET /v1/admin/billing/{invoices,fee-schedules}`, `POST /v1/admin/billing/run-monthly` |
| Planning accuracy (G6) | `GET /v1/admin/planning/accuracy`, `POST …/run-once` (`mape28`, `demoted`) |
| Partner enterprise I/O (G5) | EDI profile GET/PUT, 1C import, master-data, WMS ASN in/out |
| Dispatch honesty | `optimizer_class` HEURISTIC\|OPTIMAL; `matrix_source` haversine\|osrm |

---

## 8. How this file was built

1. Listed `apps/*` and stub `package.json` descriptions.  
2. Parsed every `.(Get|Post|Put|Patch|Delete)("…")` in `*routes/routes.go`.  
3. Read shell href arrays / Kotlin section enums / DriverRoutes.  
4. Did **not** copy feature lists from `ECOSYSTEM_FEATURES_BY_ROLE.md` or other docs.

If a feature is not a route above and not a client nav id above, it is not claimed here.

**P0 honesty (2026-08-13):** THEATRE/GONE/STUB tags above were checked against live handlers (`retailer/mobile_compat.go`, `order/retailer_request_cancel.go`, `supplier/portal_handlers.go`, `factory/service.go`, `warehouse/ops_portal.go`, `driver/service.go`). A registered route is not a live product.
