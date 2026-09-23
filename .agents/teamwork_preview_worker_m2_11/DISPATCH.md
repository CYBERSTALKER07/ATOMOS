## 2026-09-23T11:27:48Z

You are teamwork_preview_worker_m2_11.
Your working directory is:
/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m2_11

Your parent conversation ID is:
d877571c-b5bd-4489-a1cb-441c991bb03d

Authoritative user request (read this file first):
/Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md

Project scope document:
/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_11/PROJECT.md

Explorer 2 Survey Analysis and Handoff Report (read carefully for exact line numbers and formula replacements):
/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_11_2/analysis.md
/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_11_2/handoff.md

Domain skill:
Read and follow instructions in /Users/shakhzod/.gemini/config/skills/golang-pro/SKILL.md

Target codebase directory:
/Users/shakhzod/Desktop/V.O.I.D/pegasus.x

Your Mission (Milestone 2 — Requirement R1.1 & R1.2):
Refactor currency arithmetic to strict 64-bit integer tiyin minor units (int64) with ZERO floating-point math, and enforce domain state machines and concurrency controls on mutations.

Tasks to execute:
1. **Currency Arithmetic Hardening (Zero Floating-Point Money Math)**:
   - In `backend/internal/soliq/efactura.go`: replace any float VAT arithmetic with statutory Uzbekistan integer round-half-up math: `(price * 12 + 50) / 100` in `int64` tiyins.
   - In `backend/internal/rebate/rebate.go`: convert float percentages to integer basis points (`int64`, 1 basis point = 0.01%, 10,000 bp = 100%).
   - In `backend/internal/fscm/dunning.go`: replace float rate penalties with integer basis point arithmetic in `int64`.
   - In `backend/internal/copa/copa.go`: replace float fee calculations with integer basis points in `int64`.
   - In `backend/internal/ar/dunning.go`: convert bad debt reserve percentages to integer basis points in `int64`.
   - In `backend/internal/matching/matching.go`: ensure line totals and discrepancies are strictly `int64` tiyins.
   - In `backend/internal/consignment/consignment.go`: base cost calculations in `int64` tiyins.
   - In `backend/internal/supplier/service.go`: volume discount calculations in integer basis points / `int64` tiyins.
   - In `backend/internal/payout/calculator.go`: holdback reserves in integer basis points / `int64` tiyins.
   - In `backend/internal/api/handlers_supplier.go:1107`: update `VatRate` in DTO to handle both int64 basis points and backward-compatible input without float arithmetic in financial logic.
2. **Domain State Machine & Concurrency Purity**:
   - In `backend/internal/api/handlers_fleet_driver.go:986` (`handleOrderComplete`): ensure order state updates execute through the domain order state machine / service with status validation rather than bypassing domain checks.
   - In `backend/internal/epod/repository.go:211` (`UpdateStopStatus`): validate valid status transitions.
   - In `backend/internal/wmsops/repository.go:1550-1563`: fix TOCTOU concurrency race by ensuring atomic conditional updates or `FOR UPDATE` locking.
   - In `backend/internal/api/handlers_payment.go` and `backend/internal/fleet/repository.go`: fix unhandled database mutation errors (`_, _ = tx.Exec(...)`).
   - In `backend/internal/ewm/`: move in-memory repository mock code to `*_test.go` or quarantine so production compiles cleanly without mock fallbacks.
3. **Verification**:
   - Run tests: `go test -v -race ./internal/soliq/... ./internal/rebate/... ./internal/fscm/... ./internal/copa/... ./internal/ar/... ./internal/matching/... ./internal/consignment/... ./internal/supplier/... ./internal/payout/... ./internal/epod/... ./internal/wmsops/...` in `pegasus.x/backend`.
   - Run `go vet ./...` and `go build ./cmd/server` to ensure clean build.
   - Ensure 100% test pass with 0 race conditions.
