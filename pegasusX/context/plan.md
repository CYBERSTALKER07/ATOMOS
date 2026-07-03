# pegasusX Enterprise Execution Plan

Last updated: 2026-07-03 (synced with codebase audit: driver iOS path, SSMR green locally, PX-ECS-1..4 shipped)

**See also (granular per-backend-feature + per-role/app breakdown with UI replication targets, cross-sync obligations, and phase mapping):** `VEGETABLE_PLAN.md` in this directory. It is the "VeggieTales" / feature-by-feature ledger Boss requested. Read it together with this file before any phase work. All status updates must propagate to both.

**Supplier role execution ledger:** `SUPPLIER_PHASE.md` — phased tracker for supplier-portal + native apps (start here when Boss says "supplier").

**Planning brain (90-day track):** [`plan_90.md`](plan_90.md) — o9-inspired MEIO, actionable control tower, demand baseline, scenario sandbox, and EKG-lite for pegasusX (single-supplier), with pegasus multi-supplier handoff contracts. **PX91 extension (gates, ingest, confidence UI, promo sandbox):** [`PlanDigitalBrain.md`](PlanDigitalBrain.md). **Production cutover & scale (single-supplier, math-only forecast, ML collect-later):** [`plan_production_scale.md`](plan_production_scale.md). **Ecosystem data flow & realtime sync (2026-07 audit):** [`plan_ecosystem_sync.md`](plan_ecosystem_sync.md) — cross-role desync fixes, planning↔execution coherence, staging proof. **Local closure without GCP billing (2026-07):** [`plan_local_closure.md`](plan_local_closure.md) — finish all non-cloud anchors (PX-LC-0..6). Execution anchors remain in this file; PX90/PX91 anchors live in `plan_90.md` and `PlanDigitalBrain.md`.

## Plan Authority
1. This file is the canonical phased execution roadmap for `pegasusX/`.
2. Every non-trivial implementation batch must map to one or more plan anchors in this file.
3. After each batch, update plan-anchor status using `implemented`, `in progress`, `blocked`, or `deferred`.
4. If code reality diverges from the roadmap, code remains the source of truth and this file must be updated in the same change set.
5. Scope is `pegasusX/` only. `../pegasus/` remains reference-only.

## Program Goal
1. Deliver a single-supplier logistics ecosystem with role-row parity across supplier, retailer, driver, warehouse, factory, and payload surfaces.
2. Preserve architectural compatibility with the Pegasus reference (`context/PEGASUS_REFERENCE.md`) so contracts, event names, and operating concepts stay migratable.
3. Reach production quality for high-volume operation: comfortable at 1M-request-class daily traffic, durable under bursty concurrent demand, and structurally ready for multi-million request growth.
4. Keep local Docker SSMR validation as the mandatory proof loop for the same critical flows expected in production cloud environments.
5. Use Spanner, Kafka, Redis, Firebase, Kubernetes, and Terraform as first-class platform primitives, not post-launch add-ons.

## Platform Commitments
1. Spanner is the durable source of truth for transactional state, audit-critical records, and index-backed operational reads when infra adapters are available; local scaffold fallback may use in-memory repository seams, while strict validation and production-like runs require Spanner adapters (`REQUIRE_INFRA_ADAPTERS=true`).
2. Kafka is the mandatory backbone for reliable business-event fanout, replay-safe async processing, and decoupled worker execution in strict validation and production-like runs; local scaffold fallback may use non-durable publisher seams outside strict mode.
3. Redis is the fast coordination layer for cache invalidation, websocket relay, rate limiting, perimeter membership, and low-latency operational lookups in strict validation and production-like runs; local scaffold fallback may use in-memory cache/pubsub seams outside strict mode.
4. Firebase bearer auth is the mobile-facing identity path when enabled; supplier setup remains cookie-session based.
5. Kubernetes is the target runtime for stateless scale-out, graceful shutdown, and controlled rollout behavior.
6. Terraform is the canonical source for managed cloud infrastructure wiring and environment separation.

## Delivery Principles
1. Product work and engineering work ship together. A user-facing phase is not complete unless backend, contracts, UI, realtime, and operations are coherent.
2. Non-technical readiness is part of delivery. Support procedures, onboarding, training, policy, and incident handling are planned alongside code.
3. Reliability is designed in early. Outbox, idempotency, bounded retry, backoff with jitter, traceability, and cache correctness are mandatory.
4. Role-row parity is non-negotiable. A capability for one role must land across that role's supported clients unless intentionally flagged and time-boxed.
5. Local proof is mandatory. New infrastructure-sensitive behavior must pass the SSMR sandbox before it is treated as production-candidate.

## Execution Status Snapshot
1. `PX0-A1` Plan authority, sync enforcement, and reconciliation template — `implemented`.
2. `PX0-A2` Role-row capability and ownership ledger - `implemented`.
3. `PX0-A3` Contract and event baseline freeze - `implemented`.
4. `PX0-A4` Reliability and scale acceptance profile - `implemented`.
5. `PX0-A5` Support, release, and observability baseline - `implemented`.
6. `PX1-A1` Supplier identity, session, and billing durability baseline - `implemented`.
7. `PX1-A2` Supplier topology builder and node provisioning - `implemented`.
8. `PX2-A1` Retailer registration and seeded supplier relationship - `implemented`.
9. `PX2-A2` Cart sync, serviceability, and order capture - `implemented`.
10. `PX2-A3` Retailer pricing and catalog integrity - `implemented`.
11. `PX3-A1` Payment session, attempt, and webhook durability baseline - `implemented`.
12. `PX3-A2` Settlement, ledger, and reconciliation authority - `implemented`.
13. `PX4-A1` Warehouse operational durability - `implemented`.
14. `PX4-A2` Factory manifest lifecycle durability - `implemented`.
15. `PX4-A3` Payload execution and reassignment durability - `implemented`.
16. `PX5-A1` Driver execution baseline - `implemented`.
17. `PX5-A2` Telemetry and live tracking integrity - `implemented`.
18. `PX5-A3` Retailer receipt, proof, and collection integrity - `implemented`.
19. `PX7-A1` SSMR sandbox and infra proof gate baseline - `implemented`.
20. `PX7-A2` Load, resilience, and security certification - `implemented`.
21. `PX7-A3` Launch readiness, support, and hypercare - `implemented`.
22. `PX8-A1` Factory desktop portal role-row parity - `implemented` (typecheck verified 2026-06-04; Tauri build optional local proof).
23. `PX8-A2` Driver iOS canonicalization (`driver-app-ios/driverappios` Xcode target) - `implemented`.
24. `PX8-A3` Supplier native row (iOS, Android, desktop) - `implemented` (iOS + Android native; desktop via `supplier-portal` Tauri; `supplier-app-desktop` README anchor).
25. `PX8-A4` Cross-role client parity audit + Gradle wrapper baseline - `implemented` (matrix in `parity-ledger.md`; all six `*-android` apps have `gradlew`).
26. `COOL-W0` DeliveryExpectation + ExplainStatus + Pulse API + Handoff inbox metadata — `implemented` (2026-06-29).
27. `COOL-W1` Pulse UI, handoff inbox cells, explain banners — web portals + factory; Android/iOS driver/payload pulse strips, native handoff inbox, manifest-gate/seal explain — `implemented` (2026-06-29).
28. `COOL-W1-polish` Handoff `primary_link` native navigation (driver + payload Android/iOS), batch seal per-row explain (backend + payload row), factory portal explain banners — `implemented` (2026-06-29).
29. `COOL-W2` Warehouse Spanner manifests, dispatch replay, tomorrow board, yard radar, exception triage — `implemented` (2026-06-29).
30. `COOL-W3` Supplier exception weather map, broadcast templates, override preview; factory lane board + fulfill helper — `implemented` (2026-06-29).
31. `COOL-W3.5` Warehouse depot broadcasts (builtin + saved custom templates) + read-only pricing preview — `implemented` (2026-06-29).

## PX-9 Enterprise Hardening (P0 slice — 2026-06-04)

Production-safety closure before manifest Spanner durability (PX9-B).

| Anchor | Scope | Status | Notes |
|---|---|---|---|
| `PX9-A1` | Shop-closed protocol (DDL + handlers + dispatcher + SSMR) | `implemented` | `ShopClosedAttempts` table; `order/shop_closed.go`; routes wired; `PX_E2E_SHOP_CLOSED_OK` |
| `PX9-A2` | `DELIVERY_DISPUTED` producer on chargeback | `implemented` | Dual outbox emit in `payment.HandleChargeback` |
| `PX9-A3` | Production webhook secret fail-fast | `implemented` | `PEGASUSX_ENV=production` rejects `dev-*` secrets |
| `PX9-A4` | Driver compat honesty (501 / 503) | `implemented` | No fake `200 ok` on unimplemented compat routes |
| `PX9-B` | Factory/payload manifest Spanner durability | `implemented` | `SupplierTruckManifests`, `ManifestOrders`, `ManifestExceptions`, `FactoryTruckManifests` DDL; `manifest.Store` atomic commit; payload/factory `SpannerRepository` RW txn + outbox; demo seed on bootstrap warm |
| `PX9-C` | Auth scope enforcement (JWT home node + factory↔warehouse linkage) | `implemented` | `auth/{factory_scope,warehouse_scope,warehouse_ops_scope,home_node,scope_body,seed_scope}.go`; `Warehouses.PrimaryFactoryId` DDL; route middleware on `factoryroutes`/`warehouseroutes`/`supplierroutes`; `EnsureDemoScopeLinks` at bootstrap; dispatch preview + warehouse ops read scoped warehouse from context |
| `PX9-D` | Kafka consumer hardening (partition pool, trace headers, DLQ, dedup) | `implemented` | `kafka/workerpool` partition-parallel consumer; `trace_id` Kafka headers from outbox relay; malformed envelopes → DLQ; 30s WS fan-out dedup; `DRIVER_CREATED`/`VEHICLE_CREATED` split handlers |
| `PX9-E` | Supplier portal admin-parity wiring | `implemented` | `/v1/supplier/shop-closed/*`, `orders/payment-bypass`, `negotiate/resolve` (503 until DDL), `route/approve-early-complete`, `empathy/adoption`, `broadcast`, `replenishment/trigger`, `fleet/orders`; `supplier-portal` pages + `@pegasusx/api-client` |
| `PX9-F` | Load certification gate | `implemented` | `scripts/load/load_cert.sh` + k6 retailer/supplier profiles; `loadtokens` smokecheck; `make load-cert` / `load-cert-ssmr`; `docs/LOAD_TEST_REPORT.md` from artifacts |

## PX-10 Launch evidence & production closure (2026-06-04)

| Anchor | Scope | Status | Notes |
|---|---|---|---|
| `PX10-A1` | Staging load cert (`LOAD_PROFILE=cert`) | `implemented` | Passed `20260604-181832`: 200 VU reads p99 59ms, 85 orders p99 57ms, 0% failures; supplier p99 669ms; artifacts + `docs/LOAD_TEST_REPORT.md` |
| `PX10-A2` | Launch readiness bundle | `implemented` | `__SSMR_OK__` 2026-06-04 incl. `PX_E2E_NEGOTIATION_OK`; `validate-ai-worker-k8s` + `validate-launch-readiness` green |
| `PX10-B1` | Negotiation resolve (Edge 28) | `implemented` | `NegotiationProposals` DDL; `order/negotiation.go` propose + resolve; `POST /v1/delivery/negotiate` + `/v1/supplier/negotiate/resolve` |
| `PX10-B2` | Negotiation closure (E2E + dispatcher + portal) | `implemented` | `GET /v1/supplier/negotiations/pending`; Kafka fan-out `NEGOTIATION_*`; `PX_E2E_NEGOTIATION_OK`; supplier-portal `/exceptions/negotiations` + api-client |

**Plan reconciliation:** `PX4-A2` / `PX4-A3` operational semantics now include **durable manifest projection** (PX9-B). Demo payload orders remain service L1 cache until checkout-seeded `Orders` rows exist; manifest + junction tables are Spanner-authoritative under `REQUIRE_INFRA_ADAPTERS=true`.

