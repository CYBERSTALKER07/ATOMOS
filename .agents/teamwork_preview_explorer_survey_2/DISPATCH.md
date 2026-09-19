## 2026-09-16T13:16:07Z
You are teamwork_preview_explorer (Survey Specialist 2: Database Migrations & Schemas).
Your working directory is: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_2
Your parent is: teamwork_preview_orchestrator (conv ID: 755199e9-0b8c-404a-b2f0-93e7b22240ee)

MANDATORY FIRST STEP:
Read /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md completely before doing any other work.

TASK:
Perform a deep, read-only survey of the PostgreSQL 16 schema and migrations in `pegasus.x/backend`:
1. Inspect all migrations in `pegasus.x/backend/migrations/` (check the highest migration number, naming conventions, runner mechanism like migrate or golang-migrate or custom runner).
2. Examine existing table definitions for:
   - `suppliers` (current columns: id, name, tax_id/stir, phone, password_hash, onboarding_status, created_at, etc.)
   - `products` (barcode EAN-13, mxik_code, package_code, units_per_case, unit_price_tiyin, vat_percent, supplier_id, status)
   - `supplier_payment_configs` or payment gateway tables (Global Pay, cash settings)
   - `warehouses` (check if latitude/longitude are DOUBLE PRECISION, status, supplier_id)
   - `warehouse_trucks` / `trucks` (license_plate, capacity_kg, capacity_m3, fuel_type)
   - `warehouse_payloaders` / `payloaders` (name, phone, warehouse_id)
   - `inventory` / `stock` / `orders` (how to check on-hand stock > 0 or active orders for warehouse deletion guard)
3. Draft the exact PostgreSQL 16 DDL migration `069_supplier_onboarding_and_globalpay.sql`:
   - Unique constraint or index on `suppliers(tax_id)` (STIR)
   - Enum/VARCHAR for `onboarding_status` ('PENDING', 'PRODUCTS_CONFIGURED', 'PAYMENT_CONFIGURED', 'COMPLETED')
   - Products table columns: EAN-13, 17-digit MXIK, package code, units_per_case, unit_price_tiyin (BIGINT), vat_rate (12)
   - Payment gateways table (supplier_id, provider VARCHAR, enabled BOOLEAN, service_id, secret_key, allowed_bins, etc.)
   - Warehouses table updates (latitude DOUBLE PRECISION, longitude DOUBLE PRECISION)
   - Trucks and Payloaders tables
4. Note any existing foreign key relationships or constraints.

OUTPUT:
Write your comprehensive findings and drafted DDL to `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_2/report.md` and a standard `handoff.md`.
When finished, send a message to parent with the summary and report path.
