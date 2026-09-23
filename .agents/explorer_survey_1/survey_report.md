# Comprehensive Infrastructure & Database Architecture Survey Report
**Target Codebase**: `pegasus.x` (Sovereign Lean Single-Tenant National Operating Core)  
**Survey Agent**: Explorer Survey 1 (Database Migrations, Outbox, and Core Infrastructure)  
**Date**: 2026-09-22T20:40:00Z  
**Primary References**: 
- `pegasus.x/database/migrations/` (Migrations 001–073)
- `pegasus.x/backend/internal/db/`
- `pegasus.x/backend/internal/outbox/`
- `pegasus.x/backend/internal/redis/`
- Approved Hardening Specification: `prompt_draft.md`

---

## Executive Summary

`pegasus.x` is the Sovereign Lean Single-Tenant National Core architecture within `V.O.I.D.`, designed specifically for the Republic of Uzbekistan FMCG wholesale supply chain ecosystem. Its persistence is strictly **PostgreSQL 16** via `github.com/jackc/pgx/v5/pgxpool`, and its messaging/event streaming relies on **Redis 7 Streams** (`XADD`) with an atomic transactional outbox relay worker (`outbox_events` table).

This survey conducted an in-depth, line-by-line inspection of:
1. The full directory layout of `pegasus.x` (backend, migrations, config, infra).
2. The entire inventory of **74 migration files** (`001_initial_schema.sql` through `073_drop_b2b_cash_limit_constraint.sql`).
3. The PostgreSQL connection pool setup (`pgxpool`) and transactional outbox relay worker.
4. The Redis 7 connection, stream publication (`XADD`), and consumer group status.
5. The schema deltas required to support all 7 ecosystem roles according to `prompt_draft.md`.
6. Codebase boundary verification (zero Spanner/Kafka pollution).

A critical defect was uncovered during this survey: **Migration 073 targeted the non-existent table `payments` instead of `order_payment_legs` when attempting to drop the 25M UZS B2B cash limit constraint (`chk_b2b_cash_limit`). As a result, the constraint remains active on `order_payment_legs` in PostgreSQL, which will cause runtime SQL errors whenever cash collection exceeds 25,000,000 UZS.**

---

## 1. Directory Layout & Core Infrastructure Topology

```
/Users/shakhzod/Desktop/V.O.I.D/pegasus.x
├── backend/                             # Sovereign Go Backend Core (Go 1.26.0)
│   ├── cmd/
│   │   ├── server/main.go               # HTTP API server & background workers entrypoint
│   │   └── smokecheck/main.go           # End-to-end integration verification suite
│   ├── internal/                        # 83 domain packages
│   │   ├── api/                         # Chi router, HTTP handlers, middleware
│   │   ├── config/                      # Environment & HashiCorp Vault secrets ingestion
│   │   ├── db/                          # pgxpool connection pool & migration runner
│   │   ├── doorstep/                    # Proximity alerts, exceptions, urban canyon drift
│   │   ├── epod/                        # Electronic Proof of Delivery & offline handover
│   │   ├── fleet/                       # Vehicle management, shift pairing, DVIR
│   │   ├── outbox/                      # Transactional outbox emitter & relay worker
│   │   ├── payload/                     # Dock bay loading, 3L-CVRP axle statics, sealing
│   │   ├── payment/                     # Storefront tender settlement, GlobalPay, ledger
│   │   ├── redis/                       # Redis 7 client, GeoAdd telemetry, XADD publisher
│   │   ├── soliq/                       # E-Factura, E-IMZO PKCS#7 signing, fiscal QR
│   │   ├── warehouse/                   # WMS facility management, approval settings
│   │   └── ...                          # 72 additional domain packages
│   ├── go.mod                           # Strict PG16 + Redis dependencies (no Spanner, no Kafka)
│   └── go.sum
├── database/
│   ├── migrations/                      # 74 SQL migrations (001 to 073)
│   └── seeds/                           # 31 SQL test/reality seed scripts
├── apps/                                # Client applications (Next.js 15, Tauri v2, Android, iOS)
│   ├── admin-portal/
│   ├── driver-app-android/
│   ├── driver-mobile-swift/
│   ├── retailer-portal/
│   ├── supplier-desktop/
│   └── warehouse-desktop/
├── packages/                            # Shared libraries
│   ├── i18n/                            # Multilingual catalogs (uz, ru, en)
│   ├── optimizer-contract/              # CVRP / 3L-CVRP data structures
│   ├── types/                           # Shared TypeScript type definitions
│   └── ui-kit/                          # Void Tactical Design System components
└── infra/                               # Deployment configuration (Docker, Caddy, K8s, Terraform)
```

