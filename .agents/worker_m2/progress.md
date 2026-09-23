# Progress Heartbeat — Worker M2

Last visited: 2026-09-23T02:47:00+05:00
Current status: Milestone 2 Implementation Complete. All automated test suites passing with race detector.

## Completed Steps
- [x] Initialized DISPATCH.md and BRIEFING.md.
- [x] Investigate current code in:
  - `pegasus.x/backend/internal/supplier/`
  - `pegasus.x/backend/internal/warehouse/`
  - `pegasus.x/backend/internal/order/`
  - `pegasus.x/backend/internal/qm/`
  - `pegasus.x/backend/internal/soliq/eimzo.go`
- [x] Create implementation plan.
- [x] Implement Supplier Catch Weight / Variable Weight tolerances:
  - Models, tolerance validation, nominal vs actual scale adjustment calculations with integer tiyin recalculation.
  - Integration with `internal/supplier/` and `internal/order/catch_weight.go`.
  - Comprehensive unit test suite in `supplier_catch_weight_test.go`.
- [x] Enhance E-Factura PKCS#7 signing with RFC 5652 CMS SignedData container in `internal/soliq/eimzo.go`:
  - ASN.1 CMS types, `CreateSignedDataCMS`, `VerifySignedDataCMS`, and `VerifySignedDataCMSB64`.
  - Seller 9-digit INN matching and validity certificate window validation.
  - Comprehensive unit tests in `eimzo_cms_test.go`.
- [x] Database Migration `076_supplier_catch_weight_and_order_vetting.sql`:
  - Schema updates for `PENDING_APPROVAL` status, `needs_vetting`, catch weight fields, quarantine bin `location_code`, `inbound_blind_scans`, `inbound_shortage_claims`.
- [x] Wire Configurable Order Vetting & Intake Tiers in `internal/order/`:
  - State machine transition for `PENDING_APPROVAL`.
  - Auto-approval vs manual vetting based on warehouse threshold, buyer credit status, and first-time buyer checks.
  - `ApproveVettedOrder` and `RejectVettedOrder` (releasing inventory reservations).
  - Outbound dock scale certified catch weight recording and outbox event emission (`order.catch_weight_adjusted`).
  - Comprehensive test suite in `order_vetting_and_catch_weight_test.go`.
- [x] Enforce Quarantine Segregation (`WH-QUARANTINE-01`) in `internal/qm/` and `warehouse/`:
  - Enforced `CanonicalQuarantineBin = "WH-QUARANTINE-01"` with `is_atp_excluded = true` blocking quarantined stock from pick waves.
  - Implemented `PostgresQMRepo` in `internal/qm/repository.go` eliminating in-memory mock repository fallback.
  - Added `EnsureQuarantineBin` and `IsBinATPExcluded` in `internal/warehouse/`.
- [x] Implement Blind Receiving Variance Reconciliation in `internal/warehouse/`:
  - Ingestion of blind pallet scans without exposing PO expected quantities.
  - Variance detection and automatic creation of `ShortageClaim` records with 64-bit integer tiyin claim amounts.
  - Outbox event emissions (`inbound.shortage_claim_created`, `inbound.blind_reconciled`).
  - Unit tests in `blind_receiving_and_quarantine_test.go`.
- [x] Add/update tests & run verification (`go test -count=1 -v -race` across all 5 packages: PASS).
- [x] Deliver hard handoff report to `.agents/worker_m2/handoff.md`.
