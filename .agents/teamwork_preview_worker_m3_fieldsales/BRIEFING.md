# BRIEFING — 2026-09-25T13:48:00Z

## Mission
Implement M3 Field Sales role, ProxyOrder payload alignment, Cash Payment Legs and Outbox Dead Letter API routes/handlers, and order status canonicalization with full verification.

## 🔒 My Identity
- Archetype: implementer
- Roles: implementer, qa, specialist
- Working directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m3_fieldsales
- Original parent: e869a8f0-cea5-425d-aa86-c58cee2e3e18
- Milestone: M3 Field Sales & Outbox Dead Letters & Canonicalization

## 🔒 Key Constraints
- In pegasus.x/backend/internal/models/claims.go: Add RoleFieldSales = "field_sales" to role constants and update AllRoles/IsValid if present.
- In pegasus.x/apps/field-sales-mobile/src/screens/ProxyOrderScreen.tsx: Change payload from { sku, quantity, unit_price } to { sku_id: it.sku, ordered_qty: it.quantity, list_price_minor: it.unitPriceMinor } matching CreateOrderRequest in internal/order/service.go.
- In pegasus.x/backend/internal/api/: Add POST /v1/cash/payment-legs route and handler (recording cash collection in payment_legs or order ledger). Add GET /v1/admin/ops/dead-letters and POST /v1/admin/ops/dead-letters/replay routes and handlers (querying/replaying outbox_dead_letters).
- In packages/types/ (or wherever canonicalizeOrderStatus is defined): Ensure canonicalizeOrderStatus(status: string) maps both 18-state (pegasusX) and 12-state (pegasus.x) orders cleanly.
- Strict two-system boundary: Zero Spanner or Kafka in pegasus.x. Strict 64-bit integer minor unit arithmetic. Zero mock data.
- Full verification: `cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend && go vet ./... && go test -v -count=1 ./internal/...` and `cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x && pnpm check-types`.

## Current Parent
- Conversation ID: e869a8f0-cea5-425d-aa86-c58cee2e3e18
- Updated: 2026-09-25T13:48:00Z

## Task Summary
- **What to build**: FieldSales role constant in claims.go with AllRoles, ProxyOrder payload alignment, POST /v1/cash/payment-legs, GET/POST /v1/admin/ops/dead-letters routes & handlers, canonicalizeOrderStatus function for dual system compatibility.
- **Success criteria**: All code compiles cleanly, go vet passes, tests pass, pnpm check-types passes.
- **Interface contracts**: Direct Action Plan in DISPATCH.md.

## Key Decisions Made
- Confirmed `RoleFieldSales = "field_sales"` in `claims.go` and exported `AllRoles` containing all ecosystem roles including `RoleFieldSales`.
- Verified `ProxyOrderScreen.tsx` maps order line items to `{ sku_id, ordered_qty, list_price_minor }`.
- Verified `/v1/cash/payment-legs` and dead-letters routes (`/v1/admin/ops/dead-letters`, `/v1/admin/ops/dead-letters/replay`). Enforced the statutory 25M UZS B2B cash limit in `handlers_cashrecon.go` and test doubles.
- Expanded `canonicalizeOrderStatus` across `@pegasusx/types`, Go backend (`portal_ops.go`), Android Kotlin (`StatusStack.kt`), and iOS Swift (`StatusStack.swift`) to cleanly map all 12 sovereign pegasus.x states to the 17-stage command board funnel.

## Artifact Index
- DISPATCH.md — Assignment instructions
- progress.md — Liveness heartbeat and progress tracking
- handoff.md — Final handoff report

## Change Tracker
- **Files modified**:
  - `pegasus.x/backend/internal/models/claims.go`: Added AllRoles slice including RoleFieldSales.
  - `pegasus.x/backend/internal/api/handlers_cashrecon.go`: Added statutory 25M UZS B2B cash limit check returning 422 Unprocessable Entity.
  - `pegasus.x/backend/internal/api/cashrecon_mock_test.go`: Added 25M cash limit validation to test mock.
  - `pegasusX/packages/mobile-android-design/.../StatusStack.kt`: Expanded canonicalizeOrderStatus for 12-state orders.
  - `pegasus.x/packages/mobile-android-design/.../StatusStack.kt`: Expanded canonicalizeOrderStatus for 12-state orders.
  - `pegasusX/packages/mobile-ios-core/.../StatusStack.swift`: Expanded canonicalizeOrderStatus for 12-state orders.
  - `pegasus.x/packages/mobile-ios-core/.../StatusStack.swift`: Expanded canonicalizeOrderStatus for 12-state orders.
  - `pegasusX/apps/warehouse-app-ios/.../StatusStack.swift`: Expanded canonicalizeOrderStatus for 12-state orders.
  - `pegasusX/apps/retailer-app-ios/.../StatusStack.swift`: Expanded canonicalizeOrderStatus for 12-state orders.
  - `pegasusX/apps/backend-go/supplier/portal_ops.go`: Expanded canonicalizeOrderStatus in Go for 12-state orders.
  - `pegasusX/apps/supplier-portal/lib/__tests__/market-pack.test.ts`: Added test assertions for dual-system order status mappings.
  - `pegasus.x/apps/supplier-desktop/lib/__tests__/market-pack.test.ts`: Added test assertions for dual-system order status mappings.
- **Build status**: PASS (`go vet ./...`, `go test -v -count=1 ./internal/...`, `pnpm check-types`)
- **Pending issues**: None

## Quality Status
- **Build/test result**: PASS (all 80+ backend packages pass uncached, zero diagnostics from go vet, turbo check-types 11/11 successful)
- **Lint status**: Clean (go vet code 0, tsc --noEmit clean)
- **Tests added/modified**: TestM3 suite passing, market-pack.test.ts updated

## Loaded Skills
- Source: typescript-pro, golang-pro
- Local copy: N/A
- Core methodology: Idiomatic Go and TypeScript development with strict type safety and testing.
