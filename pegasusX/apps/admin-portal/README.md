# PegasusX Admin Console

Real platform-governance console for the `PLATFORM_ADMIN` surface
(`apps/backend-go/platformadmin`, `apps/backend-go/featureflags`). Replaces the
retired redirect stub.

## Surfaces

- **Tenants** — list/filter `PlatformTenants`, run lifecycle transitions
  (PENDING → APPROVED/SUSPENDED/OFFBOARDED) with optional KYB notes.
- **Flags** — evaluate + set tenant feature-flag overrides. Money-affecting
  flags (`AR_INVOICES_ENABLED`, `AR_DUNNING_ENABLED`, `AUTO_ORDER_PLACE_ENABLED`,
  `AUTO_ORDER_SHADOW_ENABLED`, `FISCAL_PROVIDER`) are **dual-controlled**: a set
  is stored PENDING and only takes effect after a *different* PLATFORM_ADMIN
  approves via the Approve action.
- **Audit** — read `PlatformAdminAudit` immutable action log.

## Auth

Bearer-token session (break-glass governance identity), held in memory only —
no persistence. Backend enforces `auth.RequireRole(auth.RolePlatformAdmin)` on
every route; this console just forwards the token.

Mint distinct subjects for dual-control (set vs approve must differ):

```bash
cd apps/backend-go
go run ./cmd/mint-dev-jwt -role PLATFORM_ADMIN -subject admin-a
go run ./cmd/mint-dev-jwt -role PLATFORM_ADMIN -subject admin-b
# Local bypass of TOTP step-up (never in prod):
go run ./cmd/mint-dev-jwt -role PLATFORM_ADMIN -subject admin-a -mfa
```

**MFA (TOTP):** production sets `PLATFORM_ADMIN_MFA_REQUIRED=true` (default when
`PEGASUSX_ENV=production`). After pasting a base token, the console enrolls /
verifies via `/v1/platform-admin/mfa/*` and replaces the session with an
`mfa_verified` JWT. Governance routes (tenants/flags/audit) reject unverified
tokens when MFA is enrolled or required.

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
