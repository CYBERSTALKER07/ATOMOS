# pegasusX Technology Inventory

## Planning and governance

- `context/plan.md` — canonical phased execution roadmap and reconciliation ledger for pegasusX; every non-trivial implementation batch maps to plan anchors there and updates status when execution reality changes.

## Backend
- **Language**: Go 1.22+
- **Router**: `github.com/go-chi/chi/v5`
- **DB**: Google Cloud Spanner (local: emulator via docker-compose)
- **Cache**: Redis 7 (with Pub/Sub invalidation channel `cache:invalidate`)
- **Events**: Apache Kafka (transactional outbox; sync writer with `RequiredAcks=all`)
- **Logging**: `log/slog` (JSON handler, `trace_id` propagated)
- **Trace ingress middleware**: `bootstrap.TraceMiddleware` normalizes/generates `X-Trace-Id`, propagates it through request context, and enables additive `trace_id` insertion in outbox payloads for both typed structs and map-based payloads.
- **Sandbox bootstrap command (additive)**: `apps/backend-go/cmd/setup` creates the isolated Spanner instance/database when needed, applies `apps/backend-go/schema/spanner.ddl`, and seeds the single supplier row for local SSMR sandbox bring-up.
- **Smokecheck command (additive)**: `apps/backend-go/cmd/ssmr-smokecheck` asserts seeded supplier + `Retailers` schema state against isolated Spanner, validates Redis-backed perimeter readiness/membership (`spatial` mode), and validates Kafka topic isolation + round-trip flow.
- **Smoke harness (additive)**: `scripts/smoke_ssmr.sh` orchestrates isolated compose bring-up, topic bootstrap, idempotent sandbox bootstrap, Redis ping, backend health, and direct Spanner/spatial/Kafka proof; `make test-ssmr-infra` and `npm run infra:ssmr:test` are thin entrypoints over the same harness.
- **Persistent Go caches (additive)**: `infra/docker-compose.ssmr.yml` shares named `pegasusx-ssmr-go-mod` and `pegasusx-ssmr-go-build` volumes across Go containers, and `scripts/smoke_ssmr.sh` preserves them by tearing the stack down without `-v`.
- **CI stage gate (additive)**: repo-root `.github/workflows/ssmr-infra.yml` runs `make test-ssmr-infra` on `pegasusX/**` push/pull-request changes and manual dispatches.
- **Scaffold outbox runtime**: `bootstrap` now starts `OutboxRelay` with store authority selected at runtime: `outbox/spanner_store.go` (`OutboxEvents` fetch/mark) when Spanner is reachable, with in-memory store fallback when unavailable; repository emit paths use a shared outbox appender seam.
- **Order durability seam (additive)**: `order/repository_spanner.go` now persists `Orders` rows + emitted outbox events in a single Spanner `ReadWriteTransaction`, and `bootstrap` now prefers this repository whenever Spanner runtime wiring succeeds (with in-memory order repository fallback retained).
- **Payment durability seam (additive)**: `payment/repository_spanner.go` now persists payment mutations (`PaymentSessions`, `PaymentAttempts`, `PaymentChargebacks`, `PaymentReversals`, `PaymentWebhooks`) plus emitted outbox events in a single Spanner `ReadWriteTransaction`, and `bootstrap` now prefers this repository whenever Spanner runtime wiring succeeds (with in-memory payment repository fallback retained).
- **Supplier durability seam (additive)**: `supplier/repository_spanner.go` now persists supplier onboarding/billing detail in `SupplierProfiles` (seed identity remains in `Suppliers`), and topology read/write is durable through `Warehouses` + `Factories` with transactional outbox buffering.
- **Adapter bridge (additive)**: `bootstrap` now attempts Redis transport via `cache/redis_backend.go` and Kafka outbox transport via `outbox/kafka_publisher.go`, with fail-open fallback to in-memory cache/logging publisher when dependencies are unavailable.
- **Strict reliability mode (additive)**: `REQUIRE_INFRA_ADAPTERS=true` enforces fail-fast bootstrap if Redis or Kafka adapter initialization fails; default mode remains fail-open for local/scaffold startup.
- **Request reliability middleware (additive)**: `bootstrap/reliability_middleware.go` applies fixed-window rate limits, priority-based in-flight shedding, and per-class circuit-open responses; mounted in `main.go` immediately after trace middleware and controlled by `RELIABILITY_MIDDLEWARE_ENABLED`.
- **Payment execution router (additive)**: `payment/execution.go` routes provider actions with bounded retries (exponential backoff + jitter), typed policy errors, and gateway capability controls; checkout/chargeback/reversal handlers in `payment/service.go` now invoke this seam before repository persistence/outbox emission.
- **Checkout attempt metadata persistence (additive)**: `payment/service.go` now persists checkout `PaymentSessions` + first `PaymentAttemptRecord` atomically through repository `CreateSessionWithAttempt` and emits additive execution metadata (`attempt_id`, `execution_action`, `execution_mode`, `provider_reference`) in checkout responses and payment-required event payloads.
- **AIRWALLEX direct execution flag**: `AIRWALLEX_DIRECT_EXECUTION_ENABLED` gates AIRWALLEX execution support in the payment router.
- **Strict-mode startup coverage**: `bootstrap/bootstrap_test.go` validates strict-mode fail-fast for missing Redis/Kafka adapters and strict-mode startup success when both adapters are healthy.
- **Outbox Spanner integration coverage**: `outbox/spanner_store_integration_test.go` verifies `Append -> Fetch -> MarkPublished` lifecycle against emulator-backed Spanner when `SPANNER_EMULATOR_HOST` and schema are available (skips gracefully otherwise).
- **WebSocket relay transport**: `ws/hub.go` now relays typed fanout envelopes over `ws:<hub>:fanout` with source-instance suppression; `main.go` starts all role-hub relay subscribers during boot.
- **AuthN**: JWT cookie sessions + optional Firebase ID token verification (RS256 + Google cert rotation cache)
- **Geospatial**: H3 15-char hex on the wire with additive Redis-backed perimeter gating at resolution 9 (`ssmr:delivery_perimeter` + compacted companion set)
- **Core order schema (additive)**: `schema/spanner.ddl` now includes `Orders` with JSON-encoded line items, amount/currency, H3/lat/lng, version, and created/updated timestamps plus supplier/retailer created-at indexes.
- **Core payment schema (additive)**: `schema/spanner.ddl` now includes durable payment write tables (`PaymentSessions`, `PaymentAttempts`, `PaymentChargebacks`, `PaymentReversals`, `PaymentWebhooks`) with supplier/retailer-scoped payment-session indexes and session/transaction lookup indexes.
- **Spatial perimeter bootstrap (additive)**: `bootstrap/data.go` + `bootstrap/bootstrap.go` precompute and warm supplier delivery coverage in Redis using supplier warehouse coordinates when available (deterministic fallback center/radius otherwise).
- **Topic isolation (additive)**: `events.TopicMain` resolves from `KAFKA_TOPIC_MAIN` at process start, allowing isolated tenant/event-bus naming without changing every outbox emit call site.
- **Order status mutation (additive)**: `POST /v1/order/create` now has a companion `PATCH /v1/order/{orderID}/status` path with transition enforcement, transactional outbox emit (`ORDER_STATUS_CHANGED`), post-commit cache invalidation, and supplier/retailer websocket fanout.
- **Supplier portal API proxy (additive)**: `apps/supplier-portal/app/api/[...path]/route.ts` proxies same-origin `/api/*` calls to backend authority (`SUPPLIER_BACKEND_BASE_URL` / `BACKEND_BASE_URL`) while preserving cookie and idempotency headers for register and billing flows.
- **Shared supplier contract bridge (additive)**: `packages/types/index.ts` now exports supplier register/login/configure/billing/profile/topology DTOs, and `packages/api-client/index.ts` now implements typed supplier API methods with `Idempotency-Key` support and structured `ApiError` responses.
- **Workspace**: `go.work` at `pegasusX/` root listing `./apps/backend-go`, `./apps/ai-worker`, `./packages/config`
- **Route coverage (additive 2026-05-17)**:
	- `supplierroutes`: register/login + supplier core operational surfaces (configure, billing/setup, profile, topology, dashboard, earnings, inventory, inventory-audit, orders, vet).
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

