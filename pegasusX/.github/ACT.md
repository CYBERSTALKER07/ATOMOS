# pegasusX ACT Protocol

ACT = **A**udit, **C**ompanion, **T**ransparency. Required for every technical request in pegasusX.

## Plan Authority

1. `context/plan.md` is the canonical phased execution roadmap for pegasusX.
2. Before any non-trivial implementation, map the work to one or more plan anchors and the active delivery batch in that file.
3. After each execution chunk, update plan-anchor status and sync related context docs in the same change set.

## Audit
Before any code edit, audit:
1. Backend contract compatibility (Spanner schema, Kafka events, DTOs).
2. Role / scope correctness (claims-derived, not body-derived).
3. Outbox + cache invalidation coverage for mutations.
4. Cross-role-row sync (a feature for a role lands on all that role's clients).

## Companion
If a prompt is risky, incomplete, or production-breaking, do NOT execute blindly. Explain the safer execution plan, then execute the safer plan.

Classify each request as one of:
- `safe` — proceed
- `risky` — explain mitigation, then proceed
- `production-breaking` — propose alternative, await confirmation
- `scope-conflict` — surface conflict, await arbitration

## Transparency
After each phase or execution chunk:
1. Reconcile session memory with `context/plan.md` as the active plan.
2. Report per plan-anchor ID: `implemented`, `in progress`, `blocked`, `deferred`.
3. Update `context/parity-ledger.md` whenever behavior diverges from Pegasus.

## Hyper-Scale Checks
All implementations are evaluated against:
- Spanner (index-backed reads, RW transactions, mutation cell limits).
- Kafka (sync writer for state changes, partition keying by aggregate root, version gating).
- Redis (Pub/Sub invalidation, singleflight stampede protection, pipelining).
- Kubernetes (stateless pods, graceful shutdown, no sticky sessions).
- Terraform (managed infra parity).
- Firebase Auth (ID-token signature verification, `aud`/`iss`/`sub` validation, key rotation cache).

## One-Eye Guard Suite (target)
Future scripts under `pegasusX/scripts/`:
- `contract_guard.py`
- `architecture_guard.py`
- `design_system_guard.py`
- `production_safety_guard.py`
- `security_guard.py`

## Runtime Additive Notes
- 2026-05-22: B02 supplier bootstrap durability has advanced additively: `apps/backend-go/schema/spanner.ddl` now provisions `SupplierProfiles`, supplier persistence in `apps/backend-go/supplier/repository_spanner.go` now stores rich onboarding/billing + topology data (`GetTopology`, `ReplaceTopology`) against `Warehouses`/`Factories`, `supplierroutes` now mounts `GET|PUT /v1/supplier/topology`, supplier portal transport now proxies same-origin `/api/*` requests through `apps/supplier-portal/app/api/[...path]/route.ts`, and shared supplier contract/client coverage now exists in `packages/types/index.ts` + `packages/api-client/index.ts`.
- 2026-05-21: Phase-1.2 payment durability is now additive in `pegasusX`: `apps/backend-go/schema/spanner.ddl` now provisions durable payment write tables (`PaymentSessions`, `PaymentAttempts`, `PaymentChargebacks`, `PaymentReversals`, `PaymentWebhooks`) with supplier/retailer scoped payment-session indexes, `apps/backend-go/payment/repository_spanner.go` now persists payment aggregates and emitted outbox events atomically inside one Spanner `ReadWriteTransaction`, checkout now persists `PaymentSessions` + first `PaymentAttempts` row through repository `CreateSessionWithAttempt` in one atomic write path, and `bootstrap/bootstrap.go` now prefers this Spanner-backed payment repository when runtime Spanner wiring is available with explicit in-memory fallback retained.
- 2026-05-21: Phase-1 backend durability implementation has started additively in `pegasusX`: `apps/backend-go/schema/spanner.ddl` now defines `Orders` + supplier/retailer created-at indexes, `apps/backend-go/order/repository_spanner.go` persists order rows and outbox events atomically inside one Spanner `ReadWriteTransaction`, and `bootstrap/bootstrap.go` now prefers the Spanner-backed order repository whenever Spanner runtime wiring is available, with explicit in-memory fallback retained for degraded/local paths.
- 2026-05-21: Phase-2 SSMR Redis-backed H3 spatial hub is now additive in `pegasusX`: `apps/backend-go/retailer/proximity_service.go` precomputes supplier delivery coverage with `h3.PolygonToCells` + `h3.CompactCells`, persists expanded and compacted sets in Redis (`ssmr:delivery_perimeter`, `ssmr:delivery_perimeter:compacted`) with TTL=0 semantics, and enforces O(1) `SISMEMBER` zone checks through `order/service.go` fail-closed `zone_miss` handling. `bootstrap/data.go` + `bootstrap/bootstrap.go` now warm the perimeter at startup from supplier warehouse coordinates when present (with deterministic config fallback), and `cmd/ssmr-smokecheck` + `scripts/smoke_ssmr.sh` now include a `spatial` assertion step.
- 2026-05-21: Phase-2 SSMR hardening is now additive in `pegasusX`: `infra/docker-compose.ssmr.yml` shares named `pegasusx-ssmr-go-mod` and `pegasusx-ssmr-go-build` volumes across `backend-setup`, `backend-go`, and `ai-worker`, while `scripts/smoke_ssmr.sh` now tears the stack down without `-v` so those Go caches survive repeated local smoke runs. The permanent stage-gate now lives at repo root in `.github/workflows/ssmr-infra.yml`, which runs `make test-ssmr-infra` for `pegasusX/**` changes and manual dispatches.
- 2026-05-21: Phase-2 SSMR smoke gate is now additive in `pegasusX`: `scripts/smoke_ssmr.sh` (also exposed as `make test-ssmr-infra` and `npm run infra:ssmr:test`) brings up the isolated compose stack, recreates isolated Kafka topics, reruns `apps/backend-go/cmd/setup` idempotently, asserts seeded supplier + `Retailers` schema state through `apps/backend-go/cmd/ssmr-smokecheck spanner`, pings isolated Redis, waits for `/v1/health`, and validates Kafka topic isolation + round-trip flow through `apps/backend-go/cmd/ssmr-smokecheck kafka` before teardown. `infra/docker-compose.ssmr.yml` now invokes `/usr/local/go/bin/go` explicitly inside Go containers so Docker shell PATH resolution cannot break bootstrap or runtime services.
- 2026-05-21: Phase-1 SSMR physical sandbox baseline is now additive in `pegasusX`: `apps/backend-go/cmd/setup` bootstraps the isolated Spanner instance/database and seeds the single supplier row, `events.TopicMain` resolves from `KAFKA_TOPIC_MAIN` at process start for client-scoped Kafka isolation, `infra/docker-compose.ssmr.yml` stands up isolated local Spanner/Redis/Kafka plus bootstrap jobs and Go runtime services on non-overlapping host ports, `.env.ssmr.example` defines hermetic local defaults, and `infra/terraform/*` now namespaces resources and Secret Manager topic wiring by `tenant_slug` / `resource_prefix` while exporting distinct orders/spatial/realtime/webhook topic secrets. Divergence remains explicit: pegasusX still has no concrete optimizer-sidecar-rust implementation, so the compose stack reserves that hook for Phase 2 instead of faking readiness.
- 2026-05-17: Supplier core surface is now mounted additively in `supplierroutes` (`/v1/supplier/configure`, `/v1/supplier/profile`, `/v1/supplier/dashboard`, `/v1/supplier/earnings`, `/v1/supplier/inventory`, `/v1/supplier/inventory/audit`, `/v1/supplier/orders`, `/v1/supplier/orders/vet`) behind cookie auth + `RequireRole(ADMIN)`.
- 2026-05-17: Retailer core surface is now mounted additively in `retailerroutes` (`/v1/retailer/profile`, `/v1/retailer/suppliers*`, `/v1/retailer/cart/sync`, `/v1/retailers/{retailerID}/orders`, cancellation, analytics, family members, AI confirm/reject, preorder, pending-payments, active-fulfillment, tracking). When `FIREBASE_AUTH_ENABLED=true` and verifier wiring is available, these routes are protected by Firebase bearer auth + `RequireRole(RETAILER)`; otherwise they remain scaffold-accessible for local development.
- 2026-05-17: Driver and warehouse route families are now mounted additively. `driverroutes` serves profile/history/earnings/availability/pending-collections/manifest-gate/manifest (plus legacy `/v1/fleet/manifest` alias) and enforces Firebase bearer + `RequireRole(DRIVER)` when enabled. `warehouseroutes` serves dashboard/inventory/orders/dispatch-preview/demand-forecast/supply-requests/dispatch-lock surfaces; auth is Firebase bearer + `RequireRole(WAREHOUSE_ADMIN|ADMIN)` when enabled, with cookie `ADMIN` fallback for local scaffold mode.
- 2026-05-17: Factory and payload route families are now mounted additively. `factoryroutes` serves analytics-overview/dashboard/profile/transfers/manifests/fleet/staff/dispatch/supply-request surfaces and enforces Firebase bearer + `RequireRole(FACTORY_ADMIN|ADMIN)` when enabled, with cookie `ADMIN` fallback for local scaffold mode. `payloaderroutes` serves trucks/orders/recommend-reassign/seal and enforces Firebase bearer + `RequireRole(PAYLOAD|ADMIN)` when enabled, with cookie `ADMIN` fallback for local scaffold mode.
- 2026-05-17: Payment and webhook route families are now mounted additively. `paymentroutes` serves checkout and payment-mutation surfaces (`/v1/checkout/{b2b,unified}`, `/v1/payment/{chargeback,chargeback/reversal}`, deprecated `/v1/payment/global_pay/initiate`) with `Idempotency-Key` replay safety and outbox emission on every mutation path. `webhookroutes` serves `/v1/webhooks/{global-pay,adyen,stripe}` using signature-first HMAC verification and transaction-id idempotency keys before webhook persistence/outbox emission.
- 2026-05-17: Advanced factory/payload workflow parity is now additive in scaffold mode. `factoryroutes` now includes manifest detail and lifecycle transitions (`start-loading`, `seal`, `dispatch`, `complete`) plus rebalance/cancel-transfer/cancel and exception queue read surfaces. `payloaderroutes` now includes manifest list/detail/start-loading/inject/seal, manifest exception write/read, recommendation scoring, and reassignment apply endpoints with reassignment-depth tracking and overflow escalation threshold handling.
- 2026-05-17: Shared event catalog and contracts now include advanced manifest workflow event types (`MANIFEST_ORDER_INJECTED`, `MANIFEST_ORDER_EXCEPTION`, `MANIFEST_DLQ_ESCALATION`, `MANIFEST_REBALANCED`, `MANIFEST_CANCELLED`) for additive role-row compatibility.
- 2026-05-17: Webhook hardening is now provider-exact in `payment/service.go`: Global Pay validates Basic auth credentials, Stripe validates `Stripe-Signature` (`t`,`v1`) against the raw body, and Adyen validates item-level `additionalData.hmacSignature` before mutation persistence. Replay safety remains transaction-id keyed, and successful webhook mutations persist through the outbox seam.
- 2026-05-17: Factory and payload manifest mutations now run through explicit repository `Apply` seams with mutation + outbox emission in one path, then post-commit cache invalidation and websocket fanout. Manifest lifecycle and exception events (`MANIFEST_LOADING_STARTED`, `MANIFEST_SEALED`, `MANIFEST_ORDER_INJECTED`, `MANIFEST_ORDER_EXCEPTION`, `MANIFEST_DLQ_ESCALATION`, `MANIFEST_REBALANCED`) now reach supplier/factory/payload rooms through `ws.Hub.Broadcast` fail-open relay.
- 2026-05-17: Manifest lifecycle/reassignment contract parity is now additive in scaffold services: sealed-manifest events now include `route_id`, `driver_id`, `vehicle_id`, and `order_count`; payload reassignment emits `from_manifest_id`, `to_manifest_id`, `from_route_id`, `to_route_id`, and `to_driver_id`, and invalidates both source/target manifest cache keys to prevent stale manifest detail reads.
- 2026-05-17: P1 scaffold reliability wiring is now active in `apps/backend-go`: `main.go` starts `OutboxRelay` and cache invalidation subscriber lifecycles, `bootstrap` stores emitted in-memory outbox events instead of dropping buffered txn payloads, and `bootstrap.TraceMiddleware` now propagates `X-Trace-Id` into request context so `outbox.EmitJSON` can add additive `trace_id` fields to map payload envelopes.
- 2026-05-17: P1 production-adapter bridge is now additive in `apps/backend-go`: `bootstrap` attempts Redis-backed `cache.Backend` (`cache/redis_backend.go`) and Kafka-backed outbox publisher (`outbox/kafka_publisher.go`) with fail-open fallback to in-memory/logging seams; `main.go` now starts websocket relay subscribers for all role hubs, and `ws/hub.go` now uses typed `ws:<hub>:fanout` envelopes with self-echo suppression so cross-pod relay is functional when Redis Pub/Sub is reachable.
- 2026-05-17: P1 strict adapter mode is now additive in `apps/backend-go`: `REQUIRE_INFRA_ADAPTERS=true` causes bootstrap fail-fast when Redis or Kafka adapters are unavailable, preserving fail-open fallback behavior when strict mode is off.
- 2026-05-17: P1 outbox durability bridge is now additive in `apps/backend-go`: `outbox/spanner_store.go` binds relay `Fetch`/`MarkPublished` authority to `OutboxEvents` when Spanner is reachable, while bootstrap keeps in-memory fallback when unavailable; scaffold repositories now emit through an appender seam so relay authority can follow the selected store.
- 2026-05-17: P1 strict-mode startup tests are now active in `apps/backend-go/bootstrap/bootstrap_test.go` and assert fail-fast behavior for missing Redis/Kafka adapters plus successful startup/cleanup when both adapters are healthy.
- 2026-05-17: P1 request-side reliability middleware is now additive in `apps/backend-go`: `bootstrap/reliability_middleware.go` enforces fixed-window rate limits, priority-based in-flight shedding (critical-path wait, low-priority shed), and per-class circuit-open responses; `main.go` mounts it immediately after trace middleware, `RELIABILITY_MIDDLEWARE_ENABLED` controls runtime activation, and focused middleware tests + Spanner-store integration coverage (`outbox/spanner_store_integration_test.go`) now validate behavior.
- 2026-05-17: P2 payment execution seam is now additive in `apps/backend-go/payment`: `payment/execution.go` introduces provider execution routing with bounded retry/backoff+jitter and policy-typed errors (`payment_gateway_policy_violation`, `card_tokenization_gateway_unsupported`), `service.go` now routes checkout/chargeback/reversal execution decisions through that seam before persistence/outbox emission, and `AIRWALLEX_DIRECT_EXECUTION_ENABLED` gates AIRWALLEX direct execution support.
- 2026-05-18: P2 checkout attempt metadata persistence is now additive in `apps/backend-go/payment`: repository seam now includes `SaveAttempt`, checkout flow persists `PaymentAttemptRecord` (`attempt_id`, `execution_action`, `execution_mode`, `provider_reference`) after session creation, and checkout responses/events now surface those additive execution fields for downstream parity.
