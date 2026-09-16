# Deep Survey Report: PostgreSQL 16 Schema & Migrations in `pegasus.x`

**Surveyor**: teamwork_preview_explorer (Survey Specialist 2: Database Migrations & Schemas)  
**Date**: 2026-09-16  
**Target Repository**: `pegasus.x` (Sovereign Lean Single-Tenant PostgreSQL 16 + Redis 7)  
**Deliverable**: Comprehensive schema audit, existing entity analysis, and production-ready DDL draft for `069_supplier_onboarding_and_globalpay.sql`.

---

## 1. Migrations Directory & Execution Runner Audit

### 1.1 Directory Structure & File Inventory
- **Filesystem Path**: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/database/migrations/`
- **Total Migration Files**: 69 files
- **Numbering Sequence**: `001_initial_schema.sql` through `068_trade_credit_quota_system.sql`
- **Version Collision / Duplicate Prefix**: Note that prefix `004` has two files:
  1. `004_dispatch_capacities_and_locks.sql`
  2. `004_enterprise_fiscal_dispatch_and_compliance.sql`
- **Current Highest Migration Number**: `068` (`068_trade_credit_quota_system.sql`)
- **Next Required Migration**: `069_supplier_onboarding_and_globalpay.sql`

### 1.2 Runner Mechanism & Protocol
- **Implementation File**: `pegasus.x/backend/internal/db/migrate.go` (lines 24–108)
- **Invocation Point**: `pegasus.x/backend/cmd/server/main.go` (lines 49–57)
- **Tracking Table**: `schema_migrations`
  ```sql
  CREATE TABLE IF NOT EXISTS schema_migrations (
      version VARCHAR(255) PRIMARY KEY,
      name VARCHAR(255) NOT NULL,
      applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      execution_time_ms INT NOT NULL
  );
  ```
- **Sorting & Execution Logic**:
  - `os.ReadDir(migrationsDir)` scans all entries ending in `.sql`.
  - Files are sorted alphabetically via `sort.Strings(sqlFiles)`.
  - `version := strings.TrimSuffix(filename, ".sql")` is compared against `applied[version]`.
  - Each unapplied file is executed atomically within a single `p.RunInTx(ctx, func(tx pgx.Tx) error { ... })` transaction.
  - On success, `INSERT INTO schema_migrations (version, name, applied_at, execution_time_ms)` is recorded.
  - Therefore, migration `069_supplier_onboarding_and_globalpay.sql` will record version `"069_supplier_onboarding_and_globalpay"` in `schema_migrations`.

---

## 2. Deep Table-by-Table Schema Inspection

### 2.1 `suppliers`
- **Original Creation**: `database/migrations/001_initial_schema.sql` (lines 8–13)
  ```sql
  CREATE TABLE IF NOT EXISTS suppliers (
      supplier_id VARCHAR(64) PRIMARY KEY,
      name VARCHAR(255) NOT NULL,
      legal_tax_id VARCHAR(32) NOT NULL, -- STIR / INN (Uzbekistan Tax ID)
      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
  );
  ```
- **Current Columns in PostgreSQL**:
  - `supplier_id VARCHAR(64) PRIMARY KEY`
  - `name VARCHAR(255) NOT NULL`
  - `legal_tax_id VARCHAR(32) NOT NULL`
  - `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
- **Observed Critical Gaps**:
  1. **No STIR Uniqueness Constraint**: Neither `legal_tax_id` nor `tax_id` has a `UNIQUE` index or constraint in PostgreSQL. Duplicate registrations with the same STIR currently fail to produce an HTTP 409 Conflict at the database layer.
  2. **Missing Columns**: `tax_id`, `phone`, `password_hash`, `onboarding_status`, and `currency` are completely missing from the `suppliers` table in PostgreSQL.
  3. **Note on `supplier_profiles`**: Migration `058_supplier_portal_core_and_operations.sql` created an auxiliary `supplier_profiles` table, but the core authentication and multi-role models bind directly to `suppliers(supplier_id)`.
