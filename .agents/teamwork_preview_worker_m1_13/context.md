# Worker M1 Context: Backend Domain Subrouter & Route Module Decomposition

Working Directory: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m1_13`
Target Workspace Directory: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`
Authoritative Requirements: `/Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md`
Explorer Findings: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_13_1/handoff.md`
Project Plan: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_13/PROJECT.md`

File Write Ownership (Exclusive):
- `backend/internal/api/modules/module.go`
- `backend/internal/api/logistics.go`
- `backend/internal/api/warehouse.go`
- `backend/internal/api/commercial.go`
- `backend/internal/api/finance.go`
- `backend/internal/api/core.go`
- `backend/internal/api/dto.go`
- `backend/internal/api/router.go`

Mission & Tasks:
1. Create `backend/internal/api/modules/module.go` defining the `Module` interface and `Registry`:
   - `type Module interface { Name() string; RegisterRoutes(r chi.Router) }`
   - `type Registry struct { modules []Module }`
   - `func NewRegistry() *Registry`, `func (reg *Registry) Register(m Module)`, `func (reg *Registry) MountAll(r chi.Router)`
2. Create `backend/internal/api/dto.go` unifying repeated inline DTO structs (Refresh Token, Supervisor PIN, Operational Status, Dispatch Buffer, Action Reason, Supplier Attachment, etc.) preserving exact JSON tags and types.
3. Create 5 domain subrouter files in `package api`:
   - `backend/internal/api/core.go`: `CoreModule` implementing `modules.Module`, with `s *Server`. Mounts probes, policy, metrics, JWKS, public auth, real-time sync/webhooks, onboarding lifecycle, users, notifications.
   - `backend/internal/api/logistics.go`: `LogisticsModule` implementing `modules.Module`, with `s *Server`. Mounts telemetry, fleet, driver, dispatch, manifests, epod, control tower, gs1, routing.
   - `backend/internal/api/warehouse.go`: `WarehouseModule` implementing `modules.Module`, with `s *Server`. Mounts warehouse onboarding/metrics/stock/inventory, dock, transfers, cycle counts, replenishment, pick waves, floor exceptions, coverage, demand forecast, scheduling, crossdock, returns, bins, lots, empties, QM, EWM.
   - `backend/internal/api/commercial.go`: `CommercialModule` implementing `modules.Module`, with `s *Server`. Mounts catalog, products, orders, checkout, supplier profile/kyc/pricing/crm/promotions, retailer store POS/shifts/holds/local SKUs/team/pulse/auto-order, UMP, claims, voice ordering.
   - `backend/internal/api/finance.go`: `FinanceModule` implementing `modules.Module`, with `s *Server`. Mounts Soliq OFD fiscalization, credit notes, cash, payments, credit limits/apps, AR, payouts, FX, seasonality, Global Pay webhooks, SoftPOS charge, COPA/FSCM/payroll.
4. Refactor `backend/internal/api/router.go`:
   - In `Router()`: initialize `modules.NewRegistry()`, register the 5 modules passing `s`, and call `reg.MountAll(r)`.
   - Preserve all middlewares (`RequestID`, `RealIP`, `StructuredLoggingMiddleware`, `Recoverer`, `metricsReg.HTTPMiddleware`, `tracer.HTTPTracingMiddleware`, `IdempotencyMiddleware`, `cors.Handler`), protected auth group (`auth.RequireAuthWithKeyManager`), and the 4 onboarding gates.
   - Verify line count of `router.go` is reduced to <900 lines (target ~765 lines, >60% reduction).
5. Verification:
   - Run `go vet ./...` in `backend/` -> must exit 0 with 0 diagnostics.
   - Run `go test -v ./internal/api/...` -> must pass 100%.
   - Run `go test -v -race ./internal/api/...` -> must pass 100% with 0 race conditions.
6. Write detailed handoff report to `handoff.md` in your working directory.
