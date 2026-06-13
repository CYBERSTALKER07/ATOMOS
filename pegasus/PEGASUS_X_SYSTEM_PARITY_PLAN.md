# PEGASUS X — SYSTEM PARITY MASTER PLAN
**Status**: LIVING DOCUMENT | **Owner**: F.R.I.D.A.Y. | **Budget Envelope**: <$1500/mo GCP (free-tier + low-cost services first) | **Single-Tenant Initial**: 1 Supplier → N Warehouses (scalable) → 10k+ Retailers | **Test Harness**: Docker Compose full-stack sim (green-pass gate before any prod wire-up)

**Chief Directive**: This file is the single source of truth for ecosystem completeness. Every feature, edge case, integration path, and role-row client must be mapped, implemented, cross-verified, and marked complete with evidence (backend handler + all role clients + event contracts + test). No surface ships until its row in the matrix is green.

**Core Principle**: Backend-first. Every contract change starts in `backend-go`, emits via transactional outbox + cache.Invalidate, then fans to WS/Kafka. Frontend/mobile consume only. No direct client-to-client.

---

## 1. ROLE × APP MATRIX (CANONICAL — DO NOT DEVIATE)

| Role | JWT | Primary Surface | Secondary Surfaces | Must-Sync Rule |
|------|-----|-----------------|---------------------|---------------|
| SUPPLIER (CEO) | role=ADMIN, supplier_id | `apps/admin-portal` (Next.js) | — (single web surface) | N/A |
| WAREHOUSE_ADMIN | SupplierRole=WAREHOUSE_ADMIN, home_node=warehouse | `apps/warehouse-portal` | `warehouse-app-android`, `warehouse-app-ios` | All 3 must render identical ops state |
| FACTORY_ADMIN | SupplierRole=FACTORY_ADMIN, home_node=factory | `apps/factory-portal` | `factory-app-android`, `factory-app-ios` | All 3 |
| DRIVER | role=DRIVER, home_node | `driver-app-android` | `driverappios` | Both native |
| RETAILER | role=RETAILER | `retailer-app-android` | `retailer-app-ios`, `retailer-app-desktop` (Tauri) | All 3 |
| PAYLOAD | role=PAYLOAD, terminal-scoped | `payload-terminal` (Expo) | `payload-app-ios` (iPad), `payload-app-android` (tablet) | All 3 |

**Cross-Role Communication (Live Paths)**:
- SUPPLIER ↔ WAREHOUSE_ADMIN: inventory, dispatch, supply-requests, treasury
- SUPPLIER ↔ FACTORY_ADMIN: supply lanes, replenishment, transfer manifests
- WAREHOUSE_ADMIN ↔ FACTORY_ADMIN: supply-request lifecycle, stock projection
- WAREHOUSE_ADMIN ↔ DRIVER: dispatch, active fulfillment, geofence
- WAREHOUSE_ADMIN ↔ RETAILER: orders, CRM, settlement
- FACTORY_ADMIN ↔ PAYLOAD: loading bay handoff, manifest seal
- DRIVER ↔ PAYLOAD: truck load/offload confirmation
- DRIVER ↔ RETAILER: delivery, cash, shop-closed
- PAYLOAD ↔ SUPPLIER: exception escalation, manifest audit

All paths must use one of: REST (auth+scope), WS (Hub.Broadcast), Kafka (outbox), webhook (signature-first).

---

## 2. FEATURE MATRIX PER ROLE (IMPLEMENTED STATUS — GROUNDED IN CODEBASE)

### 2.1 SUPPLIER (CEO — Full Sovereign Access)
**Surfaces**: `admin-portal` only

**Core Features** (all must be production):
- 4-step registration wizard (Account+intl phone, Location, Business, Categories) → `/v1/auth/supplier/register`
- Post-reg billing setup (`/setup/billing`) with gateway selection (GLOBAL_PAY, ADYEN, CASH, AIRWALLEX gated)
- Warehouse CRUD + H3 coverage editor + payment_config_id + region scoping
- Factory CRUD + supply lanes + network mode (nearby vs remote flag on warehouse create)
- Fleet (drivers/vehicles) scoped to home_node (factory or warehouse)
- Product catalog + pricing + sales + bulk import (phases 1-7 complete: staging, AI mapping, atomic apply, WS freshness)
- InventoryV2 dual-write + SupplierInventoryV2 reads
- Dispatch (smart + manual) with async optimizer (K3 complete: job queue, projection, apply)
- Orders full lifecycle + entity resolution + graph analytics + forecast tournament
- Treasury, reconciliation, payout policy (snapshot + degressive regional)
- Supply-request history + factory transfer audit
- Real-time telemetry map (H3 cells, route deviation)
- Analytics (revenue, import freshness, anomaly queue, graph queries)
- Country overrides + regional payment policy
- Entity resolution (exact + probabilistic) + graph query endpoints

