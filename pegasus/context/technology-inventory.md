# Technology Inventory

Canonical inventory of tools, technologies, and external services used across Pegasus.

This file is the human-readable companion to `pegasus/context/technology-inventory.json` and must stay synchronized with it.

## Inventory Sources

- `pegasus/**/package.json`
- `pegasus/**/go.mod`
- `pegasus/**/build.gradle.kts`
- `pegasus/**/Cargo.toml`
- `pegasus/docker-compose.yml`
- `pegasus/infra/terraform/*.tf`
- keyword sweep across `pegasus/**/*.{go,ts,tsx,js,json,kts,tf,yml,yaml,md}`

## Languages And Runtimes

- Go
- TypeScript and JavaScript
- Kotlin
- Swift and SwiftUI
- Rust
- Python
- Shell

## Web And Desktop Stack

- Next.js + React (admin, factory, warehouse, retailer desktop)
- Tailwind CSS + HeroUI + Motion + Recharts + Mapbox/MapLibre
- Tauri 2 desktop shells with Rust backends
- Expo/React Native payload terminal

## Backend Core Stack

- Go services with `chi` HTTP routing and WebSockets
- Spanner data access
- Kafka eventing
- Redis cache and Pub/Sub invalidation
- Firebase integration
- OpenTelemetry and Prometheus metrics

## Runtime Contract Surfaces

- Legacy order detail compatibility handler: `pegasus/apps/backend-go/order/legacy_orders.go`
	- Owns `GET /v1/orders/{id}`, `GET /v1/orders/{id}/events`, `PATCH /v1/orders/{id}/status`, and `PATCH /v1/orders/{id}/state`
	- Serves an additive superset detail payload for driver iOS, driver Android, and retailer desktop order detail consumers, plus the supplier portal order timeline feed
- Retailer role-row route composition: `pegasus/apps/backend-go/retailerroutes/routes.go`
	- Owns `GET /v1/retailer/analytics/{expenses,detailed}`, `POST /v1/{orders/request-cancel,order/cash-checkout,order/card-checkout,retailer/shop-closed-response}`, `GET/POST/DELETE /v1/retailer/family-members*`, `POST /v1/retailer/orders/{confirm-ai,reject-ai}`, `POST /v1/orders/{edit-preorder,confirm-preorder}`, `GET/POST /v1/retailer/cart/sync`, `GET/POST /v1/retailer/suppliers*`, `GET/PUT /v1/retailer/profile`, `GET /v1/retailers/{retailerID}/orders`, `GET /v1/retailer/{tracking,cards,pending-payments,active-fulfillment}`, `POST /v1/retailer/card/{initiate,confirm,deactivate,default}`, `PATCH|GET /v1/retailer/settings/auto-order*`, and `GET /v1/ws/retailer`
	- Current role-row consumers span retailer desktop supplier/analytics/tracking/saved-card surfaces plus retailer iOS and Android order, fulfillment, payment, settings, and realtime flows, so the shared retailer contract is now extracted out of `main.go`
- Supplier geo-planning route composition: `pegasus/apps/backend-go/proximityroutes/routes.go`
	- Owns `GET /v1/supplier/serving-warehouse`, `GET /v1/supplier/geo-report`, `GET /v1/supplier/zone-preview`, `POST /v1/supplier/warehouses/validate-coverage`, and `GET /v1/supplier/warehouse-loads`
	- Current portal consumers are `app/supplier/geo-report/page.tsx`, `app/supplier/warehouses/CoverageEditor.tsx`, and `components/warehouse/CoverageMap.tsx`; the remaining endpoints stay supplier-facing support surfaces for coverage and load planning
- Supplier self-service route composition: `pegasus/apps/backend-go/supplierroutes/routes.go`
	- Owns `POST /v1/supplier/configure`, `POST /v1/supplier/billing/setup`, `GET/PUT /v1/supplier/profile`, `PATCH /v1/supplier/shift`, `GET/POST/DELETE /v1/supplier/payment-config`, `GET/POST/DELETE /v1/supplier/gateway-onboarding`, and `POST /v1/supplier/payment/recipient/register`
	- Current portal consumers span `app/setup/billing/page.tsx`, `app/supplier/profile/page.tsx`, `app/supplier/payment-config/page.tsx`, `hooks/useSupplierShift.tsx`, and supplier profile readers in product-management screens