---

## 2. PostgreSQL Connection Pool (`pgxpool`) Analysis

**File**: `backend/internal/db/postgres.go`

### Configuration Attributes:
- **Driver**: `github.com/jackc/pgx/v5/pgxpool` (v5.10.0)
- **Max Connections (`MaxConns`)**: `25` (tailored for lean sovereign deployment on $100–$150/mo cloud tier, e.g. Servercore TAS Tier III)
- **Min Connections (`MinConns`)**: `5`
- **Max Connection Lifetime (`MaxConnLifetime`)**: `1 * time.Hour`
- **Max Connection Idle Time (`MaxConnIdleTime`)**: `15 * time.Minute`
- **Ping Verification**: Enforced at startup with a 5-second context timeout.

### Transaction Management (`RunInTx`):
- Executes closures within an explicit PostgreSQL transaction:
  ```go
  tx, err := p.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
  ```
- **Panic Safety**: Includes `defer recover()` with automatic rollback before re-throwing panic.
- **Clean Commits**: Guarantees rollback on error; commits on `nil` error.

### In-Code Migration Engine (`backend/internal/db/migrate.go`):
- Creates `schema_migrations (version, name, applied_at, execution_time_ms)` table if not present.
- Discovers `.sql` files in `cfg.MigrationsDir` (default `../database/migrations`), sorts them alphabetically, and executes unapplied migrations within isolated transactions.

---

## 3. Transactional Outbox Engine Analysis

**Files**: `backend/internal/outbox/emitter.go`, `backend/internal/outbox/relay.go`

### 1. Atomic Outbox Emitter (`emitter.go`):
```go
func Emit(ctx context.Context, tx pgx.Tx, aggregateType, aggregateID, eventType string, payload interface{}) error
```
- Marshals arbitrary Go structs/maps into JSON bytes.
- Inserts atomically into table `outbox_events` inside the active `pgx.Tx` transaction.
- Guarantees zero phantom state transitions and zero dual-write inconsistencies.

### 2. Outbox Relay Worker (`relay.go`):
- **Polling Loop**: Runs every `500ms` (configurable) with batch size `50`.
- **Concurrency Guard**:
  ```sql
  SELECT event_id, aggregate_type, aggregate_id, event_type, payload
  FROM outbox_events
  WHERE NOT published
  ORDER BY created_at ASC
  LIMIT $1
  FOR UPDATE SKIP LOCKED
  ```
  `FOR UPDATE SKIP LOCKED` allows horizontal scalability across multiple backend server replicas without event duplication or lock contention.
- **Dual-Bus Emission**:
  1. **Persistent Streaming (Redis 7 Streams)**:
     Publishes via `XADD` to stream key: `stream:<aggregate_type>:events` (e.g. `stream:order:events`, `stream:manifest:events`). Capped at 100,000 events (`MaxLen: 100000, Approx: true`).
  2. **Ephemeral Fanout (Redis Pub/Sub)**:
     Publishes to channel `events:<aggregateType>` for live WebSocket client broadcasting (`ws.Hub`).
- **Dead-Letter Isolation**:
  If Redis `XADD` fails, the error is caught and logged to `outbox_dead_letters` table with error trace.
- **Idempotent Acknowledgment**:
  ```sql
  UPDATE outbox_events
  SET published = TRUE, published_at = NOW()
  WHERE event_id = $1
  ```

---

## 4. Redis 7 Setup & Streams Architecture

**File**: `backend/internal/redis/client.go`

### Capabilities Implemented:
1. **Connection Pooling**: `github.com/redis/go-redis/v9` (v9.22.0) with startup ping validation (3s timeout).
2. **Geospatial Tracking (`UpdateDriverTelemetry`)**:
   - Executes atomic pipeline:
     - `GeoAdd("drivers:active", driverID, lng, lat)`
     - `Set("driver:presence:<driverID>", "ONLINE", presenceTTL)`
