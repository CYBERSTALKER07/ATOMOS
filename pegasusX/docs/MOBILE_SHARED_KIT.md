# §8.8 Mobile shared kit + offline queue

## Status (2026-08-06)

| Item | Status |
|------|--------|
| `packages/mobile-android-kit` | **Wired** — offline contract, HTTP flush semantics, prefs store, reconnect backoff, client-policy snapshot |
| `packages/mobile-ios-kit` (`PegasusKit`) | **Wired** — same offline/HTTP/reconnect surface; driver iOS local SPM |
| Driver Android queue | **Wired** — capture-time lat/lng columns, Room `MIGRATION_5_6`, no `fallbackToDestructiveMigration`, flush replays stored coords |
| Driver iOS queue | **Wired** — capture-time fields + `bodyJSONForFlush()` via PegasusKit |
| Warehouse / factory Android | **Wired** — `PrefsOfflineQueueStore` (kit) with legacy prefs migration |
| Payload Android | **Wired** — Room v2 + lat/lng + kit flush semantics |
| Payload iOS | **Partial** — richer model mirroring kit; no `.xcodeproj` yet to depend on PegasusKit |
| Warehouse scan UX (PR-7) | **Residual** — see [`WMS_GATE4_HARDENING.md`](./WMS_GATE4_HARDENING.md) |
| Desktop full uz/ru i18n | **Team-owned** — chrome/dashboard/orders Partial already in tree |

Version catalog: [`../gradle/libs.versions.toml`](../gradle/libs.versions.toml).

## SSMR / substance

Client-only residual for this wave: no new backend marker (queue contract + capture-time coords are mobile). Do not claim warehouse scan Done (PR-7). Do not claim desktop full uz/ru Done (team-owned).
