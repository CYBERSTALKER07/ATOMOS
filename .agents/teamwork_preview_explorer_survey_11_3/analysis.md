# Comprehensive Architectural Audit: Cross-Role Real-Time Monotonic Pipeline Parity (Requirement R3)

**Target Codebase**: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`  
**Audit Date**: 2026-09-23  
**Auditor**: Teamwork Explorer (Survey Agent 11.3)  
**Scope**: Requirement R3 — Cross-Role Real-Time Monotonic Pipeline across all 7 ecosystem roles (`Supplier`, `Warehouse Admin`, `Payloader & Picker`, `Dispatcher`, `Driver`, `Retailer Storefront`, `Finance & Auditor`), Outbox Relay, Redis 7 Streams, WebSocket Hub, and Desktop/Client Invalidation.

---

## Executive Summary

An exhaustive, line-by-line audit of `pegasus.x` was conducted across backend persistence, stream processing, WebSocket infrastructure, and client applications.

### Key Audit Findings Matrix

| Audit Dimension | Status | Key Findings |
| :--- | :---: | :--- |
| **1. Two-System Boundary** | **100% PASS** | 0 references to Google Cloud Spanner and 0 references to Apache Kafka in `pegasus.x/backend`. Dependencies strictly conform to Sovereign Core (`pgx/v5`, `go-redis/v9`, `gorilla/websocket`, `go-chi/v5`). |
| **2. Outbox Relay & Redis 7 Streams** | **PARTIAL / CRITICAL DEFECT** | Relay correctly uses `FOR UPDATE SKIP LOCKED` and publishes to Redis 7 Streams via `XADD` with aggregate root partition keys (`aggregate_id`). However, a **critical case-sensitivity defect** exists: relay publishes to lowercase Redis Pub/Sub channels (`events:order`, etc.), while WebSocket Hub subscribes to uppercase channels (`events:ORDER`, etc.), resulting in complete Pub/Sub event drops. |
| **3. WebSocket Hub & Monotonic Envelope** | **PARTIAL / HIGH RISK** | `RealtimeEnvelope` implements monotonic `seq`, dual `event_type` and `type` fields, and a 2,000-event ring buffer with `GetEventsSince`. However, **6+ backend packages bypass the envelope completely** by calling `wsHub.Broadcast([]byte)` with raw, un-sequenced bytes. Replay catch-up endpoint `/v1/supplier/sync` is restricted to suppliers only. |
| **4. Atomic Outbox Pairing (7 Roles)** | **MIXED / STRUCTURAL GAPS** | Core lifecycle flows (`order`, `pickwave`, `dock`, `dispatch`, `epod`, `cashrecon`, `creditnote`, `soliq`) pair outbox emission inside `pgx.Tx`. However: <br>1. **Separated Transactions**: `warehouse/service.go`, `rebate/service.go`, and `consignment/service.go` commit state mutations in one transaction, then emit outbox in a second detached transaction.<br>2. **Zero Outbox / Fire-and-Forget**: `retailer/service.go` (39 mutating actions), `payout/service.go`, `returns/service.go`, `wmsops/service.go`, `onboarding`, and product/topology updates in `supplier/repository.go` completely bypass outbox logging.<br>3. **Ignored Errors**: Multiple packages silently discard outbox errors (`_ = outbox.Emit`). |
| **5. Desktop / Client Invalidation** | **DEFECTS IDENTIFIED** | 1. **Warehouse Desktop Deserialization Bug**: `fleet-ws-events.ts` fails to parse JSON strings, preventing all real-time events from matching.<br>2. **Orphaned Invalidation Events**: `useNotifications.ts` dispatches `sync-invalidate`, but 0 components listen.<br>3. **Supplier Desktop Route 404**: Attempts SSE connection to `/v1/supplier/events` which does not exist.<br>4. **Naming Convention Drift**: Backend emits lowercase dot-notation (`order.created`), while frontend contracts expect UPPERCASE snake_case (`ORDER_CREATED`). |

---

## 1. Two-System Boundary Audit

### 1.1 AST & Codebase Scan
A full recursive AST and grep audit across `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend` was executed for foreign database and streaming dependencies:
- **Google Cloud Spanner**: `cloud.google.com/go/spanner`, Spanner DDL, `spanner.Client`, `spanner.ReadWriteTransaction`, `spanner.Apply`.
  - **Result**: **0 occurrences in Go source code**.
- **Apache Kafka**: `github.com/segmentio/kafka-go`, `github.com/Shopify/sarama`, `github.com/confluentinc/confluent-kafka-go`.
  - **Result**: **0 occurrences in Go source code**.

### 1.2 Dependency Manifest (`backend/go.mod`)
Inspection of `backend/go.mod` (lines 5-16) confirms sovereign technology adherence:
```go
require (
	github.com/go-chi/chi/v5 v5.3.2
	github.com/go-chi/cors v1.2.2
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/jackc/pgx/v5 v5.10.0
	github.com/redis/go-redis/v9 v9.22.0
	github.com/stretchr/testify v1.11.1
	golang.org/x/crypto v0.56.0
	golang.org/x/sync v0.22.0
)
```
- Persistence: `github.com/jackc/pgx/v5` (`*pgxpool.Pool`).
- Streaming & Cache: `github.com/redis/go-redis/v9` (Redis 7 Streams).
- WebSockets: `github.com/gorilla/websocket`.
- HTTP Routing: `github.com/go-chi/chi/v5`.

---

## 2. Outbox Relay & Redis 7 Streams Architecture

### 2.1 Polling Concurrency Control (`backend/internal/outbox/relay.go`)
Inspection of `RelayWorker.ProcessBatch` (`relay.go:121-135`):
```go
query := `
    SELECT event_id, aggregate_type, aggregate_id, event_type, payload
    FROM outbox_events
    WHERE NOT published
    ORDER BY created_at ASC
    LIMIT $1
    FOR UPDATE SKIP LOCKED
`
rows, err := tx.Query(ctx, query, w.batchSize)
```
- **Observation**: The query uses `FOR UPDATE SKIP LOCKED` and orders by `created_at ASC`.
- **Verdict**: Complies with high-concurrency multi-worker deployments. Competing relay instances will not deadlock or duplicate event processing.

### 2.2 Redis 7 Streams Publishing (`XADD`) with Aggregate Root Partitioning
In `relay.go:163-188`:
```go
canonicalStream, aggregateStream := ResolveStreamKey(it.aggregateType, it.eventType)

