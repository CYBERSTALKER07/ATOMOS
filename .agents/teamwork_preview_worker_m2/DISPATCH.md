## 2026-09-16T13:37:08Z

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A reviewer will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

MANDATORY FIRST STEP:
Read /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md, /Users/shakhzod/Desktop/V.O.I.D/PROJECT.md, and /Users/shakhzod/Desktop/V.O.I.D/TEST_READY.md.
Also inspect /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/api/supplier_onboarding_e2e_test.go to understand the exact E2E test assertions.

TASK:
Implement Milestone 2 in `pegasus.x/backend`:
1. `POST /v1/auth/supplier/register`:
   - Acceptance:
     - Accepts JSON:
       - `company_name` (string, required)
       - `tax_id` (string, Uzbekistan 9-digit STIR/INN, required)
       - `phone` (string, `+998XXXXXXXXX`, required)
       - `password` (string, required, bcrypt hashed)
     - Validate 9-digit STIR format (`^[0-9]{9}$`). If invalid, return HTTP 400 Bad Request (`{"error": "invalid_request", "message": "tax_id must be a 9-digit number"}`).
     - Validate phone format (`^\+998[0-9]{9}$`). If invalid, return HTTP 400 Bad Request.
     - Enforce STIR uniqueness in PostgreSQL. Query `GetSupplierByTaxID` or catch unique constraint violation: if already registered, return HTTP 409 Conflict (`{"error": "conflict", "message": "tax_id already registered"}`).
     - Hash password with `bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)`.
     - Atomically persist into `suppliers` (with `tax_id`, `name = company_name`, `phone`, `password_hash`, `onboarding_status = 'PENDING'`) and `supplier_profiles` (if used).
     - Return HTTP 201 Created:
       `{"supplier_id": "sup_...", "company_name": "...", "tax_id": "...", "phone": "...", "onboarding_status": "PENDING", "next_step": "/onboarding/products"}`.
2. `POST /v1/auth/supplier/login`:
   - Acceptance:
     - Accepts JSON:
       - `tax_id` (string, 9-digit STIR)
       - `password` (string)
     - Query PostgreSQL `GetSupplierByTaxID`. If not found, return HTTP 401 Unauthorized (`{"error": "unauthorized", "message": "invalid credentials"}`).
     - Verify password using `bcrypt.CompareHashAndPassword([]byte(supplier.PasswordHash), []byte(req.Password))`. If mismatch, return HTTP 401 Unauthorized (`{"error": "unauthorized", "message": "invalid credentials"}`).
     - Issue JWT token containing claims: `supplier_id`, `role: "supplier"`, `tax_id`, `onboarding_status`.
     - Return HTTP 200 OK:
       `{"token": "<jwt>", "supplier_id": "...", "tax_id": "...", "onboarding_status": supplier.OnboardingStatus, "next_step": "/onboarding/products"}` (if COMPLETED, next_step is "/dashboard").
3. Update `pegasus.x/backend/internal/api/handlers_supplier.go` and `internal/supplier/service.go` as necessary to wire these methods.
4. Verify by running:
   - `cd pegasus.x/backend && go test -v ./internal/api/ -run "TestSupplierOnboarding/Tier_1_Feature_Coverage/F1_Supplier_Registration|TestSupplierOnboarding/Tier_1_Feature_Coverage/F2_Supplier_Login|TestSupplierOnboarding/Tier_2_Boundary_And_Corner_Cases/Category_1_Duplicate_STIR|TestSupplierOnboarding/Tier_2_Boundary_And_Corner_Cases/Category_2_Invalid_STIR"`
   - Also run `go test -v -race ./internal/supplier/...` to guarantee no regressions.

WRITE OWNERSHIP:
- `pegasus.x/backend/internal/api/handlers_supplier.go`
- `pegasus.x/backend/internal/supplier/service.go`
- `pegasus.x/backend/internal/supplier/` (if service methods are added)
