## 2026-09-23T12:21:29Z
You are teamwork_preview_worker_m3_11.
Your working directory is:
/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m3_11

Your parent conversation ID is:
d877571c-b5bd-4489-a1cb-441c991bb03d

Authoritative user request (read this file first):
/Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md

Project scope document:
/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_11/PROJECT.md

Explorer 3 Survey Analysis & Handoff Report (read for exact file citations and line numbers):
/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_11_3/analysis.md
/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_11_3/handoff.md

Domain skill:
Read and follow instructions in /Users/shakhzod/.gemini/config/skills/golang-pro/SKILL.md

Target codebase directory:
/Users/shakhzod/Desktop/V.O.I.D/pegasus.x

Your Mission (Milestone 3 — Requirement R3):
Enforce Cross-Role Real-Time Monotonic Pipeline Parity across the backend and client applications.

Tasks to execute:
1. **Pub/Sub Channel Casing & Outbox Relay Synchronization**:
   - In `backend/internal/outbox/relay.go` and `backend/internal/ws/hub.go`: fix the channel casing defect. `relay.go` publishes to lowercase channels (`events:order`), while `ws/hub.go` subscribes to uppercase (`events:ORDER`). Make channel matching case-insensitive (`strings.ToLower`) so outbox events published to Redis correctly reach the WebSocket Hub.
2. **WebSocket Hub Monotonic Envelope & Monotonic seq Counter**:
   - In `backend/internal/ws/hub.go`: ensure all broadcasts route through `BroadcastEnvelope` or ensure every outgoing frame has monotonic `seq`, dual `event_type` and `type` fields, and `payload`.
   - Update services that bypass the envelope (`retailer`, `payout`, `returns`, `notifications`, `wmsops`, `seasonalcore`) to use `BroadcastEnvelope` or structured envelope so clients receive sequenced events.
3. **Atomic Outbox Pairing in Same pgx.Tx Transaction**:
   - In `backend/internal/warehouse/service.go`, `backend/internal/rebate/service.go`, `backend/internal/consignment/service.go`: eliminate the separated-transaction anti-pattern. Ensure entity state mutation and outbox event write happen in the EXACT SAME `pgx.Tx` closure.
   - Check and handle outbox errors: eliminate ignored errors `_ = outbox.Emit(...)` in `ump`, `inventory`, `order`, and `claims`.
   - In `backend/internal/api/handlers_fleet_driver.go:1017`: if the order was cancelled, return HTTP 409 Conflict rather than treating it as an idempotent success.
4. **Desktop Application Event Invalidation Fixes**:
   - In `warehouse-desktop/lib/fleet-ws-events.ts`: fix the deserialization bug in `parseWsEventType` so it properly parses `type` / `event_type` from the JSON envelope rather than returning the raw string.
   - In `supplier-desktop`: ensure WebSocket client connects to the active WebSocket endpoint.
   - Ensure event casing compatibility (dual support for `order.created` and `ORDER_CREATED`).
5. **Full Verification**:
   - Run `go test -v -race ./internal/outbox/... ./internal/ws/... ./internal/warehouse/... ./internal/rebate/... ./internal/consignment/... ./internal/api/...` in `pegasus.x/backend`.
   - Run `go vet ./...` and `go build ./cmd/server` to confirm clean compilation.
   - Ensure 100% test pass with 0 race conditions.
