# pegasusX Architecture

## Execution Plan Authority
- `context/plan.md` is the canonical phased roadmap for pegasusX.
- Every meaningful implementation batch maps to one or more plan anchors in that file.
- When architecture or delivery truth changes, update `context/plan.md` in the same change set as the related code and context docs.

## Topology (logical)
```
                ┌──────────────────────────┐
                │     Retailers (1..N)     │
                │  Android / iOS / Desktop │
                └────────────┬─────────────┘
                             │ HTTPS / WSS
                             ▼
┌──────────────────────────────────────────────────────────┐
│                     backend-go (chi)                     │
│  authroutes · retailerroutes · supplierroutes ·          │
│  driverroutes · warehouseroutes · factoryroutes ·        │
│  payloaderroutes · orderroutes · paymentroutes ·         │
│  webhookroutes · ws/{driver,retailer,warehouse,...}      │
│                                                          │
│  bootstrap/  (composition root)                          │
│  outbox/     (Spanner ⇄ Kafka atomicity)                 │
│  cache/      (Redis + Pub/Sub invalidation)              │
│  ws/         (per-role hubs, Pub/Sub fan-out)            │
│  auth/       (cookie JWT + optional Firebase bearer)      │
└─────┬───────────────┬───────────────┬────────────────────┘
      │               │               │
      ▼               ▼               ▼
   Spanner          Redis           Kafka  ──►  ai-worker
   (schema)        (cache/ps)       (outbox + telemetry)
```

## Backend Package Topology
- **Composition root**: `bootstrap/` — owns all singletons (`*spanner.Client`, `*cache.Cache`, Kafka writers, GCS, hubs, services).
- **Domain packages**: `auth/ order/ payment/ supplier/ warehouse/ factory/ driver/ retailer/ outbox/ cache/ kafka/ ws/ telemetry/ proximity/ replenishment/ vault/ countrycfg/`.
- **Route composers**: `*routes/` packages own URL mounts + middleware stacking only. `Deps` is narrow per package; handlers live in domain packages.
- **`main.go`** is lifecycle only (load config → `bootstrap.NewApp` → `chi.Router` → register routes → serve → shutdown). Target ≤200 lines.

## Mutating Handler Shape (canonical)
```go
// 1. method gate (chi handles)
// 2. auth: claims := auth.MustClaims(r.Context())
// 3. scope: supplierID := claims.ResolveSupplierID()  (single seeded supplier)
// 4. decode + validate typed request
// 5. service call → ReadWriteTransaction:
//      read → validate → write rows → outbox.EmitJSON(txn, ...)
// 6. post-commit: cache.Invalidate(ctx, keys...)
// 7. structured slog with trace_id + identifiers
// 8. respond with versioned DTO (additive only)
```

