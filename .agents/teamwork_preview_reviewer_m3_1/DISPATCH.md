## 2026-09-16T14:04:07Z
You are teamwork_preview_reviewer (Milestone 3 Reviewer 1: Onboarding Gate & Wizard Reviewer).
Your working directory is: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_reviewer_m3_1
Your parent is: teamwork_preview_orchestrator (conv ID: 755199e9-0b8c-404a-b2f0-93e7b22240ee)

MANDATORY FIRST STEP:
Read /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md and /Users/shakhzod/Desktop/V.O.I.D/PROJECT.md.

TASK:
Review Milestone 3 (Non-Bypassable Onboarding Gate & Phased Wizard):
1. Review Worker M3 Handoff: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m3/handoff.md`.
2. Inspect `pegasus.x/backend/internal/api/router.go` and `handlers_supplier.go`:
   - Verify `requireSupplierOnboardingCompleted`: blocks operational endpoints with HTTP 428 (`onboarding_incomplete`), whitelists `/v1/auth/*` and `/v1/supplier/onboarding/*`.
   - Verify Step 1 Products: Name, EAN-13 modulo-10 checksum, 17-digit statutory MXIK `^[0-9]{17}$`, 64-bit integer tiyins (rejection of floats/strings/negatives), 12% VAT, package code, units_per_case, duplicate barcode 409.
   - Verify Step 2 Payment: Cash default, Global Pay setup, and B2B corporate card BIN validation (`5614`, `9860`, `5440`, `4073`, `5168`), rejecting retail BINs (`8600`) with 400.
   - Verify Step 3 Complete: >= 1 active product check, status transition to `'COMPLETED'`, gate unblocking, outbox/WS event.
3. Run verification tests:
   `cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend`
   `go test -v -count=1 ./internal/api/ -run "TestSupplierOnboardingEndToEndSuite/Tier1_FeatureCoverage/Feature3_OnboardingGateMiddleware|TestSupplierOnboardingEndToEndSuite/Tier1_FeatureCoverage/Feature4_Step1ProductsCatalog|TestSupplierOnboardingEndToEndSuite/Tier1_FeatureCoverage/Feature5_Step2PaymentConfiguration|TestSupplierOnboardingEndToEndSuite/Tier1_FeatureCoverage/Feature6_Step3CompleteOnboarding|TestSupplierOnboardingEndToEndSuite/Tier2_BoundaryAndCornerCases/Category3_PriceValidationRules|TestSupplierOnboardingEndToEndSuite/Tier2_BoundaryAndCornerCases/Category4_InvalidMXIKCode|TestSupplierOnboardingEndToEndSuite/Tier2_BoundaryAndCornerCases/Category5_InvalidEAN13Barcode|TestSupplierOnboardingEndToEndSuite/Tier2_BoundaryAndCornerCases/Category7_CorporateCardBINValidation|TestSupplierOnboardingEndToEndSuite/Tier3_CrossFeatureCombinations/Scenario3_1_FullWizardGateSequence"`
   `go test -v -race ./internal/supplier/...`
   `go build ./...`

OUTPUT:
Write your review report and `handoff.md` with clear verdict (`APPROVE` or `REQUEST_CHANGES`).
Send a message to parent with the verdict and summary.
