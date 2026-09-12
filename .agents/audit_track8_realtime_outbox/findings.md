# Track 8 Comprehensive Audit Report: Realtime Engine, Outbox Pattern, Kafka & Multi-Hub WebSocket Fanout

**Auditor:** Codebase Explorer (Track 8 Realtime/Outbox/Kafka/WS Specialist)  
**Target Codebase:** `pegasusX/apps/backend-go`  
**Date:** 2026-08-30  
**Audit Scope:**
1. Transactional Outbox pattern & poller/relay (`outbox/`, `schema/spanner.ddl`, `bootstrap/outbox_runtime.go`)
2. Kafka consumer groups, worker pool, partitioning, deduplication, & DLQ (`kafka/`, `kafkautil/`, `events/`)
3. WebSocket Hub architecture, SSE, multi-hub routing, connection shedding, heartbeat, & backpressure (`ws/`, `platform/`)
4. Data consistency, transaction boundaries, FIFO message ordering, and reconnection replay

---

## Executive Summary

The PegasusX backend implements an ambitious, event-driven real-time backbone composed of a Transactional Outbox in Spanner, high-throughput Kafka event streaming with partition-parallel worker pools, and an 8-hub WebSocket multiplexer with cross-pod Redis Pub/Sub synchronization and Desert Protocol connection shedding.

However, line-by-line static analysis and dynamic test execution revealed **several critical defects, race conditions, and architectural flaws**:
1. **Multi-User Retailer Outbox Deafness (`ws/handler.go:161-172`):** Multi-user retailer staff connections subscribe to `"retailer:<user_id>"` instead of `"retailer:<org_id>"`, causing all order, payment, and delivery real-time WebSocket events to be completely missed by retailer desktop/mobile apps.
2. **Monotonic Kafka Offset Commit Data Loss (`kafka/workerpool/workerpool.go:137-158`):** When a message fails DLQ routing and returns `ErrSkipCommit`, processing the next message on the same partition worker commits the higher offset, permanently acknowledging and dropping the failed message in Kafka.
3. **Partition Key Fragmentation Breaking FIFO Order (`outbox/relay.go:280-312`):** `granularRoutingKey` dynamically alters Kafka partition keys by appending sub-entity IDs, causing different events for the same aggregate to hash to different Kafka partitions and breaking per-entity chronological ordering.
4. **Synchronous Fan-out Head-of-Line Blocking (`ws/hub.go:235-266`):** Hub broadcasts iterate through connections synchronously with a 5-second write deadline. A slow client blocks the broadcast caller (including Kafka consumers) for seconds.
5. **Dual-Write Topic Poisoning (`outbox/relay.go:220-261`):** In multi-topic publish mode, partial publish failures trigger retries from topic 0, causing duplicate message storms on primary topics and erroneous dead-lettering.
6. **Outbox Poller Contention & Tenant Starvation (`outbox/spanner_store.go:114-190`):** Fetch runs inside a Spanner ReadWriteTransaction with index range locks, causing transaction abort storms across multi-replica relays and HoL blocking by bulk tenants.
7. **Ghost Broadcasts to Unsubscribed Rooms (`payload/progress.go:57`, `driver/live_tracking.go:56`):** Direct calls to `Hub.Broadcast` target ad-hoc unnamespaced rooms (`"warehouse_ops"`, `"fleet_map"`, `"fleet_broadcast"`, `"SHELF_ALERT"`) that no client ever joins.
8. **Broken Build / Test Suite (`ws/hub_test.go:119`, `order/service.go:1321`):** Test mock `publishFailBackend` fails to implement `cache.Backend` interface, and `order/service.go` references undefined `StatusDraft`.

---

## Detailed Audit Findings