## Route authority freeze (B01)

| Authority surface | Canonical owner | Primary clients | Change rule |
|---|---|---|---|
| `supplierroutes` | supplier platform and business control | supplier portal | changes must update supplier contracts, onboarding/control-tower docs, and supplier role-row parity expectations |
| `retailerroutes` | retailer commerce and account authority | retailer Android, iOS, desktop | changes must update retailer contracts, cart/payment/tracking flows, and customer-support language |
| `driverroutes` | driver execution authority | driver Android, iOS | changes must update driver contracts, manifest semantics, and telemetry/delivery support scripts |
| `warehouseroutes` | warehouse operational authority | warehouse portal, Android, iOS | changes must update warehouse contracts, dispatch/support docs, and node-scoped visibility expectations |
| `factoryroutes` | factory and transfer authority | factory portal, Android, iOS | changes must update manifest contracts, transfer SOPs, and exception handling expectations |
| `payloaderroutes` | loading and manifest repair authority | payload terminal, Android, iOS | changes must update manifest event contracts, exception vocabulary, and source-target cache invalidation rules |
| `orderroutes` | order lifecycle authority | retailer, supplier, downstream operations | changes must preserve additive order states, routing/serviceability authority, and duplicate-request safety |
| `paymentroutes` | checkout and admin payment mutation authority | retailer checkout surfaces, supplier finance/support surfaces | changes must preserve idempotency, policy errors, and replay-safe provider handling |
| `webhookroutes` | gateway callback authority | internal payment and finance flows | changes must keep signature-first verification and transaction-id idempotency intact |
| `telemetryroutes` | live location and execution signal authority | driver apps, supplier and retailer tracking surfaces | changes must keep authenticated identity enforcement and reconnect-safe fanout semantics intact |

