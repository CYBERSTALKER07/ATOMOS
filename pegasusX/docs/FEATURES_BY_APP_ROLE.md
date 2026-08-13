# PegasusX — Features by App / Role

**SOURCE OF TRUTH: CODE ONLY (not other markdown).**  
Extracted 2026-08-04; client parity re-synced 2026-08-12; **G7 regen 2026-08-13** (admin ops/dead-letters, factory SLA board, planning accuracy MAPE, partner EDI/1C/ASN). Doc map: [`DOCS_SOURCE_OF_TRUTH.md`](./DOCS_SOURCE_OF_TRUTH.md).

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
| `admin-portal` | Live Next **PLATFORM_ADMIN** console: login+MFA, tenants / flags (+ dual-control approve) / audit / match queue / partner keys·AS2·SFTP / **ops outbox + Spanner dead-letters** |
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
| Cash/card checkout | `POST /v1/order/{cash,card}-checkout` |
| Unified/preview/B2B checkout | `POST /v1/checkout/{unified,preview,b2b}` (`paymentroutes`) |
| Catalog browse | `GET /v1/catalog/*`, `GET /v1/products` (`catalogroutes`) |
| Saved cards | `/v1/retailer/card*`, `GET /v1/retailer/cards` |

**Orders / delivery / AI**

| Feature | Methods |
|---------|---------|
| Order list | `GET /v1/orders`, `GET /v1/retailers/{id}/orders` |
| Cancel / request-cancel | `POST /v1/order/cancel`, `POST /v1/orders/request-cancel` |
| Preorder confirm/edit/reject | `POST /v1/orders/{confirm,edit,reject}-preorder` |
| Delivery proposal accept/reject | `POST /v1/orders/{accept,reject}-delivery-proposal` |
| Shop-closed respond | `POST /v1/retailer/shop-closed-response`, `POST /v1/retailer/orders/{id}/shop-closed/respond` |
| Confirm cash (doorstep) | `POST /v1/delivery/confirm-cash` (`orderroutes`, RoleRetailer) |
| Tracking / active fulfillment / pending payments | `GET /v1/retailer/{tracking,active-fulfillment,pending-payments}` |
| Pulse / control-tower pulse | `GET /v1/retailer/pulse`, `GET /v1/retailer/control-tower/pulse` |
| Auto-order settings + run | `/v1/retailer/settings/auto-order*` including `…/run` |
| AI predictions / confirm / reject | `/v1/retailer/ai/predictions`, `/v1/ai/*`, `POST /v1/retailer/orders/{confirm,reject}-ai` |
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
| Control Tower | Android/iOS first-class; desktop `/control-tower` in `RetailerShell` nav (R4.2) |
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
| Negotiations | `/v1/supplier/negotiations/pending`, `…/negotiate/resolve` |
| Early-complete | `/v1/supplier/route/approve-early-complete` |
| Reassign | `/v1/supplier/{recommend-reassign,reassign-order}` |
| Dispatch / fleet | `/v1/supplier/dispatch/*`, `/v1/supplier/fleet/*` |
| Manifests / exceptions | `/v1/supplier/manifests`, `exceptions*`, `manifest-exceptions`, `ops/exception-map` |
| Catalog CRUD | `catalogroutes` POST/PUT products (RoleAdmin) |
| Inventory + import + audit | `/v1/supplier/inventory*` |
| Pricing / promotions | `/v1/supplier/pricing/*`, `/v1/supplier/promotions*` |
| Topology | `/v1/supplier/topology`, `supply-lanes`, factories/warehouses via supplier APIs |
| Finance | `/v1/payment/{ledger,chargeback*,settlement/authority,reconciliation/mismatches}`, claim-chargebacks, cash-reconciliations, credit-notes |
| Credit program | `/v1/supplier/credit-{profiles,program*,relationships*}`, admin disable |
| Claims adjudicate | `/v1/supplier/claims`, `/v1/claims/{id}/{approve,reject}` |
| Compliance | `/v1/compliance/*`, tax-regimes |
| Planning / AI / MEIO / twin | `/v1/supplier/{ai,planning,meio,segmentation,knowledge-graph}*`, twin routes, replenishment* |
| Control tower | `/v1/control-tower/*`, zone-overrides |
| Returns / broadcast | `/v1/supplier/returns*`, `/v1/supplier/broadcast` |
| Demand signals | `demandroutes` `/v1/demand/signals*` |

### Parity gaps (portal shell vs Android section)

Portal-only hrefs with no matching Android section enum: `/control-tower`, `/settings/playbooks`, `/settings/segmentation`, `/settings/tax-regimes`, `/credit/policy`, `/credit/admin-disable`, `/analytics/demand/flywheel`, `/demand/payday-calendar`, `/settings/planning` (Android has analytics/AI/ops subset instead).

---

## 3. Warehouse — APIs (`warehouseroutes` 78 + returns/creditnote/claims)

Roles: `WAREHOUSE`, `WAREHOUSE_ADMIN` (many ops routes require Admin/WarehouseAdmin).

### Clients