## Event Discipline
- `events.TopicMain` now resolves from `KAFKA_TOPIC_MAIN` at process start so local/client sandboxes can keep transactional outbox traffic on tenant-specific Kafka topics without rewriting every emit call site.
- Runtime adapter bridge is now additive in `bootstrap.NewApp`: cache selection attempts Redis backend (`cache/redis_backend.go`) and falls back to in-memory cache on init/ping failure; outbox publisher selection attempts Kafka writer transport (`outbox/kafka_publisher.go`) and falls back to logging publisher on init failure.
- Runtime strict mode is now additive in `bootstrap.NewApp`: when `REQUIRE_INFRA_ADAPTERS=true`, bootstrap fails fast if Redis or Kafka adapters cannot be initialized, preventing degraded startup in production-like runs.
- Outbox relay authority is now additive in `bootstrap.NewApp`: relay `Fetch`/`MarkPublished` binds to `OutboxEvents` through `outbox/spanner_store.go` when Spanner connectivity and table probe succeed; fallback remains the in-memory outbox store when Spanner is unavailable.
- Request reliability middleware is now additive in `bootstrap/reliability_middleware.go`: fixed-window rate limiting, priority-based in-flight shedding, and per-class circuit-open responses are mounted from `main.go` immediately after trace middleware, gated by `RELIABILITY_MIDDLEWARE_ENABLED`.
- Payment execution routing is now additive in `payment/execution.go`: checkout/chargeback/reversal actions flow through a provider execution router that applies bounded retry with exponential backoff+jitter on retryable provider failures and returns typed policy errors for unsupported/disabled gateway paths (for example AIRWALLEX direct execution when feature-gated off).
- Checkout attempt execution persistence is now additive in `payment/service.go`: checkout now persists `PaymentSessions` + first `PaymentAttemptRecord` through repository `CreateSessionWithAttempt` in one atomic transaction path, and additive execution metadata (`attempt_id`, `execution_action`, `execution_mode`, `provider_reference`) is surfaced in checkout responses and payment-required event payloads.
- Payment durability is now additive in `payment/repository_spanner.go`: payment mutation writes (`CreateSession`, `SaveAttempt`, `SaveChargeback`, `SaveReversal`, `SaveWebhook`) persist domain rows and emitted outbox events atomically inside one Spanner `ReadWriteTransaction`, and `bootstrap/bootstrap.go` now selects this repository when Spanner runtime wiring is available while preserving in-memory fallback for degraded/local bootstrap paths.
- Supplier durability is now additive in `supplier/repository_spanner.go`: supplier onboarding and billing persistence now split seeded identity in `Suppliers` from rich onboarding/billing fields in `SupplierProfiles`, and supplier topology read/write now persists through `Warehouses` and `Factories` with transactional outbox emission support.
- Order durability is now additive in `order/repository_spanner.go`: order creation persists the `Orders` aggregate row and emitted outbox events inside one Spanner `ReadWriteTransaction`, and `bootstrap/bootstrap.go` now selects this repository when Spanner runtime wiring is available while preserving in-memory fallback for degraded/local bootstrap paths.
- Order lifecycle mutation wiring is now additive in `order/service.go` + `orderroutes/routes.go`: `PATCH /v1/order/{orderID}/status` enforces canonical status transitions, emits `ORDER_STATUS_CHANGED` through transactional outbox, invalidates supplier/retailer order cache keys post-commit, and fans out realtime supplier/retailer websocket updates after commit.
- Retailer spatial authority is now additive in `retailer/proximity_service.go`: supplier delivery perimeter coverage is computed with `h3.PolygonToCells` + `h3.CompactCells`, persisted to Redis sets (`ssmr:delivery_perimeter`, `ssmr:delivery_perimeter:compacted`) with TTL=0 semantics, and consumed as O(1) `SISMEMBER` checks from order creation fail-closed `zone_miss` handling.
- WebSocket relay now uses typed `ws:<hub>:fanout` envelopes with `{source, room, payload}` and source-instance suppression to avoid self-echo duplication; `main.go` starts relay subscribers for every role hub.
- WebSocket role-room mapping is now additive in `ws/handler.go`: canonical role constants (`ADMIN`, `RETAILER`, `DRIVER`, `PAYLOAD`, `FACTORY_ADMIN`, `WAREHOUSE_ADMIN`) resolve subscriptions for supplier and node-scoped rooms without legacy role-string drift.
- Outbox trace propagation is now additive in `outbox/outbox.go`: `EmitJSON` injects `trace_id` for object payloads regardless of whether the caller supplied a map or a typed struct.
- Additive contract coverage now includes advanced factory/payload manifest workflow event types: `MANIFEST_ORDER_INJECTED`, `MANIFEST_ORDER_EXCEPTION`, `MANIFEST_DLQ_ESCALATION`, `MANIFEST_REBALANCED`, and `MANIFEST_CANCELLED`.
- Factory and payload manifest mutation handlers execute through repository `Apply` seams that pair mutation + outbox emit in one path, then perform post-commit cache invalidation and websocket fanout.
- Manifest lifecycle and reassignment payloads are now additive-parity envelopes across factory/payload services: sealed-manifest events include `route_id`, `driver_id`, `vehicle_id`, and `order_count`; payload reassignment events include `from_manifest_id`, `to_manifest_id`, route metadata, and target driver metadata, and invalidate source + target manifest cache keys.
- Supplier portal transport is now additive in `apps/supplier-portal/app/api/[...path]/route.ts`: same-origin `/api/*` requests are proxied to backend authority (`SUPPLIER_BACKEND_BASE_URL`), preserving `Set-Cookie` and idempotency headers for onboarding and billing flows.
- Shared supplier contract bridge is now additive across `packages/types/index.ts` and `packages/api-client/index.ts`: typed DTOs now cover supplier register/login/configure/billing/profile/topology wire shapes and typed client methods now map to `/v1/auth/supplier/*` plus `/v1/supplier/{configure,billing/setup,profile,topology}` with structured `ApiError` handling.

