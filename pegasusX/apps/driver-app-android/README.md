# Driver App Android (pegasusX)

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


Kotlin/Compose driver execution client for the pegasusX local stack.

## Backend

- API: `http://10.0.2.2:8180` (emulator; override `dev.host` in `local.properties`)
- WebSocket: `/v1/ws` with `Authorization: Bearer <jwt>`
- Demo login: `+998901000066` / PIN `1234`

## Build

```bash
cd pegasusX/apps/driver-app-android
cp local.properties.example local.properties   # optional: dev.host, MAPS_API_KEY
./gradlew :app:assembleDebug
```

Optional websocket model regeneration (requires `quicktype` + Go):

```properties
driver.ws.codegen=true
```

Checked-in stub: `app/src/main/java/com/pegasusx/driver/generated/contracts/PegasusWSEventEnvelope.kt`

## pegasusX deltas vs legacy pegasus

| Area | pegasusX |
|------|----------|
| Package / applicationId | `com.pegasusx.driver` |
| API port | `8180` |
| Command WS | `/v1/ws` (not `/v1/ws/driver`) |
| Telemetry WS | `/v1/ws` + Bearer (not `/ws/telemetry`) |
| WS codegen | off by default (`driver.ws.codegen`) |

Several delivery edge endpoints and notifications are scaffolded on the backend for simulator flows; production handlers land with Spanner-backed order lifecycle work.

## Realtime refresh contract

`ManifestViewModel.loadManifest(silent:)` on WS/reconnect; no `key(refreshEpoch) { NavHost }`. See `pegasusX/docs/ROLE_ROW_PARITY_MATRIX.md`.
