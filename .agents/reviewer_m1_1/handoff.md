# Independent Review & Adversarial Audit Report — Milestone 1

**Reviewer**: Reviewer M1_1 (Adversarial Critic & Codebase Reviewer)  
**Working Directory**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/reviewer_m1_1`  
**Target Monorepo**: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`  
**Timestamp**: 2026-09-22T21:18:30Z  
**Verdict**: **REQUEST_CHANGES**  

---

## 1. Observation

### 1.1 Verified Passing Items
1. **Drop Cash Ceiling on `order_payment_legs`**:
   - `074_ecosystem_hardening_and_parity.sql:10`: Verbatim:
     ```sql
     ALTER TABLE order_payment_legs DROP CONSTRAINT IF EXISTS chk_b2b_cash_limit;
     ```
   - Confirmed in `004_enterprise_fiscal_dispatch_and_compliance.sql:71` that `chk_b2b_cash_limit` was originally attached to `order_payment_legs` (`method != 'CASH' OR amount_minor <= 2500000000`), fixing the defect in migration 073 where it attempted to alter non-existent table `payments`.
2. **`manifests` Columns Extended**:
   - `074_ecosystem_hardening_and_parity.sql:14-35`: All 22 columns required by `payload/repository.go:807-840` (`manifest_number`, `truck_plate`, `vehicle_class`, `driver_name`, `max_volume_vu`, `stop_count`, `front_axle_kg`, `rear_axle_kg`, `steer_tractive_ratio`, `bolt_seal_serial`, `digital_seal_hash`, `sealing_inspector_id`, `inspection_photo_url`, `loading_started_at`, `sealed_at`, `dispatched_at`, `is_axle_overridden`, `axle_override_reason`, `axle_override_by`, `supervisor_pinfl`, `supervisor_reason_code`, `updated_at`) and indexes on `bolt_seal_serial` and `manifest_number` are present with `IF NOT EXISTS`.
3. **Warehouse Auto-Apply Threshold**:
   - `074_ecosystem_hardening_and_parity.sql:41-45`: Adds `auto_apply_threshold_tiyin BIGINT DEFAULT 60000000` and syncs with `ump_auto_apply_threshold_tiyin`.
4. **Quarantine Bin Sealing**:
   - `074_ecosystem_hardening_and_parity.sql:48-105`: PL/pgSQL block inspects `warehouse_bins`, `inventory_bins`, and `warehouse_locations`. Defensively handles `UNIQUE` constraint on `warehouse_bins.location_code` by suffixing warehouse id substring when multiple warehouses exist.
5. **Driver Cash Drawer & CIT Limit**:
   - `074_ecosystem_hardening_and_parity.sql:153-158`: Adds `current_cash_drawer_minor BIGINT DEFAULT 0`, raises `cash_bag_limit_tiyins` default to `10000000000` (100M UZS in tiyins), and adds `deposit_type VARCHAR(64) DEFAULT 'END_OF_SHIFT_BANK_DEPOSIT'` to `driver_cash_deposits`.
6. **Zero Cross-Contamination**:
   - Zero imports or occurrences of `spanner` or `kafka` in `database/migrations/074*` or `backend/internal/db/migration_074_test.go`.
7. **Test Suite Execution**:
   - Executed `go test -count=1 -v -race ./internal/db/...` in `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend`.
   - Result: All tests passed cleanly in 1.504s with zero race conditions.

---

### 1.2 Verbatim Observations of Defect (Data Type Incompatibility)

In `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/database/migrations/074_ecosystem_hardening_and_parity.sql`:

1. **Lines 108–109 (`fleet_rescue_incidents`)**:
   ```sql
   ALTER TABLE fleet_rescue_incidents ADD COLUMN IF NOT EXISTS stranded_manifest_id UUID;
   ALTER TABLE fleet_rescue_incidents ADD COLUMN IF NOT EXISTS rescue_manifest_id UUID;
   ```
2. **Lines 118–120 (`manifest_stop_transfers`)**:
   ```sql
   original_manifest_id UUID,
   target_manifest_id UUID,
   order_id UUID,
   ```
3. **Lines 134–135, 140 (`doorstep_handshake_tokens`)**:
   ```sql
   order_id UUID NOT NULL,
   retailer_id UUID NOT NULL,
   driver_id UUID,
   ```
4. **Line 163 (`soliq_fiscal_receipts`)**:
   ```sql
   order_id UUID NOT NULL,
   ```

Contrast with the rest of the database schema and application codebase:
- `orders.order_id` is defined as `VARCHAR(64)` across all migrations (e.g. `001_initial_schema.sql:81`, `004_enterprise_fiscal_dispatch_and_compliance.sql`, `041_manifest_stops_and_epod_records.sql:20`, `067_soliq_facturas.sql:6`).
- In `backend/internal/api/handlers_order.go:33`, order IDs are generated as:
  ```go
  req.OrderID = "ord_" + uuid.New().String()[:8]
  ```
  (e.g., `"ord_a1b2c3d4"`).