## Auth Modes
- Supplier portal onboarding/billing: `supplier_jwt` cookie (HS256 scaffold), `auth.CookieAuth` + `auth.RequireRole(ADMIN)`.
- Retailer/mobile bearer mode (optional): Firebase ID token verification (`auth/firebase.go`) validates RS256 signature + `aud` + `iss` + `sub` with cert cache; route-level `auth.RequireRole(RETAILER)` remains enforcement gate.

## Role-Row Capability and Ownership Ledger

| Role row | Canonical surfaces | Primary business owner | Core operating responsibilities | Backend authority surfaces | Primary realtime authority |
|---|---|---|---|---|---|
| SUPPLIER (`ADMIN`) | `apps/supplier-portal` | Supplier leadership, finance, and support ops | company bootstrap, billing, topology, pricing authority, inventory oversight, earnings visibility, order oversight | `supplierroutes`, supplier-owned domain packages, admin payment mutation oversight | `supplier:{supplier_id}` |
| RETAILER | `apps/retailer-app-android`, `apps/retailer-app-ios`, `apps/retailer-app-desktop` | Commercial operations and customer support | registration, supplier relationship, cart sync, checkout initiation, order decisions, tracking, pending-payment follow-up | `retailerroutes`, `orderroutes` create/status entrypoints, retailer checkout surfaces in `paymentroutes` | `retailer:{retailer_id}` |
| DRIVER | `apps/driver-app-android`, `apps/driver-app-ios` | Delivery execution and collections ops | availability, manifest execution, stop progression, pending collections, delivery completion, telemetry | `driverroutes`, telemetry ingress, driver-permitted order-state transitions | `driver:{driver_id}`, `telemetry:live` |
| WAREHOUSE_ADMIN | `apps/warehouse-portal`, `apps/warehouse-app-android`, `apps/warehouse-app-ios` | Warehouse operations leadership | inventory, local order queue, dispatch preview, supply requests, dispatch locks | `warehouseroutes` and warehouse-scoped operational services | `warehouse:{home_node_id}`, `supplier:{supplier_id}` |
| FACTORY_ADMIN | `apps/factory-portal`, `apps/factory-app-android`, `apps/factory-app-ios` | Factory and transfer operations leadership | transfers, manifest lifecycle, exception resolution, fleet/staff oversight, supply response | `factoryroutes` and factory-scoped manifest services | `factory:{home_node_id}`, `supplier:{supplier_id}` |
| PAYLOAD | `apps/payload-terminal`, `apps/payload-app-ios`, `apps/payload-app-android` | Loading and dock operations leadership | truck readiness, load start, inject-before-seal, manifest exceptions, reassign support, seal confirmation | `payloaderroutes` and payload-scoped manifest mutation services | `payload:{subject}`, supplier and factory rooms as needed |
| SYSTEM | `apps/backend-go`, `apps/ai-worker` | Platform engineering and SRE | auth, data durability, outbox relay, cache invalidation, worker execution, observability, infra proof | all domain and platform packages | all role hubs and internal event streams |

### Role-row parity rules
1. A capability is not considered delivered for a role until every supported client in that role row can represent the same business state, even if platform UI controls differ.
2. If a capability is intentionally delayed on one client, it must be hidden or feature-flagged so the user cannot discover a broken partial rollout.
3. Every capability ledger change must identify the owning backend route family, the shared contract authority, and the affected realtime room if live updates are expected.

