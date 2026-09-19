# payload-terminal

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


Expo / React Native loading-bay terminal for payload operators (`role=PAYLOAD`).

## Local dev

```bash
cd pegasusX/apps/payload-terminal
npm install
EXPO_PUBLIC_API_URL=http://localhost:8180 npm run start
```

Demo login: `+998901110022` / PIN `33333333` → `POST /v1/auth/payloader/login`.

WebSocket: browsers mint `GET /v1/payload/ws-session` then connect `/v1/ws?token=<ws-ticket>` (`token_use=ws` only). Native apps send `Authorization: Bearer` on `/v1/ws` (no query JWT).