## Event catalog freeze (B01)

- Source of truth: `apps/backend-go/events/events.go`, `contracts/events.schema.json`, and shared event unions in `packages/types`.
- Change rule: no event is added, renamed, or repurposed unless these three authorities are updated in the same batch.

Event families:
- Supplier lifecycle: `SUPPLIER_CREATED`, `SUPPLIER_UPDATED`, `SUPPLIER_BILLING_CONFIGURED`
- Retailer and cart: `RETAILER_REGISTERED`, `CART_SYNC_UPDATED`, `SHOP_CLOSED`, `SHOP_CLOSED_RESPONSE`
- Fleet and node operations: `DRIVER_CREATED`, `VEHICLE_CREATED`, `WAREHOUSE_CREATED`, `WAREHOUSE_SUPPLY_REQUEST_OPENED`, `WAREHOUSE_DISPATCH_LOCK_CHANGED`, `FACTORY_CREATED`, `DRIVER_AVAILABILITY_CHANGED`
- Order and routing: `ORDER_CREATED`, `ORDER_STATUS_CHANGED`, `ORDER_VALIDATION_FAILED`, `ORDER_ASSIGNED`, `ORDER_REASSIGNED`, `ORDER_FINALIZED`, `ROUTE_CREATED`
- Manifest lifecycle: `MANIFEST_DRAFT_CREATED`, `MANIFEST_LOADING_STARTED`, `MANIFEST_ORDER_INJECTED`, `MANIFEST_ORDER_EXCEPTION`, `MANIFEST_DLQ_ESCALATION`, `MANIFEST_REBALANCED`, `MANIFEST_CANCELLED`, `MANIFEST_SEALED`, `MANIFEST_DISPATCHED`, `MANIFEST_COMPLETED`
- Payment and delivery state: `PAYMENT_REQUIRED`, `PAYMENT_CLEARED`, `SETTLEMENT_REQUIRED`, `DELIVERY_SESSION_UPDATED`, `DELIVERY_DISPUTED`
- Inventory and command/system state: `INVENTORY_SYNC_COMPLETE`, `COMMAND_DISPATCHED`, `COMMAND_RECEIVED`, `COMMAND_SETTLED`, `SYSTEM_APP_OUTDATED`

