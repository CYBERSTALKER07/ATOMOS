# Wave B5 — Platform admin break-glass
> **POINT-IN-TIME SNAPSHOT (2026-08-12) — do not treat as current status.**
> Re-verify any claim against live code before acting. Multiple ecosystem hardening phases have shipped since this audit.


**Date:** 2026-08-13  
**Master:** [`BACKEND_PARITY_MASTER.md`](./BACKEND_PARITY_MASTER.md)

## Changes

### M-P0-10 Partner key revoke tenant scope
- `HandleRevokeKey` for `PLATFORM_ADMIN` requires `tenant_type` + `tenant_id` (query preferred, body fallback) — mirrors list/issue
- admin-portal `revokePartnerKey` + PartnerPanel pass tenant from panel state

### M-P0-11 Money-flag approve fail-closed audit
- On approve success, if `RecordFlagAudit` fails → `RevertApproveToPending` so flag is not ACTIVE without audit trail
- `RevertApproveToPending` on featureflags service

### M-P0-12 Dunning run-once role
- `HandleRunDunningOnce` allows `RoleAdmin` **or** `RolePlatformAdmin` (matches router)

### M-P1-11 MFA step-up on break-glass mutators
- `mfa.RequireStepUp` on:
  - `/v1/admin/partner-keys` (issue/list/revoke)
  - product match queue list/resolve
  - `/v1/admin/ar/dunning/run-once`
- Middleware no-ops for non-`PLATFORM_ADMIN` (supplier ADMIN / retailer paths unchanged)

## Residual (not this wave)
- Tenant transition + audit single txn
- Partner issue/match resolve PlatformAdminAudit rows
- MFA enroll audit fail-closed
- Flag set audit atomicity (less severe than approve)

## Verification

```bash
cd apps/backend-go
go test ./partner/ ./featureflags/ ./ar/ ./mfa/ ./globalproductsroutes/ ./creditroutes/ -count=1
go build -o /tmp/pegasusx-backend .
```
