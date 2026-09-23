# Handoff Report — Milestone 3 Adversarial Review & Concurrency Verification

**Agent**: `teamwork_preview_reviewer_m3_11_2`  
**Working Directory**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_reviewer_m3_11_2`  
**Parent Agent**: `d877571c-b5bd-4489-a1cb-441c991bb03d`  
**Milestone**: Milestone 3 — Real-Time Monotonic Pipeline Parity  
**Target Codebase**: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`  
**Verdict**: **REQUEST_CHANGES**

---

## 1. Observation

Direct observations from codebase inspection, adversarial stress-testing, and automated test execution:

1. **Automated Test Suite Execution**:
   - Command executed:
     ```bash
     cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
     go test -v -race -count=1 ./internal/outbox/... ./internal/ws/... ./internal/warehouse/... ./internal/rebate/... ./internal/consignment/... ./internal/api/...
     ```
   - Verbatim outputs:
     ```
     ok  github.com/pegasus-x/core/internal/outbox      1.422s
     ok  github.com/pegasus-x/core/internal/ws          1.335s
     ok  github.com/pegasus-x/core/internal/warehouse   4.432s
     ok  github.com/pegasus-x/core/internal/rebate      1.337s
     ok  github.com/pegasus-x/core/internal/consignment  1.341s
     ok  github.com/pegasus-x/core/internal/api        47.308s
     ```
   - All existing tests in the target packages passed with exit code 0.

2. **Empirical Concurrency Defect in `ws/hub.go`**:
   - File: `backend/internal/ws/hub.go:173-188`:
     ```go
     func (h *Hub) BroadcastEnvelope(eventType string, payload any) RealtimeEnvelope {
         seq := atomic.AddInt64(&h.seq, 1)
         rawType, upperType := normalizeEventType(eventType)
         env := RealtimeEnvelope{
             Seq:       seq,
             EventType: rawType,
             Type:      upperType,
             Payload:   payload,
             Timestamp: time.Now().Unix(),
         }

         h.recentMu.Lock()
         if len(h.recentEvents) >= h.maxHistory {
             h.recentEvents = h.recentEvents[1:]
         }
         h.recentEvents = append(h.recentEvents, env)
         h.recentMu.Unlock()
     ```
   - When stress-tested with 50 concurrent goroutines broadcasting 40 events each (2,000 total events), `atomic.AddInt64(&h.seq, 1)` executed outside `h.recentMu.Lock()`, allowing goroutines to interleave and append out-of-order sequence frames to `h.recentEvents`.
   - Verbatim failure trace:
     ```
     hub_concurrency_stress_test.go:53: Out-of-order sequence detected: index 1675 has seq 1688 <= index 1674 has seq 1689
     hub_concurrency_stress_test.go:53: Out-of-order sequence detected: index 1677 has seq 1636 <= index 1676 has seq 1690
     hub_concurrency_stress_test.go:53: Out-of-order sequence detected: index 1679 has seq 1679 <= index 1678 has seq 1691
     hub_concurrency_stress_test.go:58: Returned events from GetEventsSince are NOT monotonically ordered!
     FAIL: TestHubHighConcurrencyBroadcast
     ```
   - File `backend/internal/ws/hub.go:212-215`:
     ```go
     oldestSeq := h.recentEvents[0].Seq
     if since < oldestSeq-1 {
         return nil, currentSeq, true
     }
     ```
     Because `h.recentEvents[0]` can hold an out-of-order higher sequence, `since < oldestSeq-1` miscalculates buffer boundaries and triggers false `fullResync: true` responses.

3. **Concurrent Map Mutation under RLock**:
   - File: `backend/internal/ws/hub.go:110-120`:
     ```go
     case message := <-h.broadcast:
         h.mu.RLock()
         for client := range h.clients {
             select {
             case client.send <- message:
             default:
                 close(client.send)
                 delete(h.clients, client)
             }
         }
         h.mu.RUnlock()
     ```
   - Line 111 acquires `h.mu.RLock()`, but line 117 invokes `delete(h.clients, client)` to prune slow clients whose send buffer is full. Mutating a map under an `RLock` is an anti-pattern that triggers runtime fatal panics (`concurrent map iteration and map write`) if any concurrent reader accesses `h.clients`.

