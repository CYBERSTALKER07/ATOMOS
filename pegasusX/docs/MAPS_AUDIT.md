# Maps / geography / display — code audit

**Date:** 2026-08-18  
**Tree:** `pegasusX/`  
**Kind:** point-in-time audit with `file:line` opened that day. **Not** a go-live certificate. **Not** permission to terraform apply, paste Maps/OSRM keys, flip `checkout_reads_this`, or swap MapLibre/MapKit for Google Maps.

**Honesty:** Current source is SoT. If this file and code disagree, **code wins**. Re-open the cited paths before using a row as status. Matrix **"Wired"** does not override this.

**Index:** [`SURFACE_AUDITS.md`](./SURFACE_AUDITS.md)

**Related (not this audit):**

| Doc | Role |
|-----|------|
| [`GLOBAL_SCALE_PROGRAM.md`](./GLOBAL_SCALE_PROGRAM.md) | Destination. Out of scope: swapping MapLibre/MapKit for Google Maps (`MapsAdapter` is routing). |
| [`GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`](./GLOBAL_SCALE_LOCAL_ECOSYSTEM.md) | Local-first matching law (GS-L). |
| [`OPTIMIZER_AND_ROUTING_RUNTIME.md`](./OPTIMIZER_AND_ROUTING_RUNTIME.md) | OR-Tools VRP vs heuristic. **Not** the map canvas. |
| [`MANIFEST_DUAL_PLANE.md`](./MANIFEST_DUAL_PLANE.md) | Factory vs supplier truck tables — live-map planes must not merge. |
| [`E2_PER_SUPPLIER_PERIMETER_DESIGN.md`](./E2_PER_SUPPLIER_PERIMETER_DESIGN.md) | Per-supplier Redis perimeter (helper exists; production still global). |
| `.agents/memory/WORKSPACE.md` | Compact verified bullets copied from this audit. |

---

## 0. Verdict

```
VERDICT: PARTIAL
NOT READY FOR LAYER B (maps program)
EVIDENCE: paths below, opened 2026-08-18
DOCS vs CODE: program law matches (do not swap display SDK). Control-tower camera/token and H3 leftover resolution are code drift vs that law.
BLOCKERS: ranked in §8 (code first, then ops extract/key)
NEXT: control-tower pack camera + MapLibre (one display stack). Not Maps/OSRM keys. Not Mapbox native SDK.
```

| Plane | Verdict | Meaning |
|-------|---------|---------|
| **A. Matching** (who serves this store) | **REAL** | H3 res 7 + pack country + pins + closest covering warehouse. Empty country fail-closed. |
| **B. Path** (how the truck drives) | **PARTIAL** | Google Routes → OSRM → densified haversine. OSRM pod **exits** without `/data/region.osrm`. Dense is not streets. |
| **C. Display** (what the human sees) | **PARTIAL** | MapLibre + Carto on ops web/Android; MapKit on iOS; Google Maps SDK on last-mile Android. Control-tower web is **Mapbox + San Francisco**. Factory mobile is a **list**, not a canvas. Payload has **no** map SDK. |

Do not call maps “wired,” “done,” or “cloud-ready” from this file.

---

## 1. Product law (do not violate)

From `GLOBAL_SCALE_LOCAL_ECOSYSTEM.md` §1 and `GLOBAL_SCALE_PROGRAM.md` out-of-scope:

1. Isolation key stays `SupplierId`. Pack / cell / country / city / region are attributes.
2. Market owns camera + distance unit + `MapsAdapter`. `MapsAdapter: GOOGLE_ROUTES` is **plane B** (geometry), not a UI SDK.
3. Same-market only. Else `422 cross_market_deferred`.
4. Local-first: closest covering warehouse; pins override.
5. Empty geography fail-closed: missing country / invalid lat/lng → `422 geography_incomplete`.
6. Matching H3 is **res 7**.
7. Unkeyed ≠ success: missing Maps key → Nominatim / skip Google Routes, not a fake 200 overlay.
8. **Do not** swap MapLibre/MapKit for Google Maps as product display.
9. Factory transfer trucks and supplier delivery trucks stay **two tables**. Live-map APIs stay dual-plane.
10. `checkout_reads_this` still false on the shipped UZ pack. Geography matching (GS-L) is a **separate** path from checkout PSP/fiscal flag.

