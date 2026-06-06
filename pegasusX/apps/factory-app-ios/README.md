# factory-app-ios

SwiftUI **iPad-first** factory admin app for pegasusX. Parity with `pegasus/apps/factory-app-ios`.

## Stack

- SwiftUI, iOS 17+
- Keychain JWT storage
- WebSocket: `GET /v1/ws?token=…` (factory + supplier rooms)

## Generate Xcode project

```bash
cd pegasusX/apps/factory-app-ios
brew install xcodegen   # if needed
xcodegen generate
open FactoryAppIOS.xcodeproj
```

## Local dev

1. Start pegasusX backend on **port 8180**.
2. Simulator uses `http://localhost:8180`.
3. Physical device: set scheme env `PEGASUS_DEV_HOST` to your Mac LAN IP.

Demo login (`FACTORY_DEMO_*` env or defaults):

- Phone: `+998901000099`
- Password/PIN: `1234`

## Surfaces

Dashboard, transfers (+ detail / transitions), loading bay, dispatch, supply requests, manifests (payload override), fleet, staff, insights.

## Role sync

Ship with **factory-app-android**, **factory-portal**, and `pegasusX/apps/backend-go/factory` contracts.
