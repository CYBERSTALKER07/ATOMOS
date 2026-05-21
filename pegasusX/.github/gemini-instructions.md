# pegasusX Gemini Instructions

Mirror of `copilot-instructions.md` for Gemini-flavored agents. Same boundaries, same doctrine, same sync set.

## Hard Boundaries
- Never modify files under `../pegasus/` or workspace root.
- All work goes inside `pegasusX/`.

## Plan Authority
1. `context/plan.md` is the canonical phased roadmap for pegasusX.
2. Before any non-trivial implementation or roadmap change, read `context/plan.md` with the relevant context docs.
3. Map work to plan anchors and the active delivery batch, then update plan status when execution truth changes.
4. If code and roadmap diverge, update `context/plan.md` in the same change set.

## Codebase Traversal Protocol
1. Dual-read mandate: read both the canonical source code in `pegasusX/` AND the architecture doc set in `pegasusX/context/`.
2. Index entry points via `file_search` / `grep_search`.
3. Trace definitions before guessing struct/interface shape.
4. Find usages before modifying public surfaces.
5. Map the graph (handlers → services → repository → Spanner) before code edits.
6. Dual-sync: code and architecture docs land in the same change set.

## Role / App Matrix
See `../README.md` and the canonical doctrine in `copilot-instructions.md`.

## Auth Additive Note
- pegasusX may enable Firebase ID-token bearer verification for mobile callers while preserving supplier-portal cookie JWT flow for supplier onboarding and billing setup.

## Sequential Thinking
For any non-trivial technical task, plan with internal sequential reasoning before code edits. Treat output as private reasoning; user-facing replies summarize decisions, risks, and completed work.

## Sync Set (this project)
- `.github/copilot-instructions.md`
- `.github/gemini-instructions.md`
- `.github/ACT.md`
- `context/plan.md`
- `context/architecture.md`
- `context/architecture-graph.json`
- `context/technology-inventory.md`
- `context/technology-inventory.json`
- `context/parity-ledger.md`

