# Enterprise website-only mobile updates (native)

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



**Scope:** Users install apps from the **official website** (not App Store / Play).  
**Live:** Supplier + Driver + Payload + Warehouse + Retailer + Factory (Android + iOS).
**Status:** All primary native roles covered for website-only enterprise OTA.

**Desktop (Tauri) shells:** see [`ENTERPRISE_WEBSITE_DESKTOP_UPDATES.md`](./ENTERPRISE_WEBSITE_DESKTOP_UPDATES.md).

## Mental model

| Artifact | Where it lives |
|----------|----------------|
| Source code | Monorepo `pegasusX/apps/supplier-app-*` |
| Built APK / IPA | GCS bucket (CDN) |
| `updater.json` | Same bucket — **this is the URL apps poll** |
| Version policy | Spanner `ClientVersionPolicies` via `GET /v1/platform/client-policy` |

Apps do **not** pull source code from a URL. They fetch a small **manifest JSON**, then download a signed package URL inside it.

## CDN layout

Bucket default: `pegasusx-ssmr-app-updates`  
Override origin: `UPDATES_BASE_URL` (backend + upload script).

```text
{base}/android/supplier/updater.json
{base}/android/supplier/supplier-1.2.0.apk
{base}/ios/supplier/updater.json
{base}/ios/supplier/manifest.plist
{base}/ios/supplier/supplier-1.2.0.ipa
```

### Android `updater.json`

```json
{
  "version_code": 12,
  "version_name": "1.2.0",
  "apk_url": "https://storage.googleapis.com/.../supplier-1.2.0.apk",
  "sha256": "<hex>",
  "notes": "..."
}
```

### iOS `updater.json`

```json
{
  "version_code": 12,
  "version_name": "1.2.0",
  "manifest_url": "https://storage.googleapis.com/.../manifest.plist",
  "notes": "..."
}
```

iOS install uses `itms-services://?action=download-manifest&url=<plist>`. Requires **Apple Enterprise** or equivalent distribution signing — not public App Store sideload.

## Client behavior (supplier)

1. On login / WS reconnect, app calls:

   `GET /v1/platform/client-policy?role=ADMIN&platform=android|ios&version=<semver>&channel=enterprise`

2. Backend returns `outdated`, `force_update`, `update_url` (defaults to CDN `updater.json` if policy row has empty URL).

3. App fetches `update_url` → compares `version_code` → banner **Update** / force dialog.

4. User taps Update:
   - **Android:** DownloadManager → SHA-256 verify → package installer (FileProvider).
   - **iOS:** Open `itms-services` with enterprise plist.

Channel is hard-coded to **`enterprise`** in:

- `supplier-app-android` → `EnterpriseUpdateConfig.CHANNEL`
- `supplier-app-ios` → `EnterpriseUpdateConfig.channel`

## Release steps

### 1. Build

```bash
# Android release APK/AAB (enterprise uses APK for sideload)
# iOS archive → export Enterprise / Ad Hoc IPA + generate OTA plist
```

### 2. Upload

```bash
cd pegasusX
bash scripts/upload_enterprise_mobile_app.sh driver android \
  --apk path/to/supplier.apk \
  --version-code 12 \
  --version-name 1.2.0

bash scripts/upload_supplier_enterprise_mobile.sh ios \
  --ipa path/to/supplier.ipa \
  --plist path/to/manifest.plist \
  --version-code 12 \
  --version-name 1.2.0
```

### 3. Set server policy (optional force)

```http
PUT /v1/platform/client-policy
Authorization: Bearer <ADMIN JWT>
{
  "role": "ADMIN",
  "platform": "android",
  "channel": "enterprise",
  "minimum_version": "1.2.0",
  "recommended_version": "1.2.0",
  "update_url": "https://storage.googleapis.com/pegasusx-ssmr-app-updates/android/supplier/updater.json",
  "force_update": true
}
```

Repeat for `platform=ios`.

### 4. Website download page

Link users to the **same** APK/IPA (or a stable redirect). Running apps still poll `updater.json`, not the HTML page.

## Website vs store (do not mix)

