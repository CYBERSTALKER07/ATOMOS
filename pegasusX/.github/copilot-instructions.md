# pegasusX Copilot Instructions

You are operating inside `pegasusX/`, a single-tenant logistics stack and a sibling project to `pegasus/`.

## Hard Boundaries
1. **Never modify files under `../pegasus/`.** Read-only reference.
2. **Never modify workspace-root files** (`../package.json`, `../.github/*`, `../void-theme/*`).
3. **All work goes inside `pegasusX/`.** Sibling-project isolation.
4. **Single source of truth**: this folder. Code, infra, contracts, docs all live here.

## Primary Directive
Build and operate a single-supplier logistics ecosystem with full architectural compatibility to the Pegasus multi-supplier reference. Single-tenant is a deployment constraint, not a schema simplification — `SupplierId` stays everywhere; one supplier is seeded; no API contracts are simplified away.

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
- `context/architecture.md`
- `context/architecture-graph.json`
- `context/technology-inventory.md`
- `context/technology-inventory.json`
- `context/parity-ledger.md` (when behavior diverges from Pegasus)

## Working Standard
- Verify the actual `pegasusX/` filesystem before assuming a route, type, or model exists.
- Use `../pegasus/` only as a reference grep target. Do not re-import code from Pegasus; reimplement locally.
- Hunt full chain of impact (backend, frontend, mobile, shared contracts, infra) before declaring work done.
- Record divergence from Pegasus in `context/parity-ledger.md`.

## Runtime Additive Note (2026-05-17)
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