- Supplier warehouse-ops route composition: `pegasus/apps/backend-go/supplierroutes/routes.go`
	- Owns `GET /v1/supplier/org/members`, `POST /v1/supplier/org/members/invite`, `PUT/DELETE /v1/supplier/org/members/{id}`, `GET/POST /v1/supplier/staff/payloader`, `POST /v1/supplier/staff/payloader/{id}/rotate-pin`, `GET /v1/supplier/warehouse-staff`, `PATCH /v1/supplier/warehouse-staff/{id}`, `GET/POST /v1/supplier/warehouses`, `GET/PUT/DELETE /v1/supplier/warehouses/{id}`, `POST /v1/supplier/warehouses/{id}/coverage`, and `GET /v1/supplier/warehouse-inflight-vu`
	- Current portal consumers span `app/supplier/org/page.tsx`, `app/supplier/staff/page.tsx`, `app/supplier/warehouses/page.tsx`, `app/supplier/warehouses/WarehouseStaffPanel.tsx`, `app/supplier/warehouses/CoverageEditor.tsx`, and `components/factory/FactoryNetworkMap.tsx`
	- Coverage saves now update warehouse H3 coverage through the same warehouse spatial outbox + cache refresh path as coordinate edits, so portal coverage editing is no longer a dead-end mutation
- Supplier catalog-pricing route composition: `pegasus/apps/backend-go/suppliercatalogroutes/routes.go`
	- Owns `GET /v1/supplier/products/upload-ticket`, `GET/POST /v1/supplier/products`, `GET/PUT/DELETE /v1/supplier/products/{sku_id}`, `GET/POST /v1/supplier/pricing/rules`, `DELETE /v1/supplier/pricing/rules/{tier_id}`, `GET/POST /v1/supplier/pricing/retailer-overrides`, and `DELETE /v1/supplier/pricing/retailer-overrides/{id}`
	- Current portal consumers span `components/SupplierProductForm.tsx`, `components/SupplierPromotionForm.tsx`, `app/supplier/products/page.tsx`, `app/supplier/products/[sku_id]/page.tsx`, `app/supplier/catalog/page.tsx`, `app/supplier/pricing/page.tsx`, and `app/supplier/pricing/retailer-overrides/page.tsx`
	- Pricing-rule GET support is preserved through the existing `PricingService.HandleUpsertPricingRule` method while route ownership moves out of `main.go`
- Supplier logistics route composition: `pegasus/apps/backend-go/supplierlogisticsroutes/routes.go`
	- Owns `GET /v1/supplier/picking-manifests`, `GET /v1/supplier/picking-manifests/orders`, `GET /v1/supplier/manifests`, `GET /v1/supplier/manifests/{id}`, `POST /v1/supplier/manifests/{id}/{start-loading|seal|inject-order}`, `POST /v1/payload/manifest-exception`, `GET /v1/supplier/manifest-exceptions`, `POST /v1/supplier/manifests/{auto-dispatch|dispatch-recommend|manual-dispatch}`, `GET /v1/supplier/manifests/waiting-room`, `GET /v1/supplier/fleet-volumetrics`, `POST /v1/supplier/dispatch-queue`, and `GET /v1/supplier/dispatch-preview`
	- Current portal consumers span `app/supplier/manifests/page.tsx`, `app/supplier/manifest-exceptions/page.tsx`, `app/supplier/dispatch/page.tsx`, and the supplier orders page auto-dispatch trigger
	- Includes the payload-facing manifest exception entrypoint so the supplier and payload roles keep one manifest exception contract while route ownership moves out of `main.go`
