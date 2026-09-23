## 2026-09-22T21:26:30Z
You are Worker M3 for Milestone 3 (Roles 3 & 4: Payloader & Dispatcher Hardening).
Working directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/worker_m3
Target Monorepo: /Users/shakhzod/Desktop/V.O.I.D/pegasus.x
Original Request: /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md
Approved Specification: /Users/shakhzod/.gemini/antigravity-cli/brain/07a686c0-bf89-4aac-8ca4-a8586dc28332/prompt_draft.md
Project Plan: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_9/PROJECT.md
Survey 2 Report: /Users/shakhzod/Desktop/V.O.I.D/.agents/explorer_survey_2/survey_report.md

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A reviewer will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

File Ownership:
You exclusively own:
- /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/payload/
- /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/dispatch/
- /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/fleet/

Objectives:
1. Role 3 (Payloader/Picker):
   - CRITICAL: PURGE in-memory mock fallback in `backend/internal/payload/repository.go` (the fake `r.manifests` map and mock fallback). All queries and updates must persist directly to PostgreSQL 16 via `pgxpool.Pool`. Note that `manifests` now has all columns from migration 074 (`front_axle_kg`, `rear_axle_kg`, `steer_tractive_ratio`, `bolt_seal_serial`, `digital_seal_hash`, `supervisor_pinfl`, `supervisor_reason_code`, etc.).
   - 3L-CVRP Longitudinal Statics:
     - Enforce static moment equilibrium:
       W_steer = W_curb,steer + sum(w_i * (L - x_i) / L)
       W_drive = W_curb,drive + sum(w_i * x_i / L)
     - Verify statutory 11,500 kg single axle limit and >= 20.0% steer axle tractive ratio (W_steer / W_gross >= 0.20).
     - Supervisor Override & Bolt Seal:
       - Validate supervisor PINFL (14 digits: `^[0-9]{14}$`).
       - Validate tamper-evident bolt seal serial regex (`^SEAL-UZ-[0-9A-Z]{6}$`).
       - Hash bolt seal with SHA-256 and store in `digital_seal_hash`.
       - Record supervisor override reason and PINFL.
2. Role 4 (Dispatcher):
   - CRITICAL: PURGE hardcoded mock seeds (`RSC-2026-081`, etc.) from `backend/internal/dispatch/fleet_rescue_service.go`. Persist rescue incidents in `fleet_rescue_incidents` and stop reassignments in `manifest_stop_transfers`.
   - Multi-Vehicle VRP Routing:
     - Verify dynamic vehicle pairing, on-shift driver check, and pre-trip DVIR inspection passed before dispatching.
   - Mid-Shift Breakdown & Rescue Hot-Swap:
     - `ReportBreakdown`: record incident with GPS coordinates, odometer, diagnostic code, and remaining undelivered cargo snapshot.
     - `RankRescueCandidates`: rank nearby on-shift vehicles (both idle trucks and active trucks with spare capacity) by Haversine distance and payload headroom.
     - `ExecuteRescue`: transload remaining stops to rescuer vehicle manifest in `manifest_stop_transfers` without order cancellation, update vehicle status, and emit `events:fleet:rescue_dispatched` to Redis Streams (`XADD`). Rescuer mobile app route refreshes immediately.
3. Tests & Verification:
   - Run and write unit tests across `internal/payload/...`, `internal/dispatch/...`, `internal/fleet/...`.
   - Ensure all tests pass cleanly with race detection: `go test -count=1 -v -race ./internal/payload/... ./internal/dispatch/... ./internal/fleet/...`.
   - Ensure zero Spanner and zero Kafka references.
4. Deliver hard handoff report to `/Users/shakhzod/Desktop/V.O.I.D/.agents/worker_m3/handoff.md` and message orchestrator.
