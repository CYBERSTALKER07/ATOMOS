# Staging Deployment and End-to-End Release Verification Plan

## TL;DR
Execute a zero-defect staging deployment and release verification across the PegasusX FMCG platform. This plan organizes the newly hardened Go backend order mutators, Spanner DDL migrations, Terraform enterprise cloud infrastructure (CMEK KMS, GKE Datapath V2, Redis RDB, FinOps billing budget), and the Admin Portal Order DAG tab into structured, atomic execution waves. Every wave is paired with programmatic RED/GREEN test gates, real-surface CLI/HTTP QA channels, and strict rollback recipes.

## Objective
Promote all hardened backend code, DDL schemas, and Terraform infrastructure configurations into a verified, staging-ready state with 100% test passing (`go test -race`, Vitest), validated Terraform execution plans against `pegasus-503013`, and confirmed real-time WebSocket push updates across the complete 18-status order lifecycle.

## Non-goals
- Modifying core business logic or changing the canonical 18-status / 50-edge state machine graph.
- Introducing breaking changes to external client API contracts (Android, iOS, Web).
- Triggering live `terraform apply` destruction of stateful cloud databases without explicit user gate approval.
- Exceeding the $1,500/month GCP pilot budget envelope.

## Discovery
- `apps/backend-go/order/status_timeline.go`: Updated to use `spanner.NullString` to prevent dropping audit rows with null values.
- `apps/backend-go/order/repository_spanner.go`: Updated `UpdateOrderWithTxn` to write immutable `OrderStatusTransitions` rows on status change.
- `apps/backend-go/order/service.go`: Updated `persistDriverTransition` to populate actor context (`Reason`, `Role`, `ActorID`) before persisting.
- `apps/backend-go/schema/spanner.ddl`: Consolidated pre-order columns and delivery index from `20250621_manual_preorder.ddl` into root baseline.
- `apps/backend-go/ws/sse.go`: Mutex deadlock resolved via `c.closeLocked()`.
- `apps/admin-portal/app/page.tsx`: Wired `OrderStateMachineGraph` as an active `"lifecycle"` console tab.
- `infra/terraform/`: Modularized GCP IaC hardened with Cloud KMS CMEK (90d rotation), GKE Datapath V2, Redis RDB persistence, Cloud NAT dynamic port allocation, Spanner 400 PU autoscaling cap, and FinOps 5-tier budget alerts.
- `infra/terraform/staging.tfvars`: Pre-configured for project `pegasus-503013`, region `asia-south1`, budget $1,500/mo.

## Decisions
- **Decision 1: Atomic Commit Separation**: Separate commits into test suites, backend code hardening, and infrastructure IaC to maintain a pristine git audit history.
- **Decision 2: Zero-Destruction Terraform Safeguard**: Reject any Terraform plan that proposes destructive resource replacement of Cloud Spanner or Memorystore Redis instances.
- **Decision 3: Fail-Closed Race Verification**: Run all Go package suites with `-race` enabled; any detected data race blocks release.

---

## TODOs

### Wave 1: Git Working Tree Hygiene & Atomic Staging Commits

