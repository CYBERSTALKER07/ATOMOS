# Independent Victory Audit Plan: victory_auditor_orch_4

**Objective**: Perform a blocking, independent victory audit against all claims made by `teamwork_preview_orchestrator_14` regarding Pegasus System Hardening & Reconciliation across `pegasus`, `pegasus.x`, and `pegasusX`.

## 1. Audit Tracks & Decomposition

### Track 1: R1 — Desktop & Web UX Remediation & Accessibility Hardening
- **Assigned Subagent**: `auditor_r1_ux` (`teamwork_preview_reviewer`)
- **Working Directory**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/victory_auditor_r1_ux`
- **Scope**:
  1. Inspect `/Users/shakhzod/Desktop/V.O.I.D/ux-pilot/audit-report.html`: verify health score >= 92/100 (claimed 95/100), verify 0 critical findings.
  2. Inspect form input labeling across all 16 desktop/web apps for `<label htmlFor="...">` and `aria-label`.
  3. Inspect keyboard navigation (`tabIndex`, `onKeyDown` with Enter/Space, or semantic `<button>`) on custom interactive controls and modals.
  4. Inspect UI control bars and navigation bars for raw unicode emoji glyphs vs standard SVG Lucide icons (`lucide-react`).
  5. Check responsive containers (`max-w-7xl` vs fixed `w-[...px]`).
- **Deliverable**: `handoff.md` with explicit evidence chains and PASS/FAIL verdict.

### Track 2: R2 — Architectural Boundary & Non-Contamination
- **Assigned Subagent**: `auditor_r2_arch` (`teamwork_preview_reviewer`)
- **Working Directory**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/victory_auditor_r2_arch`
- **Scope**:
  1. Run static grep across `pegasus.x/` for `cloud.google.com/go/spanner` and Kafka drivers (`kafka-go`, `sarama`, `confluent`) — verify 0 references.
  2. Run static grep on `pegasusX/apps/backend-go/schema/spanner.ddl`: verify exactly 19 interleaved child tables with `ON DELETE CASCADE` and tenant key partitioning rooted on `SupplierId STRING(36) NOT NULL`.
  3. Inspect `pegasusX/apps/backend-go/ar/double_entry.go` and payment ledgers: verify deterministic idempotency keys and pure integer minor unit currency arithmetic (`int64` tiyins), with 0 floating-point financial math.
  4. Verify Terraform configs (`enable_managed_kafka = false`).
- **Deliverable**: `handoff.md` with explicit grep outputs, line references, and PASS/FAIL verdict.

### Track 3: R3 — Cross-Role Domain Parity & Operational Alignment
- **Assigned Subagent**: `auditor_r3_parity` (`teamwork_preview_reviewer`)
- **Working Directory**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/victory_auditor_r3_parity`
- **Scope**:
  1. Audit business state machines across all 8 user roles (Supplier, Retailer, Driver, Warehouse, Payload Dock, Factory, Admin, Field Sales) across desktop portals, tablet terminals, and mobile apps per `PEGASUSX_USER_FLOWS.md` and `DUAL_SYSTEM_ARCHITECTURE_AND_PARITY.md`.
  2. Verify Field Sales role registration (`RoleFieldSales = "field_sales"`, `AgentID` in `UserClaims`, validation).
  3. Verify Proxy Ordering contract in `ProxyOrderScreen.tsx` matching `CreateOrderRequest`.
  4. Verify Central Bank statutory 25M UZS cash limit (422 Unprocessable Entity, `b2b_cash_limit_exceeded`) in `POST /v1/cash/payment-legs`.
  5. Verify Outbox DLQ endpoints (`GET /v1/admin/ops/dead-letters` and `POST /v1/admin/ops/dead-letters/replay`) with row-locking and stream re-injection.
  6. Verify canonical order status transformations across TypeScript (`@pegasusx/types`), Go (`portal_ops.go`), Android Kotlin (`StatusStack.kt`), and iOS Swift (`StatusStack.swift`).
- **Deliverable**: `handoff.md` with explicit parity matrix and PASS/FAIL verdict.

### Track 4: R4 — Programmatic Verification & Test Execution
- **Assigned Subagent**: `auditor_r4_prog` (`teamwork_preview_worker`)
- **Working Directory**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/victory_auditor_r4_prog`
- **Scope**:
  1. Execute frontend type checks (`pnpm check-types --force` or `tsc --noEmit`) across `pegasus.x` and `pegasusX` frontend apps.
  2. Execute automated static linting/audit scanner script verifying zero unlabeled inputs and zero un-roled clickable divs.
  3. Execute Go test suites across `pegasus.x/backend` (`go vet ./...` and `go test -v -count=1 ./internal/...`).
  4. Execute Go test suites across `pegasusX/apps/backend-go` (`go test -v -count=1 ./outbox/... ./ar/... ./payment/...`).
- **Deliverable**: `handoff.md` with exact command outputs, exit codes, test counts, and PASS/FAIL verdict.

## 2. Synthesis & Final Binary Verdict
- Collect handoffs from all 4 tracks.
- Evaluate against binary veto: any single test failure, unverified claim, or integrity violation results in VICTORY REJECTED.
- Synthesize findings into `/Users/shakhzod/Desktop/V.O.I.D/.agents/victory_auditor_orch_4/handoff.md`.
- Communicate definitive verdict back to parent via `send_message`.