4. **Transactional Rollback Atomicity**:
   - `backend/internal/warehouse/service.go`: `RegisterWarehouse` (lines 104-115), `CompleteOnboarding` (lines 384-399), and `ReconcileInboundBlind` (lines 652-683) execute database repository updates and `outbox.Emit` inside `s.pool.RunInTx(ctx, func(tx pgx.Tx) error { ... })`.
   - `backend/internal/rebate/service.go`: `AccrueForDeliveredOrder` (lines 67-84) and `SettleContract` (lines 145-150) execute contract/accrual mutations and `outbox.Emit` inside `s.pool.RunInTx`.
   - `backend/internal/consignment/service.go`: `InboundConsignmentReceive` (lines 86-97) and `ExecutePickOwnershipTransfer` (lines 145-163) execute agreement/voucher mutations and `outbox.Emit` inside `s.pool.RunInTx`.
   - `backend/internal/outbox/emitter.go`: `Emit` (lines 20-53) executes `tx.Exec` on the active `pgx.Tx`. If either the domain operation or `outbox.Emit` fails, `s.pool.RunInTx` calls `tx.Rollback(ctx)`, guaranteeing that outbox events are never committed on transaction failure.

5. **Event Casing Normalization & Desktop Verification**:
   - Backend `backend/internal/ws/hub.go:44-60`: `normalizeEventType` maps `order.delivered` to `rawType: "order.delivered"`, `upperType: "ORDER_DELIVERED"`. `RealtimeEnvelope` provides dual fields (`event_type` and `type`).
   - `apps/warehouse-desktop/lib/fleet-ws-events.ts:5-32`: `parseWsEventType` parses JSON strings and objects, converts dot-notation to uppercase snake_case (`ORDER_DELIVERED`), and triggers query invalidation against `ORDER_STATUS_REFRESH_EVENTS`.
   - `apps/supplier-desktop/lib/supplier-ws-events.ts:49-89`: `parseSupplierWsEventType` similarly extracts and normalizes event types.

6. **Outbox Emission Coverage Gap**:
   - File: `backend/internal/matching/service.go:59-73`:
     ```go
     if s.repo != nil {
         if err := s.repo.SaveInvoiceMatchResult(ctx, result, inv); err != nil {
             return nil, fmt.Errorf("save match result: %w", err)
         }
     }
     if s.pool != nil {
         _ = s.pool.RunInTx(ctx, func(tx pgx.Tx) error {
             _ = outbox.Emit(ctx, tx, "ENTERPRISE_INVOICE", inv.InvoiceID, "matching.invoice_evaluated", result)
             ...
     ```
     `SaveInvoiceMatchResult` is executed outside any transaction, and both `s.pool.RunInTx` and `outbox.Emit` errors are ignored with `_ =`.

---

## 2. Logic Chain

1. **From Observation 2 to Broken Monotonic Replay**:
   - Requirement R3 explicitly mandates: *"Verify sequence counter seq strictly increments monotonically without duplicates, races, or sequence gaps."*
   - Because `atomic.AddInt64(&h.seq, 1)` occurs before acquiring `recentMu.Lock()`, goroutines interleave under concurrent load, appending events with higher sequence IDs before lower sequence IDs in `recentEvents`.
   - When a client reconnects and queries `/v1/events/sync?since=X`, `GetEventsSince` iterates through `recentEvents` and emits events in non-monotonic order.
   - Furthermore, `oldestSeq := h.recentEvents[0].Seq` relies on the first element being the minimum sequence in the buffer. When an out-of-order sequence lands at index 0, buffer boundary checks fail, causing false full resyncs.
   - Therefore, `ws/hub.go` violates Requirement R3 under concurrent conditions.

2. **From Observation 3 to Server Stability Risk**:
   - Mutating a Go map (`delete(h.clients, client)`) while holding only `h.mu.RLock()` violates Go memory safety and concurrency design. Under load or future refactoring with concurrent readers, this triggers unrecoverable runtime panics.

