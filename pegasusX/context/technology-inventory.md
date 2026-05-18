# pegasusX Technology Inventory

## Backend
- **Language**: Go 1.22+
- **Router**: `github.com/go-chi/chi/v5`
- **DB**: Google Cloud Spanner (local: emulator via docker-compose)
- **Cache**: Redis 7 (with Pub/Sub invalidation channel `cache:invalidate`)
- **Events**: Apache Kafka (transactional outbox; sync writer with `RequiredAcks=all`)
- **Logging**: `log/slog` (JSON handler, `trace_id` propagated)
- **Trace ingress middleware**: `bootstrap.TraceMiddleware` normalizes/generates `X-Trace-Id`, propagates it through request context, and enables additive `trace_id` insertion in map-based outbox payloads.
- **Scaffold outbox runtime**: `bootstrap` now starts `OutboxRelay` with store authority selected at runtime: `outbox/spanner_store.go` (`OutboxEvents` fetch/mark) when Spanner is reachable, with in-memory store fallback when unavailable; repository emit paths use a shared outbox appender seam.
- **Adapter bridge (additive)**: `bootstrap` now attempts Redis transport via `cache/redis_backend.go` and Kafka outbox transport via `outbox/kafka_publisher.go`, with fail-open fallback to in-memory cache/logging publisher when dependencies are unavailable.
- **Strict reliability mode (additive)**: `REQUIRE_INFRA_ADAPTERS=true` enforces fail-fast bootstrap if Redis or Kafka adapter initialization fails; default mode remains fail-open for local/scaffold startup.
- **Request reliability middleware (additive)**: `bootstrap/reliability_middleware.go` applies fixed-window rate limits, priority-based in-flight shedding, and per-class circuit-open responses; mounted in `main.go` immediately after trace middleware and controlled by `RELIABILITY_MIDDLEWARE_ENABLED`.
- **Payment execution router (additive)**: `payment/execution.go` routes provider actions with bounded retries (exponential backoff + jitter), typed policy errors, and gateway capability controls; checkout/chargeback/reversal handlers in `payment/service.go` now invoke this seam before repository persistence/outbox emission.
- **Checkout attempt metadata persistence (additive)**: `payment/service.go` now persists `PaymentAttemptRecord` via repository `SaveAttempt` after session creation and emits additive execution metadata (`attempt_id`, `execution_action`, `execution_mode`, `provider_reference`) in checkout responses and payment-required event payloads.
- **AIRWALLEX direct execution flag**: `AIRWALLEX_DIRECT_EXECUTION_ENABLED` gates AIRWALLEX execution support in the payment router.
- **Strict-mode startup coverage**: `bootstrap/bootstrap_test.go` validates strict-mode fail-fast for missing Redis/Kafka adapters and strict-mode startup success when both adapters are healthy.
- **Outbox Spanner integration coverage**: `outbox/spanner_store_integration_test.go` verifies `Append -> Fetch -> MarkPublished` lifecycle against emulator-backed Spanner when `SPANNER_EMULATOR_HOST` and schema are available (skips gracefully otherwise).
- **WebSocket relay transport**: `ws/hub.go` now relays typed fanout envelopes over `ws:<hub>:fanout` with source-instance suppression; `main.go` starts all role-hub relay subscribers during boot.
- **AuthN**: JWT cookie sessions + optional Firebase ID token verification (RS256 + Google cert rotation cache)
- **Geospatial**: H3 resolution 7, 15-char hex on the wire
- **Workspace**: `go.work` at `pegasusX/` root listing `./apps/backend-go`, `./apps/ai-worker`, `./packages/config`
- **Route coverage (additive 2026-05-17)**:
	- `supplierroutes`: register + supplier core operational surfaces (configure, profile, dashboard, earnings, inventory, inventory-audit, orders, vet).
	- `retailerroutes`: register + retailer core operational surfaces (profile, suppliers, cart sync, order cancel/request-cancel, analytics, family-members, AI confirm/reject, preorder, pending-payments, active-fulfillment, tracking).
	- `driverroutes`: driver core operational surfaces (profile, history, earnings, availability, pending-collections, manifest-gate, manifest, legacy fleet manifest alias).
	- `warehouseroutes`: warehouse core operational surfaces (ops dashboard, inventory, orders, dispatch preview, demand forecast, supply requests, dispatch locks).
	- `factoryroutes`: factory core operational surfaces (analytics overview, dashboard, profile, transfers, manifests, manifest detail, manifest lifecycle transitions start-loading/seal/dispatch/complete, rebalance/cancel-transfer/cancel, manifest exception queue, fleet drivers, fleet vehicles, staff, dispatch, supply requests).
	- `payloaderroutes`: payload core operational surfaces (trucks, orders, manifests list/detail, lifecycle transitions start-loading/inject-order/seal, manifest exception write/read, recommend-reassign, reassignment apply, compatibility seal).
	- `paymentroutes`: checkout and payment-mutation surfaces (b2b/unified checkout, chargeback, chargeback reversal, deprecated global-pay initiate).
	- `webhookroutes`: gateway callback surfaces (global-pay, adyen, stripe).
	- Retailer protected endpoints are Firebase bearer + `RequireRole(RETAILER)` gated when `FIREBASE_AUTH_ENABLED=true` and verifier wiring is present.
	- Driver protected endpoints are Firebase bearer + `RequireRole(DRIVER)` gated when `FIREBASE_AUTH_ENABLED=true` and verifier wiring is present.
	- Warehouse protected endpoints are Firebase bearer + `RequireRole(WAREHOUSE_ADMIN|ADMIN)` gated when enabled, with cookie `ADMIN` fallback for local scaffold mode.
	- Factory protected endpoints are Firebase bearer + `RequireRole(FACTORY_ADMIN|ADMIN)` gated when enabled, with cookie `ADMIN` fallback for local scaffold mode.
	- Payload protected endpoints are Firebase bearer + `RequireRole(PAYLOAD|ADMIN)` gated when enabled, with cookie `ADMIN` fallback for local scaffold mode.
	- Payment protected endpoints are split by role when enabled: checkout uses Firebase bearer + `RequireRole(RETAILER)`, while payment mutations use Firebase bearer + `RequireRole(ADMIN)` with cookie `ADMIN` fallback for local scaffold mode.
	- Webhook endpoints are JWT-unauthenticated and rely on provider-exact signature verification + transaction-id idempotency (Global Pay Basic auth credentials, Stripe `Stripe-Signature` `t`/`v1`, Adyen item-level `additionalData.hmacSignature`).
	- Factory and payload manifest mutation paths execute through repository `Apply` seams with outbox emit in the same path, followed by cache invalidation and websocket fanout parity.
	- Factory/payload manifest event payload parity now includes additive routing metadata (`route_id`, `driver_id`, `vehicle_id`, `order_count`) on sealed lifecycle events, and payload reassignment envelopes now include `from_manifest_id`/`to_manifest_id` plus route/driver metadata while invalidating both source and target manifest cache keys.

