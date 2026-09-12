# retailer-app-desktop

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


Next.js 15 + Tauri 2 retailer desktop for pegasusX. Parity target: `apps/retailer-app-desktop` + mobile apps.

## Local dev

```bash
cd pegasusX
pnpm install
cd apps/retailer-app-desktop
NEXT_PUBLIC_API_URL=http://localhost:8180 \
NEXT_PUBLIC_WS_URL=ws://localhost:8180/v1/ws \
pnpm dev
```

Tauri dev:

```bash
pnpm tauri:dev
```

Demo login: `+998901000077` / `1234` (backend on 8180).

## Build

```bash
pnpm typecheck
TAURI_BUILD=1 pnpm build:static
pnpm tauri:build
```

Uses `@pegasusx/ui-kit`, `@pegasusx/types`, `@pegasusx/i18n` from the pegasusX workspace.
