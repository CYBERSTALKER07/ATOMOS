# Desktop Microsoft Store & Mac App Store

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



**Scope:** Public store builds of the four Tauri role portals.  
**Not this doc:** website CDN sideload OTA — see [`ENTERPRISE_WEBSITE_DESKTOP_UPDATES.md`](./ENTERPRISE_WEBSITE_DESKTOP_UPDATES.md).

## Two distribution channels (do not mix)

| | Website enterprise | Microsoft Store / Mac App Store |
|--|--------------------|--------------------------------|
| Build env | `NEXT_PUBLIC_DESKTOP_DISTRIBUTION=enterprise` (default) | **`store`** |
| Client-policy `channel` | `enterprise` | **`production`** |
| Client-policy `platform` | `desktop` | `desktop` |
| Update mechanism | Tauri `plugins.updater` + GCS | **Store auto-update** + open listing |
| Tauri config | `tauri.conf.json` | `tauri.store.conf.json` / `tauri.microsoftstore.conf.json` / `tauri.macappstore.conf.json` |
| `createUpdaterArtifacts` | `true` | **`false`** (empty updater endpoints) |

Never ship a store-signed binary with website CDN OTA enabled for the same user population.

## Apps

| App | Filter | Bundle id (default) |
|-----|--------|---------------------|
| Supplier Portal | `@pegasusx/supplier-portal` | `com.pegasusx.supplier` |
| Retailer Desktop | `@pegasusx/retailer-app-desktop` | `com.pegasus.retailer` |
| Warehouse Portal | `@pegasusx/warehouse-portal` | `com.pegasusx.warehouse` |
| Factory Portal | `@pegasusx/factory-portal` | `com.pegasusx.factory` |

## Runtime wiring

`@pegasusx/desktop-bridge`:

| API | Behavior |
|-----|----------|
| `desktopDistribution()` | `enterprise` \| `store` from `NEXT_PUBLIC_DESKTOP_DISTRIBUTION` |
| `desktopClientPolicyContext()` | store → `{ platform: desktop, channel: production }` |
| `isDesktopCdnOtaEnabled()` | `true` only for enterprise Tauri |
| `checkDesktopUpdate()` | no-op on store builds |
| `openDesktopStoreListing()` | opens `NEXT_PUBLIC_DESKTOP_STORE_URL` via shell plugin |

`ClientPolicyBanner` already uses `desktopClientPolicyContext()` — store builds poll production policy and show `update_url` (store listing).

## Microsoft Store (Windows)

Per [Tauri Microsoft Store guide](https://v2.tauri.app/distribute/microsoft-store/): Partner Center product type **EXE or MSI**, offline WebView2, silent install.

### Config overlay

`src-tauri/tauri.microsoftstore.conf.json`:

- `bundle.windows.webviewInstallMode.type = offlineInstaller`
- `bundle.createUpdaterArtifacts = false`
- `plugins.updater.endpoints = []`
- `bundle.publisher` set (must not equal product name alone)

### Build

```bash
export NEXT_PUBLIC_DESKTOP_DISTRIBUTION=store
# Optional listing deep link for in-app "Open Store":
export NEXT_PUBLIC_DESKTOP_STORE_URL="ms-windows-store://pdp/?ProductId=YOUR_PRODUCT_ID"

cd pegasusX
pnpm --filter @pegasusx/supplier-portal tauri:build:store:win
```

Silent install arg for Partner Center (NSIS): `/S`  
MSI: `/quiet`

### Upload

1. Reserve name in Partner Center → Apps and Games → EXE or MSI.  
2. Upload signed installer (Authenticode).  
3. Set silent install parameters.  
4. Link package URL if using “package from URL” flow.

### Backend policy URL

```bash
export MS_STORE_PRODUCT_ID_SUPPLIER=9NXXXXXXXXXX
# or full URL:
export MS_STORE_URL_SUPPLIER="ms-windows-store://pdp/?ProductId=9NXXXXXXXXXX"
export STORE_URL_DESKTOP_SUPPLIER="https://apps.microsoft.com/detail/9NXXXXXXXXXX"
```

`GET /v1/platform/client-policy?role=ADMIN&platform=desktop&channel=production` fills `update_url` via `DefaultStoreUpdateURL`.

## Mac App Store

Per [Tauri App Store guide](https://v2.tauri.app/distribute/app-store/): Mac App Store Connect provisioning, sandboxed `.pkg` upload.

### Config overlay

`src-tauri/tauri.macappstore.conf.json`:

- No updater artifacts / empty endpoints  
- `bundle.publisher`  
- Harden sandbox entitlements in Xcode/signing before first submission (see Tauri macOS distribution docs)

### Build

```bash
export NEXT_PUBLIC_DESKTOP_DISTRIBUTION=store
export NEXT_PUBLIC_DESKTOP_STORE_URL="https://apps.apple.com/app/idYOUR_MAC_APP_ID"

pnpm --filter @pegasusx/supplier-portal tauri:build:store:mac
```

Package/sign for Mac App Store with your Apple Distribution identity, then upload via Transporter / `xcrun altool` / Xcode Organizer as required by your account.

### Backend

```bash
export MAC_APP_STORE_ID_DESKTOP_SUPPLIER=1234567890
# or
export MAC_APP_STORE_URL_DESKTOP_SUPPLIER=https://apps.apple.com/app/id1234567890
```

## Generic store scripts

| Script | Purpose |
|--------|---------|
| `tauri:build:store` | store distribution, merge `tauri.store.conf.json` |
| `tauri:build:store:win` | MS Store offline WebView2 overlay |
| `tauri:build:store:mac` | Mac App Store overlay |
| `tauri:build:enterprise:win` | website CDN Windows |
| `tauri:build:enterprise:mac` | website CDN macOS |

## Policy seed example

```http
PUT /v1/platform/client-policy
{
  "role": "ADMIN",
  "platform": "desktop",
  "channel": "production",
  "minimum_version": "0.2.0",
  "recommended_version": "0.2.0",
  "force_update": false
}
```

Empty `update_url` → `DefaultStoreUpdateURL` (MS Store product id or Mac App Store id env).

## Checklist

- [ ] Partner Center / App Store Connect apps reserved per role  
- [ ] `NEXT_PUBLIC_DESKTOP_DISTRIBUTION=store` on store CI  
- [ ] `NEXT_PUBLIC_DESKTOP_STORE_URL` set per OS build  
- [ ] Authenticode (Windows) / Apple Distribution + notarization as required  
- [ ] Silent install params for MS Store  
- [ ] Spanner `channel=production` desktop policy rows  
- [ ] `MS_STORE_PRODUCT_ID_*` / `MAC_APP_STORE_ID_DESKTOP_*` on API  
- [ ] Do **not** use enterprise CDN upload scripts for store-only SKUs  

## Files

| Path | Role |
|------|------|
| `packages/desktop-bridge/updater.ts` | distribution mode + store open |
| `apps/*/src-tauri/tauri.microsoftstore.conf.json` | MS Store WebView2 offline |
| `apps/*/src-tauri/tauri.macappstore.conf.json` | Mac App Store overlay |
| `apps/*/src-tauri/tauri.store.conf.json` | generic store (no CDN updater) |
| `apps/*/lib/desktop-updater.tsx` | enterprise CDN toast only |
| `apps/backend-go/platform/enterprise_updates.go` | desktop store URLs |
| `contracts/desktop-store/README.md` | keys & env cheat sheet |
