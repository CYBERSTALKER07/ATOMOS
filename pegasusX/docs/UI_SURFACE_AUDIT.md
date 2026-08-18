# UI surfaces (web / desktop / native) — audit

**Date:** 2026-08-18  
**Tree:** `pegasusX/apps/*` + `packages/{types,api-client,ui-kit}`  
**Kind:** contracts + hosting + role-row display. Maps detail: [`MAPS_AUDIT.md`](./MAPS_AUDIT.md).

**Related:** [`SURFACE_AUDITS.md`](./SURFACE_AUDITS.md) · [`GLOBAL_SCALE_CLIENT_UI.md`](./GLOBAL_SCALE_CLIENT_UI.md) (destination, not status) · [`FIREBASE_AUDIT.md`](./FIREBASE_AUDIT.md)

---

## 0. Verdict

```
VERDICT: PARTIAL
KEEP packages/types + api-client + ui-kit as DTO/UI SoT
KEEP Tauri static export + GKE nginx (when hosted)
NO Vercel, NO Firebase Hosting/App Hosting for Class A
NEXT: admin-portal onto packages/*; control-tower MapLibre; factory native map or honest list; retailer iOS CI path
```

---

## 1. Shared packages (keep)

| Package | Role |
|---------|------|
| `@pegasusx/types` | Hand-aligned with `contracts/events.schema.json` (`packages/types/index.ts` header). `make gen-contracts-gate` is the event SoT. |
| `@pegasusx/api-client` | HTTP + `mapInitialViewState` / pack helpers. Do not fork DTOs inside apps. |
| `@pegasusx/ui-kit` | Portal chrome, StatusStack, **and** Mapbox control-tower (gap). |
| `ws-refresh-contract` | Dirty-slice WS for dashboards. |
| `mobile-ios-design` / `mobile-android-design` | StatusStack twins. |

`pnpm-workspace.yaml` lists portals + those packages.

---

## 2. Web / desktop matrix

| App | Tauri `output: "export"` | Emulator rewrite `:9099` | `packages/*` | Hosting |
|-----|--------------------------|--------------------------|--------------|---------|
| supplier-portal | yes | yes | types + api-client + ui-kit | Dockerfile + unused k8s yaml |
| warehouse-portal | yes | yes | same | **no** Dockerfile |
| factory-portal | yes | yes | same | **no** Dockerfile |
| retailer-app-desktop | yes | yes | same | **no** k8s |
| admin-portal | **no** | **no** | **none** (`package.json` no types/ui-kit) | **no** |
| marketing-site | **no** | **no** | types + ui-kit | **no** k8s |
| payload-terminal | — | — | local | **no** |

Zero `vercel.json` / Vercel config in `pegasusX`. Next rewrites are **dev Auth emulator proxy only** (`supplier-portal/next.config.mjs:26-33`). Tauri export sets `typescript.ignoreBuildErrors: true` — CI typecheck (`pnpm typecheck`) remains SoT (`comment :22-23`).

Ingress does **not** serve portal hosts. `WS_ALLOWED_ORIGINS` names `*.pegasusx.app` anyway — [`INFRA_AUDIT.md`](./INFRA_AUDIT.md).

---

## 3. Maps (summary — full doc MAPS_AUDIT)

| Surface | SDK | Honesty |
|---------|-----|---------|
| Ops portals + supplier/warehouse Android | MapLibre + Carto | **REAL** overlays of **server** polylines |
| Control-tower web (supplier + warehouse) | Mapbox dark-v11, SF camera, fallback pk token | **PARTIAL / wrong city** |
| Retailer/driver Android | Google Maps SDK | Leftover vs product law (do not expand) |
| iOS last-mile + ops | MapKit | Pack camera helpers exist |
| Factory native | **list** of live-map JSON | Portal has canvas |
| Payload | **no** map SDK | OK unless yard map is a product ask |

Do not swap MapLibre/MapKit for Google Maps. `MapsAdapter: GOOGLE_ROUTES` is routing.

---

## 4. Native role-row (12 apps)

Six roles × Android + iOS: supplier, warehouse, factory, retailer, driver, payload.

| Gap | Proof |
|-----|-------|
| Factory fleet UI is a list | Android `FleetScreen.kt` `getFleetLiveMap()` → cards; iOS `FleetView.swift` same. Portal `FleetLiveMap.tsx` MapLibre. Backend geometry **deferred** (`factory/fleet_live_map.go:63`). |
| Payload no map | Grep MapKit/GoogleMap/MapLibre empty. Board APIs exist. |
| Retailer Android HTTP | Session JWT only — `TokenManager.httpAuthorizationToken` returns JWT or null (`TokenManager.kt:49-54`). |
| CI iOS retailer | Workflow points at `reatilerapp.xcodeproj` — [`DEVOPS_CICD_AUDIT.md`](./DEVOPS_CICD_AUDIT.md). |

GS-U command dashboards (StatusStack, pack money, no invented UZS) shipped in program docs — **re-verify** the specific screen before claiming; this audit does not re-run those tests.

---

## 5. How it should be modularized (no rewrite)

1. **Types flow one way:** `contracts/` / Go events → `packages/types` → api-client → apps. App-local DTOs only for view models.
2. **Admin-portal** imports `@pegasusx/types` + ui-kit like other portals (largest SoT hole).
3. **One map stack per surface:** MapLibre web + ops Android; MapKit iOS; kill Mapbox in ui-kit control-tower.
4. **Hosting:** keep static export. If a portal must be on the internet, add **GKE nginx** (copy supplier-portal Dockerfile) + Ingress host — not App Hosting, not Vercel.
5. **Native:** share pack camera + StatusStack from design packages; do not add a third map SDK.
6. Factory native: same live-map canvas as portal **or** copy “pins as list; map deferred.”

---

## 6. Best practices vs skills

| Skill | vs code |
|-------|---------|
| nextjs / vercel-react-best-practices | App Router portals OK. Do **not** `npx vercel` Class A. |
| vercel-composition-patterns | Not installed this session; composition already via `packages/*`. |
| SwiftUI / Kotlin concurrency | Use existing skills when touching screens; this audit does not grade every view. |
| ui-ux-pro-max | GS-U nav ≤5 already a product law — do not grow home chrome. |

---

## 7. Ranked next (when asked)

1. Control-tower MapLibre + pack camera ([`MAPS_AUDIT.md`](./MAPS_AUDIT.md) §8).
2. Admin-portal onto `packages/types` + ui-kit.
3. Factory iOS/Android map **or** honest deferral copy.
4. Fix retailer iOS CI project name.

Not Firebase Hosting. Not a Google Maps migration. Not Layer B.
