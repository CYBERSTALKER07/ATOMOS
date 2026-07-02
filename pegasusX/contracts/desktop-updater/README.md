# Desktop Tauri updater keys

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

## Updater CDN layout (GCS)

Bucket: `pegasusx-ssmr-app-updates` (see `infra/terraform/main.tf`)

| App | Manifest path |
|-----|----------------|
| Retailer desktop | `retailer-desktop/{{target}}/{{arch}}/updater.json` |
| Supplier desktop | `supplier-desktop/{{target}}/{{arch}}/updater.json` |
| Warehouse desktop | `warehouse-desktop/{{target}}/{{arch}}/updater.json` |
| Factory desktop | `factory-desktop/{{target}}/{{arch}}/updater.json` |

Upload signed bundles + manifests after each release train cut.

## Production release train

1. `terraform apply` provisions GSM secrets (`tauri_signing_private_key`, `windows_codesign_pfx`, …) and `app_updates` bucket.
2. Fill `contracts/desktop-updater/.env.release.example` → `.env.staging.secrets`.
3. `bash scripts/sync_desktop_release_secrets.sh`
4. GitHub Actions: `desktop-windows-build` (set `TAURI_*` + `WINDOWS_CODESIGN_*` secrets).
5. `bash scripts/upload_desktop_updater_manifest.sh <app> <version> <bundle-path>`
6. Manual QA: [`docs/qa/PX-DESK_MANUAL_QA.md`](../../docs/qa/PX-DESK_MANUAL_QA.md)
