# retailer-app-android

Kotlin + Jetpack Compose Material 3 retailer client for pegasusX. Parity target: `pegasus/apps/retailer-app-android` + iOS + desktop.

## Stack

- Hilt, Retrofit, kotlinx.serialization, Navigation Compose, Room (offline cart)
- Encrypted prefs for JWT
- WebSocket: `GET /v1/ws` with `Authorization: Bearer …`

## Local dev

1. Start pegasusX backend on **port 8180**.
2. Emulator: `dev.host=10.0.2.2` in `local.properties` (default).
3. Physical device: set `dev.host` to your Mac LAN IP.

Demo login (`RETAILER_DEMO_*` or defaults):

- Phone: `+998901000077`
- Password: `1234`

## Build

```bash
cd pegasusX/apps/retailer-app-android
./gradlew :app:assembleDebug
```

Optional WS codegen:

```properties
retailer.ws.codegen=true
```

## Surfaces

Auth, catalog, cart, checkout, orders, tracking map, suppliers, analytics, profile, notifications, auto-order settings.