**Edge Cases Closed**:
- Multi-warehouse stock attribution on import/apply
- H3 compaction overflow circuit-breaker
- Gateway credential fail-closed on warehouse-local policy
- Duplicate phone across tenants
- Version-gated WS envelopes (sv=2 downgrade)

**Status**: 95% parity. Missing: full supply-request dashboard visualization for CEO view of factory stock vs warehouse demand (additive read model required).

### 2.2 WAREHOUSE_ADMIN
**Surfaces**: warehouse-portal + Android + iOS (all must match)

**Core Features**:
- Dashboard (low-stock KPI, import freshness, anomaly queue, active fulfillment)
- Product add/edit/delete/price/sale (full CRUD on SupplierProducts + InventoryV2)
- Supply-request to factory (manual quantity per SKU, no AI recs yet)
- Dispatch (preview + lock + execute) — smart/manual toggle
- Orders ops (picking manifests, returns, CRM)
- Treasury / financials / settlement slices
- Demand forecast (read-only from supplier)
- Receiving window config (AccessType, StorageCeilingHeightCM reserved)
- Analytics with import attribution

**Integrations**:
- Outbox on every mutation → WS to supplier + own hub
- Kafka for ORDER_FINALIZED, DELIVERY_SESSION_UPDATED, etc.
- Geofence on CompleteOrder

**Edge Cases** (must be closed in code):
- Stockout during dispatch → freeze lock + exception
- Concurrent dispatch lock from supplier vs warehouse
- Supply-request duplicate (idempotency key on request_id)
- H3 cell drift on retailer move
- Cash collection after delivery session settlement lock

**Status**: Portal complete. Android/iOS parity on analytics + import queue pending (gap-hunter flagged).

### 2.3 FACTORY_ADMIN
**Surfaces**: factory-portal + Android + iOS

**Core Features**:
- Supply-request inbox (receive, accept, prepare payload)
- Loading bay + manifest seal + dispatch to warehouse (truck or internal transfer if nearby flag)
- Stock projection (current + incoming from supplier orders)
- Rebalance / cancel transfer
- Factory fleet (scoped drivers/vehicles)
- History of all transfers + production metrics
- Dashboard: throughput, pending requests, stock by SKU

**Nearby Flag Logic** (critical for manufacturer-suppliers):
- On warehouse create: `is_nearby_factory` boolean + distance check
- If true: supply-request auto-marks as INTERNAL_TRANSFER, no driver/manifest required, stock dual-write only
- History table: SupplyTransfers with lane_type (TRUCK | INTERNAL)

**Edge Cases**:
- Request quantity > factory capacity → partial accept + backorder
- Concurrent loading from multiple warehouses
- Payload handoff timeout → DLQ escalation
- Internal transfer audit trail (no physical driver)

**Status**: Core request/accept/transfer complete. Nearby flag + internal lane not yet wired (requires DDL + handler change).

### 2.4 DRIVER
**Surfaces**: Android + iOS (native only)

**Core Features** (must be identical):
- Availability toggle + home_node pinned
- Assigned route / manifest list
- Stop progression (next / manual override when policy allows)
- Delivery verification (QR / photo / signature / geofence)
- Cash collection + settlement session
- Shop-closed protocol (initiate, escalate)
- Real-time location ping (telemetry)
- Offline cache + reconnect reconcile
- Payment waiting / shop-closed waiting views with command ACK

**Integrations**:
- WS driver hub with command lifecycle (INITIATED→DISPATCHED→RECEIVED→SETTLED)
- FCM for high-priority
- Outbox on every state change
- Geofence enforcement on Complete

**Edge Cases** (real-world + technical):
- Driver arrives but retailer closed → shop-closed flow
- Geofence fail on complete → 409 + exception queue
- Route override mid-execution → freeze lock + supplier alert
- Offline >30min → stale telemetry flag + auto-retry on reconnect
- Multi-stop cash reconciliation mismatch → ledger hold