## Reliability acceptance matrix (B01 draft)

| Area | Required posture | Draft acceptance target | Proof path |
|---|---|---|---|
| Transactional mutations | Spanner RW transaction + outbox emit + post-commit cache invalidation + trace logging | core order and payment write paths complete without duplicate business outcomes under retry and replay | targeted backend tests plus SSMR proof when infra wiring changes |
| Checkout and webhooks | idempotent client retries, provider-exact verification, bounded retry with jitter, typed policy errors | duplicate checkout taps and replayed webhooks collapse to one durable state transition | payment tests, webhook-path validation, provider replay scenarios |
| Kafka event relay | sync-writer semantics for state changes, aggregate-key ordering, replay-safe consumers | state-change event publish lag remains operationally visible and recoverable without silent drop | relay tests, Kafka round-trip SSMR proof, lag dashboards |
| Redis invalidation and websocket fanout | post-commit invalidation, fail-open local broadcast, source-suppressed fanout envelopes | stale reads and duplicate self-echo fanout remain detectable and bounded | cache tests, websocket relay checks, reconnect scenarios |
| Spatial order acceptance | backend-derived H3, warmed perimeter authority, fail-closed `zone_miss` handling | out-of-zone orders are blocked deterministically and in-zone orders stay stable under duplicate attempts | spatial smoke check, order tests, support reproduction steps |
| Driver telemetry and live tracking | authenticated driver identity, reconnect-safe fanout, no trust in client-supplied identity | live tracking recovers after reconnect without corrupting driver identity or route state | telemetry route validation, websocket checks, load-oriented replay drills |
| SSMR local proof gate | isolated Spanner, Redis, Kafka, bootstrap, health, and topic proof on every infra-sensitive change | no infra-sensitive batch is treated as ready without isolated smoke evidence | `scripts/smoke_ssmr.sh`, `make test-ssmr-infra`, CI gate |
| Kubernetes and rollout safety | stateless pods, graceful shutdown, no sticky-session assumptions, explicit rollback path | rolling restart or pod loss does not orphan state-change delivery or force operator re-entry | deployment review, rollout drill, shutdown-path validation |

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
- `infra/docker-compose.ssmr.yml` — isolated SSMR sandbox stack (non-overlapping host ports, Kafka topic bootstrap, backend-go + ai-worker runtime services, Spanner schema/seed bootstrap via `cmd/setup`, explicit `/usr/local/go/bin/go` invocation for Go containers, named Go module/build cache volumes)
- `scripts/smoke_ssmr.sh` — executable SSMR smoke gate (`make test-ssmr-infra`, `npm run infra:ssmr:test`) proving isolated Spanner/Redis/Kafka/bootstrap wiring, backend health, and Kafka round-trip delivery before teardown
- `../.github/workflows/ssmr-infra.yml` — permanent repo-root pegasusX SSMR stage-gate running `make test-ssmr-infra` on `pegasusX/**` changes and manual dispatches
- `infra/terraform/` — baseline cloud provisioning (VPC, Spanner, Redis, Secret Manager for Kafka/Firebase runtime wiring) with `tenant_slug` / `resource_prefix` segregation and distinct orders/spatial/realtime/webhook topic secret outputs
- `infra/k8s/` — deployment manifests
