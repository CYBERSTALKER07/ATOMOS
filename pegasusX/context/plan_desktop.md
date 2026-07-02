# pegasusX — Desktop Client Improvement Plan

Last updated: 2026-07-02

**Authority:** Subordinate to [`plan.md`](plan.md). Complements [`plan_ecosystem_sync.md`](plan_ecosystem_sync.md) (realtime spine) and [`plan_production_scale.md`](plan_production_scale.md) (cloud cutover).

**Decision (locked):** Keep **Next.js 15 + Tauri 2** for all role-row desktop shells. Do **not** rewrite to WinUI, Flutter, or Electron. Improve the shell and shared desktop packages instead.

**Scope:** Four Tauri hybrids — `retailer-app-desktop`, `supplier-portal`, `warehouse-portal`, `factory-portal`. Out of scope: payload-terminal (Expo), native iOS/Android field apps, browser-only hosting without Tauri.

**Related:**
- [`ROLE_ROW_PARITY_MATRIX.md`](../docs/ROLE_ROW_PARITY_MATRIX.md) — screen parity
- [`retailer-app-desktop/ssr_evaluation.md`](../apps/retailer-app-desktop/ssr_evaluation.md) — SSR vs static export tradeoffs
- [`DEPLOYMENT_AND_DISTRIBUTION_PLAN.md`](../docs/DEPLOYMENT_AND_DISTRIBUTION_PLAN.md) — MSI/DMG, updater CDN
- [`CLOUD_CREDENTIALS_CHECKLIST.md`](../docs/CLOUD_CREDENTIALS_CHECKLIST.md) — Tauri code-signing

---

## North star

Desktop operators get **installed-app reliability** on Windows and macOS:

1. **Instant open** — last-known data visible before network round-trip
2. **Safe auth** — JWT in OS keyring, never in `localStorage`
3. **Resilient ops** — checkout/dispatch actions survive brief offline; replay with idempotency keys
4. **Silent realtime** — WS + session reconcile; no full-page reload anti-patterns
5. **Shippable updates** — signed Tauri updater channel per app; Windows CI build green

Same TypeScript contracts as portals (`@pegasusx/api-client`, `@pegasusx/types`, `@pegasusx/ws-refresh-contract`). One UI codebase per role; Tauri is packaging + OS integration only.

---

## Current baseline (2026-07-02)

| App | Routes / depth | Tauri shell today | Gaps |
|-----|----------------|-------------------|------|
| **retailer-app-desktop** | Richest retailer surface (procurement, dock, checkout) | Keyring, updater stub | Pending checkout in `localStorage`; no SQLite; best test coverage (~6 vitest files) |
| **supplier-portal** | ~48 routes, full ops spine | Keyring, updater stub | Thin portal tests; Tauri Android redundant vs native Kotlin |
| **warehouse-portal** | ~35 routes, dispatch/fleet/treasury | Keyring, fs/dialog plugins declared, barely used | Keyboard-wedge returns only; no local dispatch cache |
| **factory-portal** | ~19 routes, manifests/loading-bay | Keyring, updater stub | Lighter than portal native on some sheets |

**Shared patterns already wired:** `isTauri()` bridge, OS keyring (`store_token` / `get_token` / `clear_token`), `TAURI_BUILD=1` static export, session reconcile hooks on hot pages, offline empty states, `window.print()` on treasury.

**Shared gaps:**
- Production updater signing keys in GSM (dev pubkey committed for CI; rotate before external release)
- `@tauri-apps/plugin-fs` / `plugin-dialog` wired for CSV export on supplier + warehouse; factory uses web fallback until plugins registered
- No shared `packages/desktop-bridge` — four copies of `lib/bridge.ts`
- True SSR impossible with bundled static export (documented; accept and compensate with cache)
- Supplier signal-ingest ops panel portal-only (native gap intentional per matrix)

---

## What we are NOT doing

- Rewriting desktop UIs in WinUI, WPF, Flutter, or Electron
- Hosting Tauri webviews against remote Next.js SSR (breaks offline install model)
- Deprecating native mobile in favor of Tauri Android (supplier native is canonical)
- Building a generic “desktop framework” — only shared packages where duplication hurts

---

## Program phases

### Phase 0 — Shell hardening & release plumbing (P0)