## PX-11 Enterprise engine + deployment (2026-06-04)

| Anchor | Scope | Status | Notes |
|---|---|---|---|
| `PX11-A1` | Staging/cloud proof scripts | `implemented` | `scripts/load/load_cert_cloud.sh`, `scripts/cloud_smoke_ssmr.sh`; `infra/terraform/gke.tf` (opt-in `enable_gke`) |
| `PX11-A2` | GKE + Artifact Registry + WI | `implemented` | Terraform outputs `gke_cluster_name`, `artifact_registry_repository`, `backend_runtime_service_account` |
| `PX11-A3` | Managed Kafka wiring | `implemented` | Existing Secret Manager bootstrap secrets; consumer uses `Push` bridge |
| `PX11-B1` | FCM/APNs transport | `implemented` | `notifications/fcm.go`, `notifications/push.go`; `FIREBASE_CREDENTIALS_PATH` in bootstrap |
| `PX11-B2` | Supplier-native WS | `implemented` | `supplier-app-android` `SupplierWebSocket`; `supplier-app-ios` `SupplierRealtimeClient` |
| `PX11-B3` | Dispatcher + manifest fan-out | `implemented` | `MANIFEST_*` prefix handler; warehouse supply events; driver/retailer FCM fallback |
| `PX11-B4` | Offline/reconnect contract doc | `implemented` | `docs/DEPLOYMENT_AND_DISTRIBUTION_PLAN.md` smart-update + deferral rules |
| `PX11-C1` | Client policy API | `implemented` | `GET/PUT /v1/platform/client-policy`; `platform/` package |
| `PX11-C2` | Policy + device token DDL | `implemented` | `ClientVersionPolicies`, `DeviceTokens` in `schema/spanner.ddl` |
| `PX11-C3` | `SYSTEM_APP_OUTDATED` producer | `implemented` | WS connect path in `ws/handler.go` via `platform.Service` |
| `PX11-C4` | Safe-update deferral | `implemented` | `platform/deferral.go` active session checks |
| `PX11-D1` | Payment Spanner path | `implemented` | `payment.NewSpannerRepository` wired when Spanner up (pre-existing) |
| `PX11-D2` | FACTORY_ADMIN warehouse scope | `implemented` | `auth/warehouse_scope.go` linkage (pre-existing PX9-C) |
| `PX11-D3` | Webhook fail-fast | `implemented` | PX9-A3 + cloud checklist |
| `PX11-D4` | Outbox audit | `implemented` | State paths use `outbox.EmitJSON`; telemetry-only inline Kafka |
| `PX11-E1` | Contract parity script | `implemented` | `scripts/parity/role_row_contract_check.sh` |
| `PX11-E2` | Feature parity matrix | `implemented` | Folded into PX-12 role-row phases + `docs/qa/PX12_ROLE_ROW_QA.md` |
| `PX11-F1` | Incident runbook | `implemented` | `docs/INCIDENT_RESPONSE_RUNBOOK.md` |
| `PX11-F2` | Release train | `implemented` | `docs/RELEASE_TRAIN.md` |

**Exit:** `PX_E2E_CLIENT_POLICY_OK` in SSMR e2e; `make validate-launch-readiness`; deployment doc published.

## PX-12 Full ecosystem deployment readiness (2026-06-04)

Production v1 bar: all six roles deployable, no silent 404/501 on shipped client APIs, role-row UI parity (Boss-approved surfaces), staging→prod cloud gates. Pegasus ~59-route parity remains out of scope (see `docs/DEPLOYMENT_READINESS_GAP_LEDGER.md`).

| Anchor | Scope | Status | Notes |
|---|---|---|---|
| `PX12-A1` | Gap ledger + production v1 matrix column | `implemented` | `docs/DEPLOYMENT_READINESS_GAP_LEDGER.md` |
| `PX12-A2` | `role_row_contract_check_full.sh` | `implemented` | `make parity-contract-full` |
| `PX12-B1` | Driver 501 closure (reorder, bypass, credit, missing, split) | `implemented` | `order/driver_edges.go` |
| `PX12-B2` | `GET /v1/catalog/categories/{id}/suppliers` | `implemented` | `catalog` |
| `PX12-B3` | Retailer auto-order + checkout truth | `implemented` | `retailer/auto_order.go`; card → `use_checkout_unified` |
| `PX12-B4` | Driver/payload device-token + FCM | `implemented` | platform + driver mobile |
| `PX12-B5` | SSMR markers for driver edges | `implemented` | `PX_E2E_DRIVER_EDGES_OK` |
| `PX12-C1` | Supply-request WS fan-out audit | `implemented` | `notification_dispatcher` |
| `PX12-C2` | `gap_hunter_gate.sh` | `implemented` | CI gate |
| `PX12-D1` | Path-filtered GitHub Actions | `implemented` | `.github/workflows/ci.yml` |
| `PX12-D2` | PR gates in Makefile + launch validator | `implemented` | |
| `PX12-E1` | Staging cutover checklist | `implemented` | Boss GCP execution per runbook |
| `PX12-F` | Driver role-row UI parity | `implemented` | contract + push wiring; QA `docs/qa/PX12_ROLE_ROW_QA.md` |
| `PX12-G` | Retailer role-row UI parity | `implemented` | catalog suppliers + auto-order API |
| `PX12-H` | Supplier role-row UI parity | `implemented` | portal primary; native ops slice unchanged |
| `PX12-I` | Warehouse role-row UI parity | `implemented` | supply + dispatch surfaces |
| `PX12-J` | Factory role-row UI parity | `implemented` | manifest + supply surfaces |
| `PX12-K` | Payload role-row UI parity | `implemented` | device-token via platformroutes |
| `PX12-L1` | Prod release train + hypercare | `implemented` | closes PX11-E2 |
| `PX12-M1` | v1 staging closure (2026-06-29) | `implemented (local)` + `deferred (staging LC-01–06, GCP billing)` | SSMR + `px12-preflight-ok`; local closure PX-LC-0..5 shipped; staging boss checklist [`docs/V1_STAGING_CLOSURE_CHECKLIST.md`](../docs/V1_STAGING_CLOSURE_CHECKLIST.md) |

**Exit:** `parity-contract-full` + `test-ssmr-infra` + `px12-preflight-ok` + staging LC sign-off + per-role QA (`docs/qa/PX12_*`) + war-story Phase C.

## PX-8 Client Surface Parity Closure

Closes the gap between pegasusX client trees and the Pegasus reference (`context/PEGASUS_REFERENCE.md`) where apps exist but pages, Tauri shells, or backend wiring are incomplete. Work is phased by largest structural drift first; each phase updates `context/parity-ledger.md` in the same change set.

| Anchor | Role / surface | Status | Exit evidence |
|---|---|---|---|
| `PX8-A1` | Factory — `factory-portal` (desktop) | `implemented` | All Pegasus portal routes + Tauri 2 + `8180` auth/WS; `pnpm typecheck` clean (Boss verified 2026-06-04) |
| `PX8-A2` | Driver — iOS | `implemented` | Canonical Xcode project at `apps/driver-app-ios/driverappios` (target `driverappios`); outer `driver-app-ios/README.md` is entry pointer only |
| `PX8-A3` | Supplier — iOS / Android / desktop | `implemented` | `supplier-app-ios`, `supplier-app-android`, `supplier-portal` Tauri desktop |
| `PX8-A4` | Warehouse / retailer / factory / driver audit | `implemented` | Role-row matrix + Gradle wrappers documented in `parity-ledger.md` |

### PX8-A1 Factory desktop portal (active)

- **Reference**: `pegasus/apps/factory-portal` (read-only).
- **Target**: `pegasusX/apps/factory-portal` with `@pegasusx/*` workspace deps, port `3003`, backend `8180`, `com.pegasusx.factory` Tauri id, WS at `/v1/ws?token=`.
- **Non-goals this phase**: UI redesign (UI freeze); backend factory durability changes.

## Ordered Plan Anchors

### PX-0 anchors
1. `PX0-A1` Governance authority and sync enforcement
  Status: `implemented`
  Non-technical: establish one roadmap, one reconciliation model, and one documentation sync rule for pegasusX.
  Technical: wire `context/plan.md` into instruction files, context docs, and architecture graph as the canonical execution ledger.
  Dependencies: none.
  Exit evidence: plan authority is referenced by local doctrine and included in the sync set.
2. `PX0-A2` Role-row capability and ownership ledger
  Status: `implemented`
  Non-technical: define what each role owns, what support covers, and what outcomes each app must provide.
  Technical: produce a capability matrix mapping roles, apps, endpoints, events, offline expectations, and realtime consumers.
  Dependencies: `PX0-A1`.
  Exit evidence: role rows have a capability ledger that future feature work can diff against.
3. `PX0-A3` Contract and event baseline freeze
  Status: `implemented`
  Non-technical: make product and support terminology stable enough for training and rollout planning.
  Technical: lock route families, event names, aggregate IDs, DTO ownership, and shared-codegen expectations across TS, Kotlin, and Swift.
  Dependencies: `PX0-A1`.
  Exit evidence: canonical route and event inventory exists and can be validated before edits.
4. `PX0-A4` Reliability and scale acceptance profile
  Status: `implemented`
  Non-technical: define what “ready for 1M-request-class operation” means in business and support terms.
  Technical: set acceptance budgets for Spanner latency and indexing, Kafka lag and replay safety, Redis invalidation correctness, websocket fanout, webhook replay, and Kubernetes rollout behavior.
  Dependencies: `PX0-A1`.
  Exit evidence: every later phase has measurable reliability gates instead of vague quality goals.
5. `PX0-A5` Support, release, and observability baseline
  Status: `implemented`
  Non-technical: define hypercare ownership, incident paths, escalation ladders, and launch-support coverage.
  Technical: inventory dashboards, alerts, audit trails, rollback steps, and release evidence required before launch.
  Dependencies: `PX0-A2`, `PX0-A4`.
  Exit evidence: support and release readiness are visible before the product enters hardening.
  Current delta (2026-05-23): `infra/terraform/observability.tf` now gives pegasusX its first launch-readiness observability automation for the ai-worker seam: optional alert policies for `void_ai_worker_up`, `void_ai_worker_ready`, and `void_kafka_consumer_lag_seconds`, optional uptime checks on `/healthz` and `/ready` when `ai_worker_monitoring_host` is provided, and a launch dashboard for the worker metrics. `infra/terraform/main.tf` now enables `monitoring.googleapis.com`, `infra/terraform/variables.tf` carries explicit observability toggles and notification-channel inputs, and `infra/terraform/README.md` plus `docs/AI_WORKER_LAUNCH_RUNBOOK.md` now document how launch owners turn that baseline on. Focused validation passed via `terraform fmt` and diff hygiene checks.
  Closure delta (2026-05-25): `scripts/validate_launch_readiness.py` now verifies the support runbook bundle, SSMR smoke gate, ai-worker Kubernetes gate, Terraform observability evidence, launch command entrypoints, and synchronized context inventory before launch approval. `docs/LAUNCH_READINESS_RUNBOOK.md` now defines release ownership, preflight sequence, rollback handling, and hypercare checks for the currently implemented engine. Entry points are wired through `make validate-launch-readiness` and `pnpm infra:launch:validate`.

### PX-1 anchors
1. `PX1-A1` Supplier identity, session, and billing durable baseline
  Status: `implemented`
  Non-technical: onboarding and billing recovery are documented for operators and support.
  Technical: durable supplier signup/login/profile/billing/session flows with claims-safe auth and idempotent retries.
  Dependencies: `PX0-A3`, `PX0-A4`.
  Exit evidence: supplier setup survives retries, partial failure, and re-login without manual repair.
2. `PX1-A2` Supplier topology builder and node provisioning
  Status: `implemented`
  Non-technical: factories and warehouses are represented as real operating nodes, not setup placeholders.
  Technical: topology creation, validation, and downstream visibility for warehouse/factory role rows and contracts.
  Dependencies: `PX1-A1`.
  Exit evidence: created nodes are immediately consumable by downstream apps and services.
