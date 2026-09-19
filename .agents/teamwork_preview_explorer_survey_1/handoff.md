# Handoff Report: Survey Specialist 1 (Supplier Domain & Mock Purge)

**Task**: Deep read-only architectural survey of supplier domain, mock purge requirements, and authentication in `pegasus.x/backend`.  
**Agent**: teamwork_preview_explorer_survey_1  
**Recipient**: teamwork_preview_orchestrator / Implementer Agent  
**Full Report**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_1/report.md`

---

## 1. Observation

Direct code and schema observations with exact file:line citations:

1. **In-Memory Mock State & Seeds in `internal/supplier/repository.go`**:
   - `repository.go:90-106`: `type MemoryRepository struct` defines 12 in-memory map/slice fields (`profiles`, `topology`, `orgMembers`, `pricing`, `overrides`, `vetLogs`, `policies`, `breaches`, `aiRecs`, `imports`, `crmRetailers`, `events`, `kycDocs`).
   - `repository.go:108-416`: `NewMemoryRepository()` seeds fake Tashkent data for `"sup_pepsico_uz"` and `"sup_tashkent_beverage"`. Seeded data includes:
     - Profile with fake INN `"302918274"`, MFO `"00444"`, and bank account (lines 129-157).
     - 5 fake KYC documents with dummy URLs like `"https://storage.pegasus.internal/kyc/guvohnoma_pepsico.pdf"` (lines 159-220).
     - Fake topology nodes `node_fac_yangiyol` and `node_wh_sergeli` (lines 223-252).
     - Fake org members `usr_admin_01` (Temur Rustamov) and `usr_dispatcher_01` (lines 254-280).
     - Fake pricing rule (lines 282-297) and Korzinka override `ovr_korzinka` (lines 299-321).
     - Fake service policy (lines 323-333), AI recommendations (lines 335-370), CRM retailers `ret_oqtepa_01` and `ret_yunusobod_02` (lines 372-405), and 4 live events `e1`-`e4` (lines 407-412).
   - `repository.go:418-1022`: 604 lines of purely in-memory CRUD methods on `*MemoryRepository`.

2. **Silent Fallback Anti-Pattern in `PostgresRepository`**:
   - `repository.go:1024-1027`: `type PostgresRepository struct { pool *db.Pool; memory *MemoryRepository }`.
   - `repository.go:1029-1038`: Factory `NewRepository(pool)` returns `mem` if `pool == nil`, and embeds `mem` into `PostgresRepository` if `pool != nil`.
   - `repository.go:1056, 1093, 1095, 1109, 1121, 1126, 1145, 1175, 1177, 1185, 1187, 1200, 1211, 1216, 1233, 1259, 1261, 1269, 1271, 1292, 1325, 1327, 1342, 1354, 1359, 1379, 1410, 1412, 1420, 1422, 1441, 1443, 1461, 1473, 1481, 1501, 1527, 1529, 1548, 1550, 1568, 1579, 1584, 1599, 1612, 1620, 1640, 1671, 1673, 1693, 1698, 1724, 1726, 1744, 1766, 1778, 1786`: 57 occurrences of `if err != nil { return p.memory... }` or `_, _ = p.memory...` dual writes.
   - `repository.go:1791-1818`: 7 methods have **NO SQL QUERIES** and delegate directly to `p.memory`: `ListCRMRetailers`, `GetCRMRetailerDetail`, `AddEvent`, `ListEvents`, `ListKycDocuments`, `SubmitKycDocument`, `ReviewKycDocument`.

3. **Authentication Theatre & Registration Gaps**:
   - `handlers_supplier.go:89-162` (`handleSupplierRegister`):
     - `SupplierRegisterPayload` (lines 74-87) lacks top-level `company_name` and `tax_id` (STIR) fields.
     - Ignores `req.Password`; never calls `bcrypt.GenerateFromPassword`.
     - Never checks STIR uniqueness in PostgreSQL; never returns HTTP 409 Conflict.
     - Saves only via `s.supplierSvc.UpdateProfile` targeting `supplier_profiles`, NEVER inserting into root `suppliers` table (`001_initial_schema.sql`).
     - Returns `next_step: "/setup/business"` and `is_configured: false`, omitting `onboarding_status: "PENDING"`.
   - `handlers_supplier.go:169-208` (`handleSupplierLogin`):
     - Line 181: Hardcodes `sid := "sup_pepsico_uz"`.
     - Ignores `req.Password`; never calls `bcrypt.CompareHashAndPassword`.
     - Never queries PostgreSQL for supplier credentials.
     - Returns hardcoded `is_registered: true` and `next_step: "/dashboard"` or `"/setup/business"`.

4. **Database Schema Realities**:
   - `001_initial_schema.sql:8-13`: `suppliers` table has `supplier_id`, `name`, `legal_tax_id`, `created_at`.
   - Missing on `suppliers`: `phone`, `password_hash`, `onboarding_status`, `updated_at`, and `UNIQUE(legal_tax_id)`.
   - Missing tables: `supplier_payment_configs` (for Cash + Global Pay), `supplier_kyc_documents`, `supplier_audit_events`, and `payloaders`.

5. **Existing Tests**:
   - `internal/supplier/supplier_test.go:10, 58, 106, 154, 217, 256, 300, 349, 417`: 9 unit tests instantiate `NewMemoryRepository()` directly and expect `"sup_pepsico_uz"`.

---

## 2. Logic Chain

1. **Why does registration currently corrupt foreign keys?**
   - Observations 1 & 3: `handleSupplierRegister` calls `s.supplierSvc.UpdateProfile`, which writes to `supplier_profiles` (`058_...sql`). It never creates a corresponding row in the primary `suppliers` table (`001_...sql`).
   - Tables such as `warehouses`, `skus`, and `drivers` have foreign key constraints: `REFERENCES suppliers(supplier_id)`.
   - If a new supplier registers and subsequently adds a warehouse or product, PostgreSQL rejects the insert with a foreign key violation because `suppliers` has no matching row.
   - **Inference**: Registration must write atomically to both `suppliers` (root tenancy) and `supplier_profiles` (portal attributes) within a single `pgx.Tx` transaction (`pool.RunInTx`).

2. **Why can duplicate STIR tax IDs currently be registered?**
   - Observation 4: `001_initial_schema.sql` defines `legal_tax_id VARCHAR(32) NOT NULL` without a `UNIQUE` constraint.
   - Observation 3: `handleSupplierRegister` does not query the database for existing tax IDs.
   - **Inference**: Migration `069_supplier_onboarding_and_globalpay.sql` must add a `UNIQUE (legal_tax_id)` constraint on `suppliers`, and `handleSupplierRegister` must perform an explicit check returning HTTP 409 Conflict (`conflict`) when a duplicate STIR is submitted.

3. **Why is the current repository vulnerable to data loss / phantom data?**
   - Observation 2: In `PostgresRepository`, if any SQL query fails (e.g. timeout, missing table, syntax error), it silently falls back to `p.memory` and returns hardcoded PepsiCo data instead of failing.
   - If a supplier saves data, it writes to Postgres and also dual-writes to the local RAM map. If the server restarts, any data that fell back to RAM is lost.
   - **Inference**: Pure PostgreSQL persistence requires removing `MemoryRepository` entirely from `repository.go`, removing the `memory` field from `PostgresRepository`, eliminating all silent fallbacks, and returning explicit errors.

4. **Why is the onboarding gate missing?**
   - Observation 4 & `internal/auth/middleware.go`: No middleware exists checking `onboarding_status != 'COMPLETED'`.
   - Observation 1: Onboarding status is only tracked in an in-memory `LifecycleManager` without database backing.
   - **Inference**: Middleware `RequireSupplierOnboardingCompleted` must be added to check `onboarding_status` from the database/claims and block operational endpoints with HTTP 428 Precondition Required (`onboarding_incomplete`).

---

## 3. Caveats & Risks

1. **Unit Test Isolation**:
   - `internal/supplier/supplier_test.go` currently relies on `NewMemoryRepository()` across 9 test functions.
   - If `MemoryRepository` is purged from `repository.go`, running `go test ./internal/supplier/...` without a database will fail unless:
     - Tests are rewritten to use a mock interface, or
     - An explicit test-only mock is provided in `supplier_test.go` or `internal/supplier/mock_test.go`.
2. **Mobile Wiring Test Reliance on Mock Seeds**:
   - `internal/api/supplier_mobile_live_wiring_test.go` executes tests querying `GET /v1/products?supplier_id=sup_pepsico_uz`.
   - Test suites running in CI without Docker must have database seeds inserted during test setup if running against PostgreSQL.
3. **Password Security**:
   - Ensure `bcrypt.MinCost` is used in unit tests for speed, but `bcrypt.DefaultCost` (cost 10) is used in production code.

---

## 4. Conclusion

The supplier domain in `pegasus.x/backend` currently exhibits significant architectural theatre: authentication does not verify passwords or deduplicate STIRs, repository queries silently fall back to hardcoded in-memory mocks, and registered suppliers are not inserted into the root `suppliers` table.

To achieve pure PostgreSQL 16 persistence and fulfill the user request, the following changes are strictly required:
1. **Migration 069**: Create `069_supplier_onboarding_and_globalpay.sql` to alter `suppliers` (`phone`, `password_hash`, `onboarding_status`, `UNIQUE(legal_tax_id)`) and create `supplier_payment_configs`, `supplier_kyc_documents`, `supplier_audit_events`, and `payloaders`.
2. **Mock Purge**: Delete `MemoryRepository` and silent fallbacks from `internal/supplier/repository.go`. Implement real SQL for `ListCRMRetailers`, `GetCRMRetailerDetail`, `ListKycDocuments`, `SubmitKycDocument`, `ReviewKycDocument`, `AddEvent`, and `ListEvents`.
3. **Auth Overhaul**: Update `handleSupplierRegister` to enforce 9-digit STIR uniqueness (returning HTTP 409 on conflict), hash passwords with bcrypt, atomically insert into `suppliers` and `supplier_profiles`, and return `onboarding_status: "PENDING", next_step: "/onboarding/products"`. Update `handleSupplierLogin` to verify passwords with bcrypt against PostgreSQL and return `onboarding_status`.
4. **Onboarding Gate & Wizard**: Build `RequireSupplierOnboardingCompleted` middleware (HTTP 428 Precondition Required) and wire `/v1/supplier/onboarding/{products,payment,complete}`.
5. **Fleet & Warehouse Hub**: Wire warehouse coordinate validation, conflict checks on active stock/orders for deletion, and warehouse trucks/payloaders endpoints.

---

## 5. Verification Method

Independent verification of the audit and proposed implementation:

1. **Verify Existing Mock Code & Citations**:
   ```bash
   grep -n "MemoryRepository" pegasus.x/backend/internal/supplier/repository.go
   grep -n "sup_pepsico_uz" pegasus.x/backend/internal/supplier/repository.go
   grep -n "p.memory" pegasus.x/backend/internal/supplier/repository.go
   ```
2. **Verify Password & STIR Absence in Auth Handlers**:
   ```bash
   view_file pegasus.x/backend/internal/api/handlers_supplier.go (lines 89-208)
   ```
3. **Verify Database Migrations**:
   ```bash
   view_file pegasus.x/database/migrations/001_initial_schema.sql (lines 8-13)
   view_file pegasus.x/database/migrations/058_supplier_portal_core_and_operations.sql (lines 8-25)
   ```
4. **Post-Implementation Verification (to be run by Worker)**:
   ```bash
   cd pegasus.x/backend
   go test -v -race ./internal/supplier/...
   go test -v -race ./internal/api/... -run "TestSupplier"
   go test -v -race ./...
   ```
5. **Invalidation Conditions**:
   - If any `MemoryRepository` remains in `internal/supplier/repository.go`.
   - If `handleSupplierRegister` accepts duplicate STIRs without HTTP 409.
   - If `handleSupplierLogin` accepts invalid passwords or hardcodes `"sup_pepsico_uz"`.
   - If calling operational supplier endpoints with `onboarding_status != 'COMPLETED'` succeeds instead of returning HTTP 428.
