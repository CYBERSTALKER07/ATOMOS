## 2026-08-20T19:42:09Z
You are the Worker for Milestone 1 (M1: DevOps and Backend Architecture).
Working Directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m1
Workspace Root: /Users/shakhzod/Desktop/V.O.I.D
Authoritative Request: /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md
Explorer Handoff Report: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_r1/handoff.md (read this thoroughly!)

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. An auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

File Ownership:
- `.github/workflows/pegasusx-ci.yml`
- `.github/workflows/pegasusx-native-mobile-build.yml`
- `.github/ACT.md`
- `pegasusX/apps/backend-go/bootstrap/*` (`config.go`, `app.go`, `infra.go`, `services.go`, `workers.go`, `queries.go`)
- `pegasusX/apps/backend-go/factory/auth_register.go`, `pegasusX/apps/backend-go/factory/planning_service.go`
- `pegasusX/apps/backend-go/warehouse/auth_register.go`, `pegasusX/apps/backend-go/warehouse/setup.go`, `pegasusX/apps/backend-go/warehouse/dispatch_runs.go`, `pegasusX/apps/backend-go/warehouse/ops_portal.go`

Tasks:
1. Fix the `reatilerapp` typo in `.github/workflows/pegasusx-native-mobile-build.yml` (scheme and project path) and in `.github/ACT.md`. Consolidate the `sandbox-infra.yml` smoke gate (`make test-sandbox-infra`) into `.github/workflows/pegasusx-ci.yml`.
2. Modularize `pegasusX/apps/backend-go/bootstrap/bootstrap.go` into `config.go`, `app.go`, `infra.go`, `services.go`, `workers.go`, `queries.go` in `package bootstrap`. Ensure all symbols remain in `package bootstrap` and the old monolith file is replaced/split cleanly so `go test ./bootstrap/...` and `go build ./...` pass without issues.
3. Migrate `spanner.Client.Apply` calls in:
   - `pegasusX/apps/backend-go/factory/auth_register.go`
   - `pegasusX/apps/backend-go/factory/planning_service.go`
   - `pegasusX/apps/backend-go/warehouse/auth_register.go`
   - `pegasusX/apps/backend-go/warehouse/setup.go`
   - `pegasusX/apps/backend-go/warehouse/dispatch_runs.go`
   - `pegasusX/apps/backend-go/warehouse/ops_portal.go`
   to use `ReadWriteTransaction` (`s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error { ... })` and outbox buffer where appropriate).

Verification:
- Run `go test ./bootstrap/...` and `go build ./...` in `pegasusX/apps/backend-go`.
- Run `go test ./factory/... ./warehouse/...` in `pegasusX/apps/backend-go`.
- Verify `grep -ri "reatilerapp" .` returns no results.
- Verify `grep -rn "\.Apply(" pegasusX/apps/backend-go/factory pegasusX/apps/backend-go/warehouse` returns no `.Apply` calls in the target files.

Deliverables:
- Write `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m1/changes.md` detailing all edits.
- Write `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m1/handoff.md` with build and test commands and outputs.
- Send a completion message to parent when done.


# Universal Agent & Engineering Guidelines
When developing, designing, or planning, always ensure to account for:
- Gaps, edge cases, and comprehensive feature validation.
- Best practices and optimized integration for Kafka, Redis, Backend, Optimizers, AI, and UI.
- Real-time concepts including WebSockets, webhooks, and their native app equivalents.
- Thorough business logic for features, understanding how the role, app, and ecosystem work together, and engagements with other roles and features.
- Best practices for backend, frontend, and infrastructure libraries/packages. Always prefer existing, high-quality open-source libraries and packages that best suit our features before creating our own.
- Optimal UI infrastructure and UX patterns (e.g., optimal screen positioning for drivers during an active route), applying the same high standards to backend and cloud architecture.
- ALWAYS search the web to find open-source code, libraries, packages, math, algorithms, approaches, and best practices for anything we are doing. If none exist, then create our own.
- Always search the web to get the correct logic, and incorporate edge cases, business logic for features, operations (ops), workflow, data consistency, finance, and AI into everything we do.
