## 2026-09-22T21:26:26Z
Worker M2 Dispatch:
You are Worker M2 for Milestone 2 (Roles 1 & 2: Supplier & Warehouse Hardening).
Working directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/worker_m2
Target Monorepo: /Users/shakhzod/Desktop/V.O.I.D/pegasus.x
Original Request: /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md
Approved Specification: /Users/shakhzod/.gemini/antigravity-cli/brain/07a686c0-bf89-4aac-8ca4-a8586dc28332/prompt_draft.md
Project Plan: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_9/PROJECT.md
Survey 2 Report: /Users/shakhzod/Desktop/V.O.I.D/.agents/explorer_survey_2/survey_report.md

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A reviewer will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

File Ownership:
You exclusively own:
- /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/supplier/
- /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/warehouse/
- /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/order/
- /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/qm/
- /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/soliq/eimzo.go

Objectives:
1. Role 1 (Supplier):
   - Implement Catch Weight / Variable Weight tolerances:
     - For variable-weight goods (bulk cheese, meat, produce), order placement reserves nominal weight.
     - Outbound weighing records certified dock catch weight with automatic balance adjustment in integer tiyins.
     - Add catch weight tolerance calculation and helpers in `internal/supplier/` and `internal/order/`.
   - Enhance E-Factura PKCS#7 signing in `internal/soliq/eimzo.go` to support full RFC 5652 CMS SignedData container wrapping the invoice payload and signing certificate with INN validation.
2. Role 2 (Warehouse Admin):
   - Wire Configurable Order Vetting & Intake Tiers:
     - In `backend/internal/order/service.go`, query `auto_apply_threshold_tiyin` / `ump_auto_apply_threshold_tiyin` from `warehouse` settings.
     - If order total <= threshold and buyer is creditworthy, auto-approve order; if total > threshold, credit-blocked, or first-time buyer, place in manual vetting queue (`PENDING_APPROVAL`).
   - Quarantine Segregation (`WH-QUARANTINE-01`):
     - In `backend/internal/qm/quarantine.go` and `warehouse/`, enforce that damaged or returned goods are relocated to `WH-QUARANTINE-01`.
     - Strictly enforce `is_atp_excluded = true` so quarantined goods are completely blocked from available pick waves.
   - Blind Receiving Variance Reconciliation:
     - In `backend/internal/inbound/` or `backend/internal/warehouse/`, ensure receiving scans inbound pallets without exposing expected count; generate shortage claim records on discrepancies.
3. Tests & Verification:
   - Run and write unit tests across `internal/supplier/...`, `internal/warehouse/...`, `internal/order/...`, `internal/qm/...`.
   - Ensure all tests pass cleanly with race detection: `go test -count=1 -v -race ./internal/supplier/... ./internal/warehouse/... ./internal/order/... ./internal/qm/... ./internal/soliq/...`.
   - Ensure zero Spanner and zero Kafka references.
4. Deliver hard handoff report to `/Users/shakhzod/Desktop/V.O.I.D/.agents/worker_m2/handoff.md` and message orchestrator.
