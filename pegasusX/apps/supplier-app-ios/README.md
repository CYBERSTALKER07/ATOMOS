# Pegasus Supplier (iOS)

Native SwiftUI supplier operations app for **iPhone and iPad**. JWT role `ADMIN`; backend routes under `/v1/supplier/*` and `/v1/auth/supplier/*`.

## Prerequisites

- Xcode 15+
- [XcodeGen](https://github.com/yonaskolb/XcodeGen): `brew install xcodegen`
- pegasusX backend on port **8180**

## Generate & open

```bash
cd pegasusX/apps/supplier-app-ios
xcodegen generate
open SupplierAppIOS.xcodeproj
```

## Simulator / device API host

- Simulator: `http://localhost:8180`
- Physical device: Xcode scheme → Environment → `PEGASUS_DEV_HOST` = your Mac LAN IP (e.g. `192.168.1.42`)

## Layout

- **Compact (iPhone)**: bottom `TabView` — Dashboard, Orders, Fleet, More
- **Regular (iPad)**: `NavigationSplitView` sidebar with full section list and detail pane
- Content uses adaptive grids and a max readable width on large screens

## Auth

Login: `POST /v1/auth/supplier/login` with `{ phone, password }`. Response includes `token` (Bearer) and `is_configured`. Protected routes accept Bearer via `CookieAuth` → `attachSessionClaims`.

## Related surfaces

- Web: `pegasusX/apps/supplier-portal`
- Android (planned): `pegasusX/apps/supplier-app-android`
- Desktop (planned): native supplier desktop target