| Build | `channel` | Update mechanism |
|-------|-----------|------------------|
| Website enterprise | `enterprise` | CDN + AutoUpdater |
| Play / App Store | `production` | Open store listing only (no CDN OTA) — see [`NATIVE_APP_STORE_DISTRIBUTION.md`](./NATIVE_APP_STORE_DISTRIBUTION.md) |

## Extending to other roles

Same layout with slugs: `driver`, `retailer`, `warehouse`, `factory`, `payload`.  
Backend already maps roles → slugs in `platform.DefaultEnterpriseManifestURL`.

## Files

| Path | Role |
|------|------|
| `apps/backend-go/platform/enterprise_updates.go` | Default CDN URLs + channel normalize |
| `apps/supplier-app-android/.../AutoUpdater.kt` | Android OTA |
| `apps/supplier-app-ios/.../AutoUpdater.swift` | iOS OTA |
| `scripts/upload_supplier_enterprise_mobile.sh` | Release upload |
| `contracts/enterprise-updates/**` | Example manifests |


## Payload release example

```bash
bash scripts/upload_enterprise_mobile_app.sh payload android \
  --apk path/to/payload.apk --version-code 12 --version-name 1.2.0

bash scripts/upload_enterprise_mobile_app.sh payload ios \
  --ipa path/to/payload.ipa --plist path/to/manifest.plist \
  --version-code 12 --version-name 1.2.0
```

Policy:

```http
PUT /v1/platform/client-policy
{
  "role": "PAYLOAD",
  "platform": "android",
  "channel": "enterprise",
  "minimum_version": "1.2.0",
  "recommended_version": "1.2.0",
  "update_url": "https://storage.googleapis.com/pegasusx-ssmr-app-updates/android/payload/updater.json",
  "force_update": true
}
```

## Warehouse release example

```bash
bash scripts/upload_enterprise_mobile_app.sh warehouse android \
  --apk path/to/warehouse.apk --version-code 12 --version-name 1.2.0

bash scripts/upload_enterprise_mobile_app.sh warehouse ios \
  --ipa path/to/warehouse.ipa --plist path/to/manifest.plist \
  --version-code 12 --version-name 1.2.0
```

Policy:

```http
PUT /v1/platform/client-policy
{
  "role": "WAREHOUSE",
  "platform": "android",
  "channel": "enterprise",
  "minimum_version": "1.2.0",
  "recommended_version": "1.2.0",
  "update_url": "https://storage.googleapis.com/pegasusx-ssmr-app-updates/android/warehouse/updater.json",
  "force_update": true
}
```


## Retailer release example

```bash
bash scripts/upload_enterprise_mobile_app.sh retailer android \
  --apk path/to/retailer.apk --version-code 12 --version-name 1.2.0

bash scripts/upload_enterprise_mobile_app.sh retailer ios \
  --ipa path/to/retailer.ipa --plist path/to/manifest.plist \
  --version-code 12 --version-name 1.2.0
```

Policy:

```http
PUT /v1/platform/client-policy
{
  "role": "RETAILER",
  "platform": "android",
  "channel": "enterprise",
  "minimum_version": "1.2.0",
  "recommended_version": "1.2.0",
  "update_url": "https://storage.googleapis.com/pegasusx-ssmr-app-updates/android/retailer/updater.json",
  "force_update": true
}
```


## Factory release example

```bash
bash scripts/upload_enterprise_mobile_app.sh factory android \
  --apk path/to/factory.apk --version-code 12 --version-name 1.2.0

bash scripts/upload_enterprise_mobile_app.sh factory ios \
  --ipa path/to/factory.ipa --plist path/to/manifest.plist \
  --version-code 12 --version-name 1.2.0
```

Policy:

```http
PUT /v1/platform/client-policy
{
  "role": "FACTORY",
  "platform": "android",
  "channel": "enterprise",
  "minimum_version": "1.2.0",
  "recommended_version": "1.2.0",
  "update_url": "https://storage.googleapis.com/pegasusx-ssmr-app-updates/android/factory/updater.json",
  "force_update": true
}
```
