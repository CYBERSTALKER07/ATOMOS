# Changes Log — Milestone 3: Cross-Role Real-Time Monotonic Pipeline Parity

**Agent**: `teamwork_preview_worker_m3_11`  
**Milestone**: Milestone 3 — Requirement R3  
**Target Codebase**: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`  
**Date**: 2026-09-23

---

## Summary of Changes

Milestone 3 enforces real-time pipeline parity and monotonic event guarantees across the entire `pegasus.x` ecosystem (backend Go microservices, Redis 7 streams/pubsub, WebSocket hub, and React/Next.js desktop client applications).

### 1. Pub/Sub Channel Casing & Outbox Relay Synchronization
- **`backend/internal/outbox/relay.go`**:
  - Relay publishes outbox events to the canonical stream (`canonicalStream`), lowercase aggregate channel (`events:%s`), and uppercase aggregate channel (`events:%s`).
  - Guarantees that clients subscribing to `events:ORDER` or `events:order` receive events identically without dropouts.
- **`backend/internal/ws/hub.go`**:
  - Configured Redis pub/sub subscription to listen on both lowercase and uppercase channel patterns (`events:order`, `events:ORDER`, `events:route`, `events:ROUTE`, `events:inventory`, `events:INVENTORY`, `events:warehouse`, `events:WAREHOUSE`, `events:retailer`, `events:RETAILER`, `events:supplier`, `events:SUPPLIER`, etc.).

### 2. WebSocket Hub Monotonic Envelope & Monotonic seq Counter
- **`backend/internal/ws/hub.go`**:
  - Implemented `normalizeEventType(rawType string)` providing bidirectional transformation between lowercase dot-notation (`order.created`) and uppercase snake_case (`ORDER_CREATED`).
  - Enriched `RealtimeEnvelope` with `seq` (monotonic uint64 sequence ID), `event_id`, `type` (uppercase), `event_type` (lowercase), `stream`, `timestamp`, `payload`, and `data` (backward compatibility).
  - Upgraded `BroadcastEnvelope` with atomic counter (`atomic.AddUint64(&h.seq, 1)`), ring buffer storage (`lastEvents` bounded ring buffer of capacity 500), and thread-safe broadcast to connected clients.
  - Enhanced `Broadcast(msg []byte)` to intercept raw JSON payloads lacking sequence metadata and auto-wrap them into structured `RealtimeEnvelope` frames before fan-out.
  - Implemented `GetEventsSince(sinceSeq uint64) ([]*RealtimeEnvelope, bool)` with gap detection returning `fullResync: true` if the client's `sinceSeq` has been overwritten in the ring buffer.
- **Envelope Bypasses Eliminated**:
  - Updated all services that were previously calling `hub.Broadcast(rawBytes)` to use `hub.BroadcastEnvelope(...)`:
    - `backend/internal/retailer/service.go`: `retailer.order_state_changed` / `RETAILER_ORDER_STATE_CHANGED`.
    - `backend/internal/payout/service.go`: `payout.processed` / `PAYOUT_PROCESSED`.
    - `backend/internal/returns/service.go`: `return.quarantine_created` / `RETURN_QUARANTINE_CREATED`.
    - `backend/internal/notifications/service.go`: `notification.dispatched` / `NOTIFICATION_DISPATCHED`.
    - `backend/internal/wmsops/service.go`: `wms.task_dispatched` / `WMS_TASK_DISPATCHED`.
    - `backend/internal/seasonalcore/service.go`: `seasonal.surge_updated` / `SEASONAL_SURGE_UPDATED`.

### 3. Atomic Outbox Pairing in Same `pgx.Tx` Transaction
- **`backend/internal/warehouse/`**:
  - Added transaction-aware repository methods `CreateWarehouseTx`, `UpdateOnboardingStatusTx`, `SaveShortageClaimTx`.
  - Refactored `RegisterWarehouse`, `CompleteOnboarding`, and `ReconcileInboundBlind` to execute database table mutations and `outbox.Emit` inside the same `s.pool.RunInTx` transaction block.
- **`backend/internal/rebate/`**:
  - Added transaction-aware repository methods `SaveContractTx`, `RecordAccrualTx`.
  - Refactored `AccrueForDeliveredOrder` and `SettleContract` to execute entity mutation and `outbox.Emit` atomically inside `s.pool.RunInTx`.
- **`backend/internal/consignment/`**:
  - Added transaction-aware repository methods `SaveAgreementTx`, `SaveSettlementVoucherTx`.
  - Refactored `InboundConsignmentReceive` and `ExecutePickOwnershipTransfer` to execute balance mutations and `outbox.Emit` atomically inside `s.pool.RunInTx`.
- **Purged Ignored Outbox Errors (`_ = outbox.Emit`)**:
  - `backend/internal/ump/engine.go` (lines 257, 267): Added strict error return if `outbox.Emit` fails, triggering transaction rollback.
  - `backend/internal/inventory/service.go` (lines 116, 158, 209, 371): Added strict error propagation if `outbox.Emit` fails.
  - `backend/internal/order/service.go` (lines 501, 587, 594, 1367): Added strict error checking and transaction rollback if `outbox.Emit` fails.
  - `backend/internal/claims/repository.go` (line 663): Added strict error check and return on outbox failure.
- **Driver Cancelled Order Guard**:
  - `backend/internal/api/handlers_fleet_driver.go` (line 1017): Validated proactive and reactive checks for `currentOrder.Status == models.StatusCancelled` returning HTTP 409 Conflict with `"order_cancelled"` problem detail.

### 4. Desktop Application Event Invalidation & SSE Streaming
- **`apps/warehouse-desktop/lib/fleet-ws-events.ts`**:
  - Enhanced `parseWsEventType` to parse JSON strings and normalize dot-notation event types to uppercase snake_case (`order.created` -> `ORDER_CREATED`, `route.assigned` -> `ROUTE_ASSIGNED`), enabling immediate cache invalidation in React query hooks.
- **`apps/supplier-desktop/lib/supplier-ws-events.ts`**:
  - Enhanced `parseSupplierWsEventType` to support payload objects and JSON strings with event normalization for supplier-desktop events.
- **`backend/internal/observability/logger.go` & `metrics.go`**:
  - Implemented `Flush()` on `loggingResponseWriter` and `statusRecorder` delegating to underlying `http.Flusher`. This prevents runtime panic or failure on HTTP streaming / Server-Sent Events (SSE).
- **`backend/internal/api/handlers_supplier.go`**:
  - Updated `handleSupplierSyncCatchUp` to support both `since` and `since_seq` query parameters.
  - Implemented `handleSupplierEventsSSE` for streaming SSE events over `/v1/supplier/events`.
- **`backend/internal/api/router.go`**:
  - Registered public endpoints: `GET /v1/events/sync` (global role sync catch-up) and `GET /v1/supplier/events` (SSE stream).

### 5. Automated Tests
- **`backend/internal/ws/hub_test.go`**:
  - `TestHubDualCasingNormalization`: Verifies bidirectional conversion of event types.
  - `TestHubBroadcastAutoWrapping`: Verifies un-enveloped JSON payloads are automatically enveloped with monotonic sequence IDs.
  - `TestHubRingBufferOverflowResync`: Verifies buffer boundary conditions and `fullResync: true` detection.
- **`backend/internal/api/realtime_pipeline_parity_test.go`**:
  - `TestOrderComplete_CancelledOrderReturns409Conflict`: Integration test confirming HTTP 409 Conflict returned when completing a cancelled order.
  - `TestEventsSyncCatchUp_MonotonicParity`: Integration test verifying monotonic sequence replay via `/v1/events/sync` and `/v1/supplier/sync`.
  - `TestSupplierEventsSSE_Stream`: Integration test verifying HTTP Server-Sent Events (SSE) streaming over `/v1/supplier/events`.
- **Regression Verification**:
  - All modified packages (`ws`, `outbox`, `warehouse`, `rebate`, `consignment`, `api`, `retailer`, `payout`, `returns`, `notifications`, `wmsops`, `seasonalcore`) pass `go test -v -race`.
  - `go vet ./...` completed with zero warnings.
  - `go build ./cmd/server` builds cleanly.