## Runtime Additive Note (2026-05-17)
- B02 supplier bootstrap durability is now additive (2026-05-22): `apps/backend-go/schema/spanner.ddl` now defines `SupplierProfiles`, `apps/backend-go/supplier/repository_spanner.go` now persists rich supplier profile and topology data (`GetTopology`/`ReplaceTopology`) against `Warehouses` + `Factories`, `supplierroutes` now mounts `GET|PUT /v1/supplier/topology`, supplier portal now proxies same-origin `/api/*` through `apps/supplier-portal/app/api/[...path]/route.ts`, and shared supplier DTO/client coverage now exists in `packages/types/index.ts` + `packages/api-client/index.ts`.
- Phase-1.2 payment durability is now additive: `apps/backend-go/schema/spanner.ddl` now provisions durable payment write tables (`PaymentSessions`, `PaymentAttempts`, `PaymentChargebacks`, `PaymentReversals`, `PaymentWebhooks`) including supplier/retailer-scoped payment-session indexes, `apps/backend-go/payment/repository_spanner.go` now persists payment aggregates plus emitted outbox events atomically in one Spanner `ReadWriteTransaction`, checkout now persists session + first attempt through repository `CreateSessionWithAttempt` in one atomic write path, and `bootstrap/bootstrap.go` now selects this Spanner payment repository when runtime Spanner wiring is available while preserving in-memory fallback in degraded/local mode.
- Phase-1 backend durability implementation has started additively: `apps/backend-go/schema/spanner.ddl` now provisions an `Orders` table with supplier/retailer created-at indexes, `apps/backend-go/order/repository_spanner.go` persists order rows and outbox events atomically in one Spanner `ReadWriteTransaction`, and `bootstrap/bootstrap.go` now prefers the Spanner-backed order repository whenever Spanner runtime wiring is available (explicit in-memory fallback remains for degraded/local paths).
- Phase-2 SSMR Redis-backed H3 spatial hub is now additive: `apps/backend-go/retailer/proximity_service.go` computes supplier delivery coverage via `h3.PolygonToCells` + `h3.CompactCells`, persists expanded and compacted sets in Redis (`ssmr:delivery_perimeter`, `ssmr:delivery_perimeter:compacted`) with TTL=0 semantics, and `order/service.go` now derives retailer H3 from request coordinates and fail-closes with `zone_miss` when `SISMEMBER` checks fail or perimeter cache is unavailable. `bootstrap/data.go` + `bootstrap/bootstrap.go` now precompute/warm delivery coverage at startup from supplier warehouse coordinates when available (config fallback otherwise), and `cmd/ssmr-smokecheck` + `scripts/smoke_ssmr.sh` now include `spatial` perimeter assertions.
- Phase-2 SSMR hardening is now additive: `infra/docker-compose.ssmr.yml` shares named `pegasusx-ssmr-go-mod` and `pegasusx-ssmr-go-build` volumes across `backend-setup`, `backend-go`, and `ai-worker`, and `scripts/smoke_ssmr.sh` now uses `docker compose down --remove-orphans` so those Go caches survive repeated local smoke runs. The permanent stage-gate now lives at repo root in `.github/workflows/ssmr-infra.yml`, which runs `make test-ssmr-infra` for `pegasusX/**` changes and manual dispatches.
- Phase-2 SSMR smoke gate is now additive: `scripts/smoke_ssmr.sh` (also exposed as `make test-ssmr-infra` and `npm run infra:ssmr:test`) brings up the isolated compose stack, recreates sandbox Kafka topics, reruns `apps/backend-go/cmd/setup`, asserts seeded supplier + `Retailers` schema state through `apps/backend-go/cmd/ssmr-smokecheck spanner`, pings isolated Redis, waits for `/v1/health`, and validates Kafka topic isolation + round-trip flow through `apps/backend-go/cmd/ssmr-smokecheck kafka` before teardown. `infra/docker-compose.ssmr.yml` now uses `/usr/local/go/bin/go` explicitly for Go service commands so bootstrap/runtime containers do not depend on shell PATH resolution.
- Phase-1 SSMR physical sandbox baseline is now additive: `apps/backend-go/cmd/setup` bootstraps isolated Spanner schema + seeded supplier state for sandbox bring-up, `apps/backend-go/events.TopicMain` now resolves from `KAFKA_TOPIC_MAIN` at process start so outbox writes target tenant-specific Kafka topics, `infra/docker-compose.ssmr.yml` provides the isolated local Spanner/Redis/Kafka stack plus bootstrap jobs and Go runtime services on non-overlapping host ports, `.env.ssmr.example` defines the hermetic local defaults, and `infra/terraform/*` now namespaces cloud resources/secrets by `tenant_slug` / `resource_prefix` with distinct orders/spatial/realtime/webhook topic secrets. The optimizer sidecar remains a declared Phase-2 hook rather than a fake ready service.
- Backend supplier and retailer route coverage is expanded additively. `supplierroutes` now mounts supplier core operational surfaces (configure/profile/dashboard/earnings/inventory/audit/orders/vet). `retailerroutes` now mounts retailer operational surfaces (profile/suppliers/cart-sync/orders/cancel/analytics/family-members/shop-closed-response/AI confirm-reject/preorder/pending-payments/active-fulfillment/tracking). Retailer protected routes enforce Firebase bearer auth + `RequireRole(RETAILER)` when verifier wiring is enabled.
- Backend driver and warehouse route coverage is expanded additively. `driverroutes` now mounts driver operational surfaces (profile/history/earnings/availability/pending-collections/manifest-gate/manifest and legacy `/v1/fleet/manifest` alias). `warehouseroutes` now mounts warehouse operational surfaces (ops dashboard/inventory/orders/dispatch-preview, demand forecast, supply-requests, dispatch-locks). Driver routes enforce Firebase bearer auth + `RequireRole(DRIVER)` when enabled; warehouse routes enforce Firebase bearer auth + `RequireRole(WAREHOUSE_ADMIN|ADMIN)` when enabled with cookie `ADMIN` fallback in local scaffold mode.
- Backend factory and payload route coverage is expanded additively. `factoryroutes` now mounts factory operational surfaces (analytics-overview/dashboard/profile/transfers/manifests/fleet-drivers/fleet-vehicles/staff/dispatch/supply-requests). `payloaderroutes` now mounts payload operational surfaces (`/v1/payloader/trucks`, `/v1/payloader/orders`, `/v1/payloader/recommend-reassign`, `/v1/payload/seal`). Factory routes enforce Firebase bearer auth + `RequireRole(FACTORY_ADMIN|ADMIN)` when enabled; payload routes enforce Firebase bearer auth + `RequireRole(PAYLOAD|ADMIN)` when enabled, with cookie `ADMIN` fallback in local scaffold mode.
- Backend payment and webhook route coverage is expanded additively. `paymentroutes` now mounts checkout and payment-mutation surfaces (`/v1/checkout/{b2b,unified}`, `/v1/payment/{chargeback,chargeback/reversal}`, deprecated `/v1/payment/global_pay/initiate`) with idempotency replay guards and outbox emission in mutation paths. `webhookroutes` now mounts `/v1/webhooks/{global-pay,adyen,stripe}` with signature-first HMAC verification and transaction-id idempotency before persistence and outbox emission.
- Backend advanced factory/payload workflow parity is now expanded additively. `factoryroutes` now mounts manifest detail and lifecycle transitions (`/v1/factory/manifests/{manifestID}/{start-loading,seal,dispatch,complete}`), plus override/exception surfaces (`/v1/factory/manifests/{rebalance,cancel-transfer,cancel}`, `/v1/factory/manifest-exceptions`). `payloaderroutes` now mounts manifest lifecycle and exception/reassignment surfaces (`/v1/payloader/manifests*`, `/v1/payload/manifest-exception`, `/v1/payloader/manifest-exceptions`, `/v1/payloader/reassign-order`) with scaffold-level state tracking for overflow escalation and reassignment depth.
- Shared contracts now include additive advanced manifest workflow event types: `MANIFEST_ORDER_INJECTED`, `MANIFEST_ORDER_EXCEPTION`, `MANIFEST_DLQ_ESCALATION`, `MANIFEST_REBALANCED`, and `MANIFEST_CANCELLED`.
- Payment webhook hardening is now provider-exact in service logic: Global Pay validates Basic auth credentials, Stripe verifies `Stripe-Signature` (`t`,`v1`) over the raw payload, and Adyen verifies item-level `additionalData.hmacSignature` before mutation persistence and outbox emission.
- Factory/payload manifest mutation handlers now execute via explicit repository seams (`Apply`) with outbox emission in the same mutation path, followed by post-commit cache invalidation and websocket fanout for supplier/factory/payload realtime consumers.
- Manifest contract parity is now additive for lifecycle/reassignment envelopes: sealed-manifest events now carry `route_id`, `driver_id`, `vehicle_id`, and `order_count`, while payload reassign events carry `from_manifest_id`, `to_manifest_id`, route metadata, and target driver metadata with source+target manifest cache invalidation.
- P1 scaffold reliability wiring is now additive: `apps/backend-go/main.go` starts outbox relay and cache invalidation subscriber lifecycles, `bootstrap` in-memory repositories persist buffered outbox events into a relay-readable store, and `bootstrap.TraceMiddleware` seeds request `trace_id` context consumed by `outbox.EmitJSON` for additive event trace propagation.
- P1 adapter bridge is now additive: runtime cache selection attempts Redis backend (`cache/redis_backend.go`) with fail-open fallback to in-memory cache, outbox relay publisher selection attempts Kafka writer transport (`outbox/kafka_publisher.go`) with logging fallback, and websocket fanout now uses typed hub-scoped envelopes with source suppression while all role-hub relay subscribers are started from `main.go`.
- P1 strict infra mode is now additive: `REQUIRE_INFRA_ADAPTERS=true` enforces bootstrap fail-fast when Redis or Kafka adapters are unavailable; default remains fail-open for local development.
- P1 outbox relay authority is now additive: bootstrap now attempts a Spanner-backed outbox store and, when available, relay `Fetch`/`MarkPublished` reads from `OutboxEvents`; in-memory fallback remains active when Spanner is unavailable.
- P1 strict startup tests now assert fail-fast adapter behavior and healthy-adapter startup in `apps/backend-go/bootstrap/bootstrap_test.go`.
- P1 request reliability middleware is now additive: `bootstrap/reliability_middleware.go` applies fixed-window rate limiting, priority load-shedding, and per-class circuit-open protection; `main.go` mounts it after trace middleware, `RELIABILITY_MIDDLEWARE_ENABLED` controls activation, and coverage includes middleware unit tests plus emulator-gated Spanner outbox store integration validation.
- P2 payment execution routing is now additive: `apps/backend-go/payment/execution.go` provides provider execution routing with bounded retry/backoff+jitter and typed gateway policy errors; `payment/service.go` now runs checkout/chargeback/reversal through that seam before persistence/outbox emission, and `AIRWALLEX_DIRECT_EXECUTION_ENABLED` feature-gates AIRWALLEX direct execution.
- P2 checkout attempt persistence is now additive: `apps/backend-go/payment/service.go` now persists `PaymentAttemptRecord` via repository `SaveAttempt` after session creation, and checkout responses/payment-required event metadata include additive `attempt_id`, `execution_action`, `execution_mode`, and `provider_reference` fields.