Shipped UZ pack camera (not invented at render time): `MapCenterLat: 41.2995`, `MapCenterLng: 69.2401` — `auth/market_pack.go:126-128`. Empty/planned pack → camera helper returns not-ok — `auth/maps_pack.go:5-12`. Clients `mapInitialViewState` → `{0,0, zoom 1}` — `packages/api-client/market-pack.ts:88-94`. iOS `packMapCoordinate` → `(0,0)` when pack empty — `packages/mobile-ios-design/MarketPack.swift:65-66`.

---

## 2. Architecture — three planes

Do not conflate these. A store pin is not a polyline. A polyline is not a tile. A hex on the control tower is not the checkout cell unless resolution matches.

```text
                    ┌─────────────────────────────────────────┐
                    │  Spanner: lat/lng, CountryCode, H3Cell  │
                    │  Orders / Warehouses / Factories / Pins │
                    └──────────────────┬──────────────────────┘
                                       │
           ┌───────────────────────────┼───────────────────────────┐
           ▼                           ▼                           ▼
   Plane A MATCHING            Plane B PATH                 Plane C DISPLAY
   H3 res 7                    Google Routes →              Pack camera +
   StampNodeGeography          OSRM → densify 25 m          overlay of A/B
   ResolveServingWarehouse     stored EncodedPolyline       (tiles are not SoT)
           │                           │                           │
           ▼                           ▼                           ▼
   catalog / checkout          GET …/fleet/live-map         MapLibre / MapKit /
   warehouse_id on order       driver MapScreen             Google Maps SDK /
                               retailer tracking            Mapbox control-tower
```

Live driver does **not** overwrite the store pin. It writes Redis last-location + WS rooms + a **throttled** outbox copy (standalone `Apply`, not same-txn as Redis) — `telemetryroutes/bus_emitter.go:13-19`, `:48-50`; throttle 5s — `telemetryroutes/routes.go:235-246`.

### Data flow (spine)

```text
onboarding / pin store
  → /v1/platform/geocode/*  (PUBLIC — no RequireRole)
  → persist WGS84 + StampNodeGeography (res 7, pack country)

checkout / catalog
  → SpannerWarehouseResolver
      → proximity.ResolveServingWarehouse (pins → closest covering)
  → Orders.H3Cell / Lat / Lng (res 7)

seal / dispatch
  → GeometryBuilder.Build (google_routes_driving | osrm | computed_dense)
  → EncodedRoutePolyline on SupplierTruckManifests

driver
  → POST /v1/telemetry/location  RequireRole DRIVER
  → Redis last-loc + WS (supplier/warehouse/retailer rooms)
  → optional DRIVER_APPROACHING when within pack breach radius
  → throttled OutboxEvents TopicRealtime

ops / last-mile UI
  → GET /v1/supplier/fleet/live-map
  → GET /v1/warehouse/ops/fleet/live-map
  → GET /v1/factory/fleet/live-map   (pins only; geometry deferred)
  → clients draw server polyline + last pin (except factory mobile = list)
```

---

## 3. Algorithms (what the code actually runs)