3. `PX1-A3` Supplier control tower v1
  Status: `implemented`
  Non-technical: supplier operators can monitor configuration, inventory, earnings, and order oversight from one surface.
  Technical: dashboard, inventory, earnings, orders, and audit surfaces wired to stable supplier contracts.
  Dependencies: `PX1-A1`, `PX1-A2`.
  Exit evidence: supplier portal can act as the daily operating cockpit.
4. `PX1-A4` Supplier org, staff, and fleet onboarding contracts
  Status: `implemented`
  Non-technical: define how staff, drivers, vehicles, and node admins are introduced into the system.
  Technical: supplier org-member, driver, and vehicle contracts now exist end to end with topology validation, idempotent create support, shared DTO/client coverage, and a supplier-portal onboarding surface.
  Dependencies: `PX1-A2`.
  Exit evidence: supplier operators can list and create org members, drivers, and vehicles without blocking downstream node-admin and driver-role execution work.

### PX-2 anchors
1. `PX2-A1` Retailer registration and seeded supplier relationship
  Status: `implemented`
  Non-technical: retailers can join the ecosystem and understand their relationship to the single supplier.
  Technical: retailer auth/profile/supplier linkage baseline across desktop and native apps.
  Dependencies: `PX0-A3`, `PX1-A1`.
  Exit evidence: retailer identity and supplier association are stable across clients.
2. `PX2-A2` Cart sync, serviceability, and order capture
  Status: `implemented`
  Non-technical: retailers can build carts, place orders, and understand why an order is accepted or blocked.
  Technical: cart sync, zone validation, nearest-warehouse selection, order creation, and duplicate-request safety.
  Dependencies: `PX2-A1`, `PX0-A4`.
  Exit evidence: retailer commerce is resilient under reconnects, duplicate taps, and spatial edge cases.
3. `PX2-A3` Retailer pricing and catalog integrity
  Status: `implemented`
  Non-technical: catalog and pricing are explainable to supplier operators and retailer support staff.
  Technical: supplier pricing rules, retailer overrides, catalog contracts, and cross-client display consistency.
  Dependencies: `PX1-A3`, `PX2-A1`.
  Exit evidence: pricing authority is backend-controlled and surfaced consistently everywhere.
4. `PX2-A4` Retailer post-order decision flows
  Status: `implemented`
  Non-technical: cancellation, AI suggestions, preorder, pending-payment, and tracking flows are supportable and understandable.
  Technical: confirm/reject AI flows, cancel/request-cancel, preorder edit/confirm, active fulfillment, and tracking surfaces.
  Dependencies: `PX2-A2`, `PX3-A1`.
  Exit evidence: post-order retailer actions stay coherent across desktop and mobile.
  Delta 2026-05-31: retailer post-order decision authority now lives on the order aggregate instead of stub handlers. `apps/backend-go/order/{service.go,repository_spanner.go}` now persist additive order-source / confirmation lifecycle fields (`OrderSource`, `ConfirmationStatus`, `RequestedDeliveryDate`, `AutoConfirmAt`, `DecisionAt`, `DecisionBy`, `DerivedFromOrderId`) and expose confirm/reject AI plus preorder edit/confirm logic; `apps/backend-go/retailer/{service.go,core_handlers.go}` now delegate the mounted retailer AI/preorder endpoints to that order-owned lifecycle and expose `GET /v1/retailer/ai/predictions`; shared `packages/{types,api-client}` now carry the additive retailer AI/preorder contracts; and focused validation passed via `go test ./apps/backend-go/order ./apps/backend-go/retailer ./apps/backend-go/bootstrap`.

### PX-3 anchors
1. `PX3-A1` Payment session, attempt, and webhook durability
  Status: `implemented`
  Non-technical: finance and support can explain payment state transitions and replay behavior.
  Technical: durable session/attempt/webhook storage, provider-exact verification, idempotent chargeback and reversal flows.
  Dependencies: `PX0-A3`, `PX0-A4`.
  Exit evidence: one payment intent leads to one durable business outcome.
2. `PX3-A2` Settlement, ledger, and reconciliation authority
  Status: `implemented`
  Non-technical: treasury and support have a clear source of truth for cleared, pending, disputed, and reversed funds.
  Technical: double-entry-ready ledger model, settlement summaries, reconciliation reports, and mismatch handling.
  Dependencies: `PX3-A1`.
  Exit evidence: finance can reconcile without ad hoc database interpretation.
3. `PX3-A3` Supplier finance and dispute operations
  Status: `implemented`
  Non-technical: supplier operators have actionable payment and dispute visibility.
  Technical: supplier-facing payment status, exception, dispute, and treasury UI plus event-driven updates.
  Dependencies: `PX3-A2`, `PX1-A3`.
  Exit evidence: payment issues are operationally manageable from product surfaces.
  Delta 2026-05-23: `apps/backend-go/supplier/service.go` now mints short-lived `GET /v1/supplier/ws-session` tokens, `apps/backend-go/ws/handler.go` accepts signed `?token=` websocket auth fallback, `apps/backend-go/kafka/notification_dispatcher.go` now fans out supplier-scoped `PAYMENT_REQUIRED`/`PAYMENT_CLEARED`/`SETTLEMENT_REQUIRED`/`DELIVERY_DISPUTED`, `packages/{types,api-client}` now expose typed chargeback/reversal mutations, `apps/supplier-portal/app/payments/page.tsx` now live-refreshes from the supplier finance stream, `apps/supplier-portal/app/earnings/page.tsx` is now a real treasury/dispute operations surface, and legacy `GET /v1/supplier/earnings` is now a ledger-backed compatibility bridge with explicit `authority_source` and `authoritative` metadata.

### PX-4 anchors
1. `PX4-A1` Warehouse operational durability
  Status: `implemented`
  Non-technical: warehouse operators can trust inventory, order queues, dispatch preview, and supply requests.
  Technical: durable warehouse repository paths, cache discipline, node scoping, and realtime updates.
  Dependencies: `PX1-A2`, `PX2-A2`.
  Exit evidence: warehouse operations are not scaffold-only.
  Delta 2026-05-31: warehouse demand forecast is now backed by the same order-owned preorder truth that powers retailer AI/manual scheduling. `apps/backend-go/warehouse/service.go` now resolves warehouse scope from claims/query, validates `start_date`/`days`, and delegates `/v1/warehouse/demand/forecast` to `order.Service.WarehouseDemandForecast`; `bootstrap/bootstrap.go` injects the order service as the warehouse planner; forecast responses now expose `committed_units` and `pending_confirmation_units` in addition to projected units/revenue; and focused validation passed via `go test ./apps/backend-go/order ./apps/backend-go/warehouse ./apps/backend-go/bootstrap`.
  Delta 2026-06-02: warehouse role-row clients are now real consumers of the existing warehouse ops authority instead of placeholder shells. `packages/{types,api-client}/index.ts` now expose typed warehouse dashboard, inventory, orders, demand forecast, supply request, dispatch preview, and dispatch-lock contracts; `apps/warehouse-portal` now ships a polling-first direct-backend Next.js/Tauri dashboard over `/v1/warehouse/ops/*`, `/v1/warehouse/demand/forecast`, `/v1/warehouse/supply-requests`, and `/v1/warehouse/dispatch-lock*`, with `src-tauri` packaging and runtime backend targeting from `NEXT_PUBLIC_WAREHOUSE_BACKEND_BASE_URL` or `NEXT_PUBLIC_BACKEND_BASE_URL`; `apps/warehouse-app-ios` now carries an XcodeGen-backed SwiftUI shell with a direct `URLSession` client and polling view model over the same slice; and `apps/warehouse-app-android` now carries a Gradle/Compose shell with a direct HTTP client and matching polling dashboard. Focused validation passed via clean shared TS diagnostics, successful desktop export, successful `pnpm tauri build --debug`, clean Swift diagnostics plus `xcodegen generate && xcodebuild -project WarehouseAppIOS.xcodeproj -target WarehouseAppIOS -sdk iphonesimulator26.5 CODE_SIGNING_ALLOWED=NO build`, and clean Android diagnostics; full Android CLI assemble remains blocked because pegasusX still has no Gradle wrapper and no system `gradle` on `PATH`.
2. `PX4-A2` Factory manifest lifecycle durability
  Status: `implemented`
  Non-technical: factories can move from transfer planning to dispatch with visible state and recoverable exceptions.
  Technical: durable manifest lifecycle, dispatch, rebalance, cancellation, and exception handling.
  Dependencies: `PX4-A1`, `PX1-A4`.
  Exit evidence: factory movement is durable, replay-safe, and observable.
3. `PX4-A3` Payload execution and reassignment durability
  Status: `implemented`
  Non-technical: payload teams can fix physical loading reality without shadow tools.
  Technical: durable load/start/inject/seal/exception/reassign flows with source-target cache invalidation and typed fanout.
  Dependencies: `PX4-A2`.
  Exit evidence: payload repair flows are operationally trustworthy.

### PX-5 anchors
1. `PX5-A1` Driver execution baseline
  Status: `implemented`
  Non-technical: drivers can work quickly, safely, and with clear manifest expectations.
  Technical: profile, availability, manifest gate, pending collections, earnings, and delivery-state transitions.
  Dependencies: `PX4-A2`, `PX1-A4`.
  Exit evidence: driver execution is no longer blocked by upstream scaffolding gaps.
  Delta 2026-05-22: Tranche 1 landed — `PATCH /v1/driver/availability` now flows through `repo.Apply` + `outbox.EmitJSON(events.AggregateDriver, events.TopicMain, DRIVER_AVAILABILITY_CHANGED)`, post-commit cache invalidation of `driver:availability:{id}`, and dual `driver:{id}` + `supplier:{supplierID}` WS broadcast. Idempotent no-op branch skips outbox/cache/WS when target on-shift state is unchanged. Bootstrap now injects `cacheClient`, supplier+driver hubs, slog, and seeded supplier ID into `driver.NewService`.
  Delta 2026-05-22: Tranche 2 landed — `GET /v1/driver/manifest-gate` now requires `manifest_id` and resolves Ghost Stop Prevention from bootstrap-wired factory manifest state via `factory.Service.ManifestGateSnapshot`. `SEALED`/`DISPATCHED`/`COMPLETED` manifests return `200` with `cleared=true`/`allowed=true`, `stop_count`, and `volume_vu`; unknown manifests return `404 manifest_not_found`; pre-seal states return `403 AWAITING_PAYLOAD_SEAL`. Focused coverage now lives in `apps/backend-go/driver/service_test.go`.
  Delta 2026-05-22: Tranche 3 landed as a contract-compatibility bridge — `GET /v1/driver/pending-collections` now reads through an additive `PendingCollectionsLookup` seam in `driver.NewService`, normalizes legacy scaffold rows into Pegasus-style `pending_collections` items (`order_id`, `retailer_id`, `amount`, `state`, `updated_at`), and returns `{pending_collections, count}` while preserving legacy `pending` plus `amount_minor`/`due_at` aliases. The route remains read-only with no outbox/cache/ws side effects, and authoritative order/payment sourcing stays deferred to delivery-state durability. Focused coverage now lives in `apps/backend-go/driver/service_test.go`.
  Delta 2026-05-22: Tranche 4 landed as a read-only earnings contract bridge — `GET /v1/driver/earnings` now returns the Pegasus mobile contract (`total_deliveries`, `total_volume`, `total_routes`, `last_30_days`) through an optional `EarningsLookup` seam, adds currency/minor-unit aliases (`currency`, `today_minor`, `week_minor`, `month_minor`, daily `volume_minor`) for financial clarity, and preserves scaffold fallback from `earningsMinor` without inventing delivery/payout math. The route emits no outbox/cache/ws side effects; authoritative completed-delivery sourcing was deferred to delivery-state durability. Focused coverage now lives in `apps/backend-go/driver/service_test.go`.
  Delta 2026-05-22: Tranche 5 landed — `orderroutes` now mounts DRIVER-auth compatibility endpoints for `/v1/delivery/arrive` and `/v1/order/{deliver,confirm-offload,complete,collect-cash}`. `order.Service` now drives durable delivery transitions with transactional outbox events (`ORDER_STATUS_CHANGED`, `SETTLEMENT_REQUIRED`, `PAYMENT_REQUIRED`, `PAYMENT_CLEARED`, `ORDER_FINALIZED`), post-commit supplier/retailer order-cache invalidation, supplier/retailer websocket fanout, 500m cash geofence enforcement, and idempotent no-op replay suppression. `contracts/events.schema.json` and shared `OrderStatus` now include `AWAITING_PAYMENT` and `PENDING_CASH_COLLECTION` plus delivery payment/finalization payload shapes; focused coverage now lives in `apps/backend-go/order/service_test.go`. Driver transitions now enforce assigned-driver ownership when `Orders.DriverId` is present.
  Delta 2026-05-22: Tranche 6 landed as the driver manifest detail bridge — `GET /v1/driver/manifest` and legacy `GET /v1/fleet/manifest` no longer return a fake demo hash list. `driver.Service` now resolves manifest detail through a bootstrap-wired read-only lookup backed by `factory.Service.ManifestDetailSnapshotForDriver`, projecting `manifest`, `transfers`, `transitions`, `reassignments`, `exceptions`, `route_id`, `stop_count`, and `order_count` for the authenticated driver while keeping compatibility fields explicit (`hashes=[]`, `legacy_hashes_available=false`) instead of inventing token data. Focused coverage now lives in `apps/backend-go/driver/service_test.go` and `apps/backend-go/factory/service_test.go`.