- **Target Schema Requirements**:
  - Add `tax_id VARCHAR(32)` populated from `legal_tax_id`.
  - Add `phone VARCHAR(32)`.
  - Add `password_hash VARCHAR(255)`.
  - Add `onboarding_status VARCHAR(32) NOT NULL DEFAULT 'PENDING' CHECK (onboarding_status IN ('PENDING', 'PRODUCTS_CONFIGURED', 'PAYMENT_CONFIGURED', 'COMPLETED'))`.
  - Add `currency VARCHAR(8) NOT NULL DEFAULT 'UZS'`.
  - Add `updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`.
  - Enforce `UNIQUE` constraint/index on `suppliers(tax_id)` and `suppliers(legal_tax_id)`.

---

### 2.2 `products` vs. `skus`
- **Current Situation in `pegasus.x`**:
  - In `pegasus.x`, catalog items were historically stored under the table name `skus` (`001_initial_schema.sql` lines 48–58).
  - Enhanced in `059_supplier_product_catalog_base.sql` (`description`, `image_url`, `units_per_pack`, `stackable`, `handling_class`, `requires_cold_chain`, `is_hazardous`, `is_perishable`, `is_active`).
  - Enhanced in `063_product_setup_pack_pricing_and_promotions.sql` (`category_id`, `pack_type`, `packaging_material`, `size_volume_ml`, `size_weight_g`, `size_label`, `floor_price_minor`, `min_order_quantity`, `unit_volume_vu`, `version`).
  - Enhanced in `064_flexible_packaging_hierarchy_and_dynamic_attributes.sql` (`base_uom`, `dynamic_attributes`).
- **Absence of Dedicated `products` Table**:
  - A table named `products` does **not** currently exist in PostgreSQL 16.
  - The Go backend handles both `sku_id` and `product_id` by aliasing (`skus[i].ProductID = skus[i].SKUID` in `handlers_catalog.go` and `handlers_order.go`).
- **Target Schema Requirements for Migration 069**:
  - Create dedicated table `products` with explicit Soliq statutory fields:
    - `id VARCHAR(64) PRIMARY KEY`
    - `supplier_id VARCHAR(64) NOT NULL REFERENCES suppliers(supplier_id) ON DELETE CASCADE`
    - `name VARCHAR(255) NOT NULL`
    - `barcode VARCHAR(64) NOT NULL UNIQUE` (EAN-13 format)
    - `mxik_code VARCHAR(17) NOT NULL` (17-digit Soliq classification)
    - `package_code VARCHAR(32) NOT NULL DEFAULT '796'`
    - `units_per_case INT NOT NULL DEFAULT 1`
    - `unit_price_tiyin BIGINT NOT NULL CHECK (unit_price_tiyin > 0)` (64-bit integer minor units)
    - `vat_rate INT NOT NULL DEFAULT 12` (Uzbekistan Soliq QQS 12%)
    - `status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE'`
    - `created_at TIMESTAMPTZ`, `updated_at TIMESTAMPTZ`
  - In addition, add `units_per_case`, `unit_price_tiyin`, `vat_rate`, and `status` to `skus` to ensure 100% backward compatibility with all existing warehouse, fulfillment, and invoicing queries.

---

### 2.3 `supplier_payment_configs` / Payment Gateway Configuration
- **Current Situation in `pegasus.x`**:
  - Migration `005` created transaction payment tables: `order_payment_legs`, `retailer_debts`, `retailer_credit_accounts`, `retailer_wallets`, `ledger_journal_entries`, `ledger_postings`.
  - Migration `016` created `globalpay_inbox_events`.
  - Migration `021` created `globalpay_split_settlements`.
  - Migration `062` added `payment_method` to `orders`.
  - **Zero PostgreSQL Tables for Supplier Payment Gateways**: Currently, `supplier.SupplierProfile` only stores `SelectedGateways []string` in-memory in `MemoryRepository` (`supplier/models.go:47`). No database table exists to store merchant credentials, Global Pay service IDs, secret keys, or corporate card BIN whitelist.
- **Target Schema Requirements for Migration 069**:
  - Create table `supplier_payment_gateways` (with alias view `supplier_payment_configs`):
    - `id VARCHAR(64) PRIMARY KEY`
    - `supplier_id VARCHAR(64) NOT NULL REFERENCES suppliers(supplier_id) ON DELETE CASCADE`
    - `provider VARCHAR(32) NOT NULL` ('CASH', 'GLOBAL_PAY', 'CLICK', 'PAYME', 'UZUM')
    - `enabled BOOLEAN NOT NULL DEFAULT TRUE`
    - `service_id VARCHAR(128)` (Global Pay Merchant Service ID)
    - `secret_key TEXT` (Encrypted / hashed API key)
    - `allowed_bins TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[]` (Corporate card BINs: `['8600', '5614', '9860']`)
    - `merchant_id VARCHAR(128)`
    - `terminal_id VARCHAR(128)`
    - `created_at TIMESTAMPTZ`, `updated_at TIMESTAMPTZ`
    - Constraint: `UNIQUE (supplier_id, provider)`

