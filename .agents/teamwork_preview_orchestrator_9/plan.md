# Orchestration Plan: Full-Ecosystem Hardening across 7 Roles in pegasus.x

## Architectural Invariants
1. Target: Strictly `pegasus.x` using PostgreSQL 16 (`pgx/v5`) + Redis 7 Streams + Transactional Outbox.
2. ZERO Google Cloud Spanner SDKs, Spanner DDL, Spanner mutations, or Apache Kafka.
3. ZERO Mock Data Policy: All domain state dynamically persisted in PostgreSQL 16.
4. Strict 64-bit integer minor unit arithmetic (`tiyins`) for all money, prices, discounts, VAT, and payments. Zero float currencies.
5. Retailer role is strictly B2B wholesale procurement (zero in-store POS, shelf counting, cashier shifts).
6. 100% Passing Automated Tests (`go test -v -race ./...`).

## Execution Phases

### Phase 0: Survey & Exploratory Audit (3 Parallel Explorers)
- **Explorer 1**: DB Migrations, Connection Pool (`pgxpool`), Outbox Relay, and Core Infrastructure in `pegasus.x`.
- **Explorer 2**: Roles 1–4 Backend Packages (`supplier`, `warehouse`, `payload`, `dispatch`) and Existing State vs Spec.
- **Explorer 3**: Roles 5–7 Backend Packages (`driver`, `doorstep`, `retailer`, `finance`, `soliq`, `payment`) and Test Suite baseline.

### Phase 1: Database Migrations & Schema Hardening
- Forward migrations for all 7 roles:
  - Supplier: MXIK 17-digit, EAN-13 barcodes, tiered volume discounts, FEFO lot/batch dates, catch weight fields.
  - Warehouse: Configurable auto-approval threshold per warehouse (`auto_apply_threshold_tiyin`), quarantine location (`WH-QUARANTINE-01`).
  - Payloader: Axle weights (`W_steer`, `W_drive`), tractive ratio, bolt seal serial (`SEAL-UZ-XXXXXX`), supervisor PINFL/override reason.
  - Dispatcher: Breakdown incident records, rescue hot-swap logs, multi-vehicle routes.
  - Driver & Doorstep: Pre-trip DVIR checklist, OTP/QR tokens, damaged item rejection logs, proof photo metadata, ePoD signatures.
  - Retailer: B2B wholesale orders, proximity logs, Soliq fiscal receipts.
  - Finance: Double-entry ledger journal entries, VAT amounts, driver CIT drawer tracking, vault drop manifests.

### Phase 2: Implementation of Role 1 (Supplier) & Role 2 (Warehouse Admin)
- Supplier: 17-digit MXIK, EAN-13 Mod-10 check, tiered volume MOQ discounts, FEFO lot tracking, catch weight tolerances, PKCS#7 E-Factura preparation.
- Warehouse Admin: Dynamic auto-approval threshold (>600,000 UZS) vs manual vetting queue, cross-dock peak waves, blind receiving variance reconciliation, quarantine segregation (`WH-QUARANTINE-01`), FEFO pick allocation.

### Phase 3: Implementation of Role 3 (Payloader/Picker) & Role 4 (Dispatcher)
- Payloader & Picker: 3L-CVRP longitudinal static moment axle calculation ($W_{steer}$, $W_{drive}$), statutory 11,500 kg limit, $\ge 20\%$ steer axle tractive ratio. Manual supervisor override with PINFL, reason code, digital bolt-seal (`SEAL-UZ-XXXXXX`).
- Dispatcher: Multi-vehicle VRP routing, live GPS telemetry, mid-shift breakdown emergency incident reporting, dynamic rescue hot-swapping to nearby truck without order cancellation via Redis Streams.

### Phase 4: Implementation of Role 5 (Driver) & Role 6 (Retailer)
- Driver: Pre-trip DVIR checklist, 100m proximity trigger, scans dynamic OTP/QR token on retailer app, itemized offload screen with damaged carton rejection, native camera lockout (gallery upload blocked), real-time bilateral tiyin price recalculation, dual-tender settlement (unlimited cash collection in drawer or corporate card webhook), Soliq OFD fiscal QR receipt, ePoD digital signature.
- Retailer: Strict B2B wholesale procurement terminal ONLY. Live truck map tracking, 100m proximity handshake pop-up with dynamic QR/OTP, doorstep inspection, payment selection, Soliq fiscal receipt download.

### Phase 5: Implementation of Role 7 (Finance & Auditor) & Redis Streams Outbox Relay
- Finance & Auditor: 12% Soliq VAT, Soliq OFD fiscalization, double-entry general ledger balance ($\sum Debits == \sum Credits$), driver cash-in-transit (CIT) drawer thresholds, mid-shift depot vault drops, end-of-shift bank deposit reconciliation.
- Redis 7 Streams events: `events:payload:sealed`, `events:fleet:breakdown_reported`, `events:fleet:rescue_dispatched`, `events:doorstep:arrived`, `events:doorstep:tender_settled`.
- Atomic PostgreSQL `pgx.Tx` outbox commit + relay worker.

### Phase 6: Comprehensive Review & Verification Gate
- Test writers & reviewers verify all packages compile cleanly and pass with race detector: `go test -v -race ./...`.
- Zero-drift verification: Zero Spanner/Kafka imports, zero mock data, zero currency floats.
- Final gate approval, report generation, and handoff to parent.
