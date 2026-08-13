# G4 Proof — Tenancy, admin, ops

## Tests

```text
go test ./auth/ ./platformadmin/ ./dispatch/plan/ ./warehouse/ -count=1
# ok
```

## Evidence

| ID | Evidence |
|----|----------|
| G4-A1 | `auth.SeedFallbackAllowed`, PreferTenant tighten, `TENANCY_ENFORCEMENT.md` |
| G4-B1 | `POST /v1/auth/platform-admin/login`, `PlatformAdminUsers` DDL, admin portal LoginForm |
| G4-B2 | `GET /v1/platform-admin/ops/outbox/*`, `/ops/runtime`, OpsPanel |
| G4-C1 | `plan.OptimizerClass`, `optimizer_class` on execute/preview; `GET /v1/health/capabilities` |

## Residual

- Full OIDC/SSO deferred  
- Kafka DLQ browser → CLI `cmd/replay-dlq` honesty note  