## Route Coverage (additive 2026-05-17)
- `supplierroutes` mounts supplier role-row operational endpoints: `POST /v1/auth/supplier/register`, `POST /v1/auth/supplier/login`, `POST /v1/supplier/configure`, `POST /v1/supplier/billing/setup`, `GET|PUT /v1/supplier/profile`, `GET|PUT /v1/supplier/topology`, `GET /v1/supplier/dashboard`, `GET /v1/supplier/earnings`, `GET|PATCH /v1/supplier/inventory`, `GET /v1/supplier/inventory/audit`, `GET /v1/supplier/orders`, `POST /v1/supplier/orders/vet`.
- `retailerroutes` mounts retailer role-row operational endpoints: `GET|PUT /v1/retailer/profile`, `GET /v1/retailer/suppliers`, `POST /v1/retailer/suppliers/{supplierID}/{action}`, `GET|POST /v1/retailer/cart/sync`, `GET /v1/retailers/{retailerID}/orders`, `POST /v1/orders/request-cancel`, `POST /v1/order/cancel`, analytics, family-member, AI confirmation/rejection, preorder, pending-payments, active-fulfillment, and tracking surfaces.
- Retailer protected routes are conditionally wrapped in Firebase bearer verification + `RequireRole(RETAILER)` when auth wiring is enabled; fallback open mounting remains for local scaffold development.
- `driverroutes` mounts driver role-row operational endpoints: `GET /v1/driver/{profile,history,earnings,availability,pending-collections,manifest-gate,manifest}` plus legacy `GET /v1/fleet/manifest` alias.
- `warehouseroutes` mounts warehouse role-row operational endpoints: `GET /v1/warehouse/ops/{dashboard,inventory,orders}`, `POST /v1/warehouse/ops/dispatch/preview`, `GET /v1/warehouse/demand/forecast`, `GET|POST /v1/warehouse/supply-requests`, `GET /v1/warehouse/dispatch-locks`, and `POST|DELETE /v1/warehouse/dispatch-lock`.
- Driver routes are conditionally wrapped in Firebase bearer verification + `RequireRole(DRIVER)` when auth wiring is enabled; warehouse routes are conditionally wrapped in Firebase bearer verification + `RequireRole(WAREHOUSE_ADMIN|ADMIN)` when enabled, with cookie `ADMIN` fallback for local scaffold mode.
- `factoryroutes` mounts factory role-row operational endpoints: `GET /v1/factory/analytics/overview`, `GET /v1/factory/dashboard`, `GET /v1/factory/profile`, `GET /v1/factory/transfers`, `POST /v1/factory/transfers/create`, `GET /v1/factory/manifests`, `GET /v1/factory/manifests/{manifestID}`, lifecycle transitions (`POST /v1/factory/manifests/{manifestID}/{start-loading,seal,dispatch,complete}`), rebalance/cancel surfaces (`POST /v1/factory/manifests/{rebalance,cancel-transfer,cancel}`), `GET /v1/factory/manifest-exceptions`, `GET /v1/factory/fleet/drivers`, `GET /v1/factory/fleet/vehicles`, `GET /v1/factory/staff`, `POST /v1/factory/dispatch`, and `GET /v1/factory/supply-requests`.
- `payloaderroutes` mounts payload role-row operational endpoints: `GET /v1/payloader/trucks`, `GET /v1/payloader/orders`, `GET /v1/payloader/manifests`, `GET /v1/payloader/manifests/{manifestID}`, lifecycle transitions (`POST /v1/payloader/manifests/{manifestID}/{start-loading,inject-order,seal}`), exception surfaces (`POST /v1/payload/manifest-exception`, `GET /v1/payloader/manifest-exceptions`), reassignment surfaces (`POST /v1/payloader/recommend-reassign`, `POST /v1/payloader/reassign-order`), and compatibility `POST /v1/payload/seal`.
- `orderroutes` now mounts order mutation surfaces: `POST /v1/order/create` and additive `PATCH /v1/order/{orderID}/status`.
- Factory routes are conditionally wrapped in Firebase bearer verification + `RequireRole(FACTORY_ADMIN|ADMIN)` when auth wiring is enabled; payload routes are conditionally wrapped in Firebase bearer verification + `RequireRole(PAYLOAD|ADMIN)` when enabled, with cookie `ADMIN` fallback for local scaffold mode.
- `paymentroutes` mounts payment principal endpoints: `POST /v1/checkout/b2b`, `POST /v1/checkout/unified`, `POST /v1/payment/chargeback`, `POST /v1/payment/chargeback/reversal`, and deprecated `POST /v1/payment/global_pay/initiate`.
- `webhookroutes` mounts gateway callbacks: `POST /v1/webhooks/global-pay`, `POST /v1/webhooks/adyen`, and `POST /v1/webhooks/stripe`.
- Checkout routes are conditionally wrapped in Firebase bearer verification + `RequireRole(RETAILER)` when auth wiring is enabled; payment mutation routes are conditionally wrapped in Firebase bearer verification + `RequireRole(ADMIN)` when enabled, with cookie `ADMIN` fallback for local scaffold mode.
- Webhook routes remain JWT-unauthenticated by design and enforce provider-exact verification before payload decode/mutation: Global Pay Basic auth credential check, Stripe `Stripe-Signature` (`t`,`v1`) verification over raw body, and Adyen item-level `additionalData.hmacSignature` verification.

