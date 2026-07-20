# Desktop store distribution (Microsoft Store + Mac App Store)

Website CDN keys live under `contracts/desktop-updater/`.  
This folder documents **store-channel** env and config for Tauri shells.

## Build

```bash
# Microsoft Store (Windows EXE/MSI + offline WebView2)
NEXT_PUBLIC_DESKTOP_DISTRIBUTION=store \
NEXT_PUBLIC_DESKTOP_STORE_URL="ms-windows-store://pdp/?ProductId=YOUR_ID" \
  pnpm --filter @pegasusx/supplier-portal tauri:build:store:win

# Mac App Store
NEXT_PUBLIC_DESKTOP_DISTRIBUTION=store \
NEXT_PUBLIC_DESKTOP_STORE_URL="https://apps.apple.com/app/idYOUR_ID" \
  pnpm --filter @pegasusx/supplier-portal tauri:build:store:mac
```

## Backend env (API host)

| Variable | Purpose |
|----------|---------|
| `MS_STORE_PRODUCT_ID_SUPPLIER` | Partner Center ProductId → `ms-windows-store://pdp/?ProductId=` |
| `MS_STORE_URL_SUPPLIER` | Full MS Store URL override |
| `MAC_APP_STORE_ID_DESKTOP_SUPPLIER` | Mac App Store numeric id |
| `MAC_APP_STORE_URL_DESKTOP_SUPPLIER` | Full Mac App Store URL |
| `STORE_URL_DESKTOP_SUPPLIER` | Generic desktop store landing (wins first) |

Repeat for `RETAILER`, `WAREHOUSE`, `FACTORY`.

## Client env (store binary)

| Variable | Purpose |
|----------|---------|
| `NEXT_PUBLIC_DESKTOP_DISTRIBUTION=store` | production channel + no CDN OTA |
| `NEXT_PUBLIC_DESKTOP_STORE_URL` | in-app open listing |

## Overlays

- `src-tauri/tauri.microsoftstore.conf.json`
- `src-tauri/tauri.macappstore.conf.json`
- `src-tauri/tauri.store.conf.json`

See `docs/DESKTOP_APP_STORE_DISTRIBUTION.md`.