3. **Event Streaming (`XAddFleetEvent`)**:
   - Writes durable fleet events into stream `events:fleet`.
   - MaxLen: 100,000 (approximate trimming).
   - Fans out to Pub/Sub channel `events:fleet`.

### Critical Finding on Consumer Groups:
- **`XADD` is implemented and functional** across `outbox/relay.go` and `fleet/service.go`.
- **`XREADGROUP` / `XACK` / `XGROUP CREATE` consumer worker pools are currently NOT implemented.**
- Downstream microservices currently rely primarily on Redis Pub/Sub channels connected to `ws.Hub` for WebSocket fanout. To achieve full enterprise resilience and replayability, a Redis Streams consumer group worker pattern (`XREADGROUP` + PEL recovery) should be established.

---

## 5. Migration Inventory & Existing Schema Reality

Total migrations: **74 files** (`001_initial_schema.sql` through `073_drop_b2b_cash_limit_constraint.sql`). Note that version `004` exists as two separate migration files: `004_dispatch_capacities_and_locks.sql` and `004_enterprise_fiscal_dispatch_and_compliance.sql`.

### Inventory of Core Tables:

| Domain / Table Name | Migration Created | Key Columns & Data Types | Current Schema Status |
| :--- | :--- | :--- | :--- |
| `suppliers` | `001`, `069` | `supplier_id`, `name`, `legal_tax_id`, `tax_id` (STIR), `phone`, `password_hash`, `onboarding_status` | Complete. Has STIR unique constraint, onboarding wizard statuses. |
| `warehouses` | `001`, `069`, `070`, `072` | `warehouse_id`, `supplier_id` (nullable), `name`, `latitude`, `longitude` (`DOUBLE PRECISION`), `facility_type`, `auto_order_approval_mode`, `ump_auto_apply_threshold_tiyin`, `auto_approval_enabled`, `max_discrepancy_tolerance_pct` | Implemented. Uses `ump_auto_apply_threshold_tiyin` for auto-approval. |
| `products` | `069` | `product_id`, `supplier_id`, `name`, `barcode` (unique), `mxik_code` (17-char), `package_code`, `units_per_case`, `unit_price_tiyin` (`BIGINT`), `vat_rate` (`12.00`) | Complete dedicated product catalog with 64-bit integer tiyin price and 12% VAT. |
| `skus` | `001`, `004`, `059`, `063`, `069` | `sku_id`, `supplier_id`, `barcode`, `mxik_code`, `package_code`, `unit_price_minor`, `unit_price_tiyin`, `vat_rate`, `category_id`, `pack_type`, `size_volume_ml`, `floor_price_minor`, `min_order_quantity` | Fully synchronized with `products` table. |
| `orders` | `001`, `004`, `025` | `order_id`, `supplier_id`, `warehouse_id`, `retailer_id`, `driver_id`, `vehicle_id`, `status`, `original_total_minor`, `effective_total_minor`, `delivery_latitude`, `delivery_longitude`, `geofence_radius_meters` (150m) | Core order state machine (11 statuses). |
| `manifests` | `001`, `004`, `025`, `072` | `manifest_id`, `warehouse_id`, `driver_id`, `vehicle_id`, `status` (DRAFT, SEALED, IN_TRANSIT, COMPLETED), `digital_seal_hash`, `total_volume_vu`, `plan_fingerprint`, `is_axle_overridden`, `axle_override_reason`, `axle_override_by`, `force_capacity_applied` | **Incomplete**: Missing axle weight columns and bolt seal serial required by `payload/repository.go` and specification. |
| `manifest_stops` | `001`, `041` | `manifest_id`, `order_id`, `stop_sequence`, `retailer_name`, `retailer_address`, `retailer_phone`, `retailer_lat`, `retailer_lng`, `total_amount_minor`, `payment_method`, `status`, `arrived_at`, `completed_at` | Fully enriched for turn-by-turn routing and ePOD handover. |
| `vehicles` | `025` | `vehicle_id`, `supplier_id`, `warehouse_id`, `license_plate`, `make_model`, `vehicle_class`, `max_volume_vu`, `payload_capacity_kg`, `has_refrigeration`, `fuel_type`, `operational_status` | Complete fleet vehicle catalog. |
| `drivers` | `001`, `004`, `025`, `071` | `driver_id`, `supplier_id`, `warehouse_id`, `name`, `phone`, `pinfl` (14-char), `driver_license_number`, `license_categories`, `pin_hash`, `password_hash`, `onboarding_status`, `shift_status`, `cash_bag_limit_tiyins` | Rich driver model with PINFL, license, and shift readiness. |
| `driver_vehicle_assignments` | `025` | `assignment_id`, `supplier_id`, `warehouse_id`, `shift_date`, `driver_id`, `vehicle_id`, `paired_at`, `released_at`, `assignment_type`, `swap_reason`, `previous_vehicle_id` | Enforces 1-to-1 active driver-to-vehicle daily shift pairing and hot-swaps. |
| `vehicle_inspections` | `025` | `inspection_id`, `assignment_id`, `vehicle_id`, `driver_id`, `inspection_type` (PRE_TRIP), `odometer_km`, `fuel_level_pct`, `tires_pressure_status`, `brakes_status`, `lights_signals_status`, `refrigeration_temp_celsius`, `cng_cylinder_seal_valid`, `fire_extinguisher_valid`, `walkaround_passed`, `is_safe_to_operate`, `driver_signature_hash` | Comprehensive digital DVIR checklist. |
| `vehicle_axle_profiles` | `013` | `vehicle_id`, `wheelbase_mm`, `curb_front_kg`, `curb_rear_kg`, `gawr_front_kg`, `gawr_rear_kg`, `max_payload_kg`, `min_steer_ratio_pct` (20%) | Exact physical parameters for 3L-CVRP static moment calculation. |
| `manifest_pallet_allocations` | `013` | `allocation_id`, `manifest_id`, `pallet_id`, `weight_kg`, `position_x_mm`, `compartment_type` | Pallet-level longitudinal distance $x_i$ from steer axle. |
| `fleet_rescue_incidents` | `037` | `id`, `incident_code`, `warehouse_id`, `stranded_vehicle`, `stranded_driver`, `stranded_driver_id`, `location_address`, `latitude`, `longitude`, `reason`, `stops_affected`, `weight_remaining_kg`, `volume_remaining_vu`, `rescue_vehicle`, `rescue_driver`, `status` | Fleet breakdown registry. Missing manifest references. |
| `delivery_epod_records` | `041` | `id`, `manifest_id`, `order_id`, `driver_id`, `recipient_name`, `recipient_role`, `signature_svg`, `photo_proof_url`, `cash_collected_minor`, `delivered_lat`, `delivered_lng`, `geofence_verified`, `distance_meters`, `status` | Lossless vector signature and GPS doorstep verification. |
| `order_payment_legs` | `004`, `005` | `payment_leg_id`, `order_id`, `driver_id`, `method` (CASH, CORPORATE_CARD, BANK_TRANSFER), `amount_minor`, `status`, `gateway`, `provider_tx_id`, `captured_at` | Payment records. **Contains lingering 25M cash limit constraint `chk_b2b_cash_limit`**. |
| `cash_reconciliations` | `004` | `reconciliation_id`, `driver_id`, `shift_date`, `expected_cash_minor`, `actual_cash_minor`, `discrepancy_minor`, `status`, `reconciled_by` | End-of-shift cash drawer balance reconciliation. |
| `driver_cash_deposits` | `066` | `deposit_id`, `supplier_id`, `warehouse_id`, `driver_id`, `manifest_id`, `shift_date`, `expected_cash_minor`, `actual_cash_minor`, `discrepancy_minor`, `deposit_method`, `tamper_bag_barcode`, `status`, `supervisor_user_id` | Depot smart safe validator & sealed tamper bag cash deposits. |
| `signed_facturas` | `067` | `factura_id`, `order_id`, `manifest_id`, `seller_inn`, `buyer_inn`, `total_sum_tiyin`, `total_vat_tiyin`, `total_with_vat_tiyin`, `soliq_doc_uuid`, `soliq_status`, `qr_verification_url` | E-Factura PKCS#7 signed e-invoices. |
| `ledger_journal_entries` | `005` | `entry_id`, `transaction_ref`, `entry_type`, `memo`, `occurred_at` | Double-entry general ledger journal master. |
| `ledger_postings` | `005` | `posting_id`, `entry_id`, `account_code`, `account_type`, `direction` (DEBIT, CREDIT), `amount_minor`, `party_id` | Double-entry general ledger line-item postings. |
| `qm_quarantine_lots` | `010` | `lot_id`, `warehouse_id`, `sku_id`, `order_id`, `ump_code`, `quarantined_qty`, `unit_cost_minor`, `status`, `is_atp_excluded`, `photo_evidence_urls`, `reported_by_driver_id` | ATP-excluded damaged goods quarantine inventory. |
| `warehouse_locations` | `003` | `location_id`, `warehouse_id`, `zone`, `aisle`, `rack`, `shelf`, `bin`, `location_type` (PICK, BULK, QUARANTINE, STAGING) | Slotting locations. |
| `outbox_events` | `002` | `event_id`, `aggregate_type`, `aggregate_id`, `event_type`, `payload`, `published`, `created_at`, `published_at` | Transactional outbox event ledger. |