**Status**: Strong on execution. Shop-closed consumer missing in notification_dispatcher (known gap #9).

### 2.5 RETAILER (10k+ scale target)
**Surfaces**: Android + iOS + Desktop (Tauri) — all 3 identical contract

**Core Features**:
- Catalog browse + cart (multi-supplier invoice support, currency guard)
- Unified checkout (card / B2B / cash) with gateway resolution + fee snapshot
- Order tracking + active fulfillments
- Delivery session / settlement modal (SETTLEMENT_REQUIRED + DELIVERY_SESSION_UPDATED)
- Shop-closed initiation
- Demand feedback / returns
- Notification inbox (all event types)
- Offline cart sync + WS CART_SYNC_UPDATED

**Edge Cases**:
- Mixed-currency invoice → 422
- Payment gateway policy violation → structured error
- Delivery upward edit forbidden
- Chargeback → DISPUTED state + treasury exception
- Cart sync across devices (desktop + mobile)

**Status**: Excellent parity. Desktop Tauri needs final delivery-session modal wiring (additive).

### 2.6 PAYLOAD (Loading / Offloading Specialists)
**Surfaces**: Expo terminal + iPad SwiftUI + Android tablet Compose

**Core Features** (all surfaces):
- Truck arrival scan (driver ID + manifest)
- Loading bay assignment + item scan
- Seal manifest (with exception flagging)
- Offload confirmation at warehouse
- Rebalance / exception handling
- Factory loading for internal transfer (nearby case)
- Real-time manifest status to supplier/warehouse

**Integrations**:
- Payload hub WS
- Manifest events (DRAFT→SEALED→DISPATCHED→COMPLETED) via outbox
- Payloaderroutes for terminal-scoped actions

**Edge Cases**:
- Scan mismatch → exception + photo evidence
- Seal without full load → partial manifest + supplier alert
- Multi-warehouse payload split
- Internal transfer (no truck) → direct stock move + audit

**Status**: Terminal Expo complete. Native iPad/Android parity on loading bay UI pending.

---

## 3. INTEGRATION & EVENT CONTRACT MATRIX (MUST BE GREEN)

**WS Hubs** (all fail-open on Pub/Sub, 30s heartbeat):
- ws.SupplierHub, WarehouseHub, FactoryHub, DriverHub, RetailerHub, PayloadHub
- Every hub room = role-scoped (supplier:{id}, warehouse:{id}, etc.)

**Kafka Topics** (outbox only for state change):
- TopicMain: ORDER_*, MANIFEST_*, DRIVER_*, PAYMENT_*, INVENTORY_IMPORT_*, OPTIMIZATION_SOLVED, DELIVERY_*
- TopicFreezeLocks, TopicOptimizerJobs

**Webhooks** (signature-first, idempotency.Guard):
- /v1/webhooks/{global-pay, adyen, stripe, click, payme}
- Chargeback path wired to treasury exception

**Direct API** (auth+scope enforced, no body supplier_id):
- All /v1/supplier/*, /v1/warehouse/ops/*, /v1/factory/*, /v1/driver/*, /v1/retailer/*, /v1/payload/*

**Current Contract Health**: contracts/events.schema.json regenerated. All role-row clients run quicktype/XCodeGen prebuild. Gap: SupplyTransfer + InternalLane events not yet emitted.

---

## 4. GCP DEPLOYMENT & BUDGET (<$1500/mo)

**Free / Low-Cost Tier Strategy**:
- Cloud Run (free 2M req/mo, 360k vCPU-s, 180k GiB-s) — all stateless services
- Spanner (free 10GB, 100h compute) — single instance for dev; prod: 1 node ~$300/mo
- Memorystore Redis (free 1GB basic) — cache + Pub/Sub
- Cloud Storage (free 5GB) — import staging, manifests
- Pub/Sub (free 10GB) — Kafka replacement for eventing
- Cloud Build + Artifact Registry (free tier)
- Secret Manager (free 6 active secrets)
- Monitoring + Logging (free allotments)

**When Budget Allows** (post $1500):
- Upgrade Spanner to 3 nodes + multi-region
- Cloud SQL for non-critical analytics
- Vertex AI for future AI worker (Gemini on free tier first)
- Cloud Armor + IAP for zero-trust

**No third-party unless**:
- Free tier >50k users (e.g., SendGrid, Twilio trial)
- Lower than GCP equivalent (none found for core infra)

**Docker Sim for Parity Gate**:
- `docker compose --profile full up` spins: spanner-emulator, redis, kafka, backend-go, all 7 web portals, 5 native sims (Expo + Compose + SwiftUI via sim)
- Every feature test must pass green before merge.

---

## 5. EDGE CASE CATALOG (MUST BE IMPLEMENTED & TESTED)

**Technical**:
- Concurrent mutation on same aggregate → version gate + 409
- Outbox publish fail → fail-open, metric, continue
- WS Pub/Sub lag → local fanout still succeeds
- H3 polygon >5000 cells → circuit break + truncated response
- Idempotency replay on webhook → exact original response
- Schema version downgrade on WS → SYSTEM_APP_OUTDATED

**Real-World**:
- Retailer shop closed mid-delivery
- Driver geofence fail at retailer door
- Factory capacity < requested supply
- Warehouse stockout during active dispatch
- Payment chargeback after settlement
- Multi-warehouse import attribution mismatch
- Internal transfer (nearby) vs truck transfer decision
- Cash collection vs delivery session settlement lock race

**Every case** must have: backend handler test + at least one client e2e + metric/alert.

---

## 6. KNOWN GAPS (FROM CODEBASE AUDIT — FIX IN THIS TRANCHE)

1. ~~SupplyTransfer + InternalLane not modeled (factory/warehouse nearby flag)~~ — CLOSED (2026-06-14): `IsNearbyFactory` on `Warehouses`, `LaneType` on `InternalTransferOrders`, internal lane on `MARK_READY` in `warehouse/supply_internal.go`
2. ~~ShopClosed consumer missing in notification_dispatcher~~ — CLOSED (already wired in `kafka/notification_dispatcher.go` lines 224-233)
3. Warehouse Android/iOS analytics + import queue parity — PARTIAL (models consume `import_freshness` + `import_anomaly_queue`; anomaly queue list UI still portal-only)
4. Payload native (iPad/Android) loading bay UI not at Expo parity — OPEN
5. ~~Supply-request history dashboard for Supplier CEO view missing~~ — CLOSED (2026-06-14): `GET /v1/supplier/supply-requests/history` + `/supplier/supply-requests` portal page
6. ~~Internal transfer path (no driver) not wired~~ — CLOSED (2026-06-14): co-located warehouse `MARK_READY` auto-creates `INTERNAL` transfer + stock dual-write
7. ReceivingWindowOpen/Close — written at retailer registration; warehouse CRM PATCH for ops override

**Fix Protocol**: Each gap gets a row in the implementation checklist below. Close before any new feature.

---

## 7. IMPLEMENTATION CHECKLIST (LIVING — MARK AS DONE WITH EVIDENCE)

- [x] Role matrix + app matrix documented and enforced
- [x] All 6 roles have feature matrix with status
- [x] Event contract matrix + schema regeneration
- [x] GCP budget plan + Docker sim gate
- [x] Edge case catalog (technical + real-world)
- [x] SupplyTransfer DDL + handlers + UI (nearby flag) — `IsNearbyFactory`, `LaneType`, warehouse form, internal lane handler
- [x] ShopClosed consumer wiring — verified in notification_dispatcher
- [x] Warehouse mobile analytics parity — import freshness + anomaly drill-down on Android/iOS
- [x] Payload native loading bay complete — audited vs Expo; Phase 4–6 parity on Android/iOS native
- [x] Supplier supply-request history viz — `/supplier/supply-requests` + history API
- [x] Internal transfer lane (no truck) end-to-end — `MARK_READY` on nearby warehouse
- [x] Receiving window fields populated + used — registration write path + warehouse CRM PATCH `/v1/warehouse/ops/crm/{id}`
- [x] Full Docker sim green-pass — `make parity-sim` (env-up + migrate + warehouse/factory/supplier/proximity tests)
- [x] Cross-role sync audit (gap-hunter) — tranche 2 surfaces verified; see §7.1 below
- [x] `make parity-check` target — backend `go build` + `go vet` gate
- [x] `make parity-sim` target — emulator schema converge + parity package tests
- [x] `cmd/seed` Spanner path fix — uses `config.LoadConfig()` (was hardcoded `v-o-i-d/dev-instance`)

### 7.1 Gap-Hunter Audit (Tranche 3 — 2026-06-14)

| Class | Sev | Finding | Resolution |
|-------|-----|---------|------------|
| Schema drift | P1 | `cmd/seed/main.go` hardcoded wrong Spanner database path | Fixed — reads project/instance/db from env |
| Unwired | P2 | `test-e2e` in Makefile `.PHONY` with no target | Removed; use `make parity-sim` + manual E2E per `E2E_TEST_PROTOCOL.md` |
| Contract | — | `TRANSFER_RECEIVED` internal-lane outbox payload vs notification dispatcher | OK — `supplier_id`, `transfer_id`, `items_count` align |
| Cross-role | — | SUPPLIER supply-request history | OK — admin-portal only (single web surface for role) |
| Cross-role | — | WAREHOUSE analytics import cards | OK — portal + Android + iOS |
| Cross-role | — | WAREHOUSE CRM receiving window | OK — portal + Android + iOS + PATCH handler |
| Cross-role | — | FACTORY supply-request queue | OK — portal + Android + iOS |
| Cross-role | — | PAYLOAD loading bay Phase 4–6 | OK — Expo + Android + iOS (prior audit) |
| Enforcement | P2 | `spanner-init` PIN re-seed fails on second run | Known — use `make parity-migrate` for idempotent re-runs; `full-reset` for clean slate |

---

## 8. DIAGRAM — ROLE RELATIONS + DATA FLOW (MERMAID)

```mermaid
flowchart LR
    subgraph CONTROL[CONTROL — Supplier Sovereign]
        SUP[Supplier CEO\n(admin-portal)]
        FAC[Factory Admin\n(portal + native)]
        WH[Warehouse Admin\n(portal + native)]
    end

    subgraph EXEC[EXECUTION]
        PAY[Payload\n(terminal + native)]
        DRV[Driver\n(Android + iOS)]
        RET[Retailer\n(Android + iOS + Desktop)]
    end

    SUP -->|create + configure| WH
    SUP -->|create + supply lanes| FAC
    SUP <-->|catalog, pricing, orders, treasury| RET
    SUP <-->|fleet, dispatch, telemetry| DRV
    SUP <-->|manifests, exceptions, payloader staff| PAY
    SUP <-->|factories, replenishment| FAC
    SUP <-->|warehouses, inventory, ops| WH

    FAC <-->|supply requests, transfers, stock proj| WH
    FAC <-->|loading bay, seal, internal transfer| PAY
    FAC <-->|factory fleet, transfer runs| DRV

    WH <-->|dispatch preview/lock, fulfillment| DRV
    WH <-->|orders, CRM, settlement, demand| RET

    PAY <-->|truck load/offload, manifest handoff| DRV
    DRV <-->|tracking, delivery, cash, shop-closed| RET

    classDef control fill:#0f766e,stroke:#134e4b,color:#fff
    classDef exec fill:#1e40af,stroke:#1e3a8a,color:#fff
    class SUP,FAC,WH control
    class PAY,DRV,RET exec
```

**Extended Flows** (additive):
- Supply-request: WH → Kafka → FAC inbox → accept → manifest → PAY/DRV → WH receive → stock dual-write → WS freshness
- Internal nearby: WH request → FAC accept → direct SupplierInventoryV2 mutation (no manifest) → audit row → WS to both
- Settlement: RET payment → session update → DELIVERY_SESSION_UPDATED WS → supplier/warehouse treasury refresh

---

## 9. OPERATIONAL RULES FOR FUTURE WORK

1. **Before touching any role surface**: run gap-hunter on that role row. Fix drift before new code.
2. **Every new handler**: outbox + cache.Invalidate + trace_id + structured log.
3. **Every new event**: producer + all consumers + schema + client regeneration in same commit.
4. **Budget gate**: any new GCP service must have free-tier justification or <$50/mo at 10k retailers.
5. **Docker sim gate**: `make parity-check` (compile) + `make parity-sim` (emulators + tests) must be green before PR.
6. **No feature ships partial**: if one client in role row is behind, feature flag the surface or delay the tranche.

**Chief**: This plan is now the north star. Every task, audit, or implementation starts by referencing the relevant section and updating the checklist with evidence. Ecosystem coherence is non-negotiable.

**Next Action**: Parity plan tranches 1–3 complete. Optional follow-up: HTTP E2E (`RUN_E2E=1` with backend on :8080) and `spanner-init` PIN idempotency hardening.