2. `PX5-A2` Telemetry and live tracking integrity
  Status: `implemented`
  Non-technical: supplier and retailer users can trust what “live” tracking means.
  Technical: authenticated telemetry ingress, websocket fanout, reconnect safety, and tracking contract parity.
  Dependencies: `PX5-A1`, `PX0-A4`.
  Exit evidence: live route progress survives network instability and scale bursts.
  Delta 2026-05-22: Tranche 1 landed — `telemetryroutes` now rejects unauthenticated/non-driver location posts, derives `driver_id` from authenticated DRIVER claims instead of request bodies, validates coordinate/timestamp input, emits typed `DRIVER_LOCATION_UPDATED` websocket envelopes with `trace_id`, and broadcasts through scoped `TelemetryHub` rooms (`telemetry:driver:{driverID}`, `telemetry:supplier:{supplierID}`). `ws.RegisterRoutes` now subscribes driver and supplier-side callers to those telemetry rooms and wraps gorilla websocket writes with a per-connection mutex plus ping/pong deadlines for reconnect safety. `events/events.go`, `contracts/events.schema.json`, and `packages/types` include the new location event contract.
  Delta 2026-05-22: Tranche 2 landed — `Orders` now carries durable nullable assignment identity (`DriverId`, `VehicleId`, `RouteId`, `ManifestId`) with driver/route/manifest indexes, `POST /v1/orders/{orderID}/assign` lets ADMIN/WAREHOUSE_ADMIN/FACTORY_ADMIN callers persist assignment/reassignment through the order repository, emits `ORDER_ASSIGNED`/`ORDER_REASSIGNED` with additive assignment metadata, invalidates supplier/retailer order caches, and fans out supplier/retailer/driver websocket envelopes. Retailer tracking now projects active orders from `retailer.Repository.ListTrackingOrders` with assignment fields.
  Delta 2026-05-22: Tranche 3 landed — `telemetry/location_store.go` now persists the latest authenticated driver point in Redis-compatible cache with bounded TTL via `bootstrap.App.DriverLocations`; `telemetryroutes` writes claims-derived locations after validation and remains fail-open on cache errors; `retailer.HandleTracking` enriches only authenticated retailer-owned active assigned orders after supplier and freshness checks, returning additive `driver_location` plus `live_location_available=true` only for fresh scoped locations. `packages/types` now mirrors the additive `RetailerTrackingLocation` DTO.
  Delta 2026-05-22: Tranche 4 resilience proof landed in focused coverage — `telemetryroutes/routes_test.go` now asserts fail-open behavior when last-location cache writes fail (request still returns `202` and scoped websocket fanout still emits), and `retailer/core_handlers_test.go` now asserts stale cached locations are withheld (`live_location_available=false` with no `driver_location`) even when assignment exists.
  Delta 2026-05-22: Tranche 5 reconnect/load proof landed in websocket coverage — `ws/handler_test.go` now adds repeated driver reconnect churn validation for `telemetry:driver:{driverID}` subscriptions with per-cycle delivery assertions plus connection-leak checks (`Hub.Stats().Connections` converges to zero after each disconnect), and `ws/hub_test.go` now adds `BenchmarkHubBroadcastTelemetryFanout` to quantify local fanout cost across subscriber scales (`1`, `10`, `100`, `500`) for burst-readiness tracking.
  Delta 2026-05-22: Tranche 6 relay-scope proof landed additively — `ws/handler_test.go` now extends reconnect churn validation to supplier telemetry subscribers (`telemetry:supplier:{supplierID}`) with per-cycle delivery and leak checks, and `ws/hub_test.go` now adds `BenchmarkHubBroadcastTelemetryFanoutRelay` to measure cross-pod relay fanout overhead across subscriber scales (`1`, `10`, `100`, `500`) using paired telemetry hubs over the in-memory Pub/Sub backend.
  Delta 2026-05-22: Tranche 7 assertion-proof landed end-to-end — `ws/hub_test.go` now adds burst relay delivery integrity assertions (`TestStartRelaySubscriberDeliversBurstIntegrity`) to prove cross-pod delivery completeness under bounded publish bursts, and `telemetryroutes/routes_test.go` now adds an ingress→tracking freshness proof (`TestLocationIngressToRetailerTrackingFreshnessUnderReconnectChurn`) that drives authenticated telemetry posts through cache-backed last-location storage under subscriber reconnect churn and then validates retailer tracking fresh-vs-stale behavior from the same persisted location state.
  Delta 2026-05-22: Tranche 8 product hardening landed in runtime code — `telemetry/location_store.go` now enforces monotonic last-location persistence by reading the existing cached point before write and suppressing out-of-order regressions (older `reported_at`, or older `received_at` when timestamps are equal), while normalizing timestamp fields for deterministic ordering. This prevents stale network replays from overwriting fresher tracking state.
  Delta 2026-05-23: Tranche 9 landed as the first supplier-facing live-order consumer — `supplier/repository_spanner.go` now reads recent supplier-scoped `Orders` rows with durable assignment identity, `supplier/portal_handlers.go` `GET /v1/supplier/orders` hydrates those rows with fresh driver last-location snapshots from the same cache-backed telemetry authority while preserving scaffold fallback for local review-only entries, `bootstrap/bootstrap.go` now wires `DriverLocations` into supplier service, and `apps/supplier-portal/app/orders/page.tsx` no longer renders a static placeholder table. The page now consumes `/v1/supplier/orders` through the same-origin proxy plus shared `packages/{types,api-client}` contracts and surfaces assignment plus live/stale state for supplier order oversight. Role-row mobile/retailer client parity in pegasusX remains deferred, so the anchor stays in progress.
  Delta 2026-05-23: Tranche 10 landed as the first retailer-facing desktop consumer outside the supplier portal — `apps/retailer-app-desktop/app/page.tsx` now consumes `GET /v1/retailer/tracking` through the new same-origin proxy at `apps/retailer-app-desktop/app/api/[...path]/route.ts`, renders assignment plus live-vs-stale driver state from the existing PX5-A2 projection, and uses shared `packages/{types,api-client}` contracts instead of leaving the package as a pure stub. Package bootstrap files (`next-env.d.ts`, `tsconfig.json`, `next.config.mjs`, `app/layout.tsx`, `app/globals.css`) were added to make the desktop shell buildable, and focused build validation passed after syncing the local pnpm importer graph. PX5-A2 still stays in progress because retailer mobile and driver role-row clients in pegasusX remain placeholder-only.
  Delta 2026-05-23: Tranche 11 landed as the first retailer-facing mobile consumer on the role row — `apps/retailer-app-ios` is no longer a README-only placeholder and now carries an XcodeGen-backed SwiftUI shell (`project.yml`, `Sources/App/**`, `Sources/Networking/**`, `Sources/ViewModels/**`, `Sources/Views/**`) that reads `GET /v1/retailer/tracking` directly from backend-go through a safe `URLSession` client with local Codable mirrors of the existing retailer tracking contract. The dashboard renders loading, empty, restricted, error, refreshing, and stale states plus assignment, live-vs-stale driver location, and derived event timeline snapshots from the existing PX5-A2/PX5-A3 backend projections. Focused validation passed via `xcodegen generate` and a direct `xcodebuild -project RetailerAppIOS.xcodeproj -target RetailerAppIOS -sdk iphonesimulator26.5 CODE_SIGNING_ALLOWED=NO build`. PX5-A2 still stays in progress because retailer Android and driver mobile clients in pegasusX remain placeholder-only.
  Delta 2026-05-23: Tranche 12 landed as the retailer Android role-row companion — `apps/retailer-app-android` is no longer a README-only placeholder and now carries Gradle Android app metadata (`settings.gradle.kts`, `build.gradle.kts`, `gradle.properties`, `app/build.gradle.kts`), a Compose Material 3 activity shell, a safe direct client for `GET /v1/retailer/tracking`, local Kotlin DTO mirrors of the existing tracking contract, and a dashboard covering loading, empty, restricted, error, refreshing, and stale states plus assignment, live-vs-stale driver location, and derived event timeline truth from the existing PX5-A2/PX5-A3 backend projections. Focused validation passed through clean workspace diagnostics for `app/src/main/kotlin`, `app/src/main/AndroidManifest.xml`, and `app/build.gradle.kts`; full CLI assemble remains pending because this machine has no reusable Gradle runtime on PATH or cached wrapper distribution. PX5-A2 still stays in progress because the driver mobile row in pegasusX remains placeholder-only.
  Delta 2026-05-23: Tranche 13 landed as the first driver-facing mobile consumer on the role row — `apps/driver-app-android` is no longer a README-only placeholder and now carries Gradle Android app metadata (`settings.gradle.kts`, `build.gradle.kts`, `gradle.properties`, `app/build.gradle.kts`), a Compose Material 3 live-ops shell, a safe direct client for `GET /v1/driver/profile`, `GET|PATCH /v1/driver/availability`, `GET /v1/driver/manifest-gate`, `GET /v1/driver/manifest`, and gated `POST /v1/telemetry/location`, plus local Kotlin DTO mirrors of the driver live-ops contract. The dashboard handles loading, restricted, error, refreshing, stale, manifest, and telemetry states while keeping the backend truth explicit: local reads can use additive `driver_id` fallback, but telemetry posting requires a real DRIVER bearer token because telemetry ingress is claims-gated. Focused validation passed through clean workspace diagnostics for `app/src/main/kotlin`, `app/src/main/AndroidManifest.xml`, and `app/build.gradle.kts`; full CLI assemble remains pending because this machine has no reusable Gradle runtime on PATH or cached wrapper distribution. PX5-A2 still stays in progress because driver iOS plus websocket/live-map parity remain deferred in pegasusX.
  Delta 2026-05-23: Tranche 14 landed as the driver-role realtime parity closure without inventing a fake map substrate — `apps/driver-app-android` now consumes authenticated `/v1/ws` live updates through `DriverLiveSocketClient`, surfacing connection state, stream identity, last live event, and last location alongside the existing live-ops HTTP slice, and `apps/driver-app-ios` is no longer a README-only placeholder and now carries an XcodeGen-backed SwiftUI shell with direct `URLSession` HTTP access plus authenticated `/v1/ws` live-state consumption for the same driver surface. `apps/backend-go/driver/service.go` now emits `DRIVER_AVAILABILITY_CHANGED` with standard envelope metadata plus additive `available`, `on_shift`, `supplier_id`, `home_node_type`, and `home_node_id`, and shared contract parity is synced in `contracts/events.schema.json` plus `packages/types`. Focused validation passed through clean Android diagnostics and `xcodegen generate && xcodebuild -project DriverAppIOS.xcodeproj -target DriverAppIOS -sdk iphonesimulator26.5 CODE_SIGNING_ALLOWED=NO build`.
  Delta 2026-05-23: Tranche 15 landed as native map parity on the driver row — `apps/driver-app-ios/Sources/Views/DriverLiveLocationMapView.swift` now renders a real MapKit live-location card from the authenticated websocket stream, `apps/driver-app-android/app/src/main/kotlin/com/pegasusx/driver/ui/DriverLiveLocationMap.kt` now renders the same live-location authority through Google Maps Compose, and Android map-key plumbing now lives in `apps/driver-app-android/{app/build.gradle.kts,app/src/main/AndroidManifest.xml}` via `DRIVER_ANDROID_MAPS_API_KEY` with `MAPS_API_KEY` fallback. Focused validation passed through clean Android diagnostics for the touched Gradle, manifest, and Compose files plus `xcodegen generate && xcodebuild -project DriverAppIOS.xcodeproj -target DriverAppIOS -sdk iphonesimulator26.5 CODE_SIGNING_ALLOWED=NO build`. PX5-A2 is now implemented because the driver role row no longer lacks a native map renderer.
  Delta 2026-05-23: Tranche 16 landed as truthful route-overlay hardening on the driver row — both driver mobile apps now persist a bounded breadcrumb of recent authenticated websocket locations and render that history as a live polyline on the existing native maps (`apps/driver-app-ios/Sources/{ViewModels/DriverLiveOpsViewModel.swift,Views/DriverLiveLocationMapView.swift}` and `apps/driver-app-android/app/src/main/kotlin/com/pegasusx/driver/ui/{DriverLiveOpsViewModel.kt,DriverLiveLocationMap.kt}`). This remains an actual traveled-path overlay only: `GET /v1/driver/manifest` still does not expose stop coordinates or planned route geometry, so pegasusX does not claim planned-stop overlays that the backend cannot yet supply. Focused validation passed through clean Android diagnostics and `xcodegen generate && xcodebuild -project DriverAppIOS.xcodeproj -target DriverAppIOS -sdk iphonesimulator26.5 CODE_SIGNING_ALLOWED=NO build`.
  Delta 2026-06-11: Tranche 17 landed as manifest route-geometry persistence — `schema/migrations/20250613_supplier_manifest_route_geometry.ddl` adds `EncodedRoutePolyline` + `RouteGeometrySource` on `SupplierTruckManifests`, `manifest/geometry.go` persists geometry at seal and on driver stop reorder, `routing/{geometry.go,builder.go,osrm.go}` builds OSRM or haversine-dense polylines, and `cmd/backfill-route-geometry` backfills legacy manifests. `GET /v1/fleet/route/{routeID}/geometry` (driverroutes) serves planned overlays with optional `include_steps` and `reroute` query params; dispatch preview routes gain `route_geometry` via `routing.AttachRouteGeometryToProposedRoutes`. `ROUTING_OSRM_URL` is wired in `bootstrap/bootstrap.go`. Ops runbook: `docs/MIGRATION_RUNBOOK_MANIFEST_ROUTE_GEOMETRY.md`.
  Delta 2026-06-11: Tranche 18 landed as driver navigation parity — both driver mobile apps now consume `GET /v1/fleet/route/{routeID}/geometry` for planned polylines, OSRM turn-by-turn steps (`include_steps=true`), voice/haptic maneuver cues, and off-route reroute (`reroute=true&from_lat=&from_lng=`) via local deviation trackers (`RouteDeviation` / `RouteDeviationTracker`). Breadcrumb overlays from Tranche 16 remain the traveled-path truth; planned geometry is a separate overlay authority.
  Delta 2026-06-11: Tranche 19 landed as supplier fleet live map — `supplier/fleet_live_map.go` exposes `GET /v1/supplier/fleet/live-map` (SEALED/DISPATCHED manifests + stored polylines + cache-backed driver locations), shared contracts in `packages/{types,api-client}`, supplier-portal `FleetLiveMap` dashboard anchor, `supplier-app-ios` `FleetLiveMapView`, and `supplier-app-android` list-first then MapLibre `FleetLiveMapScreen`. Focused validation: `go test ./supplier/...`, portal typecheck.
  Delta 2026-06-11: Tranche 20 landed as supplier realtime map refresh — `supplier-portal/lib/{supplier-ws-events.ts,use-supplier-ws-refresh.ts,use-fleet-live-map.ts}` accelerates live-map refresh on supplier websocket events with 15 s polling fallback; dashboard dispatch-queue compact list restored from `recentManifests`.
  Delta 2026-06-11: Tranche 21 landed as operator fleet-map expansion — warehouse `GET /v1/warehouse/ops/fleet/live-map` (`warehouse/fleet_live_map.go`, warehouseroutes), warehouse-portal `FleetLiveMap` on dashboard + dispatch with animated driver markers (`use-animated-driver-markers.ts`), supplier-portal matching marker animation, and supplier-app-android MapLibre polylines via `FleetLiveMapLibre.kt`. Warehouse native apps remain portal-handoff for live fleet map (documented gap in `docs/ROLE_ROW_PARITY_MATRIX.md`). Focused validation: `go test ./warehouse/...`, warehouse-portal typecheck, `./gradlew :app:compileDebugKotlin` (supplier-android).
