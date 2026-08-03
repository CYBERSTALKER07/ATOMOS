# retailer-app-ios

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


SwiftUI retailer client for pegasusX. Parity target: desktop + Android (single-tenant retailer row).

## Local dev

1. Backend on **port 8180**.
2. Simulator uses `http://localhost:8180` (DEBUG).
3. Device: set scheme env `LAB_DEV_HOST` to your Mac LAN IP (e.g. `192.168.1.42`).

Demo login: `+998901000077` / `1234`

## Build

```bash
cd pegasusX/apps/retailer-app-ios/retailerapp
xcodebuild -scheme reatilerapp build
```

WebSocket path: `/v1/ws`

## Canonical navigation map

See `Layout/RetailerSection.swift` for the full enum. Summary:

| Destination | iOS entry | Desktop |
|-------------|-----------|---------|
| Dashboard | Home tab / sidebar | `/dashboard` |
| Orders | Orders tab | `/orders` |
| Tracking | Deliveries → Map | `/tracking` |
| Dock | Deliveries → Dock (`DeliveriesHubTab.dock`) | `/dock` |
| Catalog | Catalog tab (+ `SearchView` sheet) | `/catalog` |
| Procurement | Sidebar sheet | `/procurement` |
| Insights | Sidebar sheet | `/insights` |
| Suppliers | Suppliers tab | catalog |
| Auto-order | Sidebar sheet | `/settings` |
| Future demand | Sidebar sheet | dashboard |
| Notifications | Inbox sheet | `/notifications` |
| Settings | Profile tab | `/settings` |

Post-login gate: when JWT `is_configured=false`, `SetupView` runs before `ContentView`.

Pending checkout replay: `PendingOrderReplayer` on WS reconnect in `ContentView`.
