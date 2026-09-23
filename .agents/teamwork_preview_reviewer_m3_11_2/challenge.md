# Adversarial Challenge Report — Milestone 3: Real-Time Monotonic Pipeline Parity

**Agent**: `teamwork_preview_reviewer_m3_11_2`  
**Target Codebase**: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`  
**Date**: 2026-09-23  
**Overall Risk Assessment**: **HIGH**

---

## Executive Summary

Worker 3 (`teamwork_preview_worker_m3_11`) implemented significant improvements across the `pegasus.x` real-time architecture:
- Atomic database transactions with `pgx.Tx` paired with `outbox.Emit` across `warehouse`, `rebate`, and `consignment`.
- Dual pub/sub channel casing (`events:order` and `events:ORDER`) in `outbox/relay.go`.
- Realtime event casing normalization (dot-notation `order.delivered` <-> uppercase snake_case `ORDER_DELIVERED`) in `ws/hub.go`, `fleet-ws-events.ts`, and `supplier-ws-events.ts`.
- All existing tests in `outbox`, `ws`, `warehouse`, `rebate`, `consignment`, and `api` pass cleanly under `go test -v -race`.

**HOWEVER**, under adversarial stress-testing, **two critical architectural vulnerabilities** were empirically uncovered in `backend/internal/ws/hub.go`:
1. **Out-of-Order Sequence Buffer Replay**: In `BroadcastEnvelope`, assigning `seq := atomic.AddInt64(&h.seq, 1)` *outside* the `recentMu.Lock()` ring buffer lock allows concurrent goroutines to interleave, inserting events into `recentEvents` out-of-order (e.g. `seq 1689` before `seq 1688`). When clients call `GetEventsSince` for catch-up resync, events are returned out-of-order, and `oldestSeq := h.recentEvents[0].Seq` miscalculates buffer boundaries, triggering false `fullResync: true` alerts.
2. **Concurrent Map Write under RLock**: In `Hub.Run`, line 111 takes `h.mu.RLock()`, but line 117 executes `delete(h.clients, client)` when a client channel is full. Mutating a map under a read lock is an architectural anti-pattern and a race hazard.

---

## Adversarial Challenges & Findings

### [Critical] Challenge 1: Non-Monotonic Ring Buffer Insertion under Concurrency

- **Assumption Challenged**: Calling `atomic.AddInt64(&h.seq, 1)` guarantees monotonic sequencing across the WebSocket hub replay buffer.
- **Vulnerability Location**: `backend/internal/ws/hub.go:173-188`:
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
- **Attack Scenario & Empirical Reproduction**:
  1. Goroutine A calls `atomic.AddInt64(&h.seq, 1)` and receives `seq = 1688`.
  2. Goroutine B calls `atomic.AddInt64(&h.seq, 1)` and receives `seq = 1689`.
  3. Goroutine B context-switches and acquires `h.recentMu.Lock()` *before* Goroutine A.
  4. Goroutine B appends `{Seq: 1689}` to `h.recentEvents` and unlocks.
  5. Goroutine A acquires `h.recentMu.Lock()` second and appends `{Seq: 1688}` to `h.recentEvents`.
  6. The ring buffer now stores: `[..., {Seq: 1689}, {Seq: 1688}, ...]`.
  7. A reconnecting client invokes `/v1/events/sync?since=1680` -> `wsHub.GetEventsSince(1680)`.
  8. The events returned to the client are: `[..., 1689, 1688, ...]`.
  9. Furthermore, line 212 checks `oldestSeq := h.recentEvents[0].Seq`. If the oldest retained element arrived out-of-order, `since < oldestSeq - 1` triggers an erroneous `fullResync: true`, forcing all clients into expensive REST cache invalidations.
- **Empirical Test Result**:
  When tested with 50 concurrent goroutines emitting 40 events each (2,000 total events):
  ```
  hub_concurrency_stress_test.go:53: Out-of-order sequence detected: index 1675 has seq 1688 <= index 1674 has seq 1689
  hub_concurrency_stress_test.go:53: Out-of-order sequence detected: index 1677 has seq 1636 <= index 1676 has seq 1690
  hub_concurrency_stress_test.go:53: Out-of-order sequence detected: index 1679 has seq 1679 <= index 1678 has seq 1691
  hub_concurrency_stress_test.go:58: Returned events from GetEventsSince are NOT monotonically ordered!
  FAIL: TestHubHighConcurrencyBroadcast
  ```
- **Blast Radius**: Reconnecting desktop and mobile clients receive backwards sequence numbers, causing client-side sequence validation to drop frames, report packet loss, or trigger endless loop resync requests.
- **Mitigation**:
  Unify sequence increment and buffer append inside `h.recentMu.Lock()`:
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

---

### [High] Challenge 2: Map Mutation under `RLock` in Broadcast Fanout Loop

- **Assumption Challenged**: Reading and pruning disconnected clients can be done safely using `h.mu.RLock()`.
- **Vulnerability Location**: `backend/internal/ws/hub.go:110-120`:
  ```go
  case message := <-h.broadcast:
      h.mu.RLock()
      for client := range h.clients {
          select {
          case client.send <- message:
          default:
              close(client.send)
              delete(h.clients, client) // <-- MUTATION UNDER RLOCK!
          }
      }
      h.mu.RUnlock()
  ```
- **Attack Scenario**:
  When a slow client connection buffer (capacity 64) fills up, `case message := <-h.broadcast` attempts to unregister the slow client by calling `delete(h.clients, client)`. However, `h.mu.RLock()` is a shared read lock. If any other goroutine iterates or reads `h.clients` under an RLock, Go's runtime will detect concurrent map read and map write and panic (`fatal error: concurrent map iteration and map write`).
- **Blast Radius**: Sudden server panic and crash under high client churn or network congestion.
- **Mitigation**:
  Collect dead clients during iteration, release `RLock()`, or acquire `h.mu.Lock()` exclusively when slow clients need pruning:
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

---

### [Medium] Challenge 3: Missing High-Concurrency Monotonicity Test in `ws/hub_test.go`

- **Assumption Challenged**: Sequential unit tests in `ws/hub_test.go` adequately verify monotonic sequence guarantees.
- **Vulnerability Location**: `backend/internal/ws/hub_test.go`:
  The test suite contains only sequential tests (`TestHubMonotonicSequencingAndCatchUp`, `TestHubDualCasingNormalization`, `TestHubBroadcastAutoWrapping`, `TestHubRingBufferOverflowResync`). It completely lacks a concurrent broadcast test with multiple goroutines.
- **Mitigation**: Add a permanent `TestHubHighConcurrencyBroadcast` unit test in `ws/hub_test.go` that runs 50+ goroutines concurrently and asserts that `GetEventsSince` returns strictly monotonic sequences.

---

### [Medium] Challenge 4: Ignored Outbox Emissions in `matching/service.go` (Coverage Gap)

- **Assumption Challenged**: All domain services in `pegasus.x` strictly check `outbox.Emit` errors and pair mutations in `RunInTx`.
- **Vulnerability Location**: `backend/internal/matching/service.go:59-73`:
  ```go
  if s.repo != nil {
      if err := s.repo.SaveInvoiceMatchResult(ctx, result, inv); err != nil {
          return nil, fmt.Errorf("save match result: %w", err)
      }
  }

  if s.pool != nil {
      _ = s.pool.RunInTx(ctx, func(tx pgx.Tx) error {
          _ = outbox.Emit(ctx, tx, "ENTERPRISE_INVOICE", inv.InvoiceID, "matching.invoice_evaluated", result)
          if result.DebitNoteCreated && result.DebitNote != nil {
              _ = outbox.Emit(ctx, tx, "DEBIT_NOTE", result.DebitNote.DebitNoteID, "matching.debit_note_issued", result.DebitNote)
          }
          return nil
      })
  }
  ```
- **Finding**: While `warehouse`, `rebate`, and `consignment` were hardened, `matching/service.go` continues to save `InvoiceMatchResult` outside `RunInTx` and discards errors with `_ = outbox.Emit`.
- **Mitigation**: Update `matching/service.go` to use `txRepo.SaveInvoiceMatchResultTx` inside `s.pool.RunInTx` and propagate `outbox.Emit` errors.

---

## Stress Test Results Summary

| Scenario | Expected Behavior | Actual Behavior | Result |
| :--- | :--- | :--- | :--- |
| Sequential Hub Sequencing (5 events) | Strictly monotonic 1..5 | Monotonic 1..5 | **PASS** |
| Dual Casing Normalization (`order.created` / `MANIFEST_SEALED`) | Both lowercase and uppercase aliases populated | `event_type` and `type` populated correctly | **PASS** |
| Auto-wrapping of raw JSON broadcast | Wraps un-enveloped JSON with `seq` | Wrapped with `seq=1`, `type=DISPATCH_COMMITTED` | **PASS** |
| Buffer Overflow Resync Detection (60 events in 50 buffer) | `fullResync: true` for overwritten seq 5 | `fullResync: true`, `currentSeq: 60` | **PASS** |
| **High Concurrency Hub Broadcast (50 goroutines, 2000 events)** | **Strictly monotonic sequence in `recentEvents`** | **Out-of-order sequences in `recentEvents` and `GetEventsSince`** | **FAIL** |
| Atomic Rollback in `warehouse.RegisterWarehouse` on error | Both warehouse row and outbox rolled back | Single `pgx.Tx` rolled back | **PASS** |
| Atomic Rollback in `rebate.AccrueForDeliveredOrder` on error | Both accrual row and outbox rolled back | Single `pgx.Tx` rolled back | **PASS** |
| Atomic Rollback in `consignment.ExecutePickOwnershipTransfer` on error | Both voucher and outbox rolled back | Single `pgx.Tx` rolled back | **PASS** |
| Full backend race test execution (`-race -count=1`) | Zero race conditions in existing test suite | Clean exit 0 across all 6 target packages | **PASS** |

---

## Conclusion & Recommended Action

Because Challenge 1 directly violates the core requirement of monotonic sequence delivery under concurrent load, the verdict cannot be approved in its current state. 

**Recommended Action**: **REQUEST_CHANGES** for Worker 3 to apply the 10-line fix in `backend/internal/ws/hub.go` and add the concurrency regression test in `backend/internal/ws/hub_test.go`.
