# Reviewer M1 Context: Backend Domain Subrouter & Route Module Decomposition

Working Directory: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_reviewer_m1_13`
Target Workspace Directory: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`
Authoritative Requirements: `/Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md`
Worker Handoff: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m1_13/handoff.md`
Project Plan: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_13/PROJECT.md`

Mission:
Conduct an independent, rigorous code review and adversarial verification of Milestone 1:
1. Verify `backend/internal/api/router.go` line count:
   - Must be reduced by at least 60% (from 2,448 lines to under 950 lines, target ~765-805 lines).
2. Verify `backend/internal/api/modules/module.go`:
   - Proper `Module` interface and `Registry` implementation without circular imports.
3. Verify the 5 domain subrouter files:
   - `core.go`, `logistics.go`, `warehouse.go`, `commercial.go`, `finance.go`
   - Check that route registrations match the exact HTTP paths, middleware, and handlers previously in `router.go`.
4. Verify `backend/internal/api/dto.go`:
   - Centralized DTO definitions match expected JSON tags and types.
5. Independent Test Execution:
   - Run `go vet ./...` in `backend/` -> must exit 0 with 0 diagnostics.
   - Run `go test -v ./internal/api/...` -> all tests pass.
   - Run `go test -v -race -run TestWarehouseStockManagement_E2E ./internal/api/...` (and any other e2e/race tests) -> must pass with 0 race conditions.
6. Verify Sovereign Core constraints:
   - Strictly PostgreSQL 16 + Redis 7 Streams.
   - Zero Spanner or Kafka imports.
   - Zero float currency math (strict 64-bit integer tiyins).
7. Issue Verdict in your `handoff.md`:
   - Record clear verdict: **APPROVE** or **REQUEST_CHANGES**.
   - Send completion message to parent via `send_message`.
