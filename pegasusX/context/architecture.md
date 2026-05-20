# pegasusX Architecture

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
- Checkout attempt execution persistence is now additive in `payment/service.go`: checkout persists `PaymentAttemptRecord` through repository `SaveAttempt` after session creation, and additive execution metadata (`attempt_id`, `execution_action`, `execution_mode`, `provider_reference`) is surfaced in checkout responses and payment-required event payloads.
- WebSocket relay now uses typed `ws:<hub>:fanout` envelopes with `{source, room, payload}` and source-instance suppression to avoid self-echo duplication; `main.go` starts relay subscribers for every role hub.
- Additive contract coverage now includes advanced factory/payload manifest workflow event types: `MANIFEST_ORDER_INJECTED`, `MANIFEST_ORDER_EXCEPTION`, `MANIFEST_DLQ_ESCALATION`, `MANIFEST_REBALANCED`, and `MANIFEST_CANCELLED`.
- Factory and payload manifest mutation handlers execute through repository `Apply` seams that pair mutation + outbox emit in one path, then perform post-commit cache invalidation and websocket fanout.
- Manifest lifecycle and reassignment payloads are now additive-parity envelopes across factory/payload services: sealed-manifest events include `route_id`, `driver_id`, `vehicle_id`, and `order_count`; payload reassignment events include `from_manifest_id`, `to_manifest_id`, route metadata, and target driver metadata, and invalidate source + target manifest cache keys.

## Auth Modes
- Supplier portal onboarding/billing: `supplier_jwt` cookie (HS256 scaffold), `auth.CookieAuth` + `auth.RequireRole(ADMIN)`.
- Retailer/mobile bearer mode (optional): Firebase ID token verification (`auth/firebase.go`) validates RS256 signature + `aud` + `iss` + `sub` with cert cache; route-level `auth.RequireRole(RETAILER)` remains enforcement gate.

