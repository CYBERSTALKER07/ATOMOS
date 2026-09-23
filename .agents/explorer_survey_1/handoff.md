# Handoff Report — Explorer Survey 1 (Database Migrations, Outbox, and Core Infrastructure)

**Agent**: Explorer Survey 1  
**Working Directory**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/explorer_survey_1`  
**Target Codebase**: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`  
**Timestamp**: 2026-09-22T20:45:00Z  
**Type**: Hard Handoff  

---

## 1. Observation

### 1.1 Directory Structure & Infrastructure Setup
- `pegasus.x` Go backend is defined in `backend/go.mod` (Go 1.26.0) under module `github.com/pegasus-x/core`.
- Database migrations directory is located at `database/migrations/` containing 74 files (`001_initial_schema.sql` through `073_drop_b2b_cash_limit_constraint.sql`).
- PostgreSQL pool is initialized in `backend/internal/db/postgres.go:18-43` using `pgxpool.ParseConfig` with `MaxConns = 25`, `MinConns = 5`, `MaxConnLifetime = 1 * time.Hour`, `MaxConnIdleTime = 15 * time.Minute`.
- Outbox emitter is defined in `backend/internal/outbox/emitter.go:12-28`, inserting into `outbox_events (aggregate_type, aggregate_id, event_type, payload)` inside an active `pgx.Tx`.
- Outbox relay is defined in `backend/internal/outbox/relay.go:42-139`, polling `outbox_events` with `ORDER BY created_at ASC LIMIT $1 FOR UPDATE SKIP LOCKED`, publishing to Redis 7 Streams (`streamKey := fmt.Sprintf("stream:%s:events", strings.ToLower(it.aggregateType))`) via `XADD`, publishing to Redis Pub/Sub (`events:<aggregateType>`) via `PublishEvent`, and updating `published = TRUE, published_at = NOW()`.
- Redis client in `backend/internal/redis/client.go:18-87` implements `Connect`, `UpdateDriverTelemetry` (GeoAdd + presence pipeline), `PublishEvent`, and `XAddFleetEvent` (publishing to stream `events:fleet` with capped trimming `MaxLen: 100000`).
- Consumer groups (`XREADGROUP`, `XACK`, `XGROUP CREATE`) are NOT implemented in `pegasus.x/backend` (zero results for `XReadGroup` or `XGroup`).

### 1.2 Boundary Check (Spanner & Kafka)
- `backend/go.mod` contains zero dependencies on `cloud.google.com/go/spanner` or Kafka drivers (`sarama`, `kafka-go`, `confluent-kafka-go`).
- `grep -i "spanner"` in `pegasus.x/backend` returned only one match: `backend/cmd/smokecheck/main.go:2796` (`_, childSpan := tracer22.Start(traceCtx, "SQL spanner.ExecuteBatchPayouts")`), which is a text string in an observability span.
- `grep -i "kafka"` in `pegasus.x/backend` returned 0 matches.
- In `database/migrations/`, `spanner` appears only as comments referencing architectural prototypes in `025_fleet_and_driver_lifecycle_management.sql:5` and `059_supplier_product_catalog_base.sql:5`. `kafka` returned 0 matches.

### 1.3 Key Schema Observations & Discrepancies
- **Defect in Migration 073**:
  `database/migrations/073_drop_b2b_cash_limit_constraint.sql:5` executes:
  ```sql
  ALTER TABLE payments DROP CONSTRAINT IF EXISTS chk_b2b_cash_limit;
  ```
  However, table `payments` does not exist in `pegasus.x`. The constraint was created in `004_enterprise_fiscal_dispatch_and_compliance.sql:71-73` on table `order_payment_legs`:
  ```sql
  CONSTRAINT chk_b2b_cash_limit CHECK (
      method != 'CASH' OR amount_minor <= 2500000000 -- 25,000,000 UZS in tiyins
  )
  ```
  Therefore, the constraint `chk_b2b_cash_limit` remains active on `order_payment_legs`.