| Job | Algorithm | Path | Notes |
|-----|-----------|------|-------|
| Node / checkout cell | Uber H3 **res 7** | `proximity/node_geography.go:7-22`, `order/unified_checkout.go:17` | Product law. |
| Cover test | Same country; if coverage cells set, H3 membership (child→parent); else whole country | `proximity/coverage_engine.go:121-135` | Empty country → false. |
| Serving warehouse | Pins win, else haversine among eligible **on-shift** same-country covering warehouses | `coverage_engine.go:171-207` | Resolver: `order/warehouse_resolver_spanner.go:23-24`. |
| Street geometry | Google ComputeRoutes (origin+dest+≤25 intermediates = **27** pts, extras sampled) → OSRM → densify **25 m** | `routing/google_routes.go:21-22`, `builder.go:5-44`, `geometry.go:24-45` | Fallback `source: computed_dense` is **not** streets. |
| Stop order | Nearest-neighbor + 2-opt | `routing/localsearch.go:12-19` | Dispatch resequence. |
| Doorstep unlock | H3 **res 9** (~174 m) + 100 m geofence; pack `BreachRadiusMeters` 150 | `order/proximity_settlement.go:20-25`, pack `:131` | Tighter than matching cell. |
| Zone gate | Redis `SISMEMBER` **global** `ssmr:delivery_perimeter` at **res 9** | `retailer/proximity_service.go:18-29` | `PerimeterKeyForSupplier` **not** wired (`:33-34`). |
| Exception heatmap | `H3CellFromLatLng` = **res 9** | `supplier/exception_map.go:169`, `proximity/h3_cell.go:3-8` | Will not line up with checkout res 7. |
| Coverage polygon leftover | `CoverageResolution = 9` | `proximity/h3.go:10-11` | Named leftover vs matching 7. |
| Doorstep cfg leftover | comment “usually 10 or 11” | `order/proximity.go:16-17` | Settlement path uses res 9, not this. |
| Live map list | Spanner ExactStaleness **15s** + last-location TTL | `supplier/fleet_live_map.go:75-77` | Warehouse analog: `warehouse/fleet_live_map.go`. |
| Approach WS | Haversine vs supplier/pack breach | `telemetryroutes/routes.go:168-188` | `DRIVER_APPROACHING` / `DELIVERY_ARRIVING`. |

**H3 mix is real drift.** Matching writers use 7. Perimeter, settlement, `H3CellFromLatLng`, and exception buckets use 9. A hex on the exception / control-tower map is **not** the checkout cell.

---

## 4. Backend (routes and honesty)

### 4.1 Geography writers

- `StampNodeGeography` — pack country inherit/reject + always H3 res 7 — `proximity/node_geography.go:42-56`.
- Checkout cell — `order/unified_checkout.go:17`, `:450`.
- Serving warehouse — `order/warehouse_resolver_spanner.go` calls `ResolveServingWarehouse`.

### 4.2 Geocode (address → WGS84)

| Endpoint | Auth | Provider |
|----------|------|----------|
| `GET /v1/platform/geocode/autocomplete` | **none** | Google Places Autocomplete **legacy JSON** if key; else Nominatim via forward |
| `GET /v1/platform/geocode/place` | **none** | Google Place Details JSON / Nominatim |
| `GET /v1/platform/geocode/reverse` | **none** | Google Geocoding / Nominatim |
| `POST /v1/platform/geocode/forward` | **none** | Google Geocoding / Nominatim |

Mount: `geolocation/handlers.go:20-32` via `platformroutes/routes.go:38-39` from `main.go:172-175`. Comment on the handler: “public geocode helpers used during onboarding.” Redis TTL cache 24h autocomplete / 7d others — `geolocation/service.go:21-24`. Places URL `maps.googleapis.com/maps/api/place/autocomplete/json` — `:166-170`. **No** `components=country:` / pack bias.

Portal callers (unauthenticated fetch): `apps/{supplier,warehouse,factory}-portal/lib/geocode.ts`, `apps/retailer-app-desktop/lib/geocode.ts`.

### 4.3 Live location

- `POST /v1/telemetry/location` — `RequireRole DRIVER` (`telemetryroutes/routes.go:123-129`).
- Redis last-location save `:158-161`.
- WS broadcast to telemetry rooms `:164-166`.
- Throttled Kafka via outbox `:163`, `bus_emitter.go:28-50`.

