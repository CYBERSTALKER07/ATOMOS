## 2026-09-23T14:19:16Z

You are teamwork_preview_explorer_remediation_11.
Your working directory is:
/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_remediation_11

Your parent conversation ID is:
d877571c-b5bd-4489-a1cb-441c991bb03d

Authoritative user request (read this file first):
/Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md

Project scope document:
/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_11/PROJECT.md

FULL FORENSIC AUDITOR EVIDENCE REPORT (READ IN FULL — NON-NEGOTIABLE):
/Users/shakhzod/Desktop/V.O.I.D/.agents/victory_auditor_orch_2/handoff.md

Target codebase directory:
/Users/shakhzod/Desktop/V.O.I.D/pegasus.x

Your Mission (Forensic Audit Remediation Planning):
Investigate and formulate a comprehensive, surgical, and robust remediation plan for ALL integrity violations and doctrine breaches identified by the Independent Victory Auditor:

1. **Disguised In-Memory Repositories in Production Files**:
   - Inspect:
     - `internal/cyclecount/repository.go:47-194` (`MemoryCycleCountRepo`, `memFallback`)
     - `internal/transfer/repository.go:45-185` (`MemoryTransferRepo`, `memFallback`)
     - `internal/empties/service.go:60-223` (`MemoryEmptiesRepo`, `memFallback`)
     - `internal/qm/service.go:21-80` (`MemoryQMRepo`, `NewMemoryQMRepo()`)
   - Check where they are wired in `internal/api/router.go:173-176, 230`.
   - Inspect `internal/promotion/repository.go` and `internal/commitments/repository.go` for unpersisted in-memory stores and dummy seeds.
   - Plan their complete elimination from production `.go` files, moving mocks strictly to `*_test.go`, and ensuring clean PostgreSQL 16 persistence.

2. **Missing Fail-Closed `*db.Pool` Validation**:
   - Inspect `internal/order/service.go:192` (`s.initInMemory()`, `s.inMemoryOrders`). Plan its complete removal so `order.NewService` panics if `pool == nil` and orders can NEVER be processed in volatile memory.
   - Inspect `internal/credit/service.go` (`inMemoryApps`, `inMemoryLines`, `inMemoryDebts`).
   - Audit all constructors across `backend/internal/` and list every constructor that needs `if pool == nil { panic("database connection required") }`.

3. **Currency Arithmetic & VAT Rounding**:
   - Inspect `internal/retailer/repository.go:2884`: `vat := (tot * 12) / 112` and plan replacement with statutory integer round-half-up math: `(tot*12 + 56) / 112`.
   - Inspect `internal/qm/quarantine.go:79`: plan elimination of `math.Round(float64(unitCostMinor) * qty)`.

4. **Non-Atomic Outbox Closures**:
   - Inspect `internal/warehouse/service.go:119-130, 398-412`. Plan elimination of the dual-transaction fallback so `warehouse.Service` requires `TxRepository` and entity mutation + outbox emission are 100% atomic.
   - Inspect `internal/api/handlers_soliq.go:147` and plan error propagation.

5. **Monorepo Compilation & Tests Impact**:
   - Map all affected test files, callers, and router wirings.
   - Design step-by-step worker assignments with exact code changes and verification commands.

Output requirements:
- Write `analysis.md` with deep investigation and file:line citations.
- Write `handoff.md` with complete Observation, Logic Chain, Caveats, Conclusion, and Concrete Fix Strategy.
- Send a message to parent (`d877571c-b5bd-4489-a1cb-441c991bb03d`) when done with summary and path to handoff.md.