- **Missing Columns in `manifests`**:
  `backend/internal/payload/repository.go:807-833` runs:
  ```sql
  UPDATE manifests
  SET status = 'SEALED', bolt_seal_serial = $1, digital_seal_hash = $2,
      front_axle_kg = $3, rear_axle_kg = $4, sealing_inspector_id = $5,
      inspection_photo_url = $6, sealed_at = $7, updated_at = $7,
      is_axle_overridden = $8, axle_override_reason = $9, axle_override_by = $10
  WHERE manifest_id = $11
  RETURNING manifest_id, manifest_number, warehouse_id, vehicle_id, truck_plate,
            vehicle_class, driver_id, driver_name, status, total_volume_vu,
            max_volume_vu, stop_count, front_axle_kg, rear_axle_kg, bolt_seal_serial,
            digital_seal_hash, created_at, updated_at,
            is_axle_overridden, axle_override_reason, axle_override_by
  ```
  The columns `manifest_number`, `truck_plate`, `vehicle_class`, `driver_name`, `max_volume_vu`, `stop_count`, `front_axle_kg`, `rear_axle_kg`, `bolt_seal_serial`, `sealing_inspector_id`, `inspection_photo_url`, `sealed_at`, `updated_at` are NOT defined on `manifests` in any database migration. If executed on a live database, this query errors out and falls back to an in-memory stub (`r.manifests`).
- **Warehouse Auto-Approval Column**:
  `database/migrations/072_warehouse_auto_approval_and_vetting_settings.sql:6` added `ump_auto_apply_threshold_tiyin BIGINT NOT NULL DEFAULT 60000000`. The specification expects `auto_apply_threshold_tiyin`.
- **Doorstep Handshake Verification Stub**:
  `backend/internal/fleet/repository.go:2504-2510` contains:
  ```go
  func (r *Repository) VerifyHandshake(ctx context.Context, driverID string, req VerifyHandshakeRequest) (*VerifyHandshakeResponse, error) {
      return &VerifyHandshakeResponse{
          Verified: true,
          OrderID:  req.OrderID,
          Status:   "HANDSHAKE_VERIFIED",
      }, nil
  }
  ```
  This is a static stub returning `true` without database persistence, token validation, or geofence verification.
- **Offline Handover Discrepancy Persistence**:
  In `backend/internal/epod/repository.go:636-652` (`ProcessOfflineDeliveryTx`), `rec.Adjustments` is only serialized into the outbox event JSON payload and is NOT inserted into `entity_adjustments` or `credit_notes`.

---

## 2. Logic Chain

1. **Premise 1**: The canonical engineering doctrine (`AGENTS.md`, `GEMINI.md`) mandates a Strict Two-System Architectural Boundary: `pegasus.x` is strictly PostgreSQL 16 + Redis 7 Streams, with 0 Spanner and 0 Kafka dependencies.
   - **Inference**: Our AST and dependency scans of `backend/go.mod`, `backend/cmd/`, `backend/internal/`, and `database/migrations/` confirm 100% adherence. The boundary is completely unpolluted.

2. **Premise 2**: The Zero Mock Data Policy forbids in-memory fallback repositories or fake mock returns in production packages.
   - **Inference**: In `backend/internal/payload/repository.go:801-860`, `SealManifest` attempts an UPDATE against `manifests` using columns that do not exist in the migrations (`front_axle_kg`, `rear_axle_kg`, `bolt_seal_serial`, etc.). If `r.pool` is active, the query fails on live PostgreSQL and the code falls back to `r.manifests[manifestID]`. Adding these missing columns to `manifests` in migration 074 is mandatory to eliminate this fallback.
   - **Inference**: In `backend/internal/fleet/repository.go:2504-2510`, `VerifyHandshake` returns a static dummy struct without checking any database token. Creating a dedicated `doorstep_handshake_tokens` table and wiring it to `VerifyHandshake` eliminates this mock behavior.

3. **Premise 3**: The approved specification (`prompt_draft.md`) mandates unrestricted cash collection at doorstep handover ("Unlimited cash collection supported with dual-party digital confirmation", "zero artificial Central Bank 25M UZS restriction").
   - **Inference**: Migration 073 failed to drop `chk_b2b_cash_limit` from `order_payment_legs` because it ran `DROP CONSTRAINT` on `payments`. Consequently, any cash payment > 25M UZS will be rejected by PostgreSQL. Dropping `chk_b2b_cash_limit` from `order_payment_legs` in migration 074 is required.