---

## 6. Critical Gaps & Required Schema Migrations (7-Role Specification)

### Gap 1: Dropping Cash Ceiling Constraint on `order_payment_legs` (Defect in Migration 073)
- **Observation**:
  In `004_enterprise_fiscal_dispatch_and_compliance.sql`, line 71:
  ```sql
  CONSTRAINT chk_b2b_cash_limit CHECK (
      method != 'CASH' OR amount_minor <= 2500000000 -- 25,000,000 UZS in tiyins
  )
  ```
  Migration `073_drop_b2b_cash_limit_constraint.sql` ran:
  ```sql
  ALTER TABLE payments DROP CONSTRAINT IF EXISTS chk_b2b_cash_limit;
  ```
  The table `payments` does not exist in the database; the table is `order_payment_legs`.
- **Impact**: Any driver cash collection exceeding 25,000,000 UZS (2.5B tiyins) will trigger an immediate SQL check constraint violation.
- **Required Migration**:
  ```sql
  ALTER TABLE order_payment_legs DROP CONSTRAINT IF EXISTS chk_b2b_cash_limit;
  ```

### Gap 2: Missing Manifest Sealing & Axle Weight Columns
- **Observation**:
  `backend/internal/payload/repository.go` (`SealManifest`, lines 807–818) attempts to update the following columns on `manifests`:
  - `bolt_seal_serial`
  - `front_axle_kg`
  - `rear_axle_kg`
  - `sealing_inspector_id`
  - `inspection_photo_url`
  - `sealed_at`
  - `updated_at`
  - `manifest_number`
  - `truck_plate`
  - `vehicle_class`
  - `driver_name`
  - `max_volume_vu`
  - `stop_count`
  None of these columns currently exist on `manifests` in any database migration. When running against live PostgreSQL, this query fails and silently falls back to an in-memory map stub (`r.manifests`), violating the Zero Mock Data Policy.
