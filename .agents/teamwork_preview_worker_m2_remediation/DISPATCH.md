## 2026-09-25T12:21:36Z

<USER_REQUEST>
You are teamwork_preview_worker_m2_remediation.
Working directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m2_remediation
Workspace root: /Users/shakhzod/Desktop/V.O.I.D
Authoritative Request: Read /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md completely.
Reviewer 2 Handoff: Read /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_reviewer_m2_14_2/handoff.md completely.
Project Definition: /Users/shakhzod/Desktop/V.O.I.D/PROJECT.md

Exclusive Write Ownership:
- Backend code in pegasus.x/backend/ and pegasusX/apps/backend-go/
- Infrastructure tfvars in pegasus.x/infra/terraform/
Do NOT modify any frontend files!

Objective:
Remediate all issues identified by Reviewer M2.2:
1. Eliminate floating-point financial arithmetic in pegasus.x:
   - pegasus.x/backend/internal/fleet/fx_index.go (lines 58-69): Refactor CalculateFXAdjustment to use pure 64-bit integer arithmetic (e.g. basis points fxRateBasisPoints or integer ratio multiplication before division) so that monetary adjustments remain int64 without floats.
   - pegasus.x/backend/internal/matching/matching.go (lines 50, 93, 216-220): Replace float conversions in ERS matching with integer arithmetic.
   - pegasus.x/backend/internal/warehouse/service.go (line 604): Replace int64(math.Round(shortage * float64(exp.UnitCostMinor))) with pure integer multiplication: (int64(shortage) * exp.UnitCostMinor).
   - pegasus.x/backend/internal/dispatch/shuttle.go (lines 73-88) & fleet/fuel_theft.go (lines 100, 129, 150): Replace float arithmetic on minor cost units with integer math.
   - pegasus.x/backend/cmd/smokecheck/main.go (lines 343, 406): Refactor float financial checks to integer minor units.
2. Deterministic Idempotency Keys in pegasusX:
   - pegasusX/apps/backend-go/payment/double_entry.go (lines 135, 213, 278): Replace non-deterministic time.Now().UTC().UnixNano() in EntryID and ReferenceID with deterministic identifiers (derived from payment session ID, leg ID, or order ID) so that re-executing transactions preserves true idempotency under the Spanner unique index Idx_PaymentLedgerEntries_GatewayTypeRef.
3. Align Terraform Configuration in pegasus.x:
   - In pegasus.x/infra/terraform/environments/production.tfvars (line 25) and cells/uz/cell.tfvars (line 30), change enable_managed_kafka = true to enable_managed_kafka = false to guarantee sovereign non-contamination.
4. Verify all tests pass:
   - cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend && go vet ./... && go test -v -count=1 ./internal/...
   - cd /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go && go test -v -count=1 ./outbox/... ./ar/... ./payment/...

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A teamwork_preview_auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

Document all changes and test outputs in handoff.md and send a completion message back to parent.
</USER_REQUEST>
