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

## Run

```bash
NEXT_PUBLIC_BACKEND_BASE_URL=http://localhost:8080 pnpm install
pnpm dev
```

Set `NEXT_PUBLIC_BACKEND_BASE_URL` to the backend origin (default
`http://localhost:8080`).