---

### 2.4 `warehouses`
- **Original Creation**: `database/migrations/001_initial_schema.sql` (lines 15–23)
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
- **Enhancements**:
  - `061_warehouse_enterprise_topology_and_dock_doors.sql`: added `place_id`, `coverage_radius_km DOUBLE PRECISION DEFAULT 50.0`, `has_cold_storage`, `total_bays`, `dock_doors_count`, `auto_dispatch_enabled`, `default_out_of_stock_policy`.
  - `065_warehouse_stock_management_and_backorders.sql`: added `default_backorder_policy`.
- **Observed Findings**:
  - `latitude` and `longitude` are already `DOUBLE PRECISION NOT NULL`.
  - `status` column does **not** exist in `warehouses`!
- **Target Schema Requirements for Migration 069**:
  - `ALTER TABLE warehouses ADD COLUMN IF NOT EXISTS status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE';`
  - `ALTER TABLE warehouses ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();`
  - Explicit confirmation of `DOUBLE PRECISION` coordinate types.

---

### 2.5 `warehouse_trucks` / `trucks`
- **Current Situation in `pegasus.x`**:
  - Migration `025_fleet_and_driver_lifecycle_management.sql` created `vehicles` (`vehicle_id`, `supplier_id`, `warehouse_id`, `license_plate`, `make_model`, `payload_capacity_kg`, `max_volume_vu`, `fuel_type`, `operational_status`, ...).
  - No table named `warehouse_trucks` or `trucks` exists in PostgreSQL.
- **Target Schema Requirements for Migration 069**:
  - Create dedicated table `warehouse_trucks`:
    - `id VARCHAR(64) PRIMARY KEY`
    - `warehouse_id VARCHAR(64) NOT NULL REFERENCES warehouses(warehouse_id) ON DELETE CASCADE`
    - `supplier_id VARCHAR(64) NOT NULL REFERENCES suppliers(supplier_id) ON DELETE CASCADE`
    - `license_plate VARCHAR(32) NOT NULL`
    - `capacity_kg INT NOT NULL CHECK (capacity_kg > 0)`
    - `capacity_m3 DOUBLE PRECISION NOT NULL CHECK (capacity_m3 > 0)`
    - `fuel_type VARCHAR(32) NOT NULL DEFAULT 'METHANE_CNG'`
    - `is_active BOOLEAN NOT NULL DEFAULT TRUE`
    - `created_at TIMESTAMPTZ`, `updated_at TIMESTAMPTZ`
    - Constraint: `UNIQUE (warehouse_id, license_plate)`
  - Create compatibility view `trucks AS SELECT * FROM warehouse_trucks`.

---

### 2.6 `warehouse_payloaders` / `payloaders`
- **Current Situation in `pegasus.x`**:
  - No PostgreSQL table named `warehouse_payloaders` or `payloaders` exists.
  - In `internal/onboarding/service.go`, payloaders were kept strictly in-memory (`s.payloaders[id] = profile`).
  - In `internal/payload/repository.go` (line 910), a query was written against `warehouse_staff`, which was also never created in migrations.
- **Target Schema Requirements for Migration 069**:
  - Create dedicated table `warehouse_payloaders`:
    - `id VARCHAR(64) PRIMARY KEY`
    - `warehouse_id VARCHAR(64) NOT NULL REFERENCES warehouses(warehouse_id) ON DELETE CASCADE`
    - `supplier_id VARCHAR(64) NOT NULL REFERENCES suppliers(supplier_id) ON DELETE CASCADE`
    - `name VARCHAR(255) NOT NULL`
    - `phone VARCHAR(32) NOT NULL`
    - `status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE'`
    - `created_at TIMESTAMPTZ`, `updated_at TIMESTAMPTZ`
    - Constraint: `UNIQUE (warehouse_id, phone)`
  - Create compatibility view `payloaders AS SELECT * FROM warehouse_payloaders`.

---