3. `PX5-A3` Retailer receipt, proof, and collection integrity
  Status: `implemented`
  Non-technical: delivery completion, cash collection, and receipt-side issues are supportable and auditable.
  Technical: geofence-sensitive completion, collection workflows, proof/receipt events, and retailer-visible completion state.
  Dependencies: `PX5-A1`, `PX3-A2`.
  Exit evidence: post-delivery disputes can be resolved from system truth.
  Delta 2026-05-22: Tranche 1 landed as the retailer fulfillment and pending-payment bridge — `GET /v1/retailer/active-fulfillment` and `GET /v1/retailer/pending-payments` no longer return empty scaffold arrays. Both routes now read from the existing retailer tracking projection, reuse live-location enrichment, and expose delivery/payment state directly from durable order truth: active fulfillment returns live orders with assignment/location/status metadata, while pending payments filters the same projection to `AWAITING_PAYMENT` and `PENDING_CASH_COLLECTION` with additive `{status, count, pending}` response metadata. Shared contracts now live in `packages/types`, and `packages/api-client` now exposes typed retailer methods for tracking, active fulfillment, and pending payments.
  Delta 2026-05-23: Tranche 2 landed as a derived retailer tracking-event timeline — `GET /v1/retailer/tracking` no longer hardcodes `events: []`. `retailer/core_handlers.go` now derives additive `ORDER_CREATED` and `ORDER_STATUS_SNAPSHOT` items from durable order `created_at`, `updated_at`, and current `status`, sorts them newest-first, and marks them explicitly with `derived=true` and `source=ORDER_ROW` so current receipt/payment/completion state is supportable without pretending to replay the full outbox stream. `packages/types` now types `RetailerTrackingEvent[]`, and focused coverage in `retailer/core_handlers_test.go` now proves active timelines, idle empty state, and payment-status snapshots. Dispute-specific event history remains deferred.
    Delta 2026-05-23: Tranche 3 landed as bounded recent receipt visibility on the existing tracking seam — `retailer/repository_spanner.go` now serves additive `recent_receipts` from recent `COMPLETED` `Orders` rows ordered by `updated_at`, `retailer/core_handlers.go` keeps `status` tied to active tracking rows while merging those completed snapshots into the derived timeline, and focused coverage in `retailer/core_handlers_test.go` now proves active/receipt separation plus receipt-only idle snapshots. Shared `packages/types` and all three retailer tracking consumers (`apps/retailer-app-desktop`, `apps/retailer-app-ios`, `apps/retailer-app-android`) now render recent completed receipt snapshots from durable order rows without inventing dispute-grade proof history; outbox-backed immutable receipt history remains deferred.
    Delta 2026-05-23: Tranche 4 landed as immutable payment evidence on the same retailer receipt seam — `retailer/service.go` now defines additive `TrackingPaymentEvidence`, `retailer/repository_spanner.go` batches order-scoped stale reads against `PaymentLedgerEntries` to attach latest `payment_evidence` to `recent_receipts`, and focused coverage in `retailer/core_handlers_test.go` now proves evidence passthrough plus order-id dedupe. Shared `packages/types` and all three retailer tracking consumers (`apps/retailer-app-desktop`, `apps/retailer-app-ios`, `apps/retailer-app-android`) now render latest immutable payment evidence from ledger truth on receipt cards without inventing a dispute-grade proof dossier; session-scoped-only reversal coverage remains deferred.
    Delta 2026-05-23: Tranche 5 landed as a session-scoped reversal overlay on the same retailer receipt seam — `retailer/repository_spanner.go` now keeps the existing order-scoped immutable payment-evidence read, recovers latest non-empty payment-session ids from those same ledger rows, and performs a second bounded stale read for session-scoped `CHARGEBACK_REVERSAL_RECORDED` entries so `recent_receipts.payment_evidence` promotes later immutable reversal truth when it belongs to the same payment session. Focused coverage in `retailer/core_handlers_test.go` now proves session-id dedupe plus newer-reversal override behavior; shared contracts and retailer clients required no shape change because existing payment-evidence rendering is already generic. Full dispute-grade proof dossiers and complete reversal history remain deferred.
    Delta 2026-05-23: Tranche 6 landed as a dispute-support receipt dossier on the same retailer receipt seam — `retailer/service.go` now defines additive `TrackingReceiptDossier`, `TrackingReceiptPaymentRecord`, `TrackingReceiptGatewayWebhook`, and `TrackingReceiptProofStatus`, while `retailer/repository_spanner.go` replaces the latest-only overlay path with bounded order-scoped and session-scoped `PaymentLedgerEntries` timeline reads plus order-scoped `PaymentWebhooks` reads to build `recent_receipts.receipt_dossier`. Backward compatibility is preserved because `payment_evidence` is now derived from the newest dossier timeline row, and shared `packages/types` plus retailer desktop/iOS/Android consumers now render the additive dossier on receipt cards.
    Delta 2026-05-23: Tranche 7 landed as the PX5-A3 closure slice — `schema/spanner.ddl` now provisions immutable `OrderDeliveryProofs`, `order/{service.go,repository_spanner.go}` now persist `QR_HANDOFF`, `FINALIZATION_GEOFENCE`, and `CASH_COLLECTION_GEOFENCE` artifacts atomically with driver order transitions, and `retailer/{service.go,repository_spanner.go}` now widen `recent_receipts.receipt_dossier` with first-class `delivery_proofs`, `chargebacks`, and `reversals` while driving `proof_status.delivery_proof_available` from persisted artifacts instead of a known gap. Shared `packages/types` plus retailer desktop/iOS/Android consumers now render proof and dispute-history sections from the same tracking seam. Focused validation passed via order and retailer backend tests, retailer desktop `CI=1 npm run build`, and retailer iOS `xcodegen generate && xcodebuild -project RetailerAppIOS.xcodeproj -target RetailerAppIOS -sdk iphonesimulator26.5 CODE_SIGNING_ALLOWED=NO build`; Android diagnostics are clean, while full CLI assemble remains environment-blocked because pegasusX has no Gradle wrapper and no system `gradle` on `PATH`.

### PX-6 anchors
1. `PX6-A1` Event-driven AI worker substrate
  Status: `implemented`
  Non-technical: AI operates as operator assistance, not opaque automation.
  Technical: Kafka-driven worker jobs, replay-safe advisory writes, status visibility, and bounded retry semantics.
  Dependencies: `PX0-A3`, `PX0-A4`.
  Exit evidence: AI work stays off the synchronous request path.
