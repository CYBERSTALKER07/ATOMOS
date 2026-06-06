# factory-portal (desktop)

Next.js 15 + **Tauri 2** desktop shell for **FACTORY_ADMIN** operators. Route map matches the Pegasus reference: dashboard, loading bay, transfers, supply requests, payload override, fleet, staff, insights.

## Stack

- **Next.js 15** App Router (static export for Tauri via `TAURI_BUILD=1`)
- **Tauri 2** — keyring-backed JWT, shell/dialog/fs plugins
- **Tailwind v4** + `@pegasusx/ui-kit` desktop foundation
- **API**: pegasusX backend default `http://localhost:8180`

## Auth

- `POST /v1/auth/factory/login` — phone + password
- `POST /v1/auth/factory/refresh` — refresh token rotation
- WebSocket: `GET /v1/ws?token=…` (factory hub rooms)

## Commands

```bash
cd pegasusX
pnpm install

# Web only (browser, :3003)
pnpm --filter @pegasusx/factory-portal dev

# Desktop (Tauri + Next on :3003)
pnpm --filter @pegasusx/factory-portal tauri:dev

pnpm --filter @pegasusx/factory-portal typecheck
```

## Environment

```bash
# .env.local
NEXT_PUBLIC_API_URL=http://localhost:8180
```

Port **3003** avoids collision with warehouse-portal on **3002**.
