# Dispatch Order — Worker M6 (Full Verification & Monorepo Test Suite)

**Timestamp**: 2026-09-23T06:50:00Z  
**Assigned Subagent**: `worker_m6_verification` (`teamwork_preview_worker`)  
**Working Directory**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/worker_m6_verification`  
**Target Monorepo**: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`  
**Orchestrator Conversation ID**: `9c492746-e261-4f02-867a-381f30f56aae`  

---

## Mandatory Reference Documents
- **Original User Request**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md`
- **Approved Specification**: `/Users/shakhzod/.gemini/antigravity-cli/brain/07a686c0-bf89-4aac-8ca4-a8586dc28332/prompt_draft.md`
- **Master Plan**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_10/plan.md`
- **Gate Status**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_10/GATE_STATUS.md`

---

## MANDATORY INTEGRITY WARNING
> DO NOT CHEAT. All implementations and verifications must be genuine. DO NOT fake test outputs or attestation artifacts. Integrity violations WILL be detected and your work WILL be rejected.

---

## Verification Scope & Objectives
Execute comprehensive monorepo verification for Milestone 6 across `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend`:

1. **Full Monorepo Test Suite Execution with Race Detection**:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
   go test -count=1 -v -race ./...
   ```
   Record package-by-package pass status, execution times, and confirm 0 race conditions.

2. **Monorepo Build & Vet Check**:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
   go build ./cmd/... ./internal/...
   go vet ./...
   ```
   Confirm clean compilation and 0 vet warnings.

3. **Strict Two-System Architectural Boundary Scan**:
   Verify zero Spanner or Kafka contamination across all Go source files and dependencies:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
   grep -rnE -i "(spanner|kafka)" internal/ cmd/
   grep -E "(spanner|kafka|sarama)" go.mod go.sum
   ```
   Must return 0 matches (exit code 1).

4. **Zero Mock Data Policy Audit**:
   Verify no fake in-memory repository fallbacks or hardcoded fake seeds in production packages:
   ```bash
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
   grep -rnE "(MemoryRepository|fakeData|fakeRepo)" internal/
   ```

5. **64-Bit Integer Tiyin Currency Audit**:
   Verify that all currency, pricing, fees, margins, and taxes use `int64` minor units (`tiyins`) with zero floating-point currency math.

6. **Documentation & Handoff**:
   - Write comprehensive report to `/Users/shakhzod/Desktop/V.O.I.D/.agents/worker_m6_verification/handoff.md`.
   - Send completion message to orchestrator via `send_message`.

## 2026-09-23T06:50:00Z
You are Worker M6 Verification assigned to execute the final monorepo test suite and zero-regression audit across pegasus.x/backend.

Working Directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/worker_m6_verification
Target Monorepo: /Users/shakhzod/Desktop/V.O.I.D/pegasus.x
Parent Orchestrator Conversation ID: 9c492746-e261-4f02-867a-381f30f56aae

TASKS TO EXECUTE:
1. Execute full monorepo test suite across all packages in backend with race detection:
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend
   go test -count=1 -v -race ./...
   (If any package fails or takes long, document exact package test status).
2. Execute monorepo build and vet:
   go build ./cmd/... ./internal/...
   go vet ./...
3. Perform static architectural boundary check:
   Verify 0 Spanner and 0 Kafka references in backend:
   grep -rnE -i "(spanner|kafka)" internal/ cmd/
   grep -E "(spanner|kafka|sarama)" go.mod go.sum
4. Audit zero mock data policy in production packages:
   grep -rnE "(MemoryRepository|fakeData|fakeRepo)" internal/
5. Verify 64-bit integer tiyin minor unit arithmetic for currency.
6. Deliver detailed findings in /Users/shakhzod/Desktop/V.O.I.D/.agents/worker_m6_verification/handoff.md.
7. Send completion message to parent orchestrator (9c492746-e261-4f02-867a-381f30f56aae).