### 4.4 Fleet live-map (read)

| Role | Route | Geometry | Mount |
|------|-------|----------|-------|
| Supplier | `GET /v1/supplier/fleet/live-map` | polyline + driver pin | `supplierroutes/routes.go:260` |
| Warehouse | `GET /v1/warehouse/ops/fleet/live-map` | polyline + driver pin | `warehouseroutes/routes.go:150` |
| Factory | `GET /v1/factory/fleet/live-map` | **pins only; geometry deferred** | `factoryroutes/routes.go:58`, `factory/fleet_live_map.go:63` |

Factory query is `FactoryTruckManifests`. Supplier query is `SupplierTruckManifests`. Do not join them — [`MANIFEST_DUAL_PLANE.md`](./MANIFEST_DUAL_PLANE.md).

### 4.5 Pack fields (session, not display SDK)

UZ shipped pack: `MapsAdapter: "GOOGLE_ROUTES"`, camera Tashkent, `GridSystem: "H3"`, `DistanceUnit: "km"`, `BreachRadiusMeters: 150`, `CheckoutReadsThis: false` — `auth/market_pack.go:126-139`. Camera helper fail-closed — `auth/maps_pack.go:5-12`.

---

## 5. Infra

| Piece | Fact | Path |
|-------|------|------|
| Google key | `GOOGLE_MAPS_API_KEY` from `backend-go-secrets` / `google-maps-api-key`, **optional** | `infra/k8s/backend-go/deployment.yaml:95-100` |
| Key use | Geocoding + Places JSON + Routes ComputeRoutes | ConfigMap comment `:80`; `bootstrap/bootstrap.go:364` |
| Routing mode | `ROUTING_PROVIDER: auto` | `infra/k8s/backend-go/configmap.yaml:27-30` |
| OSRM URL | `http://osrm.pegasusx.svc.cluster.local:5000` | same |
| OSRM pod | init **exits 1** if `/data/region.osrm` missing | `infra/k8s/osrm/deployment.yaml:19-28` |
| OSRM in prod overlay | included | `infra/k8s/overlays/prod/kustomization.yaml:17-19` |
| Display tiles | Carto public styles (no MapTiler/MapLibre token in k8s) | portal `mapStyle` URLs |
| Control-tower tiles | Mapbox `dark-v11` + env `NEXT_PUBLIC_MAPBOX_TOKEN` **or hardcoded fallback pk** | `packages/ui-kit/.../HexagonalControlTowerMap.tsx:10-12`, `:67` |
| Optimizer | OR-Tools VRP; prod replicas often **0** | [`OPTIMIZER_AND_ROUTING_RUNTIME.md`](./OPTIMIZER_AND_ROUTING_RUNTIME.md) — not the canvas |

OSRM extract + Maps key are **ops**. Control-tower camera/token and SDK split are **code**. Ops does not close the code gaps.

---

## 6. Role × platform display

Style URL used by MapLibre (ops): `https://basemaps.cartocdn.com/gl/positron-gl-style/style.json` (heatmap / marketing / retailer-desktop dark: `dark-matter-gl-style`).

