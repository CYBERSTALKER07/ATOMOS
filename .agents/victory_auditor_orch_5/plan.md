# Independent Victory Audit Plan: victory_auditor_orch_5

## 1. Audit Scope & Objectives
Conduct a blocking independent victory audit of all claims in `teamwork_preview_orchestrator_14/handoff.md` against `ORIGINAL_REQUEST.md`, resolving the previous rejection in `victory_auditor_orch_4/handoff.md`.

## 2. Track Decomposition

### Track 1: UX Remediation, Accessibility Hardening & Multi-Monorepo Typecheck (R1 & Programmatic)
- **Subagent**: `auditor_r1_frontend` (teamwork_preview_reviewer / teamwork_preview_worker)
- **Mandate**:
  1. Execute `/Users/shakhzod/Desktop/V.O.I.D/scripts/verify_all_16_apps_typecheck.sh` and log output and exit code for each of the 16 applications.
  2. Independently execute `tsc --noEmit` on each application:
     - `pegasusX`: `supplier-portal`, `warehouse-portal`, `factory-portal`, `admin-portal`, `retailer-app-desktop`, `payload-terminal`
     - `pegasus`: `admin-portal`, `warehouse-portal`, `factory-portal`, `retailer-app-desktop`, `payload-terminal`
     - `pegasus.x`: `supplier-desktop`, `warehouse-desktop`, `retailer-desktop`, `payloader-tablet`, `telegram-miniapp`
  3. Audit git diff and grep for `@ts-ignore`, `@ts-nocheck`, or altered `tsconfig.json` compiler flags.
  4. Verify UX audit report `ux-pilot/audit-report.html` for health score >= 92/100 and zero findings claim.
  5. Run automated static linting/scanning script confirming 0 unlabeled inputs, 0 un-roled clickable divs, keyboard navigation (Tab/Enter/Space), and 0 raw emojis in UI control bars.

### Track 2: Architectural Boundary & Non-Contamination (R2)
- **Subagent**: `auditor_r2_arch` (teamwork_preview_reviewer / teamwork_preview_worker)
- **Mandate**:
  1. Verify zero references to `cloud.google.com/go/spanner` in `pegasus.x/`.
  2. Verify zero references to Kafka packages (`kafka-go`, `sarama`, `confluent-kafka-go`) in `pegasus.x/`.
  3. Verify `pegasusX/` strictly maintains Spanner multi-tenant partitioning in `schema/spanner.ddl`: exactly 19 interleaved child tables with `ON DELETE CASCADE` and tenant key partitioning by `SupplierId`.
  4. Verify double-entry ledger idempotency keys and pure 64-bit integer tiyin currency arithmetic (zero float arithmetic for money).

### Track 3: Cross-Role Domain Parity & Go Test Execution (R3 & Programmatic)
- **Subagent**: `auditor_r3_parity_tests` (teamwork_preview_reviewer / teamwork_preview_worker)
- **Mandate**:
  1. Run Go test suites across `pegasus.x/backend` and `pegasusX/apps/backend-go`.
  2. Verify 8-role operational state machine logic parity per `PEGASUSX_USER_FLOWS.md` and `DUAL_SYSTEM_ARCHITECTURE_AND_PARITY.md`.
  3. Verify Field Sales role (`RoleFieldSales = "field_sales"` and `AgentID` in `claims.go`).
  4. Verify proxy ordering contract alignment (`ProxyOrderScreen.tsx` mapping to `CreateOrderRequest`).
  5. Verify Central Bank statutory 25M UZS cash limit (HTTP 422 `b2b_cash_limit_exceeded`).
  6. Verify Outbox DLQ replay endpoints (`/v1/admin/ops/dead-letters` and `/v1/admin/ops/dead-letters/replay`).
  7. Verify canonical order status funnels and alias mappings across TypeScript, Go, Android Kotlin, and iOS Swift.

## 3. Decision Gate
- Strictly binary: VICTORY CONFIRMED if and only if all 3 tracks pass 100% without breaches or shortcuts.
- If any track fails, issue VICTORY REJECTED with full forensic evidence.
