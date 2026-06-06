# retailer-app-ios

SwiftUI retailer client for pegasusX. Parity target: `pegasus/apps/retailer-app-ios` + Android + desktop.

## Local dev

1. Backend on **port 8180**.
2. Simulator uses `http://localhost:8180` (DEBUG).
3. Device: set scheme env `LAB_DEV_HOST` to your Mac LAN IP (e.g. `192.168.1.42`).

Demo login: `+998901000077` / `1234`

## Build

Open `retailerapp/reatilerapp.xcodeproj` in Xcode (or regenerate with XcodeGen if `project.yml` is added).

WebSocket path: `/v1/ws`