| Client | Evidence |
|--------|----------|
| Portal | `WarehouseShell.tsx`: `/`, dispatch*, orders, preorders, tomorrow-board, drivers/vehicles, inventory, **`/bins`**, **`/pick-waves`**, **`/cycle-counts`**, **`/cold-chain`**, **`/labor-capacity`**, stock-commitments, products, manifests, fleet-live-map, replenishment, demand-forecast, supply-requests, transfers, returns, claims, exceptions, rescues, analytics, crm, treasury, payment-config, staff, settings (incl. return-policy embed), operations, control-tower, dispatch-locks |
| Android | `WarehouseSection.kt`: `DASHBOARD`, `ORDERS`, `DRIVERS`, `VEHICLES`, `INVENTORY`, `DISPATCH`, `ANALYTICS`, `TREASURY`, `STAFF`, `MANIFESTS`, `DISPATCH_SETTINGS`, `FLEET_LIVE_MAP`, `TRANSFER_ACTIONS` (pick-wave/cycle nested), `PRODUCTS`, `PREORDERS`, `TOMORROW_BOARD`, `STOCK_COMMITMENTS`, `SUPPLY_REQUESTS`, `REPLENISHMENT`, `DEMAND_FORECAST`, `RETAILERS`, `RETURNS`, **`COLD_CHAIN`**, **`LABOR_CAPACITY`**, `RETURN_POLICY`, `EXCEPTIONS`, `CLAIMS`, `RESCUES`, `PAYMENT_CONFIG`, `OPS_SETTINGS`, `LOCATION_SETTINGS`, `NOTIFICATIONS`, `PORTAL_*` handoff |
| iOS | Peer screens incl. `ReturnPolicySettingsView`, **`ColdChainView`**, **`LaborCapacityView`**; pick/cycle via Transfer Actions; Control Tower remains portal-primary |

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
| Replenishment / demand / supply / transfers | insights, demand/forecast, supply-requests, transfers emergency/receive/force-receive |
| Returns inbound | `returnsroutes` `/v1/returns/inbound*` (+ `/ops/returns`) |
| Reverse logistics | `GET/POST /v1/warehouse/reverse-logistics*` (**RequireRole Warehouse only**) |
| Return policy | `GET/PUT /v1/warehouse/return-policy` (WarehouseAdmin/Admin) |
| Claims | shared orderroutes claims approve/reject (WarehouseAdmin) |
| Treasury / financials / payment-config / CRM / analytics | `/ops/{treasury,financials,payment-config,crm,analytics}` |
| Pulse | `/v1/warehouse/ops/pulse` |

### Parity gaps

| Observation | Code |
|-------------|------|
| Control Tower | Portal shell route live; **absent** from Android/iOS section enums (portal-primary) |
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
| Manifest lifecycle | start-loading, seal, dispatch, complete; factory dispatch |
| Rebalance / cancel | manifests rebalance, cancel-transfer, cancel |
| Exceptions | manifest-exceptions list/resolve |
| Transfers / supply-requests | create/transition; accept/fulfill-options/patch |
| **SLA board (G7.1)** | `GET /v1/factory/sla-board` + `sla_*` on supply-requests; portal badges |
| Fleet / staff | fleet, drivers, vehicles, staff CRUD |
| Insights | clients call `/v1/warehouse/replenishment/insights` (warehouse route allows factory roles in handler gate) |

No dedicated `FACTORY_DRIVER` app directory — role allowed on `driverroutes`.

---

## 5. Payload — APIs (`payloaderroutes` 27)

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
| Load / seal | start-loading, seal, seal-completed, seal-all, `/v1/payload/seal` |
| Inject | `…/inject-order` |
| Reassign | recommend-reassign, reassign-order, `/v1/fleet/reassign` |
| Exceptions | `/v1/payload/manifest-exception`, list, `/v1/delivery/exception-report` |
| Capacity | `GET /v1/payload/capacity/{vehicleID}` |
| Alias supplier manifests | `/v1/supplier/manifests*` same handlers |
| Pulse / notifications | `/v1/payloader/pulse`, user notifications |
| Inbound returns | `returnsroutes` (PAYLOAD allowed on inbound) |

**Client usage notes:** terminal + native apps use inbound + seal-completed / per-order seal. **`seal-all` and capacity are API-registered with no payload-terminal / Android / iOS call sites** (backend-only until a client wires them).

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
| Auth / profile / availability / pulse | `/v1/auth/driver/login`, `/v1/driver/{profile,availability,pulse,history,earnings}` |
| Manifest / fleet | manifest, manifest-gate, `/v1/fleet/{manifest,orders}`, geometry, depart, return-complete, reorder, request-early-complete |
| Telemetry | `POST /v1/telemetry/location` |
| Doorstep (orderroutes) | arrive, proximity-unlock, shop-closed, partial-offload, scan-qr, deliver, confirm-offload, collect-cash, complete, fiscal/retry |
| Doorstep (driverroutes aliases) | shop-closed, partial-offload, credit-leave, validate-qr, amend, negotiate, credit-delivery, missing-items, exception-report, split-payment, confirm-payment-bypass, bypass-offload |
| Sync offline | `POST /v1/sync/batch` |
| Supply transfers | `/v1/driver/supply-transfers*` |
| Rescue | `/v1/driver/ops/rescue/{request,respond}` |
| Cash recon | `/v1/driver/cash-reconciliations` |
| Return goods | `GET /v1/driver/return-goods` |
| Open fiscal / pending collections | `/v1/driver/open-fiscal`, `pending-collections` |
| Handshake | `deliveryroutes` verify-handshake, update-order-during-delivery |
| Force-complete | **not** driver — `RoleAdmin`, `RoleWarehouseAdmin` only |

---

## 7. Platform routes (all roles)

| Package | Routes |
|---------|--------|
| `platformroutes` | client-policy/config, media upload-ticket, device-token, auth refresh |
| `webhookroutes` | global-pay, adyen, stripe, payme, click |
| `updateroutes` | iOS plist + desktop updater.json |
| `infraroutes` | healthz/ready (+ G4 `GET /v1/health/capabilities` optimizer honesty) |
| `etaroutes` / `laborcapacityroutes` | ETA + labor capacity helpers |
| `platformadmin` (G4–G7) | login, MFA, tenants, flags, audit, match queue, partner, `/ops/outbox/{summary,events,dead-letters}`, `/ops/runtime` |
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
