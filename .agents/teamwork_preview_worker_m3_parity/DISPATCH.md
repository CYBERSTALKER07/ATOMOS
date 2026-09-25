## 2026-09-25T12:57:56Z
You are teamwork_preview_worker_m3_parity.
Working directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m3_parity
Workspace root: /Users/shakhzod/Desktop/V.O.I.D
Authoritative Request: Read /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md completely.
Survey Analysis: Read /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_14_3/survey_domain_parity.md completely.
Parity Spec: /Users/shakhzod/Desktop/V.O.I.D/DUAL_SYSTEM_ARCHITECTURE_AND_PARITY.md and /Users/shakhzod/Desktop/V.O.I.D/PEGASUSX_USER_FLOWS.md
Project Definition: /Users/shakhzod/Desktop/V.O.I.D/PROJECT.md

Exclusive Write Ownership:
- pegasus.x/backend/internal/models/claims.go
- pegasus.x/backend/internal/api/ (new endpoints for cash payment legs & outbox DLQ inspection/replay)
- pegasus.x/apps/field-sales-mobile/src/screens/ProxyOrderScreen.tsx
- Any related state machine / types harmonization files in packages/types/
Do NOT modify files in pegasus/apps/!

Objective:
Execute Milestone M3 to achieve complete cross-role domain parity and operational alignment across all 8 user roles (Supplier, Retailer, Driver, Warehouse, Payload Dock, Factory, Admin, Field Sales):
1. Reconcile Field Sales in pegasus.x:
   - In pegasus.x/backend/internal/models/claims.go, add RoleFieldSales = "field_sales" to valid roles.
   - In pegasus.x/apps/field-sales-mobile/src/screens/ProxyOrderScreen.tsx, align order submission payload to use { sku_id: it.sku, ordered_qty: it.quantity, list_price_minor: it.unitPriceMinor } matching backend CreateOrderRequest.
   - In pegasus.x/backend/internal/api/, add endpoint handler for POST /v1/cash/payment-legs allowing field sales/drivers to record cash collections, validating integer tiyin minor amounts.
2. Implement Outbox Dead-Letter Queue Inspection & Replay in pegasus.x:
   - In pegasus.x/backend/internal/api/ (e.g. core.go or dedicated admin/ops routes), add:
     - GET /v1/admin/ops/dead-letters (returns rows from outbox_dead_letters).
     - POST /v1/admin/ops/dead-letters/replay (re-queues dead letters back into outbox_events or redis stream).
3. Order State Machine Parity & UI Harmonization:
   - Ensure @pegasusx/types canonicalizeOrderStatus() cleanly maps both the 18-state flow (pegasusX) and 12-state flow (pegasus.x) so that all portals and terminals render valid badges for any order status.
4. Run tests:
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend && go vet ./... && go test -v -count=1 ./internal/...
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x && pnpm check-types

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A teamwork_preview_auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

Write handoff.md and send a completion message back to parent.
