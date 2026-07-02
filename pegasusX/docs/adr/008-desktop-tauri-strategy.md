# ADR-008: Desktop strategy — Next.js 15 + Tauri 2 role portals

**Status:** Accepted  
**Date:** 2026-07-02  
**Context:** Four operator roles (supplier, retailer, warehouse, factory) need installable desktop shells with offline tolerance, OS integrations (keyring, CSV export, print, deep links), and parity with portal web UIs. Rewriting each role as a fully native desktop app (Electron-from-scratch, WPF, SwiftUI) would duplicate the Next.js surfaces already shared with static export + Tauri.

## Decision

1. **Keep Next.js 15 + Tauri 2** for all four role-row desktops (`supplier-portal`, `retailer-app-desktop`, `warehouse-portal`, `factory-portal`). Web and desktop share one React tree; `TAURI_BUILD=1` static export feeds `frontendDist`.

2. **Shared packages** own cross-app desktop behavior:
   - `@pegasusx/desktop-bridge` — keyring, `isTauri`, CSV/print, deep links
   - `@pegasusx/desktop-cache` — SQLite hydrate for offline shells
   - `@pegasusx/ui-kit/desktop` — offline tray, connectivity, virtualized lists

3. **Add Rust / Tauri plugins** when the capability is OS-bound and cannot be done safely in the WebView:
   - Keyring token storage (`keyring` crate + invoke commands)
   - `plugin-fs` + `plugin-dialog` for native save/export
   - `plugin-sql` for offline cache
   - `plugin-single-instance` + `plugin-deep-link` for dock PCs and notification handoff
   - Updater signing via minisign + GCS manifests (`contracts/desktop-updater/`)

4. **Do not** add new Tauri Android targets. Supplier mobile is **Kotlin + Swift** (`supplier-app-android`, `supplier-app-ios`). Tauri Android in `supplier-portal` is deprecated.

5. **Do not rewrite** a role desktop to a separate native UI unless measured pain (WebView perf ceiling, hardware SDK, regulatory kiosk mode) exceeds the cost of a second client row. Phase 3+ optimizations (virtualized lists, map teardown, polling audit) stay in the shared Next.js layer first.

## When to add a Rust plugin vs stay in TypeScript

| Need | Layer |
|------|--------|
| Large list scroll perf | React (`react-virtuoso` via `@pegasusx/ui-kit/desktop`) |
| File save / print dialog | `desktop-bridge` + Tauri dialog/fs plugins |
| Deep link / second-instance focus | Tauri single-instance + deep-link plugins |
| Offline durable queue | `desktop-cache` + `plugin-sql` |
| Serial/USB scanner SDK, custom drivers | New Rust plugin + thin TS invoke — **not** WebView JS |
| Full offline VRP / map tile cache | Evaluate sidecar or native module; default defer |

## Consequences

- Release train: GSM signing secrets, Windows Authenticode, macOS notarize, GCS updater manifests (`scripts/upload_desktop_updater_manifest.sh`).
- Every desktop feature must trace portal + Tauri capabilities + parity matrix (`docs/ROLE_ROW_PARITY_MATRIX.md`).
- CI: `desktop-windows-build.yml` produces `.msi` artifacts; manual QA via `docs/qa/PX-DESK_MANUAL_QA.md`.

## References

- `context/plan_desktop.md`
- `packages/desktop-bridge/`, `packages/desktop-cache/`
- `contracts/desktop-updater/README.md`
