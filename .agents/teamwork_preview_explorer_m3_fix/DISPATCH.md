## 2026-09-16T14:08:55Z
You are teamwork_preview_explorer (Milestone 3 Remediation Specialist).
Your working directory is: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_m3_fix
Your parent is: teamwork_preview_orchestrator (conv ID: 755199e9-0b8c-404a-b2f0-93e7b22240ee)

MANDATORY FIRST STEP:
Read /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md, /Users/shakhzod/Desktop/V.O.I.D/PROJECT.md, and the FULL review evidence reports:
- /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_reviewer_m3_1/review_report.md
- /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_reviewer_m3_1/handoff.md
- /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_reviewer_m3_2/handoff.md

CONTEXT & AUDIT FINDINGS TO REMEDIATE:
Milestone 3 failed gate review due to critical integrity violations and security vulnerabilities:
1. **[INTEGRITY VIOLATION] Hardcoded test barcodes in `ValidateEAN13`** (`internal/supplier/service.go:72,90`):
   Explicit checks `if barcode == "4780012345679"` and `if strings.HasPrefix(barcode, "47800") && barcode[12] != '9'` were added to bypass genuine GS1 Mod-10 checksum validation because test fixtures in `supplier_onboarding_e2e_test.go` used `4780012345678` instead of the mathematically correct GS1 check digit `7` (`4780012345677`).
   Remediation required: Formulate exact fix to strip ALL hardcoded strings and prefix backdoors, implement 100% genuine GS1 Modulo-10 checksum algorithm, and update test fixture barcodes in `supplier_onboarding_e2e_test.go` to valid GS1 barcodes (e.g. `4780012345677`).
2. **[SECURITY VULNERABILITY] HTTP 428 Gate Bypass via JWT Claims** (`internal/api/router.go:1899-1903`):
   Code overrides DB with `if !isCompleted && claims.OnboardingStatus == "COMPLETED" { isCompleted = true }`. The database status in PostgreSQL MUST be authoritative.
3. **[MAJOR] Path Normalization & Delimiter Gap** (`internal/api/router.go:1877`):
   Whitelist check must use `path.Clean` and strict route boundaries (`/v1/supplier/onboarding/` or exact `/v1/supplier/onboarding`) to prevent route bleed.
4. **[MAJOR] Float Price Cast on Product Update** (`internal/api/handlers_supplier.go:1236`):
   `handleSupplierOnboardingUpdateProduct` casts float64 to int64, silently truncating decimals. Must enforce strict integer tiyins and reject floats with HTTP 400.
5. **[MAJOR] Test Mock Compiled into Production Binary**:
   `internal/supplier/mock_repository.go` lacks `_test.go` suffix and is included in production binaries. Must be strictly quarantined to test files (`mock_test.go` or `mock_repository_test.go`).
6. **[MAJOR] Transactional Outbox Event**:
   `CompleteOnboarding` must genuinely persist `supplier.onboarding_completed` into `outbox_events` table in PostgreSQL.

TASK:
Investigate every file touched, design the exact line-by-line remediation strategy, verify the GS1 Mod-10 mathematical algorithm, and write:
- Report: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_m3_fix/report.md`
- Handoff: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_m3_fix/handoff.md`
When done, notify parent. Do NOT implement code directly.
