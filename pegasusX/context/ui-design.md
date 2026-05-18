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
