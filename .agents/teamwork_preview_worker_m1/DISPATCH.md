## 2026-09-16T13:23:40Z

You are teamwork_preview_worker (Milestone 1 Worker: PostgreSQL 16 Migration 069 & Pure pgxpool Repository).
Your working directory is: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m1
Your parent is: teamwork_preview_orchestrator (conv ID: 755199e9-0b8c-404a-b2f0-93e7b22240ee)

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A reviewer will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

MANDATORY FIRST STEP:
Read /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md, /Users/shakhzod/Desktop/V.O.I.D/PROJECT.md, and the survey reports:
- /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_1/report.md
- /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_2/report.md

TASK:
Implement Milestone 1:
1. **Create PostgreSQL 16 Migration `069_supplier_onboarding_and_globalpay.sql`**:
   - File: `pegasus.x/database/migrations/069_supplier_onboarding_and_globalpay.sql`
   - DDL specifications from Survey 2 report:
     - Alter `suppliers`: add `tax_id VARCHAR(32)`, `phone VARCHAR(32)`, `password_hash VARCHAR(255)`, `onboarding_status VARCHAR(32) NOT NULL DEFAULT 'PENDING'`, `CHECK (onboarding_status IN ('PENDING', 'PRODUCTS_CONFIGURED', 'PAYMENT_CONFIGURED', 'COMPLETED'))`.
     - Backfill: `UPDATE suppliers SET tax_id = legal_tax_id WHERE tax_id IS NULL;`
     - Create unique indexes: `idx_suppliers_tax_id UNIQUE` on `suppliers(tax_id)` and `idx_suppliers_legal_tax_id UNIQUE` on `suppliers(legal_tax_id)`.
     - Create `products` table: `product_id VARCHAR(64) PRIMARY KEY`, `supplier_id VARCHAR(64) NOT NULL REFERENCES suppliers(supplier_id) ON DELETE CASCADE`, `name VARCHAR(255) NOT NULL`, `barcode VARCHAR(32) UNIQUE NOT NULL`, `mxik_code VARCHAR(32) NOT NULL`, `package_code VARCHAR(32) NOT NULL`, `units_per_case INT NOT NULL DEFAULT 1`, `unit_price_tiyin BIGINT NOT NULL`, `vat_rate NUMERIC(5,2) NOT NULL DEFAULT 12.00`, `status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE'`, `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`, `updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`.
     - Create `supplier_payment_gateways` table: `config_id VARCHAR(64) PRIMARY KEY`, `supplier_id VARCHAR(64) NOT NULL REFERENCES suppliers(supplier_id) ON DELETE CASCADE`, `provider VARCHAR(32) NOT NULL`, `enabled BOOLEAN NOT NULL DEFAULT TRUE`, `service_id VARCHAR(128)`, `secret_key VARCHAR(256)`, `allowed_bins TEXT[] DEFAULT '{}'`, `is_default BOOLEAN NOT NULL DEFAULT FALSE`, `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`, `updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`, `CONSTRAINT uq_supplier_provider UNIQUE (supplier_id, provider)`.
     - Create view `supplier_payment_configs AS SELECT * FROM supplier_payment_gateways;`.
     - Alter `warehouses`: add `status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE'`, `updated_at TIMESTAMPTZ DEFAULT NOW()`. Verify `latitude` and `longitude` are `DOUBLE PRECISION NOT NULL`.
     - Create `warehouse_trucks` and `warehouse_payloaders` tables with compatibility views `trucks` and `payloaders`.
     - Create `supplier_kyc_documents` and `supplier_audit_events` tables to support KYC and audit event persistence.

2. **Purge `MemoryRepository` and Mock Seeds from `pegasus.x/backend/internal/supplier/repository.go`**:
   - Delete `type MemoryRepository struct` and its 12 in-memory maps.
   - Delete `NewMemoryRepository()` and all hardcoded mock seeds.
   - Remove `memory *MemoryRepository` from `PostgresRepository`.
   - Remove all 57 occurrences of `if err != nil { return p.memory... }` silent fallbacks. Return real errors.
   - Implement real SQL queries using `p.pool` (`*db.Pool`) for all methods in `PostgresRepository` (including `ListCRMRetailers`, `GetCRMRetailerDetail`, `ListKycDocuments`, `SubmitKycDocument`, `ReviewKycDocument`, `AddEvent`, `ListEvents`).
   - Add any new repository methods needed for supplier registration (e.g. `CreateSupplier`, `GetSupplierByTaxID`, `GetSupplierByID`, `UpdateOnboardingStatus`, `SavePaymentGateway`, `GetPaymentGateways`, `CreateProduct`, `ListProducts`, `DeleteProduct`).

3. **Preserve Unit Test Stability**:
   - In `pegasus.x/backend/internal/supplier/supplier_test.go`, the existing unit tests expect an in-memory repository for offline testing.
   - Create a clean test mock `testMockRepository` in `pegasus.x/backend/internal/supplier/mock_test.go` (strictly `_test.go`) so unit tests run cleanly without any mock code in production `repository.go`.
   - Run `go test -v -race ./internal/supplier/...` in `pegasus.x/backend` and ensure all tests pass!

WRITE OWNERSHIP:
- `pegasus.x/database/migrations/069_supplier_onboarding_and_globalpay.sql`
- `pegasus.x/backend/internal/supplier/repository.go`
- `pegasus.x/backend/internal/supplier/models.go`
- `pegasus.x/backend/internal/supplier/mock_test.go`
- `pegasus.x/backend/internal/supplier/supplier_test.go`

VERIFICATION:
Execute `go test -v -race ./internal/supplier/...` and `go test -v ./internal/db/...` to verify migrations and repository. Document all outputs in your handoff report.
When complete, write `handoff.md` and send a message back.