3. **From Observation 4 and 5 to Partial Parity Success**:
   - Worker 3 successfully implemented atomic rollback pairing across `warehouse`, `rebate`, and `consignment`.
   - Worker 3 successfully implemented dual event casing normalization in both the backend hub and desktop client listeners.
   - All existing tests pass cleanly with race detection.

4. **From Deductions to Verdict**:
   - Because the concurrency defect in `ws/hub.go` causes demonstrable out-of-order event delivery and breaks monotonic client catch-up, the milestone cannot be marked complete without addressing this flaw.
   - The required fix is small, precise, and well-isolated.
   - Consequently, the verdict is **REQUEST_CHANGES**.

---

## 3. Caveats

- **Existing Tests Passed**: The existing test suite passed 100% because `ws/hub_test.go` only contained sequential tests and did not test concurrent broadcasts. The defect only manifested when subjected to an adversarial concurrency stress test.
- **`matching/service.go`**: This service was not modified by Worker 3 in Milestone 3, but contains unhandled `_ = outbox.Emit` calls that should be addressed as technical debt before final system audit.

---

## 4. Conclusion

**Verdict**: **REQUEST_CHANGES**

### Required Remediation Items:

1. **Fix `BroadcastEnvelope` in `backend/internal/ws/hub.go`**:
   Move sequence increment inside `h.recentMu.Lock()` to ensure atomic sequencing and strictly monotonic insertion into `h.recentEvents`:
   ```go
   func (h *Hub) BroadcastEnvelope(eventType string, payload any) RealtimeEnvelope {
       rawType, upperType := normalizeEventType(eventType)

       h.recentMu.Lock()
       h.seq++
       seq := h.seq
       env := RealtimeEnvelope{
           Seq:       seq,
           EventType: rawType,
           Type:      upperType,
           Payload:   payload,
           Timestamp: time.Now().Unix(),
       }
       if len(h.recentEvents) >= h.maxHistory {
           h.recentEvents = h.recentEvents[1:]
       }
       h.recentEvents = append(h.recentEvents, env)
       h.recentMu.Unlock()

       data, err := json.Marshal(env)
       if err == nil {
           h.broadcastRaw(data)
       }
       return env
   }
   ```

2. **Fix `Hub.Run` Client Pruning under Concurrency in `backend/internal/ws/hub.go`**:
   Do not delete from `h.clients` under `h.mu.RLock()`. Collect slow clients in a slice and delete them under `h.mu.Lock()`:
   ```go
   case message := <-h.broadcast:
       var slowClients []*Client
       h.mu.RLock()
       for client := range h.clients {
           select {
           case client.send <- message:
           default:
               slowClients = append(slowClients, client)
           }
       }
       h.mu.RUnlock()

       if len(slowClients) > 0 {
           h.mu.Lock()
           for _, client := range slowClients {
               if _, ok := h.clients[client]; ok {
                   close(client.send)
                   delete(h.clients, client)
               }
           }
           h.mu.Unlock()
       }
   ```

3. **Add High-Concurrency Monotonicity Test in `backend/internal/ws/hub_test.go`**:
   Add `TestHubHighConcurrencyBroadcast` with 50+ concurrent goroutines to assert that `GetEventsSince` returns strictly monotonic sequences (`events[i].Seq > events[i-1].Seq`) with 0 duplicates and 0 out-of-order frames.

---

## 5. Verification Method

Once Worker 3 applies the remediation:

1. **Verify Concurrency Stress Test**:
   Execute:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
   go test -v -race -run TestHubHighConcurrencyBroadcast ./internal/ws/...
   ```
   *Expected Output*: `PASS` with zero out-of-order sequence frames.

2. **Verify Full Milestone 3 Target Packages**:
   Execute:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
   go test -v -race -count=1 ./internal/outbox/... ./internal/ws/... ./internal/warehouse/... ./internal/rebate/... ./internal/consignment/... ./internal/api/...
   ```
   *Expected Output*: `PASS` across all packages with 0 race detector warnings.

3. **Verify Build & Vet**:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
   go vet ./...
   go build ./cmd/server
   ```
   *Expected Output*: Exit code 0, zero warnings.
