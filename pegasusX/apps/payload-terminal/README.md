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

WebSocket: `/v1/ws?token=<jwt>` (unified hub).
