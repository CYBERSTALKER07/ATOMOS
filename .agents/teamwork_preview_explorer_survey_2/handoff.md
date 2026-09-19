# Handoff Report: Survey Specialist 2 (Database Migrations & Schemas)

**Agent**: teamwork_preview_explorer (Survey Specialist 2)  
**Parent**: teamwork_preview_orchestrator (conv ID: `755199e9-0b8c-404a-b2f0-93e7b22240ee`)  
**Working Directory**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_2`  
**Date**: 2026-09-16  

---

## 1. Observation

1. **Migration Runner & File Directory**:
   - Location: `pegasus.x/database/migrations/` contains exactly 69 SQL files.
   - Sequence: Starts at `001_initial_schema.sql` and ends at `068_trade_credit_quota_system.sql`.
   - Naming convention: `<3-digit-number>_<description>.sql`.
   - Note: Prefix `004` appears twice (`004_dispatch_capacities_and_locks.sql` and `004_enterprise_fiscal_dispatch_and_compliance.sql`), making 69 total files with the highest number currently `068`.
   - Runner implementation: `pegasus.x/backend/internal/db/migrate.go` lines 24–108.
     ```go
     // Migrate executes all unapplied SQL migration files in migrationsDir in sequential order
     func (p *Pool) Migrate(ctx context.Context, migrationsDir string) (int, error) {
     ```
     Migration tracking table is `schema_migrations (version VARCHAR(255) PRIMARY KEY, name VARCHAR(255) NOT NULL, applied_at TIMESTAMPTZ, execution_time_ms INT)`.
     Runner reads files ending with `.sql`, sorts them via `sort.Strings(sqlFiles)`, derives `version := strings.TrimSuffix(filename, ".sql")`, and executes each unapplied file inside `p.RunInTx(ctx, func(tx pgx.Tx) error { ... })`.

2. **Table `suppliers`**:
   - Defined in `001_initial_schema.sql` (lines 8–13):
     ```sql
     CREATE TABLE IF NOT EXISTS suppliers (
         supplier_id VARCHAR(64) PRIMARY KEY,
         name VARCHAR(255) NOT NULL,
         legal_tax_id VARCHAR(32) NOT NULL, -- STIR / INN (Uzbekistan Tax ID)
         created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
     );
     ```
   - No `UNIQUE` constraint on `legal_tax_id` or `tax_id`.
   - Missing columns: `tax_id`, `phone`, `password_hash`, `onboarding_status`, `currency`, and `updated_at`.

3. **Table `products` vs. `skus`**:
   - No table named `products` exists in any migration file.
   - Catalog items are stored in `skus` (`001_initial_schema.sql` lines 48–58), enhanced by `059`, `063`, and `064`.
   - The Go backend aliases `skus[i].ProductID = skus[i].SKUID` (`handlers_catalog.go:58`).
   - The prompt requires a `products` table with EAN-13 barcode, 17-digit MXIK code, package code, units_per_case, unit_price_tiyin (BIGINT), vat_rate (12), and status.

4. **Payment Gateway Tables**:
   - `005`, `011`, `016`, `021`, and `062` handle payment transactions, payroll, inbox events, and splits.
   - No table exists for supplier payment gateway configuration credentials (`GLOBAL_PAY`, `CASH`, allowed BINs).
   - In `backend/internal/supplier/models.go:47`, `SelectedGateways` exists only on in-memory Go structs in `MemoryRepository`.

5. **Table `warehouses`**:
   - Created in `001_initial_schema.sql` (lines 15–23):
     ```sql
     CREATE TABLE IF NOT EXISTS warehouses (
         warehouse_id VARCHAR(64) PRIMARY KEY,
         supplier_id VARCHAR(64) NOT NULL REFERENCES suppliers(supplier_id),
         name VARCHAR(255) NOT NULL,
         address TEXT NOT NULL,
         latitude DOUBLE PRECISION NOT NULL,
         longitude DOUBLE PRECISION NOT NULL,
         created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
     );
     ```
   - `latitude` and `longitude` are already `DOUBLE PRECISION NOT NULL`.
   - `status` column does not exist on `warehouses`.

6. **Trucks and Payloaders**:
   - `025_fleet_and_driver_lifecycle_management.sql` created `vehicles`.
   - Neither `warehouse_trucks` nor `trucks` exists.
   - Neither `warehouse_payloaders` nor `payloaders` exists. In `internal/onboarding/service.go:98`, payloaders were held in `s.payloaders[id]`. In `internal/payload/repository.go:910`, `warehouse_staff` was queried without ever being created in migrations.

7. **Warehouse Deletion Guards**:
   - `stock_balances` (`001_initial_schema.sql:60–68`): tracks `on_hand_qty`. Guard check: `SELECT COALESCE(SUM(on_hand_qty), 0) FROM stock_balances WHERE warehouse_id = $1`.
   - `orders` (`001_initial_schema.sql:80–96`): terminal states are `'DELIVERED'` and `'CANCELLED'`. Guard check: `SELECT COUNT(*) FROM orders WHERE warehouse_id = $1 AND status NOT IN ('DELIVERED', 'CANCELLED')`.
   - Over 23 child tables reference `warehouses(warehouse_id)` with RESTRICT / NO ACTION.

---

## 2. Logic Chain

1. **Migration Sequence & Naming**:
   - Since the highest existing migration file in `pegasus.x/database/migrations/` is `068_trade_credit_quota_system.sql` (Observation 1), the next sequential migration is deterministically `069_supplier_onboarding_and_globalpay.sql`.
   - Because `migrate.go` sorts file names alphabetically and extracts the version string by trimming `.sql`, adding this file will cause the server boot runner to execute it inside a single transaction and record version `069_supplier_onboarding_and_globalpay` in `schema_migrations`.

2. **Supplier Authentication & Deduplication Invariant**:
   - Because `suppliers` lacks `UNIQUE(tax_id)` (Observation 2), concurrent sign-ups with the same 9-digit STIR could succeed, violating Uzbekistan legal business entity singularity and R1 requirements.
   - Adding `tax_id VARCHAR(32)`, syncing from `legal_tax_id`, and creating `idx_suppliers_tax_id UNIQUE` and `idx_suppliers_legal_tax_id UNIQUE` guarantees PostgreSQL returns code `23505` (unique_violation), allowing handlers to return HTTP 409 Conflict.
   - Adding `password_hash`, `phone`, and `onboarding_status` with `CHECK (onboarding_status IN ('PENDING', 'PRODUCTS_CONFIGURED', 'PAYMENT_CONFIGURED', 'COMPLETED'))` directly supports the non-bypassable onboarding gate.

3. **Catalog & Dual-Model Parity**:
   - Because existing services (`internal/order`, `internal/inventory`, `internal/claims`, `internal/dispatch`) query `skus` (Observation 3), creating `products` as the standalone onboarding table while synchronizing `units_per_case`, `unit_price_tiyin`, `vat_rate`, and `status` to `skus` prevents breaking existing inventory and checkout flows.

4. **Payment Gateways Table**:
   - Because no database table currently stores supplier gateway configurations (Observation 4), creating `supplier_payment_gateways` (with view `supplier_payment_configs`) allows persistence of `GLOBAL_PAY` and `CASH`, along with the `allowed_bins` array for corporate card validation.

5. **Warehouses & Logistics Fleet Invariants**:
   - Because `warehouses` already uses `DOUBLE PRECISION` for `latitude` and `longitude` (Observation 5), only `status VARCHAR(32) DEFAULT 'ACTIVE'` and `updated_at` must be added.
   - Creating `warehouse_trucks` and `warehouse_payloaders` (with compatibility views `trucks` and `payloaders`) bridges the gap where payloaders and dock trucks were previously stored in memory or non-existent tables (Observation 6).

6. **Warehouse Deletion Safety**:
   - Because `stock_balances` and `orders` have non-cascading foreign keys to `warehouses` (Observation 7), implementing pre-deletion checks for `on_hand_qty > 0` and active orders (`status NOT IN ('DELIVERED', 'CANCELLED')`) fulfills R3 and prevents raw database constraint violations.

---

## 3. Caveats

- **Existing Data Migration**: In environments where `suppliers` already contains rows, `tax_id` must be backfilled from `legal_tax_id` before creating the unique index. The drafted migration handles this with `UPDATE suppliers SET tax_id = legal_tax_id WHERE tax_id IS NULL;`.
- **Duplicate STIR in Dev/Test Environments**: If duplicate records exist in a dirty test database, `CREATE UNIQUE INDEX` will fail. `IF NOT EXISTS` is used, but dirty data would need deduplication prior to index creation.
- **pgx Enum vs. Varchar**: PostgreSQL custom enums can cause type decoding issues with standard pgx scanning if not registered. Using `VARCHAR(32)` with explicit `CHECK` constraint avoids driver type registration overhead while maintaining strict database-level integrity.

---

## 4. Conclusion

The PostgreSQL 16 schema in `pegasus.x` is clean, highly structured across 68 numbered migrations (69 files total), and managed by a built-in runner in `internal/db/migrate.go`. The next required migration is `069_supplier_onboarding_and_globalpay.sql`.

The drafted DDL in `report.md` fully satisfies all R1, R2, R3, and R4 requirements:
1. Adds STIR uniqueness and auth columns to `suppliers`.
2. Creates the dedicated `products` catalog table and keeps `skus` in sync.
3. Creates `supplier_payment_gateways` for `GLOBAL_PAY` and `CASH` configurations.
4. Adds `status` to `warehouses` and confirms `DOUBLE PRECISION` coordinate types.
5. Creates `warehouse_trucks` and `warehouse_payloaders` with compatibility views.
6. Details the exact SQL queries to enforce the HTTP 409 warehouse deletion guard.

---

## 5. Verification Method

1. **Migration Runner Verification**:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
   go test -v ./internal/db/...
   ```
2. **Inspect Existing Migration Files**:
   ```bash
   ls -la /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/database/migrations/ | tail -n 10
   ```
3. **Inspect Comprehensive Report**:
   Examine `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_2/report.md` for the verbatim DDL and schema analysis.
