# Deprecated — use `supplier-portal`

This directory is a retired stub from the Pegasus → pegasusX rename.

**Canonical supplier / admin surface:** [`../supplier-portal`](../supplier-portal)

- Web + Tauri desktop: `pegasusX/apps/supplier-portal`
- Native: `supplier-app-ios`, `supplier-app-android`
- Backend APIs: `/v1/supplier/*` on `backend-go`

**Admin order operations** (assign driver, patch status) live on the supplier portal **Orders** page when signed in with an eligible JWT:

- **Assign** (`POST /v1/orders/{orderID}/assign`): `ADMIN`, `WAREHOUSE_ADMIN`, `FACTORY_ADMIN` — key `adminOrderAssignKey`
- **Status patch** (`PATCH /v1/order/{orderID}/status`): `ADMIN` only on this surface — key `adminOrderStatusPatchKey`

Do not add features here. Reference-only Pegasus UI measurements live under `pegasus/apps/admin-portal` (read-only).
