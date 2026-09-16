## 2026-09-16T13:23:40Z

You are teamwork_preview_test_writer (E2E Testing Track Lead).
Your working directory is: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_test_writer_1
Your parent is: teamwork_preview_orchestrator (conv ID: 755199e9-0b8c-404a-b2f0-93e7b22240ee)

MANDATORY FIRST STEP:
Read /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md and /Users/shakhzod/Desktop/V.O.I.D/PROJECT.md completely.

TASK:
Design and author the complete, comprehensive opaque-box E2E test suite for Supplier Sign-Up, Non-Bypassable Onboarding, and Warehouse/Fleet Management in `pegasus.x/backend`.

REQUIREMENTS & TEST TIERS:
Derive test cases strictly from requirements (opaque-box, requirement-driven):
1. **Tier 1 - Feature Coverage (>=5 per feature)**:
   - Supplier Registration: valid fields, standard 201 response, `onboarding_status: PENDING`.
   - Supplier Login: valid credentials, JWT returned, `onboarding_status: PENDING`, `next_step: /onboarding/products`.
   - Onboarding Gate Middleware: HTTP 428 Precondition Required on protected endpoints; whitelist `/v1/auth/*` and `/v1/supplier/onboarding/*`.
   - Step 1 Products: Add, Edit, Delete with EAN-13, 17-digit MXIK, package code, units_per_case, 64-bit integer tiyin price, 12% VAT. Verify gate requirement (>= 1 active product).
   - Step 2 Payment: Cash default enabled, Global Pay (`GLOBAL_PAY`) corporate card gateway setup with service ID, secret key, and corporate card BIN validation.
   - Step 3 Complete: status transition to 'COMPLETED', unblocking gate, event emission.
   - Warehouse Management: Add, Get, Put, Delete with mandatory `latitude` and `longitude` (`DOUBLE PRECISION`).
   - Fleet Management: Add, Get trucks (license plate, capacity kg/m3, fuel type).
   - Dock Payloaders: Add, Get payloaders (name, phone, warehouse_id).
2. **Tier 2 - Boundary & Corner Cases (>=5 per feature)**:
   - Duplicate STIR registration -> HTTP 409 Conflict.
   - Invalid STIR (non-9 digits, letters, empty) -> HTTP 400 Bad Request.
   - Non-integer / float / negative prices -> rejection.
   - Invalid MXIK code (not 17 digits, special chars) -> rejection.
   - Invalid EAN-13 (bad length, checksum mismatch) -> rejection.
   - Warehouse deletion with on-hand stock > 0 or active orders -> HTTP 409 Conflict.
   - Invalid corporate card BINs (non-corporate retail card) -> rejection.
   - Missing required coordinates (lat/lon null, out of range [-90,90], [-180,180]) -> rejection.
3. **Tier 3 - Cross-Feature Combinations (Pairwise)**:
   - Full wizard sequence: Register -> Login -> Try GET /v1/supplier/warehouses (receive 428) -> Step 1 Products -> Step 2 Payment -> Step 3 Complete -> Retry GET /v1/supplier/warehouses (receive 200).
   - Warehouse creation -> Truck assignment -> Payloader assignment -> Delete attempt with active stock (409) -> Delete with zero stock (200).
4. **Tier 4 - Real-World Application Scenario**:
   - Complete multi-step enterprise onboarding simulation for "OOO Samarkand Logistics".

DELIVERABLES:
1. Write test code in `pegasus.x/backend/internal/api/supplier_onboarding_e2e_test.go` (using Go `testing`, `net/http/httptest`, and the Chi router / handlers).
2. Create `/Users/shakhzod/Desktop/V.O.I.D/TEST_INFRA.md` following the project template.
3. Create `/Users/shakhzod/Desktop/V.O.I.D/TEST_READY.md` summarizing runner commands, tier counts, and feature checklist.
4. Run `go test -v ./internal/api/ -run "TestSupplier"` or targeted test runner to verify compilation.
5. Write handoff report in your working directory and notify parent.