### 1. [CRITICAL] Multi-User Retailer WebSocket Subscription Broken (Deaf to Real-time Events)
- **File & Lines:** `apps/backend-go/ws/handler.go:161-172`, `apps/backend-go/ws/sse.go:136-140`
- **Component:** WebSocket Hub / SSE Upgrade Handler
- **Flaw Description:**
  In `subscribeRetailerRooms`, the connection is subscribed to `"retailer:" + ident.Subject`:
  ```go
  func subscribeRetailerRooms(ident auth.Claims, conn Connection, hubs roleHubs, unsubscribes []func(), cfg RegisterConfig) []func() {
      if hubs.retailer != nil && ident.Subject != "" {
          unsubscribes = append(unsubscribes, hubs.retailer.Subscribe("retailer:"+ident.Subject, conn))
          ...
      }
      return unsubscribes
  }
  ```
  In Retail OS Phase 1 multi-user organizations, a staff member's JWT `ident.Subject` is their personal User ID (e.g. `user-uuid-1234`), while their organization tenant ID is `ident.RetailerOrgID` (e.g. `ret-org-5678`).
  Meanwhile, `NotificationDispatcher` (`kafka/notification_dispatcher.go:938`) broadcasts domain events exclusively to `"retailer:" + retailerID` (which is always the organization ID `ret-org-5678`).
  Because staff connections are registered under room `"retailer:user-uuid-1234"`, they NEVER match room `"retailer:ret-org-5678"`.
  Additionally, `cfg.RetailerPromoSuppliers` is invoked with `ident.Subject` instead of `auth.ResolveRetailerOrgID(ident)`, breaking promo room attachments.
- **Blast Radius:** All Retailer Desktop (Tauri), Android (Compose), and iOS (SwiftUI) applications logged in as staff (Manager, Buyer, Cashier) receive zero real-time WebSocket/SSE updates for orders, payments, deliveries, disputes, and promotions.
- **Recommendation:**
  Update `subscribeRetailerRooms` to resolve the organization ID via `auth.ResolveRetailerOrgID(ident)`:
  ```go
  func subscribeRetailerRooms(ident auth.Claims, conn Connection, hubs roleHubs, unsubscribes []func(), cfg RegisterConfig) []func() {
      if hubs.retailer == nil {
          return unsubscribes
      }
      orgID := auth.ResolveRetailerOrgID(ident)
      if orgID != "" {
          unsubscribes = append(unsubscribes, hubs.retailer.Subscribe("retailer:"+orgID, conn))
          if cfg.RetailerPromoSuppliers != nil {
              for _, supplierID := range cfg.RetailerPromoSuppliers(context.Background(), orgID) {
                  room := SupplierPromoRoom(supplierID)
                  unsubscribes = append(unsubscribes, hubs.retailer.Subscribe(room, conn))
              }
          }
      }
      // Also subscribe to individual user room for direct user notifications
      if ident.Subject != "" && ident.Subject != orgID {
          unsubscribes = append(unsubscribes, hubs.retailer.Subscribe("retailer:user:"+ident.Subject, conn))
      }
      return unsubscribes
  }
  ```

---

### 2. [CRITICAL] Kafka Workerpool Offset Commit Skips Invalidate on Subsequent Message Commit
- **File & Lines:** `apps/backend-go/kafka/workerpool/workerpool.go:137-158`, `apps/backend-go/kafka/consumer.go:151-161`
- **Component:** Kafka Consumer Worker Pool
- **Flaw Description:**
  In `workerpool.go:137-158`:
  ```go
  func (p *Pool) runWorker(parent context.Context, in <-chan kafka.Message, wg *sync.WaitGroup) {
      defer wg.Done()
      for m := range in {
          ctx := ContextWithTrace(parent, m)
          err := p.cfg.Handler(ctx, m)
          if err != nil && !errors.Is(err, ErrSkipCommit) { ... }
          if errors.Is(err, ErrSkipCommit) {
              continue
          }
          ...
          if cerr := p.cfg.Source.CommitMessages(commitCtx, m); cerr != nil { ... }
      }
  }
  ```
  When a message `m` (e.g. Partition 0, Offset 10) exhausts retries and DLQ writing fails (`sendToDLQ` returns error), `dispatch` returns `workerpool.ErrSkipCommit`.
  The worker executes `continue`, skipping `CommitMessages` for Offset 10.
  However, the worker channel `in` immediately processes the NEXT message `m+1` (Partition 0, Offset 11).
  When message `m+1` succeeds, `CommitMessages` is called for Offset 11.
  In Apache Kafka protocol, offset commits are **cumulative and monotonic**. Committing Offset 11 tells Kafka that all offsets up to 11 have been successfully processed.
  Therefore, skipping commit on message 10 is completely nullified the instant message 11 is committed!
  The un-DLQ'd, un-processed message 10 is permanently acknowledged and permanently lost from the consumer stream.
