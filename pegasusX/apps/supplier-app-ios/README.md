# Pegasus Supplier (iOS)

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


Native SwiftUI supplier operations app for **iPhone and iPad**. JWT role `ADMIN`; backend routes under `/v1/supplier/*` and `/v1/auth/supplier/*`.

## Prerequisites

- Xcode 15+
- [XcodeGen](https://github.com/yonaskolb/XcodeGen): `brew install xcodegen` (optional — project checked in)
- pegasusX backend on port **8180**

## Open

```bash
cd pegasusX/apps/supplier-app-ios
open SupplierAppIOS.xcodeproj
```

## Simulator / device API host

- Simulator: `http://localhost:8180`
- Physical device: Xcode scheme → Environment → `PEGASUS_DEV_HOST` = your Mac LAN IP (e.g. `192.168.1.42`)

## Layout

- **Compact (iPhone)**: bottom `TabView` — Dashboard, Orders, Fleet, More
- **Regular (iPad)**: `NavigationSplitView` sidebar with full `SupplierSection` list (ops, intelligence, network, account) and detail pane
- Content uses adaptive grids, `SupplierTheme` tokens, and WS-driven refresh

## Auth & onboarding

1. `POST /v1/auth/supplier/login` or native **Register** (3-step wizard)
2. **Business setup** when `is_registered=false`
3. **Billing gate** when `is_configured=false` (skippable, same as web)
4. `SupplierAdaptiveShell` main ops

Native screens cover order vetting, inventory adjust/import, retailer overrides, chargebacks, business setup, operations, treasury hub, demand history, factories/warehouses browse, and catalog product detail — no portal handoff from More hub.

## Related surfaces

- Web / desktop: `pegasusX/apps/supplier-portal`
- Android: `pegasusX/apps/supplier-app-android`
