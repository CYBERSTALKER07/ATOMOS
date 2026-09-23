# Dispatch Order — Worker M4 Remediation (Milestone 4 Fixes)

**Timestamp**: 2026-09-23T06:31:00Z  
**Assigned Subagent**: `worker_m4_remediation` (`teamwork_preview_worker`)  
**Working Directory**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/worker_m4_remediation`  
**Target Monorepo**: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`  
**Orchestrator Conversation ID**: `9c492746-e261-4f02-867a-381f30f56aae`  

---

## Mandatory Reference Documents
- **Reviewer 1 Handoff**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/reviewer_m4_1/handoff.md` (MUST read: provides line numbers and exact failure logs)
- **Reviewer 2 Handoff**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/reviewer_m4_2/handoff.md`
- **Original User Request**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md`
- **Approved Specification**: `/Users/shakhzod/.gemini/antigravity-cli/brain/07a686c0-bf89-4aac-8ca4-a8586dc28332/prompt_draft.md`

---

## MANDATORY INTEGRITY WARNING
> DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A teamwork_preview_auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

---

## Architectural Directives
1. Strictly PostgreSQL 16 (`pgx/v5`) + Redis 7 Streams. Zero Spanner or Kafka references.
2. Zero mock data in production packages.
3. Strict 64-bit integer tiyin minor unit arithmetic.

---

## Remediation Tasks

### Task 1: [CRITICAL] Fix Unused Import in `internal/fleet/repository.go`
- **File**: `backend/internal/fleet/repository.go:17`
- Remove `"github.com/pegasus-x/core/internal/doorstep"`.
- Verify compilation of `internal/fleet`, `internal/api`, `cmd/server`, and `cmd/smokecheck`.

### Task 2: [MAJOR] Mount `/v1/retailer/quarantine-status` in `router.go`
- **File**: `backend/internal/api/router.go`
- Inside the Retailer subrouter, mount:
  ```go
  ret.Get("/quarantine-status", s.handleGetRetailerQuarantineStatus)
  ```
- Ensure `handleGetRetailerQuarantineStatus` sets quarantine headers and returns `s.retailerSvc.GetQuarantinedFeatureStatus()`.

### Task 3: [MAJOR] Plug Camera Lockout Bypass in `VerifyHandshake`
- **File**: `backend/internal/doorstep/service.go:167-185`
- Ensure that whenever `distanceMeters > MaxDoorstepGeofenceMeters`, gallery uploads (`GALLERY`, `GALLERY_UPLOAD`, `DEVICE_STORAGE`) are strictly rejected with `ErrCameraLockout`.
- `CaptureSource == CaptureSourceCameraDirect` must be affirmatively required for both emergency photo bypass paths (including when `ShopSignText` is present).

### Task 4: [DEFENSIVE] Whitelist `CaptureSourceCameraDirect` in `ProcessPartialOffload`
- **File**: `backend/internal/doorstep/service.go:259-270`
- When `item.RejectedQty > 0`, affirmatively enforce:
  ```go
  if capSource != CaptureSourceCameraDirect {
      return nil, ErrCameraLockout
  }
  ```

### Task 5: [IDEMPOTENCY] Protect `RecordSettlement` from Duplicate Mobile Retries
- **File**: `backend/internal/doorstep/repository.go:250-265`
- In `RecordSettlement`, if the order status is already `'DELIVERED'`, return early without incrementing `drivers.current_cash_drawer_minor` again.

---

## Verification Requirements
1. Run all unit and integration tests with race detection:
   ```bash
   go test -count=1 -v -race ./internal/doorstep/... ./internal/epod/... ./internal/retailer/... ./internal/fleet/... ./internal/api/...
   ```
2. Run build across all commands and packages:
   ```bash
   go build ./cmd/... ./internal/...
   go vet ./internal/doorstep/... ./internal/epod/... ./internal/retailer/... ./internal/fleet/... ./internal/api/...
   ```
3. Check zero Spanner and zero Kafka references:
   ```bash
   grep -rnE -i "(spanner|kafka)" internal/doorstep internal/epod internal/retailer internal/fleet internal/api
   ```
4. Document all changes and test outputs in `/Users/shakhzod/Desktop/V.O.I.D/.agents/worker_m4_remediation/handoff.md`.
5. Send completion message to orchestrator (`9c492746-e261-4f02-867a-381f30f56aae`).

## 2026-09-23T06:31:22Z
You are Worker M4 Remediation assigned to fix the 4 concrete findings identified by Reviewer 1 and Reviewer 2 in pegasus.x.

Working Directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/worker_m4_remediation
Target Monorepo: /Users/shakhzod/Desktop/V.O.I.D/pegasus.x
Parent Orchestrator Conversation ID: 9c492746-e261-4f02-867a-381f30f56aae

TASKS TO EXECUTE:
1. [CRITICAL] Fix Unused Import in internal/fleet/repository.go:17
2. [MAJOR] Mount /v1/retailer/quarantine-status in backend/internal/api/router.go
3. [MAJOR] Plug Camera Lockout Bypass in backend/internal/doorstep/service.go
4. [DEFENSIVE] Whitelist CaptureSourceCameraDirect in backend/internal/doorstep/service.go
5. [IDEMPOTENCY] Protect RecordSettlement in backend/internal/doorstep/repository.go

