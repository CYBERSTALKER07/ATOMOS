# Pegasus Payload — Android Tablet App

Native Android tablet client for the **PAYLOAD** role. Functional superset of the Expo `apps/payload-terminal`, optimised for the warehouse loading bay (10-inch+ landscape tablets, M3 Adaptive layouts, ML Kit barcode scan, FCM, WebSocket real-time).

Sibling iPad app: `apps/payload-app-ios`.

## Setup

1. Install Android Studio Ladybug+ (or equivalent CLI: SDK 35, build-tools 35).
2. Copy `local.properties.example` → `local.properties`.
3. Set `sdk.dir` to your Android SDK path.
4. Set `dev.host` to your Mac's LAN IP for physical-tablet dev (or leave `10.0.2.2` for emulator).
5. Drop `google-services.json` into `app/` once Firebase project is provisioned (Phase 7).

## Build

```bash
cd apps/payload-app-android
./gradlew :app:assembleDebug
```

Install on a connected tablet:

```bash
./gradlew :app:installDebug
```

## Architecture

- **DI:** Hilt (`com.pegasus.payload.di.NetworkModule`).
- **Networking:** Retrofit 2.11 + OkHttp 4.12 + kotlinx.serialization.
- **Persistence:** Room (manifest cache + offline action queue, Phase 6).
- **Secure store:** EncryptedSharedPreferences (`SecureStore`).
- **WebSocket:** OkHttp `WebSocket` listener with backoff (Phase 6).
- **Real-time push:** Firebase Cloud Messaging (Phase 7).
- **Theming:** Material 3 + Dynamic Color (Android 12+).

## Phases

| Phase | Status | Description |
|-------|--------|-------------|
| 1 | ✅ | Gradle scaffold, manifest, theme, Hilt root |
| 2 | ✅ | Auth vertical slice (login + session restore) |
| 3 | ⏳ | Truck selection + manifest fetch (master-detail) |
| 4 | ⏳ | Loading workflow (start → tap-check → seal → 60s countdown → all-sealed) |
| 5 | ⏳ | Exceptions + inject + re-dispatch |
| 6 | ⏳ | WebSocket + notifications + offline queue |
| 7 | ⏳ | FCM push wiring |
| 8 | ⏳ | Tablet polish (barcode scan deferred — see docs/BARCODE_SCANNING.md) |
| 9 | ⏳ | Cross-role gap-hunter sweep |

See `/memories/session/plan.md` for the full 51-feature inventory.