- Payload loading role-row surface: `pegasus/apps/backend-go/payloaderroutes/routes.go`, `pegasus/apps/backend-go/supplierlogisticsroutes/routes.go`, `pegasus/apps/backend-go/main.go`, `pegasus/apps/backend-go/ws/payloader_hub.go`
	- Owns `POST /v1/auth/payloader/login`, `GET /v1/payloader/trucks`, `GET /v1/payloader/orders`, `POST /v1/payloader/recommend-reassign`, the shared `GET /v1/supplier/manifests*` and `POST /v1/supplier/manifests/{id}/{start-loading|seal|inject-order}` lifecycle routes, `POST /v1/payload/{manifest-exception,seal}`, `GET /v1/user/notifications`, `POST /v1/user/notifications/read`, and `/v1/ws/payloader`
	- Current payload consumers span `apps/payload-terminal/App.tsx`, `apps/payload-app-ios/payload-app-ios/Services/APIClient.swift`, `apps/payload-app-ios/payload-app-ios/Services/WebSocketClient.swift`, `apps/payload-app-android/app/src/main/java/com/pegasus/payload/data/remote/PayloadApi.kt`, and `apps/payload-app-android/app/src/main/java/com/pegasus/payload/services/PayloadWebSocket.kt`
	- Shared supplier manifest routes and `/v1/ws/payloader` now admit `PAYLOADER`, keeping Expo, iOS, and Android payload clients aligned with the `SupplierTruckManifests` lifecycle contract
	- The payloader websocket now distinguishes `PUSH` notification frames from `PAYLOAD_SYNC` refresh frames so payload clients silently reload active manifest data on external overrides instead of surfacing empty notifications
	- `PAYLOAD_SYNC` is emitted atomically on draft creation plus the supplier manifest `start-loading`, `inject-order`, `seal`, and `manifest-exception` mutation paths so other payload surfaces stay coherent after cross-device or supplier-portal changes
- Supplier insights route composition: `pegasus/apps/backend-go/supplierinsightsroutes/routes.go`
	- Owns `GET/PUT /v1/supplier/country-overrides`, `GET/DELETE /v1/supplier/country-overrides/{code}`, `GET /v1/supplier/analytics/{velocity,demand/today,demand/history,transit-heatmap,throughput,load-distribution,node-efficiency,sla-health,revenue,top-retailers}`, `GET /v1/supplier/financials`, and `GET /v1/supplier/crm/retailers*`
	- Current portal consumers span `app/supplier/country-overrides/page.tsx`, `app/supplier/analytics/page.tsx`, `app/supplier/analytics/demand/page.tsx`, `app/supplier/dashboard/page.tsx`, `hooks/useAnalytics.ts`, `hooks/useAdvancedAnalytics.ts`, and `app/supplier/crm/page.tsx`
	- Supplier CRM list/detail payloads now include additive retailer `email` for the supplier portal contact drawer
	- Groups supplier read-side settings, analytics, financials, and CRM under one extracted contract while preserving the existing handler ownership in `countrycfg`, `analytics`, and `supplier`
- Supplier operations route composition: `pegasus/apps/backend-go/supplieroperationsroutes/routes.go`
	- Owns `GET/POST /v1/supplier/fleet/drivers`, `GET/PATCH/POST /v1/supplier/fleet/drivers/{id}`, `GET/POST /v1/supplier/fleet/vehicles`, `GET/PATCH/DELETE /v1/supplier/fleet/vehicles/{id}`, `POST /v1/supplier/fulfillment/pay`, `GET /v1/supplier/returns`, `POST /v1/supplier/returns/resolve`, `GET /v1/supplier/quarantine-stock`, and `POST /v1/inventory/reconcile-returns`
	- Current portal consumers span `app/supplier/fleet/page.tsx`, the legacy supplier driver fetch on `app/page.tsx`, `app/supplier/returns/page.tsx`, and `app/supplier/depot-reconciliation/page.tsx`
	- Preserves the existing supplier fulfillment-pay error mapping while grouping fleet and reverse-logistics surfaces under one extracted route owner
