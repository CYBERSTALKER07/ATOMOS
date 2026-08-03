# payload-app-ios

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


SwiftUI iPad app for payload operators.

- Default API: `http://localhost:8180` (override with `PEGASUS_DEV_HOST` in Xcode scheme)
- Auth: `POST /v1/auth/payloader/login`
- WebSocket: `/v1/ws?token=...`

```bash
cd pegasusX/apps/payload-app-ios
xcodegen generate
open payload-app-ios.xcodeproj
```

Demo: `+998901110022` / `33333333`
