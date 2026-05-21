# pegasusX Copilot Instructions

You are operating inside `pegasusX/`, a single-tenant logistics stack and a sibling project to `pegasus/`.

## Hard Boundaries
1. **Never modify files under `../pegasus/`.** Read-only reference.
2. **Never modify workspace-root files** (`../package.json`, `../.github/*`, `../void-theme/*`).
3. **All work goes inside `pegasusX/`.** Sibling-project isolation.
4. **Single source of truth**: this folder. Code, infra, contracts, docs all live here.

## Primary Directive
Build and operate a single-supplier logistics ecosystem with full architectural compatibility to the Pegasus multi-supplier reference. Single-tenant is a deployment constraint, not a schema simplification — `SupplierId` stays everywhere; one supplier is seeded; no API contracts are simplified away.

## Plan Authority
1. `context/plan.md` is the canonical phased roadmap for pegasusX.
2. Before any non-trivial implementation or roadmap change, read `context/plan.md` together with the relevant context docs.
3. Map work to plan anchors and the active delivery batch, and update plan status in the same change set when execution reality changes.
4. If roadmap and code diverge, code is the source of truth and `context/plan.md` must be updated immediately.

## Doctrine Inheritance
Inherit the workspace doctrine from `../.github/copilot-instructions.md` with the following adaptations:

| Pegasus Concept | pegasusX Equivalent |
|---|---|
| Multi-supplier registration | Single-tenant company bootstrap (one seeded supplier) |
| `admin-portal` (Supplier Portal) | `apps/supplier-portal` |
| `driverappios` | `apps/driver-app-ios` (renamed for consistency) |
| Supplier discovery in retailer apps | Returns the seeded supplier only |
| Step 2 of registration (single warehouse on supplier row) | Topology builder creates real `Factories` + `Warehouses` |
| Mobile bearer auth | Firebase ID token verification may be enabled additively (supplier portal cookie JWT remains canonical for supplier setup flow) |

All other rules persist verbatim:
- Backend package topology (`bootstrap/` composition root, narrow `Deps`, `*routes/` thin).
- Mutating handler shape (auth → scope → validate → RW txn → outbox → cache invalidate → slog → DTO).
- Transactional outbox for state transitions; direct Kafka writes only for telemetry.
- Redis Pub/Sub cache invalidation post-commit.
- WebSocket hubs broadcast via `Hub.Broadcast`; fail-open on Pub/Sub.
- Role scopes derived from JWT claims, never request bodies.
- H3 resolution 7, 15-char hex wire format.
- Material 3 design system; no `@material/web` Lit components; no emoji icons.
- Surface completeness (loading/empty/offline/stale/restricted) on every screen.
- Cross-role sync (a feature added for a role lands on every client in that role row).
- Documentation sync set updated in the same change set as architectural changes.

## Sync Set (this project)
Architecturally meaningful changes update, in the same change set:
- `.github/copilot-instructions.md`
- `.github/gemini-instructions.md`
- `.github/ACT.md`
- `context/plan.md`
- `context/architecture.md`
- `context/architecture-graph.json`
- `context/technology-inventory.md`
- `context/technology-inventory.json`
- `context/parity-ledger.md` (when behavior diverges from Pegasus)

## Working Standard
- Verify the actual `pegasusX/` filesystem before assuming a route, type, or model exists.
- Use `../pegasus/` only as a reference grep target. Do not re-import code from Pegasus; reimplement locally.
- Hunt full chain of impact (backend, frontend, mobile, shared contracts, infra) before declaring work done.
- Treat `context/plan.md` as the execution ledger for phased delivery, roadmap updates, and status reconciliation.
- Record divergence from Pegasus in `context/parity-ledger.md`.

