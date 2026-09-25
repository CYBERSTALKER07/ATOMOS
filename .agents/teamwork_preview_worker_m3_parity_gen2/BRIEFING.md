# BRIEFING — 2026-09-25T13:20:00Z

## Mission
Complete Milestone M3: Cross-Role Domain Parity Reconciliation.

## 🔒 My Identity
- Archetype: preview_worker
- Roles: implementer, qa, specialist
- Working directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m3_parity_gen2
- Original parent: e869a8f0-cea5-425d-aa86-c58cee2e3e18
- Milestone: M3 (Cross-Role Domain Parity Reconciliation)

## 🔒 Key Constraints
- Exclusive Write Ownership:
  - pegasus.x/backend/internal/models/claims.go
  - pegasus.x/backend/internal/api/ (cash payment legs & outbox DLQ endpoints)
  - pegasus.x/apps/field-sales-mobile/src/screens/ProxyOrderScreen.tsx
  - packages/types/
- Do NOT modify any files in pegasus/apps/!
- Integrity Mandate: No hardcoding test results, no facade implementations, maintain real state & behavior.

## Current Parent
- Conversation ID: e869a8f0-cea5-425d-aa86-c58cee2e3e18
- Updated: not yet

## Task Summary
- **What to build**:
  1. Reconcile Field Sales in pegasus.x:
     - Add `RoleFieldSales = "field_sales"` in `pegasus.x/backend/internal/models/claims.go`.
     - In `pegasus.x/apps/field-sales-mobile/src/screens/ProxyOrderScreen.tsx`, ensure order submission uses `{ sku_id: it.sku, ordered_qty: it.quantity, list_price_minor: it.unitPriceMinor }`.
     - In `pegasus.x/backend/internal/api/`, add handler for `POST /v1/cash/payment-legs` allowing recording cash collections.
  2. Implement Outbox Dead-Letter Queue Inspection & Replay:
     - In `pegasus.x/backend/internal/api/`:
       - `GET /v1/admin/ops/dead-letters` (returns dead letters from `outbox_dead_letters`).
       - `POST /v1/admin/ops/dead-letters/replay` (re-queues dead letters back into `outbox_events`).
  3. Order State Machine Parity & UI Harmonization:
     - In `packages/types/`, ensure `canonicalizeOrderStatus()` cleanly maps both the 18-state flow (pegasusX) and 12-state flow (pegasus.x).
- **Success criteria**:
  - `cd pegasus.x/backend && go vet ./... && go test -v -count=1 ./internal/...` passes.
  - `cd pegasus.x && pnpm check-types` passes.
- **Interface contracts**: /Users/shakhzod/Desktop/V.O.I.D/PROJECT.md
- **Code layout**: /Users/shakhzod/Desktop/V.O.I.D/PROJECT.md

## Key Decisions Made
- [TBD]

## Artifact Index
- DISPATCH.md — Assignment instructions
- BRIEFING.md — Situational awareness
- progress.md — Liveness & progress tracker
- handoff.md — Final 5-component handoff report

## Change Tracker
- **Files modified**: None yet
- **Build status**: Untested
- **Pending issues**: None

## Quality Status
- **Build/test result**: Not yet executed
- **Lint status**: Not yet run
- **Tests added/modified**: None yet

## Loaded Skills
- None specified in dispatch prompt.
