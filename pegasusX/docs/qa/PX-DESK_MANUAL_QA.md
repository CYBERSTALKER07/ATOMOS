# PX-DESK manual QA — Windows desktop shells

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



Last updated: 2026-07-02. Canonical tree: `pegasusX/`.

Linked from [`PX12_MANUAL_QA_RUNBOOK.md`](./PX12_MANUAL_QA_RUNBOOK.md) (desktop section).

## Prerequisites

- Clean Windows 11 VM with WebView2 runtime
- PegasusX staging API reachable (`PUBLIC_BASE_URL`)
- Optional: USB barcode wedge scanner (keyboard emulation)

## PX-DESK-0 — Install & update

| Step | Action | Pass |
|------|--------|------|
| 0.1 | Install `.msi` from `desktop-windows-build` artifact or local `tauri build` | App launches, login works |
| 0.2 | Verify updater manifest: `curl https://storage.googleapis.com/pegasusx-ssmr-app-updates/retailer-desktop/x86_64-pc-windows-msvc/x86_64/updater.json` | JSON valid, signature present |
| 0.3 | Bump version, upload via `scripts/upload_desktop_updater_manifest.sh`, in-app update prompt | Update installs without reinstall |
| 0.4 | Authenticode: `Get-AuthenticodeSignature` on `.exe`/`.msi` | Valid, timestamped (release builds only) |

## PX-DESK-2C — Warehouse returns barcode wedge (WebView2)

**Page:** warehouse-portal → **Inbound Returns** (`/returns`)

| Step | Action | Pass |
|------|--------|------|
| 2C.1 | Focus barcode field; wedge scan EAN → field fills + **Enter** submits scan | Row updates without double-submit |
| 2C.2 | Type barcode manually + Enter | Same as wedge |
| 2C.3 | Scan while confirm buttons focused | Enter does **not** accidentally confirm selection |
| 2C.4 | Rapid double-scan same code | Idempotent scan key prevents duplicate corruption |

**Regression:** `onKeyDown` on barcode input calls `preventDefault()` for Enter so nested forms do not submit.

## PX-DESK-2B — Treasury print / PDF

| App | Step | Pass |
|-----|------|------|
| Supplier | Treasury → **Export PDF** | System print dialog; “Microsoft Print to PDF” saves |
| Warehouse | Invoices tab → **Export PDF** | Same |

## PX-DESK-2D — Deep links

Register schemes: `pegasusx-retailer://`, `pegasusx-warehouse://`, `pegasusx-supplier://`, `pegasusx-factory://`

| Step | Action | Pass |
|------|--------|------|
| 2D.1 | App running; `start pegasusx-retailer://notifications` | Focuses app, navigates to `/notifications` |
| 2D.2 | App closed; open deep link | Single instance starts + routes |
| 2D.3 | Handoff card in notifications inbox | In-app route matches `deep_link` path |

## PX-DESK-2E — Single instance (dock PC)

| Step | Action | Pass |
|------|--------|------|
| 2E.1 | Launch retailer desktop twice | Second launch focuses existing window |
| 2E.2 | Same for warehouse desktop | Same |
| 2E.3 | Same for supplier + factory desktop | Same |

## macOS notarize smoke (clean VM)

Run on **unsigned** CI artifact first, then release-signed `.dmg`.

```bash
xcrun stapler validate "Pegasus Retailer Desktop.dmg"
spctl -a -vv -t install "/Applications/Pegasus Retailer Desktop.app"
```

| Step | Pass |
|------|------|
| Gatekeeper accepts stapled app | `accepted` |
| Cold launch on VM without dev cert installed | Opens without right-click Open |

## Offline / cache (Phase 1 regression)

| App | Step | Pass |
|-----|------|------|
| Retailer | Kill network; reopen dashboard | Cached KPI shell + offline tray |
| Retailer | Queue checkout offline; reconnect | Pending checkout retries |
| Warehouse | Offline dispatch page | Cached preview + runs |
| Supplier | Offline dashboard | Cached KPIs |
