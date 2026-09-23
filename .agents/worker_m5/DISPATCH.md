## 2026-09-22T22:16:15Z

You are Worker M5 for Milestone 5 (Role 7: Finance & Auditor & Redis Streams Hardening).
Working directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/worker_m5
Target Monorepo: /Users/shakhzod/Desktop/V.O.I.D/pegasus.x
Original Request: /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md
Approved Specification: /Users/shakhzod/.gemini/antigravity-cli/brain/07a686c0-bf89-4aac-8ca4-a8586dc28332/prompt_draft.md
Project Plan: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_9/PROJECT.md
Survey 1 & 3 Reports: /Users/shakhzod/Desktop/V.O.I.D/.agents/explorer_survey_1/survey_report.md, /Users/shakhzod/Desktop/V.O.I.D/.agents/explorer_survey_3/survey_report.md

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A reviewer will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

File Ownership:
You exclusively own:
- /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/fiscal/
- /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/soliq/
- /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/payment/
- /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/cashrecon/
- /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/redis/
- /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/outbox/

Objectives:
1. Role 7 (Finance & Auditor):
   - Statutory 12% Soliq VAT calculation and validation in internal/fiscal/.
   - Soliq OFD fiscal QR receipt generation and persistence in soliq_fiscal_receipts table (with 17-digit MXIK, fiscal sign, QR URL).
   - Double-entry general ledger invariant enforcement: sum(Debits) == sum(Credits) across all financial transactions in internal/payment/handover.go and internal/cashrecon/.
   - Driver Cash-in-Transit (CIT) Drawer Management:
     - Real-time tracking of cash held in driver vault (current_cash_drawer_minor).
     - Threshold alert when driver cash exceeds insurance transit limits (100,000,000 UZS / 10B tiyins cash_bag_limit_tiyins), recommending mid-shift vault drop at nearest depot or bank branch.
     - Mid-shift depot smart safe vault drops (driver_cash_deposits with deposit_type = 'MID_SHIFT_VAULT_DROP').
     - End-of-shift cash drawer reconciliation against driver cash manifest (deposit_type = 'END_OF_SHIFT_BANK_DEPOSIT').
2. Redis 7 Streams & Transactional Outbox:
   - Complete consumer group handling (XREADGROUP/XACK, XGroupCreateMkStream) in internal/redis/client.go.
   - Ensure transactional outbox emitter & relay worker in internal/outbox/ reliably stream all events (events:payload:sealed, events:fleet:breakdown_reported, events:fleet:rescue_dispatched, events:doorstep:arrived, events:doorstep:tender_settled) via XADD.
3. Tests & Verification:
   - Run and write unit tests across internal/fiscal/..., internal/soliq/..., internal/payment/..., internal/cashrecon/..., internal/redis/..., internal/outbox/...
   - Ensure all tests pass cleanly with race detection: go test -count=1 -v -race ./internal/fiscal/... ./internal/soliq/... ./internal/payment/... ./internal/cashrecon/... ./internal/redis/... ./internal/outbox/...
   - Ensure zero Spanner and zero Kafka references.
4. Deliver hard handoff report to /Users/shakhzod/Desktop/V.O.I.D/.agents/worker_m5/handoff.md and notify orchestrator.
