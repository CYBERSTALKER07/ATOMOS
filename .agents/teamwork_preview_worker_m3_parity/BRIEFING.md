# BRIEFING — 2026-09-25T17:58:00+05:00

## Mission
Execute Milestone M3 to achieve complete cross-role domain parity and operational alignment across all 8 user roles (Supplier, Retailer, Driver, Warehouse, Payload Dock, Factory, Admin, Field Sales).

## 🔒 My Identity
- Archetype: teamwork_worker
- Roles: implementer, qa, specialist
- Working directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m3_parity
- Original parent: e869a8f0-cea5-425d-aa86-c58cee2e3e18
- Milestone: M3 (Cross-Role Domain Parity & Operational Alignment)

## 🔒 Key Constraints
- Exclusive write ownership:
  - pegasus.x/backend/internal/models/claims.go
  - pegasus.x/backend/internal/api/ (new endpoints for cash payment legs & outbox DLQ inspection/replay)
  - pegasus.x/apps/field-sales-mobile/src/screens/ProxyOrderScreen.tsx
  - Any related state machine / types harmonization files in packages/types/
- Do NOT modify files in pegasus/apps/!
- Real, genuine implementations only. No hardcoded test outputs or dummy facades.

## Current Parent
- Conversation ID: e869a8f0-cea5-425d-aa86-c58cee2e3e18
- Updated: 2026-09-25T17:58:00+05:00

## Task Summary
- **What to build**:
  1. Add RoleFieldSales = "field_sales" to valid roles in claims.go.
  2. Align ProxyOrderScreen.tsx order submission payload { sku_id, ordered_qty, list_price_minor }.
  3. Add endpoint handler POST /v1/cash/payment-legs for cash collections with integer tiyin validation.
  4. Add GET /v1/admin/ops/dead-letters and POST /v1/admin/ops/dead-letters/replay.
  5. Harmonize canonicalizeOrderStatus() in @pegasusx/types across 18-state and 12-state flows.
  6. Verify with go vet, go test, and pnpm check-types.
- **Success criteria**:
  - All Go backend tests and type checks pass.
  - All TypeScript type checks pass in pegasus.x.
- **Interface contracts**: /Users/shakhzod/Desktop/V.O.I.D/DUAL_SYSTEM_ARCHITECTURE_AND_PARITY.md, PEGASUSX_USER_FLOWS.md, PROJECT.md
- **Code layout**: /Users/shakhzod/Desktop/V.O.I.D/PROJECT.md

## Change Tracker
- **Files modified**: None yet
- **Build status**: Not tested yet
- **Pending issues**: None

## Quality Status
- **Build/test result**: Untested
- **Lint status**: Clean
- **Tests added/modified**: Pending

## Loaded Skills
- None specified in dispatch

## Key Decisions Made
- Starting baseline investigation of all referenced files and docs.

## Artifact Index
- DISPATCH.md — Assignment and instructions
- progress.md — Liveness heartbeat and step tracking
- handoff.md — Final 5-component handoff report
