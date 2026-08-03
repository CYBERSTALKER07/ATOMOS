# Desktop Tauri updater keys

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



**Distribution:** website / GCS CDN only. These keys sign **sideloaded** enterprise installers — not Microsoft Store or Mac App Store packages.

## Dev / CI (committed)

- **`dev.pub`** — minisign public key used for local builds and unsigned CI Windows artifacts.
- **`dev.key`** — private key for signing **dev-channel** update bundles only. Gitignored; regenerate locally with:

```bash
cd pegasusX
CI=1 pnpm --filter @pegasusx/retailer-app-desktop exec tauri signer generate --ci \
  -w contracts/desktop-updater/dev.key -f
```

## Production

1. Generate a production keypair (password-protected) and store the private key in GSM as `PEGASUSX_TAURI_SIGNING_PRIVATE_KEY`.
2. Set `TAURI_UPDATER_PUBKEY` in the release pipeline from the production `.pub` file.
3. Run `bash scripts/apply_desktop_updater_pubkey.sh` before `tauri build`.
4. Sign update artifacts with `TAURI_SIGNING_PRIVATE_KEY` (or `TAURI_SIGNING_PRIVATE_KEY_PATH` + password).

## Updater CDN layout (GCS) — Tauri 2

Bucket: `pegasusx-ssmr-app-updates` (see `infra/terraform/main.tf`)

Endpoint template: `{slug}/{{target}}/{{arch}}/updater.json` where `target` is `windows|darwin|linux` and `arch` is `x86_64|aarch64|…`.

| App | Manifest path example |
|-----|----------------|
| Retailer desktop | `retailer-desktop/windows/x86_64/updater.json` |
| Supplier desktop | `supplier-desktop/windows/x86_64/updater.json` |
| Warehouse desktop | `warehouse-desktop/darwin/aarch64/updater.json` |
| Factory desktop | `factory-desktop/windows/x86_64/updater.json` |

Static JSON platform keys are `{os}-{arch}` (e.g. `windows-x86_64`).  
Config lives under **`plugins.updater`** (not legacy `app.updater`).

Upload signed bundles + manifests after each release train cut:

```bash
bash scripts/upload_desktop_updater_manifest.sh supplier-portal 0.1.1 \
  path/to/installer.exe windows x86_64
```

## Production release train

1. `terraform apply` provisions GSM secrets (`tauri_signing_private_key`, `windows_codesign_pfx`, …) and `app_updates` bucket.
2. Fill `contracts/desktop-updater/.env.release.example` → `.env.staging.secrets`.
3. `bash scripts/sync_desktop_release_secrets.sh`
4. GitHub Actions: `PegasusX Desktop Build (Windows + Linux)` (`.github/workflows/pegasusx-desktop-build.yml`; set `TAURI_*` + `WINDOWS_CODESIGN_*` secrets).
5. `bash scripts/upload_desktop_updater_manifest.sh <app> <version> <bundle-path>`
6. Manual QA: [`docs/qa/PX-DESK_MANUAL_QA.md`](../../docs/qa/PX-DESK_MANUAL_QA.md)
