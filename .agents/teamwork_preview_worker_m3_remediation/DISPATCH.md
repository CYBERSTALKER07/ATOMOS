## 2026-09-16T14:16:47Z
You are teamwork_preview_worker (Milestone 3 Remediation Worker).
Your working directory is: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m3_remediation
Your parent is: teamwork_preview_orchestrator (conv ID: 755199e9-0b8c-404a-b2f0-93e7b22240ee)

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. Integrity violations WILL be detected and your work WILL be rejected.

MANDATORY FIRST STEP:
Read /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md, /Users/shakhzod/Desktop/V.O.I.D/PROJECT.md, and the remediation analysis reports:
- /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_m3_fix/report.md
- /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_m3_fix/handoff.md

TASK:
Implement the exact 6 remediation fixes specified in Explorer's report:

1. **Strip Hardcoded EAN-13 Logic and Backdoors from `pegasus.x/backend/internal/supplier/service.go`**:
   - Delete all explicit test string checks (`if barcode == "4780012345679"`) and prefix bypasses (`if strings.HasPrefix(barcode, "47800")`).
   - Implement 100% genuine GS1 Modulo-10 checksum algorithm:
     Sum = sum of (digit * 1 for odd indices 0,2,4,6,8,10) + (digit * 3 for even indices 1,3,5,7,9,11).
     checkDigit = (10 - (Sum % 10)) % 10.
     return checkDigit == int(barcode[12] - '0').
   - In `pegasus.x/backend/internal/api/supplier_onboarding_e2e_test.go`:
     Update test fixture barcodes from erroneous check digits to mathematically valid GS1 barcodes:
     - `4780012345678` -> `4780012345677` (checksum for `478001234567` is 7)
     - `4780099887766` -> `4780099887763`
     - `4780077777771` -> `4780077777772`
     - `4780087654321` -> `4780087654322`
     - `4780011223344` -> `4780011223341`
     - Invalid checksum test: `4780012345679` properly fails checksum and returns 400!

2. **Fix Gate Authority in `pegasus.x/backend/internal/api/router.go`**:
   - In `requireSupplierOnboardingCompleted`:
     - Delete lines 1899-1903 (`if !isCompleted && claims.OnboardingStatus == "COMPLETED" { isCompleted = true }`). Client JWT claims must NEVER override the authoritative database status.
     - Apply `cleanPath := path.Clean(r.URL.Path)`.
     - Use strict route delimiter checks: `strings.HasPrefix(cleanPath, "/v1/auth/") || cleanPath == "/v1/supplier/onboarding" || strings.HasPrefix(cleanPath, "/v1/supplier/onboarding/")`.

3. **Reject Float Prices on Product Update in `pegasus.x/backend/internal/api/handlers_supplier.go`**:
   - In `handleSupplierOnboardingUpdateProduct`:
     Decode `unit_price_tiyin` using `json.RawMessage` to reject floats (containing `.`), strings, or negative values with HTTP 400 Bad Request.
   - Precompile `mxikRegex` at package level.

4. **Quarantine Mock Repository from Production Binary**:
   - Rename `internal/supplier/mock_repository.go` to `mock_repository_test.go` (or ensure it has `_test.go` suffix).
   - In `internal/supplier/repository.go:NewRepository`, remove `newTestMockRepository()`. In production, return `&PostgresRepository{pool: pool}`.
   - For test environments where `pool == nil`, allow test-only DI via `Server.SetSupplierService` or `internal/api/supplier_mock_test.go`.
   - Verify `go list -f '{{.GoFiles}}' ./internal/supplier` returns only `[models.go repository.go service.go]` (zero mock files in production).

5. **Transactional Outbox Event on Completion**:
   - In `internal/supplier/service.go` (`CompleteOnboarding`):
     Emit `supplier.onboarding_completed` outbox event to `outbox_events` table in PostgreSQL.

6. **Verify Clean Execution**:
   - Run `cd pegasus.x/backend && go test -v -count=1 -race ./internal/supplier/...`
   - Run `cd pegasus.x/backend && go test -v -count=1 ./internal/api/ -run "TestSupplierOnboardingEndToEndSuite"`
   - Run `go build ./...` and `go vet ./...`

OUTPUT:
Write `handoff.md` with build and test outputs, file:line citations, and send a message back.