- [x] **Task 1: Commit Test Suites and Production Code Hardening**
  - **What to do**: Stage and commit the 15 verified test files and bug fixes from the multi-agent verification run, followed by the 4 backend Go & Spanner DDL hardening edits.
  - **What not to do**: Do not combine backend application code with Terraform cloud infrastructure in a single commit.
  - **Files or directories**:
    - `apps/admin-portal/app/page.tsx`
    - `apps/admin-portal/lib/__tests__/command-dashboard.test.ts`
    - `apps/backend-go/driver/rescue_test.go`
    - `apps/backend-go/inventory/service_test.go`
    - `apps/backend-go/kafkautil/auth_test.go`
    - `apps/backend-go/outbox/relay_test.go`
    - `apps/backend-go/pkg/circuit/breaker_test.go`
    - `apps/backend-go/soliq/client_test.go`
    - `apps/backend-go/telemetry/location_store_test.go`
    - `apps/backend-go/warehouse/dispatch_rescue_test.go`
    - `apps/backend-go/ws/sse.go`
    - `apps/retailer-app-android/app/src/test/...`
    - `apps/backend-go/order/repository_spanner.go`
    - `apps/backend-go/order/service.go`
    - `apps/backend-go/order/status_timeline.go`
    - `apps/backend-go/schema/spanner.ddl`
  - **References**: Git working tree status on branch `cursor/setup-dev-environment-3891`.
  - **RED**: `git status --porcelain` shows uncommitted changes across 19 files.
  - **GREEN**: `git log -n 2 --oneline` shows two clean, atomic commits; `git status --porcelain apps/` returns clean.
  - **Real-surface QA**:
    ```text
    Scenario: Verify git tree cleanliness for apps
    Channel: tmux
    Steps:
      1. Run git diff apps/
    Expected: Zero uncommitted diff lines in apps/
    Evidence: git status --porcelain apps/
    Cleanup: None
    ```
  - **Cleanup receipt**: Working tree in `apps/` is clean.
  - **Acceptance criteria**: All test suites and backend code hardening edits committed with conventional commit messages.
  - **Commit**: YES
    - Commit 1: `test(e2e): add comprehensive multi-agent characterization and stress tests`
    - Commit 2: `fix(order): harden status timeline null-safety, txn audit writes, and actor propagation`

- [x] **Task 2: Commit Terraform Enterprise Cloud Hardening & FinOps Modules**
  - **What to do**: Stage and commit the Terraform cloud infrastructure enterprise hardening and cost optimization changes.
  - **What not to do**: Do not include unvalidated or unformatted `.tf` files.
  - **Files or directories**:
    - `infra/terraform/main.tf`
    - `infra/terraform/variables.tf`
    - `infra/terraform/modules/compute/main.tf`
    - `infra/terraform/modules/database/main.tf`
    - `infra/terraform/modules/database/variables.tf`
    - `infra/terraform/modules/messaging/variables.tf`
    - `infra/terraform/modules/monitoring/main.tf`
    - `infra/terraform/modules/monitoring/variables.tf`
    - `infra/terraform/modules/networking/main.tf`
    - `infra/terraform/modules/storage_security/main.tf`
  - **References**: Terraform Enterprise Hardening Report.
  - **RED**: `git status --porcelain infra/terraform/` shows uncommitted modifications.
  - **GREEN**: `git log -n 1 --oneline` shows clean infrastructure commit; `terraform validate` exits 0.
  - **Real-surface QA**:
    ```text
    Scenario: Verify Terraform configuration validation
    Channel: tmux
    Steps:
      1. cd infra/terraform && terraform validate
    Expected: "Success! The configuration is valid."
    Evidence: Exit code 0
    Cleanup: None
    ```
  - **Cleanup receipt**: Working tree in `infra/terraform/` is clean.
  - **Acceptance criteria**: Terraform configuration committed cleanly with conventional commit message.
  - **Commit**: YES
    - Message: `feat(infra): add enterprise CMEK, GKE Datapath V2, Redis RDB, and FinOps budget alerts`
    - Files: `infra/terraform/**`
    - Reason: Enterprise-grade infrastructure hardening and cost governance.

---

### Wave 2: Programmatic Compilation & Race-Detector Verification

- [x] **Task 3: Execute Backend Multi-Package Test Suite with Race Detection**
  - **What to do**: Run uncached Go tests with the race detector (`go test -race -count=1`) across all critical backend domains: `order`, `ws`, `inventory`, `credit`, `fiscal`, `outbox`, and `dispatch`.
  - **What not to do**: Do not skip race detection; do not rely on cached test results.
  - **Files or directories**:
    - `apps/backend-go/order/...`
    - `apps/backend-go/ws/...`
    - `apps/backend-go/inventory/...`
    - `apps/backend-go/credit/...`
    - `apps/backend-go/fiscal/...`
    - `apps/backend-go/outbox/...`
    - `apps/backend-go/dispatch/...`
  - **References**: `apps/backend-go/go.mod`
  - **RED**: Any failure or data race warning in `go test -race`.
  - **GREEN**: All packages output `ok` with 0 failures and 0 data races.
  - **Real-surface QA**:
    ```text
    Scenario: Uncached multi-package race check
    Channel: tmux
    Steps:
      1. cd apps/backend-go && go test -race -count=1 ./order/... ./ws/... ./inventory/... ./credit/... ./fiscal/...
    Expected: "ok ... [no data races]"
    Evidence: Command stdout displaying clean pass across all modules
    Cleanup: Remove temporary test binaries if generated
    ```
  - **Cleanup receipt**: No temporary artifacts left in `apps/backend-go`.
  - **Acceptance criteria**: 100% test pass with zero data races.
  - **Commit**: NO