- **Blast Radius:** Any temporary DLQ outage or broker error causes unrecoverable loss of business-critical state change events (e.g. order transitions, payments, inventory locks).
- **Recommendation:**
  When `ErrSkipCommit` occurs on a partition, the worker must pause or halt message consumption on that partition channel, and signal the main fetch loop or trigger a circuit breaker/backoff rather than continuing to consume subsequent messages on the same partition.

---

### 3. [CRITICAL] Kafka Granular Routing Key Fragmenting Per-Entity Event Order
- **File & Lines:** `apps/backend-go/outbox/relay.go:280-312`, `apps/backend-go/outbox/kafka_publisher.go:88`
- **Component:** Outbox Relay & Kafka Publisher
- **Flaw Description:**
  In `relay.go:280-312`:
  ```go
  func granularRoutingKey(e Event) []byte {
      key := []byte(e.AggregateID)
      ...
      var envelope struct {
          OrderID    string `json:"order_id"`
          ManifestID string `json:"manifest_id"`
          RouteID    string `json:"route_id"`
          DriverID   string `json:"driver_id"`
      }
      if err := json.Unmarshal(e.Payload, &envelope); err == nil {
          if envelope.OrderID != "" {
              return append(key, []byte(":"+envelope.OrderID)...)
          }
          if envelope.ManifestID != "" {
              return append(key, []byte(":"+envelope.ManifestID)...)
          }
          ...
      }
      return key
  }
  ```
  `KafkaPublisher` is configured with `Balancer: &kafka.Hash{}`.
  Kafka hash partitioning guarantees that messages with the *exact same key* land on the *same partition*, preserving FIFO ordering.
  By appending sub-entity strings (`":order-123"`), events for the SAME aggregate root (`AggregateID: "ret-1"`) get different keys:
  - `RETAILER_CREATED` -> Key: `ret-1` -> Partition 2
  - `ORDER_CREATED` -> Key: `ret-1:order-99` -> Partition 5
  - `RETAILER_UPDATED` -> Key: `ret-1` -> Partition 2
  Because Partition 2 and Partition 5 are consumed by different consumer threads, `ORDER_CREATED` can be processed before `RETAILER_CREATED`, causing foreign key lookup failures and out-of-order state transitions.
- **Blast Radius:** Race conditions in all downstream event consumers (e.g. `order-mutator`, `digital-twin`, `warehouse-mutator`) where dependent events arrive before parent aggregate state.
- **Recommendation:**
  Remove dynamic suffix concatenation in `granularRoutingKey` or ensure routing keys strictly represent the partition affinity boundary (e.g. always `e.AggregateID` or a deterministic tenant partition key).

---

### 4. [HIGH] Synchronous Fan-out Head-of-Line Blocking in WebSocket Hub
- **File & Lines:** `apps/backend-go/ws/hub.go:235-266`, `apps/backend-go/ws/connection.go:108-117`
- **Component:** WebSocket Hub Broadcast Engine
- **Flaw Description:**
  In `ws/hub.go:235-266`:
  ```go
  func (h *Hub) fanoutLocal(ctx context.Context, room string, payload []byte) {
      h.mu.RLock()
      conns := make([]Connection, 0, len(h.rooms[room]))
      for _, c := range h.rooms[room] {
          conns = append(conns, c)
      }
      h.mu.RUnlock()

      var dead []string
      for _, c := range conns {
          writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
          err := c.Send(writeCtx, payload)
          cancel()
          if err != nil {
              dead = append(dead, c.ID())
          }
      }
      ...
  }
  ```
  `fanoutLocal` iterates through all room connections synchronously.
  `gorillaConn.Send` performs `c.conn.WriteMessage(websocket.TextMessage, payload)` under a 5-second timeout.
  If a mobile driver or warehouse portal client is experiencing network degradation (e.g. high TCP retransmissions / packet loss), `c.Send` blocks for up to 5 seconds.
  If 3 clients in a room stall, `fanoutLocal` blocks the caller for **15 seconds**.
  Because `Hub.Broadcast` is called synchronously by `NotificationDispatcher.HandleEvent`, this stalls:
  - The Kafka consumer thread processing events for that partition.
  - The remaining healthy clients in the room who cannot receive messages until slow clients time out.
  - Cross-pod Redis Pub/Sub publishing (`publishCrossPod`), which only runs *after* `fanoutLocal` completes.