- **Required Migration**:
  Add all missing columns to `manifests`:
  - `manifest_number VARCHAR(64)`
  - `truck_plate VARCHAR(32)`
  - `vehicle_class VARCHAR(16) DEFAULT 'CLASS_B'`
  - `driver_name VARCHAR(128)`
  - `max_volume_vu NUMERIC(10, 2) DEFAULT 150.0`
  - `stop_count INT DEFAULT 0`
  - `front_axle_kg NUMERIC(10, 2) DEFAULT 0.0`
  - `rear_axle_kg NUMERIC(10, 2) DEFAULT 0.0`
  - `steer_tractive_ratio NUMERIC(5, 4) DEFAULT 0.0`
  - `bolt_seal_serial VARCHAR(64)` (pattern `SEAL-UZ-XXXXXX`)
  - `supervisor_pinfl VARCHAR(14)`
  - `supervisor_reason_code VARCHAR(64)`
  - `sealing_inspector_id VARCHAR(64)`
  - `inspection_photo_url TEXT`
  - `sealed_at TIMESTAMPTZ`
  - `updated_at TIMESTAMPTZ DEFAULT NOW()`

### Gap 3: Configurable Warehouse Auto-Approval Threshold Parity
- **Observation**:
  Migration 072 created `ump_auto_apply_threshold_tiyin BIGINT NOT NULL DEFAULT 60000000` on `warehouses`.
  The approved specification specifies:
  `auto_apply_threshold_tiyin` / configurable auto-approval threshold per warehouse.