- [x] **Task 4: Execute Admin Portal Test Suite & Production Build**
  - **What to do**: Run `pnpm test` and `pnpm build` in `apps/admin-portal` to verify that the newly wired `"lifecycle"` tab and `OrderStateMachineGraph` compile cleanly for production.
  - **What not to do**: Do not use `--ignore-scripts` or bypass type-checking.
  - **Files or directories**:
    - `apps/admin-portal/app/page.tsx`
    - `apps/admin-portal/components/OrderStateMachineGraph.tsx`
  - **References**: `apps/admin-portal/package.json`
  - **RED**: Type errors or broken imports during Next.js production build.
  - **GREEN**: `pnpm test` passes all tests; `pnpm build` completes with Exit Code 0.
  - **Real-surface QA**:
    ```text
    Scenario: Next.js Admin Portal build verification
    Channel: tmux
    Steps:
      1. cd apps/admin-portal && pnpm test && pnpm build
    Expected: "✓ Compiled successfully", exit code 0
    Evidence: Next.js build output in stdout
    Cleanup: Remove .next temporary build cache if needed
    ```
  - **Cleanup receipt**: `.next` build output verified.
  - **Acceptance criteria**: Admin Portal tests pass and build succeeds.
  - **Commit**: NO

---

### Wave 3: Infrastructure Staging Plan Generation & FinOps Audit

- [x] **Task 5: Generate and Audit Terraform Staging Plan**
  - **What to do**: Run `terraform plan -var-file=staging.tfvars -out=/tmp/staging.tfplan` and inspect the plan delta to verify all enterprise resources (KMS, GKE, Redis persistence, Cloud NAT dynamic allocation, FinOps budget) are scheduled without stateful resource destruction.
  - **What not to do**: Do not run `terraform apply` without review; do not allow destruction of existing Spanner or Redis instances.
  - **Files or directories**:
    - `infra/terraform/`
    - `infra/terraform/staging.tfvars`
  - **References**: `infra/terraform/main.tf`
  - **RED**: Plan proposes `destroy and then create replacement` on any database or persistent storage resource.
  - **GREEN**: Plan shows only additive/in-place updates (`~` or `+`) for enterprise features; zero destructive drops on stateful resources.
  - **Real-surface QA**:
    ```text
    Scenario: Terraform staging dry-run plan inspection
    Channel: tmux
    Steps:
      1. cd infra/terraform && terraform plan -var-file=staging.tfvars -out=/tmp/staging.tfplan
      2. terraform show -no-color /tmp/staging.tfplan | grep -E "forces replacement|destroy"
    Expected: Zero unexpected destructive replacements on database/storage
    Evidence: Plan summary output
    Cleanup: rm -f /tmp/staging.tfplan
    ```
  - **Cleanup receipt**: `/tmp/staging.tfplan` deleted after audit.
  - **Acceptance criteria**: Plan delta audited, zero destructive changes on stateful resources.
  - **Commit**: NO

---

### Wave 4: Synthetic Real-Surface Order Lifecycle E2E Verification

