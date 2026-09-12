# Task Plan: PegasusX Deep Architectural & Systemic Hardening

## Goal
Harden the PegasusX distributed logistics engine against the 12 deep systemic, concurrency, and reliability challenges identified in the backend audit and live CodeGraph analysis, implementing zero-downtime secret rotation, distributed checkout sagas, monotonic Kafka streaming, and balanced financial ledgers with zero session disruption.

## Current Phase
Phase 7: Full Ecosystem Verification, CodeGraph Audit Re-Check & Gate Validation

## Phases

### Phase 1: Zero-Downtime Secret Rotation & Cryptographic Key Management (HARDEN-01)
- [x] Design and implement `auth/keyring.go` with `JWTKeyring` supporting primary signing key and candidate verification keys
- [x] Add `kid` (Key ID) header support in `jwtHeader` and O(1) key resolution with legacy fallback
- [x] Update `auth/jwt.go` with `ParseWithKeyring` while preserving backward-compatible `Parse` and `Issue`
- [x] Write unit tests in `auth/jwt_keyring_test.go` verifying rolling secret transitions with zero invalidation
- **Status:** complete

### Phase 2: Outbox-Backed Distributed Saga Coordinator & Crash Recovery (HARDEN-02)
- [x] Extend `ParentOrders` schema in `schema/spanner.ddl` with `SagaState`, `ExpectedChildCount`, `CreatedChildOrderIds`, and `LeaseExpiresAt`
- [x] Create `order/saga.go` implementing atomic child order tracking and state machine transitions
- [x] Update `order/unified_checkout.go` to register durable saga before child creation loop
- [x] Implement `order/saga.go` background sweep worker and wire in `runtime_workers.go`
- [x] Write unit tests in `order/saga_test.go` simulating mid-flight worker crashes and automatic compensation
- **Status:** complete

### Phase 3: Kafka Monotonic Offsets, Poison Pill Isolation & DLQ Replay Engine (HARDEN-03)
- [x] Enforce monotonic offset commits in `kafka/workerpool/workerpool.go` via `OffsetTracker`
- [x] Implement bounded retry loop and poison pill isolation to DLQ in `workerpool.go`
- [x] Build `cmd/replay-dlq/main.go` CLI with `--dry-run`, `--tenant-id`, `--source`, and `--re-emit` flags
- [x] Connect `TopicExceptions` and `TopicTelemetryLogistics` in `DispatcherConsumerTopics`
- [x] Write unit tests in `kafka/workerpool/workerpool_test.go` covering monotonicity and poison pills
- **Status:** complete

### Phase 4: Ephemeral Telemetry Decoupling from Spanner Commits (HARDEN-04)
- [x] Implement `DirectKafkaLocationBusEmitter` in `telemetryroutes/bus_emitter.go` to stream GPS pings directly to Kafka `TopicRealtime`
- [x] Reserve Spanner `OutboxEvents` emissions exclusively for business milestones (`GEOFENCE_ENTERED`, `STOP_COMPLETED`, etc.)
- [x] Wire `OutboxPublisher` into `bootstrap.App` and update `runtime_workers.go` to prefer direct Kafka emitter
- [x] Write unit tests in `telemetryroutes/bus_emitter_test.go` verifying direct streaming and header propagation
- **Status:** complete

### Phase 5: Double-Entry General Ledger Balance Invariance for Split Tender (HARDEN-05)
- [x] Create `payment/double_entry.go` defining standard double-entry chart of accounts and debit/credit postings
- [x] Implement $\sum \text{Debits} \equiv \sum \text{Credits}$ invariance assertion prior to transaction commits
- [x] Build `BuildSplitTenderJournalEntry` and `BuildSettlementJournalEntry` factory constructors
- [x] Write unit tests in `payment/double_entry_test.go` verifying arithmetic discrepancies and currency invariants are strictly rejected
- **Status:** complete

