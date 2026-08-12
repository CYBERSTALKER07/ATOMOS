# payload-app-android

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


Kotlin + Jetpack Compose Material 3 tablet app for payload operators.

- Default API: `http://10.0.2.2:8180` (emulator → host pegasusX backend)
- Auth: `POST /v1/auth/payloader/login` (phone + PIN)
- WebSocket: `/v1/ws` with `Authorization: Bearer <session JWT>` (no query token)

```bash
cd pegasusX/apps/payload-app-android
./gradlew :app:assembleDebug
```

Demo: `+998901110022` / `33333333`