## Web
- **Framework**: Next.js 15 (App Router) + React 19
- **Styling**: Tailwind v4 + hand-rolled M3 CSS tokens (no `@material/web`)
- **Desktop shell**: Tauri 2 for supplier-portal, retailer-app-desktop, warehouse-portal, factory-portal
- **Charts**: Recharts
- **Maps**: MapLibre GL / Mapbox GL

## Mobile
- **Android**: Kotlin + Jetpack Compose Material 3
- **iOS**: SwiftUI + SF Symbols + system colors
- **Payload Terminal**: Expo / React Native (M3 discipline via RN styling)

## Shared Packages
- `packages/types` — TS canonical contracts
- `packages/api-client` — generated HTTP client
- `packages/validation` — Zod schemas
- `packages/i18n` — copy keys per role
- `packages/ui-kit` — shared desktop foundation + components
- `packages/config` — Go env/boot config

## Contracts
- `contracts/events.schema.json` — canonical websocket/Kafka event schema, generated by backend `cmd/gen-contracts`, consumed by native codegen (Quicktype for Android/iOS).
- Additive event-type coverage includes advanced factory/payload manifest workflow events: `MANIFEST_ORDER_INJECTED`, `MANIFEST_ORDER_EXCEPTION`, `MANIFEST_DLQ_ESCALATION`, `MANIFEST_REBALANCED`, and `MANIFEST_CANCELLED`.
- Runtime fanout coverage includes websocket propagation for manifest lifecycle/exception events emitted by factory and payload services: `MANIFEST_LOADING_STARTED`, `MANIFEST_SEALED`, `MANIFEST_ORDER_INJECTED`, `MANIFEST_ORDER_EXCEPTION`, `MANIFEST_DLQ_ESCALATION`, and `MANIFEST_REBALANCED`.

## Infra
- `infra/docker-compose.yml` — local Spanner emulator, Redis, Kafka, Kafka UI
- `infra/terraform/` — baseline cloud provisioning (VPC, Spanner, Redis, Secret Manager for Kafka/Firebase runtime wiring)
- `infra/k8s/` — deployment manifests
