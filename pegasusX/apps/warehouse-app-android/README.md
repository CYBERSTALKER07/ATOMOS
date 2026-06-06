# warehouse-app-android

Kotlin + Jetpack Compose Material 3 warehouse admin client for pegasusX. Parity target: `pegasus/apps/warehouse-app-android` + iOS More hub routes.

## Stack

- Hilt, Retrofit, kotlinx.serialization, Navigation Compose
- Encrypted prefs for JWT + refresh token
- WebSocket: `GET /v1/ws?token=…` (warehouse + supplier rooms)

## Local dev

1. Start pegasusX backend on **port 8180** (default for this tree).
2. Emulator API base: `http://10.0.2.2:8180` (override `dev.host` in `local.properties`).
3. Physical device: set `dev.host` to your machine LAN IP.

Demo login (env `WAREHOUSE_DEMO_*` or defaults):

- Phone: `+998901000088`
- PIN: `1234`

## Build

```bash
cd pegasusX/apps/warehouse-app-android
./gradlew :app:assembleDebug
```

Optional WS contract codegen (requires `quicktype` CLI):

```properties
warehouse.ws.codegen=true
quicktype.path=/path/to/quicktype
```

## Surfaces

Dashboard KPIs, orders (+ detail), drivers, vehicles, inventory, products, manifests, analytics, CRM, returns, treasury, dispatch (preview / supply / locks), staff, **More** hub (manifests, forecast, etc.), demand forecast.
