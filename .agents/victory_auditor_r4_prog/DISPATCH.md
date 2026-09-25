# Dispatch Mandate: auditor_r4_prog

- **Identity**: `auditor_r4_prog`
- **Archetype**: `teamwork_preview_worker`
- **Working Directory**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/victory_auditor_r4_prog`
- **Authoritative User Request**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md`
- **Orchestrator Handoff to Audit**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_14/handoff.md`

## Audit Mission
You are the programmatic verification specialist for live test, typecheck, and static linting execution.
MANDATORY INTEGRITY WARNING: DO NOT CHEAT. All implementations and verifications must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. Integrity violations WILL be detected and your work WILL be rejected.

You must run the live builds, tests, typechecks, and linting tools via shell commands, capture the exact output, exit codes, and durations, and document them in your report:

1. Frontend Type Checks:
   - Run type checks across `pegasus.x`:
     `cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x && pnpm check-types --force`
     (or run `pnpm typecheck` / `tsc --noEmit` on modified apps if needed).
     Verify exit code 0 and 0 type errors.
2. Automated Static Linting / Audit Scanner Script:
   - Run the audit scanner:
     `python3 /Users/shakhzod/.gemini/antigravity-cli/brain/ddb82dce-4f0b-4193-b133-2e4dcf9308ea/scratch/audit_scanner.py`
     (or write a static scanner script if the path does not exist, checking for unlabeled inputs and un-roled clickable divs across all 16 desktop/web apps).
     Confirm zero unlabeled inputs and zero un-roled clickable divs.
3. Backend Go Test Suites (`pegasus.x`):
   - Run:
     `cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend && go vet ./... && go test -v -count=1 ./internal/...`
     Confirm 100% pass and 0 vet diagnostics.
4. Backend Go Test Suites (`pegasusX`):
   - Run:
     `cd /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go && go test -v -count=1 ./outbox/... ./ar/... ./payment/...`
     Confirm 100% pass.

Output requirement:
Write your comprehensive audit report to `/Users/shakhzod/Desktop/V.O.I.D/.agents/victory_auditor_r4_prog/handoff.md` with sections:
- Status: APPROVE or REQUEST_CHANGES
- Commands Executed: Exact commands, working directories, execution times, exit codes
- Test Results: Full summary of tests run, passed, failed, skipped
- Findings: Any failures or warnings
- Verdict: PASS or FAIL
Send a completion message back to parent when done.

## 2026-09-25T14:23:42Z
You are auditor_r4_prog, a programmatic test & verification worker.
Working directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/victory_auditor_r4_prog
Read your dispatch mandate at /Users/shakhzod/Desktop/V.O.I.D/.agents/victory_auditor_r4_prog/DISPATCH.md
Read the authoritative user request at /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md
Read previous orchestrator handoff at /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_14/handoff.md

MANDATORY INTEGRITY WARNING: DO NOT CHEAT. All implementations and verifications must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. Integrity violations WILL be detected and your work WILL be rejected.

Your mission:
Execute live programmatic verifications and tests:
1. Run frontend type checks:
   Execute `pnpm check-types --force` in /Users/shakhzod/Desktop/V.O.I.D/pegasus.x (and/or typecheck across modified frontend apps). Verify 0 errors, exit code 0.
2. Run automated static linting / audit scanner:
   Execute the audit scanner script:
   `python3 /Users/shakhzod/.gemini/antigravity-cli/brain/ddb82dce-4f0b-4193-b133-2e4dcf9308ea/scratch/audit_scanner.py`
   If that exact script file is absent, run/create a static scanner script in your directory to scan all 16 desktop/web apps for unlabeled inputs and un-roled clickable divs. Confirm 0 unlabeled inputs and 0 un-roled clickable divs.
3. Run backend Go test suites for pegasus.x:
   Execute `cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend && go vet ./... && go test -v -count=1 ./internal/...`
   Confirm 100% pass and 0 vet diagnostics.
4. Run backend Go test suites for pegasusX:
   Execute `cd /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go && go test -v -count=1 ./outbox/... ./ar/... ./payment/...`
   Confirm 100% pass.

Maintain your progress.md and write your final report to /Users/shakhzod/Desktop/V.O.I.D/.agents/victory_auditor_r4_prog/handoff.md with all exact command lines, exit codes, test logs, summaries, and verdict (DONE or FAILED). When done, call send_message to report your findings to parent.
