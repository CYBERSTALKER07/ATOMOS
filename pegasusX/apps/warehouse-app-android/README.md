# warehouse-app-android

Kotlin + Jetpack Compose Material 3 warehouse admin client for pegasusX. Parity target: `warehouse-portal` + iOS warehouse app.

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

## Navigation map (`WarehouseSection`)

On tablet width, navigation uses the shared **collapsible icon rail** (88dp collapsed / 280dp expanded) via `PegasusCollapsibleRail`. Phone layout uses a 4-tab bottom bar plus More hub.

| Group | Android route | Portal |
| --- | --- | --- |
| Primary | `dashboard`, `orders`, `drivers`, `vehicles`, `inventory`, `dispatch`, `analytics`, `treasury`, `staff` | `/`, `/orders`, `/drivers`, `/vehicles`, `/inventory`, `/dispatch`, analytics, treasury, staff |
| Fulfillment | `manifests`, `dispatch_settings`, `fleet_live_map`, `transfer_actions` | manifests, dispatch settings, fleet map, transfers |
| Inventory | `products`, `supply_requests`, `replenishment`, `demand_forecast`, `ops_settings` | products, supply requests, replenishment, forecast, `/settings` |
| Operations | `crm`, `returns`, `payment_config`, `notifications` | retailers, returns, payment config, native inbox |
| Portal only | `portal/setup`, `portal/profile`, `portal/search` | setup wizard, profile, global search (desktop) |

Compact tabs: Dashboard · Orders · Dispatch · More (secondary routes).

Notifications opens the **native** `NotificationInboxScreen`, not a portal handoff.