- **Blast Radius:** Cascading latency spikes across Kafka consumer groups, delayed real-time notifications for all users, and increased consumer lag.
- **Recommendation:**
  Implement asynchronous write pumps with bounded per-connection send queues (`send chan []byte`, depth 64-128).
  If a client's send channel is full, drop non-critical frames or immediately reap the lagging connection rather than blocking the broadcast engine:
  ```go
  select {
  case c.sendQueue <- payload:
  default:
      // Slow consumer: reap or drop frame
      c.Reap()
  }
  ```

---

### 5. [HIGH] Dual-Write Topic Poisoning and Replay Duplication in Outbox Relay
- **File & Lines:** `apps/backend-go/outbox/relay.go:220-261`, `apps/backend-go/events/topic_routing.go:146-163`
- **Component:** Outbox Relay & Dual-Write Topic Router
- **Flaw Description:**
  In `relay.go:220-261`:
  ```go
  func (r *Relay) publishWithRetry(ctx context.Context, e Event) error {
      topics := events.RelayPublishTopics(e.TopicName, e.Payload)
      var lastErr error
      for attempt := 1; attempt <= r.cfg.MaxPublishTries; attempt++ {
          ...
          attemptErr := error(nil)
          for _, topic := range topics {
              pubCtx, cancel := context.WithTimeout(ctx, r.cfg.PublishTimeout)
              err := publishOutboxEvent(pubCtx, r.publisher, topic, granularRoutingKey(e), e)
              cancel()
              if err != nil {
                  attemptErr = err
                  break
              }
          }
          if attemptErr == nil {
              return nil
          }
          ...
      }
      return lastErr
  }
  ```
  When `KAFKA_TOPIC_DUAL_WRITE=true`, `topics` contains `[TopicMain, DomainTopic]` (e.g. `["pegasusx-main", "pegasusx-orders"]`).
  If publishing to `TopicMain` succeeds on Attempt 1, but publishing to `DomainTopic` fails:
  - The loop breaks and sleeps for backoff.
  - In Attempt 2, the loop restarts from `topics[0]` (`TopicMain`), sending a DUPLICATE message to `TopicMain`.
  - If `DomainTopic` fails all 5 attempts, `publishWithRetry` returns an error.
  - The event is moved to `OutboxDeadLetters`, despite having already been successfully published 5 times to `TopicMain`!
- **Blast Radius:** Downstream consumers on `TopicMain` receive multiple duplicate messages, while operational audit logs incorrectly classify the event as dead-lettered.
- **Recommendation:**
  Track publishing status per topic across retry attempts:
  ```go
  pendingTopics := make([]string, len(topics))
  copy(pendingTopics, topics)
  for attempt := 1; attempt <= r.cfg.MaxPublishTries; attempt++ {
      var remaining []string
      for _, topic := range pendingTopics {
          if err := publishOutboxEvent(...); err != nil {
              remaining = append(remaining, topic)
          }
      }
      pendingTopics = remaining
      if len(pendingTopics) == 0 {
          return nil
      }
      ...
  }
  ```

---

### 6. [HIGH] Outbox Polling Contention and Multi-Tenant Starvation
- **File & Lines:** `apps/backend-go/outbox/spanner_store.go:114-190`, `apps/backend-go/outbox/fair.go:8-52`
- **Component:** Spanner Outbox Store & Poller
- **Flaw Description:**
  1. **Lock Contention in Spanner RW Transactions:**
     `SpannerStore.Fetch` executes an index query `SELECT ... FROM OutboxEvents@{FORCE_INDEX=Idx_OutboxEvents_Unpublished} WHERE PublishedAt IS NULL AND (ClaimedUntil IS NULL OR ClaimedUntil < @now) ORDER BY CreatedAt LIMIT @limit` inside a `ReadWriteTransaction`.
     When multiple relay replicas poll Spanner concurrently, they read and attempt to lease the same rows, leading to transaction lock conflicts and abort retries in Spanner.
  2. **Candidate Window Starvation:**
     `fetchLimit` is capped at 500 rows (`spanner_store.go:110`).
     The query orders candidates strictly by `CreatedAt`.
     If Tenant A generates a bulk upload of 10,000 events, all 500 candidates fetched by the query belong to Tenant A.
     `FairInterleave` receives candidates solely from Tenant A.
     Any real-time event produced by Tenant B (e.g. live driver drop-off or POS sale) is located at offset 501+ in Spanner and is completely starved until Tenant A's 10,000 events are drained.
