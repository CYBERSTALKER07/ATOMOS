# pegasusX Role-Row Parity Matrix

> **Canonical cross-role spec:** [`FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`](./FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md) — use this matrix for screen-level parity; use the master plan for end-to-end flows, comms, and verification gates.

Last updated: 2026-06-17 (factory FA9-03 Firebase OTP). Canonical reference: `pegasus/`. Delivery tree: `pegasusX/`.

## Summary

| Role | pegasusX clients | Backend routes | Production v1 capability | UI parity (vs Pegasus) | E2E (SSMR) |
|------|------------------|----------------|--------------------------|------------------------|------------|
| SUPPLIER | supplier-portal, native iOS/Android | supplierroutes | Full ops spine on portal + native: register/business/billing onboarding, order vet, inventory adjust/import, retailer overrides, chargebacks, treasury hub, demand history, factories/warehouses browse, catalog detail; fleet live map on all clients | **Wired** — pegasusX single-tenant surface (~42 portal routes + native parity); pegasus multi-tenant extras (CRM, staff, country-overrides) out of scope | Full SSMR e2e incl. payment + factory |
| RETAILER | desktop, iOS, Android | retailerroutes, orderroutes, catalogroutes | Order lifecycle (manual pre-order Standard vs Scheduled checkout, preorder confirm/edit, request-cancel), setup wizard, insights dismiss, catalog/search, tracking + dock, unified checkout | **Wired** — desktop richest; mobile Deliveries hub; Midnight Guard + `PRE_ORDER_*` events | Register, order create, tracking, `PX_E2E_MANUAL_PREORDER_OK`, `PX_E2E_CATALOG_OK`, retailer SSMR markers |
| DRIVER | Android, iOS | driverroutes, orderroutes, telemetryroutes | Full delivery edges + reorder (PX12-B); planned route geometry + turn-by-turn + off-route reroute (`GET /v1/fleet/route/{routeID}/geometry`); maps + WS (PX12-F); Firebase phone OTP | **Wired** — live-ops + planned/breadcrumb map overlays; phone OTP + PIN dev on Android/iOS | Telemetry; shop-closed; negotiation; driver edges E2E; `PX_E2E_DRIVER_FIREBASE_OTP_OK` when test token set |
| WAREHOUSE | portal, Android, iOS | warehouseroutes | Pre-order hub, stock commitments drill-down, ops settings (express toggle), reject/cancel anytime | **Wired** — `/preorders`, `/stock-commitments`, Midnight Guard sweeper, `PRE_ORDER_*` WS | `PX_E2E_MANUAL_PREORDER_OK`, `PX_E2E_WAREHOUSE_PREORDER_REJECT_OK`, dispatch SSMR markers |
| FACTORY | portal, Android, iOS | factoryroutes | Manifests + supply requests + dispatch (PX12-J); Firebase phone OTP | Portal 13 routes; insights P1; manifest exception inbox + transfer create P2; manifest lifecycle + staff detail P3; supply-request transitions + payload override cross-manifest rebalance P4; loading-bay transfer filter/detail/dispatch + transfer state machine P5 on all factory clients; phone OTP + password dev on portal/Android/iOS | Insights + lifecycle + staff + supply transition + payload override rebalance + transfer transition + loading bay + dispatch + exceptions + transfer create SSMR markers; `PX_E2E_FACTORY_FIREBASE_OTP_OK` when test token set |
| PAYLOAD | terminal, iOS, Android | payloaderroutes, platformroutes, returnsroutes | Manifest lifecycle + reassign + device-token + Firebase OTP + manifest barcode scan | **Wired** — lifecycle APIs; phone OTP + PIN dev on all clients; catalog barcode checklist + inject scan; inbound returns EAN on all three | SSMR payload + lifecycle + reassign + driver gate + payloader device-token sub-markers |

## SUPPLIER — screen map

