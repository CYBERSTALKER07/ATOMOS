# TEST_READY: Supplier Onboarding & Fleet Hub in pegasus.x

## Status: E2E Test Suite Authored & Compilation Verified (TDD Red Baseline Ready)

The comprehensive opaque-box E2E test suite for Supplier Sign-Up, Non-Bypassable Phased Onboarding, and Warehouse/Fleet Management in `pegasus.x/backend` has been designed, authored, and verified to compile cleanly with zero errors.

---

## 1. Test Runner Commands

| Scope | Execution Command |
|---|---|
| **Compilation Verification** | `cd pegasus.x/backend && go test -c ./internal/api/` |
| **Targeted Onboarding E2E Suite** | `cd pegasus.x/backend && go test -v ./internal/api/ -run "TestSupplierOnboarding"` |
| **Full Supplier Test Suite** | `cd pegasus.x/backend && go test -v ./internal/api/ -run "TestSupplier"` |
| **Race Detector Verification** | `cd pegasus.x/backend && go test -v -race ./internal/api/ -run "TestSupplierOnboarding"` |

---

## 2. Test Tier Counts & Breakdown

| Tier | Category / Feature | Test Cases / Subtests | Status (Current Red Baseline) |
|---|---|:---:|:---:|
| **Tier 1** | F1: Supplier Registration (POST /v1/auth/supplier/register) | 5 | Failing (Pending M2 implementation) |
| **Tier 1** | F2: Supplier Login (POST /v1/auth/supplier/login) | 5 | Failing (Pending M2 implementation) |
| **Tier 1** | F3: Onboarding Gate Middleware (HTTP 428 Precondition Required) | 5 | Failing (Pending M3 implementation) |
| **Tier 1** | F4: Step 1 Products Catalog (POST/GET/PUT/DELETE /v1/supplier/onboarding/products) | 5 | Failing (Pending M3 implementation) |
| **Tier 1** | F5: Step 2 Payment Configuration (Cash + Global Pay B2B) | 5 | Failing (Pending M3 implementation) |
| **Tier 1** | F6: Step 3 Complete Onboarding (Transition to COMPLETED, Gate Unblock) | 5 | Failing (Pending M3 implementation) |
| **Tier 1** | F7: Warehouse Management (POST/GET/PUT/DELETE /v1/supplier/warehouses) | 5 | Failing (Pending M4 implementation) |
| **Tier 1** | F8: Fleet Management (Trucks: license plate, capacity, fuel type) | 5 | Failing (Pending M4 implementation) |
| **Tier 1** | F9: Dock Payloaders (name, phone, warehouse association) | 5 | Failing (Pending M4 implementation) |
| **Tier 2** | Category 1: Duplicate STIR Registration (409 Conflict) | 5 | Failing (Pending M2 implementation) |
| **Tier 2** | Category 2: Invalid STIR Formats (400 Bad Request) | 5 | Failing (Pending M2 implementation) |
| **Tier 2** | Category 3: Non-Integer / Float / Negative Prices (400/422 Rejection) | 5 | Failing (Pending M3 implementation) |
| **Tier 2** | Category 4: Invalid 17-Digit MXIK Code (400/422 Rejection) | 5 | Failing (Pending M3 implementation) |
| **Tier 2** | Category 5: Invalid EAN-13 Barcode & Modulo 10 (400/409/422) | 5 | Failing (Pending M3 implementation) |
| **Tier 2** | Category 6: Warehouse Deletion Guard (Active Stock/Orders -> 409) | 5 | Passing (Safety fallback) |
| **Tier 2** | Category 7: Corporate Card BIN Validation (B2B Humo/Uzcard) | 5 | Failing (Pending M3 implementation) |
| **Tier 2** | Category 8: Missing / Out-of-Range GPS Coordinates (400/422) | 5 | Failing (Pending M4 implementation) |
| **Tier 3** | Scenario 3.1: Full Wizard Sequence with Gate Interception | 7 steps | Failing (Pending M2-M4 integration) |
| **Tier 3** | Scenario 3.2: Warehouse, Fleet, and Deletion Lifecycle | 4 steps | Failing (Pending M4 implementation) |
| **Tier 4** | Scenario 4.1: OOO Samarkand Logistics Enterprise Onboarding Simulation | 10 stages | Failing (Pending M2-M4 integration) |
| **TOTAL** | **Comprehensive Opaque-Box Test Suite** | **88 Tests / Scenarios** | **Baseline Ready for TDD Implementation** |

