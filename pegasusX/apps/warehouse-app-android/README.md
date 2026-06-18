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

## Realtime refresh contract

Native clients do **not** consume Kafka. Operational events flow: backend Kafka → WebSocket hub → `WarehouseRealtimeSignals.refreshTick` → per-screen `load(silent = true)`.

- **No destructive remounts** — never wrap `NavHost` or routes in `key(refreshEpoch)`.
- **Stale-while-revalidate** — `load(silent: true)` skips `loading = true` when list data already exists.
- **Full-screen loading** — only on cold start (`loading && items.isEmpty()`); use `com.pegasus.design.showFullScreenLoading`.
- **Shared helper** — `RealtimeRefreshEffect(refreshTick) { silent -> load(silent) }` in `packages/mobile-android-design`.

### Manual test matrix

| Scenario | Expected |
| --- | --- |
| Rapid tab switch | No flash to empty/loading |
| Background 10s → foreground | Data refreshes without full skeleton |
| WS event (place order) | List updates in place |
| Pull-to-refresh | Normal loading indicator OK |
| Cold first visit | Full loading skeleton OK |

