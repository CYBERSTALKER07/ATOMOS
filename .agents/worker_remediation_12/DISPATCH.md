# DISPATCH — worker_remediation_12

**Role**: Senior Systems Engineer & Remediator (`teamwork_preview_worker`)  
**Working Directory**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/worker_remediation_12`  
**Parent Conversation ID**: `4b03ea3e-5816-418c-b738-53f7fc07c73e`  
**Target Directory**: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x` (`backend`)  
**Skill**: `/Users/shakhzod/.gemini/config/skills/golang-pro/SKILL.md`  

## 2026-09-23T16:43:34Z
<USER_REQUEST>
You are worker_remediation_12, a Senior Systems Engineer & Remediator.
Your working directory is: /Users/shakhzod/Desktop/V.O.I.D/.agents/worker_remediation_12
Your parent conversation ID is: 4b03ea3e-5816-418c-b738-53f7fc07c73e
Domain Skill to use: /Users/shakhzod/.gemini/config/skills/golang-pro/SKILL.md

MANDATORY FIRST STEP:
Read these reference documents:
1. /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md
2. /Users/shakhzod/Desktop/V.O.I.D/.agents/victory_auditor_orch_2/handoff.md
3. /Users/shakhzod/Desktop/V.O.I.D/.agents/worker_remediation_12/DISPATCH.md
4. /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_12/plan.md

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A teamwork_preview_auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

SCOPE & WORK ITEMS (Target: /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend):

1. Purge Disguised In-Memory Repositories from Production Files:
   - internal/cyclecount/repository.go: Delete type MemoryCycleCountRepo struct and memFallback. Move unit test doubles strictly to repository_test.go. Require pool != nil.
   - internal/transfer/repository.go: Delete type MemoryTransferRepo struct and memFallback. Move unit test doubles strictly to repository_test.go. Require pool != nil.
   - internal/empties/service.go: Delete type MemoryEmptiesRepo struct and eliminate silent fallbacks on DB error. Move unit test doubles strictly to *_test.go. Require pool != nil.
   - internal/qm/service.go: Delete type MemoryQMRepo struct and NewMemoryQMRepo. Move unit test doubles strictly to *_test.go. In internal/api/router.go, ensure QM service uses genuine postgres repository (e.g. qm.NewPostgresQMRepo(pool)).
   - internal/promotion/repository.go & internal/commitments/repository.go: Implement genuine PostgreSQL 16 persistence for promotions and commitments. Remove seedInMemory() and in-memory mock storage in production files. Ensure constructors require pool != nil.

2. Enforce Fail-Closed Constructors (*db.Pool) Across All Packages:
   - internal/order/service.go: NewService MUST panic if pool == nil (if pool == nil { panic("pool cannot be nil") }). Eliminate s.inMemoryOrders and s.initInMemory() completely from production code. If existing unit tests pass nil pool, update those unit tests to provide a test mock or valid pool mock.
   - internal/credit/service.go: NewService MUST panic if pool == nil. Eliminate in-memory maps from production code.
   - Audit and enforce if pool == nil { panic("...") } on all remaining constructors: crossdock.NewRepository, controltower.NewRepository, promotion.NewRepository, commitments.NewRepository, empties.NewPgEmptiesRepo, cyclecount.NewRepository, transfer.NewRepository, qm.NewPostgresQMRepo, dock.NewRepository, claims.NewRepository, wms.NewService, manifest.NewService, creditnote.NewService, cashrecon.NewService, etc.

3. Currency & VAT Arithmetic Fixes:
   - internal/retailer/repository.go:2884: Replace vat := (tot * 12) / 112 with statutory round-half-up integer math: vat := (tot * 12 + 56) / 112.
   - internal/qm/quarantine.go:79: Eliminate float64 conversion (math.Round(float64(unitCostMinor) * qty)) and enforce strict 64-bit integer tiyin minor unit arithmetic.

4. Real-Time Outbox Atomicity:
   - internal/warehouse/service.go: Require TxRepository and eliminate the dual-transaction fallback in service.go:119-130, 398-412.
   - internal/api/handlers_soliq.go:147: Do not discard transaction error with _ = s.pool.RunInTx(...). Return and handle the error appropriately.

5. Automated Verification & Gates:
   - Run go build -v ./cmd/server (must exit 0)
   - Run go build -v ./cmd/smokecheck (must exit 0)
   - Run go vet ./... (must exit 0 with 0 diagnostics)
   - Run go test -v -race ./... across all backend packages
   - Run scale benchmarks:
     * go test -v -run TestSmartDispatch_1000Orders_H3MacroClustering_ZeroAbandoned ./internal/dispatch/...
     * go test -v -run TestSmartDispatch_100Orders_ZeroAbandonedGuarantee ./internal/dispatch/...
     * go test -v -run "TestAxleFeasibility|TestPayload_AxleOverride" ./internal/payload/...
     * go test -v -run "Rescue" ./internal/dispatch/...
</USER_REQUEST>
