# PegasusX Admin Console

Real platform-governance console for the `PLATFORM_ADMIN` surface
(`apps/backend-go/platformadmin`, `apps/backend-go/featureflags`). Replaces the
retired redirect stub.

## Surfaces

- **Tenants** — list/filter `PlatformTenants`, run lifecycle transitions
  (PENDING → APPROVED/SUSPENDED/OFFBOARDED) with optional KYB notes.
- **Flags** — evaluate + set tenant feature-flag overrides. Money-affecting
  flags (`AR_INVOICES_ENABLED`, `AR_DUNNING_ENABLED`, `AUTO_ORDER_PLACE_ENABLED`,
  `AUTO_ORDER_SHADOW_ENABLED`, `AUTO_ORDER_SOAK_GATE_DISABLED`, `FISCAL_PROVIDER`)
  are **dual-controlled**: a set is stored PENDING and only takes effect after a
  *different* PLATFORM_ADMIN approves via the Approve action.
- **Audit** — read `PlatformAdminAudit` immutable action log.

## Auth (G4)

Primary path: **password login** via `POST /v1/auth/platform-admin/login`
(subject/email + password). Backend table `PlatformAdminUsers` or env bootstrap:

```bash
export PLATFORM_ADMIN_SUBJECT=admin-a
export PLATFORM_ADMIN_PASSWORD='…'
# optional persist after migration:
# go run ./cmd/apply-migration --ddl schema/migrations/20260813_g4_platform_admin_users.ddl
```

Token is held in memory only. Backend enforces `auth.RequireRole(auth.RolePlatformAdmin)`.

**Token paste** is secondary (dev / break-glass): shown when
`NEXT_PUBLIC_ALLOW_TOKEN_PASTE=true` or `NODE_ENV=development`.

```bash
cd apps/backend-go
go run ./cmd/mint-dev-jwt -role PLATFORM_ADMIN -subject admin-a
go run ./cmd/mint-dev-jwt -role PLATFORM_ADMIN -subject admin-b -mfa
```

**MFA (TOTP):** production sets `PLATFORM_ADMIN_MFA_REQUIRED=true` (default when
`PEGASUSX_ENV=production`). After login, the console enrolls / verifies via
`/v1/platform-admin/mfa/*` and replaces the session with an `mfa_verified` JWT.

**Ops tab:** outbox unpublished lag + runtime run_mode honesty
(`GET /v1/platform-admin/ops/*`).

**Live updates (WS):** after MFA, the console opens `/v1/platform-admin/ws-session`
→ `/v1/ws` on the `platform-admin` room. Tenant transitions, flag set/approve,
and MFA audit events refresh Tenants/Audit tabs. Push/FCM is intentionally not
used for this break-glass desktop console (WS is the real-time path).

Flag set + approve both write `PlatformAdminAudit` (fail-closed if audit insert fails).

## Run

```bash
NEXT_PUBLIC_BACKEND_BASE_URL=http://localhost:8080 pnpm install
pnpm dev
```

Set `NEXT_PUBLIC_BACKEND_BASE_URL` to the backend origin (default
`http://localhost:8080`).