**Goal:** Desktop builds are reproducible, signed, and updatable on Windows + macOS.

| Anchor | Work | Exit |
|--------|------|------|
| `PX-DESK-0A` | Replace `UPDATE_PUBLIC_KEY` with real updater signing keys; document key rotation in credentials checklist | `tauri.conf.json` pubkey set; test update manifest on GCS | **partial** — dev pubkey + GSM/terraform secrets; prod values via `sync_desktop_release_secrets.sh` |
| `PX-DESK-0B` | CI: `TAURI_BUILD=1` + `tauri build --target x86_64-pc-windows-msvc` for all four apps (nightly or release branch) | Green workflow artifact: `.msi` per app | **shipped** — `desktop-windows-build.yml` |
| `PX-DESK-0C` | Windows code-sign + timestamp (Authenticode); macOS notarize path documented | Signed installer smoke on clean VM | **partial** — `sign_desktop_windows.ps1` + QA runbook; VM smoke manual |
| `PX-DESK-0D` | Extract shared `packages/desktop-bridge` (keyring, `isTauri`, app info) | Four apps import package; thin `lib/bridge.ts` re-exports | **shipped** |
| `PX-DESK-0E` | Unify updater CDN base URLs (retailer uses GCS; others pegasus-x.com — align buckets) | Single distribution runbook section | **shipped** — all four on `pegasusx-ssmr-app-updates` |

**Anchor:** `PX-DESK-0` — installers build in CI; updater keys real; bridge DRY.

---

### Phase 1 — Offline resilience & local cache (P0)

**Goal:** Cold start feels fast; critical mutations survive network blips.

| Anchor | Work | Exit |
|--------|------|------|
| `PX-DESK-1A` | `packages/desktop-cache` — Tauri SQL plugin wrapper + migration helpers | Shared TS API + vitest; retailer Tauri registers `tauri-plugin-sql` | **shipped** |
| `PX-DESK-1B` | **Retailer:** SQLite cache for profile, catalog browse snapshot, open orders list | `useLiveData` hydrates from cache on Tauri; dashboard/orders/catalog | **shipped** — profile via `retailer-profile.ts`; URL-keyed live data cache |
| `PX-DESK-1C` | **Retailer:** Move pending checkout queue from `localStorage` to encrypted SQLite (or keyring-backed store) | `pending-checkout` via desktop-cache; legacy migration on first read | **shipped** |
| `PX-DESK-1D` | **Warehouse:** Cache dispatch preview + active manifest list for dock PCs | Dispatch page loads cached snapshot; WS/reconcile invalidates | **shipped** — dispatch preview + dispatch runs SQLite hydrate |
| `PX-DESK-1E` | **Supplier:** Cache dashboard KPIs + orders list (same keys as session reconcile) | Supplier dashboard instant shell on reopen | **shipped** — dashboard bundle + orders list per filter/page |
| `PX-DESK-1F` | Offline banner + “queued actions” tray component in `@pegasusx/ui-kit` | All four apps use shared `DesktopOfflineTray` | **shipped** |

**Reference:** [`ssr_evaluation.md`](../apps/retailer-app-desktop/ssr_evaluation.md) recommends SQLite — implement here.

**Anchor:** `PX-DESK-1` — retailer + warehouse dispatch prove local cache; checkout queue durable.

---

### Phase 2 — Desktop-native capabilities (P1)

**Goal:** Use the Tauri shell for real operator workflows, not just a webview wrapper.

| Anchor | Work | Exit |
|--------|------|------|
| `PX-DESK-2A` | Wire `plugin-dialog` + `plugin-fs` for CSV export (inventory audit, ledger, earnings) | Supplier + warehouse export buttons save to user-picked path | **shipped** — `@pegasusx/desktop-bridge` `exportCsv`; supplier fs/dialog plugins |
| `PX-DESK-2B` | Native print pipeline: wrap `window.print()` + optional PDF save via dialog on treasury/invoice pages | Supplier + warehouse treasury print tested on Windows | **shipped** — `desktopPrint` on treasury pages |
| `PX-DESK-2C` | **Warehouse returns:** document + test USB barcode wedge on Windows WebView2 | QA runbook step; no regression on Enter-to-submit | **shipped** — `docs/qa/PX-DESK_MANUAL_QA.md` + Enter `preventDefault` |
| `PX-DESK-2D` | Deep links / custom protocol (`pegasusx-retailer://`, etc.) for notification handoff cards | Handoff inbox opens correct desktop route | **shipped** — all four schemes + `DesktopDeepLinkBootstrap` |
| `PX-DESK-2E` | Single-instance + “second window focuses existing” (dock PCs launching app twice) | Tauri `single_instance` plugin on warehouse + retailer | **shipped** |
| `PX-DESK-2F` | Deprecate **supplier Tauri Android** in docs; native Kotlin is primary row client | README + matrix note; no new Tauri Android features | **shipped** |

