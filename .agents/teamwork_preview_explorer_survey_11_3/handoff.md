# Handoff Report: Requirement R3 Real-Time Monotonic Pipeline Audit across All 7 Roles

**Author**: `teamwork_preview_explorer_survey_11_3`  
**Target Codebase**: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`  
**Parent Agent**: `teamwork_preview_orchestrator_11` (`d877571c-b5bd-4489-a1cb-441c991bb03d`)  
**Milestone**: M0 (Survey & In-Depth Audit) -> M3 (Cross-Role Real-Time Monotonic Pipeline Parity)  
**Type**: Hard Handoff (Investigation & Survey Complete)  

---

## 1. Observations

### 1.1 Two-System Architectural Boundary
- `grep_search` for `spanner` across `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend`: **0 results found**.
- `grep_search` for `kafka` across `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend`: **0 results found**.
- `backend/go.mod:5-16`: Dependencies are strictly sovereign: `github.com/jackc/pgx/v5 v5.10.0`, `github.com/redis/go-redis/v9 v9.22.0`, `github.com/gorilla/websocket v1.5.3`, `github.com/go-chi/chi/v5 v5.3.2`. Zero Google Cloud Spanner or Kafka SDKs.

### 1.2 Outbox Relay & Redis 7 Streams
- `backend/internal/outbox/relay.go:123-130`:
  ```go
  query := `
      SELECT event_id, aggregate_type, aggregate_id, event_type, payload
      FROM outbox_events
      WHERE NOT published
      ORDER BY created_at ASC
      LIMIT $1
      FOR UPDATE SKIP LOCKED
  `
  ```
  Relay polls unpublished events using `FOR UPDATE SKIP LOCKED`.
- `backend/internal/outbox/relay.go:163-178`:
  ```go
  values := map[string]any{
      "event_id":       it.eventID.String(),
      "aggregate_id":   it.aggregateID, // Partition key
      "aggregate_type": it.aggregateType,
      "event_type":     it.eventType,
      "payload":        string(it.payload),
      "created_at":     time.Now().UTC().Format(time.RFC3339Nano),
  }
  _, err := w.redis.XAdd(ctx, &goredis.XAddArgs{
      Stream: canonicalStream,
      MaxLen: 100000,
      Approx: true,
      Values: values,
  }).Result()
  ```
  Durable stream persistence uses `XADD` with aggregate root partition key `aggregate_id`. Dead-letter records inserted into `outbox_dead_letters` on failure (`relay.go:191-197`).
- **Pub/Sub Channel Mismatch**:
  - `relay.go:200-203`: Publishes to `canonicalStream` (e.g. `events:order`, `events:payload:sealed`, `events:doorstep:arrived`, `events:doorstep:tender_settled`, `events:fleet:breakdown_reported`) and `aggregateChannel` (`fmt.Sprintf("events:%s", strings.ToLower(it.aggregateType))`).
  - `ws/hub.go:169`: Subscribes to UPPERCASE channels: `"telemetry:drivers", "events:ORDER", "events:UMP", "events:FLEET", "events:CLAIMS", "events:PICKWAVE", "events:MANIFEST", "events:EPOD", "events:WAREHOUSE", "events:notifications", "alerts:fleet:breakdown_rescue"`.
  - Redis channel matching is byte-exact. Lowercase channels emitted by `relay.go` are never received by `ws/hub.go`.

### 1.3 WebSocket Hub & Monotonic Envelope
- `backend/internal/ws/hub.go:31-38`:
  ```go
  type RealtimeEnvelope struct {
      Seq       int64                  `json:"seq"`
      EventType string                 `json:"event_type"`
      Type      string                 `json:"type"`
      Payload   map[string]interface{} `json:"payload"`
      Timestamp int64                  `json:"timestamp"`
  }
  ```
  `BroadcastEnvelope` (`hub.go:111-133`) atomically increments sequence counter `atomic.AddInt64(&h.seq, 1)`, appends to a 2,000-element ring buffer `recentEvents`, and broadcasts to connected clients.
- `backend/internal/ws/hub.go:137-162`: `GetEventsSince(since int64)` returns missed events or `fullResync: true` when `since < oldestSeq - 1`.
- **Raw Broadcast Bypass**:
  - `backend/internal/retailer/service.go:501`: `s.wsHub.Broadcast(data)` sends raw JSON without `seq`.
  - `backend/internal/payout/service.go:58`: `s.wsHub.Broadcast(data)` sends raw JSON without `seq`.
  - `backend/internal/returns/service.go:180`: `s.wsHub.Broadcast(payload)` sends raw JSON without `seq`.
  - `backend/internal/notifications/service.go:180`: `s.wsHub.Broadcast(payload)`.
  - `backend/internal/wmsops/service.go:122`: `s.wsHub.Broadcast(eventPayload)`.
  - `backend/internal/seasonalcore/service.go:83, 148`: `s.wsHub.Broadcast(msg)`.
- **Single-Role Replay Exposure**:
  - `backend/internal/api/router.go:414` and `handlers_supplier.go:1076`: Only `GET /v1/supplier/sync?since={seq}` is registered. No generic `/v1/events/sync` exists for retailer, warehouse, driver, or dispatcher.

### 1.4 Atomic Outbox Pairing across 7 Roles
- **Properly Paired Atomic Flows**:
  - `backend/internal/order/service.go:565-602, 729, 797, 954, 1233, 1376`: Order state transitions execute inside `s.pool.RunInTx(ctx, func(tx pgx.Tx) error { ... })` and emit `outbox.Emit(ctx, tx, "ORDER", ...)`.
  - `backend/internal/order/catch_weight.go:214`: Emits `order.catch_weight_adjusted` in `RunInTx`.
  - `backend/internal/wms/waves.go:180, 281, 360`: Wave generation, pick, and completion emit in `RunInTx`.
  - `backend/internal/wms/lots.go:137`: Putaway lot emits `stock.lot_putaway` in `RunInTx`.
  - `backend/internal/pickwave/repository.go:455, 506, 635, 664, 732, 756, 812, 833`: Pick tasks and wave sealing execute within `tx.Begin(ctx)` ... `tx.Commit(ctx)`.
  - `backend/internal/dock/repository.go:208, 247, 303, 361`: Bay operations emit inside `tx.Begin(ctx)` ... `tx.Commit(ctx)`.
  - `backend/internal/manifest/sealing.go:147, 304`: Manifest sealing emits inside `s.pool.RunInTx`.
  - `backend/internal/dispatch/service.go:441, 919`: Commit dispatch and breakdown rescue emit in `s.pool.RunInTx`.
  - `backend/internal/fleet/repository.go:273, 426, 485, 781, 843, 1023, 1259, 1430`: Fleet vehicle, driver, and DVIR safety failure emit in `r.pool.RunInTx`.
  - `backend/internal/epod/repository.go:338, 375, 409, 473, 482, 672`: Dispatch, stop arrival, delivery confirmation, and offline sync drain emit in `tx.Begin(ctx)` ... `tx.Commit(ctx)`.
  - `backend/internal/cashrecon/cit_drawer.go:260, 416, 590`: CIT threshold exceeded, mid-shift vault drop, and bank deposit emit in `s.pool.RunInTx`.
  - `backend/internal/creditnote/service.go:168`: Credit note creation emits in `s.pool.RunInTx`.
  - `backend/internal/soliq/receipt.go:122` & `soliq/service.go:211, 346`: Soliq receipts and E-Factura emit in `RunInTx`.
  - `backend/internal/api/handlers_payment.go:92, 133, 265, 346`: Payment handover, debt creation, and webhook clear emit in `s.pool.RunInTx`.
- **Separated-Transaction Anti-Pattern**:
  - `backend/internal/warehouse/service.go:94-108`: `s.repo.CreateWarehouse(ctx, wh)` executes and commits, then lines 99-108 executes a second transaction to call `outbox.Emit`.
  - `backend/internal/warehouse/service.go:352-365`: `s.repo.UpdateOnboardingStatus` commits, then a second transaction calls `outbox.Emit`.
  - `backend/internal/warehouse/service.go:592-629`: Shortage claim commits, then a second transaction calls `outbox.Emit`, and a third transaction calls `outbox.Emit` for blind reconciliation.
  - `backend/internal/rebate/service.go:61-82`: `s.repo.RecordAccrual` and `SaveContract` commit, then a second transaction calls `outbox.Emit`.
  - `backend/internal/consignment/service.go:140-154`: `s.repo.SaveAgreement` commits, then a second transaction calls `outbox.Emit`.
- **Zero-Outbox Fire-and-Forget Mutations**:
  - `backend/internal/retailer/service.go:49-1311`: 39 mutating operations (`ProcessSale`, `OpenShift`, `CloseShift`, `CommitStockCount`, `ConfirmProposal`, `EnableCapability`, etc.) call `s.broadcastEvent` directly, emitting ephemeral Redis messages on `events:RETAILER` with zero outbox table writes.
  - `backend/internal/payout/service.go:73, 172, 206, 239`: Batch creation, approval, and disbursement call `s.broadcastEvent` directly with zero outbox writes.
  - `backend/internal/returns/service.go:140, 163-182`: Return inspection and quick restock call `s.broadcastEvent` directly with zero outbox writes.
  - `backend/internal/wmsops/service.go:110-136`: Operational broadcasts call `s.wsHub.Broadcast` with zero outbox writes.
  - `backend/internal/supplier/repository.go:1488-1630`: `CreateProduct`, `UpdateProduct`, `DeleteProduct` execute SQL queries directly on `p.pool` with zero outbox writes.
- **Ignored Outbox Errors (`_ = outbox.Emit`)**:
  - `backend/internal/ump/engine.go:257, 267`.
  - `backend/internal/inventory/service.go:116, 158, 209, 371`.
  - `backend/internal/order/service.go:501, 587, 594, 1357`.
  - `backend/internal/claims/repository.go:663`.

### 1.5 Desktop & Client Real-Time Invalidation
- `apps/retailer-desktop/lib/ws.tsx:78-86`: Implements dual `type` and `event_type` parsing. `SessionReconcileListener` triggers reconciliation on `reconnectEpoch`. However, does not track `seq` or detect sequence gaps.
- `apps/warehouse-desktop/lib/fleet-ws-events.ts:5`:
  ```typescript
  export const parseWsEventType = (e: any) => (typeof e === 'string' ? e : e?.type || '');
  ```
  When WebSocket delivers string data, it returns the raw JSON string rather than parsing it. Consequently, `WAREHOUSE_ORDERS_REFRESH_EVENTS.has(eventType)` and `WAREHOUSE_LOCATION_PATCH_EVENTS.has(eventType)` evaluate to `false` on every message.
- `apps/warehouse-desktop/lib/useNotifications.ts:223`: Dispatches `CustomEvent("sync-invalidate")`, but **0 components listen to it**.
- `apps/supplier-desktop/lib/use-supplier-ws-refresh.ts:73`: Attempts `EventSource` connection to `${apiBase}/v1/supplier/events`. This endpoint does not exist in `router.go`, failing with HTTP 404.
- `packages/ws-refresh-contract/index.ts:36-145`: Expects UPPERCASE snake_case events (`ORDER_CREATED`, `MANIFEST_SEALED`), while backend emits lowercase dot-notation (`order.created`, `manifest.sealed`).

---

## 2. Logic Chain

1. **Transactional Integrity**: The outbox pattern guarantees reliability only if the state mutation and outbox record commit in the exact same transaction (`pgx.Tx`). When `warehouse/service.go`, `rebate/service.go`, and `consignment/service.go` execute the mutation and then open a second transaction for `outbox.Emit`, process crashes or network failures between the two operations leave the database mutated without an event, violating the Zero-Orphaned-Features Doctrine.
2. **Event Durability vs Ephemeral Broadcast**: In `retailer/service.go` and `payout/service.go`, calling `s.broadcastEvent` without writing to `outbox_events` means that if a client is offline, reconnecting, or partitioned during the mutation, the event is permanently lost with no chance of replay.
3. **Pub/Sub Transport Case Sensitivity**: In Redis, Pub/Sub channels are case-sensitive strings. Because `relay.go` emits to `events:order` (lowercase) and `ws/hub.go` subscribes to `events:ORDER` (uppercase), Redis filters out the messages, breaking real-time propagation from relay to WebSocket clients.
4. **Monotonic Ordering & Replay**: Sequence counter `seq` exists to detect dropped frames. When domain services call `wsHub.Broadcast([]byte)` directly with raw byte slices, frames lack `seq` and bypass the ring buffer, preventing clients from validating sequence continuity.
5. **Client Deserialization Failure**: Because `parseWsEventType` in `fleet-ws-events.ts` does not parse JSON strings, `eventType` is set to the raw JSON string. Since `'{"seq":1,...}'` never equals `'ORDER_CREATED'`, all real-time event handlers in `warehouse-desktop` are dead on arrival, and the UI never invalidates without polling.

---

## 3. Caveats

- **Test Suite Status**: Unit tests in `backend/internal/outbox/...` and `backend/internal/ws/...` currently pass because they test isolated functions (like `ResolveStreamKey` string mapping and single-instance `Hub` in-memory broadcasts) rather than an integrated end-to-end flow through PostgreSQL and Redis.
- **Mobile Apps Scope**: `apps/driver-app-android` and `apps/driver-app-ios` communicate via REST endpoints (`/v1/driver/*`) and sync queues; they do not maintain persistent WebSocket connections for general event invalidation.

---

## 4. Conclusion

1. **Two-System Boundary**: `pegasus.x` is 100% compliant with the sovereign single-tenant doctrine (0 Spanner, 0 Kafka).
2. **Pipeline Parity Gap**: The real-time pipeline is currently fractured across three failure points:
   - **Persistence Layer**: Separated transactions in warehouse, rebate, and consignment; zero outbox in retailer, payout, returns, and supplier product management.
   - **Broker/Relay Layer**: Channel name casing mismatch between `relay.go` (lowercase) and `ws/hub.go` (uppercase), plus un-subscribed canonical streams.
   - **Client/Transport Layer**: Raw broadcast bypass in 6 backend services, string deserialization bug in `warehouse-desktop`, 404 SSE endpoint in `supplier-desktop`, and missing replay endpoints for non-supplier roles.

---

## 5. Verification Method

### 5.1 Codebase Inspection Commands
1. Verify 0 Spanner and 0 Kafka in backend:
   ```bash
   grep -riE "spanner|kafka" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
   ```
2. Verify channel casing mismatch:
   ```bash
   grep -n "PublishEvent" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/outbox/relay.go
   grep -n "Subscribe" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/ws/hub.go
   ```
3. Inspect separated transactions in warehouse service:
   ```bash
   sed -n '90,115p' /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/warehouse/service.go
   sed -n '350,366p' /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/warehouse/service.go
   ```
4. Inspect raw broadcast bypass in retailer and payout:
   ```bash
   sed -n '488,508p' /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/retailer/service.go
   sed -n '45,60p' /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/payout/service.go
   ```
5. Inspect warehouse desktop deserialization bug:
   ```bash
   sed -n '1,10p' /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/apps/warehouse-desktop/lib/fleet-ws-events.ts
   ```

### 5.2 Unit Test Execution
Execute backend test suite:
```bash
cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend && go test -v -race ./internal/outbox/... ./internal/ws/...
```

---

## 6. Recommended Fix Strategy (Milestone 3 Execution Plan)

### Task 1: Fix Channel Casing & Subscriptions in Relay & WebSocket Hub
- In `backend/internal/outbox/relay.go:200-204`:
  Publish to both lowercase and uppercase aggregate channels as well as canonical streams.
- In `backend/internal/ws/hub.go:169`:
  Subscribe to canonical topic streams (`events:payload:sealed`, `events:fleet:breakdown_reported`, `events:fleet:rescue_dispatched`, `events:doorstep:arrived`, `events:doorstep:tender_settled`, `events:order`, `events:retailer`, `events:warehouse`).

### Task 2: Standardize Envelope Broadcasting & Eliminate Raw `Broadcast([]byte)`
- Refactor `retailer`, `payout`, `returns`, `notifications`, `wmsops`, `seasonalcore` to use `s.wsHub.BroadcastEnvelope(eventType, payload)`.
- Normalize event names so both dot-notation (`order.created`) and UPPERCASE snake_case (`ORDER_CREATED`) are available in the payload.

### Task 3: Unify Transactions & Eliminate Separated-Tx Anti-Pattern
- In `backend/internal/warehouse/service.go`: Refactor `RegisterWarehouse`, `CompleteOnboarding`, and `ReconcileInboundBlind` to perform entity mutations and `outbox.Emit` within the **exact same `s.pool.RunInTx` block**.
- In `backend/internal/rebate/service.go` and `backend/internal/consignment/service.go`: Pass `tx pgx.Tx` into repository saves or execute queries within transaction closures.
- Replace all `_ = outbox.Emit(...)` with explicit error propagation (`if err := outbox.Emit(...); err != nil { return err }`).

### Task 4: Add Outbox Events to Retailer, Payout, Returns, and Supplier
- Implement transactional outbox emissions in `retailer/service.go` for POS checkout, shift lifecycle, store stock adjustments, and auto-order confirmations.
- Implement transactional outbox emissions in `payout/service.go` for batch approvals and disbursements.
- Implement transactional outbox emissions in `returns/service.go` for return inspections and restocks.
- Implement transactional outbox emissions in `supplier/repository.go` for product catalog additions and updates.

### Task 5: Client-Side WebSocket & Invalidation Bug Fixes
- In `apps/warehouse-desktop/lib/fleet-ws-events.ts`: Fix `parseWsEventType` to parse JSON when `typeof e === 'string'`.
- In `apps/supplier-desktop`: Point `useSupplierWsRefresh.ts` to authenticated `/v1/ws` WebSocket or implement `/v1/supplier/events` SSE in Go backend.
- In `backend/internal/api/router.go`: Register generic `GET /v1/events/sync` (or `/v1/events/since`) backed by `wsHub.GetEventsSince(since)` accessible across all roles.
