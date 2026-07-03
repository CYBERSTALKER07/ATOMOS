# Driver App iOS (pegasusX)

Native SwiftUI driver execution client for the pegasusX local stack.

## Backend

- API base: `http://localhost:8180` (DEBUG; override with scheme env `PEGASUS_DEV_HOST`)
- WebSocket: `/v1/ws` with `Authorization: Bearer <jwt>`
- Demo login: `+998901000066` / PIN `1234` (`DRIVER_DEMO_PHONE`, `DRIVER_DEMO_PIN`, `DRIVER_DEMO_ID`)

## Open in Xcode

```bash
cd pegasusX/apps/driver-app-ios/driverappios
open driverappios.xcodeproj
```

Optional regeneration via XcodeGen:

```bash
brew install xcodegen   # once
xcodegen generate       # if project.yml is added later
```

## pegasusX deltas vs legacy pegasus

| Area | pegasusX |
|------|----------|
| API port | `8180` |
| Keychain service | `com.pegasusx.driver` |
| Command WS | `/v1/ws` (not `/v1/ws/driver`) |
| Telemetry WS | `/v1/ws` + Bearer (not `/ws/telemetry?token=`) |

Fleet orders, delivery mutations, and several order endpoints are scaffolded in `pegasusX/apps/backend-go/driver/mobile_compat.go` for simulator flows; production paths will grow with Spanner-backed handlers.
