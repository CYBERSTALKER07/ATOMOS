# Domain 3 — Planning: Forecast Accuracy Surface

Date: 2026-08-12 (UTC+5)
Scope: Ecosystem Capability Roadmap, Domain 3 (Planning, P2) — forecast accuracy UI + planning read-path wiring.

## What was delivered end-to-end

The forecast-accuracy engine (`planning/accuracy.go`) already computed nightly
WAPE7/28, bias7/28, tracking-signal, and per-day `ForecastAccuracyDaily` rows —
but nothing could read them. There was no HTTP read endpoint, `HandleRunAccuracyOnce`
was unmounted, and no UI surfaced the metrics. This change closes that loop.

### Backend
- `planning/accuracy_handlers.go` — added `HandleListAccuracy`
  (`GET /v1/admin/planning/accuracy`): admin-gated, requires `supplier_id`,
  optional `warehouse_id` / `product_id` / `days` (1–90, default 28). Returns
  `{ items, days }` from `ListAccuracyRows`.
- `planning/accuracy_routes.go` (new) — `RegisterAccuracyRoutes` mounts the read
  endpoint plus the previously-orphaned `POST /v1/admin/planning/accuracy/run-once`
  ops trigger. Nil-service safe.
- `main.go` — imported `planning` and called
  `planning.RegisterAccuracyRoutes(r, app.ForecastAccuracy)` next to the other
  route registrations.

### Shared types / client
- `packages/types/index.ts` — `ForecastAccuracyDailyRow`, `ForecastAccuracyResponse`.
- `packages/api-client/index.ts` — `getForecastAccuracy({ supplierId, days,
  warehouseId, productId })` hitting the new admin endpoint via `appendQuery`.

### UI (supplier-portal)
- `components/settings/planning/ForecastAccuracyPanel.tsx` (new) — fetches the
  admin accuracy rows for the session supplier, reduces per-day rows to the latest
  row per (warehouse, product), and renders:
  - KPI stats: series count, average WAPE (28d), tracking-signal alert count.
  - A per-series table: product, warehouse, WAPE28, Bias28 (sign-coloured),
    tracking signal, and an Alert/OK status (|TS| > 4 → Alert).
  - A window selector (7/14/28/60/90d) and refresh.
- Mounted on `app/(portal)/settings/planning/page.tsx` above the seasonal
  overrides, alongside the existing `SignalIngestOpsPanel`.

## Tests
- `planning/accuracy_test.go` — added handler tests:
  - `TestHandleListAccuracyMethodNotAllowed` (405 on POST)
  - `TestHandleListAccuracyForbidden` (403 without claims and for non-admin role)
  - `TestHandleListAccuracyRequiresSupplier` (400 without `supplier_id`)
  - `TestRegisterAccuracyRoutesNilSafe` (no panic on nil service)
- All pass: `go test ./planning/ -run TestHandleListAccuracy` → ok.
- Full backend `go build ./...` clean; `go vet ./planning/` clean.
- Supplier-portal `pnpm test` → 5 files / 8 tests pass.

## Notes / pre-existing (not introduced here)
- `pnpm typecheck` in supplier-portal reports ~100 pre-existing React-19
  `ReactNode`-assignability errors across shared `@pegasusx/ui-kit`
  (PortalPrimitives, PageChrome, GlassmorphismPanel, …) and several unrelated
  pages. None reference the files touched in this change; a scoped `tsc` over
  `ForecastAccuracyPanel.tsx` + its imports reports zero errors.
- The prod `optimizer-core` image/replicas gap flagged by the End-Product Reality
  Report is an external build/publish step (no in-repo optimizer source/Dockerfile);
  it cannot be resolved from this tree and is tracked separately.
