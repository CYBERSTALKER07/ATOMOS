# Master Plan — Generation 2: Roles 5 & 6 Hardening & Full Monorepo Verification

## Context & Completed Foundation
- **Milestone 1**: Migration 074 & schema hardening (**GATE PASS**).
- **Milestone 2**: Roles 1 & 2 (Supplier Catch Weight, E-Factura CMS, Warehouse Auto-Vetting, Quarantine) (**GATE PASS**).
- **Milestone 3**: Roles 3 & 4 (Payloader 3L-CVRP & Axle Physics, Supervisor Overrides, Dispatcher Rescue Hot-Swap) (**GATE PASS**).
- **Milestone 5**: Role 7 & Redis Streams (Soliq 12% VAT, Fiscal QR, Double-Entry GL, CIT Drawer & Smart Safe Drops, Redis Streams XREADGROUP/XACK/XADD) (**COMPLETED** by `worker_m5`).

---

## Remaining Execution Plan

### Step 1: Milestone 4 Execution (Roles 5 & 6 Hardening)
- **Role 5: Driver & Doorstep Delivery**:
  1. Unify driver delivery endpoints on PostgreSQL 16 (`internal/epod` and `internal/payment/handover`).
  2. Purge fake mock stubs in `internal/fleet/repository.go` and `internal/api/handlers_fleet_driver.go`.
  3. Wire dynamic doorstep OTP/QR handshake tokens via `doorstep_handshake_tokens` table:
     - 100m geofence trigger.
     - Rotating 6-digit OTP / QR token generation & verification.
     - Emergency fallback with photo evidence and supervisor bypass.
  4. Itemized offload with damaged carton rejection:
     - Reason codes (`TRANSIT_CRUSH`, `PACKAGE_PUNCTURE`, `EXPIRED_LOT`, `RETAILER_REFUSAL`).
     - Native camera lockout (block gallery image upload).
     - Bilateral tiyin price recalculation (gross, discount, VAT, net payable).
  5. Dual-tender doorstep settlement:
     - Cash collection recorded to driver cash drawer (`drivers.current_cash_drawer_minor`).
     - Corporate card webhook / softPOS payment leg.
     - Soliq OFD fiscal QR receipt persistence in `soliq_fiscal_receipts`.
     - Digital ePoD signed with timestamped GPS coordinates.
- **Role 6: Retailer Pure B2B Wholesale Scope**:
  1. Deprecate and quarantine consumer in-store grocery POS, cashier shifts, drawer counting, shelf counting in `internal/retailer` and client routes.
  2. Maintain pure B2B wholesale procurement: catalog browsing, tiered MOQ pricing, live inbound truck GPS tracking on map, 100m proximity handshake pop-up with dynamic QR/OTP, doorstep inspection, payment selection, Soliq fiscal receipt download.
- **Verification**:
  - Run `go test -count=1 -v -race ./internal/doorstep/... ./internal/epod/... ./internal/retailer/... ./internal/api/...`

### Step 2: Milestone 4 & 5 Review and Gating
- Dispatch 2 independent reviewers:
  - `reviewer_m4_1`: Review driver doorstep, OTP/QR tokens, itemized offload, and pure B2B retailer scope.
  - `reviewer_m4_2`: Review financial dual-tender, CIT drawer, Soliq OFD receipts, GL invariant, and Redis 7 Streams.
- Record verdicts in `GATE_STATUS.md`. Gate criteria: ALL APPROVE, zero tests failing, zero mock data, zero Spanner/Kafka.

### Step 3: Milestone 6 (Full Verification & Monorepo Test Suite)
- Dispatch test verification worker to execute:
  - `go test -count=1 -v -race ./...` across all packages in `pegasus.x/backend`.
  - Static AST/grep audit: 0 Spanner and 0 Kafka references in `pegasus.x`.
  - Zero mock data scan in production packages.
- Review results.

### Step 4: Final Completion & Reporting
- Compile final handoff report (`handoff.md`).
- Communicate complete status to parent via `send_message`.
