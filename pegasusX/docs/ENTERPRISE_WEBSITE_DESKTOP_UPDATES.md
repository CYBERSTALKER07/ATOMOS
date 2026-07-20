# Enterprise website-only desktop updates (Tauri)

**Scope:** Users install desktop apps from the **official website / CDN** (`channel=enterprise`).  
**Store builds (separate SKU):** Microsoft Store / Mac App Store — see [`DESKTOP_APP_STORE_DISTRIBUTION.md`](./DESKTOP_APP_STORE_DISTRIBUTION.md).  
**Live shells:** Supplier Portal · Retailer Desktop · Warehouse Portal · Factory Portal.

## Mental model

| Artifact | Where it lives |
|----------|----------------|
| Source | `pegasusX/apps/{supplier-portal,retailer-app-desktop,warehouse-portal,factory-portal}` |
| Built installer / updater artifact | GCS bucket (CDN) |
| `updater.json` | Same bucket — **Tauri polls this URL** |
| Minisign pubkey | `plugins.updater.pubkey` in each `tauri.conf.json` |
| Version policy (optional force) | Spanner `ClientVersionPolicies` via `GET /v1/platform/client-policy` |

Apps do **not** pull source from a URL. They fetch a signed **static JSON** manifest, download the package URL inside it, verify minisign, install, and relaunch.

## CDN layout (Tauri 2)

Bucket default: `pegasusx-ssmr-app-updates`  
Override: `DESKTOP_UPDATES_GCS_BUCKET` / `UPDATES_BASE_URL`.

Endpoint template in each `tauri.conf.json`:

```text
https://storage.googleapis.com/pegasusx-ssmr-app-updates/{slug}/{{target}}/{{arch}}/updater.json
```

| App | `slug` |
|-----|--------|
| Supplier Portal | `supplier-desktop` |
| Retailer Desktop | `retailer-desktop` |
| Warehouse Portal | `warehouse-desktop` |
| Factory Portal | `factory-desktop` |

`{{target}}` ∈ `windows` · `darwin` · `linux`  
`{{arch}}` ∈ `x86_64` · `aarch64` · …

Example paths:

```text
{base}/supplier-desktop/windows/x86_64/updater.json
{base}/retailer-desktop/darwin/aarch64/updater.json
```

### Static `updater.json` (Tauri 2)

```json
{
  "version": "0.1.1",
  "notes": "Desktop enterprise release 0.1.1",
  "pub_date": "2026-07-18T12:00:00Z",
  "platforms": {
    "windows-x86_64": {
      "signature": "<minisign signature body>",
      "url": "https://storage.googleapis.com/.../MyApp-setup.exe"
    }
  }
}
```

Platform keys are `{os}-{arch}` (e.g. `windows-x86_64`, `darwin-aarch64`).

## Client behavior

1. **Tauri plugin** (`tauri-plugin-updater`) reads `plugins.updater.endpoints` + `pubkey`.
2. On shell start, `EnterpriseDesktopUpdateBootstrap` calls `checkDesktopUpdate()` from `@pegasusx/desktop-bridge`.
3. If a newer signed version exists, a toast banner offers **Update now** → `downloadAndInstall` → `relaunch`.
4. Separately, `ClientPolicyBanner` calls:

   `GET /v1/platform/client-policy?role=…&platform=desktop&version=…&channel=enterprise`

   Backend fills `update_url` from `DefaultEnterpriseManifestURL` when the policy row is empty (defaults to Windows x86_64 path for policy links; the shell still uses its OS/arch endpoint).

## Distribution rule (website enterprise SKU)

| Concern | Value |
|---------|--------|
| Install source | Official website download page / GCS CDN |
| Build env | `NEXT_PUBLIC_DESKTOP_DISTRIBUTION=enterprise` (default) |
| Client-policy `channel` | **`enterprise`** via `desktopClientPolicyContext()` |
| Client-policy `platform` | **`desktop`** inside Tauri |
| Update mechanism | Signed CDN + Tauri `plugins.updater` (minisign) |

Do **not** mix this SKU with Microsoft Store / Mac App Store binaries. Store SKUs use `NEXT_PUBLIC_DESKTOP_DISTRIBUTION=store` and `channel=production` — see [`DESKTOP_APP_STORE_DISTRIBUTION.md`](./DESKTOP_APP_STORE_DISTRIBUTION.md).

## Release steps

### 1. Sign + build

```bash
export TAURI_SIGNING_PRIVATE_KEY_PATH=contracts/desktop-updater/dev.key   # prod: GSM secret
# optional: bash scripts/apply_desktop_updater_pubkey.sh

cd pegasusX
pnpm --filter @pegasusx/supplier-portal tauri:build:win
# or mac: tauri:build:mac
```

Ensure `bundle.createUpdaterArtifacts: true` (already set).

### 2. Upload

```bash
bash scripts/upload_desktop_updater_manifest.sh supplier-portal 0.1.1 \
  'apps/supplier-portal/src-tauri/target/x86_64-pc-windows-msvc/release/bundle/nsis/*.exe' \
  windows x86_64

bash scripts/upload_desktop_updater_manifest.sh retailer-app-desktop 0.1.1 \
  path/to/*.app.tar.gz darwin aarch64
```

### 3. Optional force policy

```http
PUT /v1/platform/client-policy
Authorization: Bearer <ADMIN JWT>
{
  "role": "ADMIN",
  "platform": "desktop",
  "channel": "enterprise",
  "minimum_version": "0.1.1",
  "recommended_version": "0.1.1",
  "update_url": "https://storage.googleapis.com/pegasusx-ssmr-app-updates/supplier-desktop/windows/x86_64/updater.json",
  "force_update": true
}
```

Repeat per role: `RETAILER`, `WAREHOUSE`, `FACTORY`.

### 4. Validate wiring

```bash
bash scripts/validate_desktop_updater.sh
```

## Files

| Path | Role |
|------|------|
| `apps/*/src-tauri/tauri.conf.json` | `plugins.updater` endpoints + pubkey |
| `apps/*/src-tauri/Cargo.toml` | `tauri-plugin-updater` + process |
| `apps/*/src-tauri/src/lib.rs` | Plugin registration |
| `apps/*/src-tauri/capabilities/default.json` | `updater:default` |
| `apps/*/lib/desktop-updater.tsx` | UI bootstrap |
| `packages/desktop-bridge/updater.ts` | `check` / `install` / relaunch |
| `apps/backend-go/platform/enterprise_updates.go` | Default desktop CDN URLs |
| `scripts/upload_desktop_updater_manifest.sh` | Sign + upload |
| `contracts/desktop-updater/` | Dev minisign keys |

## Keys

See `contracts/desktop-updater/README.md`. Production private key lives in GSM; never ship the private key in the client — only the **public** key is in `tauri.conf.json`.