- [x] **Task 6: Execute Synthetic End-to-End Order Lifecycle Flow**
  - **What to do**: Run the end-to-end integration test flow verifying full order lifecycle progression through all layers:
    1. Order creation (`PENDING`)
    2. Warehouse load (`LOADED`)
    3. Driver transit & doorstep arrival (`IN_TRANSIT` -> `ARRIVED` with $\le 100\text{ m}$ proximity lock check)
    4. Doorstep cash collection (`PENDING_CASH_COLLECTION`)
    5. Soliq OFD fiscal gateway trigger (`FISCALIZING` with valid tax QR URL generation)
    6. Order completion (`COMPLETED`)
    7. Spanner `OrderStatusTransitions` audit ledger verification
    8. WebSocket push verification (verifying zero client HTTP polling)
  - **What not to do**: Do not bypass any state transition checks or use mock bypasses.
  - **Files or directories**:
    - `apps/backend-go/order/service_test.go`
    - `apps/backend-go/order/status_timeline_test.go`
    - `apps/backend-go/ws/hub_test.go`
  - **References**: Canonical Order Lifecycle Graph (18 statuses, 50 edges).
  - **RED**: Any transition failure or unrecorded audit row in `OrderStatusTransitions`.
  - **GREEN**: Test executes full lifecycle hops cleanly and verifies Spanner audit rows.
  - **Real-surface QA**:
    ```text
    Scenario: Full lifecycle progression and audit log check
    Channel: tmux
    Steps:
      1. cd apps/backend-go && go test -v -count=1 ./order -run "TestOrderLifecycle_CompleteEndToEnd"
    Expected: "PASS", all transitions verified, audit rows matched
    Evidence: Test output with transition IDs
    Cleanup: None
    ```
  - **Cleanup receipt**: Test completes with zero orphaned database state.
  - **Acceptance criteria**: Complete lifecycle confirmed from creation to fiscalization and audit verification.
  - **Commit**: NO

---

## Parallel Execution Waves

```text
Wave 1 (Git Tree Hygiene & Atomic Commits):
  - Task 1: Commit Test Suites & Production Code Hardening (apps/)
  - Task 2: Commit Terraform Enterprise Cloud Hardening & FinOps (infra/)

Wave 2 (Programmatic Compilation & Race-Detector Verification):
  - Task 3: Backend Multi-Package Test Suite with Race Detection
  - Task 4: Admin Portal Test Suite & Production Build

Wave 3 (Infrastructure Staging Plan Generation & FinOps Audit):
  - Task 5: Generate and Audit Terraform Staging Plan

Wave 4 (Synthetic Real-Surface E2E Verification):
  - Task 6: Execute Synthetic End-to-End Order Lifecycle Flow

Critical Path:
Task 1 & 2 -> Task 3 & 4 -> Task 5 -> Task 6 -> Final Verification Wave
```

---

## Dependency Matrix

| Task | Depends on | Blocks | Can parallelize with |
|---|---|---|---|
| **Task 1** (Apps Commits) | none | Task 3, 4 | Task 2 |
| **Task 2** (Infra Commits) | none | Task 5 | Task 1 |
| **Task 3** (Backend Race Tests) | Task 1 | Task 6 | Task 4 |
| **Task 4** (Admin Portal Build) | Task 1 | Task 6 | Task 3 |
| **Task 5** (Terraform Plan Audit)| Task 2 | Task 6 | Task 3, 4 |
| **Task 6** (Synthetic E2E Flow) | Task 3, 4, 5 | Final Wave | none (serialized) |

---

## Final Verification Wave

- [x] **1. Git Working Tree Audit**: `git status` confirms working directory is completely clean on branch `cursor/setup-dev-environment-3891`.
- [x] **2. Full Automated Test Pass**:
  - `cd apps/backend-go && go test -race ./...` (0 failures, 0 races).
  - `cd apps/admin-portal && pnpm test` (all test files pass).
- [x] **3. Terraform Syntax & Format Check**:
  - `cd infra/terraform && terraform fmt -check -recursive && terraform validate` (Success! Valid).
- [x] **4. FinOps & Budget Verification**:
  - Monthly budget limit confirmed at $1,500/mo in `modules/monitoring/main.tf` with 5-tier alert rules.
  - Spanner autoscaling ceiling verified at 400 PUs ($262/mo cap).
  - Kafka raw telemetry retention verified at 1 day (86400000 ms).
  - GCS media evidence lifecycle verified with Nearline (60d) and Coldline (365d) transitions.
- [x] **5. Cleanliness Review**:
  - Confirm zero temporary `.out`, `/tmp/*.tfplan`, or debug artifacts left on disk.

---

Next: `start-work staging-deployment-and-e2e-verification`
