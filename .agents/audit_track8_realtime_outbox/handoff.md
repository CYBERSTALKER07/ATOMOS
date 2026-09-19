# Track 8 Handoff Report: Realtime Engine, Outbox Pattern, Kafka & Multi-Hub WebSocket Fanout

**Auditor:** Codebase Explorer (Track 8 Realtime/Outbox/Kafka/WS Specialist)  
**Date:** 2026-08-30  
**Handoff Type:** Hard (Task Complete)  

---

## 1. Observation

Direct code inspection and test executions in `pegasusX/apps/backend-go` revealed the following verified facts:

1. **Retailer Multi-User Room Mismatch:**
   - In `apps/backend-go/ws/handler.go:161-172`, `subscribeRetailerRooms` subscribes connections to `"retailer:" + ident.Subject`.
   - In `apps/backend-go/auth/claims.go:229-238`, `ResolveRetailerOrgID` demonstrates that for staff members, `Subject` is the user ID, while `RetailerOrgID` is the organization ID.
   - In `apps/backend-go/kafka/notification_dispatcher.go:938`, `broadcastRetailer` publishes exclusively to `"retailer:" + retailerID` (the organization ID). Staff connections are in different rooms and receive 0 messages.
2. **Monotonic Kafka Offset Commit Skip Invalidation:**
   - In `apps/backend-go/kafka/workerpool/workerpool.go:145-147`, when `Handler` returns `ErrSkipCommit`, the worker skips `CommitMessages` on message $N$.
   - In `apps/backend-go/kafka/workerpool/workerpool.go:154`, when message $N+1$ on that same partition worker channel succeeds, `p.cfg.Source.CommitMessages(commitCtx, m)` is invoked for message $N+1$.
   - Under Kafka protocol, committing offset $N+1$ commits all previous offsets $\le N+1$ for that partition, permanently dropping the failed message $N$.
3. **Kafka Partition Key Fragmentation:**
   - In `apps/backend-go/outbox/relay.go:280-312`, `granularRoutingKey` appends `:order_id`, `:manifest_id`, `:route_id`, or `:driver_id` from payload to `e.AggregateID`.
   - `outbox/kafka_publisher.go:88` configures `Balancer: &kafka.Hash{}`.
   - Different events for the same aggregate root hash to different Kafka partitions, destroying per-aggregate FIFO ordering guarantees.
4. **WebSocket Synchronous HoL Blocking:**
   - In `apps/backend-go/ws/hub.go:245-256`, `fanoutLocal` iterates over all room connections synchronously, calling `c.Send(writeCtx, payload)` with a 5-second deadline (`time.Now().Add(5 * time.Second)` in `ws/connection.go:113`).
   - A slow client stalls `Hub.Broadcast`, blocking the Kafka consumer dispatcher goroutine for the duration of the timeout.
5. **Dual-Write Retry Duplication:**
   - In `apps/backend-go/outbox/relay.go:230-237`, `publishWithRetry` iterates over `topics` returned by `RelayPublishTopics`. If topic 1 succeeds and topic 2 fails, retrying the loop re-publishes to topic 1 on every attempt.
6. **Spanner Outbox Poller Lock Contention & Starvation:**
   - In `apps/backend-go/outbox/spanner_store.go:114-190`, `Fetch` queries `OutboxEvents` inside a Spanner `ReadWriteTransaction`. Multiple relay instances experience lock contention on index `Idx_OutboxEvents_Unpublished`.
   - `fetchLimit` (max 500) combined with `ORDER BY CreatedAt` allows a single high-volume tenant to monopolize the candidate window, starving other tenants despite `FairInterleave` (`outbox/fair.go`).
7. **Ghost Broadcasts to Unsubscribed Rooms:**
   - `payload/progress.go:57` broadcasts to `"warehouse_ops"`.
   - `driver/live_tracking.go:56` broadcasts to `"fleet_map"`.
   - `driver/rescue.go:79` and `payload/exceptions.go:111` broadcast to `"fleet_broadcast"`.
   - None of these rooms exist in `ws/handler.go:132-252` (`subscribeIdentityRooms`).
8. **Unit Test Compilation Failure:**
   - Running `go test ./ws/...` fails with `ws/hub_test.go:119:28: *publishFailBackend does not implement cache.Backend (missing method DecrBy, IncrBy)`.
   - Running `go test ./order/...` fails with `order/service.go:1321:13: undefined: StatusDraft`.

---

## 2. Logic Chain

1. **Retailer Realtime Delivery:**
   - *Observation:* Staff JWT has `Subject = user_id`, `RetailerOrgID = org_id`. WS handler subscribes to `retailer:user_id`. Kafka dispatcher broadcasts to `retailer:org_id`.
   - *Deduction:* The room keys do not match. The WebSocket hub cannot route broadcasts to staff connections. Therefore, all real-time events for multi-user retailer organizations fail to deliver.
