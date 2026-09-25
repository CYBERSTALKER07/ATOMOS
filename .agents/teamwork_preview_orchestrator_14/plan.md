# Master Plan: Pegasus System Reconciliation & Hardening

## Objective
Audit, harden, and reconcile all desktop and web applications across the pegasus, pegasus.x, and pegasusX systems in `/Users/shakhzod/Desktop/V.O.I.D`, ensuring strict architectural boundaries (Spanner/Kafka vs. Postgres/Redis), complete role lifecycle coverage, and production-grade UX/a11y compliance.

## Phased Execution Strategy

### Phase 0: Deep Survey & Technical Mapping (Parallel Explorers)
- **Explorer 1 (UX/A11y Surface)**:
  - Investigate `ux-pilot/audit-report.html` and scan all 16 desktop and web applications.
  - Catalog clickable `<div>` elements requiring `<button>` / keyboard navigation conversion.
  - Catalog all `<input>` elements lacking explicit labels or `aria-label`.
  - Enumerate raw unicode emoji glyphs requiring replacement with `lucide-react` icons.
  - Locate fixed pixel containers (`w-[...px]`) causing horizontal overflows.
- **Explorer 2 (Architectural Boundaries & Data Engines)**:
  - Verify `pegasusX` (Global Multi-Tenant Cloud): Spanner DDL compliance, interleaved tables, tenant partitioning by `SupplierId`, Kafka schemas, double-entry ledger idempotency.
  - Verify `pegasus.x` (Sovereign National Core): Check for forbidden `cloud.google.com/go/spanner` and `kafka-go` dependencies; verify PostgreSQL 16 migrations + Redis 7 Streams/PubSub outbox relay.
- **Explorer 3 (Cross-Role Domain Parity & State Machines)**:
  - Inspect `PEGASUSX_USER_FLOWS.md` and `DUAL_SYSTEM_ARCHITECTURE_AND_PARITY.md`.
  - Analyze the 8 user roles (Supplier, Retailer, Driver, Warehouse, Payload Dock, Factory, Admin, Field Sales) across desktop portals, tablet terminals, and mobile clients.
  - Identify discrepancies and logic gaps in role state machines.

### Phase 1: Synthesis & PROJECT.md Compilation
- Synthesize all findings from Explorers 1, 2, and 3.
- Build comprehensive `PROJECT.md` with:
  - Architecture breakdown
  - Comprehensive Feature Inventory mapped to milestones
  - Strict Interface Contracts
  - Code Layout boundaries

### Phase 2: Milestone Execution & Verification Loops
- **Milestone M1**: Desktop & Web UX Remediation & A11y Hardening across all 16 apps
  - Worker dispatch with strict write boundaries
  - Verification: pnpm typecheck / tsc, automated static lint script, keyboard nav check
  - Reviewer verification & gate check
- **Milestone M2**: Architectural Boundary Enforcement & Engine Verification
  - Worker dispatch for any non-contamination fixes or DDL/outbox adjustments
  - Static grep verification (0 spanner/kafka in pegasus.x/)
  - Reviewer verification & gate check
- **Milestone M3**: Cross-Role Domain Parity & End-to-End Operational Alignment
  - Worker dispatch for state machine alignment across all 8 roles
  - Reviewer verification & gate check
- **Milestone M4**: Full-Stack Verification & UX Audit Report Health Score Check
  - Re-run audit script / test runner
  - Verify health score >= 92/100
  - Type checks across all apps
  - Comprehensive gate check

### Phase 3: Reporting & Victory Audit Handoff
- Produce comprehensive handoff report with verification evidence.
- Send completion message to Sentinel (caller agent) via `send_message`.