- **Required Migration**:
  Add `auto_apply_threshold_tiyin BIGINT NOT NULL DEFAULT 60000000` (or synchronize with `ump_auto_apply_threshold_tiyin`) to ensure direct naming compatibility for both order auto-approval and UMP mutation auto-approval.

### Gap 4: Formal Quarantine Location (`WH-QUARANTINE-01`)
- **Observation**:
  `warehouse_locations` supports `location_type = 'QUARANTINE'`, but there is no guaranteed standard quarantine location `WH-QUARANTINE-01` seeded or established across all active warehouses.
- **Required Migration / Seed**:
  Insert default `WH-QUARANTINE-01` location records into `warehouse_locations` and `warehouse_bins` for every active warehouse, ensuring damaged returns can be immediately isolated from pick waves.

### Gap 5: Breakdown Incident Hot-Swap Order Reassignment Tracking
- **Observation**:
  `fleet_rescue_incidents` tracks vehicle breakdown and the assigned rescuer, but lacks references to the `stranded_manifest_id` and `rescue_manifest_id`.
  There is currently no table logging which stops were transferred from the disabled truck to the rescuer truck.
- **Required Migration**:
  - Add `stranded_manifest_id VARCHAR(64) REFERENCES manifests(manifest_id)` to `fleet_rescue_incidents`.
  - Add `rescue_manifest_id VARCHAR(64) REFERENCES manifests(manifest_id)` to `fleet_rescue_incidents`.
  - Create table `manifest_stop_transfers`:
    ```sql
    CREATE TABLE IF NOT EXISTS manifest_stop_transfers (
        transfer_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        incident_id UUID REFERENCES fleet_rescue_incidents(id),
        order_id VARCHAR(64) NOT NULL REFERENCES orders(order_id),
        from_manifest_id VARCHAR(64) NOT NULL REFERENCES manifests(manifest_id),
        to_manifest_id VARCHAR(64) NOT NULL REFERENCES manifests(manifest_id),
        rescuer_driver_id VARCHAR(64) NOT NULL REFERENCES drivers(driver_id),
        transferred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );
    ```

### Gap 6: Doorstep Dynamic Token Handshake Persistence
- **Observation**:
  `fleet/repository.go` currently stubs `VerifyHandshake` with hardcoded `Verified: true`.
  To implement cryptographic dynamic OTP/QR token verification within 100 meters, tokens must be persisted with time-to-live.
- **Required Migration**:
  ```sql
  CREATE TABLE IF NOT EXISTS doorstep_handshake_tokens (
      token_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      order_id VARCHAR(64) NOT NULL REFERENCES orders(order_id),
      retailer_id VARCHAR(64) NOT NULL REFERENCES retailers(retailer_id),
      driver_id VARCHAR(64) REFERENCES drivers(driver_id),
      dynamic_otp VARCHAR(6) NOT NULL,
      qr_token VARCHAR(128) NOT NULL UNIQUE,
      proximity_meters INT NOT NULL DEFAULT 100,
      expires_at TIMESTAMPTZ NOT NULL,
      verified_at TIMESTAMPTZ,
      status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
  );
  ```

### Gap 7: Driver Cash-in-Transit (CIT) Drawer Tracking & Mid-Shift Vault Drops
- **Observation**:
  `drivers` table currently defaults `cash_bag_limit_tiyins` to 25M UZS (2,500,000,000 tiyins). Under the specification, the driver safe transit insurance limit is 100M UZS (10,000,000,000 tiyins).
  `drivers` has no tracking column for current active cash drawer balance (`cit_drawer_balance_minor`).
  `driver_cash_deposits` (066) only supports end-of-shift smart safe reconciliation, with no dedicated support for mid-shift depot or bank vault drop-offs.