## Support vocabulary baseline
1. Supplier Portal: the canonical supplier-facing web and desktop control surface. Do not describe it as a separate platform-admin console.
2. Seeded supplier: the one live supplier tenant in pegasusX. This is a deployment rule, not a schema simplification.
3. Zone miss: a fail-closed order-capture outcome when the retailer address falls outside the warmed delivery perimeter or perimeter authority is unavailable.
4. Pending payment: the order intent exists, but fulfillment cannot advance until payment authority changes state.
5. Payment cleared: the checkout or webhook path has confirmed the payment state required for fulfillment progress.
6. Settlement required: delivery or payment handoff needs an explicit settlement action before the lifecycle can fully close.
7. Dispatch lock: a temporary operator-owned freeze that prevents conflicting dispatch mutations while someone is actively controlling the plan.
8. Manifest sealed: loading is closed; any further physical change must travel through an exception, rebalance, or reassignment path.
9. Active fulfillment: the order is physically in progress and should appear in live tracking and exception surfaces.
10. Scaffold mode: local fallback behavior when strict infrastructure adapters are not required; never describe scaffold mode as production-ready.
11. SSMR proof: the isolated Docker validation loop that must pass before infra-sensitive behavior is treated as trustworthy.

## Operational ownership and escalation baseline
1. Supplier onboarding and billing failures: supplier support owns first response; backend or platform escalates when auth, persistence, or idempotency evidence shows system fault.
2. Retailer pricing, zone, and cart issues: commercial or retailer support owns first response; supplier operations arbitrates business rules; backend escalates only when contract or routing authority is inconsistent.
3. Payment incidents: finance and payment support own first response; payment engineering escalates for webhook, idempotency, provider policy, or settlement mismatches.
4. Warehouse, factory, and payload execution incidents: node operations own first response; logistics engineering escalates when manifest events, reassignment, or dispatch authority drift from system truth.
5. Driver telemetry or live-tracking incidents: delivery operations own first response; platform and mobile leads escalate when authenticated telemetry, websocket relay, or reconnect behavior are suspect.
6. Release and launch evidence always includes: plan-anchor reconciliation, contract and doc sync, validation results, and a rollback path for any architecture-sensitive batch.