| Role | Needs a map? | Backend | Web | iOS | Android | Verdict |
|------|--------------|---------|-----|-----|---------|---------|
| **Retailer** | Pin store; track driver; optional hex density | Geocode; order lat/lng; WS approach | Desktop MapLibre tracking | MapKit picker + delivery. Hex map camera uses `packMapCoordinate`; **overlay comment “wired to API later”** | **Google Maps** picker, tracking, hex. Hex camera **hardcoded SF** `LatLng(37.74, -122.4)` | **PARTIAL** |
| **Driver** | Stops + live route | Telemetry POST; stored polyline | — | MapKit `FleetMapView` | **Google Maps** `MapScreen.kt` | **PARTIAL** — tiles ≠ Routes/OSRM/dense |
| **Supplier** | Dispatch preview, live fleet, exceptions, heatmap, control tower | live-map, dispatch geometry, exception buckets | MapLibre + **Mapbox+DeckGL hex (SF)** | MapKit preview + live | MapLibre Native 11.7.1 | **PARTIAL** |
| **Warehouse** | Same as supplier, warehouse-scoped | `ops/fleet/live-map` | MapLibre + **same Mapbox hex** | MapKit | MapLibre Native | **PARTIAL** |
| **Factory** | Yard + factory-truck pins | live-map **no `route_geometry`** | MapLibre `FleetLiveMap` + `LocationPicker` | `FleetView.swift` **list** (lat/lng text) | `FleetScreen.kt:63-66` **list** | **PARTIAL** / mobile canvas **GONE** |
| **Payload** | Dock / seal — no last-mile map required | Dual-plane APIs, no map route | — | **no MapKit** | **no Google/MapLibre** | **GONE** (OK unless yard map is a product ask) |

Control-tower consumers of the Mapbox widget:

- `apps/supplier-portal/app/(portal)/control-tower/page.tsx`
- `apps/warehouse-portal/app/control-tower/page.tsx`

`@pegasusx/ui-kit` depends on **both** `mapbox-gl` and `maplibre-gl` — `packages/ui-kit/package.json:60-61`. Fleet maps import `react-map-gl/maplibre`. Hex map imports `react-map-gl/mapbox`.

---

## 7. Cross-role integration (consistency)

### Consistent

- One WGS84 pair on the order; live driver is a **separate** last-location, not a store rewrite.
- Ops/last-mile overlays that **do** draw geometry use **server** `encoded_polyline` / `coordinates`, not a client Google Directions call.
- Pack camera fail-closed on empty/planned pack (web helper + iOS helper). UZ shipped pack is Tashkent, not invented in the MapLibre components that call `mapInitialViewState`.
- Dual manifest planes: factory live-map does not join `SupplierTruckManifests`.
- Matching empty country / cross-market fail-closed in `ResolveServingWarehouse`.

### Broken / split

1. **Control-tower web** ignores pack camera. Initial view `-122.4, 37.74` (San Francisco), Mapbox dark streets, fallback public-style token in source — `HexagonalControlTowerMap.tsx:10-12`, `:43-67`.
2. **Retailer Android hex** uses the same SF camera — `HexagonalControlTowerMap.kt:25-27`. iOS hex uses pack camera but **does not draw hex data** (“wired to API later”).
3. **H3 res 7 vs 9** — checkout cell ≠ perimeter/settlement/exception hex. Heatmap `hex` from `H3CellFromLatLng` will not match matching cells.
4. **Tiles vs path** — driver/retailer Android see Google basemap; polyline may be OSRM or `computed_dense`. Roads can disagree.
5. **Factory mobile** loads live-map JSON into a list (`FleetScreen.kt`, `FleetView.swift`); portal has the canvas. Role-row incomplete.
6. **Geocode is public** — unauthenticated Google/Nominatim spend. No pack-country bias → autocomplete is worldwide, not local-first.
7. **Legacy Places Autocomplete JSON** — not the current Places API (New).
8. **Perimeter** still one global Redis set. Second supplier can share or clobber zone membership. Design: [`E2_PER_SUPPLIER_PERIMETER_DESIGN.md`](./E2_PER_SUPPLIER_PERIMETER_DESIGN.md).
9. **Telemetry outbox** is throttled standalone `Apply` (comment: location SoT is Redis/WS). Digital-twin / Kafka consumers see a **sampled** copy, not every GPS tick. Honest, but not “full fidelity on the bus.”

---

## 8. Ranked blockers

**Code (do these before treating maps as Layer B leftovers):**

