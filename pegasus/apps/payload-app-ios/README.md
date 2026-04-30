# Lab Payload — iPad App

Native iPad SwiftUI client for the **PAYLOAD** role. Functional superset of the Expo `apps/payload-terminal`, optimised for iPad Pro / iPad Air landscape use in the warehouse loading bay (NavigationSplitView, VisionKit DataScannerViewController, Core Haptics, WebSocket real-time, FCM).

Sibling Android tablet app: `apps/payload-app-android`.

## Project generation

This repo stores **sources only**. The `.xcodeproj` is generated reproducibly via [XcodeGen](https://github.com/yonaskolb/XcodeGen) from `project.yml`:

```bash
brew install xcodegen
cd apps/payload-app-ios
xcodegen generate
open payload-app-ios.xcodeproj
```

Sources are not edited inside the `.xcodeproj` — they live in `payload-app-ios/`. Add new files there and re-run `xcodegen generate`. The `.xcodeproj` directory is git-ignored.

## Physical-device dev

The `APIClient` honours `PEGASUS_DEV_HOST` from the scheme environment so a physical iPad on the LAN can reach the dev backend on the Mac:

1. Edit Scheme → Run → Arguments → Environment Variables.
2. Add `PEGASUS_DEV_HOST` = `192.168.1.42` (your Mac's LAN IP).
3. Run on device. Cleartext to LAN is permitted by `Info.plist` `NSAllowsLocalNetworking`.

## Architecture

- **DI:** Manual composition root in `LabPayloadApp.swift` (mirror of `LabDriverApp`).
- **State:** `@Observable` macro (Observation framework) on every ViewModel and TokenStore.
- **Networking:** `URLSession` (no third-party deps).
- **Persistence:** SwiftData (manifest cache + offline queue, Phase 6).
- **Secure store:** Keychain Services (`TokenStore`).
- **WebSocket:** `URLSessionWebSocketTask` with backoff (Phase 6).
- **Push:** Firebase Messaging SDK (Phase 7).
- **UI:** SwiftUI native, SF Symbols, system colours. NavigationSplitView for landscape iPad.

## Phases

| Phase | Status | Description |
|-------|--------|-------------|
| 1 | ✅ | Source tree, `project.yml`, Info.plist, Assets |
| 2 | ✅ | Auth vertical slice (LoginView + TokenStore + APIClient.login) |
| 3 | ⏳ | Truck selection + manifest fetch (NavigationSplitView) |
| 4 | ⏳ | Loading workflow (start → tap-check → seal → 60s countdown → all-sealed) |
| 5 | ⏳ | Exceptions + inject + re-dispatch |
| 6 | ⏳ | WebSocket + notifications + offline queue (SwiftData) |
| 7 | ⏳ | Firebase Messaging push wiring |
| 8 | ⏳ | Tablet polish (Pencil v1.1) — barcode scan deferred, see docs/BARCODE_SCANNING.md |
| 9 | ⏳ | Cross-role gap-hunter sweep |

See `/memories/session/plan.md` for the full 51-feature inventory.
