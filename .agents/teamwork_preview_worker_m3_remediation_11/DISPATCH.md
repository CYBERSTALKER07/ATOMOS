## 2026-09-23T13:07:25Z
You are teamwork_preview_worker_m3_remediation_11.
Your working directory is:
/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m3_remediation_11

Your parent conversation ID is:
d877571c-b5bd-4489-a1cb-441c991bb03d

Authoritative user request (read this file first):
/Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md

Project scope document:
/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_11/PROJECT.md

Challenger Review Report with exact issues and code snippets to fix:
/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_reviewer_m3_11_2/challenge.md
/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_reviewer_m3_11_2/handoff.md

Domain skill:
Read and follow instructions in /Users/shakhzod/.gemini/config/skills/golang-pro/SKILL.md

Target codebase directory:
/Users/shakhzod/Desktop/V.O.I.D/pegasus.x

Your Mission (Milestone 3 Remediation):
Fix all concurrency defects in the real-time monotonic pipeline identified by the Adversarial Challenger:

1. **Strict Monotonic Sequence Assignment under Lock in `backend/internal/ws/hub.go`**:
   - In `BroadcastEnvelope(eventType string, payload any) RealtimeEnvelope`:
     Move sequence counter increment `h.seq++` strictly inside `h.recentMu.Lock()` alongside the append to `h.recentEvents`.
     This guarantees that sequence numbers in `h.recentEvents` are strictly monotonically increasing without out-of-order interleaving under high concurrency.
   - Verify `GetEventsSince` logic in `hub.go` so replay catch-up is strictly monotonic and buffer boundary calculation (`since < oldestSeq-1`) operates correctly.

2. **Safe Slow Client Pruning in `backend/internal/ws/hub.go`**:
   - In `Hub.Run`:
     Eliminate map deletion under read lock (`h.mu.RLock()`).
     Under `h.mu.RLock()`, collect slow clients whose send buffer is full into a `slowClients []*Client` slice.
     Release `h.mu.RUnlock()`.
     If `len(slowClients) > 0`, acquire write lock `h.mu.Lock()` and safely close and delete slow clients from `h.clients`.

3. **High-Concurrency Monotonicity Test in `backend/internal/ws/hub_test.go`**:
   - Add `TestHubHighConcurrencyBroadcast(t *testing.T)`:
     Spawn 50 concurrent goroutines broadcasting 40 events each (2,000 total events).
     Verify via `GetEventsSince` that:
     (a) exactly all events are returned,
     (b) every returned event sequence is strictly monotonic (`events[i].Seq > events[i-1].Seq`),
     (c) zero sequence gaps, zero duplicates, and zero out-of-order sequence numbers occur,
     (d) `fullResync` is false when querying from a valid sequence in the buffer.

4. **Address Outbox Error Handling in `backend/internal/matching/service.go`**:
   - In `backend/internal/matching/service.go:59-73`, eliminate ignored `_ = outbox.Emit(...)` and ensure errors are checked and returned.

5. **Full Verification**:
   - Run `go test -v -race -run TestHubHighConcurrencyBroadcast ./internal/ws/...` in `pegasus.x/backend`.
   - Run `go test -v -race -count=1 ./internal/outbox/... ./internal/ws/... ./internal/warehouse/... ./internal/rebate/... ./internal/consignment/... ./internal/api/...`.
   - Run `go vet ./...` and `go build ./cmd/server` to confirm clean compilation.
   - Verify 100% test pass with 0 race conditions.