- **Required Migration**:
  - Update `drivers` default `cash_bag_limit_tiyins` to `10000000000` (100M UZS).
  - Add `current_cash_drawer_minor BIGINT NOT NULL DEFAULT 0` to `drivers`.
  - Add `deposit_type VARCHAR(32) NOT NULL DEFAULT 'END_OF_SHIFT_RECON'` ('MID_SHIFT_VAULT_DROP', 'END_OF_SHIFT_RECON') and `vault_location_id VARCHAR(64)` to `driver_cash_deposits`.

### Gap 8: Statutory Soliq OFD Fiscal Receipts Table
- **Observation**:
  Migration 067 added `signed_facturas` for E-Factura B2B electronic invoices. However, for B2B cash or card tender settlement at the doorstep, a statutory Soliq OFD fiscal cash-register receipt must be issued with a fiscal mark and dynamic fiscal verification URL.
- **Required Migration**:
  ```sql
  CREATE TABLE IF NOT EXISTS soliq_fiscal_receipts (
      receipt_id VARCHAR(64) PRIMARY KEY,
      order_id VARCHAR(64) NOT NULL REFERENCES orders(order_id),
      payment_leg_id VARCHAR(64) REFERENCES order_payment_legs(payment_leg_id),
      driver_id VARCHAR(64) NOT NULL REFERENCES drivers(driver_id),
      retailer_inn VARCHAR(16) NOT NULL,
      supplier_inn VARCHAR(16) NOT NULL,
      fiscal_sign VARCHAR(64) NOT NULL,
      fiscal_receipt_number VARCHAR(64) NOT NULL,
      terminal_id VARCHAR(32) NOT NULL,
      total_tiyin BIGINT NOT NULL,
      vat_tiyin BIGINT NOT NULL,
      qr_url TEXT NOT NULL,
      raw_payload JSONB NOT NULL,
      status VARCHAR(32) NOT NULL DEFAULT 'CONFIRMED',
      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
  );
  ```

---

## 7. Boundary Check: Spanner & Kafka Cleanliness

A full AST and pattern search across `pegasus.x` confirmed:
1. **Google Cloud Spanner**:
   - `backend/go.mod`: 0 Spanner modules (`cloud.google.com/go/spanner` is completely absent).
   - `backend/`: 0 Spanner imports or Spanner client calls.
   - Note: One string trace label `"SQL spanner.ExecuteBatchPayouts"` exists in `backend/cmd/smokecheck/main.go:2796` (observability span description). Two migration comments (`025` and `059`) reference Spanner DDL as architectural reference.
2. **Apache Kafka**:
   - `backend/go.mod`: 0 Kafka drivers (`sarama`, `kafka-go`, `confluent-kafka-go` are completely absent).
   - `backend/`: 0 Kafka imports, 0 Kafka producers, 0 Kafka consumers.
   - Note: Static deployment artifacts exist in `infra/k8s/kafka/` and `infra/terraform/modules/messaging/`, but no application code interacts with Kafka.

**Conclusion**: The Strict Two-System Architectural Boundary is fully respected in `pegasus.x` Go backend and database migrations.

---

## 8. Summary of Proposed Migration 074

All identified schema enhancements can be cleanly encapsulated into forward migration `074_ecosystem_hardening_and_parity.sql`:
1. Drop `chk_b2b_cash_limit` from `order_payment_legs`.
2. Add missing sealing, axle statics (`front_axle_kg`, `rear_axle_kg`, `steer_tractive_ratio`), bolt seal serial (`bolt_seal_serial`), and supervisor override columns to `manifests`.
3. Add `auto_apply_threshold_tiyin` to `warehouses`.
4. Ensure default `WH-QUARANTINE-01` location records exist in `warehouse_locations`.
5. Add `stranded_manifest_id` and `rescue_manifest_id` to `fleet_rescue_incidents`, and create `manifest_stop_transfers`.
6. Create `doorstep_handshake_tokens` for dynamic OTP/QR token verification.
7. Update `drivers` `cash_bag_limit_tiyins` to 100M UZS and add `current_cash_drawer_minor`.
8. Add `deposit_type` to `driver_cash_deposits`.
9. Create `soliq_fiscal_receipts` table.