2. `PX6-A2` Forecasting and recommendation products
  Status: `implemented`
  Non-technical: supplier teams can see useful recommendations and retailer AI remains opt-in.
  Technical: forecast, replenishment, dispatch, and recommendation outputs plus UI surfaces to review them.
  Dependencies: `PX6-A1`, `PX2-A4`, `PX4-A1`.
  Exit evidence: recommendations are visible, bounded, and reviewable.
  Closure delta (2026-05-25): Supplier-facing recommendation review is now implemented over the existing `AIPredictions` advisory ledger. `apps/ai-worker/main.go` writes explainable `AI_RECOMMENDATION_CREATED` outputs with action, confidence, source, reason codes, evidence, expiry, and transactional outbox emission; `apps/backend-go/schema/spanner.ddl` adds supplier/status read indexes for bounded review; `apps/backend-go/supplier/{ai_recommendations.go,repository_spanner_ai.go}` plus `supplierroutes` expose `GET|POST /v1/supplier/ai/recommendations`; shared `packages/{types,api-client}` and `contracts/events.schema.json` carry the new contracts; and `apps/supplier-portal/app/ai/recommendations/page.tsx` renders loading, empty, restricted, offline, stale, error, and ready states with operator actions.
  Delta 2026-05-31: retailer AI suggestions now materialize as durable future-dated orders instead of supplier-only advisory rows. `apps/ai-worker/main.go` now parses flat `ORDER_CREATED` events, skips non-manual sources, creates one replay-safe derived `AI_PREORDER` order per manual seed order through `Orders.DerivedFromOrderId`, emits additive `ORDER_CREATED` preorder events through the transactional outbox, and runs a minute-ticker auto-confirm sweep through `order.Service.AutoConfirmDueOrders` so pending AI suggestions eventually transition to `AUTO_CONFIRMED` with the same order-event notification path.
3. `PX6-A3` Human override and explainability
  Status: `implemented`
  Non-technical: operators understand why the system suggests something and can override it.
  Technical: explanation fields, audit trail, manual override capture, and safe rollback of automated suggestions.
  Dependencies: `PX6-A2`.
  Exit evidence: AI assistance does not create black-box operational risk.
  Closure delta (2026-05-25): Human authority is now recorded without letting AI mutate operational state directly. Supplier decisions (`ACKNOWLEDGED`, `OVERRIDDEN`, `DISMISSED`, `REOPENED`) are idempotency-guarded, supplier-scoped, persisted into `AIPredictions.PredictionData` with `decision_history`, status transitions, actor, note, and timestamp, and emit `AI_RECOMMENDATION_DECIDED` through the transactional outbox. Reopened recommendations return to `PENDING`, while override/dismiss/acknowledge remain explicit audit states for support review.

### PX-7 anchors
1. `PX7-A1` SSMR and production-parity proof gate
  Status: `implemented`
  Non-technical: stakeholders can trust local proof as a precursor to cloud deployment.
  Technical: isolated Spanner/Redis/Kafka setup, bootstrap proofs, health checks, spatial checks, and Kafka round-trip validation.
  Dependencies: `PX0-A4`.
  Exit evidence: infra-sensitive changes are blocked from merging without sandbox proof.
2. `PX7-A2` Load, resilience, and security certification
  Status: `implemented`
  Non-technical: launch readiness is judged by evidence, not optimism.
  Technical: load tests, replay drills, Redis/Kafka/Spanner failure-mode tests, auth hardening, and webhook/security review.
  Dependencies: `PX3-A2`, `PX5-A2`, `PX7-A1`.
  Exit evidence: the platform is certified for burst traffic, failures, and attack-surface review.
  Current delta (2026-05-23): realtime hardening landed additively in backend runtime: `apps/backend-go/ws/connection.go` no longer accepts arbitrary browser origins and now enforces `WS_ALLOWED_ORIGINS` with empty-origin native-client support plus localhost development exceptions; `apps/backend-go/bootstrap/bootstrap.go` now provisions a dedicated notification-consumer DLQ writer from `KAFKA_TOPIC_MAIN_DLQ` (default `<main-topic>-dlq`) and either fails strict-mode startup or disables the consumer with warning when that path cannot initialize; `apps/backend-go/kafka/{consumer.go,dlq_writer.go}` now add jittered retry backoff, annotate DLQ records with failure/original topic metadata, and refuse offset commits when DLQ routing fails; and `apps/backend-go/payment/service.go` now verifies raw Adyen signed notification items before business validation or persistence. Focused validation passed via `go test ./ws ./kafka ./bootstrap ./payment` and targeted websocket/bootstrap/payment unit suites.
3. `PX7-A3` Launch readiness, support, and hypercare
  Status: `implemented`
  Non-technical: release owners, support, and hypercare know exactly what to do at launch.
  Technical: final runbooks, dashboards, release gates, rollback rehearsal, and post-launch monitoring.
  Dependencies: `PX0-A5`, `PX7-A2`.
  Exit evidence: launch approval is operationally supportable.
  Current delta (2026-05-23): launch-readiness work is now active on the ai-worker operational seam: `apps/ai-worker/main.go` now serves `/healthz`, `/ready`, and `/metrics`, exposes `void_ai_worker_up`, `void_ai_worker_ready`, and `void_kafka_consumer_lag_seconds`, derives its monitoring port from `AI_WORKER_HTTP_PORT` with `HEALTH_PORT` fallback (default `8081`), and marks readiness false during shutdown so probes can drain cleanly. `infra/docker-compose.ssmr.yml` now exposes the worker monitoring port on host `8181->8081` for local proof and support checks; `infra/k8s/ai-worker/{configmap,deployment,service}.yaml` now provide deployment-grade packaging for that worker with readiness/liveness probes and a ClusterIP monitoring service; `scripts/validate_ai_worker_k8s.sh` plus `make validate-ai-worker-k8s` / `pnpm infra:k8s:validate` now provide a repo-local release gate for those manifests; `infra/terraform/observability.tf` now adds the first ai-worker dashboard, alert, and uptime automation; and `docs/AI_WORKER_LAUNCH_RUNBOOK.md` plus `docs/README.md` now provide the concrete PX7-A3 launch/hypercare playbook for that seam. Focused validation passed via `go test ./apps/ai-worker`, `terraform fmt`, YAML parse, and diff hygiene checks.
  Closure delta (2026-05-25): PX7-A3 launch approval is now operationally supportable for the currently implemented engine through `docs/LAUNCH_READINESS_RUNBOOK.md` and the aggregate `scripts/validate_launch_readiness.py` gate. The guard requires SSMR proof (`make test-ssmr-infra`), ai-worker manifest proof (`make validate-ai-worker-k8s`), launch evidence proof (`make validate-launch-readiness`), Terraform observability evidence, and indexed support artifacts before release approval. PX6-A2/PX6-A3 are tracked separately above and now close the AI recommendation/explainability product gap through supplier review and human-decision authority rather than being implied by launch readiness alone.

## Cross-Phase Workstreams

### Product and operations
1. Role definitions, SOPs, escalation rules, approval policies, and service-level expectations.
2. Onboarding, training, support scripts, hypercare, and operator-facing recovery paths.
3. Exception management, dispute handling, financial controls, and audit-ready business processes.

### Backend and data
1. Domain services, Spanner schema, repositories, transactional outbox, idempotency, websocket fanout, and worker orchestration.
2. Event contracts, API DTOs, versioning, auth scope enforcement, and trace propagation.
3. Payment, ledger, reconciliation, and operational analytics integrity.

### Frontend, desktop, and mobile
1. Supplier portal, retailer desktop/mobile, driver mobile, warehouse portal/mobile, factory portal/mobile, and payload terminal/tablet parity.
2. Loading, empty, offline, stale, restricted, and failure states on every live surface.
3. Realtime consumption, reconnect behavior, local persistence, and role-appropriate task flows.

### Platform and reliability
1. Kafka partitioning, Redis invalidation discipline, Spanner index coverage, Kubernetes readiness, and rollout safety.
2. Load shedding, rate limiting, circuit protection, observability, and chaos/testing gates.
3. CI gates, SSMR smoke proof, rollback discipline, and incident response evidence.

## First Delivery Batches
1. `B01` PX-0 execution hardening
  Status: `implemented`
  Anchors: `PX0-A2`, `PX0-A3`, `PX0-A4`.
  Non-technical deliverables: role-row ownership matrix, support vocabulary baseline, launch KPI draft, release and escalation expectations.
  Technical deliverables: capability ledger, route and event inventory freeze, shared-contract ownership map, reliability acceptance matrix, gap list for missing guards or parity.
  Primary surfaces: `context/plan.md`, `context/architecture.md`, `context/technology-inventory.*`, `.github/*.md`.
  Validation: docs sync complete, JSON docs valid, anchor statuses updated, no ambiguity on role rows and route authority.
2. `B02` Supplier bootstrap durability and topology batch
  Status: `implemented`
  Anchors: `PX1-A1`, `PX1-A2`.
  Non-technical deliverables: supplier onboarding SOP, billing recovery script, topology-entry support guide.
  Technical deliverables: durable signup/login/profile/billing flows, topology persistence, downstream warehouse/factory visibility, contract-safe responses.
  Primary surfaces: `apps/backend-go/supplier/**`, `apps/supplier-portal/**`, `packages/types`, `packages/api-client`.
  Validation: targeted backend tests, portal type/build checks, SSMR proof if infra-sensitive changes are introduced.
  Current delta (2026-05-22): supplier Spanner repository now persists rich profile + topology (`SupplierProfiles`, warehouse/factory read/write), supplierroutes exposes `GET|PUT /v1/supplier/topology`, supplier portal register/billing payloads now align with backend contract and idempotency header, supplier portal now has a same-origin `/api/*` proxy transport to backend, shared supplier contract/client bridges are now exported in `packages/types` + `packages/api-client`, B02 non-technical artifacts now exist in `docs/SUPPLIER_ONBOARDING_SOP.md`, `docs/BILLING_RECOVERY_SCRIPT.md`, and `docs/TOPOLOGY_ENTRY_SUPPORT_GUIDE.md`, and portal validation gate is now green (`pnpm --filter @pegasusx/supplier-portal build` + `typecheck`) after workspace tooling fix via `pnpm-workspace.yaml`.
3. `B03` Retailer commerce and pricing batch
  Status: `implemented`
  Anchors: `PX2-A1`, `PX2-A2`, `PX2-A3`.
  Non-technical deliverables: retailer onboarding/support flows, pricing authority rules, zone-miss communication policy.
  Technical deliverables: retailer registration/profile linkage, cart sync, pricing integrity, serviceability, order capture, warehouse assignment, duplicate-tap safety.
  Primary surfaces: `apps/backend-go/retailer/**`, `apps/backend-go/order/**`, `apps/backend-go/supplier/**pricing*`, retailer apps, shared contracts.
  Validation: retailer/backend tests, contract diff check, client build/type checks, spatial/SSMR proof when order-zone logic changes.
  Current delta (2026-05-22): B03 technical tranche sequence is now additive in `apps/backend-go`: `retailer/core_handlers.go` enforces claims-authoritative retailer identity with explicit `401`/`403` scope handling and profile `supplier_id` backfill (`PX2-A1`), `order/{service.go,warehouse_resolver_spanner.go}` now fail-closes serviceability with `delivery_perimeter_unavailable` and `zone_miss` semantics (`PX2-A2`), supplier pricing authority remains on `GET|PATCH /v1/supplier/pricing/rules`, and retailer-facing pricing display parity now lands through additive `pricing` snapshots on `GET /v1/retailer/suppliers` plus dedicated `GET /v1/retailer/pricing/rules` via `retailer/{service.go,core_handlers.go,repository_spanner.go}` + `retailerroutes/routes.go` with shared DTO/client bridge updates in `packages/types/index.ts` and `packages/api-client/index.ts` (`PX2-A3`).
