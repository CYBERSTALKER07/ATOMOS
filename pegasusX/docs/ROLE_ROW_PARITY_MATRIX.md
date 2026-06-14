# pegasusX Role-Row Parity Matrix

> **Canonical cross-role spec:** [`FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`](./FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md) — use this matrix for screen-level parity; use the master plan for end-to-end flows, comms, and verification gates.

Last updated: 2026-06-11. Canonical reference: `pegasus/`. Delivery tree: `pegasusX/`.

## Summary

| Role | pegasusX clients | Backend routes | Production v1 capability | UI parity (vs Pegasus) | E2E (SSMR) |
|------|------------------|----------------|--------------------------|------------------------|------------|
| SUPPLIER | supplier-portal, native iOS/Android | supplierroutes | Portal: full ops spine + fleet live map (`GET /v1/supplier/fleet/live-map`); native: fleet live map on iOS + Android MapLibre with animated driver markers; P1-01 ops panels on More hub | Partial — portal 26+ routes vs Pegasus ~59; fleet live map + animated GPS on portal + both native apps | Full SSMR e2e incl. payment + factory |
| RETAILER | desktop, iOS, Android | retailerroutes, orderroutes, catalogroutes | Order + tracking + catalog; category-suppliers (PX12-B); checkout via unified (PX12-G); P1-02 catalog browse + vendor connect on Android+iOS | Desktop procurement + mobile catalog/supplier parity (connect/search/remove, all-products browse); portal-only ops deferrals | Register, order create, tracking, `PX_E2E_CATALOG_OK` |
| DRIVER | Android, iOS | driverroutes, orderroutes, telemetryroutes | Full delivery edges + reorder (PX12-B); planned route geometry + turn-by-turn + off-route reroute (`GET /v1/fleet/route/{routeID}/geometry`); maps + WS (PX12-F) | Live-ops + planned/breadcrumb map overlays | Telemetry; shop-closed; negotiation; driver edges E2E |
| WAREHOUSE | portal, Android, iOS | warehouseroutes | Ops dashboard + dispatch live fleet map (`GET /v1/warehouse/ops/fleet/live-map`); dispatch lock, supply requests; P1-03 ops API | Portal + native fleet live map on dashboard + dispatch (MapLibre Android, MapKit iOS); animated driver markers on all three; native order detail mutations (Android+iOS) | Dispatch lock + order mutation + transfer actions SSMR markers |
| FACTORY | portal, Android, iOS | factoryroutes | Manifests + supply requests + dispatch (PX12-J) | Portal 13 routes; insights P1; manifest exception inbox + transfer create P2; manifest lifecycle + staff detail P3; supply-request transitions + payload override cross-manifest rebalance P4; loading-bay transfer filter/detail/dispatch + transfer state machine P5 on all factory clients | Insights + lifecycle + staff + supply transition + payload override rebalance + transfer transition + loading bay + dispatch + exceptions + transfer create SSMR markers |
| PAYLOAD | terminal, iOS, Android | payloaderroutes, platformroutes | Manifest lifecycle + reassign + device-token (PX12-K); canonical `/v1/payloader/manifests/*` on all three clients | Lifecycle APIs; unified `manifest.Wire`; Expo + tablet native path parity | SSMR payload + lifecycle + reassign + driver gate + payloader device-token sub-markers |

## SUPPLIER — screen map

| Pegasus (`admin-portal`) | pegasusX (`supplier-portal`) | Backend | Status |
|--------------------------|------------------------------|---------|--------|
| `/auth/register` | `/auth/register` | POST `/v1/auth/supplier/register` | Wired |
| `/setup/billing` | `/setup/billing` | POST `/v1/supplier/billing/setup` | Wired |
| `/supplier/dashboard` | `/(portal)/dashboard` | GET `/v1/supplier/dashboard` | Wired |
| `/supplier/orders` | `/(portal)/orders` | GET `/v1/supplier/orders` | Wired |
| `/fleet`, `/supplier/fleet` | `/(portal)/fleet` + dashboard `FleetLiveMap` | GET/POST `/v1/supplier/fleet/*`, `GET /v1/supplier/fleet/live-map` | Wired via org-fleet + live map |
| `/supplier/dispatch` | `/(portal)/dispatch` | dispatch preview (warehouse) | Partial |
| `/supplier/inventory` | `/(portal)/inventory` | GET/PATCH `/v1/supplier/inventory` | Wired |
| `/supplier/pricing` | `/(portal)/pricing` | GET/PATCH `/v1/supplier/pricing/rules` | Wired |
| `/supplier/catalog` | `/(portal)/catalog` | catalog routes (supplier) | Portal page |
| `/supplier/manifests` | `/(portal)/manifests` | factory/payload cross-read | Portal page |
| `/treasury/*` | `/(portal)/treasury` | payment/settlement | Portal page |
| `/supplier/analytics` | `/(portal)/analytics` | analytics (partial) | Portal page |
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
| Supplier portal | ~59 routes | Portal full; iOS/Android ops slice | E2 adds screens per Boss approval |
| Retailer desktop | Full procurement | Desktop richest; mobile tracking-first | Intentional until E2 |
| Factory / warehouse | Full portal | Portal + mobile ops dashboards | Treasury depth portal-only |

## Verification commands

```bash
cd pegasusX && make test-ssmr-infra
cd pegasusX && make validate-launch-readiness
cd pegasusX && bash scripts/parity/role_row_contract_check.sh
cd pegasusX && make backend-build
```

## SSMR fleet / dispatch / payload feature IDs (2026-06-14)

| ID | Capability | Owner row | Backend | Clients | E2E marker |
|----|------------|-----------|---------|---------|------------|
| PX-FLEET-001 | Warehouse driver+vehicle CRUD, assign/re-sign, active-route guard | WAREHOUSE_ADMIN | `warehouse/ops_portal.go`, `warehouse/fleet_guards.go`, `warehouse/fleet_ops.go` | warehouse-portal `/drivers`, native ops | `PX_E2E_WAREHOUSE_FLEET_MGMT_OK` |
| PX-DISP-002 | Manual dispatch capacity warnings + suggested unselect + force audit | WAREHOUSE_ADMIN | `warehouse/dispatch_execute.go`, `dispatch/capacity_recommend.go` | warehouse-portal + warehouse Android/iOS dispatch | `PX_E2E_DISPATCH_CAPACITY_OK` |
| PX-PAY-003 | Per-truck seal + aggregate `seal-completed` batch | PAYLOAD | `payload/service.go`, `payloaderroutes` | payload-terminal + payload Android/iOS tablet apps | `PX_E2E_PAYLOAD_SEAL_FLOWS_OK` |
| PX-REAS-004 | Durable payload reassign + recommend | PAYLOAD | `payload/service.go` | payload-terminal | `PX_E2E_REASSIGN_FLOWS_OK` |
| PX-DRV-005 | Driver profile `vehicle_id` + WS assign detection | DRIVER | `driver/service.go`, bootstrap profile lookup | driver Android/iOS profile VM | `PX_E2E_DRIVER_ASSIGN_DETECTION_OK` |

### Topology dependency (SUPPLIER root)

SUPPLIER creates warehouse/factory topology only. WAREHOUSE_ADMIN owns fleet CRUD, dispatch, and capacity overrides. PAYLOAD seals per truck then batch-activates drivers. DRIVER consumes assignment via profile + hubs.

Diagrams: `pegasusX/assets/diagrams/pegasusx-supplier-topology-dependency.mmd`, `pegasusx-fleet-dispatch-capacity-flow.mmd`, `pegasusx-payload-seal-multi-truck.mmd`.
