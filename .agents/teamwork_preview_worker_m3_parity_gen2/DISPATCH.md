## 2026-09-25T13:19:37Z

You are teamwork_preview_worker_m3_parity_gen2.
Working directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m3_parity_gen2
Workspace root: /Users/shakhzod/Desktop/V.O.I.D
Authoritative Request: Read /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md completely.
Survey Analysis: Read /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_14_3/survey_domain_parity.md completely.
Project Definition: /Users/shakhzod/Desktop/V.O.I.D/PROJECT.md

Exclusive Write Ownership:
- pegasus.x/backend/internal/models/claims.go
- pegasus.x/backend/internal/api/ (cash payment legs & outbox DLQ endpoints)
- pegasus.x/apps/field-sales-mobile/src/screens/ProxyOrderScreen.tsx
- packages/types/
Do NOT modify any files in pegasus/apps/!

Objective:
Complete Milestone M3: Cross-Role Domain Parity Reconciliation:
1. Reconcile Field Sales in pegasus.x:
   - In pegasus.x/backend/internal/models/claims.go, add RoleFieldSales = "field_sales".
   - In pegasus.x/apps/field-sales-mobile/src/screens/ProxyOrderScreen.tsx, ensure order submission uses { sku_id: it.sku, ordered_qty: it.quantity, list_price_minor: it.unitPriceMinor } matching backend CreateOrderRequest.
   - In pegasus.x/backend/internal/api/, add handler for POST /v1/cash/payment-legs allowing recording cash collections.
2. Implement Outbox Dead-Letter Queue Inspection & Replay:
   - In pegasus.x/backend/internal/api/, add:
     - GET /v1/admin/ops/dead-letters (returns dead letters from outbox_dead_letters).
     - POST /v1/admin/ops/dead-letters/replay (re-queues dead letters back into outbox_events).
3. Order State Machine Parity & UI Harmonization:
   - In packages/types/, ensure canonicalizeOrderStatus() cleanly maps both the 18-state flow (pegasusX) and 12-state flow (pegasus.x).
4. Run tests:
   cd pegasus.x/backend && go vet ./... && go test -v -count=1 ./internal/...
   cd pegasus.x && pnpm check-types

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A teamwork_preview_auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

Write handoff.md and send a completion message back to parent.