- **Blast Radius:** Multi-pod relay thrashing on Spanner; catastrophic latency spikes for low-volume tenants during high-volume batch operations.
- **Recommendation:**
  - Execute candidate read in a Read-Only transaction (or single snapshot read), compute fair claims, and use a conditional mutation in a short RW transaction to claim rows.
  - Use `Idx_OutboxEvents_Unpublished_BySupplier` to query per-supplier partitions or interleave queries across known active tenants.

---

### 7. [MEDIUM] Ghost Broadcasts to Non-Existent WebSocket Rooms
- **File & Lines:** 
  - `apps/backend-go/payload/progress.go:57` (`"warehouse_ops"`)
  - `apps/backend-go/driver/live_tracking.go:56` (`"fleet_map"`)
  - `apps/backend-go/payload/exceptions.go:104,108,111` (`"warehouse_ops"`, `"retailer_updates"`, `"fleet_broadcast"`)
  - `apps/backend-go/driver/rescue.go:79` (`"fleet_broadcast"`)
  - `apps/backend-go/retailer/shelf_intelligence.go:66` (`"SHELF_ALERT"`)
- **Component:** Domain Services / WebSocket Integration
- **Flaw Description:**
  Multiple domain handlers call `hub.Broadcast` targeting arbitrary room names that are never subscribed to by any client in `ws/handler.go` (`subscribeIdentityRooms`).
  For example, `payload/progress.go` broadcasts progress updates to `"warehouse_ops"`, but warehouse portals subscribe only to `"warehouse:<warehouse_id>"`.
  Similarly, `driver/live_tracking.go` broadcasts coordinates to `"fleet_map"`, which has 0 subscribers.
  In `retailer/shelf_intelligence.go:66`, `Broadcast` is called with invalid parameters (`Broadcast(ctx, "SHELF_ALERT", map[string]interface{}{...}, retailerID)`).
- **Blast Radius:** Real-time dock scanning progress, driver live tracking, and shelf out-of-stock alerts fail to appear on frontend dashboards despite returning HTTP 200.
- **Recommendation:**
  Replace unnamespaced room names with canonical tenant-scoped rooms (e.g. `"warehouse:" + warehouseID`, `"supplier:" + supplierID`, `"driver:" + driverID`).

---

### 8. [MEDIUM] Build & Unit Test Failures in ws and order Packages
- **File & Lines:** `apps/backend-go/ws/hub_test.go:119`, `apps/backend-go/order/service.go:1321`
- **Component:** Build & Test Integrity
- **Flaw Description:**
  1. `ws/hub_test.go:119`: `publishFailBackend` does not implement `cache.Backend` because `IncrBy` and `DecrBy` methods are missing, causing `go test ./ws/...` to fail compilation.
  2. `order/service.go:1321`: References undefined constant `StatusDraft`, causing `go test ./order/...` compilation to fail.
- **Blast Radius:** CI pipeline failures and inability to run WebSocket test suites.
- **Recommendation:**
  Add stub `IncrBy` and `DecrBy` methods to `publishFailBackend` in `ws/hub_test.go` and fix the status constant in `order/service.go`.

---

### 9. [MEDIUM] Missing Reconnection Message Replay / Catch-up Mechanism
- **File & Lines:** `apps/backend-go/ws/handler.go:44-80`, `apps/backend-go/ws/sse.go:191-271`
- **Component:** Realtime Connection Lifecycle
- **Flaw Description:**
  In mobile environments, cellular dropouts and IP handovers are frequent (Desert Protocol).
  When a driver or retailer client disconnects and reconnects to `/v1/ws` or `/v1/events`:
  - The server creates a brand new connection object and subscribes it to rooms.
  - The server does not read `Last-Event-ID` or request query parameters (e.g. `?since_seq=...`).
  - No replay or catch-up messages are dispatched.
  All events emitted during the 10-60 second disconnection window are lost to the client UI until a manual full reload occurs.
- **Blast Radius:** Mobile drivers on the road and POS cashiers frequently see stale orders/manifests after passing through poor-coverage areas.
- **Recommendation:**
  Integrate `notifications/inbox` or a Redis stream ring buffer per room to support delta sync on reconnect when `Last-Event-ID` or `since` is supplied.