values := map[string]any{
    "event_id":       it.eventID.String(),
    "aggregate_id":   it.aggregateID, // Partition key
    "aggregate_type": it.aggregateType,
    "event_type":     it.eventType,
    "payload":        string(it.payload),
    "created_at":     time.Now().UTC().Format(time.RFC3339Nano),
}

// 1. Persistent stream publication to Canonical Redis 7 Stream (XADD)
_, err := w.redis.XAdd(ctx, &goredis.XAddArgs{
    Stream: canonicalStream,
    MaxLen: 100000,
    Approx: true,
    Values: values,
}).Result()
```
- **Observation**:
  - `aggregate_id` is explicitly preserved as the partition key.
  - Streams are capped at `MaxLen: 100000` with `Approx: true` to prevent unbounded memory growth.
  - Failures in `XAdd` log to `outbox_dead_letters` table (`relay.go:191-197`).
  - Successful publish updates `outbox_events SET published = TRUE, published_at = NOW() WHERE event_id = $1` (`relay.go:207-214`).

### 2.3 Critical Defect: Redis Pub/Sub Channel Casing Mismatch
In `relay.go:199-204`:
```go
// 3. Ephemeral Pub/Sub notification for live WebSocket hub fanout
_ = w.redis.PublishEvent(ctx, canonicalStream, string(it.payload))
if aggregateChannel := fmt.Sprintf("events:%s", strings.ToLower(it.aggregateType)); aggregateChannel != canonicalStream {
    _ = w.redis.PublishEvent(ctx, aggregateChannel, string(it.payload))
}
```
And in `backend/internal/ws/hub.go:169`:
```go
pubsub := h.redis.Subscribe(ctx,
    "telemetry:drivers",
    "events:ORDER",
    "events:UMP",
    "events:FLEET",
    "events:CLAIMS",
    "events:PICKWAVE",
    "events:MANIFEST",
    "events:EPOD",
    "events:WAREHOUSE",
    "events:notifications",
    "alerts:fleet:breakdown_rescue",
)
```
- **Root Cause**:
  1. Redis Pub/Sub channels are strictly case-sensitive binary strings. `events:order` != `events:ORDER`.
  2. `relay.go` publishes to lowercase channels (`fmt.Sprintf("events:%s", strings.ToLower(it.aggregateType))`), producing `events:order`, `events:fleet`, `events:manifest`, `events:epod`.
  3. `ws/hub.go` subscribes to UPPERCASE channels (`events:ORDER`, `events:FLEET`, `events:MANIFEST`, `events:EPOD`).
  4. Furthermore, `ResolveStreamKey` assigns canonical streams such as `events:payload:sealed`, `events:fleet:breakdown_reported`, `events:doorstep:arrived`, `events:doorstep:tender_settled`. **The WebSocket Hub does not subscribe to any of these canonical streams**.
- **Impact**: All events processed by the Outbox Relay are dropped by Redis Pub/Sub before reaching the WebSocket Hub. Real-time client listeners never receive outbox-relayed events.

---

## 3. WebSocket Hub & Monotonic Envelope Architecture

### 3.1 Envelope Schema & Monotonic Sequence Numbering (`backend/internal/ws/hub.go`)
Inspection of `ws/hub.go:31-38`:
```go
type RealtimeEnvelope struct {
	Seq       int64                  `json:"seq"`
	EventType string                 `json:"event_type"`
	Type      string                 `json:"type"` // Backwards-compatible alias for client event listeners
	Payload   map[string]interface{} `json:"payload"`
	Timestamp int64                  `json:"timestamp"`
}
```
In `BroadcastEnvelope` (`ws/hub.go:111-133`):
```go
func (h *Hub) BroadcastEnvelope(eventType string, payload map[string]interface{}) RealtimeEnvelope {
	seq := atomic.AddInt64(&h.seq, 1)
	env := RealtimeEnvelope{
		Seq:       seq,
		EventType: eventType,
		Type:      eventType,
		Payload:   payload,
		Timestamp: time.Now().Unix(),
	}

	h.recentMu.Lock()
	if len(h.recentEvents) >= h.maxHistory {
		h.recentEvents = h.recentEvents[1:]
	}
	h.recentEvents = append(h.recentEvents, env)
	h.recentMu.Unlock()

	data, err := json.Marshal(env)
	if err == nil {
		h.Broadcast(data)
	}
	return env
}
```
- **Observation**:
  - Sequence numbers increment monotonically via `atomic.AddInt64`.
  - Dual fields `event_type` and `type` are populated with identical values for client compatibility.
  - History buffer retains up to 2,000 events (`maxHistory: 2000`).
  - `GetEventsSince(since int64)` (`ws/hub.go:137-162`) returns buffered envelopes or signals `fullResync: true` if `since < oldestSeq - 1`.

### 3.2 Critical Defect: Raw Broadcast Bypass across Domain Services
Multiple domain packages bypass `BroadcastEnvelope` and directly invoke `s.wsHub.Broadcast(rawBytes)`:
1. `backend/internal/retailer/service.go:501`: In `broadcastEvent`, serializes `{ event_id, event_type, timestamp, payload }` and calls `s.wsHub.Broadcast(data)`. Does NOT assign `seq`.
2. `backend/internal/payout/service.go:58`: In `broadcastEvent`, serializes `{ type, payload, timestamp }` and calls `s.wsHub.Broadcast(data)`. Does NOT assign `seq`.
3. `backend/internal/returns/service.go:180`: In `broadcastEvent`, serializes `{ type, data, timestamp }` and calls `s.wsHub.Broadcast(payload)`. Does NOT assign `seq`.
4. `backend/internal/notifications/service.go:180`: Direct `s.wsHub.Broadcast(payload)`.
5. `backend/internal/wmsops/service.go:122`: Direct `s.wsHub.Broadcast(eventPayload)`.
6. `backend/internal/seasonalcore/service.go:83, 148`: Direct `s.wsHub.Broadcast(msg)`.

**Impact**: Clients receive malformed frames lacking the `seq` counter, causing sequence gap detectors to miscalculate or fail, and bypassing the 2,000-event replay buffer.

### 3.3 Role Asymmetry in Replay Route Exposure
In `backend/internal/api/router.go:414` and `handlers_supplier.go:1076`:
- `GET /v1/supplier/sync?since={seq}` is registered for suppliers.
- **Defect**: No corresponding `/v1/events/sync`, `/v1/retailer/sync`, `/v1/warehouse/sync`, or `/v1/driver/sync` endpoints exist. Non-supplier roles have no server endpoint to fetch missed envelopes upon reconnection.

---

## 4. Atomic Outbox Pairing Audit across All 7 Roles

### Role 1: Supplier
- **Onboarding Status Update**: `backend/internal/supplier/repository.go:1390-1416`
  - In `UpdateOnboardingStatus`: updates `suppliers.onboarding_status` and invokes `outbox.Emit(ctx, tx, "SUPPLIER", supplierID, "supplier.onboarding_completed", eventPayload)` inside `p.pool.RunInTx(ctx, func(tx pgx.Tx) error { ... })`. **VERIFIED ATOMIC**.
- **Product Catalog Mutations**: `backend/internal/supplier/repository.go:1488-1630`
  - `CreateProduct`, `UpdateProduct`, `DeleteProduct` execute SQL queries directly on `p.pool` without transactions and **emit zero outbox events**.
- **Supplier Topology & Pricing**: `backend/internal/supplier/repository.go:283-620`
  - `SaveTopologyNode`, `SaveOrgMember`, `SavePricingRule`, `SaveRetailerPricingOverride` execute on `p.pool` with **zero outbox events**.
- **Dock Fleet Attachments**: `backend/internal/supplier/repository.go:1634-1750`
  - `SaveWarehouseTruck` and `SaveWarehousePayloader` execute SQL queries directly with **zero outbox events**.

### Role 2: Warehouse Admin
- **WMS Waves & Lots**: `backend/internal/wms/waves.go:180, 281, 360` and `backend/internal/wms/lots.go:137`
  - `wms.wave_generated`, `wms.task_picked`, `wms.wave_completed`, and `stock.lot_putaway` are emitted inside `RunInTx(ctx, func(tx pgx.Tx) error { ... })`. **VERIFIED ATOMIC**.
- **Warehouse Registration & Onboarding (Separated-Transaction Anti-Pattern)**: `backend/internal/warehouse/service.go:94-108, 352-365`
  - `RegisterWarehouse`: Line 94 calls `s.repo.CreateWarehouse(ctx, wh)` (independent transaction), then lines 99-108 runs a second detached transaction `s.pool.RunInTx` to emit `warehouse.registered`.
  - `CompleteOnboarding`: Line 352 calls `s.repo.UpdateOnboardingStatus(ctx, warehouseID, "COMPLETED")`, then lines 356-365 runs a second detached transaction `s.pool.RunInTx` to emit `warehouse.onboarding.completed`.
  - **Verdict**: Violates atomic pairing. If the process crashes between lines 94 and 99, entity mutation is committed but the event is permanently lost.
- **Blind Receiving Variance Reconciliation (Separated Transactions)**: `backend/internal/warehouse/service.go:592-629`
  - Shortage claims saved on line 592, outbox emitted in separate transaction on lines 598-611; blind PO reconciliation outbox emitted in a third transaction on lines 618-629.
- **Warehouse Operations Radar (Zero Outbox)**: `backend/internal/wmsops/service.go:110-136`
  - `SendBroadcast` executes `s.wsHub.Broadcast` and `s.rdb.Publish("events:WAREHOUSE", ...)` directly with **zero outbox logging**.

### Role 3: Payloader & Picker
- **Pick Waves & Sealing**: `backend/internal/pickwave/repository.go:455, 506, 635, 664, 732, 756, 812, 833`
  - `CreatePickWaveTx`, `AssignPickerTx`, `CompletePickTaskTx`, `WaiveShortsTx`, and `SealPickWaveTx` use explicit transaction boundaries (`tx.Begin(ctx)` -> mutations -> `outbox.Emit` -> `tx.Commit(ctx)`). Both `pickwave.sealed` and `manifest.sealed` are committed in the same transaction. **VERIFIED ATOMIC**.
- **Dock Bay Operations**: `backend/internal/dock/repository.go:208, 247, 303, 361`
  - `AssignBay`, `UpdateLoadingProgress`, `SealBay`, and `DepartBay` pair state updates with `DOCK_BAY_ASSIGNED`, `DOCK_PROGRESS_UPDATED`, `DOCK_BAY_SEALED`, and `DOCK_BAY_DEPARTED` within `tx.Begin(ctx)` ... `tx.Commit(ctx)`. **VERIFIED ATOMIC**.
- **Client Real-Time Deficiency**: `apps/payloader-tablet/App.tsx` has no WebSocket connection and performs only one-shot HTTP fetch on mount.

### Role 4: Dispatcher
- **Commit Dispatch Plan**: `backend/internal/dispatch/service.go:433-448`
  - Inside `s.pool.RunInTx`: updates manifests, assigns drivers, updates orders, releases dispatch lock, and emits `outbox.Emit(ctx, tx, "MANIFEST", req.WarehouseID, "DISPATCH_PLAN_COMMITTED", eventPayload)`. **VERIFIED ATOMIC**.
- **Fleet Breakdown Rescue Hot-Swap**: `backend/internal/dispatch/service.go:904-923`
  - Inside `s.pool.RunInTx`: reallocates stops, generates rescue manifest, updates original manifest and incident records, and emits `outbox.Emit(ctx, tx, "DRIVER", req.BrokenDriverID, "FLEET_BREAKDOWN_RESCUED", rescuePayload)`. **VERIFIED ATOMIC**.
- **Fleet Vehicle & Driver Lifecycle**: `backend/internal/fleet/repository.go:273, 426, 485, 781, 843, 1023, 1259, 1430`
  - `fleet.vehicle.created`, `fleet.vehicle.availability_changed`, `fleet.driver.created`, `fleet.assignment.created`, `fleet.assignment.released`, `fleet.vehicle.swapped`, `fleet.driver.swapped`, and `fleet.safety_failure` are all emitted inside `r.pool.RunInTx`. **VERIFIED ATOMIC**.

### Role 5: Driver
- **Doorstep Proximity & Arrival Handshake**: `backend/internal/epod/repository.go:375` and `backend/internal/epod/service.go:80-94`
  - `MarkStopArrivedTx` updates stop status to `ARRIVED` and emits `outbox.Emit(ctx, tx, "MANIFEST", stop.ManifestID, "manifest.stop_arrived", stop)` inside `tx.Begin(ctx)` ... `tx.Commit(ctx)`. **VERIFIED ATOMIC**.
- **Electronic Proof of Delivery (ePoD)**: `backend/internal/epod/repository.go:473, 482`
  - In `ConfirmDeliveryTx`: updates stop to `DELIVERED`, records signature and GPS coordinates, emits `epod.confirmed`, and if all stops are done, emits `manifest.completed` within the same transaction. **VERIFIED ATOMIC**.
- **Offline Sync Drain Queue**: `backend/internal/epod/repository.go:672`
  - `ProcessOfflineQueueBatchTx` updates sync status and emits `epod.offline.synced` inside `tx.Begin(ctx)` ... `tx.Commit(ctx)`. **VERIFIED ATOMIC**.
- **Doorstep Damaged Carton Discrepancy (UMP)**: `backend/internal/ump/engine.go:185-274`
  - In `ProcessAdjustment`: updates order effective total, updates delivered quantities, quarantines damaged stock, creates Credit Note, prepares Tuzatuvchi Faktura, and emits `outbox.Emit(ctx, tx, "UMP", ...)` inside `e.pool.RunInTx`. **VERIFIED ATOMIC**.

### Role 6: Retailer Storefront
- **Retailer OS Service (Zero Outbox / Fire-and-Forget Anti-Pattern)**: `backend/internal/retailer/service.go:49-1311`
  - **39 mutating state operations** in `retailer/service.go` call `s.broadcastEvent(...)` directly:
    * POS Cash Register & Shift: `retailer.register.created`, `retailer.shift.opened`, `retailer.shift.closed`, `retailer.shift.cash_drop`.
    * POS Checkout & Returns: `retailer.pos.sale_completed`, `retailer.pos.sale_voided`, `retailer.pos.refunded`.
    * Store Inventory: `retailer.stock.count_committed`, `retailer.stock.receive_confirmed`, `retailer.stock.transferred`, `retailer.stock.adjusted`.
    * Auto-Ordering Proposals: `retailer.auto_order.proposed`, `retailer.auto_order.confirmed`, `retailer.auto_order.rejected`.
    * Store Identity & Team: `retailer.auth.registered`, `retailer.team.member_added`, `retailer.team.member_updated`, `retailer.team.member_deleted`.
  - **Verdict**: None of these 39 operations write to `outbox_events`. They emit ephemeral fire-and-forget Redis Pub/Sub messages on channel `events:RETAILER` (which is not subscribed by `ws/hub.go`) and send raw un-sequenced bytes to `wsHub.Broadcast`.
- **Order Lifecycle (Retailer -> Supplier)**: `backend/internal/order/service.go:501-1376`
  - `order.created`, `order.draft_created`, `order.auto_approved`, `order.pending_approval`, `order.confirmed`, `order.draft_confirmed`, `order.status_updated`, `order.vetting_approved`, `order.vetting_rejected` are paired inside `s.pool.RunInTx`. **VERIFIED ATOMIC**.

### Role 7: Finance & Auditor
- **Driver Cash-in-Transit (CIT) Drawer Tracking**: `backend/internal/cashrecon/cit_drawer.go:260, 416, 590`
  - `IncrementDriverCashDrawer`: emits `cit.threshold_exceeded` inside `s.pool.RunInTx`. **VERIFIED ATOMIC**.
  - `RecordMidShiftVaultDrop`: decrements driver drawer, writes double-entry General Ledger journal entry and postings, and emits `cit.mid_shift_vault_dropped` inside `s.pool.RunInTx`. **VERIFIED ATOMIC**.
  - `RecordEndOfShiftBankDeposit`: writes double-entry General Ledger journal entry and postings, and emits `cit.end_of_shift_deposited` inside `s.pool.RunInTx`. **VERIFIED ATOMIC**.
- **Credit Notes**: `backend/internal/creditnote/service.go:168`
  - Inserts credit note and credit note lines, and emits `outbox.Emit(ctx, tx, "CREDIT_NOTE", cn.CreditNoteId, "credit_note.created", ...)` inside `s.pool.RunInTx`. **VERIFIED ATOMIC**.
- **Soliq OFD Receipts & E-Factura**: `backend/internal/soliq/receipt.go:122` and `backend/internal/soliq/service.go:211, 346`
  - `soliq.fiscal_receipt.issued`, `soliq.efactura.registered`, and `soliq.efactura.corrective_registered` are emitted inside `RunInTx`. **VERIFIED ATOMIC**.
- **Doorstep Payment Handover & Gateway Webhooks**: `backend/internal/api/handlers_payment.go:92, 133, 265, 346`
  - `debt.created`, `delivery.handover_completed`, `payment.cleared`, and `payment.reconciled` are emitted inside `s.pool.RunInTx`. **VERIFIED ATOMIC**.
- **Supplier Payout Engine (Zero Outbox)**: `backend/internal/payout/service.go:47-241`
  - `payout.policy_updated`, `payout.batch_created`, `payout.batch_approved`, `payout.batch_disbursed` call `s.broadcastEvent` directly with **zero outbox logging**.
- **Trade Spend Rebate Contracts (Separated Transactions)**: `backend/internal/rebate/service.go:60-116`
  - Accruals and contract updates saved first, then outbox emitted in separate transactions.
- **Consignment Stock (Separated Transactions)**: `backend/internal/consignment/service.go:140-197`
  - Agreement and voucher updates saved first, then outbox emitted in separate transactions.

### 4.8 Ignored Error Anti-Pattern (`_ = outbox.Emit`)
The audit identified multiple locations where developers used blank identifiers to discard errors returned by `outbox.Emit`:
- `backend/internal/ump/engine.go:257, 267`: Auto-applied and pending approval adjustments discard outbox error.
- `backend/internal/inventory/service.go:116, 158, 209, 371`: Stock reservations, releases, and quarantines discard outbox error.
- `backend/internal/order/service.go:501, 587, 594, 1357`: Order auto-approval and pending approval events discard outbox error.
- `backend/internal/claims/repository.go:663`: Credit note creation event inside claim adjudication discards outbox error.

**Impact**: If the outbox table query fails (e.g. disk full, lock timeout, constraint violation), the database transaction commits the entity mutation while the event silently fails to persist, breaking pipeline integrity.

---

## 5. Desktop & Client Real-Time Invalidation Audit

### 5.1 Retailer Desktop (`apps/retailer-desktop`)
- **WebSocket Provider (`lib/ws.tsx:78-86`)**:
  ```typescript
  const msg = JSON.parse(event.data) as WsMessage;
  const type = typeof msg.type === 'string' 
    ? msg.type 
    : (typeof (msg as any).event_type === 'string' ? (msg as any).event_type : undefined);
  setLastMessage(msg);
  if (type) {
    listenersRef.current.get(type)?.forEach(h => h(msg));
  }
  ```
  - **Verdict**: Dual-compatibility between `msg.type` and `msg.event_type` is correctly implemented.
  - **State Invalidation**: `SessionReconcileListener` (`lib/session-reconcile-listener.tsx`) listens to `reconnectEpoch` and triggers `reconcileRetailerSession()` upon reconnection without full reload.
  - **Defect**: Does not inspect `msg.seq` or track sequence continuity; cannot request gap replay.

### 5.2 Warehouse Desktop (`apps/warehouse-desktop`)
- **Fatal Deserialization Bug (`lib/fleet-ws-events.ts:5`)**:
  ```typescript
  export const parseWsEventType = (e: any) => (typeof e === 'string' ? e : e?.type || '');
  ```
  - **Root Cause**: `subscribeWarehouseWS` invokes `onMessage(String(event.data))` passing the raw WebSocket message string (e.g. `'{"seq":1,"type":"ORDER_CREATED"}'`).
  - Because `typeof e === 'string'`, `parseWsEventType` returns the **entire raw JSON string** instead of the parsed `type` field!
  - In `use-warehouse-ws-refresh.ts:39-44`:
    `eventTypes.has(eventType)` evaluates `WAREHOUSE_ORDERS_REFRESH_EVENTS.has('{"seq":1,"type":"ORDER_CREATED"}')` -> **ALWAYS FALSE**.
  - In `use-warehouse-fleet-live-map.ts:53`:
    `WAREHOUSE_LOCATION_PATCH_EVENTS.has(eventType)` evaluates `has('{"seq":1,...}')` -> **ALWAYS FALSE**.
  - **Impact**: Real-time WebSocket invalidation in `warehouse-desktop` is completely dead. The application silently falls back to background HTTP polling (`usePolling`).
- **Orphaned Invalidation Events (`lib/useNotifications.ts:223-233`)**:
  - `useNotifications.ts` dispatches `window.dispatchEvent(new CustomEvent("sync-invalidate", { detail: eventType }))`.
  - Search across `apps/warehouse-desktop` reveals **0 event listeners for `sync-invalidate`**. The invalidation signal is orphaned.

### 5.3 Supplier Desktop (`apps/supplier-desktop`)
- **Non-Existent SSE Endpoint (`lib/use-supplier-ws-refresh.ts:17-74`)**:
  - `useSupplierWsRefresh` instantiates `new EventSource(`${apiBase}/v1/supplier/events`)`.
  - Route audit of `backend/internal/api/router.go` confirms **/v1/supplier/events does NOT exist**.
  - **Impact**: Connection to SSE endpoint fails with HTTP 404. Real-time updates in supplier-desktop do not function.

### 5.4 Cross-Tier Event Casing Drift
- Backend outbox emits lowercase dot-notation events: `manifest.sealed`, `order.created`, `payment.cleared`, `delivery.driver.arrived`.
- Shared refresh contracts in `packages/ws-refresh-contract/index.ts` expect UPPERCASE snake_case: `MANIFEST_SEALED`, `ORDER_CREATED`, `PAYMENT_CLEARED`, `DRIVER_ARRIVED`.
- Neither the WebSocket Hub nor the client libraries normalize event names, leading to silent drops when clients check `eventTypes.has(eventType)`.

---

## 6. Recommended Fix Strategy for Milestone 3

To achieve 100% enterprise rigor and cross-role monotonic pipeline parity in Milestone 3, execute the following surgical remediation plan:

### Step 1: Fix Outbox Relay & Redis Channel Casing
1. In `backend/internal/outbox/relay.go`:
   - Emit Pub/Sub notifications to both canonical streams and uppercase/lowercase aggregate channels:
     ```go
     _ = w.redis.PublishEvent(ctx, canonicalStream, string(it.payload))
     _ = w.redis.PublishEvent(ctx, fmt.Sprintf("events:%s", strings.ToLower(it.aggregateType)), string(it.payload))
     _ = w.redis.PublishEvent(ctx, fmt.Sprintf("events:%s", strings.ToUpper(it.aggregateType)), string(it.payload))
     ```
2. In `backend/internal/ws/hub.go`:
   - Subscribe to canonical stream names (`events:payload:sealed`, `events:fleet:breakdown_reported`, `events:fleet:rescue_dispatched`, `events:doorstep:arrived`, `events:doorstep:tender_settled`) in addition to aggregate channels.

### Step 2: Enforce Envelope Normalization & Eliminate Raw Broadcasts
1. In `backend/internal/ws/hub.go`:
   - Standardize `BroadcastEnvelope` to normalize `eventType` to both forms (dot-notation and UPPERCASE snake_case) or provide dual matching.
2. In `retailer`, `payout`, `returns`, `notifications`, `wmsops`, `seasonalcore`:
   - Refactor `s.wsHub.Broadcast(...)` to `s.wsHub.BroadcastEnvelope(...)`, ensuring every frame includes monotonic `seq`.

### Step 3: Eliminate Separated Transactions & Purge Ignored Errors
1. In `backend/internal/warehouse/service.go`:
   - Wrap `CreateWarehouse` + `outbox.Emit` into a single `s.pool.RunInTx` closure.
   - Wrap `UpdateOnboardingStatus` + `outbox.Emit` into a single `s.pool.RunInTx` closure.
   - Wrap `SaveShortageClaim` + `outbox.Emit` into a single transaction.
2. In `backend/internal/rebate/service.go` and `backend/internal/consignment/service.go`:
   - Refactor repositories to accept `tx pgx.Tx` or execute contract mutations and `outbox.Emit` in the same `RunInTx` block.
3. In `ump`, `inventory`, `order`, `claims`:
   - Replace all `_ = outbox.Emit(...)` with `if err := outbox.Emit(...); err != nil { return err }`.

### Step 4: Add Missing Outbox Emissions in Retailer, Payout, Returns, and Supplier
1. In `backend/internal/retailer/service.go`:
   - Wire transactional outbox emissions for critical mutations (`retailer.pos.sale_completed`, `retailer.shift.opened`, `retailer.shift.closed`, `retailer.stock.count_committed`, `retailer.auto_order.confirmed`).
2. In `backend/internal/payout/service.go`:
   - Add transactional outbox emission on `payout.batch_approved` and `payout.batch_disbursed`.
3. In `backend/internal/returns/service.go`:
   - Add transactional outbox emission on `returns.inspected` and `returns.restocked`.

### Step 5: Fix Client-Side Deserialization & Endpoint Mappings
1. In `apps/warehouse-desktop/lib/fleet-ws-events.ts`:
   - Fix `parseWsEventType` to parse JSON when `typeof e === 'string'` (using `JSON.parse(e).type || JSON.parse(e).event_type`).
2. In `apps/supplier-desktop`:
   - Align `useSupplierWsRefresh.ts` to connect to the authenticated WebSocket `/v1/ws` or implement `/v1/supplier/events` SSE handler in `backend/internal/api/router.go`.
3. In `backend/internal/api/router.go`:
   - Expose generic `GET /v1/events/sync` (or `/v1/events/since`) backed by `wsHub.GetEventsSince(since)` accessible across all roles.