### Phase 6: Offline Driver/POS Reconciliation & Deterministic Conflict Hierarchy (HARDEN-06)
- [x] Implement deterministic physical custody supremacy rules in `order/offline_reconciliation.go`
- [x] Automatically convert concurrent online cancellations into post-delivery disputes and emit `ORDER_CONFLICT_RECONCILED`
- [x] Write unit tests in `order/offline_reconciliation_test.go` verifying physical custody supremacy and dispute logging
- **Status:** complete

### Phase 7: Full Ecosystem Verification, CodeGraph Audit Re-Check & Gate Validation
- [x] Run `go build ./...` across entire backend (passed with zero errors)
- [x] Run `go test ./...` on touched packages (`auth`, `order`, `kafka/workerpool`, `telemetryroutes`, `payment` all passed)
- [x] Re-run `make codegraph-audit` and assert contract drift and Kafka topic improvements
- [x] Run `make repo-hygiene-gate` and `make gen-contracts-gate` (all passed)
- **Status:** complete

### Phase 8: Multi-Platform Client Verification & Contract Sync
- [x] Audit multi-platform client route invocations across Android, iOS, and Web/Desktop portals
- [x] Wire missing `/v1/payload/exceptions/damaged` endpoint in `payloaderoutes/routes.go`
- [x] Document and register nested labor capacity endpoints in `laborcapacityroutes/routes.go`
- [x] Add missing `getRetailerTracking` implementation to `apps/retailer-app-desktop/lib/api.ts`
- [x] Document disabled negotiation contract hooks in `apps/supplier-portal/app/(portal)/exceptions/negotiations/page.tsx`
- [x] Provide `packages/api-client -> packages/api-core` compatibility symlink
- [x] Optimize contract parity check scripts with ripgrep and vectorized path matcher
- [x] Verify zero route drift via `role_row_contract_check.sh` and `role_row_contract_check_full.sh`
- **Status:** complete

### Phase 9: Multi-Tenancy SQL Taint Elimination & Outbox Dual-Write Hardening
- [x] Expand `audit_spanner_sql_tenancy` with all multi-tenant dimensions and entity primary keys
- [x] Remediate multi-tenant queries across `payment`, `globalproducts`, `segment`, `billing`, `retailer`, `supplier`, `partner`, `returns`, and `order`
- [x] Verify zero unscoped SQL queries across the entire repository (0 violations in CodeGraph)
- [x] Eliminate unconsumed Kafka topics (`demand.adjustment.updated`, `driver.score.updated`, `capacity.zone.updated`, `pegasusx-freezelocks`, `pegasusx-inventoryimportevents`)
- [x] Wrap state mutations across `order`, `retailer`, `returns`, `warehouse`, `factory`, `twin`, `syncroutes`, `inventory`, and `billing` with atomic `outbox.NewSpannerTxnBuffer` and `outbox.EmitJSON`
- [x] Verify 100% transactional outbox coverage (0 unprotected state mutations across 136 RW transaction files)
- [x] Re-run and pass `make codegraph-advanced-audit`, `make gen-contracts-gate`, `make repo-hygiene-gate`, and `make kafka-ha-gate`
- **Status:** complete

## Decisions & Changes Log
| Date | Phase | Decision | Rationale |
|---|---|---|---|
| 2026-09-04 | Initialization | Use multi-secret keyring with `kid` routing | Prevents mass lockout of drivers/cashiers during rotation |
| 2026-09-04 | Initialization | Spanner-backed Saga state on `ParentOrders` | Recovers stranded orders across server crashes without in-memory dependency |
| 2026-09-04 | Contract Sync | Mount `/v1/payload/exceptions/damaged` route | Eliminates runtime 404s when terminal reports damaged cargo exceptions |
| 2026-09-04 | Contract Sync | Link `packages/api-client` to `api-core` | Ensures unified tooling and zero script path divergence |

## Errors Encountered
| Error | Attempt | Resolution |
|---|---|---|
| `neo4j` module missing in system python3 | 1 | Used repository virtualenv `../.venv/bin/python3` which has Memgraph/Neo4j drivers installed |