4. `B04` Payment integrity batch
  Status: `implemented`
  Anchors: `PX3-A1`, `PX3-A2`.
  Non-technical deliverables: payment exception SOP, finance support workflow, dispute classification vocabulary.
  Technical deliverables: checkout consistency, webhook replay safety, attempt metadata continuity, settlement and ledger authority groundwork.
  Primary surfaces: `apps/backend-go/payment/**`, payment-facing retailer and supplier surfaces, contract docs.
  Validation: backend payment tests, webhook path validation, idempotency coverage, SSMR proof for infra-sensitive payment changes.
  Current delta (2026-05-22): payment webhook replay hardening is now additive in `apps/backend-go/payment/service.go`; Adyen per-item replay handling now uses non-writing idempotency checks to prevent concatenated multi-write responses on duplicate notifications while preserving signature-first verification and outbox-backed persistence. Focused coverage now lives in `apps/backend-go/payment/service_webhook_handlers_test.go` (Global Pay replay dedupe, Stripe replay conflict `409`, Adyen replay single-response semantics, Adyen signature rejection `401`). PX3-A2 settlement/reconciliation authority is now implemented additively: `apps/backend-go/payment/repository_spanner.go` now serves grouped immutable-ledger summaries (`Gateway+EntryType+Currency`) with bounded reconciliation filters, `apps/backend-go/payment/service.go` + `apps/backend-go/paymentroutes/routes.go` now expose supplier-scoped `GET /v1/payment/settlement/authority` and supplier-scoped `GET /v1/payment/reconciliation/mismatches` (bounded `gateway`/time-window/`group_limit`/`mismatch_threshold_minor`), focused mismatch coverage now lives in `apps/backend-go/payment/service_reconciliation_mismatches_test.go`, in-memory repository parity lives in `apps/backend-go/bootstrap/bootstrap.go`, and shared supplier-finance consumption now lands in `packages/{types,api-client}/index.ts` plus `apps/supplier-portal/app/payments/page.tsx` with additive `/v1/payment/ledger` fallback. B04 non-technical closure artifacts now exist in `docs/PAYMENT_EXCEPTION_SOP.md`, `docs/FINANCE_SUPPORT_WORKFLOW.md`, and `docs/DISPUTE_CLASSIFICATION_VOCABULARY.md`, and `docs/README.md` now indexes these support references.
5. `B05` Node operations durability batch
  Status: `implemented`
  Anchors: `PX4-A1`, `PX4-A2`, `PX4-A3`.
  Non-technical deliverables: warehouse/factory/payload SOPs for exceptions, reassignment, and transfer cancellation.
  Technical deliverables: durable warehouse/factory/payload repositories, manifest lifecycle integrity, reassignment correctness, realtime parity.
  Primary surfaces: `apps/backend-go/{warehouse,factory,payload}/**`, node apps, payload terminal, shared contracts.
  Validation: focused backend tests, event/contract validation, websocket parity checks, SSMR proof when infra or realtime wiring changes.
  Current delta (2026-05-22): B05 non-technical kickoff artifacts are now additive in `docs/`: `docs/WAREHOUSE_EXCEPTION_SOP.md` (warehouse exception triage and dispatch-lock handling), `docs/REASSIGNMENT_SUPPORT_PLAYBOOK.md` (payload/factory reassignment and rebalance handling), and `docs/TRANSFER_CANCELLATION_RUNBOOK.md` (factory transfer cancellation decision and escalation flow). `docs/README.md` now indexes these B05 support references. PX4-A1 technical execution is now additive in `apps/backend-go/warehouse`: `HandleDispatchLock` now returns deterministic `404 dispatch_lock_not_found` when releasing unknown lock IDs (preventing false release success fanout), and focused seam coverage now exists in `apps/backend-go/warehouse/service_test.go` validating supply-request and dispatch-lock mutation paths for repository apply + outbox event emission, post-commit cache invalidation, and websocket fanout parity. PX4-A2 hardening now adds idempotent transfer cancellation in `apps/backend-go/factory/service.go`: repeated `POST /v1/factory/manifests/cancel-transfer` for an already-cancelled transfer now returns `status=already_cancelled` without duplicate exception-count/volume mutations or additional outbox/cache/ws side effects, validated by `apps/backend-go/factory/service_test.go`. PX4-A3 hardening now adds capacity-integrity guardrails in `apps/backend-go/payload/service.go`: `POST /v1/payloader/reassign-order` now validates target-manifest capacity using reassigned order volume and returns `409 target_manifest_capacity_exceeded` when overflow would occur, preventing order-manifest drift; focused regression coverage now exists in `apps/backend-go/payload/service_test.go`. Follow-on PX4-A2/PX4-A3 hardening is now additive: `POST /v1/factory/manifests/rebalance` returns idempotent `status=already_assigned` when requested transfer target matches current assignment, rejects non-mutable transfer states with `409 transfer_not_mutable` (preventing no-op/non-mutable reassignment depth and extra outbox/cache/ws side effects), and `POST /v1/payloader/reassign-order` now returns deterministic no-op (`status=already_assigned`) for same-target replay, explicit same-route requests, and mutable-source auto-resolution or conflict (`409 reassign_target_unavailable`, `409 target_route_mismatch`) semantics instead of fallback-route drift when no mutable/capacity-valid target manifest exists; focused regression coverage now also includes these behaviors in `apps/backend-go/{factory,payload}/service_test.go`. Latest PX4-A2/PX4-A3 consistency extension is now additive: `POST /v1/factory/manifests/rebalance` now fails with `404 transfer_not_found` when transfer rows drift between manifest-local and global transfer ledgers (preventing split-state mutations), and `POST /v1/payloader/reassign-order` now permits explicit same-route reassignment to alternate mutable manifests when source manifests are non-mutable while preserving mutable-source same-route no-op behavior; focused regression coverage now includes `TestHandleManifestRebalance_GlobalTransferMissingConflict` and `TestHandleApplyReassign_ExplicitSameRouteSelectsAlternateManifestWhenSourceNotMutable`. Driver-manifest-route consistency and replay-after-success idempotency extension is now additive: `POST /v1/factory/manifests/rebalance` now rejects inconsistent transfer linkage (`409 transfer_manifest_mismatch`) and route-vehicle divergence (`409 transfer_route_mismatch`) before mutation, and replaying a successful rebalance now returns deterministic `status=already_assigned` with no extra outbox/cache/ws side effects; `POST /v1/payloader/reassign-order` now rejects target-driver/manifest divergence (`409 target_driver_manifest_mismatch`) and replaying a successful reassignment now returns deterministic `status=already_assigned` with no extra side effects. Cross-entity ownership consistency extension is now additive: factory rebalance now rejects local/global transfer parity drift with `409 transfer_ledger_mismatch`, payload reassignment now rejects source ownership drift with `409 source_manifest_not_found`, `409 source_route_manifest_mismatch`, or `409 source_manifest_order_missing`, and replay-after-success tests now assert single emitted outbox payload consistency for both rebalance and reassignment. B05 negative-path event-contract assertion pass is now additive (test-only): warehouse missing-lock release now validates conflict-after-success with unchanged outbox/cache/ws fanout envelope sequences, factory negative branches (`transfer_not_mutable`, replay `already_assigned`, `transfer_route_mismatch` after success, `already_cancelled`) now assert unchanged outbox/ws contract sequences, and payload no-op/conflict branches (replay `already_assigned`, `source_manifest_order_missing` after success, `target_route_mismatch` after success) now assert unchanged outbox/ws envelope sequences with websocket payload field checks. Focused regression coverage now includes `TestHandleManifestRebalance_TransferRouteMismatch`, `TestHandleManifestRebalance_ReplayAfterSuccessIdempotent`, `TestHandleManifestRebalance_TransferRouteMismatchAfterSuccess_NoExtraFanout`, `TestHandleApplyReassign_TargetDriverManifestMismatch`, and `TestHandleApplyReassign_ReplayAfterSuccessIdempotent`.
  Completion gate (2026-05-22): executed `go test ./warehouse ./factory ./payload ./warehouseroutes ./factoryroutes ./payloaderroutes` and `go build -buildvcs=false ./...` in `apps/backend-go`; all checks passed. Executed `make test-ssmr-infra` in `pegasusX` and the smoke gate completed successfully (Spanner probe, Redis ping, backend health, spatial check, and Kafka round-trip check passed), with terminal markers `__SSMR_OK__` and `B05_E2E_OK` emitted.
6. `B06` Driver and live-delivery batch
  Status: `implemented`
  Anchors: `PX5-A1`, `PX5-A2`, `PX5-A3`.
  Non-technical deliverables: driver support playbooks, live-tracking expectations, delivery escalation policy.
  Technical deliverables: driver execution parity, telemetry integrity, live tracking, reconnect safety, manifest-to-delivery completion chain.
  Primary surfaces: `apps/backend-go/{driver,order,orderroutes}/**`, telemetry paths, driver apps, retailer tracking surfaces.
  Validation: targeted backend tests, mobile build checks where available, websocket/telemetry validation, load-oriented proof for live updates.
  Current delta (2026-06-11): B06 non-technical kickoff artifacts are additive in `docs/`: `docs/DRIVER_SUPPORT_PLAYBOOK.md`, `docs/LIVE_TRACKING_EXPECTATIONS.md`, `docs/DELIVERY_ESCALATION_POLICY.md`, and `docs/MIGRATION_RUNBOOK_MANIFEST_ROUTE_GEOMETRY.md`; `docs/README.md` indexes these B06 support references. PX5-A1 driver execution baseline is now implemented through availability, manifest detail, manifest gate, pending collections, earnings, and delivery-state durability tranches. PX5-A2 telemetry/live-tracking integrity is now implemented with authenticated telemetry fanout, durable order assignment/tracking projection, scoped cache-backed retailer live coordinates, supplier-portal durable order oversight plus fleet live map (`GET /v1/supplier/fleet/live-map`), warehouse-portal fleet live map (`GET /v1/warehouse/ops/fleet/live-map`), retailer desktop plus retailer iOS plus retailer Android tracking consumers, driver Android plus driver iOS live-state consumers with planned route geometry (`GET /v1/fleet/route/{routeID}/geometry`), turn-by-turn, off-route reroute, and breadcrumb traveled-path overlays. PX5-A3 receipt/proof/collection integrity is now implemented with retailer fulfillment/pending-payment visibility, a derived retailer tracking-event timeline, additive recent completed receipt snapshots from durable order rows, immutable `OrderDeliveryProofs`, and an additive receipt dossier carrying payment timelines, gateway webhooks, delivery proofs, chargebacks, and reversals from system truth.

## Phase Roadmap

### PX-0: Governance, Contracts, and Delivery Control

#### Objective
Create the execution control plane for pegasusX so roadmap, architecture, contracts, and support expectations stay synchronized while the product grows.

#### Business and operational outcomes
1. One clear roadmap exists for product, engineering, and operations.
2. Stakeholders can see what each phase is meant to achieve and what evidence closes it.
3. Support, rollout, and escalation ownership are defined before deeper workflow automation lands.

#### Engineering scope

Backend and data:
1. Lock canonical route families, event names, entity IDs, and role claims.
2. Establish the baseline mutation contract: auth, scope, validation, RW transaction, outbox emit, cache invalidation, trace logging, additive DTO.
3. Keep SSMR as the proof gate for Spanner, Redis, Kafka, backend health, and spatial readiness.

Frontend, desktop, and mobile:
1. Freeze the role-row surface matrix and the platform choices per role.
2. Define shared contract consumption rules for TS, Kotlin, and Swift clients.
3. Set cross-client parity expectations before adding deeper features.

Platform and operations:
1. Define target load envelope: 1M-request-class daily volume, thousands of concurrent devices/operators, burst-safe realtime and webhook handling.
2. Lock reliability requirements for Kafka delivery, Redis invalidation, Spanner indexing, and Kubernetes statelessness.
3. Establish observability, support, release, and rollback expectations.

#### Exit gate
1. Canonical roadmap approved and referenced by local instruction files.
2. Sync protocol defined for architecture, inventory, parity, and plan files.
3. SSMR remains the mandatory local proof step for infra-sensitive work.

### PX-1: Supplier Company Bootstrap and Control Tower Foundation

#### Objective
Turn the supplier row into a complete operating business surface: identity, network topology, billing, and core control-tower visibility.

