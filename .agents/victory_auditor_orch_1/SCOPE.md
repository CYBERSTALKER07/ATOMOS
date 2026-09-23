# Scope: pegasus.x Independent Victory Audit

## Objective
Conduct an independent, adversarial, blocking victory audit of the full-ecosystem hardening across `pegasus.x` against live code, database schemas, and test execution.

## Audit Pillars & Pass/Fail Criteria

### Pillar 1: Strict Two-System Architectural Boundary
- Sovereign stack: PostgreSQL 16 (`pgx/v5`) + Redis 7 Streams.
- Absolute zero references to Spanner (`cloud.google.com/go/spanner`, Spanner DDL, Spanner mutations) in `pegasus.x/backend` (`cmd/`, `internal/`, `go.mod`, `go.sum`).
- Absolute zero references to Apache Kafka (`sarama`, `kafka-go`, `confluent-kafka-go`) in `pegasus.x/backend`.
- **Threshold**: Zero occurrences. Any match = INSTANT AUDIT FAILURE.

### Pillar 2: Zero Mock Data Policy
- Zero in-memory repository fallbacks (`MemoryRepository`, `fakeRepo`, `fakeData`), dummy seeds, or fake mocks in production packages.
- All domain entities for all 7 roles must persist to PostgreSQL 16 tables via `pgxpool.Pool`.
- **Threshold**: Zero production mock fallbacks. Any match = INSTANT AUDIT FAILURE.

### Pillar 3: Strict 64-Bit Integer Minor Unit Arithmetic
- All money, prices, fees, margins, discounts, and taxes must be calculated and stored strictly in 64-bit integer tiyins (`int64`).
- Zero floating-point arithmetic (`float32`, `float64`) for currency.
- Double-entry general ledger invariant: Total Debits == Total Credits on every financial posting.
- **Threshold**: Mathematically enforced integer arithmetic and ledger balancing.

### Pillar 4: Scope of all 7 Ecosystem Roles
1. **Role 1 (Supplier)**: MXIK 17-digit codes, EAN-13 barcodes with Mod-10 check digit, tiered MOQ discounts, batch/lot FEFO, catch weight tolerance in tiyins, E-Factura PKCS#7 signing.
2. **Role 2 (Warehouse Admin)**: Auto-approval threshold (> 600,000 UZS) vs vetting queue (PENDING_APPROVAL), cross-docking, blind receiving variance reconciliation, quarantine segregation (WH-QUARANTINE-01), FEFO pick allocation.
3. **Role 3 (Payloader/Picker)**: 3L-CVRP longitudinal static moment axle calculations (W_steer, W_drive), statutory 11,500 kg single axle limit, >= 20% steer tractive authority, supervisor PINFL override, digital bolt seal serialization (SEAL-UZ-XXXXXX).
4. **Role 4 (Dispatcher)**: Multi-vehicle VRP routing, pre-flight gates (pairing, on-shift, DVIR), mid-shift breakdown rescue hot-swapping via Redis Streams (events:fleet:rescue_dispatched).
5. **Role 5 (Driver)**: Doorstep 100m proximity trigger, dynamic OTP/QR token verification, itemized damaged carton offload with native camera lockout (CAMERA_DIRECT whitelist), bilateral tiyin recalculation, dual-tender settlement, Soliq OFD fiscal QR receipts, digital ePoD.
6. **Role 6 (Retailer)**: Pure B2B wholesale procurement terminal ONLY. In-store retail grocery POS, cashier shifts, and shelf counting strictly quarantined.
7. **Role 7 (Finance & Auditor)**: 12% Soliq VAT, Soliq OFD fiscal QR receipts, double-entry general ledger balance, Driver Cash-in-Transit (CIT) drawer tracking (> 100M UZS threshold), depot smart safe vault drops, bank deposit reconciliation.
- **Threshold**: Live code and test verification for every role capability.

### Pillar 5: Independent Live Build & Race-Free Test Execution
- `go build ./cmd/... ./internal/...` in `pegasus.x/backend` must exit 0.
- `go test -count=1 -race ./...` in `pegasus.x/backend` must pass with 0 failures, 0 panics, 0 race conditions.
- **Threshold**: 100% clean exit codes.

## Auditor Assignment Plan
1. `victory_auditor_worker_1` (Worker): Execute live builds, full race test suite, boundary scans, mock data scans, and arithmetic/GL checks.
2. `victory_auditor_reviewer_1` (Reviewer): Deep adversarial code audit across all 7 ecosystem roles, verifying implementation details, DB schema/migrations, and business logic conformance.