- Supplier planning route composition: `pegasus/apps/backend-go/supplierplanningroutes/routes.go`
	- Owns `GET/POST /v1/supplier/delivery-zones`, `PUT/DELETE /v1/supplier/delivery-zones/{id}`, `GET/POST /v1/supplier/factories`, `GET/PATCH/DELETE /v1/supplier/factories/{id}`, `GET /v1/supplier/factories/{recommend-warehouses,optimal-assignments}`, `GET /v1/supplier/geocode/reverse`, `GET /v1/supplier/retailers/locations`, `GET/POST /v1/supplier/supply-lanes`, `GET/PATCH/DELETE /v1/supplier/supply-lanes/{id}`, `GET/PUT /v1/supplier/network-mode`, `GET /v1/supplier/network-analytics`, `POST /v1/supplier/replenishment/{kill-switch,pull-matrix,predictive-push}`, `GET /v1/supplier/replenishment/audit`, and `GET/POST /v1/supplier/warehouses/{territory-preview,apply-territory}`
	- Supports the supplier map/planning loop: delivery-zone administration, supplier→factory recommendation, retailer location overlays, supply-lane management, network optimization controls, replenishment audit/triggers, and territory reassignment
	- Supplier factory create/list/detail/profile payloads now persist and return additive `h3_index` and `product_types` metadata for the supplier factory-planning surfaces
	- Supply-lane mutate actions now resolve organisation supplier scope via `claims.ResolveSupplierID()` and honor `PATCH` for plain lane updates so planning clients stop hitting success-shaped no-ops
	- Keeps cron startup in `main.go` but removes the HTTP route composition for those planning surfaces from the monolith
- Warehouse ops compatibility layer: `pegasus/apps/backend-go/warehouse/inventory.go`, `pegasus/apps/backend-go/warehouse/staff.go`, `pegasus/apps/backend-go/warehouse/vehicles.go`
	- Keeps `GET/PATCH /v1/warehouse/ops/inventory`, `GET/POST /v1/warehouse/ops/staff`, `GET/POST /v1/warehouse/ops/drivers`, `PATCH /v1/warehouse/ops/drivers/{id}/assign-vehicle`, and `GET/POST/PATCH /v1/warehouse/ops/vehicles` additive across warehouse portal, warehouse iOS, and warehouse Android
	- Inventory accepts `q` and `search`, accepts `sku_id` or `product_id` on mutation, and returns both `inventory` and `items` with `sku_id`/`product_id` aliases
	- Staff create accepts an optional PIN and returns the effective one-time PIN; vehicle responses expose both `max_volume_vu` and `capacity_vu` plus a derived `status`
	- Fleet controls now let warehouse admins assign or reset driver vehicles and toggle vehicle availability from portal, iOS, and Android against the same backend contract; dispatch preview excludes inactive vehicles from available-driver output
	- Vehicle availability is schema-backed with `Vehicles.UnavailableReason`, and portal plus native warehouse clients surface the persisted reason when a truck is unavailable
	- Driver list payloads now carry assigned vehicle availability metadata, and dispatch preview publishes both `available_drivers` and `unavailable_drivers` so warehouse clients can explain why an assigned driver is blocked by vehicle availability
- Warehouse live websocket surface: `/ws/warehouse`
	- Owned by `pegasus/apps/backend-go/ws/warehouse_hub.go` with post-commit emitters in `pegasus/apps/backend-go/warehouse/supply_requests.go` and `pegasus/apps/backend-go/warehouse/dispatch_lock.go`
	- Emits `SUPPLY_REQUEST_UPDATE` and `DISPATCH_LOCK_CHANGE` frames with `warehouse_id` and `timestamp`
	- Consumed by `pegasus/apps/warehouse-portal/app/supply-requests/page.tsx`, `pegasus/apps/warehouse-portal/app/supply-requests/[id]/page.tsx`, `pegasus/apps/warehouse-portal/app/dispatch-locks/page.tsx`, `pegasus/apps/warehouse-app-ios/WarehouseApp/Views/Dispatch/DispatchView.swift`, and `pegasus/apps/warehouse-app-android/app/src/main/java/com/pegasus/warehouse/ui/screens/dispatch/DispatchScreen.kt`
	- Client helpers in portal, iOS, and Android now auto-reconnect and surface reconnecting/offline state instead of requiring a manual screen reopen