## Runtime Additive Note (2026-05-17)
- B02 supplier bootstrap durability is now additive (2026-05-22): `apps/backend-go/schema/spanner.ddl` now defines `SupplierProfiles`, `apps/backend-go/supplier/repository_spanner.go` now persists rich supplier profile and topology data (`GetTopology`/`ReplaceTopology`) against `Warehouses` + `Factories`, `supplierroutes` now mounts `GET|PUT /v1/supplier/topology`, supplier portal now proxies same-origin `/api/*` through `apps/supplier-portal/app/api/[...path]/route.ts`, and shared supplier DTO/client coverage now exists in `packages/types/index.ts` + `packages/api-client/index.ts`.
- Phase-1.2 payment durability is now additive: `apps/backend-go/schema/spanner.ddl` now provisions durable payment write tables (`PaymentSessions`, `PaymentAttempts`, `PaymentChargebacks`, `PaymentReversals`, `PaymentWebhooks`) including supplier/retailer-scoped payment-session indexes, `apps/backend-go/payment/repository_spanner.go` now persists payment aggregates plus emitted outbox events atomically in one Spanner `ReadWriteTransaction`, checkout now persists session + first attempt through repository `CreateSessionWithAttempt` in one atomic write path, and `bootstrap/bootstrap.go` now selects this Spanner payment repository when runtime Spanner wiring is available while preserving in-memory fallback in degraded/local mode.
- Phase-1 backend durability implementation has started additively: `apps/backend-go/schema/spanner.ddl` now provisions an `Orders` table with supplier/retailer created-at indexes, `apps/backend-go/order/repository_spanner.go` persists order rows and outbox events atomically in one Spanner `ReadWriteTransaction`, and `bootstrap/bootstrap.go` now prefers the Spanner-backed order repository whenever Spanner runtime wiring is available (explicit in-memory fallback remains for degraded/local paths).
- Phase-2 SSMR Redis-backed H3 spatial hub is now additive: `apps/backend-go/retailer/proximity_service.go` computes supplier delivery coverage via `h3.PolygonToCells` + `h3.CompactCells`, persists expanded and compacted sets in Redis (`ssmr:delivery_perimeter`, `ssmr:delivery_perimeter:compacted`) with TTL=0 semantics, and `order/service.go` now derives retailer H3 from request coordinates and fail-closes with `zone_miss` when `SISMEMBER` checks fail or perimeter cache is unavailable. `bootstrap/data.go` + `bootstrap/bootstrap.go` now precompute/warm delivery coverage at startup from supplier warehouse coordinates when available (config fallback otherwise), and `cmd/ssmr-smokecheck` + `scripts/smoke_ssmr.sh` now include `spatial` perimeter assertions.
- Phase-2 SSMR hardening is now additive: `infra/docker-compose.ssmr.yml` shares named `pegasusx-ssmr-go-mod` and `pegasusx-ssmr-go-build` volumes across `backend-setup`, `backend-go`, and `ai-worker`, and `scripts/smoke_ssmr.sh` now uses `docker compose down --remove-orphans` so those Go caches survive repeated local smoke runs. The permanent stage-gate now lives at repo root in `.github/workflows/ssmr-infra.yml`, which runs `make test-ssmr-infra` for `pegasusX/**` changes and manual dispatches.
- Phase-2 SSMR smoke gate is now additive: `scripts/smoke_ssmr.sh` (also exposed as `make test-ssmr-infra` and `npm run infra:ssmr:test`) brings up the isolated compose stack, recreates sandbox Kafka topics, reruns `apps/backend-go/cmd/setup`, asserts seeded supplier + `Retailers` schema state through `apps/backend-go/cmd/ssmr-smokecheck spanner`, pings isolated Redis, waits for `/v1/health`, and validates Kafka topic isolation + round-trip flow through `apps/backend-go/cmd/ssmr-smokecheck kafka` before teardown. `infra/docker-compose.ssmr.yml` now uses `/usr/local/go/bin/go` explicitly for Go service commands so bootstrap/runtime containers do not depend on shell PATH resolution.
- Phase-1 SSMR physical sandbox baseline is now additive: `apps/backend-go/cmd/setup` bootstraps isolated Spanner schema + seeded supplier state for local/client sandbox runs, `apps/backend-go/events.TopicMain` now resolves from `KAFKA_TOPIC_MAIN` at process start so outbox writes can target hermetic tenant topics, `infra/docker-compose.ssmr.yml` stands up isolated Spanner/Redis/Kafka plus bootstrap jobs and Go runtime services on non-overlapping host ports, `.env.ssmr.example` defines the sandbox defaults, and `infra/terraform/*` now namespaces cloud resources/secrets by `tenant_slug` / `resource_prefix` while publishing distinct orders/spatial/realtime/webhook topic secrets. The optimizer sidecar remains intentionally absent in pegasusX until a concrete implementation lands; do not fake that service.
- Supplier and retailer backend parity coverage advanced in `apps/backend-go`: `supplierroutes` now mounts supplier core operational endpoints (configure/profile/dashboard/earnings/inventory/audit/orders/vet), and `retailerroutes` now mounts retailer operational endpoints (profile/suppliers/cart-sync/orders/cancel/analytics/family-members/AI confirm-reject/preorder/pending-payments/active-fulfillment/tracking). Retailer protected routes are gated by Firebase bearer auth + `RequireRole(RETAILER)` when enabled; local scaffold fallback remains additive when Firebase auth is disabled.
- Driver and warehouse backend route families are now mounted additively: `driverroutes` (profile/history/earnings/availability/pending-collections/manifest-gate/manifest + legacy fleet manifest alias) and `warehouseroutes` (ops dashboard/inventory/orders/dispatch preview + demand forecast/supply requests/dispatch locks). Driver endpoints use Firebase bearer + `RequireRole(DRIVER)` when enabled; warehouse endpoints use Firebase bearer + `RequireRole(WAREHOUSE_ADMIN|ADMIN)` when enabled, with cookie `ADMIN` fallback for local scaffold mode.
- Factory and payload backend route families are now mounted additively: `factoryroutes` (analytics overview/dashboard/profile/transfers/manifests/fleet drivers/fleet vehicles/staff/dispatch/supply requests) and `payloaderroutes` (trucks/orders/recommend-reassign/seal). Factory endpoints use Firebase bearer + `RequireRole(FACTORY_ADMIN|ADMIN)` when enabled; payload endpoints use Firebase bearer + `RequireRole(PAYLOAD|ADMIN)` when enabled, and both families keep cookie `ADMIN` fallback for local scaffold mode.
- Payment and webhook backend route families are now mounted additively: `paymentroutes` (`/v1/checkout/b2b`, `/v1/checkout/unified`, `/v1/payment/chargeback`, `/v1/payment/chargeback/reversal`, deprecated `/v1/payment/global_pay/initiate`) and `webhookroutes` (`/v1/webhooks/global-pay`, `/v1/webhooks/adyen`, `/v1/webhooks/stripe`). Checkout uses Firebase bearer + `RequireRole(RETAILER)` when enabled; payment mutations use Firebase bearer + `RequireRole(ADMIN)` when enabled with cookie `ADMIN` fallback; webhook handlers are JWT-unauthenticated and enforce signature-first HMAC validation plus transaction-id idempotency.
- Advanced factory/payload backend workflow parity is now mounted additively: `factoryroutes` includes `/v1/factory/manifests/{manifestID}` and lifecycle transitions (`/start-loading`, `/seal`, `/dispatch`, `/complete`), plus `/v1/factory/manifests/{rebalance,cancel-transfer,cancel}` and `/v1/factory/manifest-exceptions`. `payloaderroutes` includes `/v1/payloader/manifests*`, `/v1/payload/manifest-exception`, `/v1/payloader/manifest-exceptions`, and `/v1/payloader/reassign-order` alongside recommendation/seal surfaces. Service behavior now tracks manifest state machine transitions, exception escalation on overflow threshold, and reassignment depth in scaffold state.
- Shared contracts are extended additively with advanced manifest workflow event types: `MANIFEST_ORDER_INJECTED`, `MANIFEST_ORDER_EXCEPTION`, `MANIFEST_DLQ_ESCALATION`, `MANIFEST_REBALANCED`, and `MANIFEST_CANCELLED`.
- Webhook verification in `apps/backend-go/payment/service.go` is now provider-exact: Global Pay uses Basic auth credential verification, Stripe verifies `Stripe-Signature` (`t`,`v1`) over raw request body, and Adyen verifies item-level `additionalData.hmacSignature` prior to persistence/outbox emission. Transaction-id replay safety remains enforced.
- Factory and payload manifest mutation handlers now execute through repository `Apply` seams and emit lifecycle/exception events through the outbox contract, then apply cache invalidation plus websocket fanout (`supplier:*`, `factory:*`, `payload:*`) for realtime parity.
- Manifest contract parity is now additive for lifecycle/reassignment envelopes: `MANIFEST_SEALED` payloads include `route_id`/`driver_id`/`vehicle_id`/`order_count`, and payload `MANIFEST_REBALANCED` envelopes include `from_manifest_id`/`to_manifest_id` + route/driver metadata while invalidating both source and target manifest cache keys.
- P1 scaffold reliability wiring is now additive in `apps/backend-go`: `main.go` starts `OutboxRelay` and cache invalidation subscriber goroutines, `bootstrap` in-memory repositories now flush buffered outbox events into a shared in-memory outbox store, and `bootstrap.TraceMiddleware` propagates `X-Trace-Id` into `outbox` context so map payload events gain additive `trace_id` automatically.
- P1 adapter bridge is now additive in `apps/backend-go`: runtime cache wiring now attempts Redis backend selection (`cache/redis_backend.go`) with in-memory fallback on init/ping failure, outbox relay publishing now attempts Kafka writer transport (`outbox/kafka_publisher.go`) with logging fallback when broker init fails, and websocket hubs now exchange typed fanout envelopes (`ws:<hub>:fanout`) with source-instance suppression while `main.go` starts all role-hub relay subscribers.
- P1 strict reliability mode is now additive: setting `REQUIRE_INFRA_ADAPTERS=true` makes `bootstrap.NewApp` fail fast if Redis or Kafka adapter initialization fails, while default mode preserves fail-open fallback for local/scaffold execution.
- P1 outbox store authority is now additive: bootstrap attempts a Spanner-backed outbox store (`outbox/spanner_store.go`) and, when reachable, binds relay read/mark operations to `OutboxEvents`; fallback remains in-memory when Spanner is unavailable.
- P1 strict-mode startup coverage is now additive in `bootstrap/bootstrap_test.go`: tests assert strict-mode fail-fast for missing Redis/Kafka adapters and strict-mode success with healthy adapters (including cleanup paths).
- P1 request reliability middleware is now additive: `bootstrap/reliability_middleware.go` provides fixed-window rate limiting, priority-aware in-flight shedding, and per-class circuit-open protection; `main.go` mounts this middleware immediately after `TraceMiddleware`, activation is controlled by `RELIABILITY_MIDDLEWARE_ENABLED`, and coverage includes dedicated middleware tests plus Spanner-backed outbox integration validation in `outbox/spanner_store_integration_test.go` (emulator-gated).
- P2 payment execution routing is now additive: `apps/backend-go/payment/execution.go` introduces a provider execution router with bounded retries (exponential backoff + jitter) and typed gateway policy errors, `service.go` now routes checkout/chargeback/reversal decisions through that seam before repository persistence/outbox emission, and `AIRWALLEX_DIRECT_EXECUTION_ENABLED` controls AIRWALLEX direct execution availability.
- P2 checkout attempt persistence is now additive: `apps/backend-go/payment/service.go` now persists a `PaymentAttemptRecord` via repository `SaveAttempt` after session creation, and checkout responses/payment-required event metadata now include additive `attempt_id`, `execution_action`, `execution_mode`, and `provider_reference` fields for execution durability.
