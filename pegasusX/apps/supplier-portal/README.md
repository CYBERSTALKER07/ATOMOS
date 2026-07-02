# supplier-portal

Next.js 15 + React 19 **supplier portal** (web + Tauri 2 desktop + **Android**). Pegasus equivalent: `supplier-portal`. The product user is a **SUPPLIER**; JWT role remains `ADMIN` for legacy compatibility.

## Stack

- Next.js 15 App Router (static export when `TAURI_BUILD=1`)
- Tauri 2 — **desktop shell** (`com.pegasusx.supplier`)
- Tailwind v4 + hand-rolled M3 CSS tokens
- `@pegasusx/api-client` → pegasusX backend default **`http://localhost:8180`**

> **Role-row note (PX-DESK-2F):** Field supplier ops on phones/tablets use **native Kotlin (`supplier-app-android`) and SwiftUI (`supplier-app-ios`)**. This portal is the desktop/web cockpit. The Tauri **Android** target in `src-tauri/` is **deprecated** — do not add new features there; use native mobile for parity.

## Onboarding (4 steps + billing gate)

1. Account  
2. Topology (factories + warehouses)  
3. Business  
4. Categories  
5. **`/setup/billing`** (bank + payment gateway — not in the wizard)  
6. Dashboard  

## Commands

```bash
cd pegasusX
pnpm install

# Browser dev (API proxied via /api → :8180)
pnpm --filter @pegasusx/supplier-portal dev

# Tauri desktop
pnpm --filter @pegasusx/supplier-portal tauri:dev

# Tauri Android (deprecated — use supplier-app-android)
# NEXT_PUBLIC_API_URL=http://10.0.2.2:8180 pnpm --filter @pegasusx/supplier-portal tauri:android:dev
```

## Environment

```bash
# .env.local (web + build-time Tauri)
NEXT_PUBLIC_API_URL=http://localhost:8180

# Next.js API proxy (browser dev only)
SUPPLIER_BACKEND_BASE_URL=http://localhost:8180
```

## Auth surfaces

- **Web / Tauri / desktop:** `supplier-portal` (this app)
- **Native Android:** `pegasusX/apps/supplier-app-android` (Kotlin/Compose)
- **Native iOS:** `pegasusX/apps/supplier-app-ios` (SwiftUI)

All three clients share the same `/v1/supplier/*` API and single-tenant onboarding flow (register → business setup → billing gate → dashboard).

| Route | Purpose |
|-------|---------|
| `/auth/register` | 4-step supplier onboarding |
| `/auth/login` | Phone + password sign-in (required on Tauri/Android) |
| `/setup/billing` | Post-registration billing gate |

Login/register responses include an additive `token` field for native shells; browsers still receive the `supplier_jwt` cookie.

## Role sync

SUPPLIER row: **supplier-portal (desktop/web)** + **supplier-app-android** + **supplier-app-ios**. Ship API changes with `pegasusX/apps/backend-go/supplier*` and shared `packages/types` / `packages/api-client`. Native mobile is canonical for field ops; Tauri Android is not a parity target.
