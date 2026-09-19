# factory-portal (desktop)

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


Next.js 15 + **Tauri 2** desktop shell for factory operators. All authenticated factory-scoped users may edit their assigned factory location (setup + Settings → Location).

## Stack

- **Next.js 15** App Router (static export for Tauri via `TAURI_BUILD=1`)
- **Tauri 2** — keyring-backed JWT, shell/dialog/fs plugins
- **Tailwind v4** + `@pegasusx/ui-kit` desktop foundation
- **API**: pegasusX backend default `http://localhost:8180`

## Auth

- `POST /v1/auth/factory/login` — Firebase phone OTP (`id_token`) or phone + password (dev)
- `POST /v1/auth/factory/refresh` — refresh token rotation
- WebSocket: `GET /v1/factory/ws-session` → short-lived `token_use=ws` ticket → `/v1/ws?token=…` (factory hub rooms).

Firebase phone OTP uses the Auth emulator in development (`NEXT_PUBLIC_FIREBASE_AUTH_EMULATOR_HOST`, default `http://localhost:9099`).

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
NEXT_PUBLIC_FIREBASE_API_KEY=demo-key
NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN=demo-pegasus.firebaseapp.com
NEXT_PUBLIC_FIREBASE_PROJECT_ID=demo-pegasus
NEXT_PUBLIC_FIREBASE_AUTH_EMULATOR_HOST=http://localhost:9099
```

Port **3003** avoids collision with warehouse-portal on **3002**.