- Warehouse dispatch mutation surface: `/v1/warehouse/supply-requests` and `/v1/warehouse/dispatch-lock*`
	- Owned by `pegasus/apps/backend-go/warehouse/supply_requests.go` and `pegasus/apps/backend-go/warehouse/dispatch_lock.go`
	- Mobile and portal dispatch surfaces create demand-forecast-backed supply requests, cancel warehouse-owned requests, acquire `MANUAL_DISPATCH` locks, and release active locks through the same additive contract

## Android Stack

- Jetpack Compose + Material 3
- Hilt DI + Retrofit/OkHttp networking
- Room + DataStore local state
- Firebase Auth/Messaging + Google Maps Compose
- CameraX + ML Kit barcode scanning

## iOS Stack

- SwiftUI native apps
- APNs push channel and Apple-native design patterns

## Data, Messaging, And Local Emulators

From `pegasus/docker-compose.yml`:

- Kafka (KRaft) + Kafka UI + topic bootstrap job
- Redis
- Spanner emulator
- Firebase Auth emulator
- WireMock Global Pay mock

## Cloud And Infrastructure Services

From Terraform under `pegasus/infra/terraform`:

- Cloud Run
- GKE + Workload Identity + KEDA via Helm
- Cloud Spanner + Memorystore Redis
- Cloud Armor + Cloud CDN + Cloud DNS + Cloud NAT + Private Service Connect
- Artifact Registry
- Google Cloud Monitoring alert policies and uptime checks
- Multi-region Spanner/GKE topology options

## Security And Reliability Patterns

- Transactional outbox for durable state-change events
- Redis Pub/Sub invalidation after commit
- Maglev-style consistent hash affinity
- Cloud Armor WAF + OWASP rules + per-IP throttling
- Circuit-breaker and priority-guard readiness patterns

## Engineering Guard Tooling

- Contract Guard MCP: `pegasus/scripts/contract_guard_mcp.py`
	- Enforces codebase-first MCP context weighting on contract-triggered diffs (runtime code surfaces must dominate context-doc touches).
- Architecture Guard MCP: `pegasus/scripts/architecture_guard_mcp.py`
	- Enforces codebase-first MCP context weighting on architecture-triggered diffs (runtime code surfaces must dominate context-doc touches).
- Design System Guard MCP: `pegasus/scripts/design_system_guard_mcp.py`
	- Enforces codebase-first MCP context weighting on design-triggered diffs (runtime code surfaces must dominate context-doc touches).
- Production Safety Guard: `pegasus/scripts/production_safety_guard.py`
- Visual + Test Intelligence Guard: `pegasus/scripts/visual_test_intelligence_guard.py`
- Security Guard: `pegasus/scripts/security_guard.py`
- Aggregated PR workflow: `.github/workflows/one-eye-guards.yml`

## External Integrations And Providers

- Payme
- Click
- Global Pay
- Stripe
- Telegram
- Firebase
- Google Maps ecosystem

## Sync Contract

If any feature, dependency, service, or runtime changes, update all of:

1. `pegasus/context/technology-inventory.md`
2. `pegasus/context/technology-inventory.json`
3. `.github/ACT.md`
4. `.github/copilot-instructions.md`
5. `.github/gemini-instructions.md`
6. `pegasus/context/architecture.md`
7. `pegasus/context/architecture-graph.json`
- Supplier core route composition: `pegasus/apps/backend-go/suppliercoreroutes/routes.go`
	- Owns `GET /v1/supplier/dashboard`, `GET /v1/supplier/earnings`, `GET/PATCH /v1/supplier/inventory`, `GET /v1/supplier/inventory/audit`, `GET /v1/supplier/orders`, and `POST /v1/supplier/orders/vet`
	- Supports the supplier core portal loop: dashboard metrics, earnings analytics, inventory management, and supplier-side order approval
	- Supplier inventory now honors the mounted root `PATCH /v1/supplier/inventory` contract and returns additive `sku_id`/`product_name` aliases on inventory and audit rows for the supplier portal
	- Removes the final inline `/v1/supplier/*` registrations from `backend-go/main.go`