## Infrastructure Baseline
- Local dev: `infra/docker-compose.yml` runs Spanner emulator, Redis, Kafka, Kafka UI.
- Core schema durability: `schema/spanner.ddl` now includes an `Orders` table (`line_items` JSON bytes, currency/amount, H3, version, timestamps) plus supplier/retailer created-at indexes, and `cmd/setup` applies these statements idempotently for emulator and local bootstrap flows.
- Isolated SSMR sandbox: `infra/docker-compose.ssmr.yml` runs isolated Spanner/Redis/Kafka on non-overlapping host ports, explicit Kafka topic bootstrap (`ssmr.events.orders`, `ssmr.events.spatial`, `ssmr.events.realtime`, `ssmr.events.webhooks`), the local `apps/backend-go/cmd/setup` schema/seed bootstrap job, and the Go backend + ai-worker against those isolated adapters.
- Validated smoke gate: `scripts/smoke_ssmr.sh` (also exposed as `make test-ssmr-infra` and `npm run infra:ssmr:test`) orchestrates isolated compose bring-up, reruns `kafka-init` and `backend-setup`, then uses `apps/backend-go/cmd/ssmr-smokecheck` to prove seeded supplier + `Retailers` schema state, Redis reachability, backend `/v1/health`, and Kafka topic isolation + round-trip delivery before teardown. `infra/docker-compose.ssmr.yml` uses `/usr/local/go/bin/go` explicitly for Go service commands so the official Go image does not depend on shell PATH resolution.
- Spatial perimeter bootstrap and gate: `bootstrap/data.go` + `bootstrap/bootstrap.go` now precompute the Redis-backed supplier delivery perimeter at startup from supplier warehouse coordinates when present (deterministic fallback center/radius otherwise), and the smoke gate now runs `apps/backend-go/cmd/ssmr-smokecheck spatial` to assert perimeter-key existence and positive membership checks before Kafka round-trip validation.
- Persistent iteration caches: `infra/docker-compose.ssmr.yml` now mounts named `pegasusx-ssmr-go-mod` and `pegasusx-ssmr-go-build` volumes into all Go services, and `scripts/smoke_ssmr.sh` now tears down without `-v` so container-side module/build caches persist across local SSMR smoke runs while infra containers still reset.
- Permanent CI gate: repo-root `.github/workflows/ssmr-infra.yml` now runs `make test-ssmr-infra` for `pegasusX/**` changes and manual dispatches so pegasusX backend breathing is enforced before merge.
- Cloud provisioning baseline: `infra/terraform/` provisions VPC, Spanner, Redis, and Secret Manager entries for Kafka/Firebase runtime wiring.
- Terraform segregation: `tenant_slug` / `resource_prefix` now namespace cloud resources and Secret Manager entries so each SSMR tenant can keep isolated Spanner, Redis, and topic wiring without sharing state identifiers.
- Phase-2 note: pegasusX does not yet carry a concrete optimizer-sidecar-rust implementation. The sandbox stack intentionally stops at backend-go + ai-worker rather than pretending a solver runtime exists.

## Single-Supplier Enforcement
- Bootstrap seeds exactly one `Suppliers` row.
- `auth.ResolveSupplierID()` returns the seeded supplier when claim is missing (gracefully handles the single-tenant case).
- Supplier discovery endpoints return the seeded supplier only.
- Retailer ordering routes never expose a supplier selector in UI.

## Home Node Model
- `Drivers.HomeNodeType ∈ {WAREHOUSE, FACTORY}` + `HomeNodeId`.
- `Vehicles.HomeNodeType` + `HomeNodeId`.
- Supports: warehouse-only, factory-only, mixed local+remote topologies.

## Onboarding (single-tenant company bootstrap)
1. Account (company, contact, email, phone, password) — same as Pegasus step 1.
2. Topology (factories, warehouses, capture mixed local/remote layout) — replaces single-warehouse-on-supplier-row pattern.
3. Business (tax id, registration, fleet profile, cold chain, palletization).
4. Categories (operating categories multi-select).
5. → `/setup/billing` (payment gateway + bank) — same separation as Pegasus.
6. → `/supplier/dashboard`.

## Documentation Sync Set
Any architecturally meaningful change must update, in the same change set:
- `.github/ACT.md`
- `.github/copilot-instructions.md`
- `.github/gemini-instructions.md`
- `context/plan.md`
- `context/architecture.md`
- `context/architecture-graph.json`
- `context/technology-inventory.md`
- `context/technology-inventory.json`
- `context/parity-ledger.md` (when behavior diverges from Pegasus)