### 2.7 Inventory & Order Guards for Warehouse Deletion
- **Safety Requirement**: Block warehouse deletion with HTTP 409 Conflict if:
  1. On-hand stock > 0
  2. Active orders exist
- **Database Verification Queries**:
  - **Stock Guard**:
    ```sql
    SELECT COALESCE(SUM(on_hand_qty), 0) FROM stock_balances WHERE warehouse_id = $1;
    ```
    If sum > 0, return HTTP 409 with reason: `warehouse_has_on_hand_stock`.
  - **Active Orders Guard**:
    Terminal order statuses in `pegasus.x` are `'DELIVERED'` and `'CANCELLED'`.
    ```sql
    SELECT COUNT(*) FROM orders WHERE warehouse_id = $1 AND status NOT IN ('DELIVERED', 'CANCELLED');
    ```
    If count > 0, return HTTP 409 with reason: `warehouse_has_active_orders`.
- **Relational Integrity Foreign Keys**:
  Over 23 tables reference `warehouses(warehouse_id)` with `RESTRICT` / `NO ACTION` foreign key constraints:
  - `orders`, `stock_balances`, `manifests`, `vehicles`, `drivers`, `driver_vehicle_assignments`, `stock_commitments`, `forecast_demand_runs`, `delivery_exceptions`, `warehouse_geographic_coverage`, `tara_warehouse_inventory`, `dock_bays`, `warehouse_locations`, `warehouse_lots`, `wms_pick_waves`, `warehouse_stock_audit_log`, `consignment_contracts`, `warehouse_labor_shifts`, `damage_claims`, `ump_events`, `qm_quarantine_lots`.
  Attempting to delete a warehouse with attached records in PostgreSQL without cascading will immediately fail with a foreign key violation (`23503`), confirming that application-level guards must preemptively validate and return structured 409 Conflict messages.

---

## 3. Full Production DDL Draft: `069_supplier_onboarding_and_globalpay.sql`

