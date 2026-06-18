# Supplier App Android (pegasusX)

Native Kotlin/Compose SUPPLIER client — full parity with `supplier-portal` and `supplier-app-ios` on port **8180**.

## Stack

- Jetpack Compose Material 3
- Hilt + Retrofit + kotlinx.serialization
- EncryptedSharedPreferences for JWT (`com.pegasusx.supplier`)
- Hilt ViewModels: onboarding, orders, inventory, treasury

## API (authenticated Bearer)

- Auth: `POST /v1/auth/supplier/register`, `login`, `refresh`
- Onboarding: `POST /v1/supplier/business/setup`, `billing/setup`
- Ops: orders (vet), inventory (PATCH + import wizard), dispatch, manifests, fleet live map
- Finance: ledger, payments, reconciliation, chargebacks, earnings
- Catalog: products CRUD, retailer price overrides
- Analytics: velocity, revenue, demand today/history

## Build

```bash
cd pegasusX/apps/supplier-app-android
cp local.properties.example local.properties   # dev.host=10.0.2.2 for emulator
./gradlew :app:assembleDebug
```

## Navigation

Bottom tabs: **Dashboard**, **Orders**, **Fleet**, **More**.

More hub includes: manifests, dispatch, exceptions, treasury hub, chargebacks, retailer overrides, inventory import, demand history, factories/warehouses, org & fleet, catalog, pricing, operations, and account settings.

**Onboarding:** Login → Register (3-step) → Business setup → Billing gate → Dashboard. No portal handoff required.

Tauri-wrapped `supplier-portal` remains an alternate Android delivery path; this app is the **native** supplier row client.

## Realtime refresh contract

WS events bump `SupplierRealtimeSignals.refreshTick` — screens call `load(silent = true)` via ViewModel collectors or `RealtimeRefreshEffect` (`packages/mobile-android-design`). Never use `key(refreshEpoch)` on routes. See `pegasusX/docs/ROLE_ROW_PARITY_MATRIX.md` § Native realtime refresh contract.
