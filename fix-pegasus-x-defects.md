# Fix Pegasus.X Core Defects

## Goal
Remediate the four critical defects identified in `pegasus.x` (Kafka import contamination, Redis 7 Streams persistence, dispatch safety gate loophole, and desktop vehicles UI wiring) to restore clean compilation and operational integrity.

## Tasks
- [x] Task 1: Remove Kafka import contamination and dead code in `pegasus.x/backend/internal/outbox/relay.go` (line 13) and `pegasus.x/backend/cmd/server/main.go` (lines 20, 85-97) → Verify: `cd pegasus.x/backend && go test ./internal/outbox/...` compiles and passes.
- [x] Task 2: Implement persistent Redis 7 Streams method `XAddFleetEvent` in `pegasus.x/backend/internal/redis/client.go` with capped trimming (`MAXLEN ~ 100000`) → Verify: `cd pegasus.x/backend && go build ./internal/redis/...` succeeds without errors.
- [x] Task 3: Replace ephemeral `s.redis.Publish(ctx, "events:FLEET", ...)` in `pegasus.x/backend/internal/fleet/service.go` (lines 353, 372, 453, 518, 597) with persistent `XAddFleetEvent` → Verify: `cd pegasus.x/backend && go test ./internal/fleet/...` passes.
- [x] Task 4: Harden pre-trip safety gate in `pegasus.x/backend/internal/dispatch/service.go` (line 258) by enforcing non-null `is_safe_to_operate = true`, `inspection_type = 'PRE_TRIP'`, latest inspection deduplication via `LEFT JOIN LATERAL`, and `Asia/Tashkent` date boundary anchoring → Verify: `cd pegasus.x/backend && go test ./internal/dispatch/...` passes.
- [x] Task 5: Run full backend test and verification suite across all packages in `pegasus.x/backend` → Verify: `cd pegasus.x/backend && go test ./...` exits with code 0.
- [x] Task 6: Audit `pegasus.x/apps/warehouse-desktop/app/vehicles/page.tsx` for 2-column shift pairing interaction and UI design compliance → Verify: Check typecheck/build in `pegasus.x/apps/warehouse-desktop`.

## Done When
- [x] `cd pegasus.x/backend && go test ./...` exits with code 0 (zero compiler errors, no Kafka packages).
- [x] Zero Kafka imports remain anywhere in `pegasus.x/backend`.
- [x] Fleet events persist through Redis 7 Streams (`events:fleet`).
- [x] Dispatch pre-trip check strictly blocks uninspected or unsafe vehicles.

## Notes
- `pegasus.x` is strictly sovereign: NEVER import Kafka or Spanner libraries.
- All timestamps for shift and DVIR gating must account for Uzbekistan time (`Asia/Tashkent` / UTC+5).