```sql
-- ==============================================================================
-- PEGASUS.X MIGRATION 069: SUPPLIER ONBOARDING, LEGAL DEDUPLICATION & GLOBALPAY GATEWAY
-- Storage Engine: PostgreSQL 16
-- Domain Scope:
--   1. Suppliers legal STIR deduplication, phone, password_hash, onboarding_status
--   2. Dedicated Products catalog table with EAN-13, MXIK-17, Tiyins price, 12% VAT
--   3. Synchronize SKUs table with product catalog columns for ecosystem parity
--   4. Supplier Payment Gateways (CASH, GLOBAL_PAY B2B corporate card BIN validation)
--   5. Warehouses status column & GPS double precision verification
--   6. Warehouse Trucks & Warehouse Payloaders tables with relational integrity
-- ==============================================================================

-- ------------------------------------------------------------------------------
-- 1. SUPPLIERS ENHANCEMENTS FOR AUTH & LEGAL ONBOARDING
-- ------------------------------------------------------------------------------
ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS tax_id VARCHAR(32);
UPDATE suppliers SET tax_id = legal_tax_id WHERE tax_id IS NULL;

ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS phone VARCHAR(32);
ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255);
ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS onboarding_status VARCHAR(32) NOT NULL DEFAULT 'PENDING';
ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS currency VARCHAR(8) NOT NULL DEFAULT 'UZS';
ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Enforce STIR / Tax ID uniqueness
CREATE UNIQUE INDEX IF NOT EXISTS idx_suppliers_tax_id ON suppliers (tax_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_suppliers_legal_tax_id ON suppliers (legal_tax_id);
CREATE INDEX IF NOT EXISTS idx_suppliers_phone ON suppliers (phone);
CREATE INDEX IF NOT EXISTS idx_suppliers_onboarding_status ON suppliers (onboarding_status);

-- Add check constraint on onboarding_status if not already present
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_suppliers_onboarding_status'
    ) THEN
        ALTER TABLE suppliers ADD CONSTRAINT chk_suppliers_onboarding_status 
            CHECK (onboarding_status IN ('PENDING', 'PRODUCTS_CONFIGURED', 'PAYMENT_CONFIGURED', 'COMPLETED'));
    END IF;
END $$;

-- ------------------------------------------------------------------------------
-- 2. DEDICATED PRODUCTS CATALOG TABLE
-- ------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS products (
    id VARCHAR(64) PRIMARY KEY,
    supplier_id VARCHAR(64) NOT NULL REFERENCES suppliers(supplier_id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    barcode VARCHAR(64) NOT NULL,                  -- EAN-13 barcode
    mxik_code VARCHAR(17) NOT NULL,                 -- 17-digit statutory Soliq MXIK classification
    package_code VARCHAR(32) NOT NULL DEFAULT '796',-- Soliq packaging code (e.g. 796 piece)
    units_per_case INT NOT NULL DEFAULT 1,          -- Number of units per wholesale shipping case
    unit_price_tiyin BIGINT NOT NULL,              -- Minor units (1 UZS = 100 tiyins)
    vat_rate INT NOT NULL DEFAULT 12,              -- Soliq QQS statutory 12% VAT
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',  -- 'ACTIVE', 'INACTIVE', 'DRAFT'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_products_unit_price_positive CHECK (unit_price_tiyin > 0),
    CONSTRAINT chk_products_units_case_positive CHECK (units_per_case > 0),
    CONSTRAINT chk_products_vat_rate CHECK (vat_rate >= 0),
    CONSTRAINT uq_products_barcode UNIQUE (barcode)
);

CREATE INDEX IF NOT EXISTS idx_products_supplier ON products (supplier_id, status);
CREATE INDEX IF NOT EXISTS idx_products_barcode ON products (barcode);
CREATE INDEX IF NOT EXISTS idx_products_mxik ON products (mxik_code);

-- ------------------------------------------------------------------------------
-- 3. ENHANCE SKUS TABLE TO ENSURE COMPLETE PARITY WITH PRODUCTS
-- ------------------------------------------------------------------------------
ALTER TABLE skus ADD COLUMN IF NOT EXISTS units_per_case INT NOT NULL DEFAULT 1;
ALTER TABLE skus ADD COLUMN IF NOT EXISTS unit_price_tiyin BIGINT;
UPDATE skus SET unit_price_tiyin = unit_price_minor WHERE unit_price_tiyin IS NULL;
ALTER TABLE skus ADD COLUMN IF NOT EXISTS vat_rate INT NOT NULL DEFAULT 12;
ALTER TABLE skus ADD COLUMN IF NOT EXISTS status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE';

-- ------------------------------------------------------------------------------
-- 4. SUPPLIER PAYMENT CONFIGURATIONS / GATEWAYS
-- ------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS supplier_payment_gateways (
    id VARCHAR(64) PRIMARY KEY,
    supplier_id VARCHAR(64) NOT NULL REFERENCES suppliers(supplier_id) ON DELETE CASCADE,
    provider VARCHAR(32) NOT NULL,                 -- 'CASH', 'GLOBAL_PAY', 'CLICK', 'PAYME', 'UZUM'
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    service_id VARCHAR(128),                       -- Global Pay Merchant Service ID
    secret_key TEXT,                               -- Encrypted API secret key
    allowed_bins TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[], -- Corporate B2B card BIN whitelist (e.g. 8600, 5614)
    merchant_id VARCHAR(128),                      -- Terminal or merchant identifier
    terminal_id VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_supplier_payment_gateways UNIQUE (supplier_id, provider),
    CONSTRAINT chk_payment_gateways_provider CHECK (provider IN ('CASH', 'GLOBAL_PAY', 'CLICK', 'PAYME', 'UZUM'))
);

CREATE INDEX IF NOT EXISTS idx_supplier_payment_gateways_sup ON supplier_payment_gateways (supplier_id, enabled);

-- Compatibility alias view: supplier_payment_configs -> supplier_payment_gateways
CREATE OR REPLACE VIEW supplier_payment_configs AS 
SELECT 
    id,
    supplier_id,
    provider,
    enabled,
    service_id,
    secret_key,
    allowed_bins,
    merchant_id,
    terminal_id,
    created_at,
    updated_at
FROM supplier_payment_gateways;

-- ------------------------------------------------------------------------------
-- 5. WAREHOUSES TABLE ENHANCEMENTS & COORDINATES VERIFICATION
-- ------------------------------------------------------------------------------
ALTER TABLE warehouses ADD COLUMN IF NOT EXISTS status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE';
ALTER TABLE warehouses ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE warehouses ALTER COLUMN latitude TYPE DOUBLE PRECISION;
ALTER TABLE warehouses ALTER COLUMN longitude TYPE DOUBLE PRECISION;

CREATE INDEX IF NOT EXISTS idx_warehouses_supplier_status ON warehouses (supplier_id, status);

-- ------------------------------------------------------------------------------
-- 6. WAREHOUSE TRUCKS (FLEET HARDWARE AT LOADING DOCKS)
-- ------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS warehouse_trucks (
    id VARCHAR(64) PRIMARY KEY,
    warehouse_id VARCHAR(64) NOT NULL REFERENCES warehouses(warehouse_id) ON DELETE CASCADE,
    supplier_id VARCHAR(64) NOT NULL REFERENCES suppliers(supplier_id) ON DELETE CASCADE,
    license_plate VARCHAR(32) NOT NULL,            -- Vehicle registration number (e.g. '01 A 777 AA')
    capacity_kg INT NOT NULL,                      -- Maximum load capacity in kilograms
    capacity_m3 DOUBLE PRECISION NOT NULL,         -- Volumetric cargo volume in cubic meters
    fuel_type VARCHAR(32) NOT NULL DEFAULT 'METHANE_CNG', -- 'METHANE_CNG', 'PROPANE_LPG', 'DIESEL', 'PETROL', 'ELECTRIC'
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_trucks_capacity_kg_positive CHECK (capacity_kg > 0),
    CONSTRAINT chk_trucks_capacity_m3_positive CHECK (capacity_m3 > 0),
    CONSTRAINT uq_warehouse_trucks_plate UNIQUE (warehouse_id, license_plate)
);

CREATE INDEX IF NOT EXISTS idx_warehouse_trucks_wh ON warehouse_trucks (warehouse_id, is_active);
CREATE INDEX IF NOT EXISTS idx_warehouse_trucks_sup ON warehouse_trucks (supplier_id);

-- Compatibility alias view: trucks -> warehouse_trucks
CREATE OR REPLACE VIEW trucks AS SELECT * FROM warehouse_trucks;

-- ------------------------------------------------------------------------------
-- 7. WAREHOUSE PAYLOADERS (STEVEDORES / BAY OPERATORS)
-- ------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS warehouse_payloaders (
    id VARCHAR(64) PRIMARY KEY,
    warehouse_id VARCHAR(64) NOT NULL REFERENCES warehouses(warehouse_id) ON DELETE CASCADE,
    supplier_id VARCHAR(64) NOT NULL REFERENCES suppliers(supplier_id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,                    -- Operator full name
    phone VARCHAR(32) NOT NULL,                    -- Contact telephone number (+998...)
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',  -- 'ACTIVE', 'ON_LEAVE', 'INACTIVE'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_warehouse_payloaders_phone UNIQUE (warehouse_id, phone)
);

CREATE INDEX IF NOT EXISTS idx_warehouse_payloaders_wh ON warehouse_payloaders (warehouse_id, status);
CREATE INDEX IF NOT EXISTS idx_warehouse_payloaders_sup ON warehouse_payloaders (supplier_id);

-- Compatibility alias view: payloaders -> warehouse_payloaders
CREATE OR REPLACE VIEW payloaders AS SELECT * FROM warehouse_payloaders;
```

