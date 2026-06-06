# factory-app-android

Kotlin + Jetpack Compose Material 3 factory admin client for pegasusX. Parity target: `pegasus/apps/factory-app-android` + `factory-app-ios`.

## Stack

- Hilt, Retrofit, kotlinx.serialization, Navigation Compose
- Encrypted prefs for JWT + refresh token
- WebSocket: `GET /v1/ws?token=…` (factory role; shared hub with supplier/warehouse rooms)

## Local dev

1. Start pegasusX backend on **port 8180**.
2. Emulator API base: `http://10.0.2.2:8180` (override `dev.host` in `local.properties`).
3. Physical device: set `dev.host` to your machine LAN IP.

Demo login (env `FACTORY_DEMO_*` or defaults):

- Phone: `+998901000099`
- PIN: `1234`

## Build

```bash
cd pegasusX/apps/factory-app-android
./gradlew :app:assembleDebug
```

Optional WS contract codegen (requires `quicktype` CLI):

```properties
factory.ws.codegen=true
quicktype.path=/path/to/quicktype
```

## Surfaces

Dashboard, transfers (+ detail / transition), loading bay, supply requests, payload override (manifests), fleet, staff, insights (replenishment).
