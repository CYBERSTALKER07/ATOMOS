# Project: Full-Ecosystem Hardening across 7 Roles in pegasus.x

## Architecture
- **Target System**: Sovereign National Core (`pegasus.x`).
- **Primary Database**: PostgreSQL 16 with `pgx/v5` connection pool (`MaxConns: 25`, `MinConns: 5`, `MaxConnLifetime: 1h`, `MaxConnIdleTime: 15m`).
- **Event Streaming**: Redis 7 Streams (`XADD`/`XREADGROUP` consumer groups) with Transactional Outbox pattern (`pgx.Tx` atomically recording to `outbox_events` and relay worker polling with `FOR UPDATE SKIP LOCKED`).
- **Currency & Money**: Strict 64-bit integer minor units (`tiyins`). Zero floats for currency.
- **Two-System Boundary**: Strictly ZERO Spanner and ZERO Kafka dependencies in `pegasus.x`.
- **Zero Mock Data Policy**: Zero fake in-memory repository fallbacks or hardcoded seeds in production packages.

## Feature Inventory
| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| 1 | DB Migration 074 | Fix `chk_b2b_cash_limit` drop on `order_payment_legs`, add missing `manifests` columns (`front_axle_kg`, `rear_axle_kg`, `steer_tractive_ratio`, `bolt_seal_serial`, `supervisor_pinfl`, `supervisor_reason_code`, etc.), add `auto_apply_threshold_tiyin` to `warehouses`, seed `WH-QUARANTINE-01`, create `manifest_stop_transfers`, `doorstep_handshake_tokens`, `soliq_fiscal_receipts` | M1 | Survey 1 |
| 2 | Supplier Catch Weight | Add catch weight nominal vs certified dock weight tolerance logic to product catalog and outbound weighing | M2 | Survey 2 |
| 3 | Warehouse Auto-Approval Wiring | Wire `ump_auto_apply_threshold_tiyin` (> 600,000 UZS) from `warehouse` settings into `order/service.go` intake flow to auto-approve creditworthy orders and queue others for manual vetting | M2 | Survey 2 |
| 4 | Warehouse Quarantine Bin | Codify `WH-QUARANTINE-01` canonical bin and enforce quarantine segregation for damaged returns excluding them from pick stock | M2 | Survey 2 |
| 5 | Payloader 3L-CVRP & Mock Purge | Purge in-memory mock fallback in `payload/repository.go`; enforce $W_{steer}$, $W_{drive}$, 11,500 kg single axle limit, $\ge 20\%$ steer ratio, supervisor PINFL (14 digits) and bolt seal format `SEAL-UZ-XXXXXX` | M3 | Survey 2 |
| 6 | Dispatcher Breakdown & Hot-Swap | Purge mock seeds from `dispatch/fleet_rescue_service.go`; wire mid-shift dynamic rescue hot-swap stops to rescuer vehicle via Redis Streams without order cancellation | M3 | Survey 2 |
| 7 | Driver Doorstep & Offload Hardening | Wire driver delivery endpoints to PostgreSQL `epod` and `payment/handover` instead of `fleet` mock stubs; add dynamic 6-digit OTP / rotating QR token verification within 100m; damaged carton rejection with camera lockout (gallery upload blocked) and bilateral tiyin recalculation | M4 | Survey 3 |
| 8 | Retailer Pure B2B Procurement Scope | Quarantine/deprecate grocery POS, cashier shifts, and shelf counting from `internal/retailer` and client routes to enforce pure B2B wholesale procurement scope | M4 | Survey 3 |
| 9 | Finance, Soliq VAT & CIT Management | Verify 12% Soliq VAT, Soliq OFD fiscal QR receipts, double-entry ledger balance ($\sum Debits == \sum Credits$); implement Driver CIT drawer threshold (> 100M UZS), mid-shift depot smart safe vault drops, and bank deposit reconciliation | M5 | Survey 3 |
| 10 | Redis Streams Consumer Groups & Outbox | Complete Redis Streams consumer group processing (`XREADGROUP`/`XACK`) in `internal/redis` and ensure reliable outbox relay for all ecosystem events | M5 | Survey 1 |
| 11 | Full Verification & Zero-Regression Test Suite | End-to-end test suite execution (`go test -v -race ./...`) across all packages ensuring 0 race conditions, 0 mock fallbacks, and 0 contract drift | M6 | Survey 3 |

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| M1 | Database Schema & Migration 074 | Create and verify `074_ecosystem_hardening_and_parity.sql` | none | DONE |
| M2 | Roles 1 & 2: Supplier & Warehouse Hardening | Catch weight, auto-approval order intake wiring, quarantine bin `WH-QUARANTINE-01` | M1 | DONE |
| M3 | Roles 3 & 4: Payloader & Dispatcher Hardening | Purge mock fallbacks, 3L-CVRP axle statics, bolt seal `SEAL-UZ-XXXXXX`, breakdown rescue hot-swap | M1 | DONE |
| M4 | Roles 5 & 6: Driver & Retailer Hardening | Unify driver delivery onto `epod`, 100m OTP/QR handshake, damaged offload, quarantine retailer POS | M1 | PLANNED |
| M5 | Role 7 & Redis Streams Hardening | Soliq OFD fiscalization, CIT drawer thresholds & vault drops, Redis Streams consumer groups | M1 | PLANNED |
| M6 | Full-Ecosystem Verification & Test Suite | Execute all test suites with race detector (`go test -v -race ./...`), verify zero mock data, verify two-system boundary | M2, M3, M4, M5 | PLANNED |

## Interface Contracts
### Payload Sealing ↔ Manifest Verification
- `front_axle_kg`, `rear_axle_kg` (FLOAT / INT)
- `steer_tractive_ratio` ($\ge 0.20$)
- `bolt_seal_serial` (`^SEAL-UZ-[0-9A-Z]{6}$`)
- `is_axle_overridden`, `axle_override_reason`, `axle_override_by` (PINFL 14-digit)

### Doorstep Proximity & Handshake
- Retailer displays rotating dynamic 6-digit OTP / QR token
- Driver scans token within 100m haversine geofence
- Handshake record stored in `doorstep_handshake_tokens`
- Rejection of damaged carton triggers real-time price recalculation in integer tiyins

### Rescue Hot-Swapping
- Breakdown event: `events:fleet:breakdown_reported`
- Rescue dispatch: `events:fleet:rescue_dispatched`
- Stop transfer: atomic reassignment in `manifest_stop_transfers` without order cancellation

## Code Layout
- Migrations: `pegasus.x/database/migrations/`
- Domain & Services: `pegasus.x/backend/internal/`
  - `supplier/`, `product/`
  - `warehouse/`, `inbound/`, `crossdock/`, `qm/`
  - `payload/`
  - `dispatch/`, `fleet/`
  - `doorstep/`, `epod/`
  - `retailer/`
  - `fiscal/`, `soliq/`, `payment/`, `cashrecon/`
  - `db/`, `outbox/`, `redis/`
- API Routers: `pegasus.x/backend/internal/api/`
- Tests: package unit tests `*_test.go` and E2E suites in `pegasus.x/backend/`
