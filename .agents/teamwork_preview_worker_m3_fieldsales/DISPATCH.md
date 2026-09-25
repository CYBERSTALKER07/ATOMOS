## 2026-09-25T13:30:51Z
You are teamwork_preview_worker_m3_fieldsales.
Working directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m3_fieldsales
Workspace root: /Users/shakhzod/Desktop/V.O.I.D
Authoritative Request: Read /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md completely.

Direct Action Plan:
1. In pegasus.x/backend/internal/models/claims.go:
   Add RoleFieldSales = "field_sales" to role constants and update AllRoles/IsValid if present.
2. In pegasus.x/apps/field-sales-mobile/src/screens/ProxyOrderScreen.tsx:
   Find where payload is submitted (around lines 87-91) as { "sku": it.sku, "quantity": it.quantity, "unit_price": ... } and change to:
   { sku_id: it.sku, ordered_qty: it.quantity, list_price_minor: it.unitPriceMinor } matching CreateOrderRequest in internal/order/service.go.
3. In pegasus.x/backend/internal/api/:
   Add POST /v1/cash/payment-legs route and handler (recording cash collection in payment_legs or order ledger).
   Add GET /v1/admin/ops/dead-letters and POST /v1/admin/ops/dead-letters/replay routes and handlers (querying/replaying outbox_dead_letters).
4. In packages/types/ (or wherever canonicalizeOrderStatus is defined):
   Ensure canonicalizeOrderStatus(status: string) maps both 18-state (pegasusX) and 12-state (pegasus.x) orders cleanly.
5. Verification:
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend && go vet ./... && go test -v -count=1 ./internal/...
   cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x && pnpm check-types

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A teamwork_preview_auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

Write handoff.md and send completion message back to parent.
