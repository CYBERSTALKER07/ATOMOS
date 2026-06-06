# Supplier App Android (pegasusX)

Native Kotlin/Compose SUPPLIER client — mirrors `supplier-app-ios` and the supplier portal API on port **8180**.

## Stack

- Jetpack Compose Material 3
- Hilt + Retrofit + kotlinx.serialization
- EncryptedSharedPreferences for JWT (`com.pegasusx.supplier`)

## API (authenticated Bearer)

- `POST /v1/auth/supplier/login`
- `GET /v1/supplier/dashboard`, `/profile`, `/orders`, `/fleet/drivers`, `/fleet/vehicles`, `/inventory`, `/earnings`
- `POST /v1/supplier/billing/setup`

## Build

```bash
cd pegasusX/apps/supplier-app-android
cp local.properties.example local.properties   # dev.host=10.0.2.2 for emulator
./gradlew :app:assembleDebug
```

## Navigation

Bottom tabs: Dashboard, Orders, Fleet, More (inventory, earnings, profile, billing, sign out). Unconfigured suppliers are routed to billing setup after login (skippable, same as iOS/web).

Tauri-wrapped `supplier-portal` remains an alternate Android delivery path; this app is the **native** supplier row client.
