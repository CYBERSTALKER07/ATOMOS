# retailer-app-android

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


Kotlin + Jetpack Compose Material 3 retailer client for pegasusX. Parity target: desktop + iOS (single-tenant retailer row).

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

## Canonical navigation map

| Destination | Route / entry | Desktop equivalent |
|-------------|---------------|-------------------|
| Dashboard | `HOME` tab | `/dashboard` |
| Orders | `ORDERS` tab | `/orders` |
| Tracking | Deliveries → Map (`DELIVERIES`, tab 0) | `/tracking` |
| Dock | Deliveries → Dock (`DOCK`, tab 1) | `/dock` |
| Catalog | `CATALOG` tab | `/catalog` |
| Procurement | Sidebar `PROCUREMENT` | `/procurement` |
| Insights / Analytics | Sidebar `ANALYTICS` | `/insights` |
| Suppliers | `SUPPLIERS` | catalog / procurement |
| Auto-order | Sidebar `AUTO_ORDER` | `/settings` (subsection) |
| Future demand | Sidebar `FUTURE_DEMAND` | dashboard + insights |
| Notifications | Inbox overlay | `/notifications` |
| Settings / Profile | Profile stack | `/settings`, `/settings/cards`, `/settings/family` |

Dock deep-link: sidebar `DOCK` → `DeliveriesHubScreen(initialTabIndex = 1)`.

## Notes

- Post-register setup: `AuthViewModel` posts `POST /v1/retailer/setup` after registration (best-effort).
- `cashCheckout` in `NavigationViewModel` is reserved for cash-at-order experiments; delivery cash confirmation uses `confirmCash` at handoff.
- Pending checkout replay: `PendingOrderSyncWorker` enqueued on WebSocket reconnect via `NavigationViewModel`.
