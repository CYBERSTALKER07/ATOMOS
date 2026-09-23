# Dispatch Order — Worker M4 Generation 2 (Milestone 4: Roles 5 & 6)

**Timestamp**: 2026-09-23T06:10:00Z  
**Assigned Subagent**: `worker_m4_gen2` (`teamwork_preview_worker`)  
**Working Directory**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/worker_m4_gen2`  
**Target Monorepo**: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`  
**Orchestrator Conversation ID**: `9c492746-e261-4f02-867a-381f30f56aae`  

---

## Mandatory Reference Documents
- **Original User Request**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md` (MUST read first)
- **Approved Specification**: `/Users/shakhzod/.gemini/antigravity-cli/brain/07a686c0-bf89-4aac-8ca4-a8586dc28332/prompt_draft.md`
- **Master Plan**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_10/plan.md`
- **Milestone 5 Handoff**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/worker_m5/handoff.md`

---

## MANDATORY INTEGRITY WARNING
> DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A teamwork_preview_auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

---

## Architectural Directives
1. **Strictly PostgreSQL 16 (`pgx/v5`) + Redis 7 Streams**: Zero Google Cloud Spanner SDKs or Apache Kafka drivers.
2. **Zero Mock Data Policy**: Zero hardcoded mock data, in-memory repository fallbacks, or dummy stubs in production packages. Purge fake mock stubs in `internal/fleet/repository.go` and `internal/api/handlers_fleet_driver.go`.
3. **Strict 64-Bit Integer Minor Units (`tiyins`)**: All currency, pricing, and taxes must use `int64`. Floating-point arithmetic for currency is strictly prohibited.

---

## Milestone 4 Scope & Requirements

### 1. Role 5: Driver & Doorstep Delivery
- **Unify Driver Delivery Endpoints**:
  - Unify driver delivery and handover directly on PostgreSQL 16 (`internal/epod`, `internal/doorstep`, `internal/payment/handover`).
  - Purge mock repository fallbacks in `internal/fleet/repository.go` and `internal/api/handlers_fleet_driver.go`.
- **Dynamic Doorstep OTP/QR Handshake (`doorstep_handshake_tokens`)**:
  - 100m geofence proximity trigger (Haversine formula calculation).
  - Dynamic rotating 6-digit OTP / QR token generation and validation against `doorstep_handshake_tokens` table.
  - Emergency fallback with storefront photo evidence and supervisor bypass for GPS drift / urban canyons.
- **Itemized Offload & Damaged Carton Rejection**:
  - Reason codes: `TRANSIT_CRUSH`, `PACKAGE_PUNCTURE`, `EXPIRED_LOT`, `RETAILER_REFUSAL`.
  - Native camera lockout: ensure photos are taken via live camera capture (reject gallery file uploads / enforce `source=camera` verification).
  - Bilateral tiyin price recalculation in real time: gross, discount, 12% Soliq VAT, and net payable.
- **Dual-Tender Doorstep Settlement**:
  - Unlimited cash collection recorded to driver cash drawer (`drivers.current_cash_drawer_minor`).
  - Corporate card webhook / softPOS payment leg.
  - Soliq OFD fiscal QR receipt generated and persisted in `soliq_fiscal_receipts` via `internal/soliq` and `internal/fiscal`.
  - Digital ePoD signed with timestamped GPS coordinates.

### 2. Role 6: Retailer Pure B2B Wholesale Scope
- **Deprecate & Quarantine In-Store POS**:
  - Quarantine consumer grocery POS, cashier shifts, drawer counting, and shelf counting in `internal/retailer` and `handlers_retailer.go`.
  - Maintain strictly pure B2B wholesale procurement: catalog browsing, tiered MOQ pricing, live inbound truck GPS tracking on map, 100m proximity handshake pop-up with dynamic QR/OTP, doorstep inspection, payment selection, Soliq fiscal receipt download.

---

## Verification Requirements
1. Run all unit and integration tests with race detection:
   ```bash
   go test -count=1 -v -race ./internal/doorstep/... ./internal/epod/... ./internal/retailer/... ./internal/fleet/... ./internal/api/...
   ```
2. Run build and vet to ensure zero warnings:
   ```bash
   go build ./internal/doorstep/... ./internal/epod/... ./internal/retailer/... ./internal/fleet/... ./internal/api/...
   go vet ./internal/doorstep/... ./internal/epod/... ./internal/retailer/... ./internal/fleet/... ./internal/api/...
   ```
3. Verify zero Spanner and zero Kafka references in the target packages:
   ```bash
   grep -rnE -i "(spanner|kafka)" internal/doorstep internal/epod internal/retailer internal/fleet internal/api
   ```
4. Document all changes, test commands, and outputs in `/Users/shakhzod/Desktop/V.O.I.D/.agents/worker_m4_gen2/handoff.md`.
5. Send completion message to orchestrator via `send_message`.

## 2026-09-23T06:10:41Z
Received user request for Worker M4 Gen 2:
Implement Milestone 4: Roles 5 & 6 (Driver Doorstep Handshake, Damaged Offload with camera lockout, Dual-Tender Settlement, Retailer Pure B2B Wholesale Scope) in pegasus.x.