| Pegasus (`supplier-portal`) | pegasusX (`supplier-portal`) | Backend | Status |
|--------------------------|------------------------------|---------|--------|
| `/auth/register` | `/auth/register` | POST `/v1/auth/supplier/register` | Wired |
| `/setup/billing` | `/setup/billing` | POST `/v1/supplier/billing/setup` | Wired |
| `/supplier/dashboard` | `/(portal)/dashboard` | GET `/v1/supplier/dashboard` | Wired |
| `/supplier/orders` | `/(portal)/orders` | GET `/v1/supplier/orders` | Wired |
| `/fleet`, `/supplier/fleet` | `/(portal)/fleet` + dashboard `FleetLiveMap` | GET/POST `/v1/supplier/fleet/*`, `GET /v1/supplier/fleet/live-map` | Wired via org-fleet + live map |
| `/supplier/dispatch` | `/(portal)/dispatch` | dispatch preview (warehouse) | Wired — route map on portal + Android/iOS |
| `/supplier/inventory` | `/(portal)/inventory` | GET/PATCH `/v1/supplier/inventory` | Wired |
| `/supplier/pricing` | `/(portal)/pricing` | GET/PATCH `/v1/supplier/pricing/rules` | Wired |
| `/supplier/pricing/retailer-overrides` | `/(portal)/pricing/retailer-overrides` | GET/POST/DELETE `/v1/supplier/pricing/retailer-overrides` | Wired — portal + Android + iOS |
| `/supplier/catalog` | `/(portal)/catalog` + `/(portal)/catalog/[productId]` | catalog routes (supplier) | Wired — detail + inline edit |
| `/supplier/manifests` | `/(portal)/manifests` | factory/payload cross-read | Portal page |
| `/treasury/*` | `/(portal)/treasury` | payment/settlement | Portal page |
| `/supplier/analytics` | `/(portal)/analytics` + `/analytics/demand` | analytics + demand history | Wired — portal + native demand chart |
| `/inventory/import` | `/(portal)/inventory/import` | import session wizard | Wired — portal + Android + iOS |
| Native register / business / chargebacks | `Register*` / `BusinessSetup*` / `Chargebacks*` | auth + business + payment | Wired — no portal handoff |
| `/supplier/geo-report` | `/(portal)/geo-report` | H3 coverage | Wired (ops guide) |
| `/supplier/delivery-zones` | `/(portal)/delivery-zones` | perimeter / topology | Wired |
| `/supplier/factories` | `/(portal)/factories` | topology | Wired |
| `/supplier/warehouses` | `/(portal)/warehouses` | topology | Wired |
| `/supplier/profile` | `/(portal)/profile` | GET/PUT profile | Wired |
| `/supplier/returns` | `/(portal)/returns` | order returns | Wired (handoff) |
| `/org-fleet` | `/org-fleet` | org + fleet | Wired |
| `/payments`, `/earnings` | `/payments`, `/earnings` | payment + earnings | Wired |
| `/ai/recommendations` | `/ai/recommendations` | AI recommendations | Wired |

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
| Negotiation | `order/negotiation.go` | Dispatcher | Yes | Supplier portal + driver |
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
| Supply requests | — | — | arrive (Android + iOS) | portal + Android + iOS | portal + Android + iOS | — |
| Manifest lifecycle | portal read | — | — | — | portal + Android + iOS | terminal + Android + iOS |
| Reassign (redispatch) | — | — | — | — | — | terminal + Android + iOS (`/v1/payloader/reassign-order`) |
| Treasury / invoices | portal | desktop | — | portal + Android + iOS | portal | — |
| Notifications inbox | portal + Android + iOS (dashboard bell on both native) | mobile + desktop | Android + iOS | portal + Android + iOS | portal + Android (+ iOS sheet) | Android + iOS |
| Client policy banner | all clients (global shell) | all clients (global shell) | all clients (global shell) | all clients (global shell) | portal + native | all clients (global shell) |
| Idempotency on mutations | dispatch, resolve, broadcast, payment-bypass | checkout, orders | delivery edges, amend, transition, supply arrive | dispatch, supply, inbound gate, order delay/reject/overflow | manifest, transfer | seal, reassign, inbound |
| Dock inbound queue (supplier-grouped, QR reveal) | — | desktop + Android + iOS | — | — | — | — |

**Intentional portal-only deferrals (v1):** supplier empathy adoption depth on native; warehouse supply forecast create form depth on native (create from Dispatch tab); factory iOS analytics/exceptions as dashboard sheets not tabs.

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

SUPPLIER creates warehouse/factory topology only. WAREHOUSE_ADMIN owns fleet CRUD, dispatch, and capacity overrides. PAYLOAD seals per truck then batch-activates drivers. DRIVER consumes assignment via profile + hubs.

Diagrams: `pegasusX/assets/diagrams/pegasusx-supplier-topology-dependency.mmd`, `pegasusx-fleet-dispatch-capacity-flow.mmd`, `pegasusx-payload-seal-multi-truck.mmd`.