---

### 10. [LOW] Full Table Scan in Outbox Supplier ID Backfill Query
- **File & Lines:** `apps/backend-go/outbox/backfill.go:20-26`, `apps/backend-go/outbox/backfill.go:50-60`
- **Component:** Outbox Backfill Worker
- **Flaw Description:**
  `BackfillSupplierID` executes `SELECT EventId, Payload FROM OutboxEvents WHERE (SupplierId IS NULL OR SupplierId = '') ORDER BY CreatedAt DESC LIMIT @limit`.
  Because `Idx_OutboxEvents_Unpublished_BySupplier` is `NULL_FILTERED` on `SupplierId`, it excludes null rows.
  There is no secondary index on `(SupplierId, CreatedAt DESC)`, forcing Spanner to perform a full table scan across historical outbox records every 5 minutes.
  Furthermore, updates are committed one-by-one with `client.Apply` rather than in a single batched mutation.
- **Blast Radius:** Unnecessary Spanner CPU consumption and latency degradation under large outbox table sizes.
- **Recommendation:**
  Batch mutations into a single `client.Apply` call and paginate via primary key cursor or bounded time range.

---

## Architectural & Edge-Case Open Questions

1. **Transactional Boundary with Multiple Shards / Multi-Region Spanner:**
   When scaling to multi-region Spanner instances, does `OutboxEvents` primary key `EventId (UUIDv4)` introduce hot-spotting on splits during high-volume insert bursts? Should `OutboxEvents` be co-located or interleaved with tenant root tables (e.g. `INTERLEAVE IN PARENT Suppliers`), or prefix-sharded with a hash bucket?
2. **Dead Letter Queue Reprocessing & Poison Pill Isolation:**
   If a poisoned event is routed to `OutboxDeadLetters`, how does the ops team safely inspect, patch, and replay it without re-poisoning the consumer group? `cmd/replay-dlq` currently lacks schema validation, tenant filtering, and target topic extraction from headers.
3. **Cross-Pod Redis Pub/Sub Partitioning & Backpressure:**
   `Hub.publishCrossPod` uses a single Redis channel per hub (`ws:<hub_name>:fanout`). In an ecosystem with 10,000 simultaneous drivers streaming 1Hz telemetry, all telemetry traffic is broadcast to every pod in the cluster, forcing every pod to decode and filter messages even if they have no subscribers for that driver. Should Redis Pub/Sub channels be partitioned by tenant/room (e.g. `ws:<hub>:<room_id>`)?
4. **WebSocket Ingress Telemetry Rate Limiting:**
   `IngressRateLimiter` (`ws/limits.go:63-70`) enforces 5 msgs/sec with burst 10 per connection. For 1Hz driver GPS telemetry, this is sufficient; but what if driver diagnostics or sensor batches are sent over the same socket? Will high-frequency sensor bursts trigger disconnects?

---

## Conclusion & Verification Summary

| Finding ID | Severity | Category | Target File & Line | Status |
|---|---|---|---|---|
| F-01 | CRITICAL | Multi-User Retailer WS Deafness | `ws/handler.go:161-172` | Verified |
| F-02 | CRITICAL | Kafka Offset Commit Skip Loss | `kafka/workerpool/workerpool.go:137-158` | Verified |
| F-03 | CRITICAL | Granular Routing Key Order Break | `outbox/relay.go:280-312` | Verified |
| F-04 | HIGH | WS Hub HoL Blocking on Slow Clients | `ws/hub.go:235-266` | Verified |
| F-05 | HIGH | Dual-Write Topic Poisoning / Dups | `outbox/relay.go:220-261` | Verified |
| F-06 | HIGH | Outbox Poller Lock Contention & Starvation | `outbox/spanner_store.go:114-190` | Verified |
| F-07 | MEDIUM | Ghost Broadcasts to Dead Rooms | `payload/progress.go:57`, `driver/live_tracking.go:56` | Verified |
| F-08 | MEDIUM | Test Suite Compilation Failures | `ws/hub_test.go:119`, `order/service.go:1321` | Verified |
| F-09 | MEDIUM | Missing Reconnect Replay / Catch-up | `ws/handler.go:44-80`, `ws/sse.go:191-271` | Verified |
| F-10 | LOW | Outbox Backfill Full Table Scan | `outbox/backfill.go:20-26` | Verified |

