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
- Supplier geo-planning route composition: `pegasus/apps/backend-go/proximityroutes/routes.go`
	- Owns `GET /v1/supplier/serving-warehouse`, `GET /v1/supplier/geo-report`, `GET /v1/supplier/zone-preview`, `POST /v1/supplier/warehouses/validate-coverage`, and `GET /v1/supplier/warehouse-loads`
	- Current portal consumers are `app/supplier/geo-report/page.tsx`, `app/supplier/warehouses/CoverageEditor.tsx`, and `components/warehouse/CoverageMap.tsx`; the remaining endpoints stay supplier-facing support surfaces for coverage and load planning
- Warehouse ops compatibility layer: `pegasus/apps/backend-go/warehouse/inventory.go`, `pegasus/apps/backend-go/warehouse/staff.go`, `pegasus/apps/backend-go/warehouse/vehicles.go`
	- Keeps `GET/PATCH /v1/warehouse/ops/inventory`, `GET/POST /v1/warehouse/ops/staff`, `GET/POST /v1/warehouse/ops/drivers`, `PATCH /v1/warehouse/ops/drivers/{id}/assign-vehicle`, and `GET/POST/PATCH /v1/warehouse/ops/vehicles` additive across warehouse portal, warehouse iOS, and warehouse Android
	- Inventory accepts `q` and `search`, accepts `sku_id` or `product_id` on mutation, and returns both `inventory` and `items` with `sku_id`/`product_id` aliases
	- Staff create accepts an optional PIN and returns the effective one-time PIN; vehicle responses expose both `max_volume_vu` and `capacity_vu` plus a derived `status`
	- Fleet controls now let warehouse admins assign or reset driver vehicles and toggle vehicle availability from portal, iOS, and Android against the same backend contract; dispatch preview excludes inactive vehicles from available-driver output
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
