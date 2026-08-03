# pegasusX execution plan

<<<<<<< HEAD
> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


Last updated: 2026-07-05 (PX-13 ecosystem audit remediation shipped; PX-14 four mature flow hardening in progress; all gates green)

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

## PX-13 Ecosystem audit remediation (2026-07-05)

Full-stack gap-hunter + void-guardian audit of the pegasusX ecosystem. Closed P0 security/correctness gaps and P1 contract-drift gaps identified by parallel subagent scans. All fixes validated against `make backend-build parity-contract gap-hunter-gate gen-contracts-gate parity-contract-full validate-launch-readiness`.

| Anchor | Scope | Status | Notes |
|---|---|---|---|
| `PX13-A1` | Backend route auth hardening | `implemented` | `driverroutes`, `factoryroutes`, `warehouseroutes` ecosystem CRUD (`/v1/drivers`, `/v1/vehicles`, `/v1/factories`, `/v1/warehouses`) moved inside `auth.ProtectMutations` groups scoped to admin roles (`ADMIN`, `WAREHOUSE_ADMIN`, `FACTORY_ADMIN`). |
| `PX13-A2` | Body scope override rejection | `implemented` | Warehouse/factory/driver/vehicle CRUD handlers now resolve `supplier_id` from `auth.ResolveSupplierID`, reject body-provided scope overrides via `auth.RejectBodyScopeOverrides`, and ignore query `supplier_id` mismatches. |
| `PX13-A3` | CRUD transactional outbox + cache invalidation | `implemented` | `warehouse/repository_crud.go`, `factory/repository_crud.go`, `driver/repository_crud.go` Create/Update converted from `spanner.Apply` to `ReadWriteTransaction` with `outbox.EmitJSON` (`WAREHOUSE_CREATED`, `WAREHOUSE_LOCATION_UPDATED`, `FACTORY_CREATED`, `FACTORY_LOCATION_UPDATED`, `DRIVER_CREATED`, `DRIVER_AVAILABILITY_CHANGED`, `VEHICLE_CREATED`, `VEHICLE_AVAILABILITY_CHANGED`) and post-commit `cache.Invalidate`. Repository interfaces, in-memory stubs, and test spies updated. |
| `PX13-B1` | Shared EventType union completeness | `implemented` | `packages/types/index.ts` `EventType` union expanded to mirror `apps/backend-go/events/events.go`, adding pre-order, route-reordered, shop-closed/negotiation resolution, return, planning signal/forecast/promo/confidence events, and missing supplier/warehouse/factory/driver/vehicle lifecycle events. |
| `PX13-B2` | WebSocket refresh contract correctness | `implemented` | `packages/ws-refresh-contract/index.ts` `DISPATCH_REFRESH_EVENTS` corrected from non-existent `MANIFEST_CREATED` to `MANIFEST_DRAFT_CREATED` and augmented with full manifest lifecycle events. `supplier-portal/lib/supplier-ws-events.ts` dispatch refresh set aligned. |
| `PX13-C1` | K8s secret key alignment | `implemented` | `infra/k8s/optimizer-core/deployment.yaml` secret key fixed from `INTERNAL_API_KEY` to `internal-api-key` to match `backend-go-secrets`. |
| `PX13-C2` | K8s configmap completeness | `implemented` | `ai-worker-config` added `KAFKA_TOPIC_INVENTORY_IMPORT`, `GCS_BUCKET_NAME`, `IMPORT_LOCAL_FILE_ROOT`. `backend-go-config` added `PUBLIC_BASE_URL`, `FIREBASE_PROJECT_ID`, `KAFKA_TOPIC_INVENTORY_IMPORT`. |
| `PX13-C3` | K8s CronJob secret/config references | `implemented` | `predictive_push_cronjob.yaml` and `planning_training_export_cronjob.yaml` changed from non-existent `pegasusx-spanner-credentials` / `pegasusx-backend-config` to canonical `backend-go-secrets` / `backend-go-config`. |

**Exit:** All automated gates green; `go build ./apps/backend-go/...` clean; `go test ./apps/backend-go/warehouse/... ./apps/backend-go/factory/... ./apps/backend-go/driver/... -count=1` pass.

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
=======
`PX0-A5` launch readiness guard
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
  Status: `implemented`

`PX7-A3` ai-worker k8s packaging
  Status: `implemented`

`PX11-C1` cloud staging smoke
Status: `implemented`

`PX12-A1` parity contract gates
Status: `implemented`

scripts/validate_launch_readiness.py
docs/LAUNCH_READINESS_RUNBOOK.md
