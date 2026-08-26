# Worker M1 Working Directory
Assigned: Milestone 1 (DevOps & Backend Architecture)
- Consolidate sandbox-infra smoke gate into `.github/workflows/pegasusx-ci.yml` and fix `reatilerapp` typo in `.github/workflows/pegasusx-native-mobile-build.yml` and `.github/ACT.md`.
- Split `bootstrap.go` into `config.go`, `app.go`, `infra.go`, `services.go`, `workers.go`, `queries.go` in package `bootstrap`.
- Migrate `spanner.Client.Apply` usages in `factory/auth_register.go`, `factory/planning_service.go`, `warehouse/auth_register.go`, `warehouse/setup.go`, `warehouse/dispatch_runs.go`, `warehouse/ops_portal.go` to `ReadWriteTransaction` + `outbox.EmitJSON`.
- Run Go builds and tests to verify everything passes cleanly.
