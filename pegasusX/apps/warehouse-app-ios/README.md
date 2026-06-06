# Pegasus X — Warehouse Admin (iOS)

Native SwiftUI clone of `pegasus/apps/warehouse-portal` wired to `pegasusX/apps/backend-go`.

## Run

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

## Surfaces

| Tab | Portal routes |
| --- | --- |
| Dashboard | `/` |
| Orders | `/orders` |
| Drivers / Vehicles | `/drivers`, `/vehicles` |
| Inventory | `/inventory` |
| Dispatch | `/dispatch`, locks, supply (embedded) |
| Analytics / Treasury / Staff | matching portal |
| **More** | Manifests, Products, Supply Requests, Demand Forecast, CRM, Returns |

WebSocket: `GET /v1/ws?token=…` (warehouse-scoped rooms).