**Anchor:** `PX-DESK-2` — file export + print + wedge QA green on Windows.

---

### Phase 3 — Performance & operator UX (P1)

**Goal:** Large lists and maps stay smooth; loading states feel intentional.

| Anchor | Work | Exit |
|--------|------|------|
| `PX-DESK-3A` | Virtualized tables on supplier orders, warehouse dispatch, retailer orders (react-virtuoso or equivalent) | 500+ rows scroll without jank on mid-tier Windows laptop |
| `PX-DESK-3B` | MapLibre: lazy-load + destroy on unmount for fleet/live-map pages | Memory stable after 10 min on supplier fleet tab |
| `PX-DESK-3C` | Shared skeleton system in `@pegasusx/ui-kit` for BentoGrid / PageChrome | Portal + desktop use same skeleton components |
| `PX-DESK-3D` | **Retailer dock:** optimistic inbound scan queue with reconcile | Dock page matches mobile dock semantics on reconnect |
| `PX-DESK-3E` | Reduce polling where WS covers the surface (audit `usePolling` intervals on desktop-only pages) | Document per-page refresh strategy in parity matrix |

**Anchor:** `PX-DESK-3` — supplier orders + warehouse dispatch virtualization shipped.

---

### Phase 4 — Realtime parity & ecosystem alignment (P1)

**Goal:** Desktop participates fully in the PX-ECS realtime spine.

| Anchor | Work | Exit |
|--------|------|------|
| `PX-DESK-4A` | Audit session reconcile coverage — every P0 desktop page calls role reconcile on WS reconnect | Checklist in parity matrix; gap list empty for P0 screens |
| `PX-DESK-4B` | Supplier portal: signal ingest ops panel parity (already portal; ensure desktop Tauri build includes it) | `GET/POST .../planning/signals/*` on desktop build |
| `PX-DESK-4C` | Handoff timeline panels on warehouse + factory **portal** desktop (native already has 4E) | Portal dispatch + loading-bay show pulse strip |
| `PX-DESK-4D` | Dispatch fingerprint mismatch banner on supplier portal desktop (native has 4F) | Banner when preview ≠ warehouse plan |
| `PX-DESK-4E` | Desktop SSMR smoke: `PUBLIC_BASE_URL` + headless or scripted open of Tauri webview against local SSMR | Optional marker `PX_E2E_DESKTOP_WS_OK` in ssmr-smokecheck |

**Anchor:** `PX-DESK-4` — P0 pages reconcile; supplier fingerprint + signal ingest on desktop.

---

### Phase 5 — Test, QA & documentation (P2)

**Goal:** Desktop regressions caught before release train.

| Anchor | Work | Exit |
|--------|------|------|
| `PX-DESK-5A` | Vitest parity: each portal gets auth + session-reconcile + one P0 page test (match retailer depth) | `pnpm test` green per app in CI |
| `PX-DESK-5B` | `docs/qa/PX-DESK_MANUAL_QA.md` — Windows checklist (install, update, wedge scan, print, offline checkout) | Linked from `PX12_MANUAL_QA_RUNBOOK.md` | **shipped** |
| `PX-DESK-5C` | ADR `008-desktop-tauri-strategy.md` — locked decision + when to add Rust plugins vs rewrite | ADR accepted |
| `PX-DESK-5D` | Parity matrix “Desktop capabilities” appendix (cache, export, print, offline queue per role) | Matrix row per desktop-only feature | **shipped** |

**Anchor:** `PX-DESK-5` — CI tests + manual QA runbook + ADR published.

---

## Role-specific priorities