---

## 3. Feature Checklist Mapped to Milestones

- [ ] **Milestone 1 (Migration 069 & Repo Purge)**:
  - Database schema: `069_supplier_onboarding_and_globalpay.sql` created and applied.
  - Repository purge: `MemoryRepository` and silent fallbacks deleted from `internal/supplier/repository.go`.
  - Pure PostgreSQL 16 persistence via `pgxpool`.
- [ ] **Milestone 2 (Supplier Sign-Up & Sign-In)**:
  - `POST /v1/auth/supplier/register`: 9-digit STIR deduplication (409 Conflict), bcrypt hashing, `onboarding_status: "PENDING"`, `next_step: "/onboarding/products"`.
  - `POST /v1/auth/supplier/login`: Authenticate by `tax_id` + `password`, issue JWT with claims, return `onboarding_status`.
- [ ] **Milestone 3 (Non-Bypassable Onboarding Gate & Wizard)**:
  - Middleware: `RequireSupplierOnboardingCompleted` intercepting operational endpoints with HTTP 428 (`onboarding_incomplete`), whitelisting `/v1/auth/*` and `/v1/supplier/onboarding/*`.
  - Step 1 Catalog: `POST/GET/PUT/DELETE /v1/supplier/onboarding/products` with EAN-13, 17-digit MXIK, 64-bit integer tiyin price, 12% VAT, gate rule requiring >=1 active product.
  - Step 2 Payment: `POST/GET /v1/supplier/onboarding/payment` with Cash enabled, Global Pay (`GLOBAL_PAY`) corporate card BIN validation (`5614`, `9860`, `5440`, `4073`, `5168`).
  - Step 3 Complete: `POST /v1/supplier/onboarding/complete` transitioning status to `COMPLETED`, unblocking operational gate, emitting outbox event.
- [ ] **Milestone 4 (Warehouse & Fleet Hub)**:
  - `POST/GET/PUT/DELETE /v1/supplier/warehouses`: Mandatory `latitude` and `longitude` (`DOUBLE PRECISION`), Redis proximity invalidation, `warehouse.relocated` event, 409 deletion guard on active stock/orders.
  - `POST/GET /v1/supplier/warehouses/{id}/trucks`: License plate, capacity kg/m³, fuel type.
  - `POST/GET /v1/supplier/warehouses/{id}/payloaders`: Name, Uzbekistan phone, warehouse association.
- [ ] **Milestone 5 (Final E2E Pass & Hardening)**:
  - 100% of 88 test cases in `TestSupplierOnboardingEndToEndSuite` passing with `go test -v -race ./...`.

---

## 4. Implementation Gaps & Discrepancies Escalated

The initial test run of `TestSupplierOnboardingEndToEndSuite` surfaced the following concrete implementation gaps to be addressed during Milestones M1–M4:

1. **Auth Contract Discrepancy**:
   - `handleSupplierRegister` in `handlers_supplier.go:90-162` currently reads `req.Account.LegalName` instead of `req.Company_Name`, ignores `password` (no bcrypt hashing), does not enforce 9-digit STIR uniqueness, and returns `next_step: "/setup/business"` instead of `next_step: "/onboarding/products"`.
   - `handleSupplierLogin` in `handlers_supplier.go:169-208` requires `phone` instead of `tax_id` and hardcodes `sid := "sup_pepsico_uz"` without querying the database or checking password hashes.
2. **Missing Gate Middleware**:
   - `RequireSupplierOnboardingCompleted` middleware is not yet installed in Chi router (`router.go:536`). Protected supplier endpoints currently allow access even when onboarding is incomplete.
3. **Unrouted Onboarding Endpoints**:
   - `/v1/supplier/onboarding/products`, `/v1/supplier/onboarding/payment`, and `/v1/supplier/onboarding/complete` are not yet mounted on the Chi router.
4. **Unrouted Warehouse & Fleet Endpoints**:
   - `/v1/supplier/warehouses`, `/v1/supplier/warehouses/{id}/trucks`, and `/v1/supplier/warehouses/{id}/payloaders` are not yet mounted on the Chi router.
5. **Memory Repository Fallback**:
   - `internal/supplier/repository.go` still contains 57 silent fallback paths to `MemoryRepository` instead of querying PostgreSQL 16.

These findings have been escalated to `teamwork_preview_orchestrator` to guide workers for Milestones M1, M2, M3, and M4.