#### Business and operational outcomes
1. The supplier can sign up, log in, configure the company, and finish billing without manual engineering help.
2. Factories, warehouses, staffing model, and commercial setup are captured as real operating topology.
3. Internal teams have a clear control tower for finance, inventory, and order oversight.

#### Engineering scope

Backend and data:
1. Supplier auth, profile, billing setup, payment configuration, and topology creation must persist durably in Spanner.
2. Supplier-owned entities keep `SupplierId` everywhere for future compatibility.
3. Supplier configuration changes must emit outbox events where downstream state depends on them and invalidate affected caches post-commit.

Frontend and desktop:
1. Supplier onboarding wizard covers account, topology, business, categories, and billing gate.
2. Supplier portal dashboard establishes inventory, earnings, order review, and configuration entry points.
3. Error and recovery UX must exist for incomplete setup, invalid billing, and topology mistakes.

Mobile and node apps:
1. Warehouse and factory login readiness must respect the supplier-created topology.
2. Mobile contract shapes for supplier-owned nodes must be aligned before deeper node workflows ship.

Platform and operations:
1. Supplier setup must be idempotent and replay-safe.
2. Admin-support runbooks must cover failed onboarding, billing misconfiguration, and incorrect topology data.

#### Exit gate
1. A supplier can bootstrap the company and network end-to-end.
2. Warehouse and factory nodes created during onboarding are visible to downstream role surfaces.
3. Support can recover a broken onboarding without direct database surgery.

### PX-2: Retailer Commerce, Catalog, and Order Capture

#### Objective
Create a reliable retailer commerce layer that supports discovery, cart continuity, order creation, and order-side decision flows.

#### Business and operational outcomes
1. Retailers can register, manage profile, choose the seeded supplier, build carts, and place orders across mobile and desktop surfaces.
2. Retailers get clear outcomes for valid orders, service-zone misses, pending payments, AI suggestions, cancellations, and preorders.
3. Retailer support can explain why an order was accepted, blocked, changed, or cancelled.

#### Engineering scope

Backend and data:
1. Catalog, retailer pricing, cart sync, profile, family-members, order create, cancel/request-cancel, preorder, AI confirm/reject, and tracking paths must be coherent.
2. Order creation must derive serviceability from backend-controlled spatial logic and assign the right warehouse candidate.
3. Retailer commerce data must remain additive and cross-client safe.

Frontend and desktop:
1. Retailer apps expose supplier selection, cart sync, checkout initiation, order history, active fulfillment, and payment state.
2. Desktop and mobile retain feature parity while using platform-appropriate UI controls.
3. Loading, empty cart, zone miss, stale cart, and offline resume states are mandatory.

Mobile and realtime:
1. Retailer surfaces must stay consistent across Android, iOS, and desktop.
2. Websocket updates for order status, payment state, and cart sync must drive visible UI refresh.
3. Reconnect behavior must not duplicate cart or payment actions.

Platform and operations:
1. Order creation and cart sync must be safe under burst traffic and duplicate taps.
2. Support scripts must cover zone-miss, cart drift, stuck pending-payment, and cancel-policy questions.

#### Exit gate
1. Retailers can move from registration to successful order placement on every supported retailer surface.
2. Zone and warehouse-selection logic are backend authoritative and observable.
3. Duplicate requests do not create duplicate business outcomes.

### PX-3: Payment, Ledger, Settlement, and Reconciliation Integrity

#### Objective
Make money movement safe, explainable, and operationally manageable from checkout through webhook settlement and disputes.

#### Business and operational outcomes
1. Retailers can pay with supported gateways and understand whether the order is awaiting payment, cleared, failed, or disputed.
2. Suppliers can monitor settlement, payment exceptions, and chargeback/reversal outcomes.
3. Finance and support teams can reconcile payment state without manual guesswork.

#### Engineering scope

Backend and data:
1. Payment sessions, attempts, chargebacks, reversals, and webhooks persist durably with idempotency and outbox discipline.
2. Gateway execution uses bounded retries, typed policy errors, and provider-exact webhook verification.
3. Ledger and settlement data must support future double-entry and treasury reporting with no float-based money math.

Frontend and desktop:
1. Retailer surfaces show pending payment, cleared payment, retry, and failure recovery states.
2. Supplier finance surfaces expose earnings, payment status, and dispute workflow visibility.
3. Payment UX must clearly separate redirect, hosted, direct, and retry outcomes.

Mobile and realtime:
1. Payment-required and payment-cleared updates must propagate to retailer and supplier surfaces in near real time.
2. Older clients must tolerate additive payment metadata without breaking.

Platform and operations:
1. Checkout and webhook paths must stay safe under replay and concurrent retries.
2. Support and finance runbooks must cover manual review, settlement mismatch, chargeback handling, and refund/reversal communication.

#### Exit gate
1. Checkout to settlement produces one durable business outcome per payment intent.
2. Webhook replay and duplicate client calls remain safe.
3. Finance can explain payment state transitions from stored records and emitted events.

### PX-4: Warehouse, Factory, and Payload Operational Execution

#### Objective
Connect upstream production and local fulfillment into a controlled manifest-driven execution model.

#### Business and operational outcomes
1. Warehouses can manage local inventory, supply requests, dispatch preview, and dispatch locks.
2. Factories can prepare transfers, manage manifests, load, seal, dispatch, complete, rebalance, or cancel movement.
3. Payload staff can validate truck readiness, load manifests, inject late orders before seal, raise exceptions, and reassign when physical reality changes.

#### Engineering scope

Backend and data:
1. Warehouse, factory, and payload repositories must move from scaffold-only behavior toward durable transactional state.
2. Manifest lifecycle, exception, reassignment, and dispatch operations must emit additive events and invalidate source/target caches correctly.
3. Supply-request and dispatch-preview logic must stay node-scoped and claims-derived.

Frontend and desktop:
1. Warehouse and factory portals expose operational dashboards, queues, previews, and exception views.
2. Payload terminal and tablet experiences focus on fast load-time execution and exception repair.
3. Every operational surface needs explicit empty, stale, locked, and failure states.

Mobile and realtime:
1. Warehouse/factory mobile apps and payload devices consume manifest lifecycle and exception events in real time.
2. Reassignment and manifest mutation envelopes must stay contract-safe across all clients.

Platform and operations:
1. Manifest operations must remain safe under concurrent operator activity.
2. SOPs must exist for overflow, damaged goods, late order injection, seal rejection, and transfer cancellation.

#### Exit gate
1. Physical transfer and loading workflows can be executed without spreadsheet or chat-based shadow systems.
2. Manifest state changes are durable, visible, and replay-safe.
3. Payload and factory/warehouse teams share one operational truth.

### PX-5: Driver Delivery, Telemetry, and Retailer Receipt

#### Objective
Close the loop from manifest dispatch to live delivery, proof, collection, and retailer-facing receipt.

#### Business and operational outcomes
1. Drivers can go available, receive manifests, execute stops, collect cash when required, and complete deliveries with clear rules.
2. Retailers can track live fulfillment, see arrival/progress, and handle receipt-side actions.
3. Operations teams can understand where drivers are, what route they are executing, and which orders are blocked or disputed.

#### Engineering scope

Backend and data:
1. Driver profile, history, earnings, availability, pending collections, manifest gate, manifest detail, and delivery-state transitions must be authoritative.
2. Telemetry ingestion must enforce authenticated driver identity and fan out safely to interested rooms.
3. Geofence-sensitive actions and completion logic must protect financial and operational integrity.

Frontend and mobile:
1. Driver apps prioritize low-latency execution, manifest clarity, and recovery from poor connectivity.
2. Retailer tracking and active-fulfillment views must reflect live delivery progress and payment/settlement state.
3. Warehouse/factory/supplier surfaces must expose driver state and delivery exceptions without requiring database inspection.

Realtime and platform:
1. Telemetry, order status, payment state, and manifest updates must reconnect cleanly after network drops.
2. Thousands of active devices must not overwhelm websocket or telemetry pipelines.
3. Traceability from request to event to websocket delivery remains mandatory.

#### Exit gate
1. A delivery can move end-to-end from manifest dispatch to completed retailer receipt.
2. Driver connectivity loss or reconnect does not create silent data corruption.
3. Support can explain live-delivery state from platform records.

### PX-6: Intelligence, Forecasting, and Planning Automation

#### Objective
Introduce AI and planning assistance that improves operations without taking control away from human operators.

#### Business and operational outcomes
1. Supplier teams receive useful forecasting, replenishment, dispatch, and exception-prioritization help.
2. Retailers can benefit from AI-suggested orders or reorder support without forced automation.
3. Human operators retain override authority where policy allows.

#### Engineering scope

Backend and data:
1. AI worker consumes Kafka events and produces replay-safe advisory outputs rather than synchronous request-path side effects.
2. Forecasting, recommendation, and optimization jobs run asynchronously with bounded retries, status visibility, and audit trail.
3. Planning outputs must be explainable enough for operators to trust or reject them.

Frontend and clients:
1. Supplier control surfaces expose forecast, dispatch guidance, exception ranking, and action history.
2. Retailer AI order suggestion flows remain explicit opt-in/confirm-reject interactions.
3. Node apps only surface actionable recommendations that fit the role.

Platform and operations:
1. AI/optimization flows must never block the checkout or order-mutation critical path.
2. Model and recommendation quality must be measurable, reversible, and supportable.

#### Exit gate
1. AI assistance reduces operator effort without creating opaque failure modes.
2. Async worker flows are observable, replay-safe, and operationally bounded.
3. Manual override remains available for high-consequence actions.

### PX-7: Scale, Security, and Launch Readiness

#### Objective
Certify pegasusX for production launch with scale, resilience, support, and governance readiness.

#### Business and operational outcomes
1. The system is ready for live supplier and retailer traffic with clear support ownership.
2. On-call, incident, rollback, and hypercare processes are ready before launch.
3. Leadership has clear go/no-go evidence based on quality and resilience, not optimism.

#### Engineering scope

Platform and reliability:
1. Run load tests for 1M-request-class daily traffic, burst concurrency, realtime fanout, webhook replay, and queue backlog behavior.
2. Validate Kafka partition strategy, Spanner hotspot posture, Redis failure modes, and Kubernetes drain/rollout behavior.
3. Confirm autoscaling, load shedding, circuit protection, and observability thresholds.

Security and compliance:
1. Finalize auth hardening, secret rotation, audit trail completeness, least-privilege verification, and webhook security posture.
2. Confirm mobile identity and session controls remain safe under rotation and replay.

Product and support:
1. Complete release checklists, training, documentation, support scripts, and stakeholder communication.
2. Prepare hypercare dashboards and escalation routing for launch week.

#### Exit gate
1. Performance, security, and rollback gates pass with evidence.
2. Support and hypercare are staffed and documented.
3. Launch approval is based on measured readiness, not feature count.

## Phase Synchronization Protocol
1. Before each meaningful execution batch, read this file and map the work to one or more plan anchors.
2. Prefer the active or earliest queued delivery batch unless the user explicitly reprioritizes.
3. After each batch, update statuses in this file and sync the context/docs set in the same change set when architecture or delivery truth changed.
4. When behavior diverges from the Pegasus reference (`context/PEGASUS_REFERENCE.md`), update `context/parity-ledger.md` in the same batch.
5. When the roadmap changes, update local instruction files so future agents follow the revised execution model.

## Documentation Sync Set
1. `.github/ACT.md`
2. `.github/copilot-instructions.md`
3. `.github/gemini-instructions.md`
4. `context/plan.md`
5. `context/architecture.md`
6. `context/architecture-graph.json`
7. `context/technology-inventory.md`
8. `context/technology-inventory.json`
9. `context/parity-ledger.md` when divergence from Pegasus changes

## Execution Checkpoint Template
- Checkpoint timestamp:
- Scope:
- Plan anchors touched:
- Files changed:
- Contracts or events impacted:
- Validation run:
- Status by anchor:
  - implemented:
  - in progress:
  - blocked:
  - deferred:
- Follow-up sync required: