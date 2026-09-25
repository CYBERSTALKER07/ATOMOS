# Progress — teamwork_preview_worker_m3_fieldsales
Last visited: 2026-09-25T13:48:10Z

## Current Status
- Step 1: Confirmed `RoleFieldSales = "field_sales"` and added `AllRoles` in `pegasus.x/backend/internal/models/claims.go`.
- Step 2: Confirmed `ProxyOrderScreen.tsx` submits payload matching `CreateOrderRequest` with `{ sku_id, ordered_qty, list_price_minor }`.
- Step 3: Verified routes & handlers for `POST /v1/cash/payment-legs`, `GET /v1/admin/ops/dead-letters`, and `POST /v1/admin/ops/dead-letters/replay`. Enforced statutory 25M UZS B2B cash limit returning HTTP 422.
- Step 4: Ensured `canonicalizeOrderStatus` cleanly maps all 18-state (pegasusX) and 12-state (pegasus.x) orders across `@pegasusx/types`, Go backend, Android Kotlin, and iOS Swift.
- Step 5: Verified full test suite and type check:
  - `go vet ./...` in `pegasus.x/backend`: PASS (0 diagnostics)
  - `go test -v -count=1 ./internal/...` in `pegasus.x/backend`: PASS (0 failures across all 80+ packages)
  - `pnpm check-types` in `pegasus.x`: PASS (11/11 packages successful)
