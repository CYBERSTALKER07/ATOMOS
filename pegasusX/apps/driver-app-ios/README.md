# driver-app-ios

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


Canonical **DRIVER** iOS product lives in the nested Xcode project:

**`pegasusX/apps/driver-app-ios/driverappios`** (target `driverappios`)

Full SwiftUI execution client: manifest, map, offload, cash, notifications, authenticated `/v1/ws` on port **8180**.

The thin PX5-A2 duplicate shell that previously lived at this folder root was removed under **PX8-A2** to prevent role-row drift.

## Open the app

```bash
cd pegasusX/apps/driver-app-ios/driverappios
open driverappios.xcodeproj
```

## Build (CI / local)

```bash
cd pegasusX/apps/driver-app-ios/driverappios
xcodebuild -scheme driverappios -destination 'platform=iOS Simulator,name=iPhone 17' CODE_SIGNING_ALLOWED=NO build
```
