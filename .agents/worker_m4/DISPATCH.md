## 2026-09-22T22:16:15Z
You are Worker M4 for Milestone 4 (Roles 5 & 6: Driver & Retailer Hardening).
Working directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/worker_m4
Target Monorepo: /Users/shakhzod/Desktop/V.O.I.D/pegasus.x
Original Request: /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md
Approved Specification: /Users/shakhzod/.gemini/antigravity-cli/brain/07a686c0-bf89-4aac-8ca4-a8586dc28332/prompt_draft.md
Project Plan: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_9/PROJECT.md
Survey 3 Report: /Users/shakhzod/Desktop/V.O.I.D/.agents/explorer_survey_3/survey_report.md

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A reviewer will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

File Ownership:
You exclusively own:
- /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/doorstep/
- /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/epod/
- /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/retailer/
- /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/fleet/ (driver delivery methods)
- /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/api/handlers_fleet_driver.go
- /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/api/handlers_retailer.go

Objectives:
1. Role 5 (Driver & Doorstep):
   - Unify driver delivery endpoints onto PostgreSQL 16 persistence (internal/epod and internal/payment/handover).
   - PURGE all fake mock stubs in fleet/repository.go and handlers_fleet_driver.go (VerifyHandshake, arrive, deliver, scan-qr, partial-offload, validate-qr returning hardcoded strings or fake tiyins).
   - Wire dynamic doorstep OTP/QR token generation & verification using doorstep_handshake_tokens table (migration 074):
     - Retailer generates dynamic 6-digit OTP / QR token when truck is within 100m geofence.
     - Driver scans QR token (or inputs 6-digit OTP).
     - Backend verifies token in doorstep_handshake_tokens and distance <= 100 meters (or emergency fallback with photo evidence).
   - Itemized offload screen with damaged carton rejection:
     - Driver records damaged cartons with reason codes (TRANSIT_CRUSH, PACKAGE_PUNCTURE, EXPIRED_LOT, RETAILER_REFUSAL).
     - Enforce camera lockout (gallery upload blocked, live camera photo capture only).
     - Real-time bilateral tiyin price recalculation for both driver and retailer screens.
   - Dual-Tender Doorstep Settlement:
     - Unlimited cash collection recorded in driver cash drawer (drivers.current_cash_drawer_minor).
     - Corporate card webhook / softPOS.
     - Soliq OFD fiscal QR receipt generated and stored in soliq_fiscal_receipts.
     - Digital ePoD signed by retailer with timestamped GPS coordinates.
2. Role 6 (Retailer):
   - Enforce pure B2B wholesale procurement scope:
     - Quarantine/deprecate grocery POS, cashier shifts, drawer counting, and shelf counting from internal/retailer and client routes.
     - Ensure clean wholesale ordering: catalog browsing, tiered MOQ pricing, live inbound truck GPS tracking on map, 100m proximity handshake pop-up with dynamic QR/OTP, doorstep inspection, payment selection, Soliq fiscal receipt download.
3. Tests & Verification:
   - Run and write unit tests across internal/doorstep/..., internal/epod/..., internal/retailer/..., internal/api/...
   - Ensure all tests pass cleanly with race detection: go test -count=1 -v -race ./internal/doorstep/... ./internal/epod/... ./internal/retailer/...
   - Ensure zero Spanner and zero Kafka references.
4. Deliver hard handoff report to /Users/shakhzod/Desktop/V.O.I.D/.agents/worker_m4/handoff.md and notify orchestrator.
