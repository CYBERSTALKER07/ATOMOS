# pegasusX UI/UX Doctrine

## Core Directives
1. **Operational, not decorative**. No gradients, glassmorphism, decorative patterns. Solid Material 3 surface tokens only.
2. **No emojis as indicators**. Real SVG icon sets only (Material Symbols, Heroicons, Lucide for web; SF Symbols for iOS; Material Symbols for Android).
3. **Surface completeness**. Every screen must handle: loading skeleton, empty, offline/disconnected, stale, permission-restricted.
4. **Telemetry visibility**. Live execution state (drivers, routes, orders) preferred over static snapshots. Drill-down from aggregates is mandatory.

## Platform UI Stack
| Surface | Stack | UI System |
|---|---|---|
| supplier-portal (web/desktop) | Next.js 15 + Tauri | Tailwind v4 + hand-rolled M3 tokens |
| retailer-app-desktop | Next.js 15 + Tauri | Tailwind v4 + M3 tokens |
| warehouse-portal, factory-portal | Next.js 15 + Tauri | Tailwind v4 + M3 tokens |
| Android (driver/retailer/warehouse/factory/payload) | Kotlin/Compose | Jetpack Compose Material 3 |
| iOS (driver/retailer/warehouse/factory/payload) | SwiftUI | Native Apple HIG + SF Symbols |
| payload-terminal | Expo / RN | M3 discipline via RN styling |

## Layout graft rule (supplier-portal)

When adding or changing supplier-portal pages, **graft pegasus admin-portal chrome** without copying pegasus source into git:

1. **Shell** — authenticated routes render inside `components/SupplierShell` (collapsible rail, topbar, breadcrumbs, ⌘K nav search).
2. **Page chrome** — use `components/PageChrome` (`desk-page`, `desk-page-header`, `desk-toolbar`) or the `PortalSurface` re-export; do not add ad-hoc `p-6 max-w-7xl` page wrappers.
3. **Dashboard** — Bento protocol only: `BentoGrid` + `BentoCard`/`BentoSkeleton` with semantic `size` props.
4. **Auth** — `app/auth/layout.tsx` split panel + `auth-card` forms; billing gate uses `app/setup/layout.tsx` centered shell.
5. **Data layer frozen** — keep `@pegasusx/api-client` hooks and field names; layout-only diffs unless Boss requests contract changes.

Reference measurement files (read-only): `pegasus/apps/admin-portal/components/AdminShell.tsx`, `BentoGrid.tsx`, `app/auth/layout.tsx`.

## Layout graft rule (warehouse-portal)

When adding or changing warehouse-portal pages, **graft pegasus warehouse-portal chrome** without copying pegasus source into git:

1. **Shell** — authenticated routes render inside `components/WarehouseShell` (collapsible rail, topbar, breadcrumbs, ⌘K nav search). pegasusX-only `/transfers` lives under Operations.
2. **Page chrome** — use `components/PageChrome` (`desk-page`, `desk-page-header`, `desk-toolbar`); do not add ad-hoc `p-6` page wrappers.
3. **Dashboard** — KPI card grid + motion stagger (pegasus warehouse pattern), **not** supplier BentoGrid.
4. **Auth** — `app/auth/layout.tsx` split panel + `auth-card` login (phone + PIN to `/v1/auth/warehouse/login`).
5. **Data layer frozen** — keep `lib/warehouse-api.ts` and `lib/warehouse-ops.ts`; layout-only diffs unless Boss requests contract changes.

Reference measurement files (read-only): `pegasus/apps/warehouse-portal/components/WarehouseShell.tsx`, `app/page.tsx`, `app/auth/login/page.tsx`.

## Onboarding Bootstrap Flow (supplier-portal)
4 steps + separate billing gate. Same shape as Pegasus, repurposed for single-tenant company bootstrap.
1. **Account** — company, contact, email, phone (country prefix), password.
2. **Topology** — create factories + warehouses (supports local, remote, mixed). Replaces Pegasus's single-warehouse-on-supplier-row pattern.
3. **Business** — tax id, company registration number, cold-chain, palletization.
4. **Categories** — operating categories multi-select.
5. **Billing gate** (`/setup/billing`) — payment gateway + bank.
6. **Dashboard**.

Middleware enforces `IsConfigured` and redirects unconfigured tenants to `/setup/billing`.

## Frontend Context Gate
Before editing any UI feature, confirm:
1. backend endpoint, event, or DTO feeding the surface,
2. data-layer mapping (repository / view-model),
3. every client in the role row that consumes it,
4. exact UI primitive per platform,
5. loading / empty / offline / restricted / error states.
