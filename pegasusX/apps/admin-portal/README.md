# Deprecated — use `supplier-portal`

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


This directory is a **retired discoverability stub** from the Pegasus → pegasusX rename.

**Canonical supplier / admin surface:** [`../supplier-portal`](../supplier-portal)

| Capability | Location |
|------------|----------|
| Web + Tauri desktop | `pegasusX/apps/supplier-portal` |
| Native mobile | `supplier-app-ios`, `supplier-app-android` |
| Backend APIs | `/v1/supplier/*`, admin payment/order routes on `backend-go` |

## Admin order operations

Live on the supplier portal **Orders** page when signed in with an eligible JWT:

| Action | Method / path | Roles |
|--------|---------------|-------|
| Assign driver | `POST /v1/orders/{orderID}/assign` | `ADMIN`, `WAREHOUSE_ADMIN`, `FACTORY_ADMIN` |
| Status patch | `PATCH /v1/order/{orderID}/status` | `ADMIN` |
| Chargeback / ledger | `/v1/payment/*` | `ADMIN` |

Do **not** add product features here. Running `pnpm dev` / `pnpm build` in this folder prints the redirect and exits non-zero (build check exits 0).