---

## 4. Key Takeaways & Recommendations for Implementer Agent

1. **Migration Placement**: The implementer must place the file at `pegasus.x/database/migrations/069_supplier_onboarding_and_globalpay.sql`. No custom runner modifications are required because `migrate.go` will automatically discover and run it sequentially in alphabetical order.
2. **Repository Refactoring**: `internal/supplier/repository.go` currently defaults to `MemoryRepository`. For R4, the implementer must delete/deprecate `MemoryRepository` and wire `PostgresRepository` directly to `pgxpool.Pool`, reading/writing to `suppliers`, `products`, `supplier_payment_gateways`, `warehouses`, `warehouse_trucks`, and `warehouse_payloaders`.
3. **STIR Conflict Error Mapping**: On registration (`POST /v1/auth/supplier/register`), if `tax_id` violates `idx_suppliers_tax_id` or `idx_suppliers_legal_tax_id`, PostgreSQL returns error code `23505` (unique_violation). The handler must map this to HTTP 409 Conflict.
4. **Warehouse Deletion Invariant**: The handler for `DELETE /v1/supplier/warehouses/{id}` must execute the two read queries against `stock_balances` and `orders` before issuing `DELETE FROM warehouses`, returning HTTP 409 Conflict if stock > 0 or non-terminal orders exist.
