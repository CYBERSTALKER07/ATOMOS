# Pegasus X — Warehouse Admin (iOS)

Native SwiftUI client wired to `pegasusX/apps/backend-go` and parity with `warehouse-portal`.

## Run

Tablet (regular width) uses the shared **collapsible sidebar** from `packages/mobile-ios-design` (88pt icon rail ↔ 280pt labeled drawer). iPhone uses a 4-tab shell plus More hub.

```bash
cd pegasusX/apps/warehouse-app-ios
xcodegen generate
open WarehouseAppIOS.xcodeproj
```

Backend (SSMR): `cd pegasusX && make test-ssmr-infra` or run backend on port **8180**.

## Auth (demo)

| Field | Default |
| --- | --- |
| Phone | `+998901000088` |
| PIN | `1234` |

Override via `WAREHOUSE_DEMO_PHONE`, `WAREHOUSE_DEMO_PIN`, `WAREHOUSE_DEMO_ID`.

## Simulator networking

Set `PEGASUS_DEV_HOST` to your Mac LAN IP when testing on a physical device.

## Navigation map (`WarehouseSection`)

| Group | iOS section | Portal |
| --- | --- | --- |
| Primary | Dashboard, Orders, Drivers, Trucks, Inventory, Dispatch, Analytics, Treasury, Staff | matching portal routes |
| Fulfillment | Manifests, Dispatch settings, Live fleet, Transfer actions | dispatch + fleet |
| Inventory | Products, Supply requests, Replenishment, Demand forecast, **Ops settings** | `/inventory`, supply, replenishment, `/settings` |
| Operations | Retailers, Returns, Payment config, **Notifications** | CRM, returns, payment config, native inbox |
| Portal only | Warehouse setup, Profile, Global search | portal handoffs |

Compact tabs: Dashboard · Orders · Dispatch · More.

Notifications opens **NotificationInboxView** (native), not a portal handoff.

WebSocket: `GET /v1/ws?token=…` (warehouse-scoped rooms).

## Realtime refresh contract

Use `load(silent:)` + `.silentRealtimeRefresh` (`packages/mobile-ios-design`) on list screens. No tab/shell `.id(refreshEpoch)` remounts. Full-screen loading only when `loading && items.isEmpty`. See `pegasusX/docs/ROLE_ROW_PARITY_MATRIX.md`.
