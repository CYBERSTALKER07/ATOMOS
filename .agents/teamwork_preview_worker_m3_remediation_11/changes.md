# Changes Document — Milestone 3 Remediation: Real-Time Monotonic Pipeline Parity

**Agent**: `teamwork_preview_worker_m3_remediation_11`  
**Working Directory**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m3_remediation_11`  
**Target Codebase**: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`  
**Date**: 2026-09-23  

---

## Summary of Changes

This remediation addresses all concurrency defects and technical debt items identified in the Milestone 3 Adversarial Challenge Report:

1. **Strict Monotonic Sequence Assignment Under Lock (`backend/internal/ws/hub.go`)**:
   - In `BroadcastEnvelope(eventType string, payload any) RealtimeEnvelope`: Moved sequence counter increment `h.seq++` strictly inside `h.recentMu.Lock()` alongside the append to `h.recentEvents`.
   - Eliminated race window where `atomic.AddInt64(&h.seq, 1)` executed prior to acquiring `h.recentMu.Lock()`, which previously allowed concurrent goroutines to interleave and insert higher sequence frames before lower sequence frames.
   - Verified `GetEventsSince` logic where `currentSeq = h.seq` is read under `h.recentMu.RLock()`, guaranteeing strictly monotonic replay ordering and precise buffer boundary calculation (`since < oldestSeq-1`).

2. **Safe Slow Client Pruning Under Write Lock (`backend/internal/ws/hub.go`)**:
   - In `Hub.Run`: Eliminated map deletion under `h.mu.RLock()`.
   - Iteration under `h.mu.RLock()` now collects slow clients whose send buffer is full into a `slowClients []*Client` slice.
   - After releasing `h.mu.RUnlock()`, if `len(slowClients) > 0`, the hub acquires exclusive write lock `h.mu.Lock()`, verifies membership, closes channels, and deletes slow clients from `h.clients`.

3. **High-Concurrency Monotonicity & Client Pruning Tests (`backend/internal/ws/hub_test.go`)**:
   - Added `TestHubHighConcurrencyBroadcast(t *testing.T)`: Spawns 50 concurrent goroutines broadcasting 40 events each (2,000 total events). Asserts via `GetEventsSince(0)` that:
     - Exactly 2,000 events are returned with `fullResync = false`.
     - Every returned event sequence is strictly monotonic (`events[i].Seq == int64(i + 1)` and `events[i].Seq > events[i-1].Seq`).
     - Zero sequence gaps, zero duplicates, and zero out-of-order sequence numbers occur.
     - Mid-buffer query (`GetEventsSince(1500)`) returns remaining 500 events monotonically without resync.
   - Added `TestHubSlowClientPruning(t *testing.T)`: Verifies slow client buffer overflow triggers safe unregistration without panics, deadlocks, or races.

4. **Outbox Error Handling & Atomic Tx Execution (`backend/internal/matching/service.go` & `repository.go`)**:
   - In `backend/internal/matching/service.go`: Defined `TxRepository` interface supporting `SaveInvoiceMatchResultTx(ctx, tx, res, inv)`.
   - In `EvaluateInvoice`: Paired match result database updates and outbox events (`matching.invoice_evaluated`, `matching.debit_note_issued`) in the exact same `pgx.Tx` transaction closure when `TxRepository` is present.
   - Replaced ignored `_ = outbox.Emit` calls with proper error checking and returned errors.
   - In `backend/internal/matching/repository.go`: Added `SaveInvoiceMatchResultTx` to `PostgresRepository`.
   - In `backend/internal/matching/matching_test.go`: Added unit tests `TestServiceEvaluateInvoice_Success` and `TestServiceEvaluateInvoice_SaveError` verifying match result persistence and error propagation.

---

## Detailed File Modifications

### 1. `backend/internal/ws/hub.go`
- **Imports**: Removed unused `"sync/atomic"`.
- **`Hub.Run`**: Under `h.broadcast`, collect slow clients in slice during read lock iteration. Release read lock, acquire `h.mu.Lock()`, close `client.send`, and `delete(h.clients, client)`.
- **`BroadcastEnvelope`**: Execute `h.seq++` and `seq := h.seq` inside `h.recentMu.Lock()`.
- **`GetEventsSince`**: Read `currentSeq = h.seq` inside `h.recentMu.RLock()`.

### 2. `backend/internal/ws/hub_test.go`
- **Imports**: Added `"context"`, `"sync"`, `"time"`.
- **`TestHubHighConcurrencyBroadcast`**: 50 goroutines × 40 events = 2,000 events; verified strict monotonicity and 0 gaps.
- **`TestHubSlowClientPruning`**: Verified slow client eviction under full buffer.

### 3. `backend/internal/matching/service.go`
- **Interface**: Added `TxRepository`.
- **`EvaluateInvoice`**: Atomically executes `txRepo.SaveInvoiceMatchResultTx` + `outbox.Emit` inside `s.pool.RunInTx`. Checks and returns outbox emission errors.

### 4. `backend/internal/matching/repository.go`
- **`PostgresRepository.SaveInvoiceMatchResultTx`**: Executes invoice status update and debit note insert on given `pgx.Tx`.
- **`PostgresRepository.SaveInvoiceMatchResult`**: Delegates to `SaveInvoiceMatchResultTx` inside `p.pool.RunInTx`.

### 5. `backend/internal/matching/matching_test.go`
- **`TestServiceEvaluateInvoice_Success`**: Verifies 3-way match evaluation and persistence in `MemoryRepository`.
- **`TestServiceEvaluateInvoice_SaveError`**: Verifies error return when repository save fails.

---

## Verification Command & Results

```bash
# 1. High-concurrency broadcast test with race detector
cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
go test -v -race -run TestHubHighConcurrencyBroadcast ./internal/ws/...
# Output: PASS (0.01s)

# 2. All target package test suites uncached with race detector
go test -v -race -count=1 ./internal/outbox/... ./internal/ws/... ./internal/warehouse/... ./internal/rebate/... ./internal/consignment/... ./internal/matching/... ./internal/api/...
# Output: PASS across all packages (46.218s total, 0 race warnings)

# 3. Static analysis & compilation
go vet ./...
go build ./cmd/server
# Output: Clean exit code 0, zero warnings
```