| Role | Desktop is primary for… | Top 3 improvements |
|------|-------------------------|-------------------|
| **RETAILER** | Procurement, dock, checkout, insights | SQLite cache (1B/1C), dock optimistic queue (3D), updater/signing (0A–0C) |
| **SUPPLIER** | Full ops cockpit, treasury, dispatch | Shared bridge (0D), export/print (2A/2B), orders virtualization (3A), fingerprint banner (4D) |
| **WAREHOUSE** | Dispatch, returns wedge, fleet | Dispatch cache (1D), wedge QA (2C), single-instance (2E) |
| **FACTORY** | Manifests, loading-bay, transfers | Session reconcile audit (4A), handoff timeline portal (4C), skeleton polish (3C) |

**Driver row:** no desktop client — intentional.

---

## Verification loops

```bash
# Desktop bridge + all Tauri clients (PX-DESK-0D+)
cd pegasusX && pnpm --filter @pegasusx/desktop-bridge test
cd pegasusX && make validate-desktop-clients
cd pegasusX && make validate-desktop-updater

# Windows MSI (local; requires Rust + WebView2)
cd pegasusX && bash scripts/build_desktop_windows.sh retailer-app-desktop
cd pegasusX/apps/retailer-app-desktop && pnpm typecheck && pnpm build:static

# Retailer unit tests (deepest today)
cd pegasusX/apps/retailer-app-desktop && pnpm test

# Ecosystem contracts (unchanged)
cd pegasusX && make parity-contract-full
cd pegasusX && make test-ssmr-infra

# Windows release build (local)
cd pegasusX/apps/warehouse-portal && pnpm tauri:build:win
```

**Definition of done (program):**
1. Phases 0 + 1 complete for **retailer** and **warehouse** (highest desktop-only value)
2. Phase 2 file export works on Windows for supplier + warehouse
3. Updater signing live; CI produces signed `.msi`
4. Parity matrix + ADR updated; manual QA runbook exists

---

## Anchor registry

| Anchor | Phase | Scope | Status |
|--------|-------|-------|--------|
| `PX-DESK-0` | 0 | Shell hardening & release | **partial** — CI + GSM secrets + upload script; prod cert values ops-owned |
| `PX-DESK-0A`–`0E` | 0 | Signing, CI, bridge package, CDN | **partial** — 0B/0D/0E shipped; 0A/0C ops |
| `PX-DESK-1` | 1 | Offline & SQLite cache | **shipped** — retailer, warehouse, supplier cache + offline tray |
| `PX-DESK-1A`–`1F` | 1 | desktop-cache + per-role cache | **shipped** |
| `PX-DESK-2` | 2 | Native capabilities | **shipped** — 2A–2F complete |
| `PX-DESK-2A`–`2F` | 2 | fs, print, wedge, deep links | **shipped** |
| `PX-DESK-3` | 3 | Performance & UX | **pending** |
| `PX-DESK-3A`–`3E` | 3 | Virtualization, maps, skeletons | **pending** |
| `PX-DESK-4` | 4 | Realtime & ecosystem parity | **pending** |
| `PX-DESK-4A`–`4E` | 4 | Reconcile audit, ECS features | **pending** |
| `PX-DESK-5` | 5 | Test & docs | **pending** |
| `PX-DESK-5A`–`5D` | 5 | Vitest, QA runbook, ADR | **pending** |

---

## Suggested execution order

1. **Week 1:** `PX-DESK-0D` (shared bridge) + `0A`/`0B` (signing + Windows CI) — unblocks shipping
2. **Week 2–3:** `PX-DESK-1A`–`1C` (retailer SQLite + secure checkout queue) — highest user-visible win
3. **Week 4:** `PX-DESK-1D` + `2C` + `2E` (warehouse dispatch cache + wedge QA + single instance)
4. **Week 5:** `PX-DESK-2A`/`2B` + `3A` (export/print + virtualization)
5. **Week 6:** `PX-DESK-4A`–`4D` + `5A`–`5C` (ecosystem parity closure + tests + ADR)

Parallel track: whenever touching portal pages for PX-ECS features, verify `TAURI_BUILD=1` export still compiles.

---

## Non-goals

- WinUI / native Windows rewrite
- Remote-hosted desktop (always bundle static `out/`)
- Feature parity between desktop and mobile for retailer (desktop stays richer by design)
- SQLite sync conflict resolution beyond “server wins on reconcile” (v1)
