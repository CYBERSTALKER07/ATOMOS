## 2026-09-16T13:52:34Z
MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A reviewer will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

MANDATORY FIRST STEP:
Read /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md, /Users/shakhzod/Desktop/V.O.I.D/PROJECT.md, and /Users/shakhzod/Desktop/V.O.I.D/TEST_READY.md.
Also inspect /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/api/supplier_onboarding_e2e_test.go lines 270-360 (Gate), 555-850 (Step 1 Products, Step 2 Payment, Step 3 Complete), and Tier 2 categories 3, 4, 5, 7.

TASK:
Implement Milestone 3 in `pegasus.x/backend`:
1. **Middleware `RequireSupplierOnboardingCompleted`**:
   - In `internal/auth/middleware.go` or `internal/api/router.go`:
     - Checks supplier's `OnboardingStatus` (from claims or repository lookup).
     - If `status != "COMPLETED"`, blocks operational supplier endpoints with HTTP 428 Precondition Required (`{"error": "onboarding_incomplete", "onboarding_status": "PENDING", "next_step": "/onboarding/products", "message": "supplier onboarding must be completed before accessing operational endpoints"}`).
     - Whitelists `/v1/auth/*` and `/v1/supplier/onboarding/*`.
     - Mount this middleware on the protected operational supplier sub-router in `internal/api/router.go`.
2. **Step 1: Product Catalog Management**:
   - `POST /v1/supplier/onboarding/products`:
     - Validates name (not empty).
     - Validates EAN-13 barcode: 13 digits and valid modulo-10 checksum (`soliq.ValidateEAN13` or custom checksum validator). Return 400 Bad Request on invalid format/checksum, 409 Conflict if duplicate barcode for supplier.
     - Validates statutory 17-digit MXIK code: `^[0-9]{17}$`. Return 400 Bad Request if invalid.
     - Validates package code (string, not empty).
     - Validates units_per_case (int >= 1).
     - Validates unit_price_tiyin: 64-bit integer > 0. Reject negative, float, zero with HTTP 400/422.
     - Validates 12% VAT (`vat_rate` 12.0).
     - Persists product in PostgreSQL `products` table via repository.
     - Returns HTTP 201 Created with created product.
   - `GET /v1/supplier/onboarding/products`: returns product list. HTTP 200 OK.
   - `PUT /v1/supplier/onboarding/products/{id}`: updates product. HTTP 200 OK.
   - `DELETE /v1/supplier/onboarding/products/{id}`: deletes product. HTTP 200 OK.
3. **Step 2: Payment Gateway Configuration**:
   - `POST /v1/supplier/onboarding/payment`:
     - Cash enabled by default.
     - Global Pay (`GLOBAL_PAY`) corporate card gateway configuration:
       - Service ID, secret key.
       - Corporate card BIN validation: only Uzbekistan B2B corporate card prefixes allowed (`5614` Uzcard KPK, `9860` Humo KPK, `5440` Co-badged, `4073` Visa Business, `5168` Mastercard Business).
       - If non-corporate retail BIN (e.g. `8600`) provided, reject with HTTP 400 Bad Request (`{"error": "invalid_request", "message": "only B2B corporate cards supported"}`).
     - Persists gateways to `supplier_payment_gateways` via repository.
     - Returns HTTP 200 OK.
   - `GET /v1/supplier/onboarding/payment`: returns configured gateways. HTTP 200 OK.
4. **Step 3: Complete Onboarding**:
   - `POST /v1/supplier/onboarding/complete`:
     - Validates that supplier has at least 1 active product in catalog. If 0 active products, reject with HTTP 400 Bad Request (`{"error": "precondition_failed", "message": "at least 1 active product required to complete onboarding"}`).
     - Validates payment gateway configured.
     - Transitions `onboarding_status` to `'COMPLETED'` in PostgreSQL `suppliers` table and memory cache.
     - Emits transactional outbox event and fan out to WebSocket hub.
     - Returns HTTP 200 OK (`{"supplier_id": "sup_...", "onboarding_status": "COMPLETED", "next_step": "/dashboard", "message": "onboarding completed successfully"}`).
5. Mount all `/v1/supplier/onboarding/*` endpoints in `internal/api/router.go`.
6. Run tests:
   `cd pegasus.x/backend && go test -v ./internal/api/ -run "TestSupplierOnboardingEndToEndSuite/Tier1_FeatureCoverage/Feature3_OnboardingGateMiddleware|TestSupplierOnboardingEndToEndSuite/Tier1_FeatureCoverage/Feature4_Step1ProductCatalog|TestSupplierOnboardingEndToEndSuite/Tier1_FeatureCoverage/Feature5_Step2PaymentConfiguration|TestSupplierOnboardingEndToEndSuite/Tier1_FeatureCoverage/Feature6_Step3CompleteOnboarding|TestSupplierOnboardingEndToEndSuite/Tier2_BoundaryAndCornerCases/Category3_NonIntegerFloatNegativePrices|TestSupplierOnboardingEndToEndSuite/Tier2_BoundaryAndCornerCases/Category4_Invalid17DigitMXIK|TestSupplierOnboardingEndToEndSuite/Tier2_BoundaryAndCornerCases/Category5_InvalidEAN13Barcode|TestSupplierOnboardingEndToEndSuite/Tier2_BoundaryAndCornerCases/Category7_CorporateCardBINValidation|TestSupplierOnboardingEndToEndSuite/Tier3_CrossFeatureCombinations/Scenario3_1_FullWizardSequence"`
   and verify they pass!
   Also verify `go test -v -race ./internal/supplier/...` passes with zero regressions.
