# driver-app-ios (deprecated)

**Do not build or extend this folder.**

The canonical DRIVER iOS product app is **`pegasusX/apps/driverappios`** (full SwiftUI execution client: manifest, map, offload, cash, notifications, authenticated `/v1/ws` on port **8180**).

This path previously held a thin PX5-A2 live-ops shell (`DriverAppIOS` XcodeGen target). That duplicate was removed under **PX8-A2** to prevent role-row drift. Use `driverappios` for all new driver iOS work.

## Open the real app

```bash
cd pegasusX/apps/driverappios
open driverappios.xcodeproj
```
