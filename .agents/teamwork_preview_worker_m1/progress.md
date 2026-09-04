# Progress — Milestone 1: DevOps and Backend Architecture

**Last visited**: 2026-08-20T19:43:30Z

## Current Status: Investigation & Planning
- [x] Initialized DISPATCH.md and BRIEFING.md
- [ ] Task 1: Fix `reatilerapp` typo in `.github/workflows/pegasusx-native-mobile-build.yml` & `.github/ACT.md`, consolidate sandbox smoke into `pegasusx-ci.yml`
- [ ] Task 2: Modularize `pegasusX/apps/backend-go/bootstrap/bootstrap.go` into `config.go`, `app.go`, `infra.go`, `services.go`, `workers.go`, `queries.go`
- [ ] Task 3: Migrate `spanner.Client.Apply` calls to `ReadWriteTransaction` in 6 target files
- [ ] Task 4: Run full verification (builds, tests, greps)
- [ ] Task 5: Produce changes.md, handoff.md and report completion to parent


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
