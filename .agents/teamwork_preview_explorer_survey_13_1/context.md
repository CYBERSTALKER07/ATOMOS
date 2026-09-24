# Survey Explorer 1 Context: Backend Domain Subrouter & Route Module Decomposition

Mission:
Survey `backend/internal/api/router.go` and existing API structures in `pegasus.x`.
Assess how `router.go` (currently ~2,448 lines) is structured:
- List all route groups and endpoints currently registered.
- Identify the 5 target domains: Logistics, Warehouse/WMS, Commercial/Retail, Finance/Soliq, and Core System.
- Inspect how `Server` initializes handlers, services, and route trees.
- Check for existing modules or interfaces in `backend/internal/api/` or `backend/internal/api/modules/`.
- Identify inline DTO structs and duplicate request payloads in API handlers.
- Propose a clean `Module` interface pattern (e.g. `RegisterRoutes(r chi.Router)`) and file decomposition plan to reduce `router.go` to <950 lines while preserving 100% route contract parity.
- Check current compilation/test status with `go vet ./...` and `go test ./internal/api/...`.
- Write your comprehensive findings to `handoff.md` in your working directory.
