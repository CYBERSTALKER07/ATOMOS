# supplier-portal

Next.js 15 + React 19 **supplier portal** (web + Tauri 2 desktop + **Android**). Pegasus equivalent: `admin-portal`. The product user is a **SUPPLIER**; JWT role remains `ADMIN` for legacy compatibility.

## Stack

- Next.js 15 App Router (static export when `TAURI_BUILD=1`)
- Tauri 2 — desktop + Android (`com.pegasusx.supplier`)
- Tailwind v4 + hand-rolled M3 CSS tokens
- `@pegasusx/api-client` → pegasusX backend default **`http://localhost:8180`**

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

# Tauri Android (emulator → host backend)
NEXT_PUBLIC_API_URL=http://10.0.2.2:8180 pnpm --filter @pegasusx/supplier-portal tauri:android:dev
```

## Environment

```bash
# .env.local (web + build-time Tauri)
NEXT_PUBLIC_API_URL=http://localhost:8180

# Next.js API proxy (browser dev only)
SUPPLIER_BACKEND_BASE_URL=http://localhost:8180
```

## Auth surfaces

| Route | Purpose |
|-------|---------|
| `/auth/register` | 4-step supplier onboarding |
| `/auth/login` | Phone + password sign-in (required on Tauri/Android) |
| `/setup/billing` | Post-registration billing gate |

Login/register responses include an additive `token` field for native shells; browsers still receive the `supplier_jwt` cookie.

## Role sync

SUPPLIER is **web/Tauri only** in the role matrix — no separate Kotlin app. Ship portal changes with `pegasusX/apps/backend-go/supplier*` routes and shared `packages/types` / `packages/api-client`.