1. Control-tower web: pack camera + MapLibre/Carto (or Deck on MapLibre). Remove SF default and source fallback Mapbox token. Same camera fix on retailer Android hex.
2. One H3 resolution in **matching writers**: res 7. Keep settlement/perimeter at 9 only with a **named field** (not reuse `H3CellFromLatLng` on exception/heatmap paths).
3. `RequireRole` (or at least `RequireAnyAuthenticated`) + pack `components=country:` on geocode.
4. Factory iOS/Android: same live-map **canvas** as portal, or explicit “pins as list; map deferred” copy — not a silent list.
5. Retailer iOS hex: draw API hexes or drop the empty Map until the path exists (empty Map + “wired later” is theatre if sold as live).

**Ops (after 1–4, not instead of):**

6. OSRM PVC extract at `/data/region.osrm` (Gate-0). Missing extract → Google Routes or dense, not a silent empty map.
7. GSM `google-maps-api-key` for Routes + Geocoding. Empty key → Nominatim + skip ComputeRoutes.

**Not blockers for maps display (do not start as the “maps fix”):**

- Flip `checkout_reads_this`
- Terraform apply / new cell
- Stripe / Soliq / PSP keys
- Install Mapbox Maps SDK v11 on iOS/Android
- Swap product MapLibre/MapKit to Google Maps
- Optimizer-core replica count (VRP, not canvas)

---

## 9. Skills vs code (best practices)

Installed globally 2026-08-18 (does **not** change product law):

| Skill | vs this tree |
|-------|----------------|
| **MapLibre tile-sources / cartography** | Carto style URL is a valid hosted style. GeoJSON route overlays (tens of features) are in the sweet spot. Dual Mapbox+MapLibre in ui-kit violates “one display stack.” |
| **MapLibre ≠ Mapbox** | Control-tower **is still Mapbox GL**. Rest of portals already MapLibre. |
| **google-maps-platform** | Routes v2 `ComputeRoutes` is current (**good**). Autocomplete/Geocode JSON is **legacy**. Unauthenticated geocode is a cost / ToS hole. Last-mile Android **does** use Maps SDK for **display** — leftover, not to expand. |
| **Mapbox iOS/Android patterns** | **Not** the product native stack. Do not add Mapbox Maps SDK to driver/retailer. |
| **MapKit** | iOS last-mile + supplier/warehouse maps match. Camera should keep pack center. Retailer iOS hex overlay is unfinished. |

**INTEGRATE Mapbox native or Google Maps as display SoT: NO.** Keep MapLibre (web + supplier/warehouse Android) and MapKit (iOS). Google Maps SDK on driver/retailer Android is the leftover to **align later**, not a new vendor program.

---

## 10. Intuitive UI (what “done” would look like)

Not implemented. Target when a code slice is requested:

- Every map that opens **without** a pin starts at **pack camera** (or fail-closed 0,0 zoom 1) — never a leftover city.
- One basemap family per surface: MapLibre+Carto (web + ops Android), MapKit (iOS). Last-mile Android either stays Google **knowingly** or migrates to MapLibre Native — do not add a third.
- Live fleet draws **server** geometry + last GPS; stale pin labeled (`location_stale`).
- Factory mobile shows the same pins (or honest “list until geometry ships”).
- Control tower hexes are **res 7 matching cells** or a labeled res-9 doorstep layer — never unlabeled mixed res.
- Geocode suggestions biased to pack country; 401 without session after onboarding public window (if product still needs a public window, name it and rate-limit).
- Driver/retailer see the **same** stop coordinates as dispatch (order lat/lng), not a second geocode.

---

## 11. Next slice (when asked to implement)

**One phase:** control-tower web → MapLibre + `mapInitialViewState(pack)` + delete Mapbox fallback token and SF camera. Same SF camera fix on retailer Android hex.

Then: geocode auth + country bias. Then: H3 helper split (matching 7 vs settlement 9). Then: factory mobile canvas **or** explicit deferral in UI.

Do not start Layer B keys as that slice.
