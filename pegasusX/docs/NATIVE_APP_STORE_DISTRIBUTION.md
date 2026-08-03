# Native App Store / Play distribution

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



**Scope:** Public **Google Play** and **Apple App Store** builds of the six native role apps.  
**Not this doc:** website / enterprise CDN sideload — see [`ENTERPRISE_WEBSITE_MOBILE_UPDATES.md`](./ENTERPRISE_WEBSITE_MOBILE_UPDATES.md).

## Two distribution channels (do not mix)

| | Website enterprise | App Store / Play |
|--|--------------------|------------------|
| Client-policy `channel` | `enterprise` | **`production`** |
| Install source | Official website + GCS APK/IPA | Play / App Store |
| In-app update action | Download CDN package (AutoUpdater) | **Open store listing** |
| Android Gradle flavor | `enterprise` (default) | **`store`** |
| iOS switch | default / omit plist keys | `PXDistributionChannel=production` + `PXEnableCdnOta=false` |

Never enable CDN APK/itms-services OTA inside a store-signed binary.

## Apps & package IDs

| Role | Android `applicationId` | iOS bundle (primary) |
|------|-------------------------|----------------------|
| Supplier | `com.pegasusx.supplier` | `com.pegasusx.supplier` (project-specific) |
| Driver | `com.pegasusx.driver` | `com.pegasusx.driverappios` |
| Retailer | `com.pegasusx.retailer` | project-specific |
| Warehouse | `com.pegasusx.warehouse` | `com.pegasusx.warehouse` |
| Factory | `com.pegasusx.factory` | `com.pegasusx.factory` |
| Payload | `com.pegasus.payload` | `com.pegasus.payload` |

## Android (Play)

### Flavors

Each `*-app-android` app defines:

```text
flavorDimensions = distribution
productFlavors:
  enterprise  → DISTRIBUTION_CHANNEL=enterprise, ENABLE_CDN_OTA=true
  store       → DISTRIBUTION_CHANNEL=production, ENABLE_CDN_OTA=false
                STORE_LISTING_URL=https://play.google.com/store/apps/details?id=<applicationId>
```

### Build store AAB

```bash
cd apps/supplier-app-android
./gradlew :app:bundleStoreRelease

# or all roles:
bash scripts/build_native_android_store.sh
```

Upload the AAB in Play Console. Use Play App Signing.

### Runtime

- Client-policy called with `channel=production` (`EnterpriseUpdateConfig.CHANNEL` from `BuildConfig`).
- `AutoUpdater` **does not** download APKs; **Update** opens the Play listing.
- Backend fills `update_url` with Play details URL when policy row is empty (`DefaultStoreUpdateURL`).

## iOS (App Store)

### Info.plist keys (store archive)

Merge [`contracts/native-store/Info.plist.store.snippet`](../contracts/native-store/Info.plist.store.snippet):

| Key | Store value |
|-----|-------------|
| `PXDistributionChannel` | `production` |
| `PXEnableCdnOta` | `false` |
| `PXAppStoreID` | App Store Connect Apple ID (numeric) |
| `PXAppStoreURL` | optional full `https://apps.apple.com/app/id…` |

Or compile with `-DSTORE_BUILD` (Swift active compilation condition).

### Archive

1. Set store plist keys (or Store scheme with `STORE_BUILD`).
2. Archive → Upload to App Store Connect / TestFlight.
3. After first listing is live, set `APP_STORE_ID_<ROLE>` on the API host so policy `update_url` is filled.

### Runtime

- `EnterpriseUpdateConfig.channel` → `production`.
- `enableCdnOta` → `false` → prompts open App Store, never `itms-services`.

## Backend policy

```http
PUT /v1/platform/client-policy
{
  "role": "DRIVER",
  "platform": "android",
  "channel": "production",
  "minimum_version": "1.2.0",
  "recommended_version": "1.2.0",
  "force_update": false
}
```

Empty `update_url` defaults:

- Android → `https://play.google.com/store/apps/details?id=<package>`
- iOS → `STORE_URL_IOS_<slug>` or `APP_STORE_ID_<slug>` env (no fake URL without id)

Env overrides:

```bash
export STORE_URL_ANDROID_SUPPLIER=https://play.google.com/store/apps/details?id=com.pegasusx.supplier
export APP_STORE_ID_SUPPLIER=1234567890
export STORE_URL_IOS_DRIVER=https://apps.apple.com/app/id...
```

## Checklist before first store submission

- [ ] Privacy policy URL + data safety / App Privacy
- [ ] Store listing screenshots & descriptions per role
- [ ] Signing: Play App Signing / Apple Distribution cert + provisioning
- [ ] Push: FCM + APNs production certs
- [ ] API base URLs production in release `buildTypes`
- [ ] Build **store** flavor / store plist — not enterprise CDN OTA
- [ ] Seed `channel=production` client-policy rows in Spanner
- [ ] Set `APP_STORE_ID_*` after Apple assigns IDs

## Files

| Path | Role |
|------|------|
| `apps/*-android/app/build.gradle.kts` | `enterprise` / `store` flavors |
| `.../EnterpriseUpdateConfig.kt` | channel + CDN flag from BuildConfig |
| `.../AutoUpdater.kt` | store → open Play |
| `.../EnterpriseUpdateConfig.swift` | plist / `STORE_BUILD` |
| `.../AutoUpdater.swift` | store → open App Store |
| `apps/backend-go/platform/enterprise_updates.go` | `DefaultStoreUpdateURL` |
| `scripts/build_native_android_store.sh` | batch AAB |
| `contracts/native-store/Info.plist.store.snippet` | iOS store keys |