4. **Premise 4**: For 3L-CVRP longitudinal static equilibrium, the system requires calculation of front steer axle weight ($W_{\text{steer}}$), rear drive axle weight ($W_{\text{drive}}$), and traction ratio ($\ge 20\%$), as well as an immutable supervisor override logging supervisor PINFL, reason code, and bolt seal serial (`SEAL-UZ-XXXXXX`).
   - **Inference**: `vehicle_axle_profiles` (migration 013) has the physical vehicle wheelbase and curb/GAWR properties. Adding `front_axle_kg`, `rear_axle_kg`, `steer_tractive_ratio`, `bolt_seal_serial`, `supervisor_pinfl`, and `supervisor_reason_code` to `manifests` provides the exact schema required for manifest sealing and compliance auditing.

5. **Premise 5**: Breakdowns require mid-shift rescue hot-swapping without canceling retailer orders.
   - **Inference**: `fleet_rescue_incidents` (migration 037) tracks the incident, but does not link the original manifest to the rescuer manifest, nor does it track individual order reassignments. Adding `stranded_manifest_id`, `rescue_manifest_id`, and table `manifest_stop_transfers` creates complete transactional traceability.

---

## 3. Caveats

- **Test Database Availability**: In the current test environment, unit tests run against in-memory repository mocks or cached test artifacts unless a live PostgreSQL instance is provisioned at `DATABASE_URL`. The schema gaps identified were proven by static analysis and AST inspection of the raw SQL migration files against repository query strings.
- **Frontend Contract Sync**: Client applications (e.g. `driver-app-android`, `warehouse-desktop`) will need to bind to the updated manifest and doorstep token schema once implemented.

---

## 4. Conclusion

1. **Core Infrastructure Readiness**: The PostgreSQL 16 `pgxpool` configuration and transactional outbox relay worker (`outbox_events` -> Redis Streams `XADD` + Redis Pub/Sub) are robust, idiomatic, and follow Google/Microsoft distributed systems standards.
2. **Boundary Compliance**: Zero Spanner and zero Kafka imports or drivers exist in `pegasus.x/backend` and `database/migrations`.
3. **Required Action**: Create database migration `074_ecosystem_hardening_and_parity.sql` to address:
   - Dropping `chk_b2b_cash_limit` from `order_payment_legs`.
   - Adding missing columns to `manifests` (`front_axle_kg`, `rear_axle_kg`, `steer_tractive_ratio`, `bolt_seal_serial`, `supervisor_pinfl`, `supervisor_reason_code`, `manifest_number`, `truck_plate`, `vehicle_class`, `driver_name`, `max_volume_vu`, `stop_count`, `sealing_inspector_id`, `inspection_photo_url`, `sealed_at`).
   - Adding `auto_apply_threshold_tiyin` to `warehouses`.
   - Adding standard `WH-QUARANTINE-01` location records.
   - Adding `stranded_manifest_id` and `rescue_manifest_id` to `fleet_rescue_incidents` and creating `manifest_stop_transfers`.
   - Creating `doorstep_handshake_tokens` table.
   - Adding `current_cash_drawer_minor` to `drivers` and raising `cash_bag_limit_tiyins` default to 100M UZS.
   - Adding `deposit_type` to `driver_cash_deposits`.
   - Creating `soliq_fiscal_receipts` table.

---

## 5. Verification Method

To independently verify the observations and findings:

1. **Verify Boundary Cleanliness**:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
   grep -rn "cloud.google.com/go/spanner" .
   grep -rn "sarama" .
   grep -rn "kafka-go" .
   ```
   *Expected Output*: 0 matches.

2. **Verify Cash Limit Constraint Defect in Migration 073**:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x
   grep -rn "chk_b2b_cash_limit" database/migrations/
   ```
   *Observation*: Found in `004` on table `order_payment_legs` and in `073` on table `payments`.

3. **Verify Manifest Columns in `payload/repository.go` vs Migrations**:
   Inspect `backend/internal/payload/repository.go:807-820` and compare column names against `grep -rn "ALTER TABLE manifests" database/migrations/`.

4. **Verify Backend Build and Unit Tests**:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
   go test -v ./internal/db/... ./internal/config/... ./internal/outbox/...
   ```
   *Expected Output*: All tests pass cleanly.