## Route Coverage (additive 2026-05-17)
- `supplierroutes` mounts supplier role-row operational endpoints: `POST /v1/supplier/configure`, `GET|PUT /v1/supplier/profile`, `GET /v1/supplier/dashboard`, `GET /v1/supplier/earnings`, `GET|PATCH /v1/supplier/inventory`, `GET /v1/supplier/inventory/audit`, `GET /v1/supplier/orders`, `POST /v1/supplier/orders/vet`.
- `retailerroutes` mounts retailer role-row operational endpoints: `GET|PUT /v1/retailer/profile`, `GET /v1/retailer/suppliers`, `POST /v1/retailer/suppliers/{supplierID}/{action}`, `GET|POST /v1/retailer/cart/sync`, `GET /v1/retailers/{retailerID}/orders`, `POST /v1/orders/request-cancel`, `POST /v1/order/cancel`, analytics, family-member, AI confirmation/rejection, preorder, pending-payments, active-fulfillment, and tracking surfaces.
- Retailer protected routes are conditionally wrapped in Firebase bearer verification + `RequireRole(RETAILER)` when auth wiring is enabled; fallback open mounting remains for local scaffold development.
- `driverroutes` mounts driver role-row operational endpoints: `GET /v1/driver/{profile,history,earnings,availability,pending-collections,manifest-gate,manifest}` plus legacy `GET /v1/fleet/manifest` alias.
- `warehouseroutes` mounts warehouse role-row operational endpoints: `GET /v1/warehouse/ops/{dashboard,inventory,orders}`, `POST /v1/warehouse/ops/dispatch/preview`, `GET /v1/warehouse/demand/forecast`, `GET|POST /v1/warehouse/supply-requests`, `GET /v1/warehouse/dispatch-locks`, and `POST|DELETE /v1/warehouse/dispatch-lock`.
- Driver routes are conditionally wrapped in Firebase bearer verification + `RequireRole(DRIVER)` when auth wiring is enabled; warehouse routes are conditionally wrapped in Firebase bearer verification + `RequireRole(WAREHOUSE_ADMIN|ADMIN)` when enabled, with cookie `ADMIN` fallback for local scaffold mode.
- `factoryroutes` mounts factory role-row operational endpoints: `GET /v1/factory/analytics/overview`, `GET /v1/factory/dashboard`, `GET /v1/factory/profile`, `GET /v1/factory/transfers`, `POST /v1/factory/transfers/create`, `GET /v1/factory/manifests`, `GET /v1/factory/manifests/{manifestID}`, lifecycle transitions (`POST /v1/factory/manifests/{manifestID}/{start-loading,seal,dispatch,complete}`), rebalance/cancel surfaces (`POST /v1/factory/manifests/{rebalance,cancel-transfer,cancel}`), `GET /v1/factory/manifest-exceptions`, `GET /v1/factory/fleet/drivers`, `GET /v1/factory/fleet/vehicles`, `GET /v1/factory/staff`, `POST /v1/factory/dispatch`, and `GET /v1/factory/supply-requests`.
- `payloaderroutes` mounts payload role-row operational endpoints: `GET /v1/payloader/trucks`, `GET /v1/payloader/orders`, `GET /v1/payloader/manifests`, `GET /v1/payloader/manifests/{manifestID}`, lifecycle transitions (`POST /v1/payloader/manifests/{manifestID}/{start-loading,inject-order,seal}`), exception surfaces (`POST /v1/payload/manifest-exception`, `GET /v1/payloader/manifest-exceptions`), reassignment surfaces (`POST /v1/payloader/recommend-reassign`, `POST /v1/payloader/reassign-order`), and compatibility `POST /v1/payload/seal`.
- Factory routes are conditionally wrapped in Firebase bearer verification + `RequireRole(FACTORY_ADMIN|ADMIN)` when auth wiring is enabled; payload routes are conditionally wrapped in Firebase bearer verification + `RequireRole(PAYLOAD|ADMIN)` when enabled, with cookie `ADMIN` fallback for local scaffold mode.
- `paymentroutes` mounts payment principal endpoints: `POST /v1/checkout/b2b`, `POST /v1/checkout/unified`, `POST /v1/payment/chargeback`, `POST /v1/payment/chargeback/reversal`, and deprecated `POST /v1/payment/global_pay/initiate`.
- `webhookroutes` mounts gateway callbacks: `POST /v1/webhooks/global-pay`, `POST /v1/webhooks/adyen`, and `POST /v1/webhooks/stripe`.
- Checkout routes are conditionally wrapped in Firebase bearer verification + `RequireRole(RETAILER)` when auth wiring is enabled; payment mutation routes are conditionally wrapped in Firebase bearer verification + `RequireRole(ADMIN)` when enabled, with cookie `ADMIN` fallback for local scaffold mode.
- Webhook routes remain JWT-unauthenticated by design and enforce provider-exact verification before payload decode/mutation: Global Pay Basic auth credential check, Stripe `Stripe-Signature` (`t`,`v1`) verification over raw body, and Adyen item-level `additionalData.hmacSignature` verification.

## Infrastructure Baseline
- Local dev: `infra/docker-compose.yml` runs Spanner emulator, Redis, Kafka, Kafka UI.
- Isolated SSMR sandbox: `infra/docker-compose.ssmr.yml` runs isolated Spanner/Redis/Kafka on non-overlapping host ports, explicit Kafka topic bootstrap (`ssmr.events.orders`, `ssmr.events.spatial`, `ssmr.events.realtime`, `ssmr.events.webhooks`), the local `apps/backend-go/cmd/setup` schema/seed bootstrap job, and the Go backend + ai-worker against those isolated adapters.
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
- `context/architecture.md`
- `context/architecture-graph.json`
- `context/technology-inventory.md`
- `context/technology-inventory.json`
- `context/parity-ledger.md` (when behavior diverges from Pegasus)
