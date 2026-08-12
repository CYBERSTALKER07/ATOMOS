# warehouse-portal (desktop)

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


Next.js 15 + **Tauri 2** desktop shell for **WAREHOUSE_ADMIN** operators. Same route map as `pegasus/apps/warehouse-portal` (dashboard, orders, dispatch, inventory, supply, forecast, fleet, treasury, etc.).

## Stack

- **Next.js 15** App Router (static export for Tauri via `TAURI_BUILD=1`)
- **Tauri 2** — keyring-backed JWT storage, shell/dialog/fs plugins
- **Tailwind v4** + `@pegasusx/ui-kit` desktop foundation
- **API**: pegasusX backend default `http://localhost:8180`

## Auth

- `POST /v1/auth/warehouse/login` — phone + PIN (demo: `+998901000088` / `1234`)
- `POST /v1/auth/warehouse/refresh` — refresh token rotation
- WebSocket: `GET /v1/warehouse/ws-session` → short-lived `token_use=ws` ticket → `/v1/ws?token=…` (browsers cannot set Authorization on WebSocket).

## Commands

```bash
cd pegasusX
pnpm install

# Web only (browser)
pnpm --filter @pegasusx/warehouse-portal dev

# Desktop (Tauri + Next dev server on :3002)
pnpm --filter @pegasusx/warehouse-portal tauri:dev

# Release bundle
pnpm --filter @pegasusx/warehouse-portal tauri:build
```

## Environment

```bash
# .env.local
NEXT_PUBLIC_API_URL=http://localhost:8180
```

Tauri CSP allows `localhost:8180` and matching WebSocket origins.

## Role sync

Ship changes with **warehouse-app-ios**, **warehouse-app-android**, and pegasusX `backend-go/warehouse` contracts.
