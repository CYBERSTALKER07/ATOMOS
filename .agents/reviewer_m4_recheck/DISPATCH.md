# Dispatch Order — Reviewer M4 Recheck (Certification of Milestone 4 Fixes)

**Timestamp**: 2026-09-23T06:44:00Z  
**Assigned Subagent**: `reviewer_m4_recheck` (`teamwork_preview_reviewer`)  
**Working Directory**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/reviewer_m4_recheck`  
**Target Monorepo**: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`  
**Orchestrator Conversation ID**: `9c492746-e261-4f02-867a-381f30f56aae`  

---

## Mandatory Reference Documents
- **Prior Reviewer 1 Handoff**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/reviewer_m4_1/handoff.md`
- **Remediation Worker Handoff**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/worker_m4_remediation/handoff.md`
- **Original User Request**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md`
- **Approved Specification**: `/Users/shakhzod/.gemini/antigravity-cli/brain/07a686c0-bf89-4aac-8ca4-a8586dc28332/prompt_draft.md`

---

## Review Scope & Objectives
Re-review and certify whether the 4 findings from Reviewer 1 (plus idempotency guard) have been cleanly resolved:
1. **Compilation & Fleet Import**: Confirm line 17 of `backend/internal/fleet/repository.go` no longer has unused import `"github.com/pegasus-x/core/internal/doorstep"`. Confirm `internal/fleet`, `internal/api`, `cmd/server`, `cmd/smokecheck` compile and pass tests.
2. **Retailer Quarantine Endpoint**: Confirm `/v1/retailer/quarantine-status` is mounted in `backend/internal/api/router.go` and returns proper status with deprecation headers.
3. **Camera Lockout in `VerifyHandshake`**: Confirm that `backend/internal/doorstep/service.go` strictly rejects `GALLERY`, `GALLERY_UPLOAD`, `DEVICE_STORAGE` with `ErrCameraLockout` and affirmatively requires `CaptureSource == CaptureSourceCameraDirect` on all emergency bypass branches.
4. **Camera Lockout in `ProcessPartialOffload`**: Confirm that `capSource != CaptureSourceCameraDirect` returns `ErrCameraLockout` when `RejectedQty > 0`.
5. **Settlement Idempotency**: Confirm `RecordSettlement` returns early on `status == 'DELIVERED'` without double-incrementing drawer.
6. **Execution & Boundary Verification**:
   - Run tests: `go test -count=1 -v -race ./internal/doorstep/... ./internal/epod/... ./internal/retailer/... ./internal/fleet/... ./internal/api/...`
   - Run: `go build ./cmd/... ./internal/...`
   - Check zero Spanner/Kafka references.
7. **Handoff & Verdict**:
   - Write `/Users/shakhzod/Desktop/V.O.I.D/.agents/reviewer_m4_recheck/handoff.md` with explicit verdict: `APPROVE` or `REQUEST_CHANGES`.
   - Send notification to orchestrator via `send_message`.