2. **Kafka Loss on DLQ Failure:**
   - *Observation:* `runWorker` handles sequential messages on a single partition channel. Skipping commit on message $N$ does not halt the worker loop. Message $N+1$ commits offset $N+1$.
   - *Deduction:* Kafka's offset management is strictly cumulative. Acknowledging offset $N+1$ automatically acknowledges offset $N$. Therefore, un-DLQ'd messages are silently dropped.
3. **Partition Ordering Loss:**
   - *Observation:* `granularRoutingKey` outputs different keys for the same aggregate depending on payload fields. Kafka hashes keys to partition numbers.
   - *Deduction:* Events with different keys are assigned to different partitions. Partitions are processed concurrently. Therefore, sequential ordering between aggregate creation and child operations cannot be guaranteed.
4. **WebSocket Fanout Latency Propagation:**
   - *Observation:* `fanoutLocal` runs in the caller's goroutine and calls `c.Send` sequentially per client. `c.Send` sets a 5s deadline.
   - *Deduction:* Slow TCP sockets cause the broadcast loop to block proportionally to the number of degraded connections, blocking the Kafka consumer dispatcher and delaying cross-pod Redis fanout.

---

## 3. Caveats

1. **Spanner Emulator vs Production Spanner:**
   - Lock contention and abort rates on `OutboxEvents` during multi-relay polling were analyzed from Spanner's serializable ReadWriteTransaction concurrency model. Exact lock conflict frequency depends on the number of running backend pods and event emission rate.
2. **Kafka Broker Configuration:**
   - The impact of granular partition keys depends on the number of topic partitions configured in Kafka cluster. In single-partition test environments, ordering appears intact, masking the multi-partition production bug.
3. **External Redis Cluster Performance:**
   - Cross-pod Redis Pub/Sub throughput (`ws:<hub>:fanout`) was evaluated under single-channel assumptions. Production clusters with thousands of simultaneous telemetry streams may experience additional CPU overhead on Redis nodes.

---

## 4. Conclusion

Track 8's architecture provides a solid conceptual foundation (Transactional Outbox + Kafka Consumer Groups + Multi-Hub WebSockets). However, the **8 critical and high-severity findings documented in `findings.md` must be addressed before production deployment**:
1. Fix `subscribeRetailerRooms` to bind to `auth.ResolveRetailerOrgID(ident)`.
2. Stop cumulative Kafka offset advancement when `ErrSkipCommit` is encountered on a partition worker.
3. Standardize Kafka partition routing keys on the root aggregate ID.
4. Convert `Hub.fanoutLocal` to use asynchronous non-blocking per-connection write queues.
5. Make `Relay.publishWithRetry` track per-topic delivery status.
6. Refactor outbox polling to query candidates outside Spanner ReadWriteTransactions and implement per-supplier partitioning.
7. Align domain `hub.Broadcast` calls with canonical room naming conventions.
8. Fix broken test mocks in `ws/hub_test.go` and status enum in `order/service.go`.

---

## 5. Verification Method

To independently verify the observations and conclusions:

1. **Verify WebSocket Compilation & Test Failure:**
   ```bash
   cd pegasusX/apps/backend-go
   go test ./ws/...
   # Observe failure in ws/hub_test.go:119 regarding missing cache.Backend methods
   ```
2. **Verify Multi-User Retailer Room Mismatch:**
   - Inspect `apps/backend-go/ws/handler.go` lines 161-172:
     Note that it subscribes to `"retailer:" + ident.Subject`.
   - Inspect `apps/backend-go/kafka/notification_dispatcher.go` line 938:
     Note that it broadcasts to `"retailer:" + retailerID` (where retailerID is `RetailerOrgID`).
3. **Verify Kafka Workerpool Monotonic Commit Vulnerability:**
   - Inspect `apps/backend-go/kafka/workerpool/workerpool.go` lines 137-158:
     Observe that `ErrSkipCommit` continues the loop without pausing consumption on that channel, and subsequent message commits call `p.cfg.Source.CommitMessages(commitCtx, m)`.
4. **Verify Outbox Granular Routing Key Partitioning:**
   - Inspect `apps/backend-go/outbox/relay.go` lines 280-312:
     Observe dynamic appending of sub-entity IDs (`:+envelope.OrderID`) to `e.AggregateID`.
   - Inspect `apps/backend-go/outbox/kafka_publisher.go` line 88:
     Observe `Balancer: &kafka.Hash{}`.
5. **Verify Outbox Tests:**
   ```bash
   cd pegasusX/apps/backend-go
   go test -v ./outbox/...
   ```

