# pegasusX ACT Protocol

ACT = **A**udit, **C**ompanion, **T**ransparency. Required for every technical request in pegasusX.

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
1. Reconcile session memory with the active plan.
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
