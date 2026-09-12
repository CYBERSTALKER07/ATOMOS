# Worker M1 Working Directory
Assigned: Milestone 1 (DevOps & Backend Architecture)
- Consolidate sandbox-infra smoke gate into `.github/workflows/pegasusx-ci.yml` and fix `reatilerapp` typo in `.github/workflows/pegasusx-native-mobile-build.yml` and `.github/ACT.md`.
- Split `bootstrap.go` into `config.go`, `app.go`, `infra.go`, `services.go`, `workers.go`, `queries.go` in package `bootstrap`.
- Migrate `spanner.Client.Apply` usages in `factory/auth_register.go`, `factory/planning_service.go`, `warehouse/auth_register.go`, `warehouse/setup.go`, `warehouse/dispatch_runs.go`, `warehouse/ops_portal.go` to `ReadWriteTransaction` + `outbox.EmitJSON`.
- Run Go builds and tests to verify everything passes cleanly.


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