- `retailers.retailer_id` is defined as `VARCHAR(64)` in `001_initial_schema.sql:23`.
- `drivers.driver_id` is defined as `VARCHAR(64)` in `001_initial_schema.sql:35` and generated in `fleet/service.go:266` as `"drv_" + uuid.New().String()[:12]`.
- `manifests.manifest_id` is defined as `VARCHAR(64)` in `001_initial_schema.sql:112` and in existing packages as strings like `"mnf_101"`, `"mnf_tashkent_881"`, `"MNF-4010"`.

---

## 2. Logic Chain

1. In PostgreSQL, the `UUID` data type requires strict 128-bit hexadecimal format (32 hex digits separated by hyphens, e.g. `a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11`). Any string that includes non-hex characters or prefixes (such as `"ord_a1b2c3d4"`, `"drv_f89012"`, `"mnf_tashkent_881"`) causes PostgreSQL to immediately abort the query with:
   `ERROR: invalid input syntax for type uuid` (SQLSTATE `22P02`).
2. Every existing table in `pegasus.x` uses `VARCHAR(64)` for `order_id`, `retailer_id`, `driver_id`, and `manifest_id`.
3. In `074_ecosystem_hardening_and_parity.sql`, Worker M1 assigned `UUID` to:
   - `fleet_rescue_incidents.stranded_manifest_id`
   - `fleet_rescue_incidents.rescue_manifest_id`
   - `manifest_stop_transfers.original_manifest_id`
   - `manifest_stop_transfers.target_manifest_id`
   - `manifest_stop_transfers.order_id`
   - `doorstep_handshake_tokens.order_id`
   - `doorstep_handshake_tokens.retailer_id`
   - `doorstep_handshake_tokens.driver_id`
   - `soliq_fiscal_receipts.order_id`
4. When downstream services (Milestone 3 for Fleet Rescue and Stop Transfers, Milestone 4 for Doorstep Handshake Tokens and Soliq Fiscal Receipts) attempt to insert or query records using actual domain entity IDs (`ord_...`, `drv_...`, `mnf_...`), the database will reject the operations with fatal syntax errors.
5. In `migration_074_test.go:112-123`, the test explicitly verified `UUID` for `stranded_manifest_id` and `rescue_manifest_id`, and merely checked `strings.Contains` for the other column names without asserting compatible data types.

---

## 3. Caveats

- The surrogate primary keys of the new tables (`transfer_id UUID PRIMARY KEY DEFAULT gen_random_uuid()`, `token_id UUID PRIMARY KEY DEFAULT gen_random_uuid()`, and `receipt_id UUID PRIMARY KEY DEFAULT gen_random_uuid()`) and `incident_id UUID` (which references `fleet_rescue_incidents.id UUID`) are correct and should remain `UUID`.
- Only the foreign entity reference columns referencing `manifests`, `orders`, `retailers`, and `drivers` must be updated to `VARCHAR(64)`.

---

## 4. Conclusion & Required Action

**Verdict**: **REQUEST_CHANGES**

Worker M1 must make the following adjustments:

### Required Remediation in `074_ecosystem_hardening_and_parity.sql`:
1. Change `fleet_rescue_incidents` added columns:
   ```sql
   ALTER TABLE fleet_rescue_incidents ADD COLUMN IF NOT EXISTS stranded_manifest_id VARCHAR(64);
   ALTER TABLE fleet_rescue_incidents ADD COLUMN IF NOT EXISTS rescue_manifest_id VARCHAR(64);
   ```
2. In `manifest_stop_transfers`, change:
   - `original_manifest_id VARCHAR(64)`
   - `target_manifest_id VARCHAR(64)`
   - `order_id VARCHAR(64)`
   (`transfer_id UUID` and `incident_id UUID` remain as-is).
3. In `doorstep_handshake_tokens`, change:
   - `order_id VARCHAR(64) NOT NULL`
   - `retailer_id VARCHAR(64) NOT NULL`
   - `driver_id VARCHAR(64)`
   (`token_id UUID` remains as-is).
4. In `soliq_fiscal_receipts`, change:
   - `order_id VARCHAR(64) NOT NULL`
   (`receipt_id UUID` remains as-is).

### Required Remediation in `migration_074_test.go`:
1. Update regex assertions in `FleetRescueIncidentColumns` to verify `VARCHAR` instead of `UUID`:
   ```go
   patternStranded := `(?i)ALTER\s+TABLE\s+fleet_rescue_incidents\s+ADD\s+COLUMN\s+IF\s+NOT\s+EXISTS\s+stranded_manifest_id\s+VARCHAR`
   patternRescue := `(?i)ALTER\s+TABLE\s+fleet_rescue_incidents\s+ADD\s+COLUMN\s+IF\s+NOT\s+EXISTS\s+rescue_manifest_id\s+VARCHAR`
   ```

---

## 5. Verification Method

To verify the remediation after Worker M1 updates the files:

1. **Verify Column Types in Migration 074**:
   ```bash
   grep -E "stranded_manifest_id|rescue_manifest_id|original_manifest_id|target_manifest_id|order_id|retailer_id|driver_id" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/database/migrations/074_ecosystem_hardening_and_parity.sql
   ```
   *Expectation*: All foreign references to manifests, orders, retailers, and drivers use `VARCHAR(64)`.
2. **Execute Unit Tests**:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
   go test -count=1 -v -race ./internal/db/...
   ```
   *Expectation*: All tests PASS.